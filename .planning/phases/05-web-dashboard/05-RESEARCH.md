# Phase 5: Web Dashboard - Research

**Researched:** 2026-03-06
**Domain:** Go REST API + WebSocket backend, React SPA frontend, real-time blockchain visualization
**Confidence:** HIGH

## Summary

This phase adds a web dashboard to the existing blockchain node. The backend extends go-zero's already-configured REST server (`rest.RestConf` is embedded in `config.Config`) with REST endpoints for block explorer queries and a gorilla/websocket endpoint for real-time events. The frontend is a separate React + Vite + TypeScript SPA that proxies API requests to the Go backend during development.

The existing codebase provides excellent foundations: `BlockModel`/`TxModel` in `bbolt/storage_model.go` already serialize blocks/transactions to JSON with all needed fields. `ServiceContext` wires all domain aggregates (Chain, UTXOSet, Mempool, WalletRepo) and the ChainRepo provides `GetBlock`, `GetBlockByHeight`, `GetBlocksInRange`. The P2P `Server` has `OnBlockReceived` callback and `PeerCount()`. The mining loop in `cli/signal.go` needs a progress callback injected into `ProofOfWork.Mine()` for real-time nonce visualization.

**Primary recommendation:** Build a WebSocket hub (goroutine) that all domain events publish to, REST handlers that wrap existing domain queries, and a React SPA with Tailwind CSS + shadcn/ui for the UI. The event bus pattern decouples domain code from WebSocket delivery.

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions
- React + Vite + TypeScript SPA
- Separate dev server (React runs on its own port, proxies API requests to Go backend)
- Not embedded in Go binary -- frontend and backend are separate processes during development
- go-zero serves the REST API endpoints on its existing HTTP server (rest.RestConf already in config)
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

### Deferred Ideas (OUT OF SCOPE)
None -- discussion stayed within phase scope
</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|-----------------|
| DASH-01 | Web dashboard displays a block explorer where user can browse blocks and transactions | REST endpoints for blocks/txs, existing `ChainRepo.GetBlocksInRange`/`GetBlock`/`GetBlockByHeight`, `BlockModel` JSON format |
| DASH-02 | Web dashboard shows node status: connected peers, mempool size, chain height, mining status | REST status endpoint reading `Chain.Height()`, `Mempool.Count()`, `Server.PeerCount()`, plus WebSocket push on changes |
| DASH-03 | Web dashboard visualizes mining in real-time (nonce attempts, hash values, target comparison) | Mining progress callback in PoW loop, sampled WebSocket events, frontend real-time display |
| DASH-04 | Web dashboard shows mempool with pending transactions | REST endpoint wrapping `Mempool.Transactions()`, WebSocket push on mempool add/remove |
| DASH-05 | User can search by block hash, transaction hash, or address in the dashboard | REST search endpoint using `ChainRepo.GetBlock(hash)`, iterate blocks for tx lookup, `UTXOSet.GetByAddress()` |
</phase_requirements>

## Standard Stack

### Core (Backend)
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| go-zero/rest | v1.10.0 | HTTP server, routing | Already in project, `rest.RestConf` embedded in Config |
| gorilla/websocket | v1.5.3 | WebSocket server | User decision; most battle-tested Go WebSocket library |

### Core (Frontend)
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| React | 19.x | UI framework | User decision |
| Vite | 6.x | Build tool, dev server | User decision; fast HMR, native ESM |
| TypeScript | 5.x | Type safety | User decision |
| Tailwind CSS | v4.x | Utility-first CSS | Claude's discretion: fast styling, works with shadcn/ui |
| shadcn/ui | latest | Component library | Claude's discretion: copy-paste components, no runtime dep |
| react-router | v7.x | Client-side routing | Standard for multi-page SPA navigation |

### Supporting
| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| @tailwindcss/vite | latest | Tailwind v4 Vite plugin | Required for Tailwind v4 integration |
| lucide-react | latest | Icons | Used by shadcn/ui components |
| recharts | 2.x | Charts (optional) | Mining hashrate visualization if needed |

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| shadcn/ui | Plain Tailwind | Less component boilerplate but more manual work |
| react-router | Single page with tabs | Simpler but no URL-based navigation to blocks/txs |
| gorilla/websocket | nhooyr/websocket | nhooyr has cleaner API but user locked gorilla |

**Installation (Backend):**
```bash
go get github.com/gorilla/websocket@v1.5.3
```

