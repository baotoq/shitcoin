# Architecture Patterns

**Domain:** CI/CD and Kubernetes deployment for Go blockchain + React frontend
**Researched:** 2026-03-07

## Recommended Architecture

The CI/CD and K8s layer wraps the existing Go DDD application without modifying domain code. The integration is purely additive: new files at the repository root and in new directories (`deploy/`, `.github/`). No existing Go or React source files need modification.

### High-Level Flow

```
Developer pushes code
       |
       v
GitHub Actions CI
  ├── Test (go test ./...)
  ├── Lint (golangci-lint)
  ├── Build Docker images (backend + frontend)
  └── Push to registry (ghcr.io)
       |
       v
ArgoCD detects manifest changes in deploy/
       |
       v
Kustomize renders final manifests
       |
       v
Kubernetes applies deployment
```

For local development, Tilt replaces the CI/push/ArgoCD portion:

```
Developer edits code
       |
       v
Tilt watches filesystem
  ├── live_update syncs Go files, rebuilds in-container
  ├── live_update syncs React files, Vite HMR handles reload
  └── Deploys to local K8s (kind/k3d)
```

### New File Layout

All new files. Nothing in `internal/`, `cmd/`, or `web/src/` is modified.

```
shitcoin/
├── .github/
│   └── workflows/
│       └── ci.yaml                    # GitHub Actions CI pipeline
├── deploy/
│   ├── docker/
│   │   ├── Dockerfile.backend         # Multi-stage Go build
│   │   ├── Dockerfile.frontend        # Multi-stage React build + nginx
│   │   └── nginx.conf                 # Frontend reverse proxy to backend
│   └── k8s/
│       ├── base/
│       │   ├── kustomization.yaml     # Base resources list + configMapGenerator
│       │   ├── namespace.yaml         # shitcoin namespace
│       │   ├── backend-deployment.yaml
│       │   ├── backend-service.yaml
│       │   ├── frontend-deployment.yaml
│       │   ├── frontend-service.yaml
│       │   └── configs/
│       │       └── shitcoin.yaml      # Config for configMapGenerator
│       └── overlays/
│           ├── dev/
│           │   ├── kustomization.yaml
│           │   └── patches/
│           │       └── backend-resources.yaml
│           └── prod/
│               ├── kustomization.yaml
│               └── patches/
│                   ├── backend-resources.yaml
│                   └── backend-replicas.yaml
├── argocd/
│   └── application.yaml              # ArgoCD Application CR (separate from deploy/)
├── Tiltfile                           # Tilt local dev config (Starlark)
├── .golangci.yml                      # Linter config
└── .dockerignore                      # Docker build context filter
```

### Component Boundaries

| Component | Responsibility | Communicates With |
|-----------|---------------|-------------------|
| GitHub Actions CI | Test, lint, build images, push to registry | ghcr.io, repository webhooks |
| Dockerfile.backend | Multi-stage Go binary build (builder + distroless) | go.mod, cmd/, internal/, etc/ |
| Dockerfile.frontend | Multi-stage React build (node + nginx) | web/package.json, web/src/ |
| nginx.conf | Reverse-proxy /api and /ws to backend Service in K8s | Backend K8s Service |
| Kustomize base | Shared K8s manifests (deployments, services, configmap) | K8s API server |
| Kustomize overlays | Environment-specific patches (resources, replicas, image tags) | Kustomize base |
| ArgoCD Application | GitOps sync controller, watches deploy/k8s/overlays/ | Git repository, K8s API server |
| Tiltfile | Local dev orchestration with live reload | Local K8s cluster (kind/k3d), Dockerfiles, Kustomize |

### Data Flow: Code Change to Running Container

**CI Pipeline (push to master):**

1. Push triggers `.github/workflows/ci.yaml`
2. `test` and `lint` jobs run in parallel (independent)
3. `build` job (depends on test+lint passing) builds two Docker images
4. Images pushed to `ghcr.io/baotoq/shitcoin-backend:sha-<commit>` and `ghcr.io/baotoq/shitcoin-frontend:sha-<commit>`
5. Image tag updated in overlay's `kustomization.yaml` (via CI step or ArgoCD Image Updater)
6. ArgoCD detects manifest diff, syncs to cluster

