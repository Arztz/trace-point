package profiler

import (
	"context"
	"fmt"
	"time"

	"github.com/trace-point/trace-point/internal/utils/logger"
)

// QueryConfig holds configuration for profiler queries
type QueryConfig struct {
	WindowMinutes   int    // Time window around spike to query
	MaxFunctions    int    // Maximum number of functions to return
	IncludeFilePath bool   // Include file path in results
	ProfileType     string // Profile type: cpu, memory, wall
	DefaultProfile  string // Default profile type
}

// DefaultQueryConfig returns default query configuration
func DefaultQueryConfig() *QueryConfig {
	return &QueryConfig{
		WindowMinutes:   5,
		MaxFunctions:    10,
		IncludeFilePath: true,
		ProfileType:     "cpu",
		DefaultProfile:  "cpu",
	}
}

// QueryBuilder provides query building utilities for the profiler
type QueryBuilder struct {
	logger *logger.Logger
}

// NewQueryBuilder creates a new query builder
func NewQueryBuilder(logger *logger.Logger) *QueryBuilder {
	return &QueryBuilder{logger: logger}
}

// BuildTimeRangeQuery builds a query for a specific time range
// Returns the query parameters for the profiler API
func (qb *QueryBuilder) BuildTimeRangeQuery(serviceName string, spikeTime time.Time, windowMinutes int) (startTime, endTime time.Time, query string) {
	startTime = spikeTime.Add(-time.Duration(windowMinutes) * time.Minute)
	endTime = spikeTime

	query = fmt.Sprintf("CPU{service_name='%s'}", serviceName)

	qb.logger.Debug("Built profiler query for %s: %s to %s",
		serviceName, startTime.Format(time.RFC3339), endTime.Format(time.RFC3339))

	return startTime, endTime, query
}

// ExtractTopFunctions extracts the top CPU-consuming functions from profile results
// Returns functions sorted by CPU percentage (highest first)
func (qb *QueryBuilder) ExtractTopFunctions(result *ProfileResult, maxFunctions int) []FunctionProfile {
	if result == nil || len(result.TopFunctions) == 0 {
		qb.logger.Debug("No profile results to extract functions from")
		return nil
	}

	if maxFunctions <= 0 {
		maxFunctions = 10
	}

	// Limit to max functions
	topFuncs := result.TopFunctions
	if len(topFuncs) > maxFunctions {
		topFuncs = topFuncs[:maxFunctions]
	}

	qb.logger.Debug("Extracted %d top functions from profile", len(topFuncs))

	return topFuncs
}

// ExtractCulpritFunction extracts the most likely culprit function from profile results
// Criteria: highest CPU percentage that has a valid file path
func (qb *QueryBuilder) ExtractCulpritFunction(result *ProfileResult) *FunctionProfile {
	if result == nil || len(result.TopFunctions) == 0 {
		qb.logger.Debug("No profile results to extract culprit")
		return nil
	}

	// Find the top function with a valid file path
	// This helps identify actual code locations for debugging
	for _, funcProfile := range result.TopFunctions {
		if funcProfile.FilePath != "" && funcProfile.CPUPercent > 0 {
			qb.logger.Info("Identified culprit function: %s (%.2f%% CPU) at %s:%d",
				funcProfile.FunctionName, funcProfile.CPUPercent, funcProfile.FilePath, funcProfile.LineNumber)
			return &funcProfile
		}
	}

	// Fallback to the top function even without file path
	topFunc := result.TopFunctions[0]
	qb.logger.Info("Identified culprit function (no file path): %s (%.2f%% CPU)",
		topFunc.FunctionName, topFunc.CPUPercent)

	return &topFunc
}

// ExtractFunctionPaths extracts all function file paths from profile results
// Returns a map of function names to their file paths
func (qb *QueryBuilder) ExtractFunctionPaths(result *ProfileResult) map[string]string {
	pathMap := make(map[string]string)

	if result == nil {
		return pathMap
	}

	for _, funcProfile := range result.TopFunctions {
		if funcProfile.FilePath != "" {
			pathMap[funcProfile.FunctionName] = fmt.Sprintf("%s:%d", funcProfile.FilePath, funcProfile.LineNumber)
		}
	}

	return pathMap
}

// QueryService provides high-level profiler query service
type QueryService struct {
	client  *Client
	builder *QueryBuilder
	logger  *logger.Logger
}

// NewQueryService creates a new profiler query service
func NewQueryService(client *Client, logger *logger.Logger) *QueryService {
	return &QueryService{
		client:  client,
		builder: NewQueryBuilder(logger),
		logger:  logger,
	}
}

