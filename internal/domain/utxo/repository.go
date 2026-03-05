package utxo

import "github.com/baotoq/shitcoin/internal/domain/block"

// Repository defines the persistence interface for the UTXO set.
// Interface lives in the domain layer; implementation in infrastructure.
type Repository interface {
	// Put stores a UTXO in the set.
	Put(utxo UTXO) error

	// Get retrieves a specific UTXO by transaction ID and output index.
	// Returns ErrUTXONotFound if the UTXO does not exist.
	Get(txID block.Hash, vout uint32) (UTXO, error)

	// Delete removes a UTXO from the set.
	// Returns ErrUTXONotFound if the UTXO does not exist.
	Delete(txID block.Hash, vout uint32) error

	// GetByAddress returns all UTXOs belonging to the given address.
	GetByAddress(address string) ([]UTXO, error)

	// SaveUndoEntry persists an undo entry for the given block height.
	SaveUndoEntry(entry *UndoEntry) error

	// GetUndoEntry retrieves the undo entry for a block height.
	// Returns ErrUndoEntryNotFound if no entry exists.
	GetUndoEntry(blockHeight uint64) (*UndoEntry, error)

	// DeleteUndoEntry removes the undo entry for a block height.
	DeleteUndoEntry(blockHeight uint64) error
}
