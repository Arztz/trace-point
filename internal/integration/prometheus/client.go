package prometheus

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/trace-point/trace-point/internal/config"
	"github.com/trace-point/trace-point/internal/utils/logger"
)

// ExtractReplicasetName extracts the replicaset name from a pod name.
// Example: "payment-service-7d9f8b5c6-abcde" -> "payment-service"
// Handles edge cases:
// - Pods without dash: "my-service" -> "my-service"
// - Single replica pods: "my-service-0" -> "my-service"
// - Hash-style pods: "payment-service-7d9f8b5c6-abcde" -> "payment-service"
func ExtractReplicasetName(podName string) string {
	// Find the last dash
	lastDash := strings.LastIndex(podName, "-")
	if lastDash == -1 {
		// No dash found - this is the replicaset name itself
		return podName
	}

	// Get the part after the last dash
	suffix := podName[lastDash+1:]

	// Check if the suffix looks like a hash (alphanumeric, 5+ chars)
	// Pod hashes in Kubernetes are typically 5+ characters
	if len(suffix) >= 5 {
		isHash := true
		for _, r := range suffix {
			if !unicode.IsLetter(r) && !unicode.IsDigit(r) {
				isHash = false
				break
			}
		}
		if isHash {
			// This is a hash suffix - remove it to get replicaset name
			return podName[:lastDash]
		}
	}

	// The suffix doesn't look like a hash, keep the original pod name
	return podName
}

// extractValueFromResult extracts a float64 value from a Prometheus result Value field.
// Handles different formats: []interface{} (instant query), string, float64
func extractValueFromResult(value interface{}) float64 {
	if value == nil {
		return 0
	}

	// Handle []interface{} format (instant query result: [timestamp, value])
	// NOTE: Query already returns percentage (includes * 100 in PromQL), no additional multiplication needed
	if vals, ok := value.([]interface{}); ok && len(vals) >= 2 {
		switch v := vals[1].(type) {
		case float64:
			return v // Already includes * 100 from Prometheus query
		case string:
			if parsed, err := strconv.ParseFloat(v, 64); err == nil {
				return parsed
			}
		}
	}

	// Handle direct float64
	// NOTE: Query already returns percentage (includes * 100 in PromQL), no additional multiplication needed
	if val, ok := value.(float64); ok {
		return val
	}

	// Handle string
	// NOTE: Query already returns percentage (includes * 100 in PromQL), no additional multiplication needed
	if val, ok := value.(string); ok {
		if parsed, err := strconv.ParseFloat(val, 64); err == nil {
			return parsed
		}
	}

	return 0
}

// Client handles communication with Prometheus
type Client struct {
	httpClient *http.Client
	baseURL    string
	timeout    time.Duration
	logger     *logger.Logger
}

// NewClient creates a new Prometheus client
func NewClient(cfg *config.PrometheusConfig, logger *logger.Logger) *Client {
	transport := &http.Transport{
		MaxIdleConns:        10,
		MaxIdleConnsPerHost: 5,
		IdleConnTimeout:     30 * time.Second,
	}

	httpClient := &http.Client{
		Transport: transport,
		Timeout:   cfg.Timeout,
	}

	return &Client{
		httpClient: httpClient,
		baseURL:    cfg.URL,
		timeout:    cfg.Timeout,
		logger:     logger,
	}
}

// MetricQueryResult represents the result of a Prometheus query
type MetricQueryResult struct {
	Status string `json:"status"`
	Data   Data   `json:"data"`
}

// Data contains the query result data
type Data struct {
	ResultType string   `json:"resultType"`
	Result     []Result `json:"result"`
}

// Result represents a single metric result (for instant queries)
type Result struct {
	Metric map[string]string `json:"metric"`
	Value  interface{}       `json:"value"` // Can be []interface{} (instant) or [][]interface{} (range)
}

// RangeResult represents a metric result for range queries (uses "values" plural)
type RangeResult struct {
	Metric map[string]string `json:"metric"`
	Values [][]interface{}   `json:"values"`
}

// RangeQueryResult represents the result of a Prometheus range query
type RangeQueryResult struct {
	Status string    `json:"status"`
	Data   RangeData `json:"data"`
}

