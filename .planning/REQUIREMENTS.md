# Requirements: Shitcoin

**Defined:** 2026-03-05
**Core Value:** A working blockchain you built and understand end-to-end — from transaction creation to block mining to peer synchronization.

## v1 Requirements

Requirements for initial release. Each maps to roadmap phases.

### Blocks & Mining

- [x] **MINE-01**: System creates a genesis block with a configurable embedded message on chain initialization
- [x] **MINE-02**: Block structure contains header (prev hash, Merkle root, timestamp, difficulty target, nonce) and body (transaction list)
- [x] **MINE-03**: Block headers are hashed using SHA-256 double-hash with deterministic canonical serialization
- [x] **MINE-04**: User can mine a block manually via CLI command
- [x] **MINE-05**: Node can auto-mine blocks continuously in the background with context-based cancellation
- [x] **MINE-06**: Difficulty adjusts automatically every N blocks based on actual vs target block time (window-based, clamped)
- [x] **MINE-07**: Block headers include a Merkle root computed from the block's transaction hashes
- [ ] **MINE-08**: Block reward halves every N blocks (configurable interval)
- [x] **MINE-09**: Consensus parameters (block time target, difficulty interval, initial reward, halving interval) are configurable

### Transactions & Wallets

- [x] **TX-01**: User can create a new wallet with ECDSA key pair (secp256k1 curve)
- [x] **TX-02**: Public keys are converted to human-readable addresses via SHA-256 → RIPEMD-160 → Base58Check
- [x] **TX-03**: User can send coins from one address to another, creating a UTXO transaction with inputs and outputs
- [x] **TX-04**: Every transaction input references a specific unspent output and includes a valid ECDSA signature
- [x] **TX-05**: Change outputs are automatically created when input value exceeds payment amount (sum invariant enforced)
- [x] **TX-06**: Each mined block includes a coinbase transaction that creates the block reward for the miner
- [x] **TX-07**: System maintains a persistent UTXO set for efficient balance queries and transaction validation
- [x] **TX-08**: UTXO set supports reversibility (undo-log) to enable chain reorganization
- [ ] **TX-09**: Transaction fees are computed as the difference between input and output sums, collected by the miner
- [ ] **TX-10**: Miner prioritizes transactions in block construction by fee rate

### Networking

- [x] **NET-01**: User can start a node that listens on a configurable TCP port on localhost
- [x] **NET-02**: Nodes perform a version handshake when connecting to establish protocol compatibility
- [x] **NET-03**: Mempool holds validated-but-unmined transactions, protected by RWMutex for concurrent access
- [x] **NET-04**: When a user creates a transaction, it is broadcast to all connected peers
- [x] **NET-05**: When a node mines a block, it is broadcast to all connected peers
- [x] **NET-06**: Peers validate received blocks and transactions before accepting and re-broadcasting
- [ ] **NET-07**: When a new node connects, it synchronizes the full chain from peers (initial block download)
- [ ] **NET-08**: Node detects when a peer has a longer valid chain and reorganizes to the longest chain
- [ ] **NET-09**: Chain reorganization reverses UTXO changes from orphaned blocks and applies the new chain's changes

### Interface

- [x] **CLI-01**: User can create a wallet via `createwallet` command
- [x] **CLI-02**: User can list all wallet addresses via `listaddresses` command
- [x] **CLI-03**: User can check balance of an address via `getbalance` command
- [x] **CLI-04**: User can send coins between addresses via `send` command
- [x] **CLI-05**: User can mine a block via `mine` command
- [x] **CLI-06**: User can print the full blockchain via `printchain` command
- [x] **CLI-07**: User can start a full node via `startnode` command with port and mining address options
- [ ] **DASH-01**: Web dashboard displays a block explorer where user can browse blocks and transactions
- [ ] **DASH-02**: Web dashboard shows node status: connected peers, mempool size, chain height, mining status
- [ ] **DASH-03**: Web dashboard visualizes mining in real-time (nonce attempts, hash values, target comparison)
- [ ] **DASH-04**: Web dashboard shows mempool with pending transactions
- [ ] **DASH-05**: User can search by block hash, transaction hash, or address in the dashboard
- [ ] **ORCH-01**: User can launch a local multi-node testnet with a single CLI command
- [ ] **DEMO-01**: User can trigger a double-spend attempt that the network detects and rejects, demonstrating blockchain security

