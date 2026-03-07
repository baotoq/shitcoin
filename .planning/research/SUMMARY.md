# Project Research Summary

**Project:** Shitcoin v1.1 -- CI/CD & Kubernetes
**Domain:** CI/CD pipeline, containerization, local K8s development, and GitOps deployment for an existing Go blockchain + React frontend
**Researched:** 2026-03-07
**Confidence:** HIGH

## Executive Summary

This milestone wraps the existing Go blockchain node and React block explorer in production-grade CI/CD and Kubernetes infrastructure. The work is entirely additive -- no existing source code in `cmd/`, `internal/`, or `web/src/` needs modification. The proven approach is: multi-stage Docker builds for both services, GitHub Actions for CI (test, lint, build, push), Kustomize for K8s manifest management, Tilt for local K8s development with live reload, and ArgoCD for GitOps continuous delivery. All tooling is well-documented, widely adopted, and free for open-source projects.

The recommended approach follows a strict dependency chain: Dockerfiles first (everything depends on container images), then CI pipeline (validates builds in automation and pushes to registry), then Kustomize manifests (consumed by both Tilt and ArgoCD), then Tilt for local dev workflow, and finally ArgoCD for GitOps. This ordering is not arbitrary -- each layer depends on artifacts from the previous one. Skipping ahead (e.g., writing Kustomize manifests before Dockerfiles exist) creates untestable code.

The primary risks center on BoltDB's single-writer file locking model conflicting with Kubernetes deployment patterns. Using the default `RollingUpdate` strategy or multiple replicas will cause database lock contention and data corruption. The mitigation is straightforward: use `Recreate` strategy, single replica, and proper SIGTERM handling. Secondary risks include the classic SPA-on-nginx 404 issue (solved with `try_files`) and CGO-linked binaries crashing on minimal runtime images (solved with `CGO_ENABLED=0`). All pitfalls have well-known, low-cost preventions when addressed at the right phase.

## Key Findings

### Recommended Stack

The stack adds 6 categories of tooling without modifying the existing Go + React codebase. All tools are current stable releases as of March 2026.

**Core technologies:**
- **GitHub Actions** (ubuntu-latest): CI execution -- free for public repos, native Docker/buildx support
- **golangci-lint v2.11**: Go linting -- runs 100+ linters in one pass, official GitHub Action for CI integration
- **Multi-stage Dockerfiles** (golang:1.26-alpine + alpine:3.23 / node:22-alpine + nginx:alpine): Containerization -- produces ~15MB Go image and ~25MB frontend image
- **Kustomize v5.8**: K8s manifest management -- built into kubectl, base+overlay pattern without templating language
- **Tilt v0.37.0**: Local K8s dev -- live-update syncs code into running containers without full image rebuilds
- **ArgoCD v3.3**: GitOps CD -- auto-syncs K8s manifests from git, web UI for deployment visualization
- **GHCR** (ghcr.io): Container registry -- native GitHub integration, free, no rate limits with GITHUB_TOKEN

### Expected Features

**Must have (table stakes):**
- Multi-stage Dockerfiles for Go backend and React frontend
- `.dockerignore` to prevent leaking `data/`, `wallets.json` into images
- GitHub Actions CI pipeline (go test, golangci-lint, frontend lint, Docker build)
- Kustomize base manifests (Deployment, Service, ConfigMap for both services)
- Kustomize dev overlay for local K8s configuration
- Tiltfile with `docker_build` and `live_update` for both services
- ArgoCD Application manifest pointing to dev overlay

**Should have (differentiators):**
- CI Docker layer caching (GHA cache backend)
- Go test coverage reporting
- Kustomize prod overlay
- Makefile/Taskfile for common commands

**Defer (v2+):**
- Multi-node testnet in K8s (StatefulSet with headless Service)
- ArgoCD ApplicationSet for multi-environment promotion
- Trivy security scanning in CI

### Architecture Approach

The architecture is a purely additive layer: ~19 new files in `.github/`, `deploy/`, `argocd/`, and project root. No modifications to existing source. The flow is: developer pushes code, GitHub Actions runs test/lint/build in parallel, pushes images to GHCR, ArgoCD detects manifest changes, Kustomize renders final manifests, Kubernetes applies the deployment. For local dev, Tilt replaces the CI/push/ArgoCD chain with filesystem watching and live container updates.

