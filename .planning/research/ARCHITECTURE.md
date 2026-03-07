# Architecture Patterns

**Domain:** Testing & quality infrastructure for Go blockchain application (DDD architecture)
**Researched:** 2026-03-08

## Recommended Architecture

Testing integrates with the existing DDD layers without modifying production code. The codebase already has 23 test files covering all packages except `cmd/shitcoin`, `internal/handler/cli`, and `internal/svc`. The established patterns are sound -- the goal is to extend coverage, consolidate duplicated test helpers, and add missing test categories.

### Current Test Architecture (What Exists)

```
internal/
├── domain/
│   ├── block/       block_test.go, pow_test.go, merkle_test.go, difficulty_test.go
│   ├── chain/       chain_test.go  (reorg, rewards, fees -- uses hand-rolled mocks)
│   ├── tx/          transaction_test.go
│   ├── utxo/        set_test.go
│   ├── wallet/      wallet_test.go, base58_test.go
│   ├── mempool/     mempool_test.go  (uses hand-rolled memRepo)
│   ├── p2p/         p2p_test.go, server_test.go, sync_test.go, relay_test.go, reorg_test.go
│   └── events/      bus_test.go
├── handler/
│   ├── api/         block_handler_test.go, status_handler_test.go  (httptest + hand-rolled mocks)
│   ├── cli/         [NO TESTS]
│   └── ws/          hub_test.go
├── config/          config_test.go
├── infrastructure/
│   └── persistence/
│       ├── bbolt/   chain_repo_test.go, utxo_repo_test.go  (suite.Suite + real BoltDB in TempDir)
│       └── jsonfile/ wallet_repo_test.go
└── svc/             [NO TESTS]
```

### Target Test Architecture (What to Build)

```
internal/
├── testutil/                          NEW -- shared test helpers package
│   ├── chain.go                       Chain/block factory helpers
│   ├── tx.go                          Transaction + signing helpers
│   ├── utxo.go                        UTXO setup helpers
│   └── mock/                          NEW -- consolidated mock implementations
│       ├── chain_repo.go              In-memory chain.Repository
│       ├── utxo_repo.go               In-memory utxo.Repository
│       └── wallet_repo.go             In-memory wallet.Repository
├── domain/
│   ├── block/       [existing tests sufficient, add edge cases]
│   ├── chain/       [extend: mining, validation, difficulty adjustment]
│   ├── tx/          [extend: signing, validation, edge cases]
│   ├── utxo/        [extend: rollback, concurrent access]
│   ├── wallet/      [existing tests sufficient]
│   ├── mempool/     [existing tests comprehensive]
│   ├── p2p/         [existing tests comprehensive]
│   └── events/      [existing tests sufficient]
├── handler/
│   ├── api/         [extend: all 8 endpoints, error paths]
│   ├── cli/         [NEW: test command dispatch, testnet, demo]
│   └── ws/          [extend: event forwarding, connection lifecycle]
├── infrastructure/
│   └── persistence/
│       ├── bbolt/   [extend: concurrent access, undo entries, DeleteBlocksAbove]
│       └── jsonfile/ [extend: concurrent access, file corruption recovery]
└── svc/             [NEW: ServiceContext construction, Close cleanup]
```

### Component Boundaries

| Component | Responsibility | Communicates With |
|-----------|---------------|-------------------|
| `internal/testutil/` | Shared test factories and helper functions | All test files across all layers |
| `internal/testutil/mock/` | Consolidated in-memory repository implementations | Domain and handler tests |
| Domain unit tests | Test pure business logic in isolation | Mock repositories via interfaces |
| Handler tests | Test HTTP/WS handlers with httptest | Mock ServiceContext fields, mock repos |
| Infrastructure tests | Test real persistence against temp DB files | Real BoltDB/JSON in `t.TempDir()` |
| Integration tests (build tag) | Test cross-layer flows end-to-end | Real repositories, real domain objects |

### Data Flow: Test Fixture Creation

