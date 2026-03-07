# Tiltfile -- Local Kubernetes development for shitcoin
#
# Prerequisites:
#   - kind cluster running: kind create cluster --config deploy/k8s/kind-cluster.yaml
#   - Node.js installed (for frontend builds)
#
# Usage:
#   tilt up
#
# Port forwards:
#   - Backend API:  http://localhost:8080
#   - Frontend:     http://localhost:3000

# =============================================================================
# Extensions
# =============================================================================

# restart_process allows syncing a new binary into a running container and
# restarting the process without rebuilding the entire Docker image.
load('ext://restart_process', 'docker_build_with_restart')

# =============================================================================
# Backend (Go) -- compile locally, sync binary
# =============================================================================

# Compile the Go binary on the host. CGO_ENABLED=0 produces a static binary
# compatible with alpine. GOOS=linux targets Linux containers. GOARCH is omitted
# so it defaults to the host architecture, avoiding QEMU overhead on Apple Silicon.
local_resource(
    'backend-compile',
    'CGO_ENABLED=0 GOOS=linux go build -o build/shitcoin ./cmd/shitcoin',
    deps=['cmd/', 'internal/', 'go.mod', 'go.sum'],
)

# Build the dev image using Dockerfile.dev (lightweight alpine + pre-compiled binary).
# live_update syncs the new binary into the running container and restarts the process.
docker_build_with_restart(
    'shitcoin-backend',
    '.',
    entrypoint=['/app/shitcoin', '-f', '/app/etc/shitcoin.yaml', 'startnode'],
    dockerfile='Dockerfile.dev',
    only=['./build', './etc'],
    live_update=[
        sync('./build/shitcoin', '/app/shitcoin'),
    ],
)

# =============================================================================
# Frontend (React) -- build locally, sync dist assets
# =============================================================================

# Build the frontend assets on the host using Vite.
local_resource(
    'frontend-build',
    'cd web && npm run build',
    deps=['web/src/', 'web/index.html', 'web/vite.config.ts', 'web/tailwind.config.ts'],
)

# Build the frontend image. live_update syncs the built assets into the nginx
# container. A change to package.json or package-lock.json triggers a full
# rebuild (fall_back_on) since dependencies may have changed.
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

# =============================================================================
# Kubernetes Resources
# =============================================================================

# Load manifests from the existing Kustomize dev overlay (Phase 11).
k8s_yaml(kustomize('deploy/k8s/overlays/dev'))

# Configure port forwards and resource dependencies.
k8s_resource(
    'backend',
    port_forwards=[port_forward(8080, 8080, name='Backend API')],
    resource_deps=['backend-compile'],
)

k8s_resource(
    'frontend',
    port_forwards=[port_forward(3000, 8080, name='Frontend')],
    resource_deps=['frontend-build'],
)
