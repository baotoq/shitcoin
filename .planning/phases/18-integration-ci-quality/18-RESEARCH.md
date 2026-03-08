# Phase 18: Integration & CI Quality - Research

**Researched:** 2026-03-08
**Domain:** Go integration testing, race detection, CI pipeline configuration
**Confidence:** HIGH

## Summary

Phase 18 requires three deliverables: (1) P2P integration tests verifying TCP handshake, block sync, and transaction relay between 2+ in-process nodes, (2) E2E chain scenario tests covering the full wallet-to-balance workflow, and (3) enabling `-race` flag in the existing GitHub Actions CI pipeline.

The codebase already has strong foundations for all three. The P2P package has `makeTestServer` in `server_test.go` that creates in-process servers with OS-assigned ports and mock repos -- this pattern directly extends to multi-node integration tests. The `testutil` package provides `MustCreateBlock`, `MustCreateBlockChain`, `MustCreateWallet`, and `MustBuildSignedTx` builders. Running `go test -race ./...` today reveals exactly **one data race** in `TestHub_DropsMessageWhenClientFull` (ws package) -- the Hub's `Run()` loop deletes from the clients map under RLock in the eviction path (line 64 of hub.go), which races with `require.Eventually` reading `hub.clients` under RLock from the test goroutine. All other packages pass clean.

**Primary recommendation:** Fix the single ws.Hub race (delete-under-RLock in broadcast eviction), write integration tests using the existing `makeTestServer` pattern extended to 2-node topologies, and add `-race` to the CI workflow's `go test` command.

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|-----------------|
| INTG-01 | P2P integration tests verify TCP handshake, block sync, and tx relay between 2+ in-process nodes | Existing `makeTestServer` helper creates fully wired servers with mock repos and OS-assigned ports. Extend to 2-node topology with shared genesis, use `require.Eventually` for async assertions. |
| INTG-02 | E2E chain scenario tests verify full workflow: create wallet, send tx, mine block, verify UTXO updated, check balance | `testutil.MustCreateWallet`, `MustBuildSignedTx`, chain.Initialize/MineBlock, and utxo.Set.GetByAddress provide all building blocks. Single-function test wiring chain+utxo+mempool+wallet. |
| TINF-03 | Race detection enabled in CI (`go test -race ./...` in GitHub Actions) | Current CI runs `go test -coverprofile=coverage.out ./...` without `-race`. One race exists in ws.Hub eviction path -- fix it, then add `-race` flag. |
</phase_requirements>

## Standard Stack

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| testing (stdlib) | Go 1.26.1 | Test framework | Standard Go testing |
| testify | v1.11.1 | Assertions (require/assert) | Already used project-wide |
| net (stdlib) | Go 1.26.1 | TCP connections for P2P tests | In-process TCP testing |

### Supporting
| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| testutil (internal) | - | Mock repos, builders | All integration test setup |
| require.Eventually | testify | Async assertion with polling | P2P message propagation waits |

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| In-process TCP | testcontainers | Overkill -- BoltDB is embedded, P2P is localhost (explicitly out of scope per REQUIREMENTS.md) |
| time.Sleep | require.Eventually | Eventually is already the project standard (decision from Phase 15) |

**Installation:**
No new dependencies needed. Everything is already in the project.

## Architecture Patterns

### Recommended Test File Structure
```
internal/
  integration/
    integration_test.go     # P2P multi-node tests (INTG-01)
    e2e_chain_test.go       # E2E chain scenario tests (INTG-02)
```

