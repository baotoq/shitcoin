package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/baotoq/shitcoin/internal/domain/block"
	"github.com/baotoq/shitcoin/internal/domain/utxo"
	"github.com/baotoq/shitcoin/internal/svc"
	"github.com/baotoq/shitcoin/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zeromicro/go-zero/rest/pathvar"
)

func TestAddressHandler_WithUTXOs(t *testing.T) {
	utxoRepo := testutil.NewMockUTXORepo()
	// Add a UTXO for the test address
	u := utxo.NewUTXO(block.Hash{1}, 0, 5000, "1TestAddr")
	require.NoError(t, utxoRepo.Put(u))

	utxoSet := utxo.NewSet(utxoRepo)
	svcCtx := &svc.ServiceContext{UTXOSet: utxoSet}

	handler := AddressHandler(svcCtx)
	req := httptest.NewRequest(http.MethodGet, "/api/address/1TestAddr", nil)
	req = pathvar.WithVars(req, map[string]string{"addr": "1TestAddr"})
	w := httptest.NewRecorder()
	handler(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp AddressResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, "1TestAddr", resp.Address)
	assert.Equal(t, int64(5000), resp.Balance)
	assert.Len(t, resp.UTXOs, 1)
}

func TestAddressHandler_UnknownAddress(t *testing.T) {
	utxoRepo := testutil.NewMockUTXORepo()
	utxoSet := utxo.NewSet(utxoRepo)
	svcCtx := &svc.ServiceContext{UTXOSet: utxoSet}

	handler := AddressHandler(svcCtx)
	req := httptest.NewRequest(http.MethodGet, "/api/address/1UnknownAddr", nil)
	req = pathvar.WithVars(req, map[string]string{"addr": "1UnknownAddr"})
	w := httptest.NewRecorder()
	handler(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp AddressResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, "1UnknownAddr", resp.Address)
	assert.Equal(t, int64(0), resp.Balance)
	assert.Empty(t, resp.UTXOs)
}

func TestAddressHandler_RepoError(t *testing.T) {
	// Use an error-returning mock by wrapping the repo
	utxoRepo := &errUTXORepo{}
	utxoSet := utxo.NewSet(utxoRepo)
	svcCtx := &svc.ServiceContext{UTXOSet: utxoSet}

	handler := AddressHandler(svcCtx)
	req := httptest.NewRequest(http.MethodGet, "/api/address/1ErrorAddr", nil)
	req = pathvar.WithVars(req, map[string]string{"addr": "1ErrorAddr"})
	w := httptest.NewRecorder()
	handler(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)

	var resp ErrorResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Contains(t, resp.Error, "repo error")
}

// errUTXORepo is a utxo.Repository that always returns errors from GetByAddress.
type errUTXORepo struct{}

func (e *errUTXORepo) Put(_ utxo.UTXO) error                              { return nil }
func (e *errUTXORepo) Get(_ block.Hash, _ uint32) (utxo.UTXO, error)      { return utxo.UTXO{}, nil }
func (e *errUTXORepo) Delete(_ block.Hash, _ uint32) error                 { return nil }
func (e *errUTXORepo) GetByAddress(_ string) ([]utxo.UTXO, error)         { return nil, errRepoError }
func (e *errUTXORepo) SaveUndoEntry(_ *utxo.UndoEntry) error              { return nil }
func (e *errUTXORepo) GetUndoEntry(_ uint64) (*utxo.UndoEntry, error)     { return nil, nil }
func (e *errUTXORepo) DeleteUndoEntry(_ uint64) error                     { return nil }

var errRepoError = errors.New("repo error")
