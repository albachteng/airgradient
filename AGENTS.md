# airgradient — agent guide

## Stack
- Go 1.22+, `modernc.org/sqlite` (no CGO), `github.com/joho/godotenv`
- Three binaries: `cmd/poller` (long-running daemon), `cmd/summary` (manual), `cmd/placement` (CLI)
- Config via `.env` (see `.env.example`); required vars: `AIRGRADIENT_API_KEY`, `AIRGRADIENT_LOCATION_ID`

## Packages
| Package | Responsibility |
|---|---|
| `internal/client` | HTTP client for AirGradient API |
| `internal/db` | SQLite schema, migrations, queries |
| `internal/logger` | stdout + daily rotating file |
| `internal/poller` | Ticker loop: fetch → save → log |
| `internal/summary` | Generate + render HTML summary |
| `internal/emailer` | Gmail SMTP send |

## Testing
- `go test ./...` — all tests run in-process; no live network or external credentials required
- `go test -race ./...` — must pass
- Fixtures use `t.TempDir()` for SQLite files

## Rules
- Keep packages focused; don't reach across package boundaries without a small interface
- `internal/db` is the only package that touches SQLite directly
- Do not store credentials in code; all secrets come from env
- Do not modify the SQLite schema without adding a migration in `db.Open`
- The health endpoint is at `/health` on `HEALTH_PORT` (default 8080); keep it a simple 200 OK

## Issue tracker
Issues live in GitHub Issues. See `docs/agents/issue-tracker.md`.
