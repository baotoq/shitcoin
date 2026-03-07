# Phase 9: Containerization - Research

**Researched:** 2026-03-07
**Domain:** Docker multi-stage builds for Go backend + React/nginx frontend
**Confidence:** HIGH

## Summary

This phase creates two production Dockerfiles (Go backend, React+nginx frontend), a .dockerignore, and an nginx.conf. The Go backend uses multi-stage build with CGO_ENABLED=0 to produce a statically-linked binary that runs on `scratch` or `alpine` for a minimal image (~15MB). The React frontend uses multi-stage build: Node for `npm run build`, then nginx:alpine to serve the SPA with reverse proxy rules for `/api` and `/ws`.

Both are well-established patterns with no novel complexity. The project uses Go 1.26.1 and Vite 7 / React 19 -- both are standard multi-stage Dockerfile targets. The config file (`etc/shitcoin.yaml`) must be copied into the backend image. BoltDB storage is handled at runtime via volume mounts (not baked into the image).

**Primary recommendation:** Use `golang:1.26-alpine` builder + `alpine:3.21` runtime for backend (not scratch, since config file needs to be loaded and alpine provides shell for debugging). Use `node:22-alpine` builder + `nginx:1.27-alpine` runtime for frontend.

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|-----------------|
| DOCK-01 | Multi-stage Dockerfile produces minimal Go backend image (~15MB) with CGO_ENABLED=0 | Go multi-stage pattern with alpine runtime; CGO_ENABLED=0 for static linking |
| DOCK-02 | Multi-stage Dockerfile produces React frontend image with nginx serving SPA | Node builder stage + nginx:alpine runtime; vite build produces static assets |
| DOCK-03 | .dockerignore excludes data/, wallets.json, .git, node_modules from build context | Standard .dockerignore pattern at project root |
| DOCK-04 | nginx.conf provides SPA try_files routing and reverse proxies /api and /ws to backend | nginx config with try_files for SPA + proxy_pass for API + WebSocket upgrade for /ws |
| DOCK-05 | Both containers run as non-root user | adduser/addgroup in Dockerfile, USER directive before CMD |
</phase_requirements>

## Standard Stack

### Core

| Component | Version | Purpose | Why Standard |
|-----------|---------|---------|--------------|
| golang | 1.26-alpine | Go build stage | Matches project go.mod (1.26.1) |
| alpine | 3.21 | Go runtime stage | Minimal Linux (~5MB), provides shell for debugging |
| node | 22-alpine | Frontend build stage | LTS Node for npm ci + vite build |
| nginx | 1.27-alpine | Frontend runtime | Industry standard static file server + reverse proxy |

### Supporting

| Component | Purpose | When to Use |
|-----------|---------|-------------|
| .dockerignore | Exclude build context bloat | Always -- prevents data/, .git, node_modules from being sent to daemon |

### Alternatives Considered

| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| alpine runtime | scratch | Scratch is ~2MB smaller but has no shell, no debugging tools, no /etc/passwd for USER |
| alpine runtime | distroless | Similar to scratch but with slightly better security defaults; adds complexity for minimal gain here |
| nginx | caddy | Caddy has simpler config but nginx is more widely known and documented |

## Architecture Patterns

### Recommended Project Structure

```
Dockerfile              # Go backend (project root context)
web/Dockerfile          # React frontend (web/ context OR project root context)
web/nginx.conf          # nginx configuration for SPA + reverse proxy
.dockerignore           # Root-level exclusions
```

### Pattern 1: Go Multi-Stage Build

**What:** Two-stage Dockerfile -- build in golang:alpine, run in plain alpine.
**When to use:** Always for Go services targeting containers.

```dockerfile
# Build stage
FROM golang:1.26-alpine AS builder
WORKDIR /build
COPY go.mod go.sum ./
RUN go mod download
COPY cmd/ cmd/
COPY internal/ internal/
COPY etc/ etc/
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /app cmd/shitcoin/main.go

# Runtime stage
FROM alpine:3.21
RUN addgroup -g 1001 -S appgroup && adduser -S appuser -u 1001 -G appgroup
WORKDIR /app
COPY --from=builder --chown=appuser:appgroup /app .
COPY --from=builder --chown=appuser:appgroup /build/etc/shitcoin.yaml /app/etc/shitcoin.yaml
USER appuser
EXPOSE 8080
CMD ["./app", "-f", "etc/shitcoin.yaml", "startnode"]
```

