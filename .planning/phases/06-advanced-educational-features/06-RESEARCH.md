# Phase 6: Advanced Educational Features - Research

**Researched:** 2026-03-07
**Domain:** Blockchain economics (halving, fees), CLI orchestration, security demos
**Confidence:** HIGH

## Summary

Phase 6 implements four distinct feature areas: block reward halving (economic policy), transaction fees with miner prioritization (market mechanics), multi-node testnet orchestration (turnkey demo), and double-spend attack demonstration (security education). All build on the existing, well-tested codebase.

The codebase is well-structured for these additions. Block reward halving requires a small formula change in `chain.MineBlock` plus a new `HalvingInterval` config field. Transaction fees require modifying `CreateTransactionWithChange` to accept a fee, adding fee computation to mempool, and changing `DrainAll()` to sort by fee rate. The orchestration and demo commands are CLI-level features that compose existing building blocks (P2P server, wallet, send, mine).

**Primary recommendation:** Implement in order: halving (isolated, pure domain) -> fees (domain + mempool) -> orchestration command -> double-spend demo (depends on orchestration).

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|-----------------|
| MINE-08 | Block reward halves every N blocks (configurable interval) | Add `HalvingInterval` to `ConsensusConfig`, compute reward in `Chain.MineBlock` using `reward >> (height / interval)` |
| TX-09 | Transaction fees computed as input-output difference, collected by miner | Modify `CreateTransactionWithChange` to accept fee, update coinbase to include fees, fee = inputSum - outputSum already validated |
| TX-10 | Miner prioritizes transactions by fee rate | Replace `DrainAll()` with fee-aware selection, sort by fee-per-byte or fee-per-tx, cap block tx count |
| ORCH-01 | Single CLI command launches multi-node local testnet | New `testnet` subcommand spawning N `startnode` processes with pre-configured ports, peers, wallets |
| DEMO-01 | Double-spend attempt detected and rejected by network | New `demo doublespend` subcommand that creates conflicting transactions and shows rejection |
</phase_requirements>

## Standard Stack

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| Go stdlib `os/exec` | 1.26 | Process spawning for testnet orchestration | Standard library, no dependencies needed |
| Go stdlib `sort` | 1.26 | Fee-rate sorting for tx prioritization | Standard library slices.SortFunc |
| Go stdlib `slices` | 1.26 | Modern sorting with `slices.SortFunc` | Preferred over `sort` package in Go 1.21+ |

### Supporting
| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| testify | existing | Test assertions | All new tests |

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| `os/exec` for testnet | In-process goroutines | `os/exec` gives real process isolation matching production; goroutines share memory and bbolt locks |
| Priority queue (heap) | Sorted slice | Sorted slice is simpler and sufficient for small mempools in an educational project |

## Architecture Patterns

### Pattern 1: Halving Reward Calculation

**What:** Compute block reward based on height and halving interval using bit-shift.
**When to use:** Every time a coinbase transaction is created in `MineBlock`.
**Example:**
```go
// In chain.go or a helper function
func (c *Chain) rewardAtHeight(height uint64) int64 {
    if c.config.HalvingInterval <= 0 {
        return c.config.BlockReward // no halving
    }
    halvings := height / uint64(c.config.HalvingInterval)
    if halvings >= 64 {
        return 0 // reward exhausted
    }
    return c.config.BlockReward >> halvings
}
```

Key points:
- `HalvingInterval` added to `ConsensusConfig` and `ChainConfig` (default: 210000 like Bitcoin, but demo config uses small value like 10)
- Right-shift (`>>`) is the standard Bitcoin approach -- halves the reward each interval
- Guard against `halvings >= 64` to avoid shifting to zero incorrectly on 64-bit int
- `ValidateCoinbase` must also accept height-based reward (currently checks `expectedReward` as static)

### Pattern 2: Transaction Fee Computation

**What:** Fee is the difference between sum of input values and sum of output values.
**When to use:** When adding to mempool (to store fee), when selecting transactions for mining.
**Example:**
```go
// Fee calculation -- needs UTXO lookup for input values
func ComputeFee(transaction *tx.Transaction, utxoSet *utxo.Set) (int64, error) {
    var inputSum int64
    for _, input := range transaction.Inputs() {
        utxo, err := utxoSet.Get(input.TxID(), input.Vout())
        if err != nil {
            return 0, err
        }
        inputSum += utxo.Value()
    }
    var outputSum int64
    for _, output := range transaction.Outputs() {
        outputSum += output.Value()
    }
    return inputSum - outputSum, nil
}
```

