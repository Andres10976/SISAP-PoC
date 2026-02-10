# Backend Architecture Specification

## Brand Protection Monitor — Go Backend

### 1. Overview

The backend is a Go service responsible for:

- Consuming Certificate Transparency (CT) log entries from Let's Encrypt Oak 2026h2
- Parsing X.509 certificates to extract domain information (CN + SANs)
- Matching domains against user-defined keywords for brand abuse detection
- Serving a REST API for the React frontend
- Exporting matched certificates as CSV

The service follows idiomatic Go conventions: flat package structure inside `internal/`,
explicit error handling, context propagation, and minimal external dependencies.

---

### 2. Project Structure

```
backend/
├── cmd/
│   └── server/
│       └── main.go                 # Entry point, wiring, graceful shutdown
├── internal/
│   ├── config/
│   │   └── config.go               # Environment-based configuration
│   ├── database/
│   │   ├── postgres.go             # Connection pool setup (pgx)
│   │   └── migrate.go              # Embedded SQL migrations
│   ├── model/
│   │   ├── keyword.go              # Keyword domain type
│   │   ├── certificate.go          # MatchedCertificate domain type
│   │   └── monitor.go              # MonitorState domain type
│   ├── repository/
│   │   ├── keyword.go              # Keyword CRUD operations
│   │   ├── certificate.go          # Certificate CRUD + CSV query
│   │   └── monitor.go              # Monitor state singleton ops
│   ├── service/
│   │   ├── ctlog/
│   │   │   ├── client.go           # HTTP client for CT Log RFC 6962 API
│   │   │   └── parser.go           # Binary leaf_input parser + x509
│   │   ├── monitor/
│   │   │   └── monitor.go          # Polling loop orchestrator
│   │   └── matcher/
│   │       └── matcher.go          # Keyword-to-certificate matching
│   ├── handler/
│   │   ├── keyword.go              # REST handlers for /api/v1/keywords
│   │   ├── certificate.go          # REST handlers for /api/v1/certificates
│   │   ├── monitor.go              # REST handlers for /api/v1/monitor
│   │   ├── export.go               # CSV export handler
│   │   └── response.go             # Shared JSON response helpers
│   └── middleware/
│       ├── cors.go                 # CORS policy
│       ├── logging.go              # Structured request logging
│       ├── recovery.go             # Panic recovery
│       └── requestid.go            # X-Request-ID propagation
├── migrations/
│   └── 001_init.sql                # SQL schema (also embedded)
├── go.mod
├── go.sum
└── Dockerfile
```

---

### 3. Configuration

All configuration is read from environment variables with sensible defaults.
No config files — this keeps the container deployment straightforward.

```go
// internal/config/config.go
package config

import (
    "os"
    "strconv"
    "time"
)

type Config struct {
    ServerPort       string        // SERVER_PORT        (default: "8080")
    DatabaseURL      string        // DATABASE_URL       (required)
    CTLogURL         string        // CT_LOG_URL         (default: "https://oak.ct.letsencrypt.org/2026h2")
    MonitorInterval  time.Duration // MONITOR_INTERVAL   (default: 60s)
    MonitorBatchSize int           // MONITOR_BATCH_SIZE (default: 100)
    CORSAllowOrigin  string        // CORS_ALLOW_ORIGIN  (default: "http://localhost:3000")
}

func Load() *Config {
    return &Config{
        ServerPort:       getEnv("SERVER_PORT", "8080"),
        DatabaseURL:      requireEnv("DATABASE_URL"),
        CTLogURL:         getEnv("CT_LOG_URL", "https://oak.ct.letsencrypt.org/2026h2"),
        MonitorInterval:  getDuration("MONITOR_INTERVAL", 60*time.Second),
        MonitorBatchSize: getInt("MONITOR_BATCH_SIZE", 100),
        CORSAllowOrigin:  getEnv("CORS_ALLOW_ORIGIN", "http://localhost:3000"),
    }
}
```

| Variable             | Required | Default                                      | Description                        |
|----------------------|----------|----------------------------------------------|------------------------------------|
| `SERVER_PORT`        | No       | `8080`                                       | HTTP server listen port            |
| `DATABASE_URL`       | **Yes**  | —                                            | PostgreSQL connection string       |
| `CT_LOG_URL`         | No       | `https://oak.ct.letsencrypt.org/2026h2`      | CT Log base URL                    |
| `MONITOR_INTERVAL`   | No       | `60s`                                        | Polling interval for CT Log        |
| `MONITOR_BATCH_SIZE` | No       | `100`                                        | Entries to fetch per cycle         |
| `CORS_ALLOW_ORIGIN`  | No       | `http://localhost:3000`                      | Allowed CORS origin for frontend   |

---

### 4. Database Schema

PostgreSQL 17. All queries use parameterized statements via `pgx` to prevent SQL injection.

