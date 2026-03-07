# Phase 12: Local K8s Development - Research

**Researched:** 2026-03-07
**Domain:** Tilt + kind local Kubernetes development workflow
**Confidence:** HIGH

## Summary

Phase 12 sets up a local Kubernetes development environment using Tilt and kind. The project already has Dockerfiles (Phase 9) and Kustomize manifests with dev/prod overlays (Phase 11). This phase wires them together with a Tiltfile for live-reload development, a kind cluster config for local K8s, and a Makefile for common commands.

The recommended Go pattern for Tilt uses `local_resource` to compile the Go binary on the host (fast, uses local cache), then `docker_build_with_restart` to sync the compiled binary into the container and restart the process. For the React frontend, the production nginx-based image is used with `live_update` syncing rebuilt assets. Tilt's built-in `kustomize()` function integrates directly with the existing dev overlay.

**Primary recommendation:** Use the "compile locally, sync binary" pattern with `restart_process` extension for Go backend; use `docker_build` with `live_update` and `fall_back_on` for frontend; integrate with existing Kustomize dev overlay via `k8s_yaml(kustomize('deploy/k8s/overlays/dev'))`.

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|-----------------|
| DEV-01 | Tiltfile with docker_build and live_update for Go backend hot reload | Tilt `restart_process` extension + `local_resource` compile pattern; sync compiled binary into container |
| DEV-02 | Tiltfile with docker_build and live_update for React frontend | `docker_build` with `live_update` syncing `web/dist` into nginx html dir; `fall_back_on` for package.json changes |
| DEV-03 | kind cluster config and setup instructions provided | kind config YAML with `extraPortMappings` for NodePort access; single control-plane node sufficient |
| DEV-04 | Makefile with common commands (ci, docker-build, tilt-up, lint, test) | Standard Makefile with phony targets wrapping existing commands |
</phase_requirements>

## Standard Stack

### Core
| Tool | Version | Purpose | Why Standard |
|------|---------|---------|--------------|
| Tilt | latest (v0.33+) | Local K8s dev orchestration with live reload | Project decision (REQUIREMENTS.md), best live_update UX |
| kind | v0.31.0 | Local K8s cluster in Docker | Lightweight, fast, standard for local dev |
| kustomize | built-in (kubectl) | K8s manifest templating | Already used in Phase 11, Tilt has native integration |

### Supporting
| Tool | Version | Purpose | When to Use |
|------|---------|---------|-------------|
| restart_process ext | Tilt extension | Restart Go binary after live_update sync | Required for Go binary hot-reload pattern |

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| kind | minikube | minikube heavier, more features; kind simpler for this use case |
| Tilt | Skaffold | Project explicitly chose Tilt (REQUIREMENTS.md Out of Scope) |

**Installation:**
```bash
# kind
go install sigs.k8s.io/kind@v0.31.0
# or: brew install kind

# Tilt
curl -fsSL https://raw.githubusercontent.com/tilt-dev/tilt/master/scripts/install.sh | bash
# or: brew install tilt
```

## Architecture Patterns

### Recommended Project Structure
```
.
├── Tiltfile                          # Tilt orchestration (project root)
├── Makefile                          # Common dev commands
├── deploy/
│   └── k8s/
│       ├── base/                     # Existing Kustomize base (Phase 11)
│       ├── overlays/
│       │   ├── dev/                  # Existing dev overlay (Phase 11)
│       │   └── prod/                 # Existing prod overlay (Phase 11)
│       └── kind-cluster.yaml         # kind cluster config
```

### Pattern 1: Go Backend - Compile Locally, Sync Binary
**What:** Compile Go binary on the host machine, sync the compiled binary into the running container, restart the process.
**When to use:** Always for compiled languages like Go -- avoids slow in-container compilation.
**Why:** Host compilation uses local Go cache and is fast (sub-second for incremental). Syncing a single binary is faster than rebuilding an entire Docker image.

```python
# Tiltfile
load('ext://restart_process', 'docker_build_with_restart')

# Step 1: Compile Go binary locally for Linux
local_resource(
    'backend-compile',
    'CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o build/shitcoin cmd/shitcoin/main.go',
    deps=['cmd/', 'internal/', 'go.mod', 'go.sum'],
)

# Step 2: Build image with live_update that syncs the compiled binary
docker_build_with_restart(
    'shitcoin-backend',
    '.',
    entrypoint=['/app/shitcoin', '-f', '/app/etc/shitcoin.yaml', 'startnode'],
    dockerfile='Dockerfile',
    only=['./build', './etc'],
    live_update=[
        sync('./build/shitcoin', '/app/shitcoin'),
    ],
)
```

