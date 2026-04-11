package profiler

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/trace-point/trace-point/internal/config"
	"github.com/trace-point/trace-point/internal/utils/logger"
)

// Client handles communication with Gcloud Profiler API
type Client struct {
	httpClient *http.Client
	baseURL    string
	projectID  string
	timeout    time.Duration
	logger     *logger.Logger
	serviceMap map[string]string // Map service name to profiler service
}

// NewClient creates a new Gcloud Profiler client
func NewClient(cfg *config.ProfilerConfig, gcloudCfg *config.GCloudConfig, logger *logger.Logger) *Client {
	transport := &http.Transport{
		MaxIdleConns:        10,
		MaxIdleConnsPerHost: 5,
		IdleConnTimeout:     30 * time.Second,
	}

	httpClient := &http.Client{
		Transport: transport,
		Timeout:   cfg.Interval,
	}

	baseURL := cfg.PyroscopeURL
	if baseURL == "" {
		// Default to local Pyroscope installation (compatible with Gcloud profiler format)
		baseURL = "http://localhost:4040"
	}

	client := &Client{
		httpClient: httpClient,
		baseURL:    baseURL,
		timeout:    cfg.Interval,
		logger:     logger,
		serviceMap: make(map[string]string),
	}

	// Set project ID from GCloud config
	if gcloudCfg != nil {
		client.projectID = gcloudCfg.ProjectID
	}

	return client
}

// ProfileResponse represents the profiler API response
type ProfileResponse struct {
	Flamegraph FlamegraphData `json:"flamegraph"`
}

// FlamegraphData represents flamegraph data from profiler
type FlamegraphData struct {
	Functions []FunctionData  `json:"functions"`
	Metadata  ProfileMetadata `json:"metadata"`
}

// FunctionData represents a function in the flamegraph
type FunctionData struct {
	Name        string  `json:"name"`
	FilePath    string  `json:"file"`
	Line        int     `json:"line"`
	SelfCPU     float64 `json:"selfCPU"`
	TotalCPU    float64 `json:"totalCPU"`
	SelfMemory  int64   `json:"selfMemory"`
	TotalMemory int64   `json:"totalMemory"`
}

// ProfileMetadata contains metadata about the profile
type ProfileMetadata struct {
	StartTime   int64  `json:"startTime"`
	EndTime     int64  `json:"endTime"`
	ProfileType string `json:"profileType"`
	ServiceName string `json:"serviceName"`
}

// ProfileResult represents parsed profile results for correlation
type ProfileResult struct {
	TopFunctions []FunctionProfile
	ServiceName  string
	ProfileType  string
}

// FunctionProfile represents a function profile for correlation
type FunctionProfile struct {
	FunctionName string  `json:"function_name"`
	FilePath     string  `json:"file_path"`
	LineNumber   int     `json:"line_number"`
	CPUPercent   float64 `json:"cpu_percent"`
}

// QueryProfile queries the profiler for a specific time range and service
func (c *Client) QueryProfile(ctx context.Context, namespace, podName string, startTime, endTime time.Time) (*ProfileResult, error) {
	// Determine the service name from namespace/pod
	serviceName := c.resolveServiceName(namespace, podName)

	if serviceName == "" {
		c.logger.Debug("No profiler service mapping found for %s/%s", namespace, podName)
		return nil, fmt.Errorf("no profiler service configured for %s/%s", namespace, podName)
	}

	c.logger.Info("Querying profiler for service: %s, time range: %s to %s",
		serviceName, startTime.Format(time.RFC3339), endTime.Format(time.RFC3339))

	// Build the profiler query URL
	queryURL := c.buildProfileQueryURL(serviceName, startTime, endTime)

	// Execute the query
	data, err := c.executeProfileQuery(ctx, queryURL)
	if err != nil {
		c.logger.Warn("Failed to query profiler for %s: %v (continuing without profiler data)", serviceName, err)
		return nil, fmt.Errorf("profiler query failed: %w", err)
	}

	// Parse the response
	result, err := c.parseProfileResponse(data)
	if err != nil {
		c.logger.Warn("Failed to parse profiler response: %v", err)
		return nil, fmt.Errorf("failed to parse profiler response: %w", err)
	}

	c.logger.Info("Profiler query successful - Found %d functions", len(result.TopFunctions))

	return result, nil
}

// resolveServiceName determines the profiler service name from namespace/pod
func (c *Client) resolveServiceName(namespace, podName string) string {
	// Check if there's a direct mapping
	if service, ok := c.serviceMap[fmt.Sprintf("%s/%s", namespace, podName)]; ok {
		return service
	}

	// Try to extract service name from pod name patterns
	// Common pattern: service-name-xxxxx -> service-name
	if len(podName) > 5 {
		// Try common suffix patterns (deployment, statefulset, etc.)
		for _, suffix := range []string{"-", "-v", "-run"} {
			if idx := findLastIndex(podName, suffix+"[0-9]"); idx > 0 {
				return podName[:idx]
			}
		}
	}

	// Default to namespace-based service name
	if namespace != "" {
		return fmt.Sprintf("%s-service", namespace)
	}

	return ""
}

