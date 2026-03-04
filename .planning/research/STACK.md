# Technology Stack

**Project:** Shitcoin -- Educational Bitcoin-like blockchain in Go
**Researched:** 2026-03-05

## Recommended Stack

### Core Language & Runtime

| Technology | Version | Purpose | Why |
|------------|---------|---------|-----|
| Go | 1.23+ | Implementation language | Project constraint. Excellent concurrency (goroutines for mining, P2P), strong stdlib (crypto, net, encoding), single binary output. |

### Cryptography

| Technology | Version | Purpose | Why |
|------------|---------|---------|-----|
| `crypto/sha256` (stdlib) | Go 1.23+ | Block hashing, PoW, Merkle trees, TxID generation | SHA-256 is Bitcoin's hash function. Go stdlib implementation is battle-tested. Zero dependencies. HIGH confidence. |
| `crypto/rand` (stdlib) | Go 1.23+ | Cryptographically secure random number generation | Used for key generation. Stdlib, no deps needed. |
| `github.com/btcsuite/btcd/btcec/v2` | v2.3.6 | secp256k1 ECDSA signing & verification | Go stdlib does NOT support secp256k1 (only P224/P256/P384/P521 -- they require a=-3 in the curve equation, but secp256k1 has a=0). btcec/v2 is THE standard Go secp256k1 library, used by btcd (Go Bitcoin full node) and go-ethereum. Internally uses Decred's optimized pure-Go implementation (dcrd/dcrec/secp256k1/v4). No CGO required. HIGH confidence. |
| `github.com/btcsuite/btcd/btcutil` | (bundled with btcd) | Base58/Base58Check encoding for addresses | Bitcoin-standard address encoding. Same library ecosystem as btcec. Provides `base58.Encode()`, `base58.CheckEncode()` for address derivation. HIGH confidence. |
| `golang.org/x/crypto` | latest | RIPEMD-160 for address derivation | Bitcoin addresses use SHA256 + RIPEMD-160 double hash. Go stdlib lacks RIPEMD-160; the official `x/crypto` supplementary module provides it. HIGH confidence. |

**Why NOT Go stdlib alone for crypto:** Go's `crypto/ecdsa` only supports NIST curves (P224, P256, P384, P521). Bitcoin uses secp256k1 (y^2 = x^3 + 7, where a=0), which Go's elliptic package cannot handle because it hardcodes a=-3. An external library is mandatory.

**Why NOT go-ethereum/crypto:** It wraps the same Decred secp256k1 but pulls in the entire go-ethereum dependency tree (massive). btcec/v2 is the focused, minimal option.

### Storage

| Technology | Version | Purpose | Why |
|------------|---------|---------|-----|
| `go.etcd.io/bbolt` | v1.4.3 | Persistent blockchain storage (blocks, UTXOs, chain state) | Embedded B+ tree KV store. Maintained by the etcd team (CNCF project). Single-file database. ACID transactions with full serializable isolation. Read-optimized (memory-mapped). Blockchain is read-heavy (validation, UTXO lookups) with batch writes (new blocks). Tiny memory footprint -- ideal for educational project running multiple nodes on one machine. HIGH confidence. |

**Why bbolt over BadgerDB:**
- **Memory:** BadgerDB uses 5-10x more memory than bbolt. Running 3+ nodes on localhost simultaneously, this matters.
- **Simplicity:** bbolt has a simpler API with "buckets" (like tables) -- natural for organizing blocks, UTXOs, and metadata separately. BadgerDB is flat key-value only.
- **Read performance:** bbolt reads are faster and more consistent. Blockchain validation hammers reads.
- **Battle-tested for blockchains:** The original BoltDB was used in Jeiwan's canonical Go blockchain tutorial and many educational blockchain projects. bbolt is the actively maintained fork.
- **Write speed tradeoff is acceptable:** BadgerDB writes are faster, but an educational blockchain mining one block every few seconds does not need write throughput.

**Why NOT BadgerDB:** Higher memory usage, no bucket concept, more complex configuration, overkill for this use case. BadgerDB shines for write-heavy workloads (logging, time-series) -- not a fit here.

**Why NOT SQLite:** Adds CGO dependency (unless using modernc.org/sqlite pure-Go, which is slower). KV store maps more naturally to blockchain data access patterns than SQL.

### Serialization

