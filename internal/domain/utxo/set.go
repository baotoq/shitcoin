package utxo

import (
	"fmt"

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
	undo := &UndoEntry{
		BlockHeight: blockHeight,
		Spent:       make([]SpentUTXO, 0),
		Created:     make([]UTXORef, 0),
	}

	// Track UTXOs spent within this block to detect intra-block double-spend
	spentInBlock := make(map[string]bool)

	// Collect all spent UTXOs first
	type spentRecord struct {
		utxo UTXO
		key  string
	}
	var toSpend []spentRecord

	for _, transaction := range txs {
		if transaction.IsCoinbase() {
			continue
		}

		for _, input := range transaction.Inputs() {
			key := fmt.Sprintf("%s:%d", input.TxID().String(), input.Vout())

			// Check for intra-block double-spend
			if spentInBlock[key] {
				return nil, fmt.Errorf("%w: utxo %s already spent in this block", ErrDoubleSpend, key)
			}
			spentInBlock[key] = true

			// Get the UTXO being spent
			existing, err := s.repo.Get(input.TxID(), input.Vout())
			if err != nil {
				return nil, fmt.Errorf("get utxo %s: %w", key, err)
			}

			toSpend = append(toSpend, spentRecord{utxo: existing, key: key})

			undo.Spent = append(undo.Spent, SpentUTXO{
				TxID:    existing.TxID().String(),
				Vout:    existing.Vout(),
				Value:   existing.Value(),
				Address: existing.Address(),
			})
		}
	}

	// Add created UTXOs for all transactions (including coinbase)
	for _, transaction := range txs {
		txID := transaction.ID()
		for i, output := range transaction.Outputs() {
			newUTXO := NewUTXO(txID, uint32(i), output.Value(), output.Address())
			if err := s.repo.Put(newUTXO); err != nil {
				return nil, fmt.Errorf("put utxo: %w", err)
			}

			undo.Created = append(undo.Created, UTXORef{
				TxID: txID.String(),
				Vout: uint32(i),
			})
		}
	}

	// Remove spent UTXOs after all new UTXOs are added
	for _, record := range toSpend {
		if err := s.repo.Delete(record.utxo.TxID(), record.utxo.Vout()); err != nil {
			return nil, fmt.Errorf("delete spent utxo %s: %w", record.key, err)
		}
	}

	return undo, nil
}

// UndoBlock reverses the UTXO changes recorded in the given UndoEntry.
// Restores spent UTXOs and removes created UTXOs.
func (s *Set) UndoBlock(entry *UndoEntry) error {
	// Remove created UTXOs
	for _, ref := range entry.Created {
		txID, err := block.HashFromHex(ref.TxID)
		if err != nil {
			return fmt.Errorf("parse created txid %s: %w", ref.TxID, err)
		}
		if err := s.repo.Delete(txID, ref.Vout); err != nil {
			return fmt.Errorf("delete created utxo %s:%d: %w", ref.TxID, ref.Vout, err)
		}
	}

	// Restore spent UTXOs
	for _, spent := range entry.Spent {
		txID, err := block.HashFromHex(spent.TxID)
		if err != nil {
			return fmt.Errorf("parse spent txid %s: %w", spent.TxID, err)
		}
		restored := NewUTXO(txID, spent.Vout, spent.Value, spent.Address)
		if err := s.repo.Put(restored); err != nil {
			return fmt.Errorf("restore spent utxo: %w", err)
		}
	}

	return nil
}

// GetByAddress returns all UTXOs belonging to the given address.
func (s *Set) GetByAddress(address string) ([]UTXO, error) {
	return s.repo.GetByAddress(address)
}

// GetBalance returns the total balance for an address by summing all UTXO values.
func (s *Set) GetBalance(address string) (int64, error) {
	utxos, err := s.repo.GetByAddress(address)
	if err != nil {
		return 0, fmt.Errorf("get utxos by address: %w", err)
	}

	var total int64
	for _, u := range utxos {
		total += u.Value()
	}
	return total, nil
}

// Get returns a specific UTXO by transaction ID and output index.
func (s *Set) Get(txID block.Hash, vout uint32) (UTXO, error) {
	return s.repo.Get(txID, vout)
}
