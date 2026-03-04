# Architecture Patterns

**Domain:** Educational Bitcoin-like blockchain implementation in Go
**Researched:** 2026-03-05

## Recommended Architecture

Shitcoin should follow a **layered architecture with clean package boundaries**, inspired by btcd's modular design but simplified for educational clarity. Each layer depends only on layers below it, enabling incremental development and testing.

```
                    +------------------+
                    |    Web Dashboard |   (HTTP server, block explorer UI)
                    +--------+---------+
                             |
                    +--------+---------+
                    |    CLI Layer     |   (cobra commands, user interaction)
                    +--------+---------+
                             |
                    +--------+---------+
                    |    RPC / API     |   (JSON-RPC or REST, programmatic access)
                    +--------+---------+
                             |
          +------------------+------------------+
          |                  |                  |
+---------+------+  +--------+-------+  +-------+--------+
|   P2P Network  |  |     Miner      |  |    Mempool     |
|   (node sync)  |  |  (PoW engine)  |  | (pending txs)  |
+--------+-------+  +--------+-------+  +-------+--------+
         |                   |                   |
         +-------------------+-------------------+
                             |
                    +--------+---------+
                    |   Core Domain    |
                    |  (blockchain,    |
                    |   blocks, txs,   |
                    |   UTXO, wallet)  |
                    +--------+---------+
                             |
                    +--------+---------+
                    |  Storage Layer   |
                    |  (BoltDB/bbolt)  |
                    +------------------+
```

### Package Layout

Use Go's multi-package layout for clear boundaries. Based on analysis of btcd, Jeiwan/blockchain_go, TheDhejavu/the-crypto-project, and volodymyrprokopyuk/go-blockchain, the following structure balances educational clarity with proper software engineering:

```
shitcoin/
+-- cmd/
|   +-- shitcoin/
|       +-- main.go              # Entry point
+-- internal/
|   +-- core/
|   |   +-- block.go             # Block struct, hashing, serialization
|   |   +-- blockchain.go        # Chain management, block addition, validation
|   |   +-- iterator.go          # Blockchain traversal
|   |   +-- genesis.go           # Genesis block creation
|   |   +-- merkle.go            # Merkle tree for transaction hashing
|   +-- tx/
|   |   +-- transaction.go       # Transaction struct, creation, signing
|   |   +-- input.go             # TXInput (references to previous outputs)
|   |   +-- output.go            # TXOutput (value + lock script)
|   |   +-- coinbase.go          # Coinbase transaction (mining reward)
|   +-- utxo/
|   |   +-- set.go               # UTXO set: cache of unspent outputs
|   |   +-- finder.go            # Find spendable outputs for an address
|   +-- wallet/
|   |   +-- wallet.go            # Key pair generation (ECDSA)
|   |   +-- address.go           # Address derivation (Base58Check)
|   |   +-- keystore.go          # Wallet persistence (encrypted file)
|   +-- consensus/
|   |   +-- pow.go               # Proof of Work algorithm
|   |   +-- difficulty.go        # Difficulty target and adjustment
|   +-- mempool/
|   |   +-- pool.go              # Pending transaction pool
|   |   +-- validation.go        # Transaction validation before pool entry
|   +-- network/
|   |   +-- node.go              # Node lifecycle, peer management
|   |   +-- server.go            # TCP listener, connection handling
|   |   +-- protocol.go          # Message types (version, inv, getdata, block, tx)
|   |   +-- sync.go              # Chain synchronization logic
|   |   +-- peer.go              # Individual peer connection state
|   +-- storage/
|   |   +-- store.go             # Storage interface
|   |   +-- boltdb.go            # BoltDB implementation
|   |   +-- buckets.go           # Bucket definitions (blocks, utxo, metadata)
|   +-- miner/
|   |   +-- miner.go             # Mining loop (auto-mine + manual modes)
|   |   +-- worker.go            # Block assembly from mempool
+-- cli/
|   +-- cli.go                   # Root CLI setup (cobra)
|   +-- cmd_createchain.go       # Initialize new blockchain
|   +-- cmd_createwallet.go      # Generate new wallet
|   +-- cmd_send.go              # Create and broadcast transaction
|   +-- cmd_balance.go           # Query address balance
|   +-- cmd_mine.go              # Trigger manual mining
|   +-- cmd_startnode.go         # Start P2P node
|   +-- cmd_printchain.go        # Dump chain to stdout
+-- api/
|   +-- server.go                # HTTP/JSON-RPC server
|   +-- handlers.go              # API endpoint handlers
|   +-- routes.go                # Route definitions
+-- web/
|   +-- dashboard.go             # Dashboard HTTP handler
|   +-- templates/               # HTML templates
|   +-- static/                  # CSS/JS assets
+-- pkg/
|   +-- encoding/
|   |   +-- base58.go            # Base58 encoding/decoding
|   |   +-- serialize.go         # Gob serialization helpers
|   +-- crypto/
|       +-- hash.go              # SHA-256, RIPEMD-160 wrappers
|       +-- sign.go              # ECDSA sign/verify helpers
```

