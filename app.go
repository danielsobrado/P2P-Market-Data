// app.go
package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"time"

	postgres "github.com/fergusstrange/embedded-postgres"
	"github.com/google/uuid"
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

	scriptInstall     map[string]bool
	scriptInstallPath string
	startedAt         time.Time
}

// ServerStatus represents the status of the application's core services
type ServerStatus struct {
	Running           bool `json:"running"`
	DatabaseConnected bool `json:"databaseConnected"`
	P2PHostRunning    bool `json:"p2pHostRunning"`
	ScriptMgrRunning  bool `json:"scriptMgrRunning"`
	EmbeddedDBRunning bool `json:"embeddedDbRunning"`
}

// AppHealthDiagnostics is a frontend-facing snapshot for beta support and smoke checks.
type AppHealthDiagnostics struct {
	GeneratedAt          string          `json:"generatedAt"`
	UptimeSeconds        int64           `json:"uptimeSeconds"`
	Status               ServerStatus    `json:"status"`
	DatabaseURL          string          `json:"databaseUrl"`
	DatabaseLatencyMs    int64           `json:"databaseLatencyMs"`
	RequiredTables       map[string]bool `json:"requiredTables"`
	P2PHostID            string          `json:"p2pHostId"`
	P2PListenAddresses   []string        `json:"p2pListenAddresses"`
	ConnectedPeers       []string        `json:"connectedPeers"`
	P2PMetrics           P2PMetrics      `json:"p2pMetrics"`
	TransferSummary      TransferSummary `json:"transferSummary"`
	Security             SecurityHealth  `json:"security"`
	ScriptManagerRunning bool            `json:"scriptManagerRunning"`
	PythonRuntime        string          `json:"pythonRuntime"`
	LatestTransferErrors []string        `json:"latestTransferErrors"`
	OperationalWarnings  []string        `json:"operationalWarnings"`
}

type P2PMetrics struct {
	ConnectedPeers    int    `json:"connectedPeers"`
	TotalPeers        int    `json:"totalPeers"`
	MessagesProcessed int64  `json:"messagesProcessed"`
	NetworkLatencyMs  int64  `json:"networkLatencyMs"`
	RequestsReceived  int64  `json:"requestsReceived"`
	RequestsRejected  int64  `json:"requestsRejected"`
	AuthFailures      int64  `json:"authFailures"`
	TransfersStarted  int64  `json:"transfersStarted"`
	TransfersComplete int64  `json:"transfersComplete"`
	TransfersFailed   int64  `json:"transfersFailed"`
	ChunksSent        int64  `json:"chunksSent"`
	ChunksReceived    int64  `json:"chunksReceived"`
	RowsSent          int64  `json:"rowsSent"`
	RowsReceived      int64  `json:"rowsReceived"`
	BytesSent         int64  `json:"bytesSent"`
	BytesReceived     int64  `json:"bytesReceived"`
	LastError         string `json:"lastError,omitempty"`
	LastRequestAt     string `json:"lastRequestAt,omitempty"`
	LastTransferAt    string `json:"lastTransferAt,omitempty"`
	LastUpdated       string `json:"lastUpdated,omitempty"`
}

type TransferSummary struct {
	Pending      int `json:"pending"`
	Transferring int `json:"transferring"`
	Completed    int `json:"completed"`
	Failed       int `json:"failed"`
}

type SecurityHealth struct {
	RequestSigningRequired  bool   `json:"requestSigningRequired"`
	ResponseSigningRequired bool   `json:"responseSigningRequired"`
	PubSubStrictSigning     bool   `json:"pubSubStrictSigning"`
	KeyFileConfigured       bool   `json:"keyFileConfigured"`
	KeyFileExists           bool   `json:"keyFileExists"`
	AuthFailures            int64  `json:"authFailures"`
	LastSecurityError       string `json:"lastSecurityError,omitempty"`
}

// ScriptInfo is a frontend-facing script list payload.
type ScriptInfo struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Author      string `json:"author"`
	Version     string `json:"version"`
	Size        int64  `json:"size"`
	Created     string `json:"created"`
	Updated     string `json:"updated"`
	Status      string `json:"status"`
	IsInstalled bool   `json:"isInstalled"`
}

