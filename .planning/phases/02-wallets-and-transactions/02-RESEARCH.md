# Phase 2: Wallets and Transactions - Research

**Researched:** 2026-03-05
**Domain:** ECDSA cryptography, UTXO transaction model, Bitcoin-style address derivation
**Confidence:** HIGH

## Summary

Phase 2 introduces three new domain packages (`wallet`, `tx`, `utxo`) and modifies the existing `block` and `chain` domains to support typed transactions. The core work is: (1) ECDSA key pair management using btcec/v2 (secp256k1), (2) Bitcoin-style P2PKH address derivation with hand-rolled Base58Check, (3) UTXO-based transaction model with inputs/outputs/coinbase, and (4) a persistent UTXO set with undo-log for future chain reorganization.

The project already has strong patterns established in Phase 1: unexported domain entity fields with getters, repository interfaces in domain layer, storage models separate from domain types, JSON serialization for hashing, and bbolt for persistence. Phase 2 follows these patterns exactly while adding three new bbolt buckets (`utxo`, `undo`, and potentially `wallets`).

**Primary recommendation:** Build bottom-up: wallet/address first (no dependencies on existing code), then transaction types, then UTXO set + undo-log, then integrate into block/chain. The block.Block.transactions field change from `[][]byte` to `[]*tx.Transaction` is the riskiest integration point -- plan it carefully.

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions
- Use btcec library (github.com/btcsuite/btcd/btcec/v2) for real secp256k1 curve -- same as Bitcoin
- Single JSON wallet file (wallets.json) for all key pairs -- loaded into memory on startup
- Plain text hex-encoded keys -- educational project on localhost, no encryption needed
- Wallet domain type in `internal/domain/wallet` as its own package with entity and repository interface
- Full Bitcoin-style P2PKH: SHA-256 -> RIPEMD-160 -> version byte -> checksum (double SHA-256) -> Base58Check
- Implement Base58Check encoding from scratch (~50 lines) -- understanding the encoding IS the educational point
- Version byte 0x00 (Bitcoin mainnet) -- addresses start with '1', instantly recognizable
- Dedicated bbolt bucket ('utxo') keyed by txid:output_index -- fast lookups for balance queries and tx validation
- Full undo-log built now in separate 'undo' bucket keyed by block height -- records spent and created UTXOs per block
- UTXO set as separate domain package: `internal/domain/utxo` with UTXOSet aggregate and repository interface
- Atomic bbolt transactions: block save + UTXO update + undo-log write in one bbolt tx -- crash-safe consistency
- Initial block reward: 50 coins (same as Bitcoin), configurable in go-zero config
- Coin amounts stored as int64 satoshis (1 coin = 100,000,000 satoshis)
- Transaction domain type in `internal/domain/tx` -- Transaction, TxInput, TxOutput as domain entities, CoinbaseTx factory method

### Claude's Discretion
- bbolt bucket key format details (encoding of txid:vout composite key)
- Transaction serialization format for hashing (consistent with Phase 1 JSON approach)
- Wallet file location and naming convention
- Error types and validation error messages
- How Block integrates with typed Transaction (replacing current [][]byte)

### Deferred Ideas (OUT OF SCOPE)
None -- discussion stayed within phase scope
</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|-----------------|
| TX-01 | User can create a new wallet with ECDSA key pair (secp256k1 curve) | btcec/v2 NewPrivateKey() generates key; wallet package stores in JSON file |
| TX-02 | Public keys converted to addresses via SHA-256 -> RIPEMD-160 -> Base58Check | Hand-rolled Base58Check + golang.org/x/crypto/ripemd160; address derivation code examples below |
| TX-03 | User can send coins creating UTXO transaction with inputs and outputs | Transaction domain type with TxInput/TxOutput; UTXO set lookup for spendable outputs |
| TX-04 | Every transaction input references a specific unspent output with valid ECDSA signature | btcec/v2/ecdsa.Sign() for signing, Verify() for validation; signature covers tx hash |
| TX-05 | Change outputs automatically created when input exceeds payment (sum invariant) | Transaction creation logic enforces sum(inputs) == sum(outputs); change output auto-generated |
| TX-06 | Each mined block includes coinbase transaction creating block reward for miner | CoinbaseTx factory method; no inputs, single output to miner address; reward from config |
| TX-07 | System maintains persistent UTXO set for balance queries and tx validation | bbolt 'utxo' bucket keyed by txid:vout; UTXOSet aggregate with repository interface |
| TX-08 | UTXO set supports reversibility (undo-log) for chain reorganization | bbolt 'undo' bucket keyed by block height; records spent+created UTXOs per block |
</phase_requirements>

