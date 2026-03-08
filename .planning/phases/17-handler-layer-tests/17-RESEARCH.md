# Phase 17: Handler Layer Tests - Research

**Researched:** 2026-03-08
**Domain:** Go HTTP handler testing (net/http/httptest), WebSocket testing (gorilla/websocket), go-zero pathvar
**Confidence:** HIGH

## Summary

Phase 17 requires bringing API handler coverage from 41.3% to 80%+ and WebSocket hub coverage from 35.1% to 75%+. The existing test infrastructure (Phase 14) provides MockChainRepo, MockUTXORepo, and testutil builders that already work well in the existing handler tests. The established patterns in `block_handler_test.go` and `status_handler_test.go` demonstrate the exact approach: construct `svc.ServiceContext` with mock repos, create `httptest.NewRequest`, inject path vars via `pathvar.WithVars`, call handler directly, and assert on `httptest.ResponseRecorder`.

For WebSocket testing, the existing `hub_test.go` tests the hub's register/unregister/broadcast channels directly. To reach 75%, the `ServeWs` handler and client read/write pumps need testing with `httptest.Server` + `gorilla/websocket.Dial`.

**Primary recommendation:** Follow the established patterns exactly. Add tests for the 5 uncovered/partially-covered handlers (AddressHandler, BlockByHashHandler, SearchHandler, MempoolHandler with data, BlocksHandler edge cases). For WebSocket, test ServeWs via httptest.Server with real WebSocket connections.

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|-----------------|
| HNDL-01 | API handler test coverage reaches 80%+ (address, mempool, search, tx handlers) | Coverage gap analysis shows AddressHandler at 0%, SearchHandler at 0%, BlockByHashHandler at 0%, MempoolHandler at 83.3% (needs data test), BlocksHandler at 68.8% (needs edge cases). Existing test patterns with httptest + pathvar.WithVars cover all cases. |
| HNDL-02 | WebSocket hub test coverage reaches 75%+ (event subscribe, broadcast, client disconnect) | Hub at 100% for Run/NewHub but ServeWs at 0%, client readPump/writePump at 0%. Need httptest.Server with real WebSocket connections. subscribeEventBus at 75% (marshal error path uncovered). |
</phase_requirements>

## Standard Stack

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| net/http/httptest | stdlib | HTTP test server and recorder | Go standard for handler testing |
| github.com/zeromicro/go-zero/rest/pathvar | v1.10.0 | Inject URL path variables in tests | Project's routing framework; `pathvar.WithVars` is the only way to set path params |
| github.com/zeromicro/go-zero/rest/httpx | v1.10.0 | JSON response helpers used in handlers | Already used in production code |
| github.com/stretchr/testify | existing | assert/require for test assertions | Already used across all test files |
| github.com/gorilla/websocket | v1.5.3 | WebSocket client for testing ServeWs | Already a dependency; provides `Dialer` for test clients |

### Supporting
| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| internal/testutil | project | MockChainRepo, MockUTXORepo, MustCreateBlock builders | All handler tests needing chain/UTXO data |
| encoding/json | stdlib | Unmarshal response bodies | Asserting JSON response structure |

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| pathvar.WithVars | Custom router setup | pathvar.WithVars is simpler, already used in existing tests |
| Direct hub channel sends | httptest.Server + WS dial | Channel sends test hub logic; WS dial tests ServeWs + client lifecycle |

## Architecture Patterns

### Test File Organization
```
internal/handler/api/
  address_handler_test.go    # NEW - AddressHandler tests
  block_handler_test.go      # EXISTING - add BlockByHashHandler + edge cases
  mempool_handler_test.go    # NEW - MempoolHandler with transaction data
  search_handler_test.go     # NEW - SearchHandler all branches
  status_handler_test.go     # EXISTING - already has good coverage
  tx_handler_test.go         # NEW or move from status_handler_test.go

internal/handler/ws/
  hub_test.go                # EXISTING - add subscribeEventBus error path
  handler_test.go            # NEW - ServeWs with httptest.Server + WS dial
```

### Pattern 1: API Handler Test (Established)
**What:** Construct ServiceContext with mocks, call handler function directly, assert response
**When to use:** All API handler tests
**Example:**
```go
// From existing block_handler_test.go
func TestBlockByHeightHandler_ValidHeight(t *testing.T) {
    repo := testutil.NewMockChainRepo()
    genesis := testutil.MustCreateBlock(t, 0, block.Hash{})
    repo.AddBlock(genesis)

    svcCtx := &svc.ServiceContext{ChainRepo: repo}
    handler := BlockByHeightHandler(svcCtx)

    req := httptest.NewRequest(http.MethodGet, "/api/blocks/0", nil)
    req = pathvar.WithVars(req, map[string]string{"height": "0"})
    w := httptest.NewRecorder()
    handler(w, req)

    assert.Equal(t, http.StatusOK, w.Code)
    var resp bbolt.BlockModel
    require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
    assert.Equal(t, uint64(0), resp.Height)
}
```

