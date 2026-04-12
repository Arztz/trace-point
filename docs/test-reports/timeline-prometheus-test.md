# Timeline API Test Report - Real Prometheus Connection

**Date:** 2026-04-12  
**Tester:** QA Validation

---

## Executive Summary

The Timeline API is connected to a real Prometheus endpoint (`http://prod.prometheus.fundii-prod.internal`), but **no timeline metrics are being returned** despite Prometheus having relevant data. The issue is with the query execution method, not data availability.

---

## Test Results

### 1. Prometheus Connectivity

| Test | Result |
|------|--------|
| Prometheus reachable | ✅ YES |
| `up` query works | ✅ Returns 75 active targets |
| CPU metrics exist | ✅ Range query returns 952 results |
| Namespace "fundii" exists | ✅ YES |

**Finding:** Prometheus is fully accessible and contains container metrics.

### 2. Timeline API Response

```
GET /api/v1/timeline?time_range=1h

Response:
{
  "metrics_count": 0,
  "spike_markers_count": 1,
  "has_data": false
}
```

**Finding:** The Timeline API returns an empty `metrics` array but does have 1 spike marker from test data.

### 3. Query Method Comparison

| Query Type | Command | Result |
|------------|---------|--------|
| Instant query | `container_cpu_usage_seconds_total` | ❌ Returns null |
| Range query | Container CPU with time range | ✅ Returns 952 results |

**Root Cause:** The code uses Prometheus **instant queries** (`/api/v1/query`) but the metric data appears to only be available via **range queries** (`/api/v1/query_range`).

---

## Investigation Details

### What Works
- `up` metric instant query works
- Range queries return container metrics
- kube-state-metrics target is healthy (job: "kube-state-metrics", health: "up")
- Only 1 pod in fundii namespace is being scraped: `prometheus-elasticsearch-exporter`

### What Doesn't Work
- `container_cpu_usage_seconds_total` instant query returns null
- `kube_pod_container_resource_requests` instant query returns null
- All container usage metrics return null on instant query

### Possible Causes

1. **Prometheus scrape config issue:** The kubelet cadvisor metrics might be scraped but not indexed for instant queries
2. **Time range issue:** Instant queries require exact timestamp matching, range queries aggregate over time
3. **Missing scrape for fundii namespace:** Only 1 pod being scraped in fundii namespace

---

## Configuration Check

**config.yaml Prometheus settings:**
```yaml
prometheus:
  url: "http://prod.prometheus.fundii-prod.internal"
  timeout: 30s
  scrape_interval: 15s
  query_path: "/api/v1/query"  # Uses instant query
```

**namespaces monitored:**
```yaml
namespaces:
  - "fundii"
```

---

## Spike Markers

The spike marker in the response comes from test data:
```
test-1 | 2026-04-12T16:00:00Z | test-pod-1 | fundii
```

This confirms the spike_markers array works correctly with stored data.

---

## Recommendations

### Priority 1: Fix Query Method
Modify `QueryTimelineMetrics` in `internal/integration/prometheus/client.go` to use range queries instead of instant queries, or implement a fallback mechanism.

### Priority 2: Investigate Prometheus Scrape Config
The Prometheus server may need configuration changes to enable instant queries for container metrics, or ensure consistent metric labeling.

### Priority 3: Add Debug Logging
Add detailed logging in the timeline handler to capture:
- The exact Prometheus queries being executed
- Query response status and result count
- Any error messages from Prometheus

### Priority 4: Alternative Queries
Try alternative queries that work with instant queries:
```promql
# Using kube-state-metrics (which works for instant queries)
kube_pod_container_resource_requests{namespace="fundii"}
```

---

## Code Location

Key files to modify:
- `internal/integration/prometheus/client.go` - `QueryTimelineMetrics` function
- `internal/integration/prometheus/queries.go` - Query templates
- `internal/server/handlers/handlers.go` - Timeline handler

---

## Test Artifacts

- Prometheus endpoint: `http://prod.prometheus.fundii-prod.internal`
- Backend port: 8081
- Timeline endpoint: `http://localhost:8081/api/v1/timeline`
- Database: SQLite at `./data/trace-point.db`
