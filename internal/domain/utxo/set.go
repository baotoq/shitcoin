package utxo

import (
	"github.com/baotoq/shitcoin/internal/domain/block"
	"github.com/baotoq/shitcoin/internal/domain/tx"
)

// Set is the UTXO set aggregate that tracks all unspent transaction outputs.
// It delegates persistence to a Repository and provides operations for
// applying and undoing blocks.
type Set struct {
	repo Repository
}

// NewSet creates a new UTXO set aggregate backed by the given repository.
func NewSet(repo Repository) *Set {
	return &Set{repo: repo}
}

// ApplyBlock processes a block's transactions, removing spent UTXOs and adding
// created UTXOs. Returns an UndoEntry that can reverse the changes.
// Detects intra-block double-spend (same UTXO spent by multiple transactions in one block).
func (s *Set) ApplyBlock(blockHeight uint64, txs []*tx.Transaction) (*UndoEntry, error) {
	panic("not implemented")
}

// UndoBlock reverses the UTXO changes recorded in the given UndoEntry.
// Restores spent UTXOs and removes created UTXOs.
func (s *Set) UndoBlock(entry *UndoEntry) error {
	panic("not implemented")
}

// GetByAddress returns all UTXOs belonging to the given address.
func (s *Set) GetByAddress(address string) ([]UTXO, error) {
	return s.repo.GetByAddress(address)
}

// GetBalance returns the total balance for an address by summing all UTXO values.
func (s *Set) GetBalance(address string) (int64, error) {
	panic("not implemented")
}

// Get returns a specific UTXO by transaction ID and output index.
func (s *Set) Get(txID block.Hash, vout uint32) (UTXO, error) {
	return s.repo.Get(txID, vout)
}
