# Project Research Summary

**Project:** Shitcoin — Educational Bitcoin-like Blockchain in Go
**Domain:** Educational blockchain implementation (UTXO-based, PoW, P2P)
**Researched:** 2026-03-05
**Confidence:** HIGH

## Executive Summary

Shitcoin is an educational Bitcoin-like blockchain implemented in Go. The research confirms this is a well-trodden domain with strong reference implementations (Jeiwan/blockchain_go, btcd) and clear patterns for every component. The recommended approach is a layered architecture with clean Go package boundaries, using stdlib-first dependencies (crypto/sha256, encoding/gob, net, testing) augmented by a small set of well-justified external libraries: btcec/v2 for secp256k1 ECDSA (unavailable in stdlib), bbolt for embedded KV storage, cobra for CLI, and echo+templ+HTMX for the web dashboard. The entire external dependency footprint is intentionally minimal — 8 packages — so the project teaches blockchain concepts rather than dependency management.

The key differentiator of this project over comparable tutorials is the combination of three features that no single comparable Go project offers: real P2P networking with fork detection and chain reorganization (most tutorials skip reorg entirely), a web-based block explorer and mining visualization dashboard (all comparable CLI-only projects have nothing here), and multi-node testnet orchestration. These three features together elevate the project from a tutorial reproduction into a complete, demonstrable educational platform.

The most dangerous risks are architectural: non-deterministic serialization (maps in hashable structs silently fork the chain between nodes), UTXO double-spend from concurrent mempool access (race conditions invisible until P2P is added), and BoltDB deadlocks from mixed read/write transactions on the same goroutine. All three require correct design decisions at the very start — they cannot be retrofitted. The build order must start with deterministic canonical serialization, design the UTXO set for concurrency from day one, and use bbolt's `db.Update()` pattern exclusively for read-then-write operations.

## Key Findings

### Recommended Stack

The stack leans heavily on Go's stdlib for the educational core: `crypto/sha256` for all hashing, `crypto/rand` for key generation, `encoding/gob` for local storage serialization, `net` for raw TCP P2P, and `testing` for the test harness. The only external cryptographic dependency is `btcec/v2` (secp256k1 ECDSA — Go stdlib only supports NIST curves) plus `golang.org/x/crypto` (RIPEMD-160 for address derivation). Storage uses `bbolt` v1.4.3 over BadgerDB because bbolt uses 5-10x less memory (critical when running 3+ nodes on localhost), has a bucket concept that maps naturally to blockchain data, and is the established choice in every comparable Go blockchain tutorial.

The web dashboard uses echo v4 + templ + HTMX from CDN. This avoids any JavaScript build toolchain while delivering real-time dashboard updates and type-safe templates. The deliberate avoidance of React/Vue/Node.js keeps the project pure Go.

**Core technologies:**
- `btcec/v2`: secp256k1 ECDSA signing/verification — Go stdlib cannot handle Bitcoin's curve (a=0)
- `golang.org/x/crypto`: RIPEMD-160 — stdlib missing, needed for Bitcoin address derivation (SHA-256 -> RIPEMD-160 -> Base58Check pipeline)
- `bbolt v1.4.3`: Embedded KV storage — low memory, bucket concept, ACID transactions, read-optimized
- `cobra v1.10.2`: CLI framework — industry standard for subcommand CLIs, used by kubectl/docker/gh
- `echo v4.15.1`: HTTP server for dashboard — lightweight, net/http compatible, built-in middleware
- `templ v0.3.x`: Type-safe HTML templates — compile-time errors vs html/template runtime panics
- `HTMX 2.x (CDN)`: Dynamic UI without JavaScript build pipeline — server-rendered HTML fragments
- `testify v1.10.x`: Test assertions — assert/require/mock packages for readable tests

### Expected Features

The feature research clearly segments the feature set into three priority tiers based on dependencies and educational value.

