package correlation

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/trace-point/trace-point/internal/storage"
	"github.com/trace-point/trace-point/internal/utils/logger"
)

// EngineConfig holds configuration for the correlation engine
type EngineConfig struct {
	PollingIntervalSeconds      int     `mapstructure:"polling_interval_seconds"`
	ThresholdPercent            float64 `mapstructure:"threshold_percent"`
	MovingAverageWindowMinutes  int     `mapstructure:"moving_average_window_minutes"`
	BaselineLearningMinutes     int     `mapstructure:"baseline_learning_period_minutes"`
	ReconciliationBufferMinutes int     `mapstructure:"reconciliation_buffer_minutes"`
	CooldownMinutes             int     `mapstructure:"cooldown_minutes"`
	Enabled                     bool    `mapstructure:"enabled"`
}

// DefaultEngineConfig returns default configuration
func DefaultEngineConfig() *EngineConfig {
	return &EngineConfig{
		PollingIntervalSeconds:      30,
		ThresholdPercent:            50.0,
		MovingAverageWindowMinutes:  30,
		BaselineLearningMinutes:     30,
		ReconciliationBufferMinutes: 8,
		CooldownMinutes:             15,
		Enabled:                     true,
	}
}

// Engine manages the correlation loop and orchestrates spike detection
type Engine struct {
	config        *EngineConfig
	logger        *logger.Logger
	detector      *SpikeDetector
	repository    *storage.Repository
	SignozQuery   SignozQuerier
	ProfilerQuery ProfilerQuerier
	router        *Router
	alertCallback AlertCallback

	ctx     context.Context
	cancel  context.CancelFunc
	wg      sync.WaitGroup
	mu      sync.RWMutex
	running bool

	alertBuffer      []SpikeAlert               // Pending alerts waiting for reconciliation
	correlationCache map[string]*RouteSelection // Cache of route selections by spike ID
}

// SignozQuerier interface for querying Signoz
type SignozQuerier interface {
	QueryTraces(namespace, podName string, startTime, endTime time.Time) (*TraceResult, error)
	QueryMetrics(namespace, podName string, startTime, endTime time.Time) (*MetricResult, error)
}

// ProfilerQuerier interface for querying profiler (Pyroscope)
type ProfilerQuerier interface {
	QueryProfile(namespace, podName string, startTime, endTime time.Time) (*ProfileResult, error)
}

// AlertCallback callback for handling alerts
type AlertCallback func(ctx context.Context, alert *SpikeAlert) error

// TraceResult represents trace query results
type TraceResult struct {
	Traces []Trace `json:"traces"`
}

// Trace represents a single trace
type Trace struct {
	TraceID string `json:"trace_id"`
	Spans   []Span `json:"spans"`
	Route   string `json:"route"`
}

// Span represents a span in a trace
type Span struct {
	ServiceName string `json:"service_name"`
	Operation   string `json:"operation"`
	Duration    int64  `json:"duration_ms"`
}

// MetricResult represents metric query results
type MetricResult struct {
	LatencyP50   float64 `json:"latency_p50"`
	LatencyP95   float64 `json:"latency_p95"`
	LatencyP99   float64 `json:"latency_p99"`
	ErrorRate    float64 `json:"error_rate"`
	RequestCount int64   `json:"request_count"`
}

// ProfileResult represents profiler query results
type ProfileResult struct {
	TopFunctions []FunctionProfile `json:"top_functions"`
}

// FunctionProfile represents a function profile
type FunctionProfile struct {
	FunctionName string  `json:"function_name"`
	FilePath     string  `json:"file_path"`
	LineNumber   int     `json:"line_number"`
	CPUPercent   float64 `json:"cpu_percent"`
}

// NewEngine creates a new correlation engine
func NewEngine(
	cfg *EngineConfig,
	logger *logger.Logger,
	detector *SpikeDetector,
	repository *storage.Repository,
) *Engine {
	ctx, cancel := context.WithCancel(context.Background())
	return &Engine{
		config:           cfg,
		logger:           logger,
		detector:         detector,
		repository:       repository,
		ctx:              ctx,
		cancel:           cancel,
		alertBuffer:      make([]SpikeAlert, 0),
		correlationCache: make(map[string]*RouteSelection),
	}
}

