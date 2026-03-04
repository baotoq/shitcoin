# Phase 1: Core Chain Foundation - Research

**Researched:** 2026-03-05
**Domain:** Blockchain primitives -- block structure, SHA-256d hashing, Proof-of-Work mining, difficulty adjustment, persistent storage
**Confidence:** HIGH

## Summary

Phase 1 builds the foundational blockchain data structures and mining loop. The core problem is well-understood: define a Block struct with a header (prev hash, timestamp, difficulty, nonce) and body, hash it deterministically with SHA-256d (double SHA-256), run a Proof-of-Work loop that increments a nonce until the hash is below a target, persist blocks to disk with bbolt, and adjust difficulty every N blocks based on actual vs expected block time.

The technology stack is locked by user decisions: Go 1.23+, go-zero framework for project structure and configuration, bbolt (go.etcd.io/bbolt v1.4.x) for storage, JSON for serialization, and Tactical DDD for code organization. All of these are mature, well-documented technologies. Go's standard `encoding/json` provides deterministic struct serialization (fields appear in declaration order), and `crypto/sha256` is in the standard library.

**Primary recommendation:** Build domain types (Block, Chain, ProofOfWork) as pure Go structs with behavior methods, use go-zero's conf.MustLoad for YAML-based consensus parameter configuration, and wrap bbolt in a repository interface so domain logic stays storage-agnostic. Use `math/big.Int` for target/difficulty arithmetic, following Bitcoin's simplified approach from educational blockchain implementations.

<user_constraints>

## User Constraints (from CONTEXT.md)

### Locked Decisions
- BoltDB (bbolt) for persistent chain storage -- single-file B+tree, battle-tested (used in etcd)
- JSON for block encoding (hashing and storage) -- prioritizes debuggability over compactness; deterministic serialization must be enforced (sorted keys, no floating point ambiguity)
- go-zero framework for project structure and configuration -- consensus parameters via YAML config + struct binding
- go-zero project structure conventions (handler -> logic -> model layers)
- Tactical DDD approach: Entities, Value Objects, Aggregates, Repositories, Domain Services
- Domain logic stays clean and separate from framework plumbing
- Block and Chain as domain entities, not tied to storage or transport
- Tilt.dev for local dev environment orchestration
- Configurable without code changes: block time target, difficulty adjustment interval, initial difficulty

### Claude's Discretion
- BoltDB bucket layout and key design
- Block struct field ordering
- Difficulty adjustment algorithm specifics (window size, clamping bounds)
- Genesis block default embedded message
- Error handling patterns
- Package naming within go-zero + DDD structure

### Deferred Ideas (OUT OF SCOPE)
None -- discussion stayed within phase scope

</user_constraints>

<phase_requirements>

## Phase Requirements

| ID | Description | Research Support |
|----|-------------|-----------------|
| MINE-01 | System creates a genesis block with a configurable embedded message on chain initialization | Genesis block is a special Block with PrevHash=0x00..., height=0, and a message field. Configurable via go-zero YAML config. |
| MINE-02 | Block structure contains header (prev hash, Merkle root, timestamp, difficulty target, nonce) and body (transaction list) | Block struct with Header and Body sub-structs. Merkle root will be a placeholder (zero hash) in Phase 1 since transactions come in Phase 2. Transaction list will be empty or contain a placeholder. |
| MINE-03 | Block headers are hashed using SHA-256 double-hash with deterministic canonical serialization | Use Go's `encoding/json` for deterministic struct serialization (field declaration order is guaranteed). SHA-256d = sha256(sha256(data)). Use `crypto/sha256` from stdlib. |
| MINE-06 | Difficulty adjusts automatically every N blocks based on actual vs target block time (window-based, clamped) | Bitcoin-style: every N blocks compare actual elapsed time to expected time, multiply difficulty by (expected/actual), clamp adjustment factor to [0.25, 4.0]. Use `math/big.Int` for target arithmetic. |
| MINE-09 | Consensus parameters (block time target, difficulty interval, initial reward, halving interval) are configurable | go-zero conf.MustLoad with YAML config file. Define ConsensusConfig struct with json tags including defaults. Parameters: BlockTimeTarget, DifficultyAdjustInterval, InitialDifficulty. |

</phase_requirements>

