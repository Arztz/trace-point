# Trace-Point: Resource-to-Code Correlation Engine

<p align="center">
  <img src="https://img.shields.io/badge/Go-1.25+-00ADD8?style=flat&logo=go" alt="Go Version">
  <img src="https://img.shields.io/badge/React-18+-61DAFB?style=flat&logo=react" alt="React Version">
  <img src="https://img.shields.io/badge/License-MIT-green.svg" alt="License">
  <img src="https://img.shields.io/badge/Version-1.0.2-success.svg" alt="Version">
</p>

Trace-Point is an automated tool designed to correlate Kubernetes resource spikes with code-level root causes. It integrates data from Prometheus (metrics), Signoz (traces), and Google Cloud Profiler (samples) to provide complete root cause analysis through a web dashboard and Discord alerts.

---

## Table of Contents

1. [Quick Start](#quick-start)
2. [How to Use the Dashboard](#how-to-use-the-dashboard)
3. [How to Read Results](#how-to-read-results)
4. [Calculation Formulas](#calculation-formulas)
5. [API Reference](#api-reference)
6. [Configuration](#configuration)

---

## Quick Start

### 1. Run the Application

**Backend (Terminal 1):**
```bash
go run cmd/server/main.go
```

**Frontend (Terminal 2):**
```bash
cd ui && npm run dev
```

### 2. Access the Dashboard

| Service | URL | Description |
|---------|-----|-------------|
| Frontend | http://localhost:3000 | Dashboard UI |
| Backend API | http://localhost:8081 | REST API |
| Health | http://localhost:8081/health | Health check |

---

## How to Use the Dashboard

### Dashboard Overview

The dashboard displays a unified timeline of CPU and RAM utilization for your Kubernetes pods:

```
┌─────────────────────────────────────────────────────────────────────────────┐
│  Trace-Point Dashboard                                    [Refresh] [Export]│
├─────────────────────────────────────────────────────────────────────────────┤
│  Time Range: [1h] [6h] [24h] [7d]    Namespace: [All ▼]    Pod: [Search...] │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                             │
│  █                                                                            │
│  █  CPU ════════════════════════════════════════════════════════════════    │
│  █     ══════════════════════════════════════════════════════════════      │
│  █                                                                          │
│  █  ███                                                                       │
│  █  ███ RAM ████████████████████████████████████████████████████████████    │
│  █  ███    ████████████████████████████████████████████████████████████    │
│  █                                                                          │
│  █  ● ● ● ● ● ● ● ● ● ● ● ● ● ● ● ● ● ● ● ● ● ● ● ● ● ● ● ● ●             │
│     ───────────────────────────────────────────────────────────────→ Time   │
│                                                                             │
├─────────────────────────────────────────────────────────────────────────────┤
│  Spike Events                          Sort: [Time ▼]  [CPU ▼] [Pod ▼]     │
│  ┌──────────────────────────────────────────────────────────────────────┐  │
│  │ 🔴 pod-name-abc   production   CPU: 85%   10 min ago   [View Details] │  │
│  │ 🔴 pod-name-xyz   staging      RAM: 78%   25 min ago   [View Details] │  │
│  └──────────────────────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────────────────────┘
```

### Using Filters

1. **Time Range Selection**: Choose between 1 hour, 6 hours, 24 hours, or 7 days
2. **Namespace Filter**: Select specific namespaces to monitor (e.g., production, staging)
3. **Pod Filter**: Search for specific pods by name
4. **Replicaset View**: Click on legend items to highlight specific services

### Spike List

- Shows all detected resource spikes
- Sort by time, CPU usage, RAM usage, or pod name
- Click "View Details" to see:
  - Route that was active during the spike
  - Trace ID for correlation
  - Culprit function from profiler
  - Resource impact (before/after)

### Export Data

Click the **Export** button to download:
- Spike history as JSON (last 7 days)
- Refactoring recommendations with Resource Gravity Scores

---

## How to Read Results

### Timeline Chart

The timeline chart shows CPU (line) and RAM (area) utilization over time:

| Element | Meaning |
|---------|---------|
| **Solid Line** | CPU utilization % |
| **Dashed Line** | RAM utilization % |
| **Colored Areas** | Different pods/replicasets |
| **Red Markers** | Detected spikes |
| **Tooltip** | Shows current value, baseline, and % vs baseline |

### Understanding the Tooltip

When you hover over a data point, the tooltip shows:

```
timestamp: 2026-04-13 14:30:00
cpu: 75% (vs Baseline 50%) ▲
ram: 60% (vs Baseline 40%) ▲
```

- **Green (▼)** = Below baseline
- **Yellow (±)** = Near baseline (within 20%)
- **Red (▲)** = Above baseline (spike detected)

### Spike Events

Each spike event shows:

| Field | Description |
|-------|-------------|
| **Pod Name** | The Kubernetes pod that spiked |
| **Namespace** | Which namespace it's in |
| **CPU/RAM %** | Current utilization |
| **Time Ago** | When the spike occurred |
| **Route** | API route active during spike |
| **Trace ID** | Signoz trace for correlation |
| **Culprit Function** | Code function from profiler |

### Resource Gravity Scores

The Gravity Score identifies services that may need architectural refactoring:

| Score | Meaning | Action |
|-------|---------|--------|
| **0-3** | Low impact | Monitor periodically |
| **3-6** | Medium impact | Consider optimization |
| **6-10** | High impact | **Priority refactoring** |

Routes tagged with `[Suspected Job]` match patterns like `/tasks/*`, `/batch/*`, `/jobs/*` - these are candidates for extraction to separate microservices.

---

## Calculation Formulas

### 1. Spike Detection Formula

```
Spike Detected = Current Usage > Moving Average + (Threshold %)
```

**Example:**
- Moving Average: 50%
- Threshold: 50%
- Spike Trigger: > 75% (50% + 50%)

### 2. Moving Average Calculation

```
Moving Average = Σ(resource_usage) / N

Where:
- N = number of samples in the window (default: 60 samples = 30 minutes at 30s intervals)
- Window = configurable (default: 30 minutes)
```

### 3. Resource Gravity Score

```
Resource Gravity Score = Resource Peak × (1 / Request Frequency)

Where:
- Resource Peak = max(CPU peak %, RAM peak %) over analysis period
- Request Frequency = total requests / analysis period (7 days)

High Score = High Resource Usage + Low Call Frequency
```

**Example:**
- Endpoint `/v1/batch-process`: CPU peak 95%, called 5 times/day
- Score = 95 × (1/5) = 19 (very high - needs refactoring)

### 4. Route Selection (Culprit Detection)

When multiple routes are active during a spike:

```
Culprit Route = Route with Highest CPU + RAM consumption
```

Priority: CPU usage (primary) → RAM usage (secondary)

### 5. Baseline Comparison

```
vs Baseline % = ((Current - Baseline) / Baseline) × 100

Where:
- Baseline = Moving Average (30-minute window)
- Current = Latest sample
```

### 6. Cooldown Management

```
Next Alert Allowed = Last Alert Time + Cooldown Duration

Default: 15 minutes cooldown between alerts for the same pod
```

### 7. Reconciliation Buffer

```
Alert Sent = Spike Detection Time + Reconciliation Buffer

Default: 8-minute buffer to allow profiler data to become available
```

---

## API Reference

### Timeline API

```
GET /api/v1/timeline?range=24h&namespace=production&pod=api
```

**Parameters:**
| Parameter | Type | Description |
|-----------|------|-------------|
| `range` | string | Time range: 1h, 6h, 24h, 7d |
| `namespace` | string | Filter by namespace |
| `pod_name` | string | Filter by pod name (supports partial match) |

**Response:**
```json
{
  "metrics": [
    {
      "timestamp": "2026-04-13T14:30:00Z",
      "pod_name": "api-service-abc123",
      "replicaset_name": "api-service",
      "namespace": "production",
      "cpu_percent": 75.0,
      "ram_percent": 60.0,
      "baseline": 50.0
    }
  ],
  "availablePods": [
    {
      "name": "api-service",
      "namespace": "production",
      "cpuPercent": 65,
      "ramPercent": 45
    }
  ]
}
```

### Spikes API

```
GET /api/v1/spikes?limit=50&sort=timestamp&direction=desc
```

**Response:**
```json
{
  "spikes": [
    {
      "id": "uuid",
      "timestamp": "2026-04-13T14:30:00Z",
      "pod_name": "api-service-abc123",
      "namespace": "production",
      "cpu_usage_percent": 85.0,
      "ram_usage_percent": 72.0,
      "route_name": "/v1/process",
      "trace_id": "abc123-xyz",
      "culprit_function": "processBatch",
      "alert_sent": true
    }
  ],
  "total": 150
}
```

### Gravity Scores API

```
GET /api/v1/gravity-scores
```

**Response:**
```json
{
  "scores": [
    {
      "service": "api-service",
      "resourceGravityScore": 8.5,
      "cpuPeak": 95.0,
      "ramPeak": 78.0,
      "requestCount": 15000,
      "isJobRoute": false
    },
    {
      "service": "batch-worker",
      "resourceGravityScore": 19.0,
      "cpuPeak": 98.0,
      "ramPeak": 85.0,
      "requestCount": 5,
      "isJobRoute": true,
      "suggestedSeparation": "Extract to separate microservice"
    }
  ]
}
```

### Export API

```
GET /api/v1/export?type=spikes&start=2026-04-01&end=2026-04-13
GET /api/v1/export?type=refactoring
```

---

## Configuration

### Detection Settings

Edit `configs/config.yaml`:

```yaml
detection:
  threshold_percent: 50          # Spike detection threshold (default: 50%)
  polling_interval_seconds: 30   # How often to poll Prometheus
  moving_average_window_minutes: 30  # Baseline calculation window
  baseline_learning_period_minutes: 30  # Initial learning period
  reconciliation_buffer_minutes: 8  # Wait for profiler data
  cooldown_minutes: 15           # Alert cooldown
```

### Discord Alerts

```yaml
discord:
  enabled: true
  webhook_url: "https://discord.com/api/webhooks/..."
```

### Namespace Filtering

```yaml
namespaces:
  - "production"
  - "staging"
```

---

## Architecture

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                        Resource-to-Code Correlation Engine                 │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                             │
│  ┌─────────────────┐     ┌─────────────────┐     ┌─────────────────┐     │
│  │   Prometheus    │     │ Signoz/Clickhouse│    │ Gcloud Profiler │     │
│  │   (Metrics)     │     │    (Traces)     │     │   (Samples)     │     │
│  └────────┬────────┘     └────────┬────────┘     └────────┬────────┘     │
│           │                        │                        │              │
│           └────────────────────────┼────────────────────────┘              │
│                                    │                                       │
│                           ┌────────▼────────┐                               │
│                           │  Correlation   │                               │
│                           │    Engine      │                               │
│                           │   (Golang)     │                               │
│                           └────────┬────────┘                               │
│                                    │                                       │
│              ┌─────────────────────┼─────────────────────┐                  │
│              │                     │                     │                   │
│     ┌────────▼────────┐  ┌────────▼────────┐  ┌────────▼────────┐          │
│     │ SQLite Database│  │ Discord Alert  │  │  Web Dashboard │          │
│     │   (Storage)    │  │    (Webhook)   │  │  (Frontend)    │          │
│     └─────────────────┘  └─────────────────┘  └─────────────────┘          │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
```

---

## Known Issues

- **Port Mismatch**: Frontend proxy in `vite.config.ts` targets port 8080, but backend defaults to 8081. Update vite.config.ts or config.yaml to match.

---

## License

MIT License - See LICENSE file for details