// SetSignozQuerier sets the Signoz querier
func (e *Engine) SetSignozQuerier(querier SignozQuerier) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.SignozQuery = querier
}

// SetProfilerQuerier sets the profiler querier
func (e *Engine) SetProfilerQuerier(querier ProfilerQuerier) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.ProfilerQuery = querier
}

// SetRouter sets the route selection router
func (e *Engine) SetRouter(router *Router) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.router = router
}

// SetAlertCallback sets the alert callback
func (e *Engine) SetAlertCallback(callback AlertCallback) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.alertCallback = callback
}

// Start starts the correlation engine
func (e *Engine) Start(namespaces, excludePatterns []string) error {
	e.mu.Lock()
	if e.running {
		e.mu.Unlock()
		return fmt.Errorf("engine already running")
	}
	e.running = true
	e.mu.Unlock()

	e.logger.Info("Starting correlation engine")

	// Start the main loop in a goroutine
	e.wg.Add(1)
	go e.runLoop(namespaces, excludePatterns)

	return nil
}

// Stop stops the correlation engine
func (e *Engine) Stop() {
	e.logger.Info("Stopping correlation engine")

	e.mu.Lock()
	e.running = false
	e.mu.Unlock()

	e.cancel()
	e.wg.Wait()

	e.logger.Info("Correlation engine stopped")
}

// runLoop is the main correlation loop
func (e *Engine) runLoop(namespaces, excludePatterns []string) {
	defer e.wg.Done()

	// Start baseline learning period
	baselineDuration := time.Duration(e.config.BaselineLearningMinutes) * time.Minute
	e.logger.Info("Starting baseline learning period for %v", baselineDuration)

	ticker := time.NewTicker(time.Duration(e.config.PollingIntervalSeconds) * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-e.ctx.Done():
			e.logger.Info("Correlation engine context cancelled")
			return
		case <-ticker.C:
			// Run detection cycle
			e.runDetectionCycle(namespaces, excludePatterns)
		}
	}
}

// runDetectionCycle runs a single detection cycle
func (e *Engine) runDetectionCycle(namespaces, excludePatterns []string) {
	// Detect spikes
	alerts, err := e.detector.DetectSpikes(e.ctx, nil, namespaces, excludePatterns)
	if err != nil {
		e.logger.Error("Failed to detect spikes: %v", err)
		return
	}

	// Process new alerts
	for _, alert := range alerts {
		e.processAlert(alert)
	}

	// Process buffered alerts (reconciliation)
	e.processBufferedAlerts()
}

// processAlert processes a new spike alert
func (e *Engine) processAlert(alert SpikeAlert) {
	e.mu.Lock()
	e.alertBuffer = append(e.alertBuffer, alert)
	e.mu.Unlock()

	e.logger.Info("Spike alert added to buffer: %s/%s (CPU: %.2f%%, RAM: %.2f%%)",
		alert.Namespace, alert.PodName, alert.CPUPercent, alert.RAMPercent)
}

// processBufferedAlerts processes alerts in the buffer
func (e *Engine) processBufferedAlerts() {
	e.mu.Lock()
	defer e.mu.Unlock()

	now := time.Now()
	processedAlerts := make([]SpikeAlert, 0, len(e.alertBuffer))

	for _, alert := range e.alertBuffer {
		if now.After(alert.AlertTime) {
			// Alert is ready for final processing
			e.triggerCorrelation(alert)
			processedAlerts = append(processedAlerts, alert)
		}
	}

	// Remove processed alerts from buffer
	if len(processedAlerts) > 0 {
		remaining := make([]SpikeAlert, 0)
		for _, alert := range e.alertBuffer {
			keep := true
			for _, processed := range processedAlerts {
				if alert.Timestamp == processed.Timestamp && alert.PodName == processed.PodName {
					keep = false
					break
				}
			}
			if keep {
				remaining = append(remaining, alert)
			}
		}
		e.alertBuffer = remaining
	}
}

