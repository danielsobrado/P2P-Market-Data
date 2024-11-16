// cmd/app/main.go
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"go.uber.org/zap"

	"p2p_market_data/pkg/config"
	"p2p_market_data/pkg/database"
	"p2p_market_data/pkg/scripts"
)

var (
	configFile = flag.String("config", "config.yaml", "Path to configuration file")
	dataDir    = flag.String("data-dir", "./data", "Data directory path")
	debug      = flag.Bool("debug", false, "Enable debug mode")
)

// App represents the CLI application
type App struct {
	db        *database.Service 
	scriptMgr *scripts.ScriptManager 
	logger    *zap.Logger
	ctx       context.Context
	cancel    context.CancelFunc
}

func main() {
	flag.Parse()

	// Initialize logger
	logger, err := initLogger(*debug)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize logger: %v\n", err)
		os.Exit(1)
	}
	defer logger.Sync()

	// Load configuration
	cfg, err := loadConfig(*configFile)
	if err != nil {
		logger.Fatal("Failed to load configuration",
			zap.String("path", *configFile),
			zap.Error(err),
		)
	}

	// Ensure data directory exists
	if err := os.MkdirAll(*dataDir, 0755); err != nil {
		logger.Fatal("Failed to create data directory", zap.Error(err))
	}

	// Create context for application lifecycle
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Initialize application
	app, err := initializeApp(ctx, cfg, logger)
	if err != nil {
		logger.Fatal("Failed to initialize application", zap.Error(err))
	}

	// Setup shutdown handling
	setupGracefulShutdown(ctx, cancel, app, logger)

	// Block until shutdown signal
	<-ctx.Done()
}

func initializeApp(ctx context.Context, cfg *config.Config, logger *zap.Logger) (*App, error) {
	// Create services with timeouts
	initCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	// Initialize database service
	dbService, err := database.NewService(&cfg.Database, logger)
	if err != nil {
		return nil, fmt.Errorf("initializing database service: %w", err)
	}

	// Initialize script manager
	scriptConfig := &config.ScriptConfig{
		ScriptDir: filepath.Join(*dataDir, "scripts"),
		// Add other required config fields
	}
	
	scriptManager, err := scripts.NewScriptManager(scriptConfig, logger)
	if err != nil {
		logger.Fatal("Failed to initialize script manager", zap.Error(err))
	}

	app := &App{
		db:        dbService,
		scriptMgr: scriptManager,
		logger:    logger,
		ctx:       ctx,
		cancel:    cancel,
	}

	// Start all services
	if err := app.start(initCtx); err != nil {
		// Cleanup on failure
		app.stop(context.Background())
		return nil, fmt.Errorf("starting services: %w", err)
	}

	return app, nil
}

func (a *App) start(ctx context.Context) error {
	// Start services in order with proper error handling
	if err := a.db.Start(ctx); err != nil {
		return fmt.Errorf("starting database: %w", err)
	}

	if err := a.scriptMgr.Start(ctx); err != nil {
		a.db.Stop(ctx) // Cleanup started services on failure
		return fmt.Errorf("starting script manager: %w", err)
	}

	a.logger.Info("All services started successfully")
	return nil
}

func (a *App) stop(ctx context.Context) error {
	// Stop services in reverse order
	var errs []error

	if err := a.scriptMgr.Stop(ctx); err != nil {
		errs = append(errs, fmt.Errorf("stopping script manager: %w", err))
	}

	if err := a.db.Stop(ctx); err != nil {
		errs = append(errs, fmt.Errorf("stopping database: %w", err))
	}

	// Log all errors but continue shutdown
	for _, err := range errs {
		a.logger.Error("Shutdown error", zap.Error(err))
	}

	a.logger.Info("All services stopped")
	return nil
}

func setupGracefulShutdown(ctx context.Context, cancel context.CancelFunc, app *App, logger *zap.Logger) {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		select {
		case sig := <-sigChan:
			logger.Info("Received shutdown signal", zap.String("signal", sig.String()))
		case <-ctx.Done():
			logger.Info("Context cancelled")
		}

		// Create shutdown context with timeout
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer shutdownCancel()

		if err := app.stop(shutdownCtx); err != nil {
			logger.Error("Error during shutdown", zap.Error(err))
			os.Exit(1)
		}

		cancel() // Cancel main context
	}()
}

func loadConfig(path string) (*config.Config, error) {
	cfg, err := config.Load(path)
	if err != nil {
		return nil, fmt.Errorf("loading config: %w", err)
	}

	// Set default values
	if cfg.Database.Port == 0 {
		cfg.Database.Port = 5433
	}
	if cfg.P2P.Port == 0 {
		cfg.P2P.Port = 4001
	}

	return cfg, nil
}

func initLogger(debug bool) (*zap.Logger, error) {
	if debug {
		return zap.NewDevelopment()
	}
	return zap.NewProduction()
}
