# Domain Pitfalls

**Domain:** Educational blockchain implementation in Go (UTXO-based, PoW, P2P)
**Researched:** 2026-03-05

---

## Critical Pitfalls

Mistakes that cause rewrites, data corruption, or fundamentally broken blockchain logic.

---

### Pitfall 1: Non-Deterministic Serialization Breaks Consensus

**What goes wrong:** Two nodes hash the same block differently because serialization output is not byte-identical. The chain forks silently and nodes reject each other's valid blocks. This is the single most common show-stopper in Go blockchain projects.

**Why it happens:**
- Go's `map` type deliberately randomizes iteration order on every run. If any data structure used in block/transaction hashing contains a map (e.g., `map[string]TxOutput`), the serialized bytes differ between runs and between nodes.
- `encoding/json` uses sorted keys for maps, but `encoding/gob` does not guarantee cross-version stability. Gob is Go-specific and its encoding can change between Go versions.
- Struct field ordering matters: adding, removing, or reordering fields in a struct changes the serialized output, invalidating all previously stored hashes.

**Consequences:** Chain forks between nodes running same code. Blocks mined on one node are invalid on another. Stored blockchain becomes unreadable after struct changes.

**Prevention:**
- Never use `map` in any data structure that gets hashed or serialized for consensus. Use sorted slices of key-value pairs instead.
- Define an explicit canonical byte format for hashing: manually serialize fields in a fixed, documented order using `binary.Write` or a bytes buffer. Do not rely on `encoding/gob` or `encoding/json` for consensus-critical hashing.
- Keep `encoding/gob` (or JSON) only for storage/display. The "hash input" serialization must be a separate, hand-written function.
- Write a cross-node test early: serialize a block on node A, deserialize and re-hash on node B, assert identical hash.

**Detection:**
- Two nodes mining on the same chain suddenly disagree on block validity.
- Stored chain fails validation after recompiling with a newer Go version.
- Tests pass when run once but fail intermittently (map ordering flakiness).

**Phase relevance:** Must be addressed in the very first phase (block/transaction data structures). Retrofitting deterministic serialization is a full rewrite of every hash function.

---

### Pitfall 2: UTXO Double-Spend from Concurrent Mempool Access

**What goes wrong:** Two transactions spending the same UTXO are both accepted into the mempool because the UTXO set is checked without proper locking. When both are included in a block candidate, the block is invalid (or worse, one spend silently overwrites the other).

**Why it happens:**
- The mempool and UTXO set are shared mutable state accessed from multiple goroutines: the P2P message handler (receiving transactions from peers), the RPC/CLI handler (local transaction submission), and the miner (reading the mempool to build blocks).
- Educational projects often start single-threaded and add networking later without retrofitting concurrency controls.
- The check-then-act pattern (`if utxo.Exists() { spend(utxo) }`) is inherently racy without a lock held across both operations.

**Consequences:** Double-spend within a single node. Invalid blocks that waste mining work. Corrupted UTXO set that diverges from the actual chain state.

**Prevention:**
- Use a single mutex (`sync.RWMutex`) that protects the entire "validate + add to mempool" operation atomically. Read-lock for balance queries, write-lock for transaction acceptance.
- When building a block candidate, mark UTXOs consumed by selected transactions so subsequent selections skip them.
- Validate the assembled block's transactions against each other before mining (check for conflicting inputs within the same block).
- Run tests with `-race` flag from day one. Build the UTXO set with concurrency in mind from the start, not as an afterthought.

**Detection:**
- `go test -race` reports data races on the UTXO map or mempool slice.
- Mining produces blocks that fail self-validation.
- Balance queries return stale or incorrect values under load.

**Phase relevance:** Transaction/UTXO phase. The locking design must exist before P2P networking is added.

---

### Pitfall 3: Forgotten Change Outputs Destroy Coins

**What goes wrong:** A transaction spends a UTXO worth 10 coins to send 3 coins but does not create a change output for the remaining 7. Those 7 coins are permanently lost -- they are consumed from the UTXO set and never re-created.

**Why it happens:**
- In the UTXO model, you must consume entire UTXOs. If a UTXO is worth more than the payment amount, the difference must be sent back to the sender as a new output (change).
- Educational tutorials sometimes skip change output logic in early versions, or the wallet code calculates the total input value but forgets to subtract the payment and create a change output.
- Unlike the account model (Ethereum), where the "change" is implicit, UTXO requires explicit change outputs.

