package wallet

import (
	"crypto/sha256"
	"math/big"
)

// base58Alphabet is the Bitcoin Base58 alphabet (no 0, O, I, l).
const base58Alphabet = "123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz"

// Base58Encode encodes a byte slice to a Base58 string.
// Uses big.Int division by 58, preserving leading zero bytes as '1' characters.
func Base58Encode(input []byte) string {
	if len(input) == 0 {
		return ""
	}

	var result []byte
	x := new(big.Int).SetBytes(input)
	base := big.NewInt(58)
	zero := big.NewInt(0)
	mod := new(big.Int)

	for x.Cmp(zero) > 0 {
		x.DivMod(x, base, mod)
		result = append(result, base58Alphabet[mod.Int64()])
	}

	// Preserve leading zero bytes as '1' characters.
	for _, b := range input {
		if b != 0x00 {
			break
		}
		result = append(result, base58Alphabet[0])
	}

	// Reverse the result.
	for i, j := 0, len(result)-1; i < j; i, j = i+1, j-1 {
		result[i], result[j] = result[j], result[i]
	}

	return string(result)
}

// Base58Decode decodes a Base58 string back to a byte slice.
// Reverses the Base58Encode process, restoring leading zero bytes from leading '1' characters.
func Base58Decode(input string) []byte {
	if len(input) == 0 {
		return nil
	}

	result := big.NewInt(0)
	base := big.NewInt(58)

	for _, c := range input {
		idx := int64(0)
		for i, ac := range base58Alphabet {
			if ac == c {
				idx = int64(i)
				break
			}
		}
		result.Mul(result, base)
		result.Add(result, big.NewInt(idx))
	}

	decoded := result.Bytes()

	// Restore leading zero bytes from leading '1' characters.
	numLeadingOnes := 0
	for _, c := range input {
		if c != rune(base58Alphabet[0]) {
			break
		}
		numLeadingOnes++
	}

	if numLeadingOnes > 0 {
		leadingZeros := make([]byte, numLeadingOnes)
		decoded = append(leadingZeros, decoded...)
	}

	return decoded
}

// Base58CheckEncode encodes payload with a version byte and 4-byte double-SHA256 checksum.
func Base58CheckEncode(version byte, payload []byte) string {
	versionedPayload := make([]byte, 0, 1+len(payload)+4)
	versionedPayload = append(versionedPayload, version)
	versionedPayload = append(versionedPayload, payload...)

	firstHash := sha256.Sum256(versionedPayload)
	secondHash := sha256.Sum256(firstHash[:])
	checksum := secondHash[:4]

	fullPayload := append(versionedPayload, checksum...)
	return Base58Encode(fullPayload)
}

// Base58CheckDecode decodes a Base58Check string, verifying the checksum.
// Returns the version byte, payload, and an error if the checksum is invalid.
func Base58CheckDecode(input string) (version byte, payload []byte, err error) {
	decoded := Base58Decode(input)
	if len(decoded) < 5 {
		return 0, nil, ErrInvalidAddress
	}

	// Split into versioned payload and checksum.
	versionedPayload := decoded[:len(decoded)-4]
	checksum := decoded[len(decoded)-4:]

	// Verify checksum.
	firstHash := sha256.Sum256(versionedPayload)
	secondHash := sha256.Sum256(firstHash[:])
	expectedChecksum := secondHash[:4]

	for i := range 4 {
		if checksum[i] != expectedChecksum[i] {
			return 0, nil, ErrInvalidChecksum
		}
	}

	return versionedPayload[0], versionedPayload[1:], nil
}