**Must have (table stakes — P1):**
- Genesis block, block structure with SHA-256 double-hash
- Proof of Work mining with adjustable difficulty (every N blocks, window-based, clamped)
- UTXO transaction model with coinbase transactions
- ECDSA key generation and Bitcoin-style address derivation (SHA-256 -> RIPEMD-160 -> Base58Check)
- Transaction signing and ECDSA signature verification
- UTXO set management (persistent cache, never scan full chain for balance queries)
- Merkle tree in block headers
- Persistent storage (bbolt)
- CLI: createwallet, listaddresses, getbalance, send, mine, printchain, startnode

**Should have (competitive differentiators — P2):**
- P2P networking over localhost TCP with version handshake
- Mempool for pending transactions with validation
- Block and transaction broadcasting (gossip protocol)
- Chain synchronization for new nodes (initial block download)
- Fork detection and longest-chain reorganization — no comparable tutorial implements this
- Web dashboard: block explorer, node status, real-time mining visualization, mempool view

**Defer (v2+ / polish — P3):**
- Multi-node orchestration script (one-command testnet)
- Block reward halving (simple arithmetic on block height)
- Transaction fees with miner prioritization
- Double-spend detection demo
- Configurable consensus parameters
- Chain export/import for reproducible demos

**Explicitly not building:** Bitcoin protocol compatibility, Bitcoin Script VM, SPV clients, GPU mining, BIP-39 mnemonics, smart contracts. These add complexity without blockchain conceptual value and would require separate projects in their own right.

### Architecture Approach

The project follows a layered architecture with strict package boundaries, inspired by btcd's modular design. The layers from bottom to top are: Storage (bbolt, behind a `Store` interface) -> Core Domain (block, blockchain, tx, utxo, wallet, consensus, mempool) -> Services (miner, network) -> Interface (CLI, API, Dashboard). The Storage layer is abstracted behind an interface from day one, enabling in-memory test implementations. Event-driven component communication via Go channels connects the layers — when a block is mined or received from a peer, an `EventBus` emits events that other components subscribe to. The mining loop uses `context.Context` cancellation so it stops when a valid block arrives from the network.

**Major components:**
1. `internal/core` — Block, Blockchain, genesis, Merkle tree; the immutable chain backbone
2. `internal/tx` + `internal/utxo` — UTXO transaction model, signing, UTXO set (with undo-log for reorg)
3. `internal/wallet` — ECDSA key generation, address derivation, key persistence
4. `internal/consensus` — PoW nonce search (int64, context-cancellable), difficulty target computation
5. `internal/mempool` — Pending transaction pool with RWMutex, validation, conflict detection
6. `internal/miner` — Block assembly from mempool, mining loop with context cancellation
7. `internal/network` — TCP P2P, length-prefix message framing, peer lifecycle, chain sync, reorg
8. `internal/storage` — Store interface + bbolt implementation + bucket definitions (blocks/utxo/metadata)
9. `cli/`, `api/`, `web/` — Interface layers consuming all internal packages

### Critical Pitfalls

1. **Non-deterministic serialization (CRITICAL, Phase 1)** — Never use `map` in any data structure that gets hashed. Define a separate, hand-written canonical serialization function (using `binary.Write` in fixed field order) for all block/transaction hashing. Reserve `encoding/gob` for storage only. This must be correct from Phase 1 — retrofitting requires rewriting every hash function.

2. **UTXO double-spend from concurrent mempool access (CRITICAL, Phase 2-3)** — The mempool and UTXO set are shared state accessed by multiple goroutines. Use `sync.RWMutex` protecting the entire "validate + add to mempool" operation atomically. Run `go test -race` from day one in every package.

3. **Missing change outputs destroy coins (CRITICAL, Phase 2)** — In the UTXO model, always compute `change = totalInput - payment` and create a change output. Enforce invariant: `sum(outputs) == sum(inputs)` for all non-coinbase transactions.

4. **BoltDB deadlock from mixed read/write transactions (CRITICAL, Phase 1-2)** — Never open a read and write transaction on the same goroutine. Always use `db.Update()` when reading-then-writing. Store UTXO set updates in the same `db.Update()` call as the new block write.

