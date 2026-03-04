# Feature Research

**Domain:** Educational Bitcoin-like blockchain implementation in Go
**Researched:** 2026-03-05
**Confidence:** HIGH

## Feature Landscape

### Table Stakes (Must Have or the Project Fails to Teach Blockchain)

These features are non-negotiable for an educational blockchain clone. Without them, the project does not demonstrate how blockchains actually work. Every comparable educational project (Jeiwan/blockchain_go, TheODDYSEY/Blockchain-Go, Iliiasik/Blockchain) implements all of these.

| Feature | Why Expected | Complexity | Notes |
|---------|--------------|------------|-------|
| **Genesis block creation** | Every chain starts here; foundational concept | LOW | Hardcoded first block with no previous hash. Include a configurable message (like Bitcoin's "Chancellor on brink..."). |
| **Block structure** (header + body) | Core data structure of any blockchain | LOW | Header: version, prev hash, merkle root, timestamp, difficulty target, nonce. Body: list of transactions. |
| **SHA-256 hashing** | Bitcoin uses SHA-256; makes blocks immutable | LOW | Standard library `crypto/sha256`. Double-hash (SHA256d) like Bitcoin for block headers. |
| **Proof of Work mining** | The consensus mechanism that makes it a blockchain, not just a linked list | MEDIUM | Hash the block header, compare against difficulty target. Increment nonce until valid. Both manual trigger and auto-mine modes. |
| **Adjustable difficulty** | Demonstrates why mining time stays stable as hashrate changes | MEDIUM | Simplified retargeting: adjust every N blocks based on time taken vs target time. Bitcoin does every 2016 blocks; use a smaller number (e.g., 10) for faster feedback. |
| **UTXO transaction model** | Bitcoin's actual transaction model; more educational than account-based | HIGH | Inputs reference previous outputs, outputs define new spendable amounts. Change outputs. No partial spending of UTXOs. |
| **Coinbase transactions** | How new coins enter the system; every block must have one | MEDIUM | Special transaction with no inputs. Creates the block reward. Miner specifies destination address. |
| **Transaction validation** | Without validation, anyone can spend anyone's coins | MEDIUM | Verify signatures, verify input UTXOs exist and are unspent, verify input sum >= output sum. Reject double-spends. |
| **UTXO set management** | Performance optimization that Bitcoin Core depends on; teaches indexing | MEDIUM | Maintain a separate index of unspent outputs rather than scanning the entire chain. Rebuild from chain data on demand. |
| **Wallet key generation** | Can't sign transactions without keys | MEDIUM | ECDSA key pair generation (secp256k1 or P-256). Private key storage. Public key derivation. |
| **Address derivation** | How public keys become human-readable addresses | MEDIUM | Hash public key (SHA-256 then RIPEMD-160), add version byte, Base58Check encode. Matches Bitcoin's address format conceptually. |
| **Transaction signing** | Proves ownership; the "auth" of blockchain | MEDIUM | Sign transaction inputs with the private key corresponding to the referenced UTXO's address. ECDSA signatures. |
| **Signature verification** | Every node must verify independently | LOW | Verify ECDSA signatures against the public key embedded in the transaction. Reject invalid signatures. |
| **Persistent storage** | Chain must survive restarts | MEDIUM | Embedded KV store (BoltDB or BadgerDB). Store blocks, UTXO set, and chain tip. Serialize/deserialize blocks. |
| **CLI for node operations** | Primary user interface for interacting with the chain | MEDIUM | Commands: `createwallet`, `listaddresses`, `getbalance`, `send`, `mine`, `printchain`, `startnode`. Use cobra or built-in flag package. |
| **Mempool** | Where transactions wait before being mined; critical for understanding block construction | MEDIUM | In-memory pool of unvalidated-but-valid transactions. Miner selects from mempool when building blocks. Evict after inclusion. |
| **Merkle tree** | How Bitcoin efficiently summarizes transactions in a block header | MEDIUM | Binary tree of transaction hashes. Root goes into block header. Enables proof of inclusion without downloading all transactions. |
| **P2P networking (local)** | Blockchain without a network is just a database | HIGH | TCP connections between nodes on localhost. Peer list management. Message framing. Version handshake. |
| **Block broadcasting** | How new blocks propagate through the network | MEDIUM | When a node mines a block, broadcast to all connected peers. Peers validate and re-broadcast. |
| **Transaction broadcasting** | How transactions get to miners | MEDIUM | When a wallet creates a transaction, broadcast to peers. Peers add to mempool and re-broadcast. |
| **Chain synchronization** | New nodes must catch up to the current state | HIGH | When a node connects, compare chain heights. Request missing blocks from peers. Apply blocks in order. Handle the "initial block download" scenario. |

### Differentiators (Deeper Learning / Stand-Out Features)

These features go beyond what most educational blockchain tutorials cover. They demonstrate deeper understanding and make the project more impressive and useful as a learning tool.

| Feature | Value Proposition | Complexity | Notes |
|---------|-------------------|------------|-------|
| **Web dashboard with block explorer** | Visual understanding of chain state; most tutorials are CLI-only | HIGH | Browse blocks, view transactions, see UTXO set. Search by block hash, tx hash, address. Real-time updates via WebSocket. This is explicitly in the project requirements. |
| **Node status dashboard** | Operational visibility into a running node; rare in educational projects | MEDIUM | Show: connected peers, mempool size, chain height, mining status, hashrate. Auto-refresh. Pair with block explorer in single web UI. |
| **Mining visualization** | See PoW happening in real-time; extremely educational | MEDIUM | Show current nonce attempts, hash values, target comparison. Display mining progress as it happens via WebSocket stream. |
| **Fork detection and resolution** | Most tutorials ignore this; it's where consensus gets real | HIGH | Detect when two valid blocks arrive at same height. Implement longest-chain (most-work) rule. Reorganize chain when a longer fork is discovered. Crucial for understanding why consensus works. |
| **Block reward halving** | Demonstrates Bitcoin's monetary policy and scarcity model | LOW | Halve reward every N blocks (use small N like 100 for demos). Shows deflationary supply mechanics. |
| **Transaction fees** | How miners are incentivized beyond block rewards; real economic model | MEDIUM | Input sum - output sum = implicit fee. Miner collects fees. Optional: prioritize higher-fee transactions in block construction. |
| **Mempool visualization** | See pending transactions waiting to be mined | LOW | List transactions in mempool with age, fee, size. Show them being consumed when a block is mined. Part of web dashboard. |
| **Multi-node orchestration** | Spin up a local testnet with one command | MEDIUM | Script or CLI command that launches 3+ nodes on different ports, connects them, and optionally seeds with test transactions. Dramatically improves demo experience. |
| **Chain data export/import** | Save and reload chain state for testing | LOW | Export chain to JSON or binary file. Import on a fresh node. Useful for reproducible demos and testing. |
| **Configurable consensus parameters** | Experiment with different settings to understand tradeoffs | LOW | Config file or flags for: block time target, difficulty adjustment interval, initial block reward, halving interval. |
| **Double-spend detection demo** | Show why double-spending fails; the core value proposition of blockchain | MEDIUM | Deliberately attempt a double-spend and show how the network rejects it. Great for presentations and teaching. |

### Anti-Features (Deliberately NOT Building)

Features that seem attractive but add complexity without educational value, or actively mislead about how real blockchains work.

| Feature | Why Requested | Why Problematic | Alternative |
|---------|---------------|-----------------|-------------|
| **Bitcoin protocol compatibility** | "It should talk to real Bitcoin nodes" | Enormous complexity (serialization formats, version handshakes, BIP compliance). Zero educational value added. You'd spend 80% of time on protocol bytes, not blockchain concepts. | Implement the same concepts (PoW, UTXO, Merkle trees) with a simpler custom protocol. |
| **Bitcoin Script / scripting language** | "Real Bitcoin has programmable transactions" | Script is a stack-based VM with 100+ opcodes. Building it is a separate project. Educational blockchain should focus on the chain, not a scripting engine. | Simplified transaction validation: verify signature matches address. Mention Script in documentation as "what Bitcoin does here." |
| **Internet-scale P2P (NAT traversal, DHT, DNS seeds)** | "Real P2P needs to work across the internet" | NAT traversal, hole punching, and peer discovery across the internet are networking problems, not blockchain problems. Massive complexity for no blockchain learning value. | Localhost-only P2P with explicit peer addresses. Demonstrates the same concepts (gossip, sync, fork resolution) without networking nightmares. |
| **SPV / light clients** | "Not every node should store the full chain" | SPV requires Merkle proofs, bloom filters, and a different sync protocol. Doubles the networking complexity. Better tackled after the full-node implementation is solid. | All nodes are full nodes. Document SPV as a "what Bitcoin does for mobile wallets" topic. |
| **Smart contracts / EVM** | "Add Ethereum-style smart contracts" | This is an entirely different project. Conflating Bitcoin-style UTXO chains with Ethereum-style account+contract chains muddies the educational message. | Stay focused on Bitcoin mechanics. If smart contracts are interesting, that's a separate project. |
| **GPU mining / ASIC simulation** | "Real mining uses GPUs" | The point is understanding PoW conceptually, not optimizing hash throughput. GPU mining adds CUDA/OpenCL complexity with no conceptual learning. | CPU mining with intentionally low difficulty. The hash-and-compare loop is the same regardless of hardware. |
| **Wallet encryption / BIP-39 mnemonics** | "Real wallets use seed phrases" | Mnemonic generation, derivation paths (BIP-32/44), and encryption are cryptographic standards worth understanding but are a wallet project, not a blockchain project. | Store private keys in a simple file. Mention BIP-39 in docs. Focus effort on transaction signing instead. |
| **Consensus algorithm variety (PoS, DPoS, PBFT)** | "Support multiple consensus algorithms" | Each consensus algorithm is fundamentally different in design. Abstracting across them creates a leaky abstraction. Better to do one well. | Implement PoW thoroughly. Document how PoS differs conceptually. |
| **Mobile or desktop GUI** | "Build native apps" | Cross-platform GUI adds massive dependency burden (Fyne, Wails, etc.) for minimal blockchain learning. | Web dashboard served by the Go node. Works everywhere, teaches HTTP/WebSocket, no native dependencies. |
| **Real cryptographic security hardening** | "Use constant-time comparisons, HSMs, secure enclaves" | This is an educational project on localhost. Security theater adds complexity without teaching blockchain concepts. | Use standard Go crypto libraries correctly. Document "in production, you would also..." |

## Feature Dependencies

```
Genesis Block
    |
    v
Block Structure + SHA-256 Hashing
    |
    v
Proof of Work Mining (needs block structure to hash)
    |
    +---> Adjustable Difficulty (needs mining to measure block times)
    |
    v
Key Generation + Address Derivation (independent, but needed before transactions)
    |
    v
UTXO Transaction Model
    |
    +---> Coinbase Transactions (special case of transactions)
    |
    +---> Transaction Signing + Verification (needs keys + transactions)
    |
    v
Transaction Validation (needs signing, UTXO tracking)
    |
    +---> UTXO Set Management (optimization of validation)
    |
    +---> Mempool (validated but unmined transactions)
    |
    v
Merkle Tree (needs list of transactions to hash)
    |
    v
Persistent Storage (needs all above to have something to store)
    |
    v
CLI (needs all above to operate on)
    |
    v
P2P Networking
    |
    +---> Transaction Broadcasting (needs P2P + mempool)
    |
    +---> Block Broadcasting (needs P2P + mining)
    |
    +---> Chain Synchronization (needs P2P + storage)
    |
    +---> Fork Detection/Resolution (needs sync + chain comparison)
    |
    v
Web Dashboard (needs running node with P2P to display)
    |
    +---> Block Explorer (needs stored blocks/transactions)
    +---> Node Status (needs P2P + mempool + mining state)
    +---> Mining Visualization (needs mining in progress)
```

### Dependency Notes

- **Transactions require keys:** ECDSA key generation and address derivation must exist before any non-coinbase transaction can be created or validated.
- **Mempool requires validation:** Transactions entering the mempool must be validated first, which requires the full UTXO tracking pipeline.
- **P2P requires storage:** Nodes need persistent chain state before they can meaningfully sync with peers.
- **Fork resolution requires sync:** You can only detect forks when receiving blocks from peers, which requires the sync protocol.
- **Web dashboard requires everything:** It's a visualization layer on top of a fully functioning node. Build it last.
- **Mining visualization enhances mining:** Not a dependency but only meaningful when mining is already working.
- **Block reward halving is independent:** Simple arithmetic on block height, can be added any time after coinbase transactions work.
- **Transaction fees enhance mempool:** Fee-based prioritization is an optimization on top of a working mempool.

## MVP Definition

### Launch With (Phase 1-3: Core Chain)

Minimum set to have a functioning blockchain that can be mined and hold transactions.

- [x] Genesis block creation -- foundation of the chain
- [x] Block structure with SHA-256 hashing -- the data structure everything builds on
- [x] Proof of Work mining with adjustable difficulty -- makes it a blockchain
- [x] ECDSA key generation and address derivation -- needed for transactions
- [x] UTXO transaction model with coinbase transactions -- how value moves
- [x] Transaction signing and verification -- proves ownership
- [x] UTXO set management -- efficient balance/validation lookups
- [x] Persistent storage (BoltDB/BadgerDB) -- survive restarts
- [x] Basic CLI (create wallet, mine, send, check balance, print chain) -- interact with it

### Add After Core Works (Phase 4: Networking)

Features that transform it from a single-node database into a distributed system.

- [ ] P2P networking between localhost nodes -- the "distributed" in distributed ledger
- [ ] Mempool for pending transactions -- where transactions wait for mining
- [ ] Transaction and block broadcasting -- gossip protocol
- [ ] Chain synchronization -- new nodes catch up
- [ ] Fork detection and longest-chain resolution -- consensus gets real
- [ ] Merkle tree in block headers -- efficient transaction verification

### Add After Networking (Phase 5: Visualization and Polish)

Features that make the project impressive and educational to demonstrate.

- [ ] Web dashboard with block explorer -- visual chain browsing
- [ ] Node status panel (peers, mempool, chain height) -- operational visibility
- [ ] Mining visualization -- see PoW in real-time
- [ ] Mempool visualization -- see pending transaction flow
- [ ] Multi-node orchestration script -- one-command testnet
- [ ] Block reward halving -- monetary policy demo
- [ ] Transaction fees -- economic incentive model
- [ ] Double-spend detection demo -- the "why blockchain" demonstration
- [ ] Configurable consensus parameters -- experiment with settings

## Feature Prioritization Matrix

| Feature | Learning Value | Implementation Cost | Priority |
|---------|---------------|---------------------|----------|
| Block structure + hashing | HIGH | LOW | P1 |
| Proof of Work mining | HIGH | MEDIUM | P1 |
| Adjustable difficulty | HIGH | MEDIUM | P1 |
| UTXO transaction model | HIGH | HIGH | P1 |
| Coinbase transactions | HIGH | MEDIUM | P1 |
| Key generation + addresses | HIGH | MEDIUM | P1 |
| Transaction signing/verification | HIGH | MEDIUM | P1 |
| UTXO set management | MEDIUM | MEDIUM | P1 |
| Persistent storage | MEDIUM | MEDIUM | P1 |
| CLI operations | MEDIUM | MEDIUM | P1 |
| Mempool | HIGH | MEDIUM | P2 |
| Merkle tree | HIGH | MEDIUM | P2 |
| P2P networking (localhost) | HIGH | HIGH | P2 |
| Block/transaction broadcasting | HIGH | MEDIUM | P2 |
| Chain synchronization | HIGH | HIGH | P2 |
| Fork detection/resolution | HIGH | HIGH | P2 |
| Web dashboard + block explorer | HIGH | HIGH | P2 |
| Node status dashboard | MEDIUM | MEDIUM | P2 |
| Mining visualization | HIGH | MEDIUM | P3 |
| Mempool visualization | MEDIUM | LOW | P3 |
| Multi-node orchestration | MEDIUM | MEDIUM | P3 |
| Block reward halving | MEDIUM | LOW | P3 |
| Transaction fees | MEDIUM | MEDIUM | P3 |
| Double-spend demo | HIGH | MEDIUM | P3 |
| Configurable parameters | LOW | LOW | P3 |
| Chain export/import | LOW | LOW | P3 |

**Priority key:**
- P1: Core chain functionality -- without these, nothing works
- P2: Networking + visualization -- transforms it from toy to distributed system
- P3: Polish and educational extras -- makes it impressive to demonstrate

## Comparable Project Feature Analysis

| Feature | Jeiwan/blockchain_go | TheODDYSEY/Blockchain-Go | Iliiasik/Blockchain | Shitcoin (Our Plan) |
|---------|---------------------|--------------------------|---------------------|---------------------|
| Block structure + PoW | Yes | Yes | Yes | Yes |
| UTXO transactions | Yes | Yes | Yes | Yes |
| Wallets + signing | Yes | Yes | Yes | Yes |
| Merkle tree | Yes | Yes | Unknown | Yes |
| Persistent storage | BoltDB | BoltDB | In-memory | BoltDB or BadgerDB |
| CLI | Yes (basic) | Yes (basic) | No (GUI) | Yes (comprehensive) |
| P2P networking | Partial (tutorial part 7) | Yes (TCP) | Simulated | Yes (localhost TCP) |
| Chain sync | Basic | Yes | Simulated | Yes |
| Fork resolution | No | Unknown | No | **Yes (differentiator)** |
| Web dashboard | No | No | Fyne GUI | **Yes (differentiator)** |
| Mining visualization | No | No | Yes (GUI) | **Yes (differentiator)** |
| Mempool visualization | No | No | Yes (GUI) | **Yes (differentiator)** |
| Difficulty adjustment | Static | Yes | Unknown | **Yes** |
| Transaction fees | No | No | No | **Yes (differentiator)** |
| Multi-node orchestration | Manual | Manual | N/A | **Yes (differentiator)** |

**Key differentiation:** Shitcoin distinguishes itself by combining a full working P2P network with fork resolution (which most tutorials skip) and a web-based visualization layer (which no comparable Go CLI project offers). The Iliiasik project has GUI visualization but uses Fyne (desktop) and simulates networking rather than implementing real P2P. No comparable project offers all three: real P2P + fork resolution + web dashboard.

## Sources

- [Jeiwan/blockchain_go](https://github.com/Jeiwan/blockchain_go) - The canonical Go blockchain tutorial (7 parts covering all fundamentals)
- [TheODDYSEY/Blockchain-Go](https://github.com/TheODDYSEY/Blockchain-Go) - Complete educational blockchain with P2P networking
- [Iliiasik/Blockchain](https://github.com/Iliiasik/Blockchain) - GUI blockchain simulator with Fyne visualization
- [Bitcoin Core Architecture Overview](https://btctranscripts.com/scalingbitcoin/tokyo-2018/edgedevplusplus/overview-bitcoin-core-architecture/) - Chaincode Labs talk on Bitcoin Core components
- [LearnMeABitcoin - Difficulty](https://learnmeabitcoin.com/beginners/guide/difficulty/) - Bitcoin difficulty adjustment explanation
- [LearnMeABitcoin - Chain Reorganization](https://learnmeabitcoin.com/technical/blockchain/chain-reorganization/) - Fork resolution and longest-chain rule
- [LearnMeABitcoin - Coinbase Transaction](https://learnmeabitcoin.com/technical/mining/coinbase-transaction/) - Block reward mechanics
- [River - Bitcoin's UTXO Model](https://river.com/learn/bitcoins-utxo-model/) - UTXO model explanation
- [Implementing a blockchain from scratch (Springer)](https://jis-eurasipjournals.springeropen.com/articles/10.1186/s13635-019-0085-3) - Academic paper on blockchain implementation lessons

---
*Feature research for: Educational Bitcoin-like blockchain (Shitcoin)*
*Researched: 2026-03-05*
