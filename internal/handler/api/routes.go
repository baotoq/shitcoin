package api

import (
	"net/http"

	"github.com/baotoq/shitcoin/internal/svc"
	"github.com/zeromicro/go-zero/rest"
	"github.com/zeromicro/go-zero/rest/httpx"
)

// PeerCounter provides the current number of connected peers.
// Decouples API handlers from the p2p.Server implementation.
type PeerCounter interface {
	PeerCount() int
}

// RegisterRoutes registers all REST API and WebSocket routes on the server.
// Hub must implement ServeHTTP for WebSocket upgrade handling.
func RegisterRoutes(server *rest.Server, svcCtx *svc.ServiceContext, peerCounter PeerCounter, hub http.Handler) {
	notImplemented := func(w http.ResponseWriter, r *http.Request) {
		httpx.OkJson(w, map[string]string{"status": "not implemented"})
	}

	server.AddRoutes([]rest.Route{
		{Method: http.MethodGet, Path: "/api/status", Handler: notImplemented},
		{Method: http.MethodGet, Path: "/api/blocks", Handler: notImplemented},
		{Method: http.MethodGet, Path: "/api/blocks/:height", Handler: notImplemented},
		{Method: http.MethodGet, Path: "/api/blocks/hash/:hash", Handler: notImplemented},
		{Method: http.MethodGet, Path: "/api/tx/:hash", Handler: notImplemented},
		{Method: http.MethodGet, Path: "/api/mempool", Handler: notImplemented},
		{Method: http.MethodGet, Path: "/api/address/:addr", Handler: notImplemented},
		{Method: http.MethodGet, Path: "/api/search", Handler: notImplemented},
	})

	// WebSocket route with no timeout (long-lived connection).
	server.AddRoute(rest.Route{
		Method:  http.MethodGet,
		Path:    "/ws",
		Handler: hub.ServeHTTP,
	}, rest.WithTimeout(0))
}
