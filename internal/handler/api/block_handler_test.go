package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/baotoq/shitcoin/internal/domain/block"
	"github.com/baotoq/shitcoin/internal/infrastructure/persistence/bbolt"
	"github.com/baotoq/shitcoin/internal/svc"
	"github.com/baotoq/shitcoin/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zeromicro/go-zero/rest/pathvar"
)

func TestBlocksHandler_PaginatedNewestFirst(t *testing.T) {
	repo := testutil.NewMockChainRepo()

	genesis := testutil.MustCreateBlock(t, 0, block.Hash{})
	repo.AddBlock(genesis)
	prev := genesis.Hash()
	for h := uint64(1); h <= 4; h++ {
		b := testutil.MustCreateBlock(t, h, prev)
		repo.AddBlock(b)
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
	repo := testutil.NewMockChainRepo()
	genesis := testutil.MustCreateBlock(t, 0, block.Hash{})
	repo.AddBlock(genesis)

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
	repo := testutil.NewMockChainRepo()
	svcCtx := &svc.ServiceContext{ChainRepo: repo}

	handler := BlockByHeightHandler(svcCtx)
	req := httptest.NewRequest(http.MethodGet, "/api/blocks/999", nil)
	req = pathvar.WithVars(req, map[string]string{"height": "999"})
	w := httptest.NewRecorder()
	handler(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}
