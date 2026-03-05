package tx

import (
	"encoding/json"

	"github.com/baotoq/shitcoin/internal/domain/block"
)

// hashableTransaction is used for deterministic ID computation.
// Signature and PubKey are intentionally excluded to avoid the chicken-and-egg problem.
type hashableTransaction struct {
	Inputs  []hashableInput  `json:"inputs"`
	Outputs []hashableOutput `json:"outputs"`
}

// hashableInput excludes signature and pubkey from the hash.
type hashableInput struct {
	TxID string `json:"txid"`
	Vout uint32 `json:"vout"`
}

// hashableOutput includes value and address for hash computation.
type hashableOutput struct {
	Value   int64  `json:"value"`
	Address string `json:"address"`
}

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
	tx := &Transaction{
		inputs:  inputs,
		outputs: outputs,
	}
	tx.id = tx.ComputeID()
	return tx
}

// ReconstructTransaction recreates a transaction from stored data without recomputing the ID.
func ReconstructTransaction(id block.Hash, inputs []TxInput, outputs []TxOutput) *Transaction {
	return &Transaction{
		id:      id,
		inputs:  inputs,
		outputs: outputs,
	}
}

// ID returns the transaction hash.
func (t *Transaction) ID() block.Hash {
	return t.id
}

// Inputs returns the transaction inputs.
func (t *Transaction) Inputs() []TxInput {
	return t.inputs
}

// Outputs returns the transaction outputs.
func (t *Transaction) Outputs() []TxOutput {
	return t.outputs
}

// ComputeID computes the deterministic transaction ID from inputs and outputs.
// Signature and PubKey fields are excluded from the hash computation.
func (t *Transaction) ComputeID() block.Hash {
	payload := t.hashPayload()
	data, _ := json.Marshal(payload)
	return block.DoubleSHA256(data)
}

// IsCoinbase returns true if this is a coinbase transaction.
// A coinbase has exactly one input with a zero hash and vout=0xFFFFFFFF.
func (t *Transaction) IsCoinbase() bool {
	return len(t.inputs) == 1 &&
		t.inputs[0].txID.IsZero() &&
		t.inputs[0].vout == 0xFFFFFFFF
}

// hashPayload builds the hashable representation excluding signatures.
func (t *Transaction) hashPayload() hashableTransaction {
	hashInputs := make([]hashableInput, len(t.inputs))
	for i, in := range t.inputs {
		hashInputs[i] = hashableInput{
			TxID: in.txID.String(),
			Vout: in.vout,
		}
	}

	hashOutputs := make([]hashableOutput, len(t.outputs))
	for i, out := range t.outputs {
		hashOutputs[i] = hashableOutput{
			Value:   out.value,
			Address: out.address,
		}
	}

	return hashableTransaction{
		Inputs:  hashInputs,
		Outputs: hashOutputs,
	}
}