### Pattern 1: Multi-Node P2P Integration Test
**What:** Spin up 2+ P2P servers in-process with shared genesis, verify handshake/sync/relay over real TCP.
**When to use:** INTG-01 tests.
**Example:**
```go
// Based on existing makeTestServer pattern in p2p/server_test.go
func setupTwoNodeNetwork(t *testing.T) (srvA *p2p.Server, srvB *p2p.Server) {
    t.Helper()
    // Both nodes need same miner address -> same genesis hash
    minerAddr := "integration-miner"

    // Node A: chain + utxo + mempool + p2p server
    repoA := testutil.NewMockChainRepo()
    utxoRepoA := testutil.NewMockUTXORepo()
    utxoSetA := utxo.NewSet(utxoRepoA)
    pow := &block.ProofOfWork{}
    cfg := chain.ChainConfig{
        InitialDifficulty: 1,
        GenesisMessage:    "integration-test",
        BlockReward:       5000000000,
    }
    chainA := chain.NewChain(repoA, pow, cfg, utxoSetA)
    require.NoError(t, chainA.Initialize(ctx, minerAddr))
    poolA := mempool.New(utxoSetA)
    srvA = p2p.NewServer(chainA, poolA, utxoSetA, repoA, 0)
    require.NoError(t, srvA.Start(ctx))
    t.Cleanup(srvA.Stop)

    // Node B: same setup, same genesis
    // ... (mirror of A)

    return srvA, srvB
}
```

### Pattern 2: E2E Chain Scenario Test
**What:** Single test function exercises: create wallet -> initialize chain -> mine block -> create signed tx -> add to mempool -> mine block with tx -> verify UTXO updated -> check balance.
**When to use:** INTG-02 tests.
**Example:**
```go
func TestE2E_WalletToBalance(t *testing.T) {
    // 1. Create wallets
    sender := testutil.MustCreateWallet(t)
    receiver := testutil.MustCreateWallet(t)

    // 2. Setup chain with UTXO set
    repo := testutil.NewMockChainRepo()
    utxoRepo := testutil.NewMockUTXORepo()
    utxoSet := utxo.NewSet(utxoRepo)
    ch := chain.NewChain(repo, &block.ProofOfWork{}, cfg, utxoSet)

    // 3. Initialize chain (mines genesis with coinbase to sender)
    require.NoError(t, ch.Initialize(ctx, sender.Address()))

    // 4. Verify sender has UTXOs from genesis coinbase
    utxos, err := utxoSet.GetByAddress(sender.Address())
    require.NoError(t, err)
    require.Len(t, utxos, 1)

    // 5. Create and sign a transaction
    spendTx := testutil.MustBuildSignedTx(t, utxoSet, sender.PrivateKey(), sender.Address())

    // 6. Mine block containing the tx
    _, err = ch.MineBlock(ctx, sender.Address(), []*tx.Transaction{spendTx}, 0)
    require.NoError(t, err)

    // 7. Verify UTXO state updated
    // ...
}
```

### Pattern 3: Build Tag for Integration Tests
**What:** Use `//go:build integration` or `_test.go` naming with `-run Integration` pattern.
**When to use:** Tests that are slower (real TCP, mining).
**Recommendation:** Use `-run Integration` naming convention (prefix test names with `TestIntegration_`) rather than build tags. This matches the success criteria: "passing with `go test -v -run Integration`". No build tags needed since tests use in-memory repos and are fast.

### Anti-Patterns to Avoid
- **time.Sleep for sync waits:** Use `require.Eventually` with 10ms polling intervals (project convention from Phase 15).
- **Hardcoded ports:** Always use port 0 for OS-assigned ports to avoid CI conflicts.
- **Different genesis blocks:** Multi-node tests MUST use the same miner address to ensure matching genesis hashes (validated in handshake).

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Test chain setup | Manual block creation loops | `testutil.MustCreateBlockChain(t, n)` | Correct hash linkage, coinbase txs |
| Wallet key generation | Manual ECDSA key gen | `testutil.MustCreateWallet(t)` | Handles secp256k1 properly |
| Signed test transactions | Manual input/output/sign | `testutil.MustBuildSignedTx(t, ...)` | Correct UTXO referencing and signing |
| P2P server for tests | Custom TCP listeners | `p2p.NewServer(ch, pool, us, repo, 0)` | Port 0, context cancel, peer lifecycle |
| Async polling | `time.Sleep` | `require.Eventually(t, fn, timeout, interval)` | Deterministic, no flaky sleeps |

## Common Pitfalls

