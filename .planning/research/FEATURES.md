# Feature Research

**Domain:** CI/CD Pipeline, Docker, Local K8s Dev, K8s Manifests, GitOps Deployment
**Researched:** 2026-03-07
**Confidence:** HIGH

## Feature Landscape

### Table Stakes (Users Expect These)

#### CI Pipeline (GitHub Actions)

| Feature | Why Expected | Complexity | Notes |
|---------|--------------|------------|-------|
| Go test on push/PR | Every Go project runs tests in CI; foundational signal | LOW | `go test ./...` in a workflow; use `actions/setup-go` with Go 1.26, enable module cache |
| Go lint (golangci-lint) | Standard quality gate; catches bugs staticcheck/govet would find | LOW | Use `golangci/golangci-lint-action`; no `.golangci.yml` exists yet so create one with sensible defaults (govet, staticcheck, errcheck, unused, gosimple) |
| Frontend lint + type check | TypeScript errors and ESLint issues must not reach main | LOW | `npm run lint` and `tsc -b` in CI; use `actions/setup-node` with Node 22 LTS |
| Frontend build verification | Proves `npm run build` produces valid output | LOW | Run after lint/typecheck; validates Vite build succeeds |
| Docker image build | CI must prove images actually build; catches Dockerfile regressions | MEDIUM | Build but do not push on PRs; push on main merge. Use `docker/build-push-action` |
| Branch protection via status checks | PRs must pass CI before merge | LOW | Configure required status checks after workflows exist |

#### Docker

| Feature | Why Expected | Complexity | Notes |
|---------|--------------|------------|-------|
| Multi-stage Dockerfile for Go backend | Produces small image (~15MB) with only the binary; standard Go practice | MEDIUM | Stage 1: `golang:1.26-alpine` builds with `CGO_ENABLED=0`. Stage 2: `alpine:3` with ca-certificates + binary. BoltDB data dir needs a volume mount |
| Multi-stage Dockerfile for React frontend | Separates build tooling from nginx runtime; standard SPA practice | MEDIUM | Stage 1: `node:22-alpine` runs `npm ci && npm run build`. Stage 2: `nginx:alpine` serves `dist/`. Custom nginx.conf for SPA routing (try_files) and `/api`+`/ws` reverse proxy to backend |
| .dockerignore files | Prevents bloated build context; avoids leaking secrets | LOW | Exclude `data/`, `node_modules/`, `.git/`, `*.db`, `wallets.json` |
| Non-root container user | Security baseline for any container | LOW | Add `USER nonroot` or numeric UID in both Dockerfiles |

#### K8s Manifests (Kustomize)

| Feature | Why Expected | Complexity | Notes |
|---------|--------------|------------|-------|
| Base manifests (Deployment + Service) for backend | Fundamental K8s resource for running the Go node | MEDIUM | Deployment with 1 replica, liveness/readiness probes on `/api/status`, Service on port 8080. BoltDB needs a PVC or emptyDir |
| Base manifests for frontend | Fundamental K8s resource for serving the React SPA | LOW | Deployment + Service; nginx container on port 80 |
| Kustomize base + dev overlay | Minimum viable Kustomize structure; dev overlay customizes for local | MEDIUM | `k8s/base/` with shared resources, `k8s/overlays/dev/` with lower resource limits and local image refs |
| ConfigMap for app configuration | Externalizes `shitcoin.yaml` from container image | LOW | Mount as volume at `/app/etc/shitcoin.yaml`; overlay patches per environment |
| Resource requests/limits | Required for any production-like K8s deployment | LOW | Backend: 64Mi-256Mi RAM, 100m-500m CPU. Frontend: 32Mi-128Mi RAM, 50m-200m CPU |

#### Local K8s Dev (Tilt)

| Feature | Why Expected | Complexity | Notes |
|---------|--------------|------------|-------|
| Tiltfile with `docker_build` for both services | Core Tilt functionality; builds and deploys to local cluster | MEDIUM | Reference the Dockerfiles and K8s manifests; `docker_build('shitcoin-backend', '.', dockerfile='Dockerfile.backend')` |
| Live update for Go backend | The main reason to use Tilt; rebuild on save without full image rebuild | MEDIUM | `live_update` with `sync` + `run('go build ...')` + restart. Compile cmd must match project entry point |
| Live update for React frontend | Hot reload for frontend development in K8s | MEDIUM | Sync `web/src/` into container, Vite HMR handles the rest. Alternative: just use Vite dev server outside K8s and proxy to K8s backend |
| Tilt UI for log viewing | Built-in to Tilt; no extra work needed | LOW | Comes free with `tilt up`; shows build status, logs, resource health |
| Local K8s cluster setup docs | Developers need to know which local cluster to use | LOW | Document using `kind` or `minikube`; provide a `kind-config.yaml` if using kind |

