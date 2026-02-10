# SISAP-PoC — CT Brand Monitor

Monorepo for a Certificate Transparency (CT) brand-protection monitor. Watches a public CT log for newly issued certificates matching user-defined keywords to detect potential phishing or domain abuse.

## Repo Layout

```
backend/         Go REST API (see backend/CLAUDE.md)
frontend/        React + TypeScript SPA (see frontend/CLAUDE.md)
db/              Database seed script (init.sql)
docs/            Specs and requirements
docker-compose.yml
```

## Quick Start (Local Development — Recommended)

**Terminal 1: Start the database**
```bash
docker compose up -d db
```

**Terminal 2: Start the backend**
```bash
cd backend
DATABASE_URL="postgres://ctmonitor:ctmonitor_dev@localhost:5432/ct_monitor?sslmode=disable" go run ./cmd/server
```

**Terminal 3: Start the frontend**
```bash
cd frontend
npm run dev
```

The frontend dev server (`:3000`) proxies `/api` to the backend (`:8080`). This is the fastest workflow for local iteration.

## Docker Compose (Optional — for production preview)

To preview the full stack with nginx as it would run in production:

```bash
docker compose up --build
```

This starts all services (`db`, `backend`, `frontend` on nginx) and is useful for testing the production deployment configuration.

## Tech Stack

| Layer | Tech |
|---|---|
| Backend | Go, chi router, pgx/v5, log/slog |
| Frontend | React 19, TypeScript (strict), Vite 6, Tailwind CSS v4 |
| Database | PostgreSQL 17 |
| Testing | `go test` (stdlib, no testify) / Vitest + Testing Library |
| Infrastructure | Docker Compose, multi-stage Dockerfiles |

## Architecture

- **Backend** serves a REST API under `/api/v1` — keywords CRUD, certificate listing/export, and monitor start/stop/status. A background goroutine polls the CT log, parses leaf certificates, and matches domains against stored keywords.
- **Frontend** is a single-page app that consumes the API via a thin `fetch`-based client. Custom hooks own all state; components are purely presentational.
- **Database** has three tables: `keywords`, `matched_certificates` (FK cascade), `monitor_state`.

## Conventions

- Refer to `backend/CLAUDE.md` and `frontend/CLAUDE.md` for language-specific style, testing, and file-naming rules.
- No `any`/`interface{}` in either codebase (except `writeJSON`'s data param).
- Tests don't require a running database — both sides use interface-based mocks.

## Environment Variables

See each sub-project's CLAUDE.md for full details. Key ones:

| Variable | Where | Required | Default |
|---|---|---|---|
| `DATABASE_URL` | backend | **yes** | — |
| `SERVER_PORT` | backend | no | `8080` |
| `CT_LOG_URL` | backend | no | `https://oak.ct.letsencrypt.org/2026h2` |
| `CORS_ALLOW_ORIGIN` | backend | no | `http://localhost:3000` |
| `VITE_API_URL` | frontend | no | `/api/v1` |

## Docker Compose Services

| Service | Image / Build | Ports |
|---|---|---|
| `db` | `postgres:17-alpine` | 5432 |
| `backend` | `./backend/Dockerfile` | 8080 |
| `frontend` | `./frontend/Dockerfile` (nginx) | 3000 → 80 |

`backend` waits for `db` healthcheck; `frontend` depends on `backend`.
