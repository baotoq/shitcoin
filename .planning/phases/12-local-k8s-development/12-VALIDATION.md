---
phase: 12
slug: local-k8s-development
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-03-07
---

# Phase 12 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | Go testing + manual verification |
| **Config file** | N/A (infrastructure files, not application code) |
| **Quick run command** | `go test ./...` |
| **Full suite command** | `go test ./... && cd web && npm run lint && npm run build` |
| **Estimated runtime** | ~30 seconds |

---

## Sampling Rate

- **After every task commit:** Run `go test ./...`
- **After every plan wave:** Run `go test ./... && cd web && npm run lint && npm run build`
- **Before `/gsd:verify-work`:** Full suite must be green
- **Max feedback latency:** 30 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|-----------|-------------------|-------------|--------|
| 12-01-01 | 01 | 1 | DEV-03 | smoke | `kind create cluster --config deploy/k8s/kind-cluster.yaml` | ❌ W0 | ⬜ pending |
| 12-01-02 | 01 | 1 | DEV-04 | smoke | `make test && make lint` | ❌ W0 | ⬜ pending |
| 12-02-01 | 02 | 1 | DEV-01 | manual | `tilt ci` | ❌ W0 | ⬜ pending |
| 12-02-02 | 02 | 1 | DEV-02 | manual | `tilt ci` | ❌ W0 | ⬜ pending |
| 12-02-03 | 02 | 1 | DEV-04 | smoke | `make docker-build && make tilt-up` | ❌ W0 | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [ ] `build/` directory added to `.gitignore` — compiled binary output directory
- [ ] Verify existing tests pass before infrastructure changes

*Existing test infrastructure covers Go and frontend — no new test framework needed.*

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| `tilt up` builds and deploys both services | DEV-01, DEV-02 | Requires running kind cluster + Tilt | 1. `make kind-create` 2. `tilt up` 3. Verify both services appear in Tilt UI |
| Go source edit triggers rebuild | DEV-01 | Requires live running Tilt session | 1. `tilt up` 2. Edit a Go file 3. Observe Tilt rebuilds and restarts backend |
| React source edit triggers update | DEV-02 | Requires live running Tilt session | 1. `tilt up` 2. Edit a React file 3. Observe Tilt rebuilds and syncs frontend |

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 30s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
