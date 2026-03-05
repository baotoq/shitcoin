---
phase: 3
slug: mempool-mining-integration-and-cli
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-03-05
---

# Phase 3 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | go test |
| **Config file** | none — standard Go testing |
| **Quick run command** | `go test ./...` |
| **Full suite command** | `go test -v -count=1 ./...` |
| **Estimated runtime** | ~10 seconds |

---

## Sampling Rate

- **After every task commit:** Run `go test ./...`
- **After every plan wave:** Run `go test -v -count=1 ./...`
- **Before `/gsd:verify-work`:** Full suite must be green
- **Max feedback latency:** 10 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|-----------|-------------------|-------------|--------|
| 03-01-01 | 01 | 1 | MINE-04 | unit | `go test ./internal/domain/mempool/...` | ❌ W0 | ⬜ pending |
| 03-01-02 | 01 | 1 | MINE-05 | unit | `go test ./internal/domain/mempool/...` | ❌ W0 | ⬜ pending |
| 03-01-03 | 01 | 1 | MINE-07 | unit | `go test ./internal/domain/block/...` | ❌ W0 | ⬜ pending |
| 03-02-01 | 02 | 2 | CLI-01 | integration | `go test ./cmd/...` | ❌ W0 | ⬜ pending |
| 03-02-02 | 02 | 2 | CLI-02 | integration | `go test ./cmd/...` | ❌ W0 | ⬜ pending |
| 03-02-03 | 02 | 2 | CLI-03 | integration | `go test ./cmd/...` | ❌ W0 | ⬜ pending |
| 03-02-04 | 02 | 2 | CLI-04 | integration | `go test ./cmd/...` | ❌ W0 | ⬜ pending |
| 03-02-05 | 02 | 2 | CLI-05 | integration | `go test ./cmd/...` | ❌ W0 | ⬜ pending |
| 03-02-06 | 02 | 2 | CLI-06 | integration | `go test ./cmd/...` | ❌ W0 | ⬜ pending |
| 03-02-07 | 02 | 2 | CLI-07 | integration | `go test ./cmd/...` | ❌ W0 | ⬜ pending |
| 03-02-08 | 02 | 2 | NET-03 | integration | `go test ./cmd/...` | ❌ W0 | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [ ] `internal/domain/mempool/mempool_test.go` — unit tests for mempool add/reject/drain
- [ ] `internal/domain/block/merkle_test.go` — unit tests for Merkle root computation
- [ ] Test fixtures for sample transactions and blocks

*Existing infrastructure covers wallet, chain, utxo, and block domain tests.*

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| CLI end-to-end flow | CLI-01..07 | Full binary interaction | Run binary with each subcommand, verify output |
| Auto-mine background loop | MINE-04 | Requires timing/cancellation | Start mine loop, wait for blocks, cancel, verify clean stop |

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 10s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
