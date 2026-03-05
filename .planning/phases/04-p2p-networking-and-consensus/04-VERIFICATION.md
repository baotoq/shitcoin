---
phase: 04-p2p-networking-and-consensus
verified: 2026-03-05T16:10:00Z
status: passed
score: 5/5 must-haves verified
gaps: []
human_verification:
  - test: "Start node A on port 3001 with mining, start node B on port 3002 with -peers localhost:3001. Verify B syncs blocks and balances match."
    expected: "Node B logs IBD progress, reaches same chain height as A, getbalance returns correct amounts."
    why_human: "Full end-to-end multi-process verification with real bbolt storage and signal handling cannot be tested programmatically."
  - test: "Start 2 nodes mining simultaneously, disconnect one, mine different blocks, reconnect. Verify fork convergence."
    expected: "The node with the shorter chain reorganizes to the longer chain. Log output shows undo and reapply."
    why_human: "Timing-dependent multi-node fork scenario with real network requires manual orchestration."
---

# Phase 4: P2P Networking and Consensus Verification Report

**Phase Goal:** Multiple nodes on localhost discover each other, synchronize chains, broadcast blocks and transactions, and resolve forks via longest-chain rule
**Verified:** 2026-03-05T16:10:00Z
**Status:** passed
**Re-verification:** No -- initial verification

## Goal Achievement

### Observable Truths

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| 1 | User can start multiple nodes on different localhost ports and they connect with a version handshake | VERIFIED | `server.go:126-139` TCP listener on configurable port; `server.go:206-260` outbound/inbound handshake with genesis check; `cli.go:243-321` startnode with -port/-peers/-datadir flags; `TestHandshake` + `TestHandshakeGenesisMismatch` pass |
| 2 | A transaction created on one node appears in the mempool of all connected peers | VERIFIED | `handler.go:126-149` CmdTx handler adds to mempool via `mempool.Add()` and re-broadcasts inv; `cli.go:202-204` send command calls `server.BroadcastTx`; `server.go:104-118` BroadcastTx creates inv message; `relay_test.go` block broadcast tests pass |
| 3 | A block mined on one node is received, validated, and added to the chain on all peers | VERIFIED | `handler.go:154-258` handleBlock validates PoW, checks prevHash, applies UTXO, saves block, updates chain tip, re-broadcasts inv; `signal.go:103-104` autoMineWithP2P broadcasts mined blocks; `TestBlockBroadcast` passes |
| 4 | A newly started node synchronizes the full chain from an existing peer before accepting new blocks | VERIFIED | `sync.go:31-53` startSync triggered after handshake when peer.height > local; `sync.go:222-277` applySyncBlock applies blocks sequentially with UTXO; `sync.go:25-27` IsSyncing flag; `TestInitialBlockDownload` + `TestIBDUTXOConsistency` + `TestIBDSyncingFlag` pass |
| 5 | When two nodes mine competing blocks, the network converges on the longest valid chain via reorganization, correctly reversing and reapplying UTXO changes | VERIFIED | `chain.go:245-363` Reorganize undoes orphaned blocks in reverse, deletes them, applies new blocks, re-adds orphan txs to mempool; `sync.go:143-178` handleReorgBlock buffers fork blocks; `sync.go:181-220` executeReorg calls chain.Reorganize; `TestLongerChainReorg` (P2P fork convergence) + `TestReorganize_SwitchesToLongerFork` (unit with UTXO verification) pass |

**Score:** 5/5 truths verified

### Required Artifacts

