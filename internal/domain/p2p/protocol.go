package p2p

import (
	"encoding/binary"
	"fmt"
	"io"
)

// Wire format: [4-byte big-endian length][1-byte command][JSON payload]
// Length includes the command byte + payload bytes.

// WriteMessage writes a length-prefixed message to w.
func WriteMessage(w io.Writer, msg Message) error {
	// Length = 1 byte command + payload length
	frameLen := uint32(1 + len(msg.Payload))

	// Write 4-byte big-endian length header
	var header [4]byte
	binary.BigEndian.PutUint32(header[:], frameLen)
	if _, err := w.Write(header[:]); err != nil {
		return fmt.Errorf("write length header: %w", err)
	}

	// Write command byte
	if _, err := w.Write([]byte{msg.Command}); err != nil {
		return fmt.Errorf("write command: %w", err)
	}

	// Write payload
	if _, err := w.Write(msg.Payload); err != nil {
		return fmt.Errorf("write payload: %w", err)
	}

	return nil
}

// ReadMessage reads a length-prefixed message from r.
func ReadMessage(r io.Reader) (Message, error) {
	// Read 4-byte length header
	var header [4]byte
	if _, err := io.ReadFull(r, header[:]); err != nil {
		return Message{}, fmt.Errorf("read length header: %w", err)
	}

	frameLen := binary.BigEndian.Uint32(header[:])

	// Validate size
	if frameLen > MaxMessageSize {
		return Message{}, ErrMessageTooLarge
	}

	if frameLen < 1 {
		return Message{}, fmt.Errorf("invalid frame length: %d", frameLen)
	}

	// Read frame (command + payload)
	frame := make([]byte, frameLen)
	if _, err := io.ReadFull(r, frame); err != nil {
		return Message{}, fmt.Errorf("read frame: %w", err)
	}

	return Message{
		Command: frame[0],
		Payload: frame[1:],
	}, nil
}