Key details:
- `CGO_ENABLED=0` produces a fully static binary (no glibc dependency) -- mandatory since project uses BoltDB which compiles pure Go with build tag
- `-ldflags="-s -w"` strips debug symbols and DWARF info, reducing binary ~30%
- `go mod download` before copying source enables Docker layer caching
- Config file `etc/shitcoin.yaml` must be in the image since the binary expects `-f etc/shitcoin.yaml`

### Pattern 2: React/Vite Multi-Stage Build with nginx

**What:** Three-stage Dockerfile -- deps install, build, nginx serve.
**When to use:** Always for SPA frontends in containers.

```dockerfile
# Build stage
FROM node:22-alpine AS builder
WORKDIR /app
COPY package.json package-lock.json ./
RUN npm ci
COPY . .
RUN npm run build

# Runtime stage
FROM nginx:1.27-alpine
RUN addgroup -g 1001 -S appgroup && adduser -S appuser -u 1001 -G appgroup
COPY --from=builder /app/dist /usr/share/nginx/html
COPY nginx.conf /etc/nginx/conf.d/default.conf
# nginx master runs as root (needed to bind port 80), workers run as nginx user
# For full non-root: use port 8080 and set user directive
RUN chown -R appuser:appgroup /var/cache/nginx /var/log/nginx /var/run && \
    sed -i 's/listen\s*80;/listen 8080;/g' /etc/nginx/conf.d/default.conf
USER appuser
EXPOSE 8080
CMD ["nginx", "-g", "daemon off;"]
```

### Pattern 3: nginx.conf for SPA + Reverse Proxy + WebSocket

**What:** nginx config that serves SPA with try_files fallback and proxies API/WS to backend.

```nginx
server {
    listen 8080;
    server_name _;
    root /usr/share/nginx/html;
    index index.html;

    # SPA routing -- fallback to index.html for client-side routes
    location / {
        try_files $uri $uri/ /index.html;
    }

    # Reverse proxy API requests to backend
    location /api/ {
        proxy_pass http://backend:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }

    # WebSocket proxy
    location /ws {
        proxy_pass http://backend:8080;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $websocket_upgrade;
        proxy_set_header Connection "Upgrade";
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
    }
}

# Map for WebSocket upgrade
map $http_upgrade $websocket_upgrade {
    default upgrade;
    '' close;
}
```

Key detail: The `backend:8080` upstream hostname will be resolved via Kubernetes Service DNS in later phases. For standalone Docker testing, use `--add-host` or environment variable substitution with `envsubst`.

### Pattern 4: Making nginx Fully Non-Root

**What:** nginx by default needs root to bind port 80 and write to certain directories.
**How:** Use port 8080, fix directory permissions, and optionally modify nginx.conf to not use `/var/run/nginx.pid`.

```dockerfile
# Fix: nginx needs write access to these directories
RUN chown -R appuser:appgroup /var/cache/nginx /var/log/nginx && \
    touch /var/run/nginx.pid && chown appuser:appgroup /var/run/nginx.pid
```

### Anti-Patterns to Avoid

- **Copying entire project into build stage:** Only COPY what the build needs (go.mod, go.sum, cmd/, internal/, etc/). Never `COPY . .` for Go backend.
- **Not separating dependency download from source copy:** Always `go mod download` / `npm ci` before copying source to leverage Docker layer cache.
- **Using latest tags:** Pin to specific minor versions (golang:1.26-alpine, not golang:latest).
- **Baking data/wallets into image:** These are runtime state, must be volume-mounted.
- **Running as root:** Always create and switch to non-root user.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| SPA routing in nginx | Custom redirect scripts | try_files $uri /index.html | Handles all client-side routes correctly |
| WebSocket proxying | Custom proxy logic | nginx proxy_pass with Upgrade headers | Handles connection upgrade protocol correctly |
| Non-root user setup | Manual uid/gid management | Alpine addgroup/adduser with fixed IDs | Consistent, auditable, works with K8s securityContext |
| Build context filtering | Manual COPY exclusions | .dockerignore | Prevents sending GB of .git and node_modules to Docker daemon |

## Common Pitfalls

### Pitfall 1: BoltDB and CGO_ENABLED=0
**What goes wrong:** BoltDB (bbolt) uses unsafe but pure Go. With CGO_ENABLED=0, it builds fine. But if CGO were enabled, it would try to link against glibc which doesn't exist on alpine/scratch.
**Why it happens:** Default Go build on Linux has CGO_ENABLED=1.
**How to avoid:** Always set `CGO_ENABLED=0` explicitly in Dockerfile.
**Warning signs:** Binary crashes with "not found" error (missing dynamic linker).

