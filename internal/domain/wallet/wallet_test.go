package wallet

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPubKeyToAddress(t *testing.T) {
	t.Run("compressed public key produces address starting with 1", func(t *testing.T) {
		w, err := NewWallet()
		require.NoError(t, err)
		address := PubKeyToAddress(w.PublicKey())
		assert.True(t, strings.HasPrefix(address, "1"), "address %q does not start with '1'", address)
	})

	t.Run("same key always produces same address (deterministic)", func(t *testing.T) {
		w, err := NewWallet()
		require.NoError(t, err)
		addr1 := PubKeyToAddress(w.PublicKey())
		addr2 := PubKeyToAddress(w.PublicKey())
		assert.Equal(t, addr1, addr2)
	})
}

func TestNewWallet(t *testing.T) {
	t.Run("generates unique key pair and derives address", func(t *testing.T) {
		w, err := NewWallet()
		require.NoError(t, err)
		assert.NotEmpty(t, w.Address())
		assert.NotNil(t, w.PrivateKey())
		assert.NotNil(t, w.PublicKey())
		assert.True(t, strings.HasPrefix(w.Address(), "1"), "address %q does not start with '1'", w.Address())
	})

	t.Run("two wallets produce different addresses", func(t *testing.T) {
		w1, err := NewWallet()
		require.NoError(t, err)
		w2, err := NewWallet()
		require.NoError(t, err)
		assert.NotEqual(t, w1.Address(), w2.Address())
	})

	t.Run("wallet stores and returns address, public key, private key", func(t *testing.T) {
		w, err := NewWallet()
		require.NoError(t, err)

		// Address should match PubKeyToAddress derivation
		expectedAddr := PubKeyToAddress(w.PublicKey())
		assert.Equal(t, expectedAddr, w.Address())

		// PrivateKeyHex should round-trip via ReconstructWallet
		hex := w.PrivateKeyHex()
		assert.NotEmpty(t, hex)

		reconstructed, err := ReconstructWallet(hex)
		require.NoError(t, err)
		assert.Equal(t, w.Address(), reconstructed.Address())
	})
}

func TestReconstructWallet(t *testing.T) {
	t.Run("invalid hex returns error", func(t *testing.T) {
		_, err := ReconstructWallet("not-hex")
		require.Error(t, err)
	})

	t.Run("round-trip private key hex", func(t *testing.T) {
		w, err := NewWallet()
		require.NoError(t, err)

		w2, err := ReconstructWallet(w.PrivateKeyHex())
		require.NoError(t, err)

		assert.Equal(t, w.Address(), w2.Address())
	})
}
