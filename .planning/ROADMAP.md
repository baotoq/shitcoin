# Roadmap: Shitcoin

## Overview

This roadmap takes the project from zero to a working multi-node blockchain with web visualization. The build order follows hard dependency chains: deterministic block hashing must exist before transactions, transactions before mempool, a working CLI-exercisable local chain before P2P complexity, and a functioning distributed system before the dashboard visualizes it. Each phase delivers a coherent, independently verifiable capability.

## Phases

**Phase Numbering:**
- Integer phases (1, 2, 3): Planned milestone work
- Decimal phases (2.1, 2.2): Urgent insertions (marked with INSERTED)

Decimal phases appear between their surrounding integers in numeric order.

- [ ] **Phase 1: Core Chain Foundation** - Block structure, SHA-256d hashing, PoW mining, difficulty adjustment, and persistent storage
- [ ] **Phase 2: Wallets and Transactions** - ECDSA keys, Bitcoin-style addresses, UTXO transaction model with signing and reversible UTXO set
- [ ] **Phase 3: Mempool, Mining Integration, and CLI** - Transaction mempool, Merkle tree, full mining pipeline, and complete CLI for exercising all local functionality
- [ ] **Phase 4: P2P Networking and Consensus** - TCP peer connections, block/transaction broadcasting, chain synchronization, and fork resolution
- [ ] **Phase 5: Web Dashboard** - Block explorer, node status panel, real-time mining visualization, and mempool view
- [ ] **Phase 6: Advanced Educational Features** - Block reward halving, transaction fees, multi-node orchestration, and double-spend demo

## Phase Details

### Phase 1: Core Chain Foundation
**Goal**: A node can create, mine, and persist blocks with correct deterministic hashing and adjustable difficulty
**Depends on**: Nothing (first phase)
**Requirements**: MINE-01, MINE-02, MINE-03, MINE-06, MINE-09
**Success Criteria** (what must be TRUE):
  1. Running the program creates a genesis block with a custom embedded message and persists it to disk
  2. Mining produces a block whose SHA-256 double-hash meets the current difficulty target
  3. Restarting the node loads the previously mined chain from disk without data loss
  4. After N blocks are mined, the difficulty target visibly adjusts based on actual vs target block time
  5. Consensus parameters (block time target, difficulty interval) are configurable without code changes
**Plans:** 2 plans

Plans:
- [ ] 01-01-PLAN.md — Project scaffold, domain types (Block/Header/Hash), SHA-256d hashing, PoW mining service, and go-zero config
- [ ] 01-02-PLAN.md — Difficulty adjustment, Chain aggregate, bbolt persistence, and runnable main entry point

### Phase 2: Wallets and Transactions
**Goal**: Users can create wallets, derive addresses, and send coins via UTXO transactions with cryptographic signing and verification
**Depends on**: Phase 1
**Requirements**: TX-01, TX-02, TX-03, TX-04, TX-05, TX-06, TX-07, TX-08
**Success Criteria** (what must be TRUE):
  1. User can generate a new ECDSA wallet and receive a human-readable Base58Check address
  2. User can create a transaction that spends specific UTXOs, and the system automatically creates change outputs when input exceeds payment
  3. Every mined block includes a coinbase transaction that credits the block reward to the miner's address
  4. Transaction inputs with invalid signatures are rejected during validation
  5. The UTXO set persists across restarts and supports undo operations (reversibility) for future chain reorganization
**Plans**: TBD

Plans:
- [ ] 02-01: TBD
- [ ] 02-02: TBD

### Phase 3: Mempool, Mining Integration, and CLI
**Goal**: Users can operate a complete single-node blockchain through CLI commands -- creating wallets, sending transactions, mining blocks, and inspecting the chain
**Depends on**: Phase 2
**Requirements**: MINE-04, MINE-05, MINE-07, NET-03, CLI-01, CLI-02, CLI-03, CLI-04, CLI-05, CLI-06, CLI-07
**Success Criteria** (what must be TRUE):
  1. User can run `createwallet`, `listaddresses`, `getbalance`, `send`, `mine`, `printchain`, and `startnode` CLI commands successfully
  2. User can send coins via CLI, see the transaction enter the mempool, mine a block containing it, and verify the updated balance
  3. Node can auto-mine blocks continuously in the background, stoppable via cancellation
  4. Mined block headers contain a correct Merkle root computed from the block's transaction hashes
  5. Mempool correctly rejects duplicate or double-spending transactions under concurrent access
**Plans**: TBD

Plans:
- [ ] 03-01: TBD
- [ ] 03-02: TBD

### Phase 4: P2P Networking and Consensus
**Goal**: Multiple nodes on localhost discover each other, synchronize chains, broadcast blocks and transactions, and resolve forks via longest-chain rule
**Depends on**: Phase 3
**Requirements**: NET-01, NET-02, NET-04, NET-05, NET-06, NET-07, NET-08, NET-09
**Success Criteria** (what must be TRUE):
  1. User can start multiple nodes on different localhost ports and they connect with a version handshake
  2. A transaction created on one node appears in the mempool of all connected peers
  3. A block mined on one node is received, validated, and added to the chain on all peers
  4. A newly started node synchronizes the full chain from an existing peer before accepting new blocks
  5. When two nodes mine competing blocks, the network converges on the longest valid chain via reorganization, correctly reversing and reapplying UTXO changes
**Plans**: TBD

Plans:
- [ ] 04-01: TBD
- [ ] 04-02: TBD

### Phase 5: Web Dashboard
**Goal**: Users can visually explore the blockchain, monitor node health, and watch mining in real-time through a web browser
**Depends on**: Phase 4
**Requirements**: DASH-01, DASH-02, DASH-03, DASH-04, DASH-05
**Success Criteria** (what must be TRUE):
  1. User can open a browser and browse blocks and their transactions in a block explorer interface
  2. Dashboard displays live node status: connected peers, mempool size, chain height, and mining status
  3. User can watch mining in real-time, seeing nonce attempts, hash values, and target comparison
  4. User can view pending transactions in the mempool through the dashboard
  5. User can search by block hash, transaction hash, or address and get relevant results
**Plans**: TBD

Plans:
- [ ] 05-01: TBD
- [ ] 05-02: TBD

### Phase 6: Advanced Educational Features
**Goal**: The blockchain demonstrates economic mechanics (halving, fees) and provides turnkey demo scenarios (multi-node testnet, double-spend attack)
**Depends on**: Phase 5
**Requirements**: MINE-08, TX-09, TX-10, ORCH-01, DEMO-01
**Success Criteria** (what must be TRUE):
  1. Block reward visibly halves after every N blocks (configurable interval)
  2. Transactions with higher fees are prioritized by the miner during block construction
  3. User can launch a multi-node local testnet with a single CLI command
  4. User can trigger a double-spend attempt that the network detects and rejects, demonstrating blockchain security
**Plans**: TBD

Plans:
- [ ] 06-01: TBD

## Progress

**Execution Order:**
Phases execute in numeric order: 1 -> 2 -> 3 -> 4 -> 5 -> 6

| Phase | Plans Complete | Status | Completed |
|-------|----------------|--------|-----------|
| 1. Core Chain Foundation | 0/2 | Planning complete | - |
| 2. Wallets and Transactions | 0/0 | Not started | - |
| 3. Mempool, Mining Integration, and CLI | 0/0 | Not started | - |
| 4. P2P Networking and Consensus | 0/0 | Not started | - |
| 5. Web Dashboard | 0/0 | Not started | - |
| 6. Advanced Educational Features | 0/0 | Not started | - |

---
*Roadmap created: 2026-03-05*
*Last updated: 2026-03-05*
