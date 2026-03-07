package p2p_test

import (
	"context"
	"net"
	"strconv"
	"testing"
	"time"

	"github.com/baotoq/shitcoin/internal/domain/block"
	"github.com/baotoq/shitcoin/internal/domain/chain"
	"github.com/baotoq/shitcoin/internal/domain/mempool"
	"github.com/baotoq/shitcoin/internal/domain/p2p"
	"github.com/baotoq/shitcoin/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// makeTestServer creates a test P2P server with an initialized in-memory chain.
// Uses the given miner address to produce a unique genesis block.
func makeTestServer(t *testing.T, minerAddr string) (*p2p.Server, int) {
	t.Helper()

	repo := testutil.NewMockChainRepo()

	pow := &block.ProofOfWork{}
	cfg := chain.ChainConfig{
		InitialDifficulty: 1, // very low for fast test mining
		GenesisMessage:    "test genesis",
		BlockReward:       5000000000,
	}
	ch := chain.NewChain(repo, pow, cfg, nil)

	ctx := context.Background()
	require.NoError(t, ch.Initialize(ctx, minerAddr))

	pool := mempool.New(nil)

	// Use port 0 for OS-assigned port
	srv := p2p.NewServer(ch, pool, nil, repo, 0)

	require.NoError(t, srv.Start(ctx))

	t.Cleanup(func() {
		srv.Stop()
	})

	// Extract assigned port
	_, portStr, _ := net.SplitHostPort(srv.ListenAddr())
	port, _ := strconv.Atoi(portStr)

	return srv, port
}

func TestServerListen(t *testing.T) {
	srv, port := makeTestServer(t, "miner-addr-listen")

	require.NotZero(t, port)

	// Verify we can connect to the listening port
	conn, err := net.DialTimeout("tcp", srv.ListenAddr(), 2*time.Second)
	require.NoError(t, err)
	conn.Close()
}

func TestHandshake(t *testing.T) {
	// Both servers use the same miner address to ensure matching genesis hashes
	srvA, _ := makeTestServer(t, "same-miner-addr")
	srvB, _ := makeTestServer(t, "same-miner-addr")

	require.NoError(t, srvB.Connect(srvA.ListenAddr()))

	// Wait briefly for inbound handling
	time.Sleep(200 * time.Millisecond)

	assert.Equal(t, 1, srvA.PeerCount())
	assert.Equal(t, 1, srvB.PeerCount())
}

func TestHandshakeGenesisMismatch(t *testing.T) {
	// Different miner addresses produce different coinbase TXs -> different merkle roots -> different genesis hashes
	srvA, _ := makeTestServer(t, "miner-addr-alpha")
	srvB, _ := makeTestServer(t, "miner-addr-beta")

	err := srvB.Connect(srvA.ListenAddr())
	require.Error(t, err, "expected handshake to fail due to genesis mismatch")

	// Wait briefly
	time.Sleep(200 * time.Millisecond)

	assert.Equal(t, 0, srvA.PeerCount())
	assert.Equal(t, 0, srvB.PeerCount())
}
