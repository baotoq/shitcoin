# Phase 4: P2P Networking and Consensus - Research

**Researched:** 2026-03-05
**Domain:** TCP peer-to-peer networking, message-based protocol, chain synchronization, fork resolution
**Confidence:** HIGH

## Summary

Phase 4 transforms the single-node blockchain into a multi-node distributed system. The project needs a custom TCP-based message protocol (not HTTP/gRPC -- this is an educational Bitcoin-style project) where nodes connect, exchange version handshakes, broadcast transactions and blocks, perform initial block download (IBD), and resolve forks via longest-chain reorganization.

The existing codebase is well-prepared: UTXO undo-log is already implemented (`utxo.UndoBlock`), block validation exists (`ProofOfWork.Validate`), mempool has concurrent-safe Add/Remove, and the chain repository supports `GetBlocksInRange` for serving blocks during sync. The `startnode` CLI command already accepts a `-port` flag (currently unused). The key work is: (1) TCP server/client with length-prefixed message framing, (2) a typed message protocol with versioning, (3) a peer manager tracking connected peers, (4) block/tx relay logic, (5) IBD sync, and (6) chain reorganization integrating the existing undo-log.

**Primary recommendation:** Use Go's `net` package for raw TCP with a simple length-prefixed binary message protocol. Keep it educational -- no protobuf, no libp2p. Each message is: 4-byte length + 1-byte command type + JSON payload. This mirrors Bitcoin's simplicity while remaining debuggable.

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|-----------------|
| NET-01 | User can start a node that listens on a configurable TCP port on localhost | TCP server in `net` package, integrated into `startnode` CLI command |
| NET-02 | Nodes perform a version handshake when connecting to establish protocol compatibility | Version message with chain height + protocol version, handshake state machine |
| NET-04 | When a user creates a transaction, it is broadcast to all connected peers | Peer manager broadcast method, `inv`/`tx` message types |
| NET-05 | When a node mines a block, it is broadcast to all connected peers | Block relay via `inv`/`block` message types |
| NET-06 | Peers validate received blocks and transactions before accepting and re-broadcasting | Existing `ProofOfWork.Validate` + `tx.VerifyTransaction` + mempool validation |
| NET-07 | When a new node connects, it synchronizes the full chain from peers (initial block download) | `getblocks`/`blocks` message exchange, sequential download |
| NET-08 | Node detects when a peer has a longer valid chain and reorganizes to the longest chain | Height comparison during handshake + block announcements, reorg trigger |
| NET-09 | Chain reorganization reverses UTXO changes from orphaned blocks and applies the new chain's changes | Existing `utxo.Set.UndoBlock` + `ApplyBlock`, find common ancestor |
</phase_requirements>

## Standard Stack

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| `net` (stdlib) | Go 1.24 | TCP server/client connections | Standard Go networking, no external deps needed for localhost TCP |
| `encoding/json` (stdlib) | Go 1.24 | Message payload serialization | Already used throughout project for deterministic serialization |
| `encoding/binary` (stdlib) | Go 1.24 | Length-prefix framing (4-byte big-endian) | Standard binary protocol pattern |
| `sync` (stdlib) | Go 1.24 | RWMutex for peer map, chain tip | Already used in mempool; same pattern for peer registry |
| `context` (stdlib) | Go 1.24 | Goroutine lifecycle, graceful shutdown | Already used for auto-mining cancellation |

### Supporting
| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| `log/slog` (stdlib) | Go 1.24 | Structured logging for P2P events | Debugging multi-node communication |
| `io` (stdlib) | Go 1.24 | `io.ReadFull` for reliable message reading | Reading length-prefixed frames from TCP |
| `time` (stdlib) | Go 1.24 | Connection timeouts, handshake deadlines | Preventing hung connections |
| `errors` (stdlib) | Go 1.24 | Sentinel errors for protocol violations | Clean error handling in message processing |

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| Raw TCP + JSON | gRPC/protobuf | gRPC adds complexity; raw TCP is more educational and Bitcoin-like |
| Raw TCP | libp2p | Massive dependency, hides networking concepts this project teaches |
| JSON payloads | Binary encoding | JSON is debuggable (project decision from Phase 1), performance not a concern |
| Manual peer list | mDNS discovery | Out of scope (ANET-01 is v2); manual peer list via CLI flags is simpler |

