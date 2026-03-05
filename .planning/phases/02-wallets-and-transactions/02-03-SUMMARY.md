---
phase: 02-wallets-and-transactions
plan: 03
subsystem: domain
tags: [utxo, blockchain, bbolt, undo-log, coinbase, transactions]

requires:
  - phase: 02-wallets-and-transactions
    provides: "wallet domain (02-01), transaction domain (02-02)"
  - phase: 01-core-chain-foundation
    provides: "block entity, chain aggregate, bbolt persistence"
provides:
  - "UTXO domain package with set, undo-log, and repository interface"
  - "bbolt UTXO persistence with 36-byte composite keys and undo bucket"
  - "Typed transactions in Block entity (via []any to break import cycle)"
  - "Chain.MineBlock with coinbase creation and atomic UTXO updates"
  - "SaveBlockWithUTXOs for atomic multi-bucket bbolt writes"
  - "ConsensusConfig.BlockReward (50 coins default)"
affects: [mempool, p2p, reorg, api]

tech-stack:
  added: []
  patterns: ["UTXO set aggregate with repository pattern", "undo-log for block reversal", "atomic multi-bucket bbolt writes", "[]any to break import cycles between domain packages"]

key-files:
  created:
    - internal/domain/utxo/utxo.go
    - internal/domain/utxo/set.go
    - internal/domain/utxo/undo.go
    - internal/domain/utxo/repository.go
    - internal/domain/utxo/errors.go
    - internal/domain/utxo/set_test.go
    - internal/infrastructure/persistence/bbolt/utxo_repo.go
    - internal/infrastructure/persistence/bbolt/utxo_storage_model.go
    - internal/infrastructure/persistence/bbolt/utxo_repo_test.go
  modified:
    - internal/domain/block/block.go
    - internal/domain/block/block_test.go
    - internal/domain/block/pow_test.go
    - internal/domain/chain/chain.go
    - internal/domain/chain/repository.go
    - internal/config/config.go
    - internal/infrastructure/persistence/bbolt/storage_model.go
    - internal/infrastructure/persistence/bbolt/chain_repo.go
    - internal/infrastructure/persistence/bbolt/chain_repo_test.go
    - internal/svc/service_context.go
    - cmd/shitcoin/main.go

key-decisions:
  - "Used []any for Block.transactions to break block->tx->block import cycle"
  - "UTXO set uses in-memory spent tracking for intra-block double-spend detection"
  - "SaveBlockWithUTXOs writes block + UTXO + undo in single bbolt Update transaction"
  - "36-byte composite UTXO key: 32-byte txid + 4-byte big-endian vout"

patterns-established:
  - "[]any interface pattern: domain packages that would create import cycles use []any with type assertions at integration boundaries"
  - "Atomic multi-bucket writes: bbolt Update transaction spanning blocks, utxo, and undo buckets"
  - "Undo-log pattern: UndoEntry records spent and created UTXOs per block for future reorg support"

requirements-completed: [TX-07, TX-08]

duration: 11min
completed: 2026-03-05
---

# Phase 2 Plan 3: UTXO Set and Typed Transactions Summary

**UTXO domain package with set/undo-log, bbolt persistence with 36-byte composite keys, and typed transaction integration into Block/Chain with atomic multi-bucket writes**

## Performance

- **Duration:** 11 min
- **Started:** 2026-03-05T13:06:35Z
- **Completed:** 2026-03-05T13:17:23Z
- **Tasks:** 2
- **Files modified:** 20

## Accomplishments
- UTXO set correctly applies and undoes blocks with intra-block double-spend detection
- bbolt persistence with 36-byte composite keys and undo bucket round-trips correctly
- Block entity uses typed transactions (via []any to break import cycle), Chain.MineBlock creates coinbase and atomically updates UTXO set
- Full test suite passes with -race flag including all Phase 1 backward compatibility

## Task Commits

Each task was committed atomically:

1. **Task 1: UTXO domain package (TDD RED)** - `c2d8a74` (test)
2. **Task 1: UTXO domain package (TDD GREEN)** - `55f3c64` (feat)
3. **Task 2: Integrate typed transactions** - `e6d8622` (feat)

