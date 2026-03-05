package bbolt

import (
	"encoding/json"
	"path/filepath"
	"testing"

	"github.com/baotoq/shitcoin/internal/domain/block"
	"github.com/baotoq/shitcoin/internal/domain/utxo"
	"github.com/stretchr/testify/suite"
	bolt "go.etcd.io/bbolt"
)

// UTXORepoSuite tests BboltUTXORepo with a fresh DB per test.
type UTXORepoSuite struct {
	suite.Suite
	db   *bolt.DB
	repo *UTXORepo
}

func (s *UTXORepoSuite) SetupTest() {
	dbPath := filepath.Join(s.T().TempDir(), "test.db")
	db, err := bolt.Open(dbPath, 0600, nil)
	s.Require().NoError(err)
	s.T().Cleanup(func() { db.Close() })
	s.db = db

	repo, err := NewUTXORepo(db)
	s.Require().NoError(err)
	s.repo = repo
}

func TestUTXORepoSuite(t *testing.T) {
	suite.Run(t, new(UTXORepoSuite))
}

func (s *UTXORepoSuite) TestPutAndGet() {
	txID := block.DoubleSHA256([]byte("tx1"))
	u := utxo.NewUTXO(txID, 0, 5_000_000_000, "addr1")

	s.Require().NoError(s.repo.Put(u))

	got, err := s.repo.Get(txID, 0)
	s.Require().NoError(err)

	s.Assert().Equal(txID, got.TxID())
	s.Assert().Equal(uint32(0), got.Vout())
	s.Assert().Equal(int64(5_000_000_000), got.Value())
	s.Assert().Equal("addr1", got.Address())
}

func (s *UTXORepoSuite) TestGetNotFound() {
	_, err := s.repo.Get(block.DoubleSHA256([]byte("nonexistent")), 0)
	s.Require().ErrorIs(err, utxo.ErrUTXONotFound)
}

func (s *UTXORepoSuite) TestDelete() {
	txID := block.DoubleSHA256([]byte("tx1"))
	u := utxo.NewUTXO(txID, 0, 1000, "addr1")
	s.Require().NoError(s.repo.Put(u))

	s.Require().NoError(s.repo.Delete(txID, 0))

	_, err := s.repo.Get(txID, 0)
	s.Require().ErrorIs(err, utxo.ErrUTXONotFound)
}

func (s *UTXORepoSuite) TestDeleteNotFound() {
	err := s.repo.Delete(block.DoubleSHA256([]byte("nonexistent")), 0)
	s.Require().ErrorIs(err, utxo.ErrUTXONotFound)
}

func (s *UTXORepoSuite) TestGetByAddress() {
	txID1 := block.DoubleSHA256([]byte("tx1"))
	txID2 := block.DoubleSHA256([]byte("tx2"))
	txID3 := block.DoubleSHA256([]byte("tx3"))

	s.Require().NoError(s.repo.Put(utxo.NewUTXO(txID1, 0, 1000, "alice")))
	s.Require().NoError(s.repo.Put(utxo.NewUTXO(txID2, 0, 2000, "alice")))
	s.Require().NoError(s.repo.Put(utxo.NewUTXO(txID3, 0, 3000, "bob")))

	aliceUTXOs, err := s.repo.GetByAddress("alice")
	s.Require().NoError(err)
	s.Require().Len(aliceUTXOs, 2)

	var total int64
	for _, u := range aliceUTXOs {
		total += u.Value()
	}
	s.Assert().Equal(int64(3000), total)

	bobUTXOs, err := s.repo.GetByAddress("bob")
	s.Require().NoError(err)
	s.Require().Len(bobUTXOs, 1)

	unknownUTXOs, err := s.repo.GetByAddress("nobody")
	s.Require().NoError(err)
	s.Assert().Empty(unknownUTXOs)
}

func (s *UTXORepoSuite) TestUndoEntry() {
	entry := &utxo.UndoEntry{
		BlockHeight: 5,
		Spent: []utxo.SpentUTXO{
			{TxID: "aabb", Vout: 0, Value: 1000, Address: "alice"},
		},
		Created: []utxo.UTXORef{
			{TxID: "ccdd", Vout: 0},
			{TxID: "ccdd", Vout: 1},
		},
	}

	s.Require().NoError(s.repo.SaveUndoEntry(entry))

	got, err := s.repo.GetUndoEntry(5)
	s.Require().NoError(err)

	s.Assert().Equal(uint64(5), got.BlockHeight)
	s.Require().Len(got.Spent, 1)
	s.Assert().Equal("aabb", got.Spent[0].TxID)
	s.Require().Len(got.Created, 2)
}

func (s *UTXORepoSuite) TestUndoEntryNotFound() {
	_, err := s.repo.GetUndoEntry(999)
	s.Require().ErrorIs(err, utxo.ErrUndoEntryNotFound)
}

func (s *UTXORepoSuite) TestPersistence() {
	dbPath := filepath.Join(s.T().TempDir(), "utxo_persist.db")

	// Phase 1: open, save, close
	db1, err := bolt.Open(dbPath, 0600, nil)
	s.Require().NoError(err)

	repo1, err := NewUTXORepo(db1)
	s.Require().NoError(err)

	txID := block.DoubleSHA256([]byte("persist-tx"))
	s.Require().NoError(repo1.Put(utxo.NewUTXO(txID, 0, 42000, "persist-addr")))

	entry := &utxo.UndoEntry{
		BlockHeight: 10,
		Spent:       []utxo.SpentUTXO{{TxID: "aa", Vout: 0, Value: 100, Address: "x"}},
		Created:     []utxo.UTXORef{{TxID: "bb", Vout: 0}},
	}
	s.Require().NoError(repo1.SaveUndoEntry(entry))

	db1.Close()

	// Phase 2: reopen and verify
	db2, err := bolt.Open(dbPath, 0600, nil)
	s.Require().NoError(err)
	defer db2.Close()

	repo2, err := NewUTXORepo(db2)
	s.Require().NoError(err)

	got, err := repo2.Get(txID, 0)
	s.Require().NoError(err)
	s.Assert().Equal(int64(42000), got.Value())
	s.Assert().Equal("persist-addr", got.Address())

	gotUndo, err := repo2.GetUndoEntry(10)
	s.Require().NoError(err)
	s.Assert().Equal(uint64(10), gotUndo.BlockHeight)
}

func (s *UTXORepoSuite) TestCompositeKey36Bytes() {
	txID := block.DoubleSHA256([]byte("key-test"))
	key := utxoKey(txID, 42)
	s.Assert().Len(key, 36)
}

func (s *UTXORepoSuite) TestStorageModelRoundTrip() {
	txID := block.DoubleSHA256([]byte("model-test"))
	original := utxo.NewUTXO(txID, 3, 999, "test-addr")

	model := UTXOModelFromDomain(original)

	data, err := json.Marshal(model)
	s.Require().NoError(err)

	var decoded UTXOModel
	s.Require().NoError(json.Unmarshal(data, &decoded))

	restored, err := decoded.ToDomain()
	s.Require().NoError(err)

	s.Assert().Equal(original.TxID(), restored.TxID())
	s.Assert().Equal(original.Vout(), restored.Vout())
	s.Assert().Equal(original.Value(), restored.Value())
	s.Assert().Equal(original.Address(), restored.Address())
}
