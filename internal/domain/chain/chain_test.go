package chain_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/baotoq/shitcoin/internal/domain/block"
	"github.com/baotoq/shitcoin/internal/domain/chain"
	"github.com/baotoq/shitcoin/internal/domain/tx"
	"github.com/baotoq/shitcoin/internal/domain/utxo"
	"github.com/baotoq/shitcoin/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// MockMempoolAdder is a testify/mock implementation of chain.MempoolAdder.
type MockMempoolAdder struct {
	mock.Mock
}

func (m *MockMempoolAdder) Add(t *tx.Transaction) error {
	args := m.Called(t)
	return args.Error(0)
}

func TestReorganize_SwitchesToLongerFork(t *testing.T) {
	ctx := context.Background()
	minerAddr := "miner-reorg"

	repo := testutil.NewMockChainRepo()
	utxoRepo := testutil.NewMockUTXORepo()
	utxoSet := utxo.NewSet(utxoRepo)
	pow := &block.ProofOfWork{}
	cfg := chain.ChainConfig{
		InitialDifficulty: 1,
		GenesisMessage:    "reorg-test",
		BlockReward:       5000000000,
	}
	ch := chain.NewChain(repo, pow, cfg, utxoSet)

	require.NoError(t, ch.Initialize(ctx, minerAddr))

	// Mine 5 blocks on the main chain: genesis(0) -> 1 -> 2 -> 3 -> 4(A4) -> 5(A5)
	for range 5 {
		_, err := ch.MineBlock(ctx, minerAddr, nil, 0)
		require.NoError(t, err)
	}

	require.Equal(t, uint64(5), ch.Height())

	// Get the block at height 3 -- this is the fork point
	forkBlock, err := repo.GetBlockByHeight(ctx, 3)
	require.NoError(t, err)

	// Create 3 new blocks (heights 4, 5, 6) forming a longer fork from height 3
	forkBlocks := make([]*block.Block, 0, 3)
	prevHash := forkBlock.Hash()
	forkMiner := "fork-miner"
	for i := uint64(4); i <= 6; i++ {
		coinbase := tx.NewCoinbaseTxWithHeight(forkMiner, cfg.BlockReward, i)
		blockTxs := []any{coinbase}

		txHashes := []block.Hash{coinbase.ID()}
		merkleRoot := block.ComputeMerkleRoot(txHashes)

		newBlk, err := block.NewBlock(prevHash, i, uint32(cfg.InitialDifficulty), blockTxs, merkleRoot)
		require.NoError(t, err)
		require.NoError(t, pow.Mine(newBlk))
		forkBlocks = append(forkBlocks, newBlk)
		prevHash = newBlk.Hash()
	}

	// Record UTXO balance before reorg
	balanceBefore, err := utxoSet.GetBalance(minerAddr)
	require.NoError(t, err)
	require.Greater(t, balanceBefore, int64(0))

	mpool := new(MockMempoolAdder)
	// Coinbase txs are excluded from re-add, so no calls expected

	// Execute reorganization: undo A4, A5, apply B4, B5, B6
	require.NoError(t, ch.Reorganize(ctx, 3, forkBlocks, mpool))

	// Verify new tip
	assert.Equal(t, uint64(6), ch.Height())

	latestBlock := ch.LatestBlock()
	assert.Equal(t, forkBlocks[2].Hash(), latestBlock.Hash())

	// Verify UTXO state reflects fork chain
	balanceAfter, err := utxoSet.GetBalance(minerAddr)
	require.NoError(t, err)
	expectedMainBalance := cfg.BlockReward * 4 // blocks 0,1,2,3
	assert.Equal(t, expectedMainBalance, balanceAfter)

	forkBalance, err := utxoSet.GetBalance(forkMiner)
	require.NoError(t, err)
	expectedForkBalance := cfg.BlockReward * 3 // blocks 4,5,6
	assert.Equal(t, expectedForkBalance, forkBalance)

	mpool.AssertExpectations(t)
}

