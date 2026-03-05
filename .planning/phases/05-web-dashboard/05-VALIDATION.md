---
phase: 05
slug: web-dashboard
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-03-06
---

# Phase 05 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | go test + vitest (React frontend) |
| **Config file** | go test: none; vitest: web/vite.config.ts |
| **Quick run command** | `go test ./internal/handler/... ./internal/domain/...` |
| **Full suite command** | `go test ./... && cd web && npm test` |
| **Estimated runtime** | ~10 seconds (Go) + ~5 seconds (frontend) |

---

## Sampling Rate

- **After every task commit:** Run `go test ./internal/handler/... ./internal/domain/...`
- **After every plan wave:** Run `go test ./... && cd web && npm test`
- **Before `/gsd:verify-work`:** Full suite must be green
- **Max feedback latency:** 15 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|-----------|-------------------|-------------|--------|
| 05-01-01 | 01 | 1 | DASH-01 | unit/integration | `go test ./internal/handler/...` | W0 | pending |
| 05-01-02 | 01 | 1 | DASH-02 | unit | `go test ./internal/handler/...` | W0 | pending |
| 05-01-03 | 01 | 1 | DASH-04 | unit | `go test ./internal/handler/...` | W0 | pending |
| 05-01-04 | 01 | 1 | DASH-05 | unit | `go test ./internal/handler/...` | W0 | pending |
| 05-02-01 | 02 | 2 | DASH-03 | unit | `go test ./internal/domain/block/...` | W0 | pending |
| 05-02-02 | 02 | 2 | DASH-02 | integration | `go test ./internal/handler/...` | W0 | pending |

*Status: pending / green / red / flaky*

---

## Wave 0 Requirements

*Existing Go test infrastructure covers backend. Frontend test infrastructure (vitest) installed as part of React scaffold in Wave 1.*

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| Block explorer UI renders correctly | DASH-01 | Visual layout | Open browser, navigate blocks, verify block/tx details display |
| Mining visualization shows live nonce/hash updates | DASH-03 | Real-time visual | Start mining, open dashboard, verify sampled updates stream |
| WebSocket reconnects after disconnect | DASH-02 | Network behavior | Kill node, restart, verify dashboard reconnects |

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 15s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
