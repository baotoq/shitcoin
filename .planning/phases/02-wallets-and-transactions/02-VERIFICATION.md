---
phase: 02-wallets-and-transactions
verified: 2026-03-05T14:00:00Z
status: passed
score: 5/5 must-haves verified
---

# Phase 2: Wallets and Transactions Verification Report

**Phase Goal:** Users can create wallets, derive addresses, and send coins via UTXO transactions with cryptographic signing and verification
**Verified:** 2026-03-05T14:00:00Z
**Status:** passed
**Re-verification:** No -- initial verification

## Goal Achievement

### Observable Truths

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| 1 | User can generate a new ECDSA wallet and receive a human-readable Base58Check address | VERIFIED | `wallet.NewWallet()` generates secp256k1 key via btcec/v2, `PubKeyToAddress` derives P2PKH address with version 0x00 starting with '1'. Hand-rolled Base58Check with leading zero byte preservation. All wallet tests pass. |
| 2 | User can create a transaction that spends specific UTXOs, and the system automatically creates change outputs when input exceeds payment | VERIFIED | `tx.NewTransaction` creates UTXO-based tx with typed inputs (txid:vout) and outputs (satoshi value + address). `CreateTransactionWithChange` in validator.go auto-creates change output when inputSum > amount. Sum invariant enforced by `ValidateTransaction`. All 27 tx tests pass. |
| 3 | Every mined block includes a coinbase transaction that credits the block reward to the miner's address | VERIFIED | `chain.MineBlock` creates coinbase via `tx.NewCoinbaseTx(minerAddress, c.config.BlockReward)` and prepends it to the transaction list (chain.go:113-116). `Initialize` also creates genesis coinbase when minerAddress is provided. BlockReward defaults to 5,000,000,000 satoshis (50 coins). |
| 4 | Transaction inputs with invalid signatures are rejected during validation | VERIFIED | `tx.VerifyTransaction` parses pubkey via `btcec.ParsePubKey`, parses signature via `ecdsa.ParseSignature`, and calls `sig.Verify(txHash.Bytes(), pubKey)` returning false on failure. Empty signatures, wrong keys, and tampered transactions all correctly rejected per test suite. |
| 5 | The UTXO set persists across restarts and supports undo operations (reversibility) for future chain reorganization | VERIFIED | `utxo.Set.ApplyBlock` returns `UndoEntry` recording spent and created UTXOs. `UndoBlock` reverses changes. `bbolt.UTXORepo` persists UTXOs with 36-byte composite keys and undo entries by block height. `SaveBlockWithUTXOs` performs atomic multi-bucket write (block + utxo + undo) in single bbolt transaction. Persistence tests verify round-trips across DB close/reopen. |

**Score:** 5/5 truths verified

### Required Artifacts