```sql
-- Keywords being monitored for brand abuse
CREATE TABLE IF NOT EXISTS keywords (
    id         SERIAL PRIMARY KEY,
    value      TEXT        NOT NULL UNIQUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Certificates that matched at least one keyword
CREATE TABLE IF NOT EXISTS matched_certificates (
    id             SERIAL PRIMARY KEY,
    serial_number  TEXT        NOT NULL,
    common_name    TEXT        NOT NULL,
    sans           TEXT[]      NOT NULL DEFAULT '{}',
    issuer         TEXT        NOT NULL,
    not_before     TIMESTAMPTZ,
    not_after      TIMESTAMPTZ,
    keyword_id     INTEGER     NOT NULL REFERENCES keywords(id) ON DELETE CASCADE,
    matched_domain TEXT        NOT NULL,
    ct_log_index   BIGINT      NOT NULL,
    discovered_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    UNIQUE(serial_number, keyword_id)
);

CREATE INDEX IF NOT EXISTS idx_matched_certs_keyword
    ON matched_certificates(keyword_id);
CREATE INDEX IF NOT EXISTS idx_matched_certs_discovered
    ON matched_certificates(discovered_at DESC);

-- Singleton row tracking the monitor's progress
CREATE TABLE IF NOT EXISTS monitor_state (
    id                     INTEGER PRIMARY KEY DEFAULT 1 CHECK (id = 1),
    last_processed_index   BIGINT  NOT NULL DEFAULT 0,
    last_tree_size         BIGINT  NOT NULL DEFAULT 0,
    last_run_at            TIMESTAMPTZ,
    total_processed        BIGINT  NOT NULL DEFAULT 0,
    certs_in_last_cycle    INTEGER NOT NULL DEFAULT 0,
    matches_in_last_cycle  INTEGER NOT NULL DEFAULT 0,
    is_running             BOOLEAN NOT NULL DEFAULT FALSE,
    updated_at             TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

INSERT INTO monitor_state (id) VALUES (1) ON CONFLICT (id) DO NOTHING;
```

**Design rationale:**

- `matched_certificates.sans` is `TEXT[]` — PostgreSQL native array avoids a join table for a PoC.
- `UNIQUE(serial_number, keyword_id)` prevents storing the same cert–keyword pair twice
  if the same certificate appears across multiple polling cycles.
- `ON DELETE CASCADE` on `keyword_id` ensures matched certs are cleaned up when a keyword is removed.
- `monitor_state` uses a singleton pattern (`CHECK (id = 1)`) — there is exactly one monitor.
- `idx_matched_certs_discovered` supports the dashboard's default ordering (newest first).

---

### 5. REST API Contract

Base path: `/api/v1`

All responses use `Content-Type: application/json` unless otherwise noted.
Error responses follow a consistent shape:

```json
{ "error": "human-readable message" }
```

#### 5.1 Keywords

**`GET /api/v1/keywords`** — List all monitored keywords

Response `200`:
```json
{
  "keywords": [
    { "id": 1, "value": "paypal", "created_at": "2026-02-09T12:00:00Z" },
    { "id": 2, "value": "google", "created_at": "2026-02-09T12:01:00Z" }
  ]
}
```

---

**`POST /api/v1/keywords`** — Add a keyword

Request:
```json
{ "value": "paypal" }
```

Response `201`:
```json
{ "id": 1, "value": "paypal", "created_at": "2026-02-09T12:00:00Z" }
```

Errors:
- `400` — empty or whitespace-only value
- `409` — keyword already exists

---

**`DELETE /api/v1/keywords/{id}`** — Remove a keyword

Response `204` — No content

Errors:
- `404` — keyword not found

---

#### 5.2 Certificates

**`GET /api/v1/certificates`** — List matched certificates (paginated)

Query parameters:

| Param      | Type   | Default | Description              |
|------------|--------|---------|--------------------------|
| `page`     | int    | `1`     | Page number (1-based)    |
| `per_page` | int    | `20`    | Items per page (max 100) |
| `keyword`  | int    | —       | Filter by keyword ID     |

Response `200`:
```json
{
  "certificates": [
    {
      "id": 1,
      "serial_number": "04a1b2c3d4e5f6",
      "common_name": "paypal-secure.example.com",
      "sans": ["paypal-secure.example.com", "www.paypal-secure.example.com"],
      "issuer": "R11",
      "not_before": "2026-02-09T00:00:00Z",
      "not_after": "2026-05-10T00:00:00Z",
      "keyword_id": 1,
      "keyword_value": "paypal",
      "matched_domain": "paypal-secure.example.com",
      "ct_log_index": 130815650,
      "discovered_at": "2026-02-09T12:05:30Z"
    }
  ],
  "total": 42,
  "page": 1,
  "per_page": 20
}
```

> The `keyword_value` field is joined from the keywords table so the frontend
> can display the keyword name without an extra round-trip.

---

**`GET /api/v1/certificates/export`** — Download CSV

Response `200`:
```
Content-Type: text/csv
Content-Disposition: attachment; filename="matched_certificates.csv"
```

CSV columns:
```
id,serial_number,common_name,sans,issuer,not_before,not_after,keyword,matched_domain,ct_log_index,discovered_at
```

`sans` is serialized as a semicolon-separated string within the CSV field.

---

#### 5.3 Monitor

**`GET /api/v1/monitor/status`** — Get monitor status and metrics

