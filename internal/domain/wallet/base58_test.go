package wallet

import (
	"encoding/hex"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBase58Encode(t *testing.T) {
	tests := []struct {
		name     string
		inputHex string
		expected string
	}{
		{
			name:     "empty input returns empty string",
			inputHex: "",
			expected: "",
		},
		{
			name:     "known Bitcoin test vector",
			inputHex: "00010966776006953D5567439E5E39F86A0D273BEED61967F6",
			expected: "16UwLL9Risc3QfPqBUvKofHmBQ7wMtjvM",
		},
		{
			name:     "leading zero bytes produce leading 1 characters",
			inputHex: "0000010966776006953D5567439E5E39F86A0D273BEED61967F6",
			expected: "116UwLL9Risc3QfPqBUvKofHmBQ7wMtjvM",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require := require.New(t)
			assert := assert.New(t)

			input, err := hex.DecodeString(tt.inputHex)
			require.NoError(err)
			got := Base58Encode(input)
			assert.Equal(tt.expected, got)
		})
	}
}

func TestBase58Decode_RoundTrip(t *testing.T) {
	tests := []struct {
		name     string
		inputHex string
	}{
		{"single byte", "FF"},
		{"multiple bytes", "0102030405"},
		{"leading zeros", "000000FF"},
		{"bitcoin address payload", "00010966776006953D5567439E5E39F86A0D273BEED61967F6"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require := require.New(t)
			assert := assert.New(t)

			input, err := hex.DecodeString(tt.inputHex)
			require.NoError(err)
			encoded := Base58Encode(input)
			decoded := Base58Decode(encoded)
			decodedHex := hex.EncodeToString(decoded)
			inputHexLower := hex.EncodeToString(input)
			assert.Equal(inputHexLower, decodedHex)
		})
	}
}

func TestBase58CheckEncode(t *testing.T) {
	// Known test: version 0x00 with a known RIPEMD-160 hash should produce a valid address
	// Using the hash from the Bitcoin wiki Base58Check example
	payloadHex := "010966776006953D5567439E5E39F86A0D273BEE"
	payload, _ := hex.DecodeString(payloadHex)
	got := Base58CheckEncode(0x00, payload)

	// The expected address for this payload with version 0x00
	expected := "16UwLL9Risc3QfPqBUvKofHmBQ7wMtjvM"
	assert.Equal(t, expected, got)
}

func TestBase58CheckDecode(t *testing.T) {
	t.Run("valid address decodes correctly", func(t *testing.T) {
		address := "16UwLL9Risc3QfPqBUvKofHmBQ7wMtjvM"
		version, payload, err := Base58CheckDecode(address)
		require.NoError(t, err)
		assert.Equal(t, byte(0x00), version)
		expectedPayloadHex := "010966776006953d5567439e5e39f86a0d273bee"
		assert.Equal(t, expectedPayloadHex, hex.EncodeToString(payload))
	})

	t.Run("invalid checksum returns error", func(t *testing.T) {
		// Corrupt the last character of a valid address
		address := "16UwLL9Risc3QfPqBUvKofHmBQ7wMtjvN"
		_, _, err := Base58CheckDecode(address)
		require.Error(t, err)
	})

	t.Run("short input returns error", func(t *testing.T) {
		// A very short Base58 string that decodes to fewer than 5 bytes
		_, _, err := Base58CheckDecode("1")
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrInvalidAddress)
	})

	t.Run("empty input returns error", func(t *testing.T) {
		// Base58Decode of empty string returns nil, which is < 5 bytes
		_, _, err := Base58CheckDecode("")
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrInvalidAddress)
	})
}
