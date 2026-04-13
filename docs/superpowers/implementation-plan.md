# Trace-Point Implementation Plan

## Code Quality Issues Resolution

**Project**: Trace-Point  
**Version**: 1.0.2  
**Date**: 2026-04-13  
**Status**: Ready for Implementation

---

## Executive Summary

This implementation plan addresses critical performance, reliability, and frontend polish issues identified during the code review. The plan is organized into three phases:

- **Phase 1: Performance Fixes** - Addresses O(n²) sorting algorithms, unbounded data structures, and missing React memoization
- **Phase 2: Backend Reliability** - Fixes race conditions, error handling, and graceful shutdown
- **Phase 3: Frontend Polish** - Improves type safety, accessibility, and error handling

### Priority Summary

| Priority | Issue Category | Items | Impact |
|----------|----------------|-------|--------|
| CRITICAL | Performance | 4 | High latency, potential OOM |
| HIGH | Backend Reliability | 5 | Data corruption, crashes |
| MEDIUM | Frontend Polish | 6 | UX, maintainability |

---

## Phase 1: Performance Fixes

### 1.1 Frontend: Missing useMemo in TimelineChart

**Location**: `ui/src/components/TimelineChart.tsx`  
**Lines**: 127-181, 198-235, 284-337  
**Severity**: HIGH  
**Estimated Effort**: 1 hour

#### Current Code (lines 127-181)

```typescript
// Transform metrics into chart data format (one row per timestamp)
// Aggregate metrics by replicaset (average across all pods in that replicaset)
const chartData = (() => {
  // Group metrics by timestamp and replicaset
  const dataMap = new Map<string, Map<string, { cpu: number[]; ram: number[] }>>();
  
  // Sort metrics by timestamp
  const sortedMetrics = [...metrics].sort((a, b) => 
    a.timestamp.localeCompare(b.timestamp)
  );

  for (const metric of sortedMetrics) {
    // ... extensive computation
  }
  
  return result;
})();
```

#### Required Fix

Wrap `chartData`, `stats`, and `renderLines` in `useMemo` hooks to prevent expensive recalculation on every render:

```typescript
// Chart data - memoize by metrics and selected pods
const chartData = useMemo(() => {
  if (!metrics || metrics.length === 0) return [];
  
  const dataMap = new Map<string, Map<string, { cpu: number[]; ram: number[] }>>();
  const sortedMetrics = [...metrics].sort((a, b) => 
    a.timestamp.localeCompare(b.timestamp)
  );
  
  // ... computation
  
  return result;
}, [metrics, selectedPods]);

// Stats - memoize by chartData and displayReplicasets
const stats = useMemo(() => {
  // ... computation
}, [chartData, displayReplicasets]);

// Lines - memoize by displayReplicasets
const lines = useMemo(() => {
  // ... generate line elements
}, [displayReplicasets, highlightedPod, availablePods]);
```

#### Benefits

- Prevents recalculation on every render cycle
- Reduces CPU usage during scrolling and hover interactions
- Improves frame rate for animations

---

### 1.2 Frontend: Missing useCallback in Dashboard Handlers

**Location**: `ui/src/pages/Dashboard.tsx`  
**Lines**: 191-195, 197-212, 245-253  
**Severity**: HIGH  
**Estimated Effort**: 30 minutes

#### Current Code

```typescript
const handleRefresh = () => {
  queryClient.invalidateQueries({ queryKey: ['timeline'] });
  queryClient.invalidateQueries({ queryKey: ['spikes'] });
  queryClient.invalidateQueries({ queryKey: ['gravityScores'] });
};

const handleExport = async () => {
  try {
    const data = await api.export.json({ namespace, startTime: undefined, endTime: undefined });
    // ... export logic
  } catch (error) {
    console.error('Export failed:', error);
  }
};
```

#### Required Fix

Wrap handlers in `useCallback` to maintain referential equality:

