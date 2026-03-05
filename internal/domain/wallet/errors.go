package wallet

import "errors"

var (
	// ErrWalletNotFound is returned when a wallet is not found by address.
	ErrWalletNotFound = errors.New("wallet not found")

	// ErrInvalidAddress is returned when an address fails validation.
	ErrInvalidAddress = errors.New("invalid address")

	// ErrInvalidChecksum is returned when a Base58Check checksum does not match.
	ErrInvalidChecksum = errors.New("invalid checksum")
)
