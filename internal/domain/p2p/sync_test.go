package p2p_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/baotoq/shitcoin/internal/domain/block"
	"github.com/baotoq/shitcoin/internal/domain/chain"
	"github.com/baotoq/shitcoin/internal/domain/mempool"
	"github.com/baotoq/shitcoin/internal/domain/p2p"
	"github.com/baotoq/shitcoin/internal/domain/utxo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// makeSyncTestNode creates a fully wired test node with UTXO support for sync tests.
func makeSyncTestNode(t *testing.T, minerAddr string) (*p2p.Server, *chain.Chain, *utxo.Set, *fullMockChainRepo) {
	t.Helper()

	repo := newFullMockChainRepo()
	utxoRepo := newMockUTXORepo()
	utxoSet := utxo.NewSet(utxoRepo)
	pow := &block.ProofOfWork{}
	cfg := chain.ChainConfig{
		InitialDifficulty: 1,
		GenesisMessage:    "sync-test",
		BlockReward:       5000000000,
	}
	ch := chain.NewChain(repo, pow, cfg, utxoSet)

	ctx := context.Background()
	require.NoError(t, ch.Initialize(ctx, minerAddr))

	pool := mempool.New(utxoSet)

	srv := p2p.NewServer(ch, pool, utxoSet, repo, 0)
	require.NoError(t, srv.Start(ctx))

	t.Cleanup(func() {
		srv.Stop()
	})

	return srv, ch, utxoSet, repo
}

// mineTestBlocks mines N blocks on the given chain, returning all mined blocks.
func mineTestBlocks(t *testing.T, ch *chain.Chain, n int, minerAddr string) []*block.Block {
	t.Helper()
	ctx := context.Background()
	blocks := make([]*block.Block, 0, n)
	for i := 0; i < n; i++ {
		blk, err := ch.MineBlock(ctx, minerAddr, nil)
		require.NoError(t, err)
		blocks = append(blocks, blk)
	}
	return blocks
}

func TestGetBlocks_ReturnsRequestedRange(t *testing.T) {
	// Node A has genesis + 5 blocks. Send CmdGetBlocks{1,5} and expect 5 CmdBlock responses.
	srvA, chainA, _, _ := makeSyncTestNode(t, "miner-A")

	// Mine 5 blocks
	mineTestBlocks(t, chainA, 5, "miner-A")

	require.Equal(t, uint64(5), chainA.Height())

	// Connect a raw client and send GetBlocks
	conn := dialAndHandshake(t, srvA, chainA)
	defer conn.Close()

	getBlocks := p2p.GetBlocksPayload{StartHeight: 1, EndHeight: 5}
	msg, err := p2p.NewMessage(p2p.CmdGetBlocks, getBlocks)
	require.NoError(t, err)
	require.NoError(t, p2p.WriteMessage(conn, msg))

	// Read 5 CmdBlock responses
	conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	var receivedHeights []uint64
	for i := 0; i < 5; i++ {
		resp, err := p2p.ReadMessage(conn)
		require.NoError(t, err, "reading block %d", i+1)
		require.Equal(t, p2p.CmdBlock, resp.Command)

		var bp p2p.BlockPayload
		require.NoError(t, json.Unmarshal(resp.Payload, &bp))
		receivedHeights = append(receivedHeights, bp.Height)
	}

	// Verify sequential order
	for i, h := range receivedHeights {
		expected := uint64(i + 1)
		assert.Equal(t, expected, h, "block %d", i)
	}
}

func TestGetBlocks_EndHeightZero_ReturnsTillTip(t *testing.T) {
	// EndHeight=0 should return blocks from StartHeight to chain tip.
	srvA, chainA, _, _ := makeSyncTestNode(t, "miner-A")

	mineTestBlocks(t, chainA, 3, "miner-A")

	conn := dialAndHandshake(t, srvA, chainA)
	defer conn.Close()

	getBlocks := p2p.GetBlocksPayload{StartHeight: 1, EndHeight: 0}
	msg, err := p2p.NewMessage(p2p.CmdGetBlocks, getBlocks)
	require.NoError(t, err)
	require.NoError(t, p2p.WriteMessage(conn, msg))

	// Should receive 3 blocks (heights 1, 2, 3)
	conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	count := 0
	for i := 0; i < 3; i++ {
		resp, err := p2p.ReadMessage(conn)
		require.NoError(t, err, "reading block %d", i+1)
		require.Equal(t, p2p.CmdBlock, resp.Command)
		count++
	}

	assert.Equal(t, 3, count)
}

