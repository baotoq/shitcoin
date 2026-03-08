# Phase 15: Domain Layer Coverage - Research

**Researched:** 2026-03-08
**Domain:** Go test coverage for blockchain domain packages
**Confidence:** HIGH

## Summary

Phase 15 requires bringing all domain packages to specific coverage thresholds: chain 85%+, p2p 80%+, and utxo/wallet/mempool/tx each at 95%+. The current coverage baseline is well understood from per-function analysis. The testutil infrastructure (Phase 14) provides shared builders and mock repos. No new libraries are needed -- testify + stdlib covers everything.

The primary challenge is the p2p package (66.9% -> 80%+) which has significant untested handler logic and sync edge cases that require careful test setup with raw connections. The chain package (69.5% -> 85%+) needs coverage for `getCurrentBits` difficulty adjustment and `SetLatestBlock`, plus error paths in `Initialize` and `MineBlock`. The four "easy" packages (86-94% -> 95%) need targeted gap-filling for specific uncovered functions and error branches.

**Primary recommendation:** Attack coverage gaps in order of largest delta first (chain, p2p, then the 95% packages), using table-driven tests for error paths and the existing testutil builders for setup.

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|-----------------|
| DOM-01 | Chain aggregate 85%+ coverage (mining orchestration, reorg, difficulty adjustment) | Coverage analysis shows getCurrentBits at 18.8%, SetLatestBlock at 0%, Initialize branches at 75%, MineBlock at 73.7% -- all gap areas identified with specific functions |
| DOM-02 | P2P 80%+ coverage (message encoding/decoding, handler dispatch, sync logic) | handleTx at 0%, handleGetData at 37.1%, handleMessage at 50%, abortSync at 0%, applySyncBlock at 60%, several handler gaps identified |
| DOM-03 | utxo/wallet/mempool/tx packages each at 95%+ | Per-function gaps: mempool.GetByID 0%, mempool.FeeForTx 80%, wallet.PubKeyHashFromAddress 0%, utxo.UndoBlock 71.4%, tx.SignTransaction 87.5%, tx.VerifyTransaction 80%, tx.ValidateCoinbase 85.7% |
| DOM-04 | Error path tests for invalid blocks, double spends, corrupt data, nil inputs, boundary conditions | Specific error branches identified across all packages; table-driven error path tests planned |
</phase_requirements>

## Standard Stack

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| testing | stdlib | Test framework | Go standard |
| testify | v1.9+ | Assertions (require/assert) | Already used everywhere |
| testutil | internal | Shared builders/mocks | Phase 14 output |

### Supporting
| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| net.Pipe() | stdlib | In-process TCP connections for P2P handler tests | Testing handleMessage/handleTx/handleGetData without real networking |
| io | stdlib | EOF/error simulation | Testing truncated/corrupt payloads |

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| Hand-written mocks | mockgen | Out of scope per REQUIREMENTS.md -- only 3 interfaces, hand-written sufficient |

**Installation:**
```bash
# No new dependencies needed
```

## Architecture Patterns

### Recommended Test File Organization
```
internal/domain/
  chain/
    chain_test.go          # Existing + new tests for Initialize, MineBlock, getCurrentBits, SetLatestBlock
  p2p/
    p2p_test.go            # Existing protocol/peer tests
    server_test.go         # Existing server/handshake tests
    relay_test.go          # Existing relay tests
    sync_test.go           # Existing sync tests
    reorg_test.go          # Existing reorg tests
    handler_test.go        # NEW: unit tests for handleMessage, handleTx, handleGetData, handleGetBlocks
    payload_test.go        # NEW: unit tests for ToBlock/ToTransaction error paths
  tx/
    transaction_test.go    # Existing + new error path tests
  utxo/
    set_test.go            # Existing + new UndoBlock error path tests
  wallet/
    wallet_test.go         # Existing + new PubKeyHashFromAddress tests
    base58_test.go         # Existing + new edge case tests
  mempool/
    mempool_test.go        # Existing + new GetByID/FeeForTx/Remove edge case tests
```

