---
phase: 06-advanced-educational-features
plan: 02
subsystem: cli
tags: [testnet, multi-node, process-management, os-exec]

# Dependency graph
requires:
  - phase: 04-p2p-networking
    provides: "startnode subcommand with P2P, peers, datadir flags"
provides:
  - "testnet CLI command for single-command multi-node local network"
  - "-http-port flag on startnode for per-node HTTP port override"
affects: [06-advanced-educational-features]

# Tech tracking
tech-stack:
  added: []
  patterns: ["os/exec.CommandContext for child process spawning", "process group SIGTERM/SIGKILL for clean shutdown", "prefixed stdout/stderr pipes for multi-process output"]

key-files:
  created: ["internal/handler/cli/testnet.go"]
  modified: ["internal/handler/cli/cli.go", "internal/handler/cli/signal.go"]

key-decisions:
  - "os.Args[0] as child binary path for go run compatibility"
  - "SysProcAttr Setpgid for process group cleanup"
  - "5-second SIGTERM timeout before SIGKILL escalation"
  - "-http-port flag added to startnode for per-node HTTP port override"

patterns-established:
  - "Process group management: Setpgid + kill(-pid) for child cleanup"
  - "Prefixed output: scanner goroutines prefix each line with [node-N]"

requirements-completed: [ORCH-01]

# Metrics
duration: 3min
completed: 2026-03-07
---

# Phase 06 Plan 02: Testnet CLI Command Summary

**Single-command testnet launcher spawning N nodes with auto-mining, peer connections, and prefixed output**

## Performance

- **Duration:** 3 min
- **Started:** 2026-03-07T09:00:32Z
- **Completed:** 2026-03-07T09:03:52Z
- **Tasks:** 1
- **Files modified:** 14

## Accomplishments
- Testnet command spawns N nodes on sequential P2P and HTTP ports
- Node 0 auto-creates wallet and mines as seed node
- Other nodes auto-connect to node 0 via -peers flag
- All child processes terminate cleanly on Ctrl+C with SIGTERM/SIGKILL escalation
- Each node's output prefixed with [node-N] for readability

## Task Commits

Each task was committed atomically:

1. **Task 1: Implement testnet CLI subcommand** - `302711c` (feat)

## Files Created/Modified
- `internal/handler/cli/testnet.go` - Testnet command: spawns N child processes, manages lifecycle
- `internal/handler/cli/cli.go` - Added testnet case, usage, -http-port flag to startnode, fixed call sites
- `internal/handler/cli/signal.go` - Fixed MineBlock call sites for new totalFees parameter

## Decisions Made
- Used os.Args[0] as child binary path so testnet works with both compiled binary and `go run`
- Added -http-port flag to startnode (not in original plan) to allow per-node HTTP port assignment
- Process group cleanup via syscall.Kill(-pid, SIGTERM) for reliable child termination
- 5-second graceful shutdown timeout before SIGKILL escalation

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] Fixed MineBlock call sites for new totalFees parameter**
- **Found during:** Task 1 (build verification)
- **Issue:** MineBlock signature changed to require totalFees int64 but callers in cli.go, signal.go, and test files were not updated
- **Fix:** Added `0` as totalFees argument to all call sites
- **Files modified:** cli.go, signal.go, chain_test.go, relay_test.go, reorg_test.go, sync_test.go
- **Verification:** go build ./... and go test ./... pass
- **Committed in:** 302711c

**2. [Rule 3 - Blocking] Fixed CreateTransactionWithChange call site for new fee parameter**
- **Found during:** Task 1 (build verification)
- **Issue:** CreateTransactionWithChange signature changed to require fee int64 but caller in cli.go send() was not updated
- **Fix:** Added `0` as fee argument
- **Files modified:** cli.go
- **Verification:** go build ./... passes
- **Committed in:** 302711c

**3. [Rule 3 - Blocking] Added -http-port flag to startnode**
- **Found during:** Task 1 (testnet needs per-node HTTP ports)
- **Issue:** startnode had no way to override HTTP port; all testnet nodes would bind to same port
- **Fix:** Added -http-port flag to startnode that overrides config.Port when > 0
- **Files modified:** cli.go
- **Verification:** go build ./... passes
- **Committed in:** 302711c

---

**Total deviations:** 3 auto-fixed (3 blocking)
**Impact on plan:** All auto-fixes necessary for compilation and correct multi-node operation. No scope creep.

## Issues Encountered
- Pre-existing uncommitted changes from 06-01 plan (halving, fees, config) were included in commit since they affected compilation

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- Testnet command ready for double-spend demo (Plan 03)
- All nodes spawn with unique P2P and HTTP ports

---
*Phase: 06-advanced-educational-features*
*Completed: 2026-03-07*
