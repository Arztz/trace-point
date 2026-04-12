# Technical Plan: Dashboard Timeline Fix v1.0.1

## Document Information

| | |
|---|---|
| **Document Title** | Dashboard Timeline Fix - v1.0.1 Technical Plan |
| **Version** | 1.0 |
| **Date** | 2026-04-12 |
| **Status** | Draft for Review |
| **Target Release** | v1.0.1 |

---

## 1. Problem Statement

### 1.1 Current Behavior

The dashboard timeline chart displays data **only when spike events exist** in the SQLite database. When no spike events have been recorded (e.g., during normal operation without threshold breaches), the timeline shows an empty state with no data points.

### 1.2 Expected Behavior (per Requirements)

According to the Application Requirement Document (FR-001):

> "The system shall display a unified timeline graph showing CPU utilization as a line chart and RAM utilization as an area chart, synchronized on a common time axis"

The timeline should show **continuous metrics** from Prometheus at all times, regardless of whether spike events have been recorded. Spike events should trigger **notifications** (Discord alerts), not control dashboard visibility.

### 1.3 Root Cause Analysis

| Component | Issue |
|-----------|-------|
| **Backend: handleTimeline()** | Only queries `ListSpikeEvents()` from SQLite - returns data only when spikes exist |
| **Prometheus Client** | Has `QueryRange()` capability but is not used by timeline endpoint |
| **Data Flow** | Timeline endpoint conflates "spike events" with "timeline data" - they should be separate |
| **Frontend** | Dashboard shows EmptyState when no spike events exist |

### 1.4 Impact

- User cannot see resource utilization during normal operation
- Dashboard appears broken during periods without spikes
- Violates FR-001 requirement for continuous timeline display
- Poor user experience - users expect to see CPU/RAM graphs always

---

## 2. Architecture Changes

### 2.1 Current Architecture

```
/api/v1/timeline
    │
    ▼
handlers.handleTimeline()
    │
    ▼
storage.ListSpikeEvents() ──▶ SQLite (spike_events table)
    │
    ▼
Response: [SpikeEvent, SpikeEvent, ...]
```

**Problem**: Only returns spike events, not continuous metrics.

### 2.2 Target Architecture

```
/api/v1/timeline
    │
    ├─── Primary Path: Continuous Metrics
    │       │
    │       ▼
    │    Prometheus.QueryRange() ──▶ CPU/RAM time series
    │       │
    │       ▼
    │    SQLite (metrics_cache table) ──▶ Store recent for fast load
    │       │
    │       ▼
    │    Response: [MetricPoint, MetricPoint, ...]
    │
    └─── Spike Overlay Path (unchanged)
            │
            ▼
         SQLite (spike_events table)
            │
            ▼
         Response: Spike markers overlaid on timeline
```

### 2.3 Data Separation

| Data Type | Source | Purpose | Storage |
|-----------|--------|---------|---------|
| **Continuous Metrics** | Prometheus QueryRange | Timeline visualization | Redis or SQLite cache |
| **Spike Events** | Correlation Engine | Alerting + markers | SQLite (existing) |

---

## 3. API Changes

### 3.1 Modified Endpoint: GET /api/v1/timeline

**Current Response:**
```json
{
  "generated_at": "2026-04-12T10:00:00Z",
  "start_date": "2026-04-11T10:00:00Z",
  "end_date": "2026-04-12T10:00:00Z",
  "data_points": [SpikeEvent, SpikeEvent, ...]
}
```

**New Response:**
```json
{
  "generated_at": "2026-04-12T10:00:00Z",
  "start_date": "2026-04-11T10:00:00Z",
  "end_date": "2026-04-12T10:00:00Z",
  "metrics": [
    {
      "timestamp": "2026-04-12T09:30:00Z",
      "pod_name": "api-pod-abc123",
      "namespace": "production",
      "cpu_percent": 45.2,
      "ram_percent": 62.8
    },
    ...
  ],
  "spike_markers": [
    {
      "timestamp": "2026-04-12T08:15:00Z",
      "pod_name": "api-pod-abc123",
      "cpu_spike": true,
      "ram_spike": false
    },
    ...
  ]
}
```

