package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	postgres "github.com/fergusstrange/embedded-postgres"
	"github.com/jackc/pgx/v5"
	"github.com/libp2p/go-libp2p/core/peer"
	"go.uber.org/zap"

	"p2p_market_data/pkg/config"
	"p2p_market_data/pkg/data"
	"p2p_market_data/pkg/p2p/host"
	p2pHost "p2p_market_data/pkg/p2p/host"
	"p2p_market_data/pkg/scripts"
)

func initEmbeddedDB(ctx context.Context, logger *zap.Logger) (*pgx.Conn, error) {
	const (
		dbUser     = "postgres"
		dbPassword = "postgres"
		dbName     = "p2p_market_data"
		dbPort     = 5433
	)

	connString := fmt.Sprintf(
		"host=localhost port=%d user=%s password=%s dbname=%s sslmode=disable",
		dbPort, dbUser, dbPassword, dbName,
	)

	// Try connecting to existing instance first
	logger.Info("Attempting to connect to existing postgres instance")
	conn, err := pgx.Connect(ctx, connString)
	if err == nil {
		// Test connection
		if err := conn.Ping(ctx); err == nil {
			logger.Info("Connected to existing postgres instance")
			return conn, nil
		}
		conn.Close(ctx)
	}

	// Clean up any stale PID file
	pidFile := filepath.Join(".", "data", "postgres", "postmaster.pid")
	if _, err := os.Stat(pidFile); err == nil {
		content, err := os.ReadFile(pidFile)
		if err == nil {
			// Parse first line only for PID
			lines := strings.Split(string(content), "\n")
			if len(lines) > 0 {
				if pid, err := strconv.Atoi(strings.TrimSpace(lines[0])); err == nil {
					if process, err := os.FindProcess(pid); err == nil {
						logger.Info("Killing existing postgres process", zap.Int("pid", pid))
						_ = process.Kill()
						time.Sleep(time.Second) // Wait for process to die
					}
				}
			}
			os.Remove(pidFile)
		}
	}

	// Initialize new instance
	dbPath := filepath.Join(".", "data", "postgres")
	if err := os.MkdirAll(dbPath, 0700); err != nil {
		return nil, fmt.Errorf("creating db directory: %w", err)
	}

	dbConfig := postgres.DefaultConfig().
		Username(dbUser).
		Password(dbPassword).
		Database(dbName).
		Version(postgres.V13).
		DataPath(dbPath).
		Port(dbPort)

	embeddedDB := postgres.NewDatabase(dbConfig)

	// Try to start, but if port is in use, just try to connect
	if err := embeddedDB.Start(); err != nil {
		if strings.Contains(err.Error(), "already listening") {
			// Try connecting again
			conn, err := pgx.Connect(ctx, connString)
			if err == nil && conn.Ping(ctx) == nil {
				return conn, nil
			}
		}
		return nil, fmt.Errorf("starting embedded database: %w", err)
	}

	// Wait for database to be ready
	for retries := 3; retries > 0; retries-- {
		conn, err = pgx.Connect(ctx, connString)
		if err == nil {
			if err := conn.Ping(ctx); err == nil {
				return conn, nil
			}
			conn.Close(ctx)
		}
		logger.Warn("Failed to connect, retrying...",
			zap.Error(err),
			zap.Int("retriesLeft", retries-1))
		time.Sleep(time.Second)
	}

	return nil, fmt.Errorf("failed to connect to database after retries")
}

func initSchema(ctx context.Context, conn *pgx.Conn) error {
	schema := `
    CREATE TABLE IF NOT EXISTS market_data (
        id TEXT PRIMARY KEY,
        symbol TEXT NOT NULL,
        price DECIMAL NOT NULL,
        timestamp TIMESTAMP NOT NULL,
        source TEXT NOT NULL,
        data_type TEXT NOT NULL,
        validation_score DECIMAL NOT NULL
    );
    -- Add other tables as needed
    `

	_, err := conn.Exec(ctx, schema)
	return err
}

type App struct {
	ctx           context.Context
	logger        *zap.Logger
	scriptManager *scripts.ScriptManager
	networkMgr    *host.NetworkManager
	host          *p2pHost.Host
	cleanup       []func() error
	embeddedDB    *postgres.EmbeddedPostgres
	repository    data.Repository
}

