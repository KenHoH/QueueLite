# Customer Queue Page Status Plan

## Implementation status
Implemented in Build Mode:
- protected `GET /queues/{queueID}/customer-status` route and handler
- backend aggregate status service with counters, serving queues, fair waiting order, position, totals, and ETA
- frontend API types/client validation and visibility-aware polling hook
- customer ticket counter board, next-customer card, position, and estimated wait UI
- backend aggregate service test

Validation completed:
- backend `go test ./...`
- frontend `npm test`, `npm run typecheck`, and `npm run build`
- browser suite could not run because the Playwright Chromium executable is not installed

## Goal
Build the customer POV page after `Customer Page → Join Queue` to show:
- the customer’s current ticket and position in the queue
- total/open counters and what each counter is serving
- the next queue being called
- estimated waiting time

Reference sketch: `/tmp/pi-clipboard-40a5eeed-2ee7-4c92-88e9-9a63d59d6579.png`.

## Current frontend state
Already present:
- Join page: `src/pages/JoinQueue.tsx` at `/business/:businessId/join`
- Ticket/status page: `src/pages/QueueStatus.tsx` at `/queue/:queueId`
- Ticket polling hook: `src/pages/useQueueTicket.ts`
- Queue API client: `src/api/queues.ts`
- Counter API client: `src/api/counters.ts`
- Types: `src/api/types.ts`

Current `/queue/:queueId` shows:
- queue number/name
- queue state badge
- business name via `GET /businesses/:businessId`
- cancel/leave action while waiting
- fallback counter copy if `calledByCounterId` exists

Missing for the requested page:
- No queue position / “N queues more”.
- No total counters display.
- No public counter board showing `Counter 1 → Queue A`, etc.
- No “next customer” queue display.
- No estimated wait time.
- No dedicated API type/client function for a customer-facing live queue board.
- Current counter lookup from customer ticket uses `GET /counters/:counterId`, but that endpoint is staff-authenticated, so customers cannot reliably get counter names.

## Backend endpoint map

### `../backend/internal/bootstrap/routes.go` check
Current `routes.go` is missing the endpoint required by this plan.

Present under `/queues`:
- `GET /queues/me`
- `POST /queues/business/{businessID}/join`
- `GET /queues/{queueID}`
- `GET /queues/{queueID}/state`
- customer cancellation/update/delete routes

Missing:
- `GET /queues/{queueID}/customer-status`
- optional stream route for live board updates
- registered public/business summary route; `GET /queues/business/{businessID}/summary` is commented out
- registered SSE route; `QueueHandlerImpl.SSEHandler` exists but is not wired in `routes.go`

Recommended `routes.go` addition in Build Mode:
```go
r.Group(func(protected chi.Router) {
    protected.Use(middleware.PublicMiddleware(userRepo))
    protected.Use(middleware.ProtectedMiddleware(*queueService))
    protected.Get("/{queueID}", queueHandler.GetQueue)
    protected.Get("/{queueID}/state", queueHandler.GetQueueState)
    protected.Get("/{queueID}/customer-status", queueHandler.GetCustomerQueueStatus)
    // optional later:
    // protected.Get("/{queueID}/customer-status/stream", queueHandler.CustomerQueueStatusStream)
    protected.Delete("/{queueID}", queueHandler.DeleteQueue)
    protected.Put("/{queueID}", queueHandler.UpdateQueue)
    protected.Patch("/{queueID}/state", queueHandler.UpdateState)
    protected.Patch("/{queueID}/done", queueHandler.MarkAsDone)
})
```
This should be placed in the existing protected `/queues` group so it reuses queue-token/auth ownership checks from `ProtectedMiddleware`.

Note: do not solve the customer page by exposing `GET /businesses/{businessID}/counters`, `GET /businesses/{businessID}/queues`, or `GET /counters/{counterID}` publicly; those are staff/business operational endpoints.

