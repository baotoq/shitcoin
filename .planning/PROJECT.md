# Shitcoin

## What This Is

A full blockchain implementation in Go that replicates Bitcoin's core mechanics for educational purposes. Implements Proof of Work mining, UTXO-based transactions with fees and halving, P2P networking with consensus, wallet/key management, a 9-command CLI, and a React web dashboard with real-time mining visualization.

## Core Value

A working blockchain you built and understand end-to-end — from transaction creation to block mining to peer synchronization.

## Requirements

### Validated

- ✓ Proof of Work mining with adjustable difficulty, auto-mine and manual modes — v1.0
- ✓ UTXO transaction model with inputs, outputs, signing, and validation — v1.0
- ✓ P2P networking between multiple local nodes with consensus — v1.0
- ✓ Wallet/key management (ECDSA secp256k1, Base58Check addresses) — v1.0
- ✓ CLI for node operations (9 commands including testnet and demo) — v1.0
- ✓ Web dashboard with block explorer, mining visualizer, and live status — v1.0
- ✓ Persistent chain storage using BoltDB with UTXO undo-log — v1.0
- ✓ Genesis block creation and chain initialization — v1.0
- ✓ Block reward halving and fee-prioritized mining — v1.0
- ✓ Multi-node testnet orchestration and double-spend demo — v1.0

### Active

(None — v1.0 complete, define next milestone for new requirements)

### Out of Scope

- Bitcoin protocol compatibility — educational clone, not a Bitcoin client
- Internet-scale P2P (NAT traversal, DNS seeds) — localhost networking proves concepts
- Scripting system (Bitcoin Script) — simplified transaction validation instead
- SPV/light clients — all nodes are full nodes
- Mobile or desktop GUI — CLI + web dashboard covers needs

## Context

Shipped v1.0 with 11,449 Go LOC + React frontend.
Tech stack: Go 1.26.1, go-zero, BoltDB, gorilla/websocket, React + Vite + TypeScript + Tailwind CSS.
42 requirements satisfied across 8 phases (22 plans, 127 commits, 3 days).

## Constraints

- **Language**: Go 1.26.1
- **Dependencies**: Minimal — stdlib + go-zero + BoltDB + gorilla/websocket + btcec
- **Storage**: BoltDB embedded KV store
- **Networking**: localhost only
- **Frontend**: React + Vite + TypeScript SPA

## Key Decisions

| Decision | Rationale | Outcome |
|----------|-----------|---------|
| Educational clone over faithful replica | Focus on understanding concepts, not protocol compatibility | ✓ Good |
| UTXO model over account model | Matches Bitcoin's approach, more interesting to implement | ✓ Good |
| BoltDB for persistence | Simple, no external deps, atomic block+UTXO saves | ✓ Good |
| Local multi-node P2P | Proves concepts without networking complexity | ✓ Good |
| DDD with go-zero | Clean architecture, ServiceContext pattern works well | ✓ Good |
| Block txs as []any | Breaks import cycles between block and tx packages | ✓ Good |
| React + Vite SPA (not embedded) | Separate dev server, faster iteration, modern tooling | ✓ Good |
| gorilla/websocket for real-time | Mature library, simple hub pattern for broadcasting | ✓ Good |
| In-process double-spend demo | Faster and more reliable than subprocess approach | ✓ Good |
| Total-fee sorting (not fee-per-byte) | Educational project, all txs roughly same size | ✓ Good |

---
*Last updated: 2026-03-07 after v1.0 milestone*
