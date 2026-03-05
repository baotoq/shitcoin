package utxo

import (
	"testing"

	"github.com/baotoq/shitcoin/internal/domain/block"
	"github.com/baotoq/shitcoin/internal/domain/tx"
)

// --- In-memory repository for domain tests ---

type memRepo struct {
	utxos map[string]UTXO
	undos map[uint64]*UndoEntry
}

func newMemRepo() *memRepo {
	return &memRepo{
		utxos: make(map[string]UTXO),
		undos: make(map[uint64]*UndoEntry),
	}
}

func (m *memRepo) Put(u UTXO) error {
	m.utxos[u.Key()] = u
	return nil
}

func (m *memRepo) Get(txID block.Hash, vout uint32) (UTXO, error) {
	u := NewUTXO(txID, vout, 0, "")
	val, ok := m.utxos[u.Key()]
	if !ok {
		return UTXO{}, ErrUTXONotFound
	}
	return val, nil
}

func (m *memRepo) Delete(txID block.Hash, vout uint32) error {
	u := NewUTXO(txID, vout, 0, "")
	key := u.Key()
	if _, ok := m.utxos[key]; !ok {
		return ErrUTXONotFound
	}
	delete(m.utxos, key)
	return nil
}

func (m *memRepo) GetByAddress(address string) ([]UTXO, error) {
	var result []UTXO
	for _, u := range m.utxos {
		if u.Address() == address {
			result = append(result, u)
		}
	}
	return result, nil
}

func (m *memRepo) SaveUndoEntry(entry *UndoEntry) error {
	m.undos[entry.BlockHeight] = entry
	return nil
}

func (m *memRepo) GetUndoEntry(blockHeight uint64) (*UndoEntry, error) {
	entry, ok := m.undos[blockHeight]
	if !ok {
		return nil, ErrUndoEntryNotFound
	}
	return entry, nil
}

func (m *memRepo) DeleteUndoEntry(blockHeight uint64) error {
	delete(m.undos, blockHeight)
	return nil
}

// --- UTXO value object tests ---

func TestUTXOGetters(t *testing.T) {
	txID := block.DoubleSHA256([]byte("tx1"))
	u := NewUTXO(txID, 0, 5000, "addr1")

	if u.TxID() != txID {
		t.Errorf("TxID() mismatch")
	}
	if u.Vout() != 0 {
		t.Errorf("Vout() = %d; want 0", u.Vout())
	}
	if u.Value() != 5000 {
		t.Errorf("Value() = %d; want 5000", u.Value())
	}
	if u.Address() != "addr1" {
		t.Errorf("Address() = %q; want %q", u.Address(), "addr1")
	}
}

func TestUTXOKey(t *testing.T) {
	txID := block.DoubleSHA256([]byte("tx1"))
	u := NewUTXO(txID, 2, 1000, "addr1")
	key := u.Key()

	if key == "" {
		t.Error("Key() returned empty string")
	}
	// Key should include txid hex and vout
	expected := txID.String() + ":2"
	if key != expected {
		t.Errorf("Key() = %q; want %q", key, expected)
	}
}

// --- UTXOSet tests ---

func TestApplyBlockCoinbase(t *testing.T) {
	repo := newMemRepo()
	set := NewSet(repo)

	coinbase := tx.NewCoinbaseTx("miner1", 5_000_000_000)
	undo, err := set.ApplyBlock(0, []*tx.Transaction{coinbase})
	if err != nil {
		t.Fatalf("ApplyBlock: %v", err)
	}

	// Should have created 1 UTXO
	if len(undo.Created) != 1 {
		t.Fatalf("undo.Created length = %d; want 1", len(undo.Created))
	}
	// No spent UTXOs for coinbase
	if len(undo.Spent) != 0 {
		t.Errorf("undo.Spent length = %d; want 0", len(undo.Spent))
	}

	// Verify the UTXO exists in the set
	utxos, err := set.GetByAddress("miner1")
	if err != nil {
		t.Fatalf("GetByAddress: %v", err)
	}
	if len(utxos) != 1 {
		t.Fatalf("GetByAddress returned %d UTXOs; want 1", len(utxos))
	}
	if utxos[0].Value() != 5_000_000_000 {
		t.Errorf("UTXO value = %d; want 5000000000", utxos[0].Value())
	}
}

func TestApplyBlockSpendAndCreate(t *testing.T) {
	repo := newMemRepo()
	set := NewSet(repo)

	// First: create a coinbase UTXO
	coinbase := tx.NewCoinbaseTx("alice", 5_000_000_000)
	_, err := set.ApplyBlock(0, []*tx.Transaction{coinbase})
	if err != nil {
		t.Fatalf("ApplyBlock coinbase: %v", err)
	}

	// Now create a transaction spending that UTXO
	input := tx.NewTxInput(coinbase.ID(), 0)
	output1 := tx.NewTxOutput(3_000_000_000, "bob")
	output2 := tx.NewTxOutput(2_000_000_000, "alice") // change
	spendTx := tx.NewTransaction([]tx.TxInput{input}, []tx.TxOutput{output1, output2})

	coinbase2 := tx.NewCoinbaseTx("miner2", 5_000_000_000)
	undo, err := set.ApplyBlock(1, []*tx.Transaction{coinbase2, spendTx})
	if err != nil {
		t.Fatalf("ApplyBlock spend: %v", err)
	}

	// Should have spent 1 UTXO and created 3 (coinbase2 output + 2 from spendTx)
	if len(undo.Spent) != 1 {
		t.Errorf("undo.Spent = %d; want 1", len(undo.Spent))
	}
	if len(undo.Created) != 3 {
		t.Errorf("undo.Created = %d; want 3", len(undo.Created))
	}

	// Verify balances
	bobBalance, err := set.GetBalance("bob")
	if err != nil {
		t.Fatalf("GetBalance bob: %v", err)
	}
	if bobBalance != 3_000_000_000 {
		t.Errorf("bob balance = %d; want 3000000000", bobBalance)
	}

	aliceBalance, err := set.GetBalance("alice")
	if err != nil {
		t.Fatalf("GetBalance alice: %v", err)
	}
	if aliceBalance != 2_000_000_000 {
		t.Errorf("alice balance = %d; want 2000000000", aliceBalance)
	}
}