**Installation (Frontend):**
```bash
npm create vite@latest web -- --template react-ts
cd web
npm install
npx shadcn@latest init
npx shadcn@latest add button card table badge input tabs
npm install react-router lucide-react
```

## Architecture Patterns

### Recommended Project Structure
```
internal/
  handler/
    api/                    # NEW: REST API handlers
      block_handler.go      # GET /api/blocks, /api/blocks/:height, /api/blocks/hash/:hash
      tx_handler.go         # GET /api/tx/:hash
      status_handler.go     # GET /api/status
      mempool_handler.go    # GET /api/mempool
      search_handler.go     # GET /api/search?q=...
      address_handler.go    # GET /api/address/:addr
      routes.go             # RegisterRoutes(server, svcCtx)
    ws/                     # NEW: WebSocket hub + handler
      hub.go                # Hub goroutine: register/unregister/broadcast
      client.go             # Per-client read/write pumps
      handler.go            # HTTP upgrade handler
      events.go             # Event type definitions
    cli/                    # Existing CLI handlers
  domain/
    events/                 # NEW: Domain event bus (optional thin layer)
      bus.go                # Publish/Subscribe for node events
web/                        # NEW: React SPA (separate directory, not inside internal/)
  src/
    components/             # Reusable UI components
    pages/                  # Route-level page components
    hooks/                  # Custom hooks (useWebSocket, useNodeStatus)
    lib/                    # API client, WebSocket client, utilities
    types/                  # TypeScript interfaces matching API responses
```

### Pattern 1: WebSocket Hub (Gorilla Chat Example Pattern)
**What:** Central goroutine manages client registration, unregistration, and message broadcasting
**When to use:** Always -- this is the standard gorilla/websocket pattern
**Example:**
```go
// Source: gorilla/websocket chat example pattern
type Hub struct {
    clients    map[*Client]bool
    broadcast  chan []byte
    register   chan *Client
    unregister chan *Client
}

func (h *Hub) Run() {
    for {
        select {
        case client := <-h.register:
            h.clients[client] = true
        case client := <-h.unregister:
            if _, ok := h.clients[client]; ok {
                delete(h.clients, client)
                close(client.send)
            }
        case message := <-h.broadcast:
            for client := range h.clients {
                select {
                case client.send <- message:
                default:
                    close(client.send)
                    delete(h.clients, client)
                }
            }
        }
    }
}
```

### Pattern 2: Event Bus for Domain Decoupling
**What:** Thin publish/subscribe layer so domain code (mining, P2P callbacks) emits events without importing WebSocket packages
**When to use:** To keep domain layer clean of infrastructure concerns
**Example:**
```go
// internal/domain/events/bus.go
type EventType string

const (
    EventNewBlock       EventType = "new_block"
    EventNewTx          EventType = "new_tx"
    EventMiningProgress EventType = "mining_progress"
    EventPeerConnected  EventType = "peer_connected"
    EventPeerDisconnected EventType = "peer_disconnected"
    EventMempoolChanged EventType = "mempool_changed"
    EventReorg          EventType = "reorg"
)

type Event struct {
    Type    EventType   `json:"type"`
    Payload interface{} `json:"payload"`
}

type Bus struct {
    mu          sync.RWMutex
    subscribers []chan Event
}

func (b *Bus) Subscribe() chan Event {
    ch := make(chan Event, 64)
    b.mu.Lock()
    b.subscribers = append(b.subscribers, ch)
    b.mu.Unlock()
    return ch
}

func (b *Bus) Publish(e Event) {
    b.mu.RLock()
    defer b.mu.RUnlock()
    for _, ch := range b.subscribers {
        select {
        case ch <- e:
        default: // drop if subscriber is slow
        }
    }
}
```

### Pattern 3: go-zero REST Route Registration with WebSocket
**What:** Register REST routes and a WebSocket endpoint on the same go-zero server
**When to use:** This is how the backend wires up
**Example:**
```go
// Source: go-zero rest.AddRoute pattern
func RegisterRoutes(server *rest.Server, svcCtx *svc.ServiceContext, hub *ws.Hub) {
    server.AddRoutes([]rest.Route{
        {Method: http.MethodGet, Path: "/api/status", Handler: StatusHandler(svcCtx)},
        {Method: http.MethodGet, Path: "/api/blocks", Handler: BlocksHandler(svcCtx)},
        {Method: http.MethodGet, Path: "/api/blocks/:height", Handler: BlockByHeightHandler(svcCtx)},
    })

    // WebSocket endpoint -- use rest.WithTimeout(0) for persistent connections
    server.AddRoute(rest.Route{
        Method: http.MethodGet,
        Path:   "/ws",
        Handler: ws.ServeWs(hub),
    }, rest.WithTimeout(0))
}
```

