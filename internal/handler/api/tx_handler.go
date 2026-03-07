package api

import (
	"net/http"

	"github.com/baotoq/shitcoin/internal/domain/tx"
	"github.com/baotoq/shitcoin/internal/infrastructure/persistence/bbolt"
	"github.com/baotoq/shitcoin/internal/svc"
	"github.com/zeromicro/go-zero/rest/httpx"
	"github.com/zeromicro/go-zero/rest/pathvar"
)

// TxHandler returns a handler for GET /api/tx/:hash.
// Scans the chain from tip backwards looking for a matching transaction hash.
// Returns the transaction with block context (height and hash).
// This is O(n) scan -- acceptable for educational project.
func TxHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		vars := pathvar.Vars(r)
		hashStr := vars["hash"]

		chainHeight := svcCtx.Chain.Height()

		// Scan blocks from tip backwards
		for h := int64(chainHeight); h >= 0; h-- {
			b, err := svcCtx.ChainRepo.GetBlockByHeight(ctx, uint64(h))
			if err != nil {
				continue
			}

			for _, rawTx := range b.RawTransactions() {
				t, ok := rawTx.(*tx.Transaction)
				if !ok {
					continue
				}
				if t.ID().String() == hashStr {
					resp := TxResponse{
						Tx:          bbolt.TxModelFromDomain(t),
						BlockHeight: b.Height(),
						BlockHash:   b.Hash().String(),
					}
					httpx.OkJsonCtx(ctx, w, resp)
					return
				}
			}
		}

		httpx.WriteJsonCtx(ctx, w, http.StatusNotFound, ErrorResponse{Error: "transaction not found"})
	}
}
