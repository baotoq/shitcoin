# Requirements: Shitcoin

**Defined:** 2026-03-08
**Core Value:** A working blockchain you built and understand end-to-end — from transaction creation to block mining to peer synchronization.

## v1.2 Requirements

Requirements for Testing & Quality milestone. Each maps to roadmap phases.

### Test Infrastructure

- [ ] **TINF-01**: Shared test helpers with reusable block, tx, wallet, and UTXO builders in `internal/testutil/`
- [ ] **TINF-02**: Consolidated mock repositories (chain, UTXO, wallet) in shared `testutil` package, replacing duplicated mocks across 4+ packages
- [ ] **TINF-03**: Race detection enabled in CI (`go test -race ./...` in GitHub Actions)

### Domain Layer

- [ ] **DOM-01**: Chain aggregate test coverage reaches 85%+ (mining orchestration, reorg logic, difficulty adjustment edge cases)
- [ ] **DOM-02**: P2P unit test coverage reaches 80%+ (message encoding/decoding, handler dispatch, sync logic)
- [ ] **DOM-03**: Domain gap-filling brings utxo, wallet, mempool, and tx packages to 95%+ coverage each
- [ ] **DOM-04**: Error path tests cover invalid blocks, double spends, corrupt data, nil inputs, and boundary conditions across all domain packages

### Handler Layer

- [ ] **HNDL-01**: API handler test coverage reaches 80%+ (address, mempool, search, tx handlers)
- [ ] **HNDL-02**: WebSocket hub test coverage reaches 75%+ (event subscribe, broadcast, client disconnect)

### Infrastructure Layer

- [ ] **INFR-01**: BoltDB repository test coverage reaches 80%+ (atomic block+UTXO saves, range queries, reorg deletes, undo entries)
- [ ] **INFR-02**: JSON file wallet repository test coverage reaches 90%+

### Integration Tests

- [ ] **INTG-01**: P2P integration tests verify TCP handshake, block sync, and tx relay between 2+ in-process nodes
- [ ] **INTG-02**: E2E chain scenario tests verify full workflow: create wallet, send tx, mine block, verify UTXO updated, check balance

## v1.1 Requirements (Complete)

### Docker

- [x] **DOCK-01**: Multi-stage Dockerfile produces minimal Go backend image (~15MB) with CGO_ENABLED=0
- [x] **DOCK-02**: Multi-stage Dockerfile produces React frontend image with nginx serving SPA
- [x] **DOCK-03**: .dockerignore excludes data/, wallets.json, .git, node_modules from build context
- [x] **DOCK-04**: nginx.conf provides SPA try_files routing and reverse proxies /api and /ws to backend
- [x] **DOCK-05**: Both containers run as non-root user

### CI Pipeline

- [x] **CI-01**: GitHub Actions runs go test ./... on push and PR
- [x] **CI-02**: GitHub Actions runs golangci-lint with project .golangci.yml config
- [x] **CI-03**: GitHub Actions runs frontend lint, typecheck, and build verification
- [x] **CI-04**: GitHub Actions builds Docker images on PR and pushes to GHCR on main merge
- [x] **CI-05**: Go test coverage is reported in CI output

### Kubernetes

- [x] **K8S-01**: Kustomize base defines Deployment + Service for backend and frontend
- [x] **K8S-02**: Kustomize base includes PVC for BoltDB data persistence
- [x] **K8S-03**: Kustomize base uses configMapGenerator to externalize shitcoin.yaml
- [x] **K8S-04**: Backend Deployment uses Recreate strategy with single replica for BoltDB safety
- [x] **K8S-05**: Kustomize dev overlay configures local image refs and lower resource limits
- [x] **K8S-06**: Kustomize prod overlay configures pinned image tags and production resource limits
- [x] **K8S-07**: Health probes (liveness + readiness) configured on /api/status

### Local Dev

- [x] **DEV-01**: Tiltfile with docker_build and live_update for Go backend hot reload
- [x] **DEV-02**: Tiltfile with docker_build and live_update for React frontend
- [x] **DEV-03**: kind cluster config and setup instructions provided
- [x] **DEV-04**: Makefile with common commands (ci, docker-build, tilt-up, lint, test)

### GitOps

- [x] **GIT-01**: ArgoCD Application CR with auto-sync pointing to Kustomize dev overlay
- [x] **GIT-02**: ArgoCD Application CR lives outside K8s manifest watched path

## Future Requirements

### Quality Enhancements

- **QUAL-01**: Coverage CI gate — fail CI if coverage drops below per-package thresholds
- **QUAL-02**: Fuzz tests for P2P message deserialization and block serialization
- **QUAL-03**: Golden file tests for wire format snapshot detection
- **QUAL-04**: Benchmark tests for PoW mining at various difficulties
- **QUAL-05**: CLI handler tests for command dispatch

### Kubernetes Advanced

- **K8S-ADV-01**: Multi-node blockchain testnet in K8s with StatefulSet and headless Service
- **K8S-ADV-02**: ArgoCD ApplicationSet for multi-environment promotion

### CI Advanced

- **CI-ADV-01**: Trivy security scanning for Docker images in CI
- **CI-ADV-02**: ArgoCD Image Updater for automated image tag updates

## Out of Scope

| Feature | Reason |
|---------|--------|
| Mock generation framework (mockgen, moq) | Only 3 interfaces; hand-written mocks are established and sufficient |
| Property-based testing (gopter, rapid) | Overkill for educational project; table-driven tests cover same ground |
| Mutation testing (go-mutesting) | Slow, noisy; 11K LOC project doesn't justify overhead |
| Contract/API schema tests (OpenAPI) | No OpenAPI spec; REST API tested directly with httptest |
| svc.ServiceContext tests | Pure wiring code with no logic to test |
| External test infrastructure (testcontainers) | BoltDB is embedded, P2P is localhost; no external deps to containerize |
| CLI orchestration tests (testnet, demo) | High effort, low value; domain logic underneath is what matters |

## Traceability

Which phases cover which requirements. Updated during roadmap creation.

| Requirement | Phase | Status |
|-------------|-------|--------|
| TINF-01 | Phase 14 | Pending |
| TINF-02 | Phase 14 | Pending |
| TINF-03 | Phase 18 | Pending |
| DOM-01 | Phase 15 | Pending |
| DOM-02 | Phase 15 | Pending |
| DOM-03 | Phase 15 | Pending |
| DOM-04 | Phase 15 | Pending |
| HNDL-01 | Phase 17 | Pending |
| HNDL-02 | Phase 17 | Pending |
| INFR-01 | Phase 16 | Pending |
| INFR-02 | Phase 16 | Pending |
| INTG-01 | Phase 18 | Pending |
| INTG-02 | Phase 18 | Pending |

**Coverage:**
- v1.2 requirements: 13 total
- Mapped to phases: 13
- Unmapped: 0

---
*Requirements defined: 2026-03-08*
*Last updated: 2026-03-08 after roadmap creation (Phase 14-18 mappings)*
