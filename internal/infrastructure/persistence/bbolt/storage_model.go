package bbolt

import (
	"encoding/hex"
	"fmt"

	"github.com/baotoq/shitcoin/internal/domain/block"
	"github.com/baotoq/shitcoin/internal/domain/tx"
)

// HeaderModel is a JSON-serializable storage model for block headers.
type HeaderModel struct {
	Version       uint32 `json:"version"`
	PrevBlockHash string `json:"prev_block_hash"`
	MerkleRoot    string `json:"merkle_root"`
	Timestamp     int64  `json:"timestamp"`
	Bits          uint32 `json:"bits"`
	Nonce         uint32 `json:"nonce"`
}

// TxInputModel is a JSON-serializable storage model for transaction inputs.
type TxInputModel struct {
	TxID      string `json:"txid"`
	Vout      uint32 `json:"vout"`
	Signature string `json:"signature,omitempty"` // hex-encoded
	PubKey    string `json:"pubkey,omitempty"`    // hex-encoded
}

// TxOutputModel is a JSON-serializable storage model for transaction outputs.
type TxOutputModel struct {
	Value   int64  `json:"value"`
	Address string `json:"address"`
}

// TxModel is a JSON-serializable storage model for transactions.
type TxModel struct {
	ID      string          `json:"id"`
	Inputs  []TxInputModel  `json:"inputs"`
	Outputs []TxOutputModel `json:"outputs"`
}

// TxModelFromDomain converts a domain Transaction to a storage TxModel.
func TxModelFromDomain(t *tx.Transaction) TxModel {
	inputs := make([]TxInputModel, len(t.Inputs()))
	for i, in := range t.Inputs() {
		inputs[i] = TxInputModel{
			TxID:      in.TxID().String(),
			Vout:      in.Vout(),
			Signature: hex.EncodeToString(in.Signature()),
			PubKey:    hex.EncodeToString(in.PubKey()),
		}
	}

	outputs := make([]TxOutputModel, len(t.Outputs()))
	for i, out := range t.Outputs() {
		outputs[i] = TxOutputModel{
			Value:   out.Value(),
			Address: out.Address(),
		}
	}

	return TxModel{
		ID:      t.ID().String(),
		Inputs:  inputs,
		Outputs: outputs,
	}
}

// ToDomain converts a storage TxModel back to a domain Transaction.
func (m TxModel) ToDomain() (*tx.Transaction, error) {
	txID, err := block.HashFromHex(m.ID)
	if err != nil {
		return nil, fmt.Errorf("invalid tx id: %w", err)
	}

	inputs := make([]tx.TxInput, len(m.Inputs))
	for i, in := range m.Inputs {
		inTxID, err := block.HashFromHex(in.TxID)
		if err != nil {
			return nil, fmt.Errorf("invalid input txid: %w", err)
		}
		input := tx.NewTxInput(inTxID, in.Vout)

		if in.Signature != "" {
			sig, err := hex.DecodeString(in.Signature)
			if err != nil {
				return nil, fmt.Errorf("invalid input signature hex: %w", err)
			}
			input.SetSignature(sig)
		}
		if in.PubKey != "" {
			pk, err := hex.DecodeString(in.PubKey)
			if err != nil {
				return nil, fmt.Errorf("invalid input pubkey hex: %w", err)
			}
			input.SetPubKey(pk)
		}

		inputs[i] = input
	}

	outputs := make([]tx.TxOutput, len(m.Outputs))
	for i, out := range m.Outputs {
		outputs[i] = tx.NewTxOutput(out.Value, out.Address)
	}

	return tx.ReconstructTransaction(txID, inputs, outputs), nil
}

// BlockModel is a JSON-serializable storage model for blocks.
type BlockModel struct {
	Hash         string      `json:"hash"`
	Header       HeaderModel `json:"header"`
	Height       uint64      `json:"height"`
	Message      string      `json:"message,omitempty"`
	Transactions []TxModel   `json:"transactions"`
}

// BlockModelFromDomain converts a domain Block to a storage BlockModel.
func BlockModelFromDomain(b *block.Block) *BlockModel {
	h := b.Header()

	txModels := make([]TxModel, 0, len(b.RawTransactions()))
	for _, rawTx := range b.RawTransactions() {
		if t, ok := rawTx.(*tx.Transaction); ok {
			txModels = append(txModels, TxModelFromDomain(t))
		}
	}

	return &BlockModel{
		Hash: b.Hash().String(),
		Header: HeaderModel{
			Version:       h.Version(),
			PrevBlockHash: h.PrevBlockHash().String(),
			MerkleRoot:    h.MerkleRoot().String(),
			Timestamp:     h.Timestamp(),
			Bits:          h.Bits(),
			Nonce:         h.Nonce(),
		},
		Height:       b.Height(),
		Message:      b.Message(),
		Transactions: txModels,
	}
}

// ToDomain converts a storage BlockModel back to a domain Block using ReconstructBlock.
func (bm *BlockModel) ToDomain() (*block.Block, error) {
	hash, err := block.HashFromHex(bm.Hash)
	if err != nil {
		return nil, fmt.Errorf("invalid block hash: %w", err)
	}

	prevHash, err := block.HashFromHex(bm.Header.PrevBlockHash)
	if err != nil {
		return nil, fmt.Errorf("invalid prev block hash: %w", err)
	}

	merkleRoot, err := block.HashFromHex(bm.Header.MerkleRoot)
	if err != nil {
		return nil, fmt.Errorf("invalid merkle root: %w", err)
	}

	header := block.NewHeader(
		bm.Header.Version,
		prevHash,
		merkleRoot,
		bm.Header.Timestamp,
		bm.Header.Bits,
	)
	header.SetNonce(bm.Header.Nonce)

	// Convert transaction models to domain transactions as []any
	// Handle backward compatibility: nil or empty means no transactions
	txs := make([]any, 0, len(bm.Transactions))
	for _, txModel := range bm.Transactions {
		t, err := txModel.ToDomain()
		if err != nil {
			return nil, fmt.Errorf("convert tx model: %w", err)
		}
		txs = append(txs, t)
	}

	return block.ReconstructBlock(header, hash, bm.Height, bm.Message, txs), nil
}
