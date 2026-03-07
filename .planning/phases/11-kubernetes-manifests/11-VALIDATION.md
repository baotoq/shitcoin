---
phase: 11
slug: kubernetes-manifests
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-03-07
---

# Phase 11 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | kubectl + kustomize (built-in) |
| **Config file** | N/A — manifests are the config |
| **Quick run command** | `kubectl kustomize deploy/k8s/overlays/dev` |
| **Full suite command** | `kubectl apply -k deploy/k8s/overlays/dev --dry-run=client && kubectl apply -k deploy/k8s/overlays/prod --dry-run=client` |
| **Estimated runtime** | ~5 seconds |

---

## Sampling Rate

- **After every task commit:** Run `kubectl kustomize deploy/k8s/overlays/dev`
- **After every plan wave:** Run `kubectl apply -k deploy/k8s/overlays/dev --dry-run=client && kubectl apply -k deploy/k8s/overlays/prod --dry-run=client`
- **Before `/gsd:verify-work`:** Both overlays render valid YAML with all expected resources
- **Max feedback latency:** 5 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|-----------|-------------------|-------------|--------|
| 11-01-01 | 01 | 1 | K8S-01, K8S-02, K8S-03, K8S-04, K8S-07 | smoke | `kubectl kustomize deploy/k8s/base` | N/A W0 | pending |
| 11-02-01 | 02 | 1 | K8S-05 | smoke | `kubectl kustomize deploy/k8s/overlays/dev` | N/A W0 | pending |
| 11-02-02 | 02 | 1 | K8S-06 | smoke | `kubectl kustomize deploy/k8s/overlays/prod` | N/A W0 | pending |

*Status: pending / green / red / flaky*

---

## Wave 0 Requirements

None — Kustomize manifests are validated by `kubectl kustomize` rendering (no separate test framework needed).

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| Pods actually start and run | K8S-01 | Requires running K8s cluster | `kubectl apply -k deploy/k8s/overlays/dev && kubectl get pods` |
| PVC binds and data persists | K8S-02 | Requires running K8s cluster | Deploy, create data, delete pod, verify data persists |
| Health probes report healthy | K8S-07 | Requires running backend | `kubectl get pods` shows Ready, `kubectl describe pod` shows probe passing |

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 5s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
