package integration_test

import (
	"context"
	"testing"
	"time"

	"github.com/baotoq/shitcoin/internal/domain/block"
	"github.com/baotoq/shitcoin/internal/domain/chain"
	"github.com/baotoq/shitcoin/internal/domain/mempool"
	"github.com/baotoq/shitcoin/internal/domain/p2p"
	"github.com/baotoq/shitcoin/internal/domain/utxo"
	"github.com/baotoq/shitcoin/internal/testutil"
	"github.com/stretchr/testify/require"
)

// defaultCfg returns a ChainConfig suitable for integration tests.
func defaultCfg() chain.ChainConfig {
	return chain.ChainConfig{
		InitialDifficulty: 1,
		GenesisMessage:    "integration-test",
		BlockReward:       5000000000,
	}
}

// setupNode creates a fully wired node with mock repos, chain, mempool, and P2P server.
// Uses port 0 for OS-assigned ports. Registers t.Cleanup for server Stop.
// Returns the server, chain, mempool, and utxoSet.
func setupNode(t *testing.T, cfg chain.ChainConfig, minerAddr string) (*p2p.Server, *chain.Chain, *mempool.Mempool, *utxo.Set) {
	t.Helper()

	chainRepo := testutil.NewMockChainRepo()
	utxoRepo := testutil.NewMockUTXORepo()
	utxoSet := utxo.NewSet(utxoRepo)
	pow := &block.ProofOfWork{}
	ch := chain.NewChain(chainRepo, pow, cfg, utxoSet)

	ctx := context.Background()
	err := ch.Initialize(ctx, minerAddr)
	require.NoError(t, err)

	pool := mempool.New(utxoSet)
	srv := p2p.NewServer(ch, pool, utxoSet, chainRepo, 0)

	err = srv.Start(ctx)
	require.NoError(t, err)

	t.Cleanup(func() {
		srv.Stop()
	})

	return srv, ch, pool, utxoSet
}

func TestIntegration_TwoNodeHandshake(t *testing.T) {
	cfg := defaultCfg()
	w := testutil.MustCreateWallet(t)
	addr := w.Address()

	srvA, _, _, _ := setupNode(t, cfg, addr)
	srvB, _, _, _ := setupNode(t, cfg, addr)

	err := srvB.Connect(srvA.ListenAddr())
	require.NoError(t, err)

	require.Eventually(t, func() bool {
		return srvA.PeerCount() == 1
	}, 5*time.Second, 50*time.Millisecond, "node A should have 1 peer")

	require.Eventually(t, func() bool {
		return srvB.PeerCount() == 1
	}, 5*time.Second, 50*time.Millisecond, "node B should have 1 peer")
}

func TestIntegration_BlockSync(t *testing.T) {
	cfg := defaultCfg()
	w := testutil.MustCreateWallet(t)
	addr := w.Address()

	// Node A: initialize + mine 2 additional blocks (height 0 -> 2)
	srvA, chainA, _, _ := setupNode(t, cfg, addr)

	ctx := context.Background()
	_, err := chainA.MineBlock(ctx, addr, nil, 0)
	require.NoError(t, err)
	_, err = chainA.MineBlock(ctx, addr, nil, 0)
	require.NoError(t, err)
	require.Equal(t, uint64(2), chainA.Height())

	// Node B: initialize only (height 0)
	srvB, chainB, _, _ := setupNode(t, cfg, addr)

	// Connect B to A -- triggers IBD since A has a longer chain
	err = srvB.Connect(srvA.ListenAddr())
	require.NoError(t, err)

	// Wait for B to sync to A's height
	require.Eventually(t, func() bool {
		return chainB.Height() == chainA.Height()
	}, 5*time.Second, 50*time.Millisecond, "node B should sync to node A's height")
}

func TestIntegration_TxRelay(t *testing.T) {
	cfg := defaultCfg()
	w := testutil.MustCreateWallet(t)
	addr := w.Address()

	srvA, _, poolA, utxoSetA := setupNode(t, cfg, addr)
	srvB, _, poolB, _ := setupNode(t, cfg, addr)

	// Connect and wait for handshake
	err := srvB.Connect(srvA.ListenAddr())
	require.NoError(t, err)

	require.Eventually(t, func() bool {
		return srvA.PeerCount() == 1 && srvB.PeerCount() == 1
	}, 5*time.Second, 50*time.Millisecond, "both nodes should be connected")

	// Build a signed transaction from node A's UTXOs
	signedTx := testutil.MustBuildSignedTx(t, utxoSetA, w.PrivateKey(), addr)

	// Add to node A's mempool and broadcast
	err = poolA.Add(signedTx)
	require.NoError(t, err)
	srvA.BroadcastTx(signedTx, "")

	// Wait for node B's mempool to receive the transaction
	require.Eventually(t, func() bool {
		return poolB.Count() == 1
	}, 5*time.Second, 50*time.Millisecond, "node B should have 1 tx in mempool")
}
