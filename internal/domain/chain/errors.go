package chain

import "errors"

var (
	// ErrBlockNotFound is returned when a requested block does not exist.
	ErrBlockNotFound = errors.New("block not found")

	// ErrChainEmpty is returned when the chain has no blocks.
	ErrChainEmpty = errors.New("chain is empty")

	// ErrInvalidPrevHash is returned when a block's previous hash does not match.
	ErrInvalidPrevHash = errors.New("invalid previous block hash")
)
