---
phase: 18
slug: integration-ci-quality
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-03-08
---

# Phase 18 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | Go testing + testify v1.11.1 |
| **Config file** | None needed — Go test conventions |
| **Quick run command** | `go test -v -run "Integration\|E2E" ./internal/integration/ -timeout 60s` |
| **Full suite command** | `go test -race ./...` |
| **Estimated runtime** | ~30 seconds |

---

## Sampling Rate

- **After every task commit:** Run `go test -v -run "Integration\|E2E" ./internal/integration/ -timeout 60s`
- **After every plan wave:** Run `go test -race ./...`
- **Before `/gsd:verify-work`:** Full suite must be green
- **Max feedback latency:** 30 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|-----------|-------------------|-------------|--------|
| 18-01-01 | 01 | 1 | INTG-01 | integration | `go test -v -run TestIntegration ./internal/integration/ -timeout 60s` | ❌ W0 | ⬜ pending |
| 18-01-02 | 01 | 1 | INTG-02 | integration | `go test -v -run TestE2E ./internal/integration/ -timeout 60s` | ❌ W0 | ⬜ pending |
| 18-02-01 | 02 | 1 | TINF-03 | race | `go test -race ./...` | ❌ W0 | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [ ] `internal/integration/integration_test.go` — stubs for INTG-01 (P2P multi-node tests)
- [ ] `internal/integration/e2e_chain_test.go` — stubs for INTG-02 (E2E chain scenario tests)
- [ ] Fix ws.Hub race in `internal/handler/ws/hub.go` — prerequisite for TINF-03
- [ ] Update `.github/workflows/ci-go.yml` — add `-race` flag (TINF-03)

---

## Manual-Only Verifications

*All phase behaviors have automated verification.*

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 30s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
