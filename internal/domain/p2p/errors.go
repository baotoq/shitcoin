package p2p

import "errors"

var (
	// ErrMessageTooLarge is returned when a message exceeds MaxMessageSize.
	ErrMessageTooLarge = errors.New("message exceeds maximum size")

	// ErrHandshakeFailed is returned when the version handshake fails.
	ErrHandshakeFailed = errors.New("version handshake failed")

	// ErrIncompatibleGenesis is returned when peers have different genesis hashes.
	ErrIncompatibleGenesis = errors.New("incompatible genesis hash")

	// ErrProtocolViolation is returned when an unexpected message is received.
	ErrProtocolViolation = errors.New("protocol violation")
)
