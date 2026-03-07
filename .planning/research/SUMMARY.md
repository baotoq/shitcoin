# Project Research Summary

**Project:** shitcoin -- Testing & Quality Milestone (v1.2)
**Domain:** Go test coverage and quality infrastructure for an educational blockchain application
**Researched:** 2026-03-08
**Confidence:** HIGH

## Executive Summary

This milestone is about writing more tests with existing tools, not adding new dependencies. The Go blockchain application already has 23 test files with established patterns (table-driven tests, testify assertions, testify/suite for BoltDB, httptest for API handlers, net.Pipe for P2P). Coverage is uneven: domain packages range from 67-100%, but handler/api sits at 41%, handler/ws at 35%, handler/cli at 0%, and infrastructure/bbolt at 56%. The entire Go testing stack (stdlib testing, testify v1.11.1, gorilla/websocket v1.5.3) is already in go.mod. Zero new `go get` commands are required.

The recommended approach is to build a shared `internal/testutil/` package first (consolidating 4 duplicated mock implementations into one), then systematically fill coverage gaps layer by layer: domain logic first (pure, fast, no I/O), then infrastructure persistence (real BoltDB in TempDir), then handlers (httptest + mocks), and finally cross-cutting concerns (race detection, error paths). This ordering follows the dependency graph and ensures foundational test infrastructure is stable before higher-level tests depend on it.

The primary risks are flaky tests from `time.Sleep`-based synchronization (already present in 4+ test files), goroutine leaks from P2P servers and WebSocket hubs without proper cleanup, and BoltDB deadlocks from shared file handles. All three are mitigable with known patterns: polling with `require.Eventually`, `t.Cleanup()` for resource teardown, and `t.TempDir()` for DB isolation. Adding `-race` to CI is the single highest-impact change -- it catches concurrency bugs across the P2P, event bus, and WebSocket layers at near-zero cost.

## Key Findings

### Recommended Stack

No new dependencies. The existing stack covers all testing needs.

**Core technologies:**
- **Go stdlib `testing`**: Test runner, benchmarks, `t.TempDir()`, `t.Cleanup()` -- built-in, zero dependency
- **testify v1.11.1**: assert/require/mock/suite -- already used in all 23 test files, Go community standard
- **golangci-lint v2.10**: Static analysis -- already configured in CI with `.golangci.yml`
- **`net/http/httptest`**: HTTP handler testing -- stdlib, pattern already established in `block_handler_test.go`
- **gorilla/websocket v1.5.3**: WebSocket test client -- already a production dependency

**CI enhancements (flags only, no new tools):**
- Add `-race` flag to `go test` (catches concurrency bugs in P2P, event bus, WebSocket)
- Add `-covermode=atomic` (race-safe coverage)
- Add coverage threshold check at 70% (prevent regression)
- Add `-timeout 30s` per-package (surface deadlocks early)

### Expected Features

**Must have (table stakes):**
- Shared test helpers and fixtures (`internal/testutil/`) -- foundation for all other work
- API handler tests for all 8 endpoints (41.3% -> 80%+) -- 4 handler files completely untested
- WebSocket hub event broadcasting tests (34.7% -> 75%+) -- user-visible functionality
- BoltDB repository tests (55.7% -> 80%+) -- persistence correctness for data integrity
- Chain aggregate edge case tests (69.5% -> 85%+) -- mining, reorg, difficulty adjustment
- P2P message encoding and handler coverage (67.1% -> 80%+) -- wire protocol correctness
- Error path testing across all packages -- happy paths exist but error/edge cases underserved
- Race condition testing via `-race` flag -- concurrent code in P2P, mempool, WebSocket, mining

**Should have (differentiators):**
- P2P multi-node integration tests (in-process TCP, 2+ nodes)
- End-to-end chain scenario tests (create wallet -> send tx -> mine -> verify UTXO)
- UTXO undo/rollback integration tests (apply blocks, trigger reorg, verify rollback)
- Coverage enforcement in CI (per-package thresholds)
- CLI handler tests for command dispatch (0% -> 50%+)

**Defer (v2+):**
- Fuzz tests for deserialization (add after serialization paths well-tested)
- Golden file tests for wire protocol snapshots
- Benchmark tests for PoW mining
- Full CLI handler coverage for testnet/demo orchestration (high effort, low value)
- Property-based testing, mutation testing (overkill for educational project)

### Architecture Approach

Testing integrates with the existing DDD layers without modifying production code. The key architectural addition is a shared `internal/testutil/` package with consolidated mock repositories and test factories (`NewTestChain`, `MineBlocks`, `BuildSignedTx`). This eliminates ~400 lines of duplicated mock code across 4 packages and provides a consistent foundation. All repository interfaces already support dependency injection; zero production code changes are needed.

