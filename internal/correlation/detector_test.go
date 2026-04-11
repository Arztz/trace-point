package correlation

import (
	"math"
	"os"
	"testing"
	"time"

	"github.com/trace-point/trace-point/internal/utils/logger"
)

// MockLogger creates a test logger
func MockLogger() *logger.Logger {
	return logger.New(os.Stdout, logger.DebugLevel)
}

func TestCalculateMovingAverage(t *testing.T) {
	tests := []struct {
		name    string
		data    []MetricDataPoint
		wantCPU float64
		wantRAM float64
	}{
		{
			name:    "empty data",
			data:    []MetricDataPoint{},
			wantCPU: 0,
			wantRAM: 0,
		},
		{
			name: "single data point",
			data: []MetricDataPoint{
				{CPUPercent: 50.0, RAMPercent: 60.0},
			},
			wantCPU: 50.0,
			wantRAM: 60.0,
		},
		{
			name: "multiple data points",
			data: []MetricDataPoint{
				{CPUPercent: 10.0, RAMPercent: 20.0},
				{CPUPercent: 20.0, RAMPercent: 30.0},
				{CPUPercent: 30.0, RAMPercent: 40.0},
			},
			wantCPU: 20.0,
			wantRAM: 30.0,
		},
		{
			name: "with high values",
			data: []MetricDataPoint{
				{CPUPercent: 90.0, RAMPercent: 95.0},
				{CPUPercent: 100.0, RAMPercent: 100.0},
			},
			wantCPU: 95.0,
			wantRAM: 97.5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotCPU, gotRAM := calculateMovingAverage(tt.data)
			if gotCPU != tt.wantCPU {
				t.Errorf("calculateMovingAverage() CPU = %v, want %v", gotCPU, tt.wantCPU)
			}
			if gotRAM != tt.wantRAM {
				t.Errorf("calculateMovingAverage() RAM = %v, want %v", gotRAM, tt.wantRAM)
			}
		})
	}
}

