package mempool

import (
	"github.com/baotoq/shitcoin/internal/domain/block"
	"github.com/baotoq/shitcoin/internal/domain/tx"
	"github.com/baotoq/shitcoin/internal/domain/utxo"
)

// Ensure imports are used (stubs only).
var (
	_ block.Hash
	_ *tx.Transaction
	_ *utxo.Set
)

// Mempool holds validated unconfirmed transactions awaiting mining.
type Mempool struct{}

// New creates a new Mempool backed by the given UTXO set for validation.
func New(utxoSet *utxo.Set) *Mempool {
	panic("not implemented")
}

// Add validates and adds a transaction to the mempool.
func (m *Mempool) Add(transaction *tx.Transaction) error {
	panic("not implemented")
}

// DrainAll removes and returns all transactions from the mempool.
func (m *Mempool) DrainAll() []*tx.Transaction {
	panic("not implemented")
}

// Remove removes transactions by their IDs from the mempool.
func (m *Mempool) Remove(txIDs []block.Hash) {
	panic("not implemented")
}

// Count returns the number of transactions in the mempool.
func (m *Mempool) Count() int {
	panic("not implemented")
}

// Transactions returns a copy of all transactions in the mempool (read-only view).
func (m *Mempool) Transactions() []*tx.Transaction {
	panic("not implemented")
}
