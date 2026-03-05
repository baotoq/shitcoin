# Phase 3: Mempool, Mining Integration, and CLI - Research

**Researched:** 2026-03-05
**Domain:** Go concurrency (mempool), Merkle trees, CLI architecture, blockchain integration
**Confidence:** HIGH

## Summary

Phase 3 integrates the existing domain packages (block, chain, tx, wallet, utxo) into a usable single-node blockchain operated via CLI commands. The three main technical areas are: (1) a concurrent-safe mempool for holding validated-but-unmined transactions, (2) Merkle root computation for block headers, and (3) a CLI layer that wires everything together.

The codebase already has all the building blocks: `chain.MineBlock()` accepts `[]*tx.Transaction`, `utxo.Set.ApplyBlock()` handles UTXO mutations atomically, `wallet.Repository` persists key pairs, and `tx.CreateTransactionWithChange()` builds signed transactions. What's missing is the mempool domain package, the Merkle tree computation (currently hardcoded as `Hash{}` in block construction), the CLI dispatch layer, and the auto-mining background loop.

**Primary recommendation:** Build the mempool as a new domain package (`internal/domain/mempool`) with `sync.RWMutex` protection. Use Go's `flag` package (already in use) with subcommand dispatch for CLI. Implement Merkle root as a pure function in the `block` package. Wire everything through the existing `svc.ServiceContext` pattern.

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|-----------------|
| MINE-04 | User can mine a block manually via CLI command | CLI `mine` command calls `chain.MineBlock()` with transactions from mempool |
| MINE-05 | Node can auto-mine blocks continuously in background with context cancellation | Background goroutine with `context.WithCancel`, mining loop draining mempool |
| MINE-07 | Block headers include Merkle root from transaction hashes | `block.ComputeMerkleRoot()` function, integrated into `NewBlock()`/`NewGenesisBlock()` |
| NET-03 | Mempool holds validated-but-unmined transactions, RWMutex protected | New `internal/domain/mempool` package with `sync.RWMutex` |
| CLI-01 | `createwallet` command | CLI dispatches to `wallet.NewWallet()` + `walletRepo.Save()` |
| CLI-02 | `listaddresses` command | CLI dispatches to `walletRepo.ListAddresses()` |
| CLI-03 | `getbalance` command | CLI dispatches to `utxoSet.GetBalance()` |
| CLI-04 | `send` command | CLI builds transaction via `tx.CreateTransactionWithChange()`, signs, adds to mempool |
| CLI-05 | `mine` command | CLI calls `chain.MineBlock()` with mempool transactions |
| CLI-06 | `printchain` command | CLI iterates blocks via `chainRepo.GetBlocksInRange()` |
| CLI-07 | `startnode` command with port and mining address | Starts node with optional auto-mining background goroutine |
</phase_requirements>

## Standard Stack

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| `sync.RWMutex` | stdlib | Concurrent mempool access | Standard Go pattern for read-heavy concurrent maps |
| `flag` | stdlib | CLI argument parsing | Already used in project; no need for cobra/urfave for this scope |
| `context` | stdlib | Cancellation of auto-mining | Go-idiomatic; already used throughout chain operations |
| `os/signal` | stdlib | Graceful shutdown on SIGINT/SIGTERM | Standard Go pattern for daemon processes |

### Supporting
| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| `fmt` | stdlib | CLI output formatting | All CLI command output |
| `encoding/hex` | stdlib | Display transaction/block hashes | CLI printchain/send output |

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| `flag` subcommands | `cobra` or `urfave/cli` | cobra is overkill for 7 commands; flag + manual dispatch matches existing pattern |
| `sync.RWMutex` | `sync.Map` | RWMutex gives explicit control over lock scope; sync.Map optimized for different access pattern |
| Manual Merkle tree | External library | Merkle tree is ~30 lines; educational value in hand-rolling |

## Architecture Patterns