### Pattern 4: Mining Progress Callback
**What:** Inject a callback into the PoW mining loop that fires every N nonce attempts
**When to use:** For DASH-03 real-time mining visualization
**Example:**
```go
// Extend ProofOfWork with optional progress callback
type MiningProgress struct {
    Nonce      uint32 `json:"nonce"`
    Hash       string `json:"hash"`
    Target     string `json:"target"`
    Difficulty uint32 `json:"difficulty"`
}

// MineWithProgress is like Mine but calls onProgress every sampleRate nonce attempts
func (pow *ProofOfWork) MineWithProgress(b *Block, sampleRate uint32, onProgress func(MiningProgress)) error {
    target := BitsToTarget(b.header.bits)
    var nonce uint32
    for nonce <= math.MaxUint32 {
        b.header.SetNonce(nonce)
        hash := b.header.Hash()
        hashInt := new(big.Int).SetBytes(hash[:])

        if onProgress != nil && nonce%sampleRate == 0 {
            onProgress(MiningProgress{
                Nonce:      nonce,
                Hash:       hash.String(),
                Target:     fmt.Sprintf("%064x", target),
                Difficulty: b.header.bits,
            })
        }

        if hashInt.Cmp(target) == -1 {
            b.hash = hash
            return nil
        }
        if nonce == math.MaxUint32 {
            break
        }
        nonce++
    }
    return ErrNonceExhausted
}
```

### Pattern 5: Vite Proxy Configuration
**What:** Dev server proxies /api and /ws requests to Go backend
**When to use:** During development for CORS-free API access
**Example:**
```typescript
// vite.config.ts
export default defineConfig({
  plugins: [react(), tailwindcss()],
  server: {
    port: 5173,
    proxy: {
      '/api': 'http://localhost:8080',
      '/ws': {
        target: 'ws://localhost:8080',
        ws: true,
      },
    },
  },
  resolve: {
    alias: {
      '@': path.resolve(__dirname, './src'),
    },
  },
})
```

### Anti-Patterns to Avoid
- **Direct domain import in WebSocket handler:** Use event bus, not direct calls to chain/mempool from WebSocket code
- **Blocking WebSocket writes:** Always use buffered channels with non-blocking sends (select/default pattern)
- **Polling from frontend:** Use WebSocket push for all real-time data, REST only for initial page loads and search
- **Flooding mining updates:** Sample every 1000-5000 nonce attempts, not every single one
- **Missing WebSocket reconnection:** Frontend must auto-reconnect with exponential backoff
- **Forgetting rest.WithTimeout(0):** go-zero has default request timeouts that will kill WebSocket connections

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| WebSocket protocol | Raw TCP WebSocket framing | gorilla/websocket | Protocol compliance, compression, ping/pong |
| UI components | Custom buttons/cards/tables | shadcn/ui + Tailwind | Accessible, themeable, production-ready |
| Dev server proxy | Custom CORS middleware | Vite proxy config | Built-in, handles WebSocket upgrade |
| Client-side routing | Hash-based navigation | react-router v7 | URL params for block/tx lookups, browser history |
| JSON serialization for API | New response models | Reuse bbolt `BlockModel`/`TxModel` | Already have JSON tags, tested, complete |

**Key insight:** The bbolt storage models (`BlockModel`, `TxModel`, `HeaderModel`) are perfect REST API response types. They already have `json` tags and contain all the fields the dashboard needs. Reuse them directly in API responses instead of creating a separate API model layer.

## Common Pitfalls

### Pitfall 1: go-zero Request Timeout Kills WebSocket
**What goes wrong:** WebSocket connections close after go-zero's default timeout (3 seconds)
**Why it happens:** go-zero wraps handlers with a timeout middleware
**How to avoid:** Use `rest.WithTimeout(0)` when adding the WebSocket route
**Warning signs:** WebSocket connects then immediately disconnects