**Local Dev (Tilt):**

1. `tilt up` builds images from Dockerfiles, deploys via Kustomize dev overlay
2. On Go file change: `live_update` syncs files, runs `go build` in container
3. On React file change: `live_update` syncs to frontend container (Vite HMR)
4. Port-forwards: backend `:8080`, frontend on `:5173` (mapped from nginx `:80`)

## Integration Points with Existing Codebase

### Backend Dockerfile -- What It Needs from the Repo

| Source | Purpose | Docker COPY |
|--------|---------|-------------|
| `go.mod` + `go.sum` | Dependency cache layer (changes rarely) | First COPY for layer caching |
| `cmd/shitcoin/` | Entry point | `/app/cmd/shitcoin/` |
| `internal/` | All domain, handler, infra code | `/app/internal/` |
| `etc/shitcoin.yaml` | Default config (overridden by ConfigMap in K8s) | `/app/etc/` |

**No code changes needed.** The binary is built with `go build ./cmd/shitcoin/` -- same command as local development.

### Frontend Dockerfile -- What It Needs from the Repo

| Source | Purpose | Docker COPY |
|--------|---------|-------------|
| `web/package.json` + `web/package-lock.json` | npm cache layer | First COPY |
| `web/` (all) | React source, config, components | Second COPY |
| Build output: `web/dist/` | Static files served by nginx | Copied to nginx html root |

**No code changes needed.** The existing `npm run build` command produces the `dist/` output.

### Vite Proxy vs Nginx Proxy

The Vite proxy in `web/vite.config.ts` proxies `/api` and `/ws` to `localhost:8080` during development. In production (Docker/K8s), there is no Vite dev server. Nginx serves the static files and proxies API/WS requests to the backend K8s Service.

- **Dev (local, no Docker):** Vite `:5173` proxies to Go `:8080` -- existing behavior, unchanged
- **Dev (Tilt/K8s):** Nginx proxies to `shitcoin-backend:8080` Service
- **Prod (K8s):** Same nginx config

The React app already uses relative URLs (`/api/...`, `/ws`) so no frontend code change is needed.

### Config as ConfigMap

The existing `etc/shitcoin.yaml` maps directly to a Kubernetes ConfigMap via Kustomize's `configMapGenerator`. go-zero's `conf.MustLoad` reads from a file path, so the ConfigMap is volume-mounted at `/app/etc/shitcoin.yaml`.

```yaml
# deploy/k8s/base/configs/shitcoin.yaml (used by configMapGenerator)
Name: shitcoin
Host: 0.0.0.0
Port: 8080
Consensus:
  BlockTimeTarget: 1
  DifficultyAdjustInterval: 10
  InitialDifficulty: 5
Storage:
  DBPath: /data/shitcoin.db
  WalletPath: /data/wallets.json
```

**Key difference from local:** Storage paths point to `/data/` which maps to an emptyDir or PVC volume in K8s, not a relative `data/` directory.

### BoltDB Storage in Containers

BoltDB writes to a single file. In K8s:
- **Dev (emptyDir):** Data is ephemeral, lost on pod restart. Fine for development.
- **Prod (PVC):** PersistentVolumeClaim mounted at `/data/` preserves chain state across restarts.

The `ServiceContext` in `internal/svc/service_context.go` already calls `os.MkdirAll` for the DB directory, so this works without code changes.

### P2P Networking in K8s

The P2P layer listens on TCP port 3000. For the educational scope of v1.1, a single-replica Deployment is sufficient. Multi-node P2P in K8s would require a StatefulSet with a headless Service for stable DNS names -- that is out of scope for this milestone.

The backend Service exposes both port 8080 (HTTP/WS) and port 3000 (P2P) but only HTTP is needed for the frontend.

## Patterns to Follow

### Pattern 1: Multi-Stage Docker Build for Go

**What:** Two-stage Dockerfile separating compilation from runtime.
**When:** Always for the Go backend.
**Why:** Reduces image from ~1GB (golang base) to ~15MB (distroless).