## Standard Stack

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| Go | 1.23.4 | Language runtime | Installed on dev machine, current stable |
| go-zero | latest (1.7+) | Framework, config loading, project structure | User decision; provides conf.MustLoad, logx, project conventions |
| bbolt | v1.4.x (go.etcd.io/bbolt) | Persistent key-value storage for chain data | User decision; single-file, ACID, battle-tested in etcd |
| crypto/sha256 | stdlib | SHA-256 hashing | Go standard library, no external deps needed |
| encoding/json | stdlib | Deterministic JSON serialization | Go standard library, struct field order is guaranteed deterministic |
| math/big | stdlib | Big integer arithmetic for difficulty target | Go standard library, needed for 256-bit target comparisons |

### Supporting
| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| goctl | 1.9.2 | Code generation for go-zero project scaffolding | Initial project setup, generating handler/logic/types |
| Tilt.dev | latest | Local dev environment orchestration | Running the node during development |
| encoding/hex | stdlib | Hex encoding for hash display | Displaying block hashes to humans |
| encoding/binary | stdlib | Binary encoding for nonce in hash preparation | Preparing data for hashing |
| bytes | stdlib | Buffer operations for hash input assembly | Concatenating header fields for hashing |
| fmt | stdlib | Error wrapping with %w | Error handling throughout |
| time | stdlib | Timestamps and duration calculations | Block timestamps, difficulty adjustment timing |

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| bbolt | BadgerDB | BadgerDB is LSM-tree (better write throughput) but more complex; bbolt is B+tree (simpler, better reads) -- locked to bbolt |
| encoding/json | CBOR/Protobuf | More compact but less debuggable -- locked to JSON for educational transparency |
| go-zero | plain net/http | Less boilerplate but no config system, no project conventions -- locked to go-zero |

**Installation:**
```bash
go mod init github.com/baotoq/shitcoin
go get github.com/zeromicro/go-zero@latest
go get go.etcd.io/bbolt@latest
```

## Architecture Patterns

### Recommended Project Structure

This merges go-zero conventions with Tactical DDD. The go-zero handler->logic->svc layers serve as the application shell, while domain logic lives in `internal/domain/` with zero framework dependencies.

```
shitcoin/
├── etc/
│   └── shitcoin.yaml           # go-zero config (includes consensus params)
├── internal/
│   ├── config/
│   │   └── config.go           # go-zero Config struct (embeds rest.RestConf + ConsensusConfig)
│   ├── svc/
│   │   └── service_context.go  # Dependency injection: repos, domain services
│   ├── handler/                # go-zero HTTP handlers (Phase 1: minimal or none)
│   ├── logic/                  # go-zero logic layer (orchestrates domain operations)
│   ├── domain/
│   │   ├── block/
│   │   │   ├── block.go        # Block entity (aggregate root)
│   │   │   ├── header.go       # BlockHeader value object
│   │   │   ├── hash.go         # Hash value object (32-byte SHA-256d)
│   │   │   ├── pow.go          # ProofOfWork domain service
│   │   │   ├── difficulty.go   # Difficulty adjustment logic
│   │   │   └── errors.go       # Domain errors
│   │   └── chain/
│   │       ├── chain.go        # Chain aggregate (manages block sequence)
│   │       ├── repository.go   # ChainRepository interface
│   │       └── errors.go       # Domain errors
│   └── infrastructure/
│       └── persistence/
│           └── bbolt/
│               └── chain_repo.go  # bbolt implementation of ChainRepository
├── cmd/
│   └── shitcoin/
│       └── main.go             # Entry point
├── Tiltfile                    # Tilt.dev local dev config
├── Dockerfile                  # For Tilt
├── go.mod
└── go.sum
```

**Dependency rule:** `internal/domain/` has ZERO imports from `internal/infrastructure/`, `internal/svc/`, or any go-zero packages. Dependencies point inward.

### Pattern 1: Block as DDD Entity with Value Object Header

**What:** Block is an entity (identified by hash), BlockHeader is a value object (immutable once created), Hash is a value object (32 bytes).
**When to use:** Always -- this is the core data structure.