### 3.2 New Query Parameters

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `source` | string | "all" | "prometheus" (metrics only), "spikes" (markers only), "all" |
| `step` | string | "30s" | Prometheus query step (e.g., "30s", "1m", "5m") |
| `aggregation` | string | "avg" | "avg", "max", "min" for downsampling |

### 3.3 Response Fields

| Field | Type | Description |
|-------|------|-------------|
| `metrics` | array | Continuous CPU/RAM data points from Prometheus |
| `spike_markers` | array | Spike event markers (for overlay on timeline) |
| `generated_at` | string | Timestamp when response was generated |

---

## 4. Data Flow

### 4.1 Continuous Metrics Flow

```
┌─────────────────┐
│ Dashboard loads │
└────────┬────────┘
         │ GET /api/v1/timeline?time_range=24h
         ▼
┌─────────────────────────────────────────────────┐
│ handleTimeline()                                │
│                                                 │
│ 1. Parse time_range query param                │
│ 2. Calculate start/end times                   │
│ 3. Query Prometheus.QueryRange() for CPU       │
│ 4. Query Prometheus.QueryRange() for RAM        │
│ 5. Merge results by timestamp                   │
│ 6. Downsample if too many points                │
│ 7. Cache in SQLite metrics_cache table         │
└────────┬────────────────────────────────────────┘
         │ Return continuous metrics
         ▼
┌─────────────────────────────────────────────────┐
│ Frontend TimelineChart                          │
│                                                 │
│ - Render CPU line from metrics[]               │
│ - Render RAM area from metrics[]               │
│ - Overlay spike_markers as vertical lines      │
└─────────────────────────────────────────────────┘
```

### 4.2 Backend Components to Modify

| Component | File | Changes |
|-----------|------|---------|
| **Prometheus Client** | `internal/integration/prometheus/client.go` | Add `QueryTimelineMetrics()` method |
| **Handlers** | `internal/server/handlers/handlers.go` | Modify `handleTimeline()` to query Prometheus |
| **Storage** | `internal/storage/repository.go` | Add `SaveMetricsCache()`, `GetMetricsCache()` |
| **Config** | `configs/config.yaml` | Add metrics caching config |

### 4.3 Metrics Caching Strategy

**Rationale**: Querying Prometheus for every dashboard load is expensive and may be slow (NFR-001: ≤5s query, NFR-004: ≤3s dashboard load).

**Solution**: Cache recent metrics in SQLite for fast dashboard loading.

| Cache Strategy | Implementation |
|----------------|----------------|
| **Primary** | Query Prometheus directly for fresh data |
| **Fallback** | If Prometheus fails, serve from cache |
| **Cache Duration** | 5 minutes (configurable) |
| **Retention** | 24 hours of cached metrics |

**Schema Addition:**
```sql
CREATE TABLE metrics_cache (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    pod_name TEXT NOT NULL,
    namespace TEXT NOT NULL,
    cpu_percent REAL NOT NULL,
    ram_percent REAL NOT NULL,
    timestamp DATETIME NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_metrics_cache_timestamp ON metrics_cache(timestamp);
CREATE INDEX idx_metrics_cache_namespace ON metrics_cache(namespace);
```

### 4.4 Background Metric Collection

```
┌─────────────────────────────────────────────────────────────┐
│ Background Worker (correlation engine or new)              │
│                                                             │
│ Every 30s (configurable):                                   │
│  1. Fetch CPU metrics from Prometheus                     │
│  2. Fetch RAM metrics from Prometheus                     │
│  3. Merge into metrics_cache table                         │
│  4. Prune old records (>24h old)                          │
└─────────────────────────────────────────────────────────────┘
```

---

## 5. Backward Compatibility

### 5.1 Preserved Functionality

| Feature | Status | Notes |
|---------|--------|-------|
| **Spike Detection** | ✅ Unchanged | Continues as before in correlation engine |
| **Discord Alerting** | ✅ Unchanged | Spike events still trigger alerts |
| **Spike List API** | ✅ Unchanged | `/api/v1/spikes` returns spike events |
| **Gravity Scores** | ✅ Underived | Calculated from spike events |
| **Refactoring Export** | ✅ Unchanged | Based on spike events |

### 5.2 Data Model Changes

