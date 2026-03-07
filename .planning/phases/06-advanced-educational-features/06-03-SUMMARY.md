---
phase: 06-advanced-educational-features
plan: 03
subsystem: cli
tags: [demo, double-spend, mempool, utxo, educational]

# Dependency graph
requires:
  - phase: 06-02
    provides: testnet command pattern, mempool with fee tracking
provides:
  - "demo doublespend CLI command with in-process scripted scenario"
  - "Two-layer double-spend detection demonstration (mempool + UTXO set)"
affects: []

# Tech tracking
tech-stack:
  added: []
  patterns: ["In-process demo with isolated temp ServiceContext and low difficulty"]

key-files:
  created:
    - internal/handler/cli/demo.go
  modified:
    - internal/handler/cli/cli.go

key-decisions:
  - "In-process demo using domain objects directly rather than subprocess spawning"
  - "Fresh mempool for Step 4 to demonstrate UTXO-level rejection after DrainByFee clears tracking"
  - "Difficulty 1 and halving disabled for sub-second demo execution"

patterns-established:
  - "Demo pattern: isolated temp dir + custom config + ServiceContext for self-contained educational scenarios"

requirements-completed: [DEMO-01]

# Metrics
duration: 2min
completed: 2026-03-07
---

# Phase 06 Plan 03: Demo Double-Spend Summary

**In-process `demo doublespend` command showing mempool and UTXO set two-layer double-spend rejection**

## Performance

- **Duration:** 2 min
- **Started:** 2026-03-07T09:10:25Z
- **Completed:** 2026-03-07T09:12:09Z
- **Tasks:** 1
- **Files modified:** 2

## Accomplishments
- `demo doublespend` command runs complete scripted scenario with educational output
- Double-spend rejected by mempool (ErrDoubleSpend) when UTXO already claimed by pending TX
- Double-spend rejected by UTXO set (ErrUTXONotFound) after confirming TX is mined in a block
- Summary explains the two-layer protection model (mempool tracking + UTXO consumption)

## Task Commits

Each task was committed atomically:

1. **Task 1: Implement demo doublespend command** - `107d721` (feat)

## Files Created/Modified
- `internal/handler/cli/demo.go` - Demo CLI command with doublespend subcommand and in-process scenario
- `internal/handler/cli/cli.go` - Added `demo` case to Run switch and usage text

## Decisions Made
- Used in-process domain objects (chain, mempool, wallet, UTXO set) instead of subprocess spawning for reliability and speed
- Created fresh mempool for Step 4 since DrainByFee clears spent output tracking, ensuring UTXO-level rejection is demonstrated cleanly
- Set difficulty to 1 bit and disabled halving for instant mining in demo context

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered
None

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- All Phase 06 plans complete (halving/fees, testnet, demo)
- Ready for Phase 07

---
*Phase: 06-advanced-educational-features*
*Completed: 2026-03-07*
