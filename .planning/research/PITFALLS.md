# Pitfalls Research

**Domain:** CI/CD and Kubernetes deployment for Go+React blockchain project
**Researched:** 2026-03-07
**Confidence:** HIGH

## Critical Pitfalls

### Pitfall 1: BoltDB File Locking Prevents Multi-Replica and Rolling Updates

**What goes wrong:**
BoltDB uses OS-level file locking (`flock`) -- only one process can open the database file at a time. If Kubernetes scales the Deployment to 2+ replicas, or uses the default `RollingUpdate` strategy (which runs old and new pods concurrently during rollout), the new pod panics with "database already locked" or hangs indefinitely.

**Why it happens:**
Developers treat the blockchain node as a stateless service and use a Deployment with the default `RollingUpdate` strategy. During a rollout, the new pod starts before the old pod terminates. Both attempt to open the same BoltDB file on the shared PVC. With `ReadWriteOnce`, only one node can mount the volume -- but if both pods land on the same node, they share the mount and fight for the flock.

**How to avoid:**
- Use `strategy.type: Recreate` on the Deployment to ensure the old pod terminates fully before the new pod starts. Or use a StatefulSet with `replicas: 1`.
- Set `terminationGracePeriodSeconds` to at least 30 seconds so BoltDB can cleanly close (flush mmap, release flock).
- Trap SIGTERM in the Go application and call `db.Close()` before exiting. Without this, the file lock may persist until the kernel cleans up.
- Use `ReadWriteOnce` PVC access mode. Never use `ReadWriteMany` for BoltDB volumes -- concurrent access corrupts the database.

**Warning signs:**
- Pod CrashLoopBackOff with "timeout" or "flock" in logs during rollouts.
- New pod stuck in `ContainerCreating` waiting for volume detach.
- Data corruption after force-killing a pod (BoltDB mmap not flushed).

**Phase to address:**
Kustomize manifests phase -- define Recreate strategy and graceful shutdown from the start.

---

### Pitfall 2: BoltDB Data Loss from Missing Persistent Volumes

**What goes wrong:**
Container restarts wipe all blockchain data -- chain DB, UTXO set, and wallet private keys are gone. The node restarts from genesis with new wallet addresses. Any mined coins and transaction history are permanently lost.

**Why it happens:**
The project's config uses relative paths (`data/shitcoin.db`, `data/wallets.json`). Without explicit volume mounts, these resolve inside the ephemeral container filesystem. It "works" during development because `docker run` keeps the container layer alive until removal. In Kubernetes, pods are ephemeral -- any restart (OOMKill, node drain, rollout) destroys the data.

**How to avoid:**
- Define a PersistentVolumeClaim in the Kustomize base and mount it at `/app/data`.
- Ensure the config file paths point to the mounted directory (either set `Storage.DBPath` and `Storage.WalletPath` via environment variables or a ConfigMap-mounted config file).
- For Tilt local dev, use a `host_path` volume pointing to a local directory so chain data survives `tilt down && tilt up`.
- Add a startup check that logs whether `/app/data` is on a persistent mount (check if it survives a `touch` + restart).

**Warning signs:**
- Chain height resets to 0 after pod restart.
- Wallet addresses differ between restarts.
- `kubectl exec -- df -h /app/data` shows overlay filesystem instead of a PVC.

**Phase to address:**
Kustomize manifests phase -- PVC must be in the base manifests before any deployment.

---

### Pitfall 3: Go Binary Crashes on Scratch/Alpine with "not found" Error

**What goes wrong:**
The multi-stage Docker build produces a Go binary that immediately crashes with `exec /app/shitcoin: not found` in the runtime stage. The binary file exists at the path -- the "not found" error refers to the missing dynamic linker (`ld-linux`), not the binary itself.

**Why it happens:**
Go defaults to `CGO_ENABLED=1` when the build environment OS matches the target. The builder stage (`golang:1.26-alpine` or `golang:1.26`) compiles a dynamically linked binary against glibc or musl. The runtime stage (`scratch`, `distroless`, or a different Alpine version) lacks the matching libc, so the kernel cannot find the dynamic linker and reports "not found."

