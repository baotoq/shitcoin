package chain

import (
	"context"
	"errors"
	"fmt"
	"time"

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
	repo        Repository
	pow         *block.ProofOfWork
	latestBlock *block.Block
	config      ChainConfig
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
// If the chain is empty (ErrChainEmpty), it creates, mines, and persists the genesis block.
// If blocks exist, it loads the latest block as the chain tip.
func (c *Chain) Initialize(ctx context.Context) error {
	latest, err := c.repo.GetLatestBlock(ctx)
	if err != nil {
		if !errors.Is(err, ErrChainEmpty) {
			return fmt.Errorf("get latest block: %w", err)
		}

		// Chain is empty -- create genesis block
		genesis, err := block.NewGenesisBlock(c.config.GenesisMessage, uint32(c.config.InitialDifficulty))
		if err != nil {
			return fmt.Errorf("create genesis block: %w", err)
		}

		if err := c.pow.Mine(genesis); err != nil {
			return fmt.Errorf("mine genesis block: %w", err)
		}

		if err := c.repo.SaveBlock(ctx, genesis); err != nil {
			return fmt.Errorf("save genesis block: %w", err)
		}

		c.latestBlock = genesis
		return nil
	}

	// Chain exists -- load latest block as tip
	c.latestBlock = latest
	return nil
}

// MineBlock creates a new block referencing the latest block, mines it with PoW, and persists it.
// Adjusts difficulty every DifficultyAdjustInterval blocks.
func (c *Chain) MineBlock(ctx context.Context) (*block.Block, error) {
	if c.latestBlock == nil {
		return nil, fmt.Errorf("chain not initialized: call Initialize first")
	}

	newHeight := c.latestBlock.Height() + 1
	bits, err := c.getCurrentBits(ctx, newHeight)
	if err != nil {
		return nil, fmt.Errorf("get current bits: %w", err)
	}

	newBlock, err := block.NewBlock(c.latestBlock.Hash(), newHeight, bits)
	if err != nil {
		return nil, fmt.Errorf("create block: %w", err)
	}

	if err := c.pow.Mine(newBlock); err != nil {
		return nil, fmt.Errorf("mine block: %w", err)
	}

	if err := c.repo.SaveBlock(ctx, newBlock); err != nil {
		return nil, fmt.Errorf("save block: %w", err)
	}

	c.latestBlock = newBlock
	return newBlock, nil
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

// getCurrentBits determines the difficulty bits for a new block at the given height.
// If the height is a multiple of DifficultyAdjustInterval, it recalculates difficulty
// based on the actual time taken for the last adjustment window.
func (c *Chain) getCurrentBits(ctx context.Context, newHeight uint64) (uint32, error) {
	interval := uint64(c.config.DifficultyAdjustInterval)
	if interval == 0 {
		return c.latestBlock.Bits(), nil
	}

	// Only adjust at interval boundaries
	if newHeight%interval != 0 {
		return c.latestBlock.Bits(), nil
	}

	// Need to calculate the time span of the last interval window
	// Window is from block at (newHeight - interval) to block at (newHeight - 1)
	windowStart := newHeight - interval
	windowEnd := newHeight - 1

	startBlock, err := c.repo.GetBlockByHeight(ctx, windowStart)
	if err != nil {
		return 0, fmt.Errorf("get window start block (height %d): %w", windowStart, err)
	}

	endBlock, err := c.repo.GetBlockByHeight(ctx, windowEnd)
	if err != nil {
		return 0, fmt.Errorf("get window end block (height %d): %w", windowEnd, err)
	}

	actualTimeSpan := time.Duration(endBlock.Timestamp()-startBlock.Timestamp()) * time.Second
	targetTimeSpan := time.Duration(c.config.BlockTimeTarget) * time.Second * time.Duration(interval)

	return block.AdjustDifficulty(c.latestBlock.Bits(), actualTimeSpan, targetTimeSpan), nil
}
