package signoz

import (
	"bytes"
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

func TestSignozClientQueryTraces(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request
		if r.Method != "POST" {
			t.Errorf("Expected POST method, got %s", r.Method)
		}

		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("Expected Content-Type: application/json, got %s", r.Header.Get("Content-Type"))
		}

		// Decode request body
		var query ClickHouseQuery
		if err := json.NewDecoder(r.Body).Decode(&query); err != nil {
			t.Errorf("Failed to decode request body: %v", err)
		}

		// Return mock response
		response := map[string]interface{}{
			"traces": []map[string]interface{}{
				{
					"traceID":   "abc-123",
					"spans":     []map[string]interface{}{},
					"timestamp": time.Now().UnixMilli(),
					"duration":  1500000000,
				},
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	// Create client
	cfg := &config.SignozConfig{
		URL:     server.URL,
		Timeout: 10 * time.Second,
	}
	gcloudCfg := &config.GCloudConfig{}
	client := NewClient(cfg, gcloudCfg, MockLogger())

	// Execute query
	startTime := time.Now().Add(-10 * time.Minute)
	endTime := time.Now()

	result, err := client.QueryTraces(context.Background(), "default", "test-pod", startTime, endTime)
	if err != nil {
		t.Fatalf("QueryTraces failed: %v", err)
	}

	// Verify result
	if result == nil {
		t.Error("Result should not be nil")
	}
}

func TestSignozClientQueryTracesByTimeWindow(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := map[string]interface{}{
			"traces": []map[string]interface{}{},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	cfg := &config.SignozConfig{
		URL:     server.URL,
		Timeout: 10 * time.Second,
	}
	gcloudCfg := &config.GCloudConfig{}
	client := NewClient(cfg, gcloudCfg, MockLogger())

	spikeTime := time.Now()
	result, err := client.QueryTracesByTimeWindow(context.Background(), "default", "test-pod", spikeTime, 5)
	if err != nil {
		t.Fatalf("QueryTracesByTimeWindow failed: %v", err)
	}

	if result == nil {
		t.Error("Result should not be nil")
	}
}

func TestSignozClientBuildTraceQuery(t *testing.T) {
	cfg := &config.SignozConfig{
		URL:     "http://localhost:3301",
		Timeout: 10 * time.Second,
	}
	gcloudCfg := &config.GCloudConfig{}
	client := NewClient(cfg, gcloudCfg, MockLogger())

	startTime := time.Now().Add(-10 * time.Minute)
	endTime := time.Now()

	query := client.buildTraceQuery("default", "test-pod", startTime, endTime)

	// Verify query contains expected parts
	if query == "" {
		t.Error("Query should not be empty")
	}

	// The query should contain namespace filter
	if len(query) < 10 {
		t.Error("Query seems too short")
	}
}

func TestSignozClientGetServiceDependencies(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify it's a POST request with ClickHouse query
		var query ClickHouseQuery
		if err := json.NewDecoder(r.Body).Decode(&query); err != nil {
			t.Errorf("Failed to decode request: %v", err)
		}

		response := map[string]interface{}{
			"data": []map[string]interface{}{},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	cfg := &config.SignozConfig{
		URL:     server.URL,
		Timeout: 10 * time.Second,
	}
	gcloudCfg := &config.GCloudConfig{}
	client := NewClient(cfg, gcloudCfg, MockLogger())

	startTime := time.Now().Add(-1 * time.Hour)
	endTime := time.Now()

	deps, err := client.GetServiceDependencies(context.Background(), "default", startTime, endTime)
	if err != nil {
		t.Fatalf("GetServiceDependencies failed: %v", err)
	}

	// Should return empty or nil (graceful handling)
	if deps == nil {
		t.Error("Should return empty slice, not nil")
	}
}

func TestSignozClientErrorHandling(t *testing.T) {
	// Test with unreachable server
	cfg := &config.SignozConfig{
		URL:     "http://localhost:19998",
		Timeout: 1 * time.Second,
	}
	gcloudCfg := &config.GCloudConfig{}
	client := NewClient(cfg, gcloudCfg, MockLogger())

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	_, err := client.QueryTraces(ctx, "default", "test-pod", time.Now().Add(-10*time.Minute), time.Now())
	if err == nil {
		t.Error("Expected error for unreachable server")
	}
}

func TestSignozClientInvalidResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Internal Server Error"))
	}))
	defer server.Close()

	cfg := &config.SignozConfig{
		URL:     server.URL,
		Timeout: 10 * time.Second,
	}
	gcloudCfg := &config.GCloudConfig{}
	client := NewClient(cfg, gcloudCfg, MockLogger())

	_, err := client.QueryTraces(context.Background(), "default", "test-pod", time.Now().Add(-10*time.Minute), time.Now())
	if err == nil {
		t.Error("Expected error for invalid response")
	}
}

func TestSignozClientParseError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Return invalid JSON
		w.Write([]byte("{invalid json"))
	}))
	defer server.Close()

	cfg := &config.SignozConfig{
		URL:     server.URL,
		Timeout: 10 * time.Second,
	}
	gcloudCfg := &config.GCloudConfig{}
	client := NewClient(cfg, gcloudCfg, MockLogger())

	_, err := client.QueryTraces(context.Background(), "default", "test-pod", time.Now().Add(-10*time.Minute), time.Now())
	if err == nil {
		t.Error("Expected error for invalid JSON")
	}
}

func TestSignozClientAuthToken(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify auth token is set
		authHeader := r.Header.Get("Authorization")
		if authHeader != "Bearer test-token" {
			t.Errorf("Expected Authorization header, got: %s", authHeader)
		}

		response := map[string]interface{}{
			"traces": []map[string]interface{}{},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	cfg := &config.SignozConfig{
		URL:     server.URL,
		Timeout: 10 * time.Second,
	}
	gcloudCfg := &config.GCloudConfig{}
	client := NewClient(cfg, gcloudCfg, MockLogger())

	client.SetAuthToken("test-token")

	_, err := client.QueryTraces(context.Background(), "default", "test-pod", time.Now().Add(-10*time.Minute), time.Now())
	if err != nil {
		t.Fatalf("QueryTraces failed: %v", err)
	}
}

func TestSignozClientDefaultURL(t *testing.T) {
	cfg := &config.SignozConfig{
		URL:     "",
		Timeout: 10 * time.Second,
	}
	gcloudCfg := &config.GCloudConfig{}
	client := NewClient(cfg, gcloudCfg, MockLogger())

	// Should default to localhost:3301
	if client.baseURL != "http://localhost:3301" {
		t.Errorf("Expected default URL 'http://localhost:3301', got '%s'", client.baseURL)
	}
}

func TestSignozClientDefaultQueryPath(t *testing.T) {
	cfg := &config.SignozConfig{
		URL:      "http://test.com",
		QueryPath: "",
		Timeout:  10 * time.Second,
	}
	gcloudCfg := &config.GCloudConfig{}
	client := NewClient(cfg, gcloudCfg, MockLogger())

	// Should default to /api/v1/traces
	if client.queryPath != "/api/v1/traces" {
		t.Errorf("Expected default query path '/api/v1/traces', got '%s'", client.queryPath)
	}
}

// MockLogger creates a test logger
func MockLogger() *logger.Logger {
	return logger.New(os.Stdout, logger.DebugLevel)
}

// Helper to avoid unused import error
var _ = bytes.NewReader