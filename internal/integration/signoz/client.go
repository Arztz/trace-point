package signoz

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/trace-point/trace-point/internal/config"
	"github.com/trace-point/trace-point/internal/utils/logger"
)

// Client handles communication with SigNoz/ClickHouse API
type Client struct {
	httpClient *http.Client
	baseURL    string
	queryPath  string
	timeout    time.Duration
	logger     *logger.Logger
	authToken  string
	projectID  string
	useGKE     bool
}

// NewClient creates a new SigNoz client
func NewClient(cfg *config.SignozConfig, gcloudCfg *config.GCloudConfig, logger *logger.Logger) *Client {
	transport := &http.Transport{
		MaxIdleConns:        10,
		MaxIdleConnsPerHost: 5,
		IdleConnTimeout:     30 * time.Second,
	}

	httpClient := &http.Client{
		Transport: transport,
		Timeout:   cfg.Timeout,
	}

	baseURL := cfg.URL
	if baseURL == "" {
		// Default to local SigNoz installation
		baseURL = "http://localhost:3301"
	}

	queryPath := cfg.QueryPath
	if queryPath == "" {
		queryPath = "/api/v1/traces"
	}

	client := &Client{
		httpClient: httpClient,
		baseURL:    baseURL,
		queryPath:  queryPath,
		timeout:    cfg.Timeout,
		logger:     logger,
	}

	// Configure GCloud authentication if enabled
	if gcloudCfg != nil && gcloudCfg.UseGKE {
		client.useGKE = true
		client.projectID = gcloudCfg.ProjectID
	}

	return client
}

// TraceResponse represents the trace query response from SigNoz
type TraceResponse struct {
	Traces []TraceData `json:"traces"`
}

// TraceData represents a single trace
type TraceData struct {
	TraceID   string     `json:"traceID"`
	Spans     []SpanData `json:"spans"`
	Timestamp int64      `json:"timestamp"`
	Duration  int64      `json:"duration"`
}

// SpanData represents a span in a trace
type SpanData struct {
	SpanID        string      `json:"spanID"`
	OperationName string      `json:"operationName"`
	ServiceName   string      `json:"serviceName"`
	StartTime     int64       `json:"startTime"`
	Duration      int64       `json:"duration"`
	Tags          []Tag       `json:"tags"`
	References    []Reference `json:"references"`
	Kind          string      `json:"kind"`
	StatusCode    int         `json:"statusCode"`
}

// Tag represents a span tag
type Tag struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

// Reference represents a trace reference
type Reference struct {
	TraceID string `json:"traceID"`
	SpanID  string `json:"spanID"`
	RefType string `json:"refType"`
}

// TraceQueryResult represents the parsed trace result
type TraceQueryResult struct {
	Traces      []ParsedTrace
	ServiceDeps []ServiceDependency
}

// ParsedTrace represents a parsed trace with route information
type ParsedTrace struct {
	TraceID     string
	ServiceName string
	Route       string
	Duration    int64
	StatusCode  int
	HasError    bool
	Spans       []ParsedSpan
}

// ParsedSpan represents a parsed span
type ParsedSpan struct {
	Operation string
	Service   string
	Duration  int64
	Kind      string
}

// ServiceDependency represents service dependency information
type ServiceDependency struct {
	SourceService string `json:"source"`
	TargetService string `json:"target"`
	CallCount     int64  `json:"call_count"`
}

// ClickHouseQuery represents a ClickHouse query request
type ClickHouseQuery struct {
	Start int64  `json:"start"`
	End   int64  `json:"end"`
	Query string `json:"query"`
	Step  string `json:"step"`
}

// MetricsResponse represents metrics query response
type MetricsResponse struct {
	ResultType string          `json:"resultType"`
	Results    []MetricsResult `json:"results"`
}

// MetricsResult represents metrics data
type MetricsResult struct {
	Metric map[string]string `json:"metric"`
	Values [][]interface{}   `json:"values"`
}

// SetAuthToken sets the authentication token
func (c *Client) SetAuthToken(token string) {
	c.authToken = token
}

// QueryTraces queries traces by namespace and pod selector within a time range
func (c *Client) QueryTraces(ctx context.Context, namespace, podName string, startTime, endTime time.Time) (*TraceQueryResult, error) {
	// Build ClickHouse query for traces
	query := c.buildTraceQuery(namespace, podName, startTime, endTime)

	c.logger.Debug("Querying SigNoz traces: namespace=%s, pod=%s, start=%s, end=%s",
		namespace, podName, startTime.Format(time.RFC3339), endTime.Format(time.RFC3339))

	// Execute query against ClickHouse
	result, err := c.executeClickHouseQuery(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query traces: %w", err)
	}

	// Parse response
	return c.parseTraceResponse(result)
}

