package mempool

import (
	"fmt"
	"slices"
	"sync"

	"github.com/baotoq/shitcoin/internal/domain/block"
	"github.com/baotoq/shitcoin/internal/domain/tx"
	"github.com/baotoq/shitcoin/internal/domain/utxo"
)

// mempoolEntry wraps a transaction with its associated fee for priority sorting.
type mempoolEntry struct {
	tx  *tx.Transaction
	fee int64
}

// Mempool holds validated unconfirmed transactions awaiting mining.
// Thread-safe via sync.RWMutex; safe for concurrent access.
type Mempool struct {
	mu      sync.RWMutex
	entries map[block.Hash]*mempoolEntry
	utxoSet *utxo.Set
	// spentOutputs tracks which UTXO (txid:vout) is already spent by a mempool TX,
	// keyed by "txid_hex:vout".
	spentOutputs map[string]block.Hash
}

// New creates a new Mempool backed by the given UTXO set for validation.
func New(utxoSet *utxo.Set) *Mempool {
	return &Mempool{
		entries:      make(map[block.Hash]*mempoolEntry),
		utxoSet:      utxoSet,
		spentOutputs: make(map[string]block.Hash),
	}
}

// Add validates and adds a transaction to the mempool with zero fee.
// Checks: duplicate, signature validity, UTXO existence, double-spend against pool.
func (m *Mempool) Add(transaction *tx.Transaction) error {
	return m.AddWithFee(transaction, 0)
}

// AddWithFee validates and adds a transaction to the mempool with the given fee.
// Checks: duplicate, signature validity, UTXO existence, double-spend against pool.
func (m *Mempool) AddWithFee(transaction *tx.Transaction, fee int64) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	txID := transaction.ID()

	// Check duplicate
	if _, exists := m.entries[txID]; exists {
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
	m.entries[txID] = &mempoolEntry{tx: transaction, fee: fee}

	// Track spent outputs
	for _, input := range transaction.Inputs() {
		key := fmt.Sprintf("%s:%d", input.TxID().String(), input.Vout())
		m.spentOutputs[key] = txID
	}

	return nil
}

// DrainByFee removes and returns transactions sorted by fee descending.
// If maxTxs > 0, only the top maxTxs transactions are returned; the rest stay in the pool.
// If maxTxs <= 0, all transactions are drained.
// Returns the transactions and the total fees of the drained transactions.
func (m *Mempool) DrainByFee(maxTxs int) ([]*tx.Transaction, int64) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Collect all entries
	all := make([]*mempoolEntry, 0, len(m.entries))
	for _, entry := range m.entries {
		all = append(all, entry)
	}

	// Sort by fee descending
	slices.SortFunc(all, func(a, b *mempoolEntry) int {
		if a.fee > b.fee {
			return -1
		}
		if a.fee < b.fee {
			return 1
		}
		return 0
	})

	// Determine how many to take
	takeCount := len(all)
	if maxTxs > 0 && maxTxs < takeCount {
		takeCount = maxTxs
	}

	// Build result from top entries
	result := make([]*tx.Transaction, takeCount)
	var totalFees int64
	for i := 0; i < takeCount; i++ {
		result[i] = all[i].tx
		totalFees += all[i].fee
	}

	// Remove drained entries and their spent outputs
	for i := 0; i < takeCount; i++ {
		txID := all[i].tx.ID()
		for _, input := range all[i].tx.Inputs() {
			key := fmt.Sprintf("%s:%d", input.TxID().String(), input.Vout())
			delete(m.spentOutputs, key)
		}
		delete(m.entries, txID)
	}

	return result, totalFees
}

// DrainAll removes and returns all transactions from the mempool.
func (m *Mempool) DrainAll() []*tx.Transaction {
	txs, _ := m.DrainByFee(0)
	return txs
}

// Remove removes transactions by their IDs from the mempool.
func (m *Mempool) Remove(txIDs []block.Hash) {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, txID := range txIDs {
		entry, exists := m.entries[txID]
		if !exists {
			continue
		}

		// Clean up spent outputs tracking
		for _, input := range entry.tx.Inputs() {
			key := fmt.Sprintf("%s:%d", input.TxID().String(), input.Vout())
			delete(m.spentOutputs, key)
		}

		delete(m.entries, txID)
	}
}

// GetByID returns a transaction from the mempool by its hash, or nil if not found.
func (m *Mempool) GetByID(id block.Hash) *tx.Transaction {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if entry, ok := m.entries[id]; ok {
		return entry.tx
	}
	return nil
}

// Count returns the number of transactions in the mempool.
func (m *Mempool) Count() int {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return len(m.entries)
}

// Transactions returns a copy of all transactions in the mempool (read-only view).
func (m *Mempool) Transactions() []*tx.Transaction {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make([]*tx.Transaction, 0, len(m.entries))
	for _, entry := range m.entries {
		result = append(result, entry.tx)
	}
	return result
}

// FeeForTx returns the fee associated with a transaction in the mempool.
// Returns 0 if the transaction is not found.
func (m *Mempool) FeeForTx(id block.Hash) int64 {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if entry, ok := m.entries[id]; ok {
		return entry.fee
	}
	return 0
}
