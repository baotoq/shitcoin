package block

import (
	"crypto/sha256"
	"encoding/hex"
	"testing"
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
			got := tt.hash.String()
			if len(got) != 64 {
				t.Errorf("String() length = %d; want 64", len(got))
			}
			if got != tt.want {
				t.Errorf("String() = %q; want %q", got, tt.want)
			}
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
			got := tt.hash.IsZero()
			if got != tt.want {
				t.Errorf("IsZero() = %v; want %v", got, tt.want)
			}
		})
	}
}

func TestHashBytes(t *testing.T) {
	var h Hash
	h[0] = 0xff
	h[31] = 0x01

	b := h.Bytes()
	if len(b) != 32 {
		t.Errorf("Bytes() length = %d; want 32", len(b))
	}
	if b[0] != 0xff {
		t.Errorf("Bytes()[0] = %x; want ff", b[0])
	}
	if b[31] != 0x01 {
		t.Errorf("Bytes()[31] = %x; want 01", b[31])
	}
}

func TestDoubleSHA256(t *testing.T) {
	// Known test vector: DoubleSHA256 of "hello"
	// First: SHA256("hello") = 2cf24dba5fb0a30e26e83b2ac5b9e29e1b161e5c1fa7425e73043362938b9824
	// Second: SHA256(first)
	data := []byte("hello")
	first := sha256.Sum256(data)
	expectedBytes := sha256.Sum256(first[:])

	got := DoubleSHA256(data)

	if got != Hash(expectedBytes) {
		t.Errorf("DoubleSHA256(\"hello\") = %s; want %s",
			got.String(), hex.EncodeToString(expectedBytes[:]))
	}
}

func TestDoubleSHA256Deterministic(t *testing.T) {
	data := []byte("deterministic test input")
	hash1 := DoubleSHA256(data)
	hash2 := DoubleSHA256(data)

	if hash1 != hash2 {
		t.Errorf("DoubleSHA256 not deterministic: %s != %s", hash1.String(), hash2.String())
	}
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
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			// Roundtrip: hex -> Hash -> hex should match
			if h.String() != tt.input {
				t.Errorf("roundtrip failed: got %q; want %q", h.String(), tt.input)
			}
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

	if string(payload1) != string(payload2) {
		t.Error("HashPayload() not deterministic: two calls produced different results")
	}
}

func TestHeaderHashDeterministic(t *testing.T) {
	h := NewHeader(1, Hash{}, Hash{}, 1700000000, 16)

	hash1 := h.Hash()
	hash2 := h.Hash()

	if hash1 != hash2 {
		t.Errorf("Header.Hash() not deterministic: %s != %s", hash1.String(), hash2.String())
	}
}

func TestHeaderGetters(t *testing.T) {
	var prevHash Hash
	prevHash[0] = 0x01
	var merkleRoot Hash
	merkleRoot[0] = 0x02

	h := NewHeader(1, prevHash, merkleRoot, 1700000000, 16)

	if h.Version() != 1 {
		t.Errorf("Version() = %d; want 1", h.Version())
	}
	if h.PrevBlockHash() != prevHash {
		t.Error("PrevBlockHash() mismatch")
	}
	if h.MerkleRoot() != merkleRoot {
		t.Error("MerkleRoot() mismatch")
	}
	if h.Timestamp() != 1700000000 {
		t.Errorf("Timestamp() = %d; want 1700000000", h.Timestamp())
	}
	if h.Bits() != 16 {
		t.Errorf("Bits() = %d; want 16", h.Bits())
	}
	if h.Nonce() != 0 {
		t.Errorf("Nonce() = %d; want 0", h.Nonce())
	}
}

func TestHeaderSetNonce(t *testing.T) {
	h := NewHeader(1, Hash{}, Hash{}, 1700000000, 16)

	h.SetNonce(42)
	if h.Nonce() != 42 {
		t.Errorf("after SetNonce(42), Nonce() = %d; want 42", h.Nonce())
	}

	// Different nonce should produce different hash
	hash1 := h.Hash()
	h.SetNonce(43)
	hash2 := h.Hash()

	if hash1 == hash2 {
		t.Error("different nonces should produce different hashes")
	}
}

// --- Block entity tests ---

func TestNewGenesisBlock(t *testing.T) {
	b, err := NewGenesisBlock("Hello, Shitcoin!", 16, nil, Hash{})
	if err != nil {
		t.Fatalf("NewGenesisBlock failed: %v", err)
	}

	if b.Height() != 0 {
		t.Errorf("genesis Height() = %d; want 0", b.Height())
	}
	if !b.PrevBlockHash().IsZero() {
		t.Error("genesis PrevBlockHash() should be zero")
	}
	if b.Message() != "Hello, Shitcoin!" {
		t.Errorf("genesis Message() = %q; want %q", b.Message(), "Hello, Shitcoin!")
	}
	if b.Bits() != 16 {
		t.Errorf("genesis Bits() = %d; want 16", b.Bits())
	}
	if b.Timestamp() == 0 {
		t.Error("genesis Timestamp() should not be zero")
	}
	if b.Hash().IsZero() == false {
		// Hash should be zero before mining
	}
	if len(b.RawTransactions()) != 0 {
		t.Errorf("genesis Transactions() length = %d; want 0", len(b.RawTransactions()))
	}
}

func TestNewBlock(t *testing.T) {
	var prevHash Hash
	prevHash[0] = 0xab

	b, err := NewBlock(prevHash, 1, 16, nil, Hash{})
	if err != nil {
		t.Fatalf("NewBlock failed: %v", err)
	}

	if b.Height() != 1 {
		t.Errorf("Height() = %d; want 1", b.Height())
	}
	if b.PrevBlockHash() != prevHash {
		t.Error("PrevBlockHash() mismatch")
	}
	if b.Bits() != 16 {
		t.Errorf("Bits() = %d; want 16", b.Bits())
	}
	if b.Timestamp() == 0 {
		t.Error("Timestamp() should not be zero")
	}
}

func TestBlockSetHashAndNonce(t *testing.T) {
	b, err := NewGenesisBlock("test", 16, nil, Hash{})
	if err != nil {
		t.Fatalf("NewGenesisBlock failed: %v", err)
	}

	var h Hash
	h[0] = 0xde
	h[1] = 0xad

	b.SetHash(h)
	if b.Hash() != h {
		t.Error("SetHash did not set hash correctly")
	}

	b.SetHeaderNonce(12345)
	if b.Header().Nonce() != 12345 {
		t.Errorf("SetHeaderNonce(12345): got Nonce() = %d; want 12345", b.Header().Nonce())
	}
}

func TestReconstructBlock(t *testing.T) {
	header := NewHeader(1, Hash{}, Hash{}, 1700000000, 16)
	header.SetNonce(42)

	var hash Hash
	hash[0] = 0x01

	b := ReconstructBlock(header, hash, 5, "genesis msg", nil)

	if b.Height() != 5 {
		t.Errorf("Height() = %d; want 5", b.Height())
	}
	if b.Hash() != hash {
		t.Error("Hash() mismatch")
	}
	if b.Message() != "genesis msg" {
		t.Errorf("Message() = %q; want %q", b.Message(), "genesis msg")
	}
	if b.Header().Nonce() != 42 {
		t.Errorf("Header().Nonce() = %d; want 42", b.Header().Nonce())
	}
}