```typescript
const handleRefresh = useCallback(() => {
  queryClient.invalidateQueries({ queryKey: ['timeline'] });
  queryClient.invalidateQueries({ queryKey: ['spikes'] });
  queryClient.invalidateQueries({ queryKey: ['gravityScores'] });
}, [queryClient]);

const handleExport = useCallback(async () => {
  try {
    const data = await api.export.json({ namespace: debouncedNamespace, startTime: undefined, endTime: undefined });
    const blob = new Blob([JSON.stringify(data, null, 2)], { type: 'application/json' });
    const url = URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = `trace-point-export-${new Date().toISOString().split('T')[0]}.json`;
    document.body.appendChild(a);
    a.click();
    document.body.removeChild(a);
    URL.revokeObjectURL(url);
  } catch (error) {
    console.error('Export failed:', error);
  }
}, [debouncedNamespace]);
```

---

### 1.3 Backend: O(n²) Bubble Sort in handlers.go (Gravity Scores)

**Location**: `internal/server/handlers/handlers.go`  
**Lines**: 464-470, 559-565  
**Severity**: CRITICAL  
**Estimated Effort**: 1 hour

#### Current Code (lines 463-470)

```go
// Sort by gravity score
for i := 0; i < len(scores)-1; i++ {
    for j := i + 1; j < len(scores); j++ {
        if scores[j].ResourceGravityScore > scores[i].ResourceGravityScore {
            scores[i], scores[j] = scores[j], scores[i]
        }
    }
}
```

#### Current Code (lines 558-565)

```go
// Sort by gravity score descending
for i := 0; i < len(recommendations)-1; i++ {
    for j := i + 1; j < len(recommendations); j++ {
        if recommendations[j].ResourceGravityScore > recommendations[i].ResourceGravityScore {
            recommendations[i], recommendations[j] = recommendations[j], recommendations[i]
        }
    }
}
```

#### Required Fix

Replace bubble sort with Go's built-in `sort.Slice` (O(n log n)):

```go
import "sort"

// Sort by gravity score descending (using sort.Slice - O(n log n))
sort.Slice(scores, func(i, j int) bool {
    return scores[i].ResourceGravityScore < scores[j].ResourceGravityScore
})
```

```go
// For recommendations (lines 558-565)
sort.Slice(recommendations, func(i, j int) bool {
    return recommendations[i].ResourceGravityScore < recommendations[j].ResourceGravityScore
})
```

#### Performance Impact

| Dataset Size | Bubble Sort | sort.Slice | Improvement |
|-------------|------------|------------|--------------|
| 100 items   | 10,000 ops | 664 ops     | 15x          |
| 1,000 items | 1,000,000 ops | 9,964 ops | 100x        |
| 10,000 items | 100,000,000 ops | 132,866 ops | 753x      |

---

### 1.4 Backend: Unbounded Metrics History in detector.go

**Location**: `internal/correlation/detector.go`  
**Lines**: 96-111  
**Severity**: HIGH  
**Estimated Effort**: 30 minutes

#### Current Code

```go
history := d.metricsHistory[key]
history = append(history, point)

// Prune old data points outside the moving average window
windowDuration := time.Duration(d.config.MovingAverageWindowMinutes) * time.Minute
cutoff := now.Add(-windowDuration)

prunedHistory := make([]MetricDataPoint, 0)
for _, p := range history {
    if p.Timestamp.After(cutoff) {
        prprunedHistory = append(prunedHistory, p)
    }
}
d.metricsHistory[key] = prunedHistory
```

#### Issue

While pruning is performed, the implementation creates a new slice on every update, causing memory allocations. Additionally, the slice capacity may grow unboundedly over time.

#### Required Fix

Optimize memory management with proper capacity hints and in-place pruning:

```go
history := d.metricsHistory[key]
history = append(history, point)

// Prune old data points outside the moving average window
windowDuration := time.Duration(d.config.MovingAverageWindowMinutes) * time.Minute
cutoff := now.Add(-windowDuration)

// Find cutoff index and truncate in-place
cutoffIdx := 0
for _, p := range history {
    if p.Timestamp.After(cutoff) {
        break
    }
    cutoffIdx++
}

// In-place truncation with capacity preservation
if cutoffIdx > 0 {
    d.metricsHistory[key] = append(history[:0:0], history[cutoffIdx:]...)
} else {
    d.metricsHistory[key] = history
}
```

#### Further Optimization

Add maximum history size to prevent unbounded growth in edge cases:

