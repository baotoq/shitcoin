# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Educational blockchain implementation in Go that replicates Bitcoin's core mechanics: Proof of Work mining, UTXO-based transactions, P2P networking (localhost), and wallet/key management. CLI for node operations, REST API + WebSocket for real-time data, and a React block explorer frontend (`web/`).

## Build & Run Commands

```bash
# Run the node
go run cmd/shitcoin/main.go -f etc/shitcoin.yaml <subcommand> [flags]

# Subcommands:
#   createwallet
#   listaddresses
#   getbalance -address ADDR
#   send -from ADDR -to ADDR -amount AMOUNT [-fee FEE]
#   mine -address ADDR
#   startnode [-port PORT] [-http-port PORT] [-mine ADDR] [-peers HOST:PORT,...] [-datadir DIR]
#   printchain
#   testnet [-nodes N] [-base-port PORT] [-base-http-port PORT]
#   demo doublespend

# Run all tests
go test ./...

# Run a single test
go test ./internal/domain/block/ -run TestMerkleRoot

# Run tests with verbose output
go test -v ./internal/domain/chain/...
```

## Architecture

### Domain-Driven Design with Go-Zero

The project follows DDD tactical patterns with go-zero as the framework for config loading (using `json` struct tags for all formats including YAML).

**Entry point**: `cmd/shitcoin/main.go` → loads config → creates `svc.ServiceContext` → dispatches to `handler/cli`

**ServiceContext** (`internal/svc/service_context.go`): Central dependency injection container wiring all repositories, domain aggregates, and infrastructure. Follows go-zero's ServiceContext pattern.

### Domain Layer (`internal/domain/`)

- **block**: Block entity (aggregate root), Header value object, PoW mining, Merkle tree, difficulty adjustment. Transactions stored as `[]any` to avoid circular imports with `tx` package.
- **chain**: Chain aggregate root managing block sequence, mining orchestration, and difficulty. Uses `Repository` interface for persistence.
- **tx**: Transaction entity, TxInput/TxOutput value objects, coinbase creation, ECDSA signing/verification.
- **utxo**: UTXO Set with apply/rollback support via undo entries. Repository interface for persistence.
- **wallet**: Wallet entity with secp256k1 key generation, Base58Check address encoding. Repository interface for persistence.
- **mempool**: In-memory transaction pool with UTXO-based validation.
- **p2p**: TCP-based P2P networking with length-prefixed binary protocol (`[4-byte length][1-byte command][JSON payload]`). Implements version handshake, inventory-based relay (inv/getdata), and initial block download (IBD) sync.

### Handler Layer (`internal/handler/`)

- **cli**: CLI command dispatch (`cli.go`), auto-mining (`signal.go`), multi-node testnet launcher (`testnet.go`), educational demos (`demo.go`).
- **api**: REST API handlers registered in `routes.go`. Endpoints: `/api/status`, `/api/blocks`, `/api/blocks/:height`, `/api/blocks/hash/:hash`, `/api/tx/:hash`, `/api/mempool`, `/api/address/:addr`, `/api/search`.
- **ws**: WebSocket hub (`hub.go`) that fans domain events out to connected browser clients. Events: `new_block`, `mining_progress`, `mining_started`, `mining_stopped`, `peer_connected`, `peer_disconnected`, `mempool_changed`, `reorg`.

### Event Bus (`internal/domain/events/`)

Pub/sub `events.Bus` decouples domain events from WebSocket delivery. Domain code publishes events; the WebSocket hub subscribes and forwards to clients.

### Infrastructure Layer (`internal/infrastructure/persistence/`)

- **bbolt**: BoltDB-backed repositories for chain (blocks) and UTXO set. Atomic block+UTXO saves.
- **jsonfile**: JSON file-backed wallet repository.

### Key Design Decisions

- Block transactions use `[]any` (not `[]*tx.Transaction`) to break import cycles between `block` and `tx` packages. Type assertions happen at the chain/handler level.
- P2P wire format: `[4-byte big-endian length][1-byte command byte][JSON payload]`. Commands defined as byte constants in `p2p/message.go`.
- Per-node data isolation: `startnode` creates separate DB/wallet files under `data/node-{port}/`.
- Config uses go-zero's `conf.MustLoad` with `json` struct tags (not `yaml` tags) for all config formats.
- Domain events flow through `events.Bus` (pub/sub) → `ws.Hub` (WebSocket fan-out) → browser clients. This decouples domain logic from transport.
- Web frontend dev server (Vite `:5173`) proxies `/api` and `/ws` to Go backend (`:8080`). The proxy config is in `web/vite.config.ts`.

### Web Frontend (`web/`)

React 19 + Vite 7 + Tailwind CSS 4 + shadcn/ui block explorer. Dev server on `:5173` proxies `/api` and `/ws` to the Go backend on `:8080` (configured in `vite.config.ts`).

```bash
# Start web dev server (requires a running node on :8080)
cd web && npm install && npm run dev
```