```
testutil.NewTestChain(t, opts...)
    |
    v
Creates mock repos (chain + UTXO) in memory
    |
    v
Initializes chain.Chain with genesis block
    |
    v
Returns TestChain{ Chain, ChainRepo, UTXOSet, UTXORepo }
    |
    v
Test calls testutil.MineBlocks(t, tc, minerAddr, n)
    |
    v
Mines n blocks, applies UTXO changes, returns []*block.Block
    |
    v
Test calls testutil.BuildSignedTx(t, tc, from, to, amount)
    |
    v
Finds UTXOs for 'from', creates + signs tx, returns *tx.Transaction
```

## Integration Points with Existing Codebase

### New Components

| Component | Type | Purpose |
|-----------|------|---------|
| `internal/testutil/chain.go` | New file | `NewTestChain`, `MineBlocks`, `DefaultChainConfig` helpers |
| `internal/testutil/tx.go` | New file | `BuildSignedTx`, `NewTestWallet`, `FundAddress` helpers |
| `internal/testutil/utxo.go` | New file | `SeedUTXOs`, `AssertBalance` helpers |
| `internal/testutil/mock/chain_repo.go` | New file | Consolidated `mockChainRepo` (currently duplicated in 3 packages) |
| `internal/testutil/mock/utxo_repo.go` | New file | Consolidated `mockUTXORepo` (currently duplicated in 3 packages) |
| `internal/testutil/mock/wallet_repo.go` | New file | In-memory `wallet.Repository` for handler tests |
| `internal/handler/cli/cli_test.go` | New file | CLI command dispatch tests |
| `internal/svc/service_context_test.go` | New file | ServiceContext construction/teardown tests |

### Modified Components

| File | Change | Reason |
|------|--------|--------|
| `internal/domain/chain/chain_test.go` | Replace local `mockChainRepo`/`mockUTXORepo` with `testutil/mock` imports | Deduplicate ~200 lines of duplicated mock code |
| `internal/domain/mempool/mempool_test.go` | Replace local `memRepo` with `testutil/mock.UTXORepo` | Deduplicate |
| `internal/handler/api/block_handler_test.go` | Replace local `mockChainRepo` with `testutil/mock.ChainRepo` | Deduplicate |
| `internal/domain/p2p/server_test.go` | Replace local `MockChainRepo` with `testutil/mock.ChainRepo` | Deduplicate |

### Existing Code NOT Modified

No production code changes. All repository interfaces (`chain.Repository`, `utxo.Repository`, `wallet.Repository`) already support dependency injection via interfaces. The `svc.ServiceContext` struct has exported fields that tests can populate directly (already done in `status_handler_test.go`).

## Patterns to Follow

### Pattern 1: Shared Test Helpers in `internal/testutil/`

**What:** Centralized package for test factories, eliminating duplication across 4+ packages that each re-implement `mockChainRepo` and `mockUTXORepo`.
**When:** Any test that needs a chain, blocks, or signed transactions.

```go
// internal/testutil/chain.go
package testutil

import (
    "context"
    "testing"

    "github.com/baotoq/shitcoin/internal/domain/block"
    "github.com/baotoq/shitcoin/internal/domain/chain"
    "github.com/baotoq/shitcoin/internal/domain/utxo"
    mockpkg "github.com/baotoq/shitcoin/internal/testutil/mock"
    "github.com/stretchr/testify/require"
)

// TestChain bundles a chain aggregate with its in-memory dependencies.
type TestChain struct {
    Chain     *chain.Chain
    ChainRepo *mockpkg.ChainRepo
    UTXORepo  *mockpkg.UTXORepo
    UTXOSet   *utxo.Set
    PoW       *block.ProofOfWork
    Config    chain.ChainConfig
}

// DefaultChainConfig returns a config with low difficulty for fast test mining.
func DefaultChainConfig() chain.ChainConfig {
    return chain.ChainConfig{
        InitialDifficulty: 1,
        GenesisMessage:    "test-genesis",
        BlockReward:       5_000_000_000,
    }
}

// NewTestChain creates an initialized chain with genesis block, ready for testing.
func NewTestChain(t *testing.T, minerAddr string) *TestChain {
    t.Helper()
    cfg := DefaultChainConfig()
    repo := mockpkg.NewChainRepo()
    utxoRepo := mockpkg.NewUTXORepo()
    utxoSet := utxo.NewSet(utxoRepo)
    pow := &block.ProofOfWork{}
    ch := chain.NewChain(repo, pow, cfg, utxoSet)
    require.NoError(t, ch.Initialize(context.Background(), minerAddr))
    return &TestChain{Chain: ch, ChainRepo: repo, UTXORepo: utxoRepo,
        UTXOSet: utxoSet, PoW: pow, Config: cfg}
}

// MineBlocks mines n blocks and returns them.
func MineBlocks(t *testing.T, tc *TestChain, minerAddr string, n int) []*block.Block {
    t.Helper()
    var blocks []*block.Block
    for range n {
        b, err := tc.Chain.MineBlock(context.Background(), minerAddr, nil, 0)
        require.NoError(t, err)
        blocks = append(blocks, b)
    }
    return blocks
}
```

