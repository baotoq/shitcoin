package bbolt

import (
	"github.com/baotoq/shitcoin/internal/domain/block"
)

// HeaderModel is a JSON-serializable storage model for block headers.
type HeaderModel struct {
	Version       uint32 `json:"version"`
	PrevBlockHash string `json:"prev_block_hash"`
	MerkleRoot    string `json:"merkle_root"`
	Timestamp     int64  `json:"timestamp"`
	Bits          uint32 `json:"bits"`
	Nonce         uint32 `json:"nonce"`
}

// BlockModel is a JSON-serializable storage model for blocks.
type BlockModel struct {
	Hash         string      `json:"hash"`
	Header       HeaderModel `json:"header"`
	Height       uint64      `json:"height"`
	Message      string      `json:"message,omitempty"`
	Transactions [][]byte    `json:"transactions"`
}

// BlockModelFromDomain converts a domain Block to a storage BlockModel.
func BlockModelFromDomain(b *block.Block) *BlockModel {
	panic("not implemented")
}

// ToDomain converts a storage BlockModel back to a domain Block.
func (bm *BlockModel) ToDomain() (*block.Block, error) {
	panic("not implemented")
}
