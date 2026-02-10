# CT Brand Monitor — Certificate Transparency PoC

A production-ready Certificate Transparency (CT) brand-protection monitor that watches public CT logs for newly issued certificates matching user-defined keywords. Automatically detects potential phishing attacks, domain abuse, and unauthorized certificate issuance for monitored brands.

**Tech Challenge Submission** — Full-stack TypeScript/Go application with REST API, real-time monitoring, reactive web UI, and PostgreSQL persistence.

## ⚠️ Critical Notice: CT Log Deprecation

**The default CT log (`oak.ct.letsencrypt.org/2026h2`) is deprecated and read-only.**

- **Read-only since**: November 30, 2025
- **Complete shutdown**: February 28, 2026
- **Impact**: No new certificates are being added to this log

**What this means for monitoring:**

- The log contains a frozen snapshot of certificates up to November 30, 2025
- **First run**: The monitor will process certificates from the current tree position (see [First-Run Optimization](#first-run-optimization))
- **Subsequent runs**: No new certificates will be detected (the log is frozen)
- **For continuous activity**: Set `MONITOR_REPROCESS_ON_IDLE=true` to enable demo/testing mode (see [Monitoring Mode Design](#monitoring-mode-design))

**Migrating to active CT logs:**

To monitor live certificate issuance, set the `CT_LOG_URL` environment variable to an active log

**Reference**: [Let's Encrypt RFC 6962 CT Logs End of Life](https://letsencrypt.org/2025/08/14/rfc-6962-logs-eol)

## Setup/Running Instructions

### Prerequisites

**Docker Compose (Recommended):**

- Docker Desktop or Docker Engine 27.0+
- Docker Compose 2.20+

**Local Development:**

- Go 1.23 or later
- Node.js 22 or later (with npm)
- PostgreSQL 17 (running separately, or via Docker)

### Quick Start with Docker Compose

The simplest way to run the entire application:

```bash
#Navigate to the project
cd SISAP-PoC

# Full stack: database + backend + frontend
docker compose up --build

# Application will be ready at:
#   Frontend:  http://localhost:3000
#   Backend:   http://localhost:8080
#   Database:  localhost:5432
```

This spins up:

- **PostgreSQL 17** database
- **Go backend** REST API (auto-waits for DB healthcheck)
- **React frontend** Nginx server with API proxying

Press `Ctrl+C` to stop. Use `docker compose down` to remove containers and clean up.

### Local Development Setup

Run services independently for faster iteration during development:

#### 1. Start PostgreSQL Database

```bash
# Option A: Docker Compose DB only
docker compose up -d db

# Option B: Use existing PostgreSQL 17 installation
# (configure DATABASE_URL below with your connection details)
```

#### 2. Start Backend API (Go)

```bash
cd backend
go mod tidy

# Run with default configuration
export DATABASE_URL="postgres://ctmonitor:ctmonitor_dev@localhost:5432/ct_monitor?sslmode=disable"
go run ./cmd/server

# The API will be available at http://localhost:8080/api/v1
```

**Windows PowerShell:**

```powershell
$env:DATABASE_URL="postgres://ctmonitor:ctmonitor_dev@localhost:5432/ct_monitor?sslmode=disable"
go run ./cmd/server
```

#### 3. Start Frontend Dev Server (React)

```bash
cd frontend
npm install
npm run dev

# Dev server runs on http://localhost:3000
# Automatically proxies /api → http://localhost:8080
```

### Testing

**Backend (Go):**

```bash
cd backend
go test ./...           # All tests (no database required — uses mocks)
go test -v ./...        # Verbose output
go test ./internal/handler/...  # Single package
```

**Frontend (React + TypeScript):**

```bash
cd frontend
npm run test            # Watch mode
npm run test:run        # CI mode (run once)
```

### Database Setup

Database is automatically initialized on first run via embedded migrations. No manual SQL needed.

**Manual reset:**

```bash
docker compose down -v pgdata  # Remove volume
docker compose up              # Recreate fresh DB
```

### Configuration

Configure via environment variables:

| Variable                    | Service  | Required | Default                                 | Description                                                                        |
| --------------------------- | -------- | -------- | --------------------------------------- | ---------------------------------------------------------------------------------- |
| `DATABASE_URL`              | Backend  | **yes**  | —                                       | PostgreSQL connection string                                                       |
| `SERVER_PORT`               | Backend  | no       | `8080`                                  | HTTP listen port                                                                   |
| `CT_LOG_URL`                | Backend  | no       | `https://oak.ct.letsencrypt.org/2026h2` | CT log endpoint (RFC 6962). **Note:** Default log is deprecated/read-only.         |
| `MONITOR_INTERVAL`          | Backend  | no       | `60s`                                   | Polling interval (e.g., `30s`, `2m`)                                               |
| `MONITOR_BATCH_SIZE`        | Backend  | no       | `100`                                   | Certificates per batch                                                             |
| `MONITOR_REPROCESS_ON_IDLE` | Backend  | no       | `true`                                  | Re-process last batch when idle. Set `true` for demo/testing. Default: production. |
| `CORS_ALLOW_ORIGIN`         | Backend  | no       | `http://localhost:3000`                 | CORS allowed origin                                                                |
| `VITE_API_URL`              | Frontend | no       | `/api/v1`                               | Backend API base URL                                                               |

**Local development example:**

```bash
export CT_LOG_URL=https://oak.ct.letsencrypt.org/2026h2
export MONITOR_INTERVAL=60s
go run ./cmd/server
```

## Implemented Features

### ✅ Core Features

- **CT Log Monitoring** — RFC 6962-compliant, polls every 60s, batch size 100. Connects to CT logs for certificate stream monitoring.

- **Keyword-Based Matching** — Substring matching against CN/SAN. Case-insensitive. Example: "acme" matches "secure.acme.com".

- **REST API** — `/api/v1` endpoints: Keywords CRUD, certificate listing/export, monitor control (start/stop/status).

- **Web UI** — React dashboard with keyword management, real-time certificate list, color-coded associations, monitor status.

- **Data Persistence** — PostgreSQL schema: `keywords`, `matched_certificates` (cascade delete on FK), `monitor_state`.

- **Status Metrics** — Total processed, last batch size, total matches, error count, monitor state.

### ✅ Bonus Features

- **CSV Export** — Download all matched certificates (10k limit). Columns: serial, domain, keyword, dates, issuer.

### 📊 Feature Highlights

#### Keyword Matching

```
User adds keyword:  "amazon"
Detected certs:     amazon.com ✓
                    amazongas.com ✓
                    secure-amazon.co.uk ✓
                    amazon-store.ru ✓
```

- **Substring matching**: Keyword anywhere in CN or SAN
- **Case-insensitive**: "AMAZON" = "amazon"
- **Multiple matches**: Single cert can match multiple keywords
- **Deduplication**: Same cert + keyword stored once

#### Monitor Lifecycle

```
Start Monitor → Retrieve tree size → Poll batches →
  Match keywords → Store matches →
  Stop Monitor (graceful) → Mark inactive
```

- **Polling**: 60s between batches (configurable)
- **Batch size**: 100 certificates per request
- **Auto-recovery**: Retries on error, logs exposed via status
- **Graceful shutdown**: Waits for active batch to complete

## Design Decisions & Ambiguities

### Architectural Choices

#### Batch Polling Architecture

**Decision**: Poll CT log every 60 seconds for batches of 100 certificates (As suggested from the specs).

**Rationale**: Simpler than real-time streaming. RFC 6962 doesn't require real-time ingestion. Reduces load on CT log and database. 60-second interval provides near real-time detection with acceptable latency for brand protection.

#### Monitoring Mode Design

The monitor has two operational modes controlled by `MONITOR_REPROCESS_ON_IDLE`:

**Production Mode (`false`)**

- Processes **only NEW entries** from the CT log
- When `GetSTH` returns same tree size, skips processing
- With read-only CT logs: Only first run shows activity

**Demo/Testing Mode (defaut: `true`)**

- Re-fetches and re-processes the **last batch** every 60 seconds
- Creates continuous activity for UI demonstration
- Use cases: Development, demos, testing

**Why this matters with deprecated logs:** The oak 2026h2 log is read-only (frozen), so production mode only processes data on first run. Demo mode enables continuous reprocessing for testing.

#### Database Cascade Delete

**Decision**: `matched_certificates.keyword_id` has `ON DELETE CASCADE`.

**Rationale**: When user deletes a keyword, associated matches are automatically removed. Maintains clean data without orphaned records. Simpler than manual cleanup in application code.

**Trade-off**: Cannot restore matches after keyword deletion. Acceptable for PoC scope (focus is forward-looking monitoring).

#### Substring Matching

**Decision**: Case-insensitive substring matching — "amazon" matches "amazon.com", "amazongas.com", "secure-amazon.com".

**Rationale**: Real-world phishing uses typosquatting patterns where legitimate brand names appear anywhere in malicious domains:

- Prefix phishing: `secure-amazon-login.com`
- Suffix phishing: `paypal-verify.net`
- Middle embedding: `apple-id-support.com`

Substring matching detects these patterns effectively. Case-insensitive comparison (`AMAZON` = `amazon`) handles case variations used in domain registration.

**Implementation**:

```go
// service/matcher/matcher.go
strings.Contains(strings.ToLower(domain), strings.ToLower(keyword))
```

**Alternatives Considered**:

| Approach                           | Complexity  | Time Investment | Use Case                                               |
| ---------------------------------- | ----------- | --------------- | ------------------------------------------------------ |
| Exact matching only                | ✅ Trivial  | 0 hours         | Would miss 90%+ of real phishing attempts              |
| Exact/substring toggle per keyword | 🟡 Low      | ~30 mins        | Adds flexibility for power users                       |
| Regex pattern support              | 🟡 Moderate | ~1 hour         | Advanced users could write `^amazon.*\.com$` patterns  |
| Fuzzy/Levenshtein matching         | 🔴 High     | ~2-3 hours      | Typo tolerance (`amazom` ≈ `amazon`), overkill for PoC |

**Decision for PoC**: Keep simple substring matching. Clear, explainable to evaluators, covers primary use case, and already implemented. If you want something more advance feel free to move forward with my application to discuss this in depth.

**Future Enhancement**: If additional time permits, adding an exact/substring boolean flag per keyword would provide flexibility without significant complexity. Regex support would benefit advanced users but requires pattern validation and error handling. Fuzzy matching would be over-engineered for an 8-hour PoC scope.

#### First-Run Optimization

**Decision**: On first start, query CT log's tree size and begin polling from current position, not from entry 0.

**Rationale**: Avoids processing millions of historical certificates. PoC focuses on detecting future threats, not historical analysis. Users get relevant matches immediately.

**With read-only CT logs**: First run processes certificates from current (frozen) tree position. Subsequent runs find no new entries (tree size unchanged). This is why only first run shows activity unless `MONITOR_REPROCESS_ON_IDLE=true`.

#### Color-Coded Keyword Association

**Decision**: Each keyword assigned unique color from 8-color palette. UI highlights matched certs with keyword's color.

**Rationale**: Visual scanning for users with many keywords. Immediate feedback on which keyword triggered match. No additional clicks needed.

#### Single CT Log Support

**Decision**: Backend configured with single `CT_LOG_URL` environment variable.

**Rationale**: Simplifies initial implementation. Most use cases monitor single trust root. Easy to extend in future with multi-log config.

### Implementation Interpretations

**Monitor Start/Stop Behavior**

- Monitor state persisted to `monitor_state` table
- Does **not** auto-resume on backend restart
- User must explicitly call `POST /api/v1/monitor/start` after restart
- Rationale: Explicit control safer for PoC. Production version would likely auto-resume.

**Export Limit**

- Maximum 10,000 certificates per export
- Rationale: Prevents memory exhaustion. 10k practical for spreadsheet tools.

**Pagination**

- `GET /api/v1/certificates` supports `page` and `per_page` query params
- Default: page 1, 50 certificates per page
- CSV export not paginated (exports all up to 10k limit)

**Keyword Validation**

- Min 1 character, max 255 characters
- Alphanumeric, dots, hyphens, underscores allowed
- Case-insensitive storage (stored as-is, matched case-insensitively)

### Trade-offs

**Performance vs. Simplicity**

- No caching layer; query database directly
- Rationale: PoC scope. Database queries fast for typical use (100s-1000s matches). Production could add Redis caching.

**Database Abstraction**

- Raw SQL with `pgx/v5`, no ORM
- Rationale: Explicit control over queries. No ORM overhead. Embedded migrations via `//go:embed`.
- Trade-off: More SQL code to maintain. Acceptable for PoC (few tables, simple queries).
- **Security**: All SQL queries use parameterized placeholders (`$1, $2, $3`), preventing SQL injection.
- ORM could provide an enhancement but was a little unnecesary from my own perspective from the PoC view.

**Frontend State Management**

- Custom React hooks with `useState` — no Redux/Zustand
- Rationale: Simple state (keywords, certificates, monitor status). Hooks are built-in. Each component owns lifecycle.

## Limitations & Known Bugs

### Scope Limitations

- **Single CT Log** — Monitors one log at a time. Organizations using multiple CAs must coordinate via environment variable changes. Future: multi-log config.

- **Batch Polling Architecture** — 100-cert batches every 60s, not real-time streaming. Max ~1-2 minute latency. With deprecated oak 2026h2 log: No new certificates added (read-only since November 30, 2025). Updates only on first run unless `MONITOR_REPROCESS_ON_IDLE=true`. Future: Streaming via CT log gossip protocol.

- **No User Authentication** — Single-user PoC. No login, API keys, or user isolation. All keywords/certificates visible to anyone with access. Future: OAuth2/JWT, per-user keyword lists, audit logging.

- **Keyword Deletion is Permanent** — Deleting keyword also deletes associated matched certificates (cascade delete). Cannot recover history. Future: Soft delete with archival or delete confirmation with export.

- **Export Limit of 10,000 Certificates** — CSV export truncates to 10k records. Organizations with >10k matches must export in chunks or query API directly. Future: Pagination-based export or streaming response.

- **No API Rate Limiting** — Endpoints have no rate limiting. Vulnerable to DoS if exposed publicly (not intended for PoC). Future: Rate limiting middleware (e.g., "1000 req/min per IP").

### Known Bugs

**None reported.** Application tested with normal operation, concurrent keyword CRUD, monitor lifecycle, CSV export with large datasets, database reconnection. All core functionality works as designed.

### Testing Coverage

Comprehensive coverage via Go stdlib `testing` and Vitest + Testing Library.

**Backend (Go):**

- Handler tests: Success paths, validation, error handling, all endpoints with mock repositories
- Service tests: CT log client, matcher logic, monitor lifecycle
- Repository tests: Interface-based mocks (no database), keyword validation, certificate storage
- Edge cases: Invalid IDs, concurrent operations, parsing failures, network errors, state transitions
- Run: `go test ./...` (fast, no external dependencies)

**Frontend (Vitest + Testing Library):**

- Component tests: Custom hooks (`useKeywords`, `useMonitor`, `useCertificates`), UI components
- API client tests: Fetch mocking, error handling, pagination
- User interaction: Keyword CRUD flows, monitor controls, certificate filtering
- Validation: Input validation, error messages, loading states
- Run: `npm run test:run` (CI) or `npm run test` (watch)

**Integration Testing:**

- Full Docker Compose stack tested manually with end-to-end workflows
- Database initialization, backend startup, frontend proxy, all user workflows verified

## Project Structure

```
SISAP-PoC/
├── README.md                          # This file
├── CLAUDE.md                          # Development guide
├── docker-compose.yml                 # Docker Compose configuration
│
├── backend/                           # Go REST API
│   ├── CLAUDE.md                      # Backend-specific guide
│   ├── Dockerfile                     # Multi-stage build
│   ├── cmd/server/main.go             # Entry point
│   ├── internal/
│   │   ├── config/                    # Config from environment
│   │   ├── database/                  # PostgreSQL connection, migrations
│   │   ├── handler/                   # HTTP handlers for REST API
│   │   ├── middleware/                # CORS, logging, panic recovery
│   │   ├── model/                     # Domain structs (Keyword, Certificate, etc.)
│   │   ├── repository/                # PostgreSQL queries
│   │   └── service/
│   │       ├── ctlog/                 # CT Log HTTP client, certificate parser
│   │       ├── matcher/               # Keyword-to-domain matching
│   │       └── monitor/               # Background polling loop
│   └── go.mod, go.sum                 # Go dependencies
│
├── frontend/                          # React SPA
│   ├── CLAUDE.md                      # Frontend-specific guide
│   ├── Dockerfile                     # Multi-stage build with Nginx
│   ├── vite.config.ts                 # Vite dev server config
│   ├── tsconfig.json                  # TypeScript strict mode
│   ├── nginx.conf                     # Nginx reverse proxy config
│   ├── src/
│   │   ├── api/                       # HTTP client, endpoint modules
│   │   ├── components/                # React components by feature
│   │   ├── hooks/                     # Custom React hooks (state management)
│   │   ├── types/                     # TypeScript interfaces
│   │   ├── utils/                     # Helper functions
│   │   ├── App.tsx                    # Root component
│   │   └── main.tsx                   # Entry point
│   ├── package.json                   # npm dependencies
│   └── index.html                     # SPA entry HTML
```

**Note:** Database schema is managed by the backend via embedded migrations in `backend/internal/database/migrations/`.

## API Documentation

### Keywords API

- `GET /api/v1/keywords` — List all keywords
- `POST /api/v1/keywords` — Create keyword (body: `{ "value": "apple" }`)
- `DELETE /api/v1/keywords/{id}` — Delete keyword

### Certificates API

- `GET /api/v1/certificates?keyword=amazon&page=1&per_page=50` — List matched certificates
  - Response: `{ certificates: [...], total: 243, page: 1, perPage: 50 }`
- `GET /api/v1/certificates/export?keyword=amazon` — Export to CSV
  - Headers: `Content-Type: text/csv`, `Content-Disposition: attachment`

### Monitor API

- `POST /api/v1/monitor/start` — Start monitor
- `POST /api/v1/monitor/stop` — Stop monitor
- `GET /api/v1/monitor/status` — Get status
  - Response: `{ active: true, totalProcessed: 50000, lastBatchSize: 100, totalMatches: 247, errors: 2, lastPollTime: "..." }`

### Error Responses

All endpoints return errors in this format:

```json
{
  "error": "Keyword not found",
  "status": 404
}
```

Common status codes: `200 OK`, `201 Created`, `204 No Content`, `400 Bad Request`, `404 Not Found`, `500 Internal Server Error`

## Tech Stack

| Component              | Technology     | Version |
| ---------------------- | -------------- | ------- |
| **Backend**            | Go             | 1.23    |
| **Backend Framework**  | chi (router)   | v5      |
| **Database Driver**    | pgx            | v5      |
| **Database**           | PostgreSQL     | 17      |
| **Logging**            | log/slog       | stdlib  |
| **Frontend Framework** | React          | 19      |
| **Frontend Language**  | TypeScript     | 5.6     |
| **Build Tool**         | Vite           | 6       |
| **Styling**            | Tailwind CSS   | v4      |
| **Testing (Backend)**  | go test        | stdlib  |
| **Testing (Frontend)** | Vitest         | 4       |
| **Container Runtime**  | Docker         | 27.0+   |
| **Orchestration**      | Docker Compose | 2.20+   |

---

**Status**: Production-ready PoC for Tech Challenge submission
