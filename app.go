// app.go
package main

import (
	"context"
	"errors"
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

// ActiveTransfer is a frontend-facing transfer status payload.
type ActiveTransfer struct {
	ID          string  `json:"id"`
	Type        string  `json:"type"`
	Symbol      string  `json:"symbol"`
	Source      string  `json:"source"`
	Destination string  `json:"destination"`
	Progress    float64 `json:"progress"`
	Status      string  `json:"status"`
	StartTime   string  `json:"startTime"`
	EndTime     string  `json:"endTime,omitempty"`
	Size        int64   `json:"size"`
	Speed       float64 `json:"speed"`
}

// NewApp creates a new application instance using the provided configuration.
// cfg must not be nil; callers should load it via config.Load or config.LoadDefaults.
// Returns an error if logger creation or directory setup fails.
func NewApp(cfg *config.Config) (*App, error) {
	if cfg == nil {
		return nil, errors.New("config must not be nil")
	}

	// Build logger from configuration.
	var zapLogger *zap.Logger
	var err error
	if cfg.IsDevelopment() {
		zapLogger, err = zap.NewDevelopment()
	} else {
		zapLogger, err = zap.NewProduction()
	}
	if err != nil {
		return nil, fmt.Errorf("creating logger: %w", err)
	}

	ctx, cancel := context.WithCancel(context.Background())

	// Create data directories needed by the embedded Postgres runtime and scripts.
	dirs := []string{
		"./data",
		"./data/postgres",
		"./data/scripts",
		"./data/keys",
	}
	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			cancel()
			_ = zapLogger.Sync()
			return nil, fmt.Errorf("creating directory %q: %w", dir, err)
		}
	}

	// Ensure the script directory points at the embedded data layout when the
	// config is still using the generic default value.
	if cfg.Scripts.ScriptDir == "scripts" {
		cfg.Scripts.ScriptDir = "./data/scripts"
	}

	return &App{
		ctx:     ctx,
		cancel:  cancel,
		logger:  zapLogger,
		config:  cfg,
		cleanup: make([]func() error, 0),
	}, nil
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
		a.logger.Error("Failed to initialize embedded database", zap.Error(err))
		return
	}

	// Initialize database connection
	if err := a.initDatabase(ctx); err != nil {
		a.logger.Error("Failed to initialize database", zap.Error(err))
		a.cleanupEmbeddedDB()
		return
	}

	// Initialize database schema for a fresh embedded database instance
	if err := data.NewSchemaManager(a.conn).InitializeSchema(ctx); err != nil {
		a.logger.Error("Failed to initialize database schema", zap.Error(err))
		a.cleanupDBResources(ctx)
		return
	}

	// Initialize repository
	a.repo, err = data.NewPostgresRepository(ctx, a.conn, a.logger)
	if err != nil {
		a.logger.Error("Failed to create repository", zap.Error(err))
		a.cleanupDBResources(ctx)
		return
	}

	// Initialize and start services
	if err := a.initServices(ctx); err != nil {
		a.logger.Error("Failed to initialize services", zap.Error(err))
		a.cleanupDBResources(ctx)
		return
	}

	a.running = true
	a.logger.Info("Application started successfully")
}

// cleanupEmbeddedDB stops the embedded Postgres instance if it was started.
func (a *App) cleanupEmbeddedDB() {
	if a.embedded != nil {
		if err := a.embedded.Stop(); err != nil {
			a.logger.Error("Failed to stop embedded database during cleanup", zap.Error(err))
		}
	}
}

