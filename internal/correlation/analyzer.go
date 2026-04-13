package correlation

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/trace-point/trace-point/internal/integration/prometheus"
	"github.com/trace-point/trace-point/internal/utils/logger"
)

// SpikeAnalysisRequest represents the input parameters for spike analysis
type SpikeAnalysisRequest struct {
	Start      time.Time `json:"start"`
	End        time.Time `json:"end"`
	Window     string    `json:"window"`     // 5m, 15m, 30m, 1h
	Namespace  string    `json:"namespace"`  // Filter by namespace
	Replicaset string    `json:"replicaset"` // Filter by replicaset name
	Threshold  float64   `json:"threshold"`  // default 50.0
	Limit      int       `json:"limit"`      // default 1000
	Offset     int       `json:"offset"`     // default 0
}

// HistoricalSpike represents a single spike event from historical analysis
type HistoricalSpike struct {
	ID               string    `json:"id"`
	Timestamp        time.Time `json:"timestamp"`
	ReplicasetName   string    `json:"replicaset_name"`
	PodName          string    `json:"pod_name"`
	Namespace        string    `json:"namespace"`
	ContainerName    string    `json:"container_name"`
	Type             string    `json:"type"` // cpu, ram, both
	CPUPercent       float64   `json:"cpu_percent"`
	RAMPercent       float64   `json:"ram_percent"`
	MovingAverageCPU float64   `json:"moving_average_cpu"`
	MovingAverageRAM float64   `json:"moving_average_ram"`
	ThresholdPercent float64   `json:"threshold_percent"`
	DeviationPercent float64   `json:"deviation_percent"`
	Severity         string    `json:"severity"` // low, medium, critical
}

// ReplicasetSpikeCount holds spike count for a replicaset
type ReplicasetSpikeCount struct {
	Name       string `json:"name"`
	SpikeCount int    `json:"spike_count"`
}

// AnalysisSummary holds aggregate statistics for the analysis
type AnalysisSummary struct {
	TotalSpikes         int                    `json:"total_spikes"`
	TimeRangeHours      float64                `json:"time_range_hours"`
	AnalyzedReplicasets int                    `json:"analyzed_replicasets"`
	SpikesByType        map[string]int         `json:"spikes_by_type"`
	TopReplicasets      []ReplicasetSpikeCount `json:"top_replicasets"`
}

// SpikeAnalysisResponse represents the full analysis response
type SpikeAnalysisResponse struct {
	Request    SpikeAnalysisRequest `json:"request"`
	Summary    AnalysisSummary      `json:"summary"`
	Spikes     []HistoricalSpike    `json:"spikes"`
	Pagination Pagination           `json:"pagination"`
}

// Pagination holds pagination info
type Pagination struct {
	Limit   int  `json:"limit"`
	Offset  int  `json:"offset"`
	HasMore bool `json:"has_more"`
}

// Analyzer performs historical spike analysis
type Analyzer struct {
	config *AnalyzerConfig
	log    *logger.Logger
}

// AnalyzerConfig holds configuration for the analyzer
type AnalyzerConfig struct {
	DefaultWindow    time.Duration
	DefaultThreshold float64
	MaxResults       int
	CooldownMinutes  int
	MaxTimeRangeDays int
}

// DefaultAnalyzerConfig returns default configuration
func DefaultAnalyzerConfig() *AnalyzerConfig {
	return &AnalyzerConfig{
		DefaultWindow:    30 * time.Minute,
		DefaultThreshold: 50.0,
		MaxResults:       1000,
		CooldownMinutes:  15,
		MaxTimeRangeDays: 30,
	}
}

// NewAnalyzer creates a new analyzer instance
func NewAnalyzer(cfg *AnalyzerConfig, log *logger.Logger) *Analyzer {
	if cfg == nil {
		cfg = DefaultAnalyzerConfig()
	}
	if log == nil {
		log = logger.Default()
	}
	return &Analyzer{
		config: cfg,
		log:    log,
	}
}

