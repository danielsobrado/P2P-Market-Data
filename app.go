// app.go
package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	postgres "github.com/fergusstrange/embedded-postgres"
	"github.com/jackc/pgx/v5"
	"go.uber.org/zap"

	"p2p_market_data/pkg/config"
	"p2p_market_data/pkg/data"
	"p2p_market_data/pkg/p2p/host" // Updated import path
	"p2p_market_data/pkg/scripts"
)

// App represents the main application structure
type App struct {
	ctx    context.Context
	cancel context.CancelFunc
	logger *zap.Logger
	config *config.Config

	// Core services
	conn      *pgx.Conn
	repo      data.Repository
	peerHost  *host.Host
	scriptMgr *scripts.ScriptManager
	embedded  *postgres.EmbeddedPostgres

	// State
	mu      sync.RWMutex
	running bool
	cleanup []func() error
}

// ServerStatus represents the status of the application's core services
type ServerStatus struct {
	Running           bool `json:"running"`
	DatabaseConnected bool `json:"databaseConnected"`
	P2PHostRunning    bool `json:"p2pHostRunning"`
	ScriptMgrRunning  bool `json:"scriptMgrRunning"`
	EmbeddedDBRunning bool `json:"embeddedDbRunning"`
}

// NewApp creates a new application instance
func NewApp() *App {
	logger, _ := zap.NewProduction()
	ctx, cancel := context.WithCancel(context.Background())

	// Create data directories
	dirs := []string{
		"./data",
		"./data/postgres",
		"./data/scripts",
		"./data/keys",
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			logger.Fatal("Failed to create directory",
				zap.String("dir", dir),
				zap.Error(err))
		}
	}

	// Create default configuration
	cfg := &config.Config{
		Database: config.DatabaseConfig{
			Type:           "postgres",
			Port:           5433,
			MaxConnections: 10,
			MinConnections: 2,
			SSLMode:        "disable",
		},
		P2P: config.P2PConfig{
			Port:           4001,
			MaxPeers:       50,
			MinPeers:       5,
			PeerTimeout:    30 * time.Second,
			BootstrapPeers: []string{},
		},
		Scripts: config.ScriptConfig{
			ScriptDir:   "./data/scripts",
			MaxExecTime: 5 * time.Minute,
			MaxMemoryMB: 512,
			AllowedPkgs: []string{"pandas", "numpy", "requests"},
		},
		Security: config.SecurityConfig{
			KeyFile: "./data/keys/host.key",
		},
	}

	return &App{
		ctx:     ctx,
		cancel:  cancel,
		logger:  logger,
		config:  cfg,
		cleanup: make([]func() error, 0),
	}
}

func (a *App) startup(ctx context.Context) {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.running {
		return
	}

	var err error

	// Initialize embedded database
	if err := a.initEmbeddedDB(); err != nil {
		a.logger.Fatal("Failed to initialize embedded database", zap.Error(err))
		return
	}

	// Initialize database connection
	if err := a.initDatabase(ctx); err != nil {
		a.logger.Fatal("Failed to initialize database", zap.Error(err))
		return
	}

	// Initialize repository
	a.repo, err = data.NewPostgresRepository(ctx, a.conn, a.logger)
	if err != nil {
		a.logger.Fatal("Failed to create repository", zap.Error(err))
		return
	}

	// Initialize and start services
	if err := a.initServices(ctx); err != nil {
		a.logger.Fatal("Failed to initialize services", zap.Error(err))
		return
	}

	a.running = true
	a.logger.Info("Application started successfully")
}

func (a *App) initEmbeddedDB() error {
	pg := postgres.NewDatabase(
		postgres.DefaultConfig().
			Username("postgres").
			Password("postgres").
			Database("market_data").
			Version(postgres.V12).
			Port(uint32(a.config.Database.Port)).
			RuntimePath("./data/postgres"))

	a.embedded = pg
	return pg.Start()
}

func (a *App) initDatabase(ctx context.Context) error {
	connStr := fmt.Sprintf("postgres://postgres:postgres@localhost:%d/market_data?sslmode=disable",
		a.config.Database.Port)

	var err error
	a.conn, err = pgx.Connect(ctx, connStr)
	if err != nil {
		return fmt.Errorf("connecting to database: %w", err)
	}

	return nil
}

func (a *App) initServices(ctx context.Context) error {
	var err error

	// Initialize script manager
	a.scriptMgr, err = scripts.NewScriptManager(&a.config.Scripts, a.logger)
	if err != nil {
		return fmt.Errorf("initializing script manager: %w", err)
	}

	// Initialize P2P host
	a.peerHost, err = host.NewHost(ctx, a.config, a.logger, a.repo)
	if err != nil {
		return fmt.Errorf("creating p2p host: %w", err)
	}

	// Start services
	if err := a.startServices(ctx); err != nil {
		return fmt.Errorf("starting services: %w", err)
	}

	return nil
}

