package api

import (
	"net/http"

	"github.com/baotoq/shitcoin/internal/infrastructure/persistence/bbolt"
	"github.com/baotoq/shitcoin/internal/svc"
	"github.com/zeromicro/go-zero/rest/httpx"
	"github.com/zeromicro/go-zero/rest/pathvar"
)

// AddressHandler returns a handler for GET /api/address/:addr.
// Returns balance and UTXOs for the given address.
// Returns empty UTXOs and 0 balance for unknown addresses (not 404).
func AddressHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		vars := pathvar.Vars(r)
		addr := vars["addr"]

		utxos, err := svcCtx.UTXOSet.GetByAddress(addr)
		if err != nil {
			httpx.WriteJsonCtx(ctx, w, http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
			return
		}

		var balance int64
		models := make([]bbolt.UTXOModel, 0, len(utxos))
		for _, u := range utxos {
			balance += u.Value()
			models = append(models, bbolt.UTXOModelFromDomain(u))
		}

		httpx.OkJsonCtx(ctx, w, AddressResponse{
			Address: addr,
			Balance: balance,
			UTXOs:   models,
		})
	}
}
