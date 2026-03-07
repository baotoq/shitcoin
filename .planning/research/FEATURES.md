# Feature Landscape

**Domain:** Go test coverage and quality for a blockchain application
**Researched:** 2026-03-08

## Current State

The project has 23 test files with uneven coverage. All tests pass. Patterns established: table-driven tests with `testify/assert`+`require`, `testify/suite` for BoltDB repos, hand-written mocks, `httptest` for API handlers, `t.TempDir()` for temp files.

| Package | Current Coverage | Target | Gap Size |
|---------|-----------------|--------|----------|
| `config` | 100.0% | 100% | None |
| `domain/block` | 98.2% | 98%+ | None |
| `domain/events` | 100.0% | 100% | None |
| `domain/tx` | 94.4% | 95%+ | Small |
| `domain/mempool` | 90.9% | 95%+ | Small |
| `domain/wallet` | 87.6% | 95%+ | Medium |
| `domain/utxo` | 86.2% | 95%+ | Medium |
| `domain/chain` | 69.5% | 85%+ | Large |
| `domain/p2p` | 67.1% | 80%+ | Large |
| `infrastructure/bbolt` | 55.7% | 80%+ | Large |
| `infrastructure/jsonfile` | 82.5% | 90%+ | Small |
| `handler/api` | 41.3% | 80%+ | Large |
| `handler/ws` | 34.7% | 75%+ | Large |
| `handler/cli` | 0% (no tests) | 50%+ | Full |
| `svc` | 0% (no tests) | N/A | Skip (wiring only) |

## Table Stakes

Features that must exist for the test suite to be considered comprehensive. Missing any of these means the milestone is incomplete.

| Feature | Why Expected | Complexity | Depends On | Package Gap |
|---------|--------------|------------|------------|-------------|
| Test helpers and fixtures | Foundation for all other tests; every test file creates blocks/txs manually today. Reusable builders eliminate boilerplate and ensure consistency | Low | `domain/block`, `domain/tx`, `domain/wallet` | New shared helpers |
| API handler tests for all endpoints | Only 2 of 6 handler files have tests (41.3%). address, mempool, search, and tx handlers are completely untested | Medium | `handler/api`, mock repos (pattern already exists in `block_handler_test.go`) | address_handler, mempool_handler, search_handler, tx_handler |
| WebSocket hub event broadcasting tests | 34.7% coverage. Hub is the bridge between domain events and browser clients; must verify subscribe, broadcast, client disconnect | Medium | `handler/ws`, gorilla/websocket test helpers, `domain/events` | hub.go, client.go, events.go |
| Chain aggregate edge case tests | 69.5% coverage on the most critical domain code. Mining orchestration, reorg logic, and difficulty adjustment need thorough error path testing | Medium | `domain/chain`, block/tx helpers | chain_test.go expansion |
| P2P message encoding/decoding tests | Binary wire protocol (`[4-byte length][1-byte cmd][JSON payload]`) is fragile. Protocol.go serialization must be tested exhaustively | Medium | `domain/p2p` | protocol_test.go (may not exist separately) |
| P2P handler coverage | Message handlers in handler.go (308 LOC) are the most complex domain code. Block relay, tx relay, version handshake paths need coverage | High | `domain/p2p`, block/tx helpers for message payloads | handler.go untested paths |
| BoltDB repository tests | 55.7% coverage on persistence. Atomic block+UTXO saves, GetBlocksInRange, DeleteBlocksAbove (reorg), and GetUndoEntry need testing | Medium | `infrastructure/bbolt`, temp DB via `t.TempDir()` | chain_repo, utxo_repo expansion |
| Error path testing across all packages | Happy paths exist but error/edge cases are underserved everywhere. Invalid blocks, double spends, corrupt data, nil inputs, boundary conditions | Medium | All packages | Cross-cutting across all test files |
| Domain gap-filling (utxo, wallet, mempool) | Coverage is 86-91% -- close to target but missing edge cases. Small effort to reach 95%+ | Low | Per-package review of uncovered lines | utxo/set_test.go, wallet/wallet_test.go, mempool/mempool_test.go |
| Race condition testing (`go test -race`) | P2P server, mempool, and WebSocket hub all have concurrent access patterns. `-race` flag catches data races the type system cannot | Low | Existing concurrent code, all other tests passing | Add `-race` flag to test commands |

## Differentiators