// RangeData contains the range query result data
type RangeData struct {
	ResultType string        `json:"resultType"`
	Result     []RangeResult `json:"result"`
}

// Query executes a Prometheus instant query and returns results
func (c *Client) Query(ctx context.Context, query string, queryTime time.Time) ([]Result, error) {
	url := fmt.Sprintf("%s/api/v1/query", c.baseURL)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	q := req.URL.Query()
	q.Add("query", query)
	if !queryTime.IsZero() {
		q.Add("time", fmt.Sprintf("%d", queryTime.Unix()))
	}
	req.URL.RawQuery = q.Encode()

	c.logger.Debug("Executing Prometheus query: %s", query)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute query: %w", err)
	}
	// Ensure body is closed after all checks
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		// Body will still be closed by defer
		return nil, fmt.Errorf("Prometheus returned status %d", resp.StatusCode)
	}

	var result MetricQueryResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if result.Status != "success" {
		return nil, fmt.Errorf("Prometheus query failed: %s", result.Status)
	}

	return result.Data.Result, nil
}

// QueryRange executes a range query against Prometheus
func (c *Client) QueryRange(ctx context.Context, query string, start, end time.Time, step time.Duration) ([]Result, error) {
	url := fmt.Sprintf("%s/api/v1/query_range", c.baseURL)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	q := req.URL.Query()
	q.Add("query", query)
	q.Add("start", fmt.Sprintf("%d", start.Unix()))
	q.Add("end", fmt.Sprintf("%d", end.Unix()))
	q.Add("step", step.String())
	req.URL.RawQuery = q.Encode()

	c.logger.Debug("Executing Prometheus range query: %s", query)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute range query: %w", err)
	}
	// Ensure body is closed after all checks
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		// Body will still be closed by defer
		return nil, fmt.Errorf("Prometheus returned status %d", resp.StatusCode)
	}

	// Use RangeQueryResult which has Values (plural) instead of Value
	var result RangeQueryResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if result.Status != "success" {
		return nil, fmt.Errorf("Prometheus range query failed: %s", result.Status)
	}

	// Convert RangeResult to Result with Values stored in Value field
	// This makes it compatible with existing code that expects Value
	results := make([]Result, len(result.Data.Result))
	for i, rr := range result.Data.Result {
		results[i] = Result{
			Metric: rr.Metric,
			Value:  rr.Values, // Store Values as the Value for processing
		}
	}

	return results, nil
}

// ContainerMetrics represents CPU and RAM metrics for a container
type ContainerMetrics struct {
	PodName       string
	Namespace     string
	ContainerName string
	CPUPercent    float64
	RAMPercent    float64
	Timestamp     time.Time
}

// FetchContainerMetrics fetches CPU and RAM metrics for all containers
func (c *Client) FetchContainerMetrics(ctx context.Context, namespaces []string, excludePatterns []string) ([]ContainerMetrics, error) {
	// Query CPU utilization
	cpuQuery := BuildCPUUtilizationQuery(namespaces, excludePatterns)
	cpuResults, err := c.Query(ctx, cpuQuery, time.Now())
	if err != nil {
		c.logger.Error("Failed to query CPU metrics: %v", err)
		return nil, fmt.Errorf("CPU query failed: %w", err)
	}

	// Query RAM utilization
	ramQuery := BuildRAMUtilizationQuery(namespaces, excludePatterns)
	ramResults, err := c.Query(ctx, ramQuery, time.Now())
	if err != nil {
		c.logger.Error("Failed to query RAM metrics: %v", err)
		return nil, fmt.Errorf("RAM query failed: %w", err)
	}

	// Build metric map by container
	metricsMap := make(map[string]*ContainerMetrics)

	// Process CPU results
	for _, r := range cpuResults {
		podName := r.Metric["pod"]
		namespace := r.Metric["namespace"]
		containerName := r.Metric["container"]
		key := fmt.Sprintf("%s/%s/%s", namespace, podName, containerName)

		var cpuPercent float64
		// Handle both instant ([]interface{}) and range ([][]interface{}) formats
		// Note: Query already returns percentage (includes * 100), so no additional multiplication
		if vals, ok := r.Value.([]interface{}); ok && len(vals) >= 2 {
			if val, ok := vals[1].(float64); ok {
				cpuPercent = val // Already includes * 100 from Prometheus query
			}
		}

		metricsMap[key] = &ContainerMetrics{
			PodName:       podName,
			Namespace:     namespace,
			ContainerName: containerName,
			CPUPercent:    cpuPercent,
			Timestamp:     time.Now(),
		}
	}

	// Process RAM results
	for _, r := range ramResults {
		podName := r.Metric["pod"]
		namespace := r.Metric["namespace"]
		containerName := r.Metric["container"]
		key := fmt.Sprintf("%s/%s/%s", namespace, podName, containerName)

		var ramPercent float64
		// Handle both instant ([]interface{}) and range ([][]interface{}) formats
		// Note: Query already returns percentage (includes * 100), so no additional multiplication
		if vals, ok := r.Value.([]interface{}); ok && len(vals) >= 2 {
			if val, ok := vals[1].(float64); ok {
				ramPercent = val // Already includes * 100 from Prometheus query
			}
		}

		if m, exists := metricsMap[key]; exists {
			m.RAMPercent = ramPercent
		}
	}

	// Convert map to slice
	metrics := make([]ContainerMetrics, 0, len(metricsMap))
	for _, m := range metricsMap {
		metrics = append(metrics, *m)
	}

	return metrics, nil
}