func TestApplyBlockDoubleSpend(t *testing.T) {
	repo := newMemRepo()
	set := NewSet(repo)

	// Create a coinbase UTXO
	coinbase := tx.NewCoinbaseTx("alice", 5_000_000_000)
	_, err := set.ApplyBlock(0, []*tx.Transaction{coinbase})
	if err != nil {
		t.Fatalf("ApplyBlock coinbase: %v", err)
	}

	// Two transactions in same block spending the same UTXO
	input := tx.NewTxInput(coinbase.ID(), 0)
	spendTx1 := tx.NewTransaction([]tx.TxInput{input}, []tx.TxOutput{tx.NewTxOutput(5_000_000_000, "bob")})
	spendTx2 := tx.NewTransaction([]tx.TxInput{input}, []tx.TxOutput{tx.NewTxOutput(5_000_000_000, "charlie")})

	coinbase2 := tx.NewCoinbaseTx("miner", 5_000_000_000)
	_, err = set.ApplyBlock(1, []*tx.Transaction{coinbase2, spendTx1, spendTx2})
	if err == nil {
		t.Fatal("expected double-spend error, got nil")
	}
}

func TestUndoBlock(t *testing.T) {
	repo := newMemRepo()
	set := NewSet(repo)

	// Apply coinbase block
	coinbase := tx.NewCoinbaseTx("alice", 5_000_000_000)
	_, err := set.ApplyBlock(0, []*tx.Transaction{coinbase})
	if err != nil {
		t.Fatalf("ApplyBlock coinbase: %v", err)
	}

	// Apply a spend block
	input := tx.NewTxInput(coinbase.ID(), 0)
	spendTx := tx.NewTransaction([]tx.TxInput{input}, []tx.TxOutput{tx.NewTxOutput(5_000_000_000, "bob")})
	coinbase2 := tx.NewCoinbaseTx("miner", 5_000_000_000)
	undo, err := set.ApplyBlock(1, []*tx.Transaction{coinbase2, spendTx})
	if err != nil {
		t.Fatalf("ApplyBlock spend: %v", err)
	}

	// Verify bob has funds
	bobBalance, _ := set.GetBalance("bob")
	if bobBalance != 5_000_000_000 {
		t.Errorf("bob balance before undo = %d; want 5000000000", bobBalance)
	}

	// Undo the spend block
	err = set.UndoBlock(undo)
	if err != nil {
		t.Fatalf("UndoBlock: %v", err)
	}

	// Bob should have nothing, alice should have her original UTXO back
	bobBalance, _ = set.GetBalance("bob")
	if bobBalance != 0 {
		t.Errorf("bob balance after undo = %d; want 0", bobBalance)
	}

	aliceBalance, _ := set.GetBalance("alice")
	if aliceBalance != 5_000_000_000 {
		t.Errorf("alice balance after undo = %d; want 5000000000", aliceBalance)
	}
}

func TestGetBalance(t *testing.T) {
	repo := newMemRepo()
	set := NewSet(repo)

	// Two coinbase blocks paying same address but different rewards to produce unique TX IDs
	cb1 := tx.NewCoinbaseTx("miner", 5_000_000_000)
	_, err := set.ApplyBlock(0, []*tx.Transaction{cb1})
	if err != nil {
		t.Fatalf("ApplyBlock 0: %v", err)
	}

	cb2 := tx.NewCoinbaseTx("miner", 5_000_000_001) // different reward for unique tx ID
	_, err = set.ApplyBlock(1, []*tx.Transaction{cb2})
	if err != nil {
		t.Fatalf("ApplyBlock 1: %v", err)
	}

	balance, err := set.GetBalance("miner")
	if err != nil {
		t.Fatalf("GetBalance: %v", err)
	}
	if balance != 10_000_000_001 {
		t.Errorf("balance = %d; want 10000000001", balance)
	}

	// Unknown address should have zero balance
	balance, err = set.GetBalance("nobody")
	if err != nil {
		t.Fatalf("GetBalance nobody: %v", err)
	}
	if balance != 0 {
		t.Errorf("nobody balance = %d; want 0", balance)
	}
}

func TestGetUTXO(t *testing.T) {
	repo := newMemRepo()
	set := NewSet(repo)

	coinbase := tx.NewCoinbaseTx("alice", 5_000_000_000)
	_, err := set.ApplyBlock(0, []*tx.Transaction{coinbase})
	if err != nil {
		t.Fatalf("ApplyBlock: %v", err)
	}

	// Get existing UTXO
	u, err := set.Get(coinbase.ID(), 0)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if u.Value() != 5_000_000_000 {
		t.Errorf("Value = %d; want 5000000000", u.Value())
	}

	// Get non-existent UTXO
	_, err = set.Get(block.DoubleSHA256([]byte("fake")), 0)
	if err != ErrUTXONotFound {
		t.Errorf("expected ErrUTXONotFound, got: %v", err)
	}
}