// AnalyzeSpikes performs historical spike analysis over a time range
func (a *Analyzer) AnalyzeSpikes(ctx context.Context, client *prometheus.Client, req *SpikeAnalysisRequest) (*SpikeAnalysisResponse, error) {
	startTime := time.Now()
	a.log.Debug("[Analyzer] AnalyzeSpikes START | start=%s | end=%s | window=%s | threshold=%.0f%%",
		req.Start.Format(time.RFC3339), req.End.Format(time.RFC3339), req.Window, req.Threshold)

	// Apply defaults
	if req.Window == "" {
		req.Window = "30m"
	}
	if req.Threshold == 0 {
		req.Threshold = a.config.DefaultThreshold
	}
	if req.Limit == 0 {
		req.Limit = a.config.MaxResults
	}

	// Parse window duration
	windowDuration, err := time.ParseDuration(req.Window)
	if err != nil {
		a.log.Warn("Invalid window duration: %v, using default 30m", err)
		windowDuration = 30 * time.Minute
	}

	// Calculate time range hours
	timeRangeHours := req.End.Sub(req.Start).Hours()

	// Determine step based on time range
	// For 24h: 30sec step (2880 points)
	// For 7d: 5min step (2016 points)
	// For 30d: 15min step (2880 points)
	step := 30 * time.Second
	if timeRangeHours > 24*6 {
		step = 5 * time.Minute
	} else if timeRangeHours > 24*2 {
		step = 2 * time.Minute
	}

	a.log.Debug("[Analyzer] Time range: %.1f hours, step: %s", timeRangeHours, step)

	// Build namespace filter
	namespaces := []string{}
	if req.Namespace != "" {
		namespaces = []string{req.Namespace}
	}

	// Fetch CPU metrics
	cpuQuery := prometheus.BuildCPUUtilizationQuery(namespaces, nil)
	a.log.Debug("[Analyzer] Querying CPU metrics: %s", cpuQuery)
	a.log.Debug("[Analyzer] Time range: start=%s end=%s step=%s", req.Start.Format(time.RFC3339), req.End.Format(time.RFC3339), step)
	cpuResults, err := client.QueryRange(ctx, cpuQuery, req.Start, req.End, step)
	if err != nil {
		a.log.Error("Failed to query CPU range: %v", err)
		// Return empty response instead of error
		cpuResults = []prometheus.Result{}
	}
	a.log.Debug("[Analyzer] CPU range results: %d", len(cpuResults))

	// Fetch RAM metrics
	ramQuery := prometheus.BuildRAMUtilizationQuery(namespaces, nil)
	a.log.Debug("[Analyzer] Querying RAM metrics: %s", ramQuery)
	ramResults, err := client.QueryRange(ctx, ramQuery, req.Start, req.End, step)
	if err != nil {
		a.log.Error("Failed to query RAM range: %v", err)
		ramResults = []prometheus.Result{}
	}
	a.log.Debug("[Analyzer] RAM range results: %d", len(ramResults))

	// Add debug for RAM key matching
	if a.log != nil && len(ramResults) > 0 {
		// Log first few RAM keys
		for i := 0; i < min(5, len(ramResults)); i++ {
			r := ramResults[i]
			podName := r.Metric["pod"]
			namespace := r.Metric["namespace"]
			containerName := r.Metric["container"]
			key := fmt.Sprintf("%s/%s/%s", namespace, podName, containerName)
			values, ok := r.Value.([][]interface{})
			if ok {
				a.log.Debug("[Analyzer] RAM key: %s has %d values", key, len(values))
			}
		}
	}

	// Process metrics into time series data
	timeSeries := processRangeResults(cpuResults, ramResults, req.Replicaset)

	a.log.Debug("[Analyzer] Processed %d time series", len(timeSeries))

	// Detect spikes for each time series
	allSpikes := detectSpikesInTimeSeries(timeSeries, windowDuration, req.Threshold, a.config.CooldownMinutes, a.log)

	a.log.Debug("[Analyzer] Detected %d total spikes before filtering", len(allSpikes))

	// Apply replicaset filter if specified
	if req.Replicaset != "" {
		filteredSpikes := make([]HistoricalSpike, 0)
		for _, spike := range allSpikes {
			if spike.ReplicasetName == req.Replicaset {
				filteredSpikes = append(filteredSpikes, spike)
			}
		}
		allSpikes = filteredSpikes
		a.log.Debug("[Analyzer] After replicaset filter: %d spikes", len(allSpikes))
	}

	// Calculate summary statistics
	summary := calculateSummary(allSpikes, timeRangeHours)

	// Debug: print some sample time series data
	if len(timeSeries) > 0 && len(allSpikes) == 0 {
		// Print first time series as sample
		for i, ts := range timeSeries[:5] {
			a.log.Debug("[Analyzer] Sample series %d: %s/%s - %d points",
				i, ts.Namespace, ts.PodName, len(ts.Points))
			if len(ts.Points) > 0 {
				a.log.Debug("[Analyzer]   First point CPU=%.1f, RAM=%.1f",
					ts.Points[0].CPUPercent, ts.Points[0].RAMPercent)
			}
		}
	}

	// Apply pagination
	totalSpikes := len(allSpikes)
	offset := req.Offset
	limit := req.Limit

	if offset > totalSpikes {
		offset = totalSpikes
	}

	endIdx := offset + limit
	if endIdx > totalSpikes {
		endIdx = totalSpikes
	}

	paginatedSpikes := allSpikes[offset:endIdx]

	response := &SpikeAnalysisResponse{
		Request: *req,
		Summary: summary,
		Spikes:  paginatedSpikes,
		Pagination: Pagination{
			Limit:   limit,
			Offset:  offset,
			HasMore: endIdx < totalSpikes,
		},
	}

	a.log.Debug("[Analyzer] AnalyzeSpikes SUCCESS | spikes=%d | duration=%v",
		len(paginatedSpikes), time.Since(startTime))

	return response, nil
}