**Consequences:** Users silently lose coins with every transaction. Coin supply shrinks over time. Debugging is confusing because balances drop by more than the sent amount.

**Prevention:**
- In the transaction creation function, always calculate: `changeAmount = totalInputValue - paymentAmount`. If `changeAmount > 0`, create an additional output back to the sender's address.
- If `totalInputValue < paymentAmount`, reject the transaction (insufficient funds).
- If `totalInputValue == paymentAmount` exactly, no change output is needed (but this is rare).
- Add an assertion/invariant check: `sum(outputs) == sum(inputs)` for every non-coinbase transaction (coinbase creates new coins, so it is exempt).
- Write a unit test that sends a partial amount and verifies the sender's remaining balance.

**Detection:**
- User's balance drops by the full UTXO value instead of just the sent amount.
- Total coin supply decreases over time when it should only increase (from mining rewards).
- `sum(outputs) != sum(inputs)` in transaction validation.

**Phase relevance:** Transaction/UTXO phase. Must be correct before any wallet or CLI functionality is built on top.

---

### Pitfall 4: BoltDB Deadlock from Mixed Read/Write Transactions

**What goes wrong:** The application hangs permanently (deadlock) when a read transaction and a write transaction are opened on the same goroutine, or when a long-running read transaction blocks write transactions from completing.

**Why it happens:**
- BoltDB allows multiple concurrent read transactions but only one write transaction at a time. Write transactions are serialized by acquiring an exclusive lock.
- BoltDB periodically re-mmaps the data file as it grows. The writer needs an exclusive `mmaplock` to do this, but readers hold a shared lock. If a read transaction is open on the same goroutine that is trying to write, the write blocks on the mmap lock held by the read, and the read can never close because the goroutine is blocked. Classic deadlock.
- Common pattern: "read current chain tip, then write new block" -- if done in one goroutine with separate transactions, it deadlocks.

**Consequences:** Application freezes permanently. No crash, no error message -- just hangs. Extremely difficult to debug without knowing the BoltDB concurrency model.

**Prevention:**
- Never open a read transaction and a write transaction on the same goroutine. Use `db.Update()` (which gives you a read-write transaction) when you need to read-then-write.
- Keep transactions short. Do computation outside the transaction, then open a transaction only for the actual read or write.
- Set `DB.InitialMmapSize` to a large value (e.g., 1GB) to reduce remapping frequency if long-running read transactions are unavoidable.
- Alternatively, use BadgerDB which has a different concurrency model (MVCC) that avoids this class of deadlock entirely.
- Document BoltDB's concurrency rules in a comment at the database layer. Future-you will not remember.

**Detection:**
- Application hangs after running for a while with no error output.
- Adding `-race` flag does NOT catch this because it is a logical deadlock, not a data race.
- `pprof` goroutine dump shows goroutines blocked on BoltDB internal locks.

**Phase relevance:** Storage/persistence phase. Must be understood before any code that reads-then-writes to the chain database.

---

### Pitfall 5: No Chain Reorganization Handling

**What goes wrong:** When two nodes mine different blocks at the same height (a natural temporary fork), nodes cannot switch to the longer chain. They are permanently stuck on their own fork.

**Why it happens:**
- Educational projects typically implement "append only" chain logic: receive block, validate, append. They never implement "receive block, realize it is on a longer competing chain, undo my last N blocks and switch."
- Reorg requires the ability to: (a) undo blocks (restore consumed UTXOs, remove created UTXOs), (b) compare total chain work (not just height), and (c) re-apply the new chain's blocks.
- This is complex enough that most tutorials skip it entirely, but without it, multi-node operation is fragile.

**Consequences:** Nodes permanently diverge after any concurrent mining. The P2P network splits into isolated views of the chain. The blockchain is no longer a consensus system.

**Prevention:**
- Design the UTXO set to be reversible from the start: when applying a block, log which UTXOs were consumed and created so they can be undone.
- Compare chains by cumulative difficulty (sum of work), not just block count.
- Implement a `reorganize(oldTip, newTip)` function that finds the common ancestor, reverts blocks back to it, then applies the new chain's blocks forward.
- For an educational project, even a simplified reorg (only reorg if the competing chain is strictly longer and the fork is at most N blocks deep) is vastly better than no reorg.

**Detection:**
- Running two nodes that both mine: they diverge and never reconverge.
- One node cannot sync to another node's chain after a temporary network partition.

