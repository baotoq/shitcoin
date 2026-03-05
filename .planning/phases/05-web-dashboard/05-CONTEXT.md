# Phase 5: Web Dashboard - Context

**Gathered:** 2026-03-06
**Status:** Ready for planning

<domain>
## Phase Boundary

Users can visually explore the blockchain, monitor node health, and watch mining in real-time through a web browser. Covers block explorer, node status panel, real-time mining visualization, mempool view, and search by block hash/tx hash/address.

Requirements: DASH-01, DASH-02, DASH-03, DASH-04, DASH-05

</domain>

<decisions>
## Implementation Decisions

### Frontend stack
- React + Vite + TypeScript SPA
- Separate dev server (React runs on its own port, proxies API requests to Go backend)
- Not embedded in Go binary -- frontend and backend are separate processes during development
- go-zero serves the REST API endpoints on its existing HTTP server (rest.RestConf already in config)

### Real-time updates
- WebSocket for all live data (not SSE or polling)
- gorilla/websocket alongside go-zero's HTTP server
- Push all node events: new blocks, new transactions, peer connect/disconnect, mempool changes, chain reorgs
- Mining visualization: sampled updates (every Nth nonce attempt) -- shows hash values, current nonce, target comparison without flooding
- Single-node dashboard -- connects to the node it's served from, no multi-node switching

### Claude's Discretion
- Dashboard layout and page structure (single page vs multi-page, panel arrangement)
- Search implementation details (block hash, tx hash, address lookup)
- CSS framework / component library choice (Tailwind, shadcn, etc.)
- WebSocket message format and reconnection strategy
- Sampling rate for mining nonce updates (every 1000, 5000, etc.)
- REST API endpoint design for block explorer queries
- Navigation between blocks, transactions, and addresses

</decisions>

<code_context>
## Existing Code Insights

### Reusable Assets
- `rest.RestConf` already embedded in `config.Config` -- go-zero HTTP server ready to use
- `ServiceContext` (internal/svc/) wires all domain aggregates -- API handlers can access Chain, UTXOSet, Mempool, WalletRepo
- All domain data is JSON-serialized -- blocks, transactions, headers already have JSON-friendly formats
- `chain.Chain` aggregate provides block retrieval, height queries, tip access
- `utxo.Set` provides balance queries and UTXO lookups
- `mempool.Mempool` provides pending transaction access
- P2P `Server` has peer manager with connected peer info

### Established Patterns
- go-zero handler pattern: handler -> logic -> model layers
- Domain entities with getters (block.Block, tx.Transaction)
- Repository interfaces in domain, implementations in infrastructure/persistence
- `json` struct tags used for all serialization (go-zero convention)

### Integration Points
- `cmd/shitcoin/main.go` needs to start HTTP server alongside P2P and CLI
- WebSocket handler needs access to domain event bus (new blocks, txs, mining progress)
- Mining loop (in handler/cli) needs to emit progress events for WebSocket consumers
- P2P server callbacks (OnBlockReceived, peer events) need to publish to WebSocket hub

</code_context>

<specifics>
## Specific Ideas

No specific requirements -- open to standard approaches for dashboard layout, styling, and navigation.

</specifics>

<deferred>
## Deferred Ideas

None -- discussion stayed within phase scope

</deferred>

---

*Phase: 05-web-dashboard*
*Context gathered: 2026-03-06*
