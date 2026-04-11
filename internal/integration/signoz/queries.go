package signoz

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/trace-point/trace-point/internal/utils/logger"
)

// Queries provides query builders for SigNoz trace data

// QueryConfig holds configuration for trace queries
type QueryConfig struct {
	WindowMinutes  int
	MaxTraces      int
	IncludeSpans   bool
	IncludeMetrics bool
}

// DefaultQueryConfig returns default query configuration
func DefaultQueryConfig() *QueryConfig {
	return &QueryConfig{
		WindowMinutes:  5,
		MaxTraces:      100,
		IncludeSpans:   true,
		IncludeMetrics: true,
	}
}

// RouteExtractor extracts route information from trace spans
type RouteExtractor struct {
	logger *logger.Logger
}

// NewRouteExtractor creates a new route extractor
func NewRouteExtractor(logger *logger.Logger) *RouteExtractor {
	return &RouteExtractor{logger: logger}
}

// ExtractRoute extracts the route from trace spans
// It looks for HTTP spans with url/path tags
func (re *RouteExtractor) ExtractRoute(spans []ParsedSpan) string {
	for _, span := range spans {
		// Check if it's an HTTP span
		if span.Kind == "server" || span.Kind == "client" {
			// Try to extract route from operation name
			route := re.extractFromOperation(span.Operation)
			if route != "" {
				return route
			}
		}
	}
	return ""
}

// extractFromOperation extracts route from operation name
func (re *RouteExtractor) extractFromOperation(operation string) string {
	// Common patterns for operation names:
	// - HTTP method + path: GET /api/users
	// - gRPC method: /package.Service/Method
	// - Database query: SELECT * FROM...

	// If operation looks like HTTP (starts with /)
	if strings.HasPrefix(operation, "/") {
		// It's likely a route path
		return operation
	}

	// Check for common prefixes
	for _, prefix := range []string{"HTTP ", "GRPC ", "HTTP GET ", "HTTP POST ", "HTTP PUT ", "HTTP DELETE "} {
		if strings.HasPrefix(operation, prefix) {
			return strings.TrimPrefix(operation, prefix)
		}
	}

	return ""
}

// TagAsJobRoute checks if a route matches job patterns and tags it
// Patterns: /tasks/*, /batch/*, /jobs/*
func (re *RouteExtractor) TagAsJobRoute(route string) string {
	jobPatterns := []string{"/tasks/", "/batch/", "/jobs/"}

	for _, pattern := range jobPatterns {
		if strings.HasPrefix(route, pattern) {
			return "[Suspected Job] " + route
		}
	}

	return route
}

// TraceAnalyzer analyzes traces to find patterns and correlations
type TraceAnalyzer struct {
	logger *logger.Logger
}

// NewTraceAnalyzer creates a new trace analyzer
func NewTraceAnalyzer(logger *logger.Logger) *TraceAnalyzer {
	return &TraceAnalyzer{logger: logger}
}

// AnalyzeRouteActivity analyzes which routes were active during a spike
// Returns routes sorted by their resource consumption (CPU + RAM estimate)
func (ta *TraceAnalyzer) AnalyzeRouteActivity(traces []ParsedTrace, cpuPercent, ramPercent float64) []RouteActivity {
	routeMap := make(map[string]*RouteActivity)

	for _, trace := range traces {
		route := trace.Route
		if route == "" {
			route = "unknown"
		}

		activity, exists := routeMap[route]
		if !exists {
			activity = &RouteActivity{
				Route:          route,
				TraceCount:     0,
				TotalDuration:  0,
				ErrorCount:     0,
				ResourceWeight: 0,
			}
			routeMap[route] = activity
		}

		activity.TraceCount++
		activity.TotalDuration += trace.Duration

		if trace.HasError {
			activity.ErrorCount++
		}

		// Calculate resource weight based on trace duration and status
		// Longer traces with errors get higher weight
		weight := float64(trace.Duration) / 1000000 // Convert nanoseconds to ms
		if trace.HasError {
			weight *= 1.5
		}
		activity.ResourceWeight += weight
	}

	// Convert map to slice and calculate overall resource weight with pod metrics
	activities := make([]RouteActivity, 0, len(routeMap))
	for _, activity := range routeMap {
		// Combine trace weight with pod's actual resource consumption
		activity.ResourceWeight = (activity.ResourceWeight / float64(activity.TraceCount)) * (cpuPercent + ramPercent) / 100
		activities = append(activities, *activity)
	}

	// Sort by resource weight (highest first)
	for i := 0; i < len(activities)-1; i++ {
		for j := i + 1; j < len(activities); j++ {
			if activities[j].ResourceWeight > activities[i].ResourceWeight {
				activities[i], activities[j] = activities[j], activities[i]
			}
		}
	}

	return activities
}