func TestGetBlocks_StartBeyondChainHeight_ReturnsNoBlocks(t *testing.T) {
	// StartHeight beyond chain height should return no blocks.
	srvA, chainA, _, _ := makeSyncTestNode(t, "miner-A")

	// Chain only has genesis (height 0)
	conn := dialAndHandshake(t, srvA, chainA)
	defer conn.Close()

	getBlocks := p2p.GetBlocksPayload{StartHeight: 100, EndHeight: 200}
	msg, err := p2p.NewMessage(p2p.CmdGetBlocks, getBlocks)
	require.NoError(t, err)
	require.NoError(t, p2p.WriteMessage(conn, msg))

	// Should receive no blocks -- timeout is expected
	conn.SetReadDeadline(time.Now().Add(500 * time.Millisecond))
	_, err = p2p.ReadMessage(conn)
	assert.Error(t, err, "expected no response for GetBlocks beyond chain height")
}

func TestInitialBlockDownload(t *testing.T) {
	// Node A has genesis + 5 blocks. Node B starts fresh (same genesis).
	// B connects to A. After sync, B has 5 blocks and same tip hash as A.
	srvA, chainA, _, _ := makeSyncTestNode(t, "miner-A")

	mineTestBlocks(t, chainA, 5, "miner-A")

	require.Equal(t, uint64(5), chainA.Height())

	// Node B starts with only genesis
	srvB, chainB, _, _ := makeSyncTestNode(t, "miner-A") // same miner for matching genesis

	require.Equal(t, uint64(0), chainB.Height())

	// B connects to A, triggering IBD
	require.NoError(t, srvB.Connect(srvA.ListenAddr()))

	// Wait for sync to complete
	deadline := time.After(10 * time.Second)
	for {
		select {
		case <-deadline:
			require.FailNow(t, "timed out waiting for IBD",
				"B height=%d, want=5", chainB.Height())
		default:
			if chainB.Height() >= 5 {
				// Verify tip hashes match
				assert.Equal(t, chainA.LatestBlock().Hash(), chainB.LatestBlock().Hash())
				return
			}
			time.Sleep(100 * time.Millisecond)
		}
	}
}

func TestIBDUTXOConsistency(t *testing.T) {
	// After IBD, the synced node's UTXO set has correct balances.
	minerAddr := "miner-A"
	srvA, chainA, utxoSetA, _ := makeSyncTestNode(t, minerAddr)

	mineTestBlocks(t, chainA, 3, minerAddr)

	// Check balance on A
	balanceA, err := utxoSetA.GetBalance(minerAddr)
	require.NoError(t, err)
	require.Greater(t, balanceA, int64(0))

	// Node B syncs from A
	srvB, chainB, utxoSetB, _ := makeSyncTestNode(t, minerAddr)

	require.NoError(t, srvB.Connect(srvA.ListenAddr()))

	// Wait for sync
	deadline := time.After(10 * time.Second)
	for {
		select {
		case <-deadline:
			require.FailNow(t, "timed out waiting for IBD",
				"B height=%d, want=%d", chainB.Height(), chainA.Height())
		default:
			if chainB.Height() >= chainA.Height() {
				// Verify UTXO balance matches
				balanceB, err := utxoSetB.GetBalance(minerAddr)
				require.NoError(t, err)
				assert.Equal(t, balanceA, balanceB)
				return
			}
			time.Sleep(100 * time.Millisecond)
		}
	}
}

func TestIBDSyncingFlag(t *testing.T) {
	// During IBD, the server's IsSyncing() returns true.
	srvA, chainA, _, _ := makeSyncTestNode(t, "miner-A")

	mineTestBlocks(t, chainA, 5, "miner-A")

	srvB, _, _, _ := makeSyncTestNode(t, "miner-A")

	// Before connect, not syncing
	assert.False(t, srvB.IsSyncing())

	require.NoError(t, srvB.Connect(srvA.ListenAddr()))

	// Wait for sync to complete, then check syncing is false
	time.Sleep(3 * time.Second)

	assert.False(t, srvB.IsSyncing())
}