func killExistingPostgres(pidFile string, logger *zap.Logger) error {
	// Read PID file if it exists
	content, err := os.ReadFile(pidFile)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("reading PID file: %w", err)
	}

	// Split content into lines
	lines := strings.Split(string(content), "\n")
	if len(lines) == 0 {
		return fmt.Errorf("PID file is empty")
	}

	// Parse PID from the first line
	pidStr := strings.TrimSpace(lines[0])
	pid, err := strconv.Atoi(pidStr)
	if err != nil {
		return fmt.Errorf("parsing PID: %w", err)
	}

	// Check if process exists
	process, err := os.FindProcess(pid)
	if err != nil {
		// Process not found, remove stale PID file
		_ = os.Remove(pidFile)
		return nil
	}

	// Kill process
	logger.Info("Killing existing postgres process", zap.Int("pid", pid))
	if err := process.Kill(); err != nil {
		return fmt.Errorf("killing process: %w", err)
	}

	// Remove PID file
	if err := os.Remove(pidFile); err != nil {
		return fmt.Errorf("removing PID file: %w", err)
	}

	return nil
}

func NewApp() *App {
	logger, _ := zap.NewProduction()
	ctx := context.Background()

	// Use system temp directory
	runtimePath := filepath.Join(os.TempDir(), "p2p_market_data_runtime")
	if err := os.RemoveAll(runtimePath); err != nil {
		logger.Warn("Failed to clean runtime directory", zap.Error(err))
	}

	// Create data directory
	dbPath := filepath.Join(".", "data", "postgres")
	if err := os.MkdirAll(dbPath, 0700); err != nil {
		logger.Fatal("Failed to create database directory", zap.Error(err))
	}

	// Kill existing postgres if running
	pidFile := filepath.Join(dbPath, "postmaster.pid")
	if err := killExistingPostgres(pidFile, logger); err != nil {
		logger.Warn("Failed to kill existing postgres", zap.Error(err))
	}

	// Configure embedded Postgres
	embeddedDBConfig := postgres.DefaultConfig().
		Username("postgres").
		Password("postgres").
		Database("p2p_market_data").
		Version(postgres.V13).
		DataPath(dbPath).
		Port(5433).
		RuntimePath(runtimePath)

	embeddedDB := postgres.NewDatabase(embeddedDBConfig)

	// Start with retries
	var startErr error
	for i := 0; i < 3; i++ {
		if err := embeddedDB.Start(); err != nil {
			startErr = err
			logger.Warn("Failed to start DB, retrying...", zap.Error(err))
			time.Sleep(2 * time.Second)
			continue
		}
		startErr = nil
		break
	}
	if startErr != nil {
		logger.Fatal("Failed to start embedded database", zap.Error(startErr))
	}

	// Repository DB Config - use same port as embedded DB
	repoDBConfig := &config.DatabaseConfig{
		Type:     "postgres",
		URL:      buildPostgresURL("localhost", 5433, "postgres", "postgres", "p2p_market_data", "disable"),
		MaxConns: 10,
		Timeout:  30 * time.Second,
		SSLMode:  "disable",
	}

	// Initialize repository
	repo, err := data.NewPostgresRepository(ctx, repoDBConfig.URL, logger)
	if err != nil {
		logger.Fatal("Failed to create repository", zap.Error(err))
	}

	// Initialize script configuration
	scriptConfig := &config.ScriptConfig{
		ScriptDir:   "./scripts",
		PythonPath:  "python3",
		MaxExecTime: time.Minute,
		MaxMemoryMB: 512,
		AllowedPkgs: []string{"pandas", "numpy"},
	}

	// Initialize script manager
	scriptManager, err := scripts.NewScriptManager(scriptConfig, logger)
	if err != nil {
		logger.Fatal("Failed to create ScriptManager", zap.Error(err))
	}

	// Initialize P2P host
	hostConfig := &config.Config{
		P2P: config.P2PConfig{
			Port: 4001, // Set your desired port
		},
		Security: config.SecurityConfig{
			KeyFile:       "./keyfile", // Path to your key file
			MaxPenalty:    0.5,         // Valid value between 0 and 1
			MinConfidence: 0.7,         // Valid value between 0 and 1
		},
	}

	hostNode, err := p2pHost.NewHost(ctx, hostConfig, logger, repo)
	if err != nil {
		logger.Fatal("Failed to create P2P Host", zap.Error(err))
	}

	// Initialize Network Manager
	networkMgr, err := host.NewNetworkManager(hostNode, logger)
	if err != nil {
		logger.Fatal("Failed to create NetworkManager", zap.Error(err))
	}

	// Create app with all fields initialized
	app := &App{
		ctx:           ctx,
		logger:        logger,
		scriptManager: scriptManager,
		networkMgr:    networkMgr,
		host:          hostNode,
		repository:    repo,
		embeddedDB:    embeddedDB,
		cleanup: []func() error{
			embeddedDB.Stop,
		},
	}

	return app
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	// Start the P2P host
	if err := a.host.Start(ctx); err != nil {
		a.logger.Fatal("Failed to start P2P Host", zap.Error(err))
	}
}