**How to avoid:**
- Set `CGO_ENABLED=0` explicitly in the Dockerfile. The shitcoin project has zero CGO dependencies -- bbolt is pure Go, go-zero does not require CGO, and btcec uses pure Go crypto.
- Use `scratch` or `gcr.io/distroless/static` as the runtime image for minimal attack surface (~2MB vs ~5MB for Alpine).
- Add `-ldflags="-s -w"` to strip debug info, reducing binary size by ~30-40%.
- Copy CA certificates from the builder stage if the app makes HTTPS calls: `COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/`.

**Warning signs:**
- Container exits immediately with code 1, logs show "not found."
- `ldd` on the binary in the builder stage shows dynamic library dependencies.
- Works when using `golang:alpine` as runtime but fails on `scratch`.

**Phase to address:**
Dockerfile creation phase -- this must be right in the initial Dockerfile.

---

### Pitfall 4: P2P Networking Broken by Kubernetes Service Load Balancing

**What goes wrong:**
Blockchain nodes cannot discover or maintain persistent connections to specific peers. The P2P layer uses `-peers HOST:PORT` with the expectation of connecting to a specific node, but a Kubernetes ClusterIP Service load-balances across pods randomly. Node A connects to Node B, then a subsequent connection goes to Node C via the same service address.

**Why it happens:**
The default Kubernetes Service type (ClusterIP) provides a single virtual IP that round-robins across backend pods. This is ideal for stateless HTTP services but breaks P2P protocols that require stable, addressable peer identities. The shitcoin P2P handshake includes version exchange and chain state -- connecting to a random pod on each attempt means inconsistent peer state.

**How to avoid:**
- Use a Headless Service (`clusterIP: None`) combined with a StatefulSet. Each pod gets a stable DNS name: `shitcoin-0.shitcoin-headless.default.svc.cluster.local:3000`.
- Configure peer addresses using these stable DNS names in the node's config or startup arguments.
- For peer discovery, use an init container or startup script that resolves the headless service DNS to get all peer pod IPs, then passes them as `-peers` arguments.
- Expose both the HTTP API port (8080) and the P2P TCP port (3000) in the headless service.

**Warning signs:**
- Nodes start but peer count stays at 0.
- Handshake errors: nodes receive unexpected version messages from different peers.
- Connection established but immediately dropped (wrong peer responded).

**Phase to address:**
Kustomize manifests phase -- headless service and StatefulSet must be designed together for P2P to work.

---

### Pitfall 5: React SPA Returns 404 on Page Refresh in Nginx Container

**What goes wrong:**
The block explorer works when navigating by clicking links (React Router handles client-side routing), but directly visiting `/blocks/5` or refreshing the page returns Nginx's 404 page. All deep links to blocks, transactions, and addresses are broken.

**Why it happens:**
The Vite build produces static files in `dist/`. Nginx serves these files directly. When a request arrives for `/blocks/5`, Nginx looks for a file at that path, finds nothing, and returns 404. In development, Vite's dev server handles this by always serving `index.html` for any route. The production Nginx config must replicate this behavior.

**How to avoid:**
- Include a custom `nginx.conf` in the frontend Docker image with the SPA fallback:
  ```nginx
  location / {
      try_files $uri $uri/ /index.html;
  }
  ```
