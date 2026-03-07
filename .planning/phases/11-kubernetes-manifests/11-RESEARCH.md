# Phase 11: Kubernetes Manifests - Research

**Researched:** 2026-03-07
**Domain:** Kustomize-based Kubernetes manifests for a Go/React blockchain app
**Confidence:** HIGH

## Summary

This phase creates a complete Kustomize manifest set under `deploy/k8s/` that defines Kubernetes Deployments, Services, PVCs, ConfigMaps, and health probes for the shitcoin backend and frontend. The backend uses BoltDB (a single-writer embedded database), which constrains the deployment to Recreate strategy with exactly one replica and a PersistentVolumeClaim for data durability.

The project already has working Dockerfiles (Phase 9) and GHCR image push (Phase 10). The nginx.conf already proxies `/api/` and `/ws` to `http://backend:8080`, meaning the frontend Service simply needs the name `backend` to match. The `/api/status` endpoint already exists and returns JSON -- it is a suitable health probe target.

**Primary recommendation:** Use Kustomize base+overlays structure under `deploy/k8s/` with configMapGenerator for shitcoin.yaml, Recreate strategy with 1 replica for backend, and dev/prod overlays for resource limits and image tags.

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|-----------------|
| K8S-01 | Kustomize base defines Deployment + Service for backend and frontend | Base kustomization.yaml with two Deployments and two Services |
| K8S-02 | Kustomize base includes PVC for BoltDB data persistence | PVC manifest + volumeMount on /app/data in backend Deployment |
| K8S-03 | Kustomize base uses configMapGenerator to externalize shitcoin.yaml | configMapGenerator with files: directive sourcing shitcoin.yaml |
| K8S-04 | Backend Deployment uses Recreate strategy with single replica for BoltDB safety | `strategy: type: Recreate` + `replicas: 1` in backend Deployment |
| K8S-05 | Kustomize dev overlay configures local image refs and lower resource limits | Dev overlay with newName/newTag for local images and minimal resources |
| K8S-06 | Kustomize prod overlay configures pinned image tags and production resource limits | Prod overlay with GHCR image refs, SHA tags, and higher resource limits |
| K8S-07 | Health probes (liveness + readiness) configured on /api/status | httpGet probes on port 8080 path /api/status |
</phase_requirements>

## Standard Stack

### Core
| Tool | Version | Purpose | Why Standard |
|------|---------|---------|--------------|
| Kustomize | Built into kubectl (v5.x) | Template-free K8s manifest management | Simpler than Helm for educational projects; no Go template DSL |
| kubectl | 1.29+ | K8s CLI for apply/diff | Standard K8s tooling |

### Supporting
| Resource | API Version | Purpose | When to Use |
|----------|-------------|---------|-------------|
| Deployment | apps/v1 | Pod management with strategy control | Both backend and frontend |
| Service | v1 | Internal cluster networking | Expose pods within cluster |
| PersistentVolumeClaim | v1 | Durable storage for BoltDB | Backend only |
| ConfigMap (generated) | v1 | Externalize shitcoin.yaml config | Backend only |

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| Kustomize | Helm | More powerful but adds Go template complexity -- out of scope per REQUIREMENTS.md |
| Deployment+PVC | StatefulSet | Overkill for single-replica; no stable network identity needed |
| ConfigMap from file | ConfigMap literal | File-based keeps config in its original YAML format, easier to diff |

## Architecture Patterns

### Recommended Project Structure
```
deploy/k8s/
├── base/
│   ├── kustomization.yaml          # Resources + configMapGenerator
│   ├── backend-deployment.yaml     # Recreate strategy, 1 replica, probes, PVC mount
│   ├── backend-service.yaml        # ClusterIP on port 8080
│   ├── frontend-deployment.yaml    # Rolling update, nginx container
│   ├── frontend-service.yaml       # ClusterIP on port 8080
│   ├── backend-pvc.yaml            # 1Gi RWO for /app/data
│   └── shitcoin.yaml               # Config file for configMapGenerator
├── overlays/
│   ├── dev/
│   │   └── kustomization.yaml      # Local image refs, low resources
│   └── prod/
│       └── kustomization.yaml      # GHCR images, pinned tags, higher resources
```

