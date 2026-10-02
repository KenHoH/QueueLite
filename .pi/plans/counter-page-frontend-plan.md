# Counter Page Frontend Plan

**Status:** Implemented. Dashboard board, client paging/filtering, counter cards, workspace layout, lifecycle timing, responsive styles, and automated unit/API coverage are complete. Browser tests were added but could not run in the current environment because Chromium is missing the system library `libglib-2.0.so.0`.

## Goal
Build the business-facing counter board represented by the sketch:

- a live view of every counter and the customer currently being handled;
- a paged current-queue list, with a clear priority treatment;
- a selected-counter work area with current customer, next customer, elapsed service time, and supported actions.

## What already exists

- `frontend/src/pages/Dashboard.tsx` already loads the business, all queues, and accessible counters every 10 seconds, and maps `counter.currentQueueId` to a queue.
- `frontend/src/pages/CounterWorkspace.tsx` already provides the safe staff workflow: **Call next**, **Start service**, **Complete**, and **Skip**.
- `frontend/src/pages/CounterManagement.tsx` already manages counter setup and employee assignment.
- `GET /businesses/{businessId}/queues` and `GET /businesses/{businessId}/counters` provide the data needed for a first monitoring board. The queue DTO exposes a queue number/name, state, priority, and assigned counter ID, but not a customer display name or lifecycle timestamps.

## Product/technical decisions

1. **Enhance the existing business dashboard, not add a competing page.**
   Use `/business/:businessId/dashboard` as the board for queue/counter monitoring. Keep `/counter/:counterId` as the focused, authorized action workspace; counter cards link to it. This avoids giving a staff member an action UI for counters they are not assigned to.

2. **Treat `Queue A/B/C` in the sketch as a filter only for now.**
   The current domain has one waiting stream per business, not multiple service/queue-line entities. Initial filters should be `All`, `Waiting`, `Called`, `In service`, and optionally terminal states. Actual named queue lines require a backend service/queue-line model and counter-to-line assignment.

3. **Do not label queue numbers as customer names.**
   The current staff queue response only identifies a customer by `queue.name` (for example `Q004`). If the intended list must display user names, add a staff-only customer display field to the business-queue response; do not expose contact details.

4. **Do not add a staff “Cancel” action without a contract.**
   The available counter mutations are complete (only after processing) and skip. Customer cancellation is a separate customer-authorized transition. The board should show `Skip` and state-appropriate `Start service`/`Complete`; add a staff cancellation endpoint and confirmation policy only if cancellation is a distinct business requirement.

## Build steps

1. **Extract dashboard board components**
   - Add focused components (or colocated components) for a board header, queue filter/page controls, paged queue list, and counter-service grid.
   - Preserve `BusinessOperationsPage`, `useBusinessOperations`, role-based access, explicit refresh, and 10-second visible-tab polling.

2. **Implement the current-queue list**
   - Derive the selected state filter from the existing `Queue[]` response.
   - Default to active records (`waiting`, `called`, `processing`); make terminal history an explicit filter rather than mixing it into the current queue.
   - Add `10`, `25`, and `50` page-size buttons/select control, reset the page when the filter or page size changes, and clamp the page after polling changes the result count.
   - Render queue number, state badge, priority badge, and assigned/current counter where available. Use a semantic table on larger screens and an accessible stacked list on small screens.
   - Add a visually distinct priority row using the design token palette (the sketch’s pink/red treatment) without relying on color alone.

3. **Implement the every-counter view**
   - Render one card per authorized counter with counter name, availability/in-service status, and the mapped current queue number.
   - Make idle and unresolved assignment states explicit (the existing `State pending — refresh` safety behavior remains important).
   - Link each card to `/counter/:counterId`; only present action affordances when the current user can operate that counter.

4. **Refine the focused workspace to match the action portion of the sketch**
   - Reorganize `CounterWorkspace.tsx` into clear “Customer being handled,” “Next customer,” and action sections while retaining its mutation lock and refresh-after-mutation behavior.
   - Show the next waiting queue only after an authorized data source exists; do not infer it from an unordered database list, because Redis scheduling includes priority/fairness rules.
   - Keep action states truthful: called → Start service / Skip; processing → Complete / Skip; idle → Call next.

5. **Add elapsed-time support after the API contract is extended**
   - Expose `calledAt`, `processingAt`, and (for history) `doneAt` in `QueueResponse`; extend `frontend/src/api/types.ts` accordingly.
   - Calculate the displayed service timer from server `processingAt`, refresh it locally at a bounded interval, and stop it for terminal/unavailable states. Do not start a timer from the browser click because reloads, other staff actions, and server timing would make it inaccurate.

6. **Scale pagination and customer/counter data in the backend when needed**
   - Replace client-only slicing with a business-authorized paginated queue endpoint (`state`, `limit`, cursor) before queue volumes become large.
   - If the list needs customer names, add a minimal role-authorized `customerDisplayName` field.
   - If the board needs a scheduler-accurate “next customer,” expose an authorized next-queue snapshot rather than duplicating Redis priority/fairness selection in React.

7. **Style and accessibility**
   - Add responsive board styles to `frontend/src/design/foundation.css`: desktop queue/workspace layout, counter grid, compact mobile cards, and no horizontal overflow.
   - Use headings, labelled regions, native buttons/select controls, status text for refresh/mutation feedback, visible focus states, and text/badges in addition to priority color.

## Files expected to change in Build Mode

- `frontend/src/pages/Dashboard.tsx`
- `frontend/src/pages/CounterWorkspace.tsx`
- `frontend/src/pages/operationsDisplay.ts`
- `frontend/src/components/` (new board/list/counter components if extraction is chosen)
- `frontend/src/design/foundation.css` and possibly `frontend/src/design/tokens.css`
- `frontend/src/api/types.ts`, `frontend/src/api/operations.ts`, and backend queue response/routes only for the timestamp, customer-display, next-snapshot, or server-pagination follow-up
- `frontend/tests/operations.test.mjs` and browser operation tests

## Validation

- Run `npm run typecheck`, `npm test`, and `npm run build` from `frontend/`.
- Extend browser tests for priority styling/text, filtering, `10/25/50` paging and page clamping after refresh, counter-to-current-customer mapping, owner/admin versus assigned-staff access, loading/error/retry states, and mobile overflow.
- Verify that mutation failures still fail closed and require refresh before another counter action.
