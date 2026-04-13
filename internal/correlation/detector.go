package correlation

import (
	"context"
	"fmt"
	"math"
	"strings"
	"sync"
	"time"

	"github.com/trace-point/trace-point/internal/integration/prometheus"
	"github.com/trace-point/trace-point/internal/storage"
	"github.com/trace-point/trace-point/internal/utils/logger"
)

// DetectorConfig holds configuration for the spike detector
type DetectorConfig struct {
	PollingIntervalSeconds      int     `mapstructure:"polling_interval_seconds"`
	ThresholdPercent            float64 `mapstructure:"threshold_percent"`
	MovingAverageWindowMinutes  int     `mapstructure:"moving_average_window_minutes"`
	BaselineLearningMinutes     int     `mapstructure:"baseline_learning_period_minutes"`
	ReconciliationBufferMinutes int     `mapstructure:"reconciliation_buffer_minutes"`
	CooldownMinutes             int     `mapstructure:"cooldown_minutes"`
}

// DefaultDetectorConfig returns default configuration
func DefaultDetectorConfig() *DetectorConfig {
	return &DetectorConfig{
		PollingIntervalSeconds:      30,
		ThresholdPercent:            50.0,
		MovingAverageWindowMinutes:  30,
		BaselineLearningMinutes:     5, // Reduced for faster spike detection (was 30)
		ReconciliationBufferMinutes: 8,
		CooldownMinutes:             15,
	}
}

// maxHistorySize defines the maximum number of metric data points to retain per container
// This prevents unbounded memory growth: 120 samples = 1 hour at 30-second intervals
const maxHistorySize = 120

// SpikeDetector detects resource spikes using moving average algorithm
type SpikeDetector struct {
	config          *DetectorConfig
	logger          *logger.Logger
	metricsHistory  map[string][]MetricDataPoint // Key: namespace/pod/container
	mu              sync.RWMutex
	isBaselineReady bool
	baselineEndTime time.Time
	cooldowns       map[string]time.Time // Key: namespace/pod/container
}

// MetricDataPoint represents a single metric data point
type MetricDataPoint struct {
	Timestamp  time.Time
	CPUPercent float64
	RAMPercent float64
}

// NewSpikeDetector creates a new spike detector
func NewSpikeDetector(cfg *DetectorConfig, logger *logger.Logger) *SpikeDetector {
	detector := &SpikeDetector{
		config:         cfg,
		logger:         logger,
		metricsHistory: make(map[string][]MetricDataPoint),
		cooldowns:      make(map[string]time.Time),
	}

	// Initialize baseline end time
	detector.baselineEndTime = time.Now().Add(time.Duration(cfg.BaselineLearningMinutes) * time.Minute)

	return detector
}

// Key generates a unique key for a container
func Key(namespace, podName, containerName string) string {
	return fmt.Sprintf("%s/%s/%s", namespace, podName, containerName)
}

// UpdateMetrics updates the metrics history for a container
func (d *SpikeDetector) UpdateMetrics(ctx context.Context, client *prometheus.Client, namespaces, excludePatterns []string) error {
	metrics, err := client.FetchContainerMetrics(ctx, namespaces, excludePatterns)
	if err != nil {
		return fmt.Errorf("failed to fetch metrics: %w", err)
	}

	d.mu.Lock()
	defer d.mu.Unlock()

	now := time.Now()

	for _, m := range metrics {
		key := Key(m.Namespace, m.PodName, m.ContainerName)

		// Append to history
		point := MetricDataPoint{
			Timestamp:  now,
			CPUPercent: m.CPUPercent,
			RAMPercent: m.RAMPercent,
		}

		history := d.metricsHistory[key]
		history = append(history, point)

		// Prune old data points outside the moving average window
		windowDuration := time.Duration(d.config.MovingAverageWindowMinutes) * time.Minute
		cutoff := now.Add(-windowDuration)

		prunedHistory := make([]MetricDataPoint, 0)
		for _, p := range history {
			if p.Timestamp.After(cutoff) {
				prunedHistory = append(prunedHistory, p)
			}
		}
		d.metricsHistory[key] = prunedHistory

		// Enforce max history size to prevent unbounded memory growth
		if len(d.metricsHistory[key]) > maxHistorySize {
			d.metricsHistory[key] = d.metricsHistory[key][len(d.metricsHistory[key])-maxHistorySize:]
		}
	}

	// Check if baseline learning period is complete
	if !d.isBaselineReady {
		if now.After(d.baselineEndTime) {
			d.isBaselineReady = true
			d.logger.Info("Baseline learning period complete. Spike detection now active.")
		}
	}

	return nil
}

