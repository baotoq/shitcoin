# Project Retrospective

*A living document updated after each milestone. Lessons feed forward into future planning.*

## Milestone: v1.0 — Educational Blockchain

**Shipped:** 2026-03-07
**Phases:** 8 | **Plans:** 22

### What Was Built
- Complete blockchain with PoW mining, SHA-256d, difficulty adjustment, Merkle trees
- UTXO transaction system with ECDSA signing, Base58Check addresses, undo-log
- TCP P2P networking with handshake, block/tx relay, IBD sync, chain reorganization
- React web dashboard with block explorer, mining visualizer, live WebSocket updates
- Educational demos: halving, fee-prioritized mining, testnet orchestration, double-spend
- 9-command CLI covering all node operations

### What Worked
- DDD with go-zero ServiceContext pattern kept dependencies clean and testable
- Wave-based parallel execution (plans 02+03, 01+02 in phase 6) saved significant time
- TDD tasks produced reliable code — tests caught real issues during implementation
- Breaking block txs into `[]any` to avoid circular imports was the right tradeoff
- Phase 4.1 (testify migration) and 5.1 (Go upgrade) as decimal phases kept scope focused

### What Was Inefficient
- Phase 5 initial planning missed the React frontend entirely (3 backend plans, 0 frontend) — caught by plan checker
- Some ROADMAP.md plan checkboxes fell out of sync with actual completion state
- Nyquist validation frontmatter (`nyquist_compliant`) not flipped to `true` after execution in 7/8 phases

### Patterns Established
- `[]any` for block transactions to break import cycles — type assertions at chain/handler level
- Per-node data isolation via `data/node-{port}/` directories
- Event bus pattern for decoupling mining/P2P from WebSocket broadcasting
- Length-prefixed binary wire format: `[4-byte length][1-byte command][JSON payload]`

### Key Lessons
1. Plan checker is essential — it caught a fundamental gap (missing frontend plans) that would have been discovered late
2. Decimal phases (4.1, 5.1) work well for inserting non-feature work without disrupting the main sequence
3. In-process demos (double-spend) are more reliable than subprocess orchestration for scripted scenarios
4. Auto-advance (`--auto`) enables full plan→execute→verify pipeline without manual intervention

### Cost Observations
- Model mix: ~70% opus (execution), ~30% sonnet (verification/checking)
- Notable: 8 phases completed in 3 calendar days with auto-advance pipeline

---

## Cross-Milestone Trends

### Process Evolution

| Milestone | Phases | Plans | Key Change |
|-----------|--------|-------|------------|
| v1.0 | 8 | 22 | Established DDD + go-zero patterns, wave parallelization |

### Cumulative Quality

| Milestone | Test Packages | Verification Score | Tech Debt Items |
|-----------|--------------|-------------------|-----------------|
| v1.0 | 15 green | 51/51 must-haves | 3 minor |

### Top Lessons (Verified Across Milestones)

1. Plan verification catches structural gaps before execution — always run the checker
2. Wave-based parallel execution significantly reduces wall-clock time for independent plans
