package wallet

import (
	"strings"
	"testing"
)

func TestPubKeyToAddress(t *testing.T) {
	t.Run("compressed public key produces address starting with 1", func(t *testing.T) {
		w, err := NewWallet()
		if err != nil {
			t.Fatalf("NewWallet failed: %v", err)
		}
		address := PubKeyToAddress(w.PublicKey())
		if !strings.HasPrefix(address, "1") {
			t.Errorf("address %q does not start with '1'", address)
		}
	})

	t.Run("same key always produces same address (deterministic)", func(t *testing.T) {
		w, err := NewWallet()
		if err != nil {
			t.Fatalf("NewWallet failed: %v", err)
		}
		addr1 := PubKeyToAddress(w.PublicKey())
		addr2 := PubKeyToAddress(w.PublicKey())
		if addr1 != addr2 {
			t.Errorf("non-deterministic: %q != %q", addr1, addr2)
		}
	})
}

func TestNewWallet(t *testing.T) {
	t.Run("generates unique key pair and derives address", func(t *testing.T) {
		w, err := NewWallet()
		if err != nil {
			t.Fatalf("NewWallet failed: %v", err)
		}
		if w.Address() == "" {
			t.Error("address is empty")
		}
		if w.PrivateKey() == nil {
			t.Error("private key is nil")
		}
		if w.PublicKey() == nil {
			t.Error("public key is nil")
		}
		if !strings.HasPrefix(w.Address(), "1") {
			t.Errorf("address %q does not start with '1'", w.Address())
		}
	})

	t.Run("two wallets produce different addresses", func(t *testing.T) {
		w1, err := NewWallet()
		if err != nil {
			t.Fatalf("NewWallet 1 failed: %v", err)
		}
		w2, err := NewWallet()
		if err != nil {
			t.Fatalf("NewWallet 2 failed: %v", err)
		}
		if w1.Address() == w2.Address() {
			t.Error("two wallets produced the same address")
		}
	})

	t.Run("wallet stores and returns address, public key, private key", func(t *testing.T) {
		w, err := NewWallet()
		if err != nil {
			t.Fatalf("NewWallet failed: %v", err)
		}

		// Address should match PubKeyToAddress derivation
		expectedAddr := PubKeyToAddress(w.PublicKey())
		if w.Address() != expectedAddr {
			t.Errorf("Address() = %q; want %q", w.Address(), expectedAddr)
		}

		// PrivateKeyHex should round-trip via ReconstructWallet
		hex := w.PrivateKeyHex()
		if hex == "" {
			t.Error("PrivateKeyHex is empty")
		}

		reconstructed, err := ReconstructWallet(hex)
		if err != nil {
			t.Fatalf("ReconstructWallet failed: %v", err)
		}
		if reconstructed.Address() != w.Address() {
			t.Errorf("reconstructed address %q != original %q", reconstructed.Address(), w.Address())
		}
	})
}

func TestReconstructWallet(t *testing.T) {
	t.Run("invalid hex returns error", func(t *testing.T) {
		_, err := ReconstructWallet("not-hex")
		if err == nil {
			t.Error("expected error for invalid hex, got nil")
		}
	})

	t.Run("round-trip private key hex", func(t *testing.T) {
		w, err := NewWallet()
		if err != nil {
			t.Fatalf("NewWallet failed: %v", err)
		}

		w2, err := ReconstructWallet(w.PrivateKeyHex())
		if err != nil {
			t.Fatalf("ReconstructWallet failed: %v", err)
		}

		if w2.Address() != w.Address() {
			t.Errorf("addresses differ: %q vs %q", w2.Address(), w.Address())
		}
	})
}