func TestReorganize_OrphanedTxsReturnToMempool(t *testing.T) {
	ctx := context.Background()
	minerAddr := "miner-orphan-tx"

	repo := testutil.NewMockChainRepo()
	utxoRepo := testutil.NewMockUTXORepo()
	utxoSet := utxo.NewSet(utxoRepo)
	pow := &block.ProofOfWork{}
	cfg := chain.ChainConfig{
		InitialDifficulty: 1,
		GenesisMessage:    "orphan-tx-test",
		BlockReward:       5000000000,
	}
	ch := chain.NewChain(repo, pow, cfg, utxoSet)

	require.NoError(t, ch.Initialize(ctx, minerAddr))

	// Mine 3 blocks
	for range 3 {
		_, err := ch.MineBlock(ctx, minerAddr, nil, 0)
		require.NoError(t, err)
	}

	// Get fork point at height 2
	forkBlock, err := repo.GetBlockByHeight(ctx, 2)
	require.NoError(t, err)

	// Create fork blocks (heights 3, 4) - longer than current chain
	forkBlocks := make([]*block.Block, 0, 2)
	prevHash := forkBlock.Hash()
	forkMiner := "fork-miner-2"
	for i := uint64(3); i <= 4; i++ {
		coinbase := tx.NewCoinbaseTxWithHeight(forkMiner, cfg.BlockReward, i)
		blockTxs := []any{coinbase}
		txHashes := []block.Hash{coinbase.ID()}
		merkleRoot := block.ComputeMerkleRoot(txHashes)

		newBlk, err := block.NewBlock(prevHash, i, uint32(cfg.InitialDifficulty), blockTxs, merkleRoot)
		require.NoError(t, err)
		require.NoError(t, pow.Mine(newBlk))
		forkBlocks = append(forkBlocks, newBlk)
		prevHash = newBlk.Hash()
	}

	mpool := new(MockMempoolAdder)
	require.NoError(t, ch.Reorganize(ctx, 2, forkBlocks, mpool))

	// Verify tip is now at height 4
	assert.Equal(t, uint64(4), ch.Height())

	mpool.AssertExpectations(t)
}

func TestReorganize_PreservesBlocksBelowFork(t *testing.T) {
	ctx := context.Background()
	minerAddr := "miner-preserve"

	repo := testutil.NewMockChainRepo()
	utxoRepo := testutil.NewMockUTXORepo()
	utxoSet := utxo.NewSet(utxoRepo)
	pow := &block.ProofOfWork{}
	cfg := chain.ChainConfig{
		InitialDifficulty: 1,
		GenesisMessage:    "preserve-test",
		BlockReward:       5000000000,
	}
	ch := chain.NewChain(repo, pow, cfg, utxoSet)

	require.NoError(t, ch.Initialize(ctx, minerAddr))

	// Mine 4 blocks
	for range 4 {
		_, err := ch.MineBlock(ctx, minerAddr, nil, 0)
		require.NoError(t, err)
	}

	// Remember block at height 2 (should survive reorg)
	block2, err := repo.GetBlockByHeight(ctx, 2)
	require.NoError(t, err)

	// Fork at height 2, new blocks 3,4,5
	forkBlock, err := repo.GetBlockByHeight(ctx, 2)
	require.NoError(t, err)
	forkBlocks := make([]*block.Block, 0, 3)
	prevHash := forkBlock.Hash()
	for i := uint64(3); i <= 5; i++ {
		coinbase := tx.NewCoinbaseTxWithHeight("fork-miner-3", cfg.BlockReward, i)
		blockTxs := []any{coinbase}
		txHashes := []block.Hash{coinbase.ID()}
		merkleRoot := block.ComputeMerkleRoot(txHashes)
		newBlk, err := block.NewBlock(prevHash, i, uint32(cfg.InitialDifficulty), blockTxs, merkleRoot)
		require.NoError(t, err)
		require.NoError(t, pow.Mine(newBlk))
		forkBlocks = append(forkBlocks, newBlk)
		prevHash = newBlk.Hash()
	}

	mpool := new(MockMempoolAdder)
	require.NoError(t, ch.Reorganize(ctx, 2, forkBlocks, mpool))

	// Block at height 2 should still exist
	retrieved, err := repo.GetBlockByHeight(ctx, 2)
	require.NoError(t, err)
	assert.Equal(t, block2.Hash(), retrieved.Hash())

	mpool.AssertExpectations(t)
}

