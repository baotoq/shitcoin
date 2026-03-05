package tx

import "github.com/btcsuite/btcd/btcec/v2"

// SignTransaction signs all non-coinbase inputs of a transaction with the given private key.
// For each input, it computes the transaction hash (without signatures), signs it using
// ECDSA, and stores the serialized signature and compressed public key on the input.
func SignTransaction(tx *Transaction, privKey *btcec.PrivateKey) error {
	panic("not implemented")
}

// VerifyTransaction verifies the ECDSA signatures on all non-coinbase transaction inputs.
// Returns true if all signatures are valid, false otherwise.
// Coinbase transactions are always considered valid (no signatures to verify).
func VerifyTransaction(tx *Transaction) bool {
	panic("not implemented")
}