**Installation:**
```bash
# No new dependencies needed -- all stdlib
```

## Architecture Patterns

### Recommended Project Structure
```
internal/
  domain/
    p2p/
      message.go       # Message types, serialization, command constants
      protocol.go      # Framing (read/write length-prefixed messages)
      peer.go          # Peer struct (conn, addr, state, send channel)
      server.go        # TCP listener, accept loop
      handler.go       # Message dispatch (switch on command type)
      sync.go          # Initial block download logic
      errors.go        # Sentinel errors
  domain/
    chain/
      chain.go         # Extended: AddBlock (validate+store received block), Reorganize
      repository.go    # Extended: GetUndoEntry, DeleteBlocksAbove
```

### Pattern 1: Length-Prefixed Message Framing
**What:** Every TCP message is: `[4-byte big-endian length][1-byte command][JSON payload]`
**When to use:** All peer-to-peer communication
**Example:**
```go
// Writing a message
func WriteMessage(w io.Writer, msg Message) error {
    payload, err := json.Marshal(msg.Payload)
    if err != nil {
        return fmt.Errorf("marshal payload: %w", err)
    }
    frame := make([]byte, 4+1+len(payload))
    binary.BigEndian.PutUint32(frame[:4], uint32(1+len(payload)))
    frame[4] = msg.Command
    copy(frame[5:], payload)
    _, err = w.Write(frame)
    return err
}

// Reading a message
func ReadMessage(r io.Reader) (Message, error) {
    lenBuf := make([]byte, 4)
    if _, err := io.ReadFull(r, lenBuf); err != nil {
        return Message{}, err
    }
    length := binary.BigEndian.Uint32(lenBuf)
    if length > MaxMessageSize {
        return Message{}, ErrMessageTooLarge
    }
    data := make([]byte, length)
    if _, err := io.ReadFull(r, data); err != nil {
        return Message{}, err
    }
    return Message{Command: data[0], Payload: data[1:]}, nil
}
```

### Pattern 2: Per-Peer Goroutine with Send Channel
**What:** Each peer connection gets a read goroutine and a write goroutine. Writes go through a buffered channel to avoid blocking.
**When to use:** Every connected peer
**Example:**
```go
type Peer struct {
    conn    net.Conn
    addr    string
    sendCh  chan Message
    height  uint64  // peer's chain height from handshake
    version uint32  // protocol version
}

func (p *Peer) writeLoop() {
    for msg := range p.sendCh {
        if err := WriteMessage(p.conn, msg); err != nil {
            return // connection broken
        }
    }
}

func (p *Peer) readLoop(handler func(Peer, Message)) {
    for {
        msg, err := ReadMessage(p.conn)
        if err != nil {
            return // connection broken
        }
        handler(*p, msg)
    }
}
```

### Pattern 3: Peer Manager as Central Coordinator
**What:** A single PeerManager struct owns the peer registry, handles broadcast, and coordinates sync.
**When to use:** All multi-peer operations
**Example:**
```go
type PeerManager struct {
    mu       sync.RWMutex
    peers    map[string]*Peer  // addr -> Peer
    chain    *chain.Chain
    mempool  *mempool.Mempool
    utxoSet  *utxo.Set
}

func (pm *PeerManager) Broadcast(msg Message, exclude string) {
    pm.mu.RLock()
    defer pm.mu.RUnlock()
    for addr, peer := range pm.peers {
        if addr == exclude {
            continue
        }
        select {
        case peer.sendCh <- msg:
        default:
            // channel full, peer is slow -- skip or disconnect
        }
    }
}
```

### Pattern 4: Chain Reorganization
**What:** When receiving a longer valid chain, undo blocks back to common ancestor, then apply new blocks forward.
**When to use:** NET-08 and NET-09
**Example:**
```go
func (c *Chain) Reorganize(ctx context.Context, newBlocks []*block.Block) error {
    // 1. Find common ancestor (where chains diverge)
    // 2. Undo blocks from current tip back to ancestor using UndoBlock
    // 3. Apply new blocks forward using ApplyBlock
    // 4. Update chain tip
    // 5. Re-add orphaned transactions back to mempool (if not in new chain)
}
```

