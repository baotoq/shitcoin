---
phase: 03-mempool-mining-integration-and-cli
plan: 01
subsystem: domain
tags: [mempool, merkle-tree, concurrency, sync, mining]

# Dependency graph
requires:
  - phase: 02-wallets-and-transactions
    provides: "UTXO set, typed transactions, signing/verification"
provides:
  - "Thread-safe mempool with validation (Add, DrainAll, Remove, Count, Transactions)"
  - "ComputeMerkleRoot function for binary hash tree"
  - "Merkle root integrated into NewBlock/NewGenesisBlock/MineBlock"
affects: [03-02, 03-03, 04-p2p-networking]

# Tech tracking
tech-stack:
  added: []
  patterns: [sync.RWMutex for concurrent domain objects, spentOutputs tracking map for pool-level double-spend detection]

key-files:
  created:
    - internal/domain/mempool/mempool.go
    - internal/domain/mempool/errors.go
    - internal/domain/mempool/mempool_test.go
    - internal/domain/block/merkle.go
    - internal/domain/block/merkle_test.go
  modified:
    - internal/domain/block/block.go
    - internal/domain/chain/chain.go
    - internal/domain/block/block_test.go
    - internal/domain/block/pow_test.go
    - internal/infrastructure/persistence/bbolt/chain_repo_test.go

key-decisions:
  - "Bitcoin Merkle convention: single leaf hashed with itself (not returned directly)"
  - "Mempool tracks spentOutputs map separately for O(1) double-spend detection against pool"

patterns-established:
  - "Mempool validation order: duplicate check, signature verify, UTXO existence, double-spend against pool"
  - "Merkle tree: odd-count levels duplicate last hash; loop until single root"

requirements-completed: [NET-03, MINE-07]

# Metrics
duration: 6min
completed: 2026-03-05
---

# Phase 3 Plan 1: Mempool and Merkle Root Summary

**Thread-safe mempool with UTXO/signature validation and Bitcoin-style Merkle root integrated into block construction**

## Performance

- **Duration:** 6 min
- **Started:** 2026-03-05T14:45:22Z
- **Completed:** 2026-03-05T14:51:00Z
- **Tasks:** 2
- **Files modified:** 10

## Accomplishments
- Mempool domain package with concurrent-safe Add, DrainAll, Remove, Count, Transactions
- Mempool validates signatures, UTXO existence, duplicate detection, and pool-level double-spend
- ComputeMerkleRoot handles empty, single, even, and odd transaction hash counts
- Block construction (NewBlock, NewGenesisBlock) now accepts and uses merkleRoot parameter
- Chain.MineBlock and Chain.Initialize compute Merkle root from transaction hashes before block creation

## Task Commits

Each task was committed atomically:

1. **Task 1: Mempool domain package and Merkle root computation (RED)** - `07d88ab` (test)
2. **Task 1: Mempool domain package and Merkle root computation (GREEN)** - `edb645c` (feat)
3. **Task 2: Integrate Merkle root into block construction and mining** - `871a558` (feat)

_Note: TDD Task 1 has separate RED and GREEN commits_

## Files Created/Modified
- `internal/domain/mempool/mempool.go` - Thread-safe mempool with validation
- `internal/domain/mempool/errors.go` - ErrDuplicate, ErrDoubleSpend, ErrInvalidSignature, ErrUTXONotFound
- `internal/domain/mempool/mempool_test.go` - 9 tests including concurrent access with -race
- `internal/domain/block/merkle.go` - ComputeMerkleRoot with Bitcoin binary tree convention
- `internal/domain/block/merkle_test.go` - 6 tests for empty, single, two, odd, even, deterministic
- `internal/domain/block/block.go` - NewBlock/NewGenesisBlock accept merkleRoot parameter
- `internal/domain/chain/chain.go` - Initialize and MineBlock compute Merkle root
- `internal/domain/block/block_test.go` - Updated for new signature
- `internal/domain/block/pow_test.go` - Updated for new signature
- `internal/infrastructure/persistence/bbolt/chain_repo_test.go` - Updated for new signature

## Decisions Made
- Bitcoin Merkle convention: single leaf is hashed with itself (DoubleSHA256(h || h)), not returned as-is
- Mempool tracks a separate spentOutputs map (string -> Hash) for O(1) double-spend detection against the pool, cleaned up on Remove/DrainAll

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Fixed single-hash Merkle root computation**
- **Found during:** Task 1 GREEN phase
- **Issue:** Initial implementation returned single hash directly; Bitcoin convention hashes single leaf with itself
- **Fix:** Changed loop to always execute at least one hashing round, duplicating odd-count (including single) hashes
- **Files modified:** internal/domain/block/merkle.go
- **Verification:** TestMerkleRoot_Single passes
- **Committed in:** edb645c (Task 1 GREEN commit)

---

**Total deviations:** 1 auto-fixed (1 bug)
**Impact on plan:** Essential correctness fix for Bitcoin Merkle tree convention. No scope creep.

## Issues Encountered
None

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- Mempool ready for CLI integration (Plan 03-02 / 03-03)
- Merkle root integrated into mining pipeline
- All 15 new tests + full existing suite pass with -race flag

---
*Phase: 03-mempool-mining-integration-and-cli*
*Completed: 2026-03-05*
