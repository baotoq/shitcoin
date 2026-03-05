package utxo

import "errors"

var (
	// ErrUTXONotFound is returned when a UTXO does not exist in the set.
	ErrUTXONotFound = errors.New("utxo not found")

	// ErrDoubleSpend is returned when a transaction tries to spend an already-spent UTXO.
	ErrDoubleSpend = errors.New("double spend detected")

	// ErrUndoEntryNotFound is returned when an undo entry does not exist for the given block height.
	ErrUndoEntryNotFound = errors.New("undo entry not found")
)