// ActiveTransfer is a frontend-facing transfer status payload.
type ActiveTransfer struct {
	ID              string  `json:"id"`
	RequestID       string  `json:"requestId,omitempty"`
	Type            string  `json:"type"`
	Symbol          string  `json:"symbol"`
	Source          string  `json:"source"`
	Destination     string  `json:"destination"`
	Progress        float64 `json:"progress"`
	Status          string  `json:"status"`
	StartTime       string  `json:"startTime"`
	EndTime         string  `json:"endTime,omitempty"`
	Size            int64   `json:"size"`
	Speed           float64 `json:"speed"`
	Error           string  `json:"error,omitempty"`
	ChunkSize       int     `json:"chunkSize,omitempty"`
	TotalRows       int     `json:"totalRows,omitempty"`
	TotalChunks     int     `json:"totalChunks,omitempty"`
	CompletedChunks int     `json:"completedChunks,omitempty"`
	ResumeOffset    int     `json:"resumeOffset,omitempty"`
}

type splitRepository interface {
	SaveSplitData(context.Context, *data.SplitData) error
	GetSplitData(context.Context, string, string, string) ([]data.SplitData, error)
}

type transferRepository interface {
	SaveTransfer(context.Context, *data.DataTransfer) error
	ListTransfers(context.Context) ([]data.DataTransfer, error)
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
	scriptInstallPath := filepath.Join(cfg.Scripts.ScriptDir, ".install_state.json")
	scriptInstall := loadScriptInstallState(scriptInstallPath, zapLogger)

	return &App{
		ctx:               ctx,
		cancel:            cancel,
		logger:            zapLogger,
		config:            cfg,
		cleanup:           make([]func() error, 0),
		scriptInstall:     scriptInstall,
		scriptInstallPath: scriptInstallPath,
	}, nil
}

func loadScriptInstallState(path string, logger *zap.Logger) map[string]bool {
	state := make(map[string]bool)
	content, err := os.ReadFile(path)
	if err != nil {
		if !os.IsNotExist(err) {
			logger.Warn("failed to read script install state", zap.String("path", path), zap.Error(err))
		}
		return state
	}
	if err := json.Unmarshal(content, &state); err != nil {
		logger.Warn("failed to parse script install state", zap.String("path", path), zap.Error(err))
		return make(map[string]bool)
	}
	return state
}

func (a *App) saveScriptInstallState() error {
	if a.scriptInstallPath == "" {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(a.scriptInstallPath), 0755); err != nil {
		return fmt.Errorf("creating script install state directory: %w", err)
	}
	content, err := json.MarshalIndent(a.scriptInstall, "", "  ")
	if err != nil {
		return fmt.Errorf("encoding script install state: %w", err)
	}
	if err := os.WriteFile(a.scriptInstallPath, content, 0644); err != nil {
		return fmt.Errorf("writing script install state: %w", err)
	}
	return nil
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
	a.startedAt = time.Now().UTC()
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
	if a.scriptMgr == nil {
		return errors.New("script manager not initialized")
	}
	if installed, ok := a.scriptInstall[scriptID]; ok && !installed {
		return fmt.Errorf("script %s is not installed", scriptID)
	}
	return a.scriptMgr.StartScript(a.ctx, scriptID, nil)
}

func (a *App) StopScript(scriptID string) error {
	script, err := a.scriptMgr.GetScript(scriptID)
	if err != nil {
		return err
	}
	scriptPath := filepath.Join(a.config.Scripts.ScriptDir, script.Name)
	return a.scriptMgr.Executor.StopScript(scriptPath)
}