Response `200`:
```json
{
  "is_running": true,
  "last_run_at": "2026-02-09T12:05:00Z",
  "last_tree_size": 130815692,
  "last_processed_index": 130815692,
  "total_processed": 500,
  "certs_in_last_cycle": 100,
  "matches_in_last_cycle": 3,
  "updated_at": "2026-02-09T12:05:30Z"
}
```

---

**`POST /api/v1/monitor/start`** — Start the monitoring loop

Response `200`:
```json
{ "message": "Monitor started" }
```

Errors:
- `409` — Monitor is already running

---

**`POST /api/v1/monitor/stop`** — Stop the monitoring loop

Response `200`:
```json
{ "message": "Monitor stopped" }
```

Errors:
- `409` — Monitor is not running

---

### 6. Core Services

#### 6.1 CT Log Client (`internal/service/ctlog/client.go`)

Implements two RFC 6962 endpoints against the configured CT Log URL.

```go
package ctlog

import (
    "context"
    "encoding/json"
    "fmt"
    "net/http"
    "time"
)

// STH represents a Signed Tree Head response (RFC 6962 §4.3).
type STH struct {
    TreeSize  int64  `json:"tree_size"`
    Timestamp int64  `json:"timestamp"`
    RootHash  string `json:"sha256_root_hash"`
}

// RawEntry represents a single entry from get-entries (RFC 6962 §4.6).
type RawEntry struct {
    LeafInput []byte `json:"leaf_input"` // base64-decoded by json.Decoder
    ExtraData []byte `json:"extra_data"`
}

// Client talks to a Certificate Transparency log over HTTP.
type Client struct {
    baseURL    string
    httpClient *http.Client
}

func NewClient(baseURL string) *Client {
    return &Client{
        baseURL: baseURL,
        httpClient: &http.Client{
            Timeout: 30 * time.Second,
        },
    }
}

// GetSTH retrieves the latest Signed Tree Head.
// GET {baseURL}/ct/v1/get-sth
func (c *Client) GetSTH(ctx context.Context) (*STH, error) {
    url := fmt.Sprintf("%s/ct/v1/get-sth", c.baseURL)
    req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
    if err != nil {
        return nil, fmt.Errorf("create STH request: %w", err)
    }

    resp, err := c.httpClient.Do(req)
    if err != nil {
        return nil, fmt.Errorf("fetch STH: %w", err)
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        return nil, fmt.Errorf("STH returned status %d", resp.StatusCode)
    }

    var sth STH
    if err := json.NewDecoder(resp.Body).Decode(&sth); err != nil {
        return nil, fmt.Errorf("decode STH: %w", err)
    }
    return &sth, nil
}

// GetEntries retrieves log entries in range [start, end] inclusive.
// GET {baseURL}/ct/v1/get-entries?start={start}&end={end}
func (c *Client) GetEntries(ctx context.Context, start, end int64) ([]RawEntry, error) {
    url := fmt.Sprintf("%s/ct/v1/get-entries?start=%d&end=%d", c.baseURL, start, end)
    req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
    if err != nil {
        return nil, fmt.Errorf("create entries request: %w", err)
    }

    resp, err := c.httpClient.Do(req)
    if err != nil {
        return nil, fmt.Errorf("fetch entries: %w", err)
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        return nil, fmt.Errorf("get-entries returned status %d", resp.StatusCode)
    }

    var result struct {
        Entries []RawEntry `json:"entries"`
    }
    if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
        return nil, fmt.Errorf("decode entries: %w", err)
    }
    return result.Entries, nil
}
```

> **Note:** The `leaf_input` and `extra_data` fields come as base64 strings
> in the JSON. Go's `encoding/json` decodes `[]byte` fields from base64
> automatically, so no manual base64 decoding is needed.

---

#### 6.2 Certificate Parser (`internal/service/ctlog/parser.go`)

Parses the binary `leaf_input` from a CT Log entry per RFC 6962 §3.4.

**Binary layout of `MerkleTreeLeaf`:**

```
Offset  Size    Field
─────────────────────────────────────────────
0       1       version          (0x00 = v1)
1       1       leaf_type        (0x00 = timestamped_entry)

TimestampedEntry:
2       8       timestamp        (uint64 big-endian, ms since epoch)
10      2       entry_type       (uint16 big-endian)
                                  0x0000 = x509_entry
                                  0x0001 = precert_entry

For x509_entry (entry_type = 0):
12      3       cert_length      (uint24 big-endian)
15      N       cert_data        (DER-encoded X.509 certificate)

For precert_entry (entry_type = 1):
12      32      issuer_key_hash  (SHA-256 of issuer public key)
44      3       tbs_cert_length  (uint24 big-endian)
47      N       tbs_cert_data    (DER-encoded TBSCertificate)
```