// TimeSeriesPoint represents a single data point in a time series
type TimeSeriesPoint struct {
	Timestamp  time.Time
	CPUPercent float64
	RAMPercent float64
}

// TimeSeries holds metric data for a specific container
type TimeSeries struct {
	Namespace      string
	PodName        string
	ReplicasetName string
	ContainerName  string
	Points         []TimeSeriesPoint
}

// processRangeResults processes Prometheus range query results into time series
func processRangeResults(cpuResults, ramResults []prometheus.Result, replicasetFilter string) []TimeSeries {
	// Build CPU map: key -> time series
	cpuMap := make(map[string]*TimeSeries)

	for _, r := range cpuResults {
		podName := r.Metric["pod"]
		namespace := r.Metric["namespace"]
		containerName := r.Metric["container"]
		replicasetName := prometheus.ExtractReplicasetName(podName)

		// Apply replicaset filter if specified
		if replicasetFilter != "" && replicasetName != replicasetFilter {
			continue
		}

		key := fmt.Sprintf("%s/%s/%s", namespace, podName, containerName)

		ts, exists := cpuMap[key]
		if !exists {
			ts = &TimeSeries{
				Namespace:      namespace,
				PodName:        podName,
				ReplicasetName: replicasetName,
				ContainerName:  containerName,
				Points:         []TimeSeriesPoint{},
			}
			cpuMap[key] = ts
		}

		// Parse range values
		values, ok := r.Value.([][]interface{})
		if !ok {
			// Check if it's []interface{} (instant query format) - which is wrong for range
			if arr, isArr := r.Value.([]interface{}); isArr {
				fmt.Printf("[DEBUG] WARNING: Instant query format in range result: %s has %d elements\n", key, len(arr))
			}
			continue
		}

		fmt.Printf("[DEBUG] Key %s: got %d values (expected ~120 for 1h at 30s step)\n", key, len(values))
		if len(values) == 0 {
			continue
		}

		for i := range values {
			if len(values[i]) < 2 {
				continue
			}
			point := values[i]

			// Parse timestamp
			var tsUnix float64
			switch v := point[0].(type) {
			case float64:
				tsUnix = v
			case int:
				tsUnix = float64(v)
			case int64:
				tsUnix = float64(v)
			case string:
				var err error
				tsUnix, err = strconv.ParseFloat(v, 64)
				if err != nil {
					continue
				}
			default:
				fmt.Printf("[DEBUG] CPU timestamp type: %T (value: %v)\n", point[0], point[0])
				continue
			}

			// Parse value
			var cpuPercent float64
			switch v := point[1].(type) {
			case float64:
				cpuPercent = v
			case int:
				cpuPercent = float64(v)
			case int64:
				cpuPercent = float64(v)
			case string:
				var err error
				cpuPercent, err = strconv.ParseFloat(v, 64)
				if err != nil {
					continue
				}
			default:
				fmt.Printf("[DEBUG] CPU value type: %T (value: %v)\n", point[1], point[1])
				continue
			}

			timestamp := time.Unix(int64(tsUnix), 0)
			ts.Points = append(ts.Points, TimeSeriesPoint{
				Timestamp:  timestamp,
				CPUPercent: cpuPercent,
			})
		}
	}

	// Build RAM map
	ramMap := make(map[string][]float64) // key -> []RAMPercent

	// Debug: check CPU points before RAM processing
	cpuPointsCount := 0
	for _, ts := range cpuMap {
		cpuPointsCount += len(ts.Points)
	}
	fmt.Printf("[DEBUG] CPU points before RAM merge: total=%d\n", cpuPointsCount)

	for _, r := range ramResults {
		podName := r.Metric["pod"]
		namespace := r.Metric["namespace"]
		containerName := r.Metric["container"]
		key := fmt.Sprintf("%s/%s/%s", namespace, podName, containerName)

		values, ok := r.Value.([][]interface{})
		if !ok {
			continue
		}

		ramPoints := make([]float64, 0, len(values))
		for i := range values {
			if len(values[i]) < 2 {
				continue
			}
			point := values[i]

			var ramPercent float64
			switch v := point[1].(type) {
			case float64:
				ramPercent = v
			default:
				continue
			}
			ramPoints = append(ramPoints, ramPercent)
		}
		ramMap[key] = ramPoints
	}

	// Merge RAM into CPU time series using timestamp-based matching
	// Build a timestamp -> RAM value map for each time series
	result := make([]TimeSeries, 0, len(cpuMap))

	// Debug: count how many have RAM
	matchedCount := 0
	noRAMCount := 0

	for key, ts := range cpuMap {
		// Try to find matching RAM data - first try exact key match
		ramValues, hasRAM := ramMap[key]

		if !hasRAM {
			noRAMCount++
		}

		// If exact key match with same length, use index-based merge (fast path)
		if hasRAM && len(ramValues) == len(ts.Points) {
			for i := range ts.Points {
				ts.Points[i].RAMPercent = ramValues[i]
			}
			matchedCount++
		} else if hasRAM && len(ramValues) > 0 {
			// Timestamp-based merge for different lengths
			// Build timestamp -> RAM value map
			ramByTime := make(map[int64]float64, len(ramValues))
			// We need to reconstruct timestamps - assume same step as CPU
			if len(ts.Points) > 0 {
				baseTime := ts.Points[0].Timestamp.Unix()
				step := int64(30) // Default 30s step
				if len(ts.Points) > 1 {
					step = ts.Points[1].Timestamp.Unix() - ts.Points[0].Timestamp.Unix()
				}
				for i, rv := range ramValues {
					tsUnix := baseTime + int64(i)*step
					ramByTime[tsUnix] = rv
				}
				// Match by timestamp
				matched := 0
				for i := range ts.Points {
					tsUnix := ts.Points[i].Timestamp.Unix()
					if rv, ok := ramByTime[tsUnix]; ok {
						ts.Points[i].RAMPercent = rv
						matched++
					}
				}
				if matched > 0 {
					matchedCount++
				}
			}
		}
		// Sort by timestamp
		sort.Slice(ts.Points, func(i, j int) bool {
			return ts.Points[i].Timestamp.Before(ts.Points[j].Timestamp)
		})
		result = append(result, *ts)
	}

	// Debug output
	fmt.Printf("[DEBUG] Merge stats: total=%d, matchedRAM=%d, noRAM=%d\n", len(cpuMap), matchedCount, noRAMCount)

	// Debug: check first time series points count
	if len(result) > 0 {
		fmt.Printf("[DEBUG] First TS points count: %d, PodName: %s\n", len(result[0].Points), result[0].PodName)
		if len(result[0].Points) > 0 {
			fmt.Printf("[DEBUG] First point: CPU=%.2f, RAM=%.2f\n", result[0].Points[0].CPUPercent, result[0].Points[0].RAMPercent)
		}
	}

	return result
}

