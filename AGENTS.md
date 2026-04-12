# AGENTS.md - trace-point

## Quick Start

```bash
# Backend (Go)
go run cmd/server/main.go

# Frontend (React/Vite)
cd ui && npm run dev
```

## Ports

- **Backend**: Configured in `configs/config.yaml` → `app.port` (default: 8080)
- **Frontend**: 3000 (vite.config.ts) - proxies `/api` to backend
- **IMPORTANT**: Frontend proxy targets `http://localhost:8080`. Ensure backend port matches or update vite.config.ts proxy target.

## Project Structure

```
trace-point/
├── cmd/server/main.go           # Entry point, creates Chi router
├── internal/
│   ├── config/config.go         # Config loading via Viper
│   ├── storage/                 # SQLite with WAL mode, 7-day retention
│   ├── correlation/             # Spike detection engine
│   │   └── detector.go         # Moving average algorithm (30-min window, 50% threshold)
│   └── integration/             # Prometheus, Signoz, Profiler, Discord clients
├── configs/config.yaml          # All configuration
└── ui/                          # React 18 + Vite + Tailwind + Recharts
    └── src/services/api.ts      # Calls /api/* endpoints

```

## Commands

```bash
# Run Go tests
go test ./...

# Run a specific test
go test -v ./internal/correlation/

# Build frontend
cd ui && npm run build

# Lint (if golangci-lint installed)
golangci-lint run
```

## API Endpoints

| Endpoint | Description |
|----------|-------------|
| GET /health | Health check |
| GET /api/v1/spikes | List spike events |
| GET /api/v1/timeline | Timeline data (CPU/RAM) |
| GET /api/v1/export | Export JSON |
| GET /api/v1/config | Application config |
| GET /api/v1/gravity-scores | Resource gravity scores |

## Key Behavior

- **Spike Detection**: Polls Prometheus every 30s, calculates 30-min moving average, triggers if current > baseline + 50%
- **Reconciliation**: 8-minute buffer after spike before alerting (allows traces to accumulate)
- **Cooldown**: 15 minutes after alert before same pod can alert again
- **Data Retention**: Spike events auto-purge after 7 days (set in NewDatabase)

## Known Issues / Notes

- Frontend vite proxy hardcoded to `http://localhost:8080` - mismatch with config.yaml (8081) will cause 502 errors
- No CI/CD workflows yet (docs only)
- Tests exist but no test runner configured in Makefile

## References

- `docs/developer-team/SESSION_HANDOVER.md` - Detailed project state and pending tasks (T-070 to T-076)
- `docs/requirement-application.md` - Requirements specification