## Standard Stack

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| github.com/btcsuite/btcd/btcec/v2 | latest | secp256k1 ECDSA key generation, signing | Bitcoin's actual crypto library; pure Go implementation |
| github.com/btcsuite/btcd/btcec/v2/ecdsa | latest | ECDSA Sign/Verify functions | Sub-package of btcec for signature operations |
| golang.org/x/crypto/ripemd160 | latest | RIPEMD-160 hash for address derivation | Only Go implementation of RIPEMD-160; required for P2PKH |
| go.etcd.io/bbolt v1.4.3 | (already in go.mod) | Persistence for UTXO set and undo-log | Already used for block storage in Phase 1 |

### Supporting
| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| crypto/sha256 | stdlib | SHA-256 hashing in address derivation | Part of P2PKH pipeline and tx hashing |
| encoding/json | stdlib | Transaction serialization for hashing | Consistent with Phase 1 JSON serialization approach |
| encoding/hex | stdlib | Key serialization to/from hex strings | Wallet file stores keys as hex |
| math/big | stdlib | Base58Check big integer division | Core of Base58 encoding algorithm |

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| btcec/v2 | crypto/ecdsa + elliptic | btcec provides secp256k1 natively; stdlib only has P-256/P-384/P-521 |
| Hand-rolled Base58Check | github.com/btcsuite/btcutil/base58 | User decision: hand-roll for educational value |
| golang.org/x/crypto/ripemd160 | Hand-roll RIPEMD-160 | x/crypto is fine; RIPEMD-160 is not the educational point |

**Installation:**
```bash
go get github.com/btcsuite/btcd/btcec/v2
go get golang.org/x/crypto/ripemd160
```

Note: btcec/v2 depends on `github.com/decred/dcrd/dcrec/secp256k1/v4` internally. This will be pulled transitively.

## Architecture Patterns

### Recommended Project Structure
```
internal/
├── domain/
│   ├── block/           # (existing) - modified: transactions field type change
│   ├── chain/           # (existing) - modified: MineBlock accepts txs, atomic saves
│   ├── wallet/          # (NEW) - Wallet entity, KeyPair, Address value objects
│   │   ├── wallet.go    # Wallet entity with key pairs
│   │   ├── address.go   # Address value object, Base58Check encode/decode
│   │   ├── base58.go    # Hand-rolled Base58Check implementation
│   │   ├── repository.go # Repository interface (JSON file)
│   │   └── errors.go
│   ├── tx/              # (NEW) - Transaction, TxInput, TxOutput entities
│   │   ├── transaction.go # Transaction entity with ID computation
│   │   ├── input.go     # TxInput value object
│   │   ├── output.go    # TxOutput value object
│   │   ├── coinbase.go  # CoinbaseTx factory
│   │   ├── validator.go # Transaction validation logic
│   │   └── errors.go
│   └── utxo/            # (NEW) - UTXOSet aggregate, UndoLog
│       ├── utxo.go      # UTXO value object
│       ├── set.go       # UTXOSet aggregate (apply block, undo block)
│       ├── undo.go      # UndoEntry for reversibility
│       ├── repository.go # Repository interface
│       └── errors.go
├── infrastructure/
│   └── persistence/
│       ├── bbolt/       # (existing) - extended with UTXO + undo buckets
│       │   ├── chain_repo.go      # (modified) - atomic multi-bucket writes
│       │   ├── utxo_repo.go       # (NEW) - UTXO set persistence
│       │   ├── utxo_storage_model.go # (NEW) - UTXO storage models
│       │   └── storage_model.go   # (modified) - tx storage models
│       └── jsonfile/    # (NEW) - JSON file persistence for wallets
│           └── wallet_repo.go
└── config/
    └── config.go        # (modified) - add BlockReward, SatoshiPerCoin
```

