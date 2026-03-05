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
	return TxInput{
		txID: txID,
		vout: vout,
	}
}

// TxID returns the previous transaction hash this input references.
func (i TxInput) TxID() block.Hash {
	return i.txID
}

// Vout returns the output index in the previous transaction.
func (i TxInput) Vout() uint32 {
	return i.vout
}

// Signature returns the ECDSA signature bytes.
func (i TxInput) Signature() []byte {
	return i.signature
}

// PubKey returns the compressed public key bytes.
func (i TxInput) PubKey() []byte {
	return i.pubKey
}

// SetSignature sets the ECDSA signature on this input (used during signing).
func (i *TxInput) SetSignature(sig []byte) {
	i.signature = sig
}

// SetPubKey sets the compressed public key on this input (used during signing).
func (i *TxInput) SetPubKey(pk []byte) {
	i.pubKey = pk
}
