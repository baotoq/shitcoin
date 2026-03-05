---
phase: 04-p2p-networking-and-consensus
plan: 02
subsystem: p2p
tags: [tcp, relay, broadcast, inv, getdata, pow-validation, seen-hash]

# Dependency graph
requires:
  - phase: 04-01
    provides: TCP server, peer lifecycle, version handshake, message framing
  - phase: 02-03
    provides: UTXO set with ApplyBlock/UndoBlock for block acceptance
  - phase: 03-01
    provides: Mempool with signature/UTXO/double-spend validation
provides:
  - CmdInv, CmdGetData, CmdBlock, CmdTx relay handlers
  - BroadcastBlock/BroadcastTx with inv-based announcement
  - Seen-hash deduplication preventing infinite relay loops
  - PoW validation of received blocks
  - OnBlockReceived callback for mining interruption
  - CLI P2P broadcast integration (send + auto-mine)
affects: [04-03-initial-block-download, 04-04-consensus-and-fork-resolution]

# Tech tracking
tech-stack:
  added: []
  patterns: [inv-getdata relay pattern, seen-hash deduplication, mining cancellation on peer block]

key-files:
  created:
    - internal/domain/p2p/payload.go
    - internal/domain/p2p/relay_test.go
  modified:
    - internal/domain/p2p/handler.go
    - internal/domain/p2p/server.go
    - internal/domain/p2p/message.go
    - internal/domain/chain/chain.go
    - internal/domain/mempool/mempool.go
    - internal/handler/cli/cli.go
    - internal/handler/cli/signal.go

key-decisions:
  - "Inv-getdata relay pattern: nodes announce hashes via CmdInv, peers request full data via CmdGetData"
  - "Seen-hash maps (seenBlocks/seenTxs) with mutex prevent infinite broadcast loops"
  - "Chain.SetLatestBlock + RWMutex on Chain for thread-safe P2P block acceptance"
  - "autoMineWithP2P cancels mining context via OnBlockReceived callback when peer block arrives"
  - "BlockPayload/TxPayload P2P serialization types mirror bbolt storage model pattern"

patterns-established:
  - "Inv-getdata relay: announce hashes, request on demand, deduplicate via seen maps"
  - "Mining cancellation: context.WithCancel per mining attempt, cancelled by peer block callback"

requirements-completed: [NET-04, NET-05, NET-06]

# Metrics
duration: 9min
completed: 2026-03-05
---

# Phase 4 Plan 2: Transaction and Block Relay Summary

**Inv-getdata relay with PoW validation, seen-hash deduplication, and CLI P2P broadcast integration**

## Performance

- **Duration:** 9 min
- **Started:** 2026-03-05T15:27:09Z
- **Completed:** 2026-03-05T15:36:09Z
- **Tasks:** 2
- **Files modified:** 9

## Accomplishments
- Block relay: mined blocks propagate to all connected peers with PoW validation
- Seen-hash tracking prevents infinite relay loops (seenBlocks/seenTxs maps)
- Invalid blocks (bad PoW, wrong prevHash) rejected before relay
- CLI send command broadcasts transactions when running in startnode mode
- Auto-mine loop broadcasts mined blocks and cancels mining when peer block arrives

## Task Commits

Each task was committed atomically:

1. **Task 1: Block and transaction relay handlers with validation and seen-hash tracking** - `5076926` (feat)
2. **Task 2: Wire send and mine CLI commands to broadcast via P2P server** - `e9269ff` (feat)

## Files Created/Modified
- `internal/domain/p2p/payload.go` - BlockPayload/TxPayload P2P serialization with domain conversion
- `internal/domain/p2p/relay_test.go` - Tests: block broadcast, PoW validation rejection, seen-hash tracking
- `internal/domain/p2p/handler.go` - CmdInv, CmdGetData, CmdBlock, CmdTx message handlers
- `internal/domain/p2p/server.go` - BroadcastBlock/BroadcastTx, MarkSeen, OnBlockReceived callback
- `internal/domain/p2p/message.go` - BlockPayload, TxPayload, HeaderPayload, TxInputPayload types
- `internal/domain/chain/chain.go` - SetLatestBlock + RWMutex for thread-safe chain tip
- `internal/domain/mempool/mempool.go` - GetByID for GetData tx lookups
- `internal/handler/cli/cli.go` - Server field, BroadcastTx in send command
- `internal/handler/cli/signal.go` - autoMineWithP2P with peer block cancellation

## Decisions Made
- Inv-getdata relay pattern: nodes announce hashes via CmdInv, peers request full data via CmdGetData -- follows Bitcoin protocol
- Seen-hash maps (seenBlocks/seenTxs) with mutex prevent infinite broadcast loops
- Added RWMutex to Chain aggregate for thread-safe concurrent access from P2P handler goroutines
- autoMineWithP2P uses context.WithCancel per mining attempt, cancelled by OnBlockReceived callback
- BlockPayload/TxPayload mirror the bbolt storage model pattern for consistent serialization

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Added RWMutex to Chain for race safety**
- **Found during:** Task 1 (block broadcast test)
- **Issue:** Race detector flagged concurrent read/write on Chain.latestBlock from P2P handler goroutine and test goroutine
- **Fix:** Added sync.RWMutex to Chain, protected SetLatestBlock (write lock), Height/LatestBlock (read lock), MineBlock (write lock)
- **Files modified:** internal/domain/chain/chain.go
- **Verification:** All tests pass with -race flag
- **Committed in:** 5076926 (Task 1 commit)

**2. [Rule 2 - Missing Critical] Added Mempool.GetByID for GetData handler**
- **Found during:** Task 1 (implementing CmdGetData handler)
- **Issue:** Mempool had no method to look up a transaction by hash, needed for responding to GetData requests
- **Fix:** Added GetByID(id block.Hash) method to Mempool
- **Files modified:** internal/domain/mempool/mempool.go
- **Verification:** GetData handler can serve transaction requests from peers
- **Committed in:** 5076926 (Task 1 commit)

---

**Total deviations:** 2 auto-fixed (1 bug, 1 missing critical)
**Impact on plan:** Both fixes essential for correctness. No scope creep.

## Issues Encountered
- Inbound handshake protocol order required careful handling in test helper (server receives version first, sends second)

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- Relay infrastructure complete for initial block download (Plan 04-03)
- Seen-hash tracking and inv-getdata pattern ready for chain sync protocol
- OnBlockReceived callback enables mining coordination during sync

---
*Phase: 04-p2p-networking-and-consensus*
*Completed: 2026-03-05*
