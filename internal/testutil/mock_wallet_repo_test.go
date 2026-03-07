package testutil

import (
	"testing"

	"github.com/baotoq/shitcoin/internal/domain/wallet"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Compile-time interface compliance check (also in mock file).
var _ wallet.Repository = (*MockWalletRepo)(nil)

func TestMockWalletRepo_SaveAndGetByAddress(t *testing.T) {
	repo := NewMockWalletRepo()
	w := MustCreateWallet(t)

	err := repo.Save(w)
	require.NoError(t, err)

	got, err := repo.GetByAddress(w.Address())
	require.NoError(t, err)
	assert.Equal(t, w.Address(), got.Address())
}

func TestMockWalletRepo_GetByAddress_NotFound(t *testing.T) {
	repo := NewMockWalletRepo()

	_, err := repo.GetByAddress("1NonExistent")
	assert.ErrorIs(t, err, wallet.ErrWalletNotFound)
}

func TestMockWalletRepo_ListAddresses(t *testing.T) {
	repo := NewMockWalletRepo()

	// Empty repo
	addrs, err := repo.ListAddresses()
	require.NoError(t, err)
	assert.Empty(t, addrs)

	// Add wallets
	w1 := MustCreateWallet(t)
	w2 := MustCreateWallet(t)
	require.NoError(t, repo.Save(w1))
	require.NoError(t, repo.Save(w2))

	addrs, err = repo.ListAddresses()
	require.NoError(t, err)
	assert.Len(t, addrs, 2)
	assert.Contains(t, addrs, w1.Address())
	assert.Contains(t, addrs, w2.Address())
}
