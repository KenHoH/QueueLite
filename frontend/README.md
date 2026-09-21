# QueueLite frontend prototype

Self-contained React + Vite + TypeScript + Tailwind CSS application. Node.js 22.12+ (or 20.19+) is required.

```sh
cd frontend
npm install
npm run dev
```

Build with `npm run build`; serve that build with `npm run preview`.

The dev server binds to `127.0.0.1`. On Windows paths containing literal `~` characters, a small Vite middleware replaces the overly broad short-filename guard while retaining a canonical frontend-only file boundary and blocking hidden files. Other paths retain Vite's default strict file serving. Production builds do not use this middleware.

## Routes

- `/` — business discovery, search, categories, active tickets
- `/business/:id` — business information
- `/business/:id/join` — details form and success state
- `/queue/:id` — customer ticket and public counter status
- `/dashboard` — business overview, service filters, paginated queue
- `/counter/:id` — focused counter controls

Start with Northside Barbers for the complete shared customer/staff demo. Matthew already has ticket A-020; joining Northside reuses it to avoid duplicate active tickets. GreenCare and QuickFix demonstrate creating new tickets. Switch views through Business demo / Customer view. Completing, skipping, or cancelling a customer records the outcome, updates dashboard totals, and calls the next waiting customer. Call next is available only when the counter is idle. A skipped customer leaves the active queue in this prototype.

## Structure

`src/pages` contains the six experiences; `src/components` contains navigation, cards, badges, and queue table; `src/data/mock.ts` centralizes deterministic sample data; `src/types` holds entities; `src/state/QueueContext.tsx` shares in-memory queue state.

## Prototype limits

All state resets on refresh. No backend, authentication, messages, payments, persistence, or QR generation. Notification checkbox saves a preview preference only. Wait ranges, elapsed waits, average service duration, opening status and initial service start times are illustrative, not live calculations. Staff actions use FIFO, including marked priority tickets (one priority ticket in the primary 12-person waiting queue); priority scheduling is future work. Staff workspace is scoped to Northside; other businesses are discovery/join examples. Desktop-first layout with basic narrow-screen fallbacks. Contact details are fictional.

Future integration: backend state, business hours, real timing, notification delivery, priority policy, authenticated staff access, and full mobile/accessibility testing.
