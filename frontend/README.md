# QueueLite frontend

React 19, TypeScript and Vite 7 power the customer and business interfaces. All application pages use the backend API.

## Development

Use Node.js 22.12 or later. From this directory:

```sh
npm ci
npm run dev
```

Start the Go API, PostgreSQL and Redis as described in the [project README](../README.md). Vite normally serves `http://127.0.0.1:5173` and forwards `/api` requests to the Go API without the prefix. Keep a consistent hostname for cookie authentication.

The defaults require no frontend environment file. To override them, copy `.env.example` to `.env.local`:

| Variable | Purpose | Default |
| --- | --- | --- |
| `VITE_API_BASE_URL` | Browser-visible API base URL | `/api` |
| `API_PROXY_TARGET` | API origin used by the Vite development server | `http://127.0.0.1:8080` |

Never place secrets in `VITE_*` variables; they are included in browser code. API requests include cookies. Production hosting needs SPA history fallback and a same-origin `/api` reverse proxy that removes the prefix. `npm run preview` serves the production build without an API proxy. A separate API origin requires backend CORS and compatible cookie settings.

## Commands

| Command | Purpose |
| --- | --- |
| `npm run dev` | Start the development server |
| `npm run typecheck` | Check TypeScript without emitting files |
| `npm test` | Run API, form, response-validation and Vite proxy tests |
| `npm run test:browser` | Run Playwright browser tests |
| `npm run build` | Check TypeScript and generate `dist/` |
| `npm run preview` | Serve `dist/` locally |

Install Chromium for browser tests with `npx playwright install chromium`. Alternatively, set `PLAYWRIGHT_CHANNEL` to `msedge` or `chrome` to use an installed browser. For example, in PowerShell:

```powershell
$env:PLAYWRIGHT_CHANNEL = 'msedge'
npm run test:browser
```

Browser tests intercept API responses to cover customer tickets, discovery, account access, business operations, failed requests, navigation and responsive layouts. The proxy test uses a local HTTP fixture to verify path and cookie forwarding. These tests do not verify live backend integration. Screenshots are written to ignored `output/playwright/`.

## Structure

- `src/api/`: typed route wrappers, response validation and error handling. The shared client includes cookies, supports cancellation, applies a 20-second deadline and does not retry mutations.
- `src/pages/`: discovery, joining, ticket status, account, subscription and business operation pages with their form and resource helpers.
- `src/components/`: shared controls, cards, access feedback and navigation.
- `src/design/`: design tokens and application styles, including focus and reduced-motion support.
- `src/state/`: observed session state, selected business and locally known ticket context. Tokens are not persisted in browser storage.
- `src/routes.ts` and `src/App.tsx`: route metadata and page routing.

Import HTTP DTOs from `src/api/types.ts` and shared controls from `src/components/foundation.tsx`. Use API services for requests, cancel stale reads and keep form state local. Handle domain failures through `APIError.code`; a queue-cookie authorization failure does not necessarily invalidate the account session. [API error handling](docs/backend-errors.md) describes the server response format.

## Behavior

Customers can browse businesses as guests or account holders, join queues, track owned tickets and cancel waiting tickets. Active ticket pages poll every five seconds, pause while hidden and stop for terminal states. Account queue lists refresh manually; newly admitted tickets can appear after the background persistence worker runs.

Owners and admins can manage business settings and counters. Assigned staff can use authorized counter actions to call, process, skip and complete tickets. Plans pages show authorized subscription details; billing and quota administration are unavailable to ordinary account sessions. Daily business hours do not determine timezone-aware opening status.
