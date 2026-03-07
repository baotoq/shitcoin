# Requirements: Shitcoin

**Defined:** 2026-03-07
**Core Value:** A working blockchain you built and understand end-to-end -- from transaction creation to block mining to peer synchronization.

## v1.1 Requirements

Requirements for CI/CD & Kubernetes milestone. Each maps to roadmap phases.

### Docker

- [ ] **DOCK-01**: Multi-stage Dockerfile produces minimal Go backend image (~15MB) with CGO_ENABLED=0
- [ ] **DOCK-02**: Multi-stage Dockerfile produces React frontend image with nginx serving SPA
- [ ] **DOCK-03**: .dockerignore excludes data/, wallets.json, .git, node_modules from build context
- [ ] **DOCK-04**: nginx.conf provides SPA try_files routing and reverse proxies /api and /ws to backend
- [ ] **DOCK-05**: Both containers run as non-root user

### CI Pipeline

- [ ] **CI-01**: GitHub Actions runs go test ./... on push and PR
- [ ] **CI-02**: GitHub Actions runs golangci-lint with project .golangci.yml config
- [ ] **CI-03**: GitHub Actions runs frontend lint, typecheck, and build verification
- [ ] **CI-04**: GitHub Actions builds Docker images on PR and pushes to GHCR on main merge
- [ ] **CI-05**: Go test coverage is reported in CI output

### Kubernetes

- [ ] **K8S-01**: Kustomize base defines Deployment + Service for backend and frontend
- [ ] **K8S-02**: Kustomize base includes PVC for BoltDB data persistence
- [ ] **K8S-03**: Kustomize base uses configMapGenerator to externalize shitcoin.yaml
- [ ] **K8S-04**: Backend Deployment uses Recreate strategy with single replica for BoltDB safety
- [ ] **K8S-05**: Kustomize dev overlay configures local image refs and lower resource limits
- [ ] **K8S-06**: Kustomize prod overlay configures pinned image tags and production resource limits
- [ ] **K8S-07**: Health probes (liveness + readiness) configured on /api/status

### Local Dev

- [ ] **DEV-01**: Tiltfile with docker_build and live_update for Go backend hot reload
- [ ] **DEV-02**: Tiltfile with docker_build and live_update for React frontend
- [ ] **DEV-03**: kind cluster config and setup instructions provided
- [ ] **DEV-04**: Makefile with common commands (ci, docker-build, tilt-up, lint, test)

### GitOps

- [ ] **GIT-01**: ArgoCD Application CR with auto-sync pointing to Kustomize dev overlay
- [ ] **GIT-02**: ArgoCD Application CR lives outside K8s manifest watched path

## v2 Requirements

Deferred to future release. Tracked but not in current roadmap.

### Kubernetes Advanced

- **K8S-ADV-01**: Multi-node blockchain testnet in K8s with StatefulSet and headless Service
- **K8S-ADV-02**: ArgoCD ApplicationSet for multi-environment promotion

### CI Advanced

- **CI-ADV-01**: Trivy security scanning for Docker images in CI
- **CI-ADV-02**: ArgoCD Image Updater for automated image tag updates

## Out of Scope

| Feature | Reason |
|---------|--------|
| Helm charts | Kustomize is simpler and more educational; avoids Go template complexity in YAML |
| Docker Compose | Defeats K8s learning purpose; Tilt provides same local dev convenience with real K8s |
| Skaffold | Tilt has better live-update UX and active community |
| Service mesh (Istio/Linkerd) | Massive complexity for localhost P2P; solves problems this project doesn't have |
| Separate GitOps repo | Unnecessary friction for single educational project |
| Multi-environment CI promotion | Educational project runs locally; single dev overlay sufficient |

## Traceability

Which phases cover which requirements. Updated during roadmap creation.

| Requirement | Phase | Status |
|-------------|-------|--------|
| DOCK-01 | Phase 9 | Pending |
| DOCK-02 | Phase 9 | Pending |
| DOCK-03 | Phase 9 | Pending |
| DOCK-04 | Phase 9 | Pending |
| DOCK-05 | Phase 9 | Pending |
| CI-01 | Phase 10 | Pending |
| CI-02 | Phase 10 | Pending |
| CI-03 | Phase 10 | Pending |
| CI-04 | Phase 10 | Pending |
| CI-05 | Phase 10 | Pending |
| K8S-01 | Phase 11 | Pending |
| K8S-02 | Phase 11 | Pending |
| K8S-03 | Phase 11 | Pending |
| K8S-04 | Phase 11 | Pending |
| K8S-05 | Phase 11 | Pending |
| K8S-06 | Phase 11 | Pending |
| K8S-07 | Phase 11 | Pending |
| DEV-01 | Phase 12 | Pending |
| DEV-02 | Phase 12 | Pending |
| DEV-03 | Phase 12 | Pending |
| DEV-04 | Phase 12 | Pending |
| GIT-01 | Phase 13 | Pending |
| GIT-02 | Phase 13 | Pending |

**Coverage:**
- v1.1 requirements: 23 total
- Mapped to phases: 23
- Unmapped: 0

---
*Requirements defined: 2026-03-07*
*Last updated: 2026-03-07 after roadmap creation*