// triggerCorrelation triggers Signoz and Profiler queries after buffer period
func (e *Engine) triggerCorrelation(alert SpikeAlert) {
	e.logger.Info("Triggering correlation for spike: %s/%s", alert.Namespace, alert.PodName)

	// Define time range for queries (5 minutes before alert)
	startTime := alert.Timestamp.Add(-5 * time.Minute)
	endTime := alert.Timestamp

	// Query Signoz for traces and metrics
	if e.SignozQuery != nil {
		go e.querySignoz(alert, startTime, endTime)
	}

	// Query profiler
	if e.ProfilerQuery != nil {
		go e.queryProfiler(alert, startTime, endTime)
	}

	// Query router for route selection (correlates spike with trace IDs)
	if e.router != nil {
		go e.queryRouter(alert, startTime, endTime)
	}

	// Save initial alert to database
	e.saveAlert(&alert)

	// Call alert callback
	if e.alertCallback != nil {
		go func() {
			if err := e.alertCallback(e.ctx, &alert); err != nil {
				e.logger.Error("Failed to execute alert callback: %v", err)
			}
		}()
	}
}

// querySignoz queries Signoz for traces and metrics
func (e *Engine) querySignoz(alert SpikeAlert, startTime, endTime time.Time) {
	e.mu.RLock()
	querier := e.SignozQuery
	e.mu.RUnlock()

	if querier == nil {
		return
	}

	// Query traces
	traces, err := querier.QueryTraces(alert.Namespace, alert.PodName, startTime, endTime)
	if err != nil {
		e.logger.Error("Failed to query Signoz traces: %v", err)
		return
	}

	// Query metrics
	metrics, err := querier.QueryMetrics(alert.Namespace, alert.PodName, startTime, endTime)
	if err != nil {
		e.logger.Error("Failed to query Signoz metrics: %v", err)
		return
	}

	e.logger.Info("Signoz query results - Traces: %d, Latency P99: %.2fms, Error rate: %.2f%%",
		len(traces.Traces), metrics.LatencyP99, metrics.ErrorRate)

	// Log trace details - route selection is handled by queryRouter
	e.logger.Debug("Signoz traces received for %s/%s: first trace route=%s",
		alert.Namespace, alert.PodName, traces.Traces[0].Route)
}

// queryProfiler queries the profiler and updates spike event with culprit function
func (e *Engine) queryProfiler(alert SpikeAlert, startTime, endTime time.Time) {
	e.mu.RLock()
	querier := e.ProfilerQuery
	e.mu.RUnlock()

	if querier == nil {
		return
	}

	e.logger.Info("Querying profiler for spike: %s/%s", alert.Namespace, alert.PodName)

	profile, err := querier.QueryProfile(alert.Namespace, alert.PodName, startTime, endTime)
	if err != nil {
		// Graceful degradation: log warning and continue
		e.logger.Warn("Profiler unavailable for %s/%s: %v (spike alert sent without profiler data)",
			alert.Namespace, alert.PodName, err)
		return
	}

	if profile == nil || len(profile.TopFunctions) == 0 {
		e.logger.Info("No profiler data found for %s/%s", alert.Namespace, alert.PodName)
		return
	}

	// Extract culprit function (highest CPU with file path)
	culpritFunc := e.extractCulpritFunction(profile.TopFunctions)

	e.logger.Info("Profiler query results - Top function: %s (%.2f%% CPU) at %s:%d",
		culpritFunc.FunctionName, culpritFunc.CPUPercent, culpritFunc.FilePath, culpritFunc.LineNumber)

	// Update spike event with profiler data
	e.updateAlertWithProfiler(&alert, culpritFunc)
}

// extractCulpritFunction extracts the most likely culprit function from profile
func (e *Engine) extractCulpritFunction(functions []FunctionProfile) FunctionProfile {
	// Find top function with a valid file path
	for _, f := range functions {
		if f.FilePath != "" && f.CPUPercent > 0 {
			return f
		}
	}

	// Fallback to top function
	if len(functions) > 0 {
		return functions[0]
	}

	return FunctionProfile{FunctionName: "unknown"}
}