**Why this pattern:** The codebase currently has 4 independent copies of `mockChainRepo` (chain_test.go, block_handler_test.go, server_test.go -- each ~80 lines) and 3 copies of `mockUTXORepo` (chain_test.go, mempool_test.go -- each ~50 lines). Centralizing eliminates ~400 lines of duplication and ensures consistent behavior.

### Pattern 2: Table-Driven Tests (Already Established)

**What:** The codebase already uses table-driven tests extensively (see `block_test.go`, `chain_test.go`). Continue this pattern for all new tests.
**When:** Any function with multiple input/output combinations.

```go
func TestRewardAtHeight(t *testing.T) {
    tc := testutil.NewTestChain(t, "miner")
    tests := []struct {
        name   string
        height uint64
        want   int64
    }{
        {"genesis", 0, 5_000_000_000},
        {"before halving", 9, 5_000_000_000},
        {"first halving", 10, 2_500_000_000},
    }
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            assert.Equal(t, tt.want, tc.Chain.RewardAtHeight(tt.height))
        })
    }
}
```

### Pattern 3: Suite Tests for Stateful Infrastructure (Already Established)

**What:** Use `testify/suite` for infrastructure tests that need setup/teardown of real resources (BoltDB files).
**When:** Testing `bbolt` and `jsonfile` repositories.

The existing `ChainRepoSuite` in `chain_repo_test.go` is the model. Each test gets a fresh DB via `SetupTest()` with `t.TempDir()` and `t.Cleanup()`.

```go
type ChainRepoSuite struct {
    suite.Suite
    db   *bolt.DB
    repo *BboltRepository
}

func (s *ChainRepoSuite) SetupTest() {
    dbPath := filepath.Join(s.T().TempDir(), "test.db")
    db, err := bolt.Open(dbPath, 0600, nil)
    s.Require().NoError(err)
    s.T().Cleanup(func() { db.Close() })
    s.db = db
    repo, err := NewBboltRepository(db)
    s.Require().NoError(err)
    s.repo = repo
}
```

### Pattern 4: httptest for API Handler Tests (Already Established)

**What:** Use `net/http/httptest` with mock ServiceContext fields.
**When:** Testing REST API handlers.

The existing pattern in `block_handler_test.go` is correct: create mock repos, populate a `svc.ServiceContext` struct with only the needed fields, call the handler function, assert on the recorder.

```go
func TestBlocksHandler(t *testing.T) {
    repo := mock.NewChainRepo()  // from testutil/mock
    // seed blocks...
    svcCtx := &svc.ServiceContext{ChainRepo: repo}
    handler := BlocksHandler(svcCtx)
    req := httptest.NewRequest(http.MethodGet, "/api/blocks", nil)
    w := httptest.NewRecorder()
    handler(w, req)
    assert.Equal(t, http.StatusOK, w.Code)
}
```

**go-zero path variables:** The codebase uses `pathvar.WithVars(req, map[string]string{...})` from `github.com/zeromicro/go-zero/rest/pathvar` to inject route parameters. This is the correct approach -- continue using it.

### Pattern 5: net.Pipe for P2P Protocol Tests (Already Established)

**What:** Use `net.Pipe()` for in-memory TCP connections to test P2P message encoding and peer behavior.
**When:** Testing P2P wire protocol, peer lifecycle, message handling.

The existing `p2p_test.go` uses this pattern well. For server-level tests, `server_test.go` uses port 0 for OS-assigned ports.

### Pattern 6: Build Tags for Integration Tests

**What:** Separate slow integration tests (real DB, real mining, multi-component) from fast unit tests using build tags.
**When:** Tests that take >1 second or require multiple components wired together.

