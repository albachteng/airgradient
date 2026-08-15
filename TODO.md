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
- [ ] Write failing tests
- [ ] Implement `Client` struct with base URL + token
- [ ] `Ping() error`
- [ ] `GetOwnMeasure(locationID string) (*Measure, error)`
- [ ] `GetWorldMeasures() ([]Measure, error)`
- [ ] Chicago bbox filter helper

### `internal/db`
- [ ] Write failing tests
- [ ] `Open(path string) (*DB, error)` + schema migration
- [ ] `SaveReading(r Reading) error`
- [ ] `CurrentPlacement() (*Placement, error)`
- [ ] `SetPlacement(label, description string) error`
- [ ] `ListPlacements() ([]Placement, error)`
- [ ] `ReadingsSince(t time.Time) ([]Reading, error)`

### `internal/logger`
- [ ] Write failing tests
- [ ] `New(logDir string) (*Logger, error)`
- [ ] Write to stdout + daily file simultaneously
- [ ] `Log(msg string, fields map[string]any)`

### `internal/poller`
- [ ] Write failing tests
- [ ] `Run(ctx context.Context, interval time.Duration)` — ticker loop
- [ ] Fetch own + chicago → save → log each cycle

### `cmd/poller`
- [ ] Load `.env`, wire up client/db/logger/poller, start

### `cmd/placement`
- [ ] `set <label> [description]`
- [ ] `list`
- [ ] `current`

### Config & Infra
- [ ] `.env.example`
- [ ] `.gitignore`
