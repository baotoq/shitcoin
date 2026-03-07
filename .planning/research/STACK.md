# Stack Research

**Domain:** Go testing and quality tooling for existing blockchain application
**Researched:** 2026-03-08
**Confidence:** HIGH

## Current State

The project already has solid foundations:
- **22 test files** across domain, handler, and infrastructure layers
- **testify v1.11.1** (assert, require, mock, suite all in use)
- **golangci-lint v2.10** in CI with standard linters
- **GitHub Actions CI** runs `go test -coverprofile` and `golangci-lint`

Coverage by package (current):

| Package | Coverage | Gap Assessment |
|---------|----------|----------------|
| config | 100% | Done |
| domain/block | 98.2% | Done |
| domain/events | 100% | Done |
| domain/tx | 94.4% | Nearly done |
| domain/mempool | 90.9% | Minor gaps |
| domain/wallet | 87.6% | Minor gaps |
| domain/utxo | 86.2% | Minor gaps |
| infrastructure/jsonfile | 82.5% | Moderate gaps |
| domain/chain | 69.5% | Needs work |
| domain/p2p | 67.1% | Needs work |
| infrastructure/bbolt | 55.7% | Needs work |
| handler/api | 41.3% | Significant gaps |
| handler/ws | 34.7% | Significant gaps |
| handler/cli | 0% (no tests) | No test files |
| svc | 0% (no tests) | No test files |

## Recommended Stack

### Core Technologies (already present -- no changes needed)

| Technology | Version | Purpose | Why Recommended |
|------------|---------|---------|-----------------|
| Go stdlib `testing` | 1.26.1 | Test runner, benchmarks, `t.TempDir()`, `t.Cleanup()` | Built-in, zero dependency, industry standard |
| stretchr/testify | v1.11.1 | assert/require/mock/suite -- complete assertion toolkit | Already established in all 22 test files; latest stable version |
| golangci-lint | v2.10 | Static analysis in CI | Already configured in CI and .golangci.yml |

**No new test framework dependencies are needed.** The existing stack (stdlib `testing` + testify) is the Go community standard and covers all testing needs for this project.

### Supporting Libraries (already available -- no new installs)

| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| `net/http/httptest` | stdlib | HTTP handler testing without starting a server | API handler tests -- expanding coverage from 41.3% |
| `testing/fstest` | stdlib | In-memory filesystem for file-based tests | Wallet JSON file repo edge cases |
| `t.TempDir()` | stdlib | Auto-cleaned temp directories for BoltDB test instances | Already used in bbolt tests; extend to all integration tests |
| `testify/mock` | v1.11.1 | Interface mocking for repository isolation | Already used in chain_test.go and server_test.go |
| `testify/suite` | v1.11.1 | Setup/teardown lifecycle for integration tests | Already used in bbolt repo tests |
| gorilla/websocket | v1.5.3 | WebSocket client for testing ws.Hub | Already a dependency; use `websocket.Dial` in hub tests |

### Development Tools

| Tool | Purpose | Notes |
|------|---------|-------|
| `go test -race` | Race condition detection | **Add to CI** -- critical for P2P and event bus concurrency |
| `go test -coverprofile -covermode=atomic` | Race-safe coverage reporting | Replace current `-coverprofile` in CI |
| `go test -short` | Skip long-running integration tests | Use `testing.Short()` to gate slow P2P/mining tests |
| `go tool cover -html` | Visual coverage browser | Local development only, not CI |
| `go test -count=1` | Disable test caching | For debugging flaky tests locally |
| `go test -timeout 30s` | Prevent hanging tests | Add per-package in CI to catch deadlocks early |

## Installation

No new dependencies to install. Everything needed is already in `go.mod` or Go's standard library.

```bash
# Run all tests with race detection (recommended default)
go test -race ./...

# Run with coverage
go test -race -coverprofile=coverage.out -covermode=atomic ./...

# View coverage in browser
go tool cover -html=coverage.out

# Run only unit tests (skip slow integration tests)
go test -short ./...
```

## CI Enhancements

The existing `.github/workflows/ci-go.yml` should be enhanced with two additions:

### 1. Race Detection

```yaml
- name: Run tests with coverage
  run: go test -race -coverprofile=coverage.out -covermode=atomic -timeout 120s ./...
```

Why: The project has concurrent code in P2P (goroutines per peer), event bus (pub/sub), WebSocket hub (fan-out), and auto-mining. Race detection catches data races that unit tests alone miss.

### 2. Coverage Threshold

```yaml
- name: Check coverage threshold
  run: |
    TOTAL=$(go tool cover -func=coverage.out | grep total | awk '{print $3}' | tr -d '%')
    echo "Total coverage: ${TOTAL}%"
    if (( $(echo "$TOTAL < 70" | bc -l) )); then
      echo "Coverage below 70% threshold"
      exit 1
    fi
```

Why: Prevents coverage regression after the milestone raises it. Start at 70% (achievable given current state), ratchet up later.

## Alternatives Considered