// updateAlertWithProfiler updates the spike event with profiler culprit data
func (e *Engine) updateAlertWithProfiler(alert *SpikeAlert, culprit FunctionProfile) {
	if e.repository == nil {
		return
	}

	// Generate the spike event ID
	spikeID := fmt.Sprintf("%s-%s-%d", alert.Namespace, alert.PodName, alert.Timestamp.Unix())

	// Get existing event
	event, err := e.repository.GetSpikeEvent(e.ctx, spikeID)
	if err != nil {
		e.logger.Error("Failed to get spike event for profiler update: %v", err)
		return
	}

	if event == nil {
		e.logger.Debug("Spike event not found for profiler update: %s", spikeID)
		return
	}

	// Update culprit function fields
	event.CulpritFunction = &culprit.FunctionName
	event.CulpritFilePath = &culprit.FilePath

	// Update in database
	if err := e.repository.UpdateSpikeEvent(e.ctx, event); err != nil {
		e.logger.Error("Failed to update spike event with profiler data: %v", err)
		return
	}

	e.logger.Info("Updated spike event %s with profiler culprit: %s at %s:%d",
		spikeID, culprit.FunctionName, culprit.FilePath, culprit.LineNumber)
}

// queryRouter queries the router for route selection and trace correlation
func (e *Engine) queryRouter(alert SpikeAlert, startTime, endTime time.Time) {
	e.mu.RLock()
	router := e.router
	e.mu.RUnlock()

	if router == nil {
		return
	}

	// Select culprit route - correlates spike with trace route
	selection, err := router.SelectRoute(
		e.ctx,
		alert.Namespace,
		alert.PodName,
		alert.Timestamp,
		alert.CPUPercent,
		alert.RAMPercent,
	)

	if err != nil {
		e.logger.Error("Failed to select route: %v", err)
		return
	}

	// Cache the route selection for later database update
	spikeID := fmt.Sprintf("%s/%s/%d", alert.Namespace, alert.PodName, alert.Timestamp.Unix())
	e.mu.Lock()
	e.correlationCache[spikeID] = selection
	e.mu.Unlock()

	e.logger.Info("Route selection for spike %s: culprit=%s, trace_count=%d, trace_id=%s",
		spikeID, selection.CulpritRoute, selection.TraceCount, selection.TraceID)

	// Update the database with route and trace information
	e.updateAlertWithCorrelation(&alert, selection)
}

// saveAlert saves the alert to the database
func (e *Engine) saveAlert(alert *SpikeAlert) {
	if e.repository == nil {
		return
	}

	event := alert.ToStorageModel()
	if err := e.repository.CreateSpikeEvent(e.ctx, event); err != nil {
		e.logger.Error("Failed to save spike event: %v", err)
	}
}

// updateAlertWithCorrelation updates the alert with correlation data from trace analysis
func (e *Engine) updateAlertWithCorrelation(alert *SpikeAlert, selection *RouteSelection) {
	if e.repository == nil || selection == nil {
		return
	}

	if !selection.FoundRoutes {
		e.logger.Info("No routes found for spike: %s/%s", alert.Namespace, alert.PodName)
		return
	}

	// Build route name with job tag if applicable
	routeName := selection.CulpritRoute
	if selection.IsJobRoute {
		routeName = "[Suspected Job] " + selection.CulpritRoute
	}

	traceID := selection.TraceID

	e.logger.Info("Correlation complete for %s/%s - Route: %s, TraceID: %s, ActiveRoutes: %d, CPU+RAM: %.2f%%",
		alert.Namespace, alert.PodName, routeName, traceID, len(selection.ActiveRoutes), selection.CombinedResource)

	// Update the database with route and trace information
	e.updateSpikeEventRoute(alert, routeName, traceID, selection)
}

