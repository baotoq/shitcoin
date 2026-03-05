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
	if err := ch.Initialize(ctx, minerAddr); err != nil {
		t.Fatalf("chain.Initialize failed: %v", err)
	}

	pool := mempool.New(utxoSet)

	srv := p2p.NewServer(ch, pool, utxoSet, repo, 0)
	if err := srv.Start(ctx); err != nil {
		t.Fatalf("server.Start failed: %v", err)
	}

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
		if err != nil {
			t.Fatalf("MineBlock %d failed: %v", i+1, err)
		}
		blocks = append(blocks, blk)
	}
	return blocks
}

func TestGetBlocks_ReturnsRequestedRange(t *testing.T) {
	// Node A has genesis + 5 blocks. Send CmdGetBlocks{1,5} and expect 5 CmdBlock responses.
	srvA, chainA, _, _ := makeSyncTestNode(t, "miner-A")

	// Mine 5 blocks
	mineTestBlocks(t, chainA, 5, "miner-A")

	if chainA.Height() != 5 {
		t.Fatalf("expected height 5, got %d", chainA.Height())
	}

	// Connect a raw client and send GetBlocks
	conn := dialAndHandshake(t, srvA, chainA)
	defer conn.Close()

	getBlocks := p2p.GetBlocksPayload{StartHeight: 1, EndHeight: 5}
	msg, err := p2p.NewMessage(p2p.CmdGetBlocks, getBlocks)
	if err != nil {
		t.Fatalf("NewMessage failed: %v", err)
	}
	if err := p2p.WriteMessage(conn, msg); err != nil {
		t.Fatalf("WriteMessage failed: %v", err)
	}

	// Read 5 CmdBlock responses
	conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	var receivedHeights []uint64
	for i := 0; i < 5; i++ {
		resp, err := p2p.ReadMessage(conn)
		if err != nil {
			t.Fatalf("reading block %d: %v", i+1, err)
		}
		if resp.Command != p2p.CmdBlock {
			t.Fatalf("expected CmdBlock, got command %d", resp.Command)
		}
		var bp p2p.BlockPayload
		if err := json.Unmarshal(resp.Payload, &bp); err != nil {
			t.Fatalf("unmarshal block payload: %v", err)
		}
		receivedHeights = append(receivedHeights, bp.Height)
	}

	// Verify sequential order
	for i, h := range receivedHeights {
		expected := uint64(i + 1)
		if h != expected {
			t.Errorf("block %d: height = %d, want %d", i, h, expected)
		}
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
	if err != nil {
		t.Fatalf("NewMessage failed: %v", err)
	}
	if err := p2p.WriteMessage(conn, msg); err != nil {
		t.Fatalf("WriteMessage failed: %v", err)
	}

	// Should receive 3 blocks (heights 1, 2, 3)
	conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	count := 0
	for i := 0; i < 3; i++ {
		resp, err := p2p.ReadMessage(conn)
		if err != nil {
			t.Fatalf("reading block %d: %v", i+1, err)
		}
		if resp.Command != p2p.CmdBlock {
			t.Fatalf("expected CmdBlock, got command %d", resp.Command)
		}
		count++
	}

	if count != 3 {
		t.Errorf("received %d blocks, want 3", count)
	}
}

func TestGetBlocks_StartBeyondChainHeight_ReturnsNoBlocks(t *testing.T) {
	// StartHeight beyond chain height should return no blocks.
	srvA, chainA, _, _ := makeSyncTestNode(t, "miner-A")

	// Chain only has genesis (height 0)
	conn := dialAndHandshake(t, srvA, chainA)
	defer conn.Close()

	getBlocks := p2p.GetBlocksPayload{StartHeight: 100, EndHeight: 200}
	msg, err := p2p.NewMessage(p2p.CmdGetBlocks, getBlocks)
	if err != nil {
		t.Fatalf("NewMessage failed: %v", err)
	}
	if err := p2p.WriteMessage(conn, msg); err != nil {
		t.Fatalf("WriteMessage failed: %v", err)
	}

	// Should receive no blocks -- timeout is expected
	conn.SetReadDeadline(time.Now().Add(500 * time.Millisecond))
	_, err = p2p.ReadMessage(conn)
	if err == nil {
		t.Error("expected no response for GetBlocks beyond chain height, but got one")
	}
}