### Pattern 1: Kustomize Base + Overlays
**What:** Base directory contains complete, valid manifests. Overlays patch specific fields (images, resources, replicas) without duplicating the full manifest.
**When to use:** Always with Kustomize -- this is the standard pattern.
**Example:**
```yaml
# base/kustomization.yaml
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization

resources:
  - backend-deployment.yaml
  - backend-service.yaml
  - backend-pvc.yaml
  - frontend-deployment.yaml
  - frontend-service.yaml

configMapGenerator:
  - name: shitcoin-config
    files:
      - shitcoin.yaml
```

### Pattern 2: configMapGenerator with Hash Suffix
**What:** Kustomize generates ConfigMaps with hash suffixes. When config content changes, the hash changes, which triggers a pod restart automatically.
**When to use:** Externalizing config files like shitcoin.yaml.
**Example:**
```yaml
# In kustomization.yaml
configMapGenerator:
  - name: shitcoin-config
    files:
      - shitcoin.yaml

# In deployment, reference by generator name (Kustomize patches the hash automatically)
volumes:
  - name: config
    configMap:
      name: shitcoin-config
volumeMounts:
  - name: config
    mountPath: /app/etc
    readOnly: true
```

### Pattern 3: Overlay Image Override
**What:** Overlays use the `images` transformer to change image names and tags without patching the full Deployment.
**Example:**
```yaml
# overlays/dev/kustomization.yaml
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization

resources:
  - ../../base

images:
  - name: shitcoin-backend
    newName: shitcoin-backend
    newTag: latest
  - name: shitcoin-frontend
    newName: shitcoin-frontend
    newTag: latest

patches:
  - target:
      kind: Deployment
    patch: |-
      apiVersion: apps/v1
      kind: Deployment
      metadata:
        name: not-important
      spec:
        template:
          spec:
            containers:
              - name: backend
                resources:
                  requests:
                    cpu: 50m
                    memory: 64Mi
                  limits:
                    cpu: 200m
                    memory: 128Mi
```

### Anti-Patterns to Avoid
- **Duplicating full manifests in overlays:** Use patches and `images` transformer instead. Overlays should be tiny.
- **Hardcoding image tags in base:** Base uses placeholder image names; overlays set the actual tags.
- **Using RollingUpdate for BoltDB backend:** BoltDB only allows one writer. Two pods running simultaneously during a rolling update will cause data corruption.
- **Baking config into the Docker image:** The Dockerfile currently COPYs shitcoin.yaml into the image. In K8s, the ConfigMap volume mount at /app/etc overrides this, which is the correct approach.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Config injection | Custom entrypoint scripts to generate config | Kustomize configMapGenerator + volume mount | Hash suffix triggers automatic pod restart on config change |
| Image tag management | sed/envsubst scripts to patch YAML | Kustomize `images` transformer | Built-in, declarative, no scripting needed |
| Resource limit differences | Separate manifest files per environment | Kustomize overlay patches | Single source of truth in base, minimal diffs in overlays |

## Common Pitfalls

### Pitfall 1: RollingUpdate with BoltDB
**What goes wrong:** Two backend pods run simultaneously during update, both try to open the same BoltDB file, one gets a lock error or data corrupts.
**Why it happens:** RollingUpdate is the Kubernetes default strategy.
**How to avoid:** Explicitly set `strategy: type: Recreate` and `replicas: 1` in backend Deployment.
**Warning signs:** Pod CrashLoopBackOff with "database locked" errors.

### Pitfall 2: PVC AccessMode Mismatch
**What goes wrong:** PVC created with ReadWriteMany but the storage class only supports ReadWriteOnce, causing the PVC to stay in Pending state.
**Why it happens:** Most local storage provisioners (kind, minikube) only support RWO.
**How to avoid:** Use `ReadWriteOnce` access mode -- this is correct for single-replica Recreate deployments.
**Warning signs:** PVC stuck in Pending, pod stuck in ContainerCreating.

