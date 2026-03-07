---
phase: 11-kubernetes-manifests
plan: 02
subsystem: infra
tags: [kubernetes, kustomize, overlays, dev, prod]

requires:
  - phase: 11-kubernetes-manifests
    plan: 01
    provides: Kustomize base manifests with backend/frontend Deployments and Services
provides:
  - Dev overlay with local images (latest tag) and reduced resources
  - Prod overlay with GHCR images (pinned SHA) and production resources
affects: [12-tilt-dev-environment, 13-argocd-gitops]

tech-stack:
  added: []
  patterns: [Kustomize overlays, image transformer, strategic merge patches]

key-files:
  created:
    - deploy/k8s/overlays/dev/kustomization.yaml
    - deploy/k8s/overlays/prod/kustomization.yaml
  modified: []

key-decisions:
  - "Dev overlay uses local image names with :latest tag for local K8s development"
  - "Prod overlay uses placeholder SHA tags (sha-abc1234) for CI/CD to replace"
  - "Strategic merge patches via target+patch syntax for resource overrides"

patterns-established:
  - "Kustomize images transformer for environment-specific image references"
  - "Inline patch with target selector for per-deployment resource tuning"

requirements-completed: [K8S-05, K8S-06]

duration: 1min
completed: 2026-03-07
---

# Phase 11 Plan 02: Kustomize Overlays Summary

**Dev and prod Kustomize overlays with environment-specific image references and resource limits using images transformer and strategic merge patches**

## Performance

- **Duration:** 1 min
- **Started:** 2026-03-07T16:23:41Z
- **Completed:** 2026-03-07T16:24:42Z
- **Tasks:** 2
- **Files modified:** 2

## Accomplishments
- Dev overlay with shitcoin-backend:latest and shitcoin-frontend:latest for local development
- Dev resource limits reduced below base (backend cpu=50m/200m mem=64Mi/128Mi, frontend cpu=25m/100m mem=32Mi/64Mi)
- Prod overlay with ghcr.io/baotoq/shitcoin:sha-abc1234 and ghcr.io/baotoq/shitcoin-web:sha-abc1234
- Prod resource limits set for production (backend cpu=200m/1 mem=256Mi/512Mi, frontend cpu=50m/200m mem=64Mi/128Mi)
- Both overlays render all 6 base resources via kubectl kustomize without errors

## Task Commits

Each task was committed atomically:

1. **Task 1: Create dev overlay with local images and low resources** - `9205a86` (feat)
2. **Task 2: Create prod overlay with GHCR images and production resources** - `af341b2` (feat)

## Files Created/Modified
- `deploy/k8s/overlays/dev/kustomization.yaml` - Dev overlay with local images, :latest tags, reduced resources
- `deploy/k8s/overlays/prod/kustomization.yaml` - Prod overlay with GHCR images, pinned SHA tags, production resources

## Decisions Made
- Dev overlay uses local image names with :latest tag for local K8s development (no registry push needed)
- Prod overlay uses placeholder SHA tags (sha-abc1234) intended for CI/CD pipelines to substitute
- Used strategic merge patches with target+patch inline syntax for clean resource overrides

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered
- `kubectl apply --dry-run=client` requires a running K8s cluster connection; validated via `kubectl kustomize` instead (renders valid YAML)

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- Both overlays ready for Tilt dev environment (Phase 12) to use dev overlay
- Prod overlay ready for ArgoCD GitOps (Phase 13) to deploy
- SHA tags in prod overlay designed for CI/CD to update on each build

---
*Phase: 11-kubernetes-manifests*
*Completed: 2026-03-07*
