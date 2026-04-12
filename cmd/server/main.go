package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/trace-point/trace-point/internal/config"
	"github.com/trace-point/trace-point/internal/integration/discord"
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

	// API routes
	router.Route("/api/v1", func(r chi.Router) {
		// Register all handlers (spikes, timeline, config, export, gravity-scores)
		handlers.RegisterRoutesWithRepo(r, repo)

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

	// Store discord client in global for alerting (optional enhancement)
	_ = discordClient

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
