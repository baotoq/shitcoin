---
phase: 15
slug: domain-layer-coverage
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-03-08
---

# Phase 15 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | Go testing + testify v1.11.1 |
| **Config file** | None (Go convention) |
| **Quick run command** | `go test -cover ./internal/domain/...` |
| **Full suite command** | `go test -cover ./...` |
| **Estimated runtime** | ~45 seconds |

---

## Sampling Rate

- **After every task commit:** Run `go test -cover ./internal/domain/...`
- **After every plan wave:** Run `go test -cover ./...`
- **Before `/gsd:verify-work`:** Full suite must be green, all coverage thresholds met
- **Max feedback latency:** 45 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|-----------|-------------------|-------------|--------|
| 15-01-01 | 01 | 1 | DOM-03 | unit | `go test -cover ./internal/domain/tx/` | Yes | pending |
| 15-01-02 | 01 | 1 | DOM-03 | unit | `go test -cover ./internal/domain/utxo/` | Yes | pending |
| 15-01-03 | 01 | 1 | DOM-03 | unit | `go test -cover ./internal/domain/wallet/` | Yes | pending |
| 15-01-04 | 01 | 1 | DOM-03 | unit | `go test -cover ./internal/domain/mempool/` | Yes | pending |
| 15-02-01 | 02 | 1 | DOM-01 | unit | `go test -cover ./internal/domain/chain/` | Yes | pending |
| 15-03-01 | 03 | 2 | DOM-02 | unit | `go test -cover ./internal/domain/p2p/` | Partial | pending |
| 15-XX-XX | all | all | DOM-04 | unit | `go test -v ./internal/domain/... -run "Error\|Invalid\|Reject\|Fail\|Nil\|Corrupt\|Boundary"` | Wave 0 | pending |

*Status: pending / green / red / flaky*

---

## Wave 0 Requirements

- [ ] `internal/domain/p2p/handler_test.go` — covers DOM-02 handler dispatch tests
- [ ] `internal/domain/p2p/payload_test.go` — covers DOM-02 payload error paths

*Other packages have existing test files that will be extended.*

---

## Manual-Only Verifications

*All phase behaviors have automated verification.*

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 45s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
