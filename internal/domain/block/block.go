package block

import "time"

// Block is an entity (aggregate root) identified by its hash.
// All fields are unexported; access via getters, mutation via specific methods.
// Pointer receiver for entity semantics.
type Block struct {
	header       Header
	hash         Hash
	height       uint64
	message      string   // only used for genesis block
	transactions [][]byte // empty in Phase 1, typed transactions in Phase 2
}

// NewGenesisBlock creates the genesis block (height=0, zero prevHash).
// The block is created unmined -- use ProofOfWork.Mine() to find a valid nonce.
func NewGenesisBlock(message string, bits uint32) (*Block, error) {
	header := NewHeader(
		1,      // version
		Hash{}, // zero prevBlockHash
		Hash{}, // zero merkleRoot (Phase 1)
		time.Now().Unix(),
		bits,
	)

	return &Block{
		header:       header,
		height:       0,
		message:      message,
		transactions: make([][]byte, 0),
	}, nil
}

// NewBlock creates a new block with the given previous hash, height, and difficulty bits.
// The block is created unmined -- use ProofOfWork.Mine() to find a valid nonce.
func NewBlock(prevHash Hash, height uint64, bits uint32) (*Block, error) {
	header := NewHeader(
		1, // version
		prevHash,
		Hash{}, // zero merkleRoot (Phase 1)
		time.Now().Unix(),
		bits,
	)

	return &Block{
		header:       header,
		height:       height,
		transactions: make([][]byte, 0),
	}, nil
}

// ReconstructBlock creates a Block from stored data, bypassing mining.
// Used when loading blocks from persistence.
func ReconstructBlock(header Header, hash Hash, height uint64, message string, transactions [][]byte) *Block {
	return &Block{
		header:       header,
		hash:         hash,
		height:       height,
		message:      message,
		transactions: transactions,
	}
}

// Hash returns the block's hash.
func (b *Block) Hash() Hash { return b.hash }

// Height returns the block's height in the chain.
func (b *Block) Height() uint64 { return b.height }

// Header returns the block's header.
func (b *Block) Header() Header { return b.header }

// PrevBlockHash returns the previous block's hash from the header.
func (b *Block) PrevBlockHash() Hash { return b.header.prevBlockHash }

// Timestamp returns the block's timestamp as Unix seconds.
func (b *Block) Timestamp() int64 { return b.header.timestamp }

// Bits returns the block's difficulty target bits.
func (b *Block) Bits() uint32 { return b.header.bits }

// Message returns the block's embedded message (genesis block only).
func (b *Block) Message() string { return b.message }

// Transactions returns the block's transaction list.
func (b *Block) Transactions() [][]byte { return b.transactions }

// SetHash sets the block's hash. Used by PoW after mining.
func (b *Block) SetHash(hash Hash) { b.hash = hash }

// SetHeaderNonce sets the nonce on the block's header. Used by PoW during mining.
func (b *Block) SetHeaderNonce(nonce uint32) { b.header.SetNonce(nonce) }