func (a *App) GetScripts() ([]ScriptInfo, error) {
	if a.scriptMgr == nil {
		return nil, errors.New("script manager not initialized")
	}

	scripts := a.scriptMgr.ListScripts()
	result := make([]ScriptInfo, 0, len(scripts))
	for _, s := range scripts {
		scriptPath := filepath.Join(a.config.Scripts.ScriptDir, s.Name)
		status := "idle"
		if a.scriptMgr.Executor.IsScriptRunning(scriptPath) {
			status = "running"
		}
		isInstalled, ok := a.scriptInstall[s.ID]
		if !ok {
			isInstalled = true
			a.scriptInstall[s.ID] = true
			if err := a.saveScriptInstallState(); err != nil {
				a.logger.Warn("failed to persist default script install state", zap.String("scriptID", s.ID), zap.Error(err))
			}
		}
		result = append(result, ScriptInfo{
			ID:          s.ID,
			Name:        s.Name,
			Description: s.Description,
			Author:      s.Author,
			Version:     s.Version,
			Size:        s.Size,
			Created:     s.Created.Format(time.RFC3339),
			Updated:     s.Updated.Format(time.RFC3339),
			Status:      status,
			IsInstalled: isInstalled,
		})
	}
	return result, nil
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

// GetHealthDiagnostics returns a structured snapshot of app dependencies for beta support.
func (a *App) GetHealthDiagnostics() AppHealthDiagnostics {
	status := a.GetServerStatus()
	diagnostics := AppHealthDiagnostics{
		GeneratedAt:          time.Now().UTC().Format(time.RFC3339),
		Status:               status,
		DatabaseURL:          redactConnectionString(a.config.Database.URL),
		RequiredTables:       make(map[string]bool),
		ScriptManagerRunning: status.ScriptMgrRunning,
		PythonRuntime:        a.config.Scripts.PythonPath,
		Security: SecurityHealth{
			RequestSigningRequired:  true,
			ResponseSigningRequired: true,
			PubSubStrictSigning:     true,
			KeyFileConfigured:       a.config.Security.KeyFile != "",
		},
	}
	if !a.startedAt.IsZero() {
		diagnostics.UptimeSeconds = int64(time.Since(a.startedAt).Seconds())
	}
	if a.config.Security.KeyFile != "" {
		if _, err := os.Stat(a.config.Security.KeyFile); err == nil {
			diagnostics.Security.KeyFileExists = true
		}
	}

	if a.peerHost != nil {
		diagnostics.P2PHostID = a.peerHost.ID().String()
		diagnostics.P2PListenAddresses = a.peerHost.FullAddrs()
		diagnostics.ConnectedPeers = a.peerHost.ConnectedPeers()
		metrics := a.peerHost.MetricsSnapshot()
		diagnostics.P2PMetrics = P2PMetrics{
			ConnectedPeers:    metrics.ConnectedPeers,
			TotalPeers:        metrics.TotalPeers,
			MessagesProcessed: metrics.MessagesProcessed,
			NetworkLatencyMs:  metrics.NetworkLatencyMs,
			RequestsReceived:  metrics.RequestsReceived,
			RequestsRejected:  metrics.RequestsRejected,
			AuthFailures:      metrics.AuthFailures,
			TransfersStarted:  metrics.TransfersStarted,
			TransfersComplete: metrics.TransfersComplete,
			TransfersFailed:   metrics.TransfersFailed,
			ChunksSent:        metrics.ChunksSent,
			ChunksReceived:    metrics.ChunksReceived,
			RowsSent:          metrics.RowsSent,
			RowsReceived:      metrics.RowsReceived,
			BytesSent:         metrics.BytesSent,
			BytesReceived:     metrics.BytesReceived,
			LastError:         metrics.LastError,
			LastRequestAt:     metrics.LastRequestAt,
			LastTransferAt:    metrics.LastTransferAt,
			LastUpdated:       metrics.LastUpdated,
		}
		diagnostics.Security.AuthFailures = metrics.AuthFailures
		if metrics.AuthFailures > 0 {
			diagnostics.Security.LastSecurityError = metrics.LastError
		}
	}

	requiredTables := []string{"market_data", "data_sources", "splits", "transfers", "scripts", "peers"}
	if a.conn != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		start := time.Now()
		if err := a.conn.Ping(ctx); err == nil {
			diagnostics.DatabaseLatencyMs = time.Since(start).Milliseconds()
		} else {
			diagnostics.OperationalWarnings = append(diagnostics.OperationalWarnings, fmt.Sprintf("database ping failed: %v", err))
		}
		for _, table := range requiredTables {
			var exists bool
			err := a.conn.QueryRow(ctx, `
                SELECT EXISTS (
                    SELECT 1
                    FROM information_schema.tables
                    WHERE table_schema = 'public' AND table_name = $1
                )`, table).Scan(&exists)
			diagnostics.RequiredTables[table] = err == nil && exists
			if err != nil || !exists {
				diagnostics.OperationalWarnings = append(diagnostics.OperationalWarnings, fmt.Sprintf("required table missing: %s", table))
			}
		}
	}

	if repo, ok := a.repo.(transferRepository); ok {
		transfers, err := repo.ListTransfers(a.ctx)
		if err == nil {
			for _, transfer := range transfers {
				switch transfer.Status {
				case "pending":
					diagnostics.TransferSummary.Pending++
				case "transferring":
					diagnostics.TransferSummary.Transferring++
				case "completed":
					diagnostics.TransferSummary.Completed++
				case "failed":
					diagnostics.TransferSummary.Failed++
				}
				if transfer.Status == "failed" && transfer.Error != "" {
					diagnostics.LatestTransferErrors = append(diagnostics.LatestTransferErrors,
						fmt.Sprintf("%s %s %s: %s", transfer.Type, transfer.Symbol, transfer.StartTime.Format(time.RFC3339), transfer.Error))
					if len(diagnostics.LatestTransferErrors) >= 5 {
						break
					}
				}
			}
		} else {
			diagnostics.OperationalWarnings = append(diagnostics.OperationalWarnings, fmt.Sprintf("transfer history unavailable: %v", err))
		}
	}

	if !status.DatabaseConnected {
		diagnostics.OperationalWarnings = append(diagnostics.OperationalWarnings, "database is not connected")
	}
	if !status.P2PHostRunning {
		diagnostics.OperationalWarnings = append(diagnostics.OperationalWarnings, "p2p host is not running")
	}
	if !diagnostics.Security.KeyFileConfigured || !diagnostics.Security.KeyFileExists {
		diagnostics.OperationalWarnings = append(diagnostics.OperationalWarnings, "security key file is not ready")
	}

	return diagnostics
}

