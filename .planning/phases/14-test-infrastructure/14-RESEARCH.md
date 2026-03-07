# Phase 14: Test Infrastructure - Research

**Researched:** 2026-03-08
**Domain:** Go test helpers, mock repositories, builder pattern for domain objects
**Confidence:** HIGH

## Summary

Phase 14 creates a shared `internal/testutil/` package containing two things: (1) builder functions for domain objects (blocks, transactions, wallets, UTXOs) and (2) consolidated mock implementations of the three repository interfaces (chain.Repository, utxo.Repository, wallet.Repository).

The codebase currently has **7 test files across 4 packages** that duplicate mock repository implementations. The `mockUTXORepo` alone is copy-pasted in `chain_test.go`, `mempool_test.go` (as `memRepo`), and `p2p/relay_test.go`. The `mockChainRepo` is duplicated across `chain_test.go`, `p2p/relay_test.go` (as `fullMockChainRepo`), `p2p/reorg_test.go` (as `reorgMockChainRepo`), `p2p/server_test.go` (as testify mock-based `MockChainRepo`), and `handler/api/block_handler_test.go`. Block creation helpers are duplicated in `handler/api/block_handler_test.go` (`createTestBlock`) and `p2p/reorg_test.go` (`createForkBlocks`).

**Primary recommendation:** Create `internal/testutil/` with three files: `builders.go` (block/tx/wallet/UTXO builders), `mock_chain_repo.go`, `mock_utxo_repo.go`, and `mock_wallet_repo.go`. Then migrate existing tests to import from testutil, deleting package-local duplicates.

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|-----------------|
| TINF-01 | Shared test helpers with reusable block, tx, wallet, and UTXO builders in `internal/testutil/` | Builder functions modeled after existing `createTestBlock`, `createForkBlocks`, `buildSignedTx` patterns; uses `block.ReconstructBlock` for unmined test blocks |
| TINF-02 | Consolidated mock repositories (chain, UTXO, wallet) in shared `testutil` package, replacing duplicated mocks across 4+ packages | In-memory mock implementations unified from 7 test files across chain, mempool, p2p (3 files), and handler/api (2 files) packages |
</phase_requirements>

## Standard Stack

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| Go stdlib `testing` | Go 1.26 | Test framework | Built-in, no dependencies |
| testify | v1.11.1 | Assertions (assert/require) | Already used project-wide |
| btcec/v2 | (existing) | Key generation for wallet builders | Already a project dependency |

### Supporting
No new dependencies needed. The Out of Scope section in REQUIREMENTS.md explicitly states: "No new test dependencies -- existing testify + stdlib covers all needs."

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| Hand-written mocks | mockgen/moq | REQUIREMENTS.md explicitly excludes: "Only 3 interfaces; hand-written mocks are established and sufficient" |

**Installation:**
```bash
# No new packages needed
```

## Architecture Patterns

### Recommended Project Structure
```
internal/testutil/
  builders.go           # Block, tx, wallet, UTXO builder functions
  mock_chain_repo.go    # In-memory chain.Repository mock
  mock_utxo_repo.go     # In-memory utxo.Repository mock
  mock_wallet_repo.go   # In-memory wallet.Repository mock
```

### Pattern 1: Builder Functions with t.Helper()
**What:** Functions that create pre-configured domain objects for tests, using `t.Helper()` to report errors at the caller's line.
**When to use:** Any test needing blocks, transactions, UTXOs, or wallets.
**Example:**
```go
// Source: Modeled after existing createTestBlock in handler/api/block_handler_test.go
package testutil

func MustCreateBlock(t *testing.T, height uint64, prevHash block.Hash) *block.Block {
    t.Helper()
    coinbase := tx.NewCoinbaseTxWithHeight("1TestAddr", 5000000000, height)
    blockTxs := []any{coinbase}
    merkleRoot := block.ComputeMerkleRoot([]block.Hash{coinbase.ID()})

    var b *block.Block
    var err error
    if height == 0 {
        b, err = block.NewGenesisBlock("test genesis", 1, blockTxs, merkleRoot)
    } else {
        b, err = block.NewBlock(prevHash, height, 1, blockTxs, merkleRoot)
    }
    require.NoError(t, err)

    pow := &block.ProofOfWork{}
    require.NoError(t, pow.Mine(b))
    return b
}
```

