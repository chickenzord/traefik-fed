package main

import (
	"context"
	"flag"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/chickenzord/traefik-fed/internal/aggregator"
	"github.com/chickenzord/traefik-fed/internal/config"
	"github.com/chickenzord/traefik-fed/internal/output"
	"github.com/traefik/traefik/v3/pkg/config/dynamic"
)

func main() {
	configPath := flag.String("config", "config.yaml", "Path to configuration file")

	flag.Parse()

	// Load configuration first (we need it for logger setup)
	cfg, err := config.Load(*configPath)
	if err != nil {
		// Use default logger for config loading errors
		logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
			Level: slog.LevelInfo,
		}))
		logger.Error("failed to load configuration", "error", err)
		os.Exit(1)
	}

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		// Use default logger for validation errors
		logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
			Level: slog.LevelInfo,
		}))
		logger.Error("invalid configuration", "error", err)
		os.Exit(1)
	}

	// Setup logger based on configuration
	logger := setupLogger(cfg.Log)

	logger.Info("loaded configuration",
		"upstreams", len(cfg.Upstreams),
		"poll_interval", cfg.Server.PollInterval,
		"http_enabled", cfg.Output.HTTP.Enabled,
		"file_enabled", cfg.Output.File.Enabled)

	// Create aggregator
	agg := aggregator.New(cfg, logger)

	// Setup context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// Start HTTP server if enabled
	var httpServer *output.HTTPServer
	if cfg.Output.HTTP.Enabled {
		httpServer = output.NewHTTPServer(cfg.Output.HTTP.Port, cfg.Output.HTTP.Path, logger)

		go func() {
			if err := httpServer.Start(); err != nil {
				logger.Error("HTTP server failed", "error", err)
				cancel()
			}
		}()
	}

	// Start file writer if enabled
	var fileConfigChan chan *dynamic.HTTPConfiguration
	if cfg.Output.File.Enabled {
		fileConfigChan = make(chan *dynamic.HTTPConfiguration, 1)
		fileWriter := output.NewFileWriter(cfg.Output.File.Path, cfg.Output.File.Interval, logger)

		go func() {
			if err := fileWriter.Start(fileConfigChan); err != nil {
				logger.Error("file writer failed", "error", err)
				cancel()
			}
		}()
	}

	// Main polling loop
	ticker := time.NewTicker(cfg.Server.PollInterval)
	defer ticker.Stop()

	// Run initial aggregation
	runAggregation(agg, httpServer, fileConfigChan, logger)

	for {
		select {
		case <-ctx.Done():
			logger.Info("shutting down")
			return
		case <-sigChan:
			logger.Info("received shutdown signal")
			return
		case <-ticker.C:
			runAggregation(agg, httpServer, fileConfigChan, logger)
		}
	}
}

func runAggregation(
	agg *aggregator.Aggregator,
	httpServer *output.HTTPServer,
	fileConfigChan chan *dynamic.HTTPConfiguration,
	logger *slog.Logger,
) {
	httpConfig, err := agg.Aggregate()
	if err != nil {
		logger.Error("aggregation failed", "error", err)
		return
	}

	logger.Info("aggregation completed",
		"routers", len(httpConfig.Routers),
		"services", len(httpConfig.Services))

	// Update HTTP server if enabled
	if httpServer != nil {
		httpServer.UpdateConfig(httpConfig)
	}

	// Send to file writer if enabled
	if fileConfigChan != nil {
		select {
		case fileConfigChan <- httpConfig:
		default:
			// Channel full, skip this update
		}
	}
}

// setupLogger creates a logger based on configuration
func setupLogger(cfg config.LogConfig) *slog.Logger {
	// Parse log level
	var level slog.Level

	switch cfg.Level {
	case "debug":
		level = slog.LevelDebug
	case "info":
		level = slog.LevelInfo
	case "warn":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	default:
		level = slog.LevelInfo
	}

	handlerOpts := &slog.HandlerOptions{
		Level: level,
	}

	// Create handler based on format
	var handler slog.Handler

	switch cfg.Format {
	case "json":
		handler = slog.NewJSONHandler(os.Stdout, handlerOpts)
	case "plain":
		handler = slog.NewTextHandler(os.Stdout, handlerOpts)
	default:
		handler = slog.NewTextHandler(os.Stdout, handlerOpts)
	}

	return slog.New(handler)
}
