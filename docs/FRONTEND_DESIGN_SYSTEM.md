# QueueLite Frontend Design System

Shared source of truth for how the QueueLite frontend looks and is built. Every team member and every AI agent working on `frontend/` should read this before changing UI, and update it in the same change when a token, component or convention changes.

- Code that implements this document: `frontend/src/design/tokens.css`, `frontend/src/design/foundation.css`, `frontend/src/components/foundation.tsx`, `frontend/src/components/ui/*`.
- Where the values below disagree with the code, the code wins. Fix this document in the same pull request.
- Notion: the design references that were meant to be consulted could not be accessed when this document was written (see [Provenance](#12-provenance-and-open-questions)). This file was derived from the repository, not from Notion.

---

## 1. Design concept

QueueLite is a tool people use while they wait: a customer glancing at their phone in a queue, or a staff member running a counter between customers. The UI therefore aims to be:

- **Calm.** A warm off-white background, white cards, one dark green action colour and a soft mint accent. No decoration that competes with the one thing the user needs (a queue number, a status, the next action).
- **Legible at a glance.** Queue numbers and states are the largest, boldest things on the page. Status is never colour-only; every badge has a text label.
- **Honest.** Features that do not exist yet say so (for example the disabled "Upgrade — coming later" button on Plans). No fake data and no controls that pretend to work.
- **Phone-first, desktop-capable.** Customers are mostly on phones; business staff use both. Every page must work at 360 px wide with no horizontal page scroll.
- **Consistent.** One font, one spacing scale, one set of radii, one button style, one card style.

Brand wordmark: "Queue" in `--ql-action`, "Lite" in `--ql-mint`. Tagline used on auth pages: "A little less waiting".

---

## 2. Colour palette

All colours are CSS custom properties defined in `frontend/src/design/tokens.css`. Use the token, never the raw HEX.

| Token | HEX | Use |
| --- | --- | --- |
| `--ql-background` | `#FAFAF8` | Page background |
| `--ql-surface` | `#FFFFFF` | Cards, header, inputs, popovers |
| `--ql-text` | `#202825` | Body text and headings |
| `--ql-text-secondary` | `#65706A` | Descriptions, helper text, inactive nav (`.ql-muted`) |
| `--ql-action` | `#293E36` | Primary buttons, links, focus ring, active nav, wordmark "Queue" |
| `--ql-mint` | `#69C5AA` | Brand accent (wordmark "Lite"); decorative only |
| `--ql-soft-mint` | `#EDF5EF` | Secondary button and badge fill, highlighted panels, avatar |
| `--ql-border` | `#E3E7E2` | Card, input and divider borders |
| `--ql-error` | `#B75D57` | Error borders and icons |
| `--ql-error-text` | `#93453F` | Error text and destructive buttons (meets contrast on white) |
| `--ql-priority` | `#A68542` | Priority accents |
| `--ql-priority-text` | `#795F2A` | Priority and "Called" text on light surfaces |
| `--ql-priority-surface` | `#FBE9E7` | Priority/alert tinted background |
| `--ql-priority-border` | `#E8B4AF` | Priority/alert tinted border |

Rules:

- Text on `--ql-surface` or `--ql-background` uses `--ql-text` or `--ql-text-secondary`. Do not use `--ql-mint` for text; it does not have enough contrast.
- Small error or priority text uses the `-text` variants, not the base colours.
- shadcn/Tailwind semantic names (`--background`, `--primary`, `--secondary`, `--muted-foreground`, `--destructive`, `--ring`, ...) are aliases of the tokens above. Components in `components/ui/` use the aliases; product code uses `--ql-*`.
- Light theme only (`color-scheme: light`). There is no dark mode; do not add partial dark styles.

---

## 3. Typography

Font: **Inter Variable**, bundled with the app through `@fontsource-variable/inter` (imported in `src/main.tsx`) so the UI renders identically on every machine, including offline. Fallback stack: `Inter, 'Segoe UI', system-ui, sans-serif`, exposed as `--ql-font`.

| Token | Size | Use |
| --- | --- | --- |
| `--ql-display` | 2.25rem (36 px) | Queue numbers, hero headings |
| `--ql-title` | 2rem (32 px; 28 px on phones) | Page `h1` |
| `--ql-section` | 1.25rem (20 px) | Section and card `h2` |
| `--ql-body` | 0.9375rem (15 px) | Body text |
| `--ql-small` | 0.875rem (14 px) | Labels, nav, buttons, helper text |
| `--ql-meta` | 0.8125rem (13 px) | Eyebrows, badges, fine print |

- Headings use weight `--ql-weight-heading` (600). Tailwind's preflight resets heading weight to 400, so `foundation.css` re-applies it for `h1`, `h2` and `h3`. Do not set heading weights ad hoc.
- Buttons are weight 600 at `--ql-small`. Labels are weight 500-600.
- Eyebrows (`.ql-eyebrow`) are small, uppercase, letter-spaced, in `--ql-action`.
- Use semantic heading levels: one `h1` per page, `h2` for sections, `h3` for items inside a section.

---

## 4. Layout and spacing

- Page wrapper: `PageContainer` (`.ql-page`). Content max width `--ql-content-width` (68rem), centred, horizontal gutter, vertical padding that shrinks on phones.
- Narrow, form-like pages add a modifier class (`.ql-account-page`, `.ql-narrow-section`, `.ql-ticket-page`) so text lines stay readable.
- Spacing scale (use only these): `--ql-space-1` 4 px, `-2` 8, `-3` 12, `-4` 16, `-6` 24, `-8` 32, `-10` 40, `-12` 48, `-16` 64.
  - Within a card: `space-3`/`space-4` between related items, `space-6` between groups.
  - Between page sections: `space-8`.
- Radii: `--ql-radius-control` 8 px (buttons, inputs), `--ql-radius-card` 12 px (cards, panels), `--ql-radius-pill` (badges, chips, avatar).
- Elevation: borders do most of the work. `--ql-shadow-card` is a barely visible resting shadow; `--ql-shadow-raised` is used for hover on clickable cards.
- Control height: `--ql-control-height` 2.75rem (44 px). Every tappable control is at least 44 px high (`min-h-11`).

### Page widths (one rule per page type)

| Type | Class | Max width | Pages |
| --- | --- | --- | --- |
| Wide (default) | `.ql-page` only | 68rem | Home, My Queues, Manage businesses, My service counters, Plans, every business tool tab (Overview, Counters, Members, Subscription, Share queue), Counter workspace |
| Narrow | `.ql-account-page` | 46rem, centred | Single-column forms: Profile, Create business, Business settings |
| Ticket | `.ql-ticket-page` | 38rem, centred | QR landing |
| Join | `.ql-join-page` | 68rem outer, 50rem content | Join form and confirmation |
| Status | `.ql-status-page` | 68rem | Ticket + counter board; stacked below 900px |

Pages that sit under the same tab bar (business tools) must share one width, so switching tabs never changes the layout. Inside a wide page, cap line length of long paragraphs (about 46rem) instead of narrowing the whole card.

### Breakpoints

The CSS uses `max-width` media queries at these four widths. Reuse them; do not invent new ones without a reason documented here.

| Width | What changes |
| --- | --- |
| `<= 1050px` | Staff sidebar becomes an expandable workspace menu; customer header collapses into a menu |
| `<= 960px` | Business grid 4 -> 2 columns |
| `<= 900px` | Counter and metric grids 3/4 -> 2 columns; ticket and counter board stack |
| `<= 640px` | Phone layout: single column, smaller `h1`, header collapses into a menu, buttons in stacks stretch, button labels may wrap, action rows stack |

### Grid rule that prevents phone overflow

Any single-column stack must use `grid-template-columns: minmax(0, 1fr)` (this is built into `.ql-stack`). A bare `display: grid` has an implicit `auto` column that grows to fit wide children (tables, long URLs) and stretches the whole page past the viewport. Wide content (tables) goes in a `*-table-wrap` container with `overflow-x: auto` and `position: relative`; the `position: relative` is required so absolutely positioned `.ql-sr-only` text inside cells cannot escape the scroll container and widen the page.

---

## 5. Components

Use the shared component before writing a new one. Wrappers live in `frontend/src/components/foundation.tsx`; Radix/shadcn primitives live in `frontend/src/components/ui/`.

| Component | Where | Notes |
| --- | --- | --- |
| `Button` | `foundation.tsx` (wraps `ui/button`) | Variants: `default` (filled action), `secondary` (soft mint / outlined), `destructive`, `ghost`, `link`. Sizes: `default` 44 px, `sm` 36 px. Use `asChild` to render a router `Link` as a button. Labels do not wrap on desktop (`white-space: nowrap`). |
| `Input`, `PasswordInput` | `foundation.tsx` | Always paired with a `Label`. `PasswordInput` has a show/hide toggle aligned to the input's vertical centre. |
| `Card` | `foundation.tsx` (wraps `ui/card`) | White surface, 1 px border, 12 px radius. `asChild` makes the whole card a link (queue cards, business cards). |
| `PageContainer` | `foundation.tsx` | Page wrapper described above. |
| `PageHeader` | `foundation.tsx` | `title`, optional `description`, optional `action` (right-aligned button or link), bottom divider. Use on every top-level page so headers look the same. |
| `StatusBadge` | `foundation.tsx` | Queue/ticket states (waiting, called, in service, cancelled, ...). Text label always shown. |
| `Badge` | `ui/badge` | Roles and counter status. `default` = soft mint (positive/available, roles), `outline` = bordered neutral (busy / in use), `secondary` = page-background tint (low emphasis only; almost invisible on white cards). |
| `LoadingSkeleton`, `Spinner` | `foundation.tsx` | Skeleton for page/section loading, spinner inside buttons. |
| `EmptyState` | `foundation.tsx` | `title`, `description`, optional `action`. `compact` renders a slim single-row version for secondary sections (for example "Joined queues" on Home). |
| `InlineError` | `foundation.tsx` | `role="alert"` message, always with a retry action when the request can be retried. |
| `ApplicationShell` | `components/ApplicationShell.tsx` | Customer header/footer or staff sidebar/top bar, selected by the existing route. Counter reads publish their business ID through context; the shell makes no extra API requests. |
| `AccountMenu` | `components/AccountMenu.tsx` | Header avatar button that reveals Account, Create Business and Logout. Disclosure pattern (`aria-expanded`), closes on Escape (focus returns to the button), outside click and navigation. Inside the phone menu the same list is shown inline. Day-to-day work links (My queues, My Counters, Manage Business) stay in the primary nav, not in this menu. |
| `BusinessNavigation` | `components/BusinessNavigation.tsx` | Sidebar navigation in `ApplicationShell`. Public page uses `end`. Manager links are absent for counter staff; on small screens the workspace menu can be expanded without a modal. |
| `Tabs`, `Table`, `Select`, `AlertDialog`, `Label` | `ui/*` | Standard shadcn components. Tables go inside a `*-table-wrap`. Destructive actions use `AlertDialog`. |

Page-level patterns (CSS classes in `foundation.css`):

- `.ql-section-heading`: title/description on the left, link or action on the right.
- `.ql-actions-row` / `.ql-link-row`: horizontal group of buttons or links with consistent gaps; stacks and stretches on phones.
- `.ql-members-section` + `.ql-member-list`: name + role badge rows.
- `.ql-counter-editor` / `.ql-counter-facts`: counter card with heading, status badge and fact lines.
- `.ql-success-note`: confirmation message (`role="status"`).
- `.ql-toolbar-end`: right-aligned toolbar above a list.
- `.ql-manage-card`: business card on Manage businesses. Header (name, location, role badge), three shortcut buttons (Open dashboard, Counters, Share queue) and a link row for settings and subscription. Do not repeat the business tab bar inside it.
- `.ql-plan-cards` / `.ql-plan-card`: stacked plan cards. Plan comparison renders as a table above 640px and as one card per plan on phones (the table is hidden, not removed, so both stay in sync from one data source).
- `.ql-ticket-callout`, `.ql-position-summary`: the two moments on the customer ticket that must be loud (see below).

---

## 6. UI states

Every interactive or data-driven element must handle these states.

| State | Treatment |
| --- | --- |
| Hover | Buttons dim slightly (`hover:opacity`); clickable cards raise with `--ql-shadow-raised` and a 1-2 px lift (disabled under `prefers-reduced-motion`). |
| Focus | `:focus-visible` shows a 2 px `--ql-action` outline with 3 px offset (buttons use a 2 px ring with offset). Never remove it; never rely on mouse hover alone. |
| Active / pressed | Enabled buttons move down 1 px. |
| Disabled | `opacity: .6`, `pointer-events: none`. A disabled button must explain itself in nearby text when the reason is not obvious. |
| Loading | `LoadingSkeleton` with an accessible label for pages and sections; `Spinner` plus `aria-busy` for in-flight buttons. Polling pages (ticket status refreshes every 5 s) must not flash skeletons on refresh. |
| Empty | `EmptyState` with a short explanation and the next action. |
| Error | `InlineError` in a card with a retry. Show the backend-provided message when available (see `frontend/docs/backend-errors.md`). |
| Success | `.ql-success-note` with `role="status"`. |
| Ticket: called | Card gets a 2 px action border and mint glow, the badge turns solid action colour with a pulsing dot (static under reduced motion), and a callout names the counter when the backend reports one ("Please go to Front desk."). |
| Ticket: waiting | "N queues ahead of you" is the largest text after the queue number; position and estimated wait sit below it. Only backend-provided values are shown. |
| Active nav item | Filled `--ql-action` tab (business nav) or bold action-coloured link (header). |

Accessibility baseline: semantic landmarks, one `h1` per page, visible focus, 44 px targets, text labels on all statuses, `prefers-reduced-motion` respected, text contrast at or above WCAG AA (that is why the `-text` colour variants exist).

---

## 7. Implementation conventions

Stack: React 19, TypeScript, Vite, Tailwind CSS 4 (`@tailwindcss/vite`), shadcn/ui ("new-york" style, `components.json`), Radix UI, class-variance-authority, lucide-react, react-router-dom 7.

1. **Tokens first.** New colours, sizes, spacing and shadows are added to `src/design/tokens.css` and used through `var(--ql-*)`. No raw HEX or arbitrary pixel values in components or CSS.
2. **Where styles go.**
   - Shared, reusable look: `foundation.tsx` wrappers or `components/ui/*`.
   - Page or feature layout: a `.ql-*` class in `src/design/foundation.css`, in the section closest to related rules. Prefix every class with `ql-`.
   - Tailwind utilities are fine for one-off layout inside shadcn components; do not build page layouts out of long utility strings when a `.ql-*` class already expresses it.
3. **Reuse before adding.** Check `foundation.tsx`, `components/ui/` and this document first. Do not add a second button, card, badge, table or dialog.
4. **Dependencies.** Do not add UI libraries (icon sets, component kits, CSS frameworks, animation libraries) without team agreement and an entry in section 8. The one visual dependency beyond the base stack is `@fontsource-variable/inter`.
5. **Icons.** `lucide-react` only, 16 px inside buttons (`.ql-button svg`), decorative icons `aria-hidden`.
6. **Links vs buttons.** Navigation is a `Link` (use `Button asChild` for button styling). Actions that change data are `button`s.
7. **Routes** come from `src/routes.ts`; API access only through `src/api/*`. Styling changes must not alter request or response handling, polling or route behaviour.
8. **Copy.** Sentence case, plain language, curly apostrophes (don’t). Describe unavailable features as unavailable.
9. **Lockfile.** If you add a dependency, run `npm install` with the team's npm version and check `git diff package-lock.json`. If unrelated lines change (for example `libc` fields disappear), restore the lockfile and hand-add only the new package entries so the diff stays small.
10. **Checks before opening a pull request.** From `frontend/`: `npm run typecheck`, `npm run build`, `npm test`. Look at the page at 360 px, 768 px and 1280 px wide and confirm there is no horizontal page scroll.

### Rules for AI agents

- Read this file and `foundation.css` before editing UI. Prefer editing a token or shared component over adding a one-off style.
- Make the smallest change that achieves the requested look. Do not restructure markup that tests and other pages depend on (`role`, `aria-*` and visible text are asserted in `tests/`).
- Do not invent data, plans, counts or states to fill a layout. Do not add placeholder controls that do nothing.
- When you add or change a token, component or convention, update this file in the same change and say so in the PR description.
- If you cannot verify something (no backend, no Notion access, no browser), say so instead of implying it was checked.

---

## 8. Design decisions

| Decision | Reason |
| --- | --- |
| Keep the existing mint/green palette and card-based layout | It already defines the product's identity; the improvement is consistency and polish, not a rebrand. |
| Bundle Inter locally instead of relying on installed fonts | The old stack named Inter but never loaded it, so every machine rendered a different fallback font. Bundling makes screenshots, demos and team machines match. Cost: one small dependency. |
| Heading weight 600 via a token | Tailwind preflight flattened headings to weight 400, which removed hierarchy. |
| `minmax(0, 1fr)` on stacks and `position: relative` on table wrappers | Fixes real phone overflow on Dashboard, Plans and Business plans caused by implicit grid columns and absolutely positioned screen-reader text. |
| `end` on the "Public page" tab | Without it the first tab was active on every business page. |
| Buttons do not wrap on desktop but may wrap on phones | Keeps one-line buttons tidy while preventing clipped labels at 360 px. |
| `PageHeader` on top-level pages | Shared heading rhythm without an extra divider. Ticket pages use their state heading. |
| Compact `EmptyState` for secondary sections | A full empty card for "Joined queues" pushed the main content down for new users. |
| Counter cards show status as a badge, not text | Faster scanning of busy/idle counters; matches the badge pattern used elsewhere. |
| Disabled "Upgrade — coming later" and explanatory text on Plans | Payments and plan changes are not implemented; the UI must not suggest otherwise. |
| `--ql-control-height` set to 2.75rem | The token said 46 px while real controls were 44 px; the token now matches reality. |
| Account links in an avatar menu, work links stay in the nav | Six or seven inline links crowded the header. Rarely used account actions collapse into one menu; My Counters and Manage Business are daily tools and stay visible. No new dependency: the menu is a small disclosure component. |
| Explicit widths by task | Customer discovery/status and staff tools use 68rem. Join content uses 50rem; account forms remain 46rem and QR entry remains 38rem. |
| Manage card uses shortcut buttons, not the tab bar | The tab bar was duplicated inside the card and wrapped on one line. Three buttons cover the common jobs; everything else is on the dashboard tabs. |
| Ticket called/waiting states are emphasised | The ticket is the screen customers (and reviewers) look at most. "Called" and "N ahead" are the two facts that matter. |
| Plan cards on phones | A 44rem table cannot fit 390px without cutting off a plan column; cards show every plan without sideways scrolling. |
| Light theme only | Dark mode is not in scope and half-implemented dark styles would create inconsistency. |

---

## 9. Page inventory

| Page | Route | Key UI |
| --- | --- | --- |
| Home / discovery | `/` | Business search and grid, joined-queues preview |
| Business detail | `/business/:businessId` | Business info, services, join call to action |
| Join queue | `/business/:businessId/join` | Join form (guest or account) |
| QR landing | `/qr/:businessId` | Redirect/entry from a shared link |
| Ticket / queue status | `/queue/:queueId` | Position, state badge, counters, next customer |
| My queues | `/my-queues` | Queue cards |
| Login / Register | `/login`, `/register` | Auth card |
| Profile, Plans | `/profile`, `/plans` | Account details, plan comparison table |
| Business: create, manage | `/business/create`, `/business/manage` | Forms, business selector |
| Business: dashboard, counters, members, plans, share, settings | `/business/:businessId/...` | `BusinessNavigation` plus operations pages |
| Counter workspace | `/counter/:counterId` | Serve queue, current customer, next customer |
| Staff counters | `/counters` | Assigned counters per business |
| Access / 404 | `/unauthorized`, `/forbidden`, `*` | Message card with a clear way back |

---

## 10. Responsive checklist

- [ ] No horizontal scroll of the page at 360, 390, 768, 1024 and 1280 px.
- [ ] Header collapses into a menu at `<= 640px` and all links remain reachable.
- [ ] Buttons and links are at least 44 px tall and not clipped.
- [ ] Tables scroll inside their wrapper, not the page.
- [ ] Long URLs and names wrap or truncate within their card.
- [ ] Headings keep the hierarchy `h1 > h2 > h3`.

---

## 11. Changing this document

- Add a row to [Design decisions](#8-design-decisions) when a decision would otherwise be re-litigated.
- Keep tables in sync with `tokens.css`. If you add a token, add its row here.
- Mention in the PR description when this file changed so teammates and their agents re-read it.

---

## 12. Provenance and open questions

- Derived from: the repository at the time of the frontend refresh (`tokens.css`, `foundation.css`, `foundation.tsx`, `components/ui/*`, pages under `src/pages`), plus screenshots of every route at phone, tablet and desktop widths rendered against intercepted API responses.
- The Notion design references named in the task were not available: the connected Notion workspace contained no QueueLite pages. Nothing was read from, or written to, Notion. If the team has a Notion design page or export, compare it with sections 2-5 and reconcile differences here.
- The root `README.md` links to a `docs/` suite (`docs/README.md`, `docs/AI_WORKFLOW_RULES.md`) that does not exist in the repository, and this file is currently the only file in `docs/`. If those documents exist elsewhere, add this file to their index.


## 13. Reference mockup refresh

The owner’s eight supplied QueueLite screenshots guide this refresh: warm off-white surfaces, quiet green accents, a large ticket on the left with counter information on the right, two-column join forms, a centered confirmation, and a persistent business sidebar on desktop. Form labels, focus targets, and body copy remain readable at phone sizes. Desktop business cards use four columns, two at 960px, and one at 640px.

Additional tokens in `tokens.css`:

| Token | Value | Purpose |
| --- | --- | --- |
| `--ql-sidebar-width` | 14rem | Desktop workspace sidebar |
| `--ql-flow-width` | 50rem | Join-flow content |
| `--ql-ticket-display` | clamp(3rem, 6vw, 4.5rem) | Ticket number hierarchy |
| `--ql-hero-surface` | #EDF0E4 | Business banner |
| `--ql-hero-decoration` | #D4DDC7 | Decorative banner lettering, never essential text |
| `--ql-icon-surface` | #EFF2E8 | Icon and note backgrounds |
| `--ql-count-size` | 1.5rem | Queue count chip |
| `--ql-banner-height` | 10rem | Banner/metric layout |
| `--ql-icon-size` | 3rem | Shared icon tiles |
| `--ql-icon-size-large` | 4rem | Business/success icons |
| `--ql-header-height` | 5rem | Staff top bar |
| `--ql-border-accent` | #86B29B | Decorative emphasis borders, never small text |

Backend boundaries for this implementation:

- Public business pages and cards do not call staff-only queue or counter listing endpoints. No public queue statistics are fabricated.
- Categories, service selection, customer names in staff tables, business-specific ticket prefixes, notification preferences, and staff cancellation buttons are omitted because the current API does not provide those flows/fields.
- Ticket numbers are displayed exactly as returned by the server (for example A020); no new numbering scheme is generated by the browser.
- Customer position, next ticket, counter state, and estimated wait come from the owned `/queues/:id/customer-status` response. The estimate is one server-provided value, not an invented range. The standard waiting/called/processing/terminal states and confirmation before leaving remain intact.
- Availability means the server's `operational` flag. It is not presented as a timezone-aware “Open now” calculation.
- Metrics summarize recorded visits across the returned dataset. They do not claim to be today's totals; average timing only uses completed records with both processing and completion timestamps.
- Complete finishes the current visit. Start service is required after Call next. Skip also calls the next ticket according to the backend. A combined Complete & call next or independent staff Cancel action is not added.
- Guest tickets remain browser-bound. Account and membership checks, mutation locks, uncertain-result handling, and polling cleanup remain in use.
- Development accepts both `/api` and the existing `/queuelite/api` proxy paths. Production still needs the documented same-origin reverse proxy and history fallback.

Automated application checks use intercepted API responses. For local visual review, `npm run preview:ui` additionally enables a Vite-only in-memory API and page picker in `frontend/dev/uiPreview.ts`. This module is outside the browser application and is enabled only for Vite serve in `ui-preview` mode. It never proxies preview requests to the backend. Normal development and production builds retain the real API behavior; no demo businesses, identities, queue values, toolbar or prototype persistence are bundled in the production application.