**Major components:**
1. **GitHub Actions CI** -- test, lint, build images, push to GHCR; parallel jobs with dependency gates
2. **Docker images** (2 Dockerfiles) -- multi-stage builds producing minimal runtime images; nginx reverse-proxies /api and /ws to backend
3. **Kustomize manifests** (base + overlays) -- Deployments, Services, ConfigMap via configMapGenerator; dev overlay for local, prod overlay for production-like settings
4. **Tiltfile** -- local K8s dev orchestration with live_update for Go rebuild-in-container and React HMR
5. **ArgoCD Application** -- GitOps sync controller watching `deploy/k8s/overlays/` path, auto-sync with prune and self-heal

### Critical Pitfalls

1. **BoltDB file locking blocks rolling updates** -- Use `strategy.type: Recreate` and `replicas: 1` on backend Deployment. Trap SIGTERM and call `db.Close()`. Set `terminationGracePeriodSeconds: 30`.
2. **BoltDB data loss without persistent volumes** -- Define PVC in Kustomize base, mount at `/data/`. Config paths must reference the mount. Use emptyDir for dev if data loss is acceptable.
3. **CGO-linked binary crashes on minimal images** -- Set `CGO_ENABLED=0` explicitly in Dockerfile. All project dependencies (bbolt, go-zero, btcec) are pure Go.
4. **SPA 404 on nginx page refresh** -- Include `try_files $uri $uri/ /index.html` in nginx.conf. Also add WebSocket upgrade headers for `/ws` proxy.
5. **ArgoCD perpetual OutOfSync** -- Configure `ignoreDifferences` for server-mutated fields. Pin Kustomize version to match local. Test sync stability before declaring done.

## Implications for Roadmap

Based on research, suggested phase structure:

### Phase 1: Containerization (Dockerfiles + Config)

**Rationale:** Everything downstream depends on container images. Dockerfiles, .dockerignore, nginx.conf, and .golangci.yml are independent of K8s and can be validated with `docker build` locally.
**Delivers:** Buildable Docker images for both services, linter configuration, Docker context hygiene.
**Addresses:** Multi-stage Dockerfiles (backend + frontend), .dockerignore, nginx.conf for SPA + API/WS proxy, .golangci.yml.
**Avoids:** CGO binary crash (set CGO_ENABLED=0 from day one), SPA 404 (nginx.conf with try_files), wallet keys in image layers (.dockerignore), WebSocket proxy breakage (upgrade headers).

### Phase 2: CI Pipeline (GitHub Actions)

**Rationale:** CI validates Dockerfiles in automation and pushes images to GHCR. Depends on Dockerfiles from Phase 1. Independent of K8s manifests.
**Delivers:** Automated test, lint, and Docker build on PR; image push to GHCR on main merge.
**Uses:** GitHub Actions, golangci-lint-action, docker/build-push-action, docker/metadata-action, GHCR.
**Avoids:** CI cache collisions (configure caching strategy upfront), excessive image pushes (only push on main/tags).

### Phase 3: Kubernetes Manifests (Kustomize)

**Rationale:** Both Tilt and ArgoCD consume Kustomize manifests. Must exist before either can be configured. Depends on knowing image names from Phase 1-2.
**Delivers:** Complete K8s deployment definition: namespace, Deployments, Services, ConfigMap, PVC, dev and prod overlays.
**Addresses:** Base manifests for both services, configMapGenerator for shitcoin.yaml, dev overlay with local settings, resource requests/limits.
**Avoids:** BoltDB file locking (Recreate strategy), BoltDB data loss (PVC definition), P2P load-balancing issues (expose P2P port but single replica for v1.1).

### Phase 4: Local K8s Development (Tilt)

**Rationale:** Tilt provides the local feedback loop for iterating on K8s manifests and Dockerfiles. Depends on Dockerfiles (Phase 1) and Kustomize (Phase 3).
**Delivers:** `tilt up` workflow with live-update for Go and React, port-forwarding, .tiltignore, kind cluster setup docs.
**Uses:** Tilt, ctlptl, Docker Desktop K8s or kind.
**Avoids:** Full image rebuild on every change (live_update), Tilt watching node_modules (.tiltignore).

### Phase 5: GitOps Deployment (ArgoCD)

**Rationale:** ArgoCD is the consumer of all prior artifacts -- images in registry (Phase 2) and Kustomize manifests (Phase 3). Last because it requires everything else to be working.
**Delivers:** ArgoCD Application CR with auto-sync, deployment visualization, health checks.
**Uses:** ArgoCD v3.3, Kustomize integration.
**Avoids:** ArgoCD sync loops (ignoreDifferences config), Application CR inside watched path (separate argocd/ directory).

### Phase Ordering Rationale

