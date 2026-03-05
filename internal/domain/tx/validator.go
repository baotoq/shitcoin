package tx

import "fmt"

// ValidateTransaction performs structural validation on a transaction.
// It checks that output values are positive and that the sum of outputs
// does not exceed the sum of input values (when provided).
func ValidateTransaction(tx *Transaction, inputValues []int64) error {
	if len(tx.inputs) == 0 {
		return ErrNoInputs
	}
	if len(tx.outputs) == 0 {
		return ErrNoOutputs
	}

	var outputSum int64
	for _, out := range tx.outputs {
		if out.value <= 0 {
			return fmt.Errorf("%w: got %d", ErrNegativeValue, out.value)
		}
		outputSum += out.value
	}

	if inputValues != nil {
		var inputSum int64
		for _, v := range inputValues {
			inputSum += v
		}
		if outputSum > inputSum {
			return fmt.Errorf("%w: outputs=%d inputs=%d", ErrSumMismatch, outputSum, inputSum)
		}
	}

	return nil
}

// ValidateCoinbase validates a coinbase transaction structure.
// It must have exactly one input (coinbase marker) and one output
// with the expected reward value.
func ValidateCoinbase(tx *Transaction, expectedReward int64) error {
	if !tx.IsCoinbase() {
		return fmt.Errorf("%w: not a coinbase transaction", ErrInvalidCoinbase)
	}
	if len(tx.outputs) != 1 {
		return fmt.Errorf("%w: expected 1 output, got %d", ErrInvalidCoinbase, len(tx.outputs))
	}
	if tx.outputs[0].value != expectedReward {
		return fmt.Errorf("%w: expected reward %d, got %d", ErrInvalidCoinbase, expectedReward, tx.outputs[0].value)
	}
	return nil
}

// CreateTransactionWithChange creates a transaction with automatic change output.
// If the sum of input values exceeds the payment amount, a change output is
// created to the change address for the difference.
func CreateTransactionWithChange(inputs []TxInput, inputValues []int64, toAddress string, amount int64, changeAddress string) (*Transaction, error) {
	var inputSum int64
	for _, v := range inputValues {
		inputSum += v
	}

	if inputSum < amount {
		return nil, ErrInsufficientFunds
	}

	outputs := []TxOutput{NewTxOutput(amount, toAddress)}

	change := inputSum - amount
	if change > 0 {
		outputs = append(outputs, NewTxOutput(change, changeAddress))
	}

	return NewTransaction(inputs, outputs), nil
}