func (a *App) domReady(ctx context.Context) {
	// Logic to run when the frontend DOM is ready
}

func (a *App) shutdown(ctx context.Context) {
	// Clean up resources
	if err := a.scriptManager.Stop(); err != nil {
		a.logger.Error("Failed to stop ScriptManager", zap.Error(err))
	}
	if err := a.networkMgr.Close(); err != nil {
		a.logger.Error("Failed to close NetworkManager", zap.Error(err))
	}
	if err := a.host.Close(); err != nil {
		a.logger.Error("Failed to close P2P Host", zap.Error(err))
	}
}

func (a *App) beforeClose(ctx context.Context) (prevent bool) {
	// Logic to run before the application window closes
	return false
}

// Implementing the methods required by handleScripts.ts

// GetScriptContent returns the content of a script by its ID
func (a *App) GetScriptContent(scriptID string) (string, error) {
	script, err := a.scriptManager.GetScript(scriptID)
	if err != nil {
		return "", err
	}

	scriptPath := filepath.Join(a.scriptManager.Config.ScriptDir, script.Name)
	content, err := os.ReadFile(scriptPath)
	if err != nil {
		return "", err
	}

	return string(content), nil
}

// ScriptUploadData represents the data required to upload a script
type ScriptUploadData struct {
	Name     string `json:"name"`
	DataType string `json:"dataType"`
	Content  string `json:"content"`
}

// UploadScript uploads a new script to the script manager
func (a *App) UploadScript(scriptData ScriptUploadData) error {
	contentBytes := []byte(scriptData.Content)
	metadata := &scripts.ScriptMetadata{
		Name: scriptData.Name,
		// You can set additional metadata fields here
	}

	err := a.scriptManager.AddScript(scriptData.Name, contentBytes, metadata)
	if err != nil {
		return err
	}

	return nil
}

// RunScript executes a script by its ID
func (a *App) RunScript(scriptID string) error {
	ctx := context.Background() // You can use a timeout or cancel context if needed
	_, err := a.scriptManager.ExecuteScript(ctx, scriptID, nil)
	if err != nil {
		return err
	}
	return nil
}

// StopScript stops a running script by its ID
func (a *App) StopScript(scriptID string) error {
	return a.scriptManager.Executor.StopScript(scriptID)
}

// DeleteScript removes a script by its ID
func (a *App) DeleteScript(scriptID string) error {
	return a.scriptManager.DeleteScript(scriptID)
}

// InstallScript installs the dependencies of a script by its ID
func (a *App) InstallScript(scriptID string) error {
	script, err := a.scriptManager.GetScript(scriptID)
	if err != nil {
		return err
	}

	for _, dep := range script.Dependencies {
		cmd := exec.Command(a.scriptManager.Config.PythonPath, "-m", "pip", "install", dep)
		err := cmd.Run()
		if err != nil {
			return fmt.Errorf("failed to install dependency %s: %w", dep, err)
		}
	}

	return nil
}

