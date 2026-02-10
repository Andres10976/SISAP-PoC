# Backend — CT Monitor API

Go REST API that monitors Certificate Transparency logs for domain keyword matches.

## Quick Reference

```bash
# Run (requires DATABASE_URL)
DATABASE_URL="postgres://ctmonitor:ctmonitor_dev@localhost:5432/ct_monitor?sslmode=disable" go run ./cmd/server

# Test (no database needed — handlers/services use interface mocks)
go test ./...

# Test single package
go test ./internal/handler/...

# Build
go build -o server ./cmd/server

# Start database (from repo root)
docker compose up -d db
```

## Environment Variables

| Variable | Required | Default | Description |
|---|---|---|---|
| `DATABASE_URL` | **yes** | — | PostgreSQL connection string |
| `SERVER_PORT` | no | `8080` | HTTP listen port |
| `CT_LOG_URL` | no | `https://oak.ct.letsencrypt.org/2026h2` | CT log endpoint |
| `MONITOR_INTERVAL` | no | `60s` | Polling interval (Go duration) |
| `MONITOR_BATCH_SIZE` | no | `100` | Entries per batch |
| `CORS_ALLOW_ORIGIN` | no | `http://localhost:3000` | Allowed CORS origin |

## Architecture

```
cmd/server/main.go          Entry point — wires everything, graceful shutdown
internal/
  config/                    Env-based config (Load panics on missing DATABASE_URL)
  database/                  pgxpool connection + embedded SQL migrations
  model/                     Domain structs (Keyword, MatchedCertificate, MonitorState)
  repository/                PostgreSQL queries (one repo per model)
  handler/                   HTTP handlers (chi router, JSON responses)
  middleware/                 CORS, request ID, structured logging, panic recovery
  service/
    ctlog/                   CT log HTTP client + leaf certificate parser
    matcher/                 Keyword-to-domain substring matching
    monitor/                 Background polling loop (start/stop lifecycle)
```

### Key patterns

- **Dependency injection via interfaces** — handlers define small interfaces (`keywordStore`, `certStore`) rather than depending on concrete repos. Tests use inline mock structs.
- **chi router** — routes registered under `/api/v1` via `RegisterRoutes(chi.Router)` on each handler.
- **Structured logging** — `log/slog` with JSON output. No third-party logger.
- **Migrations** — single SQL file embedded with `//go:embed`, run on startup via `database.Migrate()`. Idempotent (`CREATE TABLE IF NOT EXISTS`).
- **No ORM** — raw SQL with `pgx/v5`. Repositories return model structs directly.

## API Routes

All under `/api/v1`:

| Method | Path | Handler |
|---|---|---|
| GET | `/keywords` | List all keywords |
| POST | `/keywords` | Create keyword (`{"value":"..."}`) |
| DELETE | `/keywords/{id}` | Delete keyword by ID |
| GET | `/certificates` | List matched certificates (query: `keyword`, `page`, `per_page`) |
| GET | `/certificates/export` | CSV export |
| POST | `/monitor/start` | Start background monitor |
| POST | `/monitor/stop` | Stop background monitor |
| GET | `/monitor/status` | Current monitor state |

## Conventions

- **Error handling**: explicit `if err != nil` — never swallow errors. Repository returns `repository.ErrNotFound`; handlers map it to 404.
- **Response helpers**: `writeJSON(w, status, data)` and `writeError(w, status, message)` in `handler/response.go`.
- **Naming**: exported = `PascalCase`, unexported = `camelCase`, files = `lowercase.go`.
- **Test style**: stdlib `testing` only — no testify. Mocks are local struct literals with func fields. Use `httptest.NewRecorder` + `httptest.NewRequest` for handler tests. Chi URL params set via `chi.NewRouteContext()`.
- **No `any` or `interface{}` in signatures** except `writeJSON`'s data param.
- **Imports**: stdlib first, blank line, third-party, blank line, internal packages.

## Database

PostgreSQL 17. Three tables: `keywords`, `matched_certificates`, `monitor_state`. Schema in `internal/database/migrations/001_init.sql`. Foreign key from `matched_certificates.keyword_id` to `keywords.id` with `ON DELETE CASCADE`.

## Docker

```bash
# Full stack from repo root
docker compose up --build

# Backend only
docker build -t ct-backend ./backend
docker run -e DATABASE_URL=... ct-backend
```

Multi-stage Dockerfile: builds with `golang:1.25-alpine`, runs on `alpine:3.21`.
