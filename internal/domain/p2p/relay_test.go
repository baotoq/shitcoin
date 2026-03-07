package p2p_test

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/baotoq/shitcoin/internal/domain/block"
	"github.com/baotoq/shitcoin/internal/domain/chain"
	"github.com/baotoq/shitcoin/internal/domain/mempool"
	"github.com/baotoq/shitcoin/internal/domain/p2p"
	"github.com/baotoq/shitcoin/internal/domain/utxo"
	"github.com/baotoq/shitcoin/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// makeRelayTestNode creates a fully wired test node with UTXO support.
func makeRelayTestNode(t *testing.T, minerAddr string) (*p2p.Server, *chain.Chain, *mempool.Mempool, *testutil.MockChainRepo) {
	t.Helper()

	repo := testutil.NewMockChainRepo()
	utxoRepo := testutil.NewMockUTXORepo()
	utxoSet := utxo.NewSet(utxoRepo)
	pow := &block.ProofOfWork{}
	cfg := chain.ChainConfig{
		InitialDifficulty: 1, // very low for fast test mining
		GenesisMessage:    "relay-test",
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

	return srv, ch, pool, repo
}

// connectNodes connects two servers and waits for the handshake to complete.
func connectNodes(t *testing.T, from, to *p2p.Server) {
	t.Helper()
	require.NoError(t, from.Connect(to.ListenAddr()))
	// Brief wait for inbound handling
	time.Sleep(200 * time.Millisecond)
}

// dialAndHandshake connects to a server, completes the inbound version handshake,
// and returns the raw connection. The inbound handshake flow (from the server's POV)
// is: receive Version -> check genesis -> send Version + Verack -> receive Verack.
// So as the client we: send Version -> receive Version -> receive Verack -> send Verack.
func dialAndHandshake(t *testing.T, srv *p2p.Server, ch *chain.Chain) net.Conn {
	t.Helper()

	conn, err := net.DialTimeout("tcp", srv.ListenAddr(), 2*time.Second)
	require.NoError(t, err)

	conn.SetDeadline(time.Now().Add(5 * time.Second))

	// Send our version first (server's inbound handshake expects to receive version first)
	versionPayload := p2p.VersionPayload{
		Version:    1,
		Height:     ch.Height(),
		ListenPort: 0,
	}
	versionPayload.GenesisHash = ""

	versionMsg, err := p2p.NewMessage(p2p.CmdVersion, versionPayload)
	require.NoError(t, err)
	require.NoError(t, p2p.WriteMessage(conn, versionMsg))

	// Receive server's version
	msg, err := p2p.ReadMessage(conn)
	require.NoError(t, err)
	require.Equal(t, p2p.CmdVersion, msg.Command)

	// Receive server's verack
	msg, err = p2p.ReadMessage(conn)
	require.NoError(t, err)
	require.Equal(t, p2p.CmdVerack, msg.Command)

	// Send our verack
	verack := p2p.Message{Command: p2p.CmdVerack, Payload: []byte("{}")}
	require.NoError(t, p2p.WriteMessage(conn, verack))

	// Clear deadline
	conn.SetDeadline(time.Time{})

	return conn
}

func TestBlockBroadcast(t *testing.T) {
	// Two nodes with the same genesis. Mine a block on A, verify B receives it.
	srvA, chainA, _, _ := makeRelayTestNode(t, "miner-A")
	srvB, chainB, _, _ := makeRelayTestNode(t, "miner-A") // same miner for matching genesis

	connectNodes(t, srvB, srvA)

	// Mine a block on node A
	ctx := context.Background()
	blk, err := chainA.MineBlock(ctx, "miner-A", nil, 0)
	require.NoError(t, err)

	// Broadcast the mined block
	srvA.BroadcastBlock(blk, "")

	// Wait for propagation
	deadline := time.After(5 * time.Second)
	for {
		select {
		case <-deadline:
			require.FailNow(t, "timed out waiting for block propagation",
				"B height=%d, want=%d", chainB.Height(), blk.Height())
		default:
			if chainB.Height() >= blk.Height() {
				return // success
			}
			time.Sleep(100 * time.Millisecond)
		}
	}
}

func TestBlockValidation_RejectInvalidPoW(t *testing.T) {
	// Send a block with invalid PoW to a node. Verify it's rejected.
	_, chainA, _, _ := makeRelayTestNode(t, "miner-A")
	srvB, chainB, _, _ := makeRelayTestNode(t, "miner-A")

	// Get the genesis block hash from A's chain
	genesis := chainA.LatestBlock()

	// Create a block with extremely high difficulty (bits=255) so nonce=0 won't pass PoW
	fakeTxs := make([]any, 0)
	badBlock, err := block.NewBlock(genesis.Hash(), 1, 255, fakeTxs, block.Hash{})
	require.NoError(t, err)
	// Don't mine it -- PoW is invalid

	// Send the invalid block directly via raw connection
	payload := p2p.BlockPayloadFromDomain(badBlock)
	msg, err := p2p.NewMessage(p2p.CmdBlock, payload)
	require.NoError(t, err)

	conn := dialAndHandshake(t, srvB, chainA)
	defer conn.Close()

	require.NoError(t, p2p.WriteMessage(conn, msg))

	// Wait briefly and verify B's height did NOT increase
	time.Sleep(500 * time.Millisecond)
	assert.Equal(t, uint64(0), chainB.Height(), "block should have been rejected")
}

func TestSeenTracking(t *testing.T) {
	// Send same inv twice. Verify only one getdata request is sent back.
	srvA, chainA, _, _ := makeRelayTestNode(t, "miner-A")

	conn := dialAndHandshake(t, srvA, chainA)
	defer conn.Close()

	fakeBlockHash := "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	inv := p2p.InvPayload{
		Type:   "block",
		Hashes: []string{fakeBlockHash},
	}
	invMsg, err := p2p.NewMessage(p2p.CmdInv, inv)
	require.NoError(t, err)

	// Send first inv
	require.NoError(t, p2p.WriteMessage(conn, invMsg))

	// Read the getdata response
	conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	resp, err := p2p.ReadMessage(conn)
	require.NoError(t, err)
	require.Equal(t, p2p.CmdGetData, resp.Command)

	// Send second identical inv
	require.NoError(t, p2p.WriteMessage(conn, invMsg))

	// Should NOT get another getdata (already seen)
	conn.SetReadDeadline(time.Now().Add(500 * time.Millisecond))
	_, err = p2p.ReadMessage(conn)
	assert.Error(t, err, "expected no response for second inv (already seen)")
	// timeout error is expected -- means no message was sent
}
