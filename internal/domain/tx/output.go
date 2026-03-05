package tx

// TxOutput represents a transaction output with a satoshi value and recipient address.
type TxOutput struct {
	value   int64
	address string
}

// NewTxOutput creates a new transaction output.
func NewTxOutput(value int64, address string) TxOutput {
	return TxOutput{
		value:   value,
		address: address,
	}
}

// Value returns the output value in satoshis.
func (o TxOutput) Value() int64 {
	return o.value
}

// Address returns the Base58Check recipient address.
func (o TxOutput) Address() string {
	return o.address
}
