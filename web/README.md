# Shitcoin Block Explorer

A real-time block explorer web UI for shitcoin nodes. Browse blocks, inspect transactions, watch mining in real-time, view address balances, and search the chain — all updating live via WebSocket.

## Key Features

- **Dashboard** with chain height, peer count, mempool size, and mining status
- **Block Explorer** with paginated browsing and block detail views (header, transactions, navigation)
- **Transaction Inspector** showing inputs, outputs, coinbase detection, and linked addresses
- **Mining Visualizer** displaying live nonce attempts, hash vs. target comparison, and leading-zero highlighting
- **Mempool View** with real-time updates when transactions enter or get mined
- **Address Lookup** showing balance and UTXO table
- **Universal Search** by block hash, block height, transaction hash, or address

## Tech Stack

- **Framework**: React 19 with TypeScript
- **Build Tool**: Vite 7
- **Styling**: Tailwind CSS 4 with CSS variables (OKLCH color space)
- **Component Library**: shadcn/ui (`base-nova` style, lucide icons)
- **Routing**: React Router 7 (nested routes with layout)
- **Font**: Geist Variable (via @fontsource)
- **Real-time**: WebSocket with auto-reconnect (exponential backoff + jitter)

## Prerequisites

- **Node.js 20+** and **npm**
- A running shitcoin node with HTTP server (default: `localhost:8080`)

## Getting Started

### 1. Install Dependencies

```bash
npm install
```

### 2. Start a Shitcoin Node

In a separate terminal, start a node with the HTTP server:

```bash
# From the project root
go run cmd/shitcoin/main.go startnode -port 3000 -http-port 8080 -mine YOUR_ADDRESS
```

Or use the testnet command to launch multiple nodes:

```bash
go run cmd/shitcoin/main.go testnet -nodes 3 -base-port 3000 -base-http-port 8080
```

### 3. Start the Dev Server

```bash
npm run dev
```