### Existing endpoints usable now
| Need | Endpoint | Frontend client | Auth/access | Notes |
|---|---|---|---|---|
| Join queue | `POST /queues/business/{businessID}/join` | `joinBusinessQueue` | Public middleware; guest body or authenticated cookie | Already used by `JoinQueue.tsx`. |
| Legacy/QR join | `POST /queues/qr/{businessID}` | `registerQueueByQR` | Public | Optional for QR flow, not required for normal join page. |
| Resolve QR | `GET /queues/qr/{businessID}/resolve` | `resolveQueueQR` | Public | Optional QR-specific form behavior. |
| Read ticket | `GET /queues/{queueID}` | `getQueue` | Protected by queue token/auth | Already used by `useQueueTicket`. |
| Read ticket state | `GET /queues/{queueID}/state` | `getQueueState` | Protected by queue token/auth | Already polled by `useQueueTicket`. |
| Cancel waiting ticket | `PATCH /queues/{queueID}/state` body `{ state: "cancelled" }` | `updateQueueState` | Protected by queue token/auth | Already used by `QueueStatus.tsx`. |
| Read business name/details | `GET /businesses/{businessID}` | `getBusiness` | Public | Already used by `QueueStatus.tsx`. |

### Existing endpoints not sufficient for customer queue page
| Endpoint | Issue |
|---|---|
| `GET /counters/{counterID}` | Staff/business auth only via `RequireCounter`; customer cannot use it for counter name. |
| `GET /businesses/{businessID}/counters` | Business member auth only; not customer-facing. |
| `GET /businesses/{businessID}/queues` | Business member auth only. |
| `GET /queues/business/{businessID}` | Auth + business access route is registered before the public group, so not customer/public. |
| `SSEHandler` / `GetWaitingQueueSnapshot` | Handler exists but is not registered in `routes.go`; snapshot only has waiting queue IDs/names, not counters, current calls, position, or ETA. |
| `GET /queues/business/{businessID}/summary` | Response type exists/commented, but route/service/repo are commented out and summary lacks customer position/counter board. |

## Backend API needed
Preferred: add one customer-safe aggregate endpoint so the frontend does not stitch staff-only resources.

### New endpoint: customer queue status
`GET /queues/{queueID}/customer-status`

Access:
- Same ownership/token protection as `GET /queues/{queueID}`.
- Server derives `businessId` from the queue; do not trust a client-supplied business ID.

Suggested response:
```ts
interface CustomerQueueStatus {
  queue: Queue;
  business: { id: string; name: string; operational: boolean };
  counters: Array<{
    id: string;
    name: string;
    currentQueueId?: string;
    currentQueueName?: string;
    state: 'idle' | 'called' | 'processing';
  }>;
  totalCounters: number;
  activeCounters: number;
  currentlyServing: Array<{
    counterId: string;
    counterName: string;
    queueId: string;
    queueName: string;
    state: 'called' | 'processing';
  }>;
  nextQueue?: { queueId: string; queueName: string };
  customerPosition?: {
    position: number;       // 1 means next among waiting queues
    ahead: number;          // position - 1; sketch displays this as "3 queues more"
    queueName: string;
  };
  totalWaiting: number;
  totalActiveQueues: number;
  estimatedWaitMinutes?: number;
  updatedAt: string;
}
```

Backend data sources:
- Queue identity/state: existing `QueueService.GetQueue`.
- Waiting order/next queue/position: use Redis waiting sets through existing queue cache helpers, ideally the same order used by call-next scheduling. Avoid using a different DB order if Redis/fair priority scheduling is authoritative.
- Counters + current queues: existing counter repo `ListBusinessCounters` and queue repo `GetQueue` for each `currentQueueId`.
- ETA: start with a simple formula if no analytics exist, e.g. `ceil(ahead / max(activeCounters, 1)) * DEFAULT_SERVICE_MINUTES`; later replace with real average service duration from lifecycle timestamps.

