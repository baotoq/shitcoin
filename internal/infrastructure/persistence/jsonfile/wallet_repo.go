package jsonfile

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/baotoq/shitcoin/internal/domain/wallet"
)

// Compile-time interface check.
var _ wallet.Repository = (*WalletRepo)(nil)

// walletEntry is the JSON storage model for a single wallet.
type walletEntry struct {
	Address       string `json:"address"`
	PrivateKeyHex string `json:"private_key_hex"`
}

// walletFileModel is the top-level JSON file structure.
type walletFileModel struct {
	Wallets []walletEntry `json:"wallets"`
}

// WalletRepo implements wallet.Repository using a JSON file for persistence.
// Wallets are loaded into memory on startup and written atomically on save.
type WalletRepo struct {
	filePath string
	wallets  map[string]*wallet.Wallet
}

// NewWalletRepo creates a new WalletRepo, loading existing wallets from the file if present.
func NewWalletRepo(filePath string) (*WalletRepo, error) {
	repo := &WalletRepo{
		filePath: filePath,
		wallets:  make(map[string]*wallet.Wallet),
	}

	// Load existing file if present.
	data, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return repo, nil
		}
		return nil, fmt.Errorf("read wallet file: %w", err)
	}

	var model walletFileModel
	if err := json.Unmarshal(data, &model); err != nil {
		return nil, fmt.Errorf("unmarshal wallet file: %w", err)
	}

	for _, entry := range model.Wallets {
		w, err := wallet.ReconstructWallet(entry.PrivateKeyHex)
		if err != nil {
			return nil, fmt.Errorf("reconstruct wallet %s: %w", entry.Address, err)
		}
		repo.wallets[w.Address()] = w
	}

	return repo, nil
}

// Save persists a wallet to the JSON file.
// Adds the wallet to the in-memory map and writes all wallets atomically.
func (r *WalletRepo) Save(w *wallet.Wallet) error {
	r.wallets[w.Address()] = w
	return r.flush()
}

// GetByAddress retrieves a wallet by its address.
// Returns wallet.ErrWalletNotFound if no wallet exists with the given address.
func (r *WalletRepo) GetByAddress(address string) (*wallet.Wallet, error) {
	w, ok := r.wallets[address]
	if !ok {
		return nil, wallet.ErrWalletNotFound
	}
	return w, nil
}

// ListAddresses returns all stored wallet addresses.
func (r *WalletRepo) ListAddresses() ([]string, error) {
	addresses := make([]string, 0, len(r.wallets))
	for addr := range r.wallets {
		addresses = append(addresses, addr)
	}
	return addresses, nil
}

// flush writes all in-memory wallets to the JSON file atomically.
// Uses write-to-temp-file + rename for crash safety.
func (r *WalletRepo) flush() error {
	model := walletFileModel{
		Wallets: make([]walletEntry, 0, len(r.wallets)),
	}

	for _, w := range r.wallets {
		model.Wallets = append(model.Wallets, walletEntry{
			Address:       w.Address(),
			PrivateKeyHex: w.PrivateKeyHex(),
		})
	}

	data, err := json.MarshalIndent(model, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal wallets: %w", err)
	}

	// Ensure parent directory exists.
	dir := filepath.Dir(r.filePath)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create wallet directory: %w", err)
	}

	// Write atomically: temp file + rename.
	tmpFile := r.filePath + ".tmp"
	if err := os.WriteFile(tmpFile, data, 0o644); err != nil {
		return fmt.Errorf("write temp wallet file: %w", err)
	}

	if err := os.Rename(tmpFile, r.filePath); err != nil {
		return fmt.Errorf("rename wallet file: %w", err)
	}

	return nil
}