// GetHistoricalMetrics fetches historical metrics for a specific container
func (c *Client) GetHistoricalMetrics(ctx context.Context, podName, namespace, containerName string, duration time.Duration) ([]ContainerMetrics, error) {
	end := time.Now()
	start := end.Add(-duration)

	// Query CPU history
	cpuQuery := BuildContainerCPUQuery(podName, namespace, containerName)
	cpuResults, err := c.QueryRange(ctx, cpuQuery, start, end, 30*time.Second)
	if err != nil {
		c.logger.Error("Failed to query CPU history: %v", err)
		return nil, err
	}

	// Query RAM history
	ramQuery := BuildContainerRAMQuery(podName, namespace, containerName)
	ramResults, err := c.QueryRange(ctx, ramQuery, start, end, 30*time.Second)
	if err != nil {
		c.logger.Error("Failed to query RAM history: %v", err)
		return nil, err
	}

	// Combine results
	metrics := make([]ContainerMetrics, 0)
	cpuMap := make(map[float64]float64)
	ramMap := make(map[float64]float64)

	for _, r := range cpuResults {
		// Handle range query result ([][]interface{})
		// NOTE: Query already returns percentage (includes * 100 in PromQL), no additional multiplication needed
		if values, ok := r.Value.([][]interface{}); ok {
			for i := range values {
				if len(values[i]) >= 2 {
					point := values[i]
					if ts, ok := point[0].(float64); ok {
						if val, ok := point[1].(float64); ok {
							cpuMap[ts] = val // Already includes * 100 from Prometheus query
						}
					}
				}
			}
		}
	}

	for _, r := range ramResults {
		// Handle range query result ([][]interface{})
		// NOTE: Query already returns percentage (includes * 100 in PromQL), no additional multiplication needed
		if values, ok := r.Value.([][]interface{}); ok {
			for i := range values {
				if len(values[i]) >= 2 {
					point := values[i]
					if ts, ok := point[0].(float64); ok {
						if val, ok := point[1].(float64); ok {
							ramMap[ts] = val // Already includes * 100 from Prometheus query
						}
					}
				}
			}
		}
	}

	// Merge timestamps
	for ts, cpu := range cpuMap {
		ram, _ := ramMap[ts]
		metrics = append(metrics, ContainerMetrics{
			PodName:       podName,
			Namespace:     namespace,
			ContainerName: containerName,
			CPUPercent:    cpu,
			RAMPercent:    ram,
			Timestamp:     time.Unix(int64(ts), 0),
		})
	}

	return metrics, nil
}

// Close closes the HTTP client
func (c *Client) Close() error {
	c.httpClient.CloseIdleConnections()
	return nil
}

