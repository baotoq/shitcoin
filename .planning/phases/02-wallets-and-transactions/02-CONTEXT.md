# Phase 2: Wallets and Transactions - Context

**Gathered:** 2026-03-05
**Status:** Ready for planning

<domain>
## Phase Boundary

Users can create wallets, derive addresses, and send coins via UTXO transactions with cryptographic signing and verification. Includes ECDSA key management, Bitcoin-style Base58Check addresses, UTXO transaction model with inputs/outputs, coinbase transactions, and a reversible UTXO set with undo-log for future chain reorganization.

Requirements: TX-01, TX-02, TX-03, TX-04, TX-05, TX-06, TX-07, TX-08

</domain>

<decisions>
## Implementation Decisions

### ECDSA & Key Management
- Use btcec library (github.com/btcsuite/btcd/btcec) for real secp256k1 curve -- same as Bitcoin
- Single JSON wallet file (wallets.json) for all key pairs -- loaded into memory on startup
- Plain text hex-encoded keys -- educational project on localhost, no encryption needed (PROJECT.md: "no cryptographic hardening")
- Wallet domain type in `internal/domain/wallet` as its own package with entity and repository interface

### Address Derivation
- Full Bitcoin-style P2PKH: SHA-256 -> RIPEMD-160 -> version byte -> checksum (double SHA-256) -> Base58Check
- Implement Base58Check encoding from scratch (~50 lines) -- understanding the encoding IS the educational point
- Version byte 0x00 (Bitcoin mainnet) -- addresses start with '1', instantly recognizable

### UTXO Set & Undo-Log
- Dedicated bbolt bucket ('utxo') keyed by txid:output_index -- fast lookups for balance queries and tx validation
- Full undo-log built now in separate 'undo' bucket keyed by block height -- records spent and created UTXOs per block
- Phase 4 reorg just reads the undo-log; no retrofitting needed
- UTXO set as separate domain package: `internal/domain/utxo` with UTXOSet aggregate and repository interface
- Atomic bbolt transactions: block save + UTXO update + undo-log write in one bbolt tx -- crash-safe consistency

### Coinbase & Block Reward
- Initial block reward: 50 coins (same as Bitcoin), configurable in go-zero config
- Coin amounts stored as int64 satoshis (1 coin = 100,000,000 satoshis) -- no floating point issues, educational "why satoshis exist"
- Transaction domain type in `internal/domain/tx` -- Transaction, TxInput, TxOutput as domain entities, CoinbaseTx factory method

### Claude's Discretion
- bbolt bucket key format details (encoding of txid:vout composite key)
- Transaction serialization format for hashing (consistent with Phase 1 JSON approach)
- Wallet file location and naming convention
- Error types and validation error messages
- How Block integrates with typed Transaction (replacing current [][]byte)

</decisions>

<code_context>
## Existing Code Insights

### Reusable Assets
- `block.Hash` value object: 32-byte hash with hex encoding, can be reused for transaction IDs
- `block.Header` with MerkleRoot field: currently zero, ready for Phase 3 Merkle tree
- `bbolt.BlockModel` / `HeaderModel`: storage model pattern to follow for TxModel, UTXOModel
- `block.ReconstructBlock()`: pattern for reconstructing domain entities from storage

### Established Patterns
- Domain entities with unexported fields + getters (block.Block pattern)
- Repository interface in domain layer, implementation in infrastructure/persistence/bbolt
- Storage models separate from domain types (BlockModel <-> Block conversion)
- JSON serialization for both hashing and storage
- go-zero config struct binding for consensus parameters

### Integration Points
- `block.Block.transactions` field: currently `[][]byte`, needs to become `[]*tx.Transaction` or similar
- `chain.Chain.MineBlock()`: needs to accept transactions and create coinbase tx
- `chain.Repository.SaveBlock()`: needs to atomically update UTXO set and undo-log
- `bbolt.ChainRepo`: needs to coordinate multi-bucket writes in single bbolt transaction
- `internal/config/config.go`: add block reward and satoshi conversion parameters

</code_context>

<specifics>
## Specific Ideas

- Faithful to Bitcoin wherever practical: same curve (secp256k1), same address format (P2PKH), same reward (50 coins), same unit system (satoshis)
- Base58Check implemented from scratch for educational value -- not imported from a library
- The undo-log is a Phase 2 deliverable (TX-08), not deferred to Phase 4

</specifics>

<deferred>
## Deferred Ideas

None -- discussion stayed within phase scope

</deferred>

---

*Phase: 02-wallets-and-transactions*
*Context gathered: 2026-03-05*
