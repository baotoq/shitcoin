---
phase: 16
slug: infrastructure-persistence-tests
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-03-08
---

# Phase 16 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | Go testing + testify v1.11.1 (suite + require/assert) |
| **Config file** | None (Go convention) |
| **Quick run command** | `go test -cover ./internal/infrastructure/persistence/...` |
| **Full suite command** | `go test -cover -count=2 ./internal/infrastructure/persistence/...` |
| **Estimated runtime** | ~20 seconds |

---

## Sampling Rate

- **After every task commit:** Run `go test -cover ./internal/infrastructure/persistence/...`
- **After every plan wave:** Run `go test -cover -count=2 ./internal/infrastructure/persistence/...`
- **Before `/gsd:verify-work`:** Full suite must be green, coverage thresholds met
- **Max feedback latency:** 20 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|-----------|-------------------|-------------|--------|
| 16-01-01 | 01 | 1 | INFR-01 | integration | `go test -cover ./internal/infrastructure/persistence/bbolt/ -run TestChainRepoSuite` | Partial | pending |
| 16-01-02 | 01 | 1 | INFR-01 | integration | `go test -cover ./internal/infrastructure/persistence/bbolt/ -run TestUTXORepoSuite` | Partial | pending |
| 16-02-01 | 02 | 1 | INFR-02 | unit | `go test -cover ./internal/infrastructure/persistence/jsonfile/` | Yes | pending |

*Status: pending / green / red / flaky*

---

## Wave 0 Requirements

*Existing infrastructure covers all phase requirements. No new framework or config needed.*

---

## Manual-Only Verifications

*All phase behaviors have automated verification.*

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 20s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
