package jsonfile

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/baotoq/shitcoin/internal/domain/wallet"
)

func TestWalletRepo_SaveAndGet(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "wallets.json")

	repo, err := NewWalletRepo(filePath)
	require.NoError(t, err)

	w, err := wallet.NewWallet()
	require.NoError(t, err)

	require.NoError(t, repo.Save(w))

	got, err := repo.GetByAddress(w.Address())
	require.NoError(t, err)

	assert.Equal(t, w.Address(), got.Address())
	assert.Equal(t, w.PrivateKeyHex(), got.PrivateKeyHex())
}

func TestWalletRepo_ListAddresses(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "wallets.json")

	repo, err := NewWalletRepo(filePath)
	require.NoError(t, err)

	w1, _ := wallet.NewWallet()
	w2, _ := wallet.NewWallet()
	w3, _ := wallet.NewWallet()

	_ = repo.Save(w1)
	_ = repo.Save(w2)
	_ = repo.Save(w3)

	addresses, err := repo.ListAddresses()
	require.NoError(t, err)

	assert.Len(t, addresses, 3)

	// Check all addresses are present.
	addrMap := make(map[string]bool)
	for _, a := range addresses {
		addrMap[a] = true
	}
	for _, w := range []*wallet.Wallet{w1, w2, w3} {
		assert.True(t, addrMap[w.Address()], "address %q not found in list", w.Address())
	}
}

func TestWalletRepo_GetByAddress_NotFound(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "wallets.json")

	repo, err := NewWalletRepo(filePath)
	require.NoError(t, err)

	_, err = repo.GetByAddress("1NonExistentAddress")
	assert.ErrorIs(t, err, wallet.ErrWalletNotFound)
}

func TestWalletRepo_Persistence(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "wallets.json")

	// Create repo and save a wallet.
	repo1, err := NewWalletRepo(filePath)
	require.NoError(t, err)

	w, _ := wallet.NewWallet()
	_ = repo1.Save(w)

	// Create a new repo from the same file (simulates close/reopen).
	repo2, err := NewWalletRepo(filePath)
	require.NoError(t, err)

	got, err := repo2.GetByAddress(w.Address())
	require.NoError(t, err)
	assert.Equal(t, w.Address(), got.Address())
	assert.Equal(t, w.PrivateKeyHex(), got.PrivateKeyHex())
}

func TestWalletRepo_FileFormatIsReadableJSON(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "wallets.json")

	repo, err := NewWalletRepo(filePath)
	require.NoError(t, err)

	w, _ := wallet.NewWallet()
	_ = repo.Save(w)

	// Read the raw file and verify it's valid JSON.
	data, err := os.ReadFile(filePath)
	require.NoError(t, err)

	var parsed map[string]any
	require.NoError(t, json.Unmarshal(data, &parsed))

	// Verify the structure has a wallets array.
	wallets, ok := parsed["wallets"]
	require.True(t, ok, "JSON missing 'wallets' key")

	arr, ok := wallets.([]any)
	require.True(t, ok, "'wallets' is not an array")
	assert.Len(t, arr, 1)

	// Verify each entry has address and private_key_hex fields.
	entry, ok := arr[0].(map[string]any)
	require.True(t, ok, "wallet entry is not an object")
	assert.Contains(t, entry, "address")
	assert.Contains(t, entry, "private_key_hex")
}

func TestWalletRepo_CorruptFile(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "wallets.json")

	// Write invalid JSON.
	require.NoError(t, os.WriteFile(filePath, []byte(`{not valid json`), 0o644))

	_, err := NewWalletRepo(filePath)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unmarshal wallet file")
}

func TestWalletRepo_InvalidPrivateKey(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "wallets.json")

	// Write valid JSON with an invalid private key.
	data := `{"wallets":[{"address":"1test","private_key_hex":"not-a-hex-key"}]}`
	require.NoError(t, os.WriteFile(filePath, []byte(data), 0o644))

	_, err := NewWalletRepo(filePath)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "reconstruct wallet")
}

func TestWalletRepo_UnreadableFile(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("permission-based test not supported on Windows")
	}

	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "wallets.json")

	// Write a valid file, then make it unreadable.
	require.NoError(t, os.WriteFile(filePath, []byte(`{"wallets":[]}`), 0o644))
	require.NoError(t, os.Chmod(filePath, 0o000))
	t.Cleanup(func() {
		os.Chmod(filePath, 0o644) // Restore so t.TempDir cleanup succeeds.
	})

	_, err := NewWalletRepo(filePath)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "read wallet file")
}

func TestWalletRepo_FlushError_ReadOnlyDir(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("permission-based test not supported on Windows")
	}

	tmpDir := t.TempDir()
	subDir := filepath.Join(tmpDir, "readonly")
	require.NoError(t, os.MkdirAll(subDir, 0o755))
	filePath := filepath.Join(subDir, "wallets.json")

	// Create a valid repo and save a wallet to confirm it works.
	repo, err := NewWalletRepo(filePath)
	require.NoError(t, err)

	w1, err := wallet.NewWallet()
	require.NoError(t, err)
	require.NoError(t, repo.Save(w1))

	// Make the directory read-only so flush cannot write the temp file.
	require.NoError(t, os.Chmod(subDir, 0o555))
	t.Cleanup(func() {
		os.Chmod(subDir, 0o755) // Restore so t.TempDir cleanup succeeds.
	})

	// Create a new wallet and attempt to save -- should fail.
	w2, err := wallet.NewWallet()
	require.NoError(t, err)

	err = repo.Save(w2)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "write temp wallet file")
}