### Pitfall 2: Mining Progress Blocks the Mining Loop
**What goes wrong:** Sending mining progress to the event bus blocks if the channel is full, slowing mining
**Why it happens:** Synchronous channel sends in the hot mining loop
**How to avoid:** Use non-blocking sends (`select/default`) and drop events if the subscriber is slow. Sample rate of 5000 keeps events manageable.
**Warning signs:** Mining becomes noticeably slower when dashboard is open

### Pitfall 3: Race Conditions on Domain State
**What goes wrong:** REST handler reads chain state while P2P/mining is modifying it
**Why it happens:** Concurrent access to Chain, UTXOSet, Mempool from HTTP goroutines
**How to avoid:** Chain already has `sync.RWMutex`; Mempool has `sync.RWMutex`; UTXOSet reads go through bbolt (which handles concurrency). Use existing thread-safe methods.
**Warning signs:** Panics or stale data in API responses

### Pitfall 4: WebSocket Memory Leak on Client Disconnect
**What goes wrong:** Client struct and its channels stay in memory after browser tab closes
**Why it happens:** Missing unregister logic when read/write pumps exit
**How to avoid:** Always unregister in defer; close send channel on unregister; use ping/pong for liveness detection
**Warning signs:** Growing memory usage over time

### Pitfall 5: CORS Issues in Development
**What goes wrong:** Browser blocks API requests from Vite dev server (port 5173) to Go backend (port 8080)
**Why it happens:** Same-origin policy blocks cross-origin requests
**How to avoid:** Use Vite's proxy config to route /api and /ws through the same origin
**Warning signs:** Network errors in browser console, "blocked by CORS policy"

### Pitfall 6: Block Transaction Type Assertions in API Handlers
**What goes wrong:** `rawTx.(*tx.Transaction)` type assertion panics or silently skips transactions
**Why it happens:** Block stores transactions as `[]any`; need type assertion to access tx fields
**How to avoid:** Use the existing `BlockModelFromDomain()` function which already handles type assertions safely, or follow the same pattern from `cli.go`'s `printChain`
**Warning signs:** Missing transactions in API responses

### Pitfall 7: Search by Transaction Hash Requires Full Scan
**What goes wrong:** No index exists for looking up blocks by transaction hash
**Why it happens:** bbolt stores blocks by hash and height, not by contained tx hash
**How to avoid:** For this educational project, scan blocks from tip backwards (acceptable for small chains). Add a tx-hash-to-block-height index in bbolt if performance matters.
**Warning signs:** Slow search for old transactions on long chains

## Code Examples

### REST API Response Types (Reuse Existing Models)
```go
// Source: internal/infrastructure/persistence/bbolt/storage_model.go
// BlockModel, TxModel, HeaderModel already have json tags and are perfect API responses.
// No need to create separate API response types.

// Status response is the only new type needed:
type StatusResponse struct {
    ChainHeight    uint64 `json:"chain_height"`
    LatestBlockHash string `json:"latest_block_hash"`
    MempoolSize    int    `json:"mempool_size"`
    PeerCount      int    `json:"peer_count"`
    IsMining       bool   `json:"is_mining"`
}
```

### WebSocket Event Format
```go
// Unified WebSocket message format
type WSMessage struct {
    Type    string      `json:"type"`    // "new_block", "new_tx", "mining_progress", "status", etc.
    Payload interface{} `json:"payload"` // event-specific data
}

// Mining progress payload (sampled every 5000 nonce attempts)
type MiningProgressPayload struct {
    Nonce      uint32 `json:"nonce"`
    HashHex    string `json:"hash"`
    TargetHex  string `json:"target"`
    Difficulty uint32 `json:"difficulty"`
    BlockHeight uint64 `json:"block_height"`
}
```

### Frontend WebSocket Hook
```typescript
// src/hooks/useWebSocket.ts
function useWebSocket(url: string) {
  const [lastMessage, setLastMessage] = useState<WSMessage | null>(null);
  const wsRef = useRef<WebSocket | null>(null);

  useEffect(() => {
    function connect() {
      const ws = new WebSocket(url);
      wsRef.current = ws;
      ws.onmessage = (event) => setLastMessage(JSON.parse(event.data));
      ws.onclose = () => setTimeout(connect, 1000 + Math.random() * 2000); // reconnect with jitter
    }
    connect();
    return () => wsRef.current?.close();
  }, [url]);

  return lastMessage;
}
```