// RouteActivity represents activity for a specific route
type RouteActivity struct {
	Route          string  `json:"route"`
	TraceCount     int     `json:"trace_count"`
	TotalDuration  int64   `json:"total_duration_ms"`
	ErrorCount     int     `json:"error_count"`
	ResourceWeight float64 `json:"resource_weight"`
}

// SelectCulpritRoute selects the most likely culprit route when multiple routes are active
// Criteria: highest combined CPU + RAM consumption
func (ta *TraceAnalyzer) SelectCulpritRoute(activities []RouteActivity, cpuPercent, ramPercent float64) *RouteActivity {
	if len(activities) == 0 {
		return nil
	}

	if len(activities) == 1 {
		return &activities[0]
	}

	// Calculate combined resource usage for each route
	// Weight the route's activity by the pod's actual resource consumption
	combinedResource := cpuPercent + ramPercent

	var culprit *RouteActivity
	var highestScore float64

	for i := range activities {
		// Score = route's own resource weight * pod's overall resource usage
		score := activities[i].ResourceWeight * (combinedResource / 100)

		if culprit == nil || score > highestScore {
			highestScore = score
			culprit = &activities[i]
		}
	}

	ta.logger.Info("Selected culprit route: %s (score: %.2f, CPU: %.2f%%, RAM: %.2f%%)",
		culprit.Route, highestScore, cpuPercent, ramPercent)

	return culprit
}

// BuildTraceQuery builds a ClickHouse query for trace data
func BuildTraceQuery(namespace, podName string, startTime, endTime time.Time, limit int) string {
	startTs := startTime.UnixMilli()
	endTs := endTime.UnixMilli()

	query := fmt.Sprintf(`
		SELECT 
			traceID,
			spanName,
			serviceName,
			toUnixTimestamp(startTime/1000000000) as startTime,
			durationNano as duration,
			JSONExtractString(attributes, 'http.url') as httpUrl,
			JSONExtractString(attributes, 'http.method') as httpMethod,
			JSONExtractInt(attributes, 'http.status_code') as statusCode,
			JSONExtractString(attributes, 'k8s.pod.name') as podName,
			JSONExtractString(attributes, 'k8s.namespace.name') as namespace,
			JSONExtractString(attributes, 'span.kind') as kind
		FROM signoz_traces.distributed_signoz_index_v2
		WHERE startTime >= %d AND startTime <= %d
	`, startTs, endTs)

	if namespace != "" {
		query += fmt.Sprintf(` AND JSONExtractString(attributes, 'k8s.namespace.name') = '%s'`, namespace)
	}

	if podName != "" {
		query += fmt.Sprintf(` AND JSONExtractString(attributes, 'k8s.pod.name') = '%s'`, podName)
	}

	query += fmt.Sprintf(`
		ORDER BY startTime DESC
		LIMIT %d
	`, limit)

	return query
}

// BuildSpanQuery builds a ClickHouse query for span data
func BuildSpanQuery(traceID string) string {
	return fmt.Sprintf(`
		SELECT 
			spanID,
			traceID,
			spanName,
			serviceName,
			startTime,
			durationNano,
			JSONExtractString(attributes, 'span.kind') as kind,
			JSONExtractString(attributes, 'http.url') as httpUrl,
			JSONExtractString(attributes, 'http.method') as httpMethod,
			JSONExtractInt(attributes, 'http.status_code') as statusCode
		FROM signoz_traces.distributed_signoz_index_v2
		WHERE traceID = '%s'
		ORDER BY startTime ASC
	`, traceID)
}

// BuildServiceDependencyQuery builds a ClickHouse query for service dependencies
func BuildServiceDependencyQuery(namespace string, startTime, endTime time.Time) string {
	startTs := startTime.UnixMilli()
	endTs := endTime.UnixMilli()

	query := fmt.Sprintf(`
		SELECT 
			serviceName as source_service,
			JSONExtractString(attributes, 'db.system') as target_service,
			count() as call_count,
			quantile(0.95)(durationNano/1000000) as p95_latency_ms
		FROM signoz_traces.distributed_signoz_index_v2
		WHERE startTime >= %d AND startTime <= %d
	`, startTs, endTs)

	if namespace != "" {
		query += fmt.Sprintf(` AND JSONExtractString(attributes, 'k8s.namespace.name') = '%s'`, namespace)
	}

	query += `
		GROUP BY serviceName, target_service
		ORDER BY call_count DESC
		LIMIT 50
	`

	return query
}

