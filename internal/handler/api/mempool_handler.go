package api

import (
	"net/http"

	"github.com/baotoq/shitcoin/internal/infrastructure/persistence/bbolt"
	"github.com/baotoq/shitcoin/internal/svc"
	"github.com/zeromicro/go-zero/rest/httpx"
)

// MempoolHandler returns a handler for GET /api/mempool.
// Returns all pending transactions as a TxModel array.
func MempoolHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		txs := svcCtx.Mempool.Transactions()
		models := make([]bbolt.TxModel, 0, len(txs))
		for _, t := range txs {
			models = append(models, bbolt.TxModelFromDomain(t))
		}
		httpx.OkJsonCtx(r.Context(), w, models)
	}
}
