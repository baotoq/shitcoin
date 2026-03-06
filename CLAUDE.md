# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Educational blockchain implementation in Go that replicates Bitcoin's core mechanics: Proof of Work mining, UTXO-based transactions, P2P networking (localhost), and wallet/key management. Uses a CLI interface for all node operations.

## Build & Run Commands

```bash
# Run the node
go run cmd/shitcoin/main.go -f etc/shitcoin.yaml <subcommand> [flags]

# Subcommands:
#   createwallet
#   listaddresses
#   getbalance -address ADDR
#   send -from ADDR -to ADDR -amount AMOUNT
#   mine -address ADDR
#   startnode [-port PORT] [-mine ADDR] [-peers HOST:PORT,...] [-datadir DIR]
#   printchain

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

### Infrastructure Layer (`internal/infrastructure/persistence/`)

- **bbolt**: BoltDB-backed repositories for chain (blocks) and UTXO set. Atomic block+UTXO saves.
- **jsonfile**: JSON file-backed wallet repository.

### Key Design Decisions

- Block transactions use `[]any` (not `[]*tx.Transaction`) to break import cycles between `block` and `tx` packages. Type assertions happen at the chain/handler level.
- P2P wire format: `[4-byte big-endian length][1-byte command byte][JSON payload]`. Commands defined as byte constants in `p2p/message.go`.
- Per-node data isolation: `startnode` creates separate DB/wallet files under `data/node-{port}/`.
- Config uses go-zero's `conf.MustLoad` with `json` struct tags (not `yaml` tags) for all config formats.