**Confidence:** HIGH -- This structure is derived from multiple production and educational Go blockchain implementations.

### Component Boundaries

| Component | Package | Responsibility | Depends On | Depended On By |
|-----------|---------|---------------|------------|----------------|
| Block | `internal/core` | Block struct, hash computation, serialization, genesis creation | `pkg/crypto`, `internal/tx` | Everything above |
| Blockchain | `internal/core` | Chain state, adding blocks, validation rules, longest chain | `internal/storage`, Block | Miner, Network, CLI, API |
| Transaction | `internal/tx` | TX struct, inputs/outputs, signing, coinbase creation | `pkg/crypto`, `pkg/encoding` | UTXO, Mempool, Block |
| UTXO Set | `internal/utxo` | Cache of unspent outputs, balance queries, spendable output lookup | `internal/storage`, `internal/tx` | Wallet (balance), Miner (tx creation) |
| Wallet | `internal/wallet` | Key generation, address derivation, TX signing | `pkg/crypto`, `pkg/encoding` | CLI, API |
| Consensus (PoW) | `internal/consensus` | Nonce search, difficulty validation, target adjustment | `internal/core` (Block) | Miner, Blockchain (validation) |
| Mempool | `internal/mempool` | Pending TX pool, validation before acceptance, TX selection for mining | `internal/tx`, `internal/utxo` | Miner, Network, API |
| Miner | `internal/miner` | Block assembly, mining loop, reward distribution | `internal/consensus`, `internal/mempool`, `internal/core` | CLI (manual mine), Network (broadcast) |
| Network | `internal/network` | Peer connections, message exchange, chain sync | `internal/core`, `internal/mempool`, `internal/tx` | CLI (start node) |
| Storage | `internal/storage` | Persistence interface, BoltDB implementation | None (leaf dependency) | Blockchain, UTXO Set |
| CLI | `cli/` | User commands, argument parsing | All internal packages | `cmd/shitcoin` (main) |
| API | `api/` | JSON-RPC/REST endpoints | All internal packages | Web Dashboard |
| Dashboard | `web/` | Block explorer UI, node status visualization | `api/` | End users (browser) |

### Data Flow

#### Transaction Lifecycle

```
User (CLI/API)
  |
  v
1. Wallet creates TX:
   - Find spendable UTXOs for sender address  [UTXO Set]
   - Create TXInputs referencing those UTXOs   [TX]
   - Create TXOutputs (recipient + change)     [TX]
   - Sign TX with sender's private key         [Wallet]
  |
  v
2. TX enters Mempool:
   - Validate TX structure                     [Mempool]
   - Verify signature                          [Mempool]
   - Check inputs reference valid UTXOs        [Mempool -> UTXO Set]
   - Check no double-spend within mempool      [Mempool]
   - Add to pending pool                       [Mempool]
  |
  v
3. TX broadcasts to peers:                     [Network]
   - Send "tx" message to connected peers
   - Peers validate and add to their mempools
  |
  v
4. Miner assembles block:
   - Select TXs from mempool                   [Miner -> Mempool]
   - Create coinbase TX (mining reward)        [Miner -> TX]
   - Build Merkle tree from TX hashes          [Core]
   - Run Proof of Work (find valid nonce)      [Consensus]
  |
  v
5. Block added to chain:
   - Validate block (PoW, TX validity)         [Blockchain]
   - Persist block to storage                  [Storage]
   - Update UTXO set (remove spent, add new)   [UTXO Set]
   - Remove mined TXs from mempool             [Mempool]
  |
  v
6. Block broadcasts to peers:                  [Network]
   - Send "block" message to connected peers
   - Peers validate and add to their chains
```

