---
phase: 03-mempool-mining-integration-and-cli
plan: 02
subsystem: cli
tags: [cli, wallet, mempool, mining, signal-handling]

requires:
  - phase: 03-01
    provides: Mempool domain with Add/DrainAll and Merkle root computation
  - phase: 02-01
    provides: Wallet key generation, Base58Check addresses, JSON file persistence
  - phase: 02-02
    provides: Transaction creation, signing, verification, coinbase
  - phase: 02-03
    provides: UTXO set with GetByAddress/GetBalance, Chain.MineBlock with coinbase

provides:
  - CLI handler with 7 subcommands (createwallet, listaddresses, getbalance, send, mine, startnode, printchain)
  - ServiceContext with WalletRepo and Mempool wired
  - Auto-mining loop with graceful SIGINT/SIGTERM shutdown
  - CLI-driven main.go replacing demo loop

affects: [04-p2p-networking, cli-extensions]

tech-stack:
  added: []
  patterns: [flag.NewFlagSet per subcommand, signal.Notify for graceful shutdown, context.WithCancel for auto-mine lifecycle]

key-files:
  created:
    - internal/handler/cli/cli.go
    - internal/handler/cli/signal.go
  modified:
    - internal/config/config.go
    - internal/svc/service_context.go
    - cmd/shitcoin/main.go
    - etc/shitcoin.yaml

key-decisions:
  - "flag.Args() passed to CLI.Run() so -f config flag and subcommands coexist cleanly"
  - "Auto-mine loop uses context.WithCancel + signal.Notify for clean shutdown on SIGINT/SIGTERM"
  - "Send command uses simple greedy UTXO selection (iterate until accumulated >= amount)"

patterns-established:
  - "CLI subcommand pattern: flag.NewFlagSet per command, parse args[1:]"
  - "Signal handling pattern: goroutine listening on sigCh, calling cancel()"

requirements-completed: [MINE-04, MINE-05, CLI-01, CLI-02, CLI-03, CLI-04, CLI-05, CLI-06, CLI-07]

duration: 3min
completed: 2026-03-05
---

# Phase 3 Plan 02: CLI Integration and Mining Commands Summary

**Full CLI application with 7 subcommands: wallet management, UTXO-based send, mempool-draining mine, and auto-mining startnode with graceful shutdown**

## Performance

- **Duration:** 3 min
- **Started:** 2026-03-05T14:53:55Z
- **Completed:** 2026-03-05T14:56:27Z
- **Tasks:** 2
- **Files modified:** 6

## Accomplishments
- Wired WalletRepo and Mempool into ServiceContext, completing full dependency graph
- Implemented all 7 CLI subcommands with proper flag parsing and error handling
- Replaced demo mining loop with CLI-driven main.go entry point
- Auto-mining with graceful SIGINT/SIGTERM shutdown via context cancellation

## Task Commits

Each task was committed atomically:

1. **Task 1: Config, ServiceContext, and CLI scaffold with wallet/query commands** - `243aeb2` (feat)
2. **Task 2: Send, mine, startnode commands with auto-mining and main.go** - `d4d3e71` (feat)

## Files Created/Modified
- `internal/handler/cli/cli.go` - CLI struct with Run() dispatch and all 7 subcommands (createwallet, listaddresses, getbalance, send, mine, startnode, printchain)
- `internal/handler/cli/signal.go` - Auto-mine loop and signal handling for graceful shutdown
- `internal/config/config.go` - Added WalletPath to StorageConfig
- `internal/svc/service_context.go` - Added WalletRepo and Mempool fields, wiring in NewServiceContext
- `cmd/shitcoin/main.go` - Replaced demo loop with CLI dispatch via flag.Args()
- `etc/shitcoin.yaml` - Added WalletPath under Storage section

## Decisions Made
- Used flag.Args() passed to CLI.Run() so -f config flag and subcommands coexist without conflict
- Auto-mine loop uses context.WithCancel + signal.Notify for clean shutdown on SIGINT/SIGTERM
- Send command uses simple greedy UTXO selection (iterate until accumulated >= amount)
- Separated signal handling into signal.go for clean code organization

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered
None

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- All domain packages wired into working CLI application
- Full end-to-end flow available: createwallet -> mine -> send -> mine -> getbalance
- Ready for Phase 4 (P2P networking) which will add network message handling alongside existing CLI

---
*Phase: 03-mempool-mining-integration-and-cli*
*Completed: 2026-03-05*
