package config

import (
	"testing"
)

func TestGetAddr(t *testing.T) {
	cfg := &Config{
		App: AppConfig{
			Host: "0.0.0.0",
			Port: 8080,
		},
	}

	addr := cfg.GetAddr()
	if addr != "0.0.0.0:8080" {
		t.Errorf("GetAddr() = %v, want 0.0.0.0:8080", addr)
	}
}

func TestGetAddrDifferentPort(t *testing.T) {
	cfg := &Config{
		App: AppConfig{
			Host: "localhost",
			Port: 9090,
		},
	}

	addr := cfg.GetAddr()
	if addr != "localhost:9090" {
		t.Errorf("GetAddr() = %v, want localhost:9090", addr)
	}
}

func TestDatabaseConfig(t *testing.T) {
	cfg := DatabaseConfig{
		Type:            "sqlite",
		Path:            "./data/test.db",
		MaxConnections:  25,
		IdleConnections: 10,
	}

	if cfg.Type != "sqlite" {
		t.Errorf("Type = %v, want sqlite", cfg.Type)
	}

	if cfg.Path != "./data/test.db" {
		t.Errorf("Path = %v, want ./data/test.db", cfg.Path)
	}

	if cfg.MaxConnections != 25 {
		t.Errorf("MaxConnections = %v, want 25", cfg.MaxConnections)
	}
}

func TestAppConfig(t *testing.T) {
	cfg := AppConfig{
		Host: "0.0.0.0",
		Port: 8080,
		Mode: "debug",
	}

	if cfg.Host != "0.0.0.0" {
		t.Errorf("Host = %v, want 0.0.0.0", cfg.Host)
	}

	if cfg.Port != 8080 {
		t.Errorf("Port = %v, want 8080", cfg.Port)
	}

	if cfg.Mode != "debug" {
		t.Errorf("Mode = %v, want debug", cfg.Mode)
	}
}

func TestConfigPrometheus(t *testing.T) {
	cfg := PrometheusConfig{
		URL:            "http://localhost:9090",
		Timeout:        30,
		ScrapeInterval: 15,
		QueryPath:      "/api/v1/query",
	}

	if cfg.URL != "http://localhost:9090" {
		t.Errorf("URL = %v, want http://localhost:9090", cfg.URL)
	}
}

func TestConfigDiscord(t *testing.T) {
	cfg := DiscordConfig{
		Enabled:     false,
		WebhookURL:  "",
		MentionUser: "",
		MentionRole: "",
	}

	if cfg.Enabled {
		t.Error("Discord.Enabled should be false")
	}
}

func TestConfigDetection(t *testing.T) {
	cfg := DetectionConfig{
		CPUThreshold:       80,
		MemoryThreshold:    85,
		LatencyThreshold:   1000,
		ErrorRateThreshold: 5,
		MinSamples:         3,
	}

	if cfg.CPUThreshold != 80 {
		t.Errorf("CPUThreshold = %v, want 80", cfg.CPUThreshold)
	}

	if cfg.MemoryThreshold != 85 {
		t.Errorf("MemoryThreshold = %v, want 85", cfg.MemoryThreshold)
	}
}
