package tx

import "github.com/baotoq/shitcoin/internal/domain/block"

// Transaction represents a UTXO-based transaction with inputs and outputs.
// The transaction ID is computed deterministically from inputs (excluding signatures)
// and outputs using JSON serialization + DoubleSHA256.
type Transaction struct {
	id      block.Hash
	inputs  []TxInput
	outputs []TxOutput
}

// NewTransaction creates a new transaction with the given inputs and outputs,
// computing the transaction ID deterministically.
func NewTransaction(inputs []TxInput, outputs []TxOutput) *Transaction {
	panic("not implemented")
}

// ReconstructTransaction recreates a transaction from stored data without recomputing the ID.
func ReconstructTransaction(id block.Hash, inputs []TxInput, outputs []TxOutput) *Transaction {
	panic("not implemented")
}

// ID returns the transaction hash.
func (t *Transaction) ID() block.Hash {
	panic("not implemented")
}

// Inputs returns the transaction inputs.
func (t *Transaction) Inputs() []TxInput {
	panic("not implemented")
}

// Outputs returns the transaction outputs.
func (t *Transaction) Outputs() []TxOutput {
	panic("not implemented")
}

// ComputeID computes the deterministic transaction ID from inputs and outputs.
// Signature and PubKey fields are excluded from the hash computation.
func (t *Transaction) ComputeID() block.Hash {
	panic("not implemented")
}

// IsCoinbase returns true if this is a coinbase transaction.
// A coinbase has exactly one input with a zero hash and vout=0xFFFFFFFF.
func (t *Transaction) IsCoinbase() bool {
	panic("not implemented")
}
