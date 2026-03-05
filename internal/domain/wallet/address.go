package wallet

import (
	"crypto/sha256"
	"fmt"

	"github.com/btcsuite/btcd/btcec/v2"
	"golang.org/x/crypto/ripemd160"
)

// PubKeyToAddress derives a Bitcoin-style P2PKH address from a compressed public key.
// Pipeline: compressed pubkey (33 bytes) -> SHA-256 -> RIPEMD-160 (20 bytes) -> Base58Check(version 0x00).
func PubKeyToAddress(pubKey *btcec.PublicKey) string {
	// 1. Compressed public key (33 bytes).
	pubKeyBytes := pubKey.SerializeCompressed()

	// 2. SHA-256.
	sha256Hash := sha256.Sum256(pubKeyBytes)

	// 3. RIPEMD-160.
	ripeHasher := ripemd160.New() //nolint:staticcheck // Required for Bitcoin P2PKH address derivation.
	ripeHasher.Write(sha256Hash[:])
	pubKeyHash := ripeHasher.Sum(nil) // 20 bytes

	// 4. Base58Check with version byte 0x00 (Bitcoin mainnet).
	return Base58CheckEncode(0x00, pubKeyHash)
}

// PubKeyHashFromAddress extracts the 20-byte public key hash from a Base58Check address.
// Verifies the version byte is 0x00 (P2PKH) and returns the 20-byte hash.
func PubKeyHashFromAddress(address string) ([]byte, error) {
	version, payload, err := Base58CheckDecode(address)
	if err != nil {
		return nil, fmt.Errorf("decode address: %w", err)
	}
	if version != 0x00 {
		return nil, fmt.Errorf("%w: unexpected version byte 0x%02x", ErrInvalidAddress, version)
	}
	if len(payload) != 20 {
		return nil, fmt.Errorf("%w: expected 20-byte hash, got %d bytes", ErrInvalidAddress, len(payload))
	}
	return payload, nil
}