**Major components:**
1. **`internal/testutil/`** -- Shared test factories (chain, tx, utxo builders) and consolidated mock implementations for chain.Repository, utxo.Repository, wallet.Repository
2. **Domain unit tests** -- Pure logic tests using mock repos; table-driven with testify assertions
3. **Infrastructure integration tests** -- Real BoltDB in `t.TempDir()` with testify/suite lifecycle
4. **Handler unit tests** -- httptest + mock ServiceContext for API; gorilla/websocket client for WebSocket hub
5. **Cross-cutting** -- Race detection (`-race`), error path coverage, CI threshold enforcement

### Critical Pitfalls

1. **BoltDB single-writer deadlocks** -- Always use `t.TempDir()` per test, register `t.Cleanup(func() { db.Close() })` immediately after `bolt.Open()`, never share DB files between parallel subtests. Existing pattern in `ChainRepoSuite` is correct; maintain it.

2. **`time.Sleep` synchronization causing flaky tests** -- Already present in hub_test.go, server_test.go, relay_test.go, sync_test.go. Replace with `require.Eventually` or polling-with-deadline pattern. Fix existing instances when touching those files.

3. **Goroutine leaks from P2P servers and WebSocket hubs** -- Always use `t.Cleanup()` for server shutdown. The WebSocket hub currently has no `Stop()` method, leaking goroutines. Add context-based cancellation or a quit channel.

4. **TCP port conflicts in P2P tests** -- Always use port 0 for OS-assigned ephemeral ports (existing pattern is correct). Never hardcode ports. Set connection deadlines in test clients.

5. **Duplicated mock implementations** -- 4 copies of `mockChainRepo` across packages with subtle behavior differences (some lack mutex locks). Consolidate into `internal/testutil/mock/` in the first phase to prevent divergence.

## Implications for Roadmap

Based on research, suggested phase structure:

### Phase 1: Test Infrastructure Foundation
**Rationale:** Every subsequent phase imports shared helpers. Mock duplication is the number one source of maintenance burden and inconsistency. Must be solved first.
**Delivers:** `internal/testutil/` package with chain/tx/utxo builders, consolidated mock repos (ChainRepo, UTXORepo, WalletRepo), type-assertion helpers for `[]any` transactions
**Addresses:** Test helpers (table stakes), mock deduplication (pitfall #6)
**Avoids:** Duplicated mock implementations (#6), error-swallowing helpers (#12), `[]any` type assertion bugs (#8)

### Phase 2: Domain Layer Coverage
**Rationale:** Domain logic is pure, fast, and has no I/O dependencies. Highest-value tests per effort. Must be solid before handler tests depend on domain objects.
**Delivers:** Coverage improvements for chain (69.5% -> 85%+), p2p encoding (67.1% -> 80%+), utxo (86% -> 95%+), wallet (87% -> 95%+), mempool (90% -> 95%+), tx (94% -> 95%+)
**Addresses:** Chain edge cases, P2P message encoding, domain gap-filling (all table stakes)
**Avoids:** Mining difficulty too high (#5), ECDSA non-determinism (#10), reorg edge cases (#13)

### Phase 3: Infrastructure Persistence Tests
**Rationale:** Can run in parallel with Phase 2 (independent layer). BoltDB correctness is critical for data integrity. Established suite pattern makes extension straightforward.
**Delivers:** BoltDB coverage (55.7% -> 80%+) including atomic saves, range queries, DeleteBlocksAbove (reorg), and undo entries. JSON file wallet repo (82.5% -> 90%+).
**Addresses:** BoltDB repository tests (table stakes), serialization roundtrip tests
**Avoids:** BoltDB deadlocks (#1), serialization field drift (#11)

### Phase 4: Handler Layer Tests
**Rationale:** Depends on testutil/mock from Phase 1. Handler tests verify the HTTP/WS interface layer, which has the largest absolute coverage gaps (API 41%, WS 35%).
**Delivers:** API handler coverage (41.3% -> 80%+) for all 8 endpoints, WebSocket hub coverage (34.7% -> 75%+) with proper event broadcasting tests
**Addresses:** API handler tests, WebSocket hub tests (both table stakes)
**Avoids:** go-zero pathvar coupling (#14), WebSocket implementation coupling (#9), goroutine leaks (#4)

### Phase 5: Cross-Cutting Quality
**Rationale:** Only meaningful after baseline coverage exists. Race detection, error paths, and CI enforcement are quality multipliers, not coverage builders.
**Delivers:** `-race` flag in CI, error path tests across all packages, coverage threshold enforcement (70%+), existing `time.Sleep` refactoring
**Addresses:** Race condition testing (table stakes), error path testing (table stakes), coverage CI gate (differentiator)
**Avoids:** Flaky sleep-based tests (#3), race conditions in concurrent assertions (#7)

### Phase 6: Mock Migration and Integration Tests
**Rationale:** Defer refactoring working tests until new coverage is stable. Integration tests synthesize all layers and are the most pitfall-prone.
**Delivers:** Existing tests migrated to `testutil/mock/` (eliminating ~400 lines of duplication), optional integration tests behind build tags (P2P multi-node, E2E chain scenarios, UTXO rollback)
**Addresses:** P2P integration tests, E2E scenarios, UTXO rollback tests (all differentiators)
**Avoids:** All pitfalls compounded -- this is why it comes last

### Phase Ordering Rationale

- Phases follow the dependency graph: testutil (Phase 1) -> domain (Phase 2) -> handlers (Phase 4). Infrastructure (Phase 3) is independent and can parallel with Phase 2.
- The ordering is pitfall-aware: foundational risks (mock duplication, test helpers) are resolved before they compound in integration tests.
- Coverage gaps are attacked largest-first within each layer: API (41%) and WS (35%) in Phase 4, chain (69%) and P2P (67%) in Phase 2, bbolt (56%) in Phase 3.
- Cross-cutting quality (Phase 5) comes after coverage exists, because `-race` and error path tests are meaningless without test volume.

### Research Flags

Phases likely needing deeper research during planning:
- **Phase 4 (Handler Tests):** WebSocket hub testing with gorilla/websocket client needs concrete implementation patterns. The existing hub lacks a `Stop()` method, which may require a small production code change (adding context cancellation).
- **Phase 6 (Integration Tests):** Multi-node P2P integration tests are complex. Need to validate port allocation strategy and determine how many integration scenarios provide sufficient confidence.

Phases with standard patterns (skip research-phase):
- **Phase 1 (Test Infrastructure):** Well-documented Go test helper patterns. The mock consolidation is mechanical.
- **Phase 2 (Domain Layer):** Pure unit tests with table-driven patterns already established in the codebase.
- **Phase 3 (Infrastructure Tests):** testify/suite + BoltDB pattern already working in `chain_repo_test.go`.
- **Phase 5 (Cross-Cutting):** Adding `-race` and coverage thresholds are CI configuration changes, not code decisions.

## Confidence Assessment

| Area | Confidence | Notes |
|------|------------|-------|
| Stack | HIGH | All tools already in go.mod or stdlib. Verified versions against go.mod directly. Zero new dependencies. |
| Features | HIGH | Coverage gaps measured from actual `go test -cover` output. Feature priorities derived from gap size and risk. |
| Architecture | HIGH | Test architecture extends established patterns from 23 existing test files. No novel patterns needed. |
| Pitfalls | HIGH | All pitfalls identified from direct codebase inspection. Sleep-based waits, mock duplication, and BoltDB patterns verified in source. |

**Overall confidence:** HIGH

### Gaps to Address

- **WebSocket hub Stop() method:** The hub currently has no graceful shutdown. Tests will leak goroutines without it. Determine during Phase 4 planning whether to add a `Stop()` method (small production code change) or use context cancellation.
- **CLI handler testability:** `handler/cli` orchestration code (testnet.go, demo.go) may not be testable without significant refactoring. The 50% target for CLI is aspirational -- validate during Phase 5 planning whether simple dispatch tests are sufficient.
- **Coverage targets vs effort:** The 80%+ targets for API, WS, and bbolt are ambitious. If certain paths require disproportionate mocking effort, consider lowering targets for specific packages during planning.

## Sources

### Primary (HIGH confidence)
- Direct codebase analysis: 23 test files across 15 packages (`go test -cover` output, 2026-03-08)
- `go.mod` dependency verification: testify v1.11.1, gorilla/websocket v1.5.3, bbolt v1.4.3, Go 1.26.1
- `.github/workflows/ci-go.yml` and `.golangci.yml` -- current CI configuration
- Go stdlib documentation: `testing`, `net/http/httptest`, `t.TempDir()`, `t.Cleanup()` -- stable, well-documented
- testify library documentation and established usage patterns across all test files

### Secondary (MEDIUM confidence)
- goleak (`go.uber.org/goleak`) for goroutine leak detection -- well-maintained Uber open-source, but not yet used in codebase
- `require.Eventually` polling pattern for replacing `time.Sleep` -- documented in testify, not yet used in this codebase

---
*Research completed: 2026-03-08*
*Ready for roadmap: yes*