### Recommended Project Structure
```
internal/
├── domain/
│   ├── block/
│   │   ├── merkle.go          # NEW: ComputeMerkleRoot()
│   │   └── ...existing...
│   ├── mempool/
│   │   ├── mempool.go         # NEW: Mempool aggregate
│   │   ├── mempool_test.go    # NEW: Concurrency tests
│   │   └── errors.go          # NEW: ErrDuplicate, ErrDoubleSpend
│   └── ...existing...
├── handler/
│   └── cli/
│       └── cli.go             # NEW: CLI command dispatch
├── svc/
│   └── service_context.go     # MODIFIED: add WalletRepo, Mempool
└── ...existing...
cmd/
└── shitcoin/
    └── main.go                # MODIFIED: CLI dispatch replaces demo loop
```

### Pattern 1: Mempool as Domain Aggregate
**What:** Thread-safe in-memory transaction pool with validation
**When to use:** Any time transactions need to be held before mining
**Example:**
```go
// internal/domain/mempool/mempool.go
type Mempool struct {
    mu   sync.RWMutex
    txs  map[block.Hash]*tx.Transaction // keyed by tx ID
    utxo *utxo.Set                      // for validation
}

func (m *Mempool) Add(transaction *tx.Transaction) error {
    m.mu.Lock()
    defer m.mu.Unlock()

    // 1. Check duplicate
    if _, exists := m.txs[transaction.ID()]; exists {
        return ErrDuplicate
    }

    // 2. Check double-spend against pool
    for _, input := range transaction.Inputs() {
        key := fmt.Sprintf("%s:%d", input.TxID().String(), input.Vout())
        for _, poolTx := range m.txs {
            for _, poolInput := range poolTx.Inputs() {
                poolKey := fmt.Sprintf("%s:%d", poolInput.TxID().String(), poolInput.Vout())
                if key == poolKey {
                    return ErrDoubleSpend
                }
            }
        }
    }

    // 3. Verify signature
    if !tx.VerifyTransaction(transaction) {
        return ErrInvalidSignature
    }

    // 4. Verify UTXOs exist
    for _, input := range transaction.Inputs() {
        if _, err := m.utxo.Get(input.TxID(), input.Vout()); err != nil {
            return fmt.Errorf("input utxo not found: %w", err)
        }
    }

    m.txs[transaction.ID()] = transaction
    return nil
}

func (m *Mempool) DrainAll() []*tx.Transaction {
    m.mu.Lock()
    defer m.mu.Unlock()

    result := make([]*tx.Transaction, 0, len(m.txs))
    for _, t := range m.txs {
        result = append(result, t)
    }
    m.txs = make(map[block.Hash]*tx.Transaction)
    return result
}

func (m *Mempool) Remove(txIDs []block.Hash) {
    m.mu.Lock()
    defer m.mu.Unlock()
    for _, id := range txIDs {
        delete(m.txs, id)
    }
}
```

### Pattern 2: CLI Subcommand Dispatch
**What:** Manual subcommand routing using `os.Args` with per-command flag sets
**When to use:** Simple CLIs with < 20 commands
**Example:**
```go
// internal/handler/cli/cli.go
type CLI struct {
    svc *svc.ServiceContext
}

func (cli *CLI) Run() {
    if len(os.Args) < 2 {
        cli.printUsage()
        os.Exit(1)
    }

    switch os.Args[1] {
    case "createwallet":
        cli.createWallet()
    case "listaddresses":
        cli.listAddresses()
    case "getbalance":
        cli.getBalance()
    case "send":
        cli.send()
    case "mine":
        cli.mine()
    case "printchain":
        cli.printChain()
    case "startnode":
        cli.startNode()
    default:
        cli.printUsage()
        os.Exit(1)
    }
}
```

### Pattern 3: Auto-Mining Background Loop
**What:** Goroutine that mines blocks continuously until context is cancelled
**When to use:** `startnode` with `--mine` flag
**Example:**
```go
func (cli *CLI) autoMine(ctx context.Context, minerAddress string) {
    for {
        select {
        case <-ctx.Done():
            fmt.Println("Mining stopped.")
            return
        default:
            txs := cli.svc.Mempool.DrainAll()
            blk, err := cli.svc.Chain.MineBlock(ctx, minerAddress, txs)
            if err != nil {
                if ctx.Err() != nil {
                    return // context cancelled during mining
                }
                fmt.Printf("Mining error: %v\n", err)
                continue
            }
            fmt.Printf("Mined block #%d (%s)\n", blk.Height(), blk.Hash().String()[:16])
        }
    }
}
```

