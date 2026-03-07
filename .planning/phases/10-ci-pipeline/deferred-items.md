# Deferred Items - Phase 10

## Pre-existing Frontend Lint Errors

Found during 10-02 Task 1 verification. These are pre-existing issues in web/ source files, not caused by CI workflow changes.

- `react-refresh/only-export-components` in badge.tsx, button.tsx, tabs.tsx (shadcn/ui components)
- `react-hooks/set-state-in-effect` in Address.tsx, BlockDetail.tsx, BlockExplorer.tsx, Mining.tsx, TxDetail.tsx
- `react-hooks/immutability` in useWebSocket.ts (connect accessed before declaration)

**Impact:** Frontend CI `npm run lint` step will fail until these are fixed.
**Recommendation:** Fix in a separate PR or add eslint-disable comments for shadcn/ui generated files.