```go
// internal/domain/block/hash.go
type Hash [32]byte

func (h Hash) String() string {
    return hex.EncodeToString(h[:])
}

func (h Hash) IsZero() bool {
    return h == Hash{}
}

// internal/domain/block/header.go
type Header struct {
    version       uint32
    prevBlockHash Hash
    merkleRoot    Hash      // zero in Phase 1, populated in Phase 3
    timestamp     time.Time
    bits          uint32    // compact difficulty target
    nonce         uint32
}

// internal/domain/block/block.go
type Block struct {
    header       Header
    transactions [][]byte  // empty in Phase 1, typed transactions in Phase 2
    hash         Hash      // computed from header via SHA-256d
    height       uint64
    message      string    // only used for genesis block
}

func NewGenesisBlock(message string, bits uint32) (*Block, error) {
    b := &Block{
        header: Header{
            version:       1,
            prevBlockHash: Hash{},
            merkleRoot:    Hash{},
            timestamp:     time.Now(),
            bits:          bits,
        },
        height:  0,
        message: message,
    }
    // Mine the genesis block
    return b, nil
}
```

### Pattern 2: Repository Interface in Domain, bbolt Implementation in Infrastructure

**What:** ChainRepository interface defined alongside the Chain aggregate; bbolt implementation is a separate package.
**When to use:** All storage operations.

```go
// internal/domain/chain/repository.go
type Repository interface {
    GetBlock(ctx context.Context, hash block.Hash) (*block.Block, error)
    GetBlockByHeight(ctx context.Context, height uint64) (*block.Block, error)
    SaveBlock(ctx context.Context, b *block.Block) error
    GetLatestBlock(ctx context.Context) (*block.Block, error)
    GetChainHeight(ctx context.Context) (uint64, error)
    GetBlocksInRange(ctx context.Context, startHeight, endHeight uint64) ([]*block.Block, error)
}
```

### Pattern 3: ProofOfWork as Domain Service

**What:** Stateless PoW service that takes a block and mines it (finds valid nonce).
**When to use:** During block creation.

```go
// internal/domain/block/pow.go
type ProofOfWork struct{}

func (pow *ProofOfWork) Mine(b *Block) error {
    target := bitsToTarget(b.header.bits)
    var nonce uint32
    for nonce < math.MaxUint32 {
        hash := computeHash(b.header, nonce)
        hashInt := new(big.Int).SetBytes(hash[:])
        if hashInt.Cmp(target) == -1 {
            b.header.nonce = nonce
            b.hash = hash
            return nil
        }
        nonce++
    }
    return ErrNonceExhausted
}
```

### Pattern 4: go-zero Configuration for Consensus Parameters

**What:** Consensus parameters in YAML config, loaded via conf.MustLoad.
**When to use:** Application startup.

```go
// internal/config/config.go
type Config struct {
    rest.RestConf                         // go-zero base config (Name, Host, Port, Log, etc.)
    Consensus     ConsensusConfig         // blockchain consensus parameters
    Storage       StorageConfig           // bbolt storage settings
}

type ConsensusConfig struct {
    BlockTimeTarget          int    `json:",default=10"`    // target seconds between blocks
    DifficultyAdjustInterval int    `json:",default=10"`    // adjust every N blocks
    InitialDifficulty        int    `json:",default=16"`    // initial target bits
    GenesisMessage           string `json:",default=The Times 03/Jan/2009 Chancellor on brink of second bailout for banks"`
}

type StorageConfig struct {
    DBPath string `json:",default=data/shitcoin.db"`
}
```

```yaml
# etc/shitcoin.yaml
Name: shitcoin
Host: 0.0.0.0
Port: 8080

Consensus:
  BlockTimeTarget: 10
  DifficultyAdjustInterval: 10
  InitialDifficulty: 16
  GenesisMessage: "Hello, Shitcoin!"

Storage:
  DBPath: data/shitcoin.db
```

### Anti-Patterns to Avoid
- **Importing bbolt in domain packages:** Domain must not know about storage technology. Use repository interface.
- **Mutable Hash values:** Hash should be a value object. Never modify a Hash after creation.
- **Using `map[string]interface{}` for block serialization:** Use typed structs. Maps lose field ordering guarantees and invite bugs.
- **Storing difficulty as float64:** Floating point is non-deterministic across platforms. Use integer-based target bits (uint32) and `big.Int` for target calculations.
- **Global mutable state:** No package-level variables for chain state. Use dependency injection via go-zero ServiceContext.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| SHA-256 hashing | Custom hash function | `crypto/sha256` (stdlib) | Audited, hardware-accelerated, correct |
| Key-value storage | File-based storage with manual locking | bbolt (`go.etcd.io/bbolt`) | ACID transactions, crash recovery, memory-mapped I/O |
| Configuration loading | Manual YAML parsing + env var reading | go-zero `conf.MustLoad` | Handles defaults, validation, env vars, type coercion |
| Big integer math | Custom 256-bit arithmetic | `math/big.Int` (stdlib) | Correct overflow handling, comparison operations |
| Project scaffolding | Manual directory creation | `goctl api new` | Generates consistent go-zero project structure |
| JSON serialization | Custom byte-by-byte serializer | `encoding/json` (stdlib) | Deterministic for structs (declaration order), handles escaping |