// BuildMetricsQuery builds a ClickHouse query for trace metrics
func BuildMetricsQuery(namespace, podName string, startTime, endTime time.Time) string {
	startTs := startTime.UnixMilli()
	endTs := endTime.UnixMilli()

	query := fmt.Sprintf(`
		SELECT 
			serviceName,
			count() as request_count,
			quantile(0.50)(durationNano/1000000) as latency_p50_ms,
			quantile(0.95)(durationNano/1000000) as latency_p95_ms,
			quantile(0.99)(durationNano/1000000) as latency_p99_ms,
			sum(if(statusCode >= 500, 1, 0)) as error_count,
			sum(if(statusCode >= 500, 1, 0)) * 100.0 / count() as error_rate_percent
		FROM signoz_traces.distributed_signoz_index_v2
		WHERE startTime >= %d AND startTime <= %d
	`, startTs, endTs)

	if namespace != "" {
		query += fmt.Sprintf(` AND JSONExtractString(attributes, 'k8s.namespace.name') = '%s'`, namespace)
	}

	if podName != "" {
		query += fmt.Sprintf(` AND JSONExtractString(attributes, 'k8s.pod.name') = '%s'`, podName)
	}

	query += `
		GROUP BY serviceName
		ORDER BY request_count DESC
	`

	return query
}

// CorrelationService provides correlation between spikes and trace data
type CorrelationService struct {
	client         *Client
	routeExtractor *RouteExtractor
	analyzer       *TraceAnalyzer
	logger         *logger.Logger
}

// NewCorrelationService creates a new correlation service
func NewCorrelationService(client *Client, logger *logger.Logger) *CorrelationService {
	return &CorrelationService{
		client:         client,
		routeExtractor: NewRouteExtractor(logger),
		analyzer:       NewTraceAnalyzer(logger),
		logger:         logger,
	}
}

// CorrelateSpikeWithTraces correlates a spike event with trace data
// Returns the most likely culprit route and trace ID
func (cs *CorrelationService) CorrelateSpikeWithTraces(
	ctx context.Context,
	namespace, podName string,
	spikeTime time.Time,
	cpuPercent, ramPercent float64,
	windowMinutes int,
) (*CorrelationResult, error) {

	if windowMinutes == 0 {
		windowMinutes = 5 // Default 5 minutes before/after spike
	}

	// Query traces in the time window around the spike
	traces, err := cs.client.QueryTracesByTimeWindow(ctx, namespace, podName, spikeTime, windowMinutes)
	if err != nil {
		return nil, fmt.Errorf("failed to query traces: %w", err)
	}

	if len(traces.Traces) == 0 {
		cs.logger.Info("No traces found for spike at %s in namespace %s", spikeTime.Format(time.RFC3339), namespace)
		return &CorrelationResult{
			FoundTraces: false,
		}, nil
	}

	cs.logger.Info("Found %d traces for spike at %s", len(traces.Traces), spikeTime.Format(time.RFC3339))

	// Extract routes from traces
	for i := range traces.Traces {
		route := cs.routeExtractor.ExtractRoute(traces.Traces[i].Spans)
		traces.Traces[i].Route = route
	}

	// Analyze route activity
	activities := cs.analyzer.AnalyzeRouteActivity(traces.Traces, cpuPercent, ramPercent)

	// Select culprit route
	culprit := cs.analyzer.SelectCulpritRoute(activities, cpuPercent, ramPercent)

	result := &CorrelationResult{
		FoundTraces:     true,
		TraceCount:      len(traces.Traces),
		RouteActivities: activities,
	}

	if culprit != nil {
		result.CulpritRoute = culprit.Route
		result.CulpritScore = culprit.ResourceWeight

		// Tag job routes
		taggedRoute := cs.routeExtractor.TagAsJobRoute(culprit.Route)
		result.CulpritRoute = taggedRoute
	}

	// Get trace ID from first trace (or culprit trace)
	if len(traces.Traces) > 0 {
		result.TraceID = traces.Traces[0].TraceID
	}

	return result, nil
}

// CorrelationResult holds correlation analysis results
type CorrelationResult struct {
	FoundTraces     bool            `json:"found_traces"`
	TraceCount      int             `json:"trace_count"`
	RouteActivities []RouteActivity `json:"route_activities"`
	CulpritRoute    string          `json:"culprit_route"`
	CulpritScore    float64         `json:"culprit_score"`
	TraceID         string          `json:"trace_id"`
}
