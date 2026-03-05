package block

import (
	"math/big"
	"testing"
)

func TestBitsToTarget(t *testing.T) {
	tests := []struct {
		name string
		bits uint32
		want *big.Int
	}{
		{
			name: "bits=16 produces 2^240",
			bits: 16,
			want: new(big.Int).Lsh(big.NewInt(1), 240),
		},
		{
			name: "bits=1 produces 2^255",
			bits: 1,
			want: new(big.Int).Lsh(big.NewInt(1), 255),
		},
		{
			name: "bits=8 produces 2^248",
			bits: 8,
			want: new(big.Int).Lsh(big.NewInt(1), 248),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := BitsToTarget(tt.bits)
			if got.Cmp(tt.want) != 0 {
				t.Errorf("BitsToTarget(%d) = %s; want %s", tt.bits, got.String(), tt.want.String())
			}
		})
	}
}

func TestMineGenesisBlock(t *testing.T) {
	b, err := NewGenesisBlock("Test Genesis", 16, nil, Hash{})
	if err != nil {
		t.Fatalf("NewGenesisBlock failed: %v", err)
	}

	pow := &ProofOfWork{}
	if err := pow.Mine(b); err != nil {
		t.Fatalf("Mine failed: %v", err)
	}

	// Hash should be set (non-zero)
	if b.Hash().IsZero() {
		t.Error("mined block hash should not be zero")
	}

	// Hash should be below target
	target := BitsToTarget(16)
	hashInt := new(big.Int).SetBytes(b.Hash().Bytes())
	if hashInt.Cmp(target) >= 0 {
		t.Errorf("mined hash %s is not below target", b.Hash().String())
	}
}

func TestMineValidation(t *testing.T) {
	b, err := NewGenesisBlock("Validation Test", 16, nil, Hash{})
	if err != nil {
		t.Fatalf("NewGenesisBlock failed: %v", err)
	}

	pow := &ProofOfWork{}
	if err := pow.Mine(b); err != nil {
		t.Fatalf("Mine failed: %v", err)
	}

	// Valid mined block should pass validation
	if !pow.Validate(b) {
		t.Error("Validate should return true for a properly mined block")
	}

	// Tamper with the nonce -- validation should fail
	originalNonce := b.Header().Nonce()
	b.SetHeaderNonce(originalNonce + 1)
	// Recompute hash with tampered nonce (the stored hash is now stale)
	tamperedHash := b.Header().Hash()
	b.SetHash(tamperedHash)

	// The tampered block may or may not validate depending on the new hash.
	// So instead, set a deliberately bad hash.
	b.SetHeaderNonce(originalNonce + 1)
	badHash := DoubleSHA256([]byte("definitely not a valid block"))
	b.SetHash(badHash)

	// This should almost certainly fail validation since the hash won't match header hash
	// Note: Validate recomputes the header hash, so it checks against that
	// Let's do it properly: just change the nonce and check that validate
	// uses the header's current state
	b2, _ := NewGenesisBlock("Tamper Test", 16, nil, Hash{})
	pow.Mine(b2)

	// Save the good state
	goodNonce := b2.Header().Nonce()

	// Tamper nonce to something invalid
	b2.SetHeaderNonce(goodNonce + 7)
	if pow.Validate(b2) {
		t.Error("Validate should return false after tampering nonce")
	}

	// Restore the good nonce -- should validate again
	b2.SetHeaderNonce(goodNonce)
	if !pow.Validate(b2) {
		t.Error("Validate should return true after restoring correct nonce")
	}
}

func TestMineDeterministic(t *testing.T) {
	// Same header with same nonce should produce same hash
	h := NewHeader(1, Hash{}, Hash{}, 1700000000, 16)

	h.SetNonce(42)
	hash1 := h.Hash()

	h.SetNonce(42)
	hash2 := h.Hash()

	if hash1 != hash2 {
		t.Errorf("same header+nonce produced different hashes: %s != %s",
			hash1.String(), hash2.String())
	}
}

func TestMineNonceExhausted(t *testing.T) {
	// bits=256 means target = 2^0 = 1, which is effectively impossible
	// (hash must be < 1, meaning all zeros)
	// Use MineWithMaxNonce to limit search space and avoid 4B iterations
	b, err := NewGenesisBlock("Impossible", 256, nil, Hash{})
	if err != nil {
		t.Fatalf("NewGenesisBlock failed: %v", err)
	}

	pow := &ProofOfWork{}
	err = pow.MineWithMaxNonce(b, 1000)
	if err != ErrNonceExhausted {
		t.Errorf("expected ErrNonceExhausted, got: %v", err)
	}
}

func TestMineMultipleBlocks(t *testing.T) {
	pow := &ProofOfWork{}

	// Mine genesis
	genesis, err := NewGenesisBlock("Multi-block test", 16, nil, Hash{})
	if err != nil {
		t.Fatalf("NewGenesisBlock failed: %v", err)
	}
	if err := pow.Mine(genesis); err != nil {
		t.Fatalf("Mine genesis failed: %v", err)
	}
	if !pow.Validate(genesis) {
		t.Fatal("genesis block validation failed")
	}

	// Mine block 1
	block1, err := NewBlock(genesis.Hash(), 1, 16, nil, Hash{})
	if err != nil {
		t.Fatalf("NewBlock(1) failed: %v", err)
	}
	if err := pow.Mine(block1); err != nil {
		t.Fatalf("Mine block 1 failed: %v", err)
	}
	if !pow.Validate(block1) {
		t.Fatal("block 1 validation failed")
	}

	// Mine block 2
	block2, err := NewBlock(block1.Hash(), 2, 16, nil, Hash{})
	if err != nil {
		t.Fatalf("NewBlock(2) failed: %v", err)
	}
	if err := pow.Mine(block2); err != nil {
		t.Fatalf("Mine block 2 failed: %v", err)
	}
	if !pow.Validate(block2) {
		t.Fatal("block 2 validation failed")
	}

	// Verify chain links
	if block1.PrevBlockHash() != genesis.Hash() {
		t.Error("block 1 should reference genesis hash")
	}
	if block2.PrevBlockHash() != block1.Hash() {
		t.Error("block 2 should reference block 1 hash")
	}

	// All blocks should have unique hashes
	if genesis.Hash() == block1.Hash() {
		t.Error("genesis and block 1 should have different hashes")
	}
	if block1.Hash() == block2.Hash() {
		t.Error("block 1 and block 2 should have different hashes")
	}
}
