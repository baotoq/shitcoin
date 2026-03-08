package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/baotoq/shitcoin/internal/domain/block"
	"github.com/baotoq/shitcoin/internal/domain/chain"
	"github.com/baotoq/shitcoin/internal/domain/tx"
	"github.com/baotoq/shitcoin/internal/svc"
	"github.com/baotoq/shitcoin/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupSearchContext(t *testing.T) (*svc.ServiceContext, *block.Block) {
	t.Helper()
	repo := testutil.NewMockChainRepo()
	genesis := testutil.MustCreateBlock(t, 0, block.Hash{})
	repo.AddBlock(genesis)

	pow := &block.ProofOfWork{}
	ch := chain.NewChain(repo, pow, chain.ChainConfig{InitialDifficulty: 1}, nil)
	require.NoError(t, ch.Initialize(t.Context(), ""))

	return &svc.ServiceContext{Chain: ch, ChainRepo: repo}, genesis
}

func TestSearchHandler_EmptyQuery(t *testing.T) {
	svcCtx, _ := setupSearchContext(t)
	handler := SearchHandler(svcCtx)

	req := httptest.NewRequest(http.MethodGet, "/api/search", nil)
	w := httptest.NewRecorder()
	handler(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var resp ErrorResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Contains(t, resp.Error, "required")
}

func TestSearchHandler_BlockByHash(t *testing.T) {
	svcCtx, genesis := setupSearchContext(t)
	handler := SearchHandler(svcCtx)

	hashHex := genesis.Hash().String()
	req := httptest.NewRequest(http.MethodGet, "/api/search?q="+hashHex, nil)
	w := httptest.NewRecorder()
	handler(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp SearchResult
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, "block", resp.Type)
	require.NotNil(t, resp.BlockHeight)
	assert.Equal(t, uint64(0), *resp.BlockHeight)
}

func TestSearchHandler_TxByHash(t *testing.T) {
	svcCtx, genesis := setupSearchContext(t)
	handler := SearchHandler(svcCtx)

	coinbaseTx := genesis.RawTransactions()[0].(*tx.Transaction)
	txIDHex := coinbaseTx.ID().String()

	req := httptest.NewRequest(http.MethodGet, "/api/search?q="+txIDHex, nil)
	w := httptest.NewRecorder()
	handler(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp SearchResult
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, "tx", resp.Type)
	require.NotNil(t, resp.TxHash)
	assert.Equal(t, txIDHex, *resp.TxHash)
}

func TestSearchHandler_HexNotFound(t *testing.T) {
	svcCtx, _ := setupSearchContext(t)
	handler := SearchHandler(svcCtx)

	fakeHex := "0000000000000000000000000000000000000000000000000000000000000000"
	req := httptest.NewRequest(http.MethodGet, "/api/search?q="+fakeHex, nil)
	w := httptest.NewRecorder()
	handler(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestSearchHandler_BlockByHeight(t *testing.T) {
	svcCtx, _ := setupSearchContext(t)
	handler := SearchHandler(svcCtx)

	req := httptest.NewRequest(http.MethodGet, "/api/search?q=0", nil)
	w := httptest.NewRecorder()
	handler(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp SearchResult
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, "block", resp.Type)
	require.NotNil(t, resp.BlockHeight)
	assert.Equal(t, uint64(0), *resp.BlockHeight)
}

func TestSearchHandler_HeightNotFound(t *testing.T) {
	svcCtx, _ := setupSearchContext(t)
	handler := SearchHandler(svcCtx)

	req := httptest.NewRequest(http.MethodGet, "/api/search?q=999", nil)
	w := httptest.NewRecorder()
	handler(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestSearchHandler_Address(t *testing.T) {
	svcCtx, _ := setupSearchContext(t)
	handler := SearchHandler(svcCtx)

	// A valid-looking Bitcoin address (starts with 1, 25-34 chars)
	addr := "1TestAddress1234567890abcde"
	req := httptest.NewRequest(http.MethodGet, "/api/search?q="+addr, nil)
	w := httptest.NewRecorder()
	handler(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp SearchResult
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, "address", resp.Type)
	require.NotNil(t, resp.Address)
	assert.Equal(t, addr, *resp.Address)
}

func TestSearchHandler_UnknownString(t *testing.T) {
	svcCtx, _ := setupSearchContext(t)
	handler := SearchHandler(svcCtx)

	req := httptest.NewRequest(http.MethodGet, "/api/search?q=foobar", nil)
	w := httptest.NewRecorder()
	handler(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}