#### Chain Synchronization (New Node Joining)

```
New Node                          Existing Node
   |                                    |
   |--- version (my height: 0) ------->|
   |                                    |
   |<-- version (my height: 150) ------|
   |                                    |
   |--- getblocks ---------------------->|
   |                                    |
   |<-- inv (block hashes) ------------|
   |                                    |
   |--- getdata (hash1) --------------->|
   |<-- block (block1) ----------------|
   |                                    |
   |--- getdata (hash2) --------------->|
   |<-- block (block2) ----------------|
   |    ...                             |
   |                                    |
   [Reindex UTXO set after sync]
```

#### Balance Query

```
User (CLI: "balance <address>")
  |
  v
UTXO Set: find all unspent outputs locked to <address>
  |
  v
Sum output values -> return balance
```

## Patterns to Follow

### Pattern 1: Storage Interface Abstraction

Decouple storage from business logic so the KV store can be swapped without touching core code. This is how btcd separates its database backends.

**When:** Always. Define a storage interface early.

```go
// internal/storage/store.go
type Store interface {
    GetBlock(hash []byte) (*core.Block, error)
    PutBlock(block *core.Block) error
    GetLastHash() ([]byte, error)
    SetLastHash(hash []byte) error
    GetUTXOs(address string) ([]tx.TXOutput, error)
    PutUTXOs(address string, outputs []tx.TXOutput) error
    Close() error
}
```

**Why:** Testability (use in-memory store for tests), flexibility (swap BoltDB for BadgerDB later), separation of concerns.

### Pattern 2: Message-Based P2P Protocol

Use a simple custom protocol over TCP with typed messages, following Jeiwan's approach. Each message has a command header (12 bytes) followed by gob-encoded payload.

**When:** Building the network layer. Avoid libp2p for an educational project -- it hides too much of the networking logic you want to learn.

```go
// internal/network/protocol.go
const (
    CmdVersion  = "version"
    CmdInv      = "inv"
    CmdGetData  = "getdata"
    CmdGetBlocks = "getblocks"
    CmdBlock    = "block"
    CmdTx       = "tx"
    CmdAddr     = "addr"
)

type Message struct {
    Command string
    Payload []byte
}

type VersionMsg struct {
    Version    int
    BestHeight int
    AddrFrom   string
}
```

**Why:** Educational value. Understanding the message protocol is a core learning goal. libp2p is excellent for production but obscures the mechanics.

### Pattern 3: Goroutine-Based Mining with Cancellation

Use Go's concurrency primitives (goroutines, channels, context) for the mining loop so it can be interrupted when a new block arrives from the network.

**When:** Building the miner component.

```go
// internal/miner/miner.go
func (m *Miner) Start(ctx context.Context) {
    for {
        select {
        case <-ctx.Done():
            return
        default:
            block := m.assembleBlock()
            if block != nil {
                m.mineBlock(ctx, block)
            }
        }
    }
}

func (m *Miner) mineBlock(ctx context.Context, block *core.Block) {
    pow := consensus.NewProofOfWork(block)
    nonce, hash, err := pow.Run(ctx) // ctx allows cancellation
    if err != nil {
        return // cancelled, new block arrived
    }
    block.Nonce = nonce
    block.Hash = hash
    m.onBlockMined(block)
}
```

**Why:** In a real blockchain, miners must stop mining when they receive a valid new block from the network. Context cancellation models this cleanly.

### Pattern 4: Event-Driven Component Communication

Use Go channels to decouple components. When a block is mined or received, emit events that other components react to.

**When:** Connecting miner, mempool, UTXO set, and network.

```go
type EventBus struct {
    blockMined    chan *core.Block
    blockReceived chan *core.Block
    txReceived    chan *tx.Transaction
}
```

**Why:** Avoids circular dependencies between network, miner, and mempool. Each component subscribes to events it cares about.

### Pattern 5: BoltDB Bucket Organization

Use BoltDB's bucket concept to organize different data types within a single database file.

**When:** Implementing storage layer.

