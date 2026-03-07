# Roadmap: Shitcoin

## Milestones

- ✅ **v1.0 Educational Blockchain** -- Phases 1-6 (shipped 2026-03-07)
- 🚧 **v1.1 CI/CD & Kubernetes** -- Phases 9-13 (in progress)

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

### 🚧 v1.1 CI/CD & Kubernetes

**Milestone Goal:** Add CI/CD pipeline, local K8s development, and GitOps deployment to the blockchain project.

- [x] **Phase 9: Containerization** - Multi-stage Dockerfiles for Go backend and React frontend with nginx reverse proxy (completed 2026-03-07)
- [x] **Phase 10: CI Pipeline** - GitHub Actions for test, lint, build, and image push to GHCR (completed 2026-03-07)
- [ ] **Phase 11: Kubernetes Manifests** - Kustomize base + overlays defining complete K8s deployment
- [ ] **Phase 12: Local K8s Development** - Tilt with live-update for fast iteration on a local kind cluster
- [ ] **Phase 13: GitOps Deployment** - ArgoCD Application for automated sync from git to cluster

## Phase Details

### Phase 9: Containerization
**Goal**: Both services produce minimal, secure container images that run correctly
**Depends on**: Nothing (first phase of v1.1)
**Requirements**: DOCK-01, DOCK-02, DOCK-03, DOCK-04, DOCK-05
**Success Criteria** (what must be TRUE):
  1. `docker build` produces a working Go backend image under 20MB that starts and serves /api/status
  2. `docker build` produces a working React frontend image with nginx that serves the SPA and proxies /api and /ws to the backend
  3. Build context excludes data/, wallets.json, .git, and node_modules (verified via .dockerignore)
  4. Both containers run as a non-root user (verified by `docker exec whoami`)
**Plans:** 2/2 plans complete

Plans:
- [ ] 09-01-PLAN.md -- Backend Dockerfile + .dockerignore
- [ ] 09-02-PLAN.md -- Frontend Dockerfile + nginx.conf

### Phase 10: CI Pipeline
**Goal**: Every push and PR is automatically tested, linted, and built; images are pushed to registry on main merge
**Depends on**: Phase 9
**Requirements**: CI-01, CI-02, CI-03, CI-04, CI-05
**Success Criteria** (what must be TRUE):
  1. A push to any branch triggers Go tests and golangci-lint, with pass/fail visible in GitHub Actions
  2. A push to any branch triggers frontend lint, typecheck, and build verification
  3. Merging to main builds and pushes Docker images to GHCR with correct tags
  4. Go test coverage percentage is reported in CI output
**Plans:** 2/2 plans complete

Plans:
- [ ] 10-01-PLAN.md -- Go CI workflow (test + lint + coverage) and golangci-lint v2 config
- [ ] 10-02-PLAN.md -- Frontend CI workflow and Docker build/push workflow

### Phase 11: Kubernetes Manifests
**Goal**: Complete Kustomize manifest set defines a deployable, persistent, health-checked blockchain node
**Depends on**: Phase 9
**Requirements**: K8S-01, K8S-02, K8S-03, K8S-04, K8S-05, K8S-06, K8S-07
**Success Criteria** (what must be TRUE):
  1. `kubectl apply -k deploy/k8s/overlays/dev` creates running backend and frontend pods with Services
  2. Backend pod uses Recreate strategy with single replica and mounts a PVC for BoltDB data persistence
  3. Backend config (shitcoin.yaml) is externalized via ConfigMap, not baked into the image
  4. Liveness and readiness probes on /api/status report healthy for a running backend
  5. `kubectl apply -k deploy/k8s/overlays/prod` applies production resource limits and pinned image tags
**Plans**: TBD

Plans:
- [ ] 11-01: TBD
- [ ] 11-02: TBD

### Phase 12: Local K8s Development
**Goal**: Developer runs `tilt up` and gets a live-reloading blockchain environment on a local K8s cluster
**Depends on**: Phase 9, Phase 11
**Requirements**: DEV-01, DEV-02, DEV-03, DEV-04
**Success Criteria** (what must be TRUE):
  1. `tilt up` builds and deploys both services to a local kind cluster with port-forwarding
  2. Editing a Go source file triggers automatic rebuild and restart in the backend container without a full image rebuild
  3. Editing a React source file triggers automatic update in the frontend container
  4. `make tilt-up`, `make test`, `make lint`, and `make docker-build` all work as documented
**Plans**: TBD

Plans:
- [ ] 12-01: TBD

### Phase 13: GitOps Deployment
**Goal**: ArgoCD automatically syncs Kubernetes state from git, completing the CI/CD loop
**Depends on**: Phase 11
**Requirements**: GIT-01, GIT-02
**Success Criteria** (what must be TRUE):
  1. ArgoCD Application CR points to the Kustomize dev overlay and auto-syncs changes from git
  2. The ArgoCD Application manifest lives in argocd/ directory, outside the K8s manifest path that ArgoCD watches
**Plans**: TBD

Plans:
- [ ] 13-01: TBD

## Progress

**Execution Order:**
Phases execute in numeric order: 9 -> 10 -> 11 -> 12 -> 13
(Phases 10 and 11 can begin in parallel after Phase 9 completes)

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
| 10. CI Pipeline | 2/2 | Complete   | 2026-03-07 | - |
| 11. Kubernetes Manifests | v1.1 | 0/2 | Not started | - |
| 12. Local K8s Development | v1.1 | 0/1 | Not started | - |
| 13. GitOps Deployment | v1.1 | 0/1 | Not started | - |

---
*Roadmap created: 2026-03-05*
*Last updated: 2026-03-07 -- Phase 10 planned (2 plans)*