```dockerfile
# deploy/docker/Dockerfile.backend
FROM golang:1.26-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY cmd/ cmd/
COPY internal/ internal/
COPY etc/ etc/
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /shitcoin ./cmd/shitcoin/

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=builder /shitcoin /shitcoin
COPY --from=builder /app/etc/shitcoin.yaml /app/etc/shitcoin.yaml
EXPOSE 8080 3000
ENTRYPOINT ["/shitcoin"]
CMD ["-f", "/app/etc/shitcoin.yaml", "startnode"]
```

**CGO_ENABLED=0 is safe** because bbolt (pure Go), go-zero, gorilla/websocket, and btcec have no CGO dependencies.

### Pattern 2: Multi-Stage Docker Build for React + Nginx

**What:** Build React in Node stage, serve static files from nginx.
**When:** Always for the frontend.

```dockerfile
# deploy/docker/Dockerfile.frontend
FROM node:22-alpine AS builder
WORKDIR /app
COPY web/package.json web/package-lock.json ./
RUN npm ci
COPY web/ ./
RUN npm run build

FROM nginx:alpine
COPY --from=builder /app/dist /usr/share/nginx/html
COPY deploy/docker/nginx.conf /etc/nginx/conf.d/default.conf
EXPOSE 80
```

### Pattern 3: Nginx Reverse Proxy for API/WebSocket

**What:** nginx.conf that serves static files and proxies /api and /ws to the backend.
**When:** Production and K8s dev (replaces Vite proxy).

```nginx
# deploy/docker/nginx.conf
server {
    listen 80;
    root /usr/share/nginx/html;
    index index.html;

    location /api {
        proxy_pass http://shitcoin-backend:8080;
    }

    location /ws {
        proxy_pass http://shitcoin-backend:8080;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";
    }

    location / {
        try_files $uri $uri/ /index.html;
    }
}
```

The backend hostname `shitcoin-backend` is the K8s Service name, resolved via cluster DNS.

### Pattern 4: Kustomize configMapGenerator

**What:** Generate ConfigMaps from files with content-hash suffix.
**When:** For shitcoin.yaml config.
**Why:** Content hash suffix ensures pods restart when config changes.

```yaml
# deploy/k8s/base/kustomization.yaml
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization
namespace: shitcoin
resources:
  - namespace.yaml
  - backend-deployment.yaml
  - backend-service.yaml
  - frontend-deployment.yaml
  - frontend-service.yaml
configMapGenerator:
  - name: shitcoin-config
    files:
      - shitcoin.yaml=configs/shitcoin.yaml
```

### Pattern 5: Parallel CI Jobs with Dependency Gates

**What:** Run test and lint in parallel; build images only if both pass.
**When:** Every CI run.
**Why:** Faster feedback (test and lint are independent).

```yaml
# .github/workflows/ci.yaml structure
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/setup-go@v5
        with: { go-version-file: go.mod, cache: true }
      - run: go test ./...

  lint:
    runs-on: ubuntu-latest
    steps:
      - uses: golangci/golangci-lint-action@v6

  build:
    needs: [test, lint]
    if: github.ref == 'refs/heads/master'
    runs-on: ubuntu-latest
    steps:
      - uses: docker/build-push-action@v6
        # build and push backend + frontend images
```

### Pattern 6: Tiltfile with Live Update

**What:** Tilt watches files, syncs changes into running containers, rebuilds in-place.
**When:** Local K8s development.

```python
# Tiltfile (Starlark)
docker_build(
    'shitcoin-backend',
    '.',
    dockerfile='deploy/docker/Dockerfile.backend',
    live_update=[
        sync('./cmd', '/app/cmd'),
        sync('./internal', '/app/internal'),
        run('cd /app && CGO_ENABLED=0 go build -o /shitcoin ./cmd/shitcoin/',
            trigger=['./cmd', './internal']),
    ],
)

docker_build(
    'shitcoin-frontend',
    '.',
    dockerfile='deploy/docker/Dockerfile.frontend',
    live_update=[
        sync('./web/src', '/app/src'),
    ],
)

k8s_yaml(kustomize('deploy/k8s/overlays/dev'))
k8s_resource('shitcoin-backend', port_forwards=['8080:8080', '3000:3000'])
k8s_resource('shitcoin-frontend', port_forwards=['5173:80'])
```