5. **No chain reorganization handling (HIGH, Phase 2+4)** — Design the UTXO set as reversible from Phase 2: log which outputs were consumed/created per block. Without this foundation, implementing reorg in Phase 4 is a partial rewrite.

## Implications for Roadmap

Based on the feature dependency graph (FEATURES.md) and architectural build order (ARCHITECTURE.md), the natural phase structure is:

### Phase 1: Core Chain Foundation
**Rationale:** All other components depend on a correct block structure with deterministic hashing. Serialization correctness must be established first — retrofitting it later requires rewriting every hash function in the project.
**Delivers:** Genesis block, Block struct with hand-written canonical SHA-256d hashing, Blockchain with bbolt persistence, PoW mining (int64 nonce, context-cancellable), adjustable difficulty with window-based adjustment
**Addresses:** Genesis block, block structure, SHA-256 hashing, PoW mining, adjustable difficulty (FEATURES.md P1)
**Avoids:** Non-deterministic serialization (Pitfall 1), nonce overflow/infinite loop (Pitfall 8), BoltDB deadlock (Pitfall 4), genesis block special cases (Pitfall 13)
**Stack:** `crypto/sha256`, `bbolt`, `encoding/gob` (storage only), `encoding/binary` (canonical hashing)

### Phase 2: Wallets and Transactions
**Rationale:** The UTXO transaction model is the most educationally complex component. Keys and addresses must exist before non-coinbase transactions can be created. UTXO reversibility for future chain reorg must be designed in here — it cannot be retrofitted later without rewriting the storage schema.
**Delivers:** ECDSA key generation, Bitcoin-style address derivation (SHA-256 -> RIPEMD-160 -> Base58Check), UTXO transaction model with coinbase, transaction signing and verification, UTXO set with undo-log support, balance queries
**Addresses:** Key generation, address derivation, UTXO model, coinbase transactions, tx signing/verification, UTXO set management (FEATURES.md P1)
**Avoids:** Missing change outputs (Pitfall 3 — sum invariant), UTXO set desync on startup (Pitfall 10 — rebuild from chain), signature malleability (Pitfall 11 — hash excludes signatures), concurrent mempool races (Pitfall 2 — RWMutex design)
**Stack:** `btcec/v2`, `golang.org/x/crypto`, `btcutil/base58`, `crypto/sha256`

### Phase 3: Mempool and CLI
**Rationale:** The mempool bridges transactions and networking. CLI provides the exercise harness for all local functionality before P2P complexity is introduced — having a working CLI is also essential for debugging the P2P phase effectively.
**Delivers:** Mempool with RWMutex-protected validate+add, Merkle tree (with odd-count handling), full cobra CLI (createwallet, getbalance, send, mine, printchain, startnode), `go test -race` baseline
**Addresses:** Mempool, Merkle tree, CLI operations (FEATURES.md P1/P2)
**Avoids:** Concurrent mempool double-spend (Pitfall 2 — RWMutex), Merkle tree panic with odd tx count (Pitfall 14), no race detection (Pitfall 17)
**Stack:** `cobra`, `viper`, `testify`

### Phase 4: P2P Networking and Sync
**Rationale:** The hardest phase. All local functionality must be solid first. This phase brings the project from a single-node database to a distributed consensus system, including the fork/reorg logic that most tutorials skip. Chain reorg relies on the UTXO undo-log designed in Phase 2.
**Delivers:** TCP P2P with version handshake, length-prefix message framing (io.ReadFull), JSON wire protocol with version byte, message types (version/inv/getdata/block/tx/addr), peer lifecycle with context.WithCancel + WaitGroup, chain synchronization (IBD), block and transaction broadcasting, fork detection and longest-chain reorganization using UTXO undo-log
**Addresses:** P2P networking, block/tx broadcasting, chain sync, fork detection/resolution (FEATURES.md P2)
**Avoids:** TCP message framing errors (Pitfall 7 — length-prefix from the start), goroutine leaks (Pitfall 6 — context per peer), gob wire format instability (Pitfall 15 — use JSON), no reorg handling (Pitfall 5 — UTXO reversibility from Phase 2)
**Stack:** `net`, JSON wire format, `context`, `sync.WaitGroup`

