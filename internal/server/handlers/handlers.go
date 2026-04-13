package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/trace-point/trace-point/internal/config"
	"github.com/trace-point/trace-point/internal/integration/prometheus"
	"github.com/trace-point/trace-point/internal/storage"
	"github.com/trace-point/trace-point/internal/utils/logger"
)

// Handler holds all handlers for the server
type Handler struct {
	repo             *storage.Repository
	prometheusClient *prometheus.Client
	appConfig        *config.Config
}

// NewHandler creates a new handler instance
func NewHandler(repo *storage.Repository, prometheusClient *prometheus.Client, appConfig *config.Config) *Handler {
	return &Handler{
		repo:             repo,
		prometheusClient: prometheusClient,
		appConfig:        appConfig,
	}
}

// RegisterRoutes registers all API routes
func RegisterRoutesWithRepo(r chi.Router, repo *storage.Repository, prometheusClient *prometheus.Client, appConfig *config.Config) {
	h := NewHandler(repo, prometheusClient, appConfig)

	// Export handler routes
	r.Get("/export", h.handleExportSpikes)
	r.Get("/export/refactoring", h.handleExportRefactoring)

	// Spike events routes
	r.Get("/spikes", h.handleSpikes)
	r.Get("/spikes/{id}", h.handleSpikeByID)

	// Timeline route
	r.Get("/timeline", h.handleTimeline)

	// Config route
	r.Get("/config", h.handleConfig)

	// Gravity scores route
	r.Get("/gravity-scores", h.handleGravityScores)
}

// handleSpikes returns a list of spike events
func (h *Handler) handleSpikes(w http.ResponseWriter, r *http.Request) {
	startTime := time.Now()
	ctx := r.Context()

	// Log request start
	logger.Debug("[API] handleSpikes START | Method=%s | Path=%s | Query=%s",
		r.Method, r.URL.Path, r.URL.RawQuery)

	// Parse query params
	limit := 100
	offset := 0
	namespace := r.URL.Query().Get("namespace")
	podFilter := r.URL.Query().Get("pod")

	logger.Debug("[API] handleSpikes | limit=%d | offset=%d | namespace=%s | podFilter=%s",
		limit, offset, namespace, podFilter)

	// Get time range
	endTime := time.Now()
	startTimeArg := endTime.AddDate(0, 0, -7) // Default 7 days

	events, err := h.repo.ListSpikeEvents(ctx, limit, offset, namespace, podFilter, startTimeArg, endTime)
	if err != nil {
		logger.Error("[API] handleSpikes ERROR | err=%v | duration=%v", err, time.Since(startTime))
		http.Error(w, fmt.Sprintf("Failed to fetch spike events: %v", err), http.StatusInternalServerError)
		return
	}

	logger.Debug("[API] handleSpikes SUCCESS | events_count=%d | duration=%v", len(events), time.Since(startTime))

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(events)
}

// handleSpikeByID returns a single spike event by ID
func (h *Handler) handleSpikeByID(w http.ResponseWriter, r *http.Request) {
	startTime := time.Now()
	ctx := r.Context()

	id := chi.URLParam(r, "id")

	logger.Debug("[API] handleSpikeByID START | Method=%s | Path=%s | id=%s",
		r.Method, r.URL.Path, id)

	event, err := h.repo.GetSpikeEvent(ctx, id)
	if err != nil {
		logger.Error("[API] handleSpikeByID ERROR | id=%s | err=%v | duration=%v", id, err, time.Since(startTime))
		http.Error(w, fmt.Sprintf("Failed to fetch spike event: %v", err), http.StatusInternalServerError)
		return
	}

	if event == nil {
		logger.Debug("[API] handleSpikeByID NOT_FOUND | id=%s | duration=%v", id, time.Since(startTime))
		http.Error(w, "Spike event not found", http.StatusNotFound)
		return
	}

	logger.Debug("[API] handleSpikeByID SUCCESS | id=%s | duration=%v", id, time.Since(startTime))

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(event)
}

