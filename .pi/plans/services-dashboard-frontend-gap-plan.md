# Services / Business Dashboard Frontend Gap Plan

## Goal from mockup
A business-facing dashboard page showing:
- Queue selector/tabs (`Queue A/B/C` in the sketch; currently this maps best to queue state filters unless backend adds multiple services/queue lines).
- Current queue list with all listed customers, priority customers highlighted red/pink.
- Page-size controls / pagination (`10 / 25 / 50`).
- Every counter with the customer currently handled.
- Operational totals: skipped, cancelled, finished, average service/wait time.

## Current frontend coverage
Existing routes/pages:
- `frontend/src/pages/Dashboard.tsx` at `/business/:businessId/dashboard`
  - Already loads business, queues, counters via `useBusinessOperations`.
  - Shows grouped `waiting`, `called`, `processing` queues.
  - Shows counters and their current queue through `CounterOverview`.
  - Polls every 10 seconds via `useOperations`.
- `frontend/src/pages/CounterManagement.tsx` at `/business/:businessId/counters`
  - Creates/edits counters, assigns staff, opens workspace.
- `frontend/src/pages/CounterWorkspace.tsx` at `/counter/:counterId`
  - Staff actions: call next, start service, complete, skip.
- `frontend/src/api/operations.ts`, `frontend/src/api/queues.ts`, `frontend/src/api/counters.ts`
  - API wrappers exist for business queues, counters, members, and counter workflow actions.

## Missing frontend components/pages
Recommended to enhance the existing `Dashboard.tsx` rather than create a separate route.

1. `ServicesDashboardHeader`
   - Business name, operational state, refresh button, last updated text.
   - Reuse route `/business/:businessId/dashboard`.

2. `QueueFilterTabs`
   - Initial tabs should be state-based because backend currently has one queue stream per business:
     - All
     - Waiting
     - Called
     - Processing
     - Finished
     - Cancelled
     - Skipped
   - If the product really needs multiple independent queue lines (`Queue A/B/C`), backend needs a new service/queue-line model first.

3. `QueuePageSizeControl`
   - UI controls for `10 / 25 / 50`.
   - Initially frontend-only slicing is possible, but backend pagination is recommended for large businesses.

4. `CurrentQueueList`
   - Table/list of queue customers.
   - Fields available now: queue name/number, state, priority, active counter id.
   - Priority row styling: red/pink background.
   - Missing from current DTO: customer phone/contact, joined/called/processing/done timestamps.

5. `CounterServiceGrid`
   - Counter cards (`Counter 1`, `Counter 2`, etc.).
   - Show current handled customer from `counter.currentQueueId -> queues.find(id)`.
   - Show per-counter handled count only if backend adds/keeps a stable handled-by-counter field for completed queues.

6. `QueueMetricsSummary`
   - Cards for total skipped, total cancelled, total finished.
   - Can be computed from `Queue.state` if `/businesses/{id}/queues` returns terminal queues.
   - Average time needs backend timestamp fields in responses.

7. Optional `DashboardSkeleton` / `DashboardErrorPanel`
   - Improve loading/error layout for a data-heavy dashboard.

## Endpoint map needed by this dashboard

### Existing endpoints already wrapped in frontend
- `GET /businesses/mine`
  - Used to verify accessible businesses and role.
  - Frontend: `getMyBusinesses()` in `frontend/src/api/businesses.ts`.

- `GET /businesses/{businessID}`
  - Public business details.
  - Frontend: `getBusiness()`.

- `GET /businesses/{businessID}/queues`
  - Authenticated business/member queue list.
  - Frontend: `getBusinessQueues()` in `frontend/src/api/operations.ts`.
  - Current response DTO: `Queue[]` with `id`, `businessId`, `userId?`, `calledByCounterId?`, `name`, `state`, `priority`.

- `GET /businesses/{businessID}/counters`
  - Authenticated business/member counter list.
  - Frontend: `getBusinessCounters()`.
  - Current response DTO: `Counter[]` with `id`, `businessId`, `name`, `currentEmployeeId?`, `currentQueueId?`.

- `GET /businesses/{businessID}/members`
  - Owner/admin only; useful for showing assigned staff names.
  - Frontend: `getBusinessMembers()`.

- `GET /counters/{counterID}`
  - Needed by counter workspace, not dashboard overview.

- Counter workflow endpoints used from workspace:
  - `POST /counters/{counterID}/business/{businessID}/call-next`
  - `POST /counters/{counterID}/queues/{queueID}/process`
  - `POST /counters/{counterID}/queues/{queueID}/skip`
  - `DELETE /counters/{counterID}/queues/{queueID}` complete/removes current queue.

### Backend gaps / endpoints or DTO additions needed
1. Paginated business queues
   - Current endpoint returns all queues with no query params.
   - Needed for mockup page-size controls and larger businesses.
   - Suggested contract:
     - `GET /businesses/{businessID}/queues?state=waiting&limit=10&cursorCreatedAt=...&cursorID=...`
     - Response: `CursorPage<Queue>`.

2. Queue lifecycle timestamps in response
   - Domain already has `calledAt`, `processingAt`, `doneAt`, `cancelledAt`, `createdAt`, `updatedAt`.
   - `QueueResponse` does not expose them.
   - Needed for average wait/service time and richer dashboard sorting.

3. Business queue metrics endpoint
   - Avoid computing metrics client-side over an unbounded queue list.
   - Suggested contract:
     - `GET /businesses/{businessID}/queue-metrics?from=YYYY-MM-DD&to=YYYY-MM-DD`
     - Response: `{ waiting, called, processing, skipped, cancelled, done, averageWaitSeconds, averageServiceSeconds }`.

4. Per-counter handled metrics
   - Completed queues currently clear `calledByCounterId`, so finished count per counter cannot be reliably computed from current `QueueResponse`.
   - Suggested backend options:
     - Add immutable `handledByCounterId`/`completedByCounterId` on queue records, or
     - Add a counter activity/audit table.
   - Suggested endpoint:
     - `GET /businesses/{businessID}/counter-metrics?from=...&to=...`
     - Response: `[{ counterId, counterName, currentQueueId, handledCount, skippedCount, averageServiceSeconds }]`.

5. Multiple services / queue lines, if `Queue A/B/C` are real separate queues
   - No current backend service/queue-line model exists.
   - Needed resources would be something like:
     - `GET /businesses/{businessID}/services`
     - `POST /businesses/{businessID}/services`
     - `GET /businesses/{businessID}/services/{serviceID}/queues`
     - Counters may need `serviceID` assignment.
   - If not required, implement tabs as status filters instead.

## Implementation sequence for Build Mode
1. Refactor `Dashboard.tsx` into small local or component files: header, filters, queue list, counter grid, metrics.
2. Add frontend-only filtering and page-size slicing using existing `Queue[]` data.
3. Compute available totals from current DTO: counts by `state` and active counter/customer mapping.
4. Add visual priority styling for queue rows.
5. Add placeholder/disabled average-time card with copy like “Available after backend exposes queue timestamps”.
6. After backend DTO/endpoint changes, update `frontend/src/api/types.ts` and `frontend/src/api/operations.ts` to consume metrics/pagination.

## Validation checklist
- `npm run typecheck` in `frontend/`.
- Verify `/business/:businessId/dashboard` for owner/admin/counter roles.
- Verify priority queues are visibly highlighted.
- Verify counters show current customer when `currentQueueId` is present.
- Verify pagination controls do not lose poll-refresh data.