// findLastIndex finds the last index where the pattern matches
func findLastIndex(s, pattern string) int {
	// Simplified - just return position of last dash followed by digits
	for i := len(s) - 1; i >= 0; i-- {
		if s[i] == '-' && i+1 < len(s) {
			// Check if followed by digit
			if i+1 < len(s) && s[i+1] >= '0' && s[i+1] <= '9' {
				return i
			}
		}
	}
	return -1
}

// buildProfileQueryURL builds the profiler query URL
func (c *Client) buildProfileQueryURL(serviceName string, startTime, endTime time.Time) string {
	// Pyroscope uses a specific query format
	// Format: /pyroscope/ingest?query={serviceName}&from={start}&until={end}
	startTs := startTime.UnixMilli()
	endTs := endTime.UnixMilli()

	// Build query for CPU profile
	query := fmt.Sprintf("CPU{service_name='%s'}", serviceName)

	return fmt.Sprintf("%s/pyroscope/data?query=%s&from=%d&until=%d&format=json",
		c.baseURL, query, startTs, endTs)
}

// executeProfileQuery executes the profiler API query
func (c *Client) executeProfileQuery(ctx context.Context, url string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute query: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("profiler returned status %d: %s", resp.StatusCode, string(body))
	}

	return io.ReadAll(resp.Body)
}

// parseProfileResponse parses the profiler response into profile results
func (c *Client) parseProfileResponse(data []byte) (*ProfileResult, error) {
	var response ProfileResponse
	if err := json.Unmarshal(data, &response); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}

	result := &ProfileResult{
		TopFunctions: make([]FunctionProfile, 0),
		ServiceName:  response.Flamegraph.Metadata.ServiceName,
		ProfileType:  response.Flamegraph.Metadata.ProfileType,
	}

	// Convert and sort functions by CPU usage
	functions := response.Flamegraph.Functions
	if len(functions) == 0 {
		c.logger.Debug("No functions found in profiler response")
		return result, nil
	}

	// Sort by total CPU (descending)
	for i := 0; i < len(functions)-1; i++ {
		for j := i + 1; j < len(functions); j++ {
			if functions[j].TotalCPU > functions[i].TotalCPU {
				functions[i], functions[j] = functions[j], functions[i]
			}
		}
	}

	// Take top 10 functions
	maxResults := 10
	if len(functions) < maxResults {
		maxResults = len(functions)
	}

	for i := 0; i < maxResults; i++ {
		funcProfile := FunctionProfile{
			FunctionName: functions[i].Name,
			FilePath:     functions[i].FilePath,
			LineNumber:   functions[i].Line,
			CPUPercent:   functions[i].TotalCPU,
		}
		result.TopFunctions = append(result.TopFunctions, funcProfile)
	}

	return result, nil
}

// RegisterServiceMapping registers a namespace/pod to profiler service mapping
func (c *Client) RegisterServiceMapping(namespace, podName, serviceName string) {
	key := fmt.Sprintf("%s/%s", namespace, podName)
	c.serviceMap[key] = serviceName
}

// RegisterNamespaceMapping registers a namespace to default profiler service mapping
func (c *Client) RegisterNamespaceMapping(namespace, serviceName string) {
	// This sets a default for any pod in the namespace
	c.serviceMap[namespace] = serviceName
}

// Close closes the HTTP client
func (c *Client) Close() error {
	c.httpClient.CloseIdleConnections()
	return nil
}

// MockClient provides a mock implementation for testing
type MockClient struct {
	MockProfileResult *ProfileResult
	MockError         error
}

// NewMockClient creates a new mock profiler client
func NewMockClient() *MockClient {
	return &MockClient{
		MockProfileResult: &ProfileResult{
			TopFunctions: []FunctionProfile{
				{
					FunctionName: "processRequest",
					FilePath:     "internal/handler/api.go",
					LineNumber:   42,
					CPUPercent:   45.5,
				},
				{
					FunctionName: "dbQuery",
					FilePath:     "internal/db/query.go",
					LineNumber:   128,
					CPUPercent:   30.2,
				},
			},
		},
	}
}

// QueryProfile returns mock profile data
func (m *MockClient) QueryProfile(ctx context.Context, namespace, podName string, startTime, endTime time.Time) (*ProfileResult, error) {
	if m.MockError != nil {
		return nil, m.MockError
	}
	return m.MockProfileResult, nil
}

// SetMockResult sets the mock result
func (m *MockClient) SetMockResult(result *ProfileResult) {
	m.MockProfileResult = result
}

// SetMockError sets the mock error
func (m *MockClient) SetMockError(err error) {
	m.MockError = err
}
