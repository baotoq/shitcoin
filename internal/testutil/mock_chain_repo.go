package testutil

import (
	"context"
	"errors"
	"sort"
	"sync"

	"github.com/baotoq/shitcoin/internal/domain/block"
	"github.com/baotoq/shitcoin/internal/domain/chain"
	"github.com/baotoq/shitcoin/internal/domain/utxo"
)

// Compile-time interface check.
var _ chain.Repository = (*MockChainRepo)(nil)

// MockChainRepo is an in-memory implementation of chain.Repository for testing.
type MockChainRepo struct {
	mu       sync.RWMutex
	Blocks   map[block.Hash]*block.Block
	ByHeight map[uint64]*block.Block
	Undos    map[uint64]*utxo.UndoEntry
	Latest   *block.Block
}

// NewMockChainRepo creates a new MockChainRepo with initialized maps.
func NewMockChainRepo() *MockChainRepo {
	return &MockChainRepo{
		Blocks:   make(map[block.Hash]*block.Block),
		ByHeight: make(map[uint64]*block.Block),
		Undos:    make(map[uint64]*utxo.UndoEntry),
	}
}

// AddBlock is a convenience method for test setup. Stores a block using SaveBlock
// with context.Background().
func (m *MockChainRepo) AddBlock(b *block.Block) {
	_ = m.SaveBlock(context.Background(), b)
}

func (m *MockChainRepo) SaveBlock(_ context.Context, b *block.Block) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Blocks[b.Hash()] = b
	m.ByHeight[b.Height()] = b
	if m.Latest == nil || b.Height() > m.Latest.Height() {
		m.Latest = b
	}
	return nil
}

func (m *MockChainRepo) SaveBlockWithUTXOs(_ context.Context, b *block.Block, undoEntry *utxo.UndoEntry) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Blocks[b.Hash()] = b
	m.ByHeight[b.Height()] = b
	if undoEntry != nil {
		m.Undos[b.Height()] = undoEntry
	}
	if m.Latest == nil || b.Height() > m.Latest.Height() {
		m.Latest = b
	}
	return nil
}

func (m *MockChainRepo) GetBlock(_ context.Context, hash block.Hash) (*block.Block, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	b, ok := m.Blocks[hash]
	if !ok {
		return nil, errors.New("block not found")
	}
	return b, nil
}

func (m *MockChainRepo) GetBlockByHeight(_ context.Context, height uint64) (*block.Block, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	b, ok := m.ByHeight[height]
	if !ok {
		return nil, errors.New("block not found at height")
	}
	return b, nil
}

func (m *MockChainRepo) GetLatestBlock(_ context.Context) (*block.Block, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.Latest == nil {
		return nil, errors.New("chain is empty")
	}
	return m.Latest, nil
}

func (m *MockChainRepo) GetChainHeight(_ context.Context) (uint64, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.Latest == nil {
		return 0, nil
	}
	return m.Latest.Height(), nil
}

func (m *MockChainRepo) GetBlocksInRange(_ context.Context, startHeight, endHeight uint64) ([]*block.Block, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var result []*block.Block
	for h := startHeight; h <= endHeight; h++ {
		if b, ok := m.ByHeight[h]; ok {
			result = append(result, b)
		}
	}
	return result, nil
}

func (m *MockChainRepo) GetUndoEntry(_ context.Context, blockHeight uint64) (*utxo.UndoEntry, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	entry, ok := m.Undos[blockHeight]
	if !ok {
		return nil, errors.New("undo entry not found")
	}
	return entry, nil
}

func (m *MockChainRepo) DeleteBlocksAbove(_ context.Context, height uint64) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Collect heights to delete
	var toDelete []uint64
	for h := range m.ByHeight {
		if h > height {
			toDelete = append(toDelete, h)
		}
	}

	// Sort for deterministic deletion
	sort.Slice(toDelete, func(i, j int) bool { return toDelete[i] < toDelete[j] })

	for _, h := range toDelete {
		b := m.ByHeight[h]
		delete(m.Blocks, b.Hash())
		delete(m.ByHeight, h)
		delete(m.Undos, h)
	}

	// Recalculate Latest
	m.Latest = nil
	for _, b := range m.ByHeight {
		if m.Latest == nil || b.Height() > m.Latest.Height() {
			m.Latest = b
		}
	}

	return nil
}
