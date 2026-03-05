package bbolt

import (
	"encoding/json"
	"path/filepath"
	"testing"

	"github.com/baotoq/shitcoin/internal/domain/block"
	"github.com/baotoq/shitcoin/internal/domain/utxo"
	bolt "go.etcd.io/bbolt"
)

func TestUTXORepoPutAndGet(t *testing.T) {
	db := openTestDB(t)
	repo, err := NewUTXORepo(db)
	if err != nil {
		t.Fatalf("NewUTXORepo: %v", err)
	}

	txID := block.DoubleSHA256([]byte("tx1"))
	u := utxo.NewUTXO(txID, 0, 5_000_000_000, "addr1")

	if err := repo.Put(u); err != nil {
		t.Fatalf("Put: %v", err)
	}

	got, err := repo.Get(txID, 0)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}

	if got.TxID() != txID {
		t.Errorf("TxID mismatch")
	}
	if got.Vout() != 0 {
		t.Errorf("Vout = %d; want 0", got.Vout())
	}
	if got.Value() != 5_000_000_000 {
		t.Errorf("Value = %d; want 5000000000", got.Value())
	}
	if got.Address() != "addr1" {
		t.Errorf("Address = %q; want %q", got.Address(), "addr1")
	}
}

func TestUTXORepoGetNotFound(t *testing.T) {
	db := openTestDB(t)
	repo, err := NewUTXORepo(db)
	if err != nil {
		t.Fatalf("NewUTXORepo: %v", err)
	}

	_, err = repo.Get(block.DoubleSHA256([]byte("nonexistent")), 0)
	if err != utxo.ErrUTXONotFound {
		t.Errorf("expected ErrUTXONotFound, got: %v", err)
	}
}

func TestUTXORepoDelete(t *testing.T) {
	db := openTestDB(t)
	repo, err := NewUTXORepo(db)
	if err != nil {
		t.Fatalf("NewUTXORepo: %v", err)
	}

	txID := block.DoubleSHA256([]byte("tx1"))
	u := utxo.NewUTXO(txID, 0, 1000, "addr1")
	repo.Put(u)

	if err := repo.Delete(txID, 0); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	_, err = repo.Get(txID, 0)
	if err != utxo.ErrUTXONotFound {
		t.Errorf("after delete, expected ErrUTXONotFound, got: %v", err)
	}
}

func TestUTXORepoDeleteNotFound(t *testing.T) {
	db := openTestDB(t)
	repo, err := NewUTXORepo(db)
	if err != nil {
		t.Fatalf("NewUTXORepo: %v", err)
	}

	err = repo.Delete(block.DoubleSHA256([]byte("nonexistent")), 0)
	if err != utxo.ErrUTXONotFound {
		t.Errorf("expected ErrUTXONotFound, got: %v", err)
	}
}

func TestUTXORepoGetByAddress(t *testing.T) {
	db := openTestDB(t)
	repo, err := NewUTXORepo(db)
	if err != nil {
		t.Fatalf("NewUTXORepo: %v", err)
	}

	txID1 := block.DoubleSHA256([]byte("tx1"))
	txID2 := block.DoubleSHA256([]byte("tx2"))
	txID3 := block.DoubleSHA256([]byte("tx3"))

	repo.Put(utxo.NewUTXO(txID1, 0, 1000, "alice"))
	repo.Put(utxo.NewUTXO(txID2, 0, 2000, "alice"))
	repo.Put(utxo.NewUTXO(txID3, 0, 3000, "bob"))

	aliceUTXOs, err := repo.GetByAddress("alice")
	if err != nil {
		t.Fatalf("GetByAddress alice: %v", err)
	}
	if len(aliceUTXOs) != 2 {
		t.Fatalf("alice UTXO count = %d; want 2", len(aliceUTXOs))
	}

	var total int64
	for _, u := range aliceUTXOs {
		total += u.Value()
	}
	if total != 3000 {
		t.Errorf("alice total = %d; want 3000", total)
	}

	bobUTXOs, err := repo.GetByAddress("bob")
	if err != nil {
		t.Fatalf("GetByAddress bob: %v", err)
	}
	if len(bobUTXOs) != 1 {
		t.Fatalf("bob UTXO count = %d; want 1", len(bobUTXOs))
	}

	// Unknown address
	unknownUTXOs, err := repo.GetByAddress("nobody")
	if err != nil {
		t.Fatalf("GetByAddress nobody: %v", err)
	}
	if len(unknownUTXOs) != 0 {
		t.Errorf("nobody UTXO count = %d; want 0", len(unknownUTXOs))
	}
}