func redactConnectionString(raw string) string {
	parsed, err := url.Parse(raw)
	if err != nil || parsed.User == nil {
		return raw
	}
	username := parsed.User.Username()
	if username == "" {
		parsed.User = url.UserPassword("redacted", "redacted")
	} else {
		parsed.User = url.UserPassword(username, "redacted")
	}
	return parsed.String()
}

func (a *App) SearchData(request data.DataRequest) ([]data.DataSource, error) {
	if a.repo == nil {
		return nil, errors.New("repository not initialized")
	}
	return a.repo.SearchData(a.ctx, request)
}

func (a *App) RequestData(peerID string, request data.DataRequest) error {
	transfers, ok := a.repo.(transferRepository)
	if !ok {
		return errors.New("transfer repository not available")
	}
	if request.RequestID == "" {
		request.RequestID = uuid.New().String()
	}
	if request.ChunkSize <= 0 {
		request.ChunkSize = 100
	}

	transfer := a.findResumableTransfer(transfers, peerID, request)
	if transfer == nil {
		transfer = &data.DataTransfer{
			ID:          request.TransferID,
			RequestID:   request.RequestID,
			Type:        request.Type,
			Symbol:      request.Symbol,
			Source:      peerID,
			Destination: "local",
			StartDate:   request.StartDate,
			EndDate:     request.EndDate,
			Granularity: request.Granularity,
			Progress:    0,
			Status:      "pending",
			StartTime:   time.Now().UTC(),
			ChunkSize:   request.ChunkSize,
		}
	} else {
		request.TransferID = transfer.ID
		request.RequestID = transfer.RequestID
		request.Offset = transfer.ResumeOffset
		if transfer.ChunkSize > 0 {
			request.ChunkSize = transfer.ChunkSize
		}
		transfer.Status = "pending"
		transfer.Error = ""
	}
	if transfer.ID == "" {
		transfer.ID = uuid.New().String()
	}
	if transfer.RequestID == "" {
		transfer.RequestID = request.RequestID
	}
	request.TransferID = transfer.ID
	if err := transfers.SaveTransfer(a.ctx, transfer); err != nil {
		return err
	}

	if peerID != "local" {
		if a.peerHost == nil {
			transfer.Status = "failed"
			transfer.Error = "peer host not initialized"
			transfer.EndTime = time.Now().UTC()
			_ = transfers.SaveTransfer(a.ctx, transfer)
			return errors.New(transfer.Error)
		}
		if err := a.peerHost.RequestData(a.ctx, peerID, request); err != nil {
			return err
		}
		return nil
	}

	transfer.Status = "completed"
	transfer.Progress = 100
	transfer.EndTime = time.Now().UTC()
	elapsed := transfer.EndTime.Sub(transfer.StartTime).Seconds()
	transfer.Size = estimateTransferSize(request)
	if elapsed > 0 {
		transfer.Speed = float64(transfer.Size) / elapsed
	}
	return transfers.SaveTransfer(a.ctx, transfer)
}

