package block

import (
	"fmt"
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

// MiningProgress holds sampled mining state for progress reporting.
type MiningProgress struct {
	Nonce      uint32
	Hash       string
	Target     string
	Difficulty uint32
}

// MineWithProgress is like Mine but calls onProgress every sampleRate nonce
// attempts with the current mining state. If onProgress is nil, callbacks are
// skipped (safe to call without a dashboard).
func (pow *ProofOfWork) MineWithProgress(b *Block, sampleRate uint32, onProgress func(MiningProgress)) error {
	target := BitsToTarget(b.header.bits)
	targetHex := fmt.Sprintf("%064x", target)

	var nonce uint32
	for nonce <= math.MaxUint32 {
		b.header.SetNonce(nonce)
		hash := b.header.Hash()
		hashInt := new(big.Int).SetBytes(hash[:])

		if onProgress != nil && nonce%sampleRate == 0 {
			onProgress(MiningProgress{
				Nonce:      nonce,
				Hash:       hash.String(),
				Target:     targetHex,
				Difficulty: b.header.bits,
			})
		}

		if hashInt.Cmp(target) == -1 {
			b.hash = hash
			return nil
		}

		if nonce == math.MaxUint32 {
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