| Technology | Version | Purpose | Why |
|------------|---------|---------|-----|
| `encoding/gob` (stdlib) | Go 1.23+ | Block/transaction binary serialization for storage and network transfer | Go-native binary encoding. No schema files, no code generation, no external deps. 2-4x faster than JSON for encoding/decoding complex structs. Compact binary format. Perfect for a Go-only educational project where cross-language interop is not needed. HIGH confidence. |
| `encoding/json` (stdlib) | Go 1.23+ | REST API responses (dashboard), debugging output, human-readable inspection | JSON for anything a human or browser needs to read. Stdlib, zero deps. HIGH confidence. |
| `encoding/hex` (stdlib) | Go 1.23+ | Hash/address display | Hex encoding for displaying hashes and binary data. Stdlib. |

**Why NOT protobuf:** Requires .proto schema files, `protoc` compiler, code generation step. Massive overkill for an educational project where all nodes are Go. Protobuf's cross-language and backward-compatibility advantages are irrelevant here.

**Why NOT msgpack:** External dependency for marginal benefit over gob in a Go-only system.

**Why gob works here:** All nodes run the same Go code. gob handles Go structs natively with zero configuration. When a block is mined, gob-serialize it, store in bbolt, and transmit to peers. Simple.

### Networking (P2P Layer)

| Technology | Version | Purpose | Why |
|------------|---------|---------|-----|
| `net` (stdlib) | Go 1.23+ | TCP connections between nodes | Raw TCP using Go's `net` package. Educational project runs on localhost -- no NAT traversal, no peer discovery protocols, no encryption needed between local nodes. A custom protocol over TCP teaches how blockchain P2P actually works at the wire level. HIGH confidence. |
| `encoding/gob` (stdlib) | Go 1.23+ | Message framing over TCP | Use gob encoder/decoder on TCP connections for typed message passing. Each message is a Go struct (e.g., `MsgVersion`, `MsgBlock`, `MsgTx`, `MsgGetBlocks`). Encoder handles framing automatically. |

**Why NOT libp2p:** libp2p is a full networking stack (peer discovery, NAT traversal, muxing, encryption). For localhost-only educational P2P, it hides the very concepts you want to learn. It adds massive dependency weight. The project goal is understanding how blockchain networking works -- raw TCP makes the protocol visible.

**Why NOT net/http / REST:** HTTP request-response doesn't model blockchain P2P well. Nodes need persistent connections, push-based communication (broadcast blocks/txs), and bidirectional messaging. TCP streams are the right abstraction.

**Why raw TCP:** Bitcoin's actual protocol runs over TCP with a custom binary protocol. Building a simplified version teaches the same concepts: message types, handshakes, version negotiation, inventory management. Go's `net` package + goroutines make this clean and idiomatic.

### CLI Framework

| Technology | Version | Purpose | Why |
|------------|---------|---------|-----|
| `github.com/spf13/cobra` | v1.10.2 | CLI command structure | Industry standard. Used by kubectl, docker, hugo, gh. Subcommand pattern (`shitcoin node start`, `shitcoin wallet create`, `shitcoin tx send`) maps perfectly to blockchain operations. Auto-generated help, shell completion. 184,000+ importers. HIGH confidence. |
| `github.com/spf13/viper` | latest | Configuration management | Cobra's companion. Reads config from files, env vars, flags. Useful for node configuration (port, data dir, peers, mining settings). |

**Why NOT urfave/cli:** Cobra has wider adoption, better subcommand support, and the cobra-cli scaffolding tool. Both work fine, but cobra is the stronger ecosystem choice.

**Why NOT stdlib flag:** No subcommand support. A blockchain CLI naturally has many commands (node, wallet, tx, block) -- flag package would require manual subcommand routing.

### Web Dashboard