```go
package ctlog

import (
    "crypto/x509"
    "encoding/binary"
    "errors"
    "fmt"
    "time"
)

var (
    ErrTooShort     = errors.New("leaf input too short")
    ErrUnknownType  = errors.New("unknown entry type")
    ErrParseFailed  = errors.New("certificate parse failed")
)

// ParsedCertificate holds the fields extracted from a CT log entry
// that are relevant for keyword matching and display.
type ParsedCertificate struct {
    Timestamp  time.Time
    Serial     string   // hex-encoded serial number
    CommonName string
    SANs       []string // DNS Subject Alternative Names
    Issuer     string   // Issuer Common Name or Organization
    NotBefore  time.Time
    NotAfter   time.Time
}

// ParseLeafInput decodes a MerkleTreeLeaf binary blob into a ParsedCertificate.
// It handles both x509_entry and precert_entry types.
func ParseLeafInput(data []byte) (*ParsedCertificate, error) {
    if len(data) < 15 {
        return nil, ErrTooShort
    }

    // Bytes 2–9: timestamp (uint64 big-endian, milliseconds since epoch)
    timestamp := binary.BigEndian.Uint64(data[2:10])

    // Bytes 10–11: entry type
    entryType := binary.BigEndian.Uint16(data[10:12])

    var certDER []byte

    switch entryType {
    case 0: // x509_entry
        certLen := readUint24(data[12:15])
        end := 15 + certLen
        if len(data) < end {
            return nil, ErrTooShort
        }
        certDER = data[15:end]

    case 1: // precert_entry
        if len(data) < 47 {
            return nil, ErrTooShort
        }
        // Skip 32-byte issuer_key_hash at offset 12
        tbsLen := readUint24(data[44:47])
        end := 47 + tbsLen
        if len(data) < end {
            return nil, ErrTooShort
        }
        certDER = data[47:end]

    default:
        return nil, fmt.Errorf("%w: %d", ErrUnknownType, entryType)
    }

    cert, err := x509.ParseCertificate(certDER)
    if err != nil {
        return nil, fmt.Errorf("%w: %v", ErrParseFailed, err)
    }

    issuer := cert.Issuer.CommonName
    if issuer == "" && len(cert.Issuer.Organization) > 0 {
        issuer = cert.Issuer.Organization[0]
    }

    return &ParsedCertificate{
        Timestamp:  time.UnixMilli(int64(timestamp)),
        Serial:     cert.SerialNumber.Text(16),
        CommonName: cert.Subject.CommonName,
        SANs:       cert.DNSNames,
        Issuer:     issuer,
        NotBefore:  cert.NotBefore,
        NotAfter:   cert.NotAfter,
    }, nil
}

// readUint24 reads a 3-byte big-endian unsigned integer.
func readUint24(b []byte) int {
    return int(b[0])<<16 | int(b[1])<<8 | int(b[2])
}
```

> **Precert parsing note:** `x509.ParseCertificate` may fail on some
> precert TBSCertificate blobs due to the CT poison extension.
> In the monitor loop, parse errors are logged and the entry is skipped
> — this is acceptable for a PoC since most entries are standard x509.

---

#### 6.3 Monitor Service (`internal/service/monitor/monitor.go`)

The monitor runs as a background goroutine, polling the CT Log on a fixed interval.

**Lifecycle:**
1. `Start()` launches the polling goroutine and sets `is_running = true` in DB.
2. Each tick calls `processBatch()`.
3. `Stop()` cancels the goroutine's context and sets `is_running = false`.

**Batch processing algorithm:**

```
1. GET /ct/v1/get-sth  →  tree_size
2. Load monitor_state from DB  →  last_processed_index
3. If first run (last_processed_index == 0):
       start = tree_size - batch_size
   Else:
       start = last_processed_index
4. end = min(start + batch_size - 1, tree_size - 1)
5. If start > end  →  no new entries, skip
6. GET /ct/v1/get-entries?start={start}&end={end}
7. Load all keywords from DB
8. For each entry:
     a. Parse leaf_input → ParsedCertificate
     b. Match CN and SANs against keywords
     c. For each match → INSERT into matched_certificates (ON CONFLICT DO NOTHING)
9. UPDATE monitor_state with new index, metrics, timestamp
```

