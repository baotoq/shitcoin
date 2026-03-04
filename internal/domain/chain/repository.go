package chain

import (
	"context"

	"github.com/baotoq/shitcoin/internal/domain/block"
)

// Repository defines the persistence interface for the chain aggregate.
// Interface lives in the domain layer; implementation in infrastructure.
type Repository interface {
	// SaveBlock persists a block to storage.
	SaveBlock(ctx context.Context, b *block.Block) error

	// GetBlock retrieves a block by its hash.
	GetBlock(ctx context.Context, hash block.Hash) (*block.Block, error)

	// GetBlockByHeight retrieves a block at a specific height.
	GetBlockByHeight(ctx context.Context, height uint64) (*block.Block, error)

	// GetLatestBlock returns the most recently saved block.
	// Returns ErrChainEmpty if no blocks exist.
	GetLatestBlock(ctx context.Context) (*block.Block, error)

	// GetChainHeight returns the current chain height (0 if empty).
	GetChainHeight(ctx context.Context) (uint64, error)

	// GetBlocksInRange returns blocks from startHeight to endHeight inclusive.
	GetBlocksInRange(ctx context.Context, startHeight, endHeight uint64) ([]*block.Block, error)
}
