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
	"github.com/baotoq/shitcoin/internal/domain/utxo"
)

// mockChainRepo implements chain.Repository with an in-memory genesis block.
type mockChainRepo struct {
	genesis *block.Block
}

func (m *mockChainRepo) SaveBlock(_ context.Context, b *block.Block) error {
	if b.Height() == 0 {
		m.genesis = b
	}
	return nil
}

func (m *mockChainRepo) SaveBlockWithUTXOs(_ context.Context, b *block.Block, _ *utxo.UndoEntry) error {
	if b.Height() == 0 {
		m.genesis = b
	}
	return nil
}

func (m *mockChainRepo) GetBlock(_ context.Context, hash block.Hash) (*block.Block, error) {
	if m.genesis != nil && m.genesis.Hash() == hash {
		return m.genesis, nil
	}
	return nil, chain.ErrChainEmpty
}

func (m *mockChainRepo) GetBlockByHeight(_ context.Context, height uint64) (*block.Block, error) {
	if height == 0 && m.genesis != nil {
		return m.genesis, nil
	}
	return nil, chain.ErrChainEmpty
}

func (m *mockChainRepo) GetLatestBlock(_ context.Context) (*block.Block, error) {
	if m.genesis != nil {
		return m.genesis, nil
	}
	return nil, chain.ErrChainEmpty
}

func (m *mockChainRepo) GetChainHeight(_ context.Context) (uint64, error) {
	if m.genesis != nil {
		return 0, nil
	}
	return 0, chain.ErrChainEmpty
}

func (m *mockChainRepo) GetBlocksInRange(_ context.Context, _, _ uint64) ([]*block.Block, error) {
	if m.genesis != nil {
		return []*block.Block{m.genesis}, nil
	}
	return nil, nil
}

// makeTestServer creates a test P2P server with an initialized in-memory chain.
// Uses the given miner address to produce a unique genesis block.
func makeTestServer(t *testing.T, minerAddr string) (*p2p.Server, int) {
	t.Helper()

	repo := &mockChainRepo{}
	pow := &block.ProofOfWork{}
	cfg := chain.ChainConfig{
		InitialDifficulty: 1, // very low for fast test mining
		GenesisMessage:    "test genesis",
		BlockReward:       5000000000,
	}
	ch := chain.NewChain(repo, pow, cfg, nil)

	ctx := context.Background()
	if err := ch.Initialize(ctx, minerAddr); err != nil {
		t.Fatalf("chain.Initialize failed: %v", err)
	}

	pool := mempool.New(nil)

	// Use port 0 for OS-assigned port
	srv := p2p.NewServer(ch, pool, nil, repo, 0)

	if err := srv.Start(ctx); err != nil {
		t.Fatalf("server.Start failed: %v", err)
	}

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

	if port == 0 {
		t.Fatal("expected non-zero port")
	}

	// Verify we can connect to the listening port
	conn, err := net.DialTimeout("tcp", srv.ListenAddr(), 2*time.Second)
	if err != nil {
		t.Fatalf("failed to connect to server: %v", err)
	}
	conn.Close()
}

func TestHandshake(t *testing.T) {
	// Both servers use the same miner address to ensure matching genesis hashes
	srvA, _ := makeTestServer(t, "same-miner-addr")
	srvB, _ := makeTestServer(t, "same-miner-addr")

	if err := srvB.Connect(srvA.ListenAddr()); err != nil {
		t.Fatalf("Connect failed: %v", err)
	}

	// Wait briefly for inbound handling
	time.Sleep(200 * time.Millisecond)

	if srvA.PeerCount() != 1 {
		t.Errorf("srvA.PeerCount() = %d; want 1", srvA.PeerCount())
	}
	if srvB.PeerCount() != 1 {
		t.Errorf("srvB.PeerCount() = %d; want 1", srvB.PeerCount())
	}
}

func TestHandshakeGenesisMismatch(t *testing.T) {
	// Different miner addresses produce different coinbase TXs -> different merkle roots -> different genesis hashes
	srvA, _ := makeTestServer(t, "miner-addr-alpha")
	srvB, _ := makeTestServer(t, "miner-addr-beta")

	err := srvB.Connect(srvA.ListenAddr())
	if err == nil {
		t.Fatal("expected handshake to fail due to genesis mismatch")
	}

	// Wait briefly
	time.Sleep(200 * time.Millisecond)

	if srvA.PeerCount() != 0 {
		t.Errorf("srvA.PeerCount() = %d; want 0", srvA.PeerCount())
	}
	if srvB.PeerCount() != 0 {
		t.Errorf("srvB.PeerCount() = %d; want 0", srvB.PeerCount())
	}
}