### Pattern 1: Table-Driven Error Path Tests
**What:** Group all error cases for a function into a single table-driven test.
**When to use:** Any function with multiple error returns (validators, deserializers).
**Example:**
```go
func TestValidateCoinbase_ErrorCases(t *testing.T) {
    tests := []struct {
        name    string
        tx      *tx.Transaction
        reward  int64
        wantErr error
    }{
        {"not coinbase", regularTx, 1000, tx.ErrInvalidCoinbase},
        {"wrong reward", coinbaseTx, 999, tx.ErrInvalidCoinbase},
        // multi-output coinbase if constructible
    }
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            err := tx.ValidateCoinbase(tt.tx, tt.reward)
            require.ErrorIs(t, err, tt.wantErr)
        })
    }
}
```

### Pattern 2: P2P Handler Unit Testing via dialAndHandshake
**What:** Use the existing `dialAndHandshake` helper to send raw P2P messages and verify server behavior.
**When to use:** Testing handleTx, handleGetData, handleBlock error paths.
**Example:**
```go
// Send invalid tx payload, verify it's rejected (no crash, mempool unchanged)
invalidPayload := p2p.TxPayload{ID: "invalid-hex"}
msg, _ := p2p.NewMessage(p2p.CmdTx, invalidPayload)
p2p.WriteMessage(conn, msg)
time.Sleep(200 * time.Millisecond)
assert.Equal(t, 0, pool.Count())
```

### Pattern 3: Chain Difficulty Adjustment Testing
**What:** Use MockChainRepo with pre-seeded blocks at specific heights/timestamps to test getCurrentBits.
**When to use:** Testing difficulty adjustment interval boundary conditions.
**Example:**
```go
cfg := chain.ChainConfig{
    DifficultyAdjustInterval: 10,
    BlockTimeTarget:          600,
    InitialDifficulty:        1,
    BlockReward:              5000000000,
}
// Mine exactly 10 blocks to trigger adjustment
```

### Anti-Patterns to Avoid
- **Testing internal package functions from external test packages:** The tx package uses internal test (package tx, not tx_test). Keep it that way for validator tests needing unexported field access via `&Transaction{inputs: ..., outputs: ...}`.
- **Sleeping for sync:** Prefer `require.Eventually` over `time.Sleep` for sync-dependent assertions (per STATE.md blocker note).
- **Creating real TCP listeners when net.Pipe suffices:** For handler unit tests, prefer pipe-based connections.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Mock repositories | Custom mocks per test | testutil.NewMockChainRepo/UTXORepo/WalletRepo | Already centralized in Phase 14 |
| Signed test transactions | Manual key gen + signing | testutil.MustBuildSignedTx | Handles UTXO setup, signing in one call |
| Block chain construction | Manual block linking | testutil.MustCreateBlockChain | Correct hash linkage, mining |

## Common Pitfalls

### Pitfall 1: P2P Tests That Depend on Timing
**What goes wrong:** Tests use `time.Sleep(200*time.Millisecond)` and become flaky under load.
**Why it happens:** P2P handlers run in goroutines; message propagation time varies.
**How to avoid:** Use `require.Eventually(t, func() bool { ... }, 5*time.Second, 100*time.Millisecond)` for assertions on async state changes.
**Warning signs:** Tests pass locally but fail in CI.

### Pitfall 2: Internal vs External Test Packages
**What goes wrong:** Tests can't access unexported fields needed for error path construction.
**Why it happens:** Some error paths require constructing invalid structs (e.g., `Transaction` with tampered fields).
**How to avoid:** tx package tests are already `package tx` (internal). Keep them that way. chain, mempool, wallet, utxo, p2p tests use `package X_test` (external) which is fine since they test through public API.
**Warning signs:** Can't construct invalid input for error path testing.