```go
const maxHistorySize = 120 // 1 hour of 30-second samples

// After pruning, ensure we don't exceed max size
if len(d.metricsHistory[key]) > maxHistorySize {
    d.metricsHistory[key] = d.metricsHistory[key][len(d.metricsHistory[key])-maxHistorySize:]
}
```

---

## Phase 2: Backend Reliability

### 2.1 Race Condition in Discord Cooldown Map

**Location**: `cmd/server/main.go`  
**Lines**: 114, 192-205  
**Severity**: CRITICAL  
**Estimated Effort**: 1 hour

#### Current Code

```go
// Initialize Discord cooldown tracker
discordCooldowns := make(map[string]time.Time)

// ... inside polling loop ...
if lastAlert, hasCooldown := discordCooldowns[spikeKey]; hasCooldown && timeSince(lastAlert) < cooldownDuration {
    logger.Debug("Skipping Discord alert for %s (cooldown active)", spikeKey)
} else {
    // Send Discord alert
    if discordClient != nil {
        // ... send alert
        discordCooldowns[spikeKey] = time.Now()  // RACE: concurrent read/write
    }
}
```

#### Issue

The `discordCooldowns` map is accessed concurrently by the polling goroutine without synchronization, causing potential race conditions.

#### Required Fix

Use `sync.RWMutex` for thread-safe access:

```go
import "sync"

// Discord cooldown tracker with mutex protection
var discordCooldowns struct {
    mu    sync.RWMutex
    data map[string]time.Time
}
discordCooldowns.data = make(map[string]time.Time)

// In polling loop - use lock for read
discordCooldowns.mu.RLock()
lastAlert, hasCooldown := discordCooldowns.data[spikeKey]
discordCooldowns.mu.RUnlock()

if hasCooldown && time.Since(lastAlert) < cooldownDuration {
    logger.Debug("Skipping Discord alert for %s (cooldown active)", spikeKey)
} else {
    // Send Discord alert
    if discordClient != nil {
        // ... send alert
        
        // Update cooldown with lock
        discordCooldowns.mu.Lock()
        discordCooldowns.data[spikeKey] = time.Now()
        discordCooldowns.mu.Unlock()
    }
}
```

#### Alternative: Use sync.Map

For simpler implementation with concurrent access:

```go
import "sync"

var discordCooldowns sync.Map // thread-safe map

// Check and update
if lastAlert, ok := discordCooldowns.Load(spikeKey); ok {
    if time.Since(lastAlert.(time.Time)) < cooldownDuration {
        // skip
        return
    }
}
discordCooldowns.Store(spikeKey, time.Now())
```

---

### 2.2 Goroutine Lacks Graceful Shutdown

**Location**: `cmd/server/main.go`  
**Lines**: 137-222  
**Severity**: HIGH  
**Estimated Effort**: 2 hours

#### Current Code

```go
// Start spike detection polling loop in goroutine
go func() {
    ticker := time.NewTicker(time.Duration(detectorConfig.PollingIntervalSeconds) * time.Second)
    defer ticker.Stop()

    logger.Info("Starting Prometheus metrics polling every %d seconds...", detectorConfig.PollingIntervalSeconds)

    // Initial poll
    // ... polling loop ...
    for {
        select {
        case <-ticker.C:
            // ... detection logic
        }
    }
}()
```

#### Issue

The goroutine has no mechanism to respond to shutdown signals, potentially causing:

- In-flight requests being cancelled
- Database connections left open
- Resource leaks

#### Required Fix

Add context-based graceful shutdown:

```go
// Create cancellation context for graceful shutdown
ctx, cancelPolling := context.WithCancel(context.Background())
defer cancelPolling()

// Start spike detection polling loop in goroutine
go func() {
    ticker := time.NewTicker(time.Duration(detectorConfig.PollingIntervalSeconds) * time.Second)
    defer ticker.Stop()

    logger.Info("Starting Prometheus metrics polling every %d seconds...", detectorConfig.PollingIntervalSeconds)

    for {
        select {
        case <-ticker.C:
            detectorCtx, detectorCancel := context.WithTimeout(ctx, 30*time.Second)
            spikes, err := detector.DetectSpikes(detectorCtx, prometheusClient, cfg.Namespaces, nil)
            detectorCancel()
            if err != nil {
                logger.Error("Spike detection error: %v", err)
                continue
            }
            // ... process spikes
        case <-ctx.Done():
            logger.Info("Polling loop shutting down...")
            return
        }
    }
}()

// In main(), signal handler:
// After receiving quit signal, call cancelPolling() before server.Shutdown()
```

