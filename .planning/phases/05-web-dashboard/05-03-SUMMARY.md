---
phase: 05-web-dashboard
plan: 03
subsystem: websocket
tags: [gorilla-websocket, event-bus, real-time, mining-progress, http-server]

requires:
  - phase: 05-web-dashboard
    provides: "Event bus, MineWithProgress, REST API routes, WS event types"
provides:
  - "WebSocket hub with client lifecycle management and event bus forwarding"
  - "Mining progress/start/stop event publishing in auto-mine loops"
  - "P2P block received event publishing"
  - "HTTP server startup in startnode command"
  - "Mempool change event publishing on send and after mining"
affects: [05-web-dashboard]

tech-stack:
  added: [gorilla/websocket@v1.5.3]
  patterns: [websocket-hub-pattern, mining-progress-callback-wiring, event-bridge]

key-files:
  created:
    - internal/handler/ws/hub.go
    - internal/handler/ws/client.go
    - internal/handler/ws/handler.go
    - internal/handler/ws/hub_test.go
  modified:
    - internal/domain/chain/chain.go
    - internal/handler/cli/cli.go
    - internal/handler/cli/signal.go
    - go.mod
    - go.sum

key-decisions:
  - "Chain.OnMiningProgress callback keeps event bus out of domain layer"
  - "Hub started in NewHub constructor (goroutine-based, following gorilla chat example)"
  - "Slow WebSocket clients evicted on full send buffer (non-blocking broadcast)"

patterns-established:
  - "WebSocket hub pattern: register/unregister/broadcast via channels, event bus subscriber goroutine"
  - "Mining event wiring: CLI handler sets Chain.OnMiningProgress callback, domain stays clean"

requirements-completed: [DASH-02, DASH-03]

duration: 4min
completed: 2026-03-07
---

# Phase 05 Plan 03: WebSocket Hub and Event Integration Summary

**WebSocket hub with gorilla/websocket forwarding domain events to browser clients, mining progress publishing, and HTTP server startup in startnode**

## Performance

- **Duration:** 4 min
- **Started:** 2026-03-07T08:21:07Z
- **Completed:** 2026-03-07T08:24:52Z
- **Tasks:** 2
- **Files modified:** 9

## Accomplishments
- WebSocket hub managing client register/unregister/broadcast with event bus subscription forwarding all domain events as JSON
- Mining loops (autoMine and autoMineWithP2P) publish MiningStarted, MiningProgress (sampled every 5000 nonces), MiningStopped, and NewBlock events
- startnode command starts go-zero REST server with WebSocket endpoint alongside P2P server
- Mempool change events published on send command and after mining drains mempool

## Task Commits

Each task was committed atomically:

1. **Task 1: Implement WebSocket hub, client, and handler** - `edb12b1` (test) + `f74cd5d` (feat)
2. **Task 2: Integrate event publishing into mining/P2P and start HTTP server** - `1876c48` (feat)

## Files Created/Modified
- `internal/handler/ws/hub.go` - WebSocket hub with register/unregister/broadcast select loop and event bus subscriber
- `internal/handler/ws/client.go` - Per-client read/write pumps with ping/pong keep-alive (60s timeout)
- `internal/handler/ws/handler.go` - HTTP upgrade handler with permissive CORS
- `internal/handler/ws/hub_test.go` - 5 tests for hub register, unregister, broadcast, slow client eviction, event bus forwarding
- `internal/domain/chain/chain.go` - Added OnMiningProgress callback field, MineBlock uses MineWithProgress when set
- `internal/handler/cli/cli.go` - startnode starts WebSocket hub + HTTP server, send publishes mempool events, peer connect events
- `internal/handler/cli/signal.go` - autoMine/autoMineWithP2P publish mining lifecycle and new block events
- `go.mod` / `go.sum` - Added gorilla/websocket v1.5.3

## Decisions Made
- Chain.OnMiningProgress callback keeps event bus out of domain layer -- CLI handler sets it, domain stays decoupled
- Hub is started in NewHub constructor as a goroutine (matches gorilla chat example pattern)
- Slow WebSocket clients are evicted (closed and deleted) when their send buffer is full, preventing one slow client from blocking all broadcasts

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered
None

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- WebSocket endpoint at /ws ready for dashboard frontend to connect
- All mining/P2P/mempool events flowing through event bus to WebSocket clients
- REST API stub handlers at /api/* ready for Plan 02 implementations (already done)
- Plan 04 (frontend dashboard) can consume the WebSocket and REST API

---
*Phase: 05-web-dashboard*
*Completed: 2026-03-07*