| Artifact | Expected | Status | Details |
|----------|----------|--------|---------|
| `internal/domain/wallet/wallet.go` | Wallet entity with ECDSA key generation | VERIFIED | NewWallet, ReconstructWallet, getters for Address, PrivateKey, PublicKey, PrivateKeyHex. 66 lines. |
| `internal/domain/wallet/address.go` | P2PKH address derivation | VERIFIED | PubKeyToAddress (SHA-256 -> RIPEMD-160 -> Base58Check(0x00)), PubKeyHashFromAddress for reverse. 43 lines. |
| `internal/domain/wallet/base58.go` | Hand-rolled Base58Check encode/decode | VERIFIED | Base58Encode/Decode with math/big, Base58CheckEncode/Decode with double-SHA256 checksum. 124 lines. |
| `internal/domain/wallet/repository.go` | Repository interface | VERIFIED | Save, GetByAddress, ListAddresses. 15 lines. |
| `internal/domain/wallet/errors.go` | Sentinel errors | VERIFIED | ErrWalletNotFound, ErrInvalidAddress, ErrInvalidChecksum. |
| `internal/infrastructure/persistence/jsonfile/wallet_repo.go` | JSON file wallet.Repository impl | VERIFIED | Compile-time interface check, atomic file writes (temp+rename), in-memory map. 127 lines. |
| `internal/domain/tx/transaction.go` | Transaction entity with ID computation | VERIFIED | NewTransaction, ReconstructTransaction, ComputeID via JSON+DoubleSHA256 excluding signatures, IsCoinbase. 110 lines. |
| `internal/domain/tx/input.go` | TxInput value object | VERIFIED | txID (block.Hash), vout, signature, pubKey with getters and setters. 51 lines. |
| `internal/domain/tx/output.go` | TxOutput value object | VERIFIED | value (int64 satoshis), address with getters. 25 lines. |
| `internal/domain/tx/coinbase.go` | Coinbase factory | VERIFIED | NewCoinbaseTx with zero hash + 0xFFFFFFFF vout marker, single output to miner. 25 lines. |
| `internal/domain/tx/signing.go` | ECDSA sign/verify | VERIFIED | SignTransaction uses ecdsa.Sign, VerifyTransaction uses ecdsa.ParseSignature + sig.Verify. Skips coinbase. 58 lines. |
| `internal/domain/tx/validator.go` | Structural validation + change outputs | VERIFIED | ValidateTransaction (sum invariant), ValidateCoinbase, CreateTransactionWithChange. 74 lines. |
| `internal/domain/utxo/utxo.go` | UTXO value object | VERIFIED | txID, vout, value, address with Key() method. 43 lines. |
| `internal/domain/utxo/set.go` | UTXOSet aggregate | VERIFIED | ApplyBlock with intra-block double-spend detection, UndoBlock, GetByAddress, GetBalance, Get. 151 lines. |
| `internal/domain/utxo/undo.go` | UndoEntry with SpentUTXO and UTXORef | VERIFIED | All fields exported for JSON serialization. 26 lines. |
| `internal/domain/utxo/repository.go` | Repository interface | VERIFIED | Put, Get, Delete, GetByAddress, SaveUndoEntry, GetUndoEntry, DeleteUndoEntry. 31 lines. |
| `internal/infrastructure/persistence/bbolt/utxo_repo.go` | bbolt UTXO repository | VERIFIED | 36-byte composite keys, utxo+undo buckets, compile-time interface check, byte slice copies for bbolt safety. 181 lines. |
| `internal/infrastructure/persistence/bbolt/chain_repo.go` | Extended with SaveBlockWithUTXOs | VERIFIED | Atomic multi-bucket write (blocks + utxo + undo) in single bbolt Update transaction. 319 lines. |
| `internal/domain/chain/chain.go` | Chain with utxoSet, coinbase, atomic UTXO updates | VERIFIED | MineBlock creates coinbase, prepends to txs, calls ApplyBlock, then SaveBlockWithUTXOs. Initialize handles genesis coinbase. 199 lines. |
| `internal/config/config.go` | BlockReward + SatoshiPerCoin | VERIFIED | BlockReward int64 with default 5000000000, SatoshiPerCoin constant = 100_000_000. |
| `internal/svc/service_context.go` | Wired UTXORepo, UTXOSet, wallet support | VERIFIED | Creates UTXORepo, UTXOSet, passes utxoSet to Chain constructor. BlockReward wired through chainConfig. |

### Key Link Verification

| From | To | Via | Status | Details |
|------|----|-----|--------|---------|
| `wallet/address.go` | `wallet/base58.go` | `Base58CheckEncode(0x00, ...)` | WIRED | Line 26: `return Base58CheckEncode(0x00, pubKeyHash)` |
| `jsonfile/wallet_repo.go` | `wallet/repository.go` | `var _ wallet.Repository` | WIRED | Line 13: compile-time interface check |
| `tx/transaction.go` | `block/hash.go` | `block.DoubleSHA256` | WIRED | Line 77: `return block.DoubleSHA256(data)` |
| `tx/signing.go` | `btcec/v2/ecdsa` | `ecdsa.Sign` and `sig.Verify` | WIRED | Line 19: `ecdsa.Sign(privKey, ...)`, Line 52: `sig.Verify(txHash.Bytes(), pubKey)` |
| `bbolt/chain_repo.go` | `bbolt/utxo_repo.go` | Atomic multi-bucket write | WIRED | Lines 117-118: `utxoBkt`, `undoBkt` accessed in same bbolt Update tx |
| `chain/chain.go` | `utxo/set.go` | `utxoSet.ApplyBlock` | WIRED | Lines 75 and 135: `c.utxoSet.ApplyBlock(...)` |
| `block/block.go` | `tx/transaction.go` | `[]any` for transactions | WIRED | Block uses `[]any`, chain.go type-asserts to `*tx.Transaction` at integration boundary |
| `svc/service_context.go` | All repos/aggregates | Constructor wiring | WIRED | UTXORepo, UTXOSet, Chain all wired with correct dependencies |