// buildTraceQuery builds a ClickHouse query for traces
func (c *Client) buildTraceQuery(namespace, podName string, startTime, endTime time.Time) string {
	startTs := startTime.UnixMilli()
	endTs := endTime.UnixMilli()

	// Build WHERE clause for filtering
	var conditions []string

	if namespace != "" {
		conditions = append(conditions, fmt.Sprintf(`namespace = '%s'`, namespace))
	}
	if podName != "" {
		conditions = append(conditions, fmt.Sprintf(`JSONExtractString(attributes, 'k8s.pod.name') = '%s'`, podName))
	}

	whereClause := ""
	if len(conditions) > 0 {
		whereClause = " AND " + conditions[0]
		if len(conditions) > 1 {
			whereClause = " AND " + conditions[0]
			for i := 1; i < len(conditions); i++ {
				whereClause += " AND " + conditions[i]
			}
		}
	}

	// ClickHouse query to get traces
	query := fmt.Sprintf(`
		SELECT 
			traceID,
			spanName,
			serviceName,
			toUnixTimestamp(startTime) as startTime,
			durationNano as duration,
			JSONExtractString(attributes, 'http.url') as httpUrl,
			JSONExtractString(attributes, 'http.method') as httpMethod,
			statusCode,
			JSONExtractString(attributes, 'k8s.pod.name') as podName,
			JSONExtractString(attributes, 'k8s.namespace.name') as ns
		FROM signoz_traces.distributed_signoz_index_v2
		WHERE startTime >= %d AND startTime <= %d %s
		ORDER BY startTime DESC
		LIMIT 1000
	`, startTs, endTs, whereClause)

	return query
}

// executeClickHouseQuery executes a ClickHouse query via SigNoz API
func (c *Client) executeClickHouseQuery(ctx context.Context, query string) ([]byte, error) {
	// SigNoz provides a ClickHouse query API
	url := fmt.Sprintf("%s/api/v1/query", c.baseURL)

	// Build the request body
	reqBody := ClickHouseQuery{
		Start: time.Now().Add(-10 * time.Minute).UnixMilli(),
		End:   time.Now().UnixMilli(),
		Query: query,
		Step:  "5m",
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal query: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if c.authToken != "" {
		req.Header.Set("Authorization", "Bearer "+c.authToken)
	}

	// Add GKE authentication if enabled
	if c.useGKE {
		c.addGKEHeaders(req)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute query: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("SigNoz returned status %d: %s", resp.StatusCode, string(body))
	}

	return io.ReadAll(resp.Body)
}

// parseTraceResponse parses the ClickHouse response into trace data
func (c *Client) parseTraceResponse(data []byte) (*TraceQueryResult, error) {
	var response map[string]interface{}
	if err := json.Unmarshal(data, &response); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	result := &TraceQueryResult{
		Traces:      make([]ParsedTrace, 0),
		ServiceDeps: make([]ServiceDependency, 0),
	}

	// Parse the data based on SigNoz response format
	// This is a simplified implementation - actual parsing depends on SigNoz version
	c.logger.Debug("Parsed %d traces", len(result.Traces))

	return result, nil
}

// QueryTracesByTimeWindow queries traces in a time window around a spike
func (c *Client) QueryTracesByTimeWindow(ctx context.Context, namespace, podName string, spikeTime time.Time, windowMinutes int) (*TraceQueryResult, error) {
	startTime := spikeTime.Add(-time.Duration(windowMinutes) * time.Minute)
	endTime := spikeTime.Add(time.Duration(windowMinutes) * time.Minute)

	c.logger.Info("Querying traces for time window: %s to %s", startTime.Format(time.RFC3339), endTime.Format(time.RFC3339))

	return c.QueryTraces(ctx, namespace, podName, startTime, endTime)
}

// GetServiceDependencies gets service dependencies from traces
func (c *Client) GetServiceDependencies(ctx context.Context, namespace string, startTime, endTime time.Time) ([]ServiceDependency, error) {
	query := fmt.Sprintf(`
		SELECT 
			serviceName as source,
			JSONExtractString(attributes, 'db.instance') as target,
			count() as call_count
		FROM signoz_traces.distributed_signoz_index_v2
		WHERE startTime >= %d AND startTime <= %d
		GROUP BY serviceName, target
		ORDER BY call_count DESC
		LIMIT 50
	`, startTime.UnixMilli(), endTime.UnixMilli())

	result, err := c.executeClickHouseQuery(ctx, query)
	if err != nil {
		return nil, err
	}

	// Parse the response
	var response map[string]interface{}
	if err := json.Unmarshal(result, &response); err != nil {
		return nil, fmt.Errorf("failed to parse dependencies: %w", err)
	}

	deps := make([]ServiceDependency, 0)
	return deps, nil
}

// addGKEHeaders adds GKE authentication headers
func (c *Client) addGKEHeaders(req *http.Request) {
	// For GKE, we typically use workload identity
	// The actual implementation depends on the SigNoz deployment
	// This is a placeholder for GKE authentication
}

// Close closes the HTTP client
func (c *Client) Close() error {
	c.httpClient.CloseIdleConnections()
	return nil
}
