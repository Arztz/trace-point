package profiler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/trace-point/trace-point/internal/config"
	"github.com/trace-point/trace-point/internal/utils/logger"
	"os"
)

func TestProfilerClientQueryProfile(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request
		if r.Method != "GET" {
			t.Errorf("Expected GET method, got %s", r.Method)
		}

		// Verify query parameters
		query := r.URL.Query()
		if query.Get("query") == "" {
			t.Error("Query parameter is empty")
		}
		if query.Get("from") == "" {
			t.Error("From parameter is empty")
		}
		if query.Get("until") == "" {
			t.Error("Until parameter is empty")
		}

		// Return mock response
		response := ProfileResponse{
			Flamegraph: FlamegraphData{
				Functions: []FunctionData{
					{
						Name:       "processRequest",
						FilePath:   "internal/handler/api.go",
						Line:       42,
						SelfCPU:    30.0,
						TotalCPU:   45.5,
					},
					{
						Name:       "dbQuery",
						FilePath:   "internal/db/query.go",
						Line:       128,
						SelfCPU:    20.0,
						TotalCPU:   30.2,
					},
				},
				Metadata: ProfileMetadata{
					StartTime:   time.Now().Add(-5*time.Minute).UnixMilli(),
					EndTime:    time.Now().UnixMilli(),
					ProfileType: "CPU",
					ServiceName: "test-service",
				},
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	// Create client
	cfg := &config.ProfilerConfig{
		Interval:   10 * time.Second,
		PyroscopeURL: server.URL,
	}
	gcloudCfg := &config.GCloudConfig{
		ProjectID: "test-project",
	}
	client := NewClient(cfg, gcloudCfg, MockLogger())

	// Add service mapping
	client.RegisterServiceMapping("default", "test-pod-abc123", "test-service")

	// Execute query
	startTime := time.Now().Add(-5 * time.Minute)
	endTime := time.Now()

	result, err := client.QueryProfile(context.Background(), "default", "test-pod-abc123", startTime, endTime)
	if err != nil {
		t.Fatalf("QueryProfile failed: %v", err)
	}

	// Verify results
	if result == nil {
		t.Error("Result should not be nil")
	}

	if len(result.TopFunctions) != 2 {
		t.Errorf("Expected 2 functions, got %d", len(result.TopFunctions))
	}

	if result.TopFunctions[0].FunctionName != "processRequest" {
		t.Errorf("Expected function 'processRequest', got '%s'", result.TopFunctions[0].FunctionName)
	}
}

func TestProfilerClientResolveServiceName(t *testing.T) {
	cfg := &config.ProfilerConfig{
		Interval:   10 * time.Second,
		PyroscopeURL: "http://localhost:4040",
	}
	gcloudCfg := &config.GCloudConfig{}
	client := NewClient(cfg, gcloudCfg, MockLogger())

	// Register a service mapping
	client.RegisterServiceMapping("default", "test-pod-123", "my-service")

	// Test direct mapping
	service := client.resolveServiceName("default", "test-pod-123")
	if service != "my-service" {
		t.Errorf("Expected 'my-service', got '%s'", service)
	}

	// Test namespace default
	client.RegisterNamespaceMapping("production", "prod-service")
	prodService := client.resolveServiceName("production", "other-pod")
	if prodService != "prod-service" {
		t.Errorf("Expected 'prod-service', got '%s'", prodService)
	}
}

func TestProfilerClientBuildProfileQueryURL(t *testing.T) {
	cfg := &config.ProfilerConfig{
		Interval:   10 * time.Second,
		PyroscopeURL: "http://localhost:4040",
	}
	gcloudCfg := &config.GCloudConfig{}
	client := NewClient(cfg, gcloudCfg, MockLogger())

	startTime := time.Now().Add(-5 * time.Minute)
	endTime := time.Now()

	url := client.buildProfileQueryURL("test-service", startTime, endTime)

	// Verify URL contains expected parts
	if url == "" {
		t.Error("URL should not be empty")
	}

	// Should contain service name
	if len(url) < 20 {
		t.Error("URL seems too short")
	}
}

func TestProfilerClientParseProfileResponse(t *testing.T) {
	cfg := &config.ProfilerConfig{
		Interval:   10 * time.Second,
		PyroscopeURL: "http://localhost:4040",
	}
	gcloudCfg := &config.GCloudConfig{}
	client := NewClient(cfg, gcloudCfg, MockLogger())

	// Test with valid response
	response := ProfileResponse{
		Flamegraph: FlamegraphData{
			Functions: []FunctionData{
				{Name: "func1", TotalCPU: 50.0},
				{Name: "func2", TotalCPU: 30.0},
				{Name: "func3", TotalCPU: 20.0},
			},
			Metadata: ProfileMetadata{
				ProfileType: "CPU",
				ServiceName: "test",
			},
		},
	}

	data, _ := json.Marshal(response)
	result, err := client.parseProfileResponse(data)
	if err != nil {
		t.Fatalf("parseProfileResponse failed: %v", err)
	}

	if len(result.TopFunctions) == 0 {
		t.Error("Should have parsed functions")
	}

	// Verify sorting (by total CPU descending)
	if result.TopFunctions[0].FunctionName != "func1" {
		t.Errorf("Expected 'func1' first, got '%s'", result.TopFunctions[0].FunctionName)
	}
}

func TestProfilerClientParseEmptyResponse(t *testing.T) {
	cfg := &config.ProfilerConfig{
		Interval:   10 * time.Second,
		PyroscopeURL: "http://localhost:4040",
	}
	gcloudCfg := &config.GCloudConfig{}
	client := NewClient(cfg, gcloudCfg, MockLogger())

	// Test with empty response
	response := ProfileResponse{
		Flamegraph: FlamegraphData{
			Functions: []FunctionData{},
			Metadata: ProfileMetadata{
				ProfileType: "CPU",
				ServiceName: "test",
			},
		},
	}

	data, _ := json.Marshal(response)
	result, err := client.parseProfileResponse(data)
	if err != nil {
		t.Fatalf("parseProfileResponse failed: %v", err)
	}

	// Should return empty result gracefully
	if result == nil {
		t.Error("Should return result, not nil")
	}
}

func TestProfilerClientErrorHandling(t *testing.T) {
	// Test with unreachable server
	cfg := &config.ProfilerConfig{
		Interval:   1 * time.Second,
		PyroscopeURL: "http://localhost:19999",
	}
	gcloudCfg := &config.GCloudConfig{}
	client := NewClient(cfg, gcloudCfg, MockLogger())

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	client.RegisterServiceMapping("default", "test-pod", "test-service")

	_, err := client.QueryProfile(ctx, "default", "test-pod", time.Now().Add(-5*time.Minute), time.Now())
	if err == nil {
		t.Error("Expected error for unreachable server")
	}
}

func TestProfilerClientInvalidResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Internal Server Error"))
	}))
	defer server.Close()

	cfg := &config.ProfilerConfig{
		Interval:   10 * time.Second,
		PyroscopeURL: server.URL,
	}
	gcloudCfg := &config.GCloudConfig{}
	client := NewClient(cfg, gcloudCfg, MockLogger())

	client.RegisterServiceMapping("default", "test-pod", "test-service")

	_, err := client.QueryProfile(context.Background(), "default", "test-pod", time.Now().Add(-5*time.Minute), time.Now())
	if err == nil {
		t.Error("Expected error for invalid response")
	}
}

