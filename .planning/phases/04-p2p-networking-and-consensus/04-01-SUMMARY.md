---
phase: 04-p2p-networking-and-consensus
plan: 01
subsystem: p2p
tags: [tcp, networking, handshake, peer-management, protocol-framing]

# Dependency graph
requires:
  - phase: 03-mempool-mining-integration-and-cli
    provides: "Chain aggregate, mempool, CLI dispatch, auto-mining"
provides:
  - "P2P message types and command constants (Version, Verack, GetBlocks, Inv, GetData, Block, Tx)"
  - "Length-prefixed TCP wire protocol (WriteMessage/ReadMessage)"
  - "Peer struct with buffered send channel and read/write goroutines"
  - "TCP server with accept loop and peer registry"
  - "Version handshake with genesis hash validation"
  - "Per-node data directory isolation via -datadir flag"
  - "P2PConfig in config (Port, Peers)"
affects: [04-02-tx-block-relay, 04-03-chain-sync, 04-04-reorg]

# Tech tracking
tech-stack:
  added: []
  patterns: [length-prefixed-tcp-framing, peer-goroutine-lifecycle, version-handshake-protocol]

key-files:
  created:
    - internal/domain/p2p/message.go
    - internal/domain/p2p/protocol.go
    - internal/domain/p2p/peer.go
    - internal/domain/p2p/server.go
    - internal/domain/p2p/handler.go
    - internal/domain/p2p/errors.go
    - internal/domain/p2p/p2p_test.go
    - internal/domain/p2p/server_test.go
  modified:
    - internal/config/config.go
    - internal/handler/cli/cli.go

key-decisions:
  - "Port 0 in tests for OS-assigned free ports, avoiding test flakiness"
  - "10-second handshake deadline with conn.SetDeadline, cleared after handshake"
  - "Non-blocking peer.Send via select with default (drop if buffer full)"
  - "sync.Once in Peer.Stop to prevent double-close panics"
  - "Genesis hash comparison during handshake to reject incompatible chains"

patterns-established:
  - "Length-prefixed framing: [4-byte big-endian length][1-byte command][JSON payload]"
  - "Peer goroutine lifecycle: Start launches read+write goroutines, Stop closes done channel + connection"
  - "Handshake protocol: outbound sends Version first, inbound waits for Version first"

requirements-completed: [NET-01, NET-02]

# Metrics
duration: 6min
completed: 2026-03-05
---

# Phase 4 Plan 01: P2P Protocol Layer Summary

**TCP P2P server with length-prefixed message framing, peer goroutine management, and version handshake with genesis hash validation**

## Performance

- **Duration:** 6 min
- **Started:** 2026-03-05T15:18:05Z
- **Completed:** 2026-03-05T15:24:45Z
- **Tasks:** 2
- **Files modified:** 10

## Accomplishments
- Length-prefixed TCP wire protocol with round-trip message framing and size validation
- Peer struct with buffered send channel, read/write goroutines, and clean shutdown via sync.Once
- TCP server with accept loop, peer registry, broadcast, and version handshake protocol
- Genesis hash mismatch detection during handshake disconnects incompatible nodes
- Per-node data directories (data/node-{port}/) prevent bbolt lock conflicts
- CLI startnode updated with -port, -peers, -datadir flags for multi-node operation

## Task Commits

Each task was committed atomically:

1. **Task 1: P2P protocol types, message framing, peer struct, and sentinel errors** - `14a07d6` (feat)
2. **Task 2: TCP server, peer manager, version handshake, per-node data directory, and CLI wiring** - `66c5a58` (feat)

## Files Created/Modified
- `internal/domain/p2p/errors.go` - Sentinel errors (ErrMessageTooLarge, ErrHandshakeFailed, ErrIncompatibleGenesis, ErrProtocolViolation)
- `internal/domain/p2p/message.go` - Message type, command constants, payload structs (VersionPayload, InvPayload, GetBlocksPayload)
- `internal/domain/p2p/protocol.go` - WriteMessage/ReadMessage with length-prefixed TCP framing
- `internal/domain/p2p/peer.go` - Peer struct with send channel, read/write goroutines, non-blocking Send
- `internal/domain/p2p/server.go` - TCP server, accept loop, peer registry, version handshake, broadcast
- `internal/domain/p2p/handler.go` - Message dispatch (Version/Verack post-handshake handling)
- `internal/domain/p2p/p2p_test.go` - Unit tests for framing, serialization, peer lifecycle
- `internal/domain/p2p/server_test.go` - Integration tests for server listen, handshake, genesis mismatch
- `internal/config/config.go` - Added P2PConfig struct with Port and Peers fields
- `internal/handler/cli/cli.go` - Updated startnode with P2P server, -port/-peers/-datadir flags, per-node ServiceContext

## Decisions Made
- Port 0 in tests for OS-assigned free ports, avoiding test flakiness from port conflicts
- 10-second handshake deadline using conn.SetDeadline, cleared after handshake completes
- Non-blocking peer.Send via select with default drops messages when buffer is full (cap 64)
- sync.Once in Peer.Stop prevents double-close panics on connection and done channel
- Genesis hash fetched from chain repo at height 0 during handshake; empty hash skips check (empty chain)

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Fixed genesis mismatch test using different miner addresses**
- **Found during:** Task 2 (integration tests)
- **Issue:** Genesis blocks with different GenesisMessage but same miner address produce identical hashes because the message is not part of the header hash
- **Fix:** Used different miner addresses in test to produce genuinely different coinbase TXs and merkle roots
- **Files modified:** internal/domain/p2p/server_test.go
- **Verification:** TestHandshakeGenesisMismatch passes, correctly detecting different genesis hashes
- **Committed in:** 66c5a58 (Task 2 commit)

---

**Total deviations:** 1 auto-fixed (1 bug)
**Impact on plan:** Test approach adjustment only. No scope creep.

## Issues Encountered
- MockChainRepo.SaveBlock initially did not store the genesis block (only SaveBlockWithUTXOs did), causing getGenesisHash to return empty strings for both servers. Fixed by storing genesis in both Save methods.

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- P2P protocol layer complete, ready for transaction and block relay (Plan 02)
- Server.Broadcast available for message propagation
- Handler dispatch switch ready for additional command types (GetBlocks, Inv, GetData, Block, Tx)

---
*Phase: 04-p2p-networking-and-consensus*
*Completed: 2026-03-05*
