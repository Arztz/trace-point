# AGENTS.md - trace-point

## Quick Start

```bash
# Backend (Go)
go run cmd/server/main.go

# Frontend (React/Vite)
cd ui && npm run dev
```

## Ports

- **Backend**: Configured in `configs/config.yaml` → `app.port` (default: **8081**)
- **Frontend**: 3000 (vite.config.ts) - proxies `/api` to backend
- **IMPORTANT**: Frontend proxy targets `http://localhost:8081`. Ensure backend port matches or update vite.config.ts proxy target.

## Project Structure

```
trace-point/
├── cmd/server/main.go           # Entry point, creates Chi router
├── internal/
│   ├── config/config.go         # Config loading via Viper
│   ├── storage/                 # SQLite with WAL mode, 7-day retention
│   ├── correlation/             # Spike detection engine
│   │   └── detector.go         # Moving average algorithm
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

# Run with race detection (for concurrency issues)
go run -race cmd/server/main.go

# Lint (if golangci-lint installed)
golangci-lint run
```

## Environment Variables (Optional)

### Backend
Configure in `configs/config.yaml`:
```yaml
app:
  port: 8081  # Change backend port here
```

### Frontend
Set environment variables before running:
```bash
# Custom frontend port (default: 3000)
FRONTEND_PORT=3000 npm run dev

# Custom backend URL (default: http://localhost:8081)
BACKEND_URL=http://localhost:8081 npm run dev

# Both together
FRONTEND_PORT=3000 BACKEND_URL=http://localhost:8081 npm run dev
```

Or create `ui/.env.local`:
```
FRONTEND_PORT=3000
BACKEND_URL=http://localhost:8081
```

## API Endpoints

| Endpoint | Description |
|----------|-------------|
| GET /health | Health check |
| GET /api/v1/spikes | List spike events (supports namespace, pod, sort, limit) |
| GET /api/v1/spikes/analyze | Historical spike analysis (start, end, window, namespace, replicaset, threshold, limit) |
| GET /api/v1/timeline | Timeline data (CPU/RAM) with pod filtering |
| GET /api/v1/export | Export JSON (spikes or refactoring) |
| GET /api/v1/config | Application config |
| GET /api/v1/gravity-scores | Resource gravity scores |

## Key Behavior

- **Spike Detection**: Polls Prometheus every 30s, calculates 30-min moving average, triggers if current > baseline + 50%
- **Reconciliation**: 8-minute buffer after spike before alerting (allows traces to accumulate)
- **Cooldown**: 15 minutes after alert before same pod can alert again
- **Data Retention**: Spike events auto-purge after 7 days

## Calculation Formulas

### Spike Detection
```
Spike Detected = Current Usage > Moving Average + (Threshold %)

Example:
- Moving Average: 50%
- Threshold: 50%
- Trigger: > 75% (50% + 50%)
```

### Resource Gravity Score
```
Resource Gravity Score = Resource Peak × (1 / Request Frequency)

Where:
- Resource Peak = max(CPU peak, RAM peak) over 7 days
- Request Frequency = requests per day
- High Score = High resource usage + Low call frequency

Score Range:
- 0-3: Low impact (monitor)
- 3-6: Medium impact (consider optimization)
- 6-10+: High impact (priority refactoring)
```

### Route Selection (Culprit Detection)
When multiple routes active during spike:
- Primary: Highest CPU consumer
- Secondary: Highest RAM consumer

### Baseline Comparison
```
vs Baseline % = ((Current - Baseline) / Baseline) × 100

Tooltip indicators:
- Green (▼): Below baseline
- Yellow (±): Near baseline (within 20%)
- Red (▲): Above baseline (spike)
```

### Spike Analysis (Historical)
```
For each point Pi in time range:
  - Window = points from (Pi - window_size) to (Pi-1)
  - Moving Average = sum(previous_points) / count(previous_points)
  - Deviation % = (Pi - MovingAverage) / MovingAverage × 100
  - If Deviation > Threshold → SPIKE!

Window Sliding:
- v1.0.2 (real-time): Only checks LATEST point against last 30min window
- v1.0.3 (historical): For EACH point in range, calculates moving average from previous points
```

## Known Issues / Notes

- Frontend proxy configured to `http://localhost:8081` - must match config.yaml port
- No CI/CD workflows yet
- Tests exist, run with `go test ./...`
- Pre-existing test failures (unrelated to recent changes):
  - TestDefaultDetectorConfig (config value mismatch)
  - TestProfilerClientResolveServiceName (test bug)

## Version History

### v1.0.3 (2026-04-13)
- New: Spike Explorer tab for historical spike analysis
- New endpoint: GET /api/v1/spikes/analyze
- Supports: start/end time (RFC3339 or relative: 24h, 7d), window size (5m/15m/30m/1h)
- Algorithm: For each point in range, sliding window detects ALL spikes
- Frontend: Date range picker, window selector, results table, summary cards

### v1.0.2 (2026-04-13)
- Sort dropdown on SpikeList (Time/CPU/RAM/Pod, asc/desc)
- TimelineChart tooltip shows baseline comparison
- Performance fixes (useMemo, useCallback, sort.Slice)
- Backend reliability (race condition fix, graceful shutdown)
- Export error feedback UI
- Console.log debug statements removed

### v1.0.1 (2026-04-12)
- Timeline API supports pod_name filter
- Pod Selector dropdown (multi-select, search)
- PodLegend component (click-to-highlight)
- Multi-line chart (solid=CPU, dashed=RAM per pod)
- 20-color palette for pod differentiation

## References

- `README.md` - Complete usage guide with formulas and examples
- `docs/requirement-application.md` - Requirements specification
- `docs/superpowers/implementation-plan.md` - Code quality fixes
- `docs/developer-team/DEVELOPMENT_PLAN.md` - Implementation history