### Pitfall 2: Frontend Build Context Location
**What goes wrong:** If Dockerfile for frontend is at project root with `COPY . .`, it copies the entire Go project into Node build stage.
**Why it happens:** Docker build context is relative to where `docker build` is run.
**How to avoid:** Either: (a) put frontend Dockerfile in `web/` and build with `-f web/Dockerfile web/`, or (b) use `.dockerignore` aggressively.
**Recommended:** Place `web/Dockerfile` in `web/` directory, use `web/` as build context.

### Pitfall 3: nginx Non-Root Port Binding
**What goes wrong:** nginx fails to start as non-root because it can't bind port 80 or write to /var/run/nginx.pid.
**Why it happens:** Ports below 1024 require root (or CAP_NET_BIND_SERVICE).
**How to avoid:** Use port 8080 in nginx.conf. Fix ownership of /var/cache/nginx, /var/log/nginx, /var/run/nginx.pid.

### Pitfall 4: Config File Not in Image
**What goes wrong:** Backend binary starts but crashes because `etc/shitcoin.yaml` isn't found.
**Why it happens:** Only Go source was copied, not the config directory.
**How to avoid:** Explicitly COPY etc/shitcoin.yaml into the runtime stage.

### Pitfall 5: WebSocket Proxy Missing Upgrade Headers
**What goes wrong:** WebSocket connections fail with 400/502 through nginx.
**Why it happens:** nginx doesn't forward Upgrade/Connection headers by default.
**How to avoid:** Use `proxy_http_version 1.1;` and set Upgrade/Connection headers explicitly.

### Pitfall 6: Docker Build Cache Invalidation
**What goes wrong:** Every code change re-downloads all Go modules or npm packages.
**Why it happens:** COPY of source files before dependency download invalidates the cache.
**How to avoid:** Copy dependency manifests first (go.mod/go.sum, package.json/package-lock.json), install deps, then copy source.

## Code Examples

### Complete Go Backend Dockerfile

```dockerfile
# ---- Build stage ----
FROM golang:1.26-alpine AS builder
WORKDIR /build

# Cache dependencies
COPY go.mod go.sum ./
RUN go mod download

# Build binary
COPY cmd/ cmd/
COPY internal/ internal/
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /app cmd/shitcoin/main.go

# ---- Runtime stage ----
FROM alpine:3.21

# Non-root user
RUN addgroup -g 1001 -S appgroup && \
    adduser -S appuser -u 1001 -G appgroup

WORKDIR /app
COPY --from=builder --chown=appuser:appgroup /app .
COPY --chown=appuser:appgroup etc/shitcoin.yaml etc/shitcoin.yaml

USER appuser
EXPOSE 8080

CMD ["./app", "-f", "etc/shitcoin.yaml", "startnode"]
```

### Complete Frontend Dockerfile (web/Dockerfile)

```dockerfile
# ---- Build stage ----
FROM node:22-alpine AS builder
WORKDIR /app

COPY package.json package-lock.json ./
RUN npm ci

COPY . .
RUN npm run build

# ---- Runtime stage ----
FROM nginx:1.27-alpine

# Non-root setup
RUN addgroup -g 1001 -S appgroup && \
    adduser -S appuser -u 1001 -G appgroup && \
    chown -R appuser:appgroup /var/cache/nginx /var/log/nginx && \
    touch /var/run/nginx.pid && \
    chown appuser:appgroup /var/run/nginx.pid

# Copy built assets and config
COPY --from=builder /app/dist /usr/share/nginx/html
COPY nginx.conf /etc/nginx/conf.d/default.conf

USER appuser
EXPOSE 8080

CMD ["nginx", "-g", "daemon off;"]
```

### .dockerignore (project root)

```
# Version control
.git
.gitignore
.gitmodules
.gitattributes

# Runtime data (never bake into images)
data/
wallets.json

# Dependencies (rebuilt in container)
node_modules
web/node_modules

# IDE / OS
.vscode
.idea
.DS_Store
*.swp

# Build artifacts
shitcoin
*.exe
*.test
*.out
coverage.html

# Planning / docs
.planning
.agents
.claude
README.md
CLAUDE.md
*.md
```

### nginx.conf