// handleTimeline returns timeline data for visualization
func (h *Handler) handleTimeline(w http.ResponseWriter, r *http.Request) {
	startTime := time.Now()
	ctx := r.Context()

	logger.Debug("[API] handleTimeline START | Method=%s | Path=%s | Query=%s",
		r.Method, r.URL.Path, r.URL.RawQuery)

	// Parse query params
	now := time.Now()

	// Parse time range from query param (default: 24h)
	timeRange := r.URL.Query().Get("time_range")
	startTimeArg := now.AddDate(0, 0, -1) // Default 24 hours
	switch timeRange {
	case "1h":
		startTimeArg = now.Add(-1 * time.Hour)
	case "6h":
		startTimeArg = now.Add(-6 * time.Hour)
	case "24h":
		startTimeArg = now.AddDate(0, 0, -1)
	case "7d":
		startTimeArg = now.AddDate(-7, 0, 0)
	default:
		// Default to 24h
		startTimeArg = now.AddDate(0, 0, -1)
	}

	// Parse step parameter (default: 30s)
	stepStr := r.URL.Query().Get("step")
	step := 30 * time.Second
	if stepStr != "" {
		// Parse duration string (e.g., "30s", "1m", "5m")
		if parsed, err := time.ParseDuration(stepStr); err == nil {
			step = parsed
		}
	}

	// Parse source parameter (default: "all")
	source := r.URL.Query().Get("source")
	if source == "" {
		source = "all"
	}

	// NEW: Parse pod_name filter (comma-separated)
	podNameFilter := r.URL.Query().Get("pod_name")

	logger.Debug("[API] handleTimeline | timeRange=%s | step=%s | source=%s | pod_name=%s | startTime=%s | endTime=%s",
		timeRange, step, source, podNameFilter, startTimeArg.Format(time.RFC3339), now.Format(time.RFC3339))

	// Initialize response
	response := TimelineResponse{
		GeneratedAt:   now.Format(time.RFC3339),
		StartDate:     startTimeArg.Format(time.RFC3339),
		EndDate:       now.Format(time.RFC3339),
		Metrics:       []prometheus.TimelineMetric{},
		SpikeMarkers:  []storage.SpikeMarker{},
		AvailablePods: []prometheus.AvailablePod{},
	}

	// Query Prometheus for continuous metrics if source is "prometheus" or "all"
	if (source == "prometheus" || source == "all") && h.prometheusClient != nil {
		namespaces := h.appConfig.Namespaces
		if len(namespaces) == 0 {
			namespaces = []string{} // Empty means all namespaces
		}

		// Pass podNameFilter to query function
		metrics, err := h.prometheusClient.QueryTimelineMetrics(ctx, namespaces, startTimeArg, now, step, podNameFilter)
		if err != nil {
			logger.Error("Failed to query Prometheus timeline metrics: %v", err)
			// Graceful degradation - return empty metrics
			response.Metrics = []prometheus.TimelineMetric{}
		} else {
			response.Metrics = metrics
		}

		// Fetch available pods for dropdown (only if not filtering to specific pod)
		if podNameFilter == "" {
			availablePods, err := h.prometheusClient.GetAvailablePods(ctx, namespaces)
			if err != nil {
				logger.Error("Failed to fetch available pods: %v", err)
			} else {
				response.AvailablePods = availablePods
			}
		}
	}

	// Query SQLite for spike markers if source is "spikes" or "all"
	if source == "spikes" || source == "all" {
		events, err := h.repo.ListSpikeEvents(ctx, 500, 0, "", "", startTimeArg, now)
		if err != nil {
			logger.Error("Failed to fetch spike events for timeline: %v", err)
			// Graceful degradation - return empty markers
			response.SpikeMarkers = []storage.SpikeMarker{}
		} else {
			// Convert spike events to spike markers
			for _, event := range events {
				marker := storage.SpikeMarker{
					Timestamp: event.Timestamp,
					PodName:   event.PodName,
					Namespace: event.Namespace,
					CPUSpike:  event.CPUUsagePercent > 0,
					RAMSpike:  event.RAMUsagePercent > 0,
					RouteName: "",
					TraceID:   "",
				}
				if event.RouteName != nil {
					marker.RouteName = *event.RouteName
				}
				if event.TraceID != nil {
					marker.TraceID = *event.TraceID
				}
				response.SpikeMarkers = append(response.SpikeMarkers, marker)
			}
		}
	}

	logger.Debug("[API] handleTimeline SUCCESS | metrics_count=%d | spike_markers_count=%d | duration=%v",
		len(response.Metrics), len(response.SpikeMarkers), time.Since(startTime))

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// TimelineResponse represents the timeline API response format
type TimelineResponse struct {
	GeneratedAt   string                      `json:"generated_at"`
	StartDate     string                      `json:"start_date"`
	EndDate       string                      `json:"end_date"`
	Metrics       []prometheus.TimelineMetric `json:"metrics"`
	SpikeMarkers  []storage.SpikeMarker       `json:"spike_markers"`
	AvailablePods []prometheus.AvailablePod   `json:"availablePods"`
}

// handleConfig returns current configuration (stub - returns defaults)
func (h *Handler) handleConfig(w http.ResponseWriter, r *http.Request) {
	startTime := time.Now()
	ctx := r.Context()

	logger.Debug("[API] handleConfig START | Method=%s | Path=%s",
		r.Method, r.URL.Path)

	// Return stub config for now
	configs, err := h.repo.ListSpikeEvents(ctx, 10, 0, "", "", time.Time{}, time.Now())
	if err != nil {
		logger.Debug("[API] handleConfig SUCCESS | using_default | duration=%v", time.Since(startTime))
		// Return empty config response
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"config": "default"}`))
		return
	}

	_ = ctx
	logger.Debug("[API] handleConfig SUCCESS | events_count=%d | duration=%v", len(configs), time.Since(startTime))

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(configs)
}

// handleGravityScores returns resource gravity scores
func (h *Handler) handleGravityScores(w http.ResponseWriter, r *http.Request) {
	startTime := time.Now()
	ctx := r.Context()

	logger.Debug("[API] handleGravityScores START | Method=%s | Path=%s",
		r.Method, r.URL.Path)

	endTime := time.Now()
	startTimeArg := endTime.AddDate(0, 0, -30)

	logger.Debug("[API] handleGravityScores | startTime=%s | endTime=%s",
		startTimeArg.Format(time.RFC3339), endTime.Format(time.RFC3339))

	events, err := h.repo.ListSpikeEvents(ctx, 5000, 0, "", "", startTimeArg, endTime)
	if err != nil {
		logger.Error("[API] handleGravityScores ERROR | err=%v | duration=%v", err, time.Since(startTime))
		http.Error(w, fmt.Sprintf("Failed to fetch spike events: %v", err), http.StatusInternalServerError)
		return
	}

	// Calculate gravity scores by service
	gravityScores := calculateGravityScores(events)

	logger.Debug("[API] handleGravityScores SUCCESS | services_count=%d | duration=%v", len(gravityScores), time.Since(startTime))

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(gravityScores)
}

// handleExportSpikes exports spike history for the past 7 days
func (h *Handler) handleExportSpikes(w http.ResponseWriter, r *http.Request) {
	startTime := time.Now()
	ctx := r.Context()

	logger.Debug("[API] handleExportSpikes START | Method=%s | Path=%s",
		r.Method, r.URL.Path)

	// Default to past 7 days
	endTime := time.Now()
	startTimeArg := endTime.AddDate(0, 0, -7)

	logger.Debug("[API] handleExportSpikes | startTime=%s | endTime=%s",
		startTimeArg.Format(time.RFC3339), endTime.Format(time.RFC3339))

	// Query spike events for the past 7 days
	events, err := h.repo.ListSpikeEvents(ctx, 1000, 0, "", "", startTimeArg, endTime)
	if err != nil {
		logger.Error("[API] handleExportSpikes ERROR | err=%v | duration=%v", err, time.Since(startTime))
		http.Error(w, fmt.Sprintf("Failed to fetch spike events: %v", err), http.StatusInternalServerError)
		return
	}

	// Create export response
	export := SpikeExport{
		GeneratedAt: time.Now().Format(time.RFC3339),
		StartDate:   startTimeArg.Format(time.RFC3339),
		EndDate:     endTime.Format(time.RFC3339),
		TotalEvents: len(events),
		Events:      events,
	}

	// Set headers for file download
	filename := fmt.Sprintf("spike-history-%s.json", time.Now().UTC().Format("2006-01-02"))
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))

	logger.Debug("[API] handleExportSpikes SUCCESS | events_count=%d | duration=%v", len(events), time.Since(startTime))

	// Encode and write response
	if err := json.NewEncoder(w).Encode(export); err != nil {
		logger.Error("[API] handleExportSpikes ENCODE_ERROR | err=%v | duration=%v", err, time.Since(startTime))
		http.Error(w, fmt.Sprintf("Failed to encode response: %v", err), http.StatusInternalServerError)
	}
}

// handleExportRefactoring exports refactoring recommendations
func (h *Handler) handleExportRefactoring(w http.ResponseWriter, r *http.Request) {
	startTime := time.Now()
	ctx := r.Context()

	logger.Debug("[API] handleExportRefactoring START | Method=%s | Path=%s",
		r.Method, r.URL.Path)

	// Get all spike events for analysis (longer window for refactoring analysis)
	endTime := time.Now()
	startTimeArg := endTime.AddDate(0, 0, -30) // 30 days for better analysis

	logger.Debug("[API] handleExportRefactoring | startTime=%s | endTime=%s",
		startTimeArg.Format(time.RFC3339), endTime.Format(time.RFC3339))

	events, err := h.repo.ListSpikeEvents(ctx, 5000, 0, "", "", startTimeArg, endTime)
	if err != nil {
		logger.Error("[API] handleExportRefactoring ERROR | err=%v | duration=%v", err, time.Since(startTime))
		http.Error(w, fmt.Sprintf("Failed to fetch spike events: %v", err), http.StatusInternalServerError)
		return
	}

	// Analyze and calculate refactoring recommendations
	recommendations := analyzeRefactoring(events)

	// Create export response
	export := RefactoringExport{
		GeneratedAt:     time.Now().Format(time.RFC3339),
		AnalysisWindow:  fmt.Sprintf("past %d days", 30),
		TotalEvents:     len(events),
		Recommendations: recommendations,
	}

	// Set headers for file download
	filename := fmt.Sprintf("refactoring-intel-%s.json", time.Now().UTC().Format("2006-01-02"))
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))

	logger.Debug("[API] handleExportRefactoring SUCCESS | events_count=%d | recommendations_count=%d | duration=%v",
		len(events), len(recommendations), time.Since(startTime))

	// Encode and write response
	if err := json.NewEncoder(w).Encode(export); err != nil {
		logger.Error("[API] handleExportRefactoring ENCODE_ERROR | err=%v | duration=%v", err, time.Since(startTime))
		http.Error(w, fmt.Sprintf("Failed to encode response: %v", err), http.StatusInternalServerError)
	}
}

// calculateGravityScores calculates resource gravity scores
func calculateGravityScores(events []storage.SpikeEvent) []serviceGravity {
	serviceMap := make(map[string]*serviceAnalysis)

	for _, event := range events {
		key := fmt.Sprintf("%s/%s", event.Namespace, event.PodName)
		if _, ok := serviceMap[key]; !ok {
			routeName := safeString(event.RouteName)
			serviceMap[key] = &serviceAnalysis{
				namespace:   event.Namespace,
				podName:     event.PodName,
				routeName:   routeName,
				spikeCount:  0,
				totalCPU:    0,
				totalRAM:    0,
				maxCPU:      0,
				maxRAM:      0,
				occurrences: make(map[string]int),
			}
		}

		sa := serviceMap[key]
		sa.spikeCount++
		sa.totalCPU += event.CPUUsagePercent
		sa.totalRAM += event.RAMUsagePercent

		if event.CPUUsagePercent > sa.maxCPU {
			sa.maxCPU = event.CPUUsagePercent
		}
		if event.RAMUsagePercent > sa.maxRAM {
			sa.maxRAM = event.RAMUsagePercent
		}

		day := event.Timestamp.Format("2006-01-02")
		sa.occurrences[day]++
	}

	var scores []serviceGravity
	for _, sa := range serviceMap {
		if sa.spikeCount < 1 {
			continue
		}

		avgCPU := sa.totalCPU / float64(sa.spikeCount)
		avgRAM := sa.totalRAM / float64(sa.spikeCount)
		resourcePeak := (sa.maxCPU + sa.maxRAM) / 2
		callFreq := float64(len(sa.occurrences)) / 30.0
		gravityScore := resourcePeak * (1.0 / (callFreq + 0.1))

		scores = append(scores, serviceGravity{
			ServiceName:          fmt.Sprintf("%s/%s", sa.namespace, sa.podName),
			RouteName:            sa.routeName,
			SpikeCount:           sa.spikeCount,
			MaxCPUPercent:        sa.maxCPU,
			MaxRAMPercent:        sa.maxRAM,
			AverageCPUPercent:    avgCPU,
			AverageRAMPercent:    avgRAM,
			ResourceGravityScore: gravityScore,
		})
	}

	// Sort by gravity score using efficient sort.Slice (O(n log n))
	sort.Slice(scores, func(i, j int) bool {
		return scores[i].ResourceGravityScore > scores[j].ResourceGravityScore
	})

	return scores
}

// analyzeRefactoring analyzes spike events and generates refactoring recommendations
func analyzeRefactoring(events []storage.SpikeEvent) []RefactoringRecommendation {
	// Group events by service/route
	serviceMap := make(map[string]*serviceAnalysis)

	for _, event := range events {
		routeName := "unknown"
		if event.RouteName != nil {
			routeName = *event.RouteName
		}

		key := fmt.Sprintf("%s:%s:%s", event.Namespace, event.PodName, routeName)
		if _, ok := serviceMap[key]; !ok {
			serviceMap[key] = &serviceAnalysis{
				namespace:   event.Namespace,
				podName:     event.PodName,
				routeName:   routeName,
				spikeCount:  0,
				totalCPU:    0,
				totalRAM:    0,
				maxCPU:      0,
				maxRAM:      0,
				occurrences: make(map[string]int),
			}
		}

		sa := serviceMap[key]
		sa.spikeCount++
		sa.totalCPU += event.CPUUsagePercent
		sa.totalRAM += event.RAMUsagePercent

		if event.CPUUsagePercent > sa.maxCPU {
			sa.maxCPU = event.CPUUsagePercent
		}
		if event.RAMUsagePercent > sa.maxRAM {
			sa.maxRAM = event.RAMUsagePercent
		}

		// Track call frequency (unique timestamps per day)
		day := event.Timestamp.Format("2006-01-02")
		sa.occurrences[day]++
	}

	// Generate recommendations
	var recommendations []RefactoringRecommendation
	for _, sa := range serviceMap {
		if sa.spikeCount < 2 {
			continue // Skip services with too few spikes
		}

		// Calculate average resource usage
		avgCPU := sa.totalCPU / float64(sa.spikeCount)
		avgRAM := sa.totalRAM / float64(sa.spikeCount)

		// Calculate call frequency (total unique days / analysis period)
		callFreq := float64(len(sa.occurrences)) / 30.0

		// Calculate Resource Gravity Score
		// High Score = (High Resource Peak) × (Low Call Frequency)
		resourcePeak := (sa.maxCPU + sa.maxRAM) / 2
		gravityScore := resourcePeak * (1.0 / (callFreq + 0.1))

		// Determine if this is a suspected job/batch workload
		isJobRoute := isJobRoute(sa.routeName)

		// Determine separation strategy
		strategy := determineSeparationStrategy(sa.routeName, gravityScore, isJobRoute)

		recommendations = append(recommendations, RefactoringRecommendation{
			ServiceName:                 fmt.Sprintf("%s/%s", sa.namespace, sa.podName),
			RouteName:                   sa.routeName,
			SpikeCount:                  sa.spikeCount,
			MaxCPUPercent:               sa.maxCPU,
			MaxRAMPercent:               sa.maxRAM,
			AverageCPUPercent:           avgCPU,
			AverageRAMPercent:           avgRAM,
			CallFrequency:               callFreq,
			IsSuspectedJob:              isJobRoute,
			ResourceGravityScore:        gravityScore,
			SuggestedSeparationStrategy: strategy,
		})
	}

	// Sort by gravity score descending using efficient sort.Slice (O(n log n))
	sort.Slice(recommendations, func(i, j int) bool {
		return recommendations[i].ResourceGravityScore > recommendations[j].ResourceGravityScore
	})

	return recommendations
}

// serviceAnalysis holds analysis data for a service
type serviceAnalysis struct {
	namespace   string
	podName     string
	routeName   string
	spikeCount  int
	totalCPU    float64
	totalRAM    float64
	maxCPU      float64
	maxRAM      float64
	occurrences map[string]int
}

// serviceGravity represents gravity scores for a service
type serviceGravity struct {
	ServiceName          string  `json:"service_name"`
	RouteName            string  `json:"route_name"`
	SpikeCount           int     `json:"spike_count"`
	MaxCPUPercent        float64 `json:"max_cpu_percent"`
	MaxRAMPercent        float64 `json:"max_ram_percent"`
	AverageCPUPercent    float64 `json:"average_cpu_percent"`
	AverageRAMPercent    float64 `json:"average_ram_percent"`
	ResourceGravityScore float64 `json:"resource_gravity_score"`
}

// SpikeExport represents spike history export
type SpikeExport struct {
	GeneratedAt string               `json:"generated_at"`
	StartDate   string               `json:"start_date"`
	EndDate     string               `json:"end_date"`
	TotalEvents int                  `json:"total_events"`
	Events      []storage.SpikeEvent `json:"events"`
}

// RefactoringExport represents refactoring intelligence export
type RefactoringExport struct {
	GeneratedAt     string                      `json:"generated_at"`
	AnalysisWindow  string                      `json:"analysis_window"`
	TotalEvents     int                         `json:"total_events"`
	Recommendations []RefactoringRecommendation `json:"recommendations"`
}

// RefactoringRecommendation represents a single refactoring recommendation
type RefactoringRecommendation struct {
	ServiceName                 string  `json:"service_name"`
	RouteName                   string  `json:"route_name"`
	SpikeCount                  int     `json:"spike_count"`
	MaxCPUPercent               float64 `json:"max_cpu_percent"`
	MaxRAMPercent               float64 `json:"max_ram_percent"`
	AverageCPUPercent           float64 `json:"average_cpu_percent"`
	AverageRAMPercent           float64 `json:"average_ram_percent"`
	CallFrequency               float64 `json:"call_frequency_per_day"`
	IsSuspectedJob              bool    `json:"is_suspected_job"`
	ResourceGravityScore        float64 `json:"resource_gravity_score"`
	SuggestedSeparationStrategy string  `json:"suggested_separation_strategy"`
}

// isJobRoute determines if a route is a suspected job/batch route
func isJobRoute(routeName string) bool {
	jobPrefixes := []string{"/tasks/", "/batch/", "/jobs/"}
	for _, prefix := range jobPrefixes {
		if len(routeName) >= len(prefix) && routeName[:len(prefix)] == prefix {
			return true
		}
	}
	return false
}

// determineSeparationStrategy determines the recommended separation strategy
func determineSeparationStrategy(routeName string, gravityScore float64, isJob bool) string {
	if isJob {
		return "Extract to dedicated job worker service with scheduled scaling"
	}

	if gravityScore > 100 {
		return "Split to separate microservice with dedicated compute resources"
	}

	if gravityScore > 50 {
		return "Implement circuit breaker and fallback logic"
	}

	if routeName == "unknown" || routeName == "" {
		return "Investigate route attribution; implement request tracing"
	}

	return "Optimize query patterns and add caching layer"
}

// safeString safely extracts string from pointer
func safeString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
