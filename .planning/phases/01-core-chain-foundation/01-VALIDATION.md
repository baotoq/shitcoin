---
phase: 1
slug: core-chain-foundation
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-03-05
---

# Phase 1 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | Go testing (stdlib) + go test |
| **Config file** | None needed (Go testing is zero-config) |
| **Quick run command** | `go test ./internal/domain/... -v -count=1` |
| **Full suite command** | `go test -race -cover ./...` |
| **Estimated runtime** | ~5 seconds |

---

## Sampling Rate

- **After every task commit:** Run `go test ./internal/domain/... -v -count=1`
- **After every plan wave:** Run `go test -race -cover ./...`
- **Before `/gsd:verify-work`:** Full suite must be green
- **Max feedback latency:** 10 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|-----------|-------------------|-------------|--------|
| 01-01-01 | 01 | 1 | MINE-01 | unit | `go test ./internal/domain/block/... -run TestGenesisBlock -v` | No -- Wave 0 | pending |
| 01-01-02 | 01 | 1 | MINE-01 | integration | `go test ./internal/infrastructure/persistence/bbolt/... -run TestGenesisBlockPersistence -v` | No -- Wave 0 | pending |
| 01-01-03 | 01 | 1 | MINE-02 | unit | `go test ./internal/domain/block/... -run TestBlockStructure -v` | No -- Wave 0 | pending |
| 01-01-04 | 01 | 1 | MINE-03 | unit | `go test ./internal/domain/block/... -run TestDoubleSHA256 -v` | No -- Wave 0 | pending |
| 01-01-05 | 01 | 1 | MINE-03 | unit | `go test ./internal/domain/block/... -run TestDeterministicHashing -v` | No -- Wave 0 | pending |
| 01-01-06 | 01 | 1 | MINE-03 | unit | `go test ./internal/domain/block/... -run TestProofOfWork -v` | No -- Wave 0 | pending |
| 01-01-07 | 01 | 1 | MINE-06 | unit | `go test ./internal/domain/block/... -run TestDifficultyAdjustment -v` | No -- Wave 0 | pending |
| 01-01-08 | 01 | 1 | MINE-06 | unit | `go test ./internal/domain/block/... -run TestDifficultyAdjustmentClamping -v` | No -- Wave 0 | pending |
| 01-01-09 | 01 | 1 | MINE-09 | integration | `go test ./internal/config/... -run TestConfigLoading -v` | No -- Wave 0 | pending |
| 01-01-10 | 01 | 1 | MINE-09 | unit | `go test ./internal/config/... -run TestConfigDefaults -v` | No -- Wave 0 | pending |
| 01-01-11 | 01 | 1 | E2E-01 | integration | `go test ./internal/infrastructure/persistence/bbolt/... -run TestChainPersistence -v` | No -- Wave 0 | pending |

*Status: pending / green / red / flaky*

---

## Wave 0 Requirements

- [ ] `internal/domain/block/block_test.go` -- stubs for MINE-01, MINE-02, MINE-03
- [ ] `internal/domain/block/pow_test.go` -- stubs for MINE-03 (PoW validation)
- [ ] `internal/domain/block/difficulty_test.go` -- stubs for MINE-06
- [ ] `internal/config/config_test.go` -- stubs for MINE-09
- [ ] `internal/infrastructure/persistence/bbolt/chain_repo_test.go` -- stubs for persistence (MINE-01 storage, E2E-01)
- [ ] `go.mod` initialization -- `go mod init github.com/baotoq/shitcoin`

*No test framework install needed (Go testing is stdlib)*

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| Visual difficulty adjustment | MINE-06 | Requires observing logs over multiple blocks | Mine 10+ blocks, verify log output shows difficulty changes |

---

## Validation Sign-Off

- [ ] All tasks have automated verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 10s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
