package wallet

import (
	"encoding/hex"
	"fmt"

	"github.com/btcsuite/btcd/btcec/v2"
)

// Wallet is an entity representing an ECDSA key pair with a derived Bitcoin-style address.
// All fields are unexported; access via getters. Pointer receiver for entity semantics.
type Wallet struct {
	address    string
	privateKey *btcec.PrivateKey
	publicKey  *btcec.PublicKey
}

// NewWallet generates a new ECDSA key pair (secp256k1) and derives a P2PKH address.
func NewWallet() (*Wallet, error) {
	privKey, err := btcec.NewPrivateKey()
	if err != nil {
		return nil, fmt.Errorf("generate private key: %w", err)
	}

	pubKey := privKey.PubKey()
	address := PubKeyToAddress(pubKey)

	return &Wallet{
		address:    address,
		privateKey: privKey,
		publicKey:  pubKey,
	}, nil
}

// ReconstructWallet reconstructs a wallet from a hex-encoded private key string.
// Used when loading wallets from persistence.
func ReconstructWallet(privKeyHex string) (*Wallet, error) {
	privKeyBytes, err := hex.DecodeString(privKeyHex)
	if err != nil {
		return nil, fmt.Errorf("decode private key hex: %w", err)
	}

	privKey, pubKey := btcec.PrivKeyFromBytes(privKeyBytes)
	address := PubKeyToAddress(pubKey)

	return &Wallet{
		address:    address,
		privateKey: privKey,
		publicKey:  pubKey,
	}, nil
}

// Address returns the wallet's Base58Check address.
func (w *Wallet) Address() string { return w.address }

// PrivateKey returns the wallet's ECDSA private key.
func (w *Wallet) PrivateKey() *btcec.PrivateKey { return w.privateKey }

// PublicKey returns the wallet's ECDSA public key.
func (w *Wallet) PublicKey() *btcec.PublicKey { return w.publicKey }

// PrivateKeyHex returns the private key as a hex-encoded string.
func (w *Wallet) PrivateKeyHex() string {
	return hex.EncodeToString(w.privateKey.Serialize())
}