| Technology | Version | Purpose | Why |
|------------|---------|---------|-----|
| `github.com/labstack/echo/v4` | v4.15.1 | HTTP server for web dashboard | Lightweight, fast, well-documented. Built-in middleware (CORS, logging, recovery). Clean routing API. v4 supported with security updates through 2026-12-31. Simpler than Fiber (which uses Fasthttp with different semantics than net/http). HIGH confidence. |
| `github.com/a-h/templ` | v0.3.x | Type-safe HTML templating | Compiles to Go code -- type-safe templates with IDE autocompletion. Catches template errors at compile time, not runtime. JSX-like syntax is intuitive. Much better DX than `html/template`. MEDIUM confidence (newer library, but rapidly adopted). |
| HTMX | 2.x (CDN) | Dynamic frontend without JavaScript framework | Serves HTML fragments from Go, HTMX swaps them into the DOM. Perfect for a dashboard that updates block height, mempool, peer list. No Node.js toolchain, no npm, no JavaScript build step. Keeps the project pure Go + HTML. HIGH confidence. |
| Tailwind CSS | 3.x (CDN) | Dashboard styling | Use CDN version (no build step). Clean utility classes for a functional dashboard. Not a core dependency -- CDN link in HTML head is sufficient for an educational project. |

**Why NOT React/Vue/Svelte:** Requires a JavaScript toolchain, package manager, build pipeline. This is a Go learning project -- the dashboard should be a thin HTML view over the blockchain data, not a frontend engineering project.

**Why NOT html/template (stdlib):** No type safety, runtime panics on template errors, poor IDE support, verbose syntax. templ is strictly better for anything beyond trivial templates.

**Why Echo over net/http (stdlib):** For the dashboard, you want routing, middleware (CORS for API), static file serving, and clean JSON responses. Echo provides this with minimal overhead. Using raw net/http would mean re-implementing these basics.

**Why Echo over Fiber:** Fiber uses Fasthttp (not net/http compatible). Echo builds on net/http, so all standard middleware and patterns work. Echo's API is cleaner for a project this size.

### Testing

| Technology | Version | Purpose | Why |
|------------|---------|---------|-----|
| `testing` (stdlib) | Go 1.23+ | Unit and integration tests | Go standard. No reason to use anything else for test harness. |
| `github.com/stretchr/testify` | v1.10.x | Test assertions and mocking | `assert` and `require` packages make tests readable. `mock` package useful for isolating components. Industry standard. HIGH confidence. |

### Development Tools

| Technology | Version | Purpose | Why |
|------------|---------|---------|-----|
| `golangci-lint` | latest | Linting | Aggregates 100+ linters. Catches bugs, enforces style. |
| `github.com/air-verse/air` | latest | Hot reload during development | Auto-rebuilds and restarts on file changes. Essential for dashboard development. |

## Alternatives Considered

| Category | Recommended | Alternative | Why Not |
|----------|-------------|-------------|---------|
| Crypto (secp256k1) | btcec/v2 | decred/dcrd/dcrec/secp256k1/v4 | btcec/v2 already wraps Decred internally. btcec provides Bitcoin-specific ECDSA helpers on top. Use the higher-level API. |
| Crypto (secp256k1) | btcec/v2 | go-ethereum/crypto | Pulls in entire go-ethereum dependency tree. Massive for one function. |
| Storage | bbolt | BadgerDB v4.9.1 | Higher memory (problem with multi-node localhost), no buckets, more complex. |
| Storage | bbolt | SQLite (modernc.org) | KV store maps better to blockchain patterns. SQL is unnecessary abstraction. |
| Serialization | encoding/gob | protobuf | Requires .proto files, codegen, protoc toolchain. Overkill for Go-only project. |
| Serialization | encoding/gob | msgpack | External dep for marginal gain in Go-only system. |
| Networking | raw TCP (net) | libp2p | Hides the concepts you're trying to learn. Massive dependency. |
| Networking | raw TCP (net) | gRPC | Requires protobuf. Request-response model doesn't fit P2P broadcasting well. |
| CLI | cobra | urfave/cli | Smaller ecosystem, weaker subcommand patterns. |
| CLI | cobra | stdlib flag | No subcommand support. |
| Web framework | echo v4 | fiber | Fasthttp-based (not net/http compatible). Less standard. |
| Web framework | echo v4 | gin | Similar capability but echo has cleaner API and better docs. |
| Web framework | echo v4 | net/http (stdlib) | Would need to re-implement routing, middleware, etc. |
| Templating | templ | html/template | No type safety, runtime errors, poor DX. |
| Templating | templ | React/Vue/Svelte | Requires JS toolchain. Wrong complexity for this project. |

## Full Dependency List