**Key insight:** The blockchain domain logic (PoW loop, difficulty adjustment, block validation) IS the custom code. Everything else -- hashing, storage, config, serialization -- should use battle-tested libraries.

## Common Pitfalls

### Pitfall 1: Non-Deterministic Hashing
**What goes wrong:** Different runs produce different hashes for the same block because serialization is not deterministic.
**Why it happens:** Using `map[string]interface{}` or relying on Go map iteration order (which is intentionally randomized).
**How to avoid:** Always serialize typed structs (field order = declaration order). Never use maps in the hashing path. Define a canonical `hashPayload()` method that assembles bytes in a fixed order.
**Warning signs:** Block validation fails after restart. Hashes don't match between nodes.

### Pitfall 2: Timestamp Precision Issues
**What goes wrong:** Timestamps stored with nanosecond precision cause hash mismatches when loaded from storage.
**Why it happens:** JSON marshals time.Time with nanosecond precision by default. Reloading may lose precision depending on storage format.
**How to avoid:** Store timestamps as Unix seconds (int64), not time.Time. Convert to/from time.Time at the boundary.
**Warning signs:** Blocks validate when first created but fail validation after deserialization.

### Pitfall 3: Difficulty Target vs Bits Confusion
**What goes wrong:** Mixing up the compact "bits" representation with the full 256-bit target, leading to incorrect difficulty comparisons.
**Why it happens:** Bitcoin uses a compact 4-byte "bits" encoding for the 256-bit target. Educational implementations often simplify this to "number of leading zero bits" which is less flexible.
**How to avoid:** Choose ONE approach and be consistent. For this educational project, using "target bits" (number of leading zero bits) is simpler. Store as uint32 in the header. Convert to full target with `big.Int.Lsh(big.NewInt(1), 256-targetBits)`.
**Warning signs:** Difficulty adjustment produces unexpected jumps. Very easy or very hard mining.

### Pitfall 4: bbolt Transaction Lifetime and Byte Slice Validity
**What goes wrong:** Reading a value from bbolt and using it after the transaction closes, getting corrupted or zero data.
**Why it happens:** bbolt values returned by `Get()` are only valid for the lifetime of the transaction. The underlying memory is memory-mapped and may be reclaimed.
**How to avoid:** Always copy byte slices within the transaction callback. Use `copy()` or deserialize immediately inside the `View`/`Update` callback.
**Warning signs:** Intermittent data corruption. Values that "disappear" or change after being read.

### Pitfall 5: Genesis Block Re-Creation on Restart
**What goes wrong:** Node creates a new genesis block on every startup, overwriting the existing chain.
**Why it happens:** Startup logic doesn't check if a chain already exists in storage.
**How to avoid:** On startup: check if chain exists in bbolt. If yes, load it. If no, create genesis block and persist it.
**Warning signs:** Chain height always 1 after restart. Previously mined blocks gone.

### Pitfall 6: Nonce Overflow Without Detection
**What goes wrong:** Nonce wraps around to 0 without finding a valid hash, entering an infinite loop.
**Why it happens:** Using `int` or unbounded loop without checking for exhaustion.
**How to avoid:** Use `uint32` for nonce (matching Bitcoin). Check `nonce < math.MaxUint32` in loop. If exhausted, update timestamp and retry (or return error).
**Warning signs:** Mining hangs indefinitely on high difficulty.

### Pitfall 7: go-zero Config Tag Using `json` Not `yaml`
**What goes wrong:** YAML config fields are not loaded, all values are defaults.
**Why it happens:** go-zero uses `json` struct tags for ALL config formats (yaml, json, toml), not `yaml` tags. This is counter-intuitive.
**How to avoid:** Always use `json:"fieldName"` tags in config structs, even though the config file is YAML. The go-zero conf package normalizes all formats through JSON tags.
**Warning signs:** Config values are always defaults despite being set in YAML file.

## Code Examples

### SHA-256 Double Hash (SHA-256d)

```go
// Source: Go stdlib crypto/sha256 + Bitcoin protocol convention
import "crypto/sha256"

func doubleSHA256(data []byte) [32]byte {
    first := sha256.Sum256(data)
    return sha256.Sum256(first[:])
}
```