### Requirements Coverage

| Requirement | Source Plan | Description | Status | Evidence |
|-------------|------------|-------------|--------|----------|
| TX-01 | 02-01 | User can create a new wallet with ECDSA key pair (secp256k1 curve) | SATISFIED | `wallet.NewWallet()` uses `btcec.NewPrivateKey()` for secp256k1 |
| TX-02 | 02-01 | Public keys converted to human-readable addresses via SHA-256 -> RIPEMD-160 -> Base58Check | SATISFIED | `PubKeyToAddress` implements exact pipeline with version 0x00 |
| TX-03 | 02-02 | User can send coins creating UTXO transaction with inputs and outputs | SATISFIED | `tx.NewTransaction`, `CreateTransactionWithChange` |
| TX-04 | 02-02 | Every transaction input includes valid ECDSA signature | SATISFIED | `tx.SignTransaction` and `tx.VerifyTransaction` with btcec/v2/ecdsa |
| TX-05 | 02-02 | Change outputs automatically created when input exceeds payment | SATISFIED | `CreateTransactionWithChange` creates change output when sum > amount |
| TX-06 | 02-02 | Each mined block includes coinbase transaction for block reward | SATISFIED | `chain.MineBlock` creates `tx.NewCoinbaseTx` and prepends to block |
| TX-07 | 02-03 | System maintains persistent UTXO set for balance queries and validation | SATISFIED | `utxo.Set` with `GetBalance`, `GetByAddress`; `bbolt.UTXORepo` persists |
| TX-08 | 02-03 | UTXO set supports reversibility (undo-log) for chain reorganization | SATISFIED | `UndoEntry` records changes; `UndoBlock` reverses them; undo bucket in bbolt |

No orphaned requirements found -- all 8 TX requirements mapped to Phase 2 in REQUIREMENTS.md are accounted for.

### Anti-Patterns Found

| File | Line | Pattern | Severity | Impact |
|------|------|---------|----------|--------|
| None | - | - | - | No anti-patterns detected |

No TODOs, FIXMEs, placeholders, empty implementations, or console.log-only handlers found in any Phase 2 files.

### Human Verification Required

### 1. Wallet Address Format

**Test:** Generate a wallet and inspect the address string
**Expected:** Address starts with '1', is 25-34 characters, uses only Base58 alphabet characters
**Why human:** Visual inspection of address format quality; tests verify behavior but a human can spot suspicious patterns

### 2. End-to-End Mining with Coinbase

**Test:** Initialize chain with miner address, mine a block, query miner's UTXO balance
**Expected:** Miner address shows balance of 5,000,000,000 satoshis (50 coins) after genesis + 1 block = 10,000,000,000
**Why human:** Integration across wallet, chain, and UTXO requires running the actual application

### Gaps Summary

No gaps found. All 5 observable truths are verified with supporting artifacts that are substantive and correctly wired. All 8 requirements (TX-01 through TX-08) are satisfied. Full test suite passes with race detector. No anti-patterns detected. The phase goal of "Users can create wallets, derive addresses, and send coins via UTXO transactions with cryptographic signing and verification" is achieved.

---

_Verified: 2026-03-05T14:00:00Z_
_Verifier: Claude (gsd-verifier)_
