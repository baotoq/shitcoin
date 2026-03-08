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

## Milestone: v1.2 — Testing & Quality

**Shipped:** 2026-03-08
**Phases:** 5 | **Plans:** 11

### What Was Built
- Shared testutil package with 5 builders and 3 mock repositories (1,195 LOC)
- Domain layer tests: chain 85.4%, P2P 80.1%, tx/utxo/mempool 100%, wallet 97.8%
- Infrastructure persistence tests: BoltDB 86.3%, JSON wallet 92.5%
- Handler layer tests: API 93.5%, WebSocket hub 84.0%
- P2P multi-node integration tests (handshake, block sync, tx relay)
- E2E chain scenario tests (wallet-to-balance, multi-block mining, mempool cycle)
- Fixed ws.Hub broadcast data race, enabled `-race` flag in CI

### What Worked
- Shared testutil-first approach (Phase 14) paid off — every subsequent phase imported builders/mocks instead of duplicating
- Error injection via exported mock fields (SaveBlockWithUTXOsErr, GetChainHeightErr) was simple and extensible
- `require.Eventually` for async P2P assertions eliminated flaky time.Sleep patterns
- All 5 v1.2 plans in parallel waves — no phase blocked on another (15/16/17 ran in parallel after 14)
- Research phase caught the ws.Hub race condition before it blocked CI integration

### What Was Inefficient
- Nyquist validation frontmatter still not flipped to `nyquist_compliant: true` post-execution (same issue as v1.0)
- ROADMAP.md plan checkboxes for Phase 18 not auto-checked (manually fixed during milestone completion)
- Some SUMMARY one-liner fields were empty — extraction tool didn't find them

### Patterns Established
- Error injection via exported error fields on mock repos (no framework needed)
- `require.Eventually` as project standard for async test assertions
- OS-assigned port 0 for all network tests (avoids CI conflicts)
- Two-phase lock pattern: collect under RLock, mutate under Lock
- External test packages (`package foo_test`) for better testutil import hygiene
- Permission-based error injection with `t.Cleanup` restore for file I/O tests

### Key Lessons
1. Investing in shared test infrastructure before writing tests dramatically reduces duplication — Phase 14's testutil was imported by 16 files across 4 phases
2. Running `-race` early in research reveals real races — the hub race would have been a CI blocker if discovered late
3. Integration tests with in-process servers and OS-assigned ports work reliably in CI without special configuration
4. Mock repos should return domain sentinel errors (not generic errors) for `errors.Is` compatibility

### Cost Observations
- Model mix: ~70% opus (execution), ~30% sonnet (verification/checking)
- Notable: 5 phases, 11 plans completed in 1 calendar day with auto-advance pipeline

---

## Cross-Milestone Trends

### Process Evolution

| Milestone | Phases | Plans | Key Change |
|-----------|--------|-------|------------|
| v1.0 | 8 | 22 | Established DDD + go-zero patterns, wave parallelization |
| v1.1 | 5 | 9 | CI/CD + K8s infrastructure, GitOps deployment |
| v1.2 | 5 | 11 | Test-first infrastructure, shared testutil, race detection in CI |

### Cumulative Quality

| Milestone | Test Packages | Verification Score | Tech Debt Items |
|-----------|--------------|-------------------|-----------------|
| v1.0 | 15 green | 51/51 must-haves | 3 minor |
| v1.2 | 12 green | 40/40 must-haves | 2 minor (orphaned testutil helpers) |

### Top Lessons (Verified Across Milestones)

1. Plan verification catches structural gaps before execution — always run the checker
2. Wave-based parallel execution significantly reduces wall-clock time for independent plans
3. Shared infrastructure (testutil, CI) should be its own phase before consuming phases
4. Auto-advance pipeline (`--auto`) enables full plan→execute→verify→next without manual intervention
5. Research phase catches production issues (races, API gaps) before they become execution blockers