### Pattern 5: Message Protocol Commands
**What:** Simple command byte enumeration for all message types
**When to use:** Protocol definition
```go
const (
    CmdVersion    byte = 0x01  // Handshake: send chain height + protocol version
    CmdVerack     byte = 0x02  // Handshake acknowledgment
    CmdGetBlocks  byte = 0x03  // Request block hashes from a height
    CmdInv        byte = 0x04  // Inventory announcement (block/tx hashes)
    CmdGetData    byte = 0x05  // Request full block/tx by hash
    CmdBlock      byte = 0x06  // Full block data
    CmdTx         byte = 0x07  // Full transaction data
    CmdGetHeaders byte = 0x08  // Request block headers only (for sync)
)
```

### Anti-Patterns to Avoid
- **Shared mutable state without locks:** The peer map and chain tip MUST be protected by mutex. Multiple goroutines (one per peer) will read/write concurrently.
- **Blocking writes on TCP connections:** Never write directly in the message handler goroutine. Always use a buffered send channel per peer to decouple read and write.
- **Processing received blocks without validation:** Always validate PoW, previous hash linkage, Merkle root, and transaction signatures before accepting. Never trust peers.
- **Full chain download on every connection:** Only sync if peer has a higher chain. Compare heights during handshake.
- **Mining while syncing:** Disable auto-mining during IBD. Mining on a partial chain wastes work and creates invalid blocks.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| TCP framing | Custom delimiter parsing | Length-prefixed framing with `io.ReadFull` | Delimiter-based parsing is error-prone with binary data; length-prefix is simple and reliable |
| Graceful shutdown | Manual goroutine tracking | `context.Context` + `sync.WaitGroup` | Already established pattern in the project (signal.go) |
| Block validation | New validation logic | Existing `ProofOfWork.Validate` + `tx.VerifyTransaction` | Already built and tested |
| UTXO reversal | New undo mechanism | Existing `utxo.Set.UndoBlock` with `UndoEntry` | Designed in Phase 2 specifically for this purpose |
| Concurrent map access | Channel-based map | `sync.RWMutex` + regular map | Project pattern (mempool uses this); simpler than channel coordination |

**Key insight:** Phase 2 and 3 built the UTXO undo-log, block validation, and mempool specifically to be consumed by this P2P phase. The networking layer's job is to shuttle bytes and coordinate -- the validation and state management logic already exists.

## Common Pitfalls

### Pitfall 1: Separate Database Per Node
**What goes wrong:** All nodes sharing the same bbolt database file causes corruption or lock contention (bbolt is single-writer).
**Why it happens:** The config has a single `Storage.DBPath` value; running multiple nodes without changing it means they collide.
**How to avoid:** Each node must use a unique data directory. The `-port` flag should derive a unique DB path (e.g., `data/node-{port}/shitcoin.db`). Or add a `-datadir` flag.
**Warning signs:** "database is locked" errors, or nodes seeing each other's data.

### Pitfall 2: Deadlock in Message Handler
**What goes wrong:** Message handler calls Broadcast, which tries to write to the peer that sent the message, which is waiting for the handler to finish reading.
**Why it happens:** Read and write happen on the same goroutine, or send channel is unbuffered.
**How to avoid:** Separate read/write goroutines per peer. Buffered send channel. Non-blocking sends with `select { case ch <- msg: default: }`.
**Warning signs:** All nodes freeze, no messages flowing.

### Pitfall 3: Race Condition Between Mining and Block Receipt
**What goes wrong:** Node receives a block from peer while simultaneously mining. Both try to update chain tip. UTXO state becomes inconsistent.
**Why it happens:** Mining loop and message handler both call `Chain.MineBlock` / `Chain.AddBlock` without coordination.
**How to avoid:** Use a mutex on chain state mutations. When a valid block is received for the same height being mined, cancel the current mining attempt.
**Warning signs:** Duplicate blocks at same height, UTXO "not found" errors.

