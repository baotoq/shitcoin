---
phase: 17
slug: handler-layer-tests
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-03-08
---

# Phase 17 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | Go testing + testify v1.11.1 |
| **Config file** | None (Go convention) |
| **Quick run command** | `go test ./internal/handler/api/ ./internal/handler/ws/` |
| **Full suite command** | `go test -cover ./internal/handler/api/ ./internal/handler/ws/` |
| **Estimated runtime** | ~15 seconds |

---

## Sampling Rate

- **After every task commit:** Run `go test ./internal/handler/api/ ./internal/handler/ws/`
- **After every plan wave:** Run `go test -cover ./internal/handler/api/ ./internal/handler/ws/`
- **Before `/gsd:verify-work`:** Full suite must be green, API 80%+, WS 75%+
- **Max feedback latency:** 15 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|-----------|-------------------|-------------|--------|
| 17-01-01 | 01 | 1 | HNDL-01 | unit | `go test -cover ./internal/handler/api/ -run "TestAddress\|TestBlockByHash\|TestSearch"` | No — W0 | pending |
| 17-01-02 | 01 | 1 | HNDL-01 | unit | `go test -cover ./internal/handler/api/ -run "TestBlocks\|TestMempool\|TestTx"` | Partial | pending |
| 17-02-01 | 02 | 1 | HNDL-02 | integration | `go test -cover ./internal/handler/ws/ -run "TestServeWs\|TestHub"` | Partial | pending |

*Status: pending / green / red / flaky*

---

## Wave 0 Requirements

- [ ] `internal/handler/api/address_handler_test.go` — covers HNDL-01 (address handler)
- [ ] `internal/handler/api/search_handler_test.go` — covers HNDL-01 (search handler)
- [ ] `internal/handler/ws/handler_test.go` — covers HNDL-02 (ServeWs lifecycle)

---

## Manual-Only Verifications

*All phase behaviors have automated verification.*

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 15s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