Key points:
- Fee computation happens at mempool `Add()` time (UTXO values available)
- Store fee alongside transaction in mempool (new struct or map)
- Coinbase output value = block reward + sum of all tx fees
- `send` command needs a `-fee` flag (amount in satoshis)

### Pattern 3: Fee-Prioritized Block Construction

**What:** When mining, select transactions from mempool sorted by fee rate (highest first).
**When to use:** Replace `DrainAll()` with `DrainByFeeRate(maxTxs int)`.
**Example:**
```go
// In mempool package
type mempoolEntry struct {
    tx  *tx.Transaction
    fee int64
}

func (m *Mempool) DrainByFee(maxTxs int) []*tx.Transaction {
    m.mu.Lock()
    defer m.mu.Unlock()

    entries := make([]mempoolEntry, 0, len(m.txs))
    for _, e := range m.entries {
        entries = append(entries, e)
    }
    slices.SortFunc(entries, func(a, b mempoolEntry) int {
        return cmp.Compare(b.fee, a.fee) // descending
    })

    if maxTxs > 0 && len(entries) > maxTxs {
        entries = entries[:maxTxs]
    }

    result := make([]*tx.Transaction, len(entries))
    for i, e := range entries {
        result[i] = e.tx
        delete(m.txs, e.tx.ID())
        // clean up spentOutputs...
    }
    m.spentOutputs = make(map[string]block.Hash) // rebuild if partial drain
    return result
}
```

### Pattern 4: Multi-Node Testnet Orchestration

**What:** A `testnet` CLI command that spawns N node processes with coordinated config.
**When to use:** ORCH-01 requirement.
**Design:**
```
go run cmd/shitcoin/main.go -f etc/shitcoin.yaml testnet [-nodes N] [-base-port PORT]
```

Implementation approach:
1. Create N wallet files (one per node)
2. Spawn N `startnode` processes via `os/exec.Command` with different ports
3. Node 0 is the seed node; nodes 1..N-1 connect to node 0 via `-peers`
4. Node 0 gets `-mine` flag with its wallet address
5. Print status table showing all nodes
6. Wait for Ctrl+C, then send SIGTERM to all child processes
7. Clean up data directories

Key points:
- Use `exec.CommandContext` for clean shutdown
- Pipe child stdout/stderr with node-ID prefix for distinguishable output
- Default 3 nodes, base port 3000
- Each node uses `data/node-{port}/` isolation (already supported)

### Pattern 5: Double-Spend Demo

**What:** A scripted demo that creates a double-spend and shows network rejection.
**When to use:** DEMO-01 requirement.
**Design:**
```
go run cmd/shitcoin/main.go -f etc/shitcoin.yaml demo doublespend
```

Implementation approach:
1. Start 2 nodes (reuse testnet logic or in-process)
2. Create a wallet, mine blocks to get coins
3. Create TX-A: send coins to address X (broadcast to node 1)
4. Create TX-B: spend SAME UTXOs to address Y (attempt to send to node 2)
5. Show that TX-B is rejected by mempool (ErrDoubleSpend or ErrUTXONotFound)
6. Mine a block, show TX-A is confirmed and TX-B inputs are gone
7. Print educational explanation

Key points:
- This can be done mostly in-process (two mempool instances sharing a chain view)
- Or use the testnet spawner for a real multi-node demo
- The mempool already detects double-spends via `spentOutputs` map
- The UTXO set validation also prevents double-spends at block level