## Anti-Patterns to Avoid

### Anti-Pattern 1: Single Fat Dockerfile

**What:** One Dockerfile that builds Go, builds React, and runs both.
**Why bad:** Huge image (~1.5GB), cannot scale backend/frontend independently, slow rebuilds.
**Instead:** Separate Dockerfiles per service. Two K8s Deployments.

### Anti-Pattern 2: Baking Data Volumes into Images

**What:** Including BoltDB data files in the Docker image.
**Why bad:** Data is ephemeral, lost on pod restart. Images become huge and stale.
**Instead:** Use emptyDir (dev) or PVC (prod) mounted at `/data/` in K8s.

### Anti-Pattern 3: Hardcoded Backend URL in Frontend Build

**What:** Setting `VITE_API_URL=http://specific-host:8080` at build time.
**Why bad:** Requires rebuild for each environment.
**Instead:** Use relative URLs (`/api/...`). The existing code already does this. Nginx handles proxying.

### Anti-Pattern 4: Using `latest` Image Tag

**What:** Tagging Docker images as `latest` and referencing `latest` in K8s manifests.
**Why bad:** ArgoCD cannot detect changes (same tag). No rollback to specific version.
**Instead:** Tag with git SHA (`sha-abc1234`). Kustomize `images` transformer updates tags per overlay.

### Anti-Pattern 5: ArgoCD Application CR Inside the Watched Path

**What:** Putting `argocd/application.yaml` inside `deploy/k8s/`.
**Why bad:** ArgoCD watches `deploy/k8s/` and would try to manage its own Application resource, causing loops.
**Instead:** Keep `argocd/` at repo root, separate from `deploy/k8s/`.

### Anti-Pattern 6: Running All CI Steps Sequentially

**What:** Test -> Lint -> Build (serial pipeline).
**Why bad:** Wastes time. Test and lint are independent.
**Instead:** Parallel jobs with `needs` dependency on build step.

## Build Order (Dependency Graph for Implementation)

```
Phase 1: Dockerfiles + .dockerignore + .golangci.yml + nginx.conf
    No K8s dependency. Testable with `docker build` locally.
    |
Phase 2: GitHub Actions CI
    Depends on Dockerfiles. Validates build+test+lint in automation.
    |
Phase 3: Kustomize manifests (base + overlays)
    Depends on knowing image names from Phase 1-2.
    Testable with `kubectl apply -k deploy/k8s/overlays/dev --dry-run=client`.
    |
Phase 4: Tiltfile + local K8s dev
    Depends on Dockerfiles (Phase 1) and Kustomize (Phase 3).
    Testable with `tilt up` against a kind/k3d cluster.
    |
Phase 5: ArgoCD Application
    Depends on Kustomize manifests (Phase 3) and images in registry (Phase 2).
    Testable by applying the Application CR to an ArgoCD instance.
```

**Rationale:** Dockerfiles first because everything else (CI, Tilt, K8s) depends on container images. CI second because it validates Dockerfiles in automation and pushes images. Kustomize third because both Tilt and ArgoCD consume its manifests. Tilt before ArgoCD because Tilt provides the local feedback loop for iterating on manifests. ArgoCD last because it is the consumer of all prior artifacts.

## Files: New vs Modified

### New Files (19 files)

