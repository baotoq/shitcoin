---
phase: 06-advanced-educational-features
plan: 01
subsystem: consensus
tags: [halving, transaction-fees, mempool-priority, coinbase, bitcoin-economics]

# Dependency graph
requires:
  - phase: 03-mempool-mining-cli
    provides: Mempool with DrainAll, MineBlock with coinbase
  - phase: 02-wallets-transactions
    provides: CreateTransactionWithChange, UTXO model
provides:
  - Block reward halving at configurable intervals
  - Transaction fee tracking in mempool with fee-priority drain
  - Fee-aware coinbase (reward + fees) in MineBlock
  - CLI -fee flag on send command
affects: [06-advanced-educational-features]

# Tech tracking
tech-stack:
  added: []
  patterns: [right-shift halving, fee-priority mempool sorting, mempoolEntry wrapper]

key-files:
  created: []
  modified:
    - internal/config/config.go
    - internal/domain/chain/chain.go
    - internal/domain/chain/chain_test.go
    - internal/domain/mempool/mempool.go
    - internal/domain/mempool/mempool_test.go
    - internal/domain/tx/validator.go
    - internal/domain/tx/transaction_test.go
    - internal/handler/cli/cli.go
    - internal/handler/cli/signal.go
    - internal/svc/service_context.go
    - etc/shitcoin.yaml

key-decisions:
  - "RewardAtHeight exported for testability and potential API use"
  - "AddWithFee alongside backward-compatible Add(tx) delegates to AddWithFee(tx, 0)"
  - "DrainByFee returns (txs, totalFees) tuple for direct use in MineBlock"
  - "MineBlock accepts totalFees parameter rather than computing fees internally"

patterns-established:
  - "Halving via right-shift: reward >> (height / halvingInterval), guard >= 64"
  - "mempoolEntry wrapper: struct { tx, fee } keeps fee tracking internal to mempool"
  - "Fee-priority sorting with slices.SortFunc descending by fee"

requirements-completed: [MINE-08, TX-09, TX-10]

# Metrics
duration: 7min
completed: 2026-03-07
---

# Phase 06 Plan 01: Block Reward Halving and Transaction Fees Summary

**Configurable block reward halving via right-shift, transaction fee tracking in mempool with fee-priority drain, and fee-inclusive coinbase construction**

## Performance

- **Duration:** 7 min
- **Started:** 2026-03-07T09:00:31Z
- **Completed:** 2026-03-07T09:07:39Z
- **Tasks:** 2
- **Files modified:** 11

## Accomplishments
- Block reward halves every HalvingInterval blocks (configurable, demo uses 10), reaching zero after 64 halvings
- Transaction fees tracked per mempool entry; DrainByFee sorts descending and respects MaxBlockTxs limit
- Coinbase reward in MineBlock = RewardAtHeight(height) + totalFees from drained transactions
- CLI send command accepts -fee flag; UTXO selection accounts for amount + fee
- All 8 new tests pass alongside full existing test suite

## Task Commits

Each task was committed atomically:

1. **Task 1: Halving, fee computation, and fee-aware mempool** - `b7a9202` (test)
2. **Task 2: Wire fee and halving into CLI and P2P callers** - `305783b` (feat)

## Files Created/Modified
- `internal/config/config.go` - Added HalvingInterval and MaxBlockTxs to ConsensusConfig
- `internal/domain/chain/chain.go` - Added RewardAtHeight method, MineBlock accepts totalFees
- `internal/domain/chain/chain_test.go` - TestRewardAtHeight, TestRewardAtHeightNoHalving, TestCoinbaseIncludesFees
- `internal/domain/mempool/mempool.go` - mempoolEntry wrapper, AddWithFee, DrainByFee, FeeForTx
- `internal/domain/mempool/mempool_test.go` - TestDrainByFee, TestDrainByFeeMaxTxs, TestDrainByFeeZeroLimit, TestAddStoresFee
- `internal/domain/tx/validator.go` - CreateTransactionWithChange accepts fee parameter
- `internal/domain/tx/transaction_test.go` - TestCreateTransactionWithChangeFee variants
- `internal/handler/cli/cli.go` - -fee flag, AddWithFee, DrainByFee in mine
- `internal/handler/cli/signal.go` - DrainByFee in autoMine and autoMineWithP2P
- `internal/svc/service_context.go` - Wire HalvingInterval and MaxBlockTxs to ChainConfig
- `etc/shitcoin.yaml` - HalvingInterval: 10 for demo

## Decisions Made
- Exported RewardAtHeight (was unexported rewardAtHeight) for testability and potential REST API use
- AddWithFee keeps backward-compatible Add() that delegates with fee=0 -- MempoolAdder interface unchanged
- MineBlock takes totalFees as parameter (caller computes from DrainByFee) rather than computing internally -- cleaner separation
- DrainByFee returns (txs, totalFees) tuple so callers get both in one atomic operation

## Deviations from Plan

None - plan executed exactly as written. Core implementation was partially present from a prior 06-02 commit; this plan added tests, exported RewardAtHeight, and wired CLI callers.

## Issues Encountered
None

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- Halving and fee infrastructure complete, ready for additional educational features
- MaxBlockTxs config is wired but not yet enforced in mine/autoMine (uses DrainByFee(0) which drains all)

---
*Phase: 06-advanced-educational-features*
*Completed: 2026-03-07*