### Optional live updates
After the aggregate endpoint works, add either:
- polling every 5 seconds from the frontend, or
- register an SSE route such as `GET /queues/{queueID}/customer-status/stream` or `GET /businesses/{businessID}/queue-board/stream`.

Existing `SSEHandler` can be reused only after extending its payload and registering it in `internal/bootstrap/routes.go`.

## Frontend implementation plan

### 1. Add API types
Update `src/api/types.ts` with `CustomerQueueStatus`, `CounterBoardItem`, and related small interfaces matching the backend response.

### 2. Add API client
Update `src/api/queues.ts`:
- `getCustomerQueueStatus(queueId, options?) => api<CustomerQueueStatus>(/queues/${queueId}/customer-status)`

Keep existing `getQueue`/`getQueueState` temporarily for backwards compatibility or as fallback.

### 3. Replace/extend ticket polling hook
Create or update hook:
- `src/pages/useCustomerQueueStatus.ts`

Responsibilities:
- Poll `getCustomerQueueStatus(queueId)` every 5 seconds while queue is active.
- Pause when document is hidden.
- Stop polling for terminal states: `cancelled`, `skipped`, `done`.
- Return `{ status, data, error, refresh }`.

### 4. Update `QueueStatus.tsx` UI
Use the new aggregate hook and render sections:
- Header: business name + ticket number + state badge.
- Counter board: cards/grid for each counter: counter name and current queue name or “Idle”.
- Next customer: `nextQueue.queueName` or “No waiting customer”.
- Your queue: `queue.name`, `customerPosition.ahead`, and ETA.
- Keep existing cancel/leave behavior for waiting queues.

Map sketch copy:
- `Your QUEUE: Queue G (3 queues more)` → `customerPosition.ahead`.
- `Estimation time: N minute` → `estimatedWaitMinutes`.
- `Next Customer Queue D` → `nextQueue.queueName`.
- `Counter 1 / Queue A` etc. → `counters`.

### 5. Styling/components
Likely missing components:
- `CounterBoard` component, e.g. `src/components/CounterBoard.tsx`.
- `NextQueueCard` or inline card in `QueueStatus.tsx`.
- `CustomerPositionSummary` or inline summary.

Add CSS in `src/design/foundation.css` for:
- counter grid
- next customer card
- queue position/ETA summary
- responsive single-column layout on mobile

### 6. Fallback if backend is not ready
If Build Mode must proceed before backend changes:
- Keep existing `QueueStatus.tsx` as base.
- Display ticket state and business name only.
- Do not call staff-only counter endpoints from the customer page.
- Hide/disable position/counter board with “Queue details unavailable” until `customer-status` exists.

## Validation
Frontend:
- `npm run typecheck`
- `npm run build`
- Browser manual checks:
  - guest joins queue and opens `/queue/:queueId`
  - authenticated user joins queue and opens `/queue/:queueId`
  - waiting ticket shows position/ahead/ETA
  - called ticket highlights assigned/current counter
  - terminal/cancelled ticket stops polling

Backend/API contract:
- Add handler/service/repo tests for `GET /queues/{queueID}/customer-status`:
  - unauthorized without ticket/auth is rejected
  - guest queue token can read status
  - authenticated queue owner can read status
  - unrelated user cannot read status
  - response includes counters/current queues/next queue/position
  - terminal queues omit or null position

## Files likely touched in Build Mode
Frontend:
- `src/api/types.ts`
- `src/api/queues.ts`
- `src/pages/useCustomerQueueStatus.ts` new
- `src/pages/QueueStatus.tsx`
- optional `src/components/CounterBoard.tsx` new
- `src/design/foundation.css`

Backend required for full feature:
- `../backend/internal/bootstrap/routes.go`
- `../backend/internal/queue/inbound/http/queue_handler.go`
- `../backend/internal/queue/inbound/http/queue_response.go`
- `../backend/internal/queue/app/queue_service.go`
- queue/counter repo methods as needed for public-safe board aggregation
