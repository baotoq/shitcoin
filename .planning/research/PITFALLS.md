# Domain Pitfalls

**Domain:** Adding comprehensive test coverage to an existing Go blockchain application
**Researched:** 2026-03-08
**Codebase:** shitcoin -- Go 1.26.1, BoltDB, TCP P2P, WebSocket, ECDSA crypto, goroutine-based mining

---

## Critical Pitfalls

Mistakes that cause flaky tests, CI failures, or false confidence in test coverage.

### Pitfall 1: BoltDB Single-Writer Lock Causing Test Deadlocks

**What goes wrong:** BoltDB allows only one read-write transaction at a time. Tests that open a BoltDB file and attempt concurrent writes (or forget to close the DB) will deadlock or timeout. Running multiple BoltDB tests in parallel against the same file will block indefinitely.

**Why it happens:** BoltDB uses an exclusive file lock for write transactions. Unlike SQL databases, there is no connection pool -- a single `bolt.Open()` holds the lock. If a test leaks an open DB handle, subsequent tests in the same suite hang.

**Consequences:** Tests hang in CI with no error message. Suite appears to "freeze" after N tests. Hard to debug because the deadlock is silent.

**Prevention:**
- Always use `t.TempDir()` for each test (the existing codebase does this correctly in `ChainRepoSuite.SetupTest`)
- Always register `t.Cleanup(func() { db.Close() })` immediately after `bolt.Open()`
- Never share a BoltDB file between parallel subtests
- Use testify `suite.Suite` with `SetupTest` (not `SetupSuite`) so each test gets a fresh DB -- already the pattern in `bbolt/chain_repo_test.go`
- If running `t.Parallel()` on BoltDB tests, each must have its own temp directory

**Detection:** Test suite hangs with no output. `go test -timeout 30s` will surface the issue as a timeout rather than infinite hang.

**Phase relevance:** Infrastructure/persistence test phase. The existing pattern is correct -- maintain it when adding new BoltDB tests.

### Pitfall 2: TCP Port Conflicts and Leaked Listeners in P2P Tests

**What goes wrong:** P2P tests that bind to hardcoded ports (e.g., `:3000`) will fail when run in parallel or when a previous test leaked a listener. Tests become order-dependent and fail unpredictably in CI.

**Why it happens:** TCP ports are a shared OS resource. If `srv.Stop()` is not called (or panics before cleanup), the port remains bound until the process exits. Running `go test -count=2` or parallel packages will conflict.

**Consequences:** "bind: address already in use" errors. Tests pass locally but fail in CI. Flaky test suites that pass on retry.

**Prevention:**
- Always use port 0 for OS-assigned ephemeral ports (the existing `makeTestServer` already does this correctly: `p2p.NewServer(ch, pool, nil, repo, 0)`)
- Always register `t.Cleanup(func() { srv.Stop() })` -- already done in the codebase
- Never hardcode ports in test code
- Use `srv.ListenAddr()` to discover the assigned port
- Set connection deadlines in test clients (`conn.SetDeadline`) to avoid hanging on broken connections

**Detection:** "address already in use" in test output. Tests that pass individually but fail when run together.

**Phase relevance:** P2P networking test phase.

### Pitfall 3: time.Sleep-Based Synchronization Causing Flaky Tests

**What goes wrong:** Tests that use `time.Sleep` to wait for goroutines (P2P handshakes, WebSocket message delivery, block propagation) are inherently flaky. Under CI load or on slow machines, the sleep duration may be insufficient.

**Why it happens:** Goroutine scheduling is non-deterministic. A `time.Sleep(200 * time.Millisecond)` that works on a fast laptop may fail on a loaded CI runner. The existing codebase has multiple instances of this pattern.

**Consequences:** Tests pass locally, fail intermittently in CI. Developers add longer sleeps, making the test suite slow. Eventually tests are marked as "flaky" and ignored.