Open [http://localhost:5173](http://localhost:5173) in your browser.

The Vite dev server proxies `/api/*` and `/ws` requests to the Go backend at `http://localhost:8080` (configured in `vite.config.ts`).

## Available Scripts

| Command | Description |
|---------|-------------|
| `npm run dev` | Start Vite dev server on port 5173 with HMR |
| `npm run build` | TypeScript type-check + Vite production build to `dist/` |
| `npm run lint` | Run ESLint on all `.ts` and `.tsx` files |
| `npm run preview` | Preview the production build locally |

## Architecture

### Directory Structure

```
web/
├── public/                      # Static assets
├── src/
│   ├── main.tsx                 # Entry point (React root + StrictMode)
│   ├── App.tsx                  # Router setup (BrowserRouter with nested routes)
│   ├── index.css                # Tailwind imports + CSS variables (light/dark themes)
│   ├── app.css                  # Additional app styles
│   ├── pages/
│   │   ├── Dashboard.tsx        # Chain stats cards + recent blocks table
│   │   ├── BlockExplorer.tsx    # Paginated block list with live updates
│   │   ├── BlockDetail.tsx      # Block header fields + transaction list
│   │   ├── TxDetail.tsx         # Transaction inputs/outputs with coinbase detection
│   │   ├── Mempool.tsx          # Pending transactions with live refresh
│   │   ├── Mining.tsx           # Mining status + MiningVisualizer
│   │   └── Address.tsx          # Balance + UTXO table for an address
│   ├── components/
│   │   ├── Layout.tsx           # Sidebar nav + StatusBar + SearchBar + Outlet
│   │   ├── StatusBar.tsx        # Top bar: chain height, peers, mempool, mining indicator
│   │   ├── SearchBar.tsx        # Universal search (hash, height, address)
│   │   ├── BlockCard.tsx        # Compact block summary card
│   │   ├── TxTable.tsx          # Reusable transaction table
│   │   ├── MiningVisualizer.tsx # Live nonce/hash/target display with leading-zero highlighting
│   │   └── ui/                  # shadcn/ui primitives (button, card, table, etc.)
│   ├── hooks/
│   │   ├── useWebSocket.ts     # WebSocket connection with auto-reconnect
│   │   └── useNodeStatus.ts    # Polls /api/status + merges WebSocket events
│   ├── lib/
│   │   ├── api.ts              # Typed fetch wrappers for all REST endpoints
│   │   └── utils.ts            # cn() utility for Tailwind class merging
│   └── types/
│       └── api.ts              # TypeScript interfaces matching Go API responses
├── components.json              # shadcn/ui config (base-nova style, lucide icons)
├── vite.config.ts               # Vite config with React plugin + API/WS proxy
├── tsconfig.json                # TypeScript config with @/ path alias
└── eslint.config.js             # ESLint flat config
```

### Data Flow

```
Go Backend (:8080)
  ├── REST API (/api/*)  ──>  src/lib/api.ts  ──>  Pages (fetch on mount/navigate)
  └── WebSocket (/ws)    ──>  useWebSocket()  ──>  Pages (real-time updates)
                                                     └── useNodeStatus() (status bar)
```

1. **Initial data**: Pages call typed fetch functions from `src/lib/api.ts` on mount
2. **Real-time updates**: `useWebSocket` hook connects to `/ws`, parses JSON messages into `WSMessage` objects
3. **State merging**: `useNodeStatus` combines polling (`/api/status` every 10s) with WebSocket events to keep the status bar current
4. **Auto-reconnect**: WebSocket reconnects with exponential backoff (1s base, 30s max) + random jitter

### Routes

| Path | Component | Data Source |
|------|-----------|-------------|
| `/` | `Dashboard` | `fetchBlocks`, `useNodeStatus`, WebSocket `new_block` |
| `/blocks` | `BlockExplorer` | `fetchBlocks` (paginated), WebSocket `new_block` |
| `/blocks/:height` | `BlockDetail` | `fetchBlockByHeight` |
| `/tx/:hash` | `TxDetail` | `fetchTx` |
| `/mempool` | `Mempool` | `fetchMempool`, WebSocket `mempool_changed`/`new_tx`/`new_block` |
| `/mining` | `Mining` | `fetchStatus`, WebSocket `mining_started`/`mining_progress`/`mining_stopped` |
| `/address/:addr` | `Address` | `fetchAddress` |

### REST API Endpoints (Go Backend)

The frontend consumes these endpoints (all proxied through Vite in dev):

| Method | Path | Returns |
|--------|------|---------|
| `GET` | `/api/status` | `StatusResponse` — chain height, peers, mempool, mining |
| `GET` | `/api/blocks?page=N&limit=N` | `BlockListResponse` — paginated blocks |
| `GET` | `/api/blocks/:height` | `BlockModel` — single block by height |
| `GET` | `/api/blocks/hash/:hash` | `BlockModel` — single block by hash |
| `GET` | `/api/tx/:hash` | `TxModel` — transaction by hash |
| `GET` | `/api/mempool` | `TxModel[]` — pending transactions |
| `GET` | `/api/address/:addr` | `AddressResponse` — balance + UTXOs |
| `GET` | `/api/search?q=QUERY` | `SearchResult` — block/tx/address lookup |

### WebSocket Events

All events arrive as `{ type: string, payload: unknown }`:

| Event | Trigger | UI Effect |
|-------|---------|-----------|
| `new_block` | Block mined/received | Dashboard refreshes blocks, BlockExplorer reloads page 1, Mempool refetches |
| `new_tx` | Transaction enters mempool | Mempool page refetches |
| `mempool_changed` | Mempool count changed | Mempool refetches, StatusBar updates count |
| `mining_started` | Mining begins | Mining page shows visualizer, StatusBar shows "Mining" |
| `mining_progress` | Hash attempt | Mining page updates nonce/hash/target display |
| `mining_stopped` | Block found or cancelled | Mining page shows last mined block, StatusBar shows "Idle" |
| `peer_connected` | P2P peer connects | StatusBar increments peer count |
| `peer_disconnected` | P2P peer disconnects | StatusBar decrements peer count |
| `status` | Full status update | StatusBar replaces all fields |

### TypeScript Interfaces

All API response types are defined in `src/types/api.ts` and mirror the Go backend's JSON output:

- `StatusResponse` — chain height, latest block hash, mempool size, peer count, mining status
- `BlockListResponse` — paginated block list with total count
- `BlockModel` / `HeaderModel` — block with full header fields
- `TxModel` / `TxInputModel` / `TxOutputModel` — transaction with inputs and outputs
- `AddressResponse` / `UTXOModel` — address balance and unspent outputs
- `SearchResult` — polymorphic search result (block, tx, or address)
- `WSMessage` / `MiningProgressPayload` — WebSocket message types

## UI Conventions

- **Dark theme only**: The root `<div>` has `className="dark"`. All colors use the zinc palette.
- **Color system**: CSS variables in OKLCH color space defined in `src/index.css`, consumed via Tailwind's `@theme inline` directive.
- **Font**: Geist Variable loaded via `@fontsource-variable/geist`.
- **Component style**: shadcn/ui `base-nova` style with `neutral` base color.
- **Path alias**: `@/` maps to `src/` (configured in both `tsconfig.json` and `vite.config.ts`).
- **Amounts**: Displayed in coins (satoshis / 100,000,000), formatted to 8 decimal places.
- **Hashes**: Truncated to 12-16 characters with `...` in lists; shown full with `break-all` in detail views.
- **Loading states**: Pulse-animated zinc placeholder blocks matching the layout shape.

## Adding shadcn/ui Components

```bash
npx shadcn@latest add <component-name>
```

Components are installed to `src/components/ui/`. Configuration is in `components.json`.

Currently installed: `badge`, `button`, `card`, `input`, `scroll-area`, `separator`, `table`, `tabs`.

## Production Deployment

```bash
npm run build
```

This outputs static files to `dist/`. Serve them behind a reverse proxy (e.g., nginx, Caddy) that:

1. Serves `dist/` files for all non-API routes
2. Proxies `/api/*` requests to the Go backend
3. Proxies `/ws` WebSocket connections to the Go backend

Example nginx config:

```nginx
server {
    listen 80;

    location / {
        root /path/to/web/dist;
        try_files $uri $uri/ /index.html;
    }

    location /api/ {
        proxy_pass http://localhost:8080;
    }

    location /ws {
        proxy_pass http://localhost:8080;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";
    }
}
```

## Troubleshooting

### "Loading..." or blank page

The backend is not running or not reachable. Start a node:

```bash
go run cmd/shitcoin/main.go startnode -port 3000 -http-port 8080 -mine YOUR_ADDRESS
```

### WebSocket not connecting

Check that the Vite proxy is configured correctly in `vite.config.ts`. The default proxies to `localhost:8080`. If your backend runs on a different port, update the proxy target.

### Mining page shows "Mining is idle"

Start a node with the `-mine` flag to enable auto-mining. Without it, the node runs idle and no mining events are emitted.

### API type mismatches

If the Go backend changes its response format, update the corresponding interfaces in `src/types/api.ts`. These types are the single source of truth for the frontend's API contract.
