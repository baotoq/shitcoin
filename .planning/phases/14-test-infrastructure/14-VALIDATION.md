---
phase: 14
slug: test-infrastructure
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-03-08
---

# Phase 14 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | Go testing + testify v1.11.1 |
| **Config file** | None (Go convention) |
| **Quick run command** | `go test ./internal/testutil/...` |
| **Full suite command** | `go test ./...` |
| **Estimated runtime** | ~30 seconds |

---

## Sampling Rate

- **After every task commit:** Run `go test ./internal/testutil/... && go test ./...`
- **After every plan wave:** Run `go test ./...`
- **Before `/gsd:verify-work`:** Full suite must be green
- **Max feedback latency:** 30 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|-----------|-------------------|-------------|--------|
| 14-01-01 | 01 | 1 | TINF-01 | unit | `go test ./internal/testutil/ -run TestBuilders` | No — W0 | pending |
| 14-01-02 | 01 | 1 | TINF-02 | unit | `go test ./internal/testutil/ -run TestMockChainRepo` | No — W0 | pending |
| 14-01-03 | 01 | 1 | TINF-02 | unit | `go test ./internal/testutil/ -run TestMockUTXORepo` | No — W0 | pending |
| 14-01-04 | 01 | 1 | TINF-02 | unit | `go test ./internal/testutil/ -run TestMockWalletRepo` | No — W0 | pending |
| 14-02-01 | 02 | 2 | TINF-02 | integration | `go test ./...` | Yes | pending |

*Status: pending / green / red / flaky*

---

## Wave 0 Requirements

- [ ] `internal/testutil/builders_test.go` — covers TINF-01 (verify builders produce valid objects)
- [ ] `internal/testutil/mock_chain_repo_test.go` — covers TINF-02 (verify interface compliance)
- [ ] `internal/testutil/mock_utxo_repo_test.go` — covers TINF-02
- [ ] `internal/testutil/mock_wallet_repo_test.go` — covers TINF-02

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
