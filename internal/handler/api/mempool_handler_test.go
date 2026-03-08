package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/baotoq/shitcoin/internal/domain/block"
	"github.com/baotoq/shitcoin/internal/domain/mempool"
	"github.com/baotoq/shitcoin/internal/domain/tx"
	"github.com/baotoq/shitcoin/internal/domain/utxo"
	"github.com/baotoq/shitcoin/internal/infrastructure/persistence/bbolt"
	"github.com/baotoq/shitcoin/internal/svc"
	"github.com/baotoq/shitcoin/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMempoolHandler_WithTransactions(t *testing.T) {
	utxoRepo := testutil.NewMockUTXORepo()
	// Pre-populate the UTXO that the coinbase input references (zero hash, vout 0xFFFFFFFF)
	fakeUTXO := utxo.NewUTXO(block.Hash{}, 0xFFFFFFFF, 0, "")
	require.NoError(t, utxoRepo.Put(fakeUTXO))
	utxoSet := utxo.NewSet(utxoRepo)
	pool := mempool.New(utxoSet)
	coinbaseTx := tx.NewCoinbaseTx("testaddr", 5000)
	require.NoError(t, pool.Add(coinbaseTx))

	svcCtx := &svc.ServiceContext{Mempool: pool}
	handler := MempoolHandler(svcCtx)

	req := httptest.NewRequest(http.MethodGet, "/api/mempool", nil)
	w := httptest.NewRecorder()
	handler(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp []bbolt.TxModel
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Len(t, resp, 1)
	assert.Equal(t, coinbaseTx.ID().String(), resp[0].ID)
}