### Block Header Hash Payload Assembly

```go
// Deterministic byte assembly for hashing -- NO JSON in the hot path
import (
    "bytes"
    "encoding/binary"
)

func (h *Header) hashPayload() []byte {
    var buf bytes.Buffer

    // Fixed order, fixed encoding -- deterministic
    binary.Write(&buf, binary.LittleEndian, h.version)
    buf.Write(h.prevBlockHash[:])
    buf.Write(h.merkleRoot[:])
    binary.Write(&buf, binary.LittleEndian, h.timestamp.Unix())
    binary.Write(&buf, binary.LittleEndian, h.bits)
    binary.Write(&buf, binary.LittleEndian, h.nonce)

    return buf.Bytes()
}

func (h *Header) Hash() Hash {
    payload := h.hashPayload()
    return Hash(doubleSHA256(payload))
}
```

**Note on hashing vs storage serialization:** The user decision says "JSON for block encoding (hashing and storage)." There are two valid interpretations:
1. Use JSON for storage/display but binary for hashing (more Bitcoin-like, more efficient)
2. Use JSON for both hashing and storage (simpler, more debuggable)

Option 2 aligns better with the educational/debuggability goal. If using JSON for hashing:

```go
// JSON-based deterministic hashing (alternative approach)
import "encoding/json"

type hashableHeader struct {
    Version       uint32 `json:"version"`
    PrevBlockHash string `json:"prev_block_hash"`
    MerkleRoot    string `json:"merkle_root"`
    Timestamp     int64  `json:"timestamp"`
    Bits          uint32 `json:"bits"`
    Nonce         uint32 `json:"nonce"`
}

func (h *Header) Hash() Hash {
    hh := hashableHeader{
        Version:       h.version,
        PrevBlockHash: h.prevBlockHash.String(),
        MerkleRoot:    h.merkleRoot.String(),
        Timestamp:     h.timestamp.Unix(),
        Bits:          h.bits,
        Nonce:         h.nonce,
    }
    data, _ := json.Marshal(hh) // struct field order is deterministic
    return Hash(doubleSHA256(data))
}
```

### Proof of Work Mining Loop

```go
// Source: Bitcoin PoW concept + Jeiwan "Building Blockchain in Go"
import (
    "math"
    "math/big"
)

func bitsToTarget(bits uint32) *big.Int {
    target := big.NewInt(1)
    target.Lsh(target, uint(256-bits))
    return target
}

func (pow *ProofOfWork) Mine(b *Block) error {
    target := bitsToTarget(b.header.bits)
    var nonce uint32

    for nonce < math.MaxUint32 {
        b.header.nonce = nonce
        hash := b.header.Hash()
        hashInt := new(big.Int).SetBytes(hash[:])

        if hashInt.Cmp(target) == -1 {
            b.hash = hash
            return nil
        }
        nonce++
    }
    return ErrNonceExhausted
}

func (pow *ProofOfWork) Validate(b *Block) bool {
    target := bitsToTarget(b.header.bits)
    hash := b.header.Hash()
    hashInt := new(big.Int).SetBytes(hash[:])
    return hashInt.Cmp(target) == -1
}
```

### Difficulty Adjustment (Bitcoin-style, simplified)

```go
// Source: Bitcoin protocol, btcsuite/btcd blockchain/difficulty.go concepts
func adjustDifficulty(
    currentBits uint32,
    actualTimeSpan time.Duration,
    targetTimeSpan time.Duration,
) uint32 {
    actual := actualTimeSpan.Seconds()
    target := targetTimeSpan.Seconds()

    // Clamp adjustment factor to [0.25, 4.0]
    ratio := actual / target
    if ratio < 0.25 {
        ratio = 0.25
    }
    if ratio > 4.0 {
        ratio = 4.0
    }

    // If blocks came faster than target, increase difficulty (more bits)
    // If blocks came slower than target, decrease difficulty (fewer bits)
    // ratio < 1 means blocks came too fast -> increase difficulty
    // ratio > 1 means blocks came too slow -> decrease difficulty
    //
    // In "target bits" representation:
    //   more bits = smaller target = harder
    //   fewer bits = larger target = easier
    //
    // adjustment = -log2(ratio) but we simplify for educational purposes
    newBits := float64(currentBits) / ratio

    // Clamp to reasonable range
    if newBits < 1 {
        newBits = 1
    }
    if newBits > 255 {
        newBits = 255
    }

    return uint32(math.Round(newBits))
}
```