Features that elevate test quality beyond basic coverage. Not strictly required for the milestone, but high value for an educational project demonstrating engineering maturity.

| Feature | Value Proposition | Complexity | Depends On |
|---------|-------------------|------------|------------|
| P2P integration tests (multi-node in-process) | Verify actual TCP connections, handshake, block sync, and tx relay between 2+ in-process nodes. Proves the whole P2P stack works end-to-end | High | `domain/p2p` unit tests solid, test port allocation (`net.Listen(":0")`) |
| End-to-end chain scenario tests | Full workflow: create wallet -> send tx -> mine block -> verify UTXO updated -> check balance. Proves domain packages integrate correctly | High | All domain packages, test helpers |
| UTXO undo/rollback integration tests | Apply a chain of blocks, trigger reorg, verify UTXO set rolls back to correct state. Critical for blockchain correctness | Medium | `domain/utxo`, `domain/chain`, BoltDB repo tests |
| Coverage enforcement in CI | Fail CI if coverage drops below per-package thresholds. Prevents regression as code evolves | Low | CI pipeline (exists), meaningful coverage achieved first |
| Golden file tests for serialization | Snapshot P2P messages and block serialization to files; detect accidental wire format changes. Important for protocol stability | Medium | `domain/p2p/protocol.go`, `domain/block` serialization |
| Fuzz tests for deserialization | Go native fuzzing (`func FuzzXxx`) for P2P message parsing and block deserialization. Finds edge cases humans miss | Medium | Serialization unit tests exist first |
| Benchmark tests for PoW mining | `func BenchmarkMine` at various difficulties. Establishes performance baseline and catches regressions | Low | `domain/block` |
| CLI handler tests for command dispatch | Test that `cli.go` dispatches to correct domain functions for simple commands (createwallet, getbalance, printchain) | Medium | `handler/cli`, mock ServiceContext |
| Test coverage report generation | `go test -coverprofile` + HTML report or CI integration with Codecov/Coveralls | Low | All tests passing |

## Anti-Features

Features to explicitly NOT build for this testing milestone.

| Anti-Feature | Why Avoid | What to Do Instead |
|--------------|-----------|-------------------|
| Full CLI handler coverage (testnet, demo) | `handler/cli` testnet.go (221 LOC) and demo.go (227 LOC) are orchestration code spawning subprocesses and managing multi-node setups. Mocking all dependencies is high effort, low value. The domain logic they call is what matters | Test domain logic directly. At most, test simple command dispatch in cli.go. Skip testnet orchestration and demo subprocess tests entirely |
| Mock generation framework (mockgen, moq) | Only 3 repository interfaces exist (chain.Repository, utxo.Repository, wallet.Repository). Hand-written mocks are already established in `block_handler_test.go`. Adding codegen machinery for 3 interfaces is overhead | Continue hand-written mocks. Create a shared `testutil` package if mock duplication appears across packages |
| Property-based testing (gopter, rapid) | Overkill for an educational project with well-defined inputs. Table-driven tests cover the same ground more readably | Use table-driven tests with explicit edge cases. Native Go fuzzing covers randomized input for deserialization |
| Mutation testing (go-mutesting) | Slow, noisy output, and the project is not large enough (11.4K LOC) to justify the overhead. False positives waste time | Focus on meaningful coverage metrics and explicit error path tests instead |
| Contract/API schema tests (OpenAPI) | No OpenAPI spec exists. The REST API has 7 simple endpoints consumed only by the bundled React frontend | Test handlers directly with `httptest.NewRecorder()` as already established |
| `svc.ServiceContext` unit tests | Pure wiring code (struct initialization, dependency injection). No logic to test. Testing it means asserting that Go struct fields are assigned | Skip entirely. ServiceContext correctness is proven by integration tests and the fact that the app runs |
| External test infrastructure (testcontainers, Docker) | BoltDB is embedded. All P2P is localhost TCP. There are no external dependencies to containerize | Use `t.TempDir()` for BoltDB, `net.Listen(":0")` for test TCP ports. Everything runs in-process |
| Test DSL or custom assertion library | `testify/assert` + `testify/require` are already in use and universally understood in Go. A custom DSL adds learning curve for no gain | Stick with testify. Use `require` for preconditions, `assert` for verifications |

## Feature Dependencies

