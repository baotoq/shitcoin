---
phase: 04-p2p-networking-and-consensus
plan: 04
subsystem: consensus
tags: [reorg, fork-detection, utxo-reversal, p2p, chain-reorganization]

# Dependency graph
requires:
  - phase: 04-p2p-networking-and-consensus/04-02
    provides: "Block relay, UTXO application, inv-getdata pattern"
  - phase: 04-p2p-networking-and-consensus/04-03
    provides: "IBD sync, CmdGetBlocks, sync state tracking"
provides:
  - "Chain.Reorganize method for fork resolution with UTXO reversal"
  - "P2P fork detection during sync and live relay"
  - "Repository GetUndoEntry and DeleteBlocksAbove methods"
  - "Unique coinbase transaction IDs via BIP34 height encoding"
affects: [phase-05-dashboard, phase-06-extras]

# Tech tracking
tech-stack:
  added: []
  patterns: [fork-detection-via-hash-comparison, undo-log-reversal, mempool-adder-interface]

key-files:
  created:
    - internal/domain/chain/chain_test.go
    - internal/domain/p2p/reorg_test.go
  modified:
    - internal/domain/chain/chain.go
    - internal/domain/chain/repository.go
    - internal/infrastructure/persistence/bbolt/chain_repo.go
    - internal/domain/p2p/sync.go
    - internal/domain/p2p/handler.go
    - internal/domain/tx/coinbase.go
    - internal/domain/tx/transaction.go

key-decisions:
  - "Fork detection via hash comparison: request peer's full chain from height 1, compare block-by-block to find divergence point"
  - "MempoolAdder interface decouples chain from mempool package"
  - "BIP34 coinbase uniqueness: encode block height in coinbaseData field included in tx hash computation"

patterns-established:
  - "Reorg pattern: undo in reverse order, delete orphans, apply forward, re-add orphan txs"
  - "Fork detection: sync prevHash mismatch triggers full-chain comparison mode"

requirements-completed: [NET-08, NET-09]

# Metrics
duration: 11min
completed: 2026-03-05
---

# Phase 4 Plan 4: Fork Detection and Chain Reorganization Summary

**Fork detection and chain reorganization with UTXO undo-log reversal, orphaned transaction recovery, and P2P convergence to longest valid chain**

## Performance

- **Duration:** 11 min
- **Started:** 2026-03-05T15:45:52Z
- **Completed:** 2026-03-05T15:57:16Z
- **Tasks:** 2
- **Files modified:** 11

## Accomplishments
- Chain.Reorganize correctly undoes orphaned blocks via UTXO undo-log, applies new fork blocks, and returns orphaned transactions to mempool
- P2P detects forks during IBD sync and live block relay, triggering reorganization when peer chain is longer
- Fixed coinbase transaction uniqueness bug (BIP34) -- all blocks now have unique coinbase IDs via height-encoded coinbaseData
- Repository extended with GetUndoEntry and DeleteBlocksAbove for atomic fork resolution

## Task Commits

Each task was committed atomically:

1. **Task 1: Chain.Reorganize method and extended repository** - `55d9f6b` (test), `2173404` (feat)
2. **Task 2: P2P fork detection and reorg triggering** - `e383cef` (test), `6232284` (feat)

_TDD tasks: RED (failing test) -> GREEN (implementation) pattern_

## Files Created/Modified
- `internal/domain/chain/chain.go` - Added Reorganize method, MempoolAdder interface, BIP34 coinbase
- `internal/domain/chain/repository.go` - Extended with GetUndoEntry, DeleteBlocksAbove
- `internal/domain/chain/chain_test.go` - Reorganization tests (fork switch, orphan txs, block preservation)
- `internal/infrastructure/persistence/bbolt/chain_repo.go` - Bbolt GetUndoEntry, DeleteBlocksAbove implementation
- `internal/domain/p2p/sync.go` - Fork detection state machine, reorg execution during sync
- `internal/domain/p2p/handler.go` - Fork detection on live block relay (longer fork triggers sync)
- `internal/domain/p2p/reorg_test.go` - P2P reorg tests (longer chain, UTXO, equal length)
- `internal/domain/tx/coinbase.go` - NewCoinbaseTxWithHeight for unique coinbase IDs
- `internal/domain/tx/transaction.go` - Added coinbaseData field to hash computation

## Decisions Made
- Fork detection via full-chain hash comparison (educational, clear, not optimized for scale)
- MempoolAdder interface `{ Add(*tx.Transaction) error }` avoids hard coupling chain to mempool
- BIP34 coinbase uniqueness: height encoded in `coinbaseData` field included in transaction hash

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Fixed coinbase transaction ID collision**
- **Found during:** Task 1 (Chain.Reorganize tests)
- **Issue:** All coinbase transactions at different heights produced identical IDs because ComputeID excluded the signature field where height was stored, and inputs/outputs were identical for same miner/reward
- **Fix:** Added `coinbaseData` field to Transaction struct and `hashableTransaction`, encoding block height as `"height:N"` string. Created `NewCoinbaseTxWithHeight` constructor. Updated `MineBlock` and `Initialize` to use height-aware coinbase.
- **Files modified:** internal/domain/tx/coinbase.go, internal/domain/tx/transaction.go, internal/domain/chain/chain.go
- **Verification:** UTXO accumulation now works correctly across blocks, all tests pass
- **Committed in:** 2173404 (Task 1 commit)

**2. [Rule 3 - Blocking] Updated mock chain repos for new Repository interface**
- **Found during:** Task 1 (compile errors after extending Repository interface)
- **Issue:** Adding GetUndoEntry and DeleteBlocksAbove to chain.Repository broke existing mock implementations in p2p tests
- **Fix:** Added stub implementations to mockChainRepo (server_test.go) and fullMockChainRepo (relay_test.go)
- **Files modified:** internal/domain/p2p/server_test.go, internal/domain/p2p/relay_test.go
- **Verification:** All existing P2P tests continue to pass
- **Committed in:** 2173404 (Task 1 commit)

---

**Total deviations:** 2 auto-fixed (1 bug, 1 blocking)
**Impact on plan:** Both fixes essential for correctness. The coinbase ID collision was a pre-existing bug that only manifested when UTXO accumulation was tested properly. No scope creep.

## Issues Encountered
None beyond the auto-fixed deviations.

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- Phase 4 (P2P Networking and Consensus) is now complete
- All 4 plans executed: TCP framing, block relay, IBD sync, fork resolution
- Ready for Phase 5 (Dashboard) or Phase 6 (Extras)
- Full blockchain stack operational: mining, transactions, P2P sync, fork convergence

---
*Phase: 04-p2p-networking-and-consensus*
*Completed: 2026-03-05*
