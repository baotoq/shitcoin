package tx

import "errors"

// Sentinel errors for transaction domain validation.
var (
	// ErrInsufficientFunds is returned when input values cannot cover the payment amount.
	ErrInsufficientFunds = errors.New("insufficient funds")

	// ErrInvalidSignature is returned when an ECDSA signature fails verification.
	ErrInvalidSignature = errors.New("invalid signature")

	// ErrNegativeValue is returned when a transaction output has a non-positive value.
	ErrNegativeValue = errors.New("output value must be positive")

	// ErrSumMismatch is returned when sum of outputs exceeds sum of inputs.
	ErrSumMismatch = errors.New("sum of outputs exceeds sum of inputs")

	// ErrInvalidCoinbase is returned when a coinbase transaction violates structural rules.
	ErrInvalidCoinbase = errors.New("invalid coinbase transaction")

	// ErrNoInputs is returned when a non-coinbase transaction has no inputs.
	ErrNoInputs = errors.New("transaction has no inputs")

	// ErrNoOutputs is returned when a transaction has no outputs.
	ErrNoOutputs = errors.New("transaction has no outputs")
)