---

### 2.3 HTTP Response Body Not Closed on Error

**Location**: `internal/integration/prometheus/client.go`  
**Lines**: 175-179, 215-219  
**Severity**: HIGH  
**Estimated Effort**: 30 minutes

#### Current Code (lines 175-179)

```go
resp, err := c.httpClient.Do(req)
if err != nil {
    return nil, fmt.Errorf("failed to execute query: %w", err)
}
defer resp.Body.Close()
```

#### Issue

The `defer` only executes if the function returns normally. On error paths before `defer`, the body may not be closed.

#### Required Fix

Ensure body is always closed, even on error:

```go
resp, err := c.httpClient.Do(req)
if err != nil {
    return nil, fmt.Errorf("failed to execute query: %w", err)
}

// Always close body
if resp.Body != nil {
    defer resp.Body.Close()
}

if resp.StatusCode != http.StatusOK {
    return nil, fmt.Errorf("Prometheus returned status %d", resp.StatusCode)
}
```

#### More Robust Pattern

Use a wrapper function for guaranteed cleanup:

```go
func ensureClose(resp *http.Response) {
    if resp != nil && resp.Body != nil {
        resp.Body.Close()
    }
}

resp, err := c.httpClient.Do(req)
if err != nil {
    return nil, fmt.Errorf("failed to execute query: %w", err)
}
defer ensureClose(resp)
```

---

### 2.4 Internal Errors Exposed to Clients

**Location**: `internal/server/handlers/handlers.go`  
**Lines**: Throughout error handling  
**Severity**: MEDIUM  
**Estimated Effort**: 2 hours

#### Current Code

```go
if err != nil {
    return nil, fmt.Errorf("failed to query metrics: %w", err)
}
```

#### Issue

Internal error messages (including stack traces, file paths, database details) may leak to clients via HTTP responses.

#### Required Fix

Create typed error responses that hide internal details:

```go
import "encoding/json"

// APIError represents a client-safe error response
type APIError struct {
    Code    string `json:"code"`
    Message string `json:"message"`
}

// WriteError writes a JSON-encoded error response
func WriteError(w http.ResponseWriter, status int, code, message string) {
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(status)
    json.NewEncoder(w).Encode(APIError{
        Code:    code,
        Message: message,
    })
}

// In handlers, replace:
if err != nil {
    logger.Error("Failed to query metrics: %v", err)  // Log internally
    return nil, WriteError(w, http.StatusInternalServerError, "QUERY_FAILED", "Failed to retrieve metrics")
}
```

#### Standard Error Codes

| Code | HTTP Status | Description |
|------|------------|-------------|
| `NOT_FOUND` | 404 | Resource not found |
| `INVALID_PARAMS` | 400 | Invalid request parameters |
| `QUERY_FAILED` | 500 | Backend query failed |
| `INTERNAL_ERROR` | 500 | Unexpected error |

---

### 2.5 Silent Parse Failure in detector.go

**Location**: `internal/correlation/detector.go`  
**Line**: 270  
**Severity**: MEDIUM  
**Estimated Effort**: 30 minutes

#### Current Code

```go
func parseKey(key string) struct {
    namespace     string
    podName       string
    containerName string
} {
    var parts struct {
        namespace     string
        podName       string
        containerName string
    }
    fmt.Sscanf(key, "%s/%s/%s", &parts.namespace, &parts.podName, &parts.containerName)
    return parts
}
```

#### Issue

`fmt.Sscanf` silently fails on malformed keys without any logging or error indication. Returns zero values on failure.

#### Required Fix

Return an error and log failures:

