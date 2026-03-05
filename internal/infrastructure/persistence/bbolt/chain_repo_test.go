package bbolt

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/baotoq/shitcoin/internal/domain/block"
	"github.com/baotoq/shitcoin/internal/domain/chain"
	bolt "go.etcd.io/bbolt"
)

// openTestDB opens a bbolt database in a temporary directory.
func openTestDB(t *testing.T) *bolt.DB {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "test.db")
	db, err := bolt.Open(dbPath, 0600, nil)
	if err != nil {
		t.Fatalf("failed to open bbolt: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

// createTestBlock creates a mined block at the given height for testing.
func createTestBlock(t *testing.T, prevHash block.Hash, height uint64, bits uint32) *block.Block {
	t.Helper()
	var b *block.Block
	var err error
	if height == 0 {
		b, err = block.NewGenesisBlock("test genesis", bits, nil, block.Hash{})
	} else {
		b, err = block.NewBlock(prevHash, height, bits, nil, block.Hash{})
	}
	if err != nil {
		t.Fatalf("failed to create block: %v", err)
	}
	pow := &block.ProofOfWork{}
	if err := pow.Mine(b); err != nil {
		t.Fatalf("failed to mine block: %v", err)
	}
	return b
}

func TestSaveAndGetBlock(t *testing.T) {
	db := openTestDB(t)
	repo, err := NewBboltRepository(db)
	if err != nil {
		t.Fatalf("NewBboltRepository: %v", err)
	}

	ctx := context.Background()
	b := createTestBlock(t, block.Hash{}, 0, 8)

	// Save block
	if err := repo.SaveBlock(ctx, b); err != nil {
		t.Fatalf("SaveBlock: %v", err)
	}

	// Retrieve by hash
	got, err := repo.GetBlock(ctx, b.Hash())
	if err != nil {
		t.Fatalf("GetBlock: %v", err)
	}

	// Verify all fields match
	if got.Hash() != b.Hash() {
		t.Errorf("Hash mismatch: got %s, want %s", got.Hash(), b.Hash())
	}
	if got.Height() != b.Height() {
		t.Errorf("Height mismatch: got %d, want %d", got.Height(), b.Height())
	}
	if got.Bits() != b.Bits() {
		t.Errorf("Bits mismatch: got %d, want %d", got.Bits(), b.Bits())
	}
	if got.Timestamp() != b.Timestamp() {
		t.Errorf("Timestamp mismatch: got %d, want %d", got.Timestamp(), b.Timestamp())
	}
	if got.Message() != b.Message() {
		t.Errorf("Message mismatch: got %q, want %q", got.Message(), b.Message())
	}
	if got.Header().Nonce() != b.Header().Nonce() {
		t.Errorf("Nonce mismatch: got %d, want %d", got.Header().Nonce(), b.Header().Nonce())
	}
}

func TestGetBlockByHeight(t *testing.T) {
	db := openTestDB(t)
	repo, err := NewBboltRepository(db)
	if err != nil {
		t.Fatalf("NewBboltRepository: %v", err)
	}

	ctx := context.Background()
	pow := &block.ProofOfWork{}

	// Create and save 3 blocks
	genesis, _ := block.NewGenesisBlock("test", 8, nil, block.Hash{})
	pow.Mine(genesis)
	repo.SaveBlock(ctx, genesis)

	block1, _ := block.NewBlock(genesis.Hash(), 1, 8, nil, block.Hash{})
	pow.Mine(block1)
	repo.SaveBlock(ctx, block1)

	block2, _ := block.NewBlock(block1.Hash(), 2, 8, nil, block.Hash{})
	pow.Mine(block2)
	repo.SaveBlock(ctx, block2)

	// Retrieve each by height
	tests := []struct {
		height   uint64
		wantHash block.Hash
	}{
		{0, genesis.Hash()},
		{1, block1.Hash()},
		{2, block2.Hash()},
	}

	for _, tt := range tests {
		got, err := repo.GetBlockByHeight(ctx, tt.height)
		if err != nil {
			t.Fatalf("GetBlockByHeight(%d): %v", tt.height, err)
		}
		if got.Hash() != tt.wantHash {
			t.Errorf("GetBlockByHeight(%d): hash = %s, want %s",
				tt.height, got.Hash(), tt.wantHash)
		}
	}
}

func TestGetLatestBlock(t *testing.T) {
	db := openTestDB(t)
	repo, err := NewBboltRepository(db)
	if err != nil {
		t.Fatalf("NewBboltRepository: %v", err)
	}

	ctx := context.Background()
	pow := &block.ProofOfWork{}

	// Create and save 3 blocks
	genesis, _ := block.NewGenesisBlock("test", 8, nil, block.Hash{})
	pow.Mine(genesis)
	repo.SaveBlock(ctx, genesis)

	block1, _ := block.NewBlock(genesis.Hash(), 1, 8, nil, block.Hash{})
	pow.Mine(block1)
	repo.SaveBlock(ctx, block1)

	block2, _ := block.NewBlock(block1.Hash(), 2, 8, nil, block.Hash{})
	pow.Mine(block2)
	repo.SaveBlock(ctx, block2)

	// Latest should be block2
	latest, err := repo.GetLatestBlock(ctx)
	if err != nil {
		t.Fatalf("GetLatestBlock: %v", err)
	}
	if latest.Hash() != block2.Hash() {
		t.Errorf("GetLatestBlock: hash = %s, want %s", latest.Hash(), block2.Hash())
	}
	if latest.Height() != 2 {
		t.Errorf("GetLatestBlock: height = %d, want 2", latest.Height())
	}
}

func TestGetChainHeight(t *testing.T) {
	db := openTestDB(t)
	repo, err := NewBboltRepository(db)
	if err != nil {
		t.Fatalf("NewBboltRepository: %v", err)
	}

	ctx := context.Background()
	pow := &block.ProofOfWork{}

	// Empty chain
	height, err := repo.GetChainHeight(ctx)
	if err != nil {
		t.Fatalf("GetChainHeight (empty): %v", err)
	}
	if height != 0 {
		t.Errorf("GetChainHeight (empty) = %d, want 0", height)
	}

	// After 3 blocks (heights 0, 1, 2)
	genesis, _ := block.NewGenesisBlock("test", 8, nil, block.Hash{})
	pow.Mine(genesis)
	repo.SaveBlock(ctx, genesis)

	block1, _ := block.NewBlock(genesis.Hash(), 1, 8, nil, block.Hash{})
	pow.Mine(block1)
	repo.SaveBlock(ctx, block1)

	block2, _ := block.NewBlock(block1.Hash(), 2, 8, nil, block.Hash{})
	pow.Mine(block2)
	repo.SaveBlock(ctx, block2)

	height, err = repo.GetChainHeight(ctx)
	if err != nil {
		t.Fatalf("GetChainHeight: %v", err)
	}
	if height != 2 {
		t.Errorf("GetChainHeight = %d, want 2", height)
	}
}

func TestChainPersistence(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "persist.db")

	// Phase 1: open DB, save blocks, close
	db1, err := bolt.Open(dbPath, 0600, nil)
	if err != nil {
		t.Fatalf("open db1: %v", err)
	}

	repo1, err := NewBboltRepository(db1)
	if err != nil {
		t.Fatalf("NewBboltRepository (1): %v", err)
	}

	ctx := context.Background()
	pow := &block.ProofOfWork{}

	genesis, _ := block.NewGenesisBlock("persist test", 8, nil, block.Hash{})
	pow.Mine(genesis)
	repo1.SaveBlock(ctx, genesis)

	block1, _ := block.NewBlock(genesis.Hash(), 1, 8, nil, block.Hash{})
	pow.Mine(block1)
	repo1.SaveBlock(ctx, block1)

	db1.Close()

	// Phase 2: reopen DB, verify blocks still there
	db2, err := bolt.Open(dbPath, 0600, nil)
	if err != nil {
		t.Fatalf("open db2: %v", err)
	}
	defer db2.Close()

	repo2, err := NewBboltRepository(db2)
	if err != nil {
		t.Fatalf("NewBboltRepository (2): %v", err)
	}

	// Retrieve genesis
	gotGenesis, err := repo2.GetBlockByHeight(ctx, 0)
	if err != nil {
		t.Fatalf("GetBlockByHeight(0) after reopen: %v", err)
	}
	if gotGenesis.Hash() != genesis.Hash() {
		t.Errorf("genesis hash after reopen: got %s, want %s", gotGenesis.Hash(), genesis.Hash())
	}
	if gotGenesis.Message() != "persist test" {
		t.Errorf("genesis message after reopen: got %q, want %q", gotGenesis.Message(), "persist test")
	}

	// Retrieve block1
	gotBlock1, err := repo2.GetBlockByHeight(ctx, 1)
	if err != nil {
		t.Fatalf("GetBlockByHeight(1) after reopen: %v", err)
	}
	if gotBlock1.Hash() != block1.Hash() {
		t.Errorf("block1 hash after reopen: got %s, want %s", gotBlock1.Hash(), block1.Hash())
	}

	// Verify latest is still correct
	latest, err := repo2.GetLatestBlock(ctx)
	if err != nil {
		t.Fatalf("GetLatestBlock after reopen: %v", err)
	}
	if latest.Hash() != block1.Hash() {
		t.Errorf("latest after reopen: got %s, want %s", latest.Hash(), block1.Hash())
	}

	// Verify height
	height, err := repo2.GetChainHeight(ctx)
	if err != nil {
		t.Fatalf("GetChainHeight after reopen: %v", err)
	}
	if height != 1 {
		t.Errorf("height after reopen: got %d, want 1", height)
	}
}

func TestGetBlockNotFound(t *testing.T) {
	db := openTestDB(t)
	repo, err := NewBboltRepository(db)
	if err != nil {
		t.Fatalf("NewBboltRepository: %v", err)
	}

	ctx := context.Background()
	fakeHash := block.DoubleSHA256([]byte("nonexistent"))

	_, err = repo.GetBlock(ctx, fakeHash)
	if err == nil {
		t.Fatal("expected error for non-existent block, got nil")
	}
	if err != chain.ErrBlockNotFound {
		t.Errorf("expected ErrBlockNotFound, got: %v", err)
	}
}

func TestEmptyChainLatestBlock(t *testing.T) {
	db := openTestDB(t)
	repo, err := NewBboltRepository(db)
	if err != nil {
		t.Fatalf("NewBboltRepository: %v", err)
	}

	ctx := context.Background()
	_, err = repo.GetLatestBlock(ctx)
	if err == nil {
		t.Fatal("expected error for empty chain, got nil")
	}
	if err != chain.ErrChainEmpty {
		t.Errorf("expected ErrChainEmpty, got: %v", err)
	}
}

func TestGetBlocksInRange(t *testing.T) {
	db := openTestDB(t)
	repo, err := NewBboltRepository(db)
	if err != nil {
		t.Fatalf("NewBboltRepository: %v", err)
	}

	ctx := context.Background()
	pow := &block.ProofOfWork{}

	// Create 5 blocks
	genesis, _ := block.NewGenesisBlock("range test", 8, nil, block.Hash{})
	pow.Mine(genesis)
	repo.SaveBlock(ctx, genesis)

	prev := genesis
	blocks := []*block.Block{genesis}
	for i := uint64(1); i <= 4; i++ {
		b, _ := block.NewBlock(prev.Hash(), i, 8, nil, block.Hash{})
		pow.Mine(b)
		repo.SaveBlock(ctx, b)
		blocks = append(blocks, b)
		prev = b
	}

	// Get range [1, 3]
	rangeBlocks, err := repo.GetBlocksInRange(ctx, 1, 3)
	if err != nil {
		t.Fatalf("GetBlocksInRange(1,3): %v", err)
	}
	if len(rangeBlocks) != 3 {
		t.Fatalf("GetBlocksInRange(1,3): got %d blocks, want 3", len(rangeBlocks))
	}
	for i, b := range rangeBlocks {
		expectedHeight := uint64(i + 1)
		if b.Height() != expectedHeight {
			t.Errorf("rangeBlocks[%d].Height() = %d, want %d", i, b.Height(), expectedHeight)
		}
		if b.Hash() != blocks[expectedHeight].Hash() {
			t.Errorf("rangeBlocks[%d].Hash() mismatch", i)
		}
	}
}