### Pitfall 3: Chain Tests Requiring Real Mining
**What goes wrong:** Tests are slow because PoW mining at difficulty > 1 takes noticeable time.
**Why it happens:** Using production difficulty values in tests.
**How to avoid:** Always use `InitialDifficulty: 1` (bits=1) in test configs, matching the Phase 14 convention.
**Warning signs:** Individual test takes > 2 seconds.

### Pitfall 4: Shared Mempool State Between Sub-Tests
**What goes wrong:** UTXO tracking in mempool carries over between sub-tests, causing unexpected double-spend errors.
**Why it happens:** Tests reuse the same mempool/utxo set instance.
**How to avoid:** Create fresh mempool + utxo set per sub-test, or use `t.Run` with isolated setup.

## Coverage Gap Analysis

### Chain Package (69.5% -> 85%+)

| Function | Current | Gap Reason | Test Needed |
|----------|---------|------------|-------------|
| `getCurrentBits` | 18.8% | No test at difficulty adjustment interval boundaries | Test with DifficultyAdjustInterval=10, mine 10 blocks, verify bits change |
| `SetLatestBlock` | 0.0% | Never called directly in tests | Simple test: set and read back via LatestBlock() |
| `Initialize` (error paths) | 75.0% | Missing: genesis creation error, empty miner address case | Test Initialize with no miner, test repo error during save |
| `MineBlock` (error paths) | 73.7% | Missing: nil latestBlock, MineWithProgress path, SaveBlockWithUTXOs error | Test MineBlock before Initialize, test with OnMiningProgress callback |
| `Reorganize` (edge cases) | 75.0% | Missing: nil latestBlock, invalid PoW in fork block | Test Reorganize with invalid fork blocks |

**Estimated new tests needed:** ~10-12 test functions

### P2P Package (66.9% -> 80%+)

| Function | Current | Gap Reason | Test Needed |
|----------|---------|------------|-------------|
| `handleTx` | 0.0% | No test sends CmdTx via raw connection | Send valid tx, invalid tx, mempool-rejected tx via raw conn |
| `handleGetData` | 37.1% | "tx" type path untested, error paths untested | Send getdata for block and tx types, test not-found cases |
| `handleMessage` | 50.0% | Default/unknown command, CmdVersion after handshake untested | Send unknown command byte, unexpected version after handshake |
| `BroadcastTx` | 0.0% | Never tested directly | Test with mock peers, verify inv message sent |
| `OnBlockReceived` | 0.0% | Callback never exercised in tests | Test callback invocation when block received |
| `removePeer` | 0.0% | Never tested directly | Test peer removal after protocol violation |
| `abortSync` | 0.0% | Never tested directly | Test abort during sync error paths |
| `applySyncBlock` (error paths) | 60.0% | UTXO apply error, save error paths | Test with failing mock repos during sync |
| `ToTransaction` (error paths) | 58.3% | Invalid hex in signature/pubkey | Test with corrupt hex strings |
| `ToBlock` (error paths) | 77.8% | Invalid merkle root hex | Test with invalid hash strings |
| `NewMessage` (error path) | 75.0% | Unmarshalable payload | Test with channel type (not JSON-marshalable) |
| `WriteMessage` (error path) | 70.0% | Write to closed connection | Test write after conn.Close() |
| `Peer.Send` (error path) | 50.0% | Send to stopped peer | Test send after peer.Stop() |
| `peer.writeLoop` | 60.0% | Context cancellation | Tested indirectly |

**Estimated new tests needed:** ~15-18 test functions

### Mempool Package (90.9% -> 95%+)

| Function | Current | Gap Reason | Test Needed |
|----------|---------|------------|-------------|
| `GetByID` | 0.0% | Never tested | Test found and not-found cases |
| `FeeForTx` | 80.0% | Not-found path covered, found path missing verification | Already tested via TestAddStoresFee, but need explicit not-found test |
| `Remove` | 90.0% | Remove non-existent txID path | Test removing a tx ID that's not in mempool |

**Estimated new tests needed:** ~3-4 test functions

### TX Package (94.4% -> 95%+)