### Anti-Patterns to Avoid
- **Coupling fee to transaction struct:** Fee is NOT part of the transaction data (it's derived). Store it in mempool metadata, not in `Transaction`.
- **Hardcoding halving interval:** Must be configurable via `ConsensusConfig`. Use a small default for demos (e.g., 10 blocks).
- **Blocking testnet command:** Spawned processes must be managed with context cancellation. Do not block main goroutine waiting for each process individually.
- **Complex serialization for fees:** Fee rate in this educational project can simply be total fee (not fee-per-byte). Bitcoin uses fee-per-weight-unit, but that's unnecessary complexity here.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Process management | Custom daemon manager | `os/exec.CommandContext` + process group | stdlib handles signals, pipes, cleanup |
| Fee-per-byte calculation | Custom serialization size estimator | Simple total-fee sorting | Educational project; fee-per-byte adds complexity without educational value |
| Transaction pool data structure | Custom priority queue | `slices.SortFunc` on drain | Mempool is small; O(n log n) sort on drain is fine |

## Common Pitfalls

### Pitfall 1: Coinbase Reward Validation After Halving
**What goes wrong:** P2P block validation uses a static `BlockReward` to validate incoming blocks, rejecting valid blocks with halved rewards.
**Why it happens:** `ValidateCoinbase` currently takes a static `expectedReward` parameter.
**How to avoid:** Pass height-computed reward to `ValidateCoinbase`. Both local mining and P2P validation must use the same `rewardAtHeight()` function.
**Warning signs:** Peer blocks rejected after halving boundary.

### Pitfall 2: Fee Not Included in Coinbase
**What goes wrong:** Miner creates coinbase with only block reward, not reward + fees. Fees are "burned."
**Why it happens:** `MineBlock` currently does `tx.NewCoinbaseTxWithHeight(minerAddress, c.config.BlockReward, newHeight)` -- static reward.
**How to avoid:** Sum fees from selected transactions, pass `reward + totalFees` to coinbase constructor.
**Warning signs:** Miner balance doesn't reflect collected fees.

### Pitfall 3: DrainAll vs DrainByFee Backward Compatibility
**What goes wrong:** Existing code calls `DrainAll()`. If removed, breaks auto-mine loops.
**Why it happens:** Hard-switching API without updating callers.
**How to avoid:** Keep `DrainAll()` for backward compat (internally calls `DrainByFee(0)` with no limit). Mining code can call the new method.
**Warning signs:** Compile errors in `signal.go` auto-mine code.

### Pitfall 4: Testnet Child Process Cleanup
**What goes wrong:** Orphaned processes after parent exits abnormally.
**Why it happens:** SIGKILL or crash doesn't trigger deferred cleanup.
**How to avoid:** Use process groups (`syscall.SysProcAttr{Setpgid: true}`) so children die with parent. Also register signal handler that explicitly kills children.
**Warning signs:** Port-already-in-use errors on second run.

### Pitfall 5: UTXO Lookup Race in Fee Computation
**What goes wrong:** Fee is computed using UTXO set, but UTXO might be spent by a mined block between computation and use.
**Why it happens:** Mempool stores fee at Add() time, but UTXO state changes.
**How to avoid:** Compute and store fee at `Add()` time. The stored fee remains valid because the mempool entry is removed when its inputs are spent.
**Warning signs:** Stale fee values (not actually a problem with current architecture since mempool.Remove cleans up).

### Pitfall 6: Send Command Fee Changes Transaction ID
**What goes wrong:** Adding a fee changes outputs (less change), which changes the transaction hash/ID.
**Why it happens:** Fee reduces the change output value.
**How to avoid:** This is correct behavior. Fee = inputSum - outputSum. The `send -fee` flag just reduces the change output by that amount.
**Warning signs:** None -- this is expected. Just ensure `inputSum >= amount + fee`.

## Code Examples

### Current Coinbase Creation (needs modification for halving + fees)
```go
// Current: static reward
coinbase := tx.NewCoinbaseTxWithHeight(minerAddress, c.config.BlockReward, newHeight)

// After: dynamic reward + fees
reward := c.rewardAtHeight(newHeight)
totalFees := sumFees(txs) // sum of (inputSum - outputSum) for each tx
coinbase := tx.NewCoinbaseTxWithHeight(minerAddress, reward+totalFees, newHeight)
```

### Current Send (needs fee parameter)
```go
// Current: no fee support
transaction, err := tx.CreateTransactionWithChange(inputs, inputValues, *to, *amount, *from)

// After: fee support
transaction, err := tx.CreateTransactionWithChange(inputs, inputValues, *to, *amount, *from, *fee)
// Where change = inputSum - amount - fee
```

### Config Addition
```go
// Add to ConsensusConfig
HalvingInterval int `json:",default=210000"`

// Demo config (etc/shitcoin.yaml)
Consensus:
  HalvingInterval: 10  # halve every 10 blocks for demo
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| Static block reward | Height-based halving | Bitcoin genesis (2009) | Deflationary supply model |
| FIFO transaction selection | Fee-rate prioritization | Bitcoin 0.3+ | Market-based block space allocation |

## Open Questions

1. **Fee unit: total fee vs fee-per-byte?**
   - What we know: Bitcoin uses fee-per-virtual-byte (segwit weight). This project has no segwit.
   - What's unclear: Whether to sort by total fee or fee-per-byte.
   - Recommendation: Use total fee (simpler, educational). All transactions in this project are roughly the same size anyway. Document that Bitcoin uses fee-per-byte.

2. **Max transactions per block?**
   - What we know: Currently no limit. `DrainAll()` takes everything.
   - What's unclear: Whether to add a block size or tx count limit.
   - Recommendation: Add a configurable `MaxBlockTxs` (default 100). Makes fee prioritization meaningful -- only top-N by fee get included.

3. **Testnet: in-process vs subprocess?**
   - What we know: Both approaches work. Subprocess gives real isolation.
   - What's unclear: Whether bbolt allows multiple in-process opens of different files.
   - Recommendation: Use `os/exec` subprocess approach. It matches how users would actually run nodes and avoids any shared-memory complications.

## Validation Architecture

### Test Framework
| Property | Value |
|----------|-------|
| Framework | Go testing + testify (assert/require/suite) |
| Config file | None needed -- testify loaded via go.mod |
| Quick run command | `go test ./internal/domain/chain/ ./internal/domain/mempool/ ./internal/domain/tx/ -v -count=1` |
| Full suite command | `go test ./... -count=1` |

### Phase Requirements -> Test Map
| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| MINE-08 | Block reward halves at correct interval | unit | `go test ./internal/domain/chain/ -run TestRewardHalving -v` | No - Wave 0 |
| MINE-08 | Reward reaches zero after sufficient halvings | unit | `go test ./internal/domain/chain/ -run TestRewardExhaustion -v` | No - Wave 0 |
| TX-09 | Fee computed as input-output difference | unit | `go test ./internal/domain/tx/ -run TestComputeFee -v` | No - Wave 0 |
| TX-09 | Coinbase includes block reward + collected fees | unit | `go test ./internal/domain/chain/ -run TestCoinbaseIncludesFees -v` | No - Wave 0 |
| TX-10 | Transactions sorted by fee descending | unit | `go test ./internal/domain/mempool/ -run TestDrainByFee -v` | No - Wave 0 |
| TX-10 | MaxBlockTxs limits included transactions | unit | `go test ./internal/domain/mempool/ -run TestDrainByFeeMaxTxs -v` | No - Wave 0 |
| ORCH-01 | Testnet command spawns multiple nodes | integration | Manual - spawns processes | No - Wave 0 |
| DEMO-01 | Double-spend rejected by mempool | unit | `go test ./internal/domain/mempool/ -run TestDoubleSpend -v` | Yes (exists) |
| DEMO-01 | Demo command runs end-to-end | integration | Manual - multi-process | No - Wave 0 |

### Sampling Rate
- **Per task commit:** `go test ./internal/domain/chain/ ./internal/domain/mempool/ ./internal/domain/tx/ -v -count=1`
- **Per wave merge:** `go test ./... -count=1`
- **Phase gate:** Full suite green before `/gsd:verify-work`

### Wave 0 Gaps
- [ ] `internal/domain/chain/chain_test.go` -- add TestRewardHalving, TestRewardExhaustion, TestCoinbaseIncludesFees
- [ ] `internal/domain/mempool/mempool_test.go` -- add TestDrainByFee, TestDrainByFeeMaxTxs
- [ ] `internal/domain/tx/transaction_test.go` -- add TestComputeFee (or in a new fee_test.go)

## Sources

### Primary (HIGH confidence)
- Codebase analysis: `internal/domain/chain/chain.go` -- current MineBlock, ChainConfig, reward handling
- Codebase analysis: `internal/domain/tx/validator.go` -- current CreateTransactionWithChange, ValidateTransaction
- Codebase analysis: `internal/domain/mempool/mempool.go` -- current DrainAll, Add with double-spend detection
- Codebase analysis: `internal/config/config.go` -- current ConsensusConfig fields
- Codebase analysis: `internal/handler/cli/cli.go` -- current send command, startnode command
- Codebase analysis: `internal/handler/cli/signal.go` -- current autoMineWithP2P loop

### Secondary (MEDIUM confidence)
- Bitcoin halving mechanism: well-established, right-shift by halvings count
- Bitcoin fee model: inputSum - outputSum, fee-per-vbyte prioritization

### Tertiary (LOW confidence)
- None

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH - pure Go stdlib, no new dependencies
- Architecture: HIGH - straightforward extensions to existing patterns
- Pitfalls: HIGH - derived from direct codebase analysis of integration points
- Orchestration: MEDIUM - process management complexity varies by OS

**Research date:** 2026-03-07
**Valid until:** 2026-04-07 (stable domain, no external dependency changes)