### Pattern 2: In-Memory Mock with Mutex (Thread-Safe)
**What:** Mock repositories using `sync.Mutex` for thread safety, matching the pattern in `chain_test.go`.
**When to use:** Tests involving concurrent chain operations (mining, P2P sync).
**Example:**
```go
// Source: Unified from chain_test.go and p2p/relay_test.go patterns
package testutil

type MockChainRepo struct {
    mu       sync.RWMutex
    blocks   map[block.Hash]*block.Block
    byHeight map[uint64]*block.Block
    undos    map[uint64]*utxo.UndoEntry
    latest   *block.Block
}

func NewMockChainRepo() *MockChainRepo {
    return &MockChainRepo{
        blocks:   make(map[block.Hash]*block.Block),
        byHeight: make(map[uint64]*block.Block),
        undos:    make(map[uint64]*utxo.UndoEntry),
    }
}
```

### Pattern 3: Testify Mock-Based Variant Not Needed
**What:** The `p2p/server_test.go` uses a testify `mock.Mock`-based `MockChainRepo`, but this is only for simple stub behavior where `On().Return()` is convenient. The in-memory map-based mock covers all use cases and is simpler.
**When to use:** The shared mock should use the map-based approach (covers both simple stubs and stateful scenarios). Tests currently using testify mocks can switch to the map-based shared mock.

### Anti-Patterns to Avoid
- **Exporting mock internals:** Mock fields like `blocks` map should be unexported; provide `AddBlock()` helper method instead for test setup
- **Over-parameterizing builders:** Keep builder functions simple with sensible defaults; use functional options only if complexity truly warrants it (it does not for this project)
- **Breaking existing test APIs:** When migrating, keep the same function signatures where possible to minimize churn

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Mock generation | Code-gen mocks with mockgen | Hand-written in-memory mocks | REQUIREMENTS.md decision; only 3 interfaces |
| Test assertions | Custom assert helpers | testify assert/require | Already adopted project-wide |

**Key insight:** The project has exactly 3 repository interfaces to mock. Hand-written mocks with real in-memory storage are superior here because they enable stateful tests (add block, query it back) which testify `mock.Mock` cannot do cleanly.

## Common Pitfalls

### Pitfall 1: Import Cycles with testutil
**What goes wrong:** `internal/testutil/` imports domain packages (block, tx, utxo, chain, wallet). If any domain package tries to import testutil, you get an import cycle.
**Why it happens:** Go forbids circular imports.
**How to avoid:** testutil is test-only infrastructure. It imports domain packages but no domain package imports it. Test files in domain packages use `_test` package suffix and can import testutil freely because `_test` packages have relaxed import rules.
**Warning signs:** Compilation error mentioning "import cycle."

### Pitfall 2: Package Name for External Tests
**What goes wrong:** Some existing test files use internal package access (e.g., `package chain` not `package chain_test`). These files can access unexported fields. If the shared mock needs unexported access, it cannot be in testutil.
**Why it happens:** Go test files in `package X` can access unexported symbols; `package X_test` cannot.
**How to avoid:** All three repository interfaces are fully public. All existing mock implementations use only public methods. The shared mocks will work fine from an external package. Existing tests already use `_test` suffix (verified: `chain_test`, `mempool` uses internal package but mock only uses public API).
**Warning signs:** Check that `mempool_test.go` uses `package mempool` (internal) -- the `memRepo` there only uses public UTXO API, so switching to testutil import is safe.

### Pitfall 3: Thread Safety Differences Between Mocks
**What goes wrong:** The chain_test.go mock uses `sync.RWMutex` but the mempool and p2p mocks do not. Consolidating to a single mock must use the thread-safe version.
**Why it happens:** P2P tests run concurrent goroutines; mempool tests are sequential.
**How to avoid:** Always use the mutex-protected version in the shared mock. The overhead is negligible.
**Warning signs:** Race detector failures (`go test -race`) after migration.

