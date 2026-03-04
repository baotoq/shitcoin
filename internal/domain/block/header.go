package block

import "encoding/json"

// Header is a value object representing a block header.
// All fields are unexported; access via getters.
// Timestamp is stored as int64 Unix seconds to avoid precision issues (Pitfall #2).
type Header struct {
	version       uint32
	prevBlockHash Hash
	merkleRoot    Hash
	timestamp     int64 // Unix seconds
	bits          uint32
	nonce         uint32
}

// hashableHeader is an exported struct used for deterministic JSON serialization
// during hashing. Field order is declaration order, which is deterministic in
// Go's encoding/json. Using JSON for debuggability per user decision.
type hashableHeader struct {
	Version       uint32 `json:"version"`
	PrevBlockHash string `json:"prev_block_hash"`
	MerkleRoot    string `json:"merkle_root"`
	Timestamp     int64  `json:"timestamp"`
	Bits          uint32 `json:"bits"`
	Nonce         uint32 `json:"nonce"`
}

// NewHeader creates a new Header value object.
func NewHeader(version uint32, prevBlockHash Hash, merkleRoot Hash, timestamp int64, bits uint32) Header {
	return Header{
		version:       version,
		prevBlockHash: prevBlockHash,
		merkleRoot:    merkleRoot,
		timestamp:     timestamp,
		bits:          bits,
		nonce:         0,
	}
}

// Version returns the block version.
func (h Header) Version() uint32 { return h.version }

// PrevBlockHash returns the previous block's hash.
func (h Header) PrevBlockHash() Hash { return h.prevBlockHash }

// MerkleRoot returns the Merkle root hash.
func (h Header) MerkleRoot() Hash { return h.merkleRoot }

// Timestamp returns the block timestamp as Unix seconds.
func (h Header) Timestamp() int64 { return h.timestamp }

// Bits returns the compact difficulty target.
func (h Header) Bits() uint32 { return h.bits }

// Nonce returns the mining nonce.
func (h Header) Nonce() uint32 { return h.nonce }

// SetNonce sets the nonce value. Used by the PoW mining loop.
func (h *Header) SetNonce(nonce uint32) {
	h.nonce = nonce
}

// HashPayload returns the deterministic byte representation of the header
// for hashing. Uses JSON serialization with a hashableHeader struct for
// educational debuggability.
func (h Header) HashPayload() []byte {
	hh := hashableHeader{
		Version:       h.version,
		PrevBlockHash: h.prevBlockHash.String(),
		MerkleRoot:    h.merkleRoot.String(),
		Timestamp:     h.timestamp,
		Bits:          h.bits,
		Nonce:         h.nonce,
	}
	data, _ := json.Marshal(hh)
	return data
}

// Hash computes the DoubleSHA256 of the header's hash payload.
func (h Header) Hash() Hash {
	return DoubleSHA256(h.HashPayload())
}
