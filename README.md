# Trace-Point: Resource-to-Code Correlation Engine

<p align="center">
  <img src="https://img.shields.io/badge/Go-1.25+-00ADD8?style=flat&logo=go" alt="Go Version">
  <img src="https://img.shields.io/badge/React-18+-61DAFB?style=flat&logo=react" alt="React Version">
  <img src="https://img.shields.io/badge/License-MIT-green.svg" alt="License">
  <img src="https://img.shields.io/badge/Status-Active-success.svg" alt="Status">
</p>

Trace-Point is an automated tool designed to correlate Kubernetes resource spikes with code-level root causes. It integrates data from Prometheus (metrics), Signoz (traces), and Google Cloud Profiler (samples) to provide complete root cause analysis through a web dashboard and Discord alerts.

## Key Features

- **Automatic Spike Detection**: Detects resource spikes using moving average algorithm with configurable thresholds
- **Trace Correlation**: Links resource spikes to active API routes via Signoz/Clickhouse trace data
- **Profiler Enrichment**: Identifies specific culprit functions using Google Cloud Profiler samples
- **Real-time Alerts**: Sends Discord webhooks with complete RCA information (Route, Impact, Culprit)
- **Resource Gravity Scores**: Calculates scores to identify services requiring architectural refactoring
- **JSON Export**: Exports spike history and refactoring recommendations for documentation and planning

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

## Technology Stack

| Component | Technology | Version |
|-----------|------------|---------|
| Backend | Go | 1.25.0 |
| HTTP Router | Chi | v5.0.10 |
| Database | SQLite (modernc.org/sqlite) | 3.x |
| Frontend | React | 18.x |
| Build Tool | Vite | 5.x |
| Charts | Recharts | 2.10.x |
| State Management | TanStack Query | 5.x |
| Styling | Tailwind CSS | 3.3.x |
| Configuration | Viper | 1.18.x |

## Prerequisites

- **Go** 1.21 or higher
- **Node.js** 18 or higher
- **npm** or **yarn**
- Access to a Kubernetes cluster with:
  - Prometheus (metrics)
  - Signoz (distributed tracing)
  - Google Cloud Profiler (optional, for profiler enrichment)

## Quick Start

### 1. Clone and Install Dependencies

```bash
# Clone the repository
cd trace-point

# Install Go dependencies
go mod download

# Install frontend dependencies
cd ui && npm install && cd ..
```

### 2. Configure the Application

Edit `configs/config.yaml` to match your environment:

```yaml
app:
  host: "0.0.0.0"
  port: 8081
  mode: "debug"

prometheus:
  url: "http://localhost:9090"

signoz:
  url: "http://localhost:3301"

gcloud:
  project_id: "your-gcp-project-id"
  region: "us-central1"

detection:
  cpu_threshold: 80
  memory_threshold: 85
  window_size: 5m
  min_samples: 3

discord:
  enabled: false
  webhook_url: ""

database:
  path: "./data/trace-point.db"

namespaces:
  - "production"
  - "staging"
```

### 3. Run the Application

**Backend (Terminal 1):**
```bash
go run cmd/server/main.go
```

**Frontend (Terminal 2):**
```bash
cd ui && npm run dev
```

### 4. Access the Dashboard

| Service | URL | Description |
|---------|-----|-------------|
| Frontend | http://localhost:3000 | Dashboard UI |
| Backend API | http://localhost:8081 | REST API |
| Health | http://localhost:8081/health | Health check |

## Configuration Reference

### Detection Settings

| Parameter | Default | Description |
|-----------|---------|-------------|
| `cpu_threshold` | 80% | CPU spike detection threshold |
| `memory_threshold` | 85% | Memory spike detection threshold |
| `window_size` | 5m | Detection window size |
| `min_samples` | 3 | Minimum data points required |

### Database Settings

| Parameter | Default | Description |
|-----------|---------|-------------|
| `database.path` | ./data/trace-point.db | SQLite database file path |
| `database.max_connections` | 25 | Maximum concurrent connections |
| `database.idle_connections` | 10 | Idle connection pool size |

### Namespace Filtering

Configure which Kubernetes namespaces to monitor:
```yaml
namespaces:
  - "production"
  - "staging"
```

Exclude specific pods using regex patterns:
```yaml
pod_exclude_patterns:
  - "^kube-"
  - "^system-"
```