func TestExponentialMovingAverage(t *testing.T) {
	tests := []struct {
		name  string
		data  []MetricDataPoint
		alpha float64
		want  float64
	}{
		{
			name:  "empty data",
			data:  []MetricDataPoint{},
			alpha: 0.3,
			want:  0,
		},
		{
			name: "single point",
			data: []MetricDataPoint{
				{CPUPercent: 50.0},
			},
			alpha: 0.3,
			want:  50.0,
		},
		{
			name: "multiple points",
			data: []MetricDataPoint{
				{CPUPercent: 10.0},
				{CPUPercent: 20.0},
				{CPUPercent: 30.0},
			},
			alpha: 0.5,
			want:  22.5, // EMA = 0.5*30 + 0.5*(0.5*20 + 0.5*10) = 15 + 7.5 = 22.5
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ExponentialMovingAverage(tt.data, tt.alpha)
			if got != tt.want {
				t.Errorf("ExponentialMovingAverage() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestStandardDeviation(t *testing.T) {
	tests := []struct {
		name string
		data []MetricDataPoint
		want float64
	}{
		{
			name: "empty data",
			data: []MetricDataPoint{},
			want: 0,
		},
		{
			name: "single point",
			data: []MetricDataPoint{
				{CPUPercent: 50.0},
			},
			want: 0,
		},
		{
			name: "uniform values",
			data: []MetricDataPoint{
				{CPUPercent: 50.0},
				{CPUPercent: 50.0},
				{CPUPercent: 50.0},
			},
			want: 0,
		},
		{
			name: "varying values",
			data: []MetricDataPoint{
				{CPUPercent: 10.0},
				{CPUPercent: 20.0},
				{CPUPercent: 30.0},
			},
			want: 8.164965809, // sqrt(200/3) ≈ 8.16
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := StandardDeviation(tt.data)
			// Use approximate comparison for floating point
			diff := math.Abs(got - tt.want)
			if diff > 0.001 {
				t.Errorf("StandardDeviation() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSpikeDetectorKey(t *testing.T) {
	tests := []struct {
		namespace     string
		podName       string
		containerName string
		want          string
	}{
		{
			namespace:     "default",
			podName:       "my-pod-abc123",
			containerName: "nginx",
			want:          "default/my-pod-abc123/nginx",
		},
		{
			namespace:     "production",
			podName:       "api-server-xyz",
			containerName: "main",
			want:          "production/api-server-xyz/main",
		},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got := Key(tt.namespace, tt.podName, tt.containerName)
			if got != tt.want {
				t.Errorf("Key() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSpikeDetectorNew(t *testing.T) {
	cfg := DefaultDetectorConfig()
	log := MockLogger()

	detector := NewSpikeDetector(cfg, log)

	if detector == nil {
		t.Fatal("NewSpikeDetector() returned nil")
	}

	if detector.config == nil {
		t.Error("config is nil")
	}

	if detector.metricsHistory == nil {
		t.Error("metricsHistory is nil")
	}

	if detector.cooldowns == nil {
		t.Error("cooldowns is nil")
	}

	if detector.IsBaselineReady() {
		t.Error("detector should not be baseline ready on creation")
	}
}

func TestSpikeDetectorResetCooldowns(t *testing.T) {
	cfg := DefaultDetectorConfig()
	log := MockLogger()
	detector := NewSpikeDetector(cfg, log)

	// Manually set a cooldown
	key := "default/test-pod/nginx"
	detector.cooldowns[key] = time.Now().Add(10 * time.Minute)

	if _, exists := detector.cooldowns[key]; !exists {
		t.Error("cooldown was not set")
	}

	detector.ResetCooldowns()

	if _, exists := detector.cooldowns[key]; exists {
		t.Error("cooldown was not reset")
	}
}

func TestSpikeDetectorGetMovingAverage(t *testing.T) {
	cfg := DefaultDetectorConfig()
	log := MockLogger()
	detector := NewSpikeDetector(cfg, log)

	// Add some metrics directly to history
	detector.mu.Lock()
	detector.metricsHistory["default/test-pod/nginx"] = []MetricDataPoint{
		{Timestamp: time.Now(), CPUPercent: 10.0, RAMPercent: 20.0},
		{Timestamp: time.Now(), CPUPercent: 20.0, RAMPercent: 30.0},
		{Timestamp: time.Now(), CPUPercent: 30.0, RAMPercent: 40.0},
	}
	detector.mu.Unlock()

	cpuAvg, ramAvg := detector.GetMovingAverage("default", "test-pod", "nginx")

	if cpuAvg != 20.0 {
		t.Errorf("GetMovingAverage() CPU = %v, want 20.0", cpuAvg)
	}
	if ramAvg != 30.0 {
		t.Errorf("GetMovingAverage() RAM = %v, want 30.0", ramAvg)
	}
}

func TestSpikeDetectorGetMovingAverageEmpty(t *testing.T) {
	cfg := DefaultDetectorConfig()
	log := MockLogger()
	detector := NewSpikeDetector(cfg, log)

	cpuAvg, ramAvg := detector.GetMovingAverage("default", "nonexistent", "nginx")

	if cpuAvg != 0 {
		t.Errorf("GetMovingAverage() CPU = %v, want 0", cpuAvg)
	}
	if ramAvg != 0 {
		t.Errorf("GetMovingAverage() RAM = %v, want 0", ramAvg)
	}
}

func TestSpikeAlertToStorageModel(t *testing.T) {
	alert := SpikeAlert{
		Namespace:        "default",
		PodName:          "test-pod",
		ContainerName:    "nginx",
		CPUPercent:       85.0,
		RAMPercent:       70.0,
		MovingAverageCPU: 50.0,
		MovingAverageRAM: 45.0,
		ThresholdPercent: 50.0,
		DeviationPercent: 70.0,
		Timestamp:        time.Now(),
		AlertTime:        time.Now().Add(8 * time.Minute),
		Type:             CPU,
	}

	model := alert.ToStorageModel()

	if model == nil {
		t.Fatal("ToStorageModel() returned nil")
	}

	if model.Namespace != alert.Namespace {
		t.Errorf("Namespace = %v, want %v", model.Namespace, alert.Namespace)
	}

	if model.PodName != alert.PodName {
		t.Errorf("PodName = %v, want %v", model.PodName, alert.PodName)
	}

	if model.CPUUsagePercent != alert.CPUPercent {
		t.Errorf("CPUUsagePercent = %v, want %v", model.CPUUsagePercent, alert.CPUPercent)
	}
}

func TestDefaultDetectorConfig(t *testing.T) {
	cfg := DefaultDetectorConfig()

	if cfg.PollingIntervalSeconds != 30 {
		t.Errorf("PollingIntervalSeconds = %v, want 30", cfg.PollingIntervalSeconds)
	}

	if cfg.ThresholdPercent != 50.0 {
		t.Errorf("ThresholdPercent = %v, want 50.0", cfg.ThresholdPercent)
	}

	if cfg.MovingAverageWindowMinutes != 30 {
		t.Errorf("MovingAverageWindowMinutes = %v, want 30", cfg.MovingAverageWindowMinutes)
	}

	if cfg.BaselineLearningMinutes != 30 {
		t.Errorf("BaselineLearningMinutes = %v, want 30", cfg.BaselineLearningMinutes)
	}

	if cfg.ReconciliationBufferMinutes != 8 {
		t.Errorf("ReconciliationBufferMinutes = %v, want 8", cfg.ReconciliationBufferMinutes)
	}

	if cfg.CooldownMinutes != 15 {
		t.Errorf("CooldownMinutes = %v, want 15", cfg.CooldownMinutes)
	}
}

func TestSpikeTypes(t *testing.T) {
	if CPU != "cpu" {
		t.Errorf("CPU spike type = %v, want cpu", CPU)
	}

	if RAM != "ram" {
		t.Errorf("RAM spike type = %v, want ram", RAM)
	}
}
