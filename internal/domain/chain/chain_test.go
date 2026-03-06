package chain_test

import (
	"context"
	"fmt"
	"sync"
	"testing"

	"github.com/baotoq/shitcoin/internal/domain/block"
	"github.com/baotoq/shitcoin/internal/domain/chain"
	"github.com/baotoq/shitcoin/internal/domain/tx"
	"github.com/baotoq/shitcoin/internal/domain/utxo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// mockUTXORepo implements utxo.Repository in-memory for chain tests.
type mockUTXORepo struct {
	mu    sync.Mutex
	utxos map[string]utxo.UTXO
	undos map[uint64]*utxo.UndoEntry
}

func newMockUTXORepo() *mockUTXORepo {
	return &mockUTXORepo{
		utxos: make(map[string]utxo.UTXO),
		undos: make(map[uint64]*utxo.UndoEntry),
	}
}

func (r *mockUTXORepo) utxoKey(txID block.Hash, vout uint32) string {
	return fmt.Sprintf("%s:%d", txID.String(), vout)
}

func (r *mockUTXORepo) Put(u utxo.UTXO) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.utxos[r.utxoKey(u.TxID(), u.Vout())] = u
	return nil
}

func (r *mockUTXORepo) Get(txID block.Hash, vout uint32) (utxo.UTXO, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if u, ok := r.utxos[r.utxoKey(txID, vout)]; ok {
		return u, nil
	}
	return utxo.UTXO{}, utxo.ErrUTXONotFound
}

func (r *mockUTXORepo) Delete(txID block.Hash, vout uint32) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.utxos, r.utxoKey(txID, vout))
	return nil
}

func (r *mockUTXORepo) GetByAddress(address string) ([]utxo.UTXO, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var result []utxo.UTXO
	for _, u := range r.utxos {
		if u.Address() == address {
			result = append(result, u)
		}
	}
	return result, nil
}

func (r *mockUTXORepo) SaveUndoEntry(entry *utxo.UndoEntry) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.undos[entry.BlockHeight] = entry
	return nil
}

func (r *mockUTXORepo) GetUndoEntry(blockHeight uint64) (*utxo.UndoEntry, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if e, ok := r.undos[blockHeight]; ok {
		return e, nil
	}
	return nil, utxo.ErrUndoEntryNotFound
}

func (r *mockUTXORepo) DeleteUndoEntry(blockHeight uint64) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.undos, blockHeight)
	return nil
}

// mockChainRepo implements chain.Repository in-memory with full support
// including GetUndoEntry and DeleteBlocksAbove.
type mockChainRepo struct {
	mu       sync.RWMutex
	blocks   map[block.Hash]*block.Block
	byHeight map[uint64]*block.Block
	latest   *block.Block
	undos    map[uint64]*utxo.UndoEntry
}

func newMockChainRepo() *mockChainRepo {
	return &mockChainRepo{
		blocks:   make(map[block.Hash]*block.Block),
		byHeight: make(map[uint64]*block.Block),
		undos:    make(map[uint64]*utxo.UndoEntry),
	}
}

func (m *mockChainRepo) SaveBlock(_ context.Context, b *block.Block) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.blocks[b.Hash()] = b
	m.byHeight[b.Height()] = b
	m.latest = b
	return nil
}

func (m *mockChainRepo) SaveBlockWithUTXOs(_ context.Context, b *block.Block, undo *utxo.UndoEntry) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.blocks[b.Hash()] = b
	m.byHeight[b.Height()] = b
	m.latest = b
	if undo != nil {
		m.undos[undo.BlockHeight] = undo
	}
	return nil
}

func (m *mockChainRepo) GetBlock(_ context.Context, hash block.Hash) (*block.Block, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if b, ok := m.blocks[hash]; ok {
		return b, nil
	}
	return nil, chain.ErrBlockNotFound
}

func (m *mockChainRepo) GetBlockByHeight(_ context.Context, height uint64) (*block.Block, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if b, ok := m.byHeight[height]; ok {
		return b, nil
	}
	return nil, chain.ErrBlockNotFound
}

func (m *mockChainRepo) GetLatestBlock(_ context.Context) (*block.Block, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.latest != nil {
		return m.latest, nil
	}
	return nil, chain.ErrChainEmpty
}

func (m *mockChainRepo) GetChainHeight(_ context.Context) (uint64, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.latest != nil {
		return m.latest.Height(), nil
	}
	return 0, nil
}

func (m *mockChainRepo) GetBlocksInRange(_ context.Context, start, end uint64) ([]*block.Block, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var result []*block.Block
	for h := start; h <= end; h++ {
		if b, ok := m.byHeight[h]; ok {
			result = append(result, b)
		} else {
			return nil, chain.ErrBlockNotFound
		}
	}
	return result, nil
}

func (m *mockChainRepo) GetUndoEntry(_ context.Context, blockHeight uint64) (*utxo.UndoEntry, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if e, ok := m.undos[blockHeight]; ok {
		return e, nil
	}
	return nil, utxo.ErrUndoEntryNotFound
}

func (m *mockChainRepo) DeleteBlocksAbove(_ context.Context, height uint64) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	// Find all heights above the given height and remove them
	for h, b := range m.byHeight {
		if h > height {
			delete(m.blocks, b.Hash())
			delete(m.byHeight, h)
			delete(m.undos, h)
		}
	}
	// Update latest to the block at the given height
	if b, ok := m.byHeight[height]; ok {
		m.latest = b
	}
	return nil
}

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

	repo := newMockChainRepo()
	utxoRepo := newMockUTXORepo()
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
		_, err := ch.MineBlock(ctx, minerAddr, nil)
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

	repo := newMockChainRepo()
	utxoRepo := newMockUTXORepo()
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
		_, err := ch.MineBlock(ctx, minerAddr, nil)
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

	repo := newMockChainRepo()
	utxoRepo := newMockUTXORepo()
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
		_, err := ch.MineBlock(ctx, minerAddr, nil)
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