## API Endpoints

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/health` | GET | Health check |
| `/api/v1/spikes` | GET | List spike events (supports `namespace`, `pod` query params) |
| `/api/v1/spikes/{id}` | GET | Get spike event by ID |
| `/api/v1/timeline` | GET | Get timeline data for visualization |
| `/api/v1/export` | GET | Export spike history as JSON |
| `/api/v1/export/refactoring` | GET | Export refactoring recommendations |
| `/api/v1/config` | GET | Get current configuration |
| `/api/v1/gravity-scores` | GET | Get resource gravity scores |

### Example API Calls

```bash
# Get spike events
curl http://localhost:8081/api/v1/spikes

# Get spike events for specific namespace
curl "http://localhost:8081/api/v1/spikes?namespace=production"

# Get timeline data
curl http://localhost:8081/api/v1/timeline

# Get resource gravity scores
curl http://localhost:8081/api/v1/gravity-scores
```

## Spike Detection Algorithm

```
1. Poll Prometheus every N seconds (default: 30)
2. Calculate moving average for each pod over 30-minute window
3. Compare current usage to moving average + threshold%
4. If spike detected:
   a. Log spike event to SQLite
   b. Start reconciliation timer (8 minutes)
   c. Set cooldown end time (15 minutes)
5. After reconciliation buffer:
   a. Query Signoz for trace data
   b. Fetch Gcloud Profiler samples
   c. Correlate to identify culprit
   d. Send Discord alert
6. Apply cooldown before next spike for same pod
```

## Resource Gravity Score

The Resource Gravity Score identifies APIs with high resource usage but low call frequency:

```
High Score = (High Resource Peak) × (Low Call Frequency)

Where:
- High Resource Peak = max(CPU peak, RAM peak) over analysis period
- Low Call Frequency = inverse of request count
```

Routes matching patterns (`/tasks/*`, `/batch/*`, `/jobs/*`) are tagged as `[Suspected Job]`.

## Discord Alert Format

When a spike is detected and the reconciliation buffer has passed, a Discord alert is sent:

```json
{
  "embeds": [{
    "title": "[CRITICAL] Resource Spike Detected - pod-name",
    "color": "16711680",
    "fields": [
      {"name": "Route", "value": "/v1/summarize", "inline": true},
      {"name": "CPU Impact", "value": "+45% (from 30% to 75%)", "inline": true},
      {"name": "RAM Impact", "value": "+20% (from 40% to 60%)", "inline": true},
      {"name": "Culprit Function", "value": "processBatch() in workers/processor.go"},
      {"name": "Timestamp", "value": "2026-04-11T14:30:00Z"},
      {"name": "Trace ID", "value": "abc123-xyz789"}
    ]
  }]
}
```

## Project Structure

```
trace-point/
├── cmd/
│   └── server/
│       └── main.go              # Application entry point
├── internal/
│   ├── config/
│   │   └── config.go            # Configuration loading
│   ├── correlation/
│   │   ├── detector.go          # Spike detection
│   │   ├── engine.go            # Correlation logic
│   │   └── router.go            # Route identification
│   ├── integration/
│   │   ├── prometheus/          # Prometheus client
│   │   ├── signoz/              # Signoz client
│   │   ├── profiler/            # Gcloud Profiler client
│   │   └── discord/             # Discord webhook client
│   ├── storage/
│   │   ├── database.go          # SQLite connection
│   │   ├── repository.go        # Data access layer
│   │   └── models.go            # Data models
│   └── server/
│       └── handlers/            # HTTP handlers
├── ui/
│   ├── src/
│   │   ├── components/          # Reusable components
│   │   ├── pages/               # Page components
│   │   ├── services/            # API services
│   │   └── types/               # TypeScript types
│   ├── package.json
│   ├── vite.config.ts
│   └── tailwind.config.js
├── configs/
│   └── config.yaml              # Configuration
├── data/
│   └── trace-point.db           # SQLite database (auto-created)
└── README.md
```

## Running Tests

```bash
# Run Go tests
go test ./...

# Run a specific test
go test -v ./internal/correlation/

# Run frontend tests
cd ui && npm test
```

## Known Issues

- **Port Mismatch**: The frontend proxy in `vite.config.ts` targets `http://localhost:8080`, but the default backend port is `8081`. If you see 502 errors, update the proxy target in `vite.config.ts` to match your backend port in `config.yaml`.

## License

MIT License - See LICENSE file for details

## Contributing

Contributions are welcome! Please read the contributing guidelines before submitting PRs.