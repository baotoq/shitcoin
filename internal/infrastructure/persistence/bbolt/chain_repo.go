package bbolt

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"fmt"

	"github.com/baotoq/shitcoin/internal/domain/block"
	"github.com/baotoq/shitcoin/internal/domain/chain"
	"github.com/baotoq/shitcoin/internal/domain/tx"
	"github.com/baotoq/shitcoin/internal/domain/utxo"
	bolt "go.etcd.io/bbolt"
)

var (
	blocksBucket    = []byte("blocks")
	chainMetaBucket = []byte("chain_meta")
	latestHashKey   = []byte("latest_hash")
	heightKey       = []byte("height")
)

// BboltRepository implements chain.Repository using bbolt as the storage engine.
type BboltRepository struct {
	db *bolt.DB
}

// NewBboltRepository creates a new BboltRepository and ensures required buckets exist.
func NewBboltRepository(db *bolt.DB) (*BboltRepository, error) {
	err := db.Update(func(tx *bolt.Tx) error {
		if _, err := tx.CreateBucketIfNotExists(blocksBucket); err != nil {
			return fmt.Errorf("create blocks bucket: %w", err)
		}
		if _, err := tx.CreateBucketIfNotExists(chainMetaBucket); err != nil {
			return fmt.Errorf("create chain_meta bucket: %w", err)
		}
		if _, err := tx.CreateBucketIfNotExists(utxoBucket); err != nil {
			return fmt.Errorf("create utxo bucket: %w", err)
		}
		if _, err := tx.CreateBucketIfNotExists(undoBucket); err != nil {
			return fmt.Errorf("create undo bucket: %w", err)
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("init bbolt repository: %w", err)
	}
	return &BboltRepository{db: db}, nil
}

// heightKey8 converts a uint64 height to an 8-byte big-endian key for ordered iteration.
func heightKey8(height uint64) []byte {
	key := make([]byte, 8)
	binary.BigEndian.PutUint64(key, height)
	return key
}

// heightIndexKey returns a prefixed key for the height index: "h:" + 8-byte big-endian height.
func heightIndexKey(height uint64) []byte {
	prefix := []byte("h:")
	hk := heightKey8(height)
	return append(prefix, hk...)
}

// saveBlockInTx saves a block within an existing bbolt transaction.
// This is the shared logic used by both SaveBlock and SaveBlockWithUTXOs.
func (r *BboltRepository) saveBlockInTx(boltTx *bolt.Tx, b *block.Block) error {
	model := BlockModelFromDomain(b)
	data, err := json.Marshal(model)
	if err != nil {
		return fmt.Errorf("marshal block: %w", err)
	}

	hashKey := []byte(b.Hash().String())
	blocks := boltTx.Bucket(blocksBucket)
	meta := boltTx.Bucket(chainMetaBucket)

	// Store block JSON by hash
	if err := blocks.Put(hashKey, data); err != nil {
		return fmt.Errorf("put block: %w", err)
	}

	// Store height index: h:<8-byte height> -> hash string
	if err := blocks.Put(heightIndexKey(b.Height()), hashKey); err != nil {
		return fmt.Errorf("put height index: %w", err)
	}

	// Update chain metadata
	if err := meta.Put(latestHashKey, hashKey); err != nil {
		return fmt.Errorf("put latest hash: %w", err)
	}
	if err := meta.Put(heightKey, heightKey8(b.Height())); err != nil {
		return fmt.Errorf("put height: %w", err)
	}

	return nil
}

// SaveBlock persists a block to bbolt in a single transaction.
// Stores: block JSON by hash key, height index entry, and updates chain_meta.
func (r *BboltRepository) SaveBlock(_ context.Context, b *block.Block) error {
	return r.db.Update(func(boltTx *bolt.Tx) error {
		return r.saveBlockInTx(boltTx, b)
	})
}

// SaveBlockWithUTXOs persists a block along with its UTXO changes in a single
// atomic bbolt transaction. This ensures crash-safe consistency between block
// storage, UTXO state, and the undo log.
func (r *BboltRepository) SaveBlockWithUTXOs(_ context.Context, b *block.Block, undoEntry *utxo.UndoEntry) error {
	return r.db.Update(func(boltTx *bolt.Tx) error {
		// 1. Save the block
		if err := r.saveBlockInTx(boltTx, b); err != nil {
			return err
		}

		utxoBkt := boltTx.Bucket(utxoBucket)
		undoBkt := boltTx.Bucket(undoBucket)

		// 2. Process UTXO changes from undo entry
		// Remove spent UTXOs
		for _, spent := range undoEntry.Spent {
			txID, err := block.HashFromHex(spent.TxID)
			if err != nil {
				return fmt.Errorf("parse spent txid: %w", err)
			}
			key := utxoKey(txID, spent.Vout)
			if err := utxoBkt.Delete(key); err != nil {
				return fmt.Errorf("delete spent utxo: %w", err)
			}
		}

		// Add created UTXOs -- we need to reconstruct them from the block's transactions
		for _, rawTx := range b.RawTransactions() {
			transaction, ok := rawTx.(*tx.Transaction)
			if !ok {
				continue
			}
			txID := transaction.ID()
			for i, output := range transaction.Outputs() {
				u := utxo.NewUTXO(txID, uint32(i), output.Value(), output.Address())
				model := UTXOModelFromDomain(u)
				data, err := json.Marshal(model)
				if err != nil {
					return fmt.Errorf("marshal utxo: %w", err)
				}
				key := utxoKey(txID, uint32(i))
				if err := utxoBkt.Put(key, data); err != nil {
					return fmt.Errorf("put utxo: %w", err)
				}
			}
		}

		// 3. Save undo entry
		undoData, err := json.Marshal(undoEntry)
		if err != nil {
			return fmt.Errorf("marshal undo entry: %w", err)
		}
		if err := undoBkt.Put(undoKey(undoEntry.BlockHeight), undoData); err != nil {
			return fmt.Errorf("put undo entry: %w", err)
		}

		return nil
	})
}

// GetBlock retrieves a block by its hash.
func (r *BboltRepository) GetBlock(_ context.Context, hash block.Hash) (*block.Block, error) {
	hashKey := []byte(hash.String())
	var model BlockModel

	err := r.db.View(func(tx *bolt.Tx) error {
		blocks := tx.Bucket(blocksBucket)
		data := blocks.Get(hashKey)
		if data == nil {
			return chain.ErrBlockNotFound
		}
		// CRITICAL: Copy byte slice before tx closes (bbolt pitfall #4)
		dataCopy := make([]byte, len(data))
		copy(dataCopy, data)

		return json.Unmarshal(dataCopy, &model)
	})
	if err != nil {
		return nil, err
	}

	return model.ToDomain()
}

// GetBlockByHeight retrieves a block at a specific height.
func (r *BboltRepository) GetBlockByHeight(_ context.Context, height uint64) (*block.Block, error) {
	var model BlockModel

	err := r.db.View(func(tx *bolt.Tx) error {
		blocks := tx.Bucket(blocksBucket)

		// Look up hash from height index
		hashKey := blocks.Get(heightIndexKey(height))
		if hashKey == nil {
			return chain.ErrBlockNotFound
		}

		// Get block data by hash
		data := blocks.Get(hashKey)
		if data == nil {
			return chain.ErrBlockNotFound
		}

		// Copy before tx closes
		dataCopy := make([]byte, len(data))
		copy(dataCopy, data)

		return json.Unmarshal(dataCopy, &model)
	})
	if err != nil {
		return nil, err
	}

	return model.ToDomain()
}

// GetLatestBlock returns the most recently saved block.
func (r *BboltRepository) GetLatestBlock(_ context.Context) (*block.Block, error) {
	var model BlockModel

	err := r.db.View(func(tx *bolt.Tx) error {
		meta := tx.Bucket(chainMetaBucket)
		blocks := tx.Bucket(blocksBucket)

		hashKey := meta.Get(latestHashKey)
		if hashKey == nil {
			return chain.ErrChainEmpty
		}

		data := blocks.Get(hashKey)
		if data == nil {
			return chain.ErrBlockNotFound
		}

		// Copy before tx closes
		dataCopy := make([]byte, len(data))
		copy(dataCopy, data)

		return json.Unmarshal(dataCopy, &model)
	})
	if err != nil {
		return nil, err
	}

	return model.ToDomain()
}

// GetChainHeight returns the current chain height (0 if empty).
func (r *BboltRepository) GetChainHeight(_ context.Context) (uint64, error) {
	var height uint64

	err := r.db.View(func(tx *bolt.Tx) error {
		meta := tx.Bucket(chainMetaBucket)
		data := meta.Get(heightKey)
		if data == nil {
			height = 0
			return nil
		}
		// Copy before reading
		dataCopy := make([]byte, len(data))
		copy(dataCopy, data)
		height = binary.BigEndian.Uint64(dataCopy)
		return nil
	})

	return height, err
}

// GetUndoEntry retrieves the UTXO undo entry for a block at the given height.
func (r *BboltRepository) GetUndoEntry(_ context.Context, blockHeight uint64) (*utxo.UndoEntry, error) {
	var entry utxo.UndoEntry

	err := r.db.View(func(boltTx *bolt.Tx) error {
		undoBkt := boltTx.Bucket(undoBucket)
		key := make([]byte, 8)
		binary.BigEndian.PutUint64(key, blockHeight)
		data := undoBkt.Get(key)
		if data == nil {
			return utxo.ErrUndoEntryNotFound
		}
		dataCopy := make([]byte, len(data))
		copy(dataCopy, data)
		return json.Unmarshal(dataCopy, &entry)
	})
	if err != nil {
		return nil, err
	}

	return &entry, nil
}

// DeleteBlocksAbove removes all blocks above the given height from storage.
// Deletes block data, height index entries, undo entries, and updates chain metadata.
// Uses a single bbolt Update transaction for atomicity.
func (r *BboltRepository) DeleteBlocksAbove(_ context.Context, height uint64) error {
	return r.db.Update(func(boltTx *bolt.Tx) error {
		blocks := boltTx.Bucket(blocksBucket)
		meta := boltTx.Bucket(chainMetaBucket)
		undoBkt := boltTx.Bucket(undoBucket)

		// Get current chain height
		heightData := meta.Get(heightKey)
		if heightData == nil {
			return nil // empty chain, nothing to delete
		}
		dataCopy := make([]byte, len(heightData))
		copy(dataCopy, heightData)
		currentHeight := binary.BigEndian.Uint64(dataCopy)

		// Delete blocks from height+1 to currentHeight
		for h := height + 1; h <= currentHeight; h++ {
			hk := heightIndexKey(h)

			// Get hash key from height index
			hashKey := blocks.Get(hk)
			if hashKey != nil {
				hashKeyCopy := make([]byte, len(hashKey))
				copy(hashKeyCopy, hashKey)

				// Delete block data
				if err := blocks.Delete(hashKeyCopy); err != nil {
					return fmt.Errorf("delete block at height %d: %w", h, err)
				}
			}

			// Delete height index entry
			if err := blocks.Delete(hk); err != nil {
				return fmt.Errorf("delete height index at %d: %w", h, err)
			}

			// Delete undo entry
			undoKeyBytes := make([]byte, 8)
			binary.BigEndian.PutUint64(undoKeyBytes, h)
			if err := undoBkt.Delete(undoKeyBytes); err != nil {
				return fmt.Errorf("delete undo entry at height %d: %w", h, err)
			}
		}

		// Update chain metadata to reflect new tip
		if err := meta.Put(heightKey, heightKey8(height)); err != nil {
			return fmt.Errorf("update height metadata: %w", err)
		}

		// Update latest hash to the block at the given height
		hashKey := blocks.Get(heightIndexKey(height))
		if hashKey != nil {
			hashKeyCopy := make([]byte, len(hashKey))
			copy(hashKeyCopy, hashKey)
			if err := meta.Put(latestHashKey, hashKeyCopy); err != nil {
				return fmt.Errorf("update latest hash metadata: %w", err)
			}
		}

		return nil
	})
}

// GetBlocksInRange returns blocks from startHeight to endHeight inclusive.
func (r *BboltRepository) GetBlocksInRange(_ context.Context, startHeight, endHeight uint64) ([]*block.Block, error) {
	var models []BlockModel

	err := r.db.View(func(tx *bolt.Tx) error {
		blocks := tx.Bucket(blocksBucket)

		for h := startHeight; h <= endHeight; h++ {
			hashKey := blocks.Get(heightIndexKey(h))
			if hashKey == nil {
				return chain.ErrBlockNotFound
			}

			data := blocks.Get(hashKey)
			if data == nil {
				return chain.ErrBlockNotFound
			}

			// Copy before tx closes
			dataCopy := make([]byte, len(data))
			copy(dataCopy, data)

			var model BlockModel
			if err := json.Unmarshal(dataCopy, &model); err != nil {
				return fmt.Errorf("unmarshal block at height %d: %w", h, err)
			}
			models = append(models, model)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	result := make([]*block.Block, 0, len(models))
	for i := range models {
		b, err := models[i].ToDomain()
		if err != nil {
			return nil, fmt.Errorf("convert block model at index %d: %w", i, err)
		}
		result = append(result, b)
	}
	return result, nil
}
