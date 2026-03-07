package testutil

import (
	"fmt"
	"sync"

	"github.com/baotoq/shitcoin/internal/domain/block"
	"github.com/baotoq/shitcoin/internal/domain/utxo"
)

// Compile-time interface check.
var _ utxo.Repository = (*MockUTXORepo)(nil)

// MockUTXORepo is an in-memory implementation of utxo.Repository for testing.
type MockUTXORepo struct {
	mu    sync.Mutex
	UTXOs map[string]utxo.UTXO
	Undos map[uint64]*utxo.UndoEntry
}

// NewMockUTXORepo creates a new MockUTXORepo with initialized maps.
func NewMockUTXORepo() *MockUTXORepo {
	return &MockUTXORepo{
		UTXOs: make(map[string]utxo.UTXO),
		Undos: make(map[uint64]*utxo.UndoEntry),
	}
}

func utxoKey(txID block.Hash, vout uint32) string {
	return fmt.Sprintf("%x:%d", txID, vout)
}

func (m *MockUTXORepo) Put(u utxo.UTXO) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.UTXOs[utxoKey(u.TxID(), u.Vout())] = u
	return nil
}

func (m *MockUTXORepo) Get(txID block.Hash, vout uint32) (utxo.UTXO, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	u, ok := m.UTXOs[utxoKey(txID, vout)]
	if !ok {
		return utxo.UTXO{}, utxo.ErrUTXONotFound
	}
	return u, nil
}

func (m *MockUTXORepo) Delete(txID block.Hash, vout uint32) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	key := utxoKey(txID, vout)
	if _, ok := m.UTXOs[key]; !ok {
		return utxo.ErrUTXONotFound
	}
	delete(m.UTXOs, key)
	return nil
}

func (m *MockUTXORepo) GetByAddress(address string) ([]utxo.UTXO, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var result []utxo.UTXO
	for _, u := range m.UTXOs {
		if u.Address() == address {
			result = append(result, u)
		}
	}
	if result == nil {
		result = make([]utxo.UTXO, 0)
	}
	return result, nil
}

func (m *MockUTXORepo) SaveUndoEntry(entry *utxo.UndoEntry) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Undos[entry.BlockHeight] = entry
	return nil
}

func (m *MockUTXORepo) GetUndoEntry(blockHeight uint64) (*utxo.UndoEntry, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	entry, ok := m.Undos[blockHeight]
	if !ok {
		return nil, utxo.ErrUndoEntryNotFound
	}
	return entry, nil
}

func (m *MockUTXORepo) DeleteUndoEntry(blockHeight uint64) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.Undos, blockHeight)
	return nil
}