### Pattern 1: Transaction Hashing (JSON Serialization)
**What:** Deterministic transaction ID via JSON serialization + double SHA-256
**When to use:** Computing transaction IDs, creating signing payloads
**Example:**
```go
// Consistent with Phase 1's hashableHeader pattern
type hashableTransaction struct {
    Inputs  []hashableInput  `json:"inputs"`
    Outputs []hashableOutput `json:"outputs"`
}

type hashableInput struct {
    TxID      string `json:"txid"`
    Vout      uint32 `json:"vout"`
    // Note: signature NOT included in hash (chicken-and-egg problem)
}

type hashableOutput struct {
    Value   int64  `json:"value"`   // satoshis
    Address string `json:"address"` // Base58Check address
}

func (t *Transaction) ComputeID() block.Hash {
    payload := t.hashPayload()
    data, _ := json.Marshal(payload)
    return block.DoubleSHA256(data)
}
```

### Pattern 2: UTXO Composite Key
**What:** bbolt key format for UTXO lookups
**When to use:** Storing and retrieving UTXOs
**Example:**
```go
// Key format: 32-byte txid + 4-byte big-endian vout index
// Total: 36 bytes, efficient binary key, no string conversion needed
func utxoKey(txID block.Hash, vout uint32) []byte {
    key := make([]byte, 36)
    copy(key[:32], txID.Bytes())
    binary.BigEndian.PutUint32(key[32:], vout)
    return key
}
```

### Pattern 3: Undo-Log Entry
**What:** Records per-block UTXO changes for reversibility
**When to use:** Every block save, consumed during chain reorganization
**Example:**
```go
type UndoEntry struct {
    BlockHeight uint64      `json:"block_height"`
    Spent       []SpentUTXO `json:"spent"`   // UTXOs consumed (need to restore on undo)
    Created     []UTXORef   `json:"created"` // UTXOs created (need to remove on undo)
}

type SpentUTXO struct {
    TxID    string `json:"txid"`
    Vout    uint32 `json:"vout"`
    Value   int64  `json:"value"`   // preserve original value for restoration
    Address string `json:"address"` // preserve original address
}

type UTXORef struct {
    TxID string `json:"txid"`
    Vout uint32 `json:"vout"`
}
```

### Pattern 4: Atomic Multi-Bucket Write
**What:** Single bbolt transaction spanning blocks, utxo, undo, and chain_meta buckets
**When to use:** Every SaveBlock call in Phase 2+
**Example:**
```go
// The chain repo SaveBlock must be extended to accept UTXO changes
func (r *BboltRepository) SaveBlockWithUTXOs(
    ctx context.Context,
    b *block.Block,
    utxosToSpend []utxo.UTXO,    // remove from utxo bucket
    utxosToCreate []utxo.UTXO,   // add to utxo bucket
    undoEntry *utxo.UndoEntry,   // write to undo bucket
) error {
    return r.db.Update(func(tx *bolt.Tx) error {
        // 1. Save block (existing logic)
        // 2. Remove spent UTXOs from 'utxo' bucket
        // 3. Add new UTXOs to 'utxo' bucket
        // 4. Write undo entry to 'undo' bucket
        // 5. Update chain_meta
        // All in ONE bbolt transaction = crash-safe
        return nil
    })
}
```

