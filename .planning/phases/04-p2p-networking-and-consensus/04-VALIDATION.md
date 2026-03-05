---
phase: 4
slug: p2p-networking-and-consensus
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-03-05
---

# Phase 4 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | Go testing (stdlib) |
| **Config file** | None — Go convention `*_test.go` in package dirs |
| **Quick run command** | `go test ./internal/domain/p2p/... -race -v -count=1` |
| **Full suite command** | `go test ./... -race -count=1` |
| **Estimated runtime** | ~15 seconds |

---

## Sampling Rate

- **After every task commit:** Run `go test ./internal/domain/p2p/... -race -v -count=1`
- **After every plan wave:** Run `go test ./... -race -count=1`
- **Before `/gsd:verify-work`:** Full suite must be green
- **Max feedback latency:** 15 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|-----------|-------------------|-------------|--------|
| 04-01-01 | 01 | 1 | NET-01 | integration | `go test ./internal/domain/p2p/ -run TestServerListen -v` | ❌ W0 | ⬜ pending |
| 04-01-02 | 01 | 1 | NET-02 | integration | `go test ./internal/domain/p2p/ -run TestHandshake -v` | ❌ W0 | ⬜ pending |
| 04-02-01 | 02 | 1 | NET-04 | integration | `go test ./internal/domain/p2p/ -run TestTxBroadcast -v` | ❌ W0 | ⬜ pending |
| 04-02-02 | 02 | 1 | NET-05 | integration | `go test ./internal/domain/p2p/ -run TestBlockBroadcast -v` | ❌ W0 | ⬜ pending |
| 04-02-03 | 02 | 1 | NET-06 | unit | `go test ./internal/domain/p2p/ -run TestValidation -v` | ❌ W0 | ⬜ pending |
| 04-03-01 | 03 | 2 | NET-07 | integration | `go test ./internal/domain/p2p/ -run TestInitialBlockDownload -v` | ❌ W0 | ⬜ pending |
| 04-04-01 | 04 | 2 | NET-08 | integration | `go test ./internal/domain/p2p/ -run TestLongerChainReorg -v` | ❌ W0 | ⬜ pending |
| 04-04-02 | 04 | 2 | NET-09 | integration | `go test ./internal/domain/p2p/ -run TestReorgUTXO -v` | ❌ W0 | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [ ] `internal/domain/p2p/` — entire package is new, test files need creation
- [ ] Test helpers for creating in-memory peer pairs (`net.Pipe` or localhost listener)
- [ ] Test helpers for creating populated chains (reuse from existing bbolt tests or create shared fixture)

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| Multi-node demo on localhost | All | Full E2E with multiple processes | Start 3 nodes on ports 3000-3002, mine blocks, verify sync |

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 15s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