### Pitfall 4: Genesis Block Mismatch
**What goes wrong:** Nodes cannot sync because they have different genesis blocks (different genesis messages, different difficulty).
**Why it happens:** Nodes started with different config files.
**How to avoid:** Version handshake should include genesis block hash. Disconnect peers with different genesis hashes.
**Warning signs:** Sync gets stuck at height 0, blocks rejected as invalid.

### Pitfall 5: Infinite Re-broadcast
**What goes wrong:** Node A sends block to B, B broadcasts to A, A broadcasts to B again...
**Why it happens:** No tracking of which messages have already been sent/received.
**How to avoid:** Track seen block/tx hashes in a set per peer. Use `inv` messages to announce before sending full data. Only send `getdata` for unknown hashes.
**Warning signs:** CPU spikes, network bandwidth explosion, log spam.

### Pitfall 6: Reorganization Atomicity
**What goes wrong:** Reorg crashes midway -- some blocks undone but new blocks not applied. UTXO set is in an inconsistent state.
**Why it happens:** Undo and apply are done as separate bbolt transactions.
**How to avoid:** The entire reorg (undo old blocks + apply new blocks) must happen in a single bbolt `Update` transaction, or at minimum use a "reorg in progress" flag that triggers recovery on restart.
**Warning signs:** Balances are wrong after restart, "UTXO not found" for known transactions.

## Code Examples

### Message Type Definitions
```go
// VersionPayload is sent during handshake to exchange node info.
type VersionPayload struct {
    Version     uint32 `json:"version"`      // Protocol version
    Height      uint64 `json:"height"`       // Current chain height
    GenesisHash string `json:"genesis_hash"` // For compatibility check
    ListenPort  int    `json:"listen_port"`  // For peer discovery
}

// InvPayload announces available blocks or transactions.
type InvPayload struct {
    Type   string   `json:"type"`   // "block" or "tx"
    Hashes []string `json:"hashes"` // Hex-encoded hashes
}

// GetBlocksPayload requests blocks starting from a height.
type GetBlocksPayload struct {
    StartHeight uint64 `json:"start_height"`
    EndHeight   uint64 `json:"end_height"` // 0 means "up to your tip"
}
```

### TCP Server Integration with startnode
```go
func (c *CLI) startNode(args []string) {
    fs := flag.NewFlagSet("startnode", flag.ExitOnError)
    port := fs.Int("port", 3000, "TCP port for P2P")
    mineAddr := fs.String("mine", "", "Mining address")
    peers := fs.String("peers", "", "Comma-separated peer addresses (host:port)")
    datadir := fs.String("datadir", "", "Data directory (default: data/node-{port})")
    fs.Parse(args)

    // Derive per-node data directory from port
    // Initialize chain with node-specific storage
    // Start TCP listener
    // Connect to seed peers
    // Optionally start auto-mining
    // Wait for shutdown signal
}
```

### Initial Block Download Flow
```go
// After handshake reveals peer has higher chain:
// 1. Send GetBlocks{StartHeight: myHeight+1, EndHeight: 0}
// 2. Peer responds with Block messages for each requested block
// 3. Validate and apply each block sequentially
// 4. Repeat if peer has more blocks
// 5. Once caught up, start accepting live broadcasts
```