### Starting HTTP Server Alongside P2P
```go
// In cmd/shitcoin/main.go or handler/cli startNode:
// go-zero's REST server starts on the port from rest.RestConf (Port: 8080)
server := rest.MustNewServer(c.RestConf)
defer server.Stop()

// Register API routes
api.RegisterRoutes(server, serviceCtx, hub)

// Start in background goroutine
go server.Start()

// P2P server starts on its own port (c.P2P.Port, default 3000)
```

## REST API Design

### Endpoints
| Method | Path | Response | Notes |
|--------|------|----------|-------|
| GET | `/api/status` | `StatusResponse` | Chain height, peers, mempool, mining |
| GET | `/api/blocks?page=1&limit=20` | `[]BlockModel` (without full tx details) | Paginated, newest first |
| GET | `/api/blocks/:height` | `BlockModel` | Full block with transactions |
| GET | `/api/blocks/hash/:hash` | `BlockModel` | Lookup by hash |
| GET | `/api/tx/:hash` | `TxModel` + block context | Scan blocks for tx |
| GET | `/api/mempool` | `[]TxModel` | Current pending transactions |
| GET | `/api/address/:addr` | Balance + UTXOs | Uses `UTXOSet.GetByAddress()` |
| GET | `/api/search?q=...` | Redirect or result | Detect hash length, try block/tx/address |

### WebSocket Events
| Event Type | Payload | Trigger |
|------------|---------|---------|
| `new_block` | `BlockModel` | Block mined locally or received from peer |
| `new_tx` | `TxModel` | Transaction added to mempool |
| `mining_progress` | `MiningProgressPayload` | Every 5000 nonce attempts during mining |
| `mining_started` | `{address, block_height}` | Mining begins for new block |
| `mining_stopped` | `{block_height, hash}` | Block found or mining cancelled |
| `peer_connected` | `{addr, height}` | New P2P peer connected |
| `peer_disconnected` | `{addr}` | P2P peer disconnected |
| `mempool_changed` | `{count, added?, removed?}` | Mempool size changed |
| `status` | `StatusResponse` | Periodic (every 5s) or on any state change |

## Dashboard Layout (Claude's Discretion Recommendation)

**Recommendation:** Multi-page SPA with sidebar navigation.

### Pages
1. **Dashboard** (`/`) -- Overview: chain height, peer count, mempool size, recent blocks, mining status
2. **Block Explorer** (`/blocks`) -- Paginated block list, click to view block detail
3. **Block Detail** (`/blocks/:height`) -- Full block with header fields and transaction list
4. **Transaction Detail** (`/tx/:hash`) -- Transaction inputs/outputs, block context
5. **Mempool** (`/mempool`) -- Live pending transactions table
6. **Mining** (`/mining`) -- Real-time mining visualization: nonce counter, hash display, target comparison
7. **Address** (`/address/:addr`) -- Balance, UTXO list

### UI Components
- **BlockCard** -- Compact block summary (height, hash prefix, tx count, timestamp)
- **TxTable** -- Transaction list with ID, inputs/outputs summary, value
- **StatusBar** -- Fixed top bar showing chain height, peers, mempool count
- **MiningVisualizer** -- Live nonce/hash/target display with progress animation
- **SearchBar** -- Universal search in header, auto-detects block hash/tx hash/address

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| gorilla/websocket archived | gorilla/websocket revived, actively maintained | 2023 | Safe to use, no longer archived |
| Tailwind v3 with PostCSS | Tailwind v4 with @tailwindcss/vite plugin | 2025 | Simpler setup, CSS-first config |
| Create React App | Vite | 2022+ | CRA deprecated, Vite is standard |
| shadcn/ui with Tailwind v3 | shadcn/ui supports Tailwind v4 | 2025 | New projects use v4 by default |

## Validation Architecture

### Test Framework
| Property | Value |
|----------|-------|
| Framework | Go testing + testify (backend), Vitest (frontend) |
| Config file | None for frontend yet -- Wave 0 |
| Quick run command | `go test ./internal/handler/api/... ./internal/handler/ws/...` |
| Full suite command | `go test ./...` |

### Phase Requirements to Test Map
| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| DASH-01 | Block explorer REST endpoints return correct block/tx data | unit | `go test ./internal/handler/api/ -run TestBlock -x` | Wave 0 |
| DASH-02 | Status endpoint returns chain height, peers, mempool | unit | `go test ./internal/handler/api/ -run TestStatus -x` | Wave 0 |
| DASH-03 | Mining progress callback fires at sample rate | unit | `go test ./internal/domain/block/ -run TestMineWithProgress -x` | Wave 0 |
| DASH-04 | Mempool endpoint returns pending transactions | unit | `go test ./internal/handler/api/ -run TestMempool -x` | Wave 0 |
| DASH-05 | Search endpoint resolves block hash, tx hash, address | unit | `go test ./internal/handler/api/ -run TestSearch -x` | Wave 0 |