// updateAlertWithCorrelationSignoz updates alert with Signoz trace data (legacy interface)
func (e *Engine) updateAlertWithCorrelationSignoz(alert *SpikeAlert, traces *TraceResult, metrics *MetricResult) {
	if e.repository == nil || traces == nil {
		return
	}

	// Find the route with highest latency or most spans
	routeName := ""
	traceID := ""
	culpritFunc := ""

	if len(traces.Traces) > 0 {
		// Get route from first trace
		routeName = traces.Traces[0].Route
		traceID = traces.Traces[0].TraceID

		// Find the slowest span
		var maxDuration int64
		for _, trace := range traces.Traces {
			for _, span := range trace.Spans {
				if span.Duration > maxDuration {
					maxDuration = span.Duration
					culpritFunc = span.Operation
				}
			}
		}
	}

	e.logger.Info("Signoz correlation complete for %s/%s - Route: %s, TraceID: %s, Culprit: %s",
		alert.Namespace, alert.PodName, routeName, traceID, culpritFunc)

	// Log metrics
	if metrics != nil {
		e.logger.Info("Signoz metrics for %s/%s - P50: %.2fms, P99: %.2fms, ErrorRate: %.2f%%",
			alert.Namespace, alert.PodName, metrics.LatencyP50, metrics.LatencyP99, metrics.ErrorRate)
	}
}

// updateSpikeEventRoute updates the spike event with route and trace information
func (e *Engine) updateSpikeEventRoute(alert *SpikeAlert, routeName, traceID string, selection *RouteSelection) {
	if e.repository == nil {
		return
	}

	// Generate the spike event ID
	spikeID := fmt.Sprintf("%s-%s-%d", alert.Namespace, alert.PodName, alert.Timestamp.Unix())

	// Get existing event to preserve fields
	event, err := e.repository.GetSpikeEvent(e.ctx, spikeID)
	if err != nil {
		e.logger.Error("Failed to get spike event for update: %v", err)
		return
	}

	if event == nil {
		e.logger.Debug("Spike event not found for update: %s", spikeID)
		return
	}

	// Update route and trace fields
	event.RouteName = &routeName
	event.TraceID = &traceID

	// Update in database
	if err := e.repository.UpdateSpikeEvent(e.ctx, event); err != nil {
		e.logger.Error("Failed to update spike event with correlation: %v", err)
		return
	}

	e.logger.Info("Updated spike event %s with route: %s, traceID: %s", spikeID, routeName, traceID)
}

// GetAlertBuffer returns the current alert buffer
func (e *Engine) GetAlertBuffer() []SpikeAlert {
	e.mu.RLock()
	defer e.mu.RUnlock()

	buffer := make([]SpikeAlert, len(e.alertBuffer))
	copy(buffer, e.alertBuffer)
	return buffer
}

// IsRunning returns whether the engine is running
func (e *Engine) IsRunning() bool {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.running
}

// GetDetector returns the spike detector
func (e *Engine) GetDetector() *SpikeDetector {
	return e.detector
}

// ForceDetection forces a detection cycle (useful for testing)
func (e *Engine) ForceDetection(namespaces, excludePatterns []string) {
	e.runDetectionCycle(namespaces, excludePatterns)
}

// ClearAlertBuffer clears the alert buffer
func (e *Engine) ClearAlertBuffer() {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.alertBuffer = make([]SpikeAlert, 0)
}

// GetConfig returns the engine configuration
func (e *Engine) GetConfig() *EngineConfig {
	return e.config
}

// UpdateConfig updates the engine configuration
func (e *Engine) UpdateConfig(cfg *EngineConfig) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.config = cfg
	e.detector.config = &DetectorConfig{
		PollingIntervalSeconds:      cfg.PollingIntervalSeconds,
		ThresholdPercent:            cfg.ThresholdPercent,
		MovingAverageWindowMinutes:  cfg.MovingAverageWindowMinutes,
		BaselineLearningMinutes:     cfg.BaselineLearningMinutes,
		ReconciliationBufferMinutes: cfg.ReconciliationBufferMinutes,
		CooldownMinutes:             cfg.CooldownMinutes,
	}
}
