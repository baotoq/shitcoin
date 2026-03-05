---
phase: 2
slug: wallets-and-transactions
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-03-05
---

# Phase 2 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | Go testing (stdlib) |
| **Config file** | none (Go convention) |
| **Quick run command** | `go test ./internal/domain/wallet/... ./internal/domain/tx/... ./internal/domain/utxo/... -v -count=1` |
| **Full suite command** | `go test ./... -v -race -count=1` |
| **Estimated runtime** | ~10 seconds |

---

## Sampling Rate

- **After every task commit:** Run `go test ./internal/domain/wallet/... ./internal/domain/tx/... ./internal/domain/utxo/... -v -count=1`
- **After every plan wave:** Run `go test ./... -v -race -count=1`
- **Before `/gsd:verify-work`:** Full suite must be green
- **Max feedback latency:** 10 seconds

---

## Per-Task Verification Map

| Req ID | Requirement | Test Type | Automated Command | File Exists | Status |
|--------|-------------|-----------|-------------------|-------------|--------|
| TX-01 | Generate ECDSA wallet, store/load key pairs | unit | `go test ./internal/domain/wallet/... -run TestGenerateWallet -v` | ❌ W0 | ⬜ pending |
| TX-02 | PubKey -> SHA-256 -> RIPEMD-160 -> Base58Check address | unit | `go test ./internal/domain/wallet/... -run TestAddressDerivation -v` | ❌ W0 | ⬜ pending |
| TX-03 | Create UTXO transaction with inputs/outputs | unit | `go test ./internal/domain/tx/... -run TestCreateTransaction -v` | ❌ W0 | ⬜ pending |
| TX-04 | Sign input and verify ECDSA signature | unit | `go test ./internal/domain/tx/... -run TestSignVerify -v` | ❌ W0 | ⬜ pending |
| TX-05 | Auto-create change output, sum invariant | unit | `go test ./internal/domain/tx/... -run TestChangeOutput -v` | ❌ W0 | ⬜ pending |
| TX-06 | Coinbase transaction with block reward | unit | `go test ./internal/domain/tx/... -run TestCoinbaseTx -v` | ❌ W0 | ⬜ pending |
| TX-07 | UTXO set add/remove/query persistence | integration | `go test ./internal/infrastructure/persistence/bbolt/... -run TestUTXORepo -v` | ❌ W0 | ⬜ pending |
| TX-08 | Undo-log write/read, apply/revert block UTXOs | integration | `go test ./internal/domain/utxo/... -run TestUndoLog -v` | ❌ W0 | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [ ] `internal/domain/wallet/wallet_test.go` — covers TX-01, TX-02
- [ ] `internal/domain/wallet/base58_test.go` — covers Base58Check encode/decode with known Bitcoin test vectors
- [ ] `internal/domain/tx/transaction_test.go` — covers TX-03, TX-04, TX-05, TX-06
- [ ] `internal/domain/utxo/set_test.go` — covers TX-07, TX-08 (domain logic)
- [ ] `internal/infrastructure/persistence/bbolt/utxo_repo_test.go` — covers TX-07 (persistence)
- [ ] `internal/infrastructure/persistence/jsonfile/wallet_repo_test.go` — covers wallet persistence

*All test files must be created as part of Wave 0 or early plan tasks.*

---

## Manual-Only Verifications

*All phase behaviors have automated verification.*

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 10s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