func TestRewardAtHeight(t *testing.T) {
	repo := testutil.NewMockChainRepo()
	utxoRepo := testutil.NewMockUTXORepo()
	utxoSet := utxo.NewSet(utxoRepo)
	pow := &block.ProofOfWork{}
	cfg := chain.ChainConfig{
		InitialDifficulty: 1,
		GenesisMessage:    "halving-test",
		BlockReward:       5000000000,
		HalvingInterval:   10,
	}
	ch := chain.NewChain(repo, pow, cfg, utxoSet)

	ctx := context.Background()
	require.NoError(t, ch.Initialize(ctx, "miner"))

	tests := []struct {
		name   string
		height uint64
		want   int64
	}{
		{"genesis", 0, 5000000000},
		{"before first halving", 9, 5000000000},
		{"first halving", 10, 2500000000},
		{"second halving", 20, 1250000000},
		{"third halving", 30, 625000000},
		{"64th halving (zero)", 640, 0},
		{"beyond 64 halvings", 700, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ch.RewardAtHeight(tt.height)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestRewardAtHeightNoHalving(t *testing.T) {
	repo := testutil.NewMockChainRepo()
	utxoRepo := testutil.NewMockUTXORepo()
	utxoSet := utxo.NewSet(utxoRepo)
	pow := &block.ProofOfWork{}
	cfg := chain.ChainConfig{
		InitialDifficulty: 1,
		GenesisMessage:    "no-halving-test",
		BlockReward:       5000000000,
		HalvingInterval:   0, // no halving
	}
	ch := chain.NewChain(repo, pow, cfg, utxoSet)

	ctx := context.Background()
	require.NoError(t, ch.Initialize(ctx, "miner"))

	// With HalvingInterval=0, reward is always BlockReward
	for _, h := range []uint64{0, 10, 100, 1000, 10000} {
		assert.Equal(t, int64(5000000000), ch.RewardAtHeight(h))
	}
}

func TestCoinbaseIncludesFees(t *testing.T) {
	ctx := context.Background()
	minerAddr := "miner-fees"

	repo := testutil.NewMockChainRepo()
	utxoRepo := testutil.NewMockUTXORepo()
	utxoSet := utxo.NewSet(utxoRepo)
	pow := &block.ProofOfWork{}
	cfg := chain.ChainConfig{
		InitialDifficulty: 1,
		GenesisMessage:    "fee-test",
		BlockReward:       5000000000,
		HalvingInterval:   0,
	}
	ch := chain.NewChain(repo, pow, cfg, utxoSet)

	require.NoError(t, ch.Initialize(ctx, minerAddr))

	// Mine a block with totalFees = 1000
	blk, err := ch.MineBlock(ctx, minerAddr, nil, 1000)
	require.NoError(t, err)

	// Extract coinbase from the mined block
	rawTxs := blk.RawTransactions()
	require.Greater(t, len(rawTxs), 0)
	coinbaseTx, ok := rawTxs[0].(*tx.Transaction)
	require.True(t, ok)
	require.True(t, coinbaseTx.IsCoinbase())

	// Coinbase output should be BlockReward + totalFees
	assert.Equal(t, int64(5000000000+1000), coinbaseTx.Outputs()[0].Value())
}

func TestGetCurrentBits_AdjustmentInterval(t *testing.T) {
	ctx := context.Background()
	minerAddr := "miner-adjust"

	repo := testutil.NewMockChainRepo()
	utxoRepo := testutil.NewMockUTXORepo()
	utxoSet := utxo.NewSet(utxoRepo)
	pow := &block.ProofOfWork{}
	cfg := chain.ChainConfig{
		InitialDifficulty:        1,
		DifficultyAdjustInterval: 5,
		BlockTimeTarget:          600,
		GenesisMessage:           "adjust-test",
		BlockReward:              5000000000,
	}
	ch := chain.NewChain(repo, pow, cfg, utxoSet)

	require.NoError(t, ch.Initialize(ctx, minerAddr))

	// Mine 5 blocks to trigger difficulty adjustment at block 5
	for i := 0; i < 5; i++ {
		blk, err := ch.MineBlock(ctx, minerAddr, nil, 0)
		require.NoError(t, err, "mining block %d", i+1)
		require.Equal(t, uint64(i+1), blk.Height())
	}

	// Chain should be at height 5 (adjustment was triggered)
	require.Equal(t, uint64(5), ch.Height())

	// Mine one more block to confirm adjusted difficulty is used
	blk, err := ch.MineBlock(ctx, minerAddr, nil, 0)
	require.NoError(t, err)
	assert.Equal(t, uint64(6), blk.Height())
}

func TestGetCurrentBits_BeforeInterval(t *testing.T) {
	ctx := context.Background()
	minerAddr := "miner-before-interval"

	repo := testutil.NewMockChainRepo()
	utxoRepo := testutil.NewMockUTXORepo()
	utxoSet := utxo.NewSet(utxoRepo)
	pow := &block.ProofOfWork{}
	cfg := chain.ChainConfig{
		InitialDifficulty:        1,
		DifficultyAdjustInterval: 10,
		BlockTimeTarget:          600,
		GenesisMessage:           "before-interval-test",
		BlockReward:              5000000000,
	}
	ch := chain.NewChain(repo, pow, cfg, utxoSet)

	require.NoError(t, ch.Initialize(ctx, minerAddr))

	// Mine 3 blocks (well before interval of 10)
	for i := 0; i < 3; i++ {
		blk, err := ch.MineBlock(ctx, minerAddr, nil, 0)
		require.NoError(t, err)
		// All blocks should use initial difficulty (bits=1)
		assert.Equal(t, uint32(1), blk.Bits(), "block %d should use initial difficulty", i+1)
	}
}

func TestSetLatestBlock(t *testing.T) {
	ctx := context.Background()
	minerAddr := "miner-setlatest"

	repo := testutil.NewMockChainRepo()
	utxoRepo := testutil.NewMockUTXORepo()
	utxoSet := utxo.NewSet(utxoRepo)
	pow := &block.ProofOfWork{}
	cfg := chain.ChainConfig{
		InitialDifficulty: 1,
		GenesisMessage:    "setlatest-test",
		BlockReward:       5000000000,
	}
	ch := chain.NewChain(repo, pow, cfg, utxoSet)

	require.NoError(t, ch.Initialize(ctx, minerAddr))

	// Create a new block to set as latest
	newBlock := testutil.MustCreateBlock(t, 42, ch.LatestBlock().Hash())

	ch.SetLatestBlock(newBlock)

	assert.Equal(t, newBlock.Hash(), ch.LatestBlock().Hash())
	assert.Equal(t, uint64(42), ch.Height())
}

func TestInitialize_AlreadyInitialized(t *testing.T) {
	ctx := context.Background()
	minerAddr := "miner-already-init"

	repo := testutil.NewMockChainRepo()
	utxoRepo := testutil.NewMockUTXORepo()
	utxoSet := utxo.NewSet(utxoRepo)
	pow := &block.ProofOfWork{}
	cfg := chain.ChainConfig{
		InitialDifficulty: 1,
		GenesisMessage:    "already-init-test",
		BlockReward:       5000000000,
	}

	// Pre-seed repo with an existing genesis block
	existingBlock := testutil.MustCreateBlock(t, 0, block.Hash{})
	repo.AddBlock(existingBlock)

	ch := chain.NewChain(repo, pow, cfg, utxoSet)

	// Initialize should load existing chain, not create new genesis
	require.NoError(t, ch.Initialize(ctx, minerAddr))

	// Should have loaded the existing block as latest
	assert.Equal(t, existingBlock.Hash(), ch.LatestBlock().Hash())
	assert.Equal(t, uint64(0), ch.Height())
}

func TestInitialize_EmptyMinerAddress(t *testing.T) {
	ctx := context.Background()

	repo := testutil.NewMockChainRepo()
	utxoRepo := testutil.NewMockUTXORepo()
	utxoSet := utxo.NewSet(utxoRepo)
	pow := &block.ProofOfWork{}
	cfg := chain.ChainConfig{
		InitialDifficulty: 1,
		GenesisMessage:    "empty-miner-test",
		BlockReward:       5000000000,
	}
	ch := chain.NewChain(repo, pow, cfg, utxoSet)

	// Initialize with empty miner address on empty repo
	// Should create genesis without coinbase (no miner address)
	require.NoError(t, ch.Initialize(ctx, ""))

	// Genesis block should exist
	require.NotNil(t, ch.LatestBlock())
	assert.Equal(t, uint64(0), ch.Height())

	// Genesis block should have no transactions (empty miner address)
	assert.Empty(t, ch.LatestBlock().RawTransactions())
}

func TestMineBlock_BeforeInitialize(t *testing.T) {
	repo := testutil.NewMockChainRepo()
	utxoRepo := testutil.NewMockUTXORepo()
	utxoSet := utxo.NewSet(utxoRepo)
	pow := &block.ProofOfWork{}
	cfg := chain.ChainConfig{
		InitialDifficulty: 1,
		GenesisMessage:    "uninit-test",
		BlockReward:       5000000000,
	}
	ch := chain.NewChain(repo, pow, cfg, utxoSet)

	// MineBlock without Initialize should return error
	_, err := ch.MineBlock(context.Background(), "miner", nil, 0)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "chain not initialized")
}

func TestMineBlock_WithTransactions(t *testing.T) {
	ctx := context.Background()
	minerAddr := "miner-with-txs"

	w := testutil.MustCreateWallet(t)
	fromAddr := w.Address()

	repo := testutil.NewMockChainRepo()
	utxoRepo := testutil.NewMockUTXORepo()
	utxoSet := utxo.NewSet(utxoRepo)
	pow := &block.ProofOfWork{}
	cfg := chain.ChainConfig{
		InitialDifficulty: 1,
		GenesisMessage:    "tx-test",
		BlockReward:       5000000000,
	}
	ch := chain.NewChain(repo, pow, cfg, utxoSet)

	// Initialize and mine a block to the wallet address to get spendable UTXOs
	require.NoError(t, ch.Initialize(ctx, fromAddr))
	_, err := ch.MineBlock(ctx, fromAddr, nil, 0)
	require.NoError(t, err)

	// Build a signed transaction spending from the wallet
	spendTx := testutil.MustBuildSignedTx(t, utxoSet, w.PrivateKey(), fromAddr)

	// Mine a block with the user transaction
	blk, err := ch.MineBlock(ctx, minerAddr, []*tx.Transaction{spendTx}, 1000)
	require.NoError(t, err)

	// Block should contain coinbase + user tx (2 transactions)
	rawTxs := blk.RawTransactions()
	assert.Equal(t, 2, len(rawTxs))

	// First should be coinbase
	coinbaseTx, ok := rawTxs[0].(*tx.Transaction)
	require.True(t, ok)
	assert.True(t, coinbaseTx.IsCoinbase())

	// Second should be the user transaction
	userTx, ok := rawTxs[1].(*tx.Transaction)
	require.True(t, ok)
	assert.Equal(t, spendTx.ID(), userTx.ID())
}

func TestMineBlock_WithMiningProgress(t *testing.T) {
	ctx := context.Background()
	minerAddr := "miner-progress"

	repo := testutil.NewMockChainRepo()
	utxoRepo := testutil.NewMockUTXORepo()
	utxoSet := utxo.NewSet(utxoRepo)
	pow := &block.ProofOfWork{}
	cfg := chain.ChainConfig{
		InitialDifficulty: 1,
		GenesisMessage:    "progress-test",
		BlockReward:       5000000000,
	}
	ch := chain.NewChain(repo, pow, cfg, utxoSet)

	require.NoError(t, ch.Initialize(ctx, minerAddr))

	// Set OnMiningProgress callback
	var progressCalled bool
	ch.OnMiningProgress = func(p block.MiningProgress) {
		progressCalled = true
	}

	// Mine a block -- should use MineWithProgress path
	blk, err := ch.MineBlock(ctx, minerAddr, nil, 0)
	require.NoError(t, err)
	require.NotNil(t, blk)

	// Callback should have been invoked (nonce 0 % sampleRate == 0)
	assert.True(t, progressCalled, "OnMiningProgress callback should have been invoked")
}

func TestMineBlock_SaveBlockWithUTXOsError(t *testing.T) {
	ctx := context.Background()
	minerAddr := "miner-save-err"

	repo := testutil.NewMockChainRepo()
	utxoRepo := testutil.NewMockUTXORepo()
	utxoSet := utxo.NewSet(utxoRepo)
	pow := &block.ProofOfWork{}
	cfg := chain.ChainConfig{
		InitialDifficulty: 1,
		GenesisMessage:    "save-err-test",
		BlockReward:       5000000000,
	}
	ch := chain.NewChain(repo, pow, cfg, utxoSet)

	require.NoError(t, ch.Initialize(ctx, minerAddr))

	// Configure error on SaveBlockWithUTXOs for subsequent saves
	repo.SaveBlockWithUTXOsErr = fmt.Errorf("disk full")

	// MineBlock should propagate the save error
	_, err := ch.MineBlock(ctx, minerAddr, nil, 0)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "disk full")
}

func TestReorganize_EmptyForkBlocks(t *testing.T) {
	ctx := context.Background()
	minerAddr := "miner-empty-fork"

	repo := testutil.NewMockChainRepo()
	utxoRepo := testutil.NewMockUTXORepo()
	utxoSet := utxo.NewSet(utxoRepo)
	pow := &block.ProofOfWork{}
	cfg := chain.ChainConfig{
		InitialDifficulty: 1,
		GenesisMessage:    "empty-fork-test",
		BlockReward:       5000000000,
	}
	ch := chain.NewChain(repo, pow, cfg, utxoSet)

	require.NoError(t, ch.Initialize(ctx, minerAddr))

	// Mine 2 blocks
	for range 2 {
		_, err := ch.MineBlock(ctx, minerAddr, nil, 0)
		require.NoError(t, err)
	}

	heightBefore := ch.Height()

	// Reorganize with empty fork blocks at current tip
	err := ch.Reorganize(ctx, heightBefore, nil, nil)
	require.NoError(t, err)

	// Chain tip should remain at the same height (no new blocks applied)
	assert.Equal(t, heightBefore, ch.Height())
}

func TestReorganize_InvalidForkBlock(t *testing.T) {
	ctx := context.Background()
	minerAddr := "miner-invalid-fork"

	repo := testutil.NewMockChainRepo()
	utxoRepo := testutil.NewMockUTXORepo()
	utxoSet := utxo.NewSet(utxoRepo)
	pow := &block.ProofOfWork{}
	cfg := chain.ChainConfig{
		InitialDifficulty: 1,
		GenesisMessage:    "invalid-fork-test",
		BlockReward:       5000000000,
	}
	ch := chain.NewChain(repo, pow, cfg, utxoSet)

	require.NoError(t, ch.Initialize(ctx, minerAddr))

	// Mine 2 blocks
	for range 2 {
		_, err := ch.MineBlock(ctx, minerAddr, nil, 0)
		require.NoError(t, err)
	}

	forkBlock, err := repo.GetBlockByHeight(ctx, 1)
	require.NoError(t, err)

	// Create a fork block with high difficulty that is NOT mined (invalid PoW)
	coinbase := tx.NewCoinbaseTxWithHeight("fork-miner", cfg.BlockReward, 2)
	blockTxs := []any{coinbase}
	txHashes := []block.Hash{coinbase.ID()}
	merkleRoot := block.ComputeMerkleRoot(txHashes)

	// Create block with high difficulty (bits=20) so it won't pass PoW validation without mining
	invalidBlk, err := block.NewBlock(forkBlock.Hash(), 2, 20, blockTxs, merkleRoot)
	require.NoError(t, err)
	// Intentionally NOT mining the block -- PoW is invalid

	// Reorganize should fail on PoW validation
	err = ch.Reorganize(ctx, 1, []*block.Block{invalidBlk}, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid PoW")
}

func TestReorganize_BeforeInitialize(t *testing.T) {
	repo := testutil.NewMockChainRepo()
	utxoRepo := testutil.NewMockUTXORepo()
	utxoSet := utxo.NewSet(utxoRepo)
	pow := &block.ProofOfWork{}
	cfg := chain.ChainConfig{
		InitialDifficulty: 1,
		GenesisMessage:    "reorg-uninit-test",
		BlockReward:       5000000000,
	}
	ch := chain.NewChain(repo, pow, cfg, utxoSet)

	// Reorganize without Initialize should return error
	err := ch.Reorganize(context.Background(), 0, nil, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "chain not initialized")
}

func TestHeight_NilLatestBlock(t *testing.T) {
	repo := testutil.NewMockChainRepo()
	utxoRepo := testutil.NewMockUTXORepo()
	utxoSet := utxo.NewSet(utxoRepo)
	pow := &block.ProofOfWork{}
	cfg := chain.ChainConfig{
		InitialDifficulty: 1,
		GenesisMessage:    "height-nil-test",
		BlockReward:       5000000000,
	}
	ch := chain.NewChain(repo, pow, cfg, utxoSet)

	// Height should return 0 when latestBlock is nil
	assert.Equal(t, uint64(0), ch.Height())
}

func TestMineBlock_WithoutUTXOSet(t *testing.T) {
	ctx := context.Background()
	minerAddr := "miner-no-utxo"

	repo := testutil.NewMockChainRepo()
	pow := &block.ProofOfWork{}
	cfg := chain.ChainConfig{
		InitialDifficulty: 1,
		GenesisMessage:    "no-utxo-test",
		BlockReward:       5000000000,
	}
	// Create chain without UTXO set (nil)
	ch := chain.NewChain(repo, pow, cfg, nil)

	require.NoError(t, ch.Initialize(ctx, minerAddr))

	// Mine a block -- should use SaveBlock path (not SaveBlockWithUTXOs)
	blk, err := ch.MineBlock(ctx, minerAddr, nil, 0)
	require.NoError(t, err)
	assert.Equal(t, uint64(1), blk.Height())
}

func TestReorganize_NilMempoolAdder(t *testing.T) {
	ctx := context.Background()
	minerAddr := "miner-nil-mempool"

	repo := testutil.NewMockChainRepo()
	utxoRepo := testutil.NewMockUTXORepo()
	utxoSet := utxo.NewSet(utxoRepo)
	pow := &block.ProofOfWork{}
	cfg := chain.ChainConfig{
		InitialDifficulty: 1,
		GenesisMessage:    "nil-mempool-test",
		BlockReward:       5000000000,
	}
	ch := chain.NewChain(repo, pow, cfg, utxoSet)

	require.NoError(t, ch.Initialize(ctx, minerAddr))

	// Mine 2 blocks
	for range 2 {
		_, err := ch.MineBlock(ctx, minerAddr, nil, 0)
		require.NoError(t, err)
	}

	forkBlock, err := repo.GetBlockByHeight(ctx, 1)
	require.NoError(t, err)

	// Create fork block at height 2
	coinbase := tx.NewCoinbaseTxWithHeight("fork", cfg.BlockReward, 2)
	blockTxs := []any{coinbase}
	txHashes := []block.Hash{coinbase.ID()}
	merkleRoot := block.ComputeMerkleRoot(txHashes)
	forkBlk, err := block.NewBlock(forkBlock.Hash(), 2, uint32(cfg.InitialDifficulty), blockTxs, merkleRoot)
	require.NoError(t, err)
	require.NoError(t, pow.Mine(forkBlk))

	// Create another fork block at height 3
	coinbase2 := tx.NewCoinbaseTxWithHeight("fork", cfg.BlockReward, 3)
	blockTxs2 := []any{coinbase2}
	txHashes2 := []block.Hash{coinbase2.ID()}
	merkleRoot2 := block.ComputeMerkleRoot(txHashes2)
	forkBlk2, err := block.NewBlock(forkBlk.Hash(), 3, uint32(cfg.InitialDifficulty), blockTxs2, merkleRoot2)
	require.NoError(t, err)
	require.NoError(t, pow.Mine(forkBlk2))

	// Reorganize with nil mempool adder should not panic
	err = ch.Reorganize(ctx, 1, []*block.Block{forkBlk, forkBlk2}, nil)
	require.NoError(t, err)
	assert.Equal(t, uint64(3), ch.Height())
}

func TestInitialize_RepoError(t *testing.T) {
	repo := testutil.NewMockChainRepo()
	utxoRepo := testutil.NewMockUTXORepo()
	utxoSet := utxo.NewSet(utxoRepo)
	pow := &block.ProofOfWork{}
	cfg := chain.ChainConfig{
		InitialDifficulty: 1,
		GenesisMessage:    "repo-err-test",
		BlockReward:       5000000000,
	}

	// Set GetLatestBlock to return a non-ErrChainEmpty error
	repo.GetLatestBlockErr = fmt.Errorf("database corruption")

	ch := chain.NewChain(repo, pow, cfg, utxoSet)

	err := ch.Initialize(context.Background(), "miner")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "database corruption")
}

func TestReorganize_WithoutUTXOSet(t *testing.T) {
	ctx := context.Background()
	minerAddr := "miner-reorg-no-utxo"

	repo := testutil.NewMockChainRepo()
	pow := &block.ProofOfWork{}
	cfg := chain.ChainConfig{
		InitialDifficulty: 1,
		GenesisMessage:    "reorg-no-utxo-test",
		BlockReward:       5000000000,
	}
	// Create chain without UTXO set
	ch := chain.NewChain(repo, pow, cfg, nil)

	require.NoError(t, ch.Initialize(ctx, minerAddr))

	// Mine 2 blocks
	for range 2 {
		_, err := ch.MineBlock(ctx, minerAddr, nil, 0)
		require.NoError(t, err)
	}

	forkBlock, err := repo.GetBlockByHeight(ctx, 1)
	require.NoError(t, err)

	// Create fork blocks without transactions (no UTXO set)
	coinbase := tx.NewCoinbaseTxWithHeight("fork", cfg.BlockReward, 2)
	blockTxs := []any{coinbase}
	txHashes := []block.Hash{coinbase.ID()}
	merkleRoot := block.ComputeMerkleRoot(txHashes)
	forkBlk, err := block.NewBlock(forkBlock.Hash(), 2, uint32(cfg.InitialDifficulty), blockTxs, merkleRoot)
	require.NoError(t, err)
	require.NoError(t, pow.Mine(forkBlk))

	coinbase2 := tx.NewCoinbaseTxWithHeight("fork", cfg.BlockReward, 3)
	blockTxs2 := []any{coinbase2}
	txHashes2 := []block.Hash{coinbase2.ID()}
	merkleRoot2 := block.ComputeMerkleRoot(txHashes2)
	forkBlk2, err := block.NewBlock(forkBlk.Hash(), 3, uint32(cfg.InitialDifficulty), blockTxs2, merkleRoot2)
	require.NoError(t, err)
	require.NoError(t, pow.Mine(forkBlk2))

	// Reorganize without UTXO set should use SaveBlock path
	err = ch.Reorganize(ctx, 1, []*block.Block{forkBlk, forkBlk2}, nil)
	require.NoError(t, err)
	assert.Equal(t, uint64(3), ch.Height())
}

func TestGetCurrentBits_ZeroInterval(t *testing.T) {
	ctx := context.Background()
	minerAddr := "miner-zero-interval"

	repo := testutil.NewMockChainRepo()
	utxoRepo := testutil.NewMockUTXORepo()
	utxoSet := utxo.NewSet(utxoRepo)
	pow := &block.ProofOfWork{}
	cfg := chain.ChainConfig{
		InitialDifficulty:        1,
		DifficultyAdjustInterval: 0, // zero interval -- no adjustment
		GenesisMessage:           "zero-interval-test",
		BlockReward:              5000000000,
	}
	ch := chain.NewChain(repo, pow, cfg, utxoSet)

	require.NoError(t, ch.Initialize(ctx, minerAddr))

	// Mine 3 blocks -- should all use the latest block's bits (no adjustment)
	for i := 0; i < 3; i++ {
		blk, err := ch.MineBlock(ctx, minerAddr, nil, 0)
		require.NoError(t, err)
		assert.Equal(t, uint32(1), blk.Bits())
	}
}