| Artifact | Expected | Status | Details |
|----------|----------|--------|---------|
| `internal/domain/p2p/message.go` | Message types, command constants, payload structs | VERIFIED | 96 lines. CmdVersion through CmdTx constants, BlockPayload, TxPayload, HeaderPayload, InvPayload, GetBlocksPayload. All substantive. |
| `internal/domain/p2p/protocol.go` | Length-prefixed TCP framing | VERIFIED | 66 lines. WriteMessage/ReadMessage with 4-byte big-endian length header, MaxMessageSize validation. |
| `internal/domain/p2p/peer.go` | Peer struct with send channel and goroutines | VERIFIED | 101 lines. Buffered sendCh (cap 64), non-blocking Send, readLoop/writeLoop, sync.Once Stop. |
| `internal/domain/p2p/server.go` | TCP server, accept loop, peer manager, broadcast, handshake | VERIFIED | 426 lines. Start/Stop/Connect/Broadcast, outbound/inbound handshake with genesis check, seen-hash dedup, BroadcastBlock/BroadcastTx, OnBlockReceived callback. |
| `internal/domain/p2p/handler.go` | Message dispatch for all command types | VERIFIED | 308 lines. Handles CmdVersion, CmdVerack, CmdInv, CmdGetData, CmdGetBlocks, CmdBlock, CmdTx. PoW validation, UTXO application, mempool integration. |
| `internal/domain/p2p/errors.go` | Sentinel errors | VERIFIED | 17 lines. ErrMessageTooLarge, ErrHandshakeFailed, ErrIncompatibleGenesis, ErrProtocolViolation. |
| `internal/domain/p2p/sync.go` | IBD sync and fork detection | VERIFIED | 287 lines. startSync, handleSyncBlock, handleReorgBlock, executeReorg, applySyncBlock, abortSync, IsSyncing. Fork detection via hash comparison. |
| `internal/domain/p2p/payload.go` | Block/Tx P2P serialization with domain conversion | VERIFIED | 141 lines. BlockPayloadFromDomain, ToBlock, TxPayloadFromDomain, ToTransaction. |
| `internal/domain/chain/chain.go` | Reorganize method for fork resolution | VERIFIED | 363 lines. Reorganize method (lines 245-363): undo in reverse, delete orphans, apply new, re-add orphan txs via MempoolAdder interface. |
| `internal/domain/chain/repository.go` | Extended interface with GetUndoEntry, DeleteBlocksAbove | VERIFIED | 42 lines. Both methods present in Repository interface. |
| `internal/infrastructure/persistence/bbolt/chain_repo.go` | bbolt implementation of GetUndoEntry, DeleteBlocksAbove | VERIFIED | 407 lines. GetUndoEntry reads from undo bucket. DeleteBlocksAbove iterates height range, deletes block data + height index + undo entries atomically. |
| `internal/config/config.go` | P2PConfig with Port and Peers | VERIFIED | P2PConfig struct with Port (default 3000) and Peers (optional). Wired into Config. |
| `internal/handler/cli/cli.go` | startnode with P2P server, -port/-peers/-datadir, per-node data dirs | VERIFIED | 359 lines. startNode parses flags, derives data dir from port, creates per-node ServiceContext, creates/starts P2P server, connects to seed peers, wires autoMineWithP2P. |
| `internal/handler/cli/signal.go` | autoMineWithP2P with peer block cancellation | VERIFIED | 117 lines. Mining context cancelled by OnBlockReceived callback, broadcasts mined blocks via BroadcastBlock. |

### Key Link Verification

| From | To | Via | Status | Details |
|------|----|-----|--------|---------|
| cli.go | p2p/server.go | `p2p.NewServer` | WIRED | Line 281: `srv := p2p.NewServer(nodeSvc.Chain, nodeSvc.Mempool, nodeSvc.UTXOSet, nodeSvc.ChainRepo, *port)` |
| server.go | chain/chain.go | `chain.Chain` reference | WIRED | Server struct holds `chain *chain.Chain`, used in handshake for height/genesis, in handleBlock for tip comparison |
| handler.go | block/pow.go | `pow.Validate` | WIRED | Line 173: `if !s.pow.Validate(blk)` in handleBlock; line 99: same check in handleSyncBlock |
| handler.go | mempool/mempool.go | `mempool.Add` | WIRED | Line 140: `s.mempool.Add(transaction)` in handleTx; line 107: `s.mempool.GetByID(hash)` in handleGetData |
| cli.go | p2p/server.go | `server.BroadcastTx/BroadcastBlock` | WIRED | Line 203: `c.server.BroadcastTx(transaction, "")` in send; signal.go line 104: `srv.BroadcastBlock(blk, "")` in autoMineWithP2P |
| sync.go | chain/repository.go | `SaveBlockWithUTXOs` | WIRED | Line 243: `s.chainRepo.SaveBlockWithUTXOs(ctx, blk, undoEntry)` in applySyncBlock |
| handler.go | chain/repository.go | `GetBlocksInRange` | WIRED | Line 290: `s.chainRepo.GetBlocksInRange(ctx, payload.StartHeight, endHeight)` in handleGetBlocks |
| chain.go (Reorganize) | utxo/set.go | `UndoBlock/ApplyBlock` | WIRED | Line 285: `c.utxoSet.UndoBlock(undoEntry)` in reverse loop; line 315: `c.utxoSet.ApplyBlock(newBlk.Height(), txs)` in forward loop |
| p2p/handler.go | chain.go | `Reorganize` | WIRED | sync.go line 199: `s.chain.Reorganize(ctx, forkHeight, forkBlocks, s.mempool)` in executeReorg |
| chain.go (Reorganize) | mempool | `mempool.Add` for orphan txs | WIRED | Line 358: `mempoolAdder.Add(orphanTx)` via MempoolAdder interface |

