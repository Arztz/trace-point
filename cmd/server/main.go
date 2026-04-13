package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/trace-point/trace-point/internal/config"
	"github.com/trace-point/trace-point/internal/correlation"
	"github.com/trace-point/trace-point/internal/integration/discord"
	"github.com/trace-point/trace-point/internal/integration/profiler"
	"github.com/trace-point/trace-point/internal/integration/prometheus"
	"github.com/trace-point/trace-point/internal/integration/signoz"
	"github.com/trace-point/trace-point/internal/server/handlers"
	"github.com/trace-point/trace-point/internal/storage"
	"github.com/trace-point/trace-point/internal/utils/logger"
)

func main() {
	// Load configuration
	cfg, err := config.Load("configs/config.yaml")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load config: %v\n", err)
		os.Exit(1)
	}

	// Set log level based on config
	logLevel := logger.ParseLevel(cfg.App.Mode)
	logger.SetLevel(logLevel)

	logger.Info("Starting Trace-Point server...")

	// Initialize database
	db, err := storage.NewDatabase(cfg.Database.Path, 7)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize database: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()

	// Create repository
	repo := storage.NewRepository(db)

	// Create Chi router
	router := chi.NewRouter()

	// Add middleware
	router.Use(middleware.RequestID)
	router.Use(middleware.RealIP)
	router.Use(middleware.Logger)
	router.Use(middleware.Recoverer)

	// Health check endpoint
	router.Get("/health", handleHealth)

	// Metrics endpoint
	router.Handle("/metrics", promhttp.Handler())

	// Initialize Discord client if configured
	var discordClient *discord.Client
	if cfg.Discord.Enabled && cfg.Discord.WebhookURL != "" {
		discordClient = discord.NewClient(
			cfg.Discord.WebhookURL,
			cfg.Discord.MentionUser,
			cfg.Discord.MentionRole,
		)
		logger.Info("Discord notifications enabled")
	}

	// Initialize Prometheus client
	logger.Info("Initializing Prometheus client...")
	prometheusClient := prometheus.NewClient(&cfg.Prometheus, logger.Default())

	// Initialize Signoz client if enabled
	var signozClient *signoz.Client
	if cfg.Signoz.Enabled && cfg.Signoz.URL != "" {
		signozClient = signoz.NewClient(&cfg.Signoz, &cfg.GCloud, logger.Default())
		logger.Info("SigNoz client initialized: %s", cfg.Signoz.URL)
	}

	// Initialize Profiler client if enabled
	var profilerClient *profiler.Client
	if cfg.Profiler.Enabled && cfg.Profiler.PyroscopeURL != "" {
		profilerClient = profiler.NewClient(&cfg.Profiler, &cfg.GCloud, logger.Default())
		logger.Info("Profiler client initialized: %s", cfg.Profiler.PyroscopeURL)
	}

	// Set global clients for handlers
	handlers.SetClients(signozClient, profilerClient)

	// Test Prometheus connection
	testCtx, testCancel := context.WithTimeout(context.Background(), 10*time.Second)
	var connected bool
	_, queryErr := prometheusClient.Query(testCtx, "up", time.Now())
	if queryErr != nil {
		logger.Warn("Failed to connect to Prometheus: %v (spike detection will be disabled)", queryErr)
	} else {
		connected = true
		logger.Info("Connected to Prometheus: %s", cfg.Prometheus.URL)
	}
	testCancel()

	// API routes
	router.Route("/api/v1", func(r chi.Router) {
		// Register all handlers (spikes, timeline, config, export, gravity-scores)
		handlers.RegisterRoutesWithRepo(r, repo, prometheusClient, cfg)

		r.Get("/", func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte("{\"status\": \"ok\", \"message\": \"Trace-Point API v1\"}"))
		})
	})

	// Create HTTP server
	server := &http.Server{
		Addr:         cfg.GetAddr(),
		Handler:      router,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	if !connected {
		logger.Warn("Prometheus not available - skipping spike detection polling")
	} else {
		// Initialize Discord cooldown tracker with thread-safe sync.Map
		var discordCooldowns sync.Map

		// Initialize context for graceful shutdown of polling loop
		pollingCtx, cancelPolling := context.WithCancel(context.Background())
		defer cancelPolling()

		// Initialize spike detector with config values
		detectorConfig := correlation.DefaultDetectorConfig()

		// Apply config overrides
		if cfg.Detection.CPUThreshold > 0 {
			detectorConfig.ThresholdPercent = float64(cfg.Detection.CPUThreshold)
		}
		if cfg.Detection.WindowSize > 0 {
			detectorConfig.MovingAverageWindowMinutes = int(cfg.Detection.WindowSize.Minutes())
		}
		if cfg.Detection.MinSamples > 0 {
			// Apply min samples setting if needed
		}

		detector := correlation.NewSpikeDetector(detectorConfig, logger.Default())
		logger.Info("Spike detector initialized | threshold=%.0f%% | window=%dm | polling=%ds",
			detectorConfig.ThresholdPercent,
			detectorConfig.MovingAverageWindowMinutes,
			detectorConfig.PollingIntervalSeconds,
		)

		// Start spike detection polling loop in goroutine
		go func() {
			ticker := time.NewTicker(time.Duration(detectorConfig.PollingIntervalSeconds) * time.Second)
			defer ticker.Stop()

			logger.Info("Starting Prometheus metrics polling every %d seconds...", detectorConfig.PollingIntervalSeconds)

			// Initial poll
			detectorCtx, detectorCancel := context.WithTimeout(pollingCtx, 30*time.Second)
			spikes, err := detector.DetectSpikes(detectorCtx, prometheusClient, cfg.Namespaces, nil)
			if err != nil {
				logger.Error("Initial spike detection error: %v", err)
			} else if len(spikes) > 0 {
				logger.Info("Initial scan: %d spike(s) detected", len(spikes))
				// Save initial spikes to database
				for _, spike := range spikes {
					storageModel := spike.ToStorageModel()
					if err := repo.CreateSpikeEvent(context.Background(), storageModel); err != nil {
						logger.Error("Failed to save initial spike to database: %v", err)
					} else {
						logger.Info("Initial spike saved to database: %s/%s", spike.Namespace, spike.PodName)
					}
				}
			} else {
				logger.Debug("Initial scan: no spikes detected")
			}
			detectorCancel()

			for {
				select {
				case <-pollingCtx.Done():
					logger.Info("Polling loop shutting down...")
					return
				case <-ticker.C:
					detectorCtx, detectorCancel := context.WithTimeout(pollingCtx, 30*time.Second)
					spikes, err := detector.DetectSpikes(detectorCtx, prometheusClient, cfg.Namespaces, nil)
					if err != nil {
						logger.Error("Spike detection error: %v", err)
						detectorCancel()
						continue
					}

					if len(spikes) > 0 {
						logger.Info("=== SPIKE DETECTED: %d spike(s)! ===", len(spikes))
						for _, spike := range spikes {
							logger.Info("  -> %s/%s | CPU: %.1f%% | RAM: %.1f%% | threshold: %.0f%%",
								spike.Namespace, spike.PodName, spike.CPUPercent, spike.RAMPercent, spike.ThresholdPercent)

							// Log detected spikes for debugging
							logger.Debug("Spike alert: namespace=%s pod=%s cpu=%.1f%% ram=%.1f%% threshold=%.0f%%",
								spike.Namespace, spike.PodName, spike.CPUPercent, spike.RAMPercent, spike.ThresholdPercent)

							// Check cooldown before sending Discord alert
							spikeKey := fmt.Sprintf("%s/%s", spike.Namespace, spike.PodName)
							cooldownDuration := 5 * time.Minute
							if cfg.Discord.CooldownMinutes > 0 {
								cooldownDuration = time.Duration(cfg.Discord.CooldownMinutes) * time.Minute
							}
							// Thread-safe check and update using sync.Map
							if lastAlert, ok := discordCooldowns.Load(spikeKey); ok {
								if time.Since(lastAlert.(time.Time)) < cooldownDuration {
									logger.Debug("Skipping Discord alert for %s (cooldown active)", spikeKey)
								} else {
									// Send Discord alert
									if discordClient != nil {
										storageModel := spike.ToStorageModel()
										if err := discordClient.SendSpikeAlert(context.Background(), storageModel); err != nil {
											logger.Error("Failed to send Discord alert: %v", err)
										} else {
											discordCooldowns.Store(spikeKey, time.Now())
											logger.Info("Discord alert sent for %s", spikeKey)
										}
									}
								}
							} else {
								// No previous alert, send Discord alert
								if discordClient != nil {
									storageModel := spike.ToStorageModel()
									if err := discordClient.SendSpikeAlert(context.Background(), storageModel); err != nil {
										logger.Error("Failed to send Discord alert: %v", err)
									} else {
										discordCooldowns.Store(spikeKey, time.Now())
										logger.Info("Discord alert sent for %s", spikeKey)
									}
								}
							}

							// Save spike to database
							storageModel := spike.ToStorageModel()
							if err := repo.CreateSpikeEvent(context.Background(), storageModel); err != nil {
								logger.Error("Failed to save spike to database: %v", err)
							} else {
								logger.Info("Spike saved to database: %s/%s", spike.Namespace, spike.PodName)
							}
						}
					} else {
						logger.Debug("Polling: no spikes detected")
					}
					detectorCancel()
				}
			}
		}()
	}

	// Start server in goroutine
	go func() {
		logger.Info("Server listening on %s", cfg.GetAddr())
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("Server failed: %v", err)
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down server...")

	// Graceful shutdown with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		logger.Error("Server forced to shutdown: %v", err)
	}

	logger.Info("Server exited")
}

// handleHealth handles the health check endpoint
func handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status": "healthy", "timestamp": "` + time.Now().Format(time.RFC3339) + `"}`))
}