```bash
# Initialize module
go mod init github.com/baotoq/shitcoin

# Core dependencies
go get github.com/btcsuite/btcd/btcec/v2          # secp256k1 ECDSA
go get github.com/btcsuite/btcd/btcutil            # Base58 encoding
go get golang.org/x/crypto                          # RIPEMD-160
go get go.etcd.io/bbolt                             # Embedded KV store
go get github.com/spf13/cobra                       # CLI framework
go get github.com/spf13/viper                       # Configuration
go get github.com/labstack/echo/v4                  # Web server
go get github.com/a-h/templ                         # HTML templating

# Dev dependencies
go get github.com/stretchr/testify                  # Test assertions

# Tools (install separately)
go install github.com/a-h/templ/cmd/templ@latest    # templ CLI
go install github.com/air-verse/air@latest           # Hot reload
```

**Total external dependencies: 8** (plus transitive). This is intentionally minimal -- the project leans heavily on Go's stdlib for crypto (sha256, rand), networking (net), serialization (gob, json), and testing.

## Dependency Map by Feature

```
Wallet/Keys:     btcec/v2 + x/crypto + crypto/sha256 + crypto/rand
Block Mining:    crypto/sha256 (stdlib only)
Merkle Trees:    crypto/sha256 (stdlib only, implement from scratch)
Storage:         bbolt + encoding/gob
P2P Network:     net + encoding/gob (stdlib only)
CLI:             cobra + viper
Dashboard:       echo + templ + htmx (CDN)
Serialization:   encoding/gob + encoding/json (stdlib only)
Testing:         testing + testify
```

## What to Implement From Scratch (No Library)

These components should be hand-written for educational value:

| Component | Why Build It | Approach |
|-----------|-------------|----------|
| Merkle tree | Core blockchain concept to understand | Binary tree of SHA-256 hashes using `crypto/sha256` |
| PoW algorithm | Central to mining understanding | SHA-256 double-hash with difficulty target comparison |
| UTXO set management | Core transaction model | bbolt bucket with custom indexing |
| P2P message protocol | Networking is a learning goal | Custom message types over TCP with gob encoding |
| Address derivation | Wallet internals are a learning goal | SHA-256 -> RIPEMD-160 -> Base58Check pipeline |
| Transaction validation | Consensus rule enforcement | Custom validation logic, no framework |
| Block validation | Consensus rule enforcement | Custom validation logic, no framework |

## Sources

- [Go Cryptography State of the Union 2025](https://words.filippo.io/2025-state/) -- Go crypto ecosystem overview
- [btcec/v2 on pkg.go.dev](https://pkg.go.dev/github.com/btcsuite/btcd/btcec/v2) -- v2.3.6, published Oct 2025
- [bbolt on pkg.go.dev](https://pkg.go.dev/go.etcd.io/bbolt) -- v1.4.3, published Aug 2025
- [BadgerDB on pkg.go.dev](https://pkg.go.dev/github.com/dgraph-io/badger/v4) -- v4.9.1, published Feb 2026
- [BoltDB vs Badger comparison](https://tech.townsourced.com/post/boltdb-vs-badger/) -- Memory and performance tradeoffs
- [Badger vs LMDB vs BoltDB benchmarks](https://hypermode.com/blog/badger-lmdb-boltdb/) -- Independent benchmark comparison
- [Cobra on pkg.go.dev](https://pkg.go.dev/github.com/spf13/cobra) -- v1.10.2, published Dec 2025
- [Echo v4 on pkg.go.dev](https://pkg.go.dev/github.com/labstack/echo/v4) -- v4.15.1, published Feb 2026
- [templ on pkg.go.dev](https://pkg.go.dev/github.com/a-h/templ) -- v0.3.x, published Feb 2026
- [Go crypto/ecdsa docs](https://pkg.go.dev/crypto/ecdsa) -- NIST curves only, no secp256k1
- [Jeiwan/blockchain_go](https://github.com/Jeiwan/blockchain_go) -- Canonical Go blockchain tutorial (uses BoltDB, stdlib crypto)
- [JSON vs GOB benchmarks](https://blog.vitalvas.com/post/2025/07/23/json-vs-gob-in-golang/) -- gob 2-4x faster for complex structs
- [Go serialization formats](https://rotational.io/blog/go-serialization-formats/) -- Format comparison and recommendations
- [Cobra CLI best practices 2025](https://www.glukhov.org/post/2025/11/go-cli-applications-with-cobra-and-viper/) -- Cobra + Viper patterns