```go
//go:build integration

package integration_test

import (
    "testing"
    "github.com/baotoq/shitcoin/internal/testutil"
)

func TestFullBlockLifecycle(t *testing.T) {
    // Create chain with real BoltDB repo
    // Mine block, verify UTXO changes, verify persistence
}
```

Run separately: `go test -tags integration ./...`

### Pattern 7: Testing the CLI Layer

**What:** Test CLI command dispatch by invoking handler functions directly, not by spawning subprocesses.
**When:** Testing `internal/handler/cli/`.

The CLI handlers accept `*svc.ServiceContext` and use `flag.FlagSet` for argument parsing. Test by constructing a ServiceContext with mock repos and calling the command handler functions.

```go
func TestSendCommand(t *testing.T) {
    // Build ServiceContext with mock repos
    // Call the send handler with parsed flags
    // Verify transaction created and submitted
}
```

Do NOT test via `exec.Command("go", "run", ...)` -- that is slow, flaky, and does not provide useful coverage.

## Anti-Patterns to Avoid

### Anti-Pattern 1: Duplicating Mock Implementations Per Package

**What:** Each test file re-implements `mockChainRepo`, `mockUTXORepo` from scratch.
**Why bad:** Already present in the codebase -- 4 independent copies of chain repo mock (~320 lines total), with subtle behavior differences between them. The mock in `handler/api/` lacks mutex locks; the one in `domain/chain/` has full concurrency safety. This divergence can mask concurrency bugs.
**Instead:** Single implementation in `internal/testutil/mock/` with consistent behavior.

### Anti-Pattern 2: time.Sleep for Synchronization

**What:** Using `time.Sleep(10 * time.Millisecond)` to wait for goroutine processing.
**Why bad:** Already present in `hub_test.go` (5 occurrences). Flaky under CI load, wastes time.
**Instead:** Use channels, `sync.WaitGroup`, or polling with `require.Eventually`:

```go
require.Eventually(t, func() bool {
    hub.mu.RLock()
    defer hub.mu.RUnlock()
    _, exists := hub.clients[client]
    return exists
}, 500*time.Millisecond, 5*time.Millisecond, "client should be registered")
```

### Anti-Pattern 3: Testing Implementation Details

**What:** Asserting on internal struct fields (e.g., `hub.clients[client]`) instead of observable behavior.
**Why bad:** Tests break when internals change even if behavior is preserved.
**Instead:** Test through the public API. For the hub: verify that a published event arrives on the client's send channel.

### Anti-Pattern 4: Tests That Require Specific Mining Results

**What:** Asserting on exact block hashes or nonce values from PoW mining.
**Why bad:** Mining is non-deterministic. Tests become fragile.
**Instead:** Assert on properties: block height, parent hash, valid PoW (difficulty check), transaction contents.

### Anti-Pattern 5: Skipping Error Path Tests

**What:** Only testing the happy path (valid block, valid tx, found result).
**Why bad:** Error handling bugs are common production issues.
**Instead:** Always test: not found, invalid input, duplicate, concurrent access, boundary values.

## Build Order (Dependency Graph for Test Phases)

```
Phase 1: internal/testutil/ + internal/testutil/mock/
    Foundation. All subsequent test work depends on shared helpers.
    |
Phase 2: Domain layer test extensions
    block/, chain/, tx/, utxo/ -- pure logic, no I/O, fast
    Uses testutil helpers. No infrastructure dependency.
    |
Phase 3: Infrastructure layer test extensions
    bbolt/, jsonfile/ -- real DB, TempDir isolation
    Independent of domain test extensions (can parallel with Phase 2).
    |
Phase 4: Handler layer test extensions
    api/ (all 8 endpoints), ws/ (fix time.Sleep)
    Depends on testutil/mock/ from Phase 1.
    |
Phase 5: CLI handler tests + ServiceContext tests
    cli/, svc/ -- higher-level, depends on mock repos from Phase 1.
    |
Phase 6: Migration of existing tests to use testutil/mock/
    Refactor chain_test.go, mempool_test.go, server_test.go,
    block_handler_test.go to import testutil/mock/ instead of local mocks.
    Depends on Phase 1 being stable.
    |
Phase 7: Integration tests (build tag)
    Cross-layer tests using real BoltDB + full chain lifecycle.
    Depends on all prior phases.
```