- Set aggressive caching for Vite's hashed assets (`/assets/*` with `max-age=31536000, immutable`) since Vite content-hashes every filename.
- Set `no-cache` on `index.html` itself so new deployments take effect immediately.
- Add an `/api` proxy block pointing to the backend Kubernetes Service for production API routing (replacing Vite's dev proxy).

**Warning signs:**
- 404 on any page refresh or direct URL access.
- Browser shows Nginx default error page instead of the React app.

**Phase to address:**
Dockerfile creation phase -- the Nginx config must be baked into the frontend image.

---

### Pitfall 6: ArgoCD Perpetual OutOfSync from Mutable Kubernetes Fields

**What goes wrong:**
ArgoCD shows the application as permanently "OutOfSync" even with no Git changes. It may continuously sync, causing unnecessary pod restarts, or it flaps between Synced and OutOfSync states.

**Why it happens:**
Kubernetes mutates resources after apply -- admission controllers inject defaults, `metadata.creationTimestamp` is added, `status` fields are populated by controllers. ArgoCD compares the desired state (from Git/Kustomize) to the live state and detects these server-side mutations as drift. Additionally, Kustomize-generated ConfigMaps with hash suffixes can produce different output between ArgoCD's Kustomize version and the local version.

**How to avoid:**
- Configure `resource.customizations.ignoreDifferences` in ArgoCD for known mutable fields (status, creationTimestamp, managed-fields).
- Pin the Kustomize version in ArgoCD's ConfigMap to match the version used locally. Run `kustomize build` twice and diff the output to verify determinism.
- Test sync stability: deploy once, wait 5 minutes with no changes, verify the app stays Synced.
- Use `argocd.argoproj.io/sync-options: Prune=false` on resources that should not be auto-pruned (like PVCs).

**Warning signs:**
- Application status flaps between Synced and OutOfSync in the ArgoCD UI.
- Diff view shows changes on fields you did not modify.
- High CPU on argocd-repo-server from constant re-rendering.

**Phase to address:**
ArgoCD setup phase -- configure ignore rules during initial application creation.

---

### Pitfall 7: GitHub Actions Cache Collisions Between Build and Test Jobs

**What goes wrong:**
CI builds are slower than expected despite caching, or test jobs with `-race` flag always rebuild the entire dependency tree even though modules are cached.

**Why it happens:**
`actions/setup-go` v4+ auto-caches `GOMODCACHE` and `GOCACHE` using `go.sum` as the cache key. Cache entries are immutable -- first write wins. If the `build` job finishes before the `test -race` job, it caches build artifacts without race detector instrumentation. The test job restores this incomplete cache and must rebuild everything with `-race`. Since the cache key matches, it cannot write a better cache.

**How to avoid:**
- Either disable setup-go's built-in caching (`cache: false`) and manage caching explicitly with `actions/cache` using job-specific keys (e.g., `go-${{ runner.os }}-${{ hashFiles('go.sum') }}-test-race`).
- Or structure the workflow so the test job runs first (and caches race-instrumented artifacts that the build job can also use).
- For this project: the dependency set is small (~15 direct deps). Consider skipping Go module caching entirely and relying on Docker layer cache for the image build step. Simpler is better when caching saves under 30 seconds.

**Warning signs:**
- CI logs show `downloading` lines for modules that should be cached.
- Test jobs with `-race` always show full recompilation.
- Parallel jobs have inconsistent cache behavior.

**Phase to address:**
GitHub Actions CI phase -- configure caching strategy in the initial workflow YAML.

## Technical Debt Patterns

| Shortcut | Immediate Benefit | Long-term Cost | When Acceptable |
|----------|-------------------|----------------|-----------------|
| Single monolithic Dockerfile for Go backend + React frontend | One image to build and deploy | Larger image (~300MB+ vs ~20MB), cannot scale independently, slow rebuilds when only frontend changes | Never -- always use separate Dockerfiles |
| Hardcoded image tags in K8s manifests | Quick to get running initially | Breaks GitOps (tag must match CI output), manual updates on every change | Only during initial Tilt local dev, never in Kustomize overlays |
| Skipping health probes | Pods start faster, less config to write | K8s routes traffic to unready pods, no auto-restart on hangs, rolling updates proceed before app is healthy | Never -- add liveness + readiness probes from day one |
| Using `latest` image tag | No need to update manifests | Non-reproducible deploys, ArgoCD cannot detect changes, rollback impossible | Never in any Kustomize manifest |
| Config baked into Docker image | No ConfigMap needed | Must rebuild image for any config change, cannot vary per environment | Acceptable only for truly static config like Nginx's SPA fallback rule |
| Tilt rebuilds full image on every change | Simple Tiltfile, no live_update config | 30-60 second rebuild cycle kills developer productivity | Only for initial Tilt setup; add `live_update` immediately after |

## Integration Gotchas

| Integration | Common Mistake | Correct Approach |
|-------------|----------------|------------------|
| Tilt + Kustomize | Tiltfile uses `k8s_yaml('deploy.yaml')` with raw YAML instead of rendered Kustomize | Use `k8s_yaml(kustomize('./k8s/overlays/dev'))` so Tilt renders Kustomize and picks up all overlays |
| Tilt + Go hot reload | Using `docker_build` for every code change, full image rebuild each time | Use `live_update` with `sync('./internal', '/app/internal')` and a `run('go build -o /app/shitcoin ./cmd/shitcoin')` step |
| Frontend Nginx + Backend in K8s | Nginx proxies `/api` to `localhost:8080` (copy-pasted from Vite dev config) | Nginx `proxy_pass` must point to the backend K8s Service: `proxy_pass http://shitcoin-backend:8080;` |
| ArgoCD + Kustomize version mismatch | ArgoCD uses bundled Kustomize v5.x while local uses v4.x, producing different manifest output | Pin Kustomize version in ArgoCD ConfigMap and install matching version locally |
| GitHub Actions + GHCR | Building Docker image but not pushing to a registry that K8s can pull from | Push to GHCR with `docker/build-push-action`, tag with Git SHA for immutability |
| BoltDB + SIGTERM | Container killed without closing BoltDB, file left with stale lock | Trap SIGTERM/SIGINT in Go main, call `db.Close()`, set `terminationGracePeriodSeconds: 30` |
| WebSocket through Nginx | Nginx blocks WebSocket upgrade, frontend shows connection errors | Add `proxy_http_version 1.1; proxy_set_header Upgrade $http_upgrade; proxy_set_header Connection "upgrade";` to the `/ws` location block |

## Performance Traps

| Trap | Symptoms | Prevention | When It Breaks |
|------|----------|------------|----------------|
| BoltDB mmap exceeds container memory limit | OOMKilled pods, repeated restarts | Set container memory limit well above expected DB size. bbolt mmaps the entire file. | When chain DB approaches memory limit (unlikely in educational project, but set limit to 512Mi+) |
| Tilt watching `node_modules` and `data/` | Constant rebuilds, high CPU, Tilt unusable | Add `.tiltignore` with `node_modules/`, `data/`, `dist/`, `.git/`, `web/node_modules/` | Immediately on first `tilt up` without ignore rules |
| Docker image built on every push to every branch | Exhausts GitHub Actions minutes, bloats container registry | Only build+push images on `main` or tags; run tests on all branches but skip image build | When monthly Actions minutes run out (~2000 for free tier) |
| Kustomize rendering on every ArgoCD poll | High CPU on repo-server, slow sync detection | Use `argocd.argoproj.io/manifest-generate-paths` to scope which paths trigger re-render | Not a concern at this project size, but good practice to establish |

## Security Mistakes

| Mistake | Risk | Prevention |
|---------|------|------------|
| Wallet private keys baked into Docker image layers | Anyone with image access owns all wallets; GHCR images may be public | Add `data/` and `*.db` and `wallets.json` to `.dockerignore`; mount wallet data via PVC |
| Running containers as root | Container escape risk; files written to PVC as root cause permission issues for non-root pods | Add `USER nonroot:nonroot` in Dockerfile; set `runAsNonRoot: true` and `readOnlyRootFilesystem: true` in pod securityContext |
| Secrets in workflow YAML | GHCR tokens or ArgoCD credentials exposed in git | Use `${{ secrets.GITHUB_TOKEN }}` for GHCR (auto-provided); store ArgoCD creds in repository secrets |
| Using `pull_request_target` trigger for CI | Untrusted fork PRs execute workflow with secret access | Use `pull_request` trigger (no secret access needed for test/lint/build); only `push` to main needs secrets for image push |
| BoltDB file world-readable in container | Any process in container can read wallet references | Set `0600` on DB files; run as dedicated non-root user with `fsGroup` in securityContext |

## UX Pitfalls

| Pitfall | User Impact | Better Approach |
|---------|-------------|-----------------|
| Tilt requires 10+ manual steps before `tilt up` works | Developer gives up, falls back to `go run` | Provide a `make dev-k8s` that creates kind cluster, local registry, and runs `tilt up` |
| No clear distinction between local dev and K8s dev workflows | Developer confused about which mode to use, breaks one fixing the other | Document both paths; `go run` for quick iteration, `tilt up` for integration testing |
| CI feedback loop exceeds 5 minutes | Developer context-switches, merges without waiting for green | Parallelize lint/test/build jobs; fail fast on lint; cache aggressively |
| ArgoCD UI not accessible without port-forward | Cannot see deployment status without terminal | Include port-forward command in Makefile (`make argocd-ui`) |

## "Looks Done But Isn't" Checklist

- [ ] **Dockerfile:** Builds and runs locally -- verify it also works in CI (no local Docker cache, different build context)
- [ ] **Kustomize overlays:** `kustomize build k8s/base` works -- verify each overlay (`dev`, `prod`) also renders cleanly
- [ ] **Health probes:** Pods reach Running state -- verify readiness probe checks actual app health (hit `/api/status`), not just TCP port open
- [ ] **PVC persistence:** Data survives `kubectl rollout restart` -- verify it also survives `tilt down && tilt up` and pod OOMKill
- [ ] **P2P in K8s:** Two nodes connect via `testnet` command locally -- verify they connect via headless service DNS inside K8s pods
- [ ] **CI pipeline:** Tests pass in GitHub Actions -- verify Docker image build also succeeds (different cache, no local Go modules)
- [ ] **ArgoCD sync:** Application syncs initially -- verify it stays Synced after 5 minutes with zero Git changes (no sync loop)
- [ ] **SPA routing:** Nginx serves the dashboard -- verify direct navigation to `/blocks/1` works (not just clicking from homepage)
- [ ] **WebSocket:** Works through Vite dev proxy -- verify it also works through production Nginx reverse proxy config
- [ ] **Graceful shutdown:** App responds to requests -- verify `kubectl delete pod` triggers clean BoltDB close (check logs for close confirmation)

## Recovery Strategies

| Pitfall | Recovery Cost | Recovery Steps |
|---------|---------------|----------------|
| BoltDB data loss (no PVC) | HIGH | Chain must be re-synced from peers or rebuilt from genesis. Wallet keys are permanently lost. Add PVC, restore from backup if any exists. |
| Corrupted BoltDB from ungraceful shutdown | MEDIUM | Delete the `.db` file, restart node to resync from peers. If single node, chain is lost. Add SIGTERM handler to prevent recurrence. |
| CGO binary crash on scratch | LOW | Add `CGO_ENABLED=0` to Dockerfile, rebuild and redeploy. No data loss. 5-minute fix. |
| ArgoCD sync loop | LOW | Add `resource.customizations.ignoreDifferences` for affected fields, manual sync to stabilize. |
| CI cache corruption | LOW | Delete cache entries via GitHub Actions cache management UI, re-run workflow. |
| P2P broken by ClusterIP service | MEDIUM | Replace Service with headless service, update manifests, redeploy. Requires understanding K8s DNS. |
| SPA 404 on refresh | LOW | Add `try_files` to Nginx config, rebuild frontend image. 10-minute fix. |
| Docker image pushed with secrets | HIGH | Rotate all exposed credentials immediately. Delete compromised image tags from registry. Add `.dockerignore` entries. |

## Pitfall-to-Phase Mapping

| Pitfall | Prevention Phase | Verification |
|---------|------------------|--------------|
| BoltDB file locking on rollout | Kustomize manifests | Deploy, trigger rollout, verify old pod terminates before new pod starts |
| BoltDB data loss (no PVC) | Kustomize manifests | Restart pod, verify chain height persists across restart |
| CGO dynamic binary crash | Dockerfile creation | Run `file` on binary in final stage, verify "statically linked" |
| P2P broken by load balancer | Kustomize manifests | Deploy 2+ nodes, verify peer handshake succeeds in pod logs |
| SPA 404 on refresh | Dockerfile creation | Navigate directly to `/blocks/1` in browser, verify page loads |
| ArgoCD sync loop | ArgoCD setup | Deploy, wait 5 min, verify status remains Synced |
| CI cache conflicts | GitHub Actions | Run CI twice, verify second run is faster and passes |
| Wallet keys in Docker image | Dockerfile creation | `docker history --no-trunc` on image, verify no wallet data in any layer |
| Container running as root | Dockerfile creation | `kubectl exec -- whoami` returns non-root user |
| Tilt watches node_modules | Tilt setup | Edit Go file, verify Tilt rebuild does not re-sync frontend deps |
| WebSocket blocked by Nginx | Dockerfile creation | Open block explorer, verify live mining updates appear via WebSocket |
| Ungraceful BoltDB shutdown | Kustomize + Go code | `kubectl delete pod`, verify clean shutdown logs with "database closed" |

## Sources

- [bbolt GitHub - file locking, mmap, concurrency model](https://github.com/etcd-io/bbolt) - HIGH confidence
- [Kubernetes Headless Services for P2P communication](https://kubernetes.io/docs/concepts/services-networking/service/) - HIGH confidence
- [Kubernetes Persistent Volumes](https://kubernetes.io/docs/concepts/storage/persistent-volumes/) - HIGH confidence
- [GitHub Actions Go caching pitfalls](https://danp.net/posts/github-actions-go-cache/) - MEDIUM confidence
- [GitHub Actions cache service migration (Feb 2025)](https://www.herodevs.com/blog-posts/github-actions-cache-service-goes-dark-what-devops-teams-need-to-know) - HIGH confidence
- [ArgoCD anti-patterns for GitOps](https://codefresh.io/blog/argo-cd-anti-patterns-for-gitops/) - MEDIUM confidence
- [ArgoCD Kustomize integration docs](https://argo-cd.readthedocs.io/en/stable/user-guide/kustomize/) - HIGH confidence
- [Multi-stage Docker builds for Go](https://medium.com/@kittipat_1413/optimizing-multi-stage-builds-with-dockerfile-in-golang-a2ee8ed37ec6) - MEDIUM confidence
- [SPA Nginx Docker containerization guide](https://dev.to/it-wibrc/guide-to-containerizing-a-modern-javascript-spa-vuevitereact-with-a-multi-stage-nginx-build-1lma) - MEDIUM confidence
- [Tilt FAQ and debugging](https://docs.tilt.dev/faq.html) - HIGH confidence
- [Tilt choosing local clusters](https://docs.tilt.dev/choosing_clusters.html) - HIGH confidence
- [7 Common Kubernetes Pitfalls (official blog)](https://kubernetes.io/blog/2025/10/20/seven-kubernetes-pitfalls-and-how-to-avoid/) - HIGH confidence
- [Deploying Go to Production 2026](https://dasroot.net/posts/2026/03/deploying-go-applications-production-best-practices-tools/) - MEDIUM confidence
- [Docker Volumes for Persistent Data](https://oneuptime.com/blog/post/2026-02-02-docker-volumes-persistent-data/view) - MEDIUM confidence
- [bbolt mmap issues with Kubernetes PVCs](https://github.com/etcd-io/etcd/discussions/18101) - HIGH confidence

---
*Pitfalls research for: CI/CD and Kubernetes deployment of shitcoin blockchain project*
*Researched: 2026-03-07*