```go
import "strings"

// ParseKey parses a container key into its components
// Key format: "namespace/podName/containerName"
func ParseKey(key string) (namespace, podName, containerName string, err error) {
    parts := strings.Split(key, "/")
    if len(parts) != 3 {
        return "", "", "", fmt.Errorf("invalid key format: expected 3 parts, got %d", len(parts))
    }
    
    for i, part := range parts {
        if strings.TrimSpace(part) == "" {
            return "", "", "", fmt.Errorf("empty part at index %d in key: %q", i, key)
        }
    }
    
    return parts[0], parts[1], parts[2], nil
}
```

#### Usage Update

```go
// In DetectSpikes (line 178):
parts := parseKey(key)  // OLD

// Replace with:
namespace, podName, containerName, err := ParseKey(key)
if err != nil {
    d.logger.Warn("Failed to parse key %s: %v", key, err)
    continue
}
alert := SpikeAlert{
    Namespace:     namespace,
    PodName:      podName,
    ContainerName: containerName,
    // ...
}
```

---

## Phase 3: Frontend Polish

### 3.1 Debug console.log Statements in Production

**Location**: `ui/src/components/TimelineChart.tsx`  
**Lines**: 253, 257, 263, 267, 271, 279  
**Severity**: MEDIUM  
**Estimated Effort**: 30 minutes

#### Current Code

```typescript
const handleChartClick = (event: any, activePayload: any[]) => {
    // Debug: log the click event
    console.log('[TimelineChart] onClick:', { event, activePayload });
    
    // Only process if there are active payload elements clicked
    if (!activePayload || activePayload.length === 0 || !onHighlight) {
        console.log('[TimelineChart] No valid click - missing payload or onHighlight');
        return;
    }
    
    // Get the first clicked element data
    const element = activePayload[0];
    console.log('[TimelineChart] Clicked element:', element);
    
    if (element && element.dataKey) {
        const dataKey = String(element.dataKey);
        console.log('[TimelineChart] dataKey:', dataKey);
        
        // Extract replicaset name from dataKey
        const replicasetName = dataKey.replace('_cpu', '').replace('_ram', '');
        console.log('[TimelineChart] Extracted replicasetName:', replicasetName);
        
        handleLineClick(replicasetName);
    }
};

// Line click handler
const handleLineClickEvent = (replicasetName: string) => (event: any) => {
    console.log('[TimelineChart] Line clicked:', replicasetName, event);
    handleLineClick(replicasetName);
};
```

#### Required Fix

Remove all console.log statements or use conditional logging in development only:

```typescript
const handleChartClick = (event: unknown, activePayload: unknown[]) => {
    // Type-safe event handling (see 3.2)
    if (!activePayload || activePayload.length === 0 || !onHighlight) {
        return;
    }
    
    const element = activePayload[0];
    if (element && typeof element === 'object' && 'dataKey' in element) {
        const dataKey = String((element as { dataKey: unknown }).dataKey);
        const replicasetName = dataKey.replace('_cpu', '').replace('_ram', '');
        handleLineClick(replicasetName);
    }
};

const handleLineClickEvent = (replicasetName: string) => (_event: unknown) => {
    handleLineClick(replicasetName);
};
```

---

### 3.2 Type Safety Issues (any Types)

**Location**: `ui/src/components/TimelineChart.tsx`  
**Lines**: 251, 278  
**Severity**: MEDIUM  
**Estimated Effort**: 1 hour

#### Current Code

```typescript
const handleChartClick = (event: any, activePayload: any[]) => {
    // ...
};

const handleLineClickEvent = (replicasetName: string) => (event: any) => {
    // ...
};
```

#### Required Fix

Use proper types from Recharts library:

```typescript
import type { 
    ChartMouseHandler, 
    ActivePayload 
} from 'recharts';

// Use proper Recharts types
const handleChartClick: ChartMouseHandler = (event, activePayload) => {
    if (!activePayload || activePayload.length === 0 || !onHighlight) {
        return;
    }
    
    const element = activePayload[0];
    if (element && element.dataKey) {
        const dataKey = String(element.dataKey);
        const replicasetName = dataKey.replace('_cpu', '').replace('_ram', '');
        handleLineClick(replicasetName);
    }
};

const handleLineClickEvent = (replicasetName: string) => (_event: unknown) => {
    handleLineClick(replicasetName);
};
```