**Phase relevance:** Must be considered during UTXO set design (reversibility) and implemented during P2P/sync phase.

---

## Moderate Pitfalls

Mistakes that cause bugs, poor performance, or significant debugging pain -- but are fixable without a full rewrite.

---

### Pitfall 6: Goroutine Leaks from P2P Connections

**What goes wrong:** Every peer connection spawns goroutines for reading, writing, and heartbeating. When a peer disconnects (cleanly or not), those goroutines are never terminated. Over time, the node accumulates thousands of leaked goroutines, exhausting memory.

**Why it happens:**
- Each TCP connection typically gets 2-3 goroutines (read loop, write loop, sometimes a ping/keepalive loop).
- When the connection closes, the read goroutine is usually unblocked (read returns an error), but the write goroutine may be blocked on a channel send and never wakes up.
- No `context.Context` is threaded through the connection lifecycle, so there is no cancellation mechanism.

**Prevention:**
- Use `context.WithCancel` for each peer connection. When the connection closes for any reason, cancel the context. All goroutines for that peer select on `ctx.Done()`.
- Use `sync.WaitGroup` to track goroutines per connection. The disconnect handler calls `cancel()` then `wg.Wait()` to ensure cleanup.
- Set read/write deadlines on TCP connections (`conn.SetReadDeadline`, `conn.SetWriteDeadline`) so goroutines blocked on I/O eventually timeout.
- Monitor goroutine count with `runtime.NumGoroutine()` in tests and log it periodically in production.

**Detection:**
- `runtime.NumGoroutine()` grows monotonically, never decreasing even as peers disconnect.
- Memory usage climbs steadily. Node becomes sluggish over time.
- `pprof` goroutine profile shows many goroutines blocked in `conn.Read` or channel operations.

**Phase relevance:** P2P networking phase. Build the connection lifecycle with context cancellation from the first peer connection implementation.

---

### Pitfall 7: TCP Message Framing Errors

**What goes wrong:** Messages sent over TCP are split across multiple `Read()` calls or multiple messages arrive in a single `Read()` call. The receiving node either reads partial messages (garbled data, deserialization panics) or concatenates two messages into one (corrupt data).

**Why it happens:**
- TCP is a stream protocol, not a message protocol. There is no guarantee that one `Write()` on the sender produces one `Read()` on the receiver. Nagle's algorithm may coalesce small writes; large messages may be fragmented.
- Educational projects often use a single `conn.Read(buf)` call and assume the entire message arrives at once. This works in testing on localhost (low latency, no fragmentation) but breaks under any real conditions.

**Prevention:**
- Implement length-prefix framing: send a 4-byte big-endian uint32 containing the message length, followed by the message bytes. The receiver reads 4 bytes first to know the length, then reads exactly that many bytes using `io.ReadFull`.
- Alternatively, use `encoding/gob`'s `Encoder`/`Decoder` on the connection (gob handles framing internally), but be aware of gob's other limitations.
- Add a maximum message size check (e.g., 32MB) to prevent malformed length prefixes from causing huge allocations.
- Never use newline-delimited JSON for binary data -- block data is binary and may contain delimiter bytes.

**Detection:**
- Deserialization errors that appear randomly under load but never in single-message tests.
- "unexpected EOF" or "invalid character" errors from JSON/gob decoders.
- Works perfectly in unit tests, fails when two nodes exchange messages rapidly.

**Phase relevance:** P2P networking phase. Must be the foundation of the network protocol layer before any message types are implemented.

---

### Pitfall 8: Nonce Overflow and Infinite Mining Loop

**What goes wrong:** The PoW mining loop increments a nonce looking for a valid hash, but if the nonce overflows (wraps around to 0), the miner either loops forever trying the same nonces or panics on integer overflow.

**Why it happens:**
- Using `int` or `int32` for the nonce limits the search space. With a difficulty that requires more than ~2 billion or ~4 billion attempts, the nonce wraps around and the miner enters an infinite loop.
- Even with `int64` (max ~9.2 * 10^18), extremely high difficulty could theoretically exhaust the space, though this is unlikely for educational projects.
- The mining loop often lacks any escape hatch (no timeout, no cancellation check, no extraNonce mechanism).

**Prevention:**
- Use `int64` for the nonce with a `maxNonce = math.MaxInt64` upper bound check.
- When the nonce space is exhausted, change the block timestamp or an `extraNonce` field and restart from 0. Bitcoin uses an extraNonce in the coinbase transaction for this purpose.
- Always check for context cancellation inside the mining loop (`select` on `ctx.Done()` every N iterations) so mining can be stopped when a new block arrives from a peer.
- Keep difficulty reasonable for an educational project. A target that requires millions of hashes (not billions) demonstrates the concept without causing usability problems.

