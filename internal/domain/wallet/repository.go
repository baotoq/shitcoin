package wallet

// Repository defines the persistence interface for wallets.
// Interface in domain, implementation in infrastructure.
type Repository interface {
	// Save persists a wallet.
	Save(wallet *Wallet) error

	// GetByAddress retrieves a wallet by its Base58Check address.
	// Returns ErrWalletNotFound if no wallet exists with the given address.
	GetByAddress(address string) (*Wallet, error)

	// ListAddresses returns all stored wallet addresses.
	ListAddresses() ([]string, error)
}
