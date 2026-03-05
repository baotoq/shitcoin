package p2p

import "encoding/json"

// MaxMessageSize is the maximum allowed message size (10MB).
const MaxMessageSize = 10 * 1024 * 1024

// ProtocolVersion is the current protocol version.
const ProtocolVersion uint32 = 1

// Command byte constants for P2P message types.
const (
	CmdVersion   byte = 0x01
	CmdVerack    byte = 0x02
	CmdGetBlocks byte = 0x03
	CmdInv       byte = 0x04
	CmdGetData   byte = 0x05
	CmdBlock     byte = 0x06
	CmdTx        byte = 0x07
)

// Message represents a P2P protocol message with a command type and payload.
type Message struct {
	Command byte
	Payload []byte
}

// NewMessage creates a new Message by JSON-marshaling the given payload.
func NewMessage(cmd byte, payload any) (Message, error) {
	data, err := json.Marshal(payload)
	if err != nil {
		return Message{}, err
	}
	return Message{Command: cmd, Payload: data}, nil
}

// VersionPayload is the payload for version handshake messages.
type VersionPayload struct {
	Version    uint32 `json:"version"`
	Height     uint64 `json:"height"`
	GenesisHash string `json:"genesis_hash"`
	ListenPort int    `json:"listen_port"`
}

// InvPayload is the payload for inventory announcement messages.
type InvPayload struct {
	Type   string   `json:"type"`
	Hashes []string `json:"hashes"`
}

// GetBlocksPayload is the payload for requesting a range of blocks.
type GetBlocksPayload struct {
	StartHeight uint64 `json:"start_height"`
	EndHeight   uint64 `json:"end_height"`
}
