package api

import (
	"net/http"

	"github.com/baotoq/shitcoin/internal/svc"
	"github.com/zeromicro/go-zero/rest"
)

// PeerCounter provides the current number of connected peers.
// Decouples API handlers from the p2p.Server implementation.
type PeerCounter interface {
	PeerCount() int
}

// RegisterRoutes registers all REST API and WebSocket routes on the server.
// Hub must implement ServeHTTP for WebSocket upgrade handling.
func RegisterRoutes(server *rest.Server, svcCtx *svc.ServiceContext, peerCounter PeerCounter, hub http.Handler) {
	server.AddRoutes([]rest.Route{
		{Method: http.MethodGet, Path: "/api/status", Handler: StatusHandler(svcCtx, peerCounter)},
		{Method: http.MethodGet, Path: "/api/blocks", Handler: BlocksHandler(svcCtx)},
		{Method: http.MethodGet, Path: "/api/blocks/:height", Handler: BlockByHeightHandler(svcCtx)},
		{Method: http.MethodGet, Path: "/api/blocks/hash/:hash", Handler: BlockByHashHandler(svcCtx)},
		{Method: http.MethodGet, Path: "/api/tx/:hash", Handler: TxHandler(svcCtx)},
		{Method: http.MethodGet, Path: "/api/mempool", Handler: MempoolHandler(svcCtx)},
		{Method: http.MethodGet, Path: "/api/address/:addr", Handler: AddressHandler(svcCtx)},
		{Method: http.MethodGet, Path: "/api/search", Handler: SearchHandler(svcCtx)},
	})

	// WebSocket route with no timeout (long-lived connection).
	server.AddRoute(rest.Route{
		Method:  http.MethodGet,
		Path:    "/ws",
		Handler: hub.ServeHTTP,
	}, rest.WithTimeout(0))
}
