# QueueLite

QueueLite lets customers join business queues and track their tickets while staff manage service counters.

## Features

- Business discovery, search and pagination.
- Account registration, login, profile editing and subscription summaries.
- Guest and account queue joining, ticket status polling and cancellation of waiting tickets.
- Business creation and settings with owner, admin and staff permissions.
- Business dashboards, counter management and staff assignment.
- Counter actions to call, process, skip and complete tickets.
- Redis-backed queue admission and background persistence with pending-message recovery.

Priority selection, live server events and billing administration are not exposed in the current frontend. Opening hours are displayed as daily times without timezone-aware opening status.

## Stack and layout

| Directory | Contents |
| --- | --- |
| `backend/` | Go HTTP API using Chi, GORM, PostgreSQL, Redis and JWT cookies |
| `frontend/` | React 19, TypeScript, Vite 7, React Router, Tailwind CSS and Playwright tests |

The backend separates domain models, application services, HTTP handlers and persistence adapters. The frontend uses typed API services and shared UI components.

## Canonical documentation

The repository source of truth is the [`docs/`](docs/README.md) suite. AI agents and contributors must read [`docs/AI_WORKFLOW_RULES.md`](docs/AI_WORKFLOW_RULES.md) before proposing or making changes. The older [frontend guide](frontend/README.md), [API error reference](frontend/docs/backend-errors.md), and `.documents/` reports are supplemental rather than canonical.

## Local setup

Requirements: Go 1.26.7 or later, Node.js 22.12 or later, npm, PostgreSQL and Redis 6.2 or later. Redis pending-message recovery uses `XAUTOCLAIM`.

From the repository root:

```sh
cd backend
go mod download
cp .env.example .env
```

On PowerShell, use `Copy-Item .env.example .env`. Fill in the backend configuration:

| Variable | Purpose | Default |
| --- | --- | --- |
| `DATABASE_URL` | PostgreSQL connection string | Required |
| `SECRET` | Long random signing key for account, guest and queue cookies | Required |
| `REDIS_HOST` | Redis address in `host:port` form | Required |
| `HTTP_ADDR` | HTTP listen address | `:8080` |

The Redis client currently connects to database 0 without authentication or TLS. Use a suitable local Redis instance for development.

Provision the PostgreSQL schema before starting the API. Schema helpers are in `backend/internal/adapter/postgres`: `MigrateDatabase` creates or updates the model tables; `MigrateQueueLifecycle` adds missing nullable queue lifecycle columns without backfilling historical rows. They must be called explicitly from a Go command within the backend module. This repository does not currently include a migration CLI, and API startup does not run migrations.

Once PostgreSQL and Redis are running and the schema is provisioned:

```sh
# In backend/
go run ./cmd/api

# In a separate terminal, from frontend/
npm ci
npm run dev
```

Open the address printed by Vite, normally `http://127.0.0.1:5173`. Keep the same hostname throughout a session so cookies remain available. Vite forwards `/api` requests to `http://127.0.0.1:8080` and removes the `/api` prefix. Frontend defaults require no environment file; overrides are described in its README.

## Checks

```sh
# In backend/
go test ./...
go build ./...

# In frontend/
npm run typecheck
npm test
npm run build
npx playwright install chromium
npm run test:browser

# From the repository root
git diff --check
```

Browser tests use intercepted API responses. Backend tests use in-memory Redis and persistence fixtures; these checks do not replace integration testing with PostgreSQL and Redis. Browser screenshots and build output are ignored by Git.

## Deployment notes

Serve the frontend build with SPA history fallback and a same-origin `/api` reverse proxy that removes the prefix. `npm run preview` serves the build locally but does not proxy the API. Review HTTPS cookie settings and Redis authentication/TLS before deployment. Cross-origin hosting requires backend CORS and cookie support, which the current router does not configure.

Queue admission uses Redis before asynchronous database persistence, so recently accepted tickets can take time to appear in account queue lists. Customers can cancel waiting tickets; service transitions require authorized staff. Ordinary account sessions can read authorized subscription details but cannot administer plans or quotas.