func (a *App) findResumableTransfer(repo transferRepository, peerID string, request data.DataRequest) *data.DataTransfer {
	if request.TransferID != "" {
		return nil
	}
	transfers, err := repo.ListTransfers(a.ctx)
	if err != nil {
		a.logger.Warn("failed to inspect transfers for resume", zap.Error(err))
		return nil
	}
	for _, transfer := range transfers {
		if transfer.Status != "failed" && transfer.Status != "transferring" && transfer.Status != "pending" {
			continue
		}
		if transfer.Source != peerID ||
			transfer.Type != request.Type ||
			transfer.Symbol != request.Symbol ||
			transfer.StartDate != request.StartDate ||
			transfer.EndDate != request.EndDate ||
			transfer.Granularity != request.Granularity {
			continue
		}
		if transfer.ResumeOffset <= 0 {
			continue
		}
		copyTransfer := transfer
		return &copyTransfer
	}
	return nil
}

func (a *App) GetInsiderData(symbol, startDate, endDate string) ([]data.InsiderTrade, error) {
	if a.repo == nil {
		return nil, errors.New("repository not initialized")
	}
	return a.repo.GetInsiderData(a.ctx, symbol, startDate, endDate)
}

func (a *App) GetSplitData(symbol, startDate, endDate string) ([]data.SplitData, error) {
	if a.repo == nil {
		return nil, errors.New("repository not initialized")
	}
	repo, ok := a.repo.(splitRepository)
	if !ok {
		return nil, errors.New("split repository not available")
	}
	return repo.GetSplitData(a.ctx, symbol, startDate, endDate)
}

func (a *App) GetActiveTransfers() ([]ActiveTransfer, error) {
	repo, ok := a.repo.(transferRepository)
	if !ok {
		return []ActiveTransfer{}, nil
	}
	transfers, err := repo.ListTransfers(a.ctx)
	if err != nil {
		return nil, err
	}
	result := make([]ActiveTransfer, 0, len(transfers))
	for _, transfer := range transfers {
		item := ActiveTransfer{
			ID:              transfer.ID,
			RequestID:       transfer.RequestID,
			Type:            transfer.Type,
			Symbol:          transfer.Symbol,
			Source:          transfer.Source,
			Destination:     transfer.Destination,
			Progress:        transfer.Progress,
			Status:          transfer.Status,
			StartTime:       transfer.StartTime.Format(time.RFC3339),
			Size:            transfer.Size,
			Speed:           transfer.Speed,
			Error:           transfer.Error,
			ChunkSize:       transfer.ChunkSize,
			TotalRows:       transfer.TotalRows,
			TotalChunks:     transfer.TotalChunks,
			CompletedChunks: transfer.CompletedChunks,
			ResumeOffset:    transfer.ResumeOffset,
		}
		if !transfer.EndTime.IsZero() {
			item.EndTime = transfer.EndTime.Format(time.RFC3339)
		}
		result = append(result, item)
	}
	return result, nil
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

	symbol := payloadString(payload, "symbol", "")
	source := payloadString(payload, "source", "manual_upload")
	dataType := payloadString(payload, "type", data.DataTypeEOD)

	switch dataType {
	case data.DataTypeDividend:
		return a.uploadDividendData(symbol, source, payload)
	case data.DataTypeSplit:
		return a.uploadSplitData(symbol, source, payload)
	default:
		return a.uploadEODMarketData(symbol, source, dataType, payload)
	}
}