### Requirements Coverage

| Requirement | Source Plan | Description | Status | Evidence |
|-------------|------------|-------------|--------|----------|
| NET-01 | 04-01 | User can start a node that listens on a configurable TCP port on localhost | SATISFIED | `server.go:126-139` Start method, `cli.go:243-321` startnode with -port flag |
| NET-02 | 04-01 | Nodes perform a version handshake when connecting | SATISFIED | `server.go:206-318` outbound/inbound handshake with genesis hash validation, TestHandshake passes |
| NET-04 | 04-02 | When a user creates a transaction, it is broadcast to all connected peers | SATISFIED | `cli.go:202-204` BroadcastTx in send, `handler.go:126-149` CmdTx relay |
| NET-05 | 04-02 | When a node mines a block, it is broadcast to all connected peers | SATISFIED | `signal.go:103-104` BroadcastBlock after mining, `handler.go:154-258` CmdBlock relay |
| NET-06 | 04-02 | Peers validate received blocks and transactions before accepting and re-broadcasting | SATISFIED | `handler.go:173` PoW validation, `handler.go:140` mempool validation for txs, invalid data rejected before relay |
| NET-07 | 04-03 | When a new node connects, it synchronizes the full chain from peers (IBD) | SATISFIED | `sync.go:31-53` startSync, `sync_test.go` TestInitialBlockDownload + TestIBDUTXOConsistency pass |
| NET-08 | 04-04 | Node detects when a peer has a longer valid chain and reorganizes | SATISFIED | `handler.go:188-196` fork detection triggers sync, `sync.go:181-220` executeReorg, TestLongerChainReorg passes |
| NET-09 | 04-04 | Chain reorganization reverses UTXO changes from orphaned blocks and applies new chain | SATISFIED | `chain.go:278-295` undo in reverse, `chain.go:297-329` apply new blocks, TestReorganize_SwitchesToLongerFork verifies UTXO balances |

### Anti-Patterns Found

| File | Line | Pattern | Severity | Impact |
|------|------|---------|----------|--------|
| None | - | - | - | No TODO/FIXME/HACK/PLACEHOLDER markers, no stub implementations, no empty handlers found |

### Human Verification Required

### 1. Multi-Node End-to-End Sync

**Test:** Start node A on port 3001 with `-mine ADDR`, start node B on port 3002 with `-peers localhost:3001`. Let A mine a few blocks.
**Expected:** Node B logs IBD progress, reaches same chain height as A, `getbalance` returns correct amounts on both nodes.
**Why human:** Full end-to-end multi-process verification with real bbolt storage and signal handling.

### 2. Fork Convergence with Real Nodes

**Test:** Start 2 nodes mining simultaneously on different ports. Stop one, mine different blocks on each. Reconnect. Observe log output.
**Expected:** The node with the shorter chain reorganizes to the longer chain. Log output shows "chain reorganization complete".
**Why human:** Timing-dependent multi-node fork scenario with real networking requires manual orchestration.

### Gaps Summary

No gaps found. All 5 observable truths are verified through substantive code review and passing tests. All 8 requirement IDs (NET-01, NET-02, NET-04, NET-05, NET-06, NET-07, NET-08, NET-09) are satisfied with implementation evidence. All artifacts exist at all 3 verification levels (exists, substantive, wired). All key links are connected. The full test suite passes with race detector enabled (0 failures across 11 packages). No anti-patterns detected.

---

_Verified: 2026-03-05T16:10:00Z_
_Verifier: Claude (gsd-verifier)_