```
Test Helpers/Fixtures (block builder, tx builder, wallet builder, mock repos)
  |
  +--> Domain Unit Tests
  |      +-- tx gap-filling (94.4% -> 95%+)
  |      +-- mempool gap-filling (90.9% -> 95%+)
  |      +-- wallet gap-filling (87.6% -> 95%+)
  |      +-- utxo gap-filling (86.2% -> 95%+)
  |      +-- chain edge cases (69.5% -> 85%+)
  |      +-- p2p message encoding (67.1% -> 80%+)
  |      +-- p2p handler coverage
  |
  +--> Handler Tests
  |      +-- API handler tests (41.3% -> 80%+)
  |      |     requires: mock chain/utxo repos (pattern exists)
  |      +-- WebSocket hub tests (34.7% -> 75%+)
  |      |     requires: mock event bus, websocket test client
  |      +-- CLI dispatch tests (0% -> 50%+)  [OPTIONAL]
  |            requires: mock ServiceContext
  |
  +--> Infrastructure Tests
  |      +-- BoltDB chain repo (55.7% -> 80%+)
  |      |     requires: block helpers, t.TempDir()
  |      +-- BoltDB utxo repo (part of 55.7%)
  |      |     requires: utxo helpers, t.TempDir()
  |      +-- JSON file wallet repo (82.5% -> 90%+)
  |            requires: wallet helpers, t.TempDir()
  |
  +--> Cross-cutting
         +-- Error path tests (requires: all above exist first)
         +-- Race condition tests (requires: all above pass without -race)
         +-- Coverage CI gate (requires: coverage targets met)

Integration tests (P2P multi-node, E2E scenarios, UTXO rollback)
  requires: All unit tests above are solid
```

## MVP Recommendation

Prioritize by coverage gap size and risk, with dependency ordering:

1. **Test helpers and fixtures** -- Foundation for everything. Create reusable block/tx/wallet/utxo builders and shared mock repos. Low effort, highest leverage. Do this FIRST.

2. **API handler tests** (41.3% -> 80%+) -- Largest absolute gap in the handler layer. 4 completely untested handler files. Mock pattern already established in `block_handler_test.go`. Medium effort, high impact.

3. **WebSocket hub tests** (34.7% -> 75%+) -- Second largest coverage gap. Event broadcasting is user-visible functionality. Needs websocket test client setup. Medium effort.

4. **BoltDB repository tests** (55.7% -> 80%+) -- Persistence correctness matters for data integrity. Atomic saves, range queries, delete (reorg) operations. Suite pattern already established. Medium effort.

5. **Chain aggregate tests** (69.5% -> 85%+) -- Critical domain logic: mining orchestration, reorg, difficulty adjustment. Complex but high-value paths. Medium effort.

6. **P2P unit tests** (67.1% -> 80%+) -- Message encoding, handler dispatch, sync logic. Largest and most complex domain package (1422 LOC). Medium-High effort.

7. **Domain gap-filling** (utxo 86%, wallet 87%, mempool 90%, tx 94%) -- Small gaps, quick wins. Review uncovered lines, add targeted tests. Low effort per package.

8. **Error path and edge case tests** -- Cross-cutting pass after all packages have baseline coverage. Invalid inputs, nil handling, boundary conditions. Medium effort, spread across packages.

9. **Race condition testing** -- Add `-race` flag to `go test` invocations. Fix any races found. Low effort to enable, variable effort to fix.

Defer to later:
- **P2P integration tests** (multi-node): High complexity, lower marginal value once unit tests cover handler logic.
- **E2E chain scenario tests**: Valuable but depends on all layers being tested first.
- **Fuzz tests and golden files**: Add after serialization paths are well-tested.
- **CLI handler tests**: Low value/effort ratio. Domain logic tests cover the important paths.
- **Benchmark tests**: Nice-to-have, not a coverage priority.
- **Coverage CI gate**: Only useful after targets are actually met.

## Sources

- Direct codebase analysis: `go test ./... -cover` output (2026-03-08)
- Existing test file review: 23 test files across 14 packages
- Test patterns observed: table-driven (testify), suite (testify/suite for BoltDB), hand-written mocks, httptest
- Go testing stdlib conventions (HIGH confidence -- stable, well-documented patterns)
- testify library patterns (HIGH confidence -- dominant Go testing library)

---
*Feature research for: Testing & Quality (v1.2 milestone)*
*Researched: 2026-03-08*
