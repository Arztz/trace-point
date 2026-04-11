# Session Handover - Resource-to-Code Correlation Engine

**Date:** 2026-04-12  
**Status:** Implementation Complete, Testing Pending

---

## Project Overview

This is a **Resource-to-Code Correlation Engine** - a Golang-based tool that correlates Kubernetes resource spikes with code-level root causes.

### What It Does

1. **Polls Prometheus** for CPU/RAM metrics every 30 seconds
2. **Detects spikes** using moving average algorithm (30-min window, 50% threshold)
3. **Correlates with Signoz** traces to identify the route/endpoint
4. **Enriches with Gcloud Profiler** to find the culprit function
5. **Sends Discord alerts** with RCA information
6. **Provides a React dashboard** for visualization

### Tech Stack

| Component | Technology |
|-----------|------------|
| Backend | Go 1.21+, Chi router |
| Database | SQLite with WAL mode |
| Frontend | React 18, Vite, Tailwind CSS |
| Charts | Recharts |
| State | TanStack Query |

---

## Current Status

### ✅ Completed (T-001 to T-069)

All 69 implementation tasks are complete:

**Backend:**
- Project structure, configuration, database layer
- Prometheus client, spike detection engine
- Signoz integration, Gcloud Profiler integration
- Discord webhook alerting
- JSON export endpoints

**Frontend:**
- React setup with Vite
- Timeline charts (CPU, RAM)
- Spike list with drill-down
- Filter bar, time range selector
- Resource gravity scores table

### ⏳ Pending (T-070 to T-076)

```
T-070: Unit tests for correlation engine
T-071: Integration tests for Prometheus
T-072: Integration tests for Signoz
T-073: Integration tests for Profiler
T-074: Dashboard E2E tests
T-075: Load testing (100 pods, 1000 spikes/day)
T-076: Bug fixes and improvements
```

---

## How to Run

### Backend

```bash
cd /home/arztz/Projects/opencode/trace-point
go run cmd/server/main.go
```

- Runs on port 8080
- Health check: http://localhost:8080/health

### Frontend

```bash
cd /home/arztz/Projects/opencode/trace-point/ui
npm run dev
```

- Runs on port 3000
- Dashboard: http://localhost:3000

---

## API Endpoints

| Endpoint | Description |
|----------|-------------|
| GET /health | Health check |
| GET /api/v1/spikes | List spike events |
| GET /api/v1/timeline | Timeline data |
| GET /api/v1/export | Export JSON |
| GET /api/v1/config | Configuration |
| GET /api/v1/gravity-scores | Resource gravity scores |

---

## Configuration

Edit `configs/config.yaml`:

```yaml
prometheus:
  url: "http://prometheus:9090"

signoz:
  url: "http://signoz:4318"

gcloud:
  project_id: "your-gcp-project-id"
  profiler_enabled: true

detection:
  threshold_percent: 50
  polling_interval_seconds: 30
  moving_average_window_minutes: 30
  reconciliation_buffer_minutes: 8
  cooldown_minutes: 15

discord:
  webhook_url: "${DISCORD_WEBHOOK_URL}"

database:
  path: "./data/trace-point.db"
```

---

## Key Files

```
trace-point/
├── cmd/server/main.go           # Entry point
├── internal/
│   ├── config/config.go       # Configuration
│   ├── storage/database.go   # SQLite
│   ├── correlation/engine.go # Spike detection
│   └── integration/          # Prometheus, Signoz, Profiler, Discord
├── configs/config.yaml        # Config
├── ui/                        # React frontend
│   ├── src/pages/Dashboard.tsx
│   └── src/services/api.ts
└── docs/
    ├── requirement-application.md
    └── developer-team/DEVELOPMENT_PLAN.md
```

---

## Next Steps (Priority Order)

1. **Verify the application runs** - Start backend and frontend
2. **Write unit tests** - Test correlation engine logic
3. **Integration tests** - Test external service connections
4. **E2E tests** - Test dashboard functionality
5. **Load testing** - Verify performance
6. **Bug fixes** - Address any issues

---

## Important Details

### Spike Detection Algorithm

1. Poll Prometheus every 30 seconds
2. Calculate moving average over 30-minute window
3. If current > baseline + 50%, mark as spike
4. Wait 8-minute reconciliation buffer
5. Query Signoz for traces around spike time
6. Query Profiler for function data
7. Send Discord alert with RCA

### Data Retention

- Spike events stored for 7 days
- Auto-purge runs on startup

### Reconciliation

- 8-minute buffer before alerting to allow trace accumulation
- 15-minute cooldown after alert to prevent spam

---

## Getting Help

- Requirements: `/home/arztz/Projects/opencode/trace-point/docs/requirement-application.md`
- Development plan: `/home/arztz/Projects/opencode/trace-point/docs/developer-team/DEVELOPMENT_PLAN.md`
- Quick start: Run backend + frontend as shown above

---

**Document Created:** 2026-04-12  
**For:** New development session to understand current state