package integration_test

import (
	"context"
	"testing"

	"github.com/baotoq/shitcoin/internal/domain/block"
	"github.com/baotoq/shitcoin/internal/domain/chain"
	"github.com/baotoq/shitcoin/internal/domain/mempool"
	"github.com/baotoq/shitcoin/internal/domain/tx"
	"github.com/baotoq/shitcoin/internal/domain/utxo"
	"github.com/baotoq/shitcoin/internal/testutil"
	"github.com/stretchr/testify/require"
)

// setupChain creates a chain with mock repos and the given config. Initializes with minerAddr.
// Returns the chain, utxoSet, and chainRepo.
func setupChain(t *testing.T, cfg chain.ChainConfig, minerAddr string) (*chain.Chain, *utxo.Set, *testutil.MockChainRepo) {
	t.Helper()

	chainRepo := testutil.NewMockChainRepo()
	utxoRepo := testutil.NewMockUTXORepo()
	utxoSet := utxo.NewSet(utxoRepo)
	pow := &block.ProofOfWork{}
	ch := chain.NewChain(chainRepo, pow, cfg, utxoSet)

	ctx := context.Background()
	err := ch.Initialize(ctx, minerAddr)
	require.NoError(t, err)

	return ch, utxoSet, chainRepo
}

func TestE2E_WalletToBalance(t *testing.T) {
	cfg := defaultCfg()
	sender := testutil.MustCreateWallet(t)

	ch, utxoSet, _ := setupChain(t, cfg, sender.Address())

	// Verify sender has exactly 1 UTXO (genesis coinbase)
	senderUTXOs, err := utxoSet.GetByAddress(sender.Address())
	require.NoError(t, err)
	require.Len(t, senderUTXOs, 1, "sender should have 1 UTXO after genesis")

	initialTxID := senderUTXOs[0].TxID()

	// Build a signed transaction spending from sender
	signedTx := testutil.MustBuildSignedTx(t, utxoSet, sender.PrivateKey(), sender.Address())

	// Mine a block containing the signed transaction
	ctx := context.Background()
	_, err = ch.MineBlock(ctx, sender.Address(), []*tx.Transaction{signedTx}, 0)
	require.NoError(t, err)

	// Verify chain height is now 1 (genesis=0, mined=1)
	require.Equal(t, uint64(1), ch.Height())

	// Verify sender's UTXOs have changed
	updatedUTXOs, err := utxoSet.GetByAddress(sender.Address())
	require.NoError(t, err)

	// After mining: genesis UTXO is spent, sender gets a new coinbase UTXO from the mined block.
	// The original genesis UTXO should no longer be present.
	require.NotEmpty(t, updatedUTXOs, "sender should still have UTXOs after mine")
	for _, u := range updatedUTXOs {
		require.NotEqual(t, initialTxID, u.TxID(), "genesis UTXO should have been spent")
	}
}

func TestE2E_MineMultipleBlocks(t *testing.T) {
	cfg := defaultCfg()
	w := testutil.MustCreateWallet(t)
	addr := w.Address()

	ch, utxoSet, _ := setupChain(t, cfg, addr)

	// Mine 5 blocks with no transactions
	ctx := context.Background()
	for i := 0; i < 5; i++ {
		_, err := ch.MineBlock(ctx, addr, nil, 0)
		require.NoError(t, err)
	}

	// Verify height is 5
	require.Equal(t, uint64(5), ch.Height())

	// Verify miner's UTXOs: 1 genesis coinbase + 5 block coinbases = 6 UTXOs
	utxos, err := utxoSet.GetByAddress(addr)
	require.NoError(t, err)
	require.Len(t, utxos, 6, "miner should have 6 UTXOs (1 genesis + 5 mined)")

	// Each UTXO should have the block reward value
	for _, u := range utxos {
		require.Equal(t, cfg.BlockReward, u.Value(), "each UTXO should equal block reward")
	}
}

func TestE2E_MempoolIntegration(t *testing.T) {
	cfg := defaultCfg()
	w := testutil.MustCreateWallet(t)
	addr := w.Address()

	ch, utxoSet, _ := setupChain(t, cfg, addr)
	pool := mempool.New(utxoSet)

	// Build a signed tx from the genesis coinbase
	signedTx := testutil.MustBuildSignedTx(t, utxoSet, w.PrivateKey(), addr)

	// Add tx to mempool
	err := pool.Add(signedTx)
	require.NoError(t, err)
	require.Equal(t, 1, pool.Count(), "mempool should have 1 transaction")

	// Get pending txs from mempool
	pendingTxs := pool.Transactions()
	require.Len(t, pendingTxs, 1)

	// Mine block with pending txs
	ctx := context.Background()
	minedBlock, err := ch.MineBlock(ctx, addr, pendingTxs, 0)
	require.NoError(t, err)
	require.NotNil(t, minedBlock)

	// Verify the mined block contains our transaction (coinbase + user tx)
	require.Len(t, minedBlock.RawTransactions(), 2, "block should have coinbase + 1 user tx")

	// Remove mined txs from mempool (simulating what the node does after mining)
	txIDs := make([]block.Hash, len(pendingTxs))
	for i, ptx := range pendingTxs {
		txIDs[i] = ptx.ID()
	}
	pool.Remove(txIDs)

	// Verify mempool is empty after removing mined transactions
	require.Equal(t, 0, pool.Count(), "mempool should be empty after mining")

	// Verify UTXO state reflects the mined transaction
	utxos, err := utxoSet.GetByAddress(addr)
	require.NoError(t, err)
	require.NotEmpty(t, utxos, "miner should have UTXOs after mining")
}
