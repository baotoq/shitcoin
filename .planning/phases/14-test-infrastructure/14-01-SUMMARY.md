---
phase: 14-test-infrastructure
plan: 01
subsystem: testing
tags: [testutil, mocks, builders, tdd, testify]

requires: []
provides:
  - "Shared testutil package with test builders and mock repositories"
  - "MustCreateBlock, MustCreateBlockChain, MustCreateWallet, MustBuildSignedTx helpers"
  - "MockChainRepo, MockUTXORepo, MockWalletRepo implementing all domain interfaces"
affects: [15-domain-tests, 16-infrastructure-tests, 17-handler-tests, 18-integration-tests]

tech-stack:
  added: []
  patterns: [test-builder-pattern, in-memory-mock-repos, compile-time-interface-checks]

key-files:
  created:
    - internal/testutil/builders.go
    - internal/testutil/builders_test.go
    - internal/testutil/mock_chain_repo.go
    - internal/testutil/mock_chain_repo_test.go
    - internal/testutil/mock_utxo_repo.go
    - internal/testutil/mock_utxo_repo_test.go
    - internal/testutil/mock_wallet_repo.go
    - internal/testutil/mock_wallet_repo_test.go
  modified: []

key-decisions:
  - "Difficulty bits=1 for test mining -- fast block creation while still exercising real PoW"
  - "Exported map fields on mocks for test inspection (Blocks, UTXOs, Wallets, etc.)"
  - "Used domain error vars (ErrUTXONotFound, ErrWalletNotFound) for mock error returns"

patterns-established:
  - "Builder pattern: MustXxx functions using t.Helper() and require.NoError for test setup"
  - "Mock repos: in-memory maps with mutex protection and compile-time interface checks"
  - "TDD cycle: RED (failing tests) then GREEN (implementation) for each component"

requirements-completed: [TINF-01, TINF-02]

duration: 3min
completed: 2026-03-08
---

# Phase 14 Plan 01: Test Infrastructure Summary

**Shared testutil package with 5 builder helpers and 3 thread-safe mock repositories (19 methods total) covering chain, UTXO, and wallet interfaces**

## Performance

- **Duration:** 3 min
- **Started:** 2026-03-07T18:21:26Z
- **Completed:** 2026-03-07T18:24:52Z
- **Tasks:** 2
- **Files created:** 8 (908 lines total)

## Accomplishments
- Test builders producing valid mined blocks, signed transactions, and wallets with a single function call
- MockChainRepo implementing all 9 chain.Repository methods with RWMutex protection
- MockUTXORepo implementing all 7 utxo.Repository methods with Mutex protection
- MockWalletRepo implementing all 3 wallet.Repository methods with Mutex protection
- 30 tests passing with compile-time interface compliance verified

## Task Commits

Each task was committed atomically:

1. **Task 1: Create test builders with TDD** - `7569a0a` (test)
2. **Task 2: Create mock repositories with TDD** - `1d30a21` (test)

## Files Created/Modified
- `internal/testutil/builders.go` - MustCreateBlock, MustCreateBlockWithAddr, MustCreateBlockChain, MustCreateWallet, MustBuildSignedTx helpers
- `internal/testutil/builders_test.go` - Tests verifying builder output validity (PoW, signatures, hash linkage)
- `internal/testutil/mock_chain_repo.go` - In-memory chain.Repository with SaveBlock, GetBlock, DeleteBlocksAbove, etc.
- `internal/testutil/mock_chain_repo_test.go` - Interface compliance + stateful CRUD behavior tests
- `internal/testutil/mock_utxo_repo.go` - In-memory utxo.Repository with Put/Get/Delete/GetByAddress/UndoEntry ops
- `internal/testutil/mock_utxo_repo_test.go` - Interface compliance + CRUD behavior tests
- `internal/testutil/mock_wallet_repo.go` - In-memory wallet.Repository with Save/GetByAddress/ListAddresses
- `internal/testutil/mock_wallet_repo_test.go` - Interface compliance + CRUD behavior tests

## Decisions Made
- Used difficulty bits=1 for test mining to keep tests fast while exercising real PoW logic
- Exported map fields on mocks (e.g., MockChainRepo.Blocks) for direct test inspection
- Used domain error variables (utxo.ErrUTXONotFound, wallet.ErrWalletNotFound) in mock returns for ErrorIs compatibility

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered
None

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- testutil package fully operational for import by Phase 15-18 test files
- All domain repository interfaces covered with working mocks
- Builder functions produce valid domain objects ready for test assertions

---
*Phase: 14-test-infrastructure*
*Completed: 2026-03-08*
