# Phase 1: Core Chain Foundation - Context

**Gathered:** 2026-03-05
**Status:** Ready for planning

<domain>
## Phase Boundary

A node can create, mine, and persist blocks with correct deterministic hashing and adjustable difficulty. Covers genesis block creation, block structure, SHA-256d hashing, PoW mining loop, difficulty adjustment, configurable consensus parameters, and persistent storage. No transactions, no networking, no CLI beyond a minimal runner.

Requirements: MINE-01, MINE-02, MINE-03, MINE-06, MINE-09

</domain>

<decisions>
## Implementation Decisions

### Storage Engine
- BoltDB (bbolt) for persistent chain storage
- Single-file B+tree — simple, read-optimized, easy to reason about
- Battle-tested (used in etcd)

### Serialization Format
- JSON for block encoding (hashing and storage)
- Prioritizes debuggability and readability over compactness
- Deterministic serialization must be enforced (sorted keys, no floating point ambiguity)

### Framework & Configuration
- go-zero framework for project structure and configuration
- Consensus parameters configured via go-zero's YAML config + struct binding pattern
- Configurable without code changes: block time target, difficulty adjustment interval, initial difficulty

### Code Organization
- go-zero project structure conventions (handler → logic → model layers)
- Tactical DDD approach: Entities, Value Objects, Aggregates, Repositories, Domain Services
- Domain logic stays clean and separate from framework plumbing
- Block and Chain as domain entities, not tied to storage or transport

### Local Development
- Tilt.dev for local dev environment orchestration

### Claude's Discretion
- BoltDB bucket layout and key design
- Block struct field ordering
- Difficulty adjustment algorithm specifics (window size, clamping bounds)
- Genesis block default embedded message
- Error handling patterns
- Package naming within go-zero + DDD structure

</decisions>

<specifics>
## Specific Ideas

- Tactical DDD layering within go-zero: domain types (Block, Chain) as pure Go structs with behavior, repositories as interfaces, go-zero logic layer orchestrates domain operations
- JSON serialization chosen for educational transparency — easy to inspect stored blocks

</specifics>

<code_context>
## Existing Code Insights

### Reusable Assets
- None — greenfield project, no existing code beyond placeholder

### Established Patterns
- .editorconfig configured: tabs for Go files, spaces for others
- go-zero MCP tools available for scaffolding API and RPC services

### Integration Points
- go.mod needs to be initialized
- go-zero project structure to be scaffolded as part of this phase
- Tilt.dev configuration to be set up for local dev workflow

</code_context>

<deferred>
## Deferred Ideas

None — discussion stayed within phase scope

</deferred>

---

*Phase: 01-core-chain-foundation*
*Context gathered: 2026-03-05*