func (a *App) startServices(ctx context.Context) error {
	// Start script manager
	if err := a.scriptMgr.Start(ctx); err != nil {
		return fmt.Errorf("starting script manager: %w", err)
	}

	// Start P2P host
	if err := a.peerHost.Start(ctx); err != nil {
		a.scriptMgr.Stop(ctx)
		return fmt.Errorf("starting p2p host: %w", err)
	}

	return nil
}

func (a *App) shutdown(ctx context.Context) {
	a.mu.Lock()
	defer a.mu.Unlock()

	if !a.running {
		return
	}

	// Stop services in reverse order
	if err := a.scriptMgr.Stop(ctx); err != nil {
		a.logger.Error("Error stopping script manager", zap.Error(err))
	}

	if err := a.peerHost.Close(); err != nil {
		a.logger.Error("Error stopping p2p host", zap.Error(err))
	}

	// Close database connection
	if a.conn != nil {
		a.conn.Close(ctx)
	}

	// Run cleanup functions
	for _, cleanup := range a.cleanup {
		if err := cleanup(); err != nil {
			a.logger.Error("Cleanup error", zap.Error(err))
		}
	}

	a.running = false
	a.cancel()
}

// Script management methods

func (a *App) RunScript(scriptID string) error {
	_, err := a.scriptMgr.ExecuteScript(a.ctx, scriptID, nil)
	return err
}

func (a *App) StopScript(scriptID string) error {
	return a.scriptMgr.Executor.StopScript(scriptID)
}

func (a *App) GetScriptContent(scriptID string) (string, error) {
	script, err := a.scriptMgr.GetScript(scriptID)
	if err != nil {
		return "", err
	}

	scriptPath := filepath.Join(a.config.Scripts.ScriptDir, script.Name)
	content, err := os.ReadFile(scriptPath)
	if err != nil {
		return "", fmt.Errorf("reading script file: %w", err)
	}

	return string(content), nil
}

// Repository methods

func (a *App) GetEODData(symbol, startDate, endDate string) ([]data.EODData, error) {
	return a.repo.GetEODData(a.ctx, symbol, startDate, endDate)
}

func (a *App) GetDividendData(symbol, startDate, endDate string) ([]data.DividendData, error) {
	start, err := time.Parse("2006-01-02", startDate)
	if err != nil {
		return nil, fmt.Errorf("invalid start date: %w", err)
	}
	end, err := time.Parse("2006-01-02", endDate)
	if err != nil {
		return nil, fmt.Errorf("invalid end date: %w", err)
	}

	ptrData, err := a.repo.GetDividendData(a.ctx, symbol, start, end)
	if err != nil {
		return nil, err
	}

	// Convert []*DividendData to []DividendData
	result := make([]data.DividendData, len(ptrData))
	for i, d := range ptrData {
		result[i] = *d
	}
	return result, nil
}

func (a *App) GetDataSources() ([]data.DataSource, error) {
	return a.repo.GetDataSources(a.ctx)
}

func (a *App) SearchMarketData(filter data.MarketDataFilter) ([]*data.MarketData, error) {
	return a.repo.ListMarketData(a.ctx, filter)
}

func (a *App) SaveMarketData(data *data.MarketData) error {
	return a.repo.SaveMarketData(a.ctx, data)
}

// GetPeers returns connected peers
func (a *App) GetPeers() ([]data.Peer, error) {
	// Get peer pointers from repository
	ptrPeers, err := a.repo.ListPeers(a.ctx, data.PeerFilter{})
	if err != nil {
		return nil, fmt.Errorf("listing peers: %w", err)
	}

	// Convert []*Peer to []Peer
	peers := make([]data.Peer, len(ptrPeers))
	for i, p := range ptrPeers {
		peers[i] = *p // Dereference pointer to get value
	}

	return peers, nil
}

// CheckConnection checks if P2P service is connected
func (a *App) CheckConnection() bool {
	return a.running && a.peerHost != nil
}

func (a *App) domReady(ctx context.Context) {
	a.logger.Info("DOM Ready")

	// Initialize any UI-dependent features here
	if err := a.initServices(ctx); err != nil {
		a.logger.Error("Failed to initialize services", zap.Error(err))
		return
	}
}

func (a *App) beforeClose(ctx context.Context) bool {
	// Return true to prevent closing, false to allow
	return false
}

// GetServerStatus returns the status of the application's services
func (a *App) GetServerStatus() ServerStatus {
	a.mu.RLock()
	defer a.mu.RUnlock()

	status := ServerStatus{
		Running: a.running,
	}

	// Check database connection
	status.DatabaseConnected = a.conn != nil
	if status.DatabaseConnected {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		if err := a.conn.Ping(ctx); err != nil {
			status.DatabaseConnected = false
		}
	}

	// Check P2P host status
	status.P2PHostRunning = a.peerHost != nil && a.peerHost.IsRunning()

	// Check script manager status
	status.ScriptMgrRunning = a.scriptMgr != nil && a.scriptMgr.IsRunning()

	// Check embedded Postgres status
	status.EmbeddedDBRunning = a.embedded != nil

	return status
}