**Existing instances in codebase:**
- `hub_test.go`: `time.Sleep(10 * time.Millisecond)` after channel operations
- `server_test.go`: `time.Sleep(200 * time.Millisecond)` after handshake
- `relay_test.go`: `time.Sleep(200 * time.Millisecond)` in `connectNodes`
- `sync_test.go`: `time.Sleep(3 * time.Second)` in `TestIBDSyncingFlag`

**Prevention:**
- Use polling loops with deadlines instead of fixed sleeps (the IBD tests already do this well with `time.After` + select)
- For channel-based operations (WebSocket hub), use `select` with timeout instead of sleep-then-check
- For P2P assertions, poll `PeerCount()` or `Height()` with a short interval and hard deadline
- Use `require.Eventually` or `assert.Eventually` from testify for condition-based waiting
- Pattern to follow (from `sync_test.go`):
  ```go
  deadline := time.After(5 * time.Second)
  for {
      select {
      case <-deadline:
          t.Fatal("timed out")
      default:
          if condition() { return }
          time.Sleep(50 * time.Millisecond)
      }
  }
  ```

**Detection:** Tests that fail intermittently. `go test -count=10` will surface flaky tests.

**Phase relevance:** All phases with async behavior -- P2P, WebSocket, mining. Refactor existing sleep-based tests when touching those files.

### Pitfall 4: Goroutine Leaks from P2P Servers and WebSocket Hubs

**What goes wrong:** Tests that start P2P servers or WebSocket hubs without proper cleanup leak goroutines. Over a test suite run, leaked goroutines accumulate, consuming resources and potentially interfering with later tests.

**Why it happens:** `p2p.Server.Start()` spawns listener goroutines. `ws.Hub` runs a background goroutine for register/unregister/broadcast. If `Stop()` is not called or a test panics before cleanup, these goroutines persist.

**Consequences:** `go test -race` reports data races from leaked goroutines. Memory usage grows during test suite. Goroutines from one test interfere with assertions in another.

**Prevention:**
- Always use `t.Cleanup()` for server shutdown (already done in P2P tests)
- Consider using `goleak` (`go.uber.org/goleak`) to detect goroutine leaks at test boundaries
- For the WebSocket hub: ensure hub goroutines are stoppable via context cancellation or a quit channel
- The current `ws.Hub` tests send to channels but never explicitly stop the hub -- this is a leak. Add a `Stop()` method or use context-based cancellation

**Detection:** Add `goleak.VerifyNone(t)` to critical test functions. Run tests with `-race` flag.

**Phase relevance:** P2P and WebSocket test phases. Fix the WebSocket hub leak early.

### Pitfall 5: Mining in Tests Taking Too Long or Being Non-Deterministic

**What goes wrong:** Tests that mine blocks with realistic difficulty take unpredictable time. A test that mines 5 blocks at difficulty 16 might take 50ms or 5 seconds depending on hash luck. Tests become slow and flaky.

**Why it happens:** PoW mining is intentionally random. The number of nonce iterations to find a valid hash varies. The existing tests use `InitialDifficulty: 1` (target = 2^255) which is extremely easy, but some test helper functions might use higher difficulty.

**Consequences:** Slow test suite (minutes instead of seconds). Timeout failures in CI. Tests that pass 99% of the time but occasionally hit an unlucky hash sequence.

**Prevention:**
- Always use difficulty 1 (or the minimum) in test configurations -- the existing codebase does this well
- Use `MineWithMaxNonce` for tests that verify mining failure, with a small nonce limit
- Never use production difficulty values in tests
- Consider a test-only "instant mine" that sets nonce=0 if hash happens to be valid, or a mock miner that skips PoW entirely for non-mining tests
- For tests that need mined blocks but don't test mining itself, create pre-mined fixtures

**Detection:** `go test -v -timeout 10s ./internal/domain/block/` -- if it times out, difficulty is too high.

**Phase relevance:** All phases that need test blocks. Establish the pattern in the domain test phase.

---

## Moderate Pitfalls

