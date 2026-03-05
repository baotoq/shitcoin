package tx

import "github.com/baotoq/shitcoin/internal/domain/block"

// TxInput represents a transaction input that references a specific unspent output.
// It stores the previous transaction ID and output index (vout), along with
// signature and public key bytes that are set during signing.
type TxInput struct {
	txID      block.Hash
	vout      uint32
	signature []byte
	pubKey    []byte
}

// NewTxInput creates an unsigned transaction input referencing a previous output.
func NewTxInput(txID block.Hash, vout uint32) TxInput {
	panic("not implemented")
}

// TxID returns the previous transaction hash this input references.
func (i TxInput) TxID() block.Hash {
	panic("not implemented")
}

// Vout returns the output index in the previous transaction.
func (i TxInput) Vout() uint32 {
	panic("not implemented")
}

// Signature returns the ECDSA signature bytes.
func (i TxInput) Signature() []byte {
	panic("not implemented")
}

// PubKey returns the compressed public key bytes.
func (i TxInput) PubKey() []byte {
	panic("not implemented")
}

// SetSignature sets the ECDSA signature on this input (used during signing).
func (i *TxInput) SetSignature(sig []byte) {
	panic("not implemented")
}

// SetPubKey sets the compressed public key on this input (used during signing).
func (i *TxInput) SetPubKey(pk []byte) {
	panic("not implemented")
}
