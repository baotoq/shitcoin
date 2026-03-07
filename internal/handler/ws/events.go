package ws

// WSMessage is a typed WebSocket message sent to clients.
type WSMessage struct {
	Type    string `json:"type"`
	Payload any    `json:"payload"`
}

// MiningProgressPayload is sent during active mining to show hash attempts.
type MiningProgressPayload struct {
	Nonce       uint32 `json:"nonce"`
	HashHex     string `json:"hash_hex"`
	TargetHex   string `json:"target_hex"`
	Difficulty  uint32 `json:"difficulty"`
	BlockHeight uint64 `json:"block_height"`
}

// MiningStartedPayload is sent when mining begins on a new block.
type MiningStartedPayload struct {
	BlockHeight uint64 `json:"block_height"`
	Difficulty  uint32 `json:"difficulty"`
}

// MiningStoppedPayload is sent when mining stops (block found or cancelled).
type MiningStoppedPayload struct {
	BlockHeight uint64 `json:"block_height"`
	BlockHash   string `json:"block_hash,omitempty"`
	Reason      string `json:"reason"` // "found", "cancelled", "error"
}

// PeerPayload is sent when a peer connects or disconnects.
type PeerPayload struct {
	Addr   string `json:"addr"`
	Height uint64 `json:"height"`
}

// MempoolChangedPayload is sent when the mempool transaction count changes.
type MempoolChangedPayload struct {
	Count int `json:"count"`
}
