package utxo

import (
	"fmt"

	"github.com/baotoq/shitcoin/internal/domain/block"
)

// UTXO is a value object representing an unspent transaction output.
// Immutable once created; value semantics with value receivers.
type UTXO struct {
	txID    block.Hash
	vout    uint32
	value   int64
	address string
}

// NewUTXO creates a new UTXO value object.
func NewUTXO(txID block.Hash, vout uint32, value int64, address string) UTXO {
	return UTXO{
		txID:    txID,
		vout:    vout,
		value:   value,
		address: address,
	}
}

// TxID returns the transaction hash that created this output.
func (u UTXO) TxID() block.Hash { return u.txID }

// Vout returns the output index within the transaction.
func (u UTXO) Vout() uint32 { return u.vout }

// Value returns the output value in satoshis.
func (u UTXO) Value() int64 { return u.value }

// Address returns the recipient address.
func (u UTXO) Address() string { return u.address }

// Key returns a string key "txid_hex:vout" for map lookups.
func (u UTXO) Key() string {
	return fmt.Sprintf("%s:%d", u.txID.String(), u.vout)
}
