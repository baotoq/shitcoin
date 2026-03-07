# Technology Stack

**Project:** Shitcoin v1.1 -- CI/CD & Kubernetes
**Researched:** 2026-03-07
**Scope:** New tooling for CI/CD pipeline, containerization, local K8s dev, and GitOps deployment. Existing Go + React stack is unchanged and NOT covered here.

## Recommended Stack

### CI/CD Pipeline (GitHub Actions)

| Technology | Version | Purpose | Why |
|------------|---------|---------|-----|
| GitHub Actions runner | `ubuntu-latest` (24.04) | CI execution environment | Standard, pre-installed Docker/buildx, free for public repos |
| `actions/checkout` | `v4` | Repo checkout | Current stable major version |
| `actions/setup-go` | `v6` | Go toolchain setup with caching | Latest major version (2026), built-in module caching via `cache: true` default |
| `golangci/golangci-lint-action` | `v9` | Go linting in CI | Official action from golangci-lint authors, caches lint results, adds line annotations on PRs |
| golangci-lint | `v2.11` | Go linter binary (installed by action) | Latest stable (2026-03-06), supports Go 1.26.1 |
| `actions/setup-node` | `v4` | Node.js for frontend build/lint | Current stable, built-in npm caching |
| `docker/setup-buildx-action` | `v3` | Docker Buildx builder | Multi-platform build support, build cache integration |
| `docker/build-push-action` | `v6` | Build and push images | Latest stable, full BuildKit features, GHCR push |
| `docker/login-action` | `v3` | Registry authentication | GHCR login using GITHUB_TOKEN, zero config |
| `docker/metadata-action` | `v5` | Automated image tagging | Generates tags from git SHA, branch, semver tags automatically |

### Container Images (Docker Multi-Stage Builds)

| Technology | Version | Purpose | Why |
|------------|---------|---------|-----|
| `golang:1.26-alpine` | 1.26.1 on Alpine 3.23 | Go build stage | Matches project Go 1.26.1 exactly. Alpine base keeps build layer small. Verified available on Docker Hub. |
| `alpine:3.23` | 3.23 | Go runtime stage | Minimal runtime (~7MB). Not `scratch` because ca-certificates and tzdata are useful for network apps. |
| `node:22-alpine` | 22 LTS on Alpine 3.23 | React build stage | Node 22 LTS is current. Vite 7 + React 19 build works on Node 22. Alpine for size. |
| `nginx:alpine` | Latest Alpine | React SPA runtime | Serves static dist/ files. Handles SPA routing with try_files. Proxies /api and /ws to Go backend. |

### Local K8s Development

| Technology | Version | Purpose | Why |
|------------|---------|---------|-----|
| Tilt | v0.37.0 | Local K8s dev orchestration | Live updates without full image rebuilds (sync Go binary or React build into running containers). Tiltfile-as-code is version-controllable. Built-in web UI for logs/status/errors. Released 2026-03-04. |
| ctlptl | Latest | Local K8s cluster management | From Tilt team. Creates local K8s clusters declaratively. Pairs with Docker Desktop K8s or kind. |
| Docker Desktop K8s | Built-in | Local K8s cluster | Zero extra install on macOS (already have Docker). Single-node cluster sufficient for this project. kind is the fallback if Docker Desktop unavailable. |

### GitOps Deployment

| Technology | Version | Purpose | Why |
|------------|---------|---------|-----|
| ArgoCD | v3.3 | GitOps continuous delivery | Industry-standard GitOps controller. Auto-syncs K8s manifests from git. Web UI for deployment visualization. Health checks on resources. v3.3.2 is latest (2026-02-22). |
| Kustomize | v5.8 | K8s manifest management | Built into kubectl (`kubectl apply -k`). No templating language -- pure YAML with strategic merge patches. base+overlay pattern for dev/staging/prod. v5.8.0 is latest. |

### Container Registry

| Technology | Version | Purpose | Why |
|------------|---------|---------|-----|
| GHCR (ghcr.io) | N/A | Docker image registry | Native GitHub integration, free for public repos, no rate limits with GITHUB_TOKEN, no separate account needed |

## Alternatives Considered

