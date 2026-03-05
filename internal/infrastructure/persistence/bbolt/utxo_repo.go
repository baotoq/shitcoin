package bbolt

import (
	"encoding/binary"
	"encoding/json"
	"fmt"

	"github.com/baotoq/shitcoin/internal/domain/block"
	"github.com/baotoq/shitcoin/internal/domain/utxo"
	bolt "go.etcd.io/bbolt"
)

var (
	utxoBucket = []byte("utxo")
	undoBucket = []byte("undo")
)

// UTXORepo implements utxo.Repository using bbolt as the storage engine.
type UTXORepo struct {
	db *bolt.DB
}

// Compile-time check that UTXORepo implements utxo.Repository.
var _ utxo.Repository = (*UTXORepo)(nil)

// NewUTXORepo creates a new UTXORepo and ensures required buckets exist.
func NewUTXORepo(db *bolt.DB) (*UTXORepo, error) {
	err := db.Update(func(tx *bolt.Tx) error {
		if _, err := tx.CreateBucketIfNotExists(utxoBucket); err != nil {
			return fmt.Errorf("create utxo bucket: %w", err)
		}
		if _, err := tx.CreateBucketIfNotExists(undoBucket); err != nil {
			return fmt.Errorf("create undo bucket: %w", err)
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("init utxo repo: %w", err)
	}
	return &UTXORepo{db: db}, nil
}

// utxoKey builds a 36-byte composite key: 32-byte txID + 4-byte big-endian vout.
func utxoKey(txID block.Hash, vout uint32) []byte {
	key := make([]byte, 36)
	copy(key[:32], txID.Bytes())
	binary.BigEndian.PutUint32(key[32:], vout)
	return key
}

// undoKey builds an 8-byte big-endian key from block height.
func undoKey(blockHeight uint64) []byte {
	key := make([]byte, 8)
	binary.BigEndian.PutUint64(key, blockHeight)
	return key
}

// Put stores a UTXO in the utxo bucket.
func (r *UTXORepo) Put(u utxo.UTXO) error {
	model := UTXOModelFromDomain(u)
	data, err := json.Marshal(model)
	if err != nil {
		return fmt.Errorf("marshal utxo: %w", err)
	}

	return r.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(utxoBucket)
		return bucket.Put(utxoKey(u.TxID(), u.Vout()), data)
	})
}

// Get retrieves a UTXO by transaction ID and output index.
func (r *UTXORepo) Get(txID block.Hash, vout uint32) (utxo.UTXO, error) {
	var model UTXOModel

	err := r.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(utxoBucket)
		data := bucket.Get(utxoKey(txID, vout))
		if data == nil {
			return utxo.ErrUTXONotFound
		}
		// Copy byte slice before tx closes (bbolt pitfall #4)
		dataCopy := make([]byte, len(data))
		copy(dataCopy, data)
		return json.Unmarshal(dataCopy, &model)
	})
	if err != nil {
		return utxo.UTXO{}, err
	}

	return model.ToDomain()
}

// Delete removes a UTXO from the utxo bucket.
func (r *UTXORepo) Delete(txID block.Hash, vout uint32) error {
	return r.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(utxoBucket)
		key := utxoKey(txID, vout)

		// Check existence first
		if bucket.Get(key) == nil {
			return utxo.ErrUTXONotFound
		}
		return bucket.Delete(key)
	})
}

// GetByAddress returns all UTXOs belonging to the given address.
// Iterates all entries in the utxo bucket and filters by address.
// Acceptable for educational project; production would use a secondary index.
func (r *UTXORepo) GetByAddress(address string) ([]utxo.UTXO, error) {
	var result []utxo.UTXO

	err := r.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(utxoBucket)
		return bucket.ForEach(func(k, v []byte) error {
			// Copy data before processing (bbolt pitfall #4)
			dataCopy := make([]byte, len(v))
			copy(dataCopy, v)

			var model UTXOModel
			if err := json.Unmarshal(dataCopy, &model); err != nil {
				return fmt.Errorf("unmarshal utxo: %w", err)
			}

			if model.Address == address {
				u, err := model.ToDomain()
				if err != nil {
					return fmt.Errorf("convert utxo model: %w", err)
				}
				result = append(result, u)
			}
			return nil
		})
	})

	return result, err
}

// SaveUndoEntry persists an undo entry keyed by block height.
func (r *UTXORepo) SaveUndoEntry(entry *utxo.UndoEntry) error {
	data, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("marshal undo entry: %w", err)
	}

	return r.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(undoBucket)
		return bucket.Put(undoKey(entry.BlockHeight), data)
	})
}

// GetUndoEntry retrieves the undo entry for a block height.
func (r *UTXORepo) GetUndoEntry(blockHeight uint64) (*utxo.UndoEntry, error) {
	var entry utxo.UndoEntry

	err := r.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(undoBucket)
		data := bucket.Get(undoKey(blockHeight))
		if data == nil {
			return utxo.ErrUndoEntryNotFound
		}
		// Copy before tx closes
		dataCopy := make([]byte, len(data))
		copy(dataCopy, data)
		return json.Unmarshal(dataCopy, &entry)
	})
	if err != nil {
		return nil, err
	}

	return &entry, nil
}

// DeleteUndoEntry removes the undo entry for a block height.
func (r *UTXORepo) DeleteUndoEntry(blockHeight uint64) error {
	return r.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(undoBucket)
		return bucket.Delete(undoKey(blockHeight))
	})
}
