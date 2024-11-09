package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"p2p_market_data/pkg/config"
	"p2p_market_data/pkg/data"
	"p2p_market_data/pkg/p2p/host"
	"p2p_market_data/pkg/scheduler"
	"p2p_market_data/pkg/scripts"
	"p2p_market_data/pkg/utils"

	"go.uber.org/zap"
)

var (
	configPath = flag.String("config", "config.yaml", "path to configuration file")
	debug      = flag.Bool("debug", false, "enable debug mode")
)

func main() {
	// Parse command line flags
	flag.Parse()

	// Initialize logger
	logConfig := &utils.LogConfig{Debug: *debug}
	logger, err := utils.NewLogger(logConfig)
	if err != nil {
		fmt.Printf("Failed to initialize logger: %v\n", err)
		os.Exit(1)
	}
	defer logger.Sync()

	// Load configurationc
	cfg, err := config.Load(*configPath)
	if err != nil {
		logger.Fatal("Failed to load configuration",
			zap.String("path", *configPath),
			zap.Error(err))
	}

	// Log the database URL (optional and secure handling recommended)
	logger.Info("Configuration loaded",
		zap.String("database_url", maskDatabaseURL(cfg.Database.URL)))

	// Create application context
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Initialize PostgreSQL repository
	repo, err := data.NewPostgresRepository(ctx, cfg.Database.URL, logger)
	if err != nil {
		logger.Fatal("Failed to initialize Postgres repository", zap.Error(err))
	}
	defer repo.Close() // Ensure repository is closed properly

	// Initializei P2P host
	h, err := host.NewHost(ctx, cfg, logger, repo)
	if err != nil {
		logger.Fatal("Failed to initialize P2P host", zap.Error(err))
	}
	defer h.Stop()

	// Initialize script manager
	scriptMgr, err := scripts.NewScriptManager(&cfg.Scripts, logger)
	if err != nil {
		logger.Fatal("Failed to initialize script manager",
			zap.Error(err))
	}
	defer func() {
		if err := scriptMgr.Stop(); err != nil {
			logger.Error("Failed to stop script manager", zap.Error(err))
		}
	}()

	// Initialize scheduler
	schedulerInstance := scheduler.NewScheduler(scriptMgr, &cfg.Scheduler, logger)
	defer func() {
		if err := schedulerInstance.Stop(); err != nil {
			logger.Error("Failed to stop scheduler", zap.Error(err))
		}
	}()

	// Handle graceful shutdown
	go handleShutdown(ctx, cancel, logger, h, scriptMgr, schedulerInstance)

	// Start scheduler
	if err := schedulerInstance.Start(); err != nil {
		logger.Fatal("Scheduler failed tog start", zap.Error(err))
	}

	// Block main goroutine
	select {}
}

// handleShutdown listenso for OS signals and initiates graceful shutdown.
func handleShutdown(ctx context.Context, cancel context.CancelFunc, logger *zap.Logger, h *host.Host, scriptMgr *scripts.ScriptManager, sched *scheduler.Scheduler) {
	// Listen for OS signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	<-sigChan

	logger.Info("Received shutdown signal")

	// Cancel the context to stop all operations
	cancel()

	// Perform graceful shutdown
	if err := performGracefulShutdown(ctx, logger, h, scriptMgr, sched); err != nil {
		logger.Error("Graceful shutdown failed", zap.Error(err))
		os.Exit(1)
	}

	logger.Info("Application gracefully shut down")
	os.Exit(0)
}

// performGracefulShutdown handles the shutdown of all components.
func performGracefulShutdown(ctx context.Context, logger *zap.Logger, h *host.Host, scriptMgr *scripts.ScriptManager, sched *scheduler.Scheduler) error {
	errChan := make(chan error, 3)

	// Shutdown P2P host
	go func() {
		if err := h.Stop(); err != nil {
			errChan <- fmt.Errorf("P2P host shutdown failed: %w", err)
			return
		}
		errChan <- nil
	}()

	// Stop scheduler
	go func() {
		if err := sched.Stop(); err != nil {
			errChan <- fmt.Errorf("Scheduler shutdown failed: %w", err)
			return
		}
		errChan <- nil
	}()

	// Stop script manager
	go func() {
		if err := scriptMgr.Stop(); err != nil {
			errChan <- fmt.Errorf("Script manager shutdown failed: %w", err)
			return
		}
		errChan <- nil
	}()

	// Wait for all shutdowns or timeout
	for i := 0; i < 3; i++ {
		select {
		case err := <-errChan:
			if err != nil {
				return err
			}
		case <-ctx.Done():
			return ctx.Err()
		}
	}

	return nil
}

// maskDatabaseURL masks sensitive information in the database URL for logging.
func maskDatabaseURL(url string) string {
	// Simple masking example; customize as needed
	if idx := len("postgres://user:"); idx < len(url) {
		maskedURL := url[:idx] + "******" + url[idx+len("password@"):]
		return maskedURL
	}
	return "******"
}
