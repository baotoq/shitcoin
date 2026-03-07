# Web Block Explorer

React 19 + TypeScript block explorer for shitcoin nodes.

## Commands

```bash
npm install        # Install dependencies
npm run dev        # Vite dev server on :5173 (proxies /api, /ws to :8080)
npm run build      # TypeScript check + production build to dist/
npm run lint       # ESLint
npm run preview    # Preview production build
```

## Architecture

- **Pages** (`src/pages/`): Route components — Dashboard, BlockExplorer, BlockDetail, TxDetail, Mempool, Mining, Address
- **Components** (`src/components/`): Layout (sidebar + outlet), BlockCard, TxTable, MiningVisualizer, StatusBar, SearchBar
- **UI primitives** (`src/components/ui/`): shadcn/ui components (style: `base-nova`, icon lib: lucide)
- **Hooks** (`src/hooks/`): `useWebSocket` (auto-reconnect with exponential backoff), `useNodeStatus` (polls + WS)
- **API client** (`src/lib/api.ts`): Typed fetch wrappers for all REST endpoints
- **Types** (`src/types/api.ts`): Shared TypeScript interfaces mirroring Go API response structs

## Conventions

- Path alias: `@/` maps to `src/` (configured in `tsconfig.json` and `vite.config.ts`)
- Add shadcn/ui components: `npx shadcn@latest add <component>` (config in `components.json`)
- Dark theme only: root `<div>` has `className="dark"`, uses zinc color palette throughout
- All API data fetched via `src/lib/api.ts` — never call `fetch()` directly in components
- WebSocket messages typed as `WSMessage` (`{ type: string, payload: unknown }`)
- Vite proxies `/api` → `http://localhost:8080` and `/ws` → `ws://localhost:8080` (see `vite.config.ts`)
- Amounts displayed in coins (satoshis / 100,000,000), formatted to 8 decimal places
- Hashes truncated in lists (12-16 chars + `...`), shown full with `break-all` in detail views