**Note:** This simplified "target bits" approach differs from Bitcoin's compact "nBits" encoding. Bitcoin's approach uses a 4-byte compact representation (coefficient + exponent). The simplified approach (number of leading zero bits) is appropriate for an educational project and avoids the complexity of BigCompact encoding. The planner should decide which approach to use -- the simplified version is recommended for Phase 1.

### bbolt Storage Operations

```go
// Source: go.etcd.io/bbolt documentation
import bolt "go.etcd.io/bbolt"

var (
    blocksBucket   = []byte("blocks")
    chainMetaBucket = []byte("chain_meta")
    latestHashKey  = []byte("latest_hash")
    heightKey      = []byte("height")
)

func (r *BboltChainRepo) SaveBlock(ctx context.Context, b *block.Block) error {
    return r.db.Update(func(tx *bolt.Tx) error {
        bkt := tx.Bucket(blocksBucket)

        // Serialize block to JSON
        data, err := json.Marshal(b.ToStorageModel())
        if err != nil {
            return fmt.Errorf("marshal block: %w", err)
        }

        // Store block by hash
        if err := bkt.Put(b.Hash().Bytes(), data); err != nil {
            return fmt.Errorf("put block: %w", err)
        }

        // Also store hash-by-height index for fast lookup
        heightBytes := make([]byte, 8)
        binary.BigEndian.PutUint64(heightBytes, b.Height())
        if err := bkt.Put(heightBytes, b.Hash().Bytes()); err != nil {
            return fmt.Errorf("put height index: %w", err)
        }

        // Update chain metadata
        meta := tx.Bucket(chainMetaBucket)
        if err := meta.Put(latestHashKey, b.Hash().Bytes()); err != nil {
            return fmt.Errorf("put latest hash: %w", err)
        }

        return nil
    })
}

func (r *BboltChainRepo) GetBlock(ctx context.Context, hash block.Hash) (*block.Block, error) {
    var b *block.Block
    err := r.db.View(func(tx *bolt.Tx) error {
        bkt := tx.Bucket(blocksBucket)
        data := bkt.Get(hash.Bytes())
        if data == nil {
            return chain.ErrBlockNotFound
        }

        // CRITICAL: Copy data before transaction ends
        dataCopy := make([]byte, len(data))
        copy(dataCopy, data)

        var sm StorageModel
        if err := json.Unmarshal(dataCopy, &sm); err != nil {
            return fmt.Errorf("unmarshal block: %w", err)
        }

        b = sm.ToDomain()
        return nil
    })
    return b, err
}
```

### go-zero Config Loading

