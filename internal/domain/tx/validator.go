package tx

// ValidateTransaction performs structural validation on a transaction.
// It checks that output values are positive and that the sum of outputs
// does not exceed the sum of input values (when provided).
func ValidateTransaction(tx *Transaction, inputValues []int64) error {
	panic("not implemented")
}

// ValidateCoinbase validates a coinbase transaction structure.
// It must have exactly one input (coinbase marker) and one output
// with the expected reward value.
func ValidateCoinbase(tx *Transaction, expectedReward int64) error {
	panic("not implemented")
}

// CreateTransactionWithChange creates a transaction with automatic change output.
// If the sum of input values exceeds the payment amount, a change output is
// created to the change address for the difference.
func CreateTransactionWithChange(inputs []TxInput, inputValues []int64, toAddress string, amount int64, changeAddress string) (*Transaction, error) {
	panic("not implemented")
}