### Pattern 4: Merkle Root Computation
**What:** Binary hash tree from transaction IDs
**When to use:** Block construction (MINE-07)
**Example:**
```go
// internal/domain/block/merkle.go
func ComputeMerkleRoot(txHashes []Hash) Hash {
    if len(txHashes) == 0 {
        return Hash{}
    }

    // Copy to avoid mutating input
    hashes := make([]Hash, len(txHashes))
    copy(hashes, txHashes)

    for len(hashes) > 1 {
        // If odd number, duplicate last hash
        if len(hashes)%2 != 0 {
            hashes = append(hashes, hashes[len(hashes)-1])
        }

        var nextLevel []Hash
        for i := 0; i < len(hashes); i += 2 {
            combined := append(hashes[i].Bytes(), hashes[i+1].Bytes()...)
            nextLevel = append(nextLevel, DoubleSHA256(combined))
        }
        hashes = nextLevel
    }

    return hashes[0]
}
```

### Anti-Patterns to Avoid
- **Holding RWMutex during mining:** Mining is CPU-intensive and takes variable time. Never lock the mempool during `Mine()`. Drain transactions first, then release the lock before mining.
- **Checking mempool emptiness without lock:** Always acquire at least a read lock before checking `len(m.txs)`.
- **Mining empty blocks with no transactions:** The `mine` CLI should work even with no pending transactions (just coinbase), but auto-mine should wait or continue even if mempool is empty (mine coinbase-only blocks).
- **Forgetting to remove mined txs from mempool:** After a block is mined, its transactions must be removed from the mempool so they are not double-included.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| ECDSA signing | Custom crypto | `btcec/v2` + existing `tx.SignTransaction()` | Already implemented in Phase 2 |
| Wallet persistence | New storage | Existing `jsonfile.WalletRepo` | Already implemented in Phase 2 |
| UTXO lookups | New queries | Existing `utxo.Set.GetByAddress()` / `GetBalance()` | Already implemented in Phase 2 |
| Block construction | New block builder | Existing `chain.MineBlock()` | Already handles coinbase, PoW, UTXO updates |
| Config loading | Custom parser | Existing `go-zero conf.MustLoad()` | Already wired in ServiceContext |

**Key insight:** Phase 3 is an integration phase. Almost all domain logic exists. The new code is the mempool, Merkle root, CLI dispatch, and wiring.

## Common Pitfalls

### Pitfall 1: Deadlock in Mempool Validation
**What goes wrong:** Mempool `Add()` holds a write lock and calls `utxo.Set.Get()` which may also hold a lock
**Why it happens:** Nested locking across aggregates
**How to avoid:** UTXO set repository uses bbolt transactions (not Go mutexes), so no deadlock risk in current architecture. But if in-memory UTXO cache is added later, be careful about lock ordering.
**Warning signs:** Tests hanging on concurrent `Add()` calls

### Pitfall 2: Merkle Root Computed After Mining
**What goes wrong:** Merkle root must be in the header BEFORE mining, since the header hash includes the Merkle root
**Why it happens:** Current `NewBlock()` passes `Hash{}` for Merkle root
**How to avoid:** Compute Merkle root from transaction hashes and pass it to `NewBlock()`. This requires modifying `NewBlock()` and `NewGenesisBlock()` to accept a computed Merkle root instead of `Hash{}`.
**Warning signs:** Block hash doesn't change when transactions change

### Pitfall 3: Block.RawTransactions() Returns []any
**What goes wrong:** CLI needs to iterate transactions for display, but they are `[]any`
**Why it happens:** Import cycle avoidance design (Phase 2 decision)
**How to avoid:** Type-assert in the CLI layer: `txn := rawTx.(*tx.Transaction)`. This is safe because the chain aggregate only stores `*tx.Transaction` values.
**Warning signs:** Nil pointer on type assertion

