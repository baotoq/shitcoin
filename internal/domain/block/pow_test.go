package block

import (
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
			assert.Equal(t, 0, got.Cmp(tt.want), "BitsToTarget(%d) = %s; want %s", tt.bits, got.String(), tt.want.String())
		})
	}
}

func TestMineGenesisBlock(t *testing.T) {
	b, err := NewGenesisBlock("Test Genesis", 16, nil, Hash{})
	require.NoError(t, err)

	pow := &ProofOfWork{}
	require.NoError(t, pow.Mine(b))

	// Hash should be set (non-zero)
	assert.False(t, b.Hash().IsZero(), "mined block hash should not be zero")

	// Hash should be below target
	target := BitsToTarget(16)
	hashInt := new(big.Int).SetBytes(b.Hash().Bytes())
	assert.Equal(t, -1, hashInt.Cmp(target), "mined hash should be below target")
}

func TestMineValidation(t *testing.T) {
	b, err := NewGenesisBlock("Validation Test", 16, nil, Hash{})
	require.NoError(t, err)

	pow := &ProofOfWork{}
	require.NoError(t, pow.Mine(b))

	// Valid mined block should pass validation
	assert.True(t, pow.Validate(b), "Validate should return true for a properly mined block")

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
	assert.False(t, pow.Validate(b2), "Validate should return false after tampering nonce")

	// Restore the good nonce -- should validate again
	b2.SetHeaderNonce(goodNonce)
	assert.True(t, pow.Validate(b2), "Validate should return true after restoring correct nonce")
}

func TestMineDeterministic(t *testing.T) {
	// Same header with same nonce should produce same hash
	h := NewHeader(1, Hash{}, Hash{}, 1700000000, 16)

	h.SetNonce(42)
	hash1 := h.Hash()

	h.SetNonce(42)
	hash2 := h.Hash()

	assert.Equal(t, hash1, hash2)
}

func TestMineNonceExhausted(t *testing.T) {
	// bits=256 means target = 2^0 = 1, which is effectively impossible
	// (hash must be < 1, meaning all zeros)
	// Use MineWithMaxNonce to limit search space and avoid 4B iterations
	b, err := NewGenesisBlock("Impossible", 256, nil, Hash{})
	require.NoError(t, err)

	pow := &ProofOfWork{}
	err = pow.MineWithMaxNonce(b, 1000)
	assert.ErrorIs(t, err, ErrNonceExhausted)
}

func TestMineMultipleBlocks(t *testing.T) {
	pow := &ProofOfWork{}

	// Mine genesis
	genesis, err := NewGenesisBlock("Multi-block test", 16, nil, Hash{})
	require.NoError(t, err)
	require.NoError(t, pow.Mine(genesis))
	require.True(t, pow.Validate(genesis))

	// Mine block 1
	block1, err := NewBlock(genesis.Hash(), 1, 16, nil, Hash{})
	require.NoError(t, err)
	require.NoError(t, pow.Mine(block1))
	require.True(t, pow.Validate(block1))

	// Mine block 2
	block2, err := NewBlock(block1.Hash(), 2, 16, nil, Hash{})
	require.NoError(t, err)
	require.NoError(t, pow.Mine(block2))
	require.True(t, pow.Validate(block2))

	// Verify chain links
	assert.Equal(t, genesis.Hash(), block1.PrevBlockHash())
	assert.Equal(t, block1.Hash(), block2.PrevBlockHash())

	// All blocks should have unique hashes
	assert.NotEqual(t, genesis.Hash(), block1.Hash())
	assert.NotEqual(t, block1.Hash(), block2.Hash())
}
