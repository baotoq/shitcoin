package jsonfile

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/baotoq/shitcoin/internal/domain/wallet"
)

func TestWalletRepo_SaveAndGet(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "wallets.json")

	repo, err := NewWalletRepo(filePath)
	if err != nil {
		t.Fatalf("NewWalletRepo failed: %v", err)
	}

	w, err := wallet.NewWallet()
	if err != nil {
		t.Fatalf("NewWallet failed: %v", err)
	}

	if err := repo.Save(w); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	got, err := repo.GetByAddress(w.Address())
	if err != nil {
		t.Fatalf("GetByAddress failed: %v", err)
	}

	if got.Address() != w.Address() {
		t.Errorf("address = %q; want %q", got.Address(), w.Address())
	}
	if got.PrivateKeyHex() != w.PrivateKeyHex() {
		t.Errorf("private key hex mismatch")
	}
}

func TestWalletRepo_ListAddresses(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "wallets.json")

	repo, err := NewWalletRepo(filePath)
	if err != nil {
		t.Fatalf("NewWalletRepo failed: %v", err)
	}

	w1, _ := wallet.NewWallet()
	w2, _ := wallet.NewWallet()
	w3, _ := wallet.NewWallet()

	_ = repo.Save(w1)
	_ = repo.Save(w2)
	_ = repo.Save(w3)

	addresses, err := repo.ListAddresses()
	if err != nil {
		t.Fatalf("ListAddresses failed: %v", err)
	}

	if len(addresses) != 3 {
		t.Errorf("got %d addresses; want 3", len(addresses))
	}

	// Check all addresses are present.
	addrMap := make(map[string]bool)
	for _, a := range addresses {
		addrMap[a] = true
	}
	for _, w := range []*wallet.Wallet{w1, w2, w3} {
		if !addrMap[w.Address()] {
			t.Errorf("address %q not found in list", w.Address())
		}
	}
}

func TestWalletRepo_GetByAddress_NotFound(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "wallets.json")

	repo, err := NewWalletRepo(filePath)
	if err != nil {
		t.Fatalf("NewWalletRepo failed: %v", err)
	}

	_, err = repo.GetByAddress("1NonExistentAddress")
	if err != wallet.ErrWalletNotFound {
		t.Errorf("err = %v; want ErrWalletNotFound", err)
	}
}

func TestWalletRepo_Persistence(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "wallets.json")

	// Create repo and save a wallet.
	repo1, err := NewWalletRepo(filePath)
	if err != nil {
		t.Fatalf("NewWalletRepo failed: %v", err)
	}

	w, _ := wallet.NewWallet()
	_ = repo1.Save(w)

	// Create a new repo from the same file (simulates close/reopen).
	repo2, err := NewWalletRepo(filePath)
	if err != nil {
		t.Fatalf("NewWalletRepo (reopen) failed: %v", err)
	}

	got, err := repo2.GetByAddress(w.Address())
	if err != nil {
		t.Fatalf("GetByAddress after reopen failed: %v", err)
	}
	if got.Address() != w.Address() {
		t.Errorf("address = %q; want %q", got.Address(), w.Address())
	}
	if got.PrivateKeyHex() != w.PrivateKeyHex() {
		t.Errorf("private key hex mismatch after reopen")
	}
}

func TestWalletRepo_FileFormatIsReadableJSON(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "wallets.json")

	repo, err := NewWalletRepo(filePath)
	if err != nil {
		t.Fatalf("NewWalletRepo failed: %v", err)
	}

	w, _ := wallet.NewWallet()
	_ = repo.Save(w)

	// Read the raw file and verify it's valid JSON.
	data, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("ReadFile failed: %v", err)
	}

	var parsed map[string]interface{}
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("file is not valid JSON: %v", err)
	}

	// Verify the structure has a wallets array.
	wallets, ok := parsed["wallets"]
	if !ok {
		t.Fatal("JSON missing 'wallets' key")
	}

	arr, ok := wallets.([]interface{})
	if !ok {
		t.Fatal("'wallets' is not an array")
	}
	if len(arr) != 1 {
		t.Errorf("wallets array length = %d; want 1", len(arr))
	}

	// Verify each entry has address and private_key_hex fields.
	entry, ok := arr[0].(map[string]interface{})
	if !ok {
		t.Fatal("wallet entry is not an object")
	}
	if _, ok := entry["address"]; !ok {
		t.Error("wallet entry missing 'address' field")
	}
	if _, ok := entry["private_key_hex"]; !ok {
		t.Error("wallet entry missing 'private_key_hex' field")
	}
}
