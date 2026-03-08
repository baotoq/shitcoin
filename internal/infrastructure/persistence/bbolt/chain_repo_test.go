package bbolt

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/baotoq/shitcoin/internal/domain/block"
	"github.com/baotoq/shitcoin/internal/domain/chain"
	"github.com/baotoq/shitcoin/internal/domain/tx"
	"github.com/baotoq/shitcoin/internal/domain/utxo"
	"github.com/baotoq/shitcoin/internal/testutil"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	bolt "go.etcd.io/bbolt"
)

// ChainRepoSuite tests BboltRepository with a fresh DB per test.
type ChainRepoSuite struct {
	suite.Suite
	db   *bolt.DB
	repo *BboltRepository
}

func (s *ChainRepoSuite) SetupTest() {
	dbPath := filepath.Join(s.T().TempDir(), "test.db")
	db, err := bolt.Open(dbPath, 0600, nil)
	s.Require().NoError(err)
	s.T().Cleanup(func() { db.Close() })
	s.db = db

	repo, err := NewBboltRepository(db)
	s.Require().NoError(err)
	s.repo = repo
}

// createTestBlock creates a mined block at the given height for testing.
func (s *ChainRepoSuite) createTestBlock(prevHash block.Hash, height uint64, bits uint32) *block.Block {
	s.T().Helper()
	var b *block.Block
	var err error
	if height == 0 {
		b, err = block.NewGenesisBlock("test genesis", bits, nil, block.Hash{})
	} else {
		b, err = block.NewBlock(prevHash, height, bits, nil, block.Hash{})
	}
	s.Require().NoError(err)
	pow := &block.ProofOfWork{}
	s.Require().NoError(pow.Mine(b))
	return b
}

// createChain creates and saves a chain of blocks, returning all blocks.
func (s *ChainRepoSuite) createChain(count int) []*block.Block {
	ctx := context.Background()
	pow := &block.ProofOfWork{}

	genesis, err := block.NewGenesisBlock("test", 8, nil, block.Hash{})
	s.Require().NoError(err)
	s.Require().NoError(pow.Mine(genesis))
	s.Require().NoError(s.repo.SaveBlock(ctx, genesis))

	blocks := []*block.Block{genesis}
	prev := genesis
	for i := uint64(1); i < uint64(count); i++ {
		b, err := block.NewBlock(prev.Hash(), i, 8, nil, block.Hash{})
		s.Require().NoError(err)
		s.Require().NoError(pow.Mine(b))
		s.Require().NoError(s.repo.SaveBlock(ctx, b))
		blocks = append(blocks, b)
		prev = b
	}
	return blocks
}

func TestChainRepoSuite(t *testing.T) {
	suite.Run(t, new(ChainRepoSuite))
}

func (s *ChainRepoSuite) TestSaveAndGetBlock() {
	ctx := context.Background()
	b := s.createTestBlock(block.Hash{}, 0, 8)

	s.Require().NoError(s.repo.SaveBlock(ctx, b))

	got, err := s.repo.GetBlock(ctx, b.Hash())
	s.Require().NoError(err)

	s.Assert().Equal(b.Hash(), got.Hash())
	s.Assert().Equal(b.Height(), got.Height())
	s.Assert().Equal(b.Bits(), got.Bits())
	s.Assert().Equal(b.Timestamp(), got.Timestamp())
	s.Assert().Equal(b.Message(), got.Message())
	s.Assert().Equal(b.Header().Nonce(), got.Header().Nonce())
}

func (s *ChainRepoSuite) TestGetBlockByHeight() {
	blocks := s.createChain(3)
	ctx := context.Background()

	tests := []struct {
		height   uint64
		wantHash block.Hash
	}{
		{0, blocks[0].Hash()},
		{1, blocks[1].Hash()},
		{2, blocks[2].Hash()},
	}

	for _, tt := range tests {
		got, err := s.repo.GetBlockByHeight(ctx, tt.height)
		s.Require().NoError(err)
		s.Assert().Equal(tt.wantHash, got.Hash(), "height %d", tt.height)
	}
}

func (s *ChainRepoSuite) TestGetLatestBlock() {
	blocks := s.createChain(3)
	ctx := context.Background()

	latest, err := s.repo.GetLatestBlock(ctx)
	s.Require().NoError(err)
	s.Assert().Equal(blocks[2].Hash(), latest.Hash())
	s.Assert().Equal(uint64(2), latest.Height())
}