// detectSpikesInTimeSeries detects spikes in time series data
func detectSpikesInTimeSeries(timeSeries []TimeSeries, windowDuration time.Duration, threshold float64, cooldownMinutes int, log *logger.Logger) []HistoricalSpike {
	var allSpikes []HistoricalSpike
	var mu sync.Mutex

	// Calculate points per window based on window duration and step
	// Assumption: step is 30 seconds for typical queries
	pointsPerWindow := int(windowDuration.Seconds() / 30)
	if pointsPerWindow < 1 {
		pointsPerWindow = 1
	}

	if log != nil {
		log.Debug("[Detector] Starting spike detection: timeSeries=%d, pointsPerWindow=%d, threshold=%.0f%%",
			len(timeSeries), pointsPerWindow, threshold)
	}

	// Process each time series concurrently
	var wg sync.WaitGroup
	for i := range timeSeries {
		ts := timeSeries[i] // Capture in local variable
		wg.Add(1)
		// Pass the logger to detectSpikesForSeries for debugging
		go func() {
			defer wg.Done()
			if log != nil && ts.PodName != "" {
				log.Debug("[Detector] Processing %s: %d points (need %d)", ts.PodName, len(ts.Points), pointsPerWindow)
			}
			spikes := detectSpikesForSeries(ts, pointsPerWindow, threshold, cooldownMinutes, log)
			mu.Lock()
			allSpikes = append(allSpikes, spikes...)
			mu.Unlock()
		}()
	}
	wg.Wait()

	return allSpikes
}