// TimelineMetric represents continuous CPU/RAM metrics for timeline visualization.
// Grouped by replicaset for aggregated view.
type TimelineMetric struct {
	Timestamp      time.Time `json:"timestamp"`
	ReplicasetName string    `json:"replicaset_name"` // Grouped replicaset name (e.g., "payment-service")
	PodName        string    `json:"pod_name"`        // Original pod name (kept for reference)
	Namespace      string    `json:"namespace"`
	CPUPercent     float64   `json:"cpu_percent"`
	RAMPercent     float64   `json:"ram_percent"`
}

// QueryTimelineMetrics queries CPU and RAM metrics for timeline visualization.
// Returns metrics for all containers within the specified time range.
// Supports optional podNameFilter (comma-separated replicaset names) for filtering.
// When filter is provided, matches against replicaset name and aggregates metrics.
func (c *Client) QueryTimelineMetrics(ctx context.Context, namespaces []string, startTime, endTime time.Time, step time.Duration, podNameFilter string) ([]TimelineMetric, error) {
	if startTime.IsZero() || endTime.IsZero() {
		return nil, fmt.Errorf("startTime and endTime are required")
	}

	if endTime.Before(startTime) {
		return nil, fmt.Errorf("endTime must be after startTime")
	}

	// If no step is specified, default to 30 seconds
	if step == 0 {
		step = 30 * time.Second
	}

	// Parse pod filter for inclusion logic - now filters by replicaset name
	var replicasetFilterMap map[string]bool
	if podNameFilter != "" {
		replicasetFilterMap = make(map[string]bool)
		for _, pod := range strings.Split(podNameFilter, ",") {
			trimmed := strings.TrimSpace(pod)
			if trimmed != "" {
				replicasetFilterMap[trimmed] = true
			}
		}
		c.logger.Debug("Filtering to replicasets: %v", replicasetFilterMap)
	}

	// Build CPU query
	cpuQuery := BuildCPUUtilizationQuery(namespaces, nil)
	c.logger.Debug("Querying CPU timeline metrics: %s", cpuQuery)

	// Query CPU range
	cpuResults, err := c.QueryRange(ctx, cpuQuery, startTime, endTime, step)
	if err != nil {
		c.logger.Error("Failed to query CPU timeline: %v", err)
		// Return empty but don't fail - graceful degradation
		cpuResults = []Result{}
	} else {
		c.logger.Debug("CPU range query returned %d results", len(cpuResults))
	}

	// Build RAM query
	ramQuery := BuildRAMUtilizationQuery(namespaces, nil)
	c.logger.Debug("Querying RAM timeline metrics: %s", ramQuery)

	// Query RAM range
	ramResults, err := c.QueryRange(ctx, ramQuery, startTime, endTime, step)
	if err != nil {
		c.logger.Error("Failed to query RAM timeline: %v", err)
		// Return empty but don't fail - graceful degradation
		ramResults = []Result{}
	}

	// Build metric map keyed by timestamp and container
	// Structure: map[timestamp][containerKey] -> cpuPercent
	cpuMap := make(map[string]map[string]float64)
	ramMap := make(map[string]map[string]float64)

	// Process CPU results - each result has Values array for range query
	for _, r := range cpuResults {
		podName := r.Metric["pod"]
		namespace := r.Metric["namespace"]
		containerName := r.Metric["container"]
		containerKey := fmt.Sprintf("%s/%s/%s", namespace, podName, containerName)

		// Get values from Result.Value - for range queries this contains [][]interface{}
		// Note: In QueryRange, we store [][]interface{} directly, not []interface{}
		values, ok := r.Value.([][]interface{})
		if !ok {
			c.logger.Warn("CPU result: cannot cast Value to [][]interface{} for %s", containerKey)
			continue
		}

		c.logger.Debug("CPU result for %s: %d data points", containerKey, len(values))

		// For range queries, Values is [][]interface{} where each element is [timestamp, value]
		// Access by index since val is []interface{}, not interface{}
		for i := range values {
			if len(values[i]) < 2 {
				continue
			}
			point := values[i]

			// Parse timestamp
			var ts float64
			switch v := point[0].(type) {
			case float64:
				ts = v
			case string:
				parsed, err := strconv.ParseFloat(v, 64)
				if err != nil {
					continue
				}
				ts = parsed
			default:
				continue
			}

			// Parse value
			var cpuPercent float64
			switch v := point[1].(type) {
			case float64:
				cpuPercent = v // Already includes * 100 from Prometheus query
			case string:
				parsed, err := strconv.ParseFloat(v, 64)
				if err != nil {
					continue
				}
				cpuPercent = parsed
			default:
				continue
			}

			// Handle infinity (division by zero) - skip or set to 0
			if math.IsInf(cpuPercent, 0) || math.IsNaN(cpuPercent) {
				continue
			}

			timestamp := time.Unix(int64(ts), 0).Format(time.RFC3339)

			if cpuMap[timestamp] == nil {
				cpuMap[timestamp] = make(map[string]float64)
			}
			cpuMap[timestamp][containerKey] = cpuPercent
		}
	}

	// Process RAM results
	for _, r := range ramResults {
		podName := r.Metric["pod"]
		namespace := r.Metric["namespace"]
		containerName := r.Metric["container"]
		containerKey := fmt.Sprintf("%s/%s/%s", namespace, podName, containerName)

		// Get values from Result.Value - for range queries this contains [][]interface{}
		values, ok := r.Value.([][]interface{})
		if !ok {
			continue
		}

		for i := range values {
			if len(values[i]) < 2 {
				continue
			}
			point := values[i]

			// Parse timestamp
			var ts float64
			switch v := point[0].(type) {
			case float64:
				ts = v
			case string:
				parsed, err := strconv.ParseFloat(v, 64)
				if err != nil {
					continue
				}
				ts = parsed
			default:
				continue
			}

			// Parse value
			var ramPercent float64
			switch v := point[1].(type) {
			case float64:
				ramPercent = v // Already includes * 100 from Prometheus query
			case string:
				parsed, err := strconv.ParseFloat(v, 64)
				if err != nil {
					continue
				}
				ramPercent = parsed
			default:
				continue
			}

			// Handle infinity (division by zero) - skip or set to 0
			if math.IsInf(ramPercent, 0) || math.IsNaN(ramPercent) {
				continue
			}

			timestamp := time.Unix(int64(ts), 0).Format(time.RFC3339)

			if ramMap[timestamp] == nil {
				ramMap[timestamp] = make(map[string]float64)
			}
			ramMap[timestamp][containerKey] = ramPercent
		}
	}

	// Merge CPU and RAM into unified timeline metrics
	var timelineMetrics []TimelineMetric

	// Get all unique timestamps
	timestamps := make([]string, 0)
	for ts := range cpuMap {
		timestamps = append(timestamps, ts)
	}
	for ts := range ramMap {
		if _, exists := cpuMap[ts]; !exists {
			timestamps = append(timestamps, ts)
		}
	}

	// Sort timestamps
	for i := 0; i < len(timestamps)-1; i++ {
		for j := i + 1; j < len(timestamps); j++ {
			if timestamps[j] < timestamps[i] {
				timestamps[i], timestamps[j] = timestamps[j], timestamps[i]
			}
		}
	}

	// Build metrics for each timestamp
	for _, timestampStr := range timestamps {
		// Get all containers at this timestamp
		allContainers := make(map[string]bool)

		if cpuMap[timestampStr] != nil {
			for containerKey := range cpuMap[timestampStr] {
				allContainers[containerKey] = true
			}
		}
		if ramMap[timestampStr] != nil {
			for containerKey := range ramMap[timestampStr] {
				allContainers[containerKey] = true
			}
		}

		// Create metric entry for each container
		for containerKey := range allContainers {
			parts := strings.Split(containerKey, "/")
			if len(parts) != 3 {
				continue
			}

			namespace := parts[0]
			podName := parts[1]

			// Extract replicaset name from pod name
			replicasetName := ExtractReplicasetName(podName)

			// Apply replicaset filter if specified
			if replicasetFilterMap != nil {
				if !replicasetFilterMap[replicasetName] {
					continue
				}
			}

			timestamp, err := time.Parse(time.RFC3339, timestampStr)
			if err != nil {
				continue
			}

			cpu := 0.0
			if cpuMap[timestampStr] != nil {
				if val, exists := cpuMap[timestampStr][containerKey]; exists {
					cpu = val
				}
			}

			ram := 0.0
			if ramMap[timestampStr] != nil {
				if val, exists := ramMap[timestampStr][containerKey]; exists {
					ram = val
				}
			}

			timelineMetrics = append(timelineMetrics, TimelineMetric{
				Timestamp:      timestamp,
				ReplicasetName: replicasetName,
				PodName:        podName,
				Namespace:      namespace,
				CPUPercent:     cpu,
				RAMPercent:     ram,
			})
		}
	}

	c.logger.Debug("Retrieved %d timeline metrics points", len(timelineMetrics))

	return timelineMetrics, nil
}

