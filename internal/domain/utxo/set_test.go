package utxo

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

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

	assert.Equal(t, txID, u.TxID())
	assert.Equal(t, uint32(0), u.Vout())
	assert.Equal(t, int64(5000), u.Value())
	assert.Equal(t, "addr1", u.Address())
}

func TestUTXOKey(t *testing.T) {
	txID := block.DoubleSHA256([]byte("tx1"))
	u := NewUTXO(txID, 2, 1000, "addr1")
	key := u.Key()

	assert.NotEmpty(t, key)
	// Key should include txid hex and vout
	expected := txID.String() + ":2"
	assert.Equal(t, expected, key)
}

// --- UTXOSet tests ---

func TestApplyBlockCoinbase(t *testing.T) {
	repo := newMemRepo()
	set := NewSet(repo)

	coinbase := tx.NewCoinbaseTx("miner1", 5_000_000_000)
	undo, err := set.ApplyBlock(0, []*tx.Transaction{coinbase})
	require.NoError(t, err)

	// Should have created 1 UTXO
	require.Len(t, undo.Created, 1)
	// No spent UTXOs for coinbase
	assert.Empty(t, undo.Spent)

	// Verify the UTXO exists in the set
	utxos, err := set.GetByAddress("miner1")
	require.NoError(t, err)
	require.Len(t, utxos, 1)
	assert.Equal(t, int64(5_000_000_000), utxos[0].Value())
}

func TestApplyBlockSpendAndCreate(t *testing.T) {
	repo := newMemRepo()
	set := NewSet(repo)

	// First: create a coinbase UTXO
	coinbase := tx.NewCoinbaseTx("alice", 5_000_000_000)
	_, err := set.ApplyBlock(0, []*tx.Transaction{coinbase})
	require.NoError(t, err)

	// Now create a transaction spending that UTXO
	input := tx.NewTxInput(coinbase.ID(), 0)
	output1 := tx.NewTxOutput(3_000_000_000, "bob")
	output2 := tx.NewTxOutput(2_000_000_000, "alice") // change
	spendTx := tx.NewTransaction([]tx.TxInput{input}, []tx.TxOutput{output1, output2})

	coinbase2 := tx.NewCoinbaseTx("miner2", 5_000_000_000)
	undo, err := set.ApplyBlock(1, []*tx.Transaction{coinbase2, spendTx})
	require.NoError(t, err)

	// Should have spent 1 UTXO and created 3 (coinbase2 output + 2 from spendTx)
	assert.Len(t, undo.Spent, 1)
	assert.Len(t, undo.Created, 3)

	// Verify balances
	bobBalance, err := set.GetBalance("bob")
	require.NoError(t, err)
	assert.Equal(t, int64(3_000_000_000), bobBalance)

	aliceBalance, err := set.GetBalance("alice")
	require.NoError(t, err)
	assert.Equal(t, int64(2_000_000_000), aliceBalance)
}

func TestApplyBlockDoubleSpend(t *testing.T) {
	repo := newMemRepo()
	set := NewSet(repo)

	// Create a coinbase UTXO
	coinbase := tx.NewCoinbaseTx("alice", 5_000_000_000)
	_, err := set.ApplyBlock(0, []*tx.Transaction{coinbase})
	require.NoError(t, err)

	// Two transactions in same block spending the same UTXO
	input := tx.NewTxInput(coinbase.ID(), 0)
	spendTx1 := tx.NewTransaction([]tx.TxInput{input}, []tx.TxOutput{tx.NewTxOutput(5_000_000_000, "bob")})
	spendTx2 := tx.NewTransaction([]tx.TxInput{input}, []tx.TxOutput{tx.NewTxOutput(5_000_000_000, "charlie")})

	coinbase2 := tx.NewCoinbaseTx("miner", 5_000_000_000)
	_, err = set.ApplyBlock(1, []*tx.Transaction{coinbase2, spendTx1, spendTx2})
	require.Error(t, err)
}

func TestUndoBlock(t *testing.T) {
	repo := newMemRepo()
	set := NewSet(repo)

	// Apply coinbase block
	coinbase := tx.NewCoinbaseTx("alice", 5_000_000_000)
	_, err := set.ApplyBlock(0, []*tx.Transaction{coinbase})
	require.NoError(t, err)

	// Apply a spend block
	input := tx.NewTxInput(coinbase.ID(), 0)
	spendTx := tx.NewTransaction([]tx.TxInput{input}, []tx.TxOutput{tx.NewTxOutput(5_000_000_000, "bob")})
	coinbase2 := tx.NewCoinbaseTx("miner", 5_000_000_000)
	undo, err := set.ApplyBlock(1, []*tx.Transaction{coinbase2, spendTx})
	require.NoError(t, err)

	// Verify bob has funds
	bobBalance, _ := set.GetBalance("bob")
	assert.Equal(t, int64(5_000_000_000), bobBalance)

	// Undo the spend block
	require.NoError(t, set.UndoBlock(undo))

	// Bob should have nothing, alice should have her original UTXO back
	bobBalance, _ = set.GetBalance("bob")
	assert.Equal(t, int64(0), bobBalance)

	aliceBalance, _ := set.GetBalance("alice")
	assert.Equal(t, int64(5_000_000_000), aliceBalance)
}

func TestGetBalance(t *testing.T) {
	repo := newMemRepo()
	set := NewSet(repo)

	// Two coinbase blocks paying same address but different rewards to produce unique TX IDs
	cb1 := tx.NewCoinbaseTx("miner", 5_000_000_000)
	_, err := set.ApplyBlock(0, []*tx.Transaction{cb1})
	require.NoError(t, err)

	cb2 := tx.NewCoinbaseTx("miner", 5_000_000_001) // different reward for unique tx ID
	_, err = set.ApplyBlock(1, []*tx.Transaction{cb2})
	require.NoError(t, err)

	balance, err := set.GetBalance("miner")
	require.NoError(t, err)
	assert.Equal(t, int64(10_000_000_001), balance)

	// Unknown address should have zero balance
	balance, err = set.GetBalance("nobody")
	require.NoError(t, err)
	assert.Equal(t, int64(0), balance)
}

func TestGetUTXO(t *testing.T) {
	repo := newMemRepo()
	set := NewSet(repo)

	coinbase := tx.NewCoinbaseTx("alice", 5_000_000_000)
	_, err := set.ApplyBlock(0, []*tx.Transaction{coinbase})
	require.NoError(t, err)

	// Get existing UTXO
	u, err := set.Get(coinbase.ID(), 0)
	require.NoError(t, err)
	assert.Equal(t, int64(5_000_000_000), u.Value())

	// Get non-existent UTXO
	_, err = set.Get(block.DoubleSHA256([]byte("fake")), 0)
	assert.ErrorIs(t, err, ErrUTXONotFound)
}