| File | Purpose |
|------|---------|
| `.github/workflows/ci.yaml` | CI pipeline definition |
| `.golangci.yml` | golangci-lint configuration |
| `.dockerignore` | Exclude .git/, data/, node_modules/, .planning/ from Docker context |
| `deploy/docker/Dockerfile.backend` | Go multi-stage build |
| `deploy/docker/Dockerfile.frontend` | React multi-stage build + nginx |
| `deploy/docker/nginx.conf` | Frontend reverse proxy to backend Service |
| `deploy/k8s/base/kustomization.yaml` | Base Kustomize config with configMapGenerator |
| `deploy/k8s/base/namespace.yaml` | `shitcoin` namespace |
| `deploy/k8s/base/backend-deployment.yaml` | Backend Deployment (config volume, data volume) |
| `deploy/k8s/base/backend-service.yaml` | Backend ClusterIP Service (ports 8080, 3000) |
| `deploy/k8s/base/frontend-deployment.yaml` | Frontend Deployment (nginx) |
| `deploy/k8s/base/frontend-service.yaml` | Frontend ClusterIP Service (port 80) |
| `deploy/k8s/base/configs/shitcoin.yaml` | Config file for configMapGenerator |
| `deploy/k8s/overlays/dev/kustomization.yaml` | Dev overlay (local images, emptyDir) |
| `deploy/k8s/overlays/dev/patches/backend-resources.yaml` | Dev resource limits |
| `deploy/k8s/overlays/prod/kustomization.yaml` | Prod overlay (registry images, PVC) |
| `deploy/k8s/overlays/prod/patches/backend-resources.yaml` | Prod resource limits |
| `argocd/application.yaml` | ArgoCD Application CR |
| `Tiltfile` | Tilt local dev orchestration |

### Modified Files (1 file)

| File | Change |
|------|--------|
| `.gitignore` | Add `.tilt-dev/` and `tilt_modules/` |

### Existing Files NOT Modified (0 changes to source)

No changes to any file in `cmd/`, `internal/`, `web/src/`, `etc/`, `go.mod`, or `web/package.json`. The CI/CD and K8s layer is entirely additive.

## Scalability Considerations

| Concern | Local Dev (Tilt) | Single-Node K8s | Multi-Node K8s |
|---------|-----------------|-----------------|----------------|
| Storage | emptyDir (ephemeral) | PVC with hostPath | PVC with cloud storage |
| P2P peers | Single replica, no peers | Single replica | StatefulSet + headless Service |
| Frontend scaling | Single replica | Single replica | HPA on CPU |
| Config management | Kustomize dev overlay | Kustomize dev overlay | Kustomize prod overlay |
| Image registry | Local (kind load) | ghcr.io | ghcr.io |

For this educational project, single-replica Deployments are the right scope. Multi-node StatefulSets are an interesting extension but not part of v1.1.

## Sources

- [GitHub Actions CI with Go](https://www.alexedwards.net/blog/ci-with-go-and-github-actions) - HIGH confidence
- [Go CI/CD Best Practices with GitHub Actions](https://dev.to/ticatwolves/automate-your-go-project-best-practices-cicd-with-github-actions-4bo4) - MEDIUM confidence
- [Go Linting with golangci-lint in CI](https://medium.com/@tedious/go-linting-best-practices-for-ci-cd-with-github-actions-aa6d96e0c509) - MEDIUM confidence
- [Tilt Dev Official Site](https://tilt.dev/) - HIGH confidence
- [Local K8s Development with Tilt (2026)](https://oneuptime.com/blog/post/2026-01-19-kubernetes-tilt-local-development/view) - MEDIUM confidence
- [Kustomize Official Docs](https://kubernetes.io/docs/tasks/manage-kubernetes-objects/kustomization/) - HIGH confidence
- [Kustomize Best Practices](https://www.openanalytics.eu/blog/2021/02/23/kustomize-best-practices/) - MEDIUM confidence
- [ArgoCD Kustomize Integration](https://argo-cd.readthedocs.io/en/stable/user-guide/kustomize/) - HIGH confidence
- [GitOps Repo Structure with ArgoCD](https://itnext.io/how-to-structure-your-gitops-repository-with-a-single-argocd-instance-f128b916c915) - MEDIUM confidence
- [Multi-Stage Docker Builds for Go (2026)](https://oneuptime.com/blog/post/2026-01-07-go-docker-multi-stage/view) - MEDIUM confidence
- [Deploying Go to Production (2026)](https://dasroot.net/posts/2026/03/deploying-go-applications-production-best-practices-tools/) - MEDIUM confidence