### Pattern 5: Signing Flow (What Gets Signed)
**What:** ECDSA signature covers a simplified transaction hash, not the full transaction
**When to use:** Creating and validating transaction inputs
**Example:**
```go
// Signing: for each input, sign the transaction hash (without signatures)
// This is a simplified version of Bitcoin's SIGHASH_ALL
func SignInput(tx *Transaction, inputIndex int, privKey *btcec.PrivateKey) error {
    // 1. Compute tx hash WITHOUT any signatures (clean hash)
    txHash := tx.ComputeID()

    // 2. Sign the hash with the private key
    sig := ecdsa.Sign(privKey, txHash.Bytes())

    // 3. Store serialized signature + compressed public key on the input
    tx.inputs[inputIndex].SetSignature(sig.Serialize())
    tx.inputs[inputIndex].SetPubKey(privKey.PubKey().SerializeCompressed())
    return nil
}

// Verification: recompute hash, check signature against pubkey
func VerifyInput(tx *Transaction, inputIndex int) bool {
    input := tx.inputs[inputIndex]
    txHash := tx.ComputeID()

    pubKey, err := btcec.ParsePubKey(input.PubKey())
    if err != nil {
        return false
    }

    sig, err := ecdsa.ParseSignature(input.Signature())
    if err != nil {
        return false
    }

    return sig.Verify(txHash.Bytes(), pubKey)
}
```

### Anti-Patterns to Avoid
- **Including signature in signed data:** The signature field must be excluded from the hash that gets signed (chicken-and-egg problem). Use a separate hashable struct without signature fields.
- **Floating point for amounts:** Always use int64 satoshis. Even temporary float64 conversions can introduce rounding errors.
- **Non-atomic UTXO updates:** Never update UTXO set outside the same bbolt transaction that saves the block. A crash between separate transactions would corrupt the UTXO set.
- **Storing full transaction data in UTXO set:** UTXOs only need txid, vout, value, and address. Don't duplicate the full transaction.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| secp256k1 curve math | Custom elliptic curve implementation | btcec/v2 | Curve implementation is security-critical and subtle |
| RIPEMD-160 hash | Custom hash function | golang.org/x/crypto/ripemd160 | Standard implementation, not the educational goal |
| ECDSA signing/verification | Custom signature scheme | btcec/v2/ecdsa | Deterministic signatures per RFC6979 are non-trivial |
| Key serialization formats | Custom compressed/uncompressed encoding | btcec/v2 SerializeCompressed() | Standard SEC encoding with parity byte |

**Key insight:** Hand-roll Base58Check (per user decision) because understanding the encoding is educational. Do NOT hand-roll cryptographic primitives -- the educational value is in using them correctly, not reimplementing them.

## Common Pitfalls

### Pitfall 1: Transaction ID Includes Signatures
**What goes wrong:** Transaction ID changes after signing, breaking UTXO references
**Why it happens:** Naively hashing the entire transaction struct including signature bytes
**How to avoid:** Hash only the "clean" transaction data (inputs without sigs, outputs). Use a separate hashable struct.
**Warning signs:** Transaction IDs differ before and after signing

### Pitfall 2: bbolt Byte Slice Lifetime
**What goes wrong:** Data corruption or panics when reading bbolt values outside transaction
**Why it happens:** bbolt byte slices are only valid within the transaction callback (existing Phase 1 pitfall)
**How to avoid:** Always copy byte slices before the bbolt transaction closes (already established in Phase 1)
**Warning signs:** Intermittent corruption, panics on previously-working reads

### Pitfall 3: Base58Check Leading Zeros
**What goes wrong:** Addresses lose leading '1' characters, producing wrong addresses
**Why it happens:** Big integer division strips leading zero bytes; Bitcoin encodes each leading 0x00 byte as '1'
**How to avoid:** After big.Int division, count leading zero bytes in the input and prepend that many '1' characters
**Warning signs:** Address length varies unexpectedly, addresses don't start with '1'

### Pitfall 4: UTXO Double-Spend
**What goes wrong:** Same UTXO spent twice in the same block
**Why it happens:** Validation checks UTXO set but doesn't track UTXOs spent by earlier transactions in the same block
**How to avoid:** During block validation, maintain an in-memory set of UTXOs spent by prior transactions in the current block
**Warning signs:** Balance increases unexpectedly, UTXO count doesn't decrease correctly

