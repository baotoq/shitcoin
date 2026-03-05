package mempool

import "errors"

var (
	// ErrDuplicate is returned when a transaction already exists in the mempool.
	ErrDuplicate = errors.New("transaction already in mempool")

	// ErrDoubleSpend is returned when a transaction spends a UTXO already spent by another mempool transaction.
	ErrDoubleSpend = errors.New("double spend: input already spent by mempool transaction")

	// ErrInvalidSignature is returned when a transaction has an invalid or missing signature.
	ErrInvalidSignature = errors.New("invalid transaction signature")

	// ErrUTXONotFound is returned when a transaction references a UTXO that does not exist.
	ErrUTXONotFound = errors.New("referenced UTXO not found")
)