// detectSpikesForSeries detects spikes for a single time series
func detectSpikesForSeries(ts TimeSeries, pointsPerWindow int, threshold float64, cooldownMinutes int, log *logger.Logger) []HistoricalSpike {
	var spikes []HistoricalSpike
	cooldownDuration := time.Duration(cooldownMinutes) * time.Minute

	lastCPUSpikeTime := time.Time{}
	lastRAMSpikeTime := time.Time{}

	points := ts.Points
	if len(points) < 2 {
		return spikes
	}

	// Debug: print point count and first few values for this series
	if log != nil && ts.PodName != "" {
		log.Debug("[Detector] Series %s: points=%d, pointsPerWindow=%d", ts.PodName, len(points), pointsPerWindow)
		if len(points) >= 5 {
			log.Debug("[Detector] First 5 CPU values: %.1f, %.1f, %.1f, %.1f, %.1f",
				points[0].CPUPercent, points[1].CPUPercent, points[2].CPUPercent, points[3].CPUPercent, points[4].CPUPercent)
		}
	}

	// For each point (starting from window size), calculate moving average from previous points
	for i := pointsPerWindow; i < len(points); i++ {
		current := points[i]

		// Define window: points from (i - pointsPerWindow) to (i - 1)
		windowStart := i - pointsPerWindow
		if windowStart < 0 {
			windowStart = 0
		}

		// Calculate moving average from window
		var cpuSum, ramSum float64
		windowSize := 0
		for j := windowStart; j < i; j++ {
			cpuSum += points[j].CPUPercent
			ramSum += points[j].RAMPercent
			windowSize++
		}

		if windowSize == 0 {
			continue
		}

		avgCPU := cpuSum / float64(windowSize)
		avgRAM := ramSum / float64(windowSize)

		// Detect CPU spike
		cpuSpike := false
		cpuDeviation := 0.0
		if avgCPU > 0 {
			cpuDeviation = (current.CPUPercent - avgCPU) / avgCPU * 100
			if cpuDeviation > threshold {
				// Check cooldown
				if lastCPUSpikeTime.IsZero() || current.Timestamp.Sub(lastCPUSpikeTime) > cooldownDuration {
					cpuSpike = true
					lastCPUSpikeTime = current.Timestamp
				}
			}
		}

		// Detect RAM spike
		ramSpike := false
		ramDeviation := 0.0
		if avgRAM > 0 {
			ramDeviation = (current.RAMPercent - avgRAM) / avgRAM * 100
			if ramDeviation > threshold {
				// Check cooldown
				if lastRAMSpikeTime.IsZero() || current.Timestamp.Sub(lastRAMSpikeTime) > cooldownDuration {
					ramSpike = true
					lastRAMSpikeTime = current.Timestamp
				}
			}
		}

		// Create spike events
		spikeType := ""
		if cpuSpike && ramSpike {
			spikeType = "both"
		} else if cpuSpike {
			spikeType = "cpu"
		} else if ramSpike {
			spikeType = "ram"
		}

		if spikeType != "" {
			// Calculate severity
			maxDeviation := cpuDeviation
			if ramDeviation > maxDeviation {
				maxDeviation = ramDeviation
			}
			severity := calculateSeverity(maxDeviation)

			// Determine which deviation to use based on spike type
			actualDeviation := cpuDeviation
			if spikeType == "ram" || spikeType == "both" && ramDeviation > cpuDeviation {
				actualDeviation = ramDeviation
			}

			spikeID := fmt.Sprintf("%s-%s-%d", ts.ReplicasetName, ts.PodName, current.Timestamp.Unix())

			spikes = append(spikes, HistoricalSpike{
				ID:               spikeID,
				Timestamp:        current.Timestamp,
				ReplicasetName:   ts.ReplicasetName,
				PodName:          ts.PodName,
				Namespace:        ts.Namespace,
				ContainerName:    ts.ContainerName,
				Type:             spikeType,
				CPUPercent:       current.CPUPercent,
				RAMPercent:       current.RAMPercent,
				MovingAverageCPU: avgCPU,
				MovingAverageRAM: avgRAM,
				ThresholdPercent: threshold,
				DeviationPercent: actualDeviation,
				Severity:         severity,
			})
		}
	}

	return spikes
}

