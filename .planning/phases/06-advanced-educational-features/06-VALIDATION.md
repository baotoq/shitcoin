---
phase: 06
slug: advanced-educational-features
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-03-07
---

# Phase 06 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | go test |
| **Config file** | none — existing infrastructure |
| **Quick run command** | `go test ./internal/domain/... -count=1` |
| **Full suite command** | `go test ./... -count=1` |
| **Estimated runtime** | ~15 seconds |

---

## Sampling Rate

- **After every task commit:** Run `go test ./internal/domain/... -count=1`
- **After every plan wave:** Run `go test ./... -count=1`
- **Before `/gsd:verify-work`:** Full suite must be green
- **Max feedback latency:** 15 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|-----------|-------------------|-------------|--------|
| 06-01-T1 | 01 | 1 | MINE-08 | unit | `go test ./internal/domain/chain/ -run TestHalving -v -count=1` | ❌ W0 | ⬜ pending |
| 06-01-T2 | 01 | 1 | TX-09, TX-10 | unit | `go test ./internal/domain/tx/ ./internal/domain/mempool/ -run "TestFee\|TestDrainByFee" -v -count=1` | ❌ W0 | ⬜ pending |
| 06-02-T1 | 02 | 2 | ORCH-01 | integration | `go test ./internal/handler/cli/ -run TestTestnet -v -count=1 -timeout=60s` | ❌ W0 | ⬜ pending |
| 06-02-T2 | 02 | 2 | DEMO-01 | integration | `go test ./internal/handler/cli/ -run TestDoubleSpend -v -count=1 -timeout=60s` | ❌ W0 | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [ ] Test stubs for halving, fee sorting, testnet orchestration, double-spend demo
- [ ] Existing test infrastructure covers framework needs

*TDD tasks in plans create tests inline — Wave 0 stubs may not be needed.*

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| Multi-node testnet launches and nodes connect | ORCH-01 | Requires multiple OS processes | Run `go run cmd/shitcoin/main.go -f etc/shitcoin.yaml testnet` and verify 3 nodes connect |
| Double-spend visually rejected | DEMO-01 | Requires observing node logs | Run double-spend demo and verify rejection messages |

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 15s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
