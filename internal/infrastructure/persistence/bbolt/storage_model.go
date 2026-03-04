package bbolt

import (
	"fmt"

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
	h := b.Header()
	return &BlockModel{
		Hash: b.Hash().String(),
		Header: HeaderModel{
			Version:       h.Version(),
			PrevBlockHash: h.PrevBlockHash().String(),
			MerkleRoot:    h.MerkleRoot().String(),
			Timestamp:     h.Timestamp(),
			Bits:          h.Bits(),
			Nonce:         h.Nonce(),
		},
		Height:       b.Height(),
		Message:      b.Message(),
		Transactions: b.Transactions(),
	}
}

// ToDomain converts a storage BlockModel back to a domain Block using ReconstructBlock.
func (bm *BlockModel) ToDomain() (*block.Block, error) {
	hash, err := block.HashFromHex(bm.Hash)
	if err != nil {
		return nil, fmt.Errorf("invalid block hash: %w", err)
	}

	prevHash, err := block.HashFromHex(bm.Header.PrevBlockHash)
	if err != nil {
		return nil, fmt.Errorf("invalid prev block hash: %w", err)
	}

	merkleRoot, err := block.HashFromHex(bm.Header.MerkleRoot)
	if err != nil {
		return nil, fmt.Errorf("invalid merkle root: %w", err)
	}

	header := block.NewHeader(
		bm.Header.Version,
		prevHash,
		merkleRoot,
		bm.Header.Timestamp,
		bm.Header.Bits,
	)
	header.SetNonce(bm.Header.Nonce)

	txs := bm.Transactions
	if txs == nil {
		txs = make([][]byte, 0)
	}

	return block.ReconstructBlock(header, hash, bm.Height, bm.Message, txs), nil
}