// calculateSeverity determines the severity level based on deviation
func calculateSeverity(deviation float64) string {
	switch {
	case deviation > 200:
		return "critical"
	case deviation > 100:
		return "medium"
	case deviation > 50:
		return "low"
	default:
		return "normal"
	}
}

// calculateSummary calculates the analysis summary
func calculateSummary(spikes []HistoricalSpike, timeRangeHours float64) AnalysisSummary {
	summary := AnalysisSummary{
		TotalSpikes:    len(spikes),
		TimeRangeHours: timeRangeHours,
		SpikesByType:   make(map[string]int),
		TopReplicasets: []ReplicasetSpikeCount{},
	}

	// Count spikes by type
	replicasetCounts := make(map[string]int)
	uniqueReplicasets := make(map[string]bool)

	for _, spike := range spikes {
		summary.SpikesByType[spike.Type]++

		// Track unique replicasets
		uniqueReplicasets[spike.ReplicasetName] = true

		// Count spikes per replicaset
		replicasetCounts[spike.ReplicasetName]++
	}

	summary.AnalyzedReplicasets = len(uniqueReplicasets)

	// Build top replicasets list
	type rsCount struct {
		name  string
		count int
	}

	var rsList []rsCount
	for name, count := range replicasetCounts {
		rsList = append(rsList, rsCount{name: name, count: count})
	}

	// Sort by count descending
	sort.Slice(rsList, func(i, j int) bool {
		return rsList[i].count > rsList[j].count
	})

	// Take top 10
	if len(rsList) > 10 {
		rsList = rsList[:10]
	}

	for _, rs := range rsList {
		summary.TopReplicasets = append(summary.TopReplicasets, ReplicasetSpikeCount{
			Name:       rs.name,
			SpikeCount: rs.count,
		})
	}

	return summary
}

// ParseTime parses a time string that can be either RFC3339 or relative (24h, 7d, etc.)
func ParseTime(input string) (time.Time, error) {
	input = strings.TrimSpace(input)

	// Handle "now"
	if strings.ToLower(input) == "now" {
		return time.Now(), nil
	}

	// Try relative formats (24h, 7d, 30m, etc.)
	// Note: Go's ParseDuration doesn't support "d" (days), only "h", "m", "s"
	// So we need to convert "Xd" to "X*24h"
	if strings.HasSuffix(input, "d") {
		// Handle "7d", "30d", etc.
		daysStr := strings.TrimSuffix(input, "d")
		var days int
		if _, err := fmt.Sscanf(daysStr, "%d", &days); err == nil {
			return time.Now().Add(-time.Duration(days) * 24 * time.Hour), nil
		}
	}

	if strings.HasSuffix(input, "h") || strings.HasSuffix(input, "m") {
		duration, err := time.ParseDuration(input)
		if err == nil {
			return time.Now().Add(-duration), nil
		}
	}

	// Try RFC3339 format
	return time.Parse(time.RFC3339, input)
}

// ParseWindowSize parses window size string to duration
func ParseWindowSize(window string) time.Duration {
	switch window {
	case "5m":
		return 5 * time.Minute
	case "15m":
		return 15 * time.Minute
	case "30m":
		return 30 * time.Minute
	case "1h":
		return 1 * time.Hour
	default:
		return 30 * time.Minute
	}
}