func (a *App) uploadEODMarketData(symbol, source, dataType string, payload map[string]interface{}) error {
	price := payloadFloat(payload, "price", payloadFloat(payload, "close", 1))
	volume := payloadFloat(payload, "volume", 1)
	marketData, err := data.NewMarketData(symbol, price, volume, source, dataType)
	if err != nil {
		return err
	}
	if dateText := payloadString(payload, "date", ""); dateText != "" {
		if ts, err := time.Parse("2006-01-02", dateText); err == nil {
			marketData.Timestamp = ts
		}
	}
	open := payloadFloat(payload, "open", price)
	high := payloadFloat(payload, "high", price)
	low := payloadFloat(payload, "low", price)
	closePrice := payloadFloat(payload, "close", price)
	marketData.Price = closePrice
	marketData.MetaData["open"] = strconv.FormatFloat(open, 'f', -1, 64)
	marketData.MetaData["high"] = strconv.FormatFloat(high, 'f', -1, 64)
	marketData.MetaData["low"] = strconv.FormatFloat(low, 'f', -1, 64)
	marketData.MetaData["close"] = strconv.FormatFloat(closePrice, 'f', -1, 64)
	marketData.MetaData["adjusted_close"] = strconv.FormatFloat(payloadFloat(payload, "adjustedClose", closePrice), 'f', -1, 64)
	marketData.MetaData["volume"] = strconv.FormatFloat(volume, 'f', -1, 64)
	marketData.UpdateHash()
	return a.repo.SaveMarketData(a.ctx, marketData)
}

func (a *App) uploadDividendData(symbol, source string, payload map[string]interface{}) error {
	exDate, err := payloadDate(payload, "exDate", time.Now().UTC())
	if err != nil {
		return err
	}
	dividend := &data.DividendData{
		MarketDataBase: data.MarketDataBase{
			ID:        payloadString(payload, "id", fmt.Sprintf("%s-%s-dividend", symbol, exDate.Format("2006-01-02"))),
			Symbol:    symbol,
			Timestamp: exDate,
			Source:    source,
			DataType:  data.DataTypeDividend,
			Metadata:  map[string]string{},
		},
		Amount:    payloadFloat(payload, "amount", payloadFloat(payload, "price", 0)),
		Currency:  payloadString(payload, "currency", "USD"),
		ExDate:    exDate,
		Frequency: payloadString(payload, "frequency", ""),
		Type:      payloadString(payload, "dividendType", "cash"),
	}
	if dividend.Amount <= 0 {
		return errors.New("dividend amount must be positive")
	}
	return a.repo.SaveDividendData(a.ctx, dividend)
}

