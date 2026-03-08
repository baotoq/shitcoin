---
phase: 17-handler-layer-tests
plan: 02
subsystem: testing
tags: [websocket, gorilla-websocket, httptest, require-eventually, integration-test]

requires:
  - phase: 14-testutil-foundation
    provides: testutil helpers and mock patterns
provides:
  - WebSocket handler integration tests via httptest.Server + websocket.Dial
  - Reliable hub tests using require.Eventually instead of time.Sleep
  - 84.0% WebSocket package coverage
affects: [18-race-detection]

tech-stack:
  added: []
  patterns: [httptest.Server + websocket.DefaultDialer for WS integration tests, require.Eventually for async hub assertions]

key-files:
  created: [internal/handler/ws/handler_test.go]
  modified: [internal/handler/ws/hub_test.go]

key-decisions:
  - "writePump batches queued messages with newline separators -- EventBusIntegration test splits on newline"
  - "Hub subscribeEventBus goroutine may not be ready for Publish -- use retry-publish goroutine in ForwardsEventBusEventsAsJSON test"

patterns-established:
  - "WebSocket integration test pattern: httptest.NewServer(ServeWs(hub)) + websocket.DefaultDialer.Dial"
  - "waitForClients helper using require.Eventually with hub.mu.RLock for client count assertions"

requirements-completed: [HNDL-02]

duration: 2min
completed: 2026-03-08
---

# Phase 17 Plan 02: WebSocket Handler Tests Summary

**WebSocket ServeWs integration tests with httptest.Server achieving 84.0% package coverage using require.Eventually**

## Performance

- **Duration:** 2 min
- **Started:** 2026-03-08T04:51:26Z
- **Completed:** 2026-03-08T04:53:42Z
- **Tasks:** 2
- **Files modified:** 2

## Accomplishments
- Created 4 ServeWs integration tests exercising real WebSocket connections via httptest.Server
- Replaced all time.Sleep calls in hub_test.go with require.Eventually assertions
- WebSocket package coverage at 84.0% (target was 75%+)

## Task Commits

Each task was committed atomically:

1. **Task 1: Create ServeWs integration tests with httptest.Server** - `6672925` (test)
2. **Task 2: Update hub_test.go to use require.Eventually and verify 75%+ coverage** - `5b102b6` (test)

## Files Created/Modified
- `internal/handler/ws/handler_test.go` - ServeWs integration tests: broadcast, disconnect, multi-client, event bus
- `internal/handler/ws/hub_test.go` - Replaced time.Sleep with require.Eventually in all hub tests

## Decisions Made
- writePump batches queued messages with newline separators, so EventBusIntegration test splits received frames on newlines to extract individual JSON messages
- Hub's subscribeEventBus goroutine may not have called bus.Subscribe() when Publish is called -- ForwardsEventBusEventsAsJSON uses a retry-publish goroutine to handle this race

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Fixed EventBusIntegration test for batched WebSocket messages**
- **Found during:** Task 1 (ServeWs integration tests)
- **Issue:** writePump batches queued messages into a single WebSocket frame with newline separators, causing json.Unmarshal to fail on the combined payload
- **Fix:** Split received message on newlines before unmarshaling individual JSON objects
- **Files modified:** internal/handler/ws/handler_test.go
- **Verification:** TestServeWs_EventBusIntegration passes consistently
- **Committed in:** 6672925 (Task 1 commit)

**2. [Rule 1 - Bug] Fixed ForwardsEventBusEventsAsJSON race condition**
- **Found during:** Task 2 (hub_test.go updates)
- **Issue:** Hub's subscribeEventBus goroutine may not have called bus.Subscribe() before test calls bus.Publish(), causing the event to be dropped
- **Fix:** Replaced single Publish + select/timeout with retry-publish goroutine + require.Eventually
- **Files modified:** internal/handler/ws/hub_test.go
- **Verification:** TestHub_ForwardsEventBusEventsAsJSON passes consistently
- **Committed in:** 5b102b6 (Task 2 commit)

---

**Total deviations:** 2 auto-fixed (2 bugs)
**Impact on plan:** Both fixes necessary for test correctness. No scope creep.

## Issues Encountered
None beyond the auto-fixed deviations above.

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- WebSocket handler tests complete with 84.0% coverage
- All handler layer tests (API + WebSocket) ready for Phase 18 race detection

---
*Phase: 17-handler-layer-tests*
*Completed: 2026-03-08*