#### Add Type Definitions

```typescript
// In types/timeline.ts or a new types file
interface ChartClickEvent {
    activePayload?: Array<{
        dataKey?: string;
        name?: string;
        value?: number;
        color?: string;
    }>;
}
```

---

### 3.3 Export Error Has No User Feedback

**Location**: `ui/src/pages/Dashboard.tsx`  
**Lines**: 197-212  
**Severity**: MEDIUM  
**Estimated Effort**: 30 minutes

#### Current Code

```typescript
const handleExport = async () => {
    try {
        const data = await api.export.json({ namespace, startTime: undefined, endTime: undefined });
        // ... export logic
    } catch (error) {
        console.error('Export failed:', error);
    }
};
```

#### Required Fix

Add user-facing error feedback with toast notification:

```typescript
import { useState } from 'react';
// Or use existing toast/snackbar component

const [exportError, setExportError] = useState<string | null>(null);

const handleExport = useCallback(async () => {
    setExportError(null);
    try {
        const data = await api.export.json({ namespace: debouncedNamespace, startTime: undefined, endTime: undefined });
        const blob = new Blob([JSON.stringify(data, null, 2)], { type: 'application/json' });
        const url = URL.createObjectURL(blob);
        const a = document.createElement('a');
        a.href = url;
        a.download = `trace-point-export-${new Date().toISOString().split('T')[0]}.json`;
        document.body.appendChild(a);
        a.click();
        document.body.removeChild(a);
        URL.revokeObjectURL(url);
    } catch (error) {
        const message = error instanceof Error ? error.message : 'Export failed';
        setExportError(message);
        console.error('Export failed:', error);
    }
}, [debouncedNamespace]);

// Render error in UI
{exportError && (
    <div className="text-sm text-red-600 mt-2">
        Export failed: {exportError}
    </div>
)}
```

---

### 3.4 Missing ARIA Labels for Accessibility

**Locations**: Multiple components  
**Severity**: MEDIUM  
**Estimated Effort**: 2 hours

#### Affected Components

| Component | Missing Labels | Priority |
|-----------|--------------|----------|
| FilterBar | Search input, dropdowns | HIGH |
| TimeRangeSelector | Radio group | MEDIUM |
| PodSelector | Checkboxes | HIGH |
| SpikeList | Sort dropdown | MEDIUM |
| TimelineChart | Chart container | LOW |

#### Required Fix - Example (FilterBar)

**Location**: `ui/src/components/FilterBar.tsx`

```typescript
// Add ARIA labels to interactive elements
<input
    type="search"
    aria-label="Filter by namespace"
    aria-describedby="namespace-help"
    // ... existing props
/>

<select
    aria-label="Sort spikes by"
    aria-describedby="sort-help"
    // ... existing props
>
```

#### Required Fix - Example (TimelineChart)

```typescript
<div 
    role="application"
    aria-label="Resource timeline chart"
    aria-describedby="chart-description"
>
    <span id="chart-description" className="sr-only">
        Line chart showing CPU and RAM usage over time for selected replicasets
    </span>
    {/* ... chart content */}
</div>
```

#### Standard ARIA Patterns

| Pattern | ARIA Attribute |
|--------|---------------|
| Interactive chart | `role="img"` or `role="application"` |
| Form inputs | `aria-label`, `aria-describedby` |
| Buttons | `aria-label` if icon-only |
| Live regions | `aria-live="polite"` for dynamic content |
| Error messages | `aria-live="assertive"`, `role="alert"` |

---

### 3.5 Additional Type Safety Improvements

#### Location: `ui/src/types/`  
**Severity**: LOW  
**Estimated Effort**: 2 hours

#### Actions

1. **Replace remaining `any` types** with proper interfaces
2. **Add strict null checks** using TypeScript `strictNullChecks`
3. **Use Discriminated Unions** for state types

#### Example - Create Strict Types

