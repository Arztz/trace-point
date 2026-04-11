package config

import (
	"fmt"
	"time"

	"github.com/spf13/viper"
)

// Config holds all configuration for the application
type Config struct {
	App                AppConfig        `mapstructure:"app"`
	Prometheus         PrometheusConfig `mapstructure:"prometheus"`
	Signoz             SignozConfig     `mapstructure:"signoz"`
	GCloud             GCloudConfig     `mapstructure:"gcloud"`
	Detection          DetectionConfig  `mapstructure:"detection"`
	Discord            DiscordConfig    `mapstructure:"discord"`
	Database           DatabaseConfig   `mapstructure:"database"`
	Namespaces         []string         `mapstructure:"namespaces"`
	PodExcludePatterns []string         `mapstructure:"pod_exclude_patterns"`
	Profiler           ProfilerConfig   `mapstructure:"profiler"`
}

// AppConfig holds application-level settings
type AppConfig struct {
	Host string `mapstructure:"host"`
	Port int    `mapstructure:"port"`
	Mode string `mapstructure:"mode"`
}

// PrometheusConfig holds Prometheus integration settings
type PrometheusConfig struct {
	URL            string        `mapstructure:"url"`
	Timeout        time.Duration `mapstructure:"timeout"`
	ScrapeInterval time.Duration `mapstructure:"scrape_interval"`
	QueryPath      string        `mapstructure:"query_path"`
}

// SignozConfig holds SigNoz integration settings
type SignozConfig struct {
	URL       string        `mapstructure:"url"`
	OTLPHTTP  string        `mapstructure:"otlp_http"`
	Timeout   time.Duration `mapstructure:"timeout"`
	QueryPath string        `mapstructure:"query_path"`
}

// GCloudConfig holds Google Cloud integration settings
type GCloudConfig struct {
	ProjectID         string `mapstructure:"project_id"`
	Region            string `mapstructure:"region"`
	ClusterName       string `mapstructure:"cluster_name"`
	ServiceAccountKey string `mapstructure:"service_account_key"`
	UseGKE            bool   `mapstructure:"use_gke"`
}

// DetectionConfig holds anomaly detection settings
type DetectionConfig struct {
	CPUThreshold       int           `mapstructure:"cpu_threshold"`
	MemoryThreshold    int           `mapstructure:"memory_threshold"`
	LatencyThreshold   int           `mapstructure:"latency_threshold"`
	ErrorRateThreshold int           `mapstructure:"error_rate_threshold"`
	WindowSize         time.Duration `mapstructure:"window_size"`
	MinSamples         int           `mapstructure:"min_samples"`
}

// DiscordConfig holds Discord notification settings
type DiscordConfig struct {
	Enabled     bool     `mapstructure:"enabled"`
	WebhookURL  string   `mapstructure:"webhook_url"`
	MentionUser string   `mapstructure:"mention_user"`
	MentionRole string   `mapstructure:"mention_role"`
	Alerts      []string `mapstructure:"alerts"`
}

// DatabaseConfig holds database settings
type DatabaseConfig struct {
	Type            string `mapstructure:"type"`
	Path            string `mapstructure:"path"`
	MaxConnections  int    `mapstructure:"max_connections"`
	IdleConnections int    `mapstructure:"idle_connections"`
}

// ProfilerConfig holds profiler integration settings
type ProfilerConfig struct {
	Enabled      bool          `mapstructure:"enabled"`
	PyroscopeURL string        `mapstructure:"pyroscope_url"`
	Interval     time.Duration `mapstructure:"interval"`
}

// Load loads configuration from config file
func Load(configPath string) (*Config, error) {
	if configPath == "" {
		configPath = "configs/config.yaml"
	}

	viper.SetConfigFile(configPath)
	viper.SetConfigType("yaml")

	// Set default values
	viper.SetDefault("app.host", "0.0.0.0")
	viper.SetDefault("app.port", 8080)
	viper.SetDefault("app.mode", "debug")
	viper.SetDefault("database.type", "sqlite")
	viper.SetDefault("database.path", "./data/trace-point.db")
	viper.SetDefault("database.max_connections", 25)
	viper.SetDefault("database.idle_connections", 10)
	viper.SetDefault("detection.cpu_threshold", 80)
	viper.SetDefault("detection.memory_threshold", 85)
	viper.SetDefault("detection.latency_threshold", 1000)
	viper.SetDefault("detection.error_rate_threshold", 5)
	viper.SetDefault("detection.window_size", "5m")
	viper.SetDefault("detection.min_samples", 3)
	viper.SetDefault("prometheus.timeout", "30s")
	viper.SetDefault("prometheus.scrape_interval", "15s")
	viper.SetDefault("signoz.timeout", "30s")

	// Load config
	if err := viper.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var cfg Config
	if err := viper.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	return &cfg, nil
}

// GetAddr returns the server address in host:port format
func (c *Config) GetAddr() string {
	return fmt.Sprintf("%s:%d", c.App.Host, c.App.Port)
}
