package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/baotoq/shitcoin/internal/domain/block"
	"github.com/baotoq/shitcoin/internal/domain/tx"
	"github.com/baotoq/shitcoin/internal/domain/utxo"
	"github.com/baotoq/shitcoin/internal/infrastructure/persistence/bbolt"
	"github.com/baotoq/shitcoin/internal/svc"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zeromicro/go-zero/rest/pathvar"
)

// mockChainRepo is a mock implementation of chain.Repository for testing.
type mockChainRepo struct {
	blocks       map[uint64]*block.Block
	blocksByHash map[block.Hash]*block.Block
	chainHeight  uint64
}

func newMockChainRepo() *mockChainRepo {
	return &mockChainRepo{
		blocks:       make(map[uint64]*block.Block),
		blocksByHash: make(map[block.Hash]*block.Block),
	}
}

func (m *mockChainRepo) SaveBlock(_ context.Context, _ *block.Block) error { return nil }
func (m *mockChainRepo) SaveBlockWithUTXOs(_ context.Context, _ *block.Block, _ *utxo.UndoEntry) error {
	return nil
}
func (m *mockChainRepo) GetBlock(_ context.Context, hash block.Hash) (*block.Block, error) {
	b, ok := m.blocksByHash[hash]
	if !ok {
		return nil, ErrBlockNotFound
	}
	return b, nil
}
func (m *mockChainRepo) GetBlockByHeight(_ context.Context, height uint64) (*block.Block, error) {
	b, ok := m.blocks[height]
	if !ok {
		return nil, ErrBlockNotFound
	}
	return b, nil
}
func (m *mockChainRepo) GetLatestBlock(_ context.Context) (*block.Block, error) {
	if len(m.blocks) == 0 {
		return nil, ErrBlockNotFound
	}
	return m.blocks[m.chainHeight], nil
}
func (m *mockChainRepo) GetChainHeight(_ context.Context) (uint64, error) {
	return m.chainHeight, nil
}
func (m *mockChainRepo) GetBlocksInRange(_ context.Context, start, end uint64) ([]*block.Block, error) {
	var result []*block.Block
	for h := start; h <= end; h++ {
		if b, ok := m.blocks[h]; ok {
			result = append(result, b)
		}
	}
	return result, nil
}
func (m *mockChainRepo) GetUndoEntry(_ context.Context, _ uint64) (*utxo.UndoEntry, error) {
	return nil, nil
}
func (m *mockChainRepo) DeleteBlocksAbove(_ context.Context, _ uint64) error { return nil }

func (m *mockChainRepo) addBlock(b *block.Block) {
	m.blocks[b.Height()] = b
	m.blocksByHash[b.Hash()] = b
	if b.Height() > m.chainHeight || len(m.blocks) == 1 {
		m.chainHeight = b.Height()
	}
}

// createTestBlock creates a mined test block at given height.
func createTestBlock(t *testing.T, height uint64, prevHash block.Hash) *block.Block {
	t.Helper()
	coinbase := tx.NewCoinbaseTxWithHeight("1TestAddr", 5000000000, height)
	blockTxs := []any{coinbase}
	merkleRoot := block.ComputeMerkleRoot([]block.Hash{coinbase.ID()})

	var b *block.Block
	var err error
	if height == 0 {
		b, err = block.NewGenesisBlock("test genesis", 1, blockTxs, merkleRoot)
	} else {
		b, err = block.NewBlock(prevHash, height, 1, blockTxs, merkleRoot)
	}
	require.NoError(t, err)

	pow := &block.ProofOfWork{}
	require.NoError(t, pow.Mine(b))
	return b
}

func TestBlocksHandler_PaginatedNewestFirst(t *testing.T) {
	repo := newMockChainRepo()

	genesis := createTestBlock(t, 0, block.Hash{})
	repo.addBlock(genesis)
	prev := genesis.Hash()
	for h := uint64(1); h <= 4; h++ {
		b := createTestBlock(t, h, prev)
		repo.addBlock(b)
		prev = b.Hash()
	}

	svcCtx := &svc.ServiceContext{ChainRepo: repo}
	handler := BlocksHandler(svcCtx)

	req := httptest.NewRequest(http.MethodGet, "/api/blocks?page=1&limit=3", nil)
	w := httptest.NewRecorder()
	handler(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp BlockListResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))

	assert.Equal(t, uint64(5), resp.Total)
	assert.Equal(t, 1, resp.Page)
	assert.Equal(t, 3, resp.Limit)
	assert.Len(t, resp.Blocks, 3)
	// Newest first: heights 4, 3, 2
	assert.Equal(t, uint64(4), resp.Blocks[0].Height)
	assert.Equal(t, uint64(3), resp.Blocks[1].Height)
	assert.Equal(t, uint64(2), resp.Blocks[2].Height)
}

func TestBlockByHeightHandler_ValidHeight(t *testing.T) {
	repo := newMockChainRepo()
	genesis := createTestBlock(t, 0, block.Hash{})
	repo.addBlock(genesis)

	svcCtx := &svc.ServiceContext{ChainRepo: repo}
	handler := BlockByHeightHandler(svcCtx)

	req := httptest.NewRequest(http.MethodGet, "/api/blocks/0", nil)
	req = pathvar.WithVars(req, map[string]string{"height": "0"})
	w := httptest.NewRecorder()
	handler(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp bbolt.BlockModel
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, uint64(0), resp.Height)
	assert.NotEmpty(t, resp.Hash)
}

func TestBlockByHeightHandler_NotFound(t *testing.T) {
	repo := newMockChainRepo()
	svcCtx := &svc.ServiceContext{ChainRepo: repo}

	handler := BlockByHeightHandler(svcCtx)
	req := httptest.NewRequest(http.MethodGet, "/api/blocks/999", nil)
	req = pathvar.WithVars(req, map[string]string{"height": "999"})
	w := httptest.NewRecorder()
	handler(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}
