package mempool

import (
	"sync"
	"testing"

	"github.com/btcsuite/btcd/btcec/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/baotoq/shitcoin/internal/domain/block"
	"github.com/baotoq/shitcoin/internal/domain/tx"
	"github.com/baotoq/shitcoin/internal/domain/utxo"
)

// --- In-memory UTXO repository for mempool tests ---

type memRepo struct {
	utxos map[string]utxo.UTXO
	undos map[uint64]*utxo.UndoEntry
}

func newMemRepo() *memRepo {
	return &memRepo{
		utxos: make(map[string]utxo.UTXO),
		undos: make(map[uint64]*utxo.UndoEntry),
	}
}

func (m *memRepo) Put(u utxo.UTXO) error {
	m.utxos[u.Key()] = u
	return nil
}

func (m *memRepo) Get(txID block.Hash, vout uint32) (utxo.UTXO, error) {
	u := utxo.NewUTXO(txID, vout, 0, "")
	val, ok := m.utxos[u.Key()]
	if !ok {
		return utxo.UTXO{}, utxo.ErrUTXONotFound
	}
	return val, nil
}

func (m *memRepo) Delete(txID block.Hash, vout uint32) error {
	u := utxo.NewUTXO(txID, vout, 0, "")
	key := u.Key()
	if _, ok := m.utxos[key]; !ok {
		return utxo.ErrUTXONotFound
	}
	delete(m.utxos, key)
	return nil
}

func (m *memRepo) GetByAddress(address string) ([]utxo.UTXO, error) {
	var result []utxo.UTXO
	for _, u := range m.utxos {
		if u.Address() == address {
			result = append(result, u)
		}
	}
	return result, nil
}

func (m *memRepo) SaveUndoEntry(entry *utxo.UndoEntry) error {
	m.undos[entry.BlockHeight] = entry
	return nil
}

func (m *memRepo) GetUndoEntry(blockHeight uint64) (*utxo.UndoEntry, error) {
	entry, ok := m.undos[blockHeight]
	if !ok {
		return nil, utxo.ErrUndoEntryNotFound
	}
	return entry, nil
}

func (m *memRepo) DeleteUndoEntry(blockHeight uint64) error {
	delete(m.undos, blockHeight)
	return nil
}

// buildSignedTx creates a valid signed spending transaction against the UTXO set.
// 1. Creates a coinbase, applies it to the UTXO set
// 2. Builds a spending TX referencing the coinbase output, signs it
func buildSignedTx(t *testing.T, utxoSet *utxo.Set, privKey *btcec.PrivateKey, address string) *tx.Transaction {
	t.Helper()

	// Create coinbase to fund the UTXO set
	coinbase := tx.NewCoinbaseTx(address, 5_000_000_000)
	_, err := utxoSet.ApplyBlock(0, []*tx.Transaction{coinbase})
	require.NoError(t, err)

	// Build spending TX
	input := tx.NewTxInput(coinbase.ID(), 0)
	output := tx.NewTxOutput(5_000_000_000, "recipient")
	spendTx := tx.NewTransaction([]tx.TxInput{input}, []tx.TxOutput{output})

	// Sign it
	require.NoError(t, tx.SignTransaction(spendTx, privKey))

	return spendTx
}

func TestAdd_ValidTransaction(t *testing.T) {
	repo := newMemRepo()
	utxoSet := utxo.NewSet(repo)
	privKey, _ := btcec.NewPrivateKey()
	address := "testaddr"

	spendTx := buildSignedTx(t, utxoSet, privKey, address)

	mp := New(utxoSet)
	require.NoError(t, mp.Add(spendTx))
	assert.Equal(t, 1, mp.Count())
}

func TestAdd_Duplicate(t *testing.T) {
	repo := newMemRepo()
	utxoSet := utxo.NewSet(repo)
	privKey, _ := btcec.NewPrivateKey()

	spendTx := buildSignedTx(t, utxoSet, privKey, "testaddr")

	mp := New(utxoSet)
	_ = mp.Add(spendTx)

	err := mp.Add(spendTx)
	assert.ErrorIs(t, err, ErrDuplicate)
}

func TestAdd_DoubleSpend(t *testing.T) {
	repo := newMemRepo()
	utxoSet := utxo.NewSet(repo)
	privKey, _ := btcec.NewPrivateKey()
	address := "testaddr"

	// Create a coinbase to fund the UTXO set
	coinbase := tx.NewCoinbaseTx(address, 5_000_000_000)
	_, err := utxoSet.ApplyBlock(0, []*tx.Transaction{coinbase})
	require.NoError(t, err)

	// Create two TXs spending the same UTXO
	input := tx.NewTxInput(coinbase.ID(), 0)

	spendTx1 := tx.NewTransaction([]tx.TxInput{input}, []tx.TxOutput{tx.NewTxOutput(5_000_000_000, "bob")})
	_ = tx.SignTransaction(spendTx1, privKey)

	spendTx2 := tx.NewTransaction([]tx.TxInput{input}, []tx.TxOutput{tx.NewTxOutput(5_000_000_000, "charlie")})
	_ = tx.SignTransaction(spendTx2, privKey)

	mp := New(utxoSet)
	_ = mp.Add(spendTx1)

	err = mp.Add(spendTx2)
	assert.ErrorIs(t, err, ErrDoubleSpend)
}