### Pattern 2: WebSocket Integration Test
**What:** Start httptest.Server, dial WebSocket, verify messages
**When to use:** Testing ServeWs handler and client lifecycle
**Example:**
```go
func TestServeWs_ClientReceivesBroadcast(t *testing.T) {
    bus := events.NewBus()
    hub := NewHub(bus)

    server := httptest.NewServer(ServeWs(hub))
    defer server.Close()

    // Convert http URL to ws URL
    wsURL := "ws" + strings.TrimPrefix(server.URL, "http")

    conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
    require.NoError(t, err)
    defer conn.Close()

    // Publish event and verify client receives it
    bus.Publish(events.Event{Type: events.EventNewBlock, Payload: map[string]string{"hash": "abc"}})

    conn.SetReadDeadline(time.Now().Add(time.Second))
    _, msg, err := conn.ReadMessage()
    require.NoError(t, err)

    var wsMsg WSMessage
    require.NoError(t, json.Unmarshal(msg, &wsMsg))
    assert.Equal(t, "new_block", wsMsg.Type)
}
```

### Pattern 3: AddressHandler with Mock UTXO Set
**What:** AddressHandler uses `svcCtx.UTXOSet.GetByAddress()` which delegates to repo
**When to use:** Testing AddressHandler
**Key insight:** ServiceContext.UTXOSet is `*utxo.Set` which wraps a `utxo.Repository`. Create `utxo.NewSet(testutil.NewMockUTXORepo())` and populate the mock repo with UTXOs.
```go
func TestAddressHandler_WithUTXOs(t *testing.T) {
    utxoRepo := testutil.NewMockUTXORepo()
    utxoSet := utxo.NewSet(utxoRepo)

    // Add UTXO for test address
    u := utxo.NewUTXO(block.Hash{1}, 0, 5000, "1TestAddr")
    utxoRepo.Put(u)

    svcCtx := &svc.ServiceContext{UTXOSet: utxoSet}
    handler := AddressHandler(svcCtx)
    // ... httptest request with pathvar
}
```

### Anti-Patterns to Avoid
- **Using real Chain aggregate when only ChainRepo is needed:** Several handlers only use `svcCtx.ChainRepo` (BlocksHandler, BlockByHeightHandler, BlockByHashHandler). Only StatusHandler and TxHandler also need `svcCtx.Chain` for Height()/LatestBlock(). Don't initialize a full Chain when you only need the repo.
- **time.Sleep for hub synchronization:** Existing hub_test.go uses `time.Sleep(10ms)`. Use `require.Eventually` instead (per STATE.md decision from Phase 15).
- **Testing RegisterRoutes:** This is pure wiring code (listed in REQUIREMENTS.md Out of Scope). Don't test it.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Path variable injection | Custom context/mux setup | `pathvar.WithVars(req, map)` | go-zero's built-in test support |
| Mock chain data | Manual block construction | `testutil.MustCreateBlock(t, height, prevHash)` | Handles coinbase, mining, merkle root |
| Mock UTXO data | Raw map manipulation | `testutil.NewMockUTXORepo()` + `utxo.NewSet(repo)` | Thread-safe, correct interface |
| WebSocket test client | Raw TCP + upgrade | `websocket.DefaultDialer.Dial` on httptest.Server URL | Handles upgrade, framing |

## Common Pitfalls

### Pitfall 1: AddressHandler imports bbolt models
**What goes wrong:** AddressHandler converts domain UTXOs to `bbolt.UTXOModel` via `bbolt.UTXOModelFromDomain()`. Tests must import the bbolt package for response types.
**Why it happens:** Handler layer couples to infrastructure model types for JSON serialization.
**How to avoid:** Accept the import in tests. This is a design choice, not a test problem.
**Warning signs:** Trying to avoid bbolt import leads to duplicated type definitions.

### Pitfall 2: SearchHandler has many branches
**What goes wrong:** SearchHandler detects query type (hex hash, numeric height, address format) with 6+ code paths. Missing any branch leaves coverage gaps.
**Why it happens:** Complex routing logic in single handler.
**How to avoid:** Use table-driven tests covering: empty query, 64-char hex matching block hash, 64-char hex matching tx hash, 64-char hex not found, numeric height found, numeric height not found, address-like string, unrecognized string.
**Warning signs:** Coverage stays below target despite multiple tests.