### Pitfall 5: Coinbase Transaction Validation
**What goes wrong:** Coinbase transactions rejected by the same validation rules as regular transactions
**Why it happens:** Coinbase has no inputs (no signatures to verify, no UTXOs to reference)
**How to avoid:** Check if transaction is coinbase (IsCoinbase() method) and skip input validation. Validate that only one coinbase exists per block and its output value equals the block reward.
**Warning signs:** Genesis/mined blocks fail validation

### Pitfall 6: Block.transactions Type Change Breaks Storage
**What goes wrong:** Existing blocks in bbolt can't be deserialized after changing transactions from `[][]byte` to typed transactions
**Why it happens:** Storage model format changes between phases
**How to avoid:** The BlockModel storage format must handle both old `[][]byte` (Phase 1 blocks) and new typed transactions. Since Phase 1 blocks have empty transaction arrays, this should be straightforward -- just ensure empty arrays deserialize correctly.
**Warning signs:** Existing chain data fails to load after Phase 2 deployment

### Pitfall 7: Satoshi Overflow
**What goes wrong:** int64 overflow when summing large transaction values
**Why it happens:** int64 max is ~92 billion coins in satoshis (9.2 * 10^18), which is safe for Bitcoin's 21M cap but could overflow with careless test values
**How to avoid:** Validate individual output values are positive and less than max supply. Validate sums don't overflow before comparing.
**Warning signs:** Negative balances, panic on large test values

## Code Examples

### Base58Check Encoding (Hand-Rolled)
```go
// Source: Bitcoin wiki Base58Check encoding specification
const base58Alphabet = "123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz"

func Base58Encode(input []byte) string {
    var result []byte
    x := new(big.Int).SetBytes(input)
    base := big.NewInt(58)
    zero := big.NewInt(0)
    mod := new(big.Int)

    for x.Cmp(zero) > 0 {
        x.DivMod(x, base, mod)
        result = append(result, base58Alphabet[mod.Int64()])
    }

    // Preserve leading zeros as '1' characters
    for _, b := range input {
        if b != 0x00 {
            break
        }
        result = append(result, base58Alphabet[0])
    }

    // Reverse
    for i, j := 0, len(result)-1; i < j; i, j = i+1, j-1 {
        result[i], result[j] = result[j], result[i]
    }

    return string(result)
}

func Base58CheckEncode(version byte, payload []byte) string {
    versionedPayload := append([]byte{version}, payload...)
    firstHash := sha256.Sum256(versionedPayload)
    secondHash := sha256.Sum256(firstHash[:])
    checksum := secondHash[:4]
    fullPayload := append(versionedPayload, checksum...)
    return Base58Encode(fullPayload)
}
```

### Address Derivation from Public Key
```go
// Source: Bitcoin P2PKH address derivation
func PubKeyToAddress(pubKey *btcec.PublicKey) string {
    // 1. Compressed public key (33 bytes)
    pubKeyBytes := pubKey.SerializeCompressed()

    // 2. SHA-256
    sha256Hash := sha256.Sum256(pubKeyBytes)

    // 3. RIPEMD-160
    ripeHasher := ripemd160.New()
    ripeHasher.Write(sha256Hash[:])
    pubKeyHash := ripeHasher.Sum(nil) // 20 bytes

    // 4. Base58Check with version byte 0x00
    return Base58CheckEncode(0x00, pubKeyHash)
}
```

### ECDSA Key Generation
```go
// Source: btcec/v2 official API
func GenerateKeyPair() (*btcec.PrivateKey, *btcec.PublicKey, error) {
    privKey, err := btcec.NewPrivateKey()
    if err != nil {
        return nil, nil, fmt.Errorf("generate private key: %w", err)
    }
    return privKey, privKey.PubKey(), nil
}

// Serialize/deserialize for wallet file storage
func SerializePrivKey(privKey *btcec.PrivateKey) string {
    return hex.EncodeToString(privKey.Serialize())
}

func DeserializePrivKey(hexStr string) (*btcec.PrivateKey, *btcec.PublicKey, error) {
    privKeyBytes, err := hex.DecodeString(hexStr)
    if err != nil {
        return nil, nil, fmt.Errorf("decode private key hex: %w", err)
    }
    privKey, pubKey := btcec.PrivKeyFromBytes(privKeyBytes)
    return privKey, pubKey, nil
}
```

