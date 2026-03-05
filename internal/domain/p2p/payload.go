package p2p

import (
	"encoding/hex"
	"fmt"

	"github.com/baotoq/shitcoin/internal/domain/block"
	"github.com/baotoq/shitcoin/internal/domain/tx"
)

// BlockPayloadFromDomain converts a domain Block to a P2P BlockPayload.
func BlockPayloadFromDomain(b *block.Block) BlockPayload {
	h := b.Header()

	txPayloads := make([]TxPayload, 0, len(b.RawTransactions()))
	for _, rawTx := range b.RawTransactions() {
		if t, ok := rawTx.(*tx.Transaction); ok {
			txPayloads = append(txPayloads, TxPayloadFromDomain(t))
		}
	}

	return BlockPayload{
		Hash:   b.Hash().String(),
		Height: b.Height(),
		Header: HeaderPayload{
			Version:       h.Version(),
			PrevBlockHash: h.PrevBlockHash().String(),
			MerkleRoot:    h.MerkleRoot().String(),
			Timestamp:     h.Timestamp(),
			Bits:          h.Bits(),
			Nonce:         h.Nonce(),
		},
		Txs: txPayloads,
	}
}

// ToBlock converts a BlockPayload back to a domain Block.
func (bp BlockPayload) ToBlock() (*block.Block, error) {
	hash, err := block.HashFromHex(bp.Hash)
	if err != nil {
		return nil, fmt.Errorf("invalid block hash: %w", err)
	}

	prevHash, err := block.HashFromHex(bp.Header.PrevBlockHash)
	if err != nil {
		return nil, fmt.Errorf("invalid prev block hash: %w", err)
	}

	merkleRoot, err := block.HashFromHex(bp.Header.MerkleRoot)
	if err != nil {
		return nil, fmt.Errorf("invalid merkle root: %w", err)
	}

	header := block.NewHeader(
		bp.Header.Version,
		prevHash,
		merkleRoot,
		bp.Header.Timestamp,
		bp.Header.Bits,
	)
	header.SetNonce(bp.Header.Nonce)

	txs := make([]any, 0, len(bp.Txs))
	for _, txp := range bp.Txs {
		t, err := txp.ToTransaction()
		if err != nil {
			return nil, fmt.Errorf("convert tx payload: %w", err)
		}
		txs = append(txs, t)
	}

	return block.ReconstructBlock(header, hash, bp.Height, "", txs), nil
}

// TxPayloadFromDomain converts a domain Transaction to a P2P TxPayload.
func TxPayloadFromDomain(t *tx.Transaction) TxPayload {
	inputs := make([]TxInputPayload, len(t.Inputs()))
	for i, in := range t.Inputs() {
		inputs[i] = TxInputPayload{
			TxID:      in.TxID().String(),
			Vout:      in.Vout(),
			Signature: hex.EncodeToString(in.Signature()),
			PubKey:    hex.EncodeToString(in.PubKey()),
		}
	}

	outputs := make([]TxOutputPayload, len(t.Outputs()))
	for i, out := range t.Outputs() {
		outputs[i] = TxOutputPayload{
			Value:   out.Value(),
			Address: out.Address(),
		}
	}

	return TxPayload{
		ID:      t.ID().String(),
		Inputs:  inputs,
		Outputs: outputs,
	}
}

// ToTransaction converts a TxPayload back to a domain Transaction.
func (tp TxPayload) ToTransaction() (*tx.Transaction, error) {
	txID, err := block.HashFromHex(tp.ID)
	if err != nil {
		return nil, fmt.Errorf("invalid tx id: %w", err)
	}

	inputs := make([]tx.TxInput, len(tp.Inputs))
	for i, in := range tp.Inputs {
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

	outputs := make([]tx.TxOutput, len(tp.Outputs))
	for i, out := range tp.Outputs {
		outputs[i] = tx.NewTxOutput(out.Value, out.Address)
	}

	return tx.ReconstructTransaction(txID, inputs, outputs), nil
}
