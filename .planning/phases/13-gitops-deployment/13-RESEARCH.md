# Phase 13: GitOps Deployment - Research

**Researched:** 2026-03-07
**Domain:** ArgoCD Application CR, GitOps with Kustomize
**Confidence:** HIGH

## Summary

Phase 13 is a straightforward, single-file task: create an ArgoCD Application custom resource that points to the existing Kustomize dev overlay at `deploy/k8s/overlays/dev/` and enables auto-sync. The Application manifest must live in a separate `argocd/` directory so ArgoCD does not recursively watch itself.

This is the simplest phase in the entire v1.1 milestone. The ArgoCD Application CR is a well-documented, stable Kubernetes custom resource with a small API surface. The entire deliverable is one YAML file in a new directory.

**Primary recommendation:** Create `argocd/application.yaml` with an Application CR pointing to `deploy/k8s/overlays/dev`, using `syncPolicy.automated` with `prune: true` and `selfHeal: true`.

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|-----------------|
| GIT-01 | ArgoCD Application CR with auto-sync pointing to Kustomize dev overlay | Application CR spec with syncPolicy.automated, source.path pointing to deploy/k8s/overlays/dev |
| GIT-02 | ArgoCD Application CR lives outside K8s manifest watched path | Place in argocd/ directory, which is separate from deploy/k8s/ that ArgoCD watches |
</phase_requirements>

## Standard Stack

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| ArgoCD | v2.x (stable) | GitOps continuous delivery | De facto standard for K8s GitOps; Application CR is the core primitive |
| Kustomize | Built into kubectl | K8s manifest customization | Already used in Phase 11; ArgoCD has native Kustomize support |

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| ArgoCD | Flux CD | Both are CNCF graduated; ArgoCD has better UI and wider adoption for educational projects |
| Application CR | ApplicationSet | ApplicationSet is for multi-environment; overkill for single dev overlay (deferred to v2 K8S-ADV-02) |

## Architecture Patterns

### Recommended Project Structure
```
argocd/
  application.yaml       # ArgoCD Application CR (GIT-01, GIT-02)
deploy/
  k8s/
    base/                # Kustomize base (existing, Phase 11)
    overlays/
      dev/               # Dev overlay ArgoCD watches (existing, Phase 11)
      prod/              # Prod overlay (existing, Phase 11)
    kind-cluster.yaml    # Kind config (existing, Phase 12)
```