### Pitfall 4: Stateful Mock vs Stub Mock Mismatch
**What goes wrong:** `p2p/server_test.go` uses testify `mock.Mock` for `MockChainRepo`, while all other tests use in-memory map-based mocks. The two patterns are incompatible.
**Why it happens:** Different test authors, different needs.
**How to avoid:** The shared in-memory mock covers both use cases. The `server_test.go` only uses `SaveBlock` and `SaveBlockWithUTXOs` returning nil -- the map-based mock handles this naturally. Migrate `server_test.go` to use the shared mock with pre-seeded genesis block.
**Warning signs:** Test changing from `repo.On("SaveBlock", ...).Return(nil)` pattern.

## Code Examples

Verified patterns from the existing codebase:

### Block Builder (genesis + chain)
```go
// Consolidates: handler/api/block_handler_test.go:createTestBlock
//               p2p/reorg_test.go:createForkBlocks
package testutil

func MustCreateBlock(t *testing.T, height uint64, prevHash block.Hash) *block.Block {
    t.Helper()
    return MustCreateBlockWithAddr(t, height, prevHash, "1TestAddr")
}

func MustCreateBlockWithAddr(t *testing.T, height uint64, prevHash block.Hash, minerAddr string) *block.Block {
    t.Helper()
    coinbase := tx.NewCoinbaseTxWithHeight(minerAddr, 5_000_000_000, height)
    blockTxs := []any{coinbase}
    merkleRoot := block.ComputeMerkleRoot([]block.Hash{coinbase.ID()})

    var b *block.Block
    var err error
    if height == 0 {
        b, err = block.NewGenesisBlock("test genesis", 1, blockTxs, merkleRoot)
    } else {
        b, err = block.NewBlock(prevHash, height, 1, blockTxs, merkleRoot)
    }
    require.NoError(t, err)

    pow := &block.ProofOfWork{}
    require.NoError(t, pow.Mine(b))
    return b
}

func MustCreateBlockChain(t *testing.T, count int) []*block.Block {
    t.Helper()
    blocks := make([]*block.Block, 0, count)
    genesis := MustCreateBlock(t, 0, block.Hash{})
    blocks = append(blocks, genesis)
    for i := 1; i < count; i++ {
        b := MustCreateBlock(t, uint64(i), blocks[i-1].Hash())
        blocks = append(blocks, b)
    }
    return blocks
}
```

### Signed Transaction Builder
```go
// Consolidates: mempool/mempool_test.go:buildSignedTx
func MustBuildSignedTx(t *testing.T, utxoSet *utxo.Set, privKey *btcec.PrivateKey, fromAddr string) *tx.Transaction {
    t.Helper()
    coinbase := tx.NewCoinbaseTx(fromAddr, 5_000_000_000)
    _, err := utxoSet.ApplyBlock(0, []*tx.Transaction{coinbase})
    require.NoError(t, err)

    input := tx.NewTxInput(coinbase.ID(), 0)
    output := tx.NewTxOutput(5_000_000_000, "recipient")
    spendTx := tx.NewTransaction([]tx.TxInput{input}, []tx.TxOutput{output})
    require.NoError(t, tx.SignTransaction(spendTx, privKey))
    return spendTx
}
```

### Mock Chain Repository (thread-safe, in-memory)
```go
// Consolidates: chain_test.go:mockChainRepo, p2p/relay_test.go:fullMockChainRepo,
//               p2p/reorg_test.go:reorgMockChainRepo, p2p/server_test.go:MockChainRepo,
//               handler/api/block_handler_test.go:mockChainRepo
type MockChainRepo struct {
    mu       sync.RWMutex
    Blocks   map[block.Hash]*block.Block   // exported for test inspection
    ByHeight map[uint64]*block.Block
    Undos    map[uint64]*utxo.UndoEntry
    Latest   *block.Block
}
// Implements all 9 methods of chain.Repository
```