// AvailablePod represents a replicaset with its aggregated metrics for the dropdown in frontend
type AvailablePod struct {
	Name       string  `json:"name"`        // Replicaset name (grouped)
	Namespace  string  `json:"namespace"`   // Namespace
	PodCount   int     `json:"pod_count"`   // Number of pods in this replicaset
	CurrentCPU float64 `json:"current_cpu"` // Aggregated CPU (average across all pods)
	CurrentRAM float64 `json:"current_ram"` // Aggregated RAM (average across all pods)
}

// GetAvailablePods fetches all currently running replicasets with their aggregated metrics.
// This is used for populating the frontend dropdown.
// Groups pods by replicaset (removing the trailing "-xxxx" hash) and aggregates CPU/RAM.
func (c *Client) GetAvailablePods(ctx context.Context, namespaces []string) ([]AvailablePod, error) {
	// Query CPU utilization
	cpuQuery := BuildCPUUtilizationQuery(namespaces, nil)
	cpuResults, err := c.Query(ctx, cpuQuery, time.Now())
	if err != nil {
		c.logger.Error("Failed to query CPU metrics for available pods: %v", err)
		return nil, err
	}

	// Query RAM utilization
	ramQuery := BuildRAMUtilizationQuery(namespaces, nil)
	ramResults, err := c.Query(ctx, ramQuery, time.Now())
	if err != nil {
		c.logger.Error("Failed to query RAM metrics for available pods: %v", err)
		return nil, err
	}

	// Build metric map keyed by replicaset (not individual pod)
	// Key: namespace/replicaset
	replicasetMetrics := make(map[string]*AvailablePod)

	// Process CPU results
	for _, r := range cpuResults {
		podName := r.Metric["pod"]
		namespace := r.Metric["namespace"]
		replicasetName := ExtractReplicasetName(podName)
		key := fmt.Sprintf("%s/%s", namespace, replicasetName)

		cpuPercent := extractValueFromResult(r.Value)

		if m, exists := replicasetMetrics[key]; exists {
			m.PodCount++
			m.CurrentCPU += cpuPercent
		} else {
			replicasetMetrics[key] = &AvailablePod{
				Name:       replicasetName,
				Namespace:  namespace,
				PodCount:   1,
				CurrentCPU: cpuPercent,
			}
		}
	}

	// Process RAM results
	for _, r := range ramResults {
		podName := r.Metric["pod"]
		namespace := r.Metric["namespace"]
		replicasetName := ExtractReplicasetName(podName)
		key := fmt.Sprintf("%s/%s", namespace, replicasetName)

		ramPercent := extractValueFromResult(r.Value)

		if m, exists := replicasetMetrics[key]; exists {
			m.CurrentRAM += ramPercent
		} else {
			// Also create entry if it doesn't exist (for RAM-only pods)
			replicasetMetrics[key] = &AvailablePod{
				Name:       replicasetName,
				Namespace:  namespace,
				PodCount:   0, // Will be counted in CPU processing
				CurrentRAM: ramPercent,
			}
		}
	}

	// Calculate average CPU/RAM for each replicaset
	pods := make([]AvailablePod, 0, len(replicasetMetrics))
	for _, m := range replicasetMetrics {
		if m.PodCount > 0 {
			m.CurrentCPU = m.CurrentCPU / float64(m.PodCount)
			m.CurrentRAM = m.CurrentRAM / float64(m.PodCount)
		}
		pods = append(pods, *m)
	}

	c.logger.Debug("Found %d available replicasets", len(pods))

	return pods, nil
}
