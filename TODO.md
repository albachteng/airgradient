# AirGradient Project Plan

## Goal
Poll our AirGradient monitor + a sample of Chicago public monitors on a configurable interval,
persist readings to a local SQLite database, and log to stdout + daily rotating file.
Track monitor placement (room/location in house) so we can compare readings across rooms
and against Chicago public air quality data.

## Design

### Packages

| Package | Responsibility |
|---|---|
| `internal/client` | Typed HTTP client for AirGradient public API |
| `internal/db` | SQLite schema, migrations, read/write queries |
| `internal/logger` | Structured logging to stdout + daily rotating file |
| `internal/poller` | Orchestrates periodic fetch → db → log cycle |
| `cmd/poller` | Entry point: loads config, starts poller |
| `cmd/placement` | CLI: `set`, `list`, `current` subcommands |

### API Endpoints Used
- `GET /public/api/v1/ping` — health check (no auth)
- `GET /public/api/v1/locations/{id}/measures/current` — own monitor (auth)
- `GET /public/api/v1/world/locations/measures/current` — all public monitors (no auth, filtered to Chicago bbox)

### Config (via `.env`)
- `AIRGRADIENT_API_KEY`
- `AIRGRADIENT_LOCATION_ID` — your monitor's location ID
- `POLL_INTERVAL_SECONDS` (default: 3600)
- `DB_PATH` (default: `./airgradient.db`)
- `LOG_DIR` (default: `./logs`)
- `CHICAGO_SAMPLE_SIZE` (default: 10)

### Chicago Bounding Box
- Lat: 41.644 – 42.023
- Lon: -87.940 – -87.524

### Database Schema
```sql
CREATE TABLE placements (
  id          INTEGER PRIMARY KEY AUTOINCREMENT,
  label       TEXT NOT NULL,
  description TEXT,
  started_at  TEXT NOT NULL DEFAULT (datetime('now')),
  ended_at    TEXT
);

CREATE TABLE readings (
  id            INTEGER PRIMARY KEY AUTOINCREMENT,
  location_id   INTEGER NOT NULL,
  location_name TEXT,
  placement_id  INTEGER REFERENCES placements(id),  -- NULL for chicago readings
  source        TEXT NOT NULL CHECK(source IN ('own','chicago')),
  timestamp     TEXT NOT NULL,
  pm01          REAL,
  pm02          REAL,
  pm10          REAL,
  pm003_count   INTEGER,
  atmp          REAL,
  rhum          INTEGER,
  rco2          INTEGER,
  tvoc          REAL,
  tvoc_index    INTEGER,
  nox_index     INTEGER,
  heat_index    INTEGER,
  wifi          INTEGER
);
```

---

## Tasks

### Setup
- [x] `go mod init`
- [x] Add dependencies (`modernc.org/sqlite`, `github.com/joho/godotenv`)
- [x] Create directory structure
- [x] Write TODO.md

### `internal/client`
- [x] Write failing tests
- [x] Implement `Client` struct with base URL + token
- [x] `Ping() error`
- [x] `GetOwnMeasure(locationID string) (*Measure, error)`
- [x] `GetWorldMeasures() ([]Measure, error)`
- [x] Chicago bbox filter helper

### `internal/db`
- [x] Write failing tests
- [x] `Open(path string) (*DB, error)` + schema migration
- [x] `SaveReading(r Reading) error`
- [x] `CurrentPlacement() (*Placement, error)`
- [x] `SetPlacement(label, description string) error`
- [x] `ListPlacements() ([]Placement, error)`
- [x] `ReadingsSince(t time.Time) ([]Reading, error)`

### `internal/logger`
- [x] Write failing tests
- [x] `New(logDir string) (*Logger, error)`
- [x] Write to stdout + daily file simultaneously
- [x] `Log(msg string, fields map[string]any)`

### `internal/poller`
- [x] Write failing tests
- [x] `Run(ctx context.Context, interval time.Duration)` — ticker loop
- [x] Fetch own + chicago → save → log each cycle

### `cmd/poller`
- [x] Load `.env`, wire up client/db/logger/poller, start

### `cmd/placement`
- [x] `set <label> [description]`
- [x] `list`
- [x] `current`

### `internal/db` — stats queries
- [x] Write failing tests
- [x] `OwnStatsByPlacement(from, to time.Time) ([]PlacementStats, error)`
- [x] `ChicagoStatsByLocation(from, to time.Time) ([]LocationStats, error)`

### `internal/summary`
- [x] Write failing tests
- [x] `Generate(store Store, from, to time.Time) (Summary, error)`
- [x] `RenderHTML(s Summary) (string, error)` — HTML email with placement + Chicago tables

### `internal/emailer`
- [x] Write failing tests
- [x] `New(cfg Config) *Emailer` — Gmail SMTP via `net/smtp`
- [x] `NewWithSender` — injectable transport for testing
- [x] `Send(subject, htmlBody string) error`

### `cmd/summary`
- [x] Load `.env`, generate summary, render HTML, send via SMTP
- [x] `SUMMARY_PERIOD=week|month`

### Config & Infra
- [x] `.env.example`
- [x] `.gitignore`