### Pitfall 1: Hub Race Condition (MUST FIX)
**What goes wrong:** `TestHub_DropsMessageWhenClientFull` fails under `-race` because Hub.Run() evicts a slow client by calling `delete(h.clients, client)` and `close(client.send)` while holding only an RLock (line 63-64 of hub.go). The RLock is taken for the broadcast case, but delete is a write operation.
**Why it happens:** The broadcast select case holds RLock but the eviction fallback (default branch) performs map mutation.
**How to fix:** Upgrade RLock to Lock for the broadcast case, OR schedule eviction via the unregister channel instead of inline deletion.
**Warning signs:** `DATA RACE` on `runtime.mapassign_fast64ptr` in `hub.go:64`.

### Pitfall 2: Genesis Hash Mismatch in Multi-Node Tests
**What goes wrong:** P2P handshake fails with `ErrIncompatibleGenesis` if two nodes have different genesis blocks.
**Why it happens:** Different miner addresses produce different coinbase transactions, yielding different merkle roots and block hashes.
**How to avoid:** Always use the **same** miner address when creating multiple nodes that need to connect.
**Warning signs:** `Connect()` returns error containing "incompatible genesis".

### Pitfall 3: Port Conflicts in CI
**What goes wrong:** Tests fail with "address already in use" when multiple test packages run in parallel.
**Why it happens:** Hardcoded ports collide across packages.
**How to avoid:** Always use `port 0` (OS-assigned). The existing `ListenAddr()` method returns the actual address for connecting.

### Pitfall 4: Goroutine Leaks from Hub.Run()
**What goes wrong:** Hub.Run() has an infinite for loop with no stop mechanism. Tests that create hubs leak goroutines.
**Why it happens:** Hub lacks a Stop() method (noted in STATE.md blockers).
**How to avoid:** For integration tests, the hub is not needed -- only test P2P and chain layers directly. If hub is used, accept the leak since tests are short-lived.

### Pitfall 5: Race in Peer.SetHeight/Height
**What goes wrong:** `Peer.height` is a plain `uint64` accessed without synchronization. `SetHeight` is called during handshake (one goroutine) and read during sync decisions (another goroutine).
**How to avoid:** Check if this actually triggers under `-race`. If it does, use `atomic.Uint64` or add a mutex. Currently only the ws package race is confirmed.

## Code Examples

### Multi-Node Handshake + Block Sync Integration Test
```go
func TestIntegration_TwoNodeBlockSync(t *testing.T) {
    ctx := context.Background()
    minerAddr := "sync-miner"

    // Setup node A with 3 blocks
    repoA := testutil.NewMockChainRepo()
    utxoRepoA := testutil.NewMockUTXORepo()
    utxoSetA := utxo.NewSet(utxoRepoA)
    cfg := chain.ChainConfig{InitialDifficulty: 1, GenesisMessage: "sync-test", BlockReward: 5000000000}
    chainA := chain.NewChain(repoA, &block.ProofOfWork{}, cfg, utxoSetA)
    require.NoError(t, chainA.Initialize(ctx, minerAddr))
    for range 2 {
        _, err := chainA.MineBlock(ctx, minerAddr, nil, 0)
        require.NoError(t, err)
    }
    poolA := mempool.New(utxoSetA)
    srvA := p2p.NewServer(chainA, poolA, utxoSetA, repoA, 0)
    require.NoError(t, srvA.Start(ctx))
    t.Cleanup(srvA.Stop)

    // Setup node B with only genesis
    repoB := testutil.NewMockChainRepo()
    utxoRepoB := testutil.NewMockUTXORepo()
    utxoSetB := utxo.NewSet(utxoRepoB)
    chainB := chain.NewChain(repoB, &block.ProofOfWork{}, cfg, utxoSetB)
    require.NoError(t, chainB.Initialize(ctx, minerAddr))
    poolB := mempool.New(utxoSetB)
    srvB := p2p.NewServer(chainB, poolB, utxoSetB, repoB, 0)
    require.NoError(t, srvB.Start(ctx))
    t.Cleanup(srvB.Stop)

    // Connect B to A -- triggers IBD since A has height 2 > B's height 0
    require.NoError(t, srvB.Connect(srvA.ListenAddr()))

    // Wait for sync
    require.Eventually(t, func() bool {
        return chainB.Height() == chainA.Height()
    }, 5*time.Second, 50*time.Millisecond, "node B should sync to node A's height")
}
```

