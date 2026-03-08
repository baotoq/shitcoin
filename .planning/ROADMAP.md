# Roadmap: Shitcoin

## Milestones

- ✅ **v1.0 Educational Blockchain** -- Phases 1-6 (shipped 2026-03-07)
- ✅ **v1.1 CI/CD & Kubernetes** -- Phases 9-13 (shipped 2026-03-07)
- 🚧 **v1.2 Testing & Quality** -- Phases 14-18 (in progress)

## Phases

<details>
<summary>✅ v1.0 Educational Blockchain (Phases 1-6) -- SHIPPED 2026-03-07</summary>

- [x] Phase 1: Core Chain Foundation (2/2 plans) -- completed 2026-03-04
- [x] Phase 2: Wallets and Transactions (3/3 plans) -- completed 2026-03-05
- [x] Phase 3: Mempool, Mining, CLI (2/2 plans) -- completed 2026-03-05
- [x] Phase 4: P2P Networking and Consensus (4/4 plans) -- completed 2026-03-05
- [x] Phase 4.1: Use Test Assert (2/2 plans) -- completed 2026-03-05
- [x] Phase 5: Web Dashboard (5/5 plans) -- completed 2026-03-07
- [x] Phase 5.1: Upgrade to Go 1.26.1 (1/1 plan) -- completed 2026-03-07
- [x] Phase 6: Advanced Educational Features (3/3 plans) -- completed 2026-03-07

</details>

<details>
<summary>✅ v1.1 CI/CD & Kubernetes (Phases 9-13) -- SHIPPED 2026-03-07</summary>

- [x] Phase 9: Containerization (2/2 plans) -- completed 2026-03-07
- [x] Phase 10: CI Pipeline (2/2 plans) -- completed 2026-03-07
- [x] Phase 11: Kubernetes Manifests (2/2 plans) -- completed 2026-03-07
- [x] Phase 12: Local K8s Development (2/2 plans) -- completed 2026-03-07
- [x] Phase 13: GitOps Deployment (1/1 plan) -- completed 2026-03-07

</details>

### 🚧 v1.2 Testing & Quality

**Milestone Goal:** Achieve comprehensive test coverage across all layers -- domain logic, P2P networking, API/WebSocket, and infrastructure persistence.

- [x] **Phase 14: Test Infrastructure** - Shared test helpers, consolidated mocks, and reusable builders in `internal/testutil/` (completed 2026-03-07)
- [x] **Phase 15: Domain Layer Coverage** - Unit tests for chain, P2P, utxo, wallet, mempool, tx, and error paths across all domain packages (completed 2026-03-08)
- [x] **Phase 16: Infrastructure Persistence Tests** - BoltDB repository and JSON file wallet store test coverage (completed 2026-03-08)
- [ ] **Phase 17: Handler Layer Tests** - REST API and WebSocket hub test coverage with httptest and mock dependencies
- [ ] **Phase 18: Integration & CI Quality** - P2P multi-node integration tests, E2E chain scenarios, and race detection in CI

## Phase Details

### Phase 14: Test Infrastructure
**Goal**: All subsequent test phases can import shared builders and mocks instead of duplicating test scaffolding
**Depends on**: Nothing (first phase of v1.2)
**Requirements**: TINF-01, TINF-02
**Success Criteria** (what must be TRUE):
  1. `internal/testutil/` package exists with reusable block, transaction, wallet, and UTXO builder functions that compile and are importable from any test file
  2. Mock implementations for chain.Repository, utxo.Repository, and wallet.Repository exist in a single shared location, replacing duplicated mocks across 4+ packages
  3. Existing tests that used package-local mocks still pass after migrating to the shared mocks
**Plans:** 2/2 plans complete

Plans:
- [x] 14-01-PLAN.md — Create shared testutil package with builders and mock repositories (TDD)
- [x] 14-02-PLAN.md — Migrate existing tests to shared testutil, delete local duplicates

### Phase 15: Domain Layer Coverage
**Goal**: Domain logic is thoroughly tested, covering happy paths, edge cases, and error conditions across all domain packages
**Depends on**: Phase 14
**Requirements**: DOM-01, DOM-02, DOM-03, DOM-04
**Success Criteria** (what must be TRUE):
  1. `go test -cover ./internal/domain/chain/` reports 85%+ coverage, including mining orchestration, reorg logic, and difficulty adjustment edge cases
  2. `go test -cover ./internal/domain/p2p/` reports 80%+ coverage, including message encoding/decoding, handler dispatch, and sync logic
  3. `go test -cover` for utxo, wallet, mempool, and tx packages each report 95%+ coverage
  4. Tests exist for error paths including invalid blocks, double spends, corrupt data, nil inputs, and boundary conditions -- verified by running `go test -v` and seeing explicit error-case test names
**Plans:** 3/3 plans complete

Plans:
- [x] 15-01-PLAN.md — Gap-fill tx, utxo, wallet, mempool to 95%+ coverage
- [x] 15-02-PLAN.md — Chain aggregate tests to 85%+ coverage
- [x] 15-03-PLAN.md — P2P handler and payload tests to 80%+ coverage

