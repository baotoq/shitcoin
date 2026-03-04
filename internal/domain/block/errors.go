package block

import "errors"

var (
	// ErrNonceExhausted is returned when the mining loop exhausts all uint32 nonce values
	// without finding a hash below the difficulty target.
	ErrNonceExhausted = errors.New("nonce exhausted: no valid hash found")

	// ErrInvalidBlock is returned when a block fails validation checks.
	ErrInvalidBlock = errors.New("invalid block")
)
