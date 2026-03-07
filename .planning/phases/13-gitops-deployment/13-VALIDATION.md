---
phase: 13
slug: gitops-deployment
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-03-07
---

# Phase 13 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | Manual validation (YAML lint + kubectl dry-run) |
| **Config file** | none |
| **Quick run command** | `kubectl apply --dry-run=client -f argocd/application.yaml` |
| **Full suite command** | `kubectl apply --dry-run=client -f argocd/application.yaml` |
| **Estimated runtime** | ~2 seconds |

---

## Sampling Rate

- **After every task commit:** Run `kubectl apply --dry-run=client -f argocd/application.yaml`
- **After every plan wave:** Same (single file phase)
- **Before `/gsd:verify-work`:** Full suite must be green
- **Max feedback latency:** 2 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|-----------|-------------------|-------------|--------|
| 13-01-01 | 01 | 1 | GIT-01, GIT-02 | smoke | `test -f argocd/application.yaml && grep -q 'deploy/k8s/overlays/dev' argocd/application.yaml && grep -q 'automated' argocd/application.yaml` | ❌ W0 | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

None — this phase produces a single YAML file with no test infrastructure required. Validation is structural (file exists, correct content).

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| ArgoCD auto-syncs from git | GIT-01 | Requires running ArgoCD instance | 1. Install ArgoCD in kind cluster 2. `kubectl apply -f argocd/application.yaml` 3. Verify sync status in ArgoCD UI |

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 2s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