### Coinbase Transaction Factory
```go
func NewCoinbaseTx(minerAddress string, reward int64) *Transaction {
    // Coinbase has no real inputs -- use a special marker
    input := TxInput{
        txID:      block.Hash{},  // zero hash = coinbase marker
        vout:      0xFFFFFFFF,    // max uint32 = coinbase marker
        signature: nil,
        pubKey:    nil,
    }
    output := TxOutput{
        value:   reward,
        address: minerAddress,
    }
    tx := &Transaction{
        inputs:  []TxInput{input},
        outputs: []TxOutput{output},
    }
    tx.id = tx.ComputeID()
    return tx
}

func (tx *Transaction) IsCoinbase() bool {
    return len(tx.inputs) == 1 &&
        tx.inputs[0].txID.IsZero() &&
        tx.inputs[0].vout == 0xFFFFFFFF
}
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| btcec v1 (single package) | btcec/v2 (split into btcec + ecdsa sub-packages) | btcd module restructure | Import paths changed; Sign/Verify moved to ecdsa sub-package |
| chainhash.DoubleHashB for tx hashing | Can use project's existing block.DoubleSHA256 | Phase 1 | Reuse existing hash function; don't import chainhash |
| btcutil/base58 for Base58Check | Hand-rolled per user decision | N/A | Educational value over library convenience |

**Deprecated/outdated:**
- `golang.org/x/crypto/ripemd160`: Marked deprecated for new applications but still works and is required for Bitcoin P2PKH addresses. No replacement exists for this use case.
- btcec v1 (`github.com/btcsuite/btcec`): Replaced by btcec/v2 with different import paths and API split.

## Open Questions

1. **Block.transactions type migration strategy**
   - What we know: Currently `[][]byte`, needs to become typed transactions
   - What's unclear: Whether to use `[]tx.Transaction` or keep a more generic interface for the block package
   - Recommendation: Change to `[]*tx.Transaction` directly. The block package can import tx package (block depends on tx, not vice versa). Update BlockModel.Transactions to use a TxModel slice. Existing Phase 1 blocks have empty transaction arrays so no migration is needed.

2. **Transaction validation ownership**
   - What we know: Validation involves UTXO lookups, signature verification, and sum invariant checks
   - What's unclear: Whether validation lives in the tx package, utxo package, or chain aggregate
   - Recommendation: Put structural validation (sum invariant, signature format) in `tx.Validator`. Put contextual validation (UTXO existence, double-spend checks) in the chain aggregate or a dedicated validation service that coordinates both.

3. **Wallet repository implementation**
   - What we know: Single JSON file, loaded into memory on startup
   - What's unclear: File location relative to project root
   - Recommendation: Use same directory as bbolt DB (configurable via `Storage.WalletPath` with default `data/wallets.json`). JSON file repo implements the wallet.Repository interface.

## Validation Architecture

### Test Framework
| Property | Value |
|----------|-------|
| Framework | Go testing (stdlib) |
| Config file | none (Go convention) |
| Quick run command | `go test ./internal/domain/wallet/... ./internal/domain/tx/... ./internal/domain/utxo/... -v -count=1` |
| Full suite command | `go test ./... -v -race -count=1` |

### Phase Requirements to Test Map
| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| TX-01 | Generate ECDSA wallet, store/load key pairs | unit | `go test ./internal/domain/wallet/... -run TestGenerateWallet -v` | Wave 0 |
| TX-02 | PubKey -> SHA-256 -> RIPEMD-160 -> Base58Check address | unit | `go test ./internal/domain/wallet/... -run TestAddressDerivation -v` | Wave 0 |
| TX-03 | Create UTXO transaction with inputs/outputs | unit | `go test ./internal/domain/tx/... -run TestCreateTransaction -v` | Wave 0 |
| TX-04 | Sign input and verify ECDSA signature | unit | `go test ./internal/domain/tx/... -run TestSignVerify -v` | Wave 0 |
| TX-05 | Auto-create change output, sum invariant | unit | `go test ./internal/domain/tx/... -run TestChangeOutput -v` | Wave 0 |
| TX-06 | Coinbase transaction with block reward | unit | `go test ./internal/domain/tx/... -run TestCoinbaseTx -v` | Wave 0 |
| TX-07 | UTXO set add/remove/query persistence | integration | `go test ./internal/infrastructure/persistence/bbolt/... -run TestUTXORepo -v` | Wave 0 |
| TX-08 | Undo-log write/read, apply/revert block UTXOs | integration | `go test ./internal/domain/utxo/... -run TestUndoLog -v` | Wave 0 |

### Sampling Rate
- **Per task commit:** `go test ./internal/domain/wallet/... ./internal/domain/tx/... ./internal/domain/utxo/... -v -count=1`
- **Per wave merge:** `go test ./... -v -race -count=1`
- **Phase gate:** Full suite green before `/gsd:verify-work`

### Wave 0 Gaps
- [ ] `internal/domain/wallet/wallet_test.go` -- covers TX-01, TX-02
- [ ] `internal/domain/wallet/base58_test.go` -- covers Base58Check encode/decode with known Bitcoin test vectors
- [ ] `internal/domain/tx/transaction_test.go` -- covers TX-03, TX-04, TX-05, TX-06
- [ ] `internal/domain/utxo/set_test.go` -- covers TX-07, TX-08 (domain logic)
- [ ] `internal/infrastructure/persistence/bbolt/utxo_repo_test.go` -- covers TX-07 (persistence)
- [ ] `internal/infrastructure/persistence/jsonfile/wallet_repo_test.go` -- covers wallet persistence

## Sources

### Primary (HIGH confidence)
- [btcec/v2 package docs](https://pkg.go.dev/github.com/btcsuite/btcd/btcec/v2) - Key types, NewPrivateKey, PrivKeyFromBytes, ParsePubKey
- [btcec/v2/ecdsa package docs](https://pkg.go.dev/github.com/btcsuite/btcd/btcec/v2/ecdsa) - Sign, Verify, ParseSignature
- [btcec example_test.go](https://github.com/btcsuite/btcd/blob/master/btcec/ecdsa/example_test.go) - Complete sign/verify workflow
- [btcec pubkey.go](https://github.com/btcsuite/btcd/blob/master/btcec/pubkey.go) - PublicKey = secp.PublicKey alias, SerializeCompressed available
- [golang.org/x/crypto/ripemd160](https://pkg.go.dev/golang.org/x/crypto/ripemd160) - RIPEMD-160 implementation (deprecated for new apps but required for Bitcoin addresses)
- Existing codebase analysis: block.go, chain.go, chain_repo.go, storage_model.go, config.go patterns

### Secondary (MEDIUM confidence)
- [Bitcoin wiki Base58Check encoding](https://en.bitcoin.it/wiki/Base58Check_encoding) - Algorithm specification for hand-rolled implementation
- [btcsuite/btcutil/base58](https://pkg.go.dev/github.com/btcsuite/btcutil/base58) - Reference implementation to verify hand-rolled code against

### Tertiary (LOW confidence)
- None

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH - btcec/v2 is well-documented with examples; ripemd160 is stdlib-adjacent
- Architecture: HIGH - follows established Phase 1 patterns exactly (domain entities, repos, storage models, bbolt)
- Pitfalls: HIGH - well-known Bitcoin implementation pitfalls with clear mitigations
- Integration points: MEDIUM - Block.transactions type change touches multiple layers but approach is straightforward

**Research date:** 2026-03-05
**Valid until:** 2026-04-05 (stable domain, no rapid changes expected)