**Key details:**
- `only=['./build', './etc']` limits Docker context to compiled binary + config
- `docker_build_with_restart` handles process restart automatically after sync
- `local_resource` deps watch Go source files and trigger recompilation
- `CGO_ENABLED=0 GOOS=linux GOARCH=amd64` matches the Dockerfile build flags

### Pattern 2: React Frontend - Rebuild and Sync Assets
**What:** Build the React app locally, sync the built dist into the nginx container.
**When to use:** For SPA frontends served by nginx in production-like setup.

```python
# Tiltfile

# Step 1: Build frontend locally
local_resource(
    'frontend-build',
    'cd web && npm run build',
    deps=['web/src/', 'web/index.html', 'web/vite.config.ts'],
)

# Step 2: Build image with live_update syncing dist
docker_build(
    'shitcoin-frontend',
    'web',
    dockerfile='web/Dockerfile',
    only=['./dist', './nginx.conf'],
    live_update=[
        fall_back_on(['web/package.json', 'web/package-lock.json']),
        sync('./web/dist', '/usr/share/nginx/html'),
    ],
)
```

**Alternative (simpler):** For this educational project, the frontend could also just use `docker_build` without live_update and do a full image rebuild on changes -- the frontend image builds fast (~10s) and simplicity may win here.

### Pattern 3: Kustomize Integration
**What:** Use Tilt's built-in `kustomize()` function to load the existing dev overlay.

```python
# Tiltfile
k8s_yaml(kustomize('deploy/k8s/overlays/dev'))
```

Tilt watches the kustomization directory and re-applies when config files change.

### Pattern 4: Port Forwarding
**What:** Tilt automatically port-forwards services for local access.

```python
# Tiltfile
k8s_resource('backend', port_forwards=[
    port_forward(8080, 8080, name='Backend API'),
])
k8s_resource('frontend', port_forwards=[
    port_forward(3000, 8080, name='Frontend'),
])
```

### Anti-Patterns to Avoid
- **Compiling Go inside the container:** Slow, no build cache, requires Go toolchain in image. Always compile on host.
- **Using `restart_container()` instead of `restart_process` extension:** `restart_container()` kills the entire container (slow); `restart_process` just restarts the binary (fast).
- **Running Vite dev server inside a K8s container:** Complex, requires extra ports, WebSocket forwarding. Use the production nginx pattern with synced dist files instead.
- **Not setting `only` in `docker_build`:** Without `only`, every file change triggers a full Docker build context copy.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Process restart after sync | Custom entrypoint scripts | `restart_process` Tilt extension | Handles signal forwarding, PID management correctly |
| Kustomize rendering | Manual `kubectl kustomize` | `kustomize()` Tilt function | Built-in, watches files, auto-reapplies |
| Port forwarding | NodePort + kind extraPortMappings | `k8s_resource` port_forwards | Tilt manages lifecycle, shows in UI |
| Local K8s cluster | Manual Docker networking | kind with config YAML | Reproducible, version-controlled |

**Key insight:** Tilt handles the entire inner dev loop (watch -> build -> deploy -> port-forward). Don't replicate any of this in scripts or Makefile targets.

## Common Pitfalls