```go
package monitor

import (
    "context"
    "log/slog"
    "sync"
    "time"

    "github.com/andres10976/SISAP-PoC/backend/internal/model"
    "github.com/andres10976/SISAP-PoC/backend/internal/repository"
    "github.com/andres10976/SISAP-PoC/backend/internal/service/ctlog"
    "github.com/andres10976/SISAP-PoC/backend/internal/service/matcher"
)

type Monitor struct {
    ctClient  *ctlog.Client
    keywords  *repository.KeywordRepository
    certs     *repository.CertificateRepository
    state     *repository.MonitorRepository
    batchSize int
    interval  time.Duration

    mu     sync.Mutex
    cancel context.CancelFunc
}

func New(
    ct *ctlog.Client,
    kw *repository.KeywordRepository,
    cert *repository.CertificateRepository,
    st *repository.MonitorRepository,
    batchSize int,
    interval time.Duration,
) *Monitor {
    return &Monitor{
        ctClient:  ct,
        keywords:  kw,
        certs:     cert,
        state:     st,
        batchSize: batchSize,
        interval:  interval,
    }
}

// Start launches the background monitoring loop.
// Returns an error if the monitor is already running.
func (m *Monitor) Start(ctx context.Context) error {
    m.mu.Lock()
    defer m.mu.Unlock()

    if m.cancel != nil {
        return ErrAlreadyRunning
    }

    monCtx, cancel := context.WithCancel(ctx)
    m.cancel = cancel

    if err := m.state.SetRunning(ctx, true); err != nil {
        cancel()
        m.cancel = nil
        return err
    }

    go m.run(monCtx)
    return nil
}

// Stop halts the monitoring loop.
func (m *Monitor) Stop(ctx context.Context) error {
    m.mu.Lock()
    defer m.mu.Unlock()

    if m.cancel == nil {
        return ErrNotRunning
    }

    m.cancel()
    m.cancel = nil

    return m.state.SetRunning(ctx, false)
}

// IsRunning returns whether the monitor loop is active.
func (m *Monitor) IsRunning() bool {
    m.mu.Lock()
    defer m.mu.Unlock()
    return m.cancel != nil
}

func (m *Monitor) run(ctx context.Context) {
    // Execute immediately, then on interval
    m.processBatch(ctx)

    ticker := time.NewTicker(m.interval)
    defer ticker.Stop()

    for {
        select {
        case <-ctx.Done():
            return
        case <-ticker.C:
            m.processBatch(ctx)
        }
    }
}

func (m *Monitor) processBatch(ctx context.Context) {
    logger := slog.Default()

    // 1. Get current Signed Tree Head
    sth, err := m.ctClient.GetSTH(ctx)
    if err != nil {
        logger.Error("failed to get STH", "error", err)
        return
    }

    // 2. Load current monitor state
    state, err := m.state.Get(ctx)
    if err != nil {
        logger.Error("failed to get monitor state", "error", err)
        return
    }

    // 3. Calculate batch range
    start := state.LastProcessedIndex
    if start == 0 {
        start = max(0, sth.TreeSize-int64(m.batchSize))
    }
    end := min(start+int64(m.batchSize)-1, sth.TreeSize-1)

    if start > end {
        logger.Info("no new entries to process")
        return
    }

    logger.Info("fetching CT log entries",
        "start", start, "end", end, "tree_size", sth.TreeSize)

    // 4. Fetch entries
    entries, err := m.ctClient.GetEntries(ctx, start, end)
    if err != nil {
        logger.Error("failed to fetch entries", "error", err)
        return
    }

    // 5. Load keywords
    keywords, err := m.keywords.List(ctx)
    if err != nil {
        logger.Error("failed to load keywords", "error", err)
        return
    }

    if len(keywords) == 0 {
        logger.Info("no keywords configured, skipping matching")
        m.updateState(ctx, state, end, sth.TreeSize, len(entries), 0)
        return
    }

    // 6. Parse and match
    matchCount := 0
    parseErrors := 0
    for i, entry := range entries {
        cert, err := ctlog.ParseLeafInput(entry.LeafInput)
        if err != nil {
            parseErrors++
            continue
        }

        matches := matcher.Match(cert, keywords)
        for _, match := range matches {
            err := m.certs.Create(ctx, &model.MatchedCertificate{
                SerialNumber:  cert.Serial,
                CommonName:    cert.CommonName,
                SANs:          cert.SANs,
                Issuer:        cert.Issuer,
                NotBefore:     cert.NotBefore,
                NotAfter:      cert.NotAfter,
                KeywordID:     match.KeywordID,
                MatchedDomain: match.MatchedDomain,
                CTLogIndex:    start + int64(i),
            })
            if err != nil {
                logger.Error("failed to store match", "error", err, "domain", match.MatchedDomain)
                continue
            }
            matchCount++
        }
    }

    logger.Info("batch processed",
        "entries", len(entries),
        "parse_errors", parseErrors,
        "matches", matchCount,
    )

    // 7. Update state
    m.updateState(ctx, state, end, sth.TreeSize, len(entries), matchCount)
}

func (m *Monitor) updateState(
    ctx context.Context,
    prev *model.MonitorState,
    endIndex, treeSize int64,
    processed, matches int,
) {
    err := m.state.Update(ctx, &model.MonitorState{
        LastProcessedIndex: endIndex + 1,
        LastTreeSize:       treeSize,
        TotalProcessed:     prev.TotalProcessed + int64(processed),
        CertsInLastCycle:   processed,
        MatchesInLastCycle: matches,
        IsRunning:          true,
    })
    if err != nil {
        slog.Error("failed to update monitor state", "error", err)
    }
}
```

---

#### 6.4 Keyword Matcher (`internal/service/matcher/matcher.go`)

Case-insensitive substring matching against both CN and SANs.

```go
package matcher

import (
    "strings"

    "github.com/andres10976/SISAP-PoC/backend/internal/model"
    "github.com/andres10976/SISAP-PoC/backend/internal/service/ctlog"
)

// MatchResult pairs a keyword ID with the domain that triggered the match.
type MatchResult struct {
    KeywordID     int
    MatchedDomain string
}

// Match checks a parsed certificate against all keywords.
// Returns one match per keyword (first matching domain wins).
func Match(cert *ctlog.ParsedCertificate, keywords []model.Keyword) []MatchResult {
    var results []MatchResult

    for _, kw := range keywords {
        lower := strings.ToLower(kw.Value)

        // Check Common Name first
        if cert.CommonName != "" && strings.Contains(strings.ToLower(cert.CommonName), lower) {
            results = append(results, MatchResult{
                KeywordID:     kw.ID,
                MatchedDomain: cert.CommonName,
            })
            continue
        }

        // Check each SAN
        for _, san := range cert.SANs {
            if strings.Contains(strings.ToLower(san), lower) {
                results = append(results, MatchResult{
                    KeywordID:     kw.ID,
                    MatchedDomain: san,
                })
                break // One SAN match per keyword is sufficient
            }
        }
    }

    return results
}
```