#### GitOps (ArgoCD)

| Feature | Why Expected | Complexity | Notes |
|---------|--------------|------------|-------|
| ArgoCD Application manifest | Defines what ArgoCD watches and deploys | LOW | Points to repo + `k8s/overlays/dev/` path; uses Kustomize build |
| Automated sync policy | ArgoCD auto-deploys when Git changes; core GitOps value | LOW | `syncPolicy.automated` with `prune: true` and `selfHeal: true` |
| Health checks in ArgoCD | ArgoCD reports deployment health status | LOW | Comes free when K8s probes are configured on Deployments |

### Differentiators (Competitive Advantage)

These are not expected for an educational project but demonstrate deeper DevOps knowledge.

| Feature | Value Proposition | Complexity | Notes |
|---------|-------------------|------------|-------|
| Multi-node testnet in K8s | Runs 3+ blockchain nodes as separate pods communicating via K8s DNS; demonstrates real P2P in cluster | HIGH | Each node gets its own Deployment + Service with unique port and `--peers` flags pointing to other service DNS names. Showcases the P2P networking in a realistic environment |
| CI Docker layer caching | Speeds up CI builds from ~5min to ~1min using GitHub Actions cache | LOW | `docker/build-push-action` with `cache-from: type=gha` and `cache-to: type=gha` |
| Go test coverage reporting | Shows coverage badge and trend in PRs | LOW | `go test -coverprofile=coverage.out ./...` + upload to Codecov or display in PR comment |
| Kustomize prod overlay | Separate overlay with production-like settings (higher replicas, stricter limits) | LOW | `k8s/overlays/prod/` with 2+ replicas, tighter resource limits, pinned image tags |
| Makefile or Taskfile for local commands | Single entry point for build/test/lint/docker commands | LOW | `make ci`, `make docker-build`, `make tilt-up`; reduces README friction |
| ArgoCD ApplicationSet for multi-env | Single definition that generates Application per overlay | MEDIUM | Pattern scales to dev/staging/prod from one manifest; overkill for educational project but demonstrates the concept |
| Security scanning in CI (Trivy) | Scans Docker images for CVEs before deployment | LOW | `aquasecurity/trivy-action` in GitHub Actions; catches vulnerable base images |

### Anti-Features (Commonly Requested, Often Problematic)

| Feature | Why Requested | Why Problematic | Alternative |
|---------|---------------|-----------------|-------------|
| Helm charts instead of Kustomize | "Everyone uses Helm" | Helm adds templating complexity (Go templates in YAML) that obscures what is actually deployed; Kustomize patches are declarative and easier to reason about for an educational project | Kustomize with base + overlays; simpler mental model, native kubectl support |
| Skaffold instead of Tilt | "Skaffold is Google-backed" | Skaffold development has slowed; Tilt has better live-update UX, built-in web dashboard, and more active community | Tilt with live_update |
| Docker Compose for local dev | "Simpler than K8s" | Defeats the purpose of learning K8s tooling; Docker Compose does not exercise K8s manifests, probes, or service discovery | Tilt + kind; still local, but actually uses K8s |
| Separate GitOps repo | "Best practice for production" | For a single educational project, a separate repo adds friction without benefit; monorepo with `k8s/` directory is simpler and demonstrates the same GitOps principles | In-repo `k8s/` directory with Kustomize overlays |
| Full Istio/Linkerd service mesh | "Production needs a mesh" | Massive complexity increase for localhost P2P traffic; service mesh solves problems this project does not have (mTLS between services, traffic splitting) | Plain K8s Services; if observability is wanted, add it via Prometheus annotations |
| CI/CD for multiple environments (dev/staging/prod) | "Need proper environment promotion" | Educational project runs locally; multiple environments add pipeline complexity without educational value | Single `dev` overlay; mention prod overlay pattern in docs but do not automate promotion |
| Kubernetes Operators / CRDs | "Automate node management" | Writing a CRD + controller is a massive effort orthogonal to the CI/CD learning goal | StatefulSet or multiple Deployments with static config |
| GHCR/DockerHub push on every commit | "Need images always available" | Wastes CI minutes and storage for an educational project; images are consumed locally | Build and push only on main branch merges or tags; local dev uses Tilt's local builds |