### Chain Reorganization Flow
```go
// When receiving a block that doesn't extend our tip:
// 1. Request the peer's chain from the fork point
// 2. Find common ancestor by walking back from both tips
// 3. Calculate total work (or just use height for longest-chain)
// 4. If new chain has more work/height:
//    a. Undo blocks from our tip to common ancestor (UndoBlock for each)
//    b. Apply new blocks from ancestor to new tip (ApplyBlock for each)
//    c. Update chain tip metadata
//    d. Move transactions from orphaned blocks back to mempool
//       (if not already in new chain)
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| HTTP-based P2P | Raw TCP with binary framing | N/A (design choice) | More authentic to Bitcoin, better educational value |
| Protobuf messages | JSON payloads | Project decision Phase 1 | Consistent with existing serialization, more debuggable |
| Automatic peer discovery | Manual peer list via CLI | N/A (ANET-01 deferred to v2) | Simpler implementation, sufficient for localhost demo |

## Open Questions

1. **Maximum message size limit**
   - What we know: Need a cap to prevent memory exhaustion from malicious peers
   - What's unclear: Right threshold -- blocks with many transactions could be large
   - Recommendation: Set to 10MB (generous for educational use), validate before allocating

2. **Peer connection limits**
   - What we know: On localhost, resource contention is minimal
   - What's unclear: Whether to cap max peers
   - Recommendation: Cap at 8 peers for simplicity; sufficient for demo scenarios

3. **Block propagation strategy**
   - What we know: Bitcoin uses headers-first download; simpler to do full blocks
   - What's unclear: Whether headers-first adds enough educational value to justify complexity
   - Recommendation: Full block download (simpler). Headers-first is a v2 optimization.

## Validation Architecture

### Test Framework
| Property | Value |
|----------|-------|
| Framework | Go testing (stdlib) + testify (if added) |
| Config file | None -- Go convention `*_test.go` in package dirs |
| Quick run command | `go test ./internal/domain/p2p/... -v -count=1` |
| Full suite command | `go test ./... -race -count=1` |

### Phase Requirements to Test Map
| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| NET-01 | TCP listener starts on configured port | integration | `go test ./internal/domain/p2p/ -run TestServerListen -v` | No -- Wave 0 |
| NET-02 | Version handshake exchanges height and version | integration | `go test ./internal/domain/p2p/ -run TestHandshake -v` | No -- Wave 0 |
| NET-04 | Transaction broadcast reaches all peers | integration | `go test ./internal/domain/p2p/ -run TestTxBroadcast -v` | No -- Wave 0 |
| NET-05 | Block broadcast reaches all peers | integration | `go test ./internal/domain/p2p/ -run TestBlockBroadcast -v` | No -- Wave 0 |
| NET-06 | Invalid blocks/txs rejected before relay | unit | `go test ./internal/domain/p2p/ -run TestValidation -v` | No -- Wave 0 |
| NET-07 | New node syncs full chain from peer | integration | `go test ./internal/domain/p2p/ -run TestInitialBlockDownload -v` | No -- Wave 0 |
| NET-08 | Node detects longer chain and reorganizes | integration | `go test ./internal/domain/p2p/ -run TestLongerChainReorg -v` | No -- Wave 0 |
| NET-09 | Reorg correctly reverses/reapplies UTXO changes | integration | `go test ./internal/domain/p2p/ -run TestReorgUTXO -v` | No -- Wave 0 |

### Sampling Rate
- **Per task commit:** `go test ./internal/domain/p2p/... -race -v -count=1`
- **Per wave merge:** `go test ./... -race -count=1`
- **Phase gate:** Full suite green before `/gsd:verify-work`

### Wave 0 Gaps
- [ ] `internal/domain/p2p/` -- entire package is new
- [ ] Test helpers for creating in-memory peer pairs (net.Pipe or localhost listener)
- [ ] Test helpers for creating populated chains (reuse from bbolt tests or create shared fixture)

## Sources

### Primary (HIGH confidence)
- Go `net` package documentation -- TCP server/client patterns
- Go `encoding/binary` documentation -- big-endian framing
- Existing codebase analysis -- `utxo.Set.UndoBlock`, `ProofOfWork.Validate`, `Mempool.Add`/`Remove`, `chain.Repository` interface

### Secondary (MEDIUM confidence)
- Bitcoin protocol documentation -- message structure inspiration (version, verack, inv, getdata, block, tx)
- Educational blockchain implementations -- simplified P2P patterns

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH -- all stdlib, no new dependencies, well-understood Go networking
- Architecture: HIGH -- patterns follow established Go concurrency idioms, project conventions consistent
- Pitfalls: HIGH -- common TCP/concurrency pitfalls well-documented, specific to this codebase (bbolt single-writer, UTXO atomicity)

**Research date:** 2026-03-05
**Valid until:** 2026-04-05 (stable -- stdlib-only, no external dependency versions to track)