### Pitfall 6: Duplicated Mock Implementations Across Test Packages

**What goes wrong:** Each test package re-implements the same mock repositories. The codebase already has 4 separate `mockChainRepo` implementations (in `chain_test`, `p2p_test/relay_test`, `p2p_test/server_test`, `api/block_handler_test`), plus 3 separate `mockUTXORepo` implementations. When the repository interface changes, all mocks must be updated.

**Why it happens:** Go test files in different packages cannot share test helpers directly. Developers copy-paste mock implementations rather than creating a shared test support package.

**Consequences:** Interface changes require updating N mock implementations. Mocks drift out of sync. Some mocks implement methods incorrectly or incompletely (e.g., `relay_test.go` mock's `utxoKey` uses `string(rune(vout+'0'))` which only works for single-digit vout values).

**Prevention:**
- Create an `internal/testutil/` package with shared mock implementations
- Use `mockery` or `moq` to auto-generate mocks from interfaces
- Alternatively, use the `internal/domain/*/mock_test.go` convention within each package and share via build tags
- The `server_test.go` mock uses `testify/mock` while `relay_test.go` uses a hand-rolled implementation -- pick one approach and standardize

**Detection:** `grep -rn "mockChainRepo\|MockChainRepo" internal/` shows the duplication.

**Phase relevance:** Address in the first test phase. Create shared test infrastructure before writing new tests.

### Pitfall 7: Race Conditions in Concurrent Test Assertions

**What goes wrong:** Tests that read shared state (chain height, peer count, UTXO balances) from goroutines without proper synchronization trigger data races. The `-race` detector catches these.

**Why it happens:** P2P and mining operations modify shared state from background goroutines. Test assertions that read this state without the same locks used by production code will race.

**Consequences:** `-race` failures in CI. Incorrect test results (reading stale values). Tests that pass without `-race` but fail with it.

**Prevention:**
- Always run tests with `go test -race ./...`
- Mock repositories must use `sync.Mutex` for thread safety (the existing `fullMockChainRepo` in `relay_test.go` correctly uses `sync.RWMutex`)
- Use exported thread-safe accessors (e.g., `ch.Height()`) rather than directly reading struct fields
- For tests checking eventually-consistent state, use polling with `assert.Eventually`

**Detection:** `go test -race ./...` -- make this a CI requirement.

**Phase relevance:** All phases. Run `-race` from day one.

### Pitfall 8: Testing []any Transaction Slices Without Type Assertions

**What goes wrong:** Block transactions are stored as `[]any` to avoid import cycles. Tests that create blocks with transactions must handle the type assertion from `any` to `*tx.Transaction`. Forgetting this causes nil pointer dereferences or incorrect test data.

**Why it happens:** The `block.Block.RawTransactions()` returns `[]any`. Tests must assert to `*tx.Transaction` before accessing transaction methods. This is a deliberate design decision to break import cycles but adds friction in tests.

**Consequences:** Panic in tests when type assertion fails. Tests that silently test empty transaction lists. Missing coverage for transaction-related block validation.

**Prevention:**
- Create test helpers that handle the type assertion:
  ```go
  func extractTx(t *testing.T, b *block.Block, idx int) *tx.Transaction {
      t.Helper()
      raw := b.RawTransactions()
      require.Greater(t, len(raw), idx)
      txn, ok := raw[idx].(*tx.Transaction)
      require.True(t, ok, "expected *tx.Transaction at index %d", idx)
      return txn
  }
  ```
- The existing `chain_test.go:TestCoinbaseIncludesFees` shows the correct pattern
- Always test the type assertion explicitly, not with a bare comma-ok

**Detection:** Code review. Tests that create blocks with `nil` transaction slices when they should have transactions.

**Phase relevance:** Domain layer tests, particularly chain and block phases.

### Pitfall 9: WebSocket Tests That Depend on Internal Channel Mechanics

**What goes wrong:** The current WebSocket hub tests directly access `hub.register`, `hub.unregister`, and `hub.broadcast` channels, plus `hub.mu` (the internal mutex). This couples tests to the implementation, not the behavior. Any refactoring of the hub internals breaks all tests.

**Why it happens:** The hub uses unexported channels for goroutine communication. Tests in the same package can access these, but doing so means tests test "how" not "what."

**Consequences:** Refactoring the hub (e.g., switching from channels to a different concurrency pattern) requires rewriting all tests. Tests don't verify actual WebSocket behavior (message framing, connection lifecycle).

**Prevention:**
- Test the hub through its public API or through actual WebSocket connections using `httptest.NewServer` + `gorilla/websocket.Dialer`
- For unit tests, expose a `Hub.Register(client)` method rather than testing channel sends directly
- For integration tests, use a real HTTP test server with WebSocket upgrade
- Keep the current channel-based tests as implementation tests, but add behavior-based tests on top

**Detection:** Tests that reference unexported fields (`hub.register`, `hub.mu`).

**Phase relevance:** WebSocket/API handler test phase.

### Pitfall 10: ECDSA Key Generation Making Tests Non-Deterministic

**What goes wrong:** Tests that generate ECDSA keys with `btcec.NewPrivateKey()` get random keys each run. This makes test failures hard to reproduce. If a test depends on key ordering or address values, it may be flaky.

**Why it happens:** `btcec.NewPrivateKey()` uses `crypto/rand`, producing different keys each run. The existing mempool and transaction tests use this correctly (they don't depend on specific key values), but new tests might.

**Consequences:** Non-reproducible failures. Tests that depend on address ordering or specific hash values break intermittently.

**Prevention:**
- For tests that need deterministic keys, derive from a fixed seed:
  ```go
  func fixedTestKey(t *testing.T) *btcec.PrivateKey {
      t.Helper()
      seed := sha256.Sum256([]byte("test-key-seed"))
      key, _ := btcec.PrivKeyFromBytes(seed[:])
      return key
  }
  ```
- For tests that don't care about specific keys (most tests), random is fine
- Never assert on specific address strings generated from random keys

**Detection:** Tests that fail on some runs but not others, with no obvious timing component.

**Phase relevance:** Domain layer (tx, wallet, mempool) test phases.

---

## Minor Pitfalls

### Pitfall 11: Not Testing Block Serialization Roundtrip

**What goes wrong:** The BoltDB repository serializes blocks to JSON (`BlockModel`) and deserializes them back. If a field is added to the domain model but not to the storage model, data is silently lost.

**Why it happens:** The storage model (`bbolt.BlockModel`) is a separate struct from the domain model (`block.Block`). Fields must be explicitly mapped in both directions.

**Prevention:**
- Write roundtrip tests: create domain object -> serialize to model -> deserialize back -> compare all fields
- The existing `TestStorageModelRoundTrip` in `utxo_repo_test.go` is a good pattern to follow
- Add similar roundtrip tests for `BlockModel`

**Phase relevance:** Infrastructure persistence test phase.

### Pitfall 12: Test Helpers That Swallow Errors

**What goes wrong:** Test helper functions that use `_` to ignore errors (e.g., `_, _ = utxoSet.ApplyBlock(...)`) hide bugs in test setup. If the setup fails silently, the test may pass for the wrong reason.

**Why it happens:** Convenience. Developers want concise test setup and ignore "unimportant" errors.

**Existing instances:**
- `mempool_test.go` line 164: `_, _ = utxoSet.ApplyBlock(0, []*tx.Transaction{coinbase})`
- `mempool_test.go` line 146: `_ = tx.SignTransaction(spendTx1, privKey)`

**Prevention:**
- Always use `require.NoError(t, err)` in test helpers, even for setup code
- Mark helpers with `t.Helper()` so failures show the calling test line
- The `buildSignedTx` helper in `mempool_test.go` does this correctly -- follow that pattern

**Phase relevance:** All test phases. Audit existing tests during the first phase.

### Pitfall 13: Missing Edge Case Tests for Chain Reorganization

**What goes wrong:** Reorg tests only cover the happy path (longer chain wins). Edge cases like reorg at genesis, reorg with identical block hashes, or reorg during active mining are not tested.

**Why it happens:** Reorg is complex and the happy path is hard enough to test. Edge cases require careful chain construction.

**Prevention:**
- Test reorg at height 1 (just above genesis)
- Test reorg with transactions that reference UTXOs created in the to-be-undone blocks
- Test concurrent mining + reorg (mine in one goroutine, receive reorg blocks in another)
- Test reorg failure recovery (what happens if reorg fails midway?)

**Phase relevance:** Chain domain and P2P integration test phases.

### Pitfall 14: go-zero pathvar Dependency in Handler Tests

**What goes wrong:** API handler tests must use `pathvar.WithVars()` to inject URL path variables, coupling tests to go-zero's internal routing mechanism. If go-zero changes this API, all handler tests break.

**Why it happens:** go-zero does not use standard `http.ServeMux` patterns. Path variables are injected via context, not parsed from the URL.

**Prevention:**
- Accept the coupling for unit tests (it's the framework's testing pattern)
- For integration tests, use a full go-zero server with `httptest` rather than testing handlers in isolation
- Document the `pathvar.WithVars` pattern so all handler tests use it consistently

**Phase relevance:** API handler test phase.

---

## Phase-Specific Warnings

| Phase Topic | Likely Pitfall | Mitigation |
|-------------|---------------|------------|
| Domain unit tests (block, tx, utxo, wallet) | Mining difficulty too high in tests (#5) | Enforce difficulty=1 in all test configs |
| Domain unit tests (mempool) | Race conditions in concurrent tests (#7) | Run with `-race`, use mutex in mock repos |
| Chain aggregate tests | []any type assertions (#8), reorg edge cases (#13) | Create shared type-assertion helpers |
| P2P networking tests | Port conflicts (#2), sleep-based waits (#3), goroutine leaks (#4) | Use port 0, poll-with-deadline pattern, goleak |
| WebSocket tests | Goroutine leaks (#4), implementation coupling (#9) | Add Hub.Stop(), test via public API |
| API handler tests | go-zero pathvar coupling (#14), duplicated mocks (#6) | Document pattern, create shared test infra |
| BoltDB persistence tests | Single-writer deadlocks (#1), serialization gaps (#11) | Fresh DB per test, roundtrip tests |
| Integration tests | All of the above, compounded | Start with the polling pattern, shared mocks, `-race` |
| Test infrastructure setup | Mock duplication (#6), error swallowing (#12) | Create `internal/testutil/` first |

---

## Recommended Test Phase Ordering (Pitfall-Aware)

1. **Test infrastructure first** -- Create shared mocks, test helpers, type-assertion utilities (#6, #8, #12)
2. **Pure domain logic** -- block, tx, wallet, utxo (no I/O, no goroutines, fast) (#5, #10)
3. **Chain aggregate** -- requires mocks but no networking (#5, #8, #13)
4. **BoltDB persistence** -- I/O but no networking (#1, #11)
5. **P2P networking** -- goroutines, TCP, most pitfall-prone (#2, #3, #4, #7)
6. **WebSocket + API handlers** -- depends on domain being well-tested (#4, #9, #14)
7. **Integration tests** -- combine layers, highest risk of flaky tests

This ordering ensures foundational pitfalls are addressed before they compound in integration tests.

## Sources

- Direct codebase analysis of 23 existing test files in `/Users/baotoq/Work/shitcoin/`
- BoltDB documentation on transaction locking: HIGH confidence (verified via codebase behavior)
- Go testing best practices (`testing.T.Cleanup`, `-race`, `t.TempDir()`): HIGH confidence (stdlib)
- testify `assert.Eventually` pattern: HIGH confidence (widely documented)
- goleak for goroutine leak detection: HIGH confidence (Uber open-source, well-maintained)