### Pitfall 3: ConfigMap Mount Path Collision
**What goes wrong:** Volume mount at `/app/etc` replaces the entire directory. If the image has other files in `/app/etc`, they become invisible.
**Why it happens:** Kubernetes volume mounts shadow the underlying filesystem.
**How to avoid:** Mount to `/app/etc` -- this is safe because the only file in that directory is shitcoin.yaml. Alternatively, use `subPath` to mount a single file, but this disables the automatic ConfigMap update behavior.
**Warning signs:** Application can't find config file despite ConfigMap being mounted.

### Pitfall 4: Frontend Service Name Must Be "backend"
**What goes wrong:** Frontend nginx config proxies to `http://backend:8080`. If the backend Service is named something else (e.g., `shitcoin-backend`), nginx returns 502.
**Why it happens:** The nginx.conf from Phase 9 hardcodes `backend` as the upstream hostname.
**How to avoid:** Name the backend Service `backend`, OR update nginx.conf, OR use an environment variable in nginx. Simplest approach: name the K8s Service `backend`.
**Warning signs:** Frontend loads but all API calls fail with 502/504.

### Pitfall 5: Config File Path in Container
**What goes wrong:** The Dockerfile CMD uses `./app -f etc/shitcoin.yaml startnode` (relative path). If the ConfigMap mounts to a different path, the app won't find the config.
**Why it happens:** Mismatch between the CMD in Dockerfile and the volume mount path.
**How to avoid:** Mount the ConfigMap to `/app/etc` so the relative path `etc/shitcoin.yaml` resolves correctly from WORKDIR `/app`. No need to change the Dockerfile CMD.