**Detection:**
- Mining a block takes unexpectedly long or never completes.
- CPU pegged at 100% with no block produced.
- Nonce counter wraps to 0 or negative values in block metadata.

**Phase relevance:** PoW/mining phase. Design the mining loop with cancellation and nonce overflow handling from the start.

---

### Pitfall 9: Private Keys Stored in Plaintext

**What goes wrong:** Wallet files contain raw ECDSA private keys serialized directly to disk without any encryption. Anyone with file system access (or a backup) can steal all keys.

**Why it happens:**
- Educational projects prioritize getting things working over security. The simplest wallet implementation is `json.Marshal(privateKey)` to a file.
- Go's `crypto/ecdsa` does not provide built-in key encryption. The developer must implement or import encryption separately.
- "It is just a learning project" mindset leads to skipping encryption, but this becomes the permanent architecture.

**Prevention:**
- At minimum, encrypt wallet files with a passphrase using AES-256-GCM (or `golang.org/x/crypto/nacl/secretbox`). Derive the encryption key from the passphrase with `scrypt` or `argon2`.
- Set restrictive file permissions (`0600`) on wallet files.
- For an educational project, even a simple XOR with a password is better than plaintext (though real encryption is not much harder).
- Follow go-ethereum's keystore pattern: each key is stored in a separate JSON file, encrypted with scrypt-derived key.
- Never log or print private keys. Use a separate `String()` method on the wallet type that redacts the key.

**Detection:**
- Wallet file is human-readable and contains the raw key bytes.
- `grep` for hex-encoded private key material in the data directory.

**Phase relevance:** Wallet/key management phase. Implement encryption before storing any keys to disk.

---

### Pitfall 10: UTXO Set Not Rebuilt from Chain on Startup

**What goes wrong:** The UTXO set is maintained only in memory and lost on shutdown. Or it is persisted separately from the blockchain but gets out of sync. On restart, balances are wrong or zero.

**Why it happens:**
- The UTXO set is a derived data structure (it is computed from the blockchain). Some implementations persist it independently as an optimization but fail to keep it consistent with the chain.
- Others rebuild it on startup by replaying all blocks, but as the chain grows this becomes slow and they try to cache it, introducing another synchronization problem.

**Prevention:**
- Treat the UTXO set as a write-through cache that is always derivable from the chain. On startup, rebuild the UTXO set by replaying all blocks from genesis. This is slow for large chains but correct.
- If persisting the UTXO set for faster startup, store it in the same database as the blockchain, updated atomically within the same BoltDB write transaction that stores the new block. This guarantees consistency.
- Include a "reindex" CLI command that rebuilds the UTXO set from scratch for recovery.

**Detection:**
- Balances are zero after restarting the node.
- Balances differ between nodes even though they have the same chain.
- Transactions fail validation because the UTXO set says an output does not exist, but it is clearly in the blockchain.

**Phase relevance:** Storage/persistence phase and UTXO phase. Design UTXO persistence as part of block storage from the beginning.

---

### Pitfall 11: Signature Malleability and Verification Gaps

**What goes wrong:** Transaction signatures are not verified against the correct data, or signature malleability allows a third party to modify a transaction's ID without invalidating it.

**Why it happens:**
- ECDSA signatures have a malleability property: for any valid signature `(r, s)`, the signature `(r, N-s)` is also valid (where N is the curve order). This means a third party can change the transaction hash (which typically includes the signature) without having the private key.
- Educational projects often hash the entire transaction struct including the signature to produce the transaction ID. A malleable signature changes the hash, which can confuse transaction tracking.
- Another common mistake: signing the transaction hash but including different fields in the hash vs. what is signed, or forgetting to zero out signature fields before hashing for signing.

**Prevention:**
- Compute the transaction ID (hash) from transaction data excluding the signature. The signature is metadata that authenticates the data, not part of the signed content.
- When creating the signing hash: zero out or exclude all signature fields, serialize the transaction data, hash it, sign the hash.
- Enforce low-S signatures: after signing, if `s > N/2`, replace `s` with `N - s`. This eliminates malleability.
- Verify signatures on every transaction received from peers and in every block, not just transactions submitted locally.