### Phase 16: Infrastructure Persistence Tests
**Goal**: Persistence layer correctness is verified with real BoltDB and file I/O, ensuring data integrity across block saves, queries, and reorgs
**Depends on**: Phase 14
**Requirements**: INFR-01, INFR-02
**Success Criteria** (what must be TRUE):
  1. `go test -cover ./internal/infrastructure/persistence/bbolt/` reports 80%+ coverage, including atomic block+UTXO saves, range queries, reorg deletes, and undo entries
  2. `go test -cover ./internal/infrastructure/persistence/jsonfile/` reports 90%+ coverage
  3. All persistence tests use `t.TempDir()` for isolation and pass when run with `go test -count=2` (no shared state between runs)
**Plans:** 2/2 plans complete

Plans:
- [x] 16-01-PLAN.md — BoltDB repository tests (SaveBlockWithUTXOs, DeleteBlocksAbove, undo entries, storage model round-trips)
- [x] 16-02-PLAN.md — JSON file wallet repo error-path tests (corrupt files, read-only directories)

### Phase 17: Handler Layer Tests
**Goal**: HTTP API and WebSocket handlers are tested against mock dependencies, verifying request/response behavior and event broadcasting
**Depends on**: Phase 14
**Requirements**: HNDL-01, HNDL-02
**Success Criteria** (what must be TRUE):
  1. `go test -cover ./internal/handler/api/` reports 80%+ coverage, with tests for address, mempool, search, and tx handlers using httptest
  2. `go test -cover ./internal/handler/ws/` reports 75%+ coverage, with tests for event subscribe, broadcast to connected clients, and client disconnect cleanup
  3. All handler tests use mock dependencies (no real BoltDB or network connections)
**Plans:** 1/2 plans executed

Plans:
- [ ] 17-01-PLAN.md — API handler tests (AddressHandler, BlockByHashHandler, SearchHandler, BlocksHandler edge cases, MempoolHandler with data)
- [ ] 17-02-PLAN.md — WebSocket ServeWs integration tests and hub_test.go reliability improvements

### Phase 18: Integration & CI Quality
**Goal**: Cross-layer integration tests verify end-to-end workflows, and CI enforces race-safe execution across the entire test suite
**Depends on**: Phase 15, Phase 16, Phase 17
**Requirements**: INTG-01, INTG-02, TINF-03
**Success Criteria** (what must be TRUE):
  1. Integration tests verify TCP handshake, block sync, and tx relay between 2+ in-process nodes -- passing with `go test -v -run Integration`
  2. E2E chain scenario tests verify the full workflow: create wallet, send tx, mine block, verify UTXO updated, check balance -- all in a single test function
  3. `go test -race ./...` passes in CI (GitHub Actions) with zero data race warnings
  4. CI pipeline runs all tests with `-race` flag on every push and PR
**Plans**: TBD

## Progress

**Execution Order:**
Phases execute in numeric order: 14 -> 15 -> 16 -> 17 -> 18
(Phases 15, 16, and 17 can begin in parallel after Phase 14 completes)

| Phase | Milestone | Plans Complete | Status | Completed |
|-------|-----------|----------------|--------|-----------|
| 1. Core Chain Foundation | v1.0 | 2/2 | Complete | 2026-03-04 |
| 2. Wallets and Transactions | v1.0 | 3/3 | Complete | 2026-03-05 |
| 3. Mempool, Mining, CLI | v1.0 | 2/2 | Complete | 2026-03-05 |
| 4. P2P Networking and Consensus | v1.0 | 4/4 | Complete | 2026-03-05 |
| 4.1 Use Test Assert | v1.0 | 2/2 | Complete | 2026-03-05 |
| 5. Web Dashboard | v1.0 | 5/5 | Complete | 2026-03-07 |
| 5.1 Upgrade to Go 1.26.1 | v1.0 | 1/1 | Complete | 2026-03-07 |
| 6. Advanced Educational Features | v1.0 | 3/3 | Complete | 2026-03-07 |
| 9. Containerization | v1.1 | 2/2 | Complete | 2026-03-07 |
| 10. CI Pipeline | v1.1 | 2/2 | Complete | 2026-03-07 |
| 11. Kubernetes Manifests | v1.1 | 2/2 | Complete | 2026-03-07 |
| 12. Local K8s Development | v1.1 | 2/2 | Complete | 2026-03-07 |
| 13. GitOps Deployment | v1.1 | 1/1 | Complete | 2026-03-07 |
| 14. Test Infrastructure | v1.2 | 2/2 | Complete | 2026-03-07 |
| 15. Domain Layer Coverage | v1.2 | 3/3 | Complete | 2026-03-08 |
| 16. Infrastructure Persistence Tests | v1.2 | 2/2 | Complete | 2026-03-08 |
| 17. Handler Layer Tests | 1/2 | In Progress|  | - |
| 18. Integration & CI Quality | v1.2 | 0/? | Not started | - |

---
*Roadmap created: 2026-03-05*
*Last updated: 2026-03-08 -- Phase 17 planned (2 plans)*