### Phase 5: Web Dashboard and API
**Rationale:** Pure visualization layer on top of a fully functioning distributed node. Building it last ensures the dashboard displays real data from a working system. No blockchain functionality depends on this phase.
**Delivers:** REST/JSON API layer, block explorer (browse blocks/transactions/UTXOs, search by hash/address), node status panel (peers, mempool size, chain height, hashrate), real-time mining visualization via SSE or WebSocket, mempool pending transaction view, multi-node orchestration script
**Addresses:** Web dashboard, block explorer, node status, mining visualization, mempool visualization, multi-node orchestration (FEATURES.md P2/P3)
**Avoids:** No new pitfalls; relies on all prior mitigations
**Stack:** `echo v4`, `templ`, HTMX CDN, Tailwind CDN

### Phase 6: Advanced Educational Features
**Rationale:** These features have low implementation complexity relative to their educational demo value but are not on the critical path. They should be deferred to ensure all core functionality is solid and testable.
**Delivers:** Block reward halving (every N blocks, configurable), transaction fees with miner prioritization by fee rate, double-spend detection demo, configurable consensus parameters (block time target, difficulty interval, reward schedule)
**Addresses:** Block reward halving, transaction fees, double-spend demo, configurable parameters (FEATURES.md P3)

### Phase Ordering Rationale

- Phases 1-3 are governed by hard dependency chains: hashing must exist before transactions, transactions before mempool, RWMutex before P2P.
- The most dangerous pitfalls (non-deterministic serialization, change outputs, UTXO concurrency, bbolt deadlocks) must be addressed in their respective early phases — none can be retrofitted without significant rework.
- P2P (Phase 4) is placed after a working, CLI-exercisable local blockchain so there is something real to sync, and the UTXO undo-log design from Phase 2 is available for reorg.
- The web dashboard (Phase 5) is last because it is a visualization of a working distributed system, not a dependency of anything.
- Phase 6 extras are fully decoupled and can be inserted or deferred independently.

### Research Flags

Phases likely needing `/gsd:research-phase` during planning:
- **Phase 4 (P2P Networking):** Message protocol design, chain reorg algorithm details, and peer connection lifecycle management are complex with multiple viable approaches. Deeper study of Jeiwan Part 7 and btcd sync logic would reduce implementation risk.
- **Phase 2 (UTXO reversibility):** The exact undo-log data structure needed to support chain reorganization should be researched before UTXO set implementation begins — getting this wrong requires a storage schema rewrite.

Phases with standard patterns (skip research-phase):
- **Phase 1 (Core Chain):** Extremely well-documented in multiple canonical tutorials. Standard patterns are unambiguous.
- **Phase 3 (Mempool + CLI):** cobra/viper patterns are well-established; mempool mutex design is straightforward with standard Go sync primitives.
- **Phase 5 (Dashboard):** echo + templ + HTMX patterns are well-documented. No novel integration challenges.
- **Phase 6 (Extras):** All features are trivial extensions of already-built components.

## Confidence Assessment

| Area | Confidence | Notes |
|------|------------|-------|
| Stack | HIGH | All dependencies verified against pkg.go.dev with exact versions. secp256k1 requirement cross-confirmed (Go stdlib cannot support it). bbolt vs BadgerDB decision backed by benchmark data and multi-node memory concerns. |
| Features | HIGH | Feature set cross-validated against 3 comparable open-source implementations. Feature dependency graph is thorough. MVP vs v2+ split is well-reasoned and matches educational goals. |
| Architecture | HIGH | Package structure derived from btcd (production) and multiple educational Go blockchain references. Patterns (storage interface, event bus, context cancellation) are idiomatic Go. |
| Pitfalls | HIGH | Critical pitfalls sourced from official issue trackers (BoltDB GitHub), official Go docs (race detector), and reference production implementations (btcd mempool, go-ethereum keystore). |