### Mock UTXO Repository (thread-safe, in-memory)
```go
// Consolidates: chain_test.go:mockUTXORepo, mempool_test.go:memRepo,
//               p2p/relay_test.go:mockUTXORepo
type MockUTXORepo struct {
    mu    sync.Mutex
    UTXOs map[string]utxo.UTXO
    Undos map[uint64]*utxo.UndoEntry
}
// Implements all 7 methods of utxo.Repository
```

### Mock Wallet Repository
```go
type MockWalletRepo struct {
    mu      sync.Mutex
    Wallets map[string]*wallet.Wallet
}
// Implements all 3 methods of wallet.Repository
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| Copy-paste mocks per test file | Shared testutil package | This phase | Eliminates ~300 lines of duplicated mock code |
| Per-package block creation helpers | Shared builders | This phase | Consistent test data across all packages |

**No deprecated patterns** -- the existing Go testing patterns (t.Helper, t.Cleanup, testify assertions) are current best practice.

## Validation Architecture

### Test Framework
| Property | Value |
|----------|-------|
| Framework | Go testing + testify v1.11.1 |
| Config file | None (Go convention) |
| Quick run command | `go test ./internal/testutil/...` |
| Full suite command | `go test ./...` |

### Phase Requirements to Test Map
| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| TINF-01 | Builder functions compile and produce valid domain objects | unit | `go test ./internal/testutil/ -run TestBuilders -x` | No -- Wave 0 |
| TINF-02 | Shared mocks implement repository interfaces and work in existing tests | integration | `go test ./...` (full suite must still pass after migration) | No -- Wave 0 |

### Sampling Rate
- **Per task commit:** `go test ./internal/testutil/... && go test ./...`
- **Per wave merge:** `go test ./...`
- **Phase gate:** Full suite green before `/gsd:verify-work`

### Wave 0 Gaps
- [ ] `internal/testutil/builders_test.go` -- covers TINF-01 (verify builders produce valid objects)
- [ ] `internal/testutil/mock_chain_repo_test.go` -- covers TINF-02 (verify interface compliance)
- [ ] `internal/testutil/mock_utxo_repo_test.go` -- covers TINF-02
- [ ] `internal/testutil/mock_wallet_repo_test.go` -- covers TINF-02

## Open Questions

1. **Should mock fields be exported or unexported?**
   - What we know: Existing mocks use unexported fields. Some tests (api/block_handler_test.go) use an `addBlock()` helper method on the mock.
   - What's unclear: Whether downstream test phases need direct map access for assertions.
   - Recommendation: Export map fields (e.g., `Blocks`, `UTXOs`) for test inspection convenience, plus provide helper methods like `AddBlock()` for ergonomic setup. This is test-only code; encapsulation matters less than convenience.

2. **Should p2p/server_test.go keep its testify mock.Mock pattern?**
   - What we know: It's the only test using testify's `mock.Mock` for chain repo. It uses `On().Return()` for simple stubs.
   - What's unclear: Whether future p2p tests will need `AssertCalled`/`AssertNumberOfCalls` verification.
   - Recommendation: Migrate to the shared in-memory mock. If call-counting is needed later, add it to the shared mock (simple counter field). The testify mock pattern adds unnecessary complexity for these simple cases.

## Sources

### Primary (HIGH confidence)
- Codebase analysis: All 22 existing test files examined for mock and builder patterns
- Repository interfaces: `chain/repository.go`, `utxo/repository.go`, `wallet/repository.go`
- REQUIREMENTS.md: Out of Scope decisions (no mockgen, no property tests, no testcontainers)

### Secondary (MEDIUM confidence)
- `.agents/skills/golang-testing/SKILL.md` -- Go testing patterns and conventions

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH - No new dependencies, existing testify + stdlib per REQUIREMENTS.md
- Architecture: HIGH - Clear pattern from 7 existing duplicated mocks; straightforward consolidation
- Pitfalls: HIGH - Import cycles and thread safety are well-understood Go constraints

**Research date:** 2026-03-08
**Valid until:** 2026-04-08 (stable domain, no external dependencies changing)
