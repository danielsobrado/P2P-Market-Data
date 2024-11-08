package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"p2p_market_data/pkg/config"
	"p2p_market_data/pkg/data"
	"p2p_market_data/pkg/p2p"
	"p2p_market_data/pkg/scheduler"
	"p2p_market_data/pkg/scripts"
	"p2p_market_data/pkg/utils"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
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

	// Load configuration
	cfg, err := config.Load(*configPath)
	if err != nil {
		logger.Fatal("Failed to load configuration",
			zap.String("path", *configPath),
			zap.Error(err))
	}

	// Create application context
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Initialize P2P host
	repo := data.NewRepository()
	host, err := p2p.NewHost(ctx, &cfg.P2P, repo, logger)
	if err != nil {
		logger.Fatal("Failed to initialize P2P host",
			zap.Error(err))
	}
	defer host.Stop()

	// Initialize script manager
	scriptMgr, err := scripts.NewManager(cfg.Scripts, logger)
	if err != nil {
		logger.Fatal("Failed to initialize script manager",
			zap.Error(err))
	}

	// Initialize scheduler
	sched := scheduler.NewScheduler(scriptMgr, cfg.Scheduler, logger)
	if err := sched.Start(ctx); err != nil {
		logger.Fatal("Failed to start scheduler",
			zap.Error(err))
	}
	defer sched.Stop()

	// Create app structure for Wails
	app := NewApp(cfg, host, scriptMgr, sched, logger)

	// Setup graceful shutdown
	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, syscall.SIGINT, syscall.SIGTERM)

	// Start Wails application
	go func() {
		err := wails.Run(&options.App{
			Title:     "P2P Market Data Platform",
			Width:     1024,
			Height:    768,
			MinWidth:  800,
			MinHeight: 600,
			Bind: []interface{}{
				app,
			},
			// Add other Wails options as needed
		})
		if err != nil {
			logger.Error("Wails application failed",
				zap.Error(err))
			shutdown <- syscall.SIGTERM
		}
	}()

	// Wait for shutdown signal
	<-shutdown
	logger.Info("Shutdown signal received")

	// Create context with timeout for graceful shutdown
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	// Perform graceful shutdown
	if err := performGracefulShutdown(shutdownCtx, app, logger); err != nil {
		logger.Error("Error during shutdown",
			zap.Error(err))
		os.Exit(1)
	}

	logger.Info("Application shutdown complete")
}

// App represents the main application structure
type App struct {
	cfg       *config.Config
	host      *p2p.Host
	scriptMgr *scripts.Manager
	scheduler *scheduler.Scheduler
	logger    *zap.Logger
}

// NewApp creates a new application instance
func NewApp(cfg *config.Config, host *p2p.Host, scriptMgr *scripts.Manager,
	sched *scheduler.Scheduler, logger *zap.Logger) *App {
	return &App{
		cfg:       cfg,
		host:      host,
		scriptMgr: scriptMgr,
		scheduler: sched,
		logger:    logger,
	}
}

// performGracefulShutdown handles graceful shutdown of all components
func performGracefulShutdown(ctx context.Context, app *App, logger *zap.Logger) error {
	errChan := make(chan error, 3)

	// Shutdown P2P host
	go func() {
		if err := app.host.Stop(); err != nil {
			errChan <- fmt.Errorf("p2p host shutdown failed: %w", err)
			return
		}
		errChan <- nil
	}()

	// Stop scheduler
	go func() {
		app.scheduler.Stop()
		errChan <- nil
	}()

	// Cleanup script manager resources
	go func() {
		if err := app.scriptMgr.Cleanup(); err != nil {
			errChan <- fmt.Errorf("script manager cleanup failed: %w", err)
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