### Pattern: Separation of Application CR from Watched Path
**What:** The ArgoCD Application CR must NOT live inside the path it watches. If it did, ArgoCD would try to manage itself, creating a recursive sync loop.
**When to use:** Always when using declarative Application CRs in the same repo as the app manifests.
**Example:**
```yaml
# argocd/application.yaml
# This file is in argocd/ -- ArgoCD watches deploy/k8s/overlays/dev/
# These paths MUST NOT overlap
apiVersion: argoproj.io/v1alpha1
kind: Application
metadata:
  name: shitcoin
  namespace: argocd
  finalizers:
    - resources-finalizer.argocd.argoproj.io
spec:
  project: default
  source:
    repoURL: https://github.com/baotoq/shitcoin.git
    targetRevision: HEAD
    path: deploy/k8s/overlays/dev
  destination:
    server: https://kubernetes.default.svc
    namespace: default
  syncPolicy:
    automated:
      prune: true
      selfHeal: true
    syncOptions:
      - CreateNamespace=true
```
**Source:** [ArgoCD Application Specification](https://argo-cd.readthedocs.io/en/stable/user-guide/application-specification/)

### Anti-Patterns to Avoid
- **Application CR inside watched path:** Causes recursive sync. ArgoCD tries to manage its own Application resource, leading to loops or errors.
- **Missing finalizers:** Without `resources-finalizer.argocd.argoproj.io`, deleting the Application CR leaves orphaned K8s resources.
- **Hardcoded repoURL for local dev:** Use the actual GitHub repo URL even for local dev. ArgoCD clones from git, not from local filesystem. For local kind clusters, users must install ArgoCD and point it at the repo.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| GitOps sync | Custom scripts watching git | ArgoCD Application CR | Handles drift detection, self-healing, pruning, rollback |
| Multi-env promotion | Separate Application CRs per env | ApplicationSet (v2) | Deferred to v2 per requirements; single Application CR sufficient now |

## Common Pitfalls

### Pitfall 1: Application CR in Watched Path
**What goes wrong:** ArgoCD detects its own Application CR as a resource to manage, creating infinite sync loops or "out of sync" status.
**Why it happens:** Developers put all K8s manifests in one directory.
**How to avoid:** Place Application CR in `argocd/` directory, completely separate from `deploy/k8s/`.
**Warning signs:** ArgoCD shows the Application resource itself as "out of sync" or constantly syncing.

### Pitfall 2: Wrong namespace for Application CR
**What goes wrong:** Application CR is not recognized by ArgoCD.
**Why it happens:** Application CRs must be in the `argocd` namespace (where ArgoCD is installed).
**How to avoid:** Always set `metadata.namespace: argocd` in the Application CR.

### Pitfall 3: Missing prune policy
**What goes wrong:** Deleted manifests from git leave orphaned resources in the cluster.
**Why it happens:** `prune: false` is the default for automated sync.
**How to avoid:** Explicitly set `prune: true` in `syncPolicy.automated`.

### Pitfall 4: repoURL mismatch
**What goes wrong:** ArgoCD cannot clone the repository.
**Why it happens:** Using SSH URL when only HTTPS is configured, or wrong repo path.
**How to avoid:** Use HTTPS URL matching the GitHub repo. For private repos, configure repo credentials in ArgoCD.

## Code Examples

### Complete Application CR for This Project
```yaml
# Source: ArgoCD official docs + project-specific values
# File: argocd/application.yaml
apiVersion: argoproj.io/v1alpha1
kind: Application
metadata:
  name: shitcoin
  namespace: argocd
  finalizers:
    - resources-finalizer.argocd.argoproj.io
spec:
  project: default
  source:
    repoURL: https://github.com/baotoq/shitcoin.git
    targetRevision: HEAD
    path: deploy/k8s/overlays/dev
  destination:
    server: https://kubernetes.default.svc
    namespace: default
  syncPolicy:
    automated:
      prune: true
      selfHeal: true
    syncOptions:
      - CreateNamespace=true
```

### Key Fields Explained
| Field | Value | Purpose |
|-------|-------|---------|
| `metadata.namespace` | `argocd` | Application CRs must be in ArgoCD's namespace |
| `spec.project` | `default` | ArgoCD's built-in project; sufficient for single-app repos |
| `spec.source.path` | `deploy/k8s/overlays/dev` | Points to existing Kustomize dev overlay |
| `spec.source.targetRevision` | `HEAD` | Tracks latest commit on default branch |
| `syncPolicy.automated.prune` | `true` | Removes K8s resources when deleted from git |
| `syncPolicy.automated.selfHeal` | `true` | Reverts manual cluster changes to match git |
| `finalizers` | `resources-finalizer.argocd.argoproj.io` | Cleans up managed resources when Application is deleted |

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| `argoproj.io/v1alpha1` Application | Still `v1alpha1` (stable despite alpha label) | Unchanged since ArgoCD v1.0 | Use v1alpha1; no v1 exists yet |
| Manual sync | `syncPolicy.automated` | ArgoCD v1.1+ | Standard for GitOps; auto-sync on git push |

## Open Questions

1. **GitHub repo URL**
   - What we know: The project is at `github.com/baotoq/shitcoin` based on git remote
   - What's unclear: Whether the repo is public or private (affects ArgoCD repo configuration)
   - Recommendation: Use HTTPS URL; if private, user adds credentials to ArgoCD separately (out of scope for this phase)

## Validation Architecture

### Test Framework
| Property | Value |
|----------|-------|
| Framework | Manual validation (YAML lint + kubectl dry-run) |
| Config file | none |
| Quick run command | `kubectl apply --dry-run=client -f argocd/application.yaml` |
| Full suite command | `kubectl apply --dry-run=client -f argocd/application.yaml` |

### Phase Requirements to Test Map
| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| GIT-01 | Application CR with auto-sync pointing to dev overlay | smoke | `grep -q 'deploy/k8s/overlays/dev' argocd/application.yaml && grep -q 'automated' argocd/application.yaml` | N/A Wave 0 |
| GIT-02 | Application CR outside watched path | smoke | `test -f argocd/application.yaml && ! echo deploy/k8s | grep -q argocd` | N/A Wave 0 |

### Sampling Rate
- **Per task commit:** `kubectl apply --dry-run=client -f argocd/application.yaml`
- **Per wave merge:** Same (single file phase)
- **Phase gate:** Verify file exists, valid YAML, correct path and sync policy

### Wave 0 Gaps
None -- this phase produces a single YAML file with no test infrastructure required. Validation is structural (file exists, correct content).

## Sources

### Primary (HIGH confidence)
- [ArgoCD Application Specification](https://argo-cd.readthedocs.io/en/stable/user-guide/application-specification/) - Full CR spec reference
- [ArgoCD Kustomize Integration](https://argo-cd.readthedocs.io/en/stable/user-guide/kustomize/) - Native Kustomize support docs
- [ArgoCD Example Apps](https://github.com/argoproj/argocd-example-apps) - Official example repository

### Secondary (MEDIUM confidence)
- [ArgoCD Official Repo - application.yaml](https://github.com/argoproj/argo-cd/blob/master/docs/operator-manual/application.yaml) - Reference Application CR

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH - ArgoCD Application CR is well-documented and stable
- Architecture: HIGH - Separation pattern is universally recommended and documented
- Pitfalls: HIGH - Well-known issues with clear solutions

**Research date:** 2026-03-07
**Valid until:** 2026-06-07 (stable, ArgoCD Application CR API has not changed in years)
