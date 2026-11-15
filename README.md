# airgradient

Polls an [AirGradient](https://www.airgradient.com/) monitor and a sample of Chicago public monitors on a configurable interval, stores readings in a local SQLite database, and sends a periodic HTML email summary.

## Prerequisites

- Go 1.25+
- A Gmail account with an [App Password](https://myaccount.google.com/apppasswords) (2FA must be enabled)
- An AirGradient API token — dashboard → General Settings → Connectivity

## Setup

```bash
cp .env.example .env
# edit .env with your values
```

Key `.env` variables:

| Variable | Description | Default |
|---|---|---|
| `AIRGRADIENT_API_KEY` | AirGradient API token | required |
| `AIRGRADIENT_LOCATION_ID` | Your monitor's location ID (from dashboard) | required |
| `POLL_INTERVAL_SECONDS` | How often to fetch readings | `3600` |
| `DB_PATH` | SQLite database file path | `./airgradient.db` |
| `LOG_DIR` | Directory for daily rotating log files | `./logs` |
| `CHICAGO_SAMPLE_SIZE` | Number of public Chicago monitors to sample per cycle | `10` |
| `SMTP_USERNAME` | Gmail address used to send | required for summary |
| `SMTP_PASSWORD` | Gmail App Password | required for summary |
| `SUMMARY_TO` | Recipient address(es), comma-separated | required for summary |
| `SUMMARY_PERIOD` | `week` or `month` | `week` |

## Build

```bash
go build -o bin/poller    ./cmd/poller
go build -o bin/summary   ./cmd/summary
go build -o bin/placement ./cmd/placement
```

## Test

```bash
go test ./...
```

## Commands

### `bin/poller`

Starts the polling loop. Fetches your monitor and a random sample of Chicago public monitors on each tick, saves to SQLite, and logs JSON to stdout and `logs/YYYY-MM-DD.log`.

```bash
./bin/poller
```

Runs once immediately on startup, then on the configured interval. Send `SIGINT` (Ctrl-C) or `SIGTERM` to stop cleanly.

---

### `bin/placement`

Tags where your monitor is currently located so readings can be compared room-by-room in the summary email.

```bash
# set current placement (closes any previous one)
./bin/placement set "bedroom"
./bin/placement set "kitchen" "near the window"

# show the active placement
./bin/placement current

# show full placement history
./bin/placement list
```

Move the monitor → run `placement set` before the next poll cycle so the new readings are correctly tagged.

---

### `bin/summary`

Generates and sends the HTML email summary for the configured period. Safe to run manually at any time.

```bash
./bin/summary

# override period for this run only
SUMMARY_PERIOD=month ./bin/summary
```

The email contains two tables:
- **Your Monitor** — avg PM2.5, CO2, temp, humidity broken down by placement
- **Chicago Comparison** — the same metrics per sampled public monitor

## Cron schedule

The crontab (installed at setup) runs:

```
@reboot   — poller starts on boot         → logs/poller.log
0 8 * * 0 — summary email every Sunday 8am → logs/summary.log
```

To edit: `crontab -e`

> **WSL2 note:** cron doesn't start automatically. Run `sudo service cron start` after booting, or add it to your WSL startup.

To rebuild after code changes and have cron pick up the new binary:

```bash
go build -o bin/poller ./cmd/poller && go build -o bin/summary ./cmd/summary
```

## Project layout

```
cmd/
  poller/     — entry point for the polling loop
  summary/    — entry point for the email summary
  placement/  — CLI for tagging monitor location
internal/
  client/     — AirGradient HTTP client, Chicago bbox filter
  db/         — SQLite schema, readings, placements, stats queries
  emailer/    — Gmail SMTP sender
  logger/     — stdout + daily rotating file logger
  poller/     — polling orchestrator (fetch → save → log)
  summary/    — data aggregation and HTML rendering
```

## API reference

- Public API docs: https://api.airgradient.com/public/docs/api/v1/
- AirGradient documentation: https://www.airgradient.com/documentation

