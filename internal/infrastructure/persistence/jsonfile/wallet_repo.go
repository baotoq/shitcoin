package jsonfile

import (
	"github.com/baotoq/shitcoin/internal/domain/wallet"
)

// Compile-time interface check.
var _ wallet.Repository = (*WalletRepo)(nil)

// WalletRepo implements wallet.Repository using a JSON file for persistence.
type WalletRepo struct {
	filePath string
	wallets  map[string]*wallet.Wallet
}

// NewWalletRepo creates a new WalletRepo, loading existing wallets from the file if present.
func NewWalletRepo(filePath string) (*WalletRepo, error) {
	panic("not implemented")
}

// Save persists a wallet to the JSON file.
func (r *WalletRepo) Save(w *wallet.Wallet) error {
	panic("not implemented")
}

// GetByAddress retrieves a wallet by its address.
func (r *WalletRepo) GetByAddress(address string) (*wallet.Wallet, error) {
	panic("not implemented")
}

// ListAddresses returns all stored wallet addresses.
func (r *WalletRepo) ListAddresses() ([]string, error) {
	panic("not implemented")
}
