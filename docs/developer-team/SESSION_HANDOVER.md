# Session Handover - Resource-to-Code Correlation Engine

**Date:** 2026-04-12  
**Status:** v1.0.2 Released - Pod-Level Breakdown Complete

**Latest Release:** v1.0.2 - Pod-Level Resource Breakdown (2026-04-12)
- Timeline now shows per-pod metrics (not aggregated)
- Pod selector dropdown for filtering
- Pod legend with click-to-highlight
- Multi-line chart (solid=CPU, dashed=RAM per pod)
- QA tested and bugs fixed (availablePods field name, filter logic)

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

### ✅ v1.0.2 - Pod-Level Resource Breakdown (Released 2026-04-12)

- Timeline API supports `pod_name` filter parameter
- Timeline response includes `availablePods` for frontend dropdown
- Pod selector dropdown (multi-select with search)
- Pod legend component with click-to-highlight
- Multi-line chart with per-pod data (solid=CPU, dashed=RAM)
- 20-color palette for pod differentiation
- Build passes: backend and frontend

### ⏳ Pending - User Feedback (2026-04-12)

```
T-077: Group metrics by Replicaset (not individual pod)
       - Currently shows per-pod: "my-service-abc-1234"
       - Should group by: "my-service-abc" (remove trailing "-xxxx")
       
T-078: Dashboard Performance Issue
       - Website is "very slow"
       - Need to investigate and optimize
```

---

## How to Run

### Backend

```bash
cd /home/rut/Project/trace-point
go run cmd/server/main.go
```

- Runs on port 8081 (configured in configs/config.yaml)
- Health check: http://localhost:8081/health
- Prometheus: http://prod.prometheus.fundii-prod.internal

### Frontend

```bash
cd /home/rut/Project/trace-point/ui
npm run dev
```

- Runs on port 3000 (proxies /api to backend:8081)
- Dashboard: http://localhost:3000

---

## API Endpoints

| Endpoint | Description |
|----------|-------------|
| GET /health | Health check |
| GET /api/v1/spikes | List spike events |
| GET /api/v1/timeline | Timeline data (supports pod_name filter) |
| GET /api/v1/export | Export JSON |
| GET /api/v1/config | Configuration |
| GET /api/v1/gravity-scores | Resource gravity scores |

### Timeline API Usage

```bash
# Get all pods + availablePods for dropdown
GET /api/v1/timeline

# Filter to specific pod(s)
GET /api/v1/timeline?pod_name=pod-a,pod-b

# Filter by namespace
GET /api/v1/timeline?namespace=production

# Combined filters
GET /api/v1/timeline?namespace=production&pod_name=pod-a&time_range=6h
```

---

## Configuration

Edit `configs/config.yaml`:

```yaml
app:
  host: "0.0.0.0"
  port: 8081

prometheus:
  url: "http://prod.prometheus.fundii-prod.internal"

signoz:
  url: "http://signoz:4318"

gcloud:
  project_id: "fundii-prod"
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
│   ├── config/config.go        # Configuration
│   ├── storage/
│   │   ├── database.go         # SQLite with WAL mode, metrics_cache
│   │   ├── repository.go      # CRUD for spikes + timeline metrics
│   │   └── models.go          # Data models (SpikeEvent, TimelineMetric)
│   ├── correlation/engine.go  # Spike detection (moving average)
│   └── integration/
│       ├── prometheus/client.go  # Prometheus queries (QueryTimelineMetrics, GetAvailablePods)
│       ├── signoz/client.go     # Trace correlation
│       ├── profiler/client.go   # Gcloud Profiler
│       └── discord/client.go   # Webhook alerts
├── configs/config.yaml         # App config (port 8081)
├── ui/                        # React frontend
│   ├── src/
│   │   ├── pages/Dashboard.tsx    # Main dashboard (selectedPods, highlightedPod state)
│   │   ├── components/
│   │   │   ├── TimelineChart.tsx  # Multi-line chart (per-pod)
│   │   │   ├── PodSelector.tsx    # Multi-select dropdown (NEW v1.0.2)
│   │   │   ├── PodLegend.tsx      # Click-to-highlight (NEW v1.0.2)
│   │   │   ├── FilterBar.tsx      # Updated with pod selector
│   │   │   └── ...
│   │   ├── services/api.ts       # API calls
│   │   └── types/timeline.ts      # Timeline types (PodInfo, POD_COLORS)
│   └── vite.config.ts            # Proxy to localhost:8081
└── docs/
    ├── requirement-application.md
    ├── planning/                              # Implementation plans
    └── test-reports/                          # QA reports
```

---

## Next Steps (Priority Order)

1. **T-077: Replicaset Grouping** - Group metrics by replicaset name (remove "-xxxx" suffix)
2. **T-078: Performance Optimization** - Investigate and fix "very slow" dashboard
3. **Unit tests** - Test correlation engine logic (T-070)
4. **Integration tests** - Test external service connections (T-071 to T-073)
5. **E2E tests** - Browser testing (T-074)
6. **Load testing** - Performance under load (T-075)

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

- Requirements: `/home/rut/Project/trace-point/docs/requirement-application.md`
- Quick start: Run backend + frontend as shown above
- API docs: See Timeline API Usage section above

---

## v1.0.2 Changes (Pod-Level Resource Breakdown)

### Problem
Dashboard showed aggregated metrics across ALL pods. Users cannot see which pod is using how much CPU/RAM.

### Solution
- Pod selector dropdown for filtering
- Multi-line chart showing each pod separately
- Pod legend with click-to-highlight

### Backend Changes
- `internal/integration/prometheus/client.go`
  - Added `podNameFilter` parameter to `QueryTimelineMetrics()`
  - Added `GetAvailablePods()` method
  - Pod filter correctly applied in processing (lines 589-593)

- `internal/server/handlers/handlers.go`
  - Added `pod_name` query parameter parsing
  - Changed JSON field: `available_pods` → `availablePods` (line 250)

### Frontend Changes
- `ui/src/types/timeline.ts` - Added `PodInfo` type, `POD_COLORS`, `getPodColor()`
- `ui/src/components/PodSelector.tsx` - NEW multi-select dropdown
- `ui/src/components/PodLegend.tsx` - NEW click-to-highlight component
- `ui/src/components/TimelineChart.tsx` - Multi-line per pod support
- `ui/src/components/FilterBar.tsx` - Integrated PodSelector
- `ui/src/pages/Dashboard.tsx` - State: selectedPods, highlightedPod

### QA Bug Fixes
- Fixed field name: `available_pods` → `availablePods`
- Verified pod filter is correctly applied in backend

---

## v1.0.3 Pending (Next Session)

### T-077: Replicaset Grouping

**Requirement from User:** Filter and data should be by Replicaset (cut last part "-xxxx")

**Current behavior:**
- Shows individual pods: "my-service-abc-1234"

**Desired behavior:**
- Group by replicaset: "my-service-abc"
- Remove trailing "-xxxx" suffix from pod name

**Implementation approach:**
1. Extract replicaset name by removing trailing "-xxxx" from pod name
2. Update AvailablePods to return replicaset names
3. Aggregate metrics by replicaset (average across all pods)
4. Update filter logic to match replicaset names

### T-078: Performance Issue

**User feedback:** "Website is very slow"

**Need to investigate:**
1. Large data payloads causing slow render
2. Too many re-renders in React
3. Inefficient chart rendering (many lines)
4. API response size
5. Bundle size optimization

---

**Document Created:** 2026-04-12  
**Last Updated:** 2026-04-12 (v1.0.2 release + pending user feedback)
**For:** New development session to understand current state