**Detection:**
- Transaction IDs change after being broadcast (because peers or intermediaries modify the signature).
- The same transaction appears with two different IDs in different nodes' mempools.
- Signature verification passes for transactions the node did not create (which is correct, but verify you are checking against the right public key and data).

**Phase relevance:** Transaction/wallet phase. Define the signing/verification protocol clearly before implementing transaction propagation.

---

## Minor Pitfalls

Issues that cause confusion, inefficiency, or technical debt -- but are relatively easy to fix.

---

### Pitfall 12: Mining Blocks with Invalid or Empty Transaction Sets

**What goes wrong:** The miner creates blocks with no transactions (only the coinbase), or includes transactions that fail validation. Blocks are mined successfully but rejected by peers.

**Prevention:**
- Always include a coinbase transaction as the first transaction in every block.
- Validate every transaction against the current UTXO set before including it in a block candidate.
- After assembling the block, validate it as a whole (check for duplicate inputs across transactions within the block).
- Even if the mempool is empty, a block with just a coinbase transaction is valid and useful for testing.

---

### Pitfall 13: Genesis Block Special Cases Not Handled

**What goes wrong:** The genesis block has no previous block hash, no real transactions (except an initial coinbase), and potentially different validation rules. Code that processes blocks generically fails on the genesis block.

**Prevention:**
- Hardcode the genesis block. Do not mine it or validate it through the normal pipeline.
- Special-case genesis block handling in chain validation: skip previous-hash check, accept the hardcoded coinbase without signature verification.
- Ensure the genesis block is identical across all nodes (same hash). Generate it once and embed it as a constant.

---

### Pitfall 14: Merkle Tree with Odd Transaction Count

**What goes wrong:** The Merkle tree implementation assumes an even number of transactions. With an odd number, it either panics (index out of bounds) or produces an incorrect root hash.

**Prevention:**
- When the number of leaf nodes is odd, duplicate the last hash and pair it with itself. This is the standard Bitcoin approach.
- Write tests with 1, 2, 3, 4, and 5 transactions to cover all edge cases.
- An empty Merkle tree (0 transactions) should return a zero hash, not panic.

---

### Pitfall 15: Using Go's `encoding/gob` for Network Wire Format

**What goes wrong:** Gob is used to serialize messages sent between nodes. This locks the project into Go-only interoperability, and gob's format is not stable across Go versions, meaning nodes compiled with different Go versions may not be able to communicate.

**Prevention:**
- Use a stable, language-neutral wire format for P2P messages. For an educational project, JSON is fine for simplicity (human-readable, easy to debug). Protocol Buffers are better for performance but add complexity.
- Reserve `encoding/gob` for local storage only, if at all.
- Whatever format is chosen, version the protocol (include a version byte in every message) so nodes can negotiate compatibility.

---

### Pitfall 16: Difficulty Adjustment Based on Single Block Timestamps

**What goes wrong:** Difficulty adjusts wildly because it is calculated from only the last block's timestamp, which can be manipulated or simply noisy (a single lucky/unlucky block skews the estimate).

**Prevention:**
- Calculate difficulty adjustment over a window of N blocks (Bitcoin uses 2016 blocks). Average the time span and compare to the target.
- Clamp the adjustment factor (e.g., difficulty can at most double or halve per adjustment) to prevent oscillation.
- Validate that block timestamps are reasonable: not before the previous block's timestamp, not too far in the future.

---

### Pitfall 17: Not Using `go test -race` from Day One

**What goes wrong:** Subtle data races accumulate throughout the codebase. By the time concurrency is introduced (P2P phase), there are dozens of races that interact in complex ways. Debugging them all at once is extremely painful.

**Prevention:**
- Add `-race` to the test command from the very first test. Make it the default in CI/Makefile.
- Run the race detector during development, not just in CI. It catches races that only manifest under specific goroutine scheduling.
- Note that the race detector slows execution ~10x and uses more memory. This is acceptable for tests.

---

## Phase-Specific Warnings