```go
// Source: go-zero conf package documentation
import (
    "flag"
    "github.com/zeromicro/go-zero/core/conf"
)

func main() {
    var configFile = flag.String("f", "etc/shitcoin.yaml", "config file")
    flag.Parse()

    var c config.Config
    conf.MustLoad(*configFile, &c)

    // c.Consensus.BlockTimeTarget is now populated from YAML
    // c.Consensus.DifficultyAdjustInterval is populated
    // defaults applied for any missing fields
}
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| BoltDB (github.com/boltdb/bolt) | bbolt (go.etcd.io/bbolt) | 2018 | BoltDB is abandoned; bbolt is the maintained fork |
| go-zero v1.3 | go-zero v1.7+ | 2024 | Improved conf package, better goctl, Go 1.22+ support |
| Manual difficulty "number of zeros" | big.Int target comparison | Bitcoin protocol standard | Proper difficulty granularity, not limited to whole-byte steps |
| encoding/json v1 | encoding/json/v2 (experimental) | 2025 proposal | v2 has explicit Deterministic option, but v1 is sufficient for structs |

**Deprecated/outdated:**
- `github.com/boltdb/bolt`: Abandoned. Use `go.etcd.io/bbolt` instead.
- goctl styles without `--style` flag: Always specify `--style go_zero` for consistent naming.

## Open Questions

1. **Hashing Format: Binary vs JSON**
   - What we know: User decided "JSON for block encoding (hashing and storage)." Go struct JSON marshaling is deterministic. Binary encoding is more Bitcoin-like and faster.
   - What's unclear: Should we use JSON serialization for the hash input, or binary encoding? JSON is more debuggable (can inspect what's being hashed), binary is more efficient and closer to Bitcoin.
   - Recommendation: Use JSON for hashing to maximize debuggability (aligned with user's stated priority). Performance is not a concern for an educational project. Document the difference from Bitcoin's binary approach.

2. **Difficulty Representation: Simple Bits vs Bitcoin Compact nBits**
   - What we know: Bitcoin uses a compact 4-byte encoding with coefficient+exponent. Many educational implementations use a simpler "number of leading zero bits."
   - What's unclear: Which is appropriate for this project's educational goals?
   - Recommendation: Start with simple "target bits" (uint32, number of leading zero bits). This is clearer to understand and sufficient for educational purposes. Can be enhanced later if needed.

3. **Tilt.dev Setup Without Docker/Kubernetes**
   - What we know: Tilt.dev is typically used with Docker/Kubernetes for local dev. This project is a standalone Go binary.
   - What's unclear: Is Tilt necessary for Phase 1? A simple `go run` or Makefile may suffice.
   - Recommendation: Defer Tilt setup to a later phase when multi-node orchestration is needed (Phase 4 or Phase 6). For Phase 1, use a Makefile with `go run` and `go build` targets. Note: Tilt is not currently installed on the dev machine.

## Validation Architecture

### Test Framework
| Property | Value |
|----------|-------|
| Framework | Go testing (stdlib) + go test |
| Config file | None needed (Go testing is zero-config) |
| Quick run command | `go test ./internal/domain/... -v -count=1` |
| Full suite command | `go test -race -cover ./...` |

### Phase Requirements -> Test Map
| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| MINE-01 | Genesis block creation with configurable message | unit | `go test ./internal/domain/block/... -run TestGenesisBlock -v` | No -- Wave 0 |
| MINE-01 | Genesis block persisted to disk | integration | `go test ./internal/infrastructure/persistence/bbolt/... -run TestGenesisBlockPersistence -v` | No -- Wave 0 |
| MINE-02 | Block structure has correct header and body fields | unit | `go test ./internal/domain/block/... -run TestBlockStructure -v` | No -- Wave 0 |
| MINE-03 | SHA-256d produces correct deterministic hash | unit | `go test ./internal/domain/block/... -run TestDoubleSHA256 -v` | No -- Wave 0 |
| MINE-03 | Same block always produces same hash | unit | `go test ./internal/domain/block/... -run TestDeterministicHashing -v` | No -- Wave 0 |
| MINE-03 | Mined block hash meets difficulty target | unit | `go test ./internal/domain/block/... -run TestProofOfWork -v` | No -- Wave 0 |
| MINE-06 | Difficulty adjusts after N blocks (increases when fast) | unit | `go test ./internal/domain/block/... -run TestDifficultyAdjustment -v` | No -- Wave 0 |
| MINE-06 | Difficulty adjustment clamped to [0.25, 4.0] factor | unit | `go test ./internal/domain/block/... -run TestDifficultyAdjustmentClamping -v` | No -- Wave 0 |
| MINE-09 | Consensus params loaded from YAML config | integration | `go test ./internal/config/... -run TestConfigLoading -v` | No -- Wave 0 |
| MINE-09 | Default values applied when params not in config | unit | `go test ./internal/config/... -run TestConfigDefaults -v` | No -- Wave 0 |
| E2E-01 | Chain persists and loads across restarts | integration | `go test ./internal/infrastructure/persistence/bbolt/... -run TestChainPersistence -v` | No -- Wave 0 |

### Sampling Rate
- **Per task commit:** `go test ./internal/domain/... -v -count=1`
- **Per wave merge:** `go test -race -cover ./...`
- **Phase gate:** Full suite green before `/gsd:verify-work`

### Wave 0 Gaps
- [ ] `internal/domain/block/block_test.go` -- covers MINE-01, MINE-02, MINE-03
- [ ] `internal/domain/block/pow_test.go` -- covers MINE-03 (PoW validation)
- [ ] `internal/domain/block/difficulty_test.go` -- covers MINE-06
- [ ] `internal/config/config_test.go` -- covers MINE-09
- [ ] `internal/infrastructure/persistence/bbolt/chain_repo_test.go` -- covers persistence (MINE-01 storage, E2E-01)
- [ ] `go.mod` initialization -- `go mod init github.com/baotoq/shitcoin`
- [ ] No test framework install needed (Go testing is stdlib)

## bbolt Bucket Layout (Claude's Discretion)

### Recommended Design

```
Bucket: "blocks"
  Key: block_hash (32 bytes)     -> Value: JSON-serialized block
  Key: h:<height> (prefix + 8 bytes big-endian) -> Value: block_hash (32 bytes)

