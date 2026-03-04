package bbolt

import (
	"context"

	"github.com/baotoq/shitcoin/internal/domain/block"
	bolt "go.etcd.io/bbolt"
)

// BboltRepository implements chain.Repository using bbolt as the storage engine.
type BboltRepository struct {
	db *bolt.DB
}

// NewBboltRepository creates a new BboltRepository and ensures required buckets exist.
func NewBboltRepository(db *bolt.DB) (*BboltRepository, error) {
	panic("not implemented")
}

func (r *BboltRepository) SaveBlock(ctx context.Context, b *block.Block) error {
	panic("not implemented")
}

func (r *BboltRepository) GetBlock(ctx context.Context, hash block.Hash) (*block.Block, error) {
	panic("not implemented")
}

func (r *BboltRepository) GetBlockByHeight(ctx context.Context, height uint64) (*block.Block, error) {
	panic("not implemented")
}

func (r *BboltRepository) GetLatestBlock(ctx context.Context) (*block.Block, error) {
	panic("not implemented")
}

func (r *BboltRepository) GetChainHeight(ctx context.Context) (uint64, error) {
	panic("not implemented")
}

func (r *BboltRepository) GetBlocksInRange(ctx context.Context, startHeight, endHeight uint64) ([]*block.Block, error) {
	panic("not implemented")
}