| Function | Current | Gap Reason | Test Needed |
|----------|---------|------------|-------------|
| `SignTransaction` (coinbase branch) | 87.5% | Coinbase early return implicitly tested via VerifyCoinbase, but SignTransaction(coinbase, key) not explicit | Explicit test: sign coinbase returns nil, no mutation |
| `VerifyTransaction` (empty sig/key) | 80.0% | Missing: empty signature bytes, empty pubkey bytes, invalid sig parse, invalid pubkey parse | Table-driven test with various invalid signature/key combos |
| `ValidateCoinbase` (multi-output) | 85.7% | Missing: coinbase with 2+ outputs | Construct multi-output coinbase, verify error |

**Estimated new tests needed:** ~4-5 test functions

### UTXO Package (86.2% -> 95%+)

| Function | Current | Gap Reason | Test Needed |
|----------|---------|------------|-------------|
| `UndoBlock` (error paths) | 71.4% | Missing: invalid created txid hex, repo Delete error, invalid spent txid hex, repo Put error | Table-driven error path tests with corrupted UndoEntry data |
| `ApplyBlock` (repo errors) | 89.3% | Missing: repo.Put error, repo.Delete error | Test with failing mock repo |
| `GetBalance` (repo error) | 85.7% | Missing: repo.GetByAddress returning error | Test with error-returning mock |

**Estimated new tests needed:** ~5-6 test functions

### Wallet Package (87.6% -> 95%+)

| Function | Current | Gap Reason | Test Needed |
|----------|---------|------------|-------------|
| `PubKeyHashFromAddress` | 0.0% | Never tested | Test valid address decode, invalid checksum, wrong version, wrong payload length |
| `NewWallet` (error path) | 83.3% | btcec.NewPrivateKey error is unreachable in normal conditions | LOW priority -- crypto error not practically testable without mocking stdlib |
| `Base58CheckDecode` (short input) | 91.7% | Missing: input too short (< 5 bytes decoded) | Test with very short Base58 string |

**Estimated new tests needed:** ~4-5 test functions

## Code Examples

### Chain: Testing Difficulty Adjustment
```go
func TestGetCurrentBits_AdjustmentInterval(t *testing.T) {
    repo := testutil.NewMockChainRepo()
    utxoRepo := testutil.NewMockUTXORepo()
    utxoSet := utxo.NewSet(utxoRepo)
    pow := &block.ProofOfWork{}
    cfg := chain.ChainConfig{
        InitialDifficulty:        1,
        GenesisMessage:           "difficulty-test",
        BlockReward:              5000000000,
        DifficultyAdjustInterval: 5,
        BlockTimeTarget:          600,
    }
    ch := chain.NewChain(repo, pow, cfg, utxoSet)
    ctx := context.Background()
    require.NoError(t, ch.Initialize(ctx, "miner"))

    // Mine 5 blocks to trigger first adjustment
    for range 5 {
        _, err := ch.MineBlock(ctx, "miner", nil, 0)
        require.NoError(t, err)
    }
    // Block at height 5 should have adjusted difficulty
    assert.Equal(t, uint64(5), ch.Height())
}
```

### P2P: Testing handleTx via Raw Connection
```go
func TestHandleTx_InvalidPayload(t *testing.T) {
    srvA, chainA, pool, _ := makeRelayTestNode(t, "miner-A")
    conn := dialAndHandshake(t, srvA, chainA)
    defer conn.Close()

    // Send invalid JSON for CmdTx
    badMsg := p2p.Message{Command: p2p.CmdTx, Payload: []byte("not-json")}
    require.NoError(t, p2p.WriteMessage(conn, badMsg))

    time.Sleep(200 * time.Millisecond)
    assert.Equal(t, 0, pool.Count(), "invalid tx should be rejected")
}
```

