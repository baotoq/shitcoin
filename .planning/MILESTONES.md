# Milestones

## v1.2 Testing & Quality (Shipped: 2026-03-08)

**Phases completed:** 5 phases (14-18), 11 plans
**Timeline:** 1 day (2026-03-08)
**Stats:** 32 commits, 7,375 test LOC across 12 test packages

**Key accomplishments:**
- Shared testutil package with 5 builders and 3 mock repositories, eliminating ~700 lines of duplicated mock code
- Domain layer coverage: chain 85.4%, P2P 80.1%, tx/utxo/mempool 100%, wallet 97.8%
- BoltDB persistence tests at 86.3% coverage with atomic save, reorg, and undo entry verification
- API handler tests at 93.5% and WebSocket hub tests at 84.0% using httptest and mock dependencies
- P2P integration tests verifying TCP handshake, block sync via IBD, and transaction relay between in-process nodes
- E2E chain scenario tests covering full wallet-to-balance workflow in single test functions
- Fixed ws.Hub broadcast data race and enabled `-race` flag in CI pipeline

---

## v1.0 Educational Blockchain (Shipped: 2026-03-07)

**Phases completed:** 8 phases, 22 plans
**Timeline:** 3 days (2026-03-05 to 2026-03-07)
**Stats:** 127 commits, 335 files, 11,449 Go LOC

**Key accomplishments:**
- Complete blockchain with PoW mining, SHA-256 double-hash, difficulty adjustment, and Merkle trees
- UTXO-based transactions with ECDSA signing (secp256k1), Base58Check addresses, and undo-log for chain reorgs
- TCP-based P2P networking with version handshake, block/tx relay, initial block download, and longest-chain consensus
- React + Vite web dashboard with block explorer, mining visualizer, mempool view, and WebSocket live updates
- Educational demos: block reward halving, fee-prioritized mining, multi-node testnet, double-spend attack
- Full CLI with 9 commands: createwallet, listaddresses, getbalance, send, mine, startnode, printchain, testnet, demo

---

## v1.1 CI/CD & Kubernetes (Shipped: 2026-03-07)

**Phases completed:** 5 phases (9-13), 9 plans
**Timeline:** 1 day (2026-03-07)

**Key accomplishments:**
- Multi-stage Dockerfiles for Go backend (~15MB) and React frontend with nginx reverse proxy
- GitHub Actions CI for Go tests, golangci-lint, frontend lint/typecheck/build, Docker image push to GHCR
- Kustomize base + dev/prod overlays with PVC, ConfigMap, health probes, Recreate strategy
- Tiltfile with live-update for Go and React hot reload on local kind cluster
- ArgoCD Application CR for GitOps auto-sync from git to cluster

---

