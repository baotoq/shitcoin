package wallet

import (
	"github.com/btcsuite/btcd/btcec/v2"
)

// Wallet is an entity representing an ECDSA key pair with a derived Bitcoin-style address.
// All fields are unexported; access via getters.
type Wallet struct {
	address    string
	privateKey *btcec.PrivateKey
	publicKey  *btcec.PublicKey
}

// NewWallet generates a new ECDSA key pair (secp256k1) and derives a P2PKH address.
func NewWallet() (*Wallet, error) {
	panic("not implemented")
}

// ReconstructWallet reconstructs a wallet from a hex-encoded private key string.
func ReconstructWallet(privKeyHex string) (*Wallet, error) {
	panic("not implemented")
}

// Address returns the wallet's Base58Check address.
func (w *Wallet) Address() string { return w.address }

// PrivateKey returns the wallet's ECDSA private key.
func (w *Wallet) PrivateKey() *btcec.PrivateKey { return w.privateKey }

// PublicKey returns the wallet's ECDSA public key.
func (w *Wallet) PublicKey() *btcec.PublicKey { return w.publicKey }

// PrivateKeyHex returns the private key as a hex-encoded string.
func (w *Wallet) PrivateKeyHex() string {
	panic("not implemented")
}
