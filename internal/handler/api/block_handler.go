package api

import (
	"net/http"
	"strconv"

	"github.com/baotoq/shitcoin/internal/domain/block"
	"github.com/baotoq/shitcoin/internal/infrastructure/persistence/bbolt"
	"github.com/baotoq/shitcoin/internal/svc"
	"github.com/zeromicro/go-zero/rest/httpx"
	"github.com/zeromicro/go-zero/rest/pathvar"
)

// BlocksHandler returns a handler for GET /api/blocks?page=1&limit=20.
// Returns paginated block list newest-first.
func BlocksHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		page, _ := strconv.Atoi(r.URL.Query().Get("page"))
		if page < 1 {
			page = 1
		}
		limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
		if limit < 1 {
			limit = 20
		}
		if limit > 100 {
			limit = 100
		}

		ctx := r.Context()
		chainHeight, err := svcCtx.ChainRepo.GetChainHeight(ctx)
		if err != nil {
			httpx.WriteJsonCtx(ctx, w, http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
			return
		}

		total := chainHeight + 1 // heights are 0-indexed

		// Calculate range for newest-first pagination
		// Page 1 with limit 3 on height 4: blocks 4, 3, 2
		offset := uint64((page - 1) * limit)
		if offset >= total {
			httpx.OkJsonCtx(ctx, w, BlockListResponse{
				Blocks: []bbolt.BlockModel{},
				Total:  total,
				Page:   page,
				Limit:  limit,
			})
			return
		}

		// endHeight is the highest block to return (newest first)
		endHeight := chainHeight - offset
		count := uint64(limit)
		if count > endHeight+1 {
			count = endHeight + 1
		}
		startHeight := endHeight - count + 1

		blocks, err := svcCtx.ChainRepo.GetBlocksInRange(ctx, startHeight, endHeight)
		if err != nil {
			httpx.WriteJsonCtx(ctx, w, http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
			return
		}

		// Reverse to newest-first order (GetBlocksInRange returns ascending)
		models := make([]bbolt.BlockModel, len(blocks))
		for i, b := range blocks {
			models[len(blocks)-1-i] = *bbolt.BlockModelFromDomain(b)
		}

		httpx.OkJsonCtx(ctx, w, BlockListResponse{
			Blocks: models,
			Total:  total,
			Page:   page,
			Limit:  limit,
		})
	}
}

// BlockByHeightHandler returns a handler for GET /api/blocks/:height.
// Returns full block with transactions for the given height.
func BlockByHeightHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		vars := pathvar.Vars(r)
		heightStr := vars["height"]

		height, err := strconv.ParseUint(heightStr, 10, 64)
		if err != nil {
			httpx.WriteJsonCtx(ctx, w, http.StatusBadRequest, ErrorResponse{Error: "invalid height"})
			return
		}

		b, err := svcCtx.ChainRepo.GetBlockByHeight(ctx, height)
		if err != nil {
			httpx.WriteJsonCtx(ctx, w, http.StatusNotFound, ErrorResponse{Error: "block not found"})
			return
		}

		httpx.OkJsonCtx(ctx, w, bbolt.BlockModelFromDomain(b))
	}
}

// BlockByHashHandler returns a handler for GET /api/blocks/hash/:hash.
// Returns full block with transactions for the given hash.
func BlockByHashHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		vars := pathvar.Vars(r)
		hashStr := vars["hash"]

		hash, err := block.HashFromHex(hashStr)
		if err != nil {
			httpx.WriteJsonCtx(ctx, w, http.StatusBadRequest, ErrorResponse{Error: "invalid hash"})
			return
		}

		b, err := svcCtx.ChainRepo.GetBlock(ctx, hash)
		if err != nil {
			httpx.WriteJsonCtx(ctx, w, http.StatusNotFound, ErrorResponse{Error: "block not found"})
			return
		}

		httpx.OkJsonCtx(ctx, w, bbolt.BlockModelFromDomain(b))
	}
}