```go
var (
    blocksBucket   = []byte("blocks")    // hash -> serialized block
    utxoBucket     = []byte("utxo")      // txid -> serialized outputs
    metadataBucket = []byte("metadata")  // "lastHash" -> last block hash
                                         // "height" -> chain height
)
```

**Why:** BoltDB buckets act like tables, keeping block data, UTXO cache, and metadata cleanly separated within one DB file.

## Anti-Patterns to Avoid

### Anti-Pattern 1: Single-Package Monolith

**What:** Putting all code in the `main` package (as Jeiwan's implementation does).

**Why bad:** Files become tightly coupled, testing requires building everything, and it becomes hard to reason about dependencies. While acceptable for a tutorial, it hinders learning about Go package design.

**Instead:** Use the multi-package layout described above. Each package has a clear API surface. This also teaches Go package design patterns.

### Anti-Pattern 2: Global State for Blockchain Instance

**What:** Using package-level variables to hold the blockchain, UTXO set, or mempool.

**Why bad:** Makes testing impossible (tests leak state), prevents running multiple nodes in the same process for testing, and creates hidden dependencies.

**Instead:** Use dependency injection. Pass the blockchain, UTXO set, and mempool as constructor parameters to components that need them.

### Anti-Pattern 3: Synchronous P2P Communication

**What:** Blocking the main goroutine while waiting for peer responses.

**Why bad:** One slow or malicious peer blocks the entire node.

**Instead:** Handle each peer connection in its own goroutine. Use channels or sync primitives for coordination.

### Anti-Pattern 4: Scanning Full Chain for Balance Queries

**What:** Iterating through every block and transaction to find unspent outputs.

**Why bad:** O(n) where n is total transactions in the chain. Gets slow quickly even for educational purposes.

**Instead:** Maintain a UTXO set that is updated incrementally when blocks are added. This is how Bitcoin Core works and is a key concept to learn.

### Anti-Pattern 5: Mixing Consensus Validation with Block Storage

**What:** Validating PoW inside the storage layer or blockchain.AddBlock().

**Why bad:** Conflates two distinct concerns. When receiving a block from the network you validate before storing; when mining you've already done the PoW.

**Instead:** Validate in the caller (network sync validates received blocks; miner produces valid blocks by construction). Blockchain.AddBlock() should only check structural validity.

## Suggested Build Order

Build order is dictated by dependency relationships. Each phase produces a working, testable artifact.

```
Phase 1: Foundation
  [Block] -> [Blockchain (in-memory)] -> [Genesis]
  Testable: Create genesis, add blocks, traverse chain

Phase 2: Persistence
  [Storage Interface] -> [BoltDB Implementation]
  [Blockchain uses Storage instead of in-memory slice]
  Testable: Restart process, chain persists

Phase 3: Proof of Work
  [PoW Algorithm] -> [Difficulty Target]
  [Block creation now requires mining]
  Testable: Mine blocks, verify PoW, adjust difficulty

Phase 4: Transactions
  [TXOutput, TXInput] -> [Transaction] -> [Coinbase TX]
  [Blocks now contain transactions instead of arbitrary data]
  Testable: Create coinbase TX, include in mined block

Phase 5: Wallet & Addresses
  [ECDSA Key Generation] -> [Address Derivation] -> [TX Signing]
  Testable: Generate wallet, derive address, sign TX

Phase 6: UTXO Model
  [UTXO Set] -> [Spendable Output Finder] -> [Balance Query]
  [Full transaction flow: find UTXOs -> create TX -> mine -> update UTXOs]
  Testable: Send coins, check balances, verify UTXO updates

Phase 7: Mempool
  [Transaction Pool] -> [TX Validation] -> [TX Selection for Mining]
  Testable: Submit TX, validate, mine from pool

Phase 8: CLI
  [Cobra commands wrapping all above functionality]
  Testable: Full workflow via command line

Phase 9: P2P Networking
  [TCP Server] -> [Message Protocol] -> [Peer Management]
  [Chain Sync] -> [TX Relay] -> [Block Relay]
  Testable: Run 2-3 nodes, sync chains, relay transactions

Phase 10: Merkle Trees
  [Merkle Tree] -> [Block header includes Merkle root]
  Can be added at any point after Phase 4 but fits here
  Testable: Verify transaction inclusion via Merkle proof

Phase 11: API Layer
  [HTTP/JSON-RPC Server] -> [Endpoint Handlers]
  Testable: curl commands for all operations

Phase 12: Web Dashboard
  [Block Explorer] -> [Node Status] -> [Mining Controls]
  Testable: Visual verification in browser
```

**Build order rationale:**
- Phases 1-3 establish the core blockchain with real mining (no shortcuts).
- Phases 4-6 add the transaction model, which is the most complex and educational part.
- Phase 7 bridges transactions and mining with the mempool.
- Phase 8 provides a user interface before networking complexity.
- Phase 9 is the hardest phase and benefits from having all local functionality solid first.
- Phases 10-12 are enhancements that layer on top of working infrastructure.

## Scalability Considerations

This is an educational project, but understanding scalability teaches important concepts.

| Concern | At 100 blocks | At 10K blocks | At 100K blocks |
|---------|---------------|---------------|----------------|
| Chain storage | Trivial (~few MB) | Moderate (~100s MB) | BoltDB handles fine for educational use |
| UTXO set scan | No issue with cached set | UTXO set cache essential | UTXO set with periodic flush |
| Block sync | Near instant | Seconds | Minutes (acceptable for local) |
| Mempool size | Trivial | Cap at ~1000 TXs | Cap and eviction policy |
| PoW difficulty | Low (instant mine) | Adjustable | Target 10-30 second blocks for demos |

**Key insight:** For an educational blockchain on localhost, the bottleneck is never raw performance. The architecture should optimize for **clarity and debuggability** over throughput.

## Technology Choices (Architecture-Driven)

| Component | Choice | Rationale |
|-----------|--------|-----------|
| P2P | Raw TCP with custom protocol | Educational value; libp2p hides networking mechanics |
| Storage | bbolt (maintained BoltDB fork) | Simple, embedded, bucket concept maps well to blockchain data; BoltDB is archived but bbolt is actively maintained by etcd team |
| Serialization | encoding/gob | Go-native, simple, sufficient for localhost-only communication |
| Hashing | crypto/sha256 (stdlib) | Standard, matches Bitcoin's approach |
| Signing | crypto/ecdsa (stdlib) | No external deps, sufficient for educational ECDSA |
| CLI | spf13/cobra | De facto standard for Go CLIs, excellent DX |
| Web | net/http (stdlib) + html/template | No framework needed for a simple dashboard |
| Logging | log/slog (stdlib) | Structured logging, built into Go 1.21+ |

## Sources

- [Jeiwan - Building Blockchain in Go (7-part series)](https://jeiwan.net/posts/building-blockchain-in-go-part-1/) - Canonical educational Go blockchain, HIGH confidence
- [Jeiwan/blockchain_go GitHub](https://github.com/Jeiwan/blockchain_go) - Reference implementation, HIGH confidence
- [TheODDYSEY/Blockchain-Go](https://github.com/TheODDYSEY/Blockchain-Go) - Progressive build structure, MEDIUM confidence
- [volodymyrprokopyuk/go-blockchain](https://github.com/volodymyrprokopyuk/go-blockchain) - Clean package architecture with gRPC, MEDIUM confidence
- [TheDhejavu/the-crypto-project](https://github.com/TheDhejavu/the-crypto-project) - Multi-package layout with libp2p, MEDIUM confidence
- [btcsuite/btcd](https://github.com/btcsuite/btcd) - Production Go Bitcoin implementation, HIGH confidence for architecture patterns
- [Jeiwan - Transactions Part 1](https://jeiwan.net/posts/building-blockchain-in-go-part-4/) - UTXO model architecture, HIGH confidence
- [Jeiwan - Network](https://jeiwan.net/posts/building-blockchain-in-go-part-7/) - P2P protocol design, HIGH confidence
- [BoltDB vs Badger Comparison](https://tech.townsourced.com/post/boltdb-vs-badger/) - Storage engine comparison, MEDIUM confidence
- [freeCodeCamp - Build a Blockchain from Scratch with Go](https://www.freecodecamp.org/news/build-a-blockchain-in-golang-from-scratch/) - Build order reference, MEDIUM confidence
- [btcsuite/btcd mempool](https://github.com/btcsuite/btcd/blob/master/mempool/mempool.go) - Mempool architecture reference, HIGH confidence
