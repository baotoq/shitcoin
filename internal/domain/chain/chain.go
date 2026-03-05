package chain

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/baotoq/shitcoin/internal/domain/block"
	"github.com/baotoq/shitcoin/internal/domain/tx"
	"github.com/baotoq/shitcoin/internal/domain/utxo"
)

// ChainConfig holds consensus parameters for the chain aggregate.
type ChainConfig struct {
	BlockTimeTarget          int    // target seconds between blocks
	DifficultyAdjustInterval int    // blocks between difficulty adjustments
	InitialDifficulty        int    // initial bits value
	GenesisMessage           string // message embedded in genesis block
	BlockReward              int64  // block reward in satoshis
}

// Chain is the aggregate root that manages the block sequence, mining, and difficulty.
type Chain struct {
	repo        Repository
	pow         *block.ProofOfWork
	latestBlock *block.Block
	config      ChainConfig
	utxoSet     *utxo.Set
}

// NewChain creates a new Chain aggregate with the given dependencies.
func NewChain(repo Repository, pow *block.ProofOfWork, config ChainConfig, utxoSet *utxo.Set) *Chain {
	return &Chain{
		repo:    repo,
		pow:     pow,
		config:  config,
		utxoSet: utxoSet,
	}
}

// Initialize loads an existing chain or creates the genesis block.
// If the chain is empty (ErrChainEmpty), it creates, mines, and persists the genesis block.
// If blocks exist, it loads the latest block as the chain tip.
func (c *Chain) Initialize(ctx context.Context, minerAddress string) error {
	latest, err := c.repo.GetLatestBlock(ctx)
	if err != nil {
		if !errors.Is(err, ErrChainEmpty) {
			return fmt.Errorf("get latest block: %w", err)
		}

		// Chain is empty -- create genesis block with coinbase
		var txs []*tx.Transaction
		var blockTxs []any
		if minerAddress != "" && c.config.BlockReward > 0 {
			coinbase := tx.NewCoinbaseTx(minerAddress, c.config.BlockReward)
			txs = []*tx.Transaction{coinbase}
			blockTxs = make([]any, len(txs))
			for i, t := range txs {
				blockTxs[i] = t
			}
		}

		// Compute Merkle root from transaction hashes
		var merkleRoot block.Hash
		if len(txs) > 0 {
			txHashes := make([]block.Hash, len(txs))
			for i, t := range txs {
				txHashes[i] = t.ID()
			}
			merkleRoot = block.ComputeMerkleRoot(txHashes)
		}

		genesis, err := block.NewGenesisBlock(c.config.GenesisMessage, uint32(c.config.InitialDifficulty), blockTxs, merkleRoot)
		if err != nil {
			return fmt.Errorf("create genesis block: %w", err)
		}

		if err := c.pow.Mine(genesis); err != nil {
			return fmt.Errorf("mine genesis block: %w", err)
		}

		// Apply UTXO changes if we have a UTXO set and transactions
		if c.utxoSet != nil && len(txs) > 0 {
			undoEntry, err := c.utxoSet.ApplyBlock(0, txs)
			if err != nil {
				return fmt.Errorf("apply genesis utxo: %w", err)
			}

			if err := c.repo.SaveBlockWithUTXOs(ctx, genesis, undoEntry); err != nil {
				return fmt.Errorf("save genesis block with utxos: %w", err)
			}
		} else {
			if err := c.repo.SaveBlock(ctx, genesis); err != nil {
				return fmt.Errorf("save genesis block: %w", err)
			}
		}

		c.latestBlock = genesis
		return nil
	}

	// Chain exists -- load latest block as tip
	c.latestBlock = latest
	return nil
}

// MineBlock creates a new block with transactions, mines it with PoW, and persists it.
// Creates a coinbase transaction crediting the miner. Adjusts difficulty every
// DifficultyAdjustInterval blocks. Atomically updates block storage and UTXO set.
func (c *Chain) MineBlock(ctx context.Context, minerAddress string, txs []*tx.Transaction) (*block.Block, error) {
	if c.latestBlock == nil {
		return nil, fmt.Errorf("chain not initialized: call Initialize first")
	}

	newHeight := c.latestBlock.Height() + 1
	bits, err := c.getCurrentBits(ctx, newHeight)
	if err != nil {
		return nil, fmt.Errorf("get current bits: %w", err)
	}

	// Create coinbase transaction and prepend to transaction list
	coinbase := tx.NewCoinbaseTx(minerAddress, c.config.BlockReward)
	allTxs := make([]*tx.Transaction, 0, 1+len(txs))
	allTxs = append(allTxs, coinbase)
	allTxs = append(allTxs, txs...)

	// Convert to []any for block construction (avoids block->tx import cycle)
	blockTxs := make([]any, len(allTxs))
	for i, t := range allTxs {
		blockTxs[i] = t
	}

	// Compute Merkle root from all transaction hashes (coinbase + user txs)
	txHashes := make([]block.Hash, len(allTxs))
	for i, t := range allTxs {
		txHashes[i] = t.ID()
	}
	merkleRoot := block.ComputeMerkleRoot(txHashes)

	newBlock, err := block.NewBlock(c.latestBlock.Hash(), newHeight, bits, blockTxs, merkleRoot)
	if err != nil {
		return nil, fmt.Errorf("create block: %w", err)
	}

	if err := c.pow.Mine(newBlock); err != nil {
		return nil, fmt.Errorf("mine block: %w", err)
	}

	// Apply UTXO changes and save atomically
	if c.utxoSet != nil {
		undoEntry, err := c.utxoSet.ApplyBlock(newHeight, allTxs)
		if err != nil {
			return nil, fmt.Errorf("apply utxo: %w", err)
		}

		if err := c.repo.SaveBlockWithUTXOs(ctx, newBlock, undoEntry); err != nil {
			return nil, fmt.Errorf("save block with utxos: %w", err)
		}
	} else {
		if err := c.repo.SaveBlock(ctx, newBlock); err != nil {
			return nil, fmt.Errorf("save block: %w", err)
		}
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
