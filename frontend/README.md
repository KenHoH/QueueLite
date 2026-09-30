# QueueLite — Stage 3B customer queues

React, TypeScript, Vite and React Router. Login/Register, Home discovery, Business Detail, Join/Success, Queue Ticket and My Queues use real backend services. Later-stage routes retain temporary placeholders. Production pages use no mock business or queue data.

## Run

Use Node 22.12+ (or compatible newer Node). From `frontend/`:

```sh
npm ci
npm run dev
npm run typecheck
npm test
npm run build
```

The defaults need no frontend environment file. Copy `.env.example` to `.env.local` only to override settings. `VITE_API_BASE_URL=/api` is browser-visible; `API_PROXY_TARGET=http://127.0.0.1:8080` is Vite-server-only. Do not put secrets in VITE variables. Keep the same frontend hostname throughout a session for cookies.

The Go backend defaults to port 8080 and native paths such as `/users/login`; Vite forwards `/api/users/login` to it and preserves cookies. Backend setup requires PostgreSQL, Redis and DATABASE_URL, SECRET, REDIS_HOST; HTTP_ADDR is optional. Backend startup/migrations were not changed. Review the backend contract before live integration.

`npm run preview` serves the production artifact only; it does not proxy the API. Production hosting must provide SPA history fallback and a same-origin `/api` reverse proxy that strips the prefix, or an explicitly configured API origin with appropriate backend CORS/cookie support. Current backend cookie security settings need production review.

The existing Windows literal-tilde path guard is preserved. It restricts Vite file serving to the frontend and blocks hidden/certificate files when replacing Vite's broad Windows short-name guard.

## Architecture

- `src/api/client.ts`: one fetch client, credentials included, JSON and typed errors, AbortSignal support. No mutation retries.
- `src/api/{auth,users,businesses,queues,counters,subscriptions}.ts`: real route wrappers, including authenticated `GET /queues/me`.
- `src/api/types.ts`: HTTP DTOs and request types. Optional response keys are omitted; timestamps remain strings. Database models are not response schemas.
- `src/api/errors.ts`: APIError with status/code/kind, plain-text middleware fallback, queue domain code groups.
- `src/design/tokens.css`: centralized palette, typography, spacing, dimensions and radii. `foundation.css` styles the active app. Inter uses local availability, with Segoe UI/system fallbacks; no external font request.
- `src/components/foundation.tsx`: Button, Input, PasswordInput, Card, StatusBadge, PageContainer, LoadingSkeleton, EmptyState, InlineError, Spinner. Inputs have labels/descriptions/errors; buttons default to type=button; icons are decorative; focus and reduced motion are supported.
- `src/routes.ts`, `src/App.tsx`: route metadata, real customer pages and later-stage placeholders. Guest browsing and cookie-owned tickets are supported.
- `src/components/CustomerHeader.tsx`, `BusinessCard.tsx`: Home/Businesses navigation for guests, Home/My Queues navigation for accounts, and real business cards.
- `src/pages/useBusinessDiscovery.ts`: list/search fetching, 300ms debounce, AbortController cancellation and cursor Load more. Business Detail uses its UUID route parameter with loading/not-found/retry states.
- `src/pages/{JoinQueue,QueueStatus,MyQueues}.tsx`: account/guest join, inline success, owned ticket reads, confirmed cancellation via state PATCH and active account queues. `useQueueTicket.ts` polls active tickets every five seconds, pauses hidden tabs and stops at terminal states/unmount. My Queues has manual refresh; recently accepted queues appear after worker persistence.
- `src/state/AppState.tsx`: lightweight context for unknown/observed auth, locally known ticket and join mode, selected business. No token persistence. Selected business is UI context, not authorization.

Future pages should import API types from `src/api/types.ts` and components from `src/components/foundation.tsx`. Keep form/loading/error state local, use services rather than fetch in components, and cancel stale reads with AbortController. Inspect APIError.code for domain failures; a queue-cookie 401 must not automatically log out the account. Login loads real identity through `/users/me`. Do not infer identity from a fetched arbitrary profile.

## Legacy prototype references

Remaining prototype staff pages, `src/data/mock.ts`, `src/types/index.ts`, `src/state/QueueContext.tsx`, `src/components/{Layout,QueueTable,ui}.tsx` and `src/styles.css` are retained but disconnected from the entry point and production bundle. They contain fictional businesses, fields and behavior. Customer pages have been replaced by real pages. Do not import legacy references into new work. Their mock queue scheduling and wait times are not backend behavior.

## Integration reference

Read [backend-contract.md](docs/backend-contract.md) for every route, exact request/response shapes, authentication, cookies, semantics, confirmed gaps, evidence and integration order. [backend-errors.md](docs/backend-errors.md) indexes source error codes.

Stages 1–3A provide current-user discovery/logout, business mapping and customer join/ownership/UUID contracts. Stage 3B connects the complete customer queue experience and adds `GET /queues/me`. Stage 4 still needs lifecycle timestamp/schema verification, worker recovery and staff role authorization. Business counter listing, SSE frontend and Priority behavior remain deferred. Business hours do not imply timezone-aware opening status.

## Verification

`npm test` checks credential/JSON/query handling, backend error variants, cancellation, network failures, malformed success responses and subscription business rejections. A local HTTP fixture checks the actual Vite proxy's path and cookie forwarding plus startup/deep-link serving. It does not claim live Go/Postgres/Redis integration. TypeScript and production build are separate checks.

`npm run test:browser` uses Playwright with intercepted API responses to test customer pages, join/error/ownership/cancellation/polling behavior, search races, pagination, account states and mobile layout. Install a Chromium test browser with `npx playwright install chromium`, or use an installed supported browser: in PowerShell, `$env:PLAYWRIGHT_CHANNEL = 'msedge'` (or `chrome`) before running the command. These tests never register real accounts or create real businesses/queues. Desktop/mobile screenshots go to ignored `output/playwright/`.

See [stage3b-report.md](docs/stage3b-report.md) for checks, behavior and live-integration status. Earlier stage reports are preserved.
