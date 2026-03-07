---
phase: 14-test-infrastructure
plan: 02
subsystem: testing
tags: [go-test, mocks, testutil, refactoring]

requires:
  - phase: 14-01
    provides: shared testutil package with MockChainRepo, MockUTXORepo, MustCreateBlock builders
provides:
  - All test files migrated to shared testutil mocks (zero local mock duplication)
  - Fixed testutil MockChainRepo to return proper domain errors (chain.ErrBlockNotFound, chain.ErrChainEmpty, utxo.ErrUndoEntryNotFound)
affects: [15-domain-tests, 16-integration-tests, 17-handler-tests]

tech-stack:
  added: []
  patterns: [shared-testutil-mocks, external-test-packages]

key-files:
  created: []
  modified:
    - internal/testutil/mock_chain_repo.go
    - internal/domain/chain/chain_test.go
    - internal/domain/mempool/mempool_test.go
    - internal/domain/p2p/relay_test.go
    - internal/domain/p2p/reorg_test.go
    - internal/domain/p2p/server_test.go
    - internal/domain/p2p/sync_test.go
    - internal/handler/api/block_handler_test.go
    - internal/handler/api/status_handler_test.go

key-decisions:
  - "Fixed testutil MockChainRepo to return domain errors (chain.ErrBlockNotFound, chain.ErrChainEmpty) instead of generic errors.New -- required for chain.Initialize errors.Is checks"
  - "Kept MockMempoolAdder (testify mock.Mock) in chain_test.go -- different interface, not a repo mock"
  - "Kept buildSignedTx helper in mempool_test.go -- creates coinbase + applies block, different from testutil.MustBuildSignedTx"
  - "Kept createForkBlocks helper in reorg_test.go -- fork-specific block creation logic not covered by testutil builders"
  - "Switched mempool_test.go from package mempool to package mempool_test (external test package)"

patterns-established:
  - "External test packages: use package foo_test with testutil imports for clean dependency boundaries"
  - "Domain error returns: mock repos must return domain-specific sentinel errors, not generic errors"

requirements-completed: [TINF-02]

duration: 6min
completed: 2026-03-08
---

# Phase 14 Plan 02: Test Migration Summary

**Migrated 8 test files to shared testutil mocks, eliminating ~700 lines of duplicated mock code across chain, mempool, P2P, and API tests**

## Performance

- **Duration:** 6 min
- **Started:** 2026-03-07T18:27:56Z
- **Completed:** 2026-03-07T18:34:42Z
- **Tasks:** 2
- **Files modified:** 9

## Accomplishments
- Replaced all local mock repository implementations in 8 test files with testutil.NewMockChainRepo/NewMockUTXORepo
- Fixed testutil MockChainRepo error returns to use domain sentinel errors (critical for chain.Initialize)
- Removed testify mock.Mock dependency from server_test.go, replaced with stateful in-memory mock
- Discovered and migrated 2 additional files not in plan (sync_test.go, status_handler_test.go)

## Task Commits

Each task was committed atomically:

1. **Task 1: Migrate chain and mempool tests** - `41dc637` (refactor)
2. **Task 2: Migrate P2P and API handler tests** - `b4598b6` (refactor)

## Files Created/Modified
- `internal/testutil/mock_chain_repo.go` - Fixed to return domain errors instead of generic errors
- `internal/domain/chain/chain_test.go` - Uses testutil.NewMockChainRepo/NewMockUTXORepo
- `internal/domain/mempool/mempool_test.go` - Uses testutil.NewMockUTXORepo, switched to external test package
- `internal/domain/p2p/relay_test.go` - Uses testutil mocks, removed fullMockChainRepo/mockUTXORepo
- `internal/domain/p2p/reorg_test.go` - Uses testutil.NewMockChainRepo, removed reorgMockChainRepo
- `internal/domain/p2p/server_test.go` - Uses testutil.NewMockChainRepo, removed testify mock.Mock
- `internal/domain/p2p/sync_test.go` - Uses testutil mocks, removed fullMockChainRepo/mockUTXORepo references
- `internal/handler/api/block_handler_test.go` - Uses testutil.NewMockChainRepo/MustCreateBlock
- `internal/handler/api/status_handler_test.go` - Uses testutil.NewMockChainRepo/MustCreateBlock

## Decisions Made
- Fixed testutil MockChainRepo to return chain.ErrBlockNotFound, chain.ErrChainEmpty, and utxo.ErrUndoEntryNotFound instead of generic errors.New() -- chain.Initialize uses errors.Is() for these
- Kept 3 test-specific helpers (MockMempoolAdder, buildSignedTx, createForkBlocks) that serve distinct purposes not covered by testutil

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Fixed testutil MockChainRepo domain error returns**
- **Found during:** Task 1 (chain test migration)
- **Issue:** MockChainRepo returned errors.New("block not found") instead of chain.ErrBlockNotFound; chain.Initialize checks errors.Is(err, ErrChainEmpty) which would fail
- **Fix:** Changed GetBlock/GetBlockByHeight to return chain.ErrBlockNotFound, GetLatestBlock to return chain.ErrChainEmpty, GetUndoEntry to return utxo.ErrUndoEntryNotFound
- **Files modified:** internal/testutil/mock_chain_repo.go
- **Verification:** All testutil tests pass, all chain tests pass with Initialize working correctly
- **Committed in:** 41dc637 (Task 1 commit)

**2. [Rule 3 - Blocking] Migrated sync_test.go and status_handler_test.go**
- **Found during:** Task 2 (P2P test migration)
- **Issue:** sync_test.go and status_handler_test.go referenced the removed fullMockChainRepo/mockChainRepo causing compilation failure
- **Fix:** Migrated both files to use testutil mocks (same pattern as planned files)
- **Files modified:** internal/domain/p2p/sync_test.go, internal/handler/api/status_handler_test.go
- **Verification:** go test ./... passes with zero failures
- **Committed in:** b4598b6 (Task 2 commit)

---

**Total deviations:** 2 auto-fixed (1 bug, 1 blocking)
**Impact on plan:** Both fixes necessary for correctness. Bug fix prevents chain.Initialize from failing with wrong error type. Blocking fix addresses files that share test package with migrated files.

## Issues Encountered
None

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- All test files now use shared testutil package consistently
- Foundation ready for Phase 15 (domain test expansion) -- new tests can follow established testutil patterns
- No blockers

---
*Phase: 14-test-infrastructure*
*Completed: 2026-03-08*
