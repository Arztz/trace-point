package storage

import (
	"time"
)

// SpikeEvent represents a resource spike event in the database.
type SpikeEvent struct {
	ID                   string     `json:"id"`
	Timestamp            time.Time  `json:"timestamp"`
	PodName              string     `json:"pod_name"`
	Namespace            string     `json:"namespace"`
	CPUUsagePercent      float64    `json:"cpu_usage_percent"`
	CPULimitPercent      float64    `json:"cpu_limit_percent"`
	RAMUsagePercent      float64    `json:"ram_usage_percent"`
	RAMLimitPercent      float64    `json:"ram_limit_percent"`
	ThresholdPercent     float64    `json:"threshold_percent"`
	MovingAveragePercent float64    `json:"moving_average_percent"`
	RouteName            *string    `json:"route_name"`
	TraceID              *string    `json:"trace_id"`
	CulpritFunction      *string    `json:"culprit_function"`
	CulpritFilePath      *string    `json:"culprit_file_path"`
	AlertSent            bool       `json:"alert_sent"`
	CooldownEnd          *time.Time `json:"cooldown_end"`
	CreatedAt            time.Time  `json:"created_at"`
}

// Config represents application configuration stored in the database.
type Config struct {
	ID        string    `json:"id"`
	Key       string    `json:"key"`
	Value     string    `json:"value"`
	UpdatedAt time.Time `json:"updated_at"`
}

// TimelineMetric represents continuous CPU/RAM metrics for timeline visualization.
type TimelineMetric struct {
	ID         int64     `json:"id,omitempty"`
	Timestamp  time.Time `json:"timestamp"`
	PodName    string    `json:"pod_name"`
	Namespace  string    `json:"namespace"`
	CPUPercent float64   `json:"cpu_percent"`
	RAMPercent float64   `json:"ram_percent"`
	CreatedAt  time.Time `json:"created_at,omitempty"`
}

// SpikeMarker represents a spike event for timeline overlay.
type SpikeMarker struct {
	Timestamp time.Time `json:"timestamp"`
	PodName   string    `json:"pod_name"`
	Namespace string    `json:"namespace"`
	CPUSpike  bool      `json:"cpu_spike"`
	RAMSpike  bool      `json:"ram_spike"`
	RouteName string    `json:"route_name,omitempty"`
	TraceID   string    `json:"trace_id,omitempty"`
}
