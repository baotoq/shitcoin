package testutil

import (
	"sync"

	"github.com/baotoq/shitcoin/internal/domain/wallet"
)

// Compile-time interface check.
var _ wallet.Repository = (*MockWalletRepo)(nil)

// MockWalletRepo is an in-memory implementation of wallet.Repository for testing.
type MockWalletRepo struct {
	mu      sync.Mutex
	Wallets map[string]*wallet.Wallet
}

// NewMockWalletRepo creates a new MockWalletRepo with initialized map.
func NewMockWalletRepo() *MockWalletRepo {
	return &MockWalletRepo{
		Wallets: make(map[string]*wallet.Wallet),
	}
}

func (m *MockWalletRepo) Save(w *wallet.Wallet) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Wallets[w.Address()] = w
	return nil
}

func (m *MockWalletRepo) GetByAddress(address string) (*wallet.Wallet, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	w, ok := m.Wallets[address]
	if !ok {
		return nil, wallet.ErrWalletNotFound
	}
	return w, nil
}

func (m *MockWalletRepo) ListAddresses() ([]string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	addrs := make([]string, 0, len(m.Wallets))
	for addr := range m.Wallets {
		addrs = append(addrs, addr)
	}
	return addrs, nil
}