// UninstallScript uninstalls the dependencies of a script by its ID
func (a *App) UninstallScript(scriptID string) error {
	script, err := a.scriptManager.GetScript(scriptID)
	if err != nil {
		return err
	}

	for _, dep := range script.Dependencies {
		cmd := exec.Command(a.scriptManager.Config.PythonPath, "-m", "pip", "uninstall", "-y", dep)
		err := cmd.Run()
		if err != nil {
			return fmt.Errorf("failed to uninstall dependency %s: %w", dep, err)
		}
	}

	return nil
}

// Implementing the methods required by PeerManagement.tsx

// Peer represents a peer in the network (matching the frontend interface)
type Peer struct {
	ID          string   `json:"id"`
	Address     string   `json:"address"`
	Reputation  float64  `json:"reputation"`
	IsConnected bool     `json:"isConnected"`
	LastSeen    string   `json:"lastSeen"`
	Roles       []string `json:"roles"`
}

// GetPeers returns a list of connected peers
func (a *App) GetPeers() ([]Peer, error) {
	connectedPeerIDs := a.networkMgr.GetConnectedPeers()
	var peers []Peer

	for _, peerID := range connectedPeerIDs {
		// Get peer info from the host
		peerInfo, err := a.host.GetPeerInfo(peerID)
		if err != nil {
			a.logger.Warn("Failed to get peer info", zap.Error(err))
			continue
		}

		// Get peer data from the peer store
		peerData, exists := a.host.GetPeerstore().GetPeer(peerID)
		if !exists {
			continue
		}

		peers = append(peers, Peer{
			ID:          peerID.String(),
			Address:     peerInfo.Addrs[0].String(), // Assuming at least one address
			Reputation:  peerData.Reputation,
			IsConnected: true,
			LastSeen:    peerData.LastSeen.Format(time.RFC3339),
			Roles:       peerData.Roles,
		})
	}

	return peers, nil
}

// DisconnectPeer disconnects from a peer by its ID
func (a *App) DisconnectPeer(peerID string) error {
	pid, err := peer.Decode(peerID)
	if err != nil {
		return fmt.Errorf("invalid peer ID: %w", err)
	}

	err = a.host.DisconnectPeer(pid)
	if err != nil {
		return fmt.Errorf("failed to disconnect peer %s: %w", peerID, err)
	}
	return nil
}

// Helper methods for the App struct

// Executor returns the ScriptExecutor instance
func (a *App) Executor() *scripts.ScriptExecutor {
	return a.scriptManager.Executor
}

// Helper function to build PostgreSQL URL
func buildPostgresURL(host string, port int, user, password, dbname, sslmode string) string {
	return fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=%s",
		user, password, host, port, dbname, sslmode)
}

type Config struct {
	Database config.DatabaseConfig
	Script   config.ScriptConfig
	P2P      config.P2PConfig
	Security config.SecurityConfig
}

func (a *App) Close() error {
	for _, cleanup := range a.cleanup {
		if err := cleanup(); err != nil {
			a.logger.Error("Cleanup error", zap.Error(err))
		}
	}
	return nil
}

// Add to App struct methods
func (a *App) GetEODData(symbol, startDate, endDate string) ([]data.EODData, error) {
	return a.repository.GetEODData(a.ctx, symbol, startDate, endDate)
}

func (a *App) GetDividendData(symbol, startDate, endDate string) ([]data.DividendData, error) {
	return a.repository.GetDividendData(a.ctx, symbol, startDate, endDate)
}

func (a *App) GetInsiderData(symbol, startDate, endDate string) ([]data.InsiderTrade, error) {
	return a.repository.GetInsiderData(a.ctx, symbol, startDate, endDate)
}

func (a *App) GetDataSources() ([]data.DataSource, error) {
	return a.repository.GetDataSources(a.ctx)
}

func (a *App) SearchData(request data.DataRequest) ([]data.DataSource, error) {
	return a.repository.SearchData(a.ctx, request)
}

func (a *App) RequestData(peerId string, request data.DataRequest) error {
	return a.networkMgr.RequestData(a.ctx, peerId, request)
}

func (a *App) ResetDataConnection() error {
	return a.networkMgr.ResetConnection()
}

func (a *App) ResetDataProcessing() error {
	return a.networkMgr.ResetProcessing()
}

func (a *App) RetryConnection() error {
	return a.networkMgr.RetryConnection()
}
