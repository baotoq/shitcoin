package block

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
)

// Hash is a 32-byte value object representing a SHA-256d hash.
// Value semantics: immutable once created, compared by value.
type Hash [32]byte

// String returns the hex-encoded representation of the hash (64 characters).
func (h Hash) String() string {
	return hex.EncodeToString(h[:])
}

// IsZero returns true if the hash is all zeros (the zero value).
func (h Hash) IsZero() bool {
	return h == Hash{}
}

// Bytes returns the hash as a byte slice.
func (h Hash) Bytes() []byte {
	return h[:]
}

// DoubleSHA256 computes SHA-256(SHA-256(data)), the double-hash used in Bitcoin.
func DoubleSHA256(data []byte) Hash {
	first := sha256.Sum256(data)
	second := sha256.Sum256(first[:])
	return Hash(second)
}

// HashFromHex creates a Hash from a hex-encoded string.
// Returns an error if the string is not valid hex or not exactly 32 bytes.
func HashFromHex(s string) (Hash, error) {
	b, err := hex.DecodeString(s)
	if err != nil {
		return Hash{}, fmt.Errorf("invalid hex string: %w", err)
	}
	if len(b) != 32 {
		return Hash{}, fmt.Errorf("invalid hash length: got %d bytes, want 32", len(b))
	}
	var h Hash
	copy(h[:], b)
	return h, nil
}
