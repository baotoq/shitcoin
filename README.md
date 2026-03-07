# Shitcoin

A full blockchain implementation in Go that replicates Bitcoin's core mechanics for educational purposes. Implements Proof of Work mining, UTXO-based transactions, P2P networking between local nodes, and wallet/key management — all simplified to focus on understanding how blockchains actually work. Includes a real-time block explorer web UI.

## Key Features

- **Proof of Work mining** with adjustable difficulty and automatic difficulty adjustment
- **UTXO transaction model** with inputs, outputs, coinbase rewards, and double-spend detection
- **P2P networking** between multiple local nodes with version handshake, inventory-based relay, and initial block download (IBD)
- **Wallet management** with secp256k1 key generation and Bitcoin-style Base58Check P2PKH addresses
- **Chain reorganization** for fork resolution when peers have divergent chains
- **Persistent storage** using BoltDB for blocks/UTXOs and JSON files for wallets
- **CLI interface** for all node operations: mining, sending coins, checking balances, running nodes
- **Block explorer web UI** with real-time WebSocket updates for mining visualization, block browsing, and transaction inspection
- **REST API** for querying chain status, blocks, transactions, mempool, and addresses
- **Local testnet** launcher that spawns a multi-node network with a single command

## Table of Contents

- [Tech Stack](#tech-stack)
- [Prerequisites](#prerequisites)
- [Getting Started](#getting-started)
- [CLI Commands](#cli-commands)
- [Web Explorer](#web-explorer)
- [REST API](#rest-api)
- [Running a Multi-Node Network](#running-a-multi-node-network)
- [Architecture](#architecture)
- [Configuration](#configuration)
- [Testing](#testing)
- [How It Works](#how-it-works)

## Tech Stack

**Backend:**
- **Language**: Go 1.26+
- **Framework**: [go-zero](https://github.com/zeromicro/go-zero) (config loading, HTTP server, service context pattern)
- **Storage**: [bbolt](https://github.com/etcd-io/bbolt) (embedded key-value store for blocks and UTXOs)
- **Cryptography**: [btcec](https://github.com/btcsuite/btcd/tree/master/btcec) (secp256k1 ECDSA), SHA-256, RIPEMD-160
- **Networking**: Raw TCP with custom length-prefixed binary protocol

**Frontend (Block Explorer):**
- **Framework**: React 19 with TypeScript
- **Build Tool**: Vite 7
- **Styling**: Tailwind CSS 4 with shadcn/ui components
- **Routing**: React Router 7
- **Real-time**: WebSocket for live mining updates and event streaming

## Prerequisites

- **Go 1.26** or higher
- **Node.js 20+** and **npm** (only if running the web explorer)

No other external dependencies required — storage is embedded (BoltDB) and networking is localhost TCP.

## Getting Started

### 1. Clone the Repository

```bash
git clone https://github.com/baotoq/shitcoin.git
cd shitcoin
```

### 2. Install Go Dependencies

```bash
go mod download
```

### 3. Create a Wallet

```bash
go run cmd/shitcoin/main.go createwallet
```

This outputs a new Base58Check address like `1A1zP1eP5QGefi2DMPTfTL5SLmv7DivfNa`. The wallet (private key) is persisted to `data/wallets.json`.

### 4. Mine the Genesis Block

```bash
go run cmd/shitcoin/main.go mine -address YOUR_ADDRESS
```

This initializes the chain with a genesis block, then mines a new block on top of it. The miner receives a coinbase reward (default: 50 coins = 5,000,000,000 satoshis).

### 5. Check Your Balance

```bash
go run cmd/shitcoin/main.go getbalance -address YOUR_ADDRESS
```

### 6. Send Coins

First create a second wallet, then send:

```bash
go run cmd/shitcoin/main.go createwallet
go run cmd/shitcoin/main.go send -from SENDER_ADDRESS -to RECEIVER_ADDRESS -amount 1000000000
```

Amounts are in **satoshis** (1 coin = 100,000,000 satoshis). You can optionally attach a fee with `-fee SATOSHIS`. This adds the transaction to the mempool. Mine a block to confirm it:

```bash
go run cmd/shitcoin/main.go mine -address SENDER_ADDRESS
```

### 7. View the Chain

```bash
go run cmd/shitcoin/main.go printchain
```

## CLI Commands

All commands are run as subcommands to the main binary. Use `-f` to specify a config file (default: `etc/shitcoin.yaml`).

```bash
go run cmd/shitcoin/main.go [-f config.yaml] <command> [flags]
```

| Command | Description | Flags |
|---------|-------------|-------|
| `createwallet` | Generate a new secp256k1 wallet and save it | — |
| `listaddresses` | List all stored wallet addresses | — |
| `getbalance` | Get balance for an address | `-address ADDR` |
| `send` | Create and sign a transaction, add to mempool | `-from ADDR -to ADDR -amount SATOSHIS [-fee SATOSHIS]` |
| `mine` | Drain mempool and mine a new block | `-address ADDR` (miner reward address) |
| `startnode` | Start a P2P node with optional auto-mining | `-port PORT -http-port PORT -mine ADDR -peers HOST:PORT,... -datadir DIR` |
| `printchain` | Print all blocks in the chain | — |
| `testnet` | Launch a local multi-node testnet | `-nodes N -base-port PORT -base-http-port PORT -config PATH` |
| `demo doublespend` | Run a scripted double-spend attack demo | — |

## Web Explorer

The block explorer is a React single-page application that connects to a running node's HTTP/WebSocket server.

### Setup

```bash
cd web
npm install
npm run dev
```

The dev server starts on `http://localhost:5173` and proxies API/WebSocket requests to the backend at `http://localhost:8080`.

Make sure a node is running with the HTTP server:

```bash
# In another terminal
go run cmd/shitcoin/main.go startnode -port 3000 -http-port 8080 -mine YOUR_ADDRESS
```

### Pages

| Route | Page | Description |
|-------|------|-------------|
| `/` | Dashboard | Chain status, latest blocks, real-time event feed |
| `/blocks` | Block Explorer | Paginated block list |
| `/blocks/:height` | Block Detail | Block header, transactions, and hashes |
| `/tx/:hash` | Transaction Detail | Inputs, outputs, and values |
| `/mempool` | Mempool | Pending unconfirmed transactions |
| `/mining` | Mining | Real-time mining visualization (nonce attempts, hash vs target) |
| `/address/:addr` | Address | Balance and transaction history for an address |

### WebSocket Events

The web UI connects to `/ws` for real-time updates:

| Event | Description |
|-------|-------------|
| `new_block` | A new block was mined or received |
| `new_tx` | A new transaction entered the mempool |
| `mining_started` | Mining began on a new block |
| `mining_progress` | Hash attempt update (nonce, hash, target) |
| `mining_stopped` | Mining finished (found, cancelled, or error) |
| `peer_connected` | A P2P peer connected |
| `peer_disconnected` | A P2P peer disconnected |
| `mempool_changed` | Mempool transaction count changed |
| `reorg` | Chain reorganization occurred |

### Production Build

```bash
cd web
npm run build
```

Output goes to `web/dist/`. Serve these static files behind a reverse proxy that forwards `/api/*` and `/ws` to the Go backend.

## REST API

When a node is running via `startnode`, the HTTP server exposes these endpoints:

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/api/status` | Chain height, latest block hash, mempool size, peer count |
| `GET` | `/api/blocks?page=1&limit=20` | Paginated block list (newest first) |
| `GET` | `/api/blocks/:height` | Block by height |
| `GET` | `/api/blocks/hash/:hash` | Block by hash |
| `GET` | `/api/tx/:hash` | Transaction by hash |
| `GET` | `/api/mempool` | All pending mempool transactions |
| `GET` | `/api/address/:addr` | Balance and UTXOs for an address |
| `GET` | `/api/search?q=QUERY` | Search by block height, block hash, or tx hash |
| `GET` | `/ws` | WebSocket upgrade for real-time events |

## Running a Multi-Node Network

### Quick Start: Testnet Command

The easiest way to run a multi-node network:

```bash
go run cmd/shitcoin/main.go testnet -nodes 3 -base-port 3000 -base-http-port 8080
```

This spawns 3 nodes:
- **Node 0** (P2P: 3000, HTTP: 8080): seed node + auto-miner
- **Node 1** (P2P: 3001, HTTP: 8081): connects to node 0
- **Node 2** (P2P: 3002, HTTP: 8082): connects to node 0

Press Ctrl+C to stop all nodes. Each node's output is prefixed with `[node-N]`.

### Manual Setup

Each node gets its own data directory (database + wallets) isolated by port number.

**Terminal 1: Start Node A (miner)**

```bash
go run cmd/shitcoin/main.go startnode -port 3000 -mine ADDRESS_A
```

This creates `data/node-3000/` with its own database and wallet files, then auto-mines blocks continuously.

**Terminal 2: Start Node B (connects to A)**

```bash
go run cmd/shitcoin/main.go startnode -port 3001 -peers localhost:3000 -mine ADDRESS_B
```

Node B connects to Node A, performs initial block download (IBD) to sync the chain, then begins mining. When either node mines a block, it broadcasts to the other via the P2P network.

**Terminal 3: Start Node C (idle observer)**

```bash
go run cmd/shitcoin/main.go startnode -port 3002 -peers localhost:3000,localhost:3001
```

Node C syncs the chain from peers but doesn't mine (no `-mine` flag). Press Ctrl+C to stop any node.

## Architecture

### Directory Structure

```
shitcoin/
├── cmd/shitcoin/
│   └── main.go                  # Entry point: config -> ServiceContext -> CLI
├── etc/
│   └── shitcoin.yaml            # Default configuration
├── internal/
│   ├── config/
│   │   └── config.go            # Config structs (go-zero json tags)
│   ├── svc/
│   │   └── service_context.go   # Dependency injection container
│   ├── handler/
│   │   ├── cli/
│   │   │   ├── cli.go           # CLI command dispatch and handlers
│   │   │   ├── signal.go        # Auto-mining loops and signal handling
│   │   │   ├── testnet.go       # Multi-node testnet launcher
│   │   │   └── demo.go          # Educational demo scenarios
│   │   ├── api/
│   │   │   ├── routes.go        # REST API route registration
│   │   │   ├── status_handler.go
│   │   │   ├── block_handler.go
│   │   │   ├── tx_handler.go
│   │   │   ├── mempool_handler.go
│   │   │   ├── address_handler.go
│   │   │   └── search_handler.go
│   │   └── ws/
│   │       ├── hub.go           # WebSocket hub (fan-out to clients)
│   │       ├── client.go        # WebSocket client connection
│   │       ├── handler.go       # HTTP upgrade handler
│   │       └── events.go        # WebSocket event payload types
│   ├── domain/
│   │   ├── block/               # Block entity, Header, PoW, Merkle tree, difficulty
│   │   ├── chain/               # Chain aggregate root (mining orchestration, reorgs)
│   │   ├── tx/                  # Transaction entity, inputs/outputs, signing
│   │   ├── utxo/                # UTXO Set with apply/undo, double-spend detection
│   │   ├── wallet/              # Wallet entity, secp256k1 keys, Base58Check addresses
│   │   ├── mempool/             # In-memory transaction pool with validation
│   │   ├── p2p/                 # TCP P2P server, protocol, message types, sync/IBD
│   │   └── events/              # Pub/sub event bus for domain events
│   └── infrastructure/
│       └── persistence/
│           ├── bbolt/           # BoltDB repos for chain and UTXO storage
│           └── jsonfile/        # JSON file repo for wallet storage
├── web/                         # Block explorer frontend (React + Vite)
│   ├── src/
│   │   ├── pages/               # Dashboard, BlockExplorer, Mining, Mempool, etc.
│   │   ├── components/          # Layout, BlockCard, StatusBar, MiningVisualizer
│   │   ├── hooks/               # useWebSocket, useNodeStatus
│   │   └── lib/                 # API client, utilities
│   └── vite.config.ts           # Dev server with API/WS proxy to backend
└── data/                        # Runtime data (created automatically)
    ├── shitcoin.db              # BoltDB database (default single-node)
    ├── wallets.json             # Wallet keys (default single-node)
    └── node-{port}/             # Per-node data directories (startnode)
```

### Request Flow

```
main.go
  -> config.Config (loaded via go-zero conf.MustLoad)
  -> svc.NewServiceContext (opens BoltDB, creates repos, wires Chain aggregate)
  -> cli.New(serviceCtx).Run(args)
    -> command handler (mine, send, startnode, etc.)
      -> domain logic (chain.MineBlock, mempool.Add, p2p.Server)
        -> infrastructure (bbolt repos, jsonfile repos)
```

When running `startnode`, the node also starts:
- **HTTP server** (go-zero `rest.Server`) serving the REST API
- **WebSocket hub** forwarding domain events to connected browser clients
- **P2P server** for inter-node communication

### Domain Model

The codebase follows **Domain-Driven Design** tactical patterns:

- **Entities** (identity-based, pointer receiver): `Block`, `Transaction`, `Wallet`, `UTXO`
- **Value Objects** (immutable): `Header`, `Hash`, `TxInput`, `TxOutput`
- **Aggregate Roots**: `Chain` (block sequence + mining), `Set` (UTXO tracking)
- **Domain Services**: `ProofOfWork` (stateless mining/validation)
- **Repositories** (interfaces in domain, implementations in infrastructure): `chain.Repository`, `utxo.Repository`, `wallet.Repository`
- **Event Bus**: Pub/sub `events.Bus` decouples domain events (new block, mining progress) from WebSocket delivery

### P2P Protocol

Custom TCP protocol with length-prefixed framing:

```
Wire format: [4-byte big-endian length][1-byte command][JSON payload]
```

| Command | Byte | Description |
|---------|------|-------------|
| Version | `0x01` | Handshake with chain height and genesis hash |
| Verack | `0x02` | Handshake acknowledgment |
| GetBlocks | `0x03` | Request block range (IBD) |
| Inv | `0x04` | Announce block/tx hashes |
| GetData | `0x05` | Request full block/tx by hash |
| Block | `0x06` | Full block payload |
| Tx | `0x07` | Full transaction payload |

**Handshake flow** (outbound): Send Version -> Receive Version -> Verify genesis hash -> Send Verack -> Receive Verack

**Initial Block Download (IBD)**: After handshake, if a peer has a higher chain height, the node requests blocks from its tip+1 to the peer's height. Includes fork detection and chain reorganization.

**Relay**: Blocks and transactions are broadcast via inventory messages (inv). Seen-hash tracking prevents infinite relay loops.

## Configuration

Configuration file: `etc/shitcoin.yaml`

```yaml
Name: shitcoin
Host: 0.0.0.0
Port: 8080

Consensus:
  BlockTimeTarget: 1          # Target seconds between blocks
  DifficultyAdjustInterval: 10 # Blocks between difficulty adjustments
  InitialDifficulty: 5        # Leading zero bits in block hash
  GenesisMessage: "Hello, Shitcoin!"
  HalvingInterval: 10         # Blocks between reward halvings (0 = disabled)

Storage:
  DBPath: data/shitcoin.db
  WalletPath: data/wallets.json

P2P:
  Port: 3000                   # TCP port for P2P server
  Peers: ""                    # Comma-separated seed peers (host:port)
```

| Parameter | Description | Default |
|-----------|-------------|---------|
| `Consensus.BlockTimeTarget` | Target seconds between blocks | `10` |
| `Consensus.DifficultyAdjustInterval` | Blocks between difficulty adjustments | `10` |
| `Consensus.InitialDifficulty` | Initial number of leading zero bits required in block hash | `16` |
| `Consensus.GenesisMessage` | Message embedded in genesis block | `"The Times 03/Jan/2009..."` |
| `Consensus.BlockReward` | Coinbase reward in satoshis | `5000000000` (50 coins) |
| `Consensus.HalvingInterval` | Blocks between reward halvings (0 = no halving) | `210000` |
| `Consensus.MaxBlockTxs` | Max non-coinbase transactions per block (0 = unlimited) | `100` |
| `Storage.DBPath` | BoltDB database file path | `data/shitcoin.db` |
| `Storage.WalletPath` | Wallet JSON file path | `data/wallets.json` |
| `P2P.Port` | TCP listen port for P2P | `3000` |
| `P2P.Peers` | Comma-separated seed peer addresses | (empty) |

**Note**: go-zero uses `json` struct tags for all config formats (YAML, JSON, TOML), not `yaml` tags.

## Testing

### Run All Tests

```bash
go test ./...
```

### Run Tests for a Specific Package

```bash
go test ./internal/domain/block/
go test ./internal/domain/chain/
go test ./internal/domain/tx/
go test ./internal/domain/utxo/
go test ./internal/domain/wallet/
go test ./internal/domain/mempool/
go test ./internal/domain/p2p/
go test ./internal/infrastructure/persistence/bbolt/
```

### Run a Specific Test

```bash
go test ./internal/domain/block/ -run TestMerkleRoot
go test -v ./internal/domain/p2p/ -run TestHandshake
```

### Run Tests with Race Detection

```bash
go test -race ./...
```

Tests use standard Go testing with table-driven test patterns. P2P tests use `net.Pipe()` for in-memory connection simulation. No external test frameworks or test databases required.

## How It Works

### Mining a Block

1. **Coinbase transaction** created: miner address receives block reward
2. **Mempool drained**: all pending transactions collected, sorted by fee
3. **Merkle root** computed from all transaction hashes
4. **Block header** assembled: version, prev hash, merkle root, timestamp, difficulty bits
5. **Proof of Work**: nonce incremented from 0 until `SHA256(SHA256(header)) < target`
6. **UTXO set updated**: spent UTXOs removed, new UTXOs added (atomically with block save)
7. **Block persisted** to BoltDB
8. **Events published**: `new_block` event broadcast to WebSocket clients and P2P peers

### Transaction Flow

1. Sender's wallet loaded from JSON file (secp256k1 private key)
2. UTXOs for sender address queried (greedy selection to cover amount + fee)
3. Transaction built with inputs (UTXO references) and outputs (recipient + change)
4. Transaction signed with ECDSA (secp256k1)
5. Transaction added to mempool (validated: signature, UTXO existence, no double-spend)
6. Broadcast to P2P peers via inventory relay
7. Included in next mined block

### Address Derivation

Follows Bitcoin's P2PKH scheme:

```
secp256k1 private key
  -> compressed public key (33 bytes)
  -> SHA-256
  -> RIPEMD-160 (20-byte public key hash)
  -> Base58Check encode with version byte 0x00
  -> address (e.g., "1A1zP1...")
```

### Difficulty Adjustment

Every `DifficultyAdjustInterval` blocks, the difficulty adjusts based on actual vs. target time span:

- If blocks are too fast: difficulty increases (more leading zero bits)
- If blocks are too slow: difficulty decreases (fewer leading zero bits)

The difficulty target is `2^(256 - bits)` — a lower target means higher difficulty.

### Double-Spend Prevention

The blockchain prevents double-spending through two layers:

1. **Mempool**: Tracks which UTXOs are already claimed by pending transactions. A second transaction referencing the same UTXO is rejected immediately.
2. **UTXO Set**: Only unspent outputs can be used as inputs. Once a block confirms a transaction, those UTXOs are consumed and cannot be reused.

Run `demo doublespend` to see this in action with a scripted scenario.
