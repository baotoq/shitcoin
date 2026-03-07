package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/baotoq/shitcoin/internal/domain/block"
	"github.com/baotoq/shitcoin/internal/domain/chain"
	"github.com/baotoq/shitcoin/internal/domain/mempool"
	"github.com/baotoq/shitcoin/internal/domain/tx"
	"github.com/baotoq/shitcoin/internal/infrastructure/persistence/bbolt"
	"github.com/baotoq/shitcoin/internal/svc"
	"github.com/baotoq/shitcoin/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zeromicro/go-zero/rest/pathvar"
)

type mockPeerCounter struct {
	count int
}

func (m *mockPeerCounter) PeerCount() int { return m.count }

func TestStatusHandler_ReturnsNodeMetrics(t *testing.T) {
	repo := testutil.NewMockChainRepo()
	genesis := testutil.MustCreateBlock(t, 0, block.Hash{})
	repo.AddBlock(genesis)

	pow := &block.ProofOfWork{}
	ch := chain.NewChain(repo, pow, chain.ChainConfig{InitialDifficulty: 1}, nil)
	require.NoError(t, ch.Initialize(t.Context(), ""))

	pool := mempool.New(nil)
	svcCtx := &svc.ServiceContext{Chain: ch, Mempool: pool}

	pc := &mockPeerCounter{count: 3}
	handler := StatusHandler(svcCtx, pc)

	req := httptest.NewRequest(http.MethodGet, "/api/status", nil)
	w := httptest.NewRecorder()
	handler(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp StatusResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))

	assert.Equal(t, uint64(0), resp.ChainHeight)
	assert.Equal(t, 0, resp.MempoolSize)
	assert.Equal(t, 3, resp.PeerCount)
	assert.NotEmpty(t, resp.LatestBlockHash)
}

func TestStatusHandler_NilPeerCounter(t *testing.T) {
	repo := testutil.NewMockChainRepo()
	genesis := testutil.MustCreateBlock(t, 0, block.Hash{})
	repo.AddBlock(genesis)

	pow := &block.ProofOfWork{}
	ch := chain.NewChain(repo, pow, chain.ChainConfig{InitialDifficulty: 1}, nil)
	require.NoError(t, ch.Initialize(t.Context(), ""))

	pool := mempool.New(nil)
	svcCtx := &svc.ServiceContext{Chain: ch, Mempool: pool}

	handler := StatusHandler(svcCtx, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/status", nil)
	w := httptest.NewRecorder()
	handler(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp StatusResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, 0, resp.PeerCount)
}

func TestMempoolHandler_ReturnsEmptyArray(t *testing.T) {
	pool := mempool.New(nil)
	svcCtx := &svc.ServiceContext{Mempool: pool}

	handler := MempoolHandler(svcCtx)

	req := httptest.NewRequest(http.MethodGet, "/api/mempool", nil)
	w := httptest.NewRecorder()
	handler(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp []bbolt.TxModel
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Empty(t, resp)
}

func TestTxHandler_NotFound(t *testing.T) {
	repo := testutil.NewMockChainRepo()
	genesis := testutil.MustCreateBlock(t, 0, block.Hash{})
	repo.AddBlock(genesis)

	pow := &block.ProofOfWork{}
	ch := chain.NewChain(repo, pow, chain.ChainConfig{InitialDifficulty: 1}, nil)
	require.NoError(t, ch.Initialize(t.Context(), ""))

	svcCtx := &svc.ServiceContext{Chain: ch, ChainRepo: repo}

	handler := TxHandler(svcCtx)
	fakeHash := "0000000000000000000000000000000000000000000000000000000000000000"
	req := httptest.NewRequest(http.MethodGet, "/api/tx/"+fakeHash, nil)
	req = pathvar.WithVars(req, map[string]string{"hash": fakeHash})
	w := httptest.NewRecorder()
	handler(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestTxHandler_FindsTxInBlock(t *testing.T) {
	repo := testutil.NewMockChainRepo()
	genesis := testutil.MustCreateBlock(t, 0, block.Hash{})
	repo.AddBlock(genesis)

	pow := &block.ProofOfWork{}
	ch := chain.NewChain(repo, pow, chain.ChainConfig{InitialDifficulty: 1}, nil)
	require.NoError(t, ch.Initialize(t.Context(), ""))

	svcCtx := &svc.ServiceContext{Chain: ch, ChainRepo: repo}

	// Get the coinbase tx ID from genesis block
	coinbaseTx := genesis.RawTransactions()[0].(*tx.Transaction)
	txIDStr := coinbaseTx.ID().String()

	handler := TxHandler(svcCtx)
	req := httptest.NewRequest(http.MethodGet, "/api/tx/"+txIDStr, nil)
	req = pathvar.WithVars(req, map[string]string{"hash": txIDStr})
	w := httptest.NewRecorder()
	handler(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp TxResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, txIDStr, resp.Tx.ID)
	assert.Equal(t, uint64(0), resp.BlockHeight)
}
