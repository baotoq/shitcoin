package wallet

// base58Alphabet is the Bitcoin Base58 alphabet (no 0, O, I, l).
const base58Alphabet = "123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz"

// Base58Encode encodes a byte slice to a Base58 string.
func Base58Encode(input []byte) string {
	panic("not implemented")
}

// Base58Decode decodes a Base58 string back to a byte slice.
func Base58Decode(input string) []byte {
	panic("not implemented")
}

// Base58CheckEncode encodes payload with a version byte and 4-byte checksum.
func Base58CheckEncode(version byte, payload []byte) string {
	panic("not implemented")
}

// Base58CheckDecode decodes a Base58Check string, returning version and payload.
func Base58CheckDecode(input string) (version byte, payload []byte, err error) {
	panic("not implemented")
}