---

### 7. Domain Models

```go
// internal/model/keyword.go
package model

import "time"

type Keyword struct {
    ID        int       `json:"id"`
    Value     string    `json:"value"`
    CreatedAt time.Time `json:"created_at"`
}
```

```go
// internal/model/certificate.go
package model

import "time"

type MatchedCertificate struct {
    ID            int       `json:"id"`
    SerialNumber  string    `json:"serial_number"`
    CommonName    string    `json:"common_name"`
    SANs          []string  `json:"sans"`
    Issuer        string    `json:"issuer"`
    NotBefore     time.Time `json:"not_before"`
    NotAfter      time.Time `json:"not_after"`
    KeywordID     int       `json:"keyword_id"`
    KeywordValue  string    `json:"keyword_value,omitempty"` // joined field
    MatchedDomain string    `json:"matched_domain"`
    CTLogIndex    int64     `json:"ct_log_index"`
    DiscoveredAt  time.Time `json:"discovered_at"`
}
```

```go
// internal/model/monitor.go
package model

import "time"

type MonitorState struct {
    LastProcessedIndex int64      `json:"last_processed_index"`
    LastTreeSize       int64      `json:"last_tree_size"`
    LastRunAt          *time.Time `json:"last_run_at"`
    TotalProcessed     int64      `json:"total_processed"`
    CertsInLastCycle   int        `json:"certs_in_last_cycle"`
    MatchesInLastCycle int        `json:"matches_in_last_cycle"`
    IsRunning          bool       `json:"is_running"`
    UpdatedAt          time.Time  `json:"updated_at"`
}
```

---

### 8. Repository Layer

All repositories accept `context.Context` for cancellation and use `pgxpool.Pool`.

**Example: Keyword Repository**

```go
// internal/repository/keyword.go
package repository

import (
    "context"
    "errors"

    "github.com/jackc/pgx/v5"
    "github.com/jackc/pgx/v5/pgxpool"

    "github.com/andres10976/SISAP-PoC/backend/internal/model"
)

type KeywordRepository struct {
    pool *pgxpool.Pool
}

func NewKeywordRepository(pool *pgxpool.Pool) *KeywordRepository {
    return &KeywordRepository{pool: pool}
}

func (r *KeywordRepository) List(ctx context.Context) ([]model.Keyword, error) {
    rows, err := r.pool.Query(ctx,
        `SELECT id, value, created_at FROM keywords ORDER BY created_at DESC`)
    if err != nil {
        return nil, err
    }
    defer rows.Close()

    var keywords []model.Keyword
    for rows.Next() {
        var kw model.Keyword
        if err := rows.Scan(&kw.ID, &kw.Value, &kw.CreatedAt); err != nil {
            return nil, err
        }
        keywords = append(keywords, kw)
    }
    return keywords, rows.Err()
}

func (r *KeywordRepository) Create(ctx context.Context, value string) (*model.Keyword, error) {
    var kw model.Keyword
    err := r.pool.QueryRow(ctx,
        `INSERT INTO keywords (value) VALUES ($1)
         RETURNING id, value, created_at`, value,
    ).Scan(&kw.ID, &kw.Value, &kw.CreatedAt)
    return &kw, err
}

func (r *KeywordRepository) Delete(ctx context.Context, id int) error {
    tag, err := r.pool.Exec(ctx, `DELETE FROM keywords WHERE id = $1`, id)
    if err != nil {
        return err
    }
    if tag.RowsAffected() == 0 {
        return ErrNotFound
    }
    return nil
}
```

**Certificate Repository** follows the same pattern, with additional:
- `ListPaginated(ctx, page, perPage, keywordID)` — returns certs with joined keyword value
- `ExportAll(ctx)` — returns all matched certs for CSV generation

**Monitor Repository:**
- `Get(ctx)` — reads the singleton row
- `Update(ctx, state)` — updates metrics and timestamps
- `SetRunning(ctx, running)` — toggles the is_running flag

---

### 9. HTTP Handlers

Handlers are thin — they validate input, call the service/repository layer,
and format the JSON response. They do NOT contain business logic.

**Example: Keyword handler**