func (s *ChainRepoSuite) TestGetChainHeight() {
	ctx := context.Background()

	// Empty chain
	height, err := s.repo.GetChainHeight(ctx)
	s.Require().NoError(err)
	s.Assert().Equal(uint64(0), height)

	// After 3 blocks (heights 0, 1, 2)
	s.createChain(3)

	height, err = s.repo.GetChainHeight(ctx)
	s.Require().NoError(err)
	s.Assert().Equal(uint64(2), height)
}

func (s *ChainRepoSuite) TestChainPersistence() {
	dbPath := filepath.Join(s.T().TempDir(), "persist.db")

	// Phase 1: open DB, save blocks, close
	db1, err := bolt.Open(dbPath, 0600, nil)
	s.Require().NoError(err)

	repo1, err := NewBboltRepository(db1)
	s.Require().NoError(err)

	ctx := context.Background()
	pow := &block.ProofOfWork{}

	genesis, err := block.NewGenesisBlock("persist test", 8, nil, block.Hash{})
	s.Require().NoError(err)
	s.Require().NoError(pow.Mine(genesis))
	s.Require().NoError(repo1.SaveBlock(ctx, genesis))

	block1, err := block.NewBlock(genesis.Hash(), 1, 8, nil, block.Hash{})
	s.Require().NoError(err)
	s.Require().NoError(pow.Mine(block1))
	s.Require().NoError(repo1.SaveBlock(ctx, block1))

	db1.Close()

	// Phase 2: reopen DB, verify blocks still there
	db2, err := bolt.Open(dbPath, 0600, nil)
	s.Require().NoError(err)
	defer db2.Close()

	repo2, err := NewBboltRepository(db2)
	s.Require().NoError(err)

	gotGenesis, err := repo2.GetBlockByHeight(ctx, 0)
	s.Require().NoError(err)
	s.Assert().Equal(genesis.Hash(), gotGenesis.Hash())
	s.Assert().Equal("persist test", gotGenesis.Message())

	gotBlock1, err := repo2.GetBlockByHeight(ctx, 1)
	s.Require().NoError(err)
	s.Assert().Equal(block1.Hash(), gotBlock1.Hash())

	latest, err := repo2.GetLatestBlock(ctx)
	s.Require().NoError(err)
	s.Assert().Equal(block1.Hash(), latest.Hash())

	height, err := repo2.GetChainHeight(ctx)
	s.Require().NoError(err)
	s.Assert().Equal(uint64(1), height)
}

func (s *ChainRepoSuite) TestGetBlockNotFound() {
	ctx := context.Background()
	fakeHash := block.DoubleSHA256([]byte("nonexistent"))

	_, err := s.repo.GetBlock(ctx, fakeHash)
	s.Require().ErrorIs(err, chain.ErrBlockNotFound)
}

func (s *ChainRepoSuite) TestEmptyChainLatestBlock() {
	ctx := context.Background()
	_, err := s.repo.GetLatestBlock(ctx)
	s.Require().ErrorIs(err, chain.ErrChainEmpty)
}

func (s *ChainRepoSuite) TestGetBlocksInRange() {
	blocks := s.createChain(5)
	ctx := context.Background()

	// Get range [1, 3]
	rangeBlocks, err := s.repo.GetBlocksInRange(ctx, 1, 3)
	s.Require().NoError(err)
	s.Require().Len(rangeBlocks, 3)

	for i, b := range rangeBlocks {
		expectedHeight := uint64(i + 1)
		s.Assert().Equal(expectedHeight, b.Height())
		s.Assert().Equal(blocks[expectedHeight].Hash(), b.Hash())
	}
}

// BboltRepository type assertion to ensure suite uses correct type.
func (s *ChainRepoSuite) TestRepoNotNil() {
	require.NotNil(s.T(), s.repo)
}