| Category | Recommended | Alternative | Why Not |
|----------|-------------|-------------|---------|
| K8s manifests | Kustomize | Helm | Helm charts add a templating language (Go templates in YAML) and chart packaging complexity. Educational project benefits from seeing raw YAML with simple overlays. Kustomize is built into kubectl. |
| Local K8s dev | Tilt | Skaffold | Tilt has superior live-update UX (sync binary without rebuild), built-in web dashboard. Skaffold is more CI-focused and less interactive. |
| Local cluster | Docker Desktop K8s | kind/minikube/k3d | Docker Desktop is already installed for Docker. No extra tool needed. kind is a good fallback but adds another binary. |
| GitOps | ArgoCD | Flux | ArgoCD has a much better web UI for learning and demos. Wider community adoption. Flux is more "headless" and operator-centric. |
| Linter | golangci-lint v2 | go vet only | golangci-lint runs 100+ linters (staticcheck, errcheck, gosec, etc.) in one pass. go vet catches only a subset. The GitHub Action makes CI integration trivial. |
| Registry | GHCR | Docker Hub | GHCR has native GITHUB_TOKEN auth, no separate account, no pull rate limits for public images. Docker Hub rate-limits anonymous pulls (100/6h). |
| Go container build | Dockerfile | ko | ko is elegant for pure-Go apps but our project has a React SPA frontend that needs nginx. Standard multi-stage Dockerfile handles both services. |

## Docker Multi-Stage Build Patterns

### Go Backend Dockerfile

```dockerfile
# ---- Build Stage ----
FROM golang:1.26-alpine AS builder

WORKDIR /app

# Cache dependencies separately from source
COPY go.mod go.sum ./
RUN go mod download

# Build static binary
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /shitcoin ./cmd/shitcoin/

# ---- Runtime Stage ----
FROM alpine:3.23

RUN apk add --no-cache ca-certificates tzdata

COPY --from=builder /shitcoin /usr/local/bin/shitcoin
COPY etc/shitcoin.yaml /etc/shitcoin/shitcoin.yaml

EXPOSE 9000 8080

ENTRYPOINT ["shitcoin"]
CMD ["-f", "/etc/shitcoin/shitcoin.yaml", "startnode"]
```

Key decisions:
- **CGO_ENABLED=0**: bbolt (BoltDB) is pure Go, no C dependencies. Static binary runs on any Linux.
- **`-ldflags="-s -w"`**: Strip debug symbols and DWARF info. Reduces binary size ~30%.
- **alpine:3.23 runtime** (not scratch): ca-certificates needed for any HTTPS calls, tzdata for time formatting. Adds ~7MB but avoids subtle runtime issues.
- **Separate COPY for go.mod/go.sum**: Docker layer caching -- dependency layer is rebuilt only when deps change, not on every code change.

### React Frontend Dockerfile

```dockerfile
# ---- Build Stage ----
FROM node:22-alpine AS builder

WORKDIR /app

# Cache dependencies separately
COPY package.json package-lock.json ./
RUN npm ci

# Build production bundle
COPY . .
RUN npm run build

# ---- Runtime Stage ----
FROM nginx:alpine

COPY --from=builder /app/dist /usr/share/nginx/html
COPY nginx.conf /etc/nginx/conf.d/default.conf

EXPOSE 80
```

Key decisions:
- **npm ci** over npm install: Deterministic installs, faster in CI, respects lockfile exactly, fails on lockfile mismatch.
- **nginx:alpine runtime**: Serves static files with excellent performance. Handles SPA routing via try_files.
- **Separate package.json COPY**: Layer caching for npm dependencies (changes infrequently).

### nginx.conf for SPA + API Proxy

```nginx
server {
    listen 80;
    root /usr/share/nginx/html;
    index index.html;

    # Proxy API requests to Go backend service
    location /api/ {
        proxy_pass http://shitcoin-backend:8080;
    }

    # Proxy WebSocket to Go backend service
    location /ws {
        proxy_pass http://shitcoin-backend:8080;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";
    }

    # SPA fallback -- all other routes serve index.html
    location / {
        try_files $uri $uri/ /index.html;
    }
}
```

## Kustomize Directory Structure

```
k8s/
  base/
    kustomization.yaml          # Resources list, common labels
    namespace.yaml              # shitcoin namespace
    backend-deployment.yaml     # Go node deployment
    backend-service.yaml        # ClusterIP service on 8080
    frontend-deployment.yaml    # nginx SPA deployment
    frontend-service.yaml       # ClusterIP service on 80
    configmap.yaml              # shitcoin.yaml config
  overlays/
    dev/
      kustomization.yaml        # namePrefix: dev-, replicas: 1, dev image tags
    production/
      kustomization.yaml        # replicas: 3, resource limits, prod image tags
```

