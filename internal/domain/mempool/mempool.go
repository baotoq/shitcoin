package mempool

import (
	"fmt"
	"sync"

	"github.com/baotoq/shitcoin/internal/domain/block"
	"github.com/baotoq/shitcoin/internal/domain/tx"
	"github.com/baotoq/shitcoin/internal/domain/utxo"
)

// Mempool holds validated unconfirmed transactions awaiting mining.
// Thread-safe via sync.RWMutex; safe for concurrent access.
type Mempool struct {
	mu      sync.RWMutex
	txs     map[block.Hash]*tx.Transaction
	utxoSet *utxo.Set
	// spentOutputs tracks which UTXO (txid:vout) is already spent by a mempool TX,
	// keyed by "txid_hex:vout".
	spentOutputs map[string]block.Hash
}

// New creates a new Mempool backed by the given UTXO set for validation.
func New(utxoSet *utxo.Set) *Mempool {
	return &Mempool{
		txs:          make(map[block.Hash]*tx.Transaction),
		utxoSet:      utxoSet,
		spentOutputs: make(map[string]block.Hash),
	}
}

// Add validates and adds a transaction to the mempool.
// Checks: duplicate, signature validity, UTXO existence, double-spend against pool.
func (m *Mempool) Add(transaction *tx.Transaction) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	txID := transaction.ID()

	// Check duplicate
	if _, exists := m.txs[txID]; exists {
		return ErrDuplicate
	}

	// Verify signature (skip for coinbase, but coinbase should not enter mempool)
	if !tx.VerifyTransaction(transaction) {
		return ErrInvalidSignature
	}

	// Check each input: UTXO existence and double-spend against pool
	for _, input := range transaction.Inputs() {
		key := fmt.Sprintf("%s:%d", input.TxID().String(), input.Vout())

		// Check double-spend against other mempool transactions
		if _, spent := m.spentOutputs[key]; spent {
			return ErrDoubleSpend
		}

		// Check UTXO exists in the confirmed set
		if _, err := m.utxoSet.Get(input.TxID(), input.Vout()); err != nil {
			return ErrUTXONotFound
		}
	}

	// All checks passed -- add to mempool
	m.txs[txID] = transaction

	// Track spent outputs
	for _, input := range transaction.Inputs() {
		key := fmt.Sprintf("%s:%d", input.TxID().String(), input.Vout())
		m.spentOutputs[key] = txID
	}

	return nil
}

// DrainAll removes and returns all transactions from the mempool.
func (m *Mempool) DrainAll() []*tx.Transaction {
	m.mu.Lock()
	defer m.mu.Unlock()

	result := make([]*tx.Transaction, 0, len(m.txs))
	for _, transaction := range m.txs {
		result = append(result, transaction)
	}

	m.txs = make(map[block.Hash]*tx.Transaction)
	m.spentOutputs = make(map[string]block.Hash)

	return result
}

// Remove removes transactions by their IDs from the mempool.
func (m *Mempool) Remove(txIDs []block.Hash) {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, txID := range txIDs {
		transaction, exists := m.txs[txID]
		if !exists {
			continue
		}

		// Clean up spent outputs tracking
		for _, input := range transaction.Inputs() {
			key := fmt.Sprintf("%s:%d", input.TxID().String(), input.Vout())
			delete(m.spentOutputs, key)
		}

		delete(m.txs, txID)
	}
}

// GetByID returns a transaction from the mempool by its hash, or nil if not found.
func (m *Mempool) GetByID(id block.Hash) *tx.Transaction {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.txs[id]
}

// Count returns the number of transactions in the mempool.
func (m *Mempool) Count() int {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return len(m.txs)
}

// Transactions returns a copy of all transactions in the mempool (read-only view).
func (m *Mempool) Transactions() []*tx.Transaction {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make([]*tx.Transaction, 0, len(m.txs))
	for _, transaction := range m.txs {
		result = append(result, transaction)
	}
	return result
}
