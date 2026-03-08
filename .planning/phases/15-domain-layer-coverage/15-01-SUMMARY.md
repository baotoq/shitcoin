---
phase: 15-domain-layer-coverage
plan: 01
subsystem: testing
tags: [go-test, coverage, error-paths, utxo, tx, wallet, mempool]

requires:
  - phase: 14-test-infrastructure
    provides: shared testutil package with mock repos and builders
provides:
  - tx package 100% coverage with error path tests
  - utxo package 100% coverage with repo error injection tests
  - wallet package 97.8% coverage with PubKeyHashFromAddress tests
  - mempool package 100% coverage with GetByID/FeeForTx/Remove tests
affects: [15-domain-layer-coverage]

tech-stack:
  added: []
  patterns: [error-returning mock repo pattern for UTXO error injection]

key-files:
  created: []
  modified:
    - internal/domain/tx/transaction_test.go
    - internal/domain/utxo/set_test.go
    - internal/domain/wallet/wallet_test.go
    - internal/domain/wallet/base58_test.go
    - internal/domain/mempool/mempool_test.go

key-decisions:
  - "Error-returning mock repo (errRepo) wraps memRepo for targeted error injection in utxo tests"
  - "Wallet coverage at 97.8% exceeds 93% target -- unreachable NewWallet crypto branch is only uncovered line"

patterns-established:
  - "errRepo pattern: wrap existing memRepo with configurable error fields for targeted failure injection"
  - "Table-driven error path tests with setup functions for repo configuration"

requirements-completed: [DOM-03, DOM-04]

duration: 2min
completed: 2026-03-08
---

# Phase 15 Plan 01: Domain Layer Gap-Fill Summary

**Error path and edge case tests for tx, utxo, wallet, and mempool packages bringing all four to 95%+ coverage**

## Performance

- **Duration:** 2 min
- **Started:** 2026-03-08T03:52:44Z
- **Completed:** 2026-03-08T03:55:04Z
- **Tasks:** 2
- **Files modified:** 5

## Accomplishments
- tx package coverage: 94.4% -> 100% (coinbase signing noop, invalid signatures, multi-output coinbase)
- mempool package coverage: 90.9% -> 100% (GetByID, FeeForTx, Remove non-existent)
- utxo package coverage: 86.2% -> 100% (UndoBlock error paths, ApplyBlock repo errors, GetBalance errors)
- wallet package coverage: 87.6% -> 97.8% (PubKeyHashFromAddress, Base58CheckDecode short input)

## Task Commits

Each task was committed atomically:

1. **Task 1: Gap-fill tx and mempool test coverage** - `42f306c` (test)
2. **Task 2: Gap-fill utxo and wallet test coverage** - `e7741a3` (test)

## Files Created/Modified
- `internal/domain/tx/transaction_test.go` - Added coinbase signing noop, invalid signature table-driven tests, multi-output coinbase validation
- `internal/domain/utxo/set_test.go` - Added errRepo mock, UndoBlock/ApplyBlock/GetBalance error path tests
- `internal/domain/wallet/wallet_test.go` - Added PubKeyHashFromAddress tests (valid, invalid checksum, wrong version, short)
- `internal/domain/wallet/base58_test.go` - Added Base58CheckDecode short/empty input edge cases
- `internal/domain/mempool/mempool_test.go` - Added GetByID found/not-found, FeeForTx not-found, Remove non-existent

## Decisions Made
- Created errRepo wrapper around memRepo in utxo package tests for error injection rather than modifying shared testutil mock
- Wallet 97.8% coverage accepted (exceeds 93% plan target) since remaining uncovered line is unreachable crypto error in NewWallet

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered
None

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- All four domain packages exceed 95% coverage target
- Error path patterns established for use in remaining domain coverage plans
- Ready for 15-02 (block/chain/events coverage) and 15-03 (p2p coverage)

---
*Phase: 15-domain-layer-coverage*
*Completed: 2026-03-08*
