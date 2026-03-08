package utxo

import (
	"fmt"
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

// --- Error-returning mock repo for error path tests ---

type errRepo struct {
	memRepo
	putErr       error
	deleteErr    error
	getErr       error
	getByAddrErr error
	// Track call counts to fail on specific calls
	putCount    int
	putFailAt   int // fail on Nth put call (0 = never)
	deleteCount int
	deleteFailAt int
}

func newErrRepo() *errRepo {
	return &errRepo{
		memRepo: *newMemRepo(),
	}
}

func (e *errRepo) Put(u UTXO) error {
	e.putCount++
	if e.putFailAt > 0 && e.putCount >= e.putFailAt {
		return e.putErr
	}
	if e.putErr != nil && e.putFailAt == 0 {
		return e.putErr
	}
	return e.memRepo.Put(u)
}

func (e *errRepo) Get(txID block.Hash, vout uint32) (UTXO, error) {
	if e.getErr != nil {
		return UTXO{}, e.getErr
	}
	return e.memRepo.Get(txID, vout)
}

func (e *errRepo) Delete(txID block.Hash, vout uint32) error {
	e.deleteCount++
	if e.deleteFailAt > 0 && e.deleteCount >= e.deleteFailAt {
		return e.deleteErr
	}
	if e.deleteErr != nil && e.deleteFailAt == 0 {
		return e.deleteErr
	}
	return e.memRepo.Delete(txID, vout)
}

func (e *errRepo) GetByAddress(address string) ([]UTXO, error) {
	if e.getByAddrErr != nil {
		return nil, e.getByAddrErr
	}
	return e.memRepo.GetByAddress(address)
}

// --- UndoBlock error path tests ---

func TestUndoBlock_ErrorPaths(t *testing.T) {
	tests := []struct {
		name    string
		undo    *UndoEntry
		setup   func(repo *errRepo)
		wantErr string
	}{
		{
			name: "invalid created txID hex",
			undo: &UndoEntry{
				Created: []UTXORef{{TxID: "not-valid-hex", Vout: 0}},
			},
			wantErr: "parse created txid",
		},
		{
			name: "repo Delete error during undo of created UTXO",
			undo: &UndoEntry{
				Created: []UTXORef{{TxID: block.DoubleSHA256([]byte("tx1")).String(), Vout: 0}},
			},
			setup: func(repo *errRepo) {
				// Put the UTXO so it exists, then configure delete to fail
				txID := block.DoubleSHA256([]byte("tx1"))
				_ = repo.memRepo.Put(NewUTXO(txID, 0, 1000, "addr"))
				repo.deleteErr = fmt.Errorf("disk full")
			},
			wantErr: "delete created utxo",
		},
		{
			name: "invalid spent txID hex",
			undo: &UndoEntry{
				Created: []UTXORef{},
				Spent:   []SpentUTXO{{TxID: "not-valid-hex", Vout: 0, Value: 1000, Address: "addr"}},
			},
			wantErr: "parse spent txid",
		},
		{
			name: "repo Put error during undo of spent UTXO",
			undo: &UndoEntry{
				Created: []UTXORef{},
				Spent:   []SpentUTXO{{TxID: block.DoubleSHA256([]byte("tx2")).String(), Vout: 0, Value: 1000, Address: "addr"}},
			},
			setup: func(repo *errRepo) {
				repo.putErr = fmt.Errorf("disk full")
			},
			wantErr: "restore spent utxo",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := newErrRepo()
			if tt.setup != nil {
				tt.setup(repo)
			}
			set := NewSet(repo)

			err := set.UndoBlock(tt.undo)
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.wantErr)
		})
	}
}

// --- ApplyBlock repo error tests ---

func TestApplyBlock_RepoPutError(t *testing.T) {
	repo := newErrRepo()
	repo.putErr = fmt.Errorf("disk full")
	set := NewSet(repo)

	coinbase := tx.NewCoinbaseTx("miner", 5_000_000_000)
	_, err := set.ApplyBlock(0, []*tx.Transaction{coinbase})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "put utxo")
}

func TestApplyBlock_RepoDeleteError(t *testing.T) {
	repo := newErrRepo()
	set := NewSet(repo)

	// First apply a coinbase to create a UTXO
	coinbase := tx.NewCoinbaseTx("alice", 5_000_000_000)
	_, err := set.ApplyBlock(0, []*tx.Transaction{coinbase})
	require.NoError(t, err)

	// Now configure delete to fail and try to spend the UTXO
	repo.deleteErr = fmt.Errorf("disk full")
	input := tx.NewTxInput(coinbase.ID(), 0)
	spendTx := tx.NewTransaction([]tx.TxInput{input}, []tx.TxOutput{tx.NewTxOutput(5_000_000_000, "bob")})
	coinbase2 := tx.NewCoinbaseTx("miner", 5_000_000_000)

	_, err = set.ApplyBlock(1, []*tx.Transaction{coinbase2, spendTx})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "delete spent utxo")
}

func TestApplyBlock_RepoGetError(t *testing.T) {
	repo := newErrRepo()
	set := NewSet(repo)

	// Apply coinbase first
	coinbase := tx.NewCoinbaseTx("alice", 5_000_000_000)
	_, err := set.ApplyBlock(0, []*tx.Transaction{coinbase})
	require.NoError(t, err)

	// Configure Get to fail (simulating corrupt data)
	repo.getErr = fmt.Errorf("corrupt data")
	input := tx.NewTxInput(coinbase.ID(), 0)
	spendTx := tx.NewTransaction([]tx.TxInput{input}, []tx.TxOutput{tx.NewTxOutput(5_000_000_000, "bob")})
	coinbase2 := tx.NewCoinbaseTx("miner", 5_000_000_000)

	_, err = set.ApplyBlock(1, []*tx.Transaction{coinbase2, spendTx})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "get utxo")
}

// --- GetBalance repo error test ---

func TestGetBalance_RepoError(t *testing.T) {
	repo := newErrRepo()
	repo.getByAddrErr = fmt.Errorf("db connection lost")
	set := NewSet(repo)

	_, err := set.GetBalance("someaddr")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "get utxos by address")
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