| Change | Compatibility | Migration |
|--------|---------------|-----------|
| New `metrics` field in timeline response | ✅ Backward compatible | Frontend handles missing field gracefully |
| New `spike_markers` field | ✅ Backward compatible | Frontend handles missing field |
| New `metrics_cache` table | ✅ Additive | No migration needed |

### 5.3 Frontend Compatibility

The frontend transformer function (`transformTimelineData` in Dashboard.tsx) should handle both old and new response formats:

```typescript
// Pseudocode for frontend transformation
function transformTimelineData(raw: unknown): TimelineData {
  // Check for new format (prometheus metrics)
  if (raw.metrics && Array.isArray(raw.metrics)) {
    return transformFromPrometheusFormat(raw);
  }
  
  // Fallback to old format (spike events)
  return transformFromSpikeEventsFormat(raw);
}
```

---

## 6. Implementation Steps

### Phase 1: Backend Changes (Day 1)

**Step 1.1**: Add Prometheus range query method
- File: `internal/integration/prometheus/client.go`
- Add `QueryTimelineMetrics()` method that queries CPU and RAM with range
- Return normalized `TimelineMetric` struct

**Step 1.2**: Add metrics cache storage
- File: `internal/storage/repository.go`
- Add `SaveTimelineMetrics()` method
- Add `GetTimelineMetrics()` method
- Add `PruneOldMetrics()` method

**Step 1.3**: Modify timeline handler
- File: `internal/server/handlers/handlers.go`
- Update `handleTimeline()` to:
  1. Accept new query params (`source`, `step`, `aggregation`)
  2. Query Prometheus for continuous metrics
  3. Query SQLite for spike markers
  4. Merge into unified response

**Step 1.4**: Add background metric collection
- File: `internal/correlation/engine.go` or new file
- Add goroutine to collect metrics every 30s
- Store in metrics_cache table

### Phase 2: Frontend Changes (Day 1-2)

**Step 2.1**: Update API service
- File: `ui/src/services/api.ts`
- Update `timeline.get()` to handle new response format
- Update types in `ui/src/types/index.ts`

**Step 2.2**: Update TimelineChart component
- File: `ui/src/components/TimelineChart.tsx`
- Accept continuous metrics format
- Render CPU line and RAM area from metrics array
- Render spike markers as overlay

**Step 2.3**: Update Dashboard transformer
- File: `ui/src/pages/Dashboard.tsx`
- Update `transformTimelineData()` to handle both formats

### Phase 3: Testing & Integration (Day 2)

**Step 3.1**: Backend unit tests
- Test Prometheus query methods
- Test handler response format
- Test metrics caching

**Step 3.2**: Integration test
- Test timeline endpoint with Prometheus connected
- Verify response contains metrics

**Step 3.3**: Frontend visual test
- Verify timeline shows data when no spikes exist
- Verify spike markers overlay correctly

---

## 7. Testing Strategy

### 7.1 Backend Tests

| Test | Type | Method |
|------|------|--------|
| Prometheus QueryRange | Unit | Mock Prometheus, verify query params |
| Timeline Handler | Integration | Call endpoint, verify response format |
| Metrics Cache | Unit | Test save/get/prune operations |
| Spike Detection | Regression | Verify unchanged behavior |

### 7.2 API Contract Tests

| Test | Description |
|------|-------------|
| New timeline response format | Verify `metrics` and `spike_markers` fields present |
| Query params | Verify `source`, `step`, `aggregation` work |
| Backward compatibility | Mock old response, verify frontend handles it |

### 7.3 Frontend Tests

| Test | Description |
|------|-------------|
| Empty spikes | Verify timeline shows when no spike events |
| With spikes | Verify spike markers overlay correctly |
| Data transformation | Both old and new formats handled |

### 7.4 Manual Testing Checklist

- [ ] Dashboard loads with data when no spikes exist
- [ ] Timeline shows CPU line and RAM area charts
- [ ] Spike markers appear on timeline at correct timestamps
- [ ] Time range selector (1h, 6h, 24h, 7d) works correctly
- [ ] Namespace/pod filters apply to metrics
- [ ] Refresh button updates data
- [ ] Spike list still works (separate tab)
- [ ] Discord alerts still fire on spikes (unchanged)