**Overall confidence:** HIGH

### Gaps to Address

- **Wire format decision:** PITFALLS.md recommends against `encoding/gob` for the P2P wire format (version instability across Go releases), but ARCHITECTURE.md uses gob for messages. Final decision needed before Phase 4: JSON for wire protocol (human-readable, debuggable, stable) and gob for local storage only. This is the recommended resolution.
- **UTXO reversibility data structure:** The exact undo-log structure needed to support chain reorg is not fully specified in the architecture research. Resolve at the start of Phase 2 before UTXO set implementation begins — getting this wrong is expensive to fix.
- **Difficulty adjustment parameters:** Research recommends window-based adjustment with a clamped factor, but exact values for N (adjustment interval), target block time, and clamp ratio are not specified. Make these configurable parameters decided and documented before Phase 1 is finalized.
- **templ maturity risk:** templ is marked MEDIUM confidence (newer library, v0.3.x). Fallback to `html/template` is available if templ proves problematic, with significant DX tradeoff.

## Sources

### Primary (HIGH confidence)
- [btcsuite/btcd](https://github.com/btcsuite/btcd) — Production Go Bitcoin full node; architecture reference for all patterns
- [Jeiwan/blockchain_go](https://github.com/Jeiwan/blockchain_go) — Canonical 7-part Go blockchain tutorial; all fundamental patterns
- [BoltDB GitHub issue #378](https://github.com/boltdb/bolt/issues/378) — bbolt deadlock concurrency model, read/write transaction rules
- [Go race detector official docs](https://go.dev/doc/articles/race_detector) — Race detection guidance
- [btcec/v2 pkg.go.dev](https://pkg.go.dev/github.com/btcsuite/btcd/btcec/v2) — v2.3.6, Oct 2025
- [bbolt pkg.go.dev](https://pkg.go.dev/go.etcd.io/bbolt) — v1.4.3, Aug 2025
- [Echo v4 pkg.go.dev](https://pkg.go.dev/github.com/labstack/echo/v4) — v4.15.1, Feb 2026
- [Cobra pkg.go.dev](https://pkg.go.dev/github.com/spf13/cobra) — v1.10.2, Dec 2025
- [btcd mempool implementation](https://github.com/btcsuite/btcd/blob/master/mempool/mempool.go) — Mempool architecture and concurrency reference

### Secondary (MEDIUM confidence)
- [TheODDYSEY/Blockchain-Go](https://github.com/TheODDYSEY/Blockchain-Go) — Progressive build structure, TCP P2P reference
- [BoltDB vs Badger comparison](https://tech.townsourced.com/post/boltdb-vs-badger/) — Memory and performance tradeoffs
- [Badger vs LMDB vs BoltDB benchmarks](https://hypermode.com/blog/badger-lmdb-boltdb/) — Independent benchmark data
- [LearnMeABitcoin - Chain Reorganization](https://learnmeabitcoin.com/technical/blockchain/chain-reorganization/) — Fork resolution and longest-chain rule
- [LearnMeABitcoin - Difficulty](https://learnmeabitcoin.com/beginners/guide/difficulty/) — Bitcoin difficulty adjustment explanation
- [Jeiwan - Network (Part 7)](https://jeiwan.net/posts/building-blockchain-in-go-part-7/) — P2P protocol design reference
- [templ pkg.go.dev](https://pkg.go.dev/github.com/a-h/templ) — v0.3.x, Feb 2026
- [JSON vs GOB benchmarks](https://blog.vitalvas.com/post/2025/07/23/json-vs-gob-in-golang/) — gob 2-4x faster for complex structs (relevant to storage choice)

### Tertiary (LOW confidence)
- [Go map non-determinism in Cosmos SDK](https://ashourics.medium.com/the-challenge-of-gos-map-iteration-in-the-cosmos-sdk-blockchain-a-dive-into-determinism-bd5a99260519) — Non-determinism pitfall example; single source, needs validation

---
*Research completed: 2026-03-05*
*Ready for roadmap: yes*