### Pitfall 4: Send Command Needs Full Transaction Pipeline
**What goes wrong:** `send` command is more complex than it appears -- needs wallet lookup, UTXO selection, transaction building, signing, mempool insertion
**Why it happens:** Multiple domain objects must coordinate
**How to avoid:** Build a clear pipeline: (1) load sender wallet from repo, (2) get sender's UTXOs, (3) select enough UTXOs to cover amount, (4) build transaction with change, (5) sign with sender's private key, (6) add to mempool
**Warning signs:** Insufficient funds errors when balance should suffice (UTXO selection bug)

### Pitfall 5: ServiceContext Needs WalletRepo and Mempool
**What goes wrong:** Current `ServiceContext` does not include `WalletRepo` or `Mempool`
**Why it happens:** They weren't needed for the demo in Phase 2
**How to avoid:** Extend `ServiceContext` to include `WalletRepo wallet.Repository` and `Mempool *mempool.Mempool`. Wire them in `NewServiceContext()`.
**Warning signs:** Nil pointer on first CLI command

### Pitfall 6: Auto-Mine Must Handle Context Cancellation Mid-PoW
**What goes wrong:** `ProofOfWork.Mine()` does not currently accept context, so cancellation only happens between blocks
**Why it happens:** PoW loop is a tight CPU loop checking nonces
**How to avoid:** For Phase 3, accept that cancellation granularity is per-block (between `Mine()` calls). The PoW loop completes the current block before checking `ctx.Done()`. This is acceptable for single-node demo. Adding context to PoW can be deferred.
**Warning signs:** Long delay on shutdown when difficulty is high

### Pitfall 7: Wallet File Path Configuration
**What goes wrong:** Wallet JSON file path needs to be configurable like DB path
**Why it happens:** Current config only has `Storage.DBPath`
**How to avoid:** Add `WalletPath` to `StorageConfig` with a default like `data/wallets.json`
**Warning signs:** Wallets saved to wrong directory or hardcoded path

## Code Examples

### UTXO Selection for Send Command
```go
// Select UTXOs to cover amount, returns selected UTXOs and their total value
func selectUTXOs(utxos []utxo.UTXO, amount int64) ([]utxo.UTXO, int64, error) {
    var selected []utxo.UTXO
    var total int64

    for _, u := range utxos {
        selected = append(selected, u)
        total += u.Value()
        if total >= amount {
            return selected, total, nil
        }
    }

    return nil, 0, fmt.Errorf("insufficient funds: have %d, need %d", total, amount)
}
```

### Merkle Root Integration in Block Construction
```go
// In chain.MineBlock(), compute merkle root before creating block:
txHashes := make([]block.Hash, len(allTxs))
for i, t := range allTxs {
    txHashes[i] = t.ID()
}
merkleRoot := block.ComputeMerkleRoot(txHashes)

// Pass merkleRoot to NewBlock (requires signature change)
newBlock, err := block.NewBlock(c.latestBlock.Hash(), newHeight, bits, blockTxs, merkleRoot)
```