### Sampling Rate
- **Per task commit:** `go test ./internal/handler/api/... ./internal/handler/ws/... -x`
- **Per wave merge:** `go test ./...`
- **Phase gate:** Full suite green before `/gsd:verify-work`

### Wave 0 Gaps
- [ ] `internal/handler/api/*_test.go` -- REST handler tests with mock ServiceContext
- [ ] `internal/handler/ws/hub_test.go` -- Hub register/unregister/broadcast tests
- [ ] `internal/domain/block/pow_test.go` -- Add `TestMineWithProgress` test
- [ ] `web/vitest.config.ts` -- Frontend test configuration (if testing frontend)
- [ ] Framework install: `npm install -D vitest @testing-library/react` (frontend, optional)

## Open Questions

1. **HTTP Server Lifecycle in startnode**
   - What we know: `startnode` currently only starts the P2P server. The HTTP server (go-zero REST) needs to start alongside it.
   - What's unclear: Whether to start the HTTP server unconditionally or only when `startnode` is used. The config already has `Host: 0.0.0.0, Port: 8080`.
   - Recommendation: Start HTTP server in `startnode` command only. Other commands (mine, send) are one-shot and don't need a dashboard. The HTTP port comes from `rest.RestConf` (already configured at 8080).

2. **P2P Server Access from API Handlers**
   - What we know: `ServiceContext` does not currently hold the P2P `Server`. It's created in `cli.startNode()`.
   - What's unclear: How to give API handlers access to peer count and peer info.
   - Recommendation: Either add a `P2PServer *p2p.Server` field to ServiceContext (set during startnode), or pass peer info through the event bus. Adding to ServiceContext is simpler.

3. **Transaction Search Without Index**
   - What we know: No tx-hash-to-block index exists. Must scan blocks.
   - What's unclear: Performance for long chains.
   - Recommendation: Scan from tip backwards, stop on first match. For an educational project with short chains, this is fine. Document as a known limitation.

## Sources

### Primary (HIGH confidence)
- Project codebase: `internal/config/config.go` -- rest.RestConf already embedded
- Project codebase: `internal/infrastructure/persistence/bbolt/storage_model.go` -- BlockModel/TxModel JSON models
- Project codebase: `internal/domain/p2p/server.go` -- Server structure, PeerCount(), OnBlockReceived()
- Project codebase: `internal/domain/block/pow.go` -- Mining loop structure for progress callback
- Project codebase: `internal/domain/mempool/mempool.go` -- Thread-safe mempool with Transactions()
- [gorilla/websocket GitHub](https://github.com/gorilla/websocket) -- Chat example hub pattern
- [gorilla/websocket pkg.go.dev](https://pkg.go.dev/github.com/gorilla/websocket) -- API documentation

### Secondary (MEDIUM confidence)
- [go-zero SSE docs](https://go-zero.dev/en/docs/tutorials/http/server/sse) -- Confirms rest.WithTimeout(0) pattern for long-lived connections
- [shadcn/ui Vite installation](https://ui.shadcn.com/docs/installation/vite) -- Setup instructions
- [shadcn/ui Tailwind v4](https://ui.shadcn.com/docs/tailwind-v4) -- Tailwind v4 compatibility

### Tertiary (LOW confidence)
- [WebSocket hub broadcast patterns](https://leapcell.io/blog/real-time-communication-with-gorilla-websocket-in-go-applications) -- Community article on hub pattern

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH - All libraries are user-locked or well-established, go-zero REST already configured in project
- Architecture: HIGH - Codebase thoroughly reviewed, existing patterns (BlockModel, ServiceContext, event callbacks) directly support the architecture
- Pitfalls: HIGH - Based on direct codebase analysis (go-zero timeout, []any type assertions, thread safety) and established gorilla/websocket patterns
- API design: HIGH - REST endpoints map directly to existing domain methods (GetBlock, GetBlockByHeight, GetByAddress, etc.)

**Research date:** 2026-03-06
**Valid until:** 2026-04-06 (30 days -- stable domain, no fast-moving dependencies)