### Pitfall 6: Graceful Shutdown / SIGTERM Handling
**What goes wrong:** BoltDB database not cleanly closed on pod termination, potentially corrupting data.
**Why it happens:** Kubernetes sends SIGTERM, then SIGKILL after terminationGracePeriodSeconds (default 30s). If the app doesn't handle SIGTERM, BoltDB may not flush.
**How to avoid:** Verify the Go app handles SIGTERM gracefully (it likely does via go-zero's framework). Set a reasonable `terminationGracePeriodSeconds` (30s default is fine).
**Warning signs:** Data loss or corruption after pod restart.

## Code Examples

### Backend Deployment with Recreate Strategy, PVC, and Probes
```yaml
# deploy/k8s/base/backend-deployment.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: backend
  labels:
    app: shitcoin
    component: backend
spec:
  replicas: 1
  strategy:
    type: Recreate
  selector:
    matchLabels:
      app: shitcoin
      component: backend
  template:
    metadata:
      labels:
        app: shitcoin
        component: backend
    spec:
      containers:
        - name: backend
          image: shitcoin-backend
          ports:
            - containerPort: 8080
          volumeMounts:
            - name: data
              mountPath: /app/data
            - name: config
              mountPath: /app/etc
              readOnly: true
          livenessProbe:
            httpGet:
              path: /api/status
              port: 8080
            initialDelaySeconds: 10
            periodSeconds: 30
            failureThreshold: 3
          readinessProbe:
            httpGet:
              path: /api/status
              port: 8080
            initialDelaySeconds: 5
            periodSeconds: 10
            failureThreshold: 2
          resources:
            requests:
              cpu: 100m
              memory: 128Mi
            limits:
              cpu: 500m
              memory: 256Mi
      volumes:
        - name: data
          persistentVolumeClaim:
            claimName: backend-data
        - name: config
          configMap:
            name: shitcoin-config
```

### Backend PVC
```yaml
# deploy/k8s/base/backend-pvc.yaml
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: backend-data
  labels:
    app: shitcoin
    component: backend
spec:
  accessModes:
    - ReadWriteOnce
  resources:
    requests:
      storage: 1Gi
```

### Backend Service (named "backend" for nginx compatibility)
```yaml
# deploy/k8s/base/backend-service.yaml
apiVersion: v1
kind: Service
metadata:
  name: backend
  labels:
    app: shitcoin
    component: backend
spec:
  selector:
    app: shitcoin
    component: backend
  ports:
    - port: 8080
      targetPort: 8080
      protocol: TCP
```

### Frontend Deployment
```yaml
# deploy/k8s/base/frontend-deployment.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: frontend
  labels:
    app: shitcoin
    component: frontend
spec:
  replicas: 1
  selector:
    matchLabels:
      app: shitcoin
      component: frontend
  template:
    metadata:
      labels:
        app: shitcoin
        component: frontend
    spec:
      containers:
        - name: frontend
          image: shitcoin-frontend
          ports:
            - containerPort: 8080
```

### Kustomize Base kustomization.yaml
```yaml
# deploy/k8s/base/kustomization.yaml
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization

resources:
  - backend-deployment.yaml
  - backend-service.yaml
  - backend-pvc.yaml
  - frontend-deployment.yaml
  - frontend-service.yaml

configMapGenerator:
  - name: shitcoin-config
    files:
      - shitcoin.yaml
```

### Dev Overlay
```yaml
# deploy/k8s/overlays/dev/kustomization.yaml
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization

resources:
  - ../../base

images:
  - name: shitcoin-backend
    newName: shitcoin-backend
    newTag: latest
  - name: shitcoin-frontend
    newName: shitcoin-frontend
    newTag: latest

patches:
  - target:
      kind: Deployment
      name: backend
    patch: |-
      apiVersion: apps/v1
      kind: Deployment
      metadata:
        name: backend
      spec:
        template:
          spec:
            containers:
              - name: backend
                resources:
                  requests:
                    cpu: 50m
                    memory: 64Mi
                  limits:
                    cpu: 200m
                    memory: 128Mi
  - target:
      kind: Deployment
      name: frontend
    patch: |-
      apiVersion: apps/v1
      kind: Deployment
      metadata:
        name: frontend
      spec:
        template:
          spec:
            containers:
              - name: frontend
                resources:
                  requests:
                    cpu: 25m
                    memory: 32Mi
                  limits:
                    cpu: 100m
                    memory: 64Mi
```

### Prod Overlay
```yaml
# deploy/k8s/overlays/prod/kustomization.yaml
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization

resources:
  - ../../base

images:
  - name: shitcoin-backend
    newName: ghcr.io/baotoq/shitcoin
    newTag: sha-abc1234  # Pinned to specific commit SHA
  - name: shitcoin-frontend
    newName: ghcr.io/baotoq/shitcoin-web
    newTag: sha-abc1234  # Pinned to specific commit SHA

patches:
  - target:
      kind: Deployment
      name: backend
    patch: |-
      apiVersion: apps/v1
      kind: Deployment
      metadata:
        name: backend
      spec:
        template:
          spec:
            containers:
              - name: backend
                resources:
                  requests:
                    cpu: 200m
                    memory: 256Mi
                  limits:
                    cpu: "1"
                    memory: 512Mi
  - target:
      kind: Deployment
      name: frontend
    patch: |-
      apiVersion: apps/v1
      kind: Deployment
      metadata:
        name: frontend
      spec:
        template:
          spec:
            containers:
              - name: frontend
                resources:
                  requests:
                    cpu: 50m
                    memory: 64Mi
                  limits:
                    cpu: 200m
                    memory: 128Mi
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| kubectl create configmap | configMapGenerator in kustomization.yaml | Kustomize 3.x+ (2020) | Hash suffix triggers automatic pod restart on config change |
| Strategic merge patches only | Both strategic merge and JSON 6902 patches | Kustomize 4.x+ | More flexible overlay patches |
| Separate kustomize binary | Built into kubectl (`kubectl apply -k`) | kubectl 1.14+ (2019) | No separate installation needed |

## Open Questions

1. **SIGTERM Handling**
   - What we know: BoltDB needs clean shutdown. Go-zero framework likely handles SIGTERM.
   - What's unclear: Whether the current shitcoin code properly closes BoltDB on SIGTERM.
   - Recommendation: Verify during implementation. If not handled, this is a Phase 11 concern flagged in STATE.md.

2. **Storage Class**
   - What we know: Kind uses `standard` StorageClass by default. Minikube uses `standard`. Cloud providers vary.
   - What's unclear: Which local K8s environment users will run.
   - Recommendation: Do NOT specify storageClassName in PVC -- let the cluster default handle it. This works on kind, minikube, and most cloud providers.

## Validation Architecture

### Test Framework
| Property | Value |
|----------|-------|
| Framework | kubectl + kustomize (built-in) |
| Config file | N/A -- manifests are the config |
| Quick run command | `kubectl kustomize deploy/k8s/overlays/dev` (dry-run render) |
| Full suite command | `kubectl apply -k deploy/k8s/overlays/dev --dry-run=client` |

### Phase Requirements to Test Map
| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| K8S-01 | Base Deployment+Service for backend and frontend | smoke | `kubectl kustomize deploy/k8s/base \| grep -c 'kind: Deployment'` (expect 2) | N/A Wave 0 |
| K8S-02 | PVC for BoltDB data persistence | smoke | `kubectl kustomize deploy/k8s/base \| grep 'kind: PersistentVolumeClaim'` | N/A Wave 0 |
| K8S-03 | configMapGenerator externalizes shitcoin.yaml | smoke | `kubectl kustomize deploy/k8s/base \| grep 'kind: ConfigMap'` | N/A Wave 0 |
| K8S-04 | Recreate strategy with single replica | smoke | `kubectl kustomize deploy/k8s/base \| grep -A1 'strategy'` (expect Recreate) | N/A Wave 0 |
| K8S-05 | Dev overlay with local images and low resources | smoke | `kubectl kustomize deploy/k8s/overlays/dev \| grep 'newTag\|limits'` | N/A Wave 0 |
| K8S-06 | Prod overlay with pinned tags and prod resources | smoke | `kubectl kustomize deploy/k8s/overlays/prod \| grep 'ghcr.io'` | N/A Wave 0 |
| K8S-07 | Health probes on /api/status | smoke | `kubectl kustomize deploy/k8s/base \| grep '/api/status'` | N/A Wave 0 |

### Sampling Rate
- **Per task commit:** `kubectl kustomize deploy/k8s/overlays/dev` (valid YAML renders without error)
- **Per wave merge:** `kubectl apply -k deploy/k8s/overlays/dev --dry-run=client && kubectl apply -k deploy/k8s/overlays/prod --dry-run=client`
- **Phase gate:** Both overlays render valid YAML with all expected resources

### Wave 0 Gaps
None -- Kustomize manifests are validated by `kubectl kustomize` rendering (no separate test framework needed).

## Sources

### Primary (HIGH confidence)
- [Kubernetes official docs - Kustomize](https://kubernetes.io/docs/tasks/manage-kubernetes-objects/kustomization/) - kustomization.yaml structure, configMapGenerator, patches
- [kubernetes-sigs/kustomize examples](https://github.com/kubernetes-sigs/kustomize/blob/master/examples/configGeneration.md) - configMapGenerator patterns
- Project codebase: Dockerfile, nginx.conf, shitcoin.yaml, /api/status handler - verified all integration points

### Secondary (MEDIUM confidence)
- [Kustomize configMapGenerator guide](https://oneuptime.com/blog/post/2026-02-09-kustomize-configmapgenerator/view) - hash suffix behavior verification
- [Kubernetes deployment strategies](https://www.groundcover.com/blog/kubernetes-deployment-strategies) - Recreate strategy behavior

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH - Kustomize is built into kubectl, well-documented, and explicitly chosen by project requirements
- Architecture: HIGH - base+overlays is the standard Kustomize pattern; BoltDB constraints are well-understood
- Pitfalls: HIGH - BoltDB single-writer constraint and nginx service name dependency verified from project source code

**Research date:** 2026-03-07
**Valid until:** 2026-04-07 (stable domain, Kustomize API is stable)