## GitHub Actions Workflow Structure

```
.github/
  workflows/
    ci.yaml                     # On PR: test, lint, type-check frontend
    build-push.yaml             # On push to main: build images, push to GHCR
```

### CI Workflow Key Steps

1. **Go tests**: `go test ./...` with race detector
2. **Go lint**: golangci-lint via official action
3. **Frontend lint**: `npm run lint` in web/
4. **Frontend type-check**: `tsc -b` in web/ (already part of `npm run build`)
5. **Docker build** (no push): Verify Dockerfiles build successfully on PRs

### Build+Push Workflow Key Steps

1. **Checkout + setup**: Go, Node, Docker Buildx
2. **Test + lint**: Same as CI
3. **Build + push backend**: Multi-stage build, push to ghcr.io
4. **Build + push frontend**: Multi-stage build, push to ghcr.io
5. **Tag strategy**: `sha-<commit>` for every push, `v*` semver tags on releases

## golangci-lint Configuration

```yaml
# .golangci.yml
version: "2"
linters:
  enable:
    - errcheck
    - govet
    - staticcheck
    - unused
    - gosimple
    - ineffassign
    - typecheck
run:
  go: "1.26"
```

Keep the linter set focused. The default set plus errcheck and staticcheck catch the most real bugs. Avoid enabling too many style linters that create noise.

## Installation (Local Development)

```bash
# Tilt for local K8s dev
brew install tilt-dev/tap/tilt
brew install tilt-dev/tap/ctlptl

# ArgoCD CLI (optional, for managing ArgoCD from terminal)
brew install argocd

# Kustomize standalone (optional, kubectl has it built-in)
brew install kustomize

# golangci-lint (for local linting, CI uses the action)
brew install golangci-lint
```

## What NOT to Add

| Tool | Why Not |
|------|---------|
| Helm | Templating overhead for a 2-service educational project. Kustomize overlays are simpler and show raw YAML. |
| ko | Only works for pure-Go apps. Our project needs nginx for the React SPA. Standard Dockerfile is the right choice. |
| Skaffold | Tilt provides better live-update DX and a web dashboard. Don't install both. |
| Flux | ArgoCD has the better learning UI. Pick one GitOps tool. |
| Terraform/Pulumi | Infrastructure provisioning is out of scope. This milestone is app deployment, not cloud infra. |
| Istio/Linkerd | Service mesh is overkill for 2 services in an educational project. |
| cert-manager | No TLS needed for local/educational deployment. |
| External Secrets Operator | No real secrets to manage. ConfigMaps suffice for config. |
| Docker Compose for K8s | Tilt replaces docker-compose for K8s dev workflow. Don't use both. |
| Kaniko | Only needed for building images inside K8s (no Docker daemon). GitHub Actions has Docker natively. |

## Sources

- [GitHub Actions Runner Images](https://github.com/actions/runner-images) -- ubuntu-latest = Ubuntu 24.04
- [actions/setup-go v6](https://github.com/actions/setup-go) -- latest major version with Go module caching
- [golangci-lint v2.11.1 release](https://github.com/golangci/golangci-lint/releases) -- released 2026-03-06
- [golangci-lint-action v9](https://github.com/golangci/golangci-lint-action) -- latest action version
- [docker/build-push-action v6](https://github.com/docker/build-push-action) -- latest stable
- [golang:1.26-alpine on Docker Hub](https://hub.docker.com/layers/library/golang/1.26-alpine/) -- verified available
- [node:22-alpine on Docker Hub](https://hub.docker.com/layers/library/node/22-alpine/) -- Node 22 LTS
- [Tilt v0.37.0 release](https://github.com/tilt-dev/tilt/releases) -- released 2026-03-04
- [ArgoCD v3.3.2 release](https://github.com/argoproj/argo-cd/releases) -- released 2026-02-22
- [Kustomize v5.8.0 release](https://github.com/kubernetes-sigs/kustomize/releases) -- latest stable
- [Docker multi-stage builds guide](https://docs.docker.com/build/building/multi-stage/)
- [ArgoCD upgrade guide v2.14 to 3.0](https://argo-cd.readthedocs.io/en/stable/operator-manual/upgrading/2.14-3.0/)
