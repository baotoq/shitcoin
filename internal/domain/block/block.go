package block

import "time"

// Block is an entity (aggregate root) identified by its hash.
// All fields are unexported; access via getters, mutation via specific methods.
// Pointer receiver for entity semantics.
//
// Transactions are stored as []any to avoid circular imports between block and tx packages.
// The chain aggregate is responsible for type-safe access via type assertions.
type Block struct {
	header       Header
	hash         Hash
	height       uint64
	message      string // only used for genesis block
	transactions []any  // typed transactions stored as any to break import cycle
}

// NewGenesisBlock creates the genesis block (height=0, zero prevHash).
// The block is created unmined -- use ProofOfWork.Mine() to find a valid nonce.
// Accepts transactions as []any (typically []*tx.Transaction cast to any).
func NewGenesisBlock(message string, bits uint32, txs []any) (*Block, error) {
	header := NewHeader(
		1,      // version
		Hash{}, // zero prevBlockHash
		Hash{}, // zero merkleRoot (Phase 1)
		time.Now().Unix(),
		bits,
	)

	if txs == nil {
		txs = make([]any, 0)
	}

	return &Block{
		header:       header,
		height:       0,
		message:      message,
		transactions: txs,
	}, nil
}

// NewBlock creates a new block with the given previous hash, height, and difficulty bits.
// The block is created unmined -- use ProofOfWork.Mine() to find a valid nonce.
// Accepts transactions as []any (typically []*tx.Transaction cast to any).
func NewBlock(prevHash Hash, height uint64, bits uint32, txs []any) (*Block, error) {
	header := NewHeader(
		1, // version
		prevHash,
		Hash{}, // zero merkleRoot (Phase 1)
		time.Now().Unix(),
		bits,
	)

	if txs == nil {
		txs = make([]any, 0)
	}

	return &Block{
		header:       header,
		height:       height,
		transactions: txs,
	}, nil
}

// ReconstructBlock creates a Block from stored data, bypassing mining.
// Used when loading blocks from persistence.
func ReconstructBlock(header Header, hash Hash, height uint64, message string, transactions []any) *Block {
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

// RawTransactions returns the block's transactions as untyped slice.
// Use TypedTransactions() or type-assert individual elements to *tx.Transaction.
func (b *Block) RawTransactions() []any { return b.transactions }

// SetHash sets the block's hash. Used by PoW after mining.
func (b *Block) SetHash(hash Hash) { b.hash = hash }

// SetHeaderNonce sets the nonce on the block's header. Used by PoW during mining.
func (b *Block) SetHeaderNonce(nonce uint32) { b.header.SetNonce(nonce) }
