package block

import (
	"crypto/sha256"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- Hash value object tests ---

func TestHashString(t *testing.T) {
	tests := []struct {
		name string
		hash Hash
		want string
	}{
		{
			name: "zero hash",
			hash: Hash{},
			want: "0000000000000000000000000000000000000000000000000000000000000000",
		},
		{
			name: "non-zero hash",
			hash: func() Hash {
				var h Hash
				h[0] = 0xab
				h[31] = 0xcd
				return h
			}(),
			want: "ab000000000000000000000000000000000000000000000000000000000000cd",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)

			got := tt.hash.String()
			assert.Len(got, 64)
			assert.Equal(tt.want, got)
		})
	}
}

func TestHashIsZero(t *testing.T) {
	tests := []struct {
		name string
		hash Hash
		want bool
	}{
		{
			name: "zero hash",
			hash: Hash{},
			want: true,
		},
		{
			name: "non-zero hash",
			hash: func() Hash {
				var h Hash
				h[0] = 1
				return h
			}(),
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.hash.IsZero())
		})
	}
}

func TestHashBytes(t *testing.T) {
	var h Hash
	h[0] = 0xff
	h[31] = 0x01

	b := h.Bytes()
	assert.Len(t, b, 32)
	assert.Equal(t, byte(0xff), b[0])
	assert.Equal(t, byte(0x01), b[31])
}

func TestDoubleSHA256(t *testing.T) {
	// Known test vector: DoubleSHA256 of "hello"
	// First: SHA256("hello") = 2cf24dba5fb0a30e26e83b2ac5b9e29e1b161e5c1fa7425e73043362938b9824
	// Second: SHA256(first)
	data := []byte("hello")
	first := sha256.Sum256(data)
	expectedBytes := sha256.Sum256(first[:])

	got := DoubleSHA256(data)

	assert.Equal(t, Hash(expectedBytes), got)
}

func TestDoubleSHA256Deterministic(t *testing.T) {
	data := []byte("deterministic test input")
	hash1 := DoubleSHA256(data)
	hash2 := DoubleSHA256(data)

	assert.Equal(t, hash1, hash2)
}

func TestHashFromHex(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{
			name:    "valid hex string",
			input:   "0000000000000000000000000000000000000000000000000000000000000001",
			wantErr: false,
		},
		{
			name:    "invalid hex",
			input:   "not-hex",
			wantErr: true,
		},
		{
			name:    "wrong length",
			input:   "abcd",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, err := HashFromHex(tt.input)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			// Roundtrip: hex -> Hash -> hex should match
			assert.Equal(t, tt.input, h.String())
		})
	}
}

// --- Header value object tests ---

func TestHeaderHashPayloadDeterministic(t *testing.T) {
	var prevHash Hash
	prevHash[0] = 0xaa
	var merkleRoot Hash
	merkleRoot[0] = 0xbb

	h := NewHeader(1, prevHash, merkleRoot, 1700000000, 16)

	payload1 := h.HashPayload()
	payload2 := h.HashPayload()

	assert.Equal(t, string(payload1), string(payload2))
}

func TestHeaderHashDeterministic(t *testing.T) {
	h := NewHeader(1, Hash{}, Hash{}, 1700000000, 16)

	hash1 := h.Hash()
	hash2 := h.Hash()

	assert.Equal(t, hash1, hash2)
}

func TestHeaderGetters(t *testing.T) {
	var prevHash Hash
	prevHash[0] = 0x01
	var merkleRoot Hash
	merkleRoot[0] = 0x02

	h := NewHeader(1, prevHash, merkleRoot, 1700000000, 16)

	assert.Equal(t, uint32(1), h.Version())
	assert.Equal(t, prevHash, h.PrevBlockHash())
	assert.Equal(t, merkleRoot, h.MerkleRoot())
	assert.Equal(t, int64(1700000000), h.Timestamp())
	assert.Equal(t, uint32(16), h.Bits())
	assert.Equal(t, uint32(0), h.Nonce())
}

func TestHeaderSetNonce(t *testing.T) {
	h := NewHeader(1, Hash{}, Hash{}, 1700000000, 16)

	h.SetNonce(42)
	assert.Equal(t, uint32(42), h.Nonce())

	// Different nonce should produce different hash
	hash1 := h.Hash()
	h.SetNonce(43)
	hash2 := h.Hash()

	assert.NotEqual(t, hash1, hash2)
}

// --- Block entity tests ---

func TestNewGenesisBlock(t *testing.T) {
	b, err := NewGenesisBlock("Hello, Shitcoin!", 16, nil, Hash{})
	require.NoError(t, err)

	assert.Equal(t, uint64(0), b.Height())
	assert.True(t, b.PrevBlockHash().IsZero())
	assert.Equal(t, "Hello, Shitcoin!", b.Message())
	assert.Equal(t, uint32(16), b.Bits())
	assert.NotZero(t, b.Timestamp())
	assert.Empty(t, b.RawTransactions())
}

func TestNewBlock(t *testing.T) {
	var prevHash Hash
	prevHash[0] = 0xab

	b, err := NewBlock(prevHash, 1, 16, nil, Hash{})
	require.NoError(t, err)

	assert.Equal(t, uint64(1), b.Height())
	assert.Equal(t, prevHash, b.PrevBlockHash())
	assert.Equal(t, uint32(16), b.Bits())
	assert.NotZero(t, b.Timestamp())
}

func TestBlockSetHashAndNonce(t *testing.T) {
	b, err := NewGenesisBlock("test", 16, nil, Hash{})
	require.NoError(t, err)

	var h Hash
	h[0] = 0xde
	h[1] = 0xad

	b.SetHash(h)
	assert.Equal(t, h, b.Hash())

	b.SetHeaderNonce(12345)
	assert.Equal(t, uint32(12345), b.Header().Nonce())
}

func TestReconstructBlock(t *testing.T) {
	header := NewHeader(1, Hash{}, Hash{}, 1700000000, 16)
	header.SetNonce(42)

	var hash Hash
	hash[0] = 0x01

	b := ReconstructBlock(header, hash, 5, "genesis msg", nil)

	assert.Equal(t, uint64(5), b.Height())
	assert.Equal(t, hash, b.Hash())
	assert.Equal(t, "genesis msg", b.Message())
	assert.Equal(t, uint32(42), b.Header().Nonce())
}