func (s *ChainRepoSuite) TestSaveBlockWithUTXOs() {
	ctx := context.Background()
	b := testutil.MustCreateBlock(s.T(), 0, block.Hash{})

	// Extract coinbase tx from the block
	coinbaseTx := b.RawTransactions()[0].(*tx.Transaction)

	// Build undo entry: genesis has no spent inputs, only created outputs
	undoEntry := &utxo.UndoEntry{
		BlockHeight: 0,
		Spent:       []utxo.SpentUTXO{},
		Created: []utxo.UTXORef{
			{TxID: coinbaseTx.ID().String(), Vout: 0},
		},
	}

	s.Require().NoError(s.repo.SaveBlockWithUTXOs(ctx, b, undoEntry))

	// Verify block is retrievable
	got, err := s.repo.GetBlock(ctx, b.Hash())
	s.Require().NoError(err)
	s.Assert().Equal(b.Hash(), got.Hash())
	s.Assert().Equal(b.Height(), got.Height())

	// Verify undo entry is retrievable
	gotUndo, err := s.repo.GetUndoEntry(ctx, 0)
	s.Require().NoError(err)
	s.Assert().Equal(uint64(0), gotUndo.BlockHeight)
	s.Require().Len(gotUndo.Created, 1)
	s.Assert().Equal(coinbaseTx.ID().String(), gotUndo.Created[0].TxID)
}

func (s *ChainRepoSuite) TestSaveBlockWithUTXOs_WithSpentInputs() {
	ctx := context.Background()

	// Block 0 (genesis) -- save normally
	block0 := testutil.MustCreateBlock(s.T(), 0, block.Hash{})
	coinbaseTx0 := block0.RawTransactions()[0].(*tx.Transaction)
	s.Require().NoError(s.repo.SaveBlock(ctx, block0))

	// Block 1 -- save via SaveBlockWithUTXOs with spent input referencing block0's coinbase
	block1 := testutil.MustCreateBlock(s.T(), 1, block0.Hash())
	coinbaseTx1 := block1.RawTransactions()[0].(*tx.Transaction)

	undoEntry := &utxo.UndoEntry{
		BlockHeight: 1,
		Spent: []utxo.SpentUTXO{
			{
				TxID:    coinbaseTx0.ID().String(),
				Vout:    0,
				Value:   5_000_000_000,
				Address: "1TestAddr",
			},
		},
		Created: []utxo.UTXORef{
			{TxID: coinbaseTx1.ID().String(), Vout: 0},
		},
	}

	s.Require().NoError(s.repo.SaveBlockWithUTXOs(ctx, block1, undoEntry))

	// Verify block stored
	got, err := s.repo.GetBlock(ctx, block1.Hash())
	s.Require().NoError(err)
	s.Assert().Equal(block1.Hash(), got.Hash())

	// Verify undo entry stored
	gotUndo, err := s.repo.GetUndoEntry(ctx, 1)
	s.Require().NoError(err)
	s.Assert().Equal(uint64(1), gotUndo.BlockHeight)
	s.Require().Len(gotUndo.Spent, 1)
	s.Assert().Equal(coinbaseTx0.ID().String(), gotUndo.Spent[0].TxID)
	s.Require().Len(gotUndo.Created, 1)
}

func (s *ChainRepoSuite) TestDeleteBlocksAbove() {
	blocks := s.createChain(5) // heights 0-4
	ctx := context.Background()

	s.Require().NoError(s.repo.DeleteBlocksAbove(ctx, 2))

	// Blocks 0-2 should still exist
	for h := uint64(0); h <= 2; h++ {
		got, err := s.repo.GetBlockByHeight(ctx, h)
		s.Require().NoError(err, "block at height %d should exist", h)
		s.Assert().Equal(blocks[h].Hash(), got.Hash())
	}

	// Blocks 3-4 should be gone
	for h := uint64(3); h <= 4; h++ {
		_, err := s.repo.GetBlockByHeight(ctx, h)
		s.Require().ErrorIs(err, chain.ErrBlockNotFound, "block at height %d should be gone", h)
	}

	// Chain height should be 2
	height, err := s.repo.GetChainHeight(ctx)
	s.Require().NoError(err)
	s.Assert().Equal(uint64(2), height)

	// Latest block should be at height 2
	latest, err := s.repo.GetLatestBlock(ctx)
	s.Require().NoError(err)
	s.Assert().Equal(blocks[2].Hash(), latest.Hash())
}

func (s *ChainRepoSuite) TestDeleteBlocksAbove_EmptyChain() {
	ctx := context.Background()
	// DeleteBlocksAbove on empty chain should return nil (early return)
	err := s.repo.DeleteBlocksAbove(ctx, 0)
	s.Require().NoError(err)
}

func (s *ChainRepoSuite) TestGetUndoEntry_NotFound() {
	ctx := context.Background()
	_, err := s.repo.GetUndoEntry(ctx, 999)
	s.Require().ErrorIs(err, utxo.ErrUndoEntryNotFound)
}