**Rationale:** The `testutil` package is the foundation -- every subsequent phase imports it. Domain tests come before handler tests because handlers depend on domain objects (blocks, transactions). Infrastructure tests can run in parallel with domain tests since they are independent (real DB vs mocks). CLI tests come later because they orchestrate multiple domain operations. Migration of existing tests (Phase 6) is deferred to avoid disrupting working tests while new coverage is being added. Integration tests come last because they synthesize all layers.

## Files: New vs Modified

### New Files (8 files)

| File | Purpose |
|------|---------|
| `internal/testutil/chain.go` | `NewTestChain`, `MineBlocks`, `DefaultChainConfig` |
| `internal/testutil/tx.go` | `BuildSignedTx`, `NewTestWallet`, `FundAddress` |
| `internal/testutil/utxo.go` | `SeedUTXOs`, `AssertBalance` |
| `internal/testutil/mock/chain_repo.go` | Consolidated thread-safe in-memory `chain.Repository` |
| `internal/testutil/mock/utxo_repo.go` | Consolidated thread-safe in-memory `utxo.Repository` |
| `internal/testutil/mock/wallet_repo.go` | In-memory `wallet.Repository` |
| `internal/handler/cli/cli_test.go` | CLI command dispatch and subcommand tests |
| `internal/svc/service_context_test.go` | ServiceContext construction and Close tests |

### Modified Files (4 files, test-only changes)

| File | Change |
|------|--------|
| `internal/domain/chain/chain_test.go` | Replace ~200 lines of local mocks with `testutil/mock` imports |
| `internal/domain/mempool/mempool_test.go` | Replace ~80 lines of local `memRepo` with `testutil/mock.UTXORepo` |
| `internal/handler/api/block_handler_test.go` | Replace ~80 lines of local `mockChainRepo` with `testutil/mock.ChainRepo` |
| `internal/domain/p2p/server_test.go` | Replace ~60 lines of local `MockChainRepo` with `testutil/mock.ChainRepo` |

### Existing Production Code NOT Modified

Zero changes to any file outside `*_test.go` files and the new `testutil/` package. The existing interface-based DI already supports testing without modification.

## Test Organization Per Layer

| Layer | Test Type | Isolation | Speed | Pattern |
|-------|-----------|-----------|-------|---------|
| `domain/block` | Unit | No deps | <1ms/test | Table-driven, pure functions |
| `domain/chain` | Unit | Mock repos | ~100ms/test (PoW mining) | `testutil.NewTestChain` |
| `domain/tx` | Unit | No deps | <1ms/test | Table-driven |
| `domain/utxo` | Unit | Mock repo | <1ms/test | Seed + assert |
| `domain/wallet` | Unit | No deps | <5ms/test (key gen) | Pure functions |
| `domain/mempool` | Unit | Mock UTXO repo | <5ms/test (signing) | `testutil.BuildSignedTx` |
| `domain/p2p` | Unit + Integration | `net.Pipe`, port 0 | ~200ms/test (TCP) | `makeTestServer` |
| `domain/events` | Unit | No deps | <1ms/test | Channel assertions |
| `handler/api` | Unit | Mock repos | <1ms/test | httptest + pathvar |
| `handler/ws` | Unit | Real Bus | <50ms/test | Channel + Eventually |
| `handler/cli` | Unit | Mock ServiceContext | ~100ms/test | Direct function calls |
| `infrastructure/bbolt` | Integration | Real BoltDB in TempDir | ~50ms/test | suite.Suite |
| `infrastructure/jsonfile` | Integration | Real files in TempDir | <10ms/test | t.TempDir |
| `svc` | Integration | Real BoltDB in TempDir | ~100ms/test | t.TempDir + Cleanup |

## Sources

- Existing codebase analysis: 23 test files across 15 packages - HIGH confidence (direct code inspection)
- Go testing patterns from `testing` package stdlib documentation - HIGH confidence
- testify library (already in go.mod: `github.com/stretchr/testify`) for assertions, require, mock, suite - HIGH confidence
- go-zero `pathvar.WithVars` for route parameter injection in handler tests - HIGH confidence (already used in codebase)
- `net.Pipe()` and `httptest` patterns from Go stdlib - HIGH confidence
