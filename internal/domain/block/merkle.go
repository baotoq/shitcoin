package block

// ComputeMerkleRoot computes the Merkle root hash from a list of transaction hashes.
// Uses Bitcoin's binary Merkle tree construction with DoubleSHA256 at each level.
func ComputeMerkleRoot(txHashes []Hash) Hash {
	panic("not implemented")
}
