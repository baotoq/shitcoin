---
phase: 02-wallets-and-transactions
plan: 02
subsystem: transactions
tags: [utxo, ecdsa, btcec, secp256k1, coinbase, signing, transaction-validation]

requires:
  - phase: 01-core-chain-foundation
    provides: "block.Hash, block.DoubleSHA256 for transaction ID computation"
provides:
  - "Transaction entity with deterministic ID via JSON + DoubleSHA256"
  - "TxInput/TxOutput value objects for UTXO model"
  - "Coinbase transaction factory for block rewards"
  - "ECDSA signing/verification using btcec/v2"
  - "Structural validator with sum invariant enforcement"
  - "Automatic change output generation"
affects: [utxo-set, block-integration, mempool, mining]

tech-stack:
  added: [btcec/v2, btcec/v2/ecdsa, dcrd/dcrec/secp256k1/v4]
  patterns: [hashable-struct-for-id, signature-excluded-from-hash, coinbase-marker-pattern]

key-files:
  created:
    - internal/domain/tx/transaction.go
    - internal/domain/tx/input.go
    - internal/domain/tx/output.go
    - internal/domain/tx/coinbase.go
    - internal/domain/tx/signing.go
    - internal/domain/tx/validator.go
    - internal/domain/tx/errors.go
    - internal/domain/tx/transaction_test.go
  modified: []

key-decisions:
  - "Hashable struct pattern for TX ID: JSON-serialize inputs (without sig/pubkey) and outputs, then DoubleSHA256"
  - "Coinbase marker: zero hash + 0xFFFFFFFF vout, consistent with Bitcoin convention"
  - "Sign full transaction hash (simplified SIGHASH_ALL) rather than per-input signing"

patterns-established:
  - "hashableTransaction/hashableInput/hashableOutput structs for deterministic ID computation excluding signatures"
  - "Coinbase detection via IsZero() txID + max uint32 vout"
  - "Validator split: structural validation in tx package, contextual validation (UTXO lookups) deferred to chain layer"

requirements-completed: [TX-03, TX-04, TX-05, TX-06]

duration: 5min
completed: 2026-03-05
---

# Phase 2 Plan 02: Transaction Domain Summary

**UTXO transaction model with ECDSA signing via btcec/v2, coinbase factory, structural validation, and automatic change output generation**

## Performance

- **Duration:** 5 min
- **Started:** 2026-03-05T12:57:55Z
- **Completed:** 2026-03-05T13:03:00Z
- **Tasks:** 1 (TDD: RED-GREEN)
- **Files modified:** 8

## Accomplishments
- Transaction entity with deterministic ID computation via JSON + DoubleSHA256 (signatures excluded)
- TxInput/TxOutput value objects following project's unexported-fields-with-getters pattern
- Coinbase factory creating valid block reward transactions with zero-hash marker
- ECDSA sign/verify using btcec/v2 with secp256k1 curve (Bitcoin's actual crypto library)
- Structural validator enforcing sum invariant, positive output values, and input/output presence
- CreateTransactionWithChange for automatic change output when input exceeds payment
- 27 comprehensive tests all passing with -race flag

## Task Commits

Each task was committed atomically:

1. **Task 1 (RED): Transaction types, coinbase, signing tests** - `ade127b` (test)
2. **Task 1 (GREEN): Full implementation passing all tests** - `927616f` (feat)

## Files Created/Modified
- `internal/domain/tx/transaction.go` - Transaction entity with ComputeID, IsCoinbase, hashPayload
- `internal/domain/tx/input.go` - TxInput value object referencing UTXO by txid:vout
- `internal/domain/tx/output.go` - TxOutput value object with satoshi value and address
- `internal/domain/tx/coinbase.go` - NewCoinbaseTx factory for block reward transactions
- `internal/domain/tx/signing.go` - SignTransaction and VerifyTransaction using btcec/v2/ecdsa
- `internal/domain/tx/validator.go` - ValidateTransaction, ValidateCoinbase, CreateTransactionWithChange
- `internal/domain/tx/errors.go` - Sentinel errors for transaction domain
- `internal/domain/tx/transaction_test.go` - 27 tests covering all behaviors

## Decisions Made
- Hashable struct pattern for TX ID: JSON-serialize inputs (without sig/pubkey) and outputs, then DoubleSHA256 -- consistent with Phase 1's hashableHeader approach
- Coinbase marker uses zero hash + 0xFFFFFFFF vout, matching Bitcoin convention
- Simplified SIGHASH_ALL: sign full transaction hash rather than per-input signing -- appropriate for educational project
- int64 satoshis throughout (1 coin = 100,000,000 satoshis) per user constraint

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered
None

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- Transaction domain complete, ready for UTXO set and undo-log (plan 02-03)
- Block integration (typed transactions replacing [][]byte) ready for plan 02-04
- Signing infrastructure available for wallet-to-transaction flow

## Self-Check: PASSED

All 8 created files verified. Both commits (ade127b, 927616f) confirmed in git log.

---
*Phase: 02-wallets-and-transactions*
*Completed: 2026-03-05*