### Graceful Shutdown for startnode
```go
ctx, cancel := context.WithCancel(context.Background())
defer cancel()

// Handle OS signals
sigCh := make(chan os.Signal, 1)
signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

go func() {
    <-sigCh
    fmt.Println("\nShutting down...")
    cancel()
}()

// Start auto-mining if address provided
if mineAddr != "" {
    cli.autoMine(ctx, mineAddr)
}
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| Merkle root as `Hash{}` placeholder | Compute from tx hashes | Phase 3 | Block headers now correctly commit to transactions |
| Demo `main.go` with hardcoded mining | CLI-driven operations | Phase 3 | Users interact via commands, not code changes |
| No transaction pool | RWMutex-protected mempool | Phase 3 | Transactions can be queued before mining |

**Current state of codebase:**
- Block Merkle root: hardcoded `Hash{}` -- must be replaced
- CLI: none -- `main.go` is a demo script
- Mempool: does not exist
- Wallet in ServiceContext: not wired -- `WalletRepo` exists but not in `ServiceContext`

## Open Questions

1. **Should mine command mine only mempool transactions or allow mining empty blocks?**
   - What we know: Bitcoin mines empty blocks (coinbase only) when mempool is empty
   - Recommendation: Allow mining with empty mempool (coinbase-only block). This matches Bitcoin behavior and is simpler.

2. **Should PoW.Mine() accept context for mid-block cancellation?**
   - What we know: Current PoW loop has no context awareness. With InitialDifficulty=5, mining is fast.
   - Recommendation: Defer context-aware PoW to a later phase. For Phase 3 demo difficulty, per-block cancellation is sufficient.

3. **Should `startnode` start an HTTP server via go-zero RestConf?**
   - What we know: Config already embeds `rest.RestConf` with Host/Port. Phase 4 will add P2P networking.
   - Recommendation: For Phase 3, `startnode` just initializes the chain and runs auto-mining. The HTTP/P2P server is Phase 4. The command should accept `--port` and `--mine` flags to match CLI-07 spec, but port is stored for future use.

## Validation Architecture

### Test Framework
| Property | Value |
|----------|-------|
| Framework | Go testing (stdlib) |
| Config file | none (go test convention) |
| Quick run command | `go test ./internal/domain/mempool/... -race -count=1` |
| Full suite command | `go test ./... -race -count=1` |

### Phase Requirements -> Test Map
| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| MINE-04 | Mine block via CLI | integration | `go test ./internal/handler/cli/... -run TestMineCLI -race` | No - Wave 0 |
| MINE-05 | Auto-mine with cancellation | unit | `go test ./internal/handler/cli/... -run TestAutoMine -race` | No - Wave 0 |
| MINE-07 | Merkle root computation | unit | `go test ./internal/domain/block/... -run TestMerkle -race` | No - Wave 0 |
| NET-03 | Mempool with RWMutex | unit | `go test ./internal/domain/mempool/... -race -count=1` | No - Wave 0 |
| CLI-01 | createwallet command | integration | `go test ./internal/handler/cli/... -run TestCreateWallet -race` | No - Wave 0 |
| CLI-02 | listaddresses command | integration | `go test ./internal/handler/cli/... -run TestListAddresses -race` | No - Wave 0 |
| CLI-03 | getbalance command | integration | `go test ./internal/handler/cli/... -run TestGetBalance -race` | No - Wave 0 |
| CLI-04 | send command with mempool | integration | `go test ./internal/handler/cli/... -run TestSend -race` | No - Wave 0 |
| CLI-05 | mine command | integration | `go test ./internal/handler/cli/... -run TestMine -race` | No - Wave 0 |
| CLI-06 | printchain command | integration | `go test ./internal/handler/cli/... -run TestPrintChain -race` | No - Wave 0 |
| CLI-07 | startnode command | integration | `go test ./internal/handler/cli/... -run TestStartNode -race` | No - Wave 0 |

### Sampling Rate
- **Per task commit:** `go test ./internal/domain/mempool/... ./internal/domain/block/... -race -count=1`
- **Per wave merge:** `go test ./... -race -count=1`
- **Phase gate:** Full suite green before `/gsd:verify-work`

### Wave 0 Gaps
- [ ] `internal/domain/mempool/mempool_test.go` -- covers NET-03 (concurrent add, duplicate rejection, double-spend detection, drain)
- [ ] `internal/domain/block/merkle_test.go` -- covers MINE-07 (empty, single, even, odd tx counts)
- [ ] Framework install: none needed -- Go stdlib testing already in use

## Sources

### Primary (HIGH confidence)
- Project source code analysis -- all existing interfaces, types, and patterns reviewed directly
- Go stdlib documentation -- sync.RWMutex, context, flag, os/signal patterns
- Bitcoin wiki Merkle tree specification -- standard binary hash tree algorithm

### Secondary (MEDIUM confidence)
- Project skill files (`golang-pro/SKILL.md`, `golang-patterns/SKILL.md`) -- Go concurrency and interface patterns

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH -- all stdlib, no new dependencies needed
- Architecture: HIGH -- follows established project patterns (domain packages, ServiceContext, repository interfaces)
- Pitfalls: HIGH -- identified from direct code review of existing interfaces and their gaps
- Merkle tree: HIGH -- well-documented algorithm, straightforward implementation

**Research date:** 2026-03-05
**Valid until:** 2026-04-05 (stable domain, no external dependency changes expected)
