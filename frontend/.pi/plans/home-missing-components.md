# Home Missing Components Plan

## Goal
Add the missing Home Page pieces:
- current queue number per business/service
- total queue persons/users per business/service
- joined queue business list on Home

## Current state
Implemented:
- Header/taskbar: `src/components/CustomerHeader.tsx`
- Home search and discovery: `src/pages/Home.tsx`
- Business/service card: `src/components/BusinessCard.tsx`
- Business detail page: `src/pages/BusinessDetail.tsx`
- Join queue flow: `src/pages/JoinQueue.tsx`
- Joined queues page: `src/pages/MyQueues.tsx`

Missing:
- Queue summary data on Home/business cards/detail
- Joined queues embedded on Home

## Proposed implementation

### 1. Queue summary component
Create a reusable component, e.g. `src/components/BusinessQueueSummary.tsx`.

Responsibilities:
- Accept `businessId`.
- Load queues for that business using existing API `getQueuesByBusiness` from `src/api/queues.ts`.
- Compute:
  - `waitingCount`: `queues.filter(q => q.state === 'waiting').length`
  - `activeCount`: queues in `waiting`, `called`, `processing`
  - `currentQueue`: first `called` queue, otherwise first `processing` queue, otherwise first waiting queue
- Render:
  - Current queue number: queue `name`, or `No active queue`
  - Total queue persons/users: active/waiting count
- Handle loading/error gracefully.

Risk:
- `/queues/business/:businessId` may require authentication or business permission depending on backend. If it is not public, backend may need a public queue summary endpoint instead.

### 2. Add queue summary to business cards
Update `src/components/BusinessCard.tsx`:
- Render `<BusinessQueueSummary businessId={business.id} compact />` near the card footer.
- Keep the card clickable.
- Avoid making the summary buttons/links nested inside the existing card link.

Alternative if nested async content is too heavy:
- Add summary only on the detail page first.

### 3. Add queue summary to business detail
Update `src/pages/BusinessDetail.tsx`:
- Render the same queue summary in the aside before/near the Join Queue button.

### 4. Joined queues section on Home
Extract reusable joined queue list from `src/pages/MyQueues.tsx`, or create `src/components/JoinedQueuesPreview.tsx`.

Responsibilities:
- If authenticated, call existing `getMyQueues()`.
- Resolve business names with existing `getBusiness()` like `MyQueues.tsx` does.
- Render a compact list of joined businesses/services with:
  - business name
  - queue number
  - status badge
  - link to `/queue/:queueId`
- If guest, either hide the section or show a sign-in prompt. Current app only persists guest queue via active ticket, not a full guest list.

Then update `src/pages/Home.tsx`:
- Show `JoinedQueuesPreview` below intro/search or below discovery.

### 5. Validation
Run:
- `npm run build`
- Existing browser tests if available, likely `npm test` or project-specific test command from `package.json`.
- Manually verify:
  - Home renders as guest and authenticated user
  - Search still works
  - Business card/detail show current queue and total queue count
  - Joined queue preview appears for authenticated users
  - Join queue still redirects/displays ticket correctly

## Files likely touched in Build Mode
- `src/components/BusinessQueueSummary.tsx` new
- `src/components/JoinedQueuesPreview.tsx` new, or refactor `src/pages/MyQueues.tsx`
- `src/components/BusinessCard.tsx`
- `src/pages/BusinessDetail.tsx`
- `src/pages/Home.tsx`
- Possibly tests under `tests/browser/`