```go
// internal/handler/keyword.go
package handler

import (
    "encoding/json"
    "net/http"
    "strconv"
    "strings"

    "github.com/go-chi/chi/v5"

    "github.com/andres10976/SISAP-PoC/backend/internal/repository"
)

type KeywordHandler struct {
    repo *repository.KeywordRepository
}

func NewKeywordHandler(repo *repository.KeywordRepository) *KeywordHandler {
    return &KeywordHandler{repo: repo}
}

// RegisterRoutes mounts keyword routes onto the given chi router.
func (h *KeywordHandler) RegisterRoutes(r chi.Router) {
    r.Get("/keywords", h.List)
    r.Post("/keywords", h.Create)
    r.Delete("/keywords/{id}", h.Delete)
}

func (h *KeywordHandler) List(w http.ResponseWriter, r *http.Request) {
    keywords, err := h.repo.List(r.Context())
    if err != nil {
        writeError(w, http.StatusInternalServerError, "failed to list keywords")
        return
    }
    // Return empty array instead of null when no keywords exist
    if keywords == nil {
        keywords = []model.Keyword{}
    }
    writeJSON(w, http.StatusOK, map[string]any{"keywords": keywords})
}

func (h *KeywordHandler) Create(w http.ResponseWriter, r *http.Request) {
    var req struct {
        Value string `json:"value"`
    }
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        writeError(w, http.StatusBadRequest, "invalid request body")
        return
    }

    value := strings.TrimSpace(req.Value)
    if value == "" {
        writeError(w, http.StatusBadRequest, "keyword value cannot be empty")
        return
    }

    kw, err := h.repo.Create(r.Context(), value)
    if err != nil {
        // Check for unique constraint violation
        if isDuplicateKeyError(err) {
            writeError(w, http.StatusConflict, "keyword already exists")
            return
        }
        writeError(w, http.StatusInternalServerError, "failed to create keyword")
        return
    }

    writeJSON(w, http.StatusCreated, kw)
}

func (h *KeywordHandler) Delete(w http.ResponseWriter, r *http.Request) {
    id, err := strconv.Atoi(chi.URLParam(r, "id"))
    if err != nil {
        writeError(w, http.StatusBadRequest, "invalid keyword id")
        return
    }

    if err := h.repo.Delete(r.Context(), id); err != nil {
        if errors.Is(err, repository.ErrNotFound) {
            writeError(w, http.StatusNotFound, "keyword not found")
            return
        }
        writeError(w, http.StatusInternalServerError, "failed to delete keyword")
        return
    }

    w.WriteHeader(http.StatusNoContent)
}
```

**Response helpers (`internal/handler/response.go`):**

```go
package handler

import (
    "encoding/json"
    "net/http"
)

func writeJSON(w http.ResponseWriter, status int, data any) {
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(status)
    json.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, status int, message string) {
    writeJSON(w, status, map[string]string{"error": message})
}
```

**CSV Export handler (`internal/handler/export.go`):**

```go
func (h *CertificateHandler) Export(w http.ResponseWriter, r *http.Request) {
    certs, err := h.repo.ExportAll(r.Context())
    if err != nil {
        writeError(w, http.StatusInternalServerError, "failed to export certificates")
        return
    }

    w.Header().Set("Content-Type", "text/csv")
    w.Header().Set("Content-Disposition", `attachment; filename="matched_certificates.csv"`)

    writer := csv.NewWriter(w)
    defer writer.Flush()

    // Header row
    writer.Write([]string{
        "id", "serial_number", "common_name", "sans", "issuer",
        "not_before", "not_after", "keyword", "matched_domain",
        "ct_log_index", "discovered_at",
    })

    for _, c := range certs {
        writer.Write([]string{
            strconv.Itoa(c.ID),
            c.SerialNumber,
            c.CommonName,
            strings.Join(c.SANs, ";"),
            c.Issuer,
            c.NotBefore.Format(time.RFC3339),
            c.NotAfter.Format(time.RFC3339),
            c.KeywordValue,
            c.MatchedDomain,
            strconv.FormatInt(c.CTLogIndex, 10),
            c.DiscoveredAt.Format(time.RFC3339),
        })
    }
}
```

---

### 10. Middleware

```go
// Middleware stack (applied in order via chi)
r := chi.NewRouter()

// 1. Request ID — generates or propagates X-Request-ID
r.Use(middleware.RequestID)

// 2. Structured logging — logs method, path, status, duration
r.Use(middleware.Logger)

// 3. Panic recovery — catches panics, returns 500, logs stack trace
r.Use(middleware.Recovery)

// 4. CORS — allows frontend origin
r.Use(middleware.CORS(cfg.CORSAllowOrigin))
```

**CORS middleware (`internal/middleware/cors.go`):**

```go
package middleware

import "net/http"

func CORS(allowOrigin string) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            w.Header().Set("Access-Control-Allow-Origin", allowOrigin)
            w.Header().Set("Access-Control-Allow-Methods", "GET, POST, DELETE, OPTIONS")
            w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

            if r.Method == http.MethodOptions {
                w.WriteHeader(http.StatusNoContent)
                return
            }

            next.ServeHTTP(w, r)
        })
    }
}
```

---

### 11. Entry Point (`cmd/server/main.go`)

