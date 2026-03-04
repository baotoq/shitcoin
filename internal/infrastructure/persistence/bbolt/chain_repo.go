package bbolt

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"fmt"

	"github.com/baotoq/shitcoin/internal/domain/block"
	"github.com/baotoq/shitcoin/internal/domain/chain"
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

// SaveBlock persists a block to bbolt in a single transaction.
// Stores: block JSON by hash key, height index entry, and updates chain_meta.
func (r *BboltRepository) SaveBlock(_ context.Context, b *block.Block) error {
	model := BlockModelFromDomain(b)
	data, err := json.Marshal(model)
	if err != nil {
		return fmt.Errorf("marshal block: %w", err)
	}

	hashKey := []byte(b.Hash().String())

	return r.db.Update(func(tx *bolt.Tx) error {
		blocks := tx.Bucket(blocksBucket)
		meta := tx.Bucket(chainMetaBucket)

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
