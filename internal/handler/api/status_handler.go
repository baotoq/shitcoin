package api

import (
	"net/http"

	"github.com/baotoq/shitcoin/internal/svc"
	"github.com/zeromicro/go-zero/rest/httpx"
)

// StatusHandler returns a handler for GET /api/status.
// Returns chain height, latest block hash, mempool size, peer count, and mining status.
func StatusHandler(svcCtx *svc.ServiceContext, peerCounter PeerCounter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		peerCount := 0
		if peerCounter != nil {
			peerCount = peerCounter.PeerCount()
		}

		latestBlock := svcCtx.Chain.LatestBlock()
		latestHash := ""
		if latestBlock != nil {
			latestHash = latestBlock.Hash().String()
		}

		resp := StatusResponse{
			ChainHeight:     svcCtx.Chain.Height(),
			LatestBlockHash: latestHash,
			MempoolSize:     svcCtx.Mempool.Count(),
			PeerCount:       peerCount,
			IsMining:        false,
		}

		httpx.OkJsonCtx(r.Context(), w, resp)
	}
}