```go
package main

import (
    "context"
    "fmt"
    "log/slog"
    "net/http"
    "os"
    "os/signal"
    "syscall"

    "github.com/go-chi/chi/v5"
    "github.com/jackc/pgx/v5/pgxpool"

    "github.com/andres10976/SISAP-PoC/backend/internal/config"
    "github.com/andres10976/SISAP-PoC/backend/internal/database"
    "github.com/andres10976/SISAP-PoC/backend/internal/handler"
    "github.com/andres10976/SISAP-PoC/backend/internal/middleware"
    "github.com/andres10976/SISAP-PoC/backend/internal/repository"
    "github.com/andres10976/SISAP-PoC/backend/internal/service/ctlog"
    "github.com/andres10976/SISAP-PoC/backend/internal/service/monitor"
)

func main() {
    // Structured logging
    slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))

    cfg := config.Load()

    // Database
    pool, err := database.Connect(cfg.DatabaseURL)
    if err != nil {
        slog.Error("database connection failed", "error", err)
        os.Exit(1)
    }
    defer pool.Close()

    if err := database.Migrate(pool); err != nil {
        slog.Error("migration failed", "error", err)
        os.Exit(1)
    }

    // Repositories
    keywordRepo := repository.NewKeywordRepository(pool)
    certRepo := repository.NewCertificateRepository(pool)
    monitorRepo := repository.NewMonitorRepository(pool)

    // Services
    ctClient := ctlog.NewClient(cfg.CTLogURL)
    mon := monitor.New(ctClient, keywordRepo, certRepo, monitorRepo, cfg.MonitorBatchSize, cfg.MonitorInterval)

    // Handlers
    kwHandler := handler.NewKeywordHandler(keywordRepo)
    certHandler := handler.NewCertificateHandler(certRepo)
    monHandler := handler.NewMonitorHandler(mon, monitorRepo)

    // Router
    r := chi.NewRouter()
    r.Use(middleware.RequestID)
    r.Use(middleware.Logger)
    r.Use(middleware.Recovery)
    r.Use(middleware.CORS(cfg.CORSAllowOrigin))

    r.Route("/api/v1", func(r chi.Router) {
        kwHandler.RegisterRoutes(r)
        certHandler.RegisterRoutes(r)
        monHandler.RegisterRoutes(r)
    })

    // Server with graceful shutdown
    srv := &http.Server{
        Addr:    fmt.Sprintf(":%s", cfg.ServerPort),
        Handler: r,
    }

    ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
    defer stop()

    go func() {
        slog.Info("server starting", "port", cfg.ServerPort)
        if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
            slog.Error("server error", "error", err)
            os.Exit(1)
        }
    }()

    <-ctx.Done()
    slog.Info("shutting down")

    // Stop the monitor if running
    mon.Stop(context.Background())

    // Give in-flight requests time to complete
    shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()
    srv.Shutdown(shutdownCtx)
}
```

---

### 12. Dependencies

```
go.mod:
    github.com/go-chi/chi/v5       — Lightweight, idiomatic HTTP router (stdlib-compatible)
    github.com/jackc/pgx/v5         — High-performance PostgreSQL driver with connection pooling
```

All other functionality uses the Go standard library:
- `crypto/x509` — Certificate parsing
- `encoding/base64` — Base64 decoding (handled automatically by `encoding/json`)
- `encoding/binary` — Binary parsing of MerkleTreeLeaf
- `encoding/csv` — CSV export generation
- `encoding/json` — JSON serialization/deserialization
- `log/slog` — Structured logging
- `net/http` — HTTP client (CT Log) and server
- `time`, `context`, `sync` — Concurrency primitives

---

### 13. Dockerfile

Multi-stage build for minimal final image:

```dockerfile
# Build stage
FROM golang:1.25-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /server ./cmd/server

# Run stage
FROM alpine:3.21
RUN apk add --no-cache ca-certificates
COPY --from=builder /server /server
EXPOSE 8080
ENTRYPOINT ["/server"]
```

- `ca-certificates` is required for HTTPS calls to the CT Log.
- `CGO_ENABLED=0` produces a static binary — no libc dependency.
- Final image is ~15 MB.

---

### 14. Error Handling Patterns

1. **Sentinel errors** for expected conditions:
   ```go
   var (
       ErrNotFound       = errors.New("not found")
       ErrAlreadyRunning = errors.New("monitor already running")
       ErrNotRunning     = errors.New("monitor not running")
   )
   ```

2. **Error wrapping** for context:
   ```go
   return nil, fmt.Errorf("fetch STH: %w", err)
   ```

3. **Handler error mapping**: Handlers check `errors.Is()` and map to appropriate HTTP status codes. Internal errors always return generic messages — never expose stack traces or SQL details to the client.

4. **Database constraint errors**: The pgx driver returns specific error codes for unique violations, which handlers map to `409 Conflict`.

---

### 15. Security Considerations

- **SQL Injection**: All queries use parameterized placeholders (`$1`, `$2`) via pgx. No string concatenation.
- **Input Validation**: Keyword values are trimmed and checked for emptiness before DB insertion.
- **CORS**: Restricted to the configured frontend origin. No wildcards in production.
- **Error Responses**: Generic messages only — no internal details, no stack traces.
- **Graceful Shutdown**: SIGINT/SIGTERM trigger ordered shutdown (stop monitor → drain HTTP → close DB).
- **Context Propagation**: All operations accept `context.Context` for timeout and cancellation.
- **No Secrets in Code**: Database URL and all sensitive config come from environment variables.
