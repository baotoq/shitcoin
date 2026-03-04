# Shitcoin

## What This Is

A full blockchain implementation in Go that replicates Bitcoin's core mechanics for educational purposes. It implements Proof of Work mining, UTXO-based transactions, P2P networking between local nodes, and wallet/key management — all simplified to focus on understanding how blockchains work rather than protocol compatibility. Includes a CLI for node operations and a web dashboard for chain visualization.

## Core Value

A working blockchain you built and understand end-to-end — from transaction creation to block mining to peer synchronization.

## Requirements

### Validated

(None yet — ship to validate)

### Active

- [ ] Proof of Work mining with adjustable difficulty (auto-mine and manual modes)
- [ ] UTXO transaction model with inputs, outputs, and validation
- [ ] P2P networking between multiple local nodes (different ports on localhost)
- [ ] Wallet/key management (key generation, address derivation, signing)
- [ ] CLI for node operations (start node, send transactions, mine blocks, check balances)
- [ ] Web dashboard with block/transaction explorer and node status (peers, mempool, chain height, mining status)
- [ ] Persistent chain storage using embedded key-value store (BoltDB or BadgerDB)
- [ ] Genesis block creation and chain initialization

### Out of Scope

- Bitcoin protocol compatibility — this is an educational clone, not a Bitcoin client
- Internet-scale P2P (NAT traversal, DNS seeds, peer discovery beyond localhost)
- Scripting system (Bitcoin Script) — simplified transaction validation instead
- SPV/light clients — all nodes are full nodes
- Mobile or desktop GUI — CLI + web dashboard only

## Context

- Personal learning project to understand blockchain internals by building one
- Go chosen as implementation language (good concurrency support, systems-level)
- Educational clone approach: same concepts as Bitcoin (PoW, UTXO, Merkle trees) but simplified implementations
- Local multi-node testing: run 2-3+ nodes on localhost with different ports to demonstrate consensus and sync
- Dashboard serves dual purpose: block explorer (browse chain data) and operational view (node health)

## Constraints

- **Language**: Go — chosen for learning and concurrency primitives
- **Dependencies**: Minimal external dependencies — prefer standard library where reasonable
- **Storage**: Embedded KV store (BoltDB/BadgerDB) — no external database servers
- **Networking**: localhost only — no NAT traversal or internet-facing requirements

## Key Decisions

| Decision | Rationale | Outcome |
|----------|-----------|---------|
| Educational clone over faithful replica | Focus on understanding concepts, not protocol compatibility | — Pending |
| UTXO model over account model | Matches Bitcoin's approach, more interesting to implement | — Pending |
| Embedded KV store for persistence | Simple, no external deps, good for learning storage patterns | — Pending |
| Local multi-node P2P | Proves concepts without networking complexity | — Pending |
| Both auto-mine and manual mine modes | Auto for demos, manual for testing/learning | — Pending |

---
*Last updated: 2026-03-05 after initialization*