func (a *App) uploadSplitData(symbol, source string, payload map[string]interface{}) error {
	repo, ok := a.repo.(splitRepository)
	if !ok {
		return errors.New("split repository not available")
	}
	exDate, err := payloadDate(payload, "exDate", time.Now().UTC())
	if err != nil {
		return err
	}
	oldShares := int(payloadFloat(payload, "oldShares", 1))
	newShares := int(payloadFloat(payload, "newShares", payloadFloat(payload, "ratio", 2)))
	ratio := payloadFloat(payload, "ratio", 0)
	if ratio == 0 && oldShares > 0 {
		ratio = float64(newShares) / float64(oldShares)
	}
	split := &data.SplitData{
		MarketDataBase: data.MarketDataBase{
			ID:        payloadString(payload, "id", fmt.Sprintf("%s-%s-split", symbol, exDate.Format("2006-01-02"))),
			Symbol:    symbol,
			Timestamp: exDate,
			Source:    source,
			DataType:  data.DataTypeSplit,
			Metadata:  map[string]string{},
		},
		SplitRatio: ratio,
		ExDate:     exDate,
		OldShares:  oldShares,
		NewShares:  newShares,
		Status:     payloadString(payload, "status", "completed"),
	}
	if announcementDate, err := payloadDate(payload, "announcementDate", time.Time{}); err == nil {
		split.AnnouncementDate = announcementDate
	}
	return repo.SaveSplitData(a.ctx, split)
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
	if err := a.scriptMgr.AddScript(name, []byte(content), metadata); err != nil {
		return err
	}
	a.scriptInstall[metadata.ID] = true
	return a.saveScriptInstallState()
}

func (a *App) DeleteScript(scriptID string) error {
	if a.scriptMgr == nil {
		return errors.New("script manager not initialized")
	}
	delete(a.scriptInstall, scriptID)
	if err := a.scriptMgr.DeleteScript(scriptID); err != nil {
		return err
	}
	return a.saveScriptInstallState()
}

func (a *App) InstallScript(scriptID string) error {
	if a.scriptMgr == nil {
		return errors.New("script manager not initialized")
	}
	if _, err := a.scriptMgr.GetScript(scriptID); err != nil {
		return err
	}
	a.scriptInstall[scriptID] = true
	return a.saveScriptInstallState()
}

func (a *App) UninstallScript(scriptID string) error {
	if a.scriptMgr == nil {
		return errors.New("script manager not initialized")
	}
	if _, err := a.scriptMgr.GetScript(scriptID); err != nil {
		return err
	}
	if err := a.StopScript(scriptID); err != nil {
		a.logger.Warn("failed to stop script during uninstall", zap.String("scriptID", scriptID), zap.Error(err))
	}
	a.scriptInstall[scriptID] = false
	return a.saveScriptInstallState()
}

func estimateTransferSize(request data.DataRequest) int64 {
	days := 1
	start, startErr := time.Parse("2006-01-02", request.StartDate)
	end, endErr := time.Parse("2006-01-02", request.EndDate)
	if startErr == nil && endErr == nil && !end.Before(start) {
		days = int(end.Sub(start).Hours()/24) + 1
	}
	switch request.Granularity {
	case "WEEKLY":
		days = maxInt(1, days/7)
	case "MONTHLY":
		days = maxInt(1, days/30)
	case "YEARLY":
		days = maxInt(1, days/365)
	}
	switch request.Type {
	case data.DataTypeInsiderTrade:
		return int64(days * 384)
	case data.DataTypeDividend, data.DataTypeSplit:
		return int64(days * 128)
	default:
		return int64(days * 256)
	}
}

func payloadString(payload map[string]interface{}, key, fallback string) string {
	if value, ok := payload[key]; ok && value != nil {
		if text, ok := value.(string); ok && text != "" {
			return text
		}
		return fmt.Sprint(value)
	}
	return fallback
}

func payloadFloat(payload map[string]interface{}, key string, fallback float64) float64 {
	value, ok := payload[key]
	if !ok || value == nil {
		return fallback
	}
	switch v := value.(type) {
	case float64:
		return v
	case float32:
		return float64(v)
	case int:
		return float64(v)
	case int64:
		return float64(v)
	case string:
		if parsed, err := strconv.ParseFloat(v, 64); err == nil {
			return parsed
		}
	}
	return fallback
}

func payloadDate(payload map[string]interface{}, key string, fallback time.Time) (time.Time, error) {
	value := payloadString(payload, key, "")
	if value == "" {
		return fallback, nil
	}
	for _, layout := range []string{"2006-01-02", time.RFC3339} {
		parsed, err := time.Parse(layout, value)
		if err == nil {
			return parsed, nil
		}
	}
	return time.Time{}, fmt.Errorf("invalid %s date", key)
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
