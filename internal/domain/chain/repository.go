package chain

import (
	"context"

	"github.com/baotoq/shitcoin/internal/domain/block"
	"github.com/baotoq/shitcoin/internal/domain/utxo"
)

// Repository defines the persistence interface for the chain aggregate.
// Interface lives in the domain layer; implementation in infrastructure.
type Repository interface {
	// SaveBlock persists a block to storage.
	SaveBlock(ctx context.Context, b *block.Block) error

	// SaveBlockWithUTXOs persists a block along with its UTXO changes atomically.
	// The undo entry records all UTXO mutations for reversibility.
	SaveBlockWithUTXOs(ctx context.Context, b *block.Block, undoEntry *utxo.UndoEntry) error

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

	// GetUndoEntry retrieves the UTXO undo entry for a block at the given height.
	GetUndoEntry(ctx context.Context, blockHeight uint64) (*utxo.UndoEntry, error)

	// DeleteBlocksAbove removes all blocks above the given height.
	// Used during reorganization to remove orphaned blocks.
	DeleteBlocksAbove(ctx context.Context, height uint64) error
}
