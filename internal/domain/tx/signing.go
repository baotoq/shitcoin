package tx

import (
	"github.com/btcsuite/btcd/btcec/v2"
	"github.com/btcsuite/btcd/btcec/v2/ecdsa"
)

// SignTransaction signs all non-coinbase inputs of a transaction with the given private key.
// For each input, it computes the transaction hash (without signatures), signs it using
// ECDSA, and stores the serialized signature and compressed public key on the input.
func SignTransaction(tx *Transaction, privKey *btcec.PrivateKey) error {
	if tx.IsCoinbase() {
		return nil
	}

	txHash := tx.ComputeID()

	for i := range tx.inputs {
		sig := ecdsa.Sign(privKey, txHash.Bytes())
		tx.inputs[i].SetSignature(sig.Serialize())
		tx.inputs[i].SetPubKey(privKey.PubKey().SerializeCompressed())
	}

	return nil
}

// VerifyTransaction verifies the ECDSA signatures on all non-coinbase transaction inputs.
// Returns true if all signatures are valid, false otherwise.
// Coinbase transactions are always considered valid (no signatures to verify).
func VerifyTransaction(tx *Transaction) bool {
	if tx.IsCoinbase() {
		return true
	}

	txHash := tx.ComputeID()

	for _, input := range tx.inputs {
		if len(input.Signature()) == 0 || len(input.PubKey()) == 0 {
			return false
		}

		pubKey, err := btcec.ParsePubKey(input.PubKey())
		if err != nil {
			return false
		}

		sig, err := ecdsa.ParseSignature(input.Signature())
		if err != nil {
			return false
		}

		if !sig.Verify(txHash.Bytes(), pubKey) {
			return false
		}
	}

	return true
}