// DetectSpikes detects spikes in the current metrics
func (d *SpikeDetector) DetectSpikes(ctx context.Context, client *prometheus.Client, namespaces, excludePatterns []string) ([]SpikeAlert, error) {
	// First update metrics
	if err := d.UpdateMetrics(ctx, client, namespaces, excludePatterns); err != nil {
		return nil, err
	}

	d.mu.RLock()
	defer d.mu.RUnlock()

	// If baseline not ready, don't detect spikes yet
	if !d.isBaselineReady {
		return nil, nil
	}

	var alerts []SpikeAlert

	now := time.Now()

	for key, history := range d.metricsHistory {
		// Check cooldown
		if cooldownEnd, exists := d.cooldowns[key]; exists {
			if now.Before(cooldownEnd) {
				continue
			}
			// Cooldown expired, remove from map
			delete(d.cooldowns, key)
		}

		// Need at least 2 data points to calculate moving average
		if len(history) < 2 {
			continue
		}

		// Calculate moving average
		avgCPU, avgRAM := calculateMovingAverage(history)

		// Get latest metric
		latest := history[len(history)-1]

		// Check for CPU spike
		if avgCPU > 0 {
			cpuDeviation := (latest.CPUPercent - avgCPU) / avgCPU * 100
			if cpuDeviation > d.config.ThresholdPercent {
				// Check if we're in reconciliation buffer (before alerting)
				alertTime := latest.Timestamp.Add(time.Duration(d.config.ReconciliationBufferMinutes) * time.Minute)

				if now.Before(alertTime) {
					// Still in buffer period, continue
					d.logger.Debug("CPU spike detected for %s: %.2f%% (avg: %.2f%%) - in reconciliation buffer", key, latest.CPUPercent, avgCPU)
					continue
				}

				// Spike confirmed, create alert
				namespace, podName, containerName, err := parseKey(key)
				if err != nil {
					d.logger.Error("Failed to parse key %s: %v", key, err)
					continue
				}
				alert := SpikeAlert{
					Namespace:        namespace,
					PodName:          podName,
					ContainerName:    containerName,
					CPUPercent:       latest.CPUPercent,
					RAMPercent:       latest.RAMPercent,
					MovingAverageCPU: avgCPU,
					MovingAverageRAM: avgRAM,
					ThresholdPercent: d.config.ThresholdPercent,
					DeviationPercent: cpuDeviation,
					Timestamp:        latest.Timestamp,
					AlertTime:        alertTime,
					Type:             CPU,
				}
				alerts = append(alerts, alert)

				// Set cooldown
				d.cooldowns[key] = now.Add(time.Duration(d.config.CooldownMinutes) * time.Minute)

				d.logger.Info("CPU spike alert: %s at %.2f%% (avg: %.2f%%, threshold: %.2f%%)", key, latest.CPUPercent, avgCPU, d.config.ThresholdPercent)
			}
		}

		// Check for RAM spike
		if avgRAM > 0 {
			ramDeviation := (latest.RAMPercent - avgRAM) / avgRAM * 100
			if ramDeviation > d.config.ThresholdPercent {
				// Check if we're in reconciliation buffer
				alertTime := latest.Timestamp.Add(time.Duration(d.config.ReconciliationBufferMinutes) * time.Minute)

				if now.Before(alertTime) {
					// Still in buffer period, continue
					d.logger.Debug("RAM spike detected for %s: %.2f%% (avg: %.2f%%) - in reconciliation buffer", key, latest.RAMPercent, avgRAM)
					continue
				}

				// Spike confirmed, create alert
				namespace, podName, containerName, err := parseKey(key)
				if err != nil {
					d.logger.Error("Failed to parse key %s: %v", key, err)
					continue
				}
				alert := SpikeAlert{
					Namespace:        namespace,
					PodName:          podName,
					ContainerName:    containerName,
					CPUPercent:       latest.CPUPercent,
					RAMPercent:       latest.RAMPercent,
					MovingAverageCPU: avgCPU,
					MovingAverageRAM: avgRAM,
					ThresholdPercent: d.config.ThresholdPercent,
					DeviationPercent: ramDeviation,
					Timestamp:        latest.Timestamp,
					AlertTime:        alertTime,
					Type:             RAM,
				}
				alerts = append(alerts, alert)

				// Set cooldown
				d.cooldowns[key] = now.Add(time.Duration(d.config.CooldownMinutes) * time.Minute)

				d.logger.Info("RAM spike alert: %s at %.2f%% (avg: %.2f%%, threshold: %.2f%%)", key, latest.RAMPercent, avgRAM, d.config.ThresholdPercent)
			}
		}
	}

	return alerts, nil
}

