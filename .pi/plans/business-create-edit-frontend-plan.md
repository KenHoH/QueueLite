# Create and Edit Business Frontend Audit

## Finding
No create-business or edit-business page/component is missing in the current frontend. Both requested flows are implemented, routed, API-connected, permission-aware, and covered by browser tests.

### Existing implementation

| Requirement | Route | Implementation |
| --- | --- | --- |
| Create business | `/business/create` | `frontend/src/pages/BusinessSetup.tsx` → `CreateBusiness` and shared `BusinessForm` |
| Edit business | `/business/:businessId/settings` | `frontend/src/pages/BusinessSetup.tsx` → `BusinessSettings`, `SettingsAccess`, and the same `BusinessForm` |

Supporting pieces already present:
- `frontend/src/App.tsx` registers both routes.
- `frontend/src/api/businesses.ts` exposes `createBusiness`, `getBusiness`, and `updateBusiness`.
- `frontend/src/api/types.ts` defines the request/response DTOs.
- `frontend/src/pages/accountForms.ts` supplies shared form defaults, client validation, role detection, and safe error messages.
- `frontend/src/components/AccountAccess.tsx` gates unauthenticated users; `BusinessSettings` additionally restricts editing to owners/admins.
- `frontend/src/components/BusinessNavigation.tsx` links settings from business management pages.
- `frontend/src/pages/ManageBusinesses.tsx` links to create, settings, and selection of owned/managed businesses.
- `frontend/tests/browser/account.test.mjs` covers create validation/submission, duplicate-submit prevention, ambiguous-create handling, edit/save including `operational: false`, role denial, and mobile layout.

## Behavior already delivered
1. **Create** validates name, location, open/close time, email, and phone; posts only backend-supported fields; prevents duplicate submits; refreshes memberships; then takes the owner to the new business settings page.
2. **Edit** fetches the current business, pre-fills the same form, submits `PUT /businesses/:businessId`, supports description and operational-state changes, and refreshes the managed-business list.
3. **Access/error states** include sign-in redirect, invalid ID, membership lookup loading/error, owner/admin authorization, request errors, and an explicit uncertain-create state that blocks duplicate business creation.

## Build-mode plan
No product-code build is required for the two checklist items. If the intent is to polish these existing flows, use this low-risk sequence:

1. Run the existing frontend checks to establish the baseline:
   - `npm --prefix frontend run typecheck`
   - `npm --prefix frontend test`
   - `npm --prefix frontend run test:browser`
2. Manually verify the entry points while authenticated:
   - `/business/create`
   - `/business/manage` → selected business → **Business settings**
3. Only if product requirements differ, update the existing shared `BusinessForm` rather than creating duplicate create/edit pages. Possible scoped additions are a cancel/back action, field-length hints aligned with backend limits, or a post-create redirect to the dashboard instead of settings.
4. Extend `frontend/tests/browser/account.test.mjs` for any changed copy, navigation destination, or newly added form fields.
5. Re-run the checks above plus `git diff --check`.

## Decisions needed before any enhancement
- Should a newly created business open **Settings** (current behavior) or **Dashboard**?
- Is deletion intended to be part of “edit business”? The API has `deleteBusiness`, but no deletion control currently exists; it should be a separately confirmed destructive-flow requirement.
- Are additional fields required beyond the backend-supported name, location, description, hours, email, phone number, and operational status? Any new persisted field requires a backend API/DTO change first.