### Pitfall 3: Hub.Run() goroutine leak
**What goes wrong:** `NewHub()` starts a goroutine via `go h.Run()` that never stops (infinite select loop). STATE.md notes: "WebSocket hub lacks Stop() -- may need small production code change for test cleanup."
**Why it happens:** Hub was designed for application lifetime, not test lifetime.
**How to avoid:** For hub_test.go tests, the goroutine leak is acceptable (tests finish quickly). For ServeWs tests with httptest.Server, the leak is also acceptable since `server.Close()` will clean up. If the leak causes test warnings, add a context-based shutdown to Hub.Run().
**Warning signs:** `go test -race` warnings about goroutine leaks.

### Pitfall 4: WebSocket client readPump/writePump are hard to unit test
**What goes wrong:** readPump and writePump have complex timer-based logic (pingPeriod, pongWait, writeWait).
**Why it happens:** They are designed for long-lived connections with keepalive.
**How to avoid:** Test via integration (ServeWs + httptest.Server + real WS connection). The basic flow (connect, receive message, disconnect) exercises the critical paths. Don't try to test ping/pong timing in unit tests -- that's testing gorilla/websocket, not our code.
**Warning signs:** Tests with long timeouts or flaky timing assertions.

### Pitfall 5: MempoolHandler needs real Mempool with transactions
**What goes wrong:** Current test only covers empty mempool. To test with data, need to add transactions to mempool, but mempool.New() takes an optional UTXOSet for validation.
**How to avoid:** Use `mempool.New(nil)` to skip validation, then directly add transactions. Check if Mempool has an AddWithoutValidation method, or if Add() works with nil UTXOSet for coinbase txs.
**Warning signs:** Panics when adding tx to mempool with nil UTXOSet.

## Code Examples

### SearchHandler Table-Driven Tests
```go
func TestSearchHandler_EmptyQuery(t *testing.T) {
    svcCtx := &svc.ServiceContext{}
    handler := SearchHandler(svcCtx)

    req := httptest.NewRequest(http.MethodGet, "/api/search", nil) // no ?q=
    w := httptest.NewRecorder()
    handler(w, req)

    assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestSearchHandler_BlockByHash(t *testing.T) {
    repo := testutil.NewMockChainRepo()
    genesis := testutil.MustCreateBlock(t, 0, block.Hash{})
    repo.AddBlock(genesis)

    pow := &block.ProofOfWork{}
    ch := chain.NewChain(repo, pow, chain.ChainConfig{InitialDifficulty: 1}, nil)
    require.NoError(t, ch.Initialize(t.Context(), ""))

    svcCtx := &svc.ServiceContext{Chain: ch, ChainRepo: repo}
    handler := SearchHandler(svcCtx)

    hashStr := genesis.Hash().String()
    req := httptest.NewRequest(http.MethodGet, "/api/search?q="+hashStr, nil)
    w := httptest.NewRecorder()
    handler(w, req)

    assert.Equal(t, http.StatusOK, w.Code)
    var result SearchResult
    require.NoError(t, json.Unmarshal(w.Body.Bytes(), &result))
    assert.Equal(t, "block", result.Type)
}
```

### BlockByHashHandler Test
```go
func TestBlockByHashHandler_ValidHash(t *testing.T) {
    repo := testutil.NewMockChainRepo()
    genesis := testutil.MustCreateBlock(t, 0, block.Hash{})
    repo.AddBlock(genesis)

    svcCtx := &svc.ServiceContext{ChainRepo: repo}
    handler := BlockByHashHandler(svcCtx)

    hashStr := genesis.Hash().String()
    req := httptest.NewRequest(http.MethodGet, "/api/blocks/hash/"+hashStr, nil)
    req = pathvar.WithVars(req, map[string]string{"hash": hashStr})
    w := httptest.NewRecorder()
    handler(w, req)

    assert.Equal(t, http.StatusOK, w.Code)
}

func TestBlockByHashHandler_InvalidHash(t *testing.T) {
    svcCtx := &svc.ServiceContext{ChainRepo: testutil.NewMockChainRepo()}
    handler := BlockByHashHandler(svcCtx)

    req := httptest.NewRequest(http.MethodGet, "/api/blocks/hash/notahex", nil)
    req = pathvar.WithVars(req, map[string]string{"hash": "notahex"})
    w := httptest.NewRecorder()
    handler(w, req)

    assert.Equal(t, http.StatusBadRequest, w.Code)
}
```

## Coverage Gap Analysis

### API Handlers (Current: 41.3%, Target: 80%)

