---
phase: 01-core-chain-foundation
plan: 01
subsystem: domain
tags: [blockchain, sha256, pow, ddd, go-zero, config]

# Dependency graph
requires: []
provides:
  - "Block entity with NewGenesisBlock and NewBlock factories"
  - "Header value object with deterministic JSON-based HashPayload"
  - "Hash value object with DoubleSHA256, String, IsZero, Bytes, HashFromHex"
  - "ProofOfWork domain service with Mine, MineWithMaxNonce, and Validate"
  - "BitsToTarget difficulty target computation"
  - "Sentinel errors ErrNonceExhausted, ErrInvalidBlock"
  - "Config struct with ConsensusConfig and StorageConfig, go-zero compatible"
  - "YAML config file with consensus parameters"
affects: [01-02, 02-wallets-transactions, 03-mempool-mining-cli]

# Tech tracking
tech-stack:
  added: [go-zero v1.10.0, bbolt v1.4.3, crypto/sha256, math/big, encoding/json]
  patterns: [DDD tactical (unexported fields, factory functions, value receivers for VOs, pointer receivers for entities), go-zero json struct tags for config, JSON-based deterministic hashing]

key-files:
  created:
    - go.mod
    - go.sum
    - etc/shitcoin.yaml
    - internal/config/config.go
    - internal/config/config_test.go
    - internal/domain/block/hash.go
    - internal/domain/block/header.go
    - internal/domain/block/block.go
    - internal/domain/block/block_test.go
    - internal/domain/block/pow.go
    - internal/domain/block/pow_test.go
    - internal/domain/block/errors.go
  modified: []

key-decisions:
  - "JSON serialization for hashing (debuggable, deterministic via struct field order)"
  - "Timestamp as int64 Unix seconds (not time.Time) to avoid precision issues"
  - "GenesisMessage default via ApplyDefaults() method to avoid go vet struct tag warning"
  - "MineWithMaxNonce added for testable nonce exhaustion"

patterns-established:
  - "DDD value objects: Hash and Header use value receivers, unexported fields, factory functions"
  - "DDD entity: Block uses pointer receivers, unexported fields, SetHash/SetHeaderNonce for PoW mutation"
  - "go-zero config: json struct tags with defaults, optional for string defaults with spaces"
  - "hashableHeader struct: deterministic JSON serialization for block hashing"

requirements-completed: [MINE-02, MINE-03, MINE-09]

# Metrics
duration: 6min
completed: 2026-03-05
---

# Phase 1 Plan 01: Project Scaffold and Domain Types Summary

**Go project with DDD block types, SHA-256d double-hashing, and ProofOfWork mining service using JSON-based deterministic serialization**

## Performance

- **Duration:** 6 min
- **Started:** 2026-03-04T18:11:26Z
- **Completed:** 2026-03-04T18:17:33Z
- **Tasks:** 2
- **Files modified:** 12

## Accomplishments
- Go module initialized with go-zero v1.10.0 and bbolt v1.4.3 dependencies
- Core blockchain domain types: Hash (value object), Header (value object), Block (entity) following DDD tactical patterns
- SHA-256d double-hash function verified against known test vectors with deterministic JSON serialization
- ProofOfWork domain service mines blocks with bits=16 difficulty in ~30ms, validates mined blocks, detects nonce exhaustion
- Config system loads consensus parameters from YAML with defaults for omitted fields
- 22 passing tests covering hash, header, block, config, and PoW behaviors

## Task Commits

Each task was committed atomically:

1. **Task 1: Project scaffold, config, and domain type contracts** - `374e5c2` (feat)
2. **Task 2a: PoW failing tests (TDD red)** - `25d70a5` (test)
3. **Task 2b: ProofOfWork mining service implementation (TDD green)** - `84bb0b3` (feat)

## Files Created/Modified
- `go.mod` - Go module with github.com/baotoq/shitcoin, go-zero and bbolt deps
- `go.sum` - Dependency checksums
- `etc/shitcoin.yaml` - Consensus params: BlockTimeTarget=10, DifficultyAdjustInterval=10, InitialDifficulty=16, GenesisMessage="Hello, Shitcoin!"
- `internal/config/config.go` - Config struct with ConsensusConfig (defaults via json tags) and StorageConfig
- `internal/config/config_test.go` - Tests for config loading and defaults
- `internal/domain/block/hash.go` - Hash [32]byte value object with DoubleSHA256, String, IsZero, Bytes, HashFromHex
- `internal/domain/block/header.go` - Header value object with deterministic HashPayload via JSON-serialized hashableHeader
- `internal/domain/block/block.go` - Block entity with NewGenesisBlock, NewBlock, ReconstructBlock factories
- `internal/domain/block/block_test.go` - 14 tests for hash, header, and block behaviors
- `internal/domain/block/pow.go` - ProofOfWork domain service with Mine, MineWithMaxNonce, Validate, BitsToTarget
- `internal/domain/block/pow_test.go` - 6 tests for PoW mining, validation, determinism, exhaustion, multi-block chain
- `internal/domain/block/errors.go` - Sentinel errors ErrNonceExhausted, ErrInvalidBlock

## Decisions Made
- **JSON for hashing:** Used json.Marshal on a hashableHeader struct for deterministic serialization, aligned with user's debuggability priority
- **Timestamp as int64:** Stored as Unix seconds to avoid time.Time precision issues across serialization boundaries (Pitfall #2)
- **GenesisMessage default pattern:** Used `json:",optional"` tag + ApplyDefaults() method instead of inline default to satisfy go vet (spaces in struct tag values trigger vet warning)
- **MineWithMaxNonce:** Added alongside Mine() to enable practical nonce exhaustion testing without iterating 4B values

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] Missing transitive dependencies in go.sum**
- **Found during:** Task 1 (initial test run)
- **Issue:** `go get` for go-zero didn't pull all transitive deps into go.sum
- **Fix:** Ran `go mod tidy` to resolve all dependencies
- **Files modified:** go.sum
- **Verification:** Tests compile and pass
- **Committed in:** 374e5c2 (Task 1 commit)

**2. [Rule 1 - Bug] go vet warning on GenesisMessage struct tag**
- **Found during:** Task 2 (final verification)
- **Issue:** `json:",default=The Times 03/Jan/2009..."` struct tag contained spaces, triggering go vet "suspicious space" warning
- **Fix:** Changed to `json:",optional"` tag with ApplyDefaults() method for runtime default application
- **Files modified:** internal/config/config.go, internal/config/config_test.go
- **Verification:** go vet passes cleanly, TestConfigDefaults still passes
- **Committed in:** 84bb0b3 (Task 2 commit)

---

**Total deviations:** 2 auto-fixed (1 blocking, 1 bug)
**Impact on plan:** Both fixes necessary for correctness. No scope creep.

## Issues Encountered
None beyond the auto-fixed deviations above.

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- Block, Header, Hash domain types ready for chain aggregate and storage in Plan 02
- ProofOfWork service ready for integration with chain management
- Config infrastructure ready for difficulty adjustment parameters
- Sentinel errors ready for use in chain validation

## Self-Check: PASSED

All 12 files verified present. All 3 commits verified in git log.

---
*Phase: 01-core-chain-foundation*
*Completed: 2026-03-05*
