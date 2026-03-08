# Phase 16: Infrastructure Persistence Tests - Research

**Researched:** 2026-03-08
**Domain:** Go persistence testing (BoltDB + JSON file I/O)
**Confidence:** HIGH

## Summary

Phase 16 targets two persistence packages: `bbolt` (BoltDB-backed chain and UTXO repositories) and `jsonfile` (JSON file-backed wallet repository). Both packages already have test suites with meaningful coverage, but significant gaps remain.

**BoltDB (`bbolt`) currently at 55.7%**, target 80%. The major uncovered areas are: `SaveBlockWithUTXOs` (0% -- the atomic block+UTXO save), `DeleteBlocksAbove` (0% -- reorg deletion), `GetUndoEntry` on BboltRepository (0%), `DeleteUndoEntry` on UTXORepo (0%), and `TxModelFromDomain`/`TxModel.ToDomain` storage model conversions (0%). These are all critical correctness paths.

**JSON file (`jsonfile`) currently at 82.5%**, target 90%. The uncovered paths are error branches in `NewWalletRepo` (corrupt JSON, unreadable file) and `flush` (directory creation errors, write failures).

**Primary recommendation:** Add tests for all 0%-covered functions in bbolt (especially `SaveBlockWithUTXOs` and `DeleteBlocksAbove` which are the reorg-critical atomic operations), then fill error-path gaps in jsonfile. All tests must use `t.TempDir()` for isolation and pass with `go test -count=2`.

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|-----------------|
| INFR-01 | BoltDB repository test coverage reaches 80%+ (atomic block+UTXO saves, range queries, reorg deletes, undo entries) | Coverage analysis shows 55.7% baseline; uncovered: SaveBlockWithUTXOs, DeleteBlocksAbove, GetUndoEntry (chain), DeleteUndoEntry (utxo), TxModel conversions |
| INFR-02 | JSON file wallet repository test coverage reaches 90%+ | Coverage at 82.5%; uncovered: NewWalletRepo error paths (corrupt file, bad permissions), flush error paths |
</phase_requirements>

## Standard Stack

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| go.etcd.io/bbolt | (project dep) | Embedded key-value store | Already used in production code |
| testing (stdlib) | Go 1.26 | Test framework | Standard Go testing |
| github.com/stretchr/testify | (project dep) | Assertions (require/assert) and suite runner | Already used in existing tests |

### Supporting
| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| testutil (internal) | n/a | MustCreateBlock, MustCreateBlockWithAddr, MustCreateBlockChain, MustBuildSignedTx | Building blocks with transactions for SaveBlockWithUTXOs tests |
| os (stdlib) | Go 1.26 | File permission manipulation, temp files | jsonfile error path tests |

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| testify suite (current) | Plain t.Run subtests | Existing tests use suite pattern; keep consistent |
| External test fixtures | t.TempDir() | t.TempDir() preferred per success criteria requirement |

## Architecture Patterns

### Existing Test Structure (follow it)
```
internal/infrastructure/persistence/
  bbolt/
    chain_repo.go           # BboltRepository (chain.Repository)
    chain_repo_test.go      # ChainRepoSuite (testify suite)
    utxo_repo.go            # UTXORepo (utxo.Repository)
    utxo_repo_test.go       # UTXORepoSuite (testify suite)
    storage_model.go        # BlockModel, TxModel, HeaderModel conversions
    utxo_storage_model.go   # UTXOModel conversions
  jsonfile/
    wallet_repo.go          # WalletRepo (wallet.Repository)
    wallet_repo_test.go     # Flat tests with t.TempDir()
```

