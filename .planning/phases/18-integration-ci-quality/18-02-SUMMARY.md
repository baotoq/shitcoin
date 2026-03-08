---
phase: 18-integration-ci-quality
plan: 02
subsystem: infra
tags: [race-detection, ci, websocket, concurrency]

# Dependency graph
requires:
  - phase: 17-handler-layer-tests
    provides: "WebSocket hub tests confirming broadcast behavior"
provides:
  - "Race-safe ws.Hub broadcast eviction"
  - "CI pipeline with -race flag on all tests"
affects: []

# Tech tracking
tech-stack:
  added: []
  patterns: ["two-phase lock pattern: collect under RLock, mutate under Lock"]

key-files:
  created: []
  modified:
    - internal/handler/ws/hub.go
    - .github/workflows/ci-go.yml

key-decisions:
  - "Two-phase eviction pattern: collect slow clients under RLock, delete under full Lock"

patterns-established:
  - "Two-phase lock pattern: read under RLock, collect mutations, apply under Lock"

requirements-completed: [TINF-03]

# Metrics
duration: 1min
completed: 2026-03-08
---

# Phase 18 Plan 02: Race Detection Summary

**Two-phase broadcast eviction in ws.Hub eliminates data race, -race flag enabled in CI pipeline**

## Performance

- **Duration:** 1 min
- **Started:** 2026-03-08T05:24:50Z
- **Completed:** 2026-03-08T05:25:36Z
- **Tasks:** 1
- **Files modified:** 2

## Accomplishments
- Fixed delete-under-RLock data race in Hub.Run() broadcast case with two-phase eviction
- Enabled `-race` flag in CI workflow for all pushes and PRs
- All tests pass with `-race` detector (zero warnings)

## Task Commits

Each task was committed atomically:

1. **Task 1: Fix ws.Hub broadcast race and enable -race in CI** - `930f999` (fix)

## Files Created/Modified
- `internal/handler/ws/hub.go` - Two-phase broadcast eviction: collect slow clients under RLock, evict under Lock
- `.github/workflows/ci-go.yml` - Added -race flag to test step

## Decisions Made
- Two-phase eviction pattern: collect slow clients into slice under RLock, then iterate and delete under full Lock with existence check to avoid double-close

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered
None

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- Race detection active in CI, ensuring no future races are introduced
- All existing tests pass with race detector enabled

---
*Phase: 18-integration-ci-quality*
*Completed: 2026-03-08*
