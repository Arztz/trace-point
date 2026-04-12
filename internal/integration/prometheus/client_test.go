package prometheus

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

func MockLogger() *logger.Logger {
	return logger.New(os.Stdout, logger.DebugLevel)
}

func TestPrometheusClientQuery(t *testing.T) {
	// Create a mock Prometheus server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify query parameter
		query := r.URL.Query().Get("query")
		if query == "" {
			t.Error("Query parameter is empty")
		}

		// Return mock response
		response := MetricQueryResult{
			Status: "success",
			Data: Data{
				ResultType: "vector",
				Result: []Result{
					{
						Metric: map[string]string{
							"pod":       "test-pod",
							"namespace": "default",
							"container": "nginx",
						},
						Value: []interface{}{time.Now().Unix(), 50.5},
					},
				},
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	// Create client
	cfg := &config.PrometheusConfig{
		URL:     server.URL,
		Timeout: 10 * time.Second,
	}
	client := NewClient(cfg, MockLogger())

	// Execute query
	results, err := client.Query(context.Background(), "test_query", time.Now())
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}

	// Verify results
	if len(results) != 1 {
		t.Errorf("Expected 1 result, got %d", len(results))
	}

	if results[0].Metric["pod"] != "test-pod" {
		t.Errorf("Expected pod 'test-pod', got '%s'", results[0].Metric["pod"])
	}
}

func TestPrometheusClientQueryRange(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify range query parameters
		if r.URL.Query().Get("query") == "" {
			t.Error("Query parameter is empty")
		}
		if r.URL.Query().Get("start") == "" {
			t.Error("Start parameter is empty")
		}
		if r.URL.Query().Get("end") == "" {
			t.Error("End parameter is empty")
		}
		if r.URL.Query().Get("step") == "" {
			t.Error("Step parameter is empty")
		}

		response := MetricQueryResult{
			Status: "success",
			Data: Data{
				ResultType: "matrix",
				Result:     []Result{},
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	cfg := &config.PrometheusConfig{
		URL:     server.URL,
		Timeout: 10 * time.Second,
	}
	client := NewClient(cfg, MockLogger())

	start := time.Now().Add(-1 * time.Hour)
	end := time.Now()

	results, err := client.QueryRange(context.Background(), "test_query", start, end, 30*time.Second)
	if err != nil {
		t.Fatalf("QueryRange failed: %v", err)
	}

	if results == nil {
		t.Error("Results should not be nil")
	}
}

func TestPrometheusClientFetchContainerMetrics(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path

		var response MetricQueryResult

		if path == "/api/v1/query" {
			query := r.URL.Query().Get("query")

			if query == "container_cpu_usage_seconds_total" || query == "container_cpu_usage_seconds_total{namespace=~\"default|production\"}" {
				response = MetricQueryResult{
					Status: "success",
					Data: Data{
						ResultType: "vector",
						Result: []Result{
							{
								Metric: map[string]string{
									"pod":       "test-pod",
									"namespace": "default",
									"container": "nginx",
								},
								Value: []interface{}{time.Now().Unix(), 0.5},
							},
						},
					},
				}
			} else if query == "container_memory_working_set_bytes" || query == "container_memory_working_set_bytes{namespace=~\"default|production\"}" {
				response = MetricQueryResult{
					Status: "success",
					Data: Data{
						ResultType: "vector",
						Result: []Result{
							{
								Metric: map[string]string{
									"pod":       "test-pod",
									"namespace": "default",
									"container": "nginx",
								},
								Value: []interface{}{time.Now().Unix(), 0.7},
							},
						},
					},
				}
			} else {
				response = MetricQueryResult{
					Status: "success",
					Data:   Data{ResultType: "vector", Result: []Result{}},
				}
			}
		} else {
			http.NotFound(w, r)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	cfg := &config.PrometheusConfig{
		URL:     server.URL,
		Timeout: 10 * time.Second,
	}
	client := NewClient(cfg, MockLogger())

	metrics, err := client.FetchContainerMetrics(context.Background(), []string{"default", "production"}, nil)
	if err != nil {
		t.Fatalf("FetchContainerMetrics failed: %v", err)
	}

	if len(metrics) != 1 {
		t.Errorf("Expected 1 metric, got %d", len(metrics))
	}

	if metrics[0].PodName != "test-pod" {
		t.Errorf("Expected pod 'test-pod', got '%s'", metrics[0].PodName)
	}

	if metrics[0].CPUPercent != 50.0 {
		t.Errorf("Expected CPU 50.0, got %f", metrics[0].CPUPercent)
	}

	if metrics[0].RAMPercent != 70.0 {
		t.Errorf("Expected RAM 70.0, got %f", metrics[0].RAMPercent)
	}
}

func TestPrometheusClientGetHistoricalMetrics(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		query := r.URL.Query().Get("query")

		var response MetricQueryResult

		if path == "/api/v1/query_range" {
			// Return mock historical data
			response = MetricQueryResult{
				Status: "success",
				Data: Data{
					ResultType: "matrix",
					Result: []Result{
						{
							Metric: map[string]string{
								"pod":       "test-pod",
								"namespace": "default",
								"container": "nginx",
							},
							Value: []interface{}{
								[]interface{}{time.Now().Add(-30*time.Second).Unix(), 50.0},
								[]interface{}{time.Now().Unix(), 60.0},
							},
						},
					},
				},
			}

			// Check query contains pod name
			if query == "" {
				t.Error("Query is empty")
			}
		} else {
			response = MetricQueryResult{
				Status: "success",
				Data:   Data{ResultType: "matrix", Result: []Result{}},
			}
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	cfg := &config.PrometheusConfig{
		URL:     server.URL,
		Timeout: 10 * time.Second,
	}
	client := NewClient(cfg, MockLogger())

	metrics, err := client.GetHistoricalMetrics(context.Background(), "test-pod", "default", "nginx", 1*time.Hour)
	if err != nil {
		t.Fatalf("GetHistoricalMetrics failed: %v", err)
	}

	// Should have some metrics (even if query range returns empty results due to mock)
	if metrics == nil {
		t.Error("Metrics should not be nil")
	}
}

func TestPrometheusClientErrorHandling(t *testing.T) {
	// Test with unreachable server
	cfg := &config.PrometheusConfig{
		URL:     "http://localhost:19999",
		Timeout: 1 * time.Second,
	}
	client := NewClient(cfg, MockLogger())

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	_, err := client.Query(ctx, "test_query", time.Now())
	if err == nil {
		t.Error("Expected error for unreachable server")
	}
}

func TestPrometheusClientInvalidResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Bad Request"))
	}))
	defer server.Close()

	cfg := &config.PrometheusConfig{
		URL:     server.URL,
		Timeout: 10 * time.Second,
	}
	client := NewClient(cfg, MockLogger())

	_, err := client.Query(context.Background(), "test_query", time.Now())
	if err == nil {
		t.Error("Expected error for invalid response")
	}
}

func TestPrometheusClientJSONError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Return invalid JSON
		w.Write([]byte("{invalid json"))
	}))
	defer server.Close()

	cfg := &config.PrometheusConfig{
		URL:     server.URL,
		Timeout: 10 * time.Second,
	}
	client := NewClient(cfg, MockLogger())

	_, err := client.Query(context.Background(), "test_query", time.Now())
	if err == nil {
		t.Error("Expected error for invalid JSON")
	}
}

func TestPrometheusClientNonSuccessStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Return success status but error in body
		response := MetricQueryResult{
			Status: "error",
			Data:   Data{ResultType: "", Result: nil},
		}
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	cfg := &config.PrometheusConfig{
		URL:     server.URL,
		Timeout: 10 * time.Second,
	}
	client := NewClient(cfg, MockLogger())

	_, err := client.Query(context.Background(), "test_query", time.Now())
	if err == nil {
		t.Error("Expected error for non-success status")
	}
}