package api

import (
	"net/http"
	"regexp"
	"strconv"

	"github.com/baotoq/shitcoin/internal/domain/block"
	"github.com/baotoq/shitcoin/internal/domain/tx"
	"github.com/baotoq/shitcoin/internal/svc"
	"github.com/zeromicro/go-zero/rest/httpx"
)

var hexPattern = regexp.MustCompile(`^[0-9a-fA-F]{64}$`)

// SearchHandler returns a handler for GET /api/search?q=...
// Detects query type by format:
//   - 64 hex chars: try as block hash first, then scan as tx hash
//   - Valid address (starts with 1 or 3, 25-34 chars): treat as address
//   - Numeric string: treat as block height
func SearchHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		q := r.URL.Query().Get("q")
		if q == "" {
			httpx.WriteJsonCtx(ctx, w, http.StatusBadRequest, ErrorResponse{Error: "query parameter 'q' is required"})
			return
		}

		// Try as 64-char hex (block hash or tx hash)
		if hexPattern.MatchString(q) {
			hash, err := block.HashFromHex(q)
			if err == nil {
				// Try as block hash
				b, err := svcCtx.ChainRepo.GetBlock(ctx, hash)
				if err == nil {
					height := b.Height()
					hashStr := b.Hash().String()
					httpx.OkJsonCtx(ctx, w, SearchResult{
						Type:        "block",
						BlockHeight: &height,
						BlockHash:   &hashStr,
					})
					return
				}

				// Try as tx hash -- scan chain
				chainHeight := svcCtx.Chain.Height()
				for h := int64(chainHeight); h >= 0; h-- {
					blk, err := svcCtx.ChainRepo.GetBlockByHeight(ctx, uint64(h))
					if err != nil {
						continue
					}
					for _, rawTx := range blk.RawTransactions() {
						t, ok := rawTx.(*tx.Transaction)
						if !ok {
							continue
						}
						if t.ID().String() == q {
							txHash := t.ID().String()
							blkHeight := blk.Height()
							httpx.OkJsonCtx(ctx, w, SearchResult{
								Type:        "tx",
								TxHash:      &txHash,
								BlockHeight: &blkHeight,
							})
							return
						}
					}
				}
			}

			httpx.WriteJsonCtx(ctx, w, http.StatusNotFound, ErrorResponse{Error: "not found"})
			return
		}

		// Try as numeric block height
		if height, err := strconv.ParseUint(q, 10, 64); err == nil {
			b, err := svcCtx.ChainRepo.GetBlockByHeight(ctx, height)
			if err == nil {
				hashStr := b.Hash().String()
				bHeight := b.Height()
				httpx.OkJsonCtx(ctx, w, SearchResult{
					Type:        "block",
					BlockHeight: &bHeight,
					BlockHash:   &hashStr,
				})
				return
			}
			httpx.WriteJsonCtx(ctx, w, http.StatusNotFound, ErrorResponse{Error: "block not found at height"})
			return
		}

		// Try as address (starts with 1 or 3, 25-34 chars)
		if isLikelyAddress(q) {
			httpx.OkJsonCtx(ctx, w, SearchResult{
				Type:    "address",
				Address: &q,
			})
			return
		}

		httpx.WriteJsonCtx(ctx, w, http.StatusNotFound, ErrorResponse{Error: "not found"})
	}
}

// isLikelyAddress checks if a string looks like a Base58Check bitcoin address.
func isLikelyAddress(s string) bool {
	if len(s) < 25 || len(s) > 34 {
		return false
	}
	return s[0] == '1' || s[0] == '3'
}
