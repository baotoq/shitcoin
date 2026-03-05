---
phase: 02-wallets-and-transactions
plan: 01
subsystem: wallet
tags: [btcec, secp256k1, ecdsa, base58check, p2pkh, json-persistence]

# Dependency graph
requires:
  - phase: 01-core-chain-foundation
    provides: "block.Hash, block.DoubleSHA256 (double-SHA256 hashing pattern)"
provides:
  - "Wallet entity with ECDSA key generation (btcec/v2 secp256k1)"
  - "Base58Check encode/decode (hand-rolled, Bitcoin-compatible)"
  - "P2PKH address derivation (SHA-256 -> RIPEMD-160 -> Base58Check)"
  - "wallet.Repository interface and JSON file implementation"
  - "PubKeyHashFromAddress for tx validation"
affects: [02-wallets-and-transactions, 03-mempool-and-validation]

# Tech tracking
tech-stack:
  added: [btcec/v2, golang.org/x/crypto/ripemd160]
  patterns: [hand-rolled-base58check, p2pkh-address-derivation, json-file-persistence, atomic-file-write]

key-files:
  created:
    - internal/domain/wallet/base58.go
    - internal/domain/wallet/address.go
    - internal/domain/wallet/wallet.go
    - internal/domain/wallet/repository.go
    - internal/domain/wallet/errors.go
    - internal/domain/wallet/base58_test.go
    - internal/domain/wallet/wallet_test.go
    - internal/infrastructure/persistence/jsonfile/wallet_repo.go
    - internal/infrastructure/persistence/jsonfile/wallet_repo_test.go
  modified:
    - go.mod
    - go.sum

key-decisions:
  - "btcec/v2 for secp256k1 ECDSA key generation (per user constraint)"
  - "Hand-rolled Base58Check for educational value (per user constraint)"
  - "Atomic JSON file writes via temp file + rename for crash safety"

patterns-established:
  - "Base58Check: version byte + payload + 4-byte double-SHA256 checksum"
  - "P2PKH address pipeline: compressed pubkey -> SHA-256 -> RIPEMD-160 -> Base58Check(0x00)"
  - "JSON file persistence: walletFileModel with walletEntry slice, loaded into memory map on startup"
  - "ReconstructWallet pattern: rebuild wallet from hex private key (bypass generation)"

requirements-completed: [TX-01, TX-02]

# Metrics
duration: 5min
completed: 2026-03-05
---

# Phase 2 Plan 01: Wallet Domain Summary

**ECDSA wallet with secp256k1 key pairs, hand-rolled Base58Check encoding, and Bitcoin-style P2PKH address derivation with JSON file persistence**

## Performance

- **Duration:** 5 min
- **Started:** 2026-03-05T12:58:00Z
- **Completed:** 2026-03-05T13:03:00Z
- **Tasks:** 2
- **Files modified:** 11

## Accomplishments
- Hand-rolled Base58Check encode/decode matching Bitcoin test vectors (leading zero byte preservation)
- P2PKH address derivation producing addresses starting with '1' (version byte 0x00)
- Wallet entity with btcec/v2 secp256k1 ECDSA key generation and hex serialization
- JSON file wallet repository with atomic writes and persistence across close/reopen

## Task Commits

Each task was committed atomically (TDD: RED + GREEN):

1. **Task 1: Wallet domain -- Base58Check, address derivation, and wallet entity**
   - `46cf092` (test: failing tests for wallet domain)
   - `42e63e8` (feat: implement wallet domain)
2. **Task 2: JSON file wallet repository**
   - `b461109` (test: failing tests for JSON file wallet repository)
   - `ad358c7` (feat: implement JSON file wallet repository)

## Files Created/Modified
- `internal/domain/wallet/base58.go` - Hand-rolled Base58 encode/decode with math/big, Base58Check with double-SHA256 checksum
- `internal/domain/wallet/address.go` - PubKeyToAddress (P2PKH pipeline) and PubKeyHashFromAddress (for tx validation)
- `internal/domain/wallet/wallet.go` - Wallet entity with NewWallet (key generation) and ReconstructWallet (from hex)
- `internal/domain/wallet/repository.go` - Repository interface: Save, GetByAddress, ListAddresses
- `internal/domain/wallet/errors.go` - ErrWalletNotFound, ErrInvalidAddress, ErrInvalidChecksum
- `internal/domain/wallet/base58_test.go` - Base58 encode/decode tests with Bitcoin test vectors
- `internal/domain/wallet/wallet_test.go` - Wallet generation, reconstruction, address derivation tests
- `internal/infrastructure/persistence/jsonfile/wallet_repo.go` - JSON file wallet.Repository implementation
- `internal/infrastructure/persistence/jsonfile/wallet_repo_test.go` - Persistence, round-trip, and file format tests

## Decisions Made
- btcec/v2 for secp256k1 ECDSA key generation (per user constraint)
- Hand-rolled Base58Check encoding for educational value (per user constraint)
- Atomic JSON file writes via temp file + rename for crash safety
- Wallet entries stored as {address, private_key_hex} in JSON array

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered

None.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness
- Wallet entity and address derivation ready for transaction signing (TX-03, TX-04)
- PubKeyHashFromAddress exported for future tx input validation
- Repository interface ready for service layer integration

## Self-Check: PASSED

All 9 created files verified. All 4 task commits verified (46cf092, 42e63e8, b461109, ad358c7).

---
*Phase: 02-wallets-and-transactions*
*Completed: 2026-03-05*