// cleanupDBResources closes the database connection and stops the embedded
// Postgres instance. Called on partial startup failure.
func (a *App) cleanupDBResources(ctx context.Context) {
	if a.conn != nil {
		a.conn.Close(ctx)
		a.conn = nil
	}
	a.cleanupEmbeddedDB()
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
	if a.scriptMgr != nil {
		if err := a.scriptMgr.Stop(ctx); err != nil {
			a.logger.Error("Error stopping script manager", zap.Error(err))
		}
	}

	if a.peerHost != nil {
		if err := a.peerHost.Close(); err != nil {
			a.logger.Error("Error stopping p2p host", zap.Error(err))
		}
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
	script, err := a.scriptMgr.GetScript(scriptID)
	if err != nil {
		return err
	}
	scriptPath := filepath.Join(a.config.Scripts.ScriptDir, script.Name)
	return a.scriptMgr.Executor.StopScript(scriptPath)
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

func (a *App) SearchData(request data.DataRequest) ([]data.DataSource, error) {
	if a.repo == nil {
		return nil, errors.New("repository not initialized")
	}
	return a.repo.SearchData(a.ctx, request)
}

func (a *App) RequestData(peerID string, request data.DataRequest) error {
	if a.peerHost == nil {
		return errors.New("peer host not initialized")
	}
	return a.peerHost.RequestData(a.ctx, peerID, request)
}

func (a *App) GetInsiderData(symbol, startDate, endDate string) ([]data.InsiderTrade, error) {
	if a.repo == nil {
		return nil, errors.New("repository not initialized")
	}
	return a.repo.GetInsiderData(a.ctx, symbol, startDate, endDate)
}

func (a *App) GetActiveTransfers() ([]ActiveTransfer, error) {
	// Transfer tracking is not yet persisted; return a stable empty payload.
	return []ActiveTransfer{}, nil
}

func (a *App) ResetDataConnection() error {
	if a.peerHost == nil {
		return nil
	}
	return a.peerHost.ResetConnection()
}

func (a *App) ResetDataProcessing() error {
	if a.peerHost == nil {
		return nil
	}
	return a.peerHost.ResetProcessing()
}

func (a *App) RetryConnection() error {
	if a.peerHost == nil {
		return nil
	}
	return a.peerHost.RetryConnection()
}

func (a *App) UpdateMarketData(items []data.MarketData) error {
	if a.repo == nil {
		return errors.New("repository not initialized")
	}
	for i := range items {
		item := items[i]
		if err := a.repo.SaveMarketData(a.ctx, &item); err != nil {
			return err
		}
	}
	return nil
}

func (a *App) UploadMarketData(payload map[string]interface{}) error {
	if a.repo == nil {
		return errors.New("repository not initialized")
	}

	symbol, _ := payload["symbol"].(string)
	source, _ := payload["source"].(string)
	dataType, _ := payload["type"].(string)
	price, _ := payload["price"].(float64)
	volume, _ := payload["volume"].(float64)

	if dataType == "" {
		dataType = data.DataTypeEOD
	}
	if source == "" {
		source = "manual_upload"
	}
	if volume == 0 {
		volume = 1
	}
	if price == 0 {
		price = 1
	}

	marketData, err := data.NewMarketData(symbol, price, volume, source, dataType)
	if err != nil {
		return err
	}
	return a.repo.SaveMarketData(a.ctx, marketData)
}

func (a *App) UploadScript(scriptData map[string]string) error {
	if a.scriptMgr == nil {
		return errors.New("script manager not initialized")
	}

	name := scriptData["name"]
	content := scriptData["content"]
	if name == "" || content == "" {
		return errors.New("script name and content are required")
	}

	metadata := &scripts.ScriptMetadata{
		Name:        name,
		Description: scriptData["description"],
		Author:      scriptData["author"],
		Version:     scriptData["version"],
	}
	return a.scriptMgr.AddScript(name, []byte(content), metadata)
}

func (a *App) DeleteScript(scriptID string) error {
	if a.scriptMgr == nil {
		return errors.New("script manager not initialized")
	}
	return a.scriptMgr.DeleteScript(scriptID)
}

func (a *App) InstallScript(scriptID string) error {
	// Installation lifecycle is currently equivalent to script availability.
	_, err := a.scriptMgr.GetScript(scriptID)
	return err
}

func (a *App) UninstallScript(scriptID string) error {
	return a.DeleteScript(scriptID)
}
