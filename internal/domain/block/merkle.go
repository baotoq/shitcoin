package block

// ComputeMerkleRoot computes the Merkle root hash from a list of transaction hashes.
// Uses Bitcoin's binary Merkle tree construction with DoubleSHA256 at each level.
// Empty input returns a zero Hash. Odd-count levels duplicate the last hash.
func ComputeMerkleRoot(txHashes []Hash) Hash {
	if len(txHashes) == 0 {
		return Hash{}
	}

	// Copy to avoid mutating the caller's slice
	level := make([]Hash, len(txHashes))
	copy(level, txHashes)

	// Bitcoin convention: always run at least one round of hashing.
	// A single leaf is hashed with itself (duplicated).
	for {
		// If odd count (including single element), duplicate the last hash
		if len(level)%2 != 0 {
			level = append(level, level[len(level)-1])
		}

		nextLevel := make([]Hash, 0, len(level)/2)
		for i := 0; i < len(level); i += 2 {
			var combined []byte
			combined = append(combined, level[i].Bytes()...)
			combined = append(combined, level[i+1].Bytes()...)
			nextLevel = append(nextLevel, DoubleSHA256(combined))
		}
		level = nextLevel

		if len(level) == 1 {
			break
		}
	}

	return level[0]
}
