---
gsd_state_version: 1.0
milestone: v1.0
milestone_name: milestone
status: in-progress
stopped_at: "Completed 04.1-02-PLAN.md"
last_updated: "2026-03-05T16:45:00Z"
last_activity: 2026-03-05 -- Plan 04.1-02 executed, complex test migration (suites, mocks)
progress:
  total_phases: 7
  completed_phases: 5
  total_plans: 13
  completed_plans: 13
  percent: 100
---

# Project State

## Project Reference

See: .planning/PROJECT.md (updated 2026-03-05)

**Core value:** A working blockchain you built and understand end-to-end -- from transaction creation to block mining to peer synchronization.
**Current focus:** Phase 04.1 complete -- all test files migrated to testify

## Current Position

Phase: 04.1 of 6 (Use Test Assert)
Plan: 2 of 2 in current phase (04.1-02 complete)
Status: Phase 04.1 Complete
Last activity: 2026-03-05 -- Plan 04.1-02 executed, complex test migration (suites, mocks)

Progress: [██████████] 100%

## Performance Metrics

**Velocity:**
- Total plans completed: 12
- Average duration: 9min
- Total execution time: 1.8 hours

**By Phase:**

| Phase | Plans | Total | Avg/Plan |
|-------|-------|-------|----------|
| 1. Core Chain Foundation | 2/2 | 32min | 16min |
| 2. Wallets and Transactions | 3/3 | 26min | 9min |
| 3. Mempool, Mining, CLI | 2/2 | 9min | 5min |
| 4. P2P Networking | 4/4 | 31min | 8min |

**Recent Trend:**
- Last 5 plans: 04-02 (9min), 04-03 (5min), 04-04 (11min), 04.1-01 (7min), 04.1-02 (7min)
- Trend: Consistent velocity, test migration fast mechanical work

*Updated after each plan completion*
| Phase 03 P01 | 6min | 2 tasks | 10 files |
| Phase 03 P02 | 3min | 2 tasks | 6 files |
| Phase 04 P01 | 6min | 2 tasks | 10 files |
| Phase 04 P02 | 9min | 2 tasks | 9 files |
| Phase 04 P03 | 5min | 2 tasks | 4 files |
| Phase 04 P04 | 11min | 2 tasks | 11 files |
| Phase 04.1 P01 | 7min | 2 tasks | 14 files |
| Phase 04.1 P02 | 7min | 2 tasks | 7 files |

## Accumulated Context

### Decisions

Decisions are logged in PROJECT.md Key Decisions table.
Recent decisions affecting current work:

- [Roadmap]: 6-phase build order following hard dependency chains (hashing -> transactions -> mempool -> P2P -> dashboard -> extras)
- [Roadmap]: UTXO undo-log designed in Phase 2, consumed by Phase 4 reorg -- cannot be deferred
- [01-01]: JSON serialization for hashing (debuggable, deterministic via struct field order)
- [01-01]: Timestamp as int64 Unix seconds (not time.Time) to avoid precision issues
- [01-01]: GenesisMessage default via ApplyDefaults() method to avoid go vet struct tag warning
- [01-01]: MineWithMaxNonce added for testable nonce exhaustion
- [01-02]: Height index key format: 'h:' prefix + 8-byte big-endian for ordered bbolt iteration
- [01-02]: Copy byte slices inside bolt tx callbacks (bbolt pitfall #4)
- [01-02]: Demo config InitialDifficulty=5 for practical CPU mining demo
- [01-02]: go-zero stat/logx disabled in main.go for clean demo output
- [02-01]: btcec/v2 for secp256k1 ECDSA key generation (per user constraint)
- [02-01]: Hand-rolled Base58Check encoding for educational value (per user constraint)
- [02-01]: Atomic JSON file writes via temp file + rename for crash safety
- [02-02]: Hashable struct pattern for TX ID: JSON-serialize inputs (without sig/pubkey) and outputs, then DoubleSHA256
- [02-02]: Coinbase marker: zero hash + 0xFFFFFFFF vout (Bitcoin convention)
- [02-02]: Simplified SIGHASH_ALL: sign full transaction hash rather than per-input signing
- [02-03]: []any for Block.transactions to break block->tx->block import cycle
- [02-03]: 36-byte composite UTXO key (32-byte txid + 4-byte big-endian vout)
- [02-03]: Atomic multi-bucket bbolt writes for block + UTXO + undo consistency
- [02-03]: SatoshiPerCoin constant and 50-coin default block reward (5B satoshis)
- [03-01]: Bitcoin Merkle convention: single leaf hashed with itself (not returned directly)
- [03-01]: Mempool tracks spentOutputs map separately for O(1) double-spend detection against pool
- [Phase 03]: flag.Args() passed to CLI.Run() so -f config flag and subcommands coexist cleanly
- [Phase 03]: Auto-mine loop uses context.WithCancel + signal.Notify for clean shutdown
- [Phase 03]: Simple greedy UTXO selection for send command
- [04-01]: Length-prefixed TCP framing: [4-byte BE length][1-byte command][JSON payload]
- [04-01]: 10-second handshake deadline via conn.SetDeadline, cleared after completion
- [04-01]: Non-blocking peer.Send with select/default drops messages when buffer full (cap 64)
- [04-01]: Genesis hash comparison during handshake rejects incompatible chains
- [04-01]: Per-node data directories (data/node-{port}/) prevent bbolt lock conflicts
- [04-02]: Inv-getdata relay pattern: announce hashes via CmdInv, request data via CmdGetData
- [04-02]: Seen-hash maps (seenBlocks/seenTxs) with mutex prevent infinite broadcast loops
- [04-02]: RWMutex on Chain aggregate for thread-safe P2P block acceptance
- [04-02]: autoMineWithP2P cancels mining context via OnBlockReceived callback
- [04-03]: CmdGetBlocks caps at 500 blocks per batch to prevent memory exhaustion
- [04-03]: IBD triggered automatically after handshake when peer height > local height
- [04-03]: Sync blocks routed separately from live relay via handleSyncBlock
- [04-03]: Invalid block during sync aborts IBD and disconnects the peer
- [04-04]: Fork detection via full-chain hash comparison (educational, clear approach)
- [04-04]: MempoolAdder interface decouples chain from mempool package
- [04-04]: BIP34 coinbase uniqueness: block height in coinbaseData field for unique tx IDs
- [04-04]: Reorg pattern: undo reverse order, delete orphans, apply forward, re-add orphan txs
- [04.1-01]: require for setup/preconditions, assert for value comparisons
- [04.1-01]: assert.New(t) pattern in table-driven subtests for cleaner syntax
- [04.1-01]: assert.ErrorIs for sentinel error checks
- [04.1-02]: testify/suite for bbolt repos (per-test DB via SetupTest), plain functions for P2P tests
- [04.1-02]: testify/mock only for simple stub mocks; hand-rolled fakes for stateful stores with maps/mutexes

### Roadmap Evolution

- Phase 04.1 inserted after Phase 4: use test assert (URGENT)

### Pending Todos

None yet.

### Blockers/Concerns

- [Research]: UTXO reversibility data structure needs deeper design at start of Phase 2
- [Research]: Phase 4 (P2P) flagged for potential research-phase before planning

## Session Continuity

Last session: 2026-03-05T16:45:00Z
Stopped at: Completed 04.1-02-PLAN.md
Resume file: .planning/phases/04.1-use-test-assert/04.1-02-SUMMARY.md
