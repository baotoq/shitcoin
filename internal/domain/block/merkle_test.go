package block

import "testing"

func TestMerkleRoot_Empty(t *testing.T) {
	root := ComputeMerkleRoot(nil)
	if root != (Hash{}) {
		t.Errorf("MerkleRoot of empty = %x; want zero hash", root)
	}
}

func TestMerkleRoot_Single(t *testing.T) {
	h := DoubleSHA256([]byte("tx1"))
	root := ComputeMerkleRoot([]Hash{h})

	// Bitcoin convention: single leaf is hashed with itself
	var combined []byte
	combined = append(combined, h.Bytes()...)
	combined = append(combined, h.Bytes()...)
	expected := DoubleSHA256(combined)

	if root != expected {
		t.Errorf("MerkleRoot of single = %x; want %x", root, expected)
	}
}

func TestMerkleRoot_Two(t *testing.T) {
	h1 := DoubleSHA256([]byte("tx1"))
	h2 := DoubleSHA256([]byte("tx2"))
	root := ComputeMerkleRoot([]Hash{h1, h2})

	var combined []byte
	combined = append(combined, h1.Bytes()...)
	combined = append(combined, h2.Bytes()...)
	expected := DoubleSHA256(combined)

	if root != expected {
		t.Errorf("MerkleRoot of two = %x; want %x", root, expected)
	}
}

func TestMerkleRoot_Odd(t *testing.T) {
	h1 := DoubleSHA256([]byte("tx1"))
	h2 := DoubleSHA256([]byte("tx2"))
	h3 := DoubleSHA256([]byte("tx3"))

	root := ComputeMerkleRoot([]Hash{h1, h2, h3})

	// Three hashes -> duplicate last to make four, then compute binary tree
	// Level 1: hash(h1+h2), hash(h3+h3)
	var c12, c33 []byte
	c12 = append(c12, h1.Bytes()...)
	c12 = append(c12, h2.Bytes()...)
	h12 := DoubleSHA256(c12)

	c33 = append(c33, h3.Bytes()...)
	c33 = append(c33, h3.Bytes()...)
	h33 := DoubleSHA256(c33)

	// Level 2: hash(h12+h33)
	var cRoot []byte
	cRoot = append(cRoot, h12.Bytes()...)
	cRoot = append(cRoot, h33.Bytes()...)
	expected := DoubleSHA256(cRoot)

	if root != expected {
		t.Errorf("MerkleRoot of odd(3) = %x; want %x", root, expected)
	}
}

func TestMerkleRoot_Even(t *testing.T) {
	h1 := DoubleSHA256([]byte("tx1"))
	h2 := DoubleSHA256([]byte("tx2"))
	h3 := DoubleSHA256([]byte("tx3"))
	h4 := DoubleSHA256([]byte("tx4"))

	root := ComputeMerkleRoot([]Hash{h1, h2, h3, h4})

	// Level 1: hash(h1+h2), hash(h3+h4)
	var c12, c34 []byte
	c12 = append(c12, h1.Bytes()...)
	c12 = append(c12, h2.Bytes()...)
	h12 := DoubleSHA256(c12)

	c34 = append(c34, h3.Bytes()...)
	c34 = append(c34, h4.Bytes()...)
	h34 := DoubleSHA256(c34)

	// Level 2: hash(h12+h34)
	var cRoot []byte
	cRoot = append(cRoot, h12.Bytes()...)
	cRoot = append(cRoot, h34.Bytes()...)
	expected := DoubleSHA256(cRoot)

	if root != expected {
		t.Errorf("MerkleRoot of even(4) = %x; want %x", root, expected)
	}
}

func TestMerkleRoot_Deterministic(t *testing.T) {
	hashes := []Hash{
		DoubleSHA256([]byte("a")),
		DoubleSHA256([]byte("b")),
		DoubleSHA256([]byte("c")),
	}

	root1 := ComputeMerkleRoot(hashes)
	root2 := ComputeMerkleRoot(hashes)

	if root1 != root2 {
		t.Errorf("MerkleRoot not deterministic: %x != %x", root1, root2)
	}
}
