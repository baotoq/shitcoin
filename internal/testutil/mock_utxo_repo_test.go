package testutil

import (
	"testing"

	"github.com/baotoq/shitcoin/internal/domain/block"
	"github.com/baotoq/shitcoin/internal/domain/utxo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Compile-time interface compliance check (also in mock file).
var _ utxo.Repository = (*MockUTXORepo)(nil)

func TestMockUTXORepo_PutGet_Roundtrip(t *testing.T) {
	repo := NewMockUTXORepo()
	txID := block.DoubleSHA256([]byte("test-tx"))
	u := utxo.NewUTXO(txID, 0, 5000, "1TestAddr")

	err := repo.Put(u)
	require.NoError(t, err)

	got, err := repo.Get(txID, 0)
	require.NoError(t, err)
	assert.Equal(t, u.TxID(), got.TxID())
	assert.Equal(t, u.Vout(), got.Vout())
	assert.Equal(t, u.Value(), got.Value())
	assert.Equal(t, u.Address(), got.Address())
}

func TestMockUTXORepo_Get_NotFound(t *testing.T) {
	repo := NewMockUTXORepo()
	txID := block.DoubleSHA256([]byte("missing"))

	_, err := repo.Get(txID, 0)
	assert.ErrorIs(t, err, utxo.ErrUTXONotFound)
}

func TestMockUTXORepo_Delete(t *testing.T) {
	repo := NewMockUTXORepo()
	txID := block.DoubleSHA256([]byte("test-tx"))
	u := utxo.NewUTXO(txID, 0, 5000, "1TestAddr")

	require.NoError(t, repo.Put(u))
	require.NoError(t, repo.Delete(txID, 0))

	_, err := repo.Get(txID, 0)
	assert.ErrorIs(t, err, utxo.ErrUTXONotFound)
}

func TestMockUTXORepo_Delete_NotFound(t *testing.T) {
	repo := NewMockUTXORepo()
	txID := block.DoubleSHA256([]byte("missing"))

	err := repo.Delete(txID, 0)
	assert.ErrorIs(t, err, utxo.ErrUTXONotFound)
}

func TestMockUTXORepo_GetByAddress(t *testing.T) {
	repo := NewMockUTXORepo()
	txID1 := block.DoubleSHA256([]byte("tx1"))
	txID2 := block.DoubleSHA256([]byte("tx2"))

	require.NoError(t, repo.Put(utxo.NewUTXO(txID1, 0, 1000, "1Alice")))
	require.NoError(t, repo.Put(utxo.NewUTXO(txID2, 0, 2000, "1Alice")))
	require.NoError(t, repo.Put(utxo.NewUTXO(txID1, 1, 3000, "1Bob")))

	aliceUTXOs, err := repo.GetByAddress("1Alice")
	require.NoError(t, err)
	assert.Len(t, aliceUTXOs, 2)

	bobUTXOs, err := repo.GetByAddress("1Bob")
	require.NoError(t, err)
	assert.Len(t, bobUTXOs, 1)

	// Unknown address returns empty slice
	unknownUTXOs, err := repo.GetByAddress("1Unknown")
	require.NoError(t, err)
	assert.Empty(t, unknownUTXOs)
}

func TestMockUTXORepo_UndoEntry_Roundtrip(t *testing.T) {
	repo := NewMockUTXORepo()
	entry := &utxo.UndoEntry{
		BlockHeight: 5,
		Spent:       []utxo.SpentUTXO{{TxID: "abc", Vout: 0, Value: 100, Address: "1Test"}},
		Created:     []utxo.UTXORef{{TxID: "def", Vout: 0}},
	}

	require.NoError(t, repo.SaveUndoEntry(entry))

	got, err := repo.GetUndoEntry(5)
	require.NoError(t, err)
	assert.Equal(t, uint64(5), got.BlockHeight)
	assert.Len(t, got.Spent, 1)
	assert.Len(t, got.Created, 1)
}

func TestMockUTXORepo_GetUndoEntry_NotFound(t *testing.T) {
	repo := NewMockUTXORepo()

	_, err := repo.GetUndoEntry(99)
	assert.ErrorIs(t, err, utxo.ErrUndoEntryNotFound)
}

func TestMockUTXORepo_DeleteUndoEntry(t *testing.T) {
	repo := NewMockUTXORepo()
	entry := &utxo.UndoEntry{BlockHeight: 5}

	require.NoError(t, repo.SaveUndoEntry(entry))
	require.NoError(t, repo.DeleteUndoEntry(5))

	_, err := repo.GetUndoEntry(5)
	assert.ErrorIs(t, err, utxo.ErrUndoEntryNotFound)
}