### Pattern 1: Suite Setup with Fresh DB
**What:** Each test gets a fresh bbolt database via `t.TempDir()` in `SetupTest()`
**When to use:** All bbolt tests (already established)
**Example:**
```go
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

### Pattern 2: Blocks with Transactions for Atomic Save Tests
**What:** Use testutil.MustCreateBlockWithAddr to create blocks that contain coinbase transactions, enabling SaveBlockWithUTXOs testing
**When to use:** Testing SaveBlockWithUTXOs which needs blocks with real `*tx.Transaction` objects in `RawTransactions()`
**Key insight:** The testutil builders already create blocks with coinbase transactions as `[]any{coinbaseTx}`. These blocks will have `RawTransactions()` returning the tx objects needed for the UTXO extraction logic in `SaveBlockWithUTXOs`.

### Pattern 3: Error Path Testing in jsonfile
**What:** Create corrupt files or use read-only directories to exercise error branches
**When to use:** Testing NewWalletRepo with bad JSON, flush with unwritable paths
**Example:**
```go
func TestWalletRepo_CorruptFile(t *testing.T) {
    tmpDir := t.TempDir()
    filePath := filepath.Join(tmpDir, "wallets.json")
    os.WriteFile(filePath, []byte("{invalid json"), 0644)
    _, err := NewWalletRepo(filePath)
    require.Error(t, err)
}
```

### Anti-Patterns to Avoid
- **Shared state between tests:** Each test must create its own DB/file via t.TempDir(). Never reuse a database across tests.
- **Mining at high difficulty:** Use `testutil.TestDifficultyBits` (1) for fast block creation. The existing suite tests use bits=8 which is still fast but testutil builders are faster.
- **Testing internal key formats directly:** Test through the public repository interface, not by inspecting raw bbolt bucket contents.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Test blocks with transactions | Manual block construction | `testutil.MustCreateBlockWithAddr` | Already creates coinbase tx with correct merkle root |
| Test block chains | Manual prev-hash chaining | `testutil.MustCreateBlockChain` | Handles genesis + linking automatically |
| Test wallets | Manual key generation | `testutil.MustCreateWallet` or `wallet.NewWallet()` | Generates valid secp256k1 keys |
| Signed transactions | Manual ECDSA signing | `testutil.MustBuildSignedTx` | Handles UTXO lookup, input/output creation, signing |

## Common Pitfalls

### Pitfall 1: bbolt Byte Slice Lifetime
**What goes wrong:** Reading a value from bbolt inside a View transaction, then using the byte slice after the transaction closes -- data becomes invalid.
**Why it happens:** bbolt returns direct pointers to mmap'd data; the slice is only valid during the transaction.
**How to avoid:** Always copy byte slices before the transaction function returns (the production code already does this correctly).
**Warning signs:** Intermittent test failures, corrupt data in assertions.

### Pitfall 2: SaveBlockWithUTXOs Requires UndoEntry with Valid Hex TxIDs
**What goes wrong:** Creating an UndoEntry with invalid hex strings in `Spent[].TxID` causes `block.HashFromHex` to fail inside SaveBlockWithUTXOs.
**Why it happens:** The function parses `Spent[].TxID` as hex to build utxo keys for deletion.
**How to avoid:** Use real transaction IDs (from `tx.ID().String()`) or valid 64-char hex strings in test UndoEntry Spent items.
**Warning signs:** "parse spent txid" error from SaveBlockWithUTXOs.

### Pitfall 3: jsonfile Error Path Requires OS-Level Manipulation
**What goes wrong:** Cannot test `flush()` write errors without filesystem-level setup.
**Why it happens:** flush writes to a temp file then renames -- both can fail on permission issues.
**How to avoid:** Use `os.Chmod` on the parent directory to make it read-only (works on macOS/Linux). Remember to restore permissions in cleanup to avoid `t.TempDir()` cleanup failures.
**Warning signs:** Tests that skip error paths, leaving coverage at 73.3% for flush.

### Pitfall 4: DeleteBlocksAbove on Empty Chain
**What goes wrong:** Assuming DeleteBlocksAbove will error on empty chain -- it actually returns nil (early return when no height metadata exists).
**How to avoid:** Test this case explicitly and assert no error, not an error.

## Code Examples

### SaveBlockWithUTXOs Test Pattern
```go
func (s *ChainRepoSuite) TestSaveBlockWithUTXOs() {
    ctx := context.Background()
    // Use testutil to create a block with a coinbase transaction
    b := testutil.MustCreateBlock(s.T(), 0, block.Hash{})

    // Get the coinbase tx to build the undo entry
    coinbaseTx := b.RawTransactions()[0].(*tx.Transaction)

    undoEntry := &utxo.UndoEntry{
        BlockHeight: b.Height(),
        Spent:       []utxo.SpentUTXO{}, // Genesis has no spent inputs
        Created: []utxo.UTXORef{
            {TxID: coinbaseTx.ID().String(), Vout: 0},
        },
    }

    err := s.repo.SaveBlockWithUTXOs(ctx, b, undoEntry)
    s.Require().NoError(err)

    // Verify block was saved
    got, err := s.repo.GetBlock(ctx, b.Hash())
    s.Require().NoError(err)
    s.Assert().Equal(b.Hash(), got.Hash())

    // Verify undo entry was saved (via chain repo's GetUndoEntry)
    gotUndo, err := s.repo.GetUndoEntry(ctx, b.Height())
    s.Require().NoError(err)
    s.Assert().Equal(b.Height(), gotUndo.BlockHeight)
}
```

### DeleteBlocksAbove Test Pattern
```go
func (s *ChainRepoSuite) TestDeleteBlocksAbove() {
    ctx := context.Background()
    blocks := s.createChain(5) // heights 0-4

    // Delete blocks above height 2
    err := s.repo.DeleteBlocksAbove(ctx, 2)
    s.Require().NoError(err)

    // Blocks 0-2 should still exist
    for i := uint64(0); i <= 2; i++ {
        _, err := s.repo.GetBlockByHeight(ctx, i)
        s.Require().NoError(err, "block at height %d should exist", i)
    }

    // Blocks 3-4 should be gone
    for i := uint64(3); i <= 4; i++ {
        _, err := s.repo.GetBlockByHeight(ctx, i)
        s.Require().ErrorIs(err, chain.ErrBlockNotFound, "height %d", i)
    }

    // Chain height should be 2
    height, err := s.repo.GetChainHeight(ctx)
    s.Require().NoError(err)
    s.Assert().Equal(uint64(2), height)

    // Latest block should be at height 2
    latest, err := s.repo.GetLatestBlock(ctx)
    s.Require().NoError(err)
    s.Assert().Equal(blocks[2].Hash(), latest.Hash())
}
```

### TxModel Round-Trip Test Pattern
```go
func TestTxModelRoundTrip(t *testing.T) {
    w := testutil.MustCreateWallet(t)
    // Create a coinbase tx
    coinbase := tx.NewCoinbaseTxWithHeight(w.Address(), 5000000000, 0)

    model := TxModelFromDomain(coinbase)
    restored, err := model.ToDomain()
    require.NoError(t, err)

    assert.Equal(t, coinbase.ID(), restored.ID())
    assert.Len(t, restored.Outputs(), len(coinbase.Outputs()))
}
```

## Coverage Gap Analysis

### bbolt Package (55.7% -> 80%+ target)

| Function | Current | Gap | Test Needed |
|----------|---------|-----|-------------|
| SaveBlockWithUTXOs | 0% | Full function | Atomic save with block+UTXOs+undo, verify all three stored |
| DeleteBlocksAbove | 0% | Full function | Delete middle of chain, verify metadata updated, empty chain case |
| GetUndoEntry (chain_repo) | 0% | Full function | Save via SaveBlockWithUTXOs, retrieve via GetUndoEntry |
| DeleteUndoEntry (utxo_repo) | 0% | Full function | Save then delete undo entry |
| TxModelFromDomain | 0% | Full function | Coinbase tx round-trip, signed tx round-trip |
| TxModel.ToDomain | 0% | Full function | Covered by round-trip test above |
| BlockModelFromDomain (with txs) | 66.7% | Tx branch | Block with transactions (testutil blocks have txs) |
| BlockModel.ToDomain (with txs) | 61.1% | Tx branch | Covered by SaveBlockWithUTXOs retrieval |
| NewBboltRepository | 61.5% | Error branches | Hard to trigger; bucket creation errors are unlikely with real bbolt |
| NewUTXORepo | 66.7% | Error branches | Same -- consider acceptable |

### jsonfile Package (82.5% -> 90%+ target)

| Function | Current | Gap | Test Needed |
|----------|---------|-----|-------------|
| NewWalletRepo | 80% | Corrupt JSON file, non-existent parent dir read error | Write invalid JSON, test recovery |
| flush | 73.3% | Write error paths (unwritable dir, rename failure) | Chmod dir to read-only, attempt save |

## Validation Architecture

### Test Framework
| Property | Value |
|----------|-------|
| Framework | Go testing + testify v1.x (suite + require/assert) |
| Config file | None needed -- Go conventions |
| Quick run command | `go test -cover ./internal/infrastructure/persistence/...` |
| Full suite command | `go test -cover -count=2 ./internal/infrastructure/persistence/...` |

### Phase Requirements -> Test Map
| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| INFR-01 | Atomic block+UTXO saves | integration | `go test ./internal/infrastructure/persistence/bbolt/ -run TestChainRepoSuite/TestSaveBlockWithUTXOs -v` | Partial (file exists, test missing) |
| INFR-01 | Reorg deletes | integration | `go test ./internal/infrastructure/persistence/bbolt/ -run TestChainRepoSuite/TestDeleteBlocksAbove -v` | Partial |
| INFR-01 | Undo entries | integration | `go test ./internal/infrastructure/persistence/bbolt/ -run TestChainRepoSuite/TestGetUndoEntry -v` | Partial |
| INFR-01 | Range queries | integration | `go test ./internal/infrastructure/persistence/bbolt/ -run TestChainRepoSuite/TestGetBlocksInRange -v` | Exists |
| INFR-01 | Storage model round-trips | unit | `go test ./internal/infrastructure/persistence/bbolt/ -run TestTxModel -v` | Missing |
| INFR-01 | UTXORepo DeleteUndoEntry | integration | `go test ./internal/infrastructure/persistence/bbolt/ -run TestUTXORepoSuite/TestDeleteUndoEntry -v` | Missing |
| INFR-02 | Wallet repo error paths | unit | `go test ./internal/infrastructure/persistence/jsonfile/ -run TestWalletRepo_Corrupt -v` | Missing |
| INFR-02 | Flush error paths | unit | `go test ./internal/infrastructure/persistence/jsonfile/ -run TestWalletRepo_FlushError -v` | Missing |

### Sampling Rate
- **Per task commit:** `go test -cover ./internal/infrastructure/persistence/...`
- **Per wave merge:** `go test -cover -count=2 ./internal/infrastructure/persistence/...`
- **Phase gate:** Full suite green + coverage thresholds met

### Wave 0 Gaps
None -- existing test infrastructure (test files, testify, testutil builders) covers all needs. No new framework or config needed.

## Open Questions

1. **NewBboltRepository/NewUTXORepo error branches**
   - What we know: These error branches cover `CreateBucketIfNotExists` failures, which are extremely hard to trigger with real bbolt (only on disk full or corruption).
   - What's unclear: Whether these need testing for the 80% target.
   - Recommendation: Skip these error branches. The 80% target is achievable without them by covering all 0% functions. These are defensive error wrapping, not business logic.

## Sources

### Primary (HIGH confidence)
- Direct code analysis of production files in `internal/infrastructure/persistence/`
- `go test -coverprofile` output showing exact per-function coverage
- Existing test files showing established patterns (testify suite, t.TempDir())
- `internal/testutil/builders.go` showing available test helpers

### Secondary (MEDIUM confidence)
- bbolt documentation on byte slice lifetime (well-known pitfall, already handled in production code)

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH - all libraries already in use, no new dependencies
- Architecture: HIGH - following existing patterns exactly
- Pitfalls: HIGH - based on direct code reading and known bbolt behavior
- Coverage gaps: HIGH - measured with `go test -coverprofile`

**Research date:** 2026-03-08
**Valid until:** 2026-04-08 (stable -- no library changes expected)