### Mempool: Testing GetByID
```go
func TestGetByID_Found(t *testing.T) {
    repo := testutil.NewMockUTXORepo()
    utxoSet := utxo.NewSet(repo)
    privKey, _ := btcec.NewPrivateKey()
    spendTx := buildSignedTx(t, utxoSet, privKey, "addr")

    mp := mempool.New(utxoSet)
    require.NoError(t, mp.Add(spendTx))

    found := mp.GetByID(spendTx.ID())
    require.NotNil(t, found)
    assert.Equal(t, spendTx.ID(), found.ID())
}

func TestGetByID_NotFound(t *testing.T) {
    mp := mempool.New(nil)
    found := mp.GetByID(block.DoubleSHA256([]byte("nonexistent")))
    assert.Nil(t, found)
}
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| Duplicated mocks per package | Centralized testutil package | Phase 14 (v1.2) | All new tests use testutil builders/mocks |
| `time.Sleep` for async assertions | `require.Eventually` recommended | Phase 14 decision | More reliable, no flaky timing issues |
| Internal test packages everywhere | External test packages (package X_test) | Phase 14 decision | Better test isolation, forces public API testing |

## Open Questions

1. **NewWallet error path unreachable**
   - What we know: `btcec.NewPrivateKey()` uses crypto/rand internally; errors are extremely unlikely
   - What's unclear: Whether we should aim for 95% ignoring this one unreachable branch
   - Recommendation: Accept 93-94% for wallet package if this is the only gap, or test via init-time monkey-patching (not recommended). The 95% target should account for this.

2. **P2P handler test flakiness**
   - What we know: Existing P2P tests use `time.Sleep` extensively
   - What's unclear: Whether all sleep-based assertions can be converted to `require.Eventually`
   - Recommendation: Use `require.Eventually` for new tests; refactor existing sleeps opportunistically

## Validation Architecture

### Test Framework
| Property | Value |
|----------|-------|
| Framework | Go testing + testify v1.9 |
| Config file | None (Go convention) |
| Quick run command | `go test -cover ./internal/domain/...` |
| Full suite command | `go test -cover ./...` |

### Phase Requirements -> Test Map
| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| DOM-01 | Chain 85%+ coverage | unit | `go test -cover ./internal/domain/chain/ -run Test` | Partial -- chain_test.go exists |
| DOM-02 | P2P 80%+ coverage | unit | `go test -cover ./internal/domain/p2p/ -run Test` | Partial -- multiple test files exist |
| DOM-03 | utxo/wallet/mempool/tx 95%+ | unit | `go test -cover ./internal/domain/utxo/ ./internal/domain/wallet/ ./internal/domain/mempool/ ./internal/domain/tx/` | Partial -- test files exist for all |
| DOM-04 | Error path tests | unit | `go test -v ./internal/domain/... -run "Error\|Invalid\|Reject\|Fail\|Nil\|Corrupt\|Boundary"` | Wave 0 |

### Sampling Rate
- **Per task commit:** `go test -cover ./internal/domain/...`
- **Per wave merge:** `go test -cover ./...`
- **Phase gate:** All domain packages meet coverage thresholds

### Wave 0 Gaps
- [ ] `internal/domain/p2p/handler_test.go` -- covers DOM-02 handler dispatch tests
- [ ] `internal/domain/p2p/payload_test.go` -- covers DOM-02 payload error paths
- None for other packages -- existing test files are extended

## Sources

### Primary (HIGH confidence)
- Direct codebase analysis: `go test -cover -coverprofile` with per-function output
- Source code reading of all domain packages and existing test files
- Phase 14 testutil package (builders.go, mock_*_repo.go)

### Secondary (MEDIUM confidence)
- `.claude/skills/golang-testing/SKILL.md` -- Go testing patterns and conventions

**Confidence breakdown:**
- Standard stack: HIGH - no new libraries, everything verified in codebase
- Architecture: HIGH - patterns derived from existing test code
- Coverage gaps: HIGH - per-function coverage data from `go tool cover`
- Pitfalls: HIGH - observed from existing test patterns in codebase

**Research date:** 2026-03-08
**Valid until:** 2026-04-08 (stable -- no dependency changes expected)
