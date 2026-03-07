package api

import (
	"errors"

	"github.com/baotoq/shitcoin/internal/infrastructure/persistence/bbolt"
)

// ErrBlockNotFound is returned when a block is not found.
var ErrBlockNotFound = errors.New("block not found")

// StatusResponse is the response for GET /api/status.
type StatusResponse struct {
	ChainHeight     uint64 `json:"chain_height"`
	LatestBlockHash string `json:"latest_block_hash"`
	MempoolSize     int    `json:"mempool_size"`
	PeerCount       int    `json:"peer_count"`
	IsMining        bool   `json:"is_mining"`
}

// BlockListResponse is the response for GET /api/blocks.
type BlockListResponse struct {
	Blocks []bbolt.BlockModel `json:"blocks"`
	Total  uint64             `json:"total"`
	Page   int                `json:"page"`
	Limit  int                `json:"limit"`
}

// AddressResponse is the response for GET /api/address/:addr.
type AddressResponse struct {
	Address string           `json:"address"`
	Balance int64            `json:"balance"`
	UTXOs   []bbolt.UTXOModel `json:"utxos"`
}

// SearchResult is the response for GET /api/search.
type SearchResult struct {
	Type        string  `json:"type"` // "block", "tx", or "address"
	BlockHeight *uint64 `json:"block_height,omitempty"`
	BlockHash   *string `json:"block_hash,omitempty"`
	TxHash      *string `json:"tx_hash,omitempty"`
	Address     *string `json:"address,omitempty"`
}

// TxResponse is the response for GET /api/tx/:hash.
type TxResponse struct {
	Tx          bbolt.TxModel `json:"tx"`
	BlockHeight uint64        `json:"block_height"`
	BlockHash   string        `json:"block_hash"`
}

// ErrorResponse is a standard error response.
type ErrorResponse struct {
	Error string `json:"error"`
}