// calculateMovingAverage calculates the moving average for CPU and RAM
func calculateMovingAverage(data []MetricDataPoint) (float64, float64) {
	if len(data) == 0 {
		return 0, 0
	}

	var cpuSum, ramSum float64
	for _, p := range data {
		cpuSum += p.CPUPercent
		ramSum += p.RAMPercent
	}

	return cpuSum / float64(len(data)), ramSum / float64(len(data))
}

// parseKey parses a key back into namespace, podName, containerName
// Returns an error if the key format is invalid
func parseKey(key string) (namespace, podName, containerName string, err error) {
	parts := strings.Split(key, "/")
	if len(parts) != 3 {
		return "", "", "", fmt.Errorf("invalid key format: expected 3 parts, got %d", len(parts))
	}
	return parts[0], parts[1], parts[2], nil
}

// ResetCooldowns resets all cooldowns (useful for testing)
func (d *SpikeDetector) ResetCooldowns() {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.cooldowns = make(map[string]time.Time)
}

// IsBaselineReady returns whether baseline learning is complete
func (d *SpikeDetector) IsBaselineReady() bool {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.isBaselineReady
}

// GetMetricsHistory returns the metrics history for debugging
func (d *SpikeDetector) GetMetricsHistory() map[string][]MetricDataPoint {
	d.mu.RLock()
	defer d.mu.RUnlock()

	// Return a copy
	history := make(map[string][]MetricDataPoint)
	for k, v := range d.metricsHistory {
		history[k] = append([]MetricDataPoint{}, v...)
	}
	return history
}

// GetMovingAverage returns the current moving average for a container
func (d *SpikeDetector) GetMovingAverage(namespace, podName, containerName string) (float64, float64) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	key := Key(namespace, podName, containerName)
	history, exists := d.metricsHistory[key]
	if !exists || len(history) == 0 {
		return 0, 0
	}

	return calculateMovingAverage(history)
}

// ExponentialMovingAverage calculates EMA for more recent values
func ExponentialMovingAverage(data []MetricDataPoint, alpha float64) float64 {
	if len(data) == 0 {
		return 0
	}

	ema := data[0].CPUPercent
	for i := 1; i < len(data); i++ {
		ema = alpha*data[i].CPUPercent + (1-alpha)*ema
	}
	return ema
}

// StandardDeviation calculates the standard deviation
func StandardDeviation(data []MetricDataPoint) float64 {
	if len(data) < 2 {
		return 0
	}

	avg, _ := calculateMovingAverage(data)

	var varianceSum float64
	for _, p := range data {
		diff := p.CPUPercent - avg
		varianceSum += diff * diff
	}

	return math.Sqrt(varianceSum / float64(len(data)))
}

// SpikeType represents the type of spike
type SpikeType string

const (
	CPU SpikeType = "cpu"
	RAM SpikeType = "ram"
)

// SpikeAlert represents a detected spike alert
type SpikeAlert struct {
	Namespace        string
	PodName          string
	ContainerName    string
	CPUPercent       float64
	RAMPercent       float64
	MovingAverageCPU float64
	MovingAverageRAM float64
	ThresholdPercent float64
	DeviationPercent float64
	Timestamp        time.Time
	AlertTime        time.Time
	Type             SpikeType
}

// ToStorageModel converts to storage model
func (s *SpikeAlert) ToStorageModel() *storage.SpikeEvent {
	// Generate unique ID using namespace-pod-timestamp to avoid duplicate key errors
	uniqueID := fmt.Sprintf("%s-%s-%d", s.Namespace, s.PodName, s.Timestamp.Unix())

	// Use the moving average that corresponds to the spike type
	var movingAvg float64
	if s.Type == RAM {
		movingAvg = s.MovingAverageRAM
	} else {
		movingAvg = s.MovingAverageCPU
	}

	return &storage.SpikeEvent{
		ID:                   uniqueID,
		Timestamp:            s.Timestamp,
		PodName:              s.PodName,
		Namespace:            s.Namespace,
		CPUUsagePercent:      s.CPUPercent,
		RAMUsagePercent:      s.RAMPercent,
		ThresholdPercent:     s.ThresholdPercent,
		MovingAveragePercent: movingAvg,
		CooldownEnd:          &s.AlertTime,
		CreatedAt:            time.Now(),
	}
}