## Feature Dependencies

```
[.dockerignore]
    └──required-by──> [Multi-stage Dockerfile (backend)]
    └──required-by──> [Multi-stage Dockerfile (frontend)]

[Multi-stage Dockerfile (backend)]
    └──required-by──> [CI Docker image build]
    └──required-by──> [Tiltfile docker_build]
    └──required-by──> [K8s Deployment (backend)]

[Multi-stage Dockerfile (frontend)]
    └──required-by──> [CI Docker image build]
    └──required-by──> [Tiltfile docker_build]
    └──required-by──> [K8s Deployment (frontend)]

[K8s base manifests]
    └──required-by──> [Kustomize overlays]
    └──required-by──> [Tiltfile k8s_yaml]
    └──required-by──> [ArgoCD Application]

[Kustomize base + dev overlay]
    └──required-by──> [ArgoCD Application manifest]
    └──required-by──> [Tiltfile k8s_yaml reference]

[golangci-lint config]
    └──required-by──> [CI lint job]

[Go test + lint CI]
    └──enhances──> [Branch protection status checks]

[Tilt live_update]
    └──requires──> [Tiltfile docker_build]
    └──requires──> [K8s base manifests]
    └──requires──> [Local K8s cluster (kind)]

[ArgoCD Application]
    └──requires──> [Kustomize overlays]
    └──requires──> [ArgoCD installed in cluster]
```

### Dependency Notes

- **Dockerfiles must exist before CI can build images:** CI workflow references Dockerfile paths; build them first.
- **K8s manifests must exist before Tilt or ArgoCD can use them:** Tilt's `k8s_yaml()` and ArgoCD's Application both point to manifest paths.
- **Kustomize overlays require base:** Overlays patch the base; base must be correct and complete first.
- **Tilt requires a local K8s cluster:** `kind create cluster` or equivalent must happen before `tilt up`.
- **ArgoCD requires itself installed:** ArgoCD must be running in the cluster to process Application resources; install it via `kubectl apply` or Helm in kind.
- **Frontend nginx.conf replaces Vite proxy:** In production/K8s, Vite dev proxy does not exist. The nginx config must proxy `/api` and `/ws` to the backend K8s Service name.

## MVP Definition

### Launch With (v1.1 Core)

- [ ] Multi-stage Dockerfiles for Go backend and React frontend -- everything else depends on containerization
- [ ] `.dockerignore` files -- prevents leaking `data/`, `wallets.json` into images
- [ ] `golangci-lint` config (`.golangci.yml`) -- enables CI linting
- [ ] GitHub Actions CI workflow (test, lint, build Docker) -- core CI pipeline
- [ ] Kustomize base manifests (Deployment, Service, ConfigMap for both services) -- K8s foundation
- [ ] Kustomize dev overlay -- local development configuration
- [ ] Tiltfile with `docker_build` and `live_update` for both services -- local K8s dev loop
- [ ] `kind` cluster config and setup instructions -- local cluster for Tilt
- [ ] ArgoCD Application manifest pointing to dev overlay -- demonstrates GitOps

### Add After Validation (v1.1.x)

- [ ] CI Docker layer caching -- when CI build times become annoying (>3 min)
- [ ] Go test coverage reporting -- when wanting to track test quality trends
- [ ] Kustomize prod overlay -- when demonstrating multi-environment patterns
- [ ] Makefile/Taskfile -- when command list grows beyond 5-6 common operations

### Future Consideration (v2+)

- [ ] Multi-node testnet in K8s -- when wanting to showcase P2P in a real cluster
- [ ] ArgoCD ApplicationSet for multi-env -- only if adding staging/prod environments
- [ ] Trivy security scanning -- nice-to-have for educational depth
- [ ] CI image push to GHCR -- only if sharing images outside local dev

## Feature Prioritization Matrix