| Phase Topic | Likely Pitfall | Mitigation |
|-------------|---------------|------------|
| Block/Transaction data structures | Non-deterministic serialization (Pitfall 1) | Hand-write canonical serialization for hashing. Never use maps in hashable structs. |
| UTXO model | Missing change outputs (Pitfall 3) | Invariant: `sum(outputs) == sum(inputs)` for non-coinbase transactions. |
| UTXO model | UTXO set not rebuildable (Pitfall 10) | Design UTXO set as derived state, always rebuildable from chain. |
| Proof of Work | Nonce overflow / infinite loop (Pitfall 8) | Use int64 nonce, add context cancellation, include extraNonce fallback. |
| Proof of Work | Difficulty adjustment instability (Pitfall 16) | Use multi-block window, clamp adjustment factor. |
| Wallet/keys | Plaintext key storage (Pitfall 9) | Encrypt wallet files with scrypt-derived key from the start. |
| Wallet/keys | Signature malleability (Pitfall 11) | Hash transaction data excluding signatures. Enforce low-S. |
| Storage/persistence | BoltDB deadlock (Pitfall 4) | Never mix read and write transactions on one goroutine. Use `db.Update()`. |
| Storage/persistence | UTXO set desync (Pitfall 10) | Update UTXO set in same DB transaction as block storage. |
| P2P networking | Goroutine leaks (Pitfall 6) | Context-based lifecycle for every connection. WaitGroup for cleanup. |
| P2P networking | TCP message framing (Pitfall 7) | Length-prefix framing from the start. `io.ReadFull` for reads. |
| P2P networking | Gob as wire format (Pitfall 15) | Use JSON or protobuf for wire format. Version the protocol. |
| Multi-node sync | No reorg handling (Pitfall 5) | Design UTXO set for reversibility. Implement at least simplified reorg. |
| Mining | Empty/invalid blocks (Pitfall 12) | Validate all transactions before inclusion. Always include coinbase. |
| Genesis/Init | Genesis block special cases (Pitfall 13) | Hardcode genesis. Skip normal validation for it. |
| Testing | No race detection (Pitfall 17) | `go test -race` from day one, in every package. |

---

## Sources

- [Go map non-determinism in Cosmos SDK blockchain](https://ashourics.medium.com/the-challenge-of-gos-map-iteration-in-the-cosmos-sdk-blockchain-a-dive-into-determinism-bd5a99260519) - MEDIUM confidence
- [Avoiding non-determinism in Go workflows](https://docs.chain.link/cre/concepts/non-determinism-go) - MEDIUM confidence
- [BoltDB deadlock with mixed read/write transactions](https://github.com/boltdb/bolt/issues/378) - HIGH confidence (official issue tracker)
- [BoltDB concurrent writes and deadlocks](https://github.com/boltdb/bolt/issues/739) - HIGH confidence (official issue tracker)
- [BoltDB transaction concurrency clarification](https://github.com/boltdb/bolt/issues/392) - HIGH confidence (official issue tracker)
- [Go-ethereum mining progress attack via NewBlockEvent reset](https://github.com/ethereum/go-ethereum/issues/274) - HIGH confidence (official issue tracker)
- [Go race detector documentation](https://go.dev/doc/articles/race_detector) - HIGH confidence (official docs)
- [Go-ethereum keystore implementation](https://pkg.go.dev/github.com/ethereum/go-ethereum/accounts/keystore) - HIGH confidence (official package docs)
- [Go ECDSA signature verification after signing](https://github.com/golang/go/issues/54681) - HIGH confidence (official issue tracker)
- [Building Blockchain in Go tutorial series](https://jeiwan.net/posts/building-blockchain-in-go-part-3/) - MEDIUM confidence
- [btcd mempool implementation](https://github.com/btcsuite/btcd/blob/master/mempool/mempool.go) - HIGH confidence (reference implementation)
- [UTXO lifecycle and debugging](https://medium.com/@aaayushhh/under-the-hood-of-bitcoin-utxo-lifecycle-script-execution-and-practical-debugging-d724cc63c46b) - MEDIUM confidence
- [Length-prefix message framing in Go](https://ogu.nz/binproto.html) - MEDIUM confidence
- [Blockchain reorganization handling](https://learnmeabitcoin.com/technical/blockchain/chain-reorganization/) - MEDIUM confidence
- [Goroutine leak prevention patterns](https://dev.to/serifcolakel/go-concurrency-mastery-preventing-goroutine-leaks-with-context-timeout-cancellation-best-1lg0) - MEDIUM confidence
- [Coinbase transaction mechanics](https://learnmeabitcoin.com/technical/mining/coinbase-transaction/) - MEDIUM confidence
- [Concurrency safety: race conditions, mutex, and deadlocks in Go](https://samueltuoyo9082.medium.com/concurrency-safety-in-go-race-conditions-mutex-and-deadlocks-no-theory-just-practice-cce2c4caa22f) - MEDIUM confidence