func TestProfilerClientJSONError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Return invalid JSON
		w.Write([]byte("{invalid json"))
	}))
	defer server.Close()

	cfg := &config.ProfilerConfig{
		Interval:   10 * time.Second,
		PyroscopeURL: server.URL,
	}
	gcloudCfg := &config.GCloudConfig{}
	client := NewClient(cfg, gcloudCfg, MockLogger())

	client.RegisterServiceMapping("default", "test-pod", "test-service")

	_, err := client.QueryProfile(context.Background(), "default", "test-pod", time.Now().Add(-5*time.Minute), time.Now())
	if err == nil {
		t.Error("Expected error for invalid JSON")
	}
}

func TestProfilerClientNoServiceMapping(t *testing.T) {
	cfg := &config.ProfilerConfig{
		Interval:   10 * time.Second,
		PyroscopeURL: "http://localhost:4040",
	}
	gcloudCfg := &config.GCloudConfig{}
	client := NewClient(cfg, gcloudCfg, MockLogger())

	// Try to query without any service mapping
	_, err := client.QueryProfile(context.Background(), "unknown-namespace", "unknown-pod", time.Now().Add(-5*time.Minute), time.Now())
	if err == nil {
		t.Error("Expected error for no service mapping")
	}
}

func TestProfilerClientDefaultURL(t *testing.T) {
	cfg := &config.ProfilerConfig{
		Interval:   10 * time.Second,
		PyroscopeURL: "",
	}
	gcloudCfg := &config.GCloudConfig{}
	client := NewClient(cfg, gcloudCfg, MockLogger())

	// Should default to localhost:4040
	if client.baseURL != "http://localhost:4040" {
		t.Errorf("Expected default URL 'http://localhost:4040', got '%s'", client.baseURL)
	}
}

func TestProfilerMockClient(t *testing.T) {
	// Test the mock client functionality
	mockClient := NewMockClient()

	result, err := mockClient.QueryProfile(context.Background(), "default", "test-pod", time.Now(), time.Now())
	if err != nil {
		t.Fatalf("Mock client QueryProfile failed: %v", err)
	}

	if len(result.TopFunctions) != 2 {
		t.Errorf("Expected 2 mock functions, got %d", len(result.TopFunctions))
	}

	// Test setting mock error
	mockClient.SetMockError(context.DeadlineExceeded)
	_, err = mockClient.QueryProfile(context.Background(), "default", "test-pod", time.Now(), time.Now())
	if err != context.DeadlineExceeded {
		t.Errorf("Expected context.DeadlineExceeded error")
	}
}

// MockLogger creates a test logger
func MockLogger() *logger.Logger {
	return logger.New(os.Stdout, logger.DebugLevel)
}