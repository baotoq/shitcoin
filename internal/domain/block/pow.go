package block

import (
	"math"
	"math/big"
)

// ProofOfWork is a stateless domain service that mines blocks by finding
// a nonce that produces a hash below the difficulty target.
type ProofOfWork struct{}

// BitsToTarget converts a compact difficulty representation (number of leading
// zero bits) to a full 256-bit target value.
//
// target = 1 << (256 - bits)
//
// A smaller target (more bits) means higher difficulty.
func BitsToTarget(bits uint32) *big.Int {
	target := big.NewInt(1)
	target.Lsh(target, uint(256-bits))
	return target
}

// Mine searches for a nonce that makes the block's header hash fall below
// the difficulty target. On success, it sets the block's nonce and hash.
//
// Returns ErrNonceExhausted if all uint32 nonce values are tried without
// finding a valid hash.
func (pow *ProofOfWork) Mine(b *Block) error {
	return pow.MineWithMaxNonce(b, math.MaxUint32)
}

// MineWithMaxNonce is like Mine but limits the nonce search space to [0, maxNonce].
// Useful for testing nonce exhaustion without iterating all 2^32 values.
func (pow *ProofOfWork) MineWithMaxNonce(b *Block, maxNonce uint32) error {
	target := BitsToTarget(b.header.bits)

	var nonce uint32
	for nonce <= maxNonce {
		b.header.SetNonce(nonce)
		hash := b.header.Hash()
		hashInt := new(big.Int).SetBytes(hash[:])

		if hashInt.Cmp(target) == -1 {
			b.hash = hash
			return nil
		}

		if nonce == maxNonce {
			break
		}
		nonce++
	}
	return ErrNonceExhausted
}

// Validate checks whether the block's header hash (recomputed from current
// header state) falls below the difficulty target derived from the block's bits.
//
// Returns true if the block is valid, false otherwise.
func (pow *ProofOfWork) Validate(b *Block) bool {
	target := BitsToTarget(b.header.bits)
	hash := b.header.Hash()
	hashInt := new(big.Int).SetBytes(hash[:])
	return hashInt.Cmp(target) == -1
}