---

## 8. Configuration Changes

### 8.1 New Config Options

```yaml
metrics:
  # Cache duration for timeline metrics (minutes)
  cache_duration_minutes: 5
  
  # How long to retain cached metrics (hours)
  cache_retention_hours: 24
  
  # Prometheus query step for timeline
  default_step: "30s"
  
  # Background collection interval (seconds)
  collection_interval_seconds: 30
```

### 8.2 Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `METRICS_CACHE_ENABLED` | Enable metrics caching | "true" |
| `METRICS_STEP` | Default Prometheus step | "30s" |

---

## 9. Risk Assessment

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| Prometheus query slow | Medium | High | Use cached data as fallback |
| Too many data points | Medium | Medium | Implement downsampling |
| Frontend breaking change | Low | High | Maintain backward compatibility |
| Cache out of sync | Medium | Medium | Timestamp validation |

### 9.1 Mitigation Strategies

| Risk | Mitigation |
|------|------------|
| Slow Prometheus | Cache 5 min, show stale data with indicator |
| Too many points | Server-side aggregation to max 1000 points |
| Breaking frontend | Both old and new response formats supported |

---

## 10. Success Criteria

### 10.1 Functional Criteria

- [ ] Timeline shows continuous CPU/RAM data when no spikes exist
- [ ] Spike markers still overlay on timeline
- [ ] Discord alerting continues to work (unchanged)
- [ ] Time range selection works (1h, 6h, 24h, 7d)
- [ ] Namespace/pod filters apply to metrics

### 10.2 Performance Criteria

| Metric | Target | Measurement |
|--------|--------|-------------|
| Timeline API response | ≤500ms | Manual test |
| Dashboard load time | ≤3s (NFR-004) | Manual test |
| Prometheus query | ≤5s (NFR-001) | Manual test |

### 10.3 Compatibility Criteria

- [ ] Old API response format still works
- [ ] Spike detection unchanged
- [ ] Refactoring intelligence unchanged

---

## 11. Out of Scope (v1.0.1)

The following are explicitly NOT in scope for v1.0.1:

| Feature | Reason | Future Version |
|---------|--------|----------------|
| Real-time WebSocket updates | Enhancement | v1.1.0 |
| Multi-cluster support | Enhancement | v1.2.0 |
| Advanced aggregation UI | Enhancement | v1.1.0 |
| Prometheus recording rules | Infrastructure | - |

---

## 12. References

- **Requirements Document**: `docs/requirement-application.md`
- **FR-001**: Unified timeline graph requirement
- **NFR-001**: Prometheus polling ≤5s
- **NFR-004**: Dashboard load ≤3s
- **Current Implementation**: `internal/server/handlers/handlers.go:112-151`

---

## Appendix A: Timeline API Response Schema

```typescript
interface TimelineResponse {
  // Timing metadata
  generated_at: string;    // RFC3339 timestamp
  start_date: string;      // RFC3339 timestamp
  end_date: string;        // RFC3339 timestamp
  
  // Continuous metrics from Prometheus
  metrics: TimelineMetric[];
  
  // Spike markers for overlay
  spike_markers: SpikeMarker[];
}

interface TimelineMetric {
  timestamp: string;       // RFC3339
  pod_name: string;
  namespace: string;
  cpu_percent: number;     // 0-100
  ram_percent: number;     // 0-100
}

interface SpikeMarker {
  timestamp: string;       // RFC3339
  pod_name: string;
  namespace: string;
  cpu_spike: boolean;
  ram_spike: boolean;
  route_name?: string;
  trace_id?: string;
}
```

---

## Appendix B: Related Files

| File | Description |
|------|-------------|
| `internal/server/handlers/handlers.go` | Timeline handler (line 112-151) |
| `internal/integration/prometheus/client.go` | Prometheus client with QueryRange |
| `internal/storage/repository.go` | Database repository |
| `ui/src/pages/Dashboard.tsx` | Main dashboard page |
| `ui/src/components/TimelineChart.tsx` | Timeline visualization |
| `ui/src/services/api.ts` | Frontend API service |
| `docs/requirement-application.md` | Requirements specification |