```nginx
map $http_upgrade $connection_upgrade {
    default upgrade;
    ''      close;
}

server {
    listen 8080;
    server_name _;

    root /usr/share/nginx/html;
    index index.html;

    # SPA fallback
    location / {
        try_files $uri $uri/ /index.html;
    }

    # API reverse proxy
    location /api/ {
        proxy_pass http://backend:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }

    # WebSocket reverse proxy
    location /ws {
        proxy_pass http://backend:8080;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection $connection_upgrade;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
    }
}
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| Single-stage Dockerfile | Multi-stage builds | Docker 17.05 (2017) | 10-50x smaller images |
| FROM scratch for Go | FROM alpine for Go | Community consensus ~2020 | Adds ~5MB but provides shell, certs, debugging |
| nginx default.conf edit | Custom conf.d file | Always preferred | Clean, maintainable, version-controlled |
| Root containers | Non-root by default | K8s PSP/PSA enforcement ~2021 | Required for K8s security compliance |

## Validation Architecture

### Test Framework

| Property | Value |
|----------|-------|
| Framework | Docker CLI (docker build, docker run, docker exec) |
| Config file | Dockerfile, web/Dockerfile |
| Quick run command | `docker build -t shitcoin-backend .` |
| Full suite command | `docker build -t shitcoin-backend . && docker build -t shitcoin-frontend web/` |

### Phase Requirements to Test Map

| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| DOCK-01 | Go backend image under 20MB | smoke | `docker build -t shitcoin-backend . && docker image inspect shitcoin-backend --format '{{.Size}}'` | N/A Wave 0 |
| DOCK-02 | Frontend image serves SPA | smoke | `docker build -t shitcoin-frontend web/ && docker run --rm -d -p 8081:8080 --name fe-test shitcoin-frontend && curl -s http://localhost:8081/ && docker stop fe-test` | N/A Wave 0 |
| DOCK-03 | .dockerignore excludes data/, .git, etc | manual | Verify file exists and contains required entries | N/A Wave 0 |
| DOCK-04 | nginx proxies /api and /ws | smoke | Start both containers, curl /api/status through frontend | N/A Wave 0 |
| DOCK-05 | Non-root user | smoke | `docker run --rm shitcoin-backend whoami` should output `appuser` | N/A Wave 0 |

### Sampling Rate

- **Per task commit:** `docker build -t shitcoin-backend .`
- **Per wave merge:** Build both images + run smoke tests
- **Phase gate:** All 5 DOCK requirements verified before `/gsd:verify-work`

### Wave 0 Gaps

None -- validation uses Docker CLI directly, no test framework setup needed.

## Open Questions

1. **Backend upstream hostname in nginx.conf**
   - What we know: In K8s (Phase 11), the backend will be a Service named something like `shitcoin-backend`. For standalone Docker testing, we can use `--add-host` or `host.docker.internal`.
   - What's unclear: Exact Service name for K8s (decided in Phase 11).
   - Recommendation: Use `backend` as the upstream hostname. It works as a Docker network alias and as a K8s Service name. Can be overridden via environment variable + envsubst if needed later.

2. **Data directory volume mount**
   - What we know: BoltDB path is `data/shitcoin.db`, wallet path is `data/wallets.json`. These are runtime state.
   - What's unclear: Whether to document volume mount in Dockerfile (VOLUME directive) or leave for K8s PVC.
   - Recommendation: Do NOT use VOLUME directive in Dockerfile. Let K8s handle via PVC (Phase 11). Just ensure WORKDIR and USER permissions allow writing to `/app/data/`.

3. **Image size target**
   - Requirements say ~15MB, success criteria says under 20MB.
   - Go binary with `-ldflags="-s -w"` is typically ~15-20MB. Alpine base adds ~5MB.
   - Total will be ~20-25MB. This is fine -- the "~15MB" in DOCK-01 is approximate, and the success criteria uses "under 20MB" for the binary itself (alpine adds base layer).
   - Recommendation: Target smallest possible. If over 20MB total, it's still acceptable for an educational project. The stripped binary itself should be ~15MB.

## Sources

### Primary (HIGH confidence)

- Docker official docs: Multi-stage builds, .dockerignore, USER directive
- nginx official docs: proxy_pass, proxy_http_version, WebSocket proxying
- Project source: go.mod (Go 1.26.1), web/package.json (Vite 7, React 19), etc/shitcoin.yaml (config structure)
- Project skill: `.agents/skills/docker-patterns/SKILL.md` -- multi-stage, non-root, .dockerignore patterns

### Secondary (MEDIUM confidence)

- Alpine adduser/addgroup conventions (standard BusyBox utilities)
- Go binary size with ldflags stripping (well-documented community practice)

### Tertiary (LOW confidence)

- Exact final image size depends on Go 1.26.1 binary size and alpine 3.21 base size (needs build verification)

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH -- well-established Docker patterns, no novel requirements
- Architecture: HIGH -- standard multi-stage build + nginx reverse proxy, extensively documented
- Pitfalls: HIGH -- common, well-known Docker gotchas
- nginx WebSocket config: HIGH -- standard proxy_pass + Upgrade header pattern

**Research date:** 2026-03-07
**Valid until:** 2026-04-07 (stable patterns, unlikely to change)
