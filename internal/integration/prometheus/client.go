package prometheus

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/trace-point/trace-point/internal/config"
	"github.com/trace-point/trace-point/internal/utils/logger"
)

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

// Result represents a single metric result
type Result struct {
	Metric map[string]string `json:"metric"`
	Value  []interface{}     `json:"value"`
}

// Query executes a Prometheus query and returns results
func (c *Client) Query(ctx context.Context, query string, time time.Time) ([]Result, error) {
	url := fmt.Sprintf("%s/api/v1/query", c.baseURL)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	q := req.URL.Query()
	q.Add("query", query)
	if !time.IsZero() {
		q.Add("time", fmt.Sprintf("%d", time.Unix()))
	}
	req.URL.RawQuery = q.Encode()

	c.logger.Debug("Executing Prometheus query: %s", query)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute query: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
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
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Prometheus returned status %d", resp.StatusCode)
	}

	var result MetricQueryResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if result.Status != "success" {
		return nil, fmt.Errorf("Prometheus range query failed: %s", result.Status)
	}

	return result.Data.Result, nil
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
		if len(r.Value) >= 2 {
			if val, ok := r.Value[1].(float64); ok {
				cpuPercent = val * 100 // Convert to percentage
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
		if len(r.Value) >= 2 {
			if val, ok := r.Value[1].(float64); ok {
				ramPercent = val * 100 // Convert to percentage
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
		if len(r.Value) >= 2 {
			if ts, ok := r.Value[0].(float64); ok {
				if val, ok := r.Value[1].(float64); ok {
					cpuMap[ts] = val * 100
				}
			}
		}
	}

	for _, r := range ramResults {
		if len(r.Value) >= 2 {
			if ts, ok := r.Value[0].(float64); ok {
				if val, ok := r.Value[1].(float64); ok {
					ramMap[ts] = val * 100
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