// QuerySpikeProfile queries the profiler for a spike event time range
// Returns profile results for correlation
func (qs *QueryService) QuerySpikeProfile(
	ctx context.Context,
	namespace, podName string,
	spikeTime time.Time,
	windowMinutes int,
) (*ProfileResult, error) {

	if windowMinutes <= 0 {
		windowMinutes = 5 // Default 5-minute window
	}

	// Calculate time range around spike
	startTime := spikeTime.Add(-time.Duration(windowMinutes) * time.Minute)
	endTime := spikeTime

	qs.logger.Info("Querying profiler for spike at %s in %s/%s (window: %d minutes)",
		spikeTime.Format(time.RFC3339), namespace, podName, windowMinutes)

	// Execute profiler query
	result, err := qs.client.QueryProfile(ctx, namespace, podName, startTime, endTime)
	if err != nil {
		qs.logger.Warn("Profiler query failed for %s/%s: %v (continuing without profiler data)",
			namespace, podName, err)
		return nil, err
	}

	return result, nil
}

// QueryForCorrelation queries profiler and extracts culprit function for spike correlation
// This is the main entry point for the correlation engine
func (qs *QueryService) QueryForCorrelation(
	ctx context.Context,
	namespace, podName string,
	spikeTime time.Time,
	cpuPercent, ramPercent float64,
) (*CorrelationResult, error) {

	result, err := qs.QuerySpikeProfile(ctx, namespace, podName, spikeTime, 0)
	if err != nil {
		// Graceful degradation: return empty result
		qs.logger.Info("Correlation: profiler unavailable, skipping profiler correlation")
		return &CorrelationResult{
			ProfilerAvailable: false,
		}, nil
	}

	// Extract culprit function
	culprit := qs.builder.ExtractCulpritFunction(result)
	if culprit == nil {
		return &CorrelationResult{
			ProfilerAvailable: true,
			CulpritFound:      false,
			ProfileResult:     result,
		}, nil
	}

	correlation := &CorrelationResult{
		ProfilerAvailable: true,
		CulpritFound:      true,
		CulpritFunction:   culprit.FunctionName,
		CulpritFilePath:   culprit.FilePath,
		CulpritLine:       culprit.LineNumber,
		CulpritCPU:        culprit.CPUPercent,
		ProfileResult:     result,
	}

	qs.logger.Info("Profiler correlation complete - Culprit: %s at %s:%d (%.2f%% CPU)",
		culprit.FunctionName, culprit.FilePath, culprit.LineNumber, culprit.CPUPercent)

	return correlation, nil
}

// CorrelationResult holds profiler correlation results
type CorrelationResult struct {
	ProfilerAvailable bool           // Whether profiler was available
	CulpritFound      bool           // Whether a culprit function was found
	CulpritFunction   string         // Name of the culprit function (hot path)
	CulpritFilePath   string         // File path of culprit function
	CulpritLine       int            // Line number of culprit function
	CulpritCPU        float64        // CPU percentage of culprit
	ProfileResult     *ProfileResult // Full profile results
}

// FormatCulprit formats the culprit information for display
func (cr *CorrelationResult) FormatCulprit() string {
	if !cr.CulpritFound {
		return "No culprit function identified"
	}

	location := cr.CulpritFunction
	if cr.CulpritFilePath != "" {
		location = fmt.Sprintf("%s:%d", cr.CulpritFilePath, cr.CulpritLine)
	}

	return fmt.Sprintf("%s (%.2f%% CPU) at %s",
		cr.CulpritFunction, cr.CulpritCPU, location)
}

// HasProfilerData checks if profiler data is available
func (cr *CorrelationResult) HasProfilerData() bool {
	return cr.ProfilerAvailable && cr.CulpritFound
}

// ServiceQuerier interface for the correlation engine
// Allows injecting different profiler implementations
type ServiceQuerier interface {
	QueryProfile(ctx context.Context, namespace, podName string, startTime, endTime time.Time) (*ProfileResult, error)
}

// NewServiceQuerier creates a service querier from a client
func NewServiceQuerier(client *Client) ServiceQuerier {
	return &ServiceQuerierAdapter{client: client}
}

// ServiceQuerierAdapter wraps Client to implement ServiceQuerier
type ServiceQuerierAdapter struct {
	client *Client
}

// QueryProfile delegates to client
func (s *ServiceQuerierAdapter) QueryProfile(ctx context.Context, namespace, podName string, startTime, endTime time.Time) (*ProfileResult, error) {
	return s.client.QueryProfile(ctx, namespace, podName, startTime, endTime)
}