| Recommended | Alternative | When to Use Alternative |
|-------------|-------------|-------------------------|
| testify/mock (hand-written) | mockery (code generation) | Projects with 20+ interfaces; this project has ~5 repository interfaces |
| testify/mock (hand-written) | gomock / uber/mock | Never for this project; testify/mock is already established |
| testify/suite for integration | go-testfixtures | SQL database projects needing fixture loading; irrelevant for BoltDB |
| `go test -race` | go-deadlock | Only for mutex deadlock diagnosis; race detector is more comprehensive |
| `-coverprofile` (stdlib) | codecov/coveralls | When you want PR-level coverage diffs and badges; overkill for educational project |
| `t.TempDir()` + real BoltDB | testcontainers-go | Projects with external services (PostgreSQL, Redis); BoltDB is embedded |
| `httptest.NewRecorder` | httpexpect | Heavy HTTP testing DSL; stdlib + testify assertions are simpler and already used |

## What NOT to Add

| Avoid | Why | Use Instead |
|-------|-----|-------------|
| testcontainers-go | BoltDB is embedded -- no external services to containerize | `t.TempDir()` + real BoltDB instances (already the pattern) |
| mockery / mockgen | Only ~5 interfaces to mock; generators add build toolchain complexity | Hand-written testify/mock structs (already established) |
| goconvey | BDD-style adds learning curve and rewrites all existing tests | testify/assert + testify/require (already used) |
| ginkgo + gomega | Full BDD framework; would require rewriting all 22 existing test files | stdlib testing + testify (already used) |
| go-sqlmock | No SQL database in this project | Direct BoltDB testing with temp dirs |
| httpexpect | Heavy DSL for a project with ~8 REST endpoints | `net/http/httptest` + testify |
| Separate e2e test binary | Educational project; in-process integration is fast enough | In-process tests with real BoltDB |
| gotestsum | Pretty test output formatter; nice but not essential for coverage milestone | `go test -v` for local debugging |

## Testing Patterns by Layer

**Domain layer (block, chain, tx, utxo, wallet, mempool, events):**
- Pure unit tests with testify/assert and testify/require
- No mocks needed for most domain logic (pure computation)
- Mock repository interfaces only in chain.Chain orchestration tests
- Pattern already established; extend coverage to uncovered branches

**P2P layer:**
- `net.Pipe()` or localhost TCP for peer simulation (already done in server_test.go)
- Mock chain.Repository with testify/mock for isolating P2P from storage
- Message encoding/decoding as pure unit tests
- Gate slow multi-node tests behind `testing.Short()`

**Handler layer (API, WebSocket):**
- `httptest.NewRecorder` + `httptest.NewRequest` for REST handlers (pattern established)
- gorilla/websocket client dialing `httptest.NewServer` for WebSocket hub tests
- Mock ServiceContext dependencies; avoid real BoltDB in handler tests

**Infrastructure layer (bbolt, jsonfile):**
- Integration tests with real BoltDB on `t.TempDir()` (already the pattern)
- testify/suite for setup/teardown lifecycle (already used in bbolt tests)
- These ARE the integration tests -- BoltDB is fast enough to test directly

**CLI handler (currently 0% -- new):**
- Test command dispatch functions, not `main()`
- Capture stdout with `os.Pipe()` or pass `io.Writer` for output verification
- Mock ServiceContext to avoid real blockchain initialization

## Version Compatibility

| Package | Compatible With | Notes |
|---------|-----------------|-------|
| testify v1.11.1 | Go 1.26.1 | Latest stable; fully compatible |
| golangci-lint v2.10 | Go 1.26.1 | golangci-lint-action@v9 handles installation |
| gorilla/websocket v1.5.3 | Go 1.26.1 | Stable; used in both production and test code |
| bbolt v1.4.3 | Go 1.26.1 | `t.TempDir()` integration tests work across versions |

## Key Insight

This milestone is about **writing more tests with existing tools**, not adding new tools. The gaps are:

1. **Coverage gaps** in chain (69.5%), p2p (67.1%), bbolt (55.7%), API (41.3%), WS (34.7%), CLI (0%)
2. **Race detection** not enabled in CI (add `-race` flag)
3. **No coverage threshold** enforcement in CI (add threshold check)
4. **No `-short` flag convention** to separate fast unit tests from slower integration tests

All addressed by writing more test code and adjusting CI flags -- zero new `go get` commands required.

## Sources

- `go.mod` -- verified all dependency versions (testify v1.11.1, gorilla/websocket v1.5.3, bbolt v1.4.3)
- 22 existing test files -- analyzed imports, patterns, mock usage, suite usage
- `.github/workflows/ci-go.yml` -- verified current CI configuration (coverprofile, golangci-lint v2.10)
- `.golangci.yml` -- verified linter configuration (v2 format, standard linters)
- `go test -coverprofile` output -- measured current coverage per package (2026-03-08)
- Go stdlib documentation (testing, net/http/httptest) -- HIGH confidence, built-in

---
*Stack research for: Go testing & quality tools (v1.2 milestone)*
*Researched: 2026-03-08*