| Handler | Current | Lines Missing | Tests Needed |
|---------|---------|---------------|--------------|
| AddressHandler | 0% | All (10 lines) | Happy path with UTXOs, empty address, error path |
| BlocksHandler | 68.8% | Error + offset>=total paths | GetChainHeight error, page beyond total, defaults |
| BlockByHeightHandler | 84.6% | Invalid height string | Already tested, minor gap |
| BlockByHashHandler | 0% | All (10 lines) | Valid hash, invalid hex, not found |
| MempoolHandler | 83.3% | Non-empty mempool path | Mempool with transactions |
| SearchHandler | 0% | All (28 lines) | Empty q, hex-block, hex-tx, hex-not-found, height, address, unknown |
| StatusHandler | 100% | None | Done |
| TxHandler | 88.9% | Minor gap | Already well tested |
| RegisterRoutes | 0% | N/A | Out of scope (wiring) |
| isLikelyAddress | 0% | All (3 lines) | Tested indirectly via SearchHandler |

### WebSocket (Current: 35.1%, Target: 75%)

| Function | Current | Tests Needed |
|----------|---------|--------------|
| NewHub | 100% | Done |
| Run | 100% | Done |
| subscribeEventBus | 75% | Marshal error path (unlikely to reach without complex setup) |
| ServeWs | 0% | httptest.Server + WS dial, verify client registration + message receipt |
| readPump | 0% | Tested via ServeWs integration (client disconnect triggers unregister) |
| writePump | 0% | Tested via ServeWs integration (receiving broadcast messages) |

**Reaching 75% WS coverage:** ServeWs (7 lines), readPump (14 lines), writePump (20 lines) = 41 uncovered lines out of ~49 total. Testing ServeWs with a real connection will cover ServeWs fully + the happy paths of readPump/writePump, getting to ~75%.

## Validation Architecture

### Test Framework
| Property | Value |
|----------|-------|
| Framework | Go testing + testify v1.10.0 |
| Config file | None needed (stdlib test runner) |
| Quick run command | `go test ./internal/handler/api/ ./internal/handler/ws/` |
| Full suite command | `go test -cover ./internal/handler/api/ ./internal/handler/ws/` |

### Phase Requirements to Test Map
| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| HNDL-01a | AddressHandler returns balance+UTXOs | unit | `go test ./internal/handler/api/ -run TestAddressHandler -v` | No - Wave 0 |
| HNDL-01b | BlockByHashHandler valid/invalid/notfound | unit | `go test ./internal/handler/api/ -run TestBlockByHashHandler -v` | No - Wave 0 |
| HNDL-01c | SearchHandler all query types | unit | `go test ./internal/handler/api/ -run TestSearchHandler -v` | No - Wave 0 |
| HNDL-01d | BlocksHandler edge cases | unit | `go test ./internal/handler/api/ -run TestBlocksHandler -v` | Partial |
| HNDL-01e | MempoolHandler with data | unit | `go test ./internal/handler/api/ -run TestMempoolHandler -v` | Partial |
| HNDL-02a | ServeWs client lifecycle | integration | `go test ./internal/handler/ws/ -run TestServeWs -v` | No - Wave 0 |
| HNDL-02b | Hub event bus error path | unit | `go test ./internal/handler/ws/ -run TestHub -v` | Partial |

### Sampling Rate
- **Per task commit:** `go test ./internal/handler/api/ ./internal/handler/ws/`
- **Per wave merge:** `go test -cover ./internal/handler/api/ ./internal/handler/ws/`
- **Phase gate:** API 80%+, WS 75%+

### Wave 0 Gaps
- [ ] `internal/handler/api/address_handler_test.go` -- covers HNDL-01a
- [ ] `internal/handler/api/search_handler_test.go` -- covers HNDL-01c
- [ ] `internal/handler/ws/handler_test.go` -- covers HNDL-02a

## Sources

### Primary (HIGH confidence)
- Project source code: `internal/handler/api/*.go`, `internal/handler/ws/*.go`
- Existing tests: `block_handler_test.go`, `status_handler_test.go`, `hub_test.go`
- `internal/testutil/` mock implementations
- `go test -cover` output for current coverage baselines

### Secondary (MEDIUM confidence)
- go-zero pathvar.WithVars usage verified in existing test code
- gorilla/websocket v1.5.3 Dialer API (well-established, stable)

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH - all tools already used in existing tests, no new dependencies
- Architecture: HIGH - patterns established in block_handler_test.go and status_handler_test.go
- Pitfalls: HIGH - derived from actual code analysis and STATE.md blockers

**Research date:** 2026-03-08
**Valid until:** 2026-04-08 (stable project, no external dependencies changing)