- Phases follow a strict dependency chain: Dockerfiles -> CI -> Kustomize -> Tilt -> ArgoCD. Each phase produces artifacts consumed by later phases.
- Dockerfiles and CI are grouped early because they are independently testable without a K8s cluster.
- Kustomize comes before Tilt and ArgoCD because both consume its manifests -- writing manifests once and testing with two consumers is more efficient than the reverse.
- Tilt before ArgoCD because Tilt provides the fast local iteration loop needed to debug manifest issues before committing them for ArgoCD to sync.
- BoltDB pitfalls (locking, data loss) must be addressed in Phase 3 (Kustomize manifests) -- not deferred to later phases.

### Research Flags

Phases likely needing deeper research during planning:
- **Phase 3 (Kustomize):** BoltDB volume management and Recreate strategy interaction needs careful manifest design. P2P port exposure strategy (single replica vs headless Service) warrants validation.
- **Phase 4 (Tilt):** Live-update for Go backend (sync + recompile in container) has nuances around binary path and restart behavior. Tilt + Kustomize integration (`k8s_yaml(kustomize(...))`) should be tested early.

Phases with standard patterns (skip research-phase):
- **Phase 1 (Dockerfiles):** Multi-stage Docker builds for Go and React+nginx are extremely well-documented. Patterns are established and verified.
- **Phase 2 (CI):** GitHub Actions CI for Go projects is a solved problem. Official actions exist for every step.
- **Phase 5 (ArgoCD):** ArgoCD Application CR with Kustomize is a standard configuration. Official docs cover it thoroughly.

## Confidence Assessment

| Area | Confidence | Notes |
|------|------------|-------|
| Stack | HIGH | All tools are current stable releases with verified version numbers. Docker Hub images confirmed available. |
| Features | HIGH | Feature set is well-scoped with clear dependencies. MVP is achievable within the milestone. |
| Architecture | HIGH | Purely additive architecture with no source code changes. File layout and component boundaries are clear. |
| Pitfalls | HIGH | BoltDB-specific pitfalls are well-documented in etcd/bbolt community. K8s and Docker pitfalls are standard knowledge. |

**Overall confidence:** HIGH

### Gaps to Address

- **Graceful BoltDB shutdown:** The existing Go code may not trap SIGTERM and call `db.Close()`. This needs verification during Phase 3 implementation. If missing, a small code change in `cmd/shitcoin/main.go` is required -- the only potential source modification in this milestone.
- **P2P multi-node in K8s:** Deferred to v2+, but the headless Service + StatefulSet pattern should be documented for future reference. Phase 3 should expose the P2P port (3000) even if multi-node is not implemented.
- **Frontend live-update strategy:** Two options exist (Vite HMR inside container vs. Vite dev server outside K8s proxying to K8s backend). The simpler approach (run Vite locally, proxy to K8s backend) may be preferable. Validate during Phase 4.
- **ArgoCD image tag update automation:** How CI updates the image tag in Kustomize overlays after pushing a new image is not fully specified. Options: CI commits tag change to repo, ArgoCD Image Updater, or manual. Decide during Phase 5 planning.

## Sources

### Primary (HIGH confidence)
- [bbolt GitHub](https://github.com/etcd-io/bbolt) -- file locking, mmap, concurrency model
- [Kubernetes official docs](https://kubernetes.io/docs/) -- Services, PVCs, headless services, Kustomize
- [ArgoCD official docs](https://argo-cd.readthedocs.io/) -- Kustomize integration, sync policies
- [Tilt official docs](https://docs.tilt.dev/) -- live_update, choosing clusters, FAQ
- [GitHub Actions runner images](https://github.com/actions/runner-images) -- ubuntu-latest = Ubuntu 24.04
- [Docker multi-stage builds guide](https://docs.docker.com/build/building/multi-stage/)
- [7 Common K8s Pitfalls (official blog)](https://kubernetes.io/blog/2025/10/20/seven-kubernetes-pitfalls-and-how-to-avoid/)

### Secondary (MEDIUM confidence)
- [GitHub Actions Go CI pipeline guides](https://oneuptime.com/blog/post/2025-12-20-go-ci-pipeline-github-actions/view) -- workflow structure
- [Kustomize best practices](https://pauldally.medium.com/kustomize-best-practices-part-2-c560f1fa1409) -- overlay patterns
- [Go linting in CI](https://medium.com/@tedious/go-linting-best-practices-for-ci-cd-with-github-actions-aa6d96e0c509) -- golangci-lint configuration
- [React Vite + Docker + Nginx production guide](https://www.buildwithmatija.com/blog/production-react-vite-docker-deployment) -- frontend containerization
- [GitOps repo structure with ArgoCD](https://itnext.io/how-to-structure-your-gitops-repository-with-a-single-argocd-instance-f128b916c915) -- Application CR placement

---
*Research completed: 2026-03-07*
*Ready for roadmap: yes*