Bucket: "chain_meta"
  Key: "latest_hash"             -> Value: block_hash (32 bytes)
  Key: "height"                  -> Value: 8 bytes big-endian uint64
```

**Rationale:**
- Block lookup by hash is the primary operation (O(1) with bbolt B+tree)
- Height-to-hash index enables sequential chain traversal (difficulty adjustment needs blocks by height)
- Separate metadata bucket keeps chain state (tip hash, height) queryable without scanning blocks
- Using `h:` prefix for height keys prevents collision with 32-byte hash keys in the same bucket (alternatively, use a separate `"height_index"` bucket)

## Sources

### Primary (HIGH confidence)
- [go.etcd.io/bbolt v1.4.3 - pkg.go.dev](https://pkg.go.dev/go.etcd.io/bbolt) -- API reference, transaction patterns, version
- [crypto/sha256 - pkg.go.dev](https://pkg.go.dev/crypto/sha256) -- SHA-256 implementation in Go stdlib
- [encoding/json - pkg.go.dev](https://pkg.go.dev/encoding/json) -- JSON marshaling determinism guarantees for structs
- [go-zero configuration overview](https://go-zero.dev/en/docs/tutorials/go-zero/configuration/overview) -- conf.MustLoad, struct tags, YAML support
- [GitHub - etcd-io/bbolt](https://github.com/etcd-io/bbolt) -- bbolt README, usage guide
- [GitHub - zeromicro/go-zero](https://github.com/zeromicro/go-zero) -- go-zero framework
- [go-zero zero-skills SKILL.md](/Users/baotoq/Work/shitcoin/.claude/skills/zero-skills/SKILL.md) -- Project skill patterns
- [golang-ddd SKILL.md](/Users/baotoq/Work/shitcoin/.claude/skills/golang-ddd/SKILL.md) -- DDD tactical patterns for Go
- [golang-testing SKILL.md](/Users/baotoq/Work/shitcoin/.claude/skills/golang-testing/SKILL.md) -- Go testing patterns
- [encoding/json Issue #15424](https://github.com/golang/go/issues/15424) -- Confirmation that JSON object keys from structs are sorted (declaration order)

### Secondary (MEDIUM confidence)
- [Jeiwan: Building Blockchain in Go Part 2](https://jeiwan.net/posts/building-blockchain-in-go-part-2/) -- PoW implementation pattern with big.Int target
- [btcsuite/btcd difficulty.go](https://github.com/btcsuite/btcd/blob/master/blockchain/difficulty.go) -- Reference difficulty adjustment implementation in Go
- [learnmeabitcoin.com: Difficulty](https://learnmeabitcoin.com/beginners/guide/difficulty/) -- Difficulty adjustment formula and clamping (4x max factor)
- [Bitcoin Optech: Difficulty adjustment algorithms](https://bitcoinops.org/en/topics/difficulty-adjustment-algorithms/) -- Adjustment algorithm overview
- [go-zero goctl-commands.md](/Users/baotoq/Work/shitcoin/.claude/skills/zero-skills/references/goctl-commands.md) -- goctl code generation commands

### Tertiary (LOW confidence)
- [Tilt.dev Go example](https://docs.tilt.dev/example_go.html) -- Tilt setup for Go; may not apply directly since project is not containerized in Phase 1

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH -- all libraries are well-documented, versions verified, installed on dev machine (Go 1.23.4, goctl 1.9.2)
- Architecture: HIGH -- DDD + go-zero pattern is well-established, skills docs provide detailed guidance
- Hashing/PoW: HIGH -- SHA-256d and PoW loop are well-documented in multiple educational blockchain projects and Bitcoin protocol specs
- Difficulty adjustment: HIGH -- Bitcoin's algorithm is simple (ratio * clamp), well-documented with multiple sources agreeing
- bbolt usage: HIGH -- API verified against pkg.go.dev, v1.4.3 confirmed
- JSON determinism: HIGH -- confirmed by Go issue tracker and official docs that struct field order is declaration order
- Pitfalls: MEDIUM -- derived from multiple educational blockchain projects and bbolt documentation; some are from training data
- Tilt.dev integration: LOW -- not installed, may be deferred

**Research date:** 2026-03-05
**Valid until:** 2026-04-05 (30 days -- all technologies are stable)