| Feature | User Value | Implementation Cost | Priority |
|---------|------------|---------------------|----------|
| Multi-stage Dockerfiles (both) | HIGH | MEDIUM | P1 |
| .dockerignore | MEDIUM | LOW | P1 |
| golangci-lint config | MEDIUM | LOW | P1 |
| GitHub Actions CI (test/lint/build) | HIGH | LOW | P1 |
| Kustomize base manifests | HIGH | MEDIUM | P1 |
| Kustomize dev overlay | HIGH | LOW | P1 |
| Tiltfile with live_update | HIGH | MEDIUM | P1 |
| kind cluster config | MEDIUM | LOW | P1 |
| ArgoCD Application | MEDIUM | LOW | P1 |
| CI Docker layer caching | MEDIUM | LOW | P2 |
| Makefile/Taskfile | MEDIUM | LOW | P2 |
| Go test coverage | LOW | LOW | P2 |
| Kustomize prod overlay | LOW | LOW | P2 |
| Multi-node testnet in K8s | MEDIUM | HIGH | P3 |
| ArgoCD ApplicationSet | LOW | MEDIUM | P3 |
| Trivy scanning | LOW | LOW | P3 |

**Priority key:**
- P1: Must have for v1.1 milestone completion
- P2: Should have, add when possible within milestone
- P3: Nice to have, defer to future milestone

## Existing Codebase Dependencies

| New Feature | Depends On (Existing) | Notes |
|-------------|----------------------|-------|
| Backend Dockerfile | `cmd/shitcoin/main.go`, `go.mod`, `go.sum` | Entry point and dependencies for `go build` |
| Backend Dockerfile | `etc/shitcoin.yaml` | Config file; mount via ConfigMap in K8s, copy in Docker for standalone use |
| Backend Dockerfile | `data/` directory | BoltDB writes here at runtime; needs writable volume in K8s |
| Frontend Dockerfile | `web/package.json`, `web/package-lock.json` | Dependencies for `npm ci` |
| Frontend Dockerfile | `web/vite.config.ts` proxy settings | Proxy config is dev-only; production nginx.conf replaces it |
| K8s backend Deployment | Port 8080 (HTTP + WS) | Configured in `etc/shitcoin.yaml`; Service must expose this |
| K8s frontend Deployment | Nginx must proxy `/api` and `/ws` to backend Service | Replaces Vite dev proxy; nginx.conf needed |
| CI Go test | `go test ./...` | Already works locally; just run in CI |
| CI frontend lint | `npm run lint` (ESLint 9) | Already configured in `web/` |
| Tilt live_update (backend) | `go build -o shitcoin cmd/shitcoin/main.go` | Binary name and path matter for restart |
| P2P in K8s | `--peers` CLI flag on `startnode` | Nodes discover peers via K8s Service DNS names |

## Sources

- [Tilt Go Example](https://docs.tilt.dev/example_go.html)
- [Tilt Getting Started](https://docs.tilt.dev/)
- [GitHub Actions Go CI Pipeline](https://oneuptime.com/blog/post/2025-12-20-go-ci-pipeline-github-actions/view)
- [Go Linting Best Practices for CI/CD](https://medium.com/@tedious/go-linting-best-practices-for-ci-cd-with-github-actions-aa6d96e0c509)
- [GitHub Actions CI/CD Best Practices for Docker & K8s](https://github.com/orgs/community/discussions/184874)
- [Docker Multi-Stage Builds](https://docs.docker.com/build/building/multi-stage/)
- [React Vite + Docker + Nginx Production Guide](https://www.buildwithmatija.com/blog/production-react-vite-docker-deployment)
- [Kustomize Tutorial](https://devopscube.com/kustomize-tutorial/)
- [Kustomize Best Practices](https://pauldally.medium.com/kustomize-best-practices-part-2-c560f1fa1409)
- [ArgoCD Kustomize Integration](https://argo-cd.readthedocs.io/en/stable/user-guide/kustomize/)
- [ArgoCD Complete Guide 2026](https://devtoolbox.dedyn.io/blog/argocd-complete-guide)
- [Tilt Alternatives Comparison 2026](https://northflank.com/blog/tilt-alternatives)

---
*Feature research for: CI/CD & Kubernetes tooling (v1.1 milestone)*
*Researched: 2026-03-07*