```typescript
// types/api.ts
export interface TimelineResponse {
    dataPoints: DataPoint[];
    routes: Route[];
    startTime: string;
    endTime: string;
    metrics?: TimelineMetric[];
    availablePods?: PodInfo[];
}

export interface DataPoint {
    timestamp: string;
    cpu: number;
    ram: number;
}

// Strict - no optional properties in core types
export interface SpikeEvent {
    id: string;
    timestamp: string;
    podName: string;
    namespace: string;
    resourceType: 'cpu' | 'memory';
    threshold: number;
    currentValue: number;
    movingAverage: number;
    activeRoutes: string[];
    possibleRootCauses: string[];
}
```

---

### 3.6 Error Boundary for React Components

**Location**: `ui/src/App.tsx`  
**Severity**: LOW  
**Estimated Effort**: 1 hour

#### Issue

App crashes may show blank screen instead of recovery UI.

#### Required Fix

Add React Error Boundary:

```typescript
import { Component, ReactNode } from 'react';

interface Props {
    children: ReactNode;
    fallback?: ReactNode;
}

interface State {
    hasError: boolean;
    error?: Error;
}

class ErrorBoundary extends Component<Props, State> {
    state: State = { hasError: false };
    
    static getDerivedStateFromError(error: Error): State {
        return { hasError: true, error };
    }
    
    componentDidCatch(error: Error, errorInfo: React.ErrorInfo) {
        console.error('Error caught by boundary:', error, errorInfo);
    }
    
    render() {
        if (this.state.hasError) {
            return this.props.fallback || (
                <div className="p-8 text-center">
                    <h2 className="text-xl font-bold text-red-600">Something went wrong</h2>
                    <p className="text-gray-600 mt-2">
                        Please refresh the page to try again.
                    </p>
                    <button 
                        onClick={() => window.location.reload()}
                        className="btn btn-primary mt-4"
                    >
                        Refresh Page
                    </button>
                </div>
            );
        }
        return this.props.children;
    }
}

// Usage in App.tsx
function App() {
    return (
        <ErrorBoundary>
            <Router>
                {/* app content */}
            </Router>
        </ErrorBoundary>
    );
}
```

---

## Implementation Order

### Recommended Sequence

```
Phase 1 (Performance) - Start First
├── 1.3 Backend O(n²) Sorting - Immediate, high impact
├── 1.4 Unbounded Metrics History - After sorting
├── 1.1 Frontend useMemo - Parallel with 1.2
└── 1.2 Frontend useCallback - Parallel with 1.1

Phase 2 (Backend Reliability)
├── 2.1 Race Condition - CRITICAL, fix before deployment
├── 2.2 Graceful Shutdown - After race condition
├── 2.3 HTTP Response Body Close - Quick win
├── 2.4 Internal Error Exposure - Medium priority
└── 2.5 Silent Parse Failure - Low priority

Phase 3 (Frontend Polish)
├── 3.1 Console.log Removal - Quick win
├── 3.2 Type Safety - With 3.1
├── 3.3 Export Error Feedback - After types
├── 3.4 Accessibility - Ongoing
└── 3.5-3.6 - As time permits
```

### Parallelization Opportunities

| Tasks | Can Run In Parallel |
|-------|---------------------|
| 1.1 + 1.2 | Both React performance |
| 2.4 + 2.5 | Both error handling |
| 3.1 + 3.2 | Both code cleanup |
| 3.3 + 3.4 | UX improvements |

---

## Testing Requirements

### Unit Tests

- `handlers.go` - Sorting algorithm tests
- `detector.go` - ParseKey tests
- Frontend - useMemo/useCallback render tests

### Integration Tests

- Race condition verification (run with `-race` flag)
- Graceful shutdown verification

### Manual Testing

- Export error feedback
- Accessibility audit (axe-core or similar)

---

## Success Metrics

| Metric | Before | After Target |
|--------|--------|---------------|
| Sorting (10k items) | ~100M ops | <200k ops |
| Timeline render | ~500ms | <50ms |
| Memory (detector) | Unbounded | <10MB |
| Race conditions | Present | 0 |
| TypeScript any | 6 instances | 0 |
| console.log | 6 statements | 0 |

---

## Related Documentation

- `docs/requirement-application.md` - Feature requirements
- `docs/developer-team/SESSION_HANDOVER.md` - Session context
- `internal/correlation/detector.go` - Spike detection
- `ui/src/components/TimelineChart.tsx` - Timeline visualization