func TestAdd_InvalidSignature(t *testing.T) {
	repo := newMemRepo()
	utxoSet := utxo.NewSet(repo)

	// Create coinbase
	coinbase := tx.NewCoinbaseTx("testaddr", 5_000_000_000)
	_, _ = utxoSet.ApplyBlock(0, []*tx.Transaction{coinbase})

	// Create unsigned spending TX (no signature)
	input := tx.NewTxInput(coinbase.ID(), 0)
	unsignedTx := tx.NewTransaction([]tx.TxInput{input}, []tx.TxOutput{tx.NewTxOutput(5_000_000_000, "bob")})

	mp := New(utxoSet)
	err := mp.Add(unsignedTx)
	assert.ErrorIs(t, err, ErrInvalidSignature)
}

func TestAdd_UTXONotFound(t *testing.T) {
	repo := newMemRepo()
	utxoSet := utxo.NewSet(repo)
	privKey, _ := btcec.NewPrivateKey()

	// Create a TX referencing a non-existent UTXO (no coinbase applied)
	fakeHash := block.DoubleSHA256([]byte("fake"))
	input := tx.NewTxInput(fakeHash, 0)
	fakeTx := tx.NewTransaction([]tx.TxInput{input}, []tx.TxOutput{tx.NewTxOutput(1000, "bob")})
	_ = tx.SignTransaction(fakeTx, privKey)

	mp := New(utxoSet)
	err := mp.Add(fakeTx)
	assert.ErrorIs(t, err, ErrUTXONotFound)
}

func TestDrainAll(t *testing.T) {
	repo := newMemRepo()
	utxoSet := utxo.NewSet(repo)
	privKey, _ := btcec.NewPrivateKey()

	// Create 3 coinbases with different rewards for unique TX IDs
	var txs []*tx.Transaction
	for i := 0; i < 3; i++ {
		coinbase := tx.NewCoinbaseTx("addr", int64(5_000_000_000+i))
		_, _ = utxoSet.ApplyBlock(uint64(i), []*tx.Transaction{coinbase})

		input := tx.NewTxInput(coinbase.ID(), 0)
		spendTx := tx.NewTransaction([]tx.TxInput{input}, []tx.TxOutput{tx.NewTxOutput(int64(5_000_000_000+i), "bob")})
		_ = tx.SignTransaction(spendTx, privKey)
		txs = append(txs, spendTx)
	}

	mp := New(utxoSet)
	for _, transaction := range txs {
		require.NoError(t, mp.Add(transaction))
	}

	drained := mp.DrainAll()
	assert.Len(t, drained, 3)
	assert.Equal(t, 0, mp.Count())
}

func TestRemove(t *testing.T) {
	repo := newMemRepo()
	utxoSet := utxo.NewSet(repo)
	privKey, _ := btcec.NewPrivateKey()

	// Create 2 distinct txs
	var txs []*tx.Transaction
	for i := 0; i < 2; i++ {
		coinbase := tx.NewCoinbaseTx("addr", int64(5_000_000_000+i))
		_, _ = utxoSet.ApplyBlock(uint64(i), []*tx.Transaction{coinbase})

		input := tx.NewTxInput(coinbase.ID(), 0)
		spendTx := tx.NewTransaction([]tx.TxInput{input}, []tx.TxOutput{tx.NewTxOutput(int64(5_000_000_000+i), "bob")})
		_ = tx.SignTransaction(spendTx, privKey)
		txs = append(txs, spendTx)
	}

	mp := New(utxoSet)
	for _, transaction := range txs {
		_ = mp.Add(transaction)
	}

	// Remove the first tx
	mp.Remove([]block.Hash{txs[0].ID()})

	assert.Equal(t, 1, mp.Count())
}

func TestTransactions(t *testing.T) {
	repo := newMemRepo()
	utxoSet := utxo.NewSet(repo)
	privKey, _ := btcec.NewPrivateKey()

	var txs []*tx.Transaction
	for i := 0; i < 2; i++ {
		coinbase := tx.NewCoinbaseTx("addr", int64(5_000_000_000+i))
		_, _ = utxoSet.ApplyBlock(uint64(i), []*tx.Transaction{coinbase})

		input := tx.NewTxInput(coinbase.ID(), 0)
		spendTx := tx.NewTransaction([]tx.TxInput{input}, []tx.TxOutput{tx.NewTxOutput(int64(5_000_000_000+i), "bob")})
		_ = tx.SignTransaction(spendTx, privKey)
		txs = append(txs, spendTx)
	}

	mp := New(utxoSet)
	for _, transaction := range txs {
		_ = mp.Add(transaction)
	}

	all := mp.Transactions()
	assert.Len(t, all, 2)
}

func TestConcurrentAccess(t *testing.T) {
	repo := newMemRepo()
	utxoSet := utxo.NewSet(repo)
	privKey, _ := btcec.NewPrivateKey()

	// Pre-create 10 coinbases each funding a unique UTXO
	var txs []*tx.Transaction
	for i := 0; i < 10; i++ {
		coinbase := tx.NewCoinbaseTx("addr", int64(5_000_000_000+i))
		_, _ = utxoSet.ApplyBlock(uint64(i), []*tx.Transaction{coinbase})

		input := tx.NewTxInput(coinbase.ID(), 0)
		spendTx := tx.NewTransaction([]tx.TxInput{input}, []tx.TxOutput{tx.NewTxOutput(int64(5_000_000_000+i), "bob")})
		_ = tx.SignTransaction(spendTx, privKey)
		txs = append(txs, spendTx)
	}

	mp := New(utxoSet)

	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			_ = mp.Add(txs[idx])
		}(i)
	}
	wg.Wait()

	assert.Equal(t, 10, mp.Count())
}
