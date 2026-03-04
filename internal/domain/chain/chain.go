package chain

import (
	"context"

	"github.com/baotoq/shitcoin/internal/domain/block"
)

// ChainConfig holds consensus parameters for the chain aggregate.
type ChainConfig struct {
	BlockTimeTarget          int    // target seconds between blocks
	DifficultyAdjustInterval int    // blocks between difficulty adjustments
	InitialDifficulty        int    // initial bits value
	GenesisMessage           string // message embedded in genesis block
}

// Chain is the aggregate root that manages the block sequence, mining, and difficulty.
type Chain struct {
	repo         Repository
	pow          *block.ProofOfWork
	latestBlock  *block.Block
	config       ChainConfig
}

// NewChain creates a new Chain aggregate with the given dependencies.
func NewChain(repo Repository, pow *block.ProofOfWork, config ChainConfig) *Chain {
	return &Chain{
		repo:   repo,
		pow:    pow,
		config: config,
	}
}

// Initialize loads an existing chain or creates the genesis block.
func (c *Chain) Initialize(ctx context.Context) error {
	panic("not implemented")
}

// MineBlock creates, mines, and persists a new block.
func (c *Chain) MineBlock(ctx context.Context) (*block.Block, error) {
	panic("not implemented")
}

// LatestBlock returns the current tip of the chain.
func (c *Chain) LatestBlock() *block.Block {
	return c.latestBlock
}

// Height returns the height of the latest block.
func (c *Chain) Height() uint64 {
	if c.latestBlock == nil {
		return 0
	}
	return c.latestBlock.Height()
}