func TestUTXORepoUndoEntry(t *testing.T) {
	db := openTestDB(t)
	repo, err := NewUTXORepo(db)
	if err != nil {
		t.Fatalf("NewUTXORepo: %v", err)
	}

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

	if err := repo.SaveUndoEntry(entry); err != nil {
		t.Fatalf("SaveUndoEntry: %v", err)
	}

	got, err := repo.GetUndoEntry(5)
	if err != nil {
		t.Fatalf("GetUndoEntry: %v", err)
	}

	if got.BlockHeight != 5 {
		t.Errorf("BlockHeight = %d; want 5", got.BlockHeight)
	}
	if len(got.Spent) != 1 {
		t.Fatalf("Spent length = %d; want 1", len(got.Spent))
	}
	if got.Spent[0].TxID != "aabb" {
		t.Errorf("Spent[0].TxID = %q; want %q", got.Spent[0].TxID, "aabb")
	}
	if len(got.Created) != 2 {
		t.Fatalf("Created length = %d; want 2", len(got.Created))
	}
}

func TestUTXORepoUndoEntryNotFound(t *testing.T) {
	db := openTestDB(t)
	repo, err := NewUTXORepo(db)
	if err != nil {
		t.Fatalf("NewUTXORepo: %v", err)
	}

	_, err = repo.GetUndoEntry(999)
	if err != utxo.ErrUndoEntryNotFound {
		t.Errorf("expected ErrUndoEntryNotFound, got: %v", err)
	}
}

func TestUTXORepoPersistence(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "utxo_persist.db")

	// Phase 1: open, save, close
	db1, err := bolt.Open(dbPath, 0600, nil)
	if err != nil {
		t.Fatalf("open db1: %v", err)
	}

	repo1, err := NewUTXORepo(db1)
	if err != nil {
		t.Fatalf("NewUTXORepo (1): %v", err)
	}

	txID := block.DoubleSHA256([]byte("persist-tx"))
	repo1.Put(utxo.NewUTXO(txID, 0, 42000, "persist-addr"))

	entry := &utxo.UndoEntry{
		BlockHeight: 10,
		Spent:       []utxo.SpentUTXO{{TxID: "aa", Vout: 0, Value: 100, Address: "x"}},
		Created:     []utxo.UTXORef{{TxID: "bb", Vout: 0}},
	}
	repo1.SaveUndoEntry(entry)

	db1.Close()

	// Phase 2: reopen and verify
	db2, err := bolt.Open(dbPath, 0600, nil)
	if err != nil {
		t.Fatalf("open db2: %v", err)
	}
	defer db2.Close()

	repo2, err := NewUTXORepo(db2)
	if err != nil {
		t.Fatalf("NewUTXORepo (2): %v", err)
	}

	got, err := repo2.Get(txID, 0)
	if err != nil {
		t.Fatalf("Get after reopen: %v", err)
	}
	if got.Value() != 42000 {
		t.Errorf("Value after reopen = %d; want 42000", got.Value())
	}
	if got.Address() != "persist-addr" {
		t.Errorf("Address after reopen = %q; want %q", got.Address(), "persist-addr")
	}

	gotUndo, err := repo2.GetUndoEntry(10)
	if err != nil {
		t.Fatalf("GetUndoEntry after reopen: %v", err)
	}
	if gotUndo.BlockHeight != 10 {
		t.Errorf("Undo BlockHeight after reopen = %d; want 10", gotUndo.BlockHeight)
	}
}

func TestUTXORepoCompositeKey36Bytes(t *testing.T) {
	txID := block.DoubleSHA256([]byte("key-test"))
	key := utxoKey(txID, 42)
	if len(key) != 36 {
		t.Errorf("utxoKey length = %d; want 36", len(key))
	}
}

func TestUTXOStorageModelRoundTrip(t *testing.T) {
	txID := block.DoubleSHA256([]byte("model-test"))
	original := utxo.NewUTXO(txID, 3, 999, "test-addr")

	model := UTXOModelFromDomain(original)

	// Marshal/unmarshal
	data, err := json.Marshal(model)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}

	var decoded UTXOModel
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}

	restored, err := decoded.ToDomain()
	if err != nil {
		t.Fatalf("ToDomain: %v", err)
	}

	if restored.TxID() != original.TxID() {
		t.Errorf("TxID mismatch after roundtrip")
	}
	if restored.Vout() != original.Vout() {
		t.Errorf("Vout mismatch: got %d, want %d", restored.Vout(), original.Vout())
	}
	if restored.Value() != original.Value() {
		t.Errorf("Value mismatch: got %d, want %d", restored.Value(), original.Value())
	}
	if restored.Address() != original.Address() {
		t.Errorf("Address mismatch: got %q, want %q", restored.Address(), original.Address())
	}
}
