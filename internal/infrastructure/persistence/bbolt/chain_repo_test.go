package bbolt

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/baotoq/shitcoin/internal/domain/block"
	"github.com/baotoq/shitcoin/internal/domain/chain"
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
