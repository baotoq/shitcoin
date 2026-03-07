package chain

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/baotoq/shitcoin/internal/domain/block"
	"github.com/baotoq/shitcoin/internal/domain/tx"
	"github.com/baotoq/shitcoin/internal/domain/utxo"
)

// MempoolAdder is a minimal interface for adding transactions back to the mempool.
// Avoids hard coupling to the mempool package.
type MempoolAdder interface {
	Add(transaction *tx.Transaction) error
}

// ChainConfig holds consensus parameters for the chain aggregate.
type ChainConfig struct {
	BlockTimeTarget          int    // target seconds between blocks
	DifficultyAdjustInterval int    // blocks between difficulty adjustments
	InitialDifficulty        int    // initial bits value
	GenesisMessage           string // message embedded in genesis block
	BlockReward              int64  // block reward in satoshis
	HalvingInterval          int    // blocks between reward halvings (0 = no halving)
	MaxBlockTxs              int    // max non-coinbase transactions per block (0 = unlimited)
}

// Chain is the aggregate root that manages the block sequence, mining, and difficulty.
type Chain struct {
	mu          sync.RWMutex
	repo        Repository
	pow         *block.ProofOfWork
	latestBlock *block.Block
	config      ChainConfig
	utxoSet     *utxo.Set

	// OnMiningProgress is an optional callback invoked during mining with sampled
	// progress reports. Set by the handler layer to publish events without coupling
	// the domain to the event bus. Nil-safe: if not set, MineBlock uses pow.Mine.
	OnMiningProgress func(block.MiningProgress)
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
		genesisReward := c.rewardAtHeight(0)
		if minerAddress != "" && genesisReward > 0 {
			coinbase := tx.NewCoinbaseTxWithHeight(minerAddress, genesisReward, 0)
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

// rewardAtHeight computes the block reward at the given height, halving every
// HalvingInterval blocks. After 64 halvings the reward is zero. If HalvingInterval
// is <= 0, halving is disabled and the full BlockReward is always returned.
func (c *Chain) rewardAtHeight(height uint64) int64 {
	if c.config.HalvingInterval <= 0 {
		return c.config.BlockReward
	}
	halvings := height / uint64(c.config.HalvingInterval)
	if halvings >= 64 {
		return 0
	}
	return c.config.BlockReward >> halvings
}

// MineBlock creates a new block with transactions, mines it with PoW, and persists it.
// Creates a coinbase transaction crediting the miner. Adjusts difficulty every
// DifficultyAdjustInterval blocks. Atomically updates block storage and UTXO set.
// totalFees is the sum of transaction fees to include in the coinbase reward.
func (c *Chain) MineBlock(ctx context.Context, minerAddress string, txs []*tx.Transaction, totalFees int64) (*block.Block, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.latestBlock == nil {
		return nil, fmt.Errorf("chain not initialized: call Initialize first")
	}

	newHeight := c.latestBlock.Height() + 1
	bits, err := c.getCurrentBits(ctx, newHeight)
	if err != nil {
		return nil, fmt.Errorf("get current bits: %w", err)
	}

	// Create coinbase transaction and prepend to transaction list
	coinbaseReward := c.rewardAtHeight(newHeight) + totalFees
	coinbase := tx.NewCoinbaseTxWithHeight(minerAddress, coinbaseReward, newHeight)
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

	if c.OnMiningProgress != nil {
		if err := c.pow.MineWithProgress(newBlock, 5000, c.OnMiningProgress); err != nil {
			return nil, fmt.Errorf("mine block: %w", err)
		}
	} else {
		if err := c.pow.Mine(newBlock); err != nil {
			return nil, fmt.Errorf("mine block: %w", err)
		}
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
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.latestBlock
}

// SetLatestBlock sets the chain tip to the given block.
// Used by P2P when accepting a valid block from a peer.
func (c *Chain) SetLatestBlock(b *block.Block) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.latestBlock = b
}

// Height returns the height of the latest block.
func (c *Chain) Height() uint64 {
	c.mu.RLock()
	defer c.mu.RUnlock()
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

// Reorganize switches the chain from the current fork to a longer fork.
// It undoes blocks from the current tip down to forkHeight, then applies newBlocks.
// Orphaned non-coinbase transactions are offered back to the mempool via mempoolAdder.
func (c *Chain) Reorganize(ctx context.Context, forkHeight uint64, newBlocks []*block.Block, mempoolAdder MempoolAdder) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.latestBlock == nil {
		return fmt.Errorf("chain not initialized")
	}

	currentHeight := c.latestBlock.Height()
	slog.Info("starting chain reorganization",
		"fork_height", forkHeight,
		"current_height", currentHeight,
		"new_blocks", len(newBlocks),
	)

	// 1. Get orphaned blocks (from forkHeight+1 to current tip)
	orphanedBlocks, err := c.repo.GetBlocksInRange(ctx, forkHeight+1, currentHeight)
	if err != nil {
		return fmt.Errorf("get orphaned blocks: %w", err)
	}

	// 2. Collect orphaned non-coinbase transactions
	orphanedTxs := make(map[string]*tx.Transaction)
	for _, ob := range orphanedBlocks {
		for _, rawTx := range ob.RawTransactions() {
			if t, ok := rawTx.(*tx.Transaction); ok {
				if !t.IsCoinbase() {
					orphanedTxs[t.ID().String()] = t
				}
			}
		}
	}

	// 3. Undo orphaned blocks in reverse order (from tip down to forkHeight+1)
	if c.utxoSet != nil {
		for h := currentHeight; h >= forkHeight+1; h-- {
			undoEntry, err := c.repo.GetUndoEntry(ctx, h)
			if err != nil {
				return fmt.Errorf("get undo entry at height %d: %w", h, err)
			}
			if err := c.utxoSet.UndoBlock(undoEntry); err != nil {
				return fmt.Errorf("undo block at height %d: %w", h, err)
			}
			slog.Debug("undid block", "height", h)
		}
	}

	// 4. Delete orphaned blocks from storage
	if err := c.repo.DeleteBlocksAbove(ctx, forkHeight); err != nil {
		return fmt.Errorf("delete blocks above height %d: %w", forkHeight, err)
	}

	// 5. Apply new blocks in forward order
	for _, newBlk := range newBlocks {
		// Validate PoW
		pow := &block.ProofOfWork{}
		if !pow.Validate(newBlk) {
			return fmt.Errorf("invalid PoW for new block at height %d", newBlk.Height())
		}

		// Extract transactions
		txs := make([]*tx.Transaction, 0, len(newBlk.RawTransactions()))
		for _, rawTx := range newBlk.RawTransactions() {
			if t, ok := rawTx.(*tx.Transaction); ok {
				txs = append(txs, t)
			}
		}

		// Apply UTXO changes and save
		if c.utxoSet != nil && len(txs) > 0 {
			undoEntry, err := c.utxoSet.ApplyBlock(newBlk.Height(), txs)
			if err != nil {
				return fmt.Errorf("apply UTXO for new block at height %d: %w", newBlk.Height(), err)
			}
			if err := c.repo.SaveBlockWithUTXOs(ctx, newBlk, undoEntry); err != nil {
				return fmt.Errorf("save new block at height %d: %w", newBlk.Height(), err)
			}
		} else {
			if err := c.repo.SaveBlock(ctx, newBlk); err != nil {
				return fmt.Errorf("save new block at height %d: %w", newBlk.Height(), err)
			}
		}

		slog.Debug("applied new block", "height", newBlk.Height(), "hash", newBlk.Hash().String()[:16])
	}

	// 6. Update chain tip to last new block
	if len(newBlocks) > 0 {
		c.latestBlock = newBlocks[len(newBlocks)-1]
	}

	slog.Info("chain reorganization complete",
		"new_height", c.latestBlock.Height(),
		"new_tip", c.latestBlock.Hash().String()[:16],
	)

	// 7. Re-add orphaned transactions to mempool (exclude those in the new chain)
	if mempoolAdder != nil {
		// Collect all tx IDs from new chain blocks
		newChainTxIDs := make(map[string]bool)
		for _, nb := range newBlocks {
			for _, rawTx := range nb.RawTransactions() {
				if t, ok := rawTx.(*tx.Transaction); ok {
					newChainTxIDs[t.ID().String()] = true
				}
			}
		}

		for txID, orphanTx := range orphanedTxs {
			if newChainTxIDs[txID] {
				continue // tx is in the new chain, skip
			}
			// Ignore errors -- some orphaned txs may now be invalid
			_ = mempoolAdder.Add(orphanTx)
		}
	}

	return nil
}
