package wallet

import (
	"github.com/btcsuite/btcd/btcec/v2"
)

// PubKeyToAddress derives a Bitcoin-style P2PKH address from a compressed public key.
// Pipeline: compressed pubkey -> SHA-256 -> RIPEMD-160 -> Base58Check(version 0x00).
func PubKeyToAddress(pubKey *btcec.PublicKey) string {
	panic("not implemented")
}

// PubKeyHashFromAddress extracts the 20-byte public key hash from a Base58Check address.
func PubKeyHashFromAddress(address string) ([]byte, error) {
	panic("not implemented")
}