### CI Workflow Change (TINF-03)
```yaml
# .github/workflows/ci-go.yml -- test job
- name: Run tests with coverage and race detection
  run: go test -race -coverprofile=coverage.out ./...
```

### Hub Race Fix (approach: schedule eviction via unregister)
```go
// In Hub.Run(), broadcast case:
case message := <-h.broadcast:
    h.mu.RLock()
    var evict []*Client
    for client := range h.clients {
        select {
        case client.send <- message:
        default:
            evict = append(evict, client)
        }
    }
    h.mu.RUnlock()
    // Evict slow clients outside the RLock
    for _, client := range evict {
        h.mu.Lock()
        if _, ok := h.clients[client]; ok {
            delete(h.clients, client)
            close(client.send)
        }
        h.mu.Unlock()
    }
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| `time.Sleep` for test sync | `require.Eventually` | Phase 15 (project decision) | No flaky tests |
| No race detection | `-race` flag in CI | Phase 18 (this phase) | Catches data races on every push |
| Tests without coverage | `-coverprofile` in CI | Already implemented (CI-05) | Coverage visible in CI output |

## Open Questions

1. **Peer.height race safety**
   - What we know: `height` is a plain uint64, set during handshake and read during sync decisions
   - What's unclear: Whether the race detector actually flags this under real test conditions (only ws race confirmed)
   - Recommendation: Run integration tests with `-race` first. If Peer.height races, convert to `atomic.Uint64`

2. **Integration test package location**
   - What we know: Could go in `internal/integration/` (new package) or as files in existing packages
   - Recommendation: New `internal/integration/` package -- keeps integration tests separate from unit tests, allows `go test -run Integration` pattern, and avoids bloating existing test files

## Validation Architecture

### Test Framework
| Property | Value |
|----------|-------|
| Framework | Go testing + testify v1.11.1 |
| Config file | None needed -- Go test conventions |
| Quick run command | `go test -v -run Integration ./internal/integration/` |
| Full suite command | `go test -race ./...` |

### Phase Requirements -> Test Map
| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| INTG-01 | TCP handshake, block sync, tx relay between 2+ nodes | integration | `go test -v -run TestIntegration ./internal/integration/ -timeout 60s` | No -- Wave 0 |
| INTG-02 | E2E wallet->tx->mine->utxo->balance | integration | `go test -v -run TestE2E ./internal/integration/ -timeout 60s` | No -- Wave 0 |
| TINF-03 | Race detection passes in CI | CI config | `go test -race ./...` | No -- CI change needed |

### Sampling Rate
- **Per task commit:** `go test -v -run "Integration|E2E" ./internal/integration/ -timeout 60s`
- **Per wave merge:** `go test -race ./...`
- **Phase gate:** `go test -race ./...` passes with zero race warnings

### Wave 0 Gaps
- [ ] `internal/integration/integration_test.go` -- P2P multi-node tests (INTG-01)
- [ ] `internal/integration/e2e_chain_test.go` -- E2E chain scenario tests (INTG-02)
- [ ] Fix ws.Hub race in `internal/handler/ws/hub.go` -- prerequisite for TINF-03
- [ ] Update `.github/workflows/ci-go.yml` -- add `-race` flag (TINF-03)

## Sources

### Primary (HIGH confidence)
- Project codebase: `internal/domain/p2p/server_test.go` -- existing makeTestServer pattern
- Project codebase: `internal/handler/ws/hub.go` + `hub_test.go` -- confirmed race location
- Project codebase: `.github/workflows/ci-go.yml` -- current CI without `-race`
- Project codebase: `internal/testutil/builders.go` -- available test helpers
- `go test -race ./...` output -- confirmed single race in ws package

### Secondary (MEDIUM confidence)
- Go documentation: race detector behavior with `sync.RWMutex` -- write-under-RLock is a race

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH -- no new dependencies, all existing tools
- Architecture: HIGH -- extends existing patterns (makeTestServer, testutil builders)
- Pitfalls: HIGH -- race confirmed by running `-race`, genesis mismatch confirmed by existing test
- CI change: HIGH -- single line change to existing workflow

**Research date:** 2026-03-08
**Valid until:** 2026-04-08 (stable -- no external dependency changes)