## Files Created/Modified
- `internal/domain/utxo/utxo.go` - UTXO value object with txID, vout, value, address
- `internal/domain/utxo/set.go` - UTXOSet aggregate with ApplyBlock, UndoBlock, GetBalance
- `internal/domain/utxo/undo.go` - UndoEntry recording spent and created UTXOs per block
- `internal/domain/utxo/repository.go` - Repository interface for UTXO persistence
- `internal/domain/utxo/errors.go` - ErrUTXONotFound, ErrDoubleSpend, ErrUndoEntryNotFound
- `internal/domain/utxo/set_test.go` - Domain-level tests with in-memory repository
- `internal/infrastructure/persistence/bbolt/utxo_repo.go` - bbolt UTXO repository with 36-byte composite keys
- `internal/infrastructure/persistence/bbolt/utxo_storage_model.go` - UTXOModel storage conversion
- `internal/infrastructure/persistence/bbolt/utxo_repo_test.go` - Persistence tests including cross-restart
- `internal/domain/block/block.go` - Changed transactions from [][]byte to []any
- `internal/domain/chain/chain.go` - Added utxoSet, coinbase creation, atomic UTXO updates
- `internal/domain/chain/repository.go` - Added SaveBlockWithUTXOs method
- `internal/config/config.go` - Added BlockReward (5B satoshis default) and SatoshiPerCoin constant
- `internal/infrastructure/persistence/bbolt/storage_model.go` - TxModel/TxInputModel/TxOutputModel for typed serialization
- `internal/infrastructure/persistence/bbolt/chain_repo.go` - SaveBlockWithUTXOs atomic multi-bucket write
- `internal/svc/service_context.go` - Wired UTXORepo and UTXOSet

## Decisions Made
- Used `[]any` for Block.transactions to break the block->tx->block import cycle (Go does not support circular imports)
- Identical coinbase transactions produce the same TX ID; tests use different reward amounts to create unique UTXOs
- SaveBlockWithUTXOs atomically writes to blocks, utxo, and undo buckets in a single bbolt Update transaction
- Chain.Initialize accepts minerAddress parameter for genesis coinbase; empty address skips coinbase creation

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] Resolved block->tx->block import cycle**
- **Found during:** Task 2 (Block typed transactions integration)
- **Issue:** Plan specified `[]*tx.Transaction` in block.go, but tx package already imports block for block.Hash, creating a circular import
- **Fix:** Used `[]any` in block.go instead of `[]*tx.Transaction`, with type assertions at integration boundaries (chain.go, storage_model.go, chain_repo.go)
- **Files modified:** internal/domain/block/block.go, internal/domain/chain/chain.go, internal/infrastructure/persistence/bbolt/storage_model.go, chain_repo.go
- **Verification:** `go build ./...` succeeds, all tests pass

**2. [Rule 1 - Bug] Fixed duplicate coinbase TX ID in GetBalance test**
- **Found during:** Task 1 (TDD GREEN phase)
- **Issue:** Two `NewCoinbaseTx("miner", 5_000_000_000)` calls produce identical TX IDs (same inputs/outputs), causing UTXO key collision
- **Fix:** Used different reward amounts (5B and 5B+1) to produce unique TX IDs
- **Files modified:** internal/domain/utxo/set_test.go
- **Verification:** TestGetBalance passes correctly

---

**Total deviations:** 2 auto-fixed (1 blocking, 1 bug)
**Impact on plan:** Both fixes necessary for correctness. The []any pattern is idiomatic Go for breaking import cycles. No scope creep.

## Issues Encountered
None beyond the documented deviations.

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- UTXO set complete with balance queries and undo capability
- Ready for Phase 3 (Mempool) which will use UTXO set for transaction validation
- Undo-log ready for Phase 4 (Reorg) chain reorganization
- All Phase 1 and Phase 2 tests pass with backward compatibility

---
*Phase: 02-wallets-and-transactions*
*Completed: 2026-03-05*