### Pitfall 1: GOARCH Mismatch on Apple Silicon
**What goes wrong:** Compiling with `GOARCH=amd64` on an M-series Mac produces a binary that runs slowly under QEMU emulation in the kind container (kind uses the host Docker's architecture).
**Why it happens:** kind nodes run the same architecture as the host Docker daemon. On Apple Silicon, kind runs arm64 containers.
**How to avoid:** Use `GOARCH=$(go env GOARCH)` or detect architecture dynamically. On Apple Silicon Macs, kind runs arm64 nodes, so compile for arm64.
**Warning signs:** Very slow container startup, or binary crashes with "exec format error".

```python
# Correct approach - detect host architecture
compile_cmd = 'CGO_ENABLED=0 GOOS=linux go build -o build/shitcoin cmd/shitcoin/main.go'
```
Omitting GOARCH entirely defaults to host architecture, which matches kind's architecture.

### Pitfall 2: BoltDB File Locking in PVC
**What goes wrong:** BoltDB holds a file lock. If the pod restarts and the old process doesn't release the lock, the new process fails to start.
**Why it happens:** Recreate strategy kills old pod first, but PVC might still be mounted.
**How to avoid:** The existing Recreate strategy (Phase 11) already handles this. Tilt's `restart_process` only restarts the binary, not the container, so the lock is released cleanly.
**Warning signs:** "database is locked" errors on startup.

### Pitfall 3: Docker Build Context Too Large
**What goes wrong:** Tilt copies the entire project directory as Docker context on every change.
**Why it happens:** Not using `only` parameter or having a weak `.dockerignore`.
**How to avoid:** Always set `only` in `docker_build` to include only what the Dockerfile needs. The existing `.dockerignore` helps but `only` is more precise.

### Pitfall 4: kind Cluster Already Exists
**What goes wrong:** `kind create cluster` fails if a cluster with the same name already exists.
**Why it happens:** Developer runs setup twice.
**How to avoid:** Use `kind create cluster --name shitcoin || true` or check first with `kind get clusters`. Makefile target should be idempotent.

### Pitfall 5: Tilt Extension Not Found
**What goes wrong:** `load('ext://restart_process', ...)` fails on first run.
**Why it happens:** Tilt extensions are fetched on demand. Network issues or version incompatibility.
**How to avoid:** This is rare -- Tilt handles extension download automatically. Document that internet access is required on first `tilt up`.

## Code Examples

### Complete Tiltfile
```python
# Load restart_process extension for Go binary hot-reload
load('ext://restart_process', 'docker_build_with_restart')

# --- Backend (Go) ---

# Compile Go binary locally (uses host Go cache for speed)
local_resource(
    'backend-compile',
    'CGO_ENABLED=0 GOOS=linux go build -o build/shitcoin cmd/shitcoin/main.go',
    deps=['cmd/', 'internal/', 'go.mod', 'go.sum'],
)

# Build backend image with live_update
docker_build_with_restart(
    'shitcoin-backend',
    '.',
    entrypoint=['/app/shitcoin', '-f', '/app/etc/shitcoin.yaml', 'startnode'],
    dockerfile='Dockerfile',
    only=['./build', './etc'],
    live_update=[
        sync('./build/shitcoin', '/app/shitcoin'),
    ],
)

# --- Frontend (React) ---

# Build frontend locally
local_resource(
    'frontend-build',
    'cd web && npm run build',
    deps=['web/src/', 'web/index.html', 'web/vite.config.ts', 'web/tailwind.config.ts'],
)

# Build frontend image with live_update
docker_build(
    'shitcoin-frontend',
    'web',
    dockerfile='web/Dockerfile',
    only=['./dist', './nginx.conf'],
    live_update=[
        fall_back_on(['package.json', 'package-lock.json']),
        sync('./dist', '/usr/share/nginx/html'),
    ],
)

# --- Kubernetes ---

# Load Kustomize dev overlay (existing from Phase 11)
k8s_yaml(kustomize('deploy/k8s/overlays/dev'))

# Configure resources with port forwarding
k8s_resource('backend', port_forwards=[
    port_forward(8080, 8080, name='Backend API'),
], resource_deps=['backend-compile'])

k8s_resource('frontend', port_forwards=[
    port_forward(3000, 8080, name='Frontend'),
], resource_deps=['frontend-build'])
```

### kind Cluster Config
```yaml
# deploy/k8s/kind-cluster.yaml
kind: Cluster
apiVersion: kind.x-k8s.io/v1alpha4
name: shitcoin
nodes:
  - role: control-plane
```
Note: Tilt handles port forwarding, so kind `extraPortMappings` are not needed.

### Makefile
```makefile
.PHONY: test lint ci docker-build tilt-up kind-create kind-delete

# Run all Go tests
test:
	go test ./...

# Run linter
lint:
	golangci-lint run

# Run full CI checks locally (tests + lint + frontend)
ci: test lint
	cd web && npm run lint && npm run build

# Build Docker images locally
docker-build:
	docker build -t shitcoin-backend .
	docker build -t shitcoin-frontend web/

# Create kind cluster
kind-create:
	kind create cluster --config deploy/k8s/kind-cluster.yaml || true

# Delete kind cluster
kind-delete:
	kind delete cluster --name shitcoin

# Start Tilt (creates kind cluster if needed)
tilt-up: kind-create
	tilt up
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| Skaffold for local K8s dev | Tilt with live_update | 2020+ | Better UX, extension ecosystem |
| `restart_container()` | `restart_process` extension | Tilt 2021+ | Faster restarts, no container kill |
| In-container Go compilation | Local compilation + binary sync | Standard pattern | Sub-second rebuilds vs 30s+ |
| minikube | kind | 2019+ | Faster startup, lighter footprint |

## Open Questions

1. **Frontend live_update approach**
   - What we know: Two options -- (a) rebuild locally + sync dist, or (b) just do full image rebuild (fast enough at ~10s)
   - What's unclear: Whether the `only` + `live_update` pattern works cleanly with the web/Dockerfile multi-stage build
   - Recommendation: Try the live_update approach first; fall back to simple `docker_build` without live_update if complexity outweighs benefit. The Tiltfile should use `docker_build` for frontend with a simpler approach since frontend rebuilds are fast.

2. **Dockerfile compatibility with Tilt `only` parameter**
   - What we know: The existing Dockerfile copies `cmd/` and `internal/` from context. With `only=['./build', './etc']`, those COPY steps will fail.
   - What's unclear: N/A -- this is a known issue.
   - Recommendation: Create a lightweight `Dockerfile.dev` for the backend that copies just the pre-compiled binary and config, skipping the Go build stage entirely. Or use `dockerfile_contents` in the Tiltfile to inline a simple Dockerfile.

## Validation Architecture

### Test Framework
| Property | Value |
|----------|-------|
| Framework | Go testing + manual verification |
| Config file | N/A (infrastructure files, not code) |
| Quick run command | `go test ./...` |
| Full suite command | `go test ./... && cd web && npm run lint && npm run build` |

### Phase Requirements to Test Map
| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| DEV-01 | `tilt up` builds and deploys backend with live_update | manual | `tilt ci` (headless mode) | N/A - new file |
| DEV-02 | Frontend live_update works | manual | `tilt ci` | N/A - new file |
| DEV-03 | kind cluster config works | manual | `kind create cluster --config deploy/k8s/kind-cluster.yaml` | N/A - new file |
| DEV-04 | Makefile targets work | smoke | `make test && make lint && make docker-build` | N/A - new file |

### Sampling Rate
- **Per task commit:** `make test && make lint`
- **Per wave merge:** `make ci && make docker-build`
- **Phase gate:** `kind create cluster --config deploy/k8s/kind-cluster.yaml && tilt ci` (verifies end-to-end)

### Wave 0 Gaps
- [ ] `build/` directory in `.gitignore` -- compiled binary output directory
- [ ] Verify `tilt ci` works in headless mode for CI validation

## Sources

### Primary (HIGH confidence)
- [Tilt Live Update Reference](https://docs.tilt.dev/live_update_reference.html) - sync, run, fall_back_on, restart_container APIs
- [Tilt API Reference](https://docs.tilt.dev/api.html) - docker_build, k8s_yaml, k8s_resource, kustomize, port_forward signatures
- [Tilt Go Example](https://docs.tilt.dev/example_go.html) - recommended Go pattern with local_resource + docker_build_with_restart
- [tilt-example-go 3-recommended Tiltfile](https://github.com/tilt-dev/tilt-example-go/blob/master/3-recommended/Tiltfile) - reference implementation
- [kind Configuration](https://kind.sigs.k8s.io/docs/user/configuration/) - cluster YAML format, extraPortMappings

### Secondary (MEDIUM confidence)
- [Tilt Kustomize integration](https://docs.tilt.dev/templating.html) - kustomize() function usage
- [kind releases](https://github.com/kubernetes-sigs/kind) - v0.31.0 latest

### Tertiary (LOW confidence)
- None

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH - Tilt and kind are well-documented, project explicitly chose these tools
- Architecture: HIGH - Go compile-locally pattern is the official recommended approach from Tilt
- Pitfalls: HIGH - GOARCH mismatch and BoltDB locking are well-known issues documented in project state

**Research date:** 2026-03-07
**Valid until:** 2026-04-07 (stable tools, slow-moving ecosystem)