## v2 Requirements

Deferred to future release. Tracked but not in current roadmap.

### Advanced Networking

- **ANET-01**: Node discovers peers on the local network automatically
- **ANET-02**: Chain data export to JSON file for sharing/analysis
- **ANET-03**: Chain data import from JSON file to bootstrap a node

## Out of Scope

| Feature | Reason |
|---------|--------|
| Bitcoin protocol compatibility | Educational clone — same concepts, simpler protocol |
| Bitcoin Script / scripting VM | Separate project scope — simplified validation instead |
| Internet-scale P2P (NAT, DNS seeds) | Networking problem, not blockchain learning |
| SPV / light clients | All nodes are full nodes — SPV adds protocol complexity |
| Smart contracts / EVM | Different blockchain paradigm entirely |
| GPU mining | PoW concept is the same regardless of hardware |
| BIP-39 mnemonics / HD wallets | Wallet project, not blockchain project |
| Multiple consensus algorithms | Do PoW thoroughly; document alternatives |
| Mobile / desktop native GUI | Web dashboard covers visualization needs |
| Cryptographic hardening (HSM, constant-time) | Educational project on localhost |

## Traceability

Which phases cover which requirements. Updated during roadmap creation.

| Requirement | Phase | Status |
|-------------|-------|--------|
| MINE-01 | Phase 1 | Complete |
| MINE-02 | Phase 1 | Complete |
| MINE-03 | Phase 1 | Complete |
| MINE-04 | Phase 3 | Complete |
| MINE-05 | Phase 3 | Complete |
| MINE-06 | Phase 1 | Complete |
| MINE-07 | Phase 3 | Complete |
| MINE-08 | Phase 6 | Pending |
| MINE-09 | Phase 1 | Complete |
| TX-01 | Phase 2 | Complete |
| TX-02 | Phase 2 | Complete |
| TX-03 | Phase 2 | Complete |
| TX-04 | Phase 2 | Complete |
| TX-05 | Phase 2 | Complete |
| TX-06 | Phase 2 | Complete |
| TX-07 | Phase 2 | Complete |
| TX-08 | Phase 2 | Complete |
| TX-09 | Phase 6 | Pending |
| TX-10 | Phase 6 | Pending |
| NET-01 | Phase 4 | Complete |
| NET-02 | Phase 4 | Complete |
| NET-03 | Phase 3 | Complete |
| NET-04 | Phase 4 | Complete |
| NET-05 | Phase 4 | Complete |
| NET-06 | Phase 4 | Complete |
| NET-07 | Phase 4 | Pending |
| NET-08 | Phase 4 | Pending |
| NET-09 | Phase 4 | Pending |
| CLI-01 | Phase 3 | Complete |
| CLI-02 | Phase 3 | Complete |
| CLI-03 | Phase 3 | Complete |
| CLI-04 | Phase 3 | Complete |
| CLI-05 | Phase 3 | Complete |
| CLI-06 | Phase 3 | Complete |
| CLI-07 | Phase 3 | Complete |
| DASH-01 | Phase 5 | Pending |
| DASH-02 | Phase 5 | Pending |
| DASH-03 | Phase 5 | Pending |
| DASH-04 | Phase 5 | Pending |
| DASH-05 | Phase 5 | Pending |
| ORCH-01 | Phase 6 | Pending |
| DEMO-01 | Phase 6 | Pending |

**Coverage:**
- v1 requirements: 42 total
- Mapped to phases: 42
- Unmapped: 0

---
*Requirements defined: 2026-03-05*
*Last updated: 2026-03-05 after roadmap creation*
