package main

import (
	"context"
	"encoding/csv"
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	postgres "github.com/fergusstrange/embedded-postgres"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/libp2p/go-libp2p/core/peer"
	"go.uber.org/zap"

	"p2p_market_data/pkg/config"
	"p2p_market_data/pkg/data"
	"p2p_market_data/pkg/p2p/host"
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

type App struct {
	ctx           context.Context
	logger        *zap.Logger
	scriptManager *scripts.ScriptManager
	networkMgr    *host.NetworkManager
	host          *host.Host
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

	// Create data directories
	dataDir := "./data"
	keyDir := filepath.Join(dataDir, "keys")
	if err := os.MkdirAll(keyDir, 0700); err != nil {
		logger.Fatal("Failed to create key directory", zap.Error(err))
	}

	// Initialize configuration
	cfg := &config.Config{
		P2P: config.P2PConfig{
			Port:           4001,
			BootstrapPeers: []string{},
			MaxPeers:       50,
			MinPeers:       1,
		},
		Security: config.SecurityConfig{
			KeyFile:       filepath.Join(keyDir, "host.key"),
			MaxPenalty:    0.5,  // Set to a value between 0 and 1
			MinConfidence: 0.75, // Set to a value between 0 and 1
		},
	}

	// Initialize database
	conn, err := initEmbeddedDB(ctx, logger)
	if err != nil {
		logger.Fatal("Failed to initialize database", zap.Error(err))
	}

	// Initialize repository
	repo, err := data.NewPostgresRepository(ctx, conn, logger)
	if err != nil {
		logger.Fatal("Failed to create repository", zap.Error(err))
	}

	// Initialize P2P host
	p2pHost, err := host.NewHost(ctx, cfg, logger, repo)
	if err != nil {
		logger.Fatal("Failed to create P2P host", zap.Error(err))
	}

	// Initialize schema
	schemaManager := data.NewSchemaManager(conn)
	if err := schemaManager.InitializeSchema(ctx); err != nil {
		logger.Fatal("Failed to initialize schema", zap.Error(err))
	}

	// Initialize NetworkManager with the P2P host instance
	networkMgr, err := host.NewNetworkManager(p2pHost, logger)
	if err != nil {
		logger.Fatal("Failed to create NetworkManager", zap.Error(err))
	}

	app := &App{
		ctx:        ctx,
		logger:     logger,
		repository: repo,
		host:       p2pHost,
		networkMgr: networkMgr,
		cleanup: []func() error{
			func() error { return conn.Close(ctx) },
			p2pHost.Close,
		},
	}

	return app
}

func (a *App) startup(ctx context.Context) {
	// Start P2P host
	if err := a.host.Start(ctx); err != nil {
		a.logger.Error("Failed to start P2P host", zap.Error(err))
		return
	}

	// Start other services
	a.logger.Info("Application started successfully")
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
	// Close database connection and other resources before the application window closes
	for _, cleanupFunc := range a.cleanup {
		if err := cleanupFunc(); err != nil {
			a.logger.Error("Cleanup error", zap.Error(err))
		}
	}
	return false
}

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
	start, err := time.Parse("2006-01-02", startDate)
	if err != nil {
		return nil, fmt.Errorf("invalid start date format: %w", err)
	}
	end, err := time.Parse("2006-01-02", endDate)
	if err != nil {
		return nil, fmt.Errorf("invalid end date format: %w", err)
	}

	ptrData, err := a.repository.GetDividendData(a.ctx, symbol, start, end)
	if err != nil {
		return nil, err
	}

	result := make([]data.DividendData, len(ptrData))
	for i, d := range ptrData {
		result[i] = *d
	}
	return result, nil
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

// UploadMarketData handles market data upload from CSV files
func (a *App) UploadMarketData(formData map[string]interface{}) error {
	file, ok := formData["file"].(*multipart.FileHeader)
	if !ok {
		return fmt.Errorf("invalid file format")
	}

	source := formData["source"].(string)
	dataType := formData["type"].(string)

	// Open uploaded file
	f, err := file.Open()
	if err != nil {
		return fmt.Errorf("opening file: %w", err)
	}
	defer f.Close()

	// Parse CSV
	reader := csv.NewReader(f)
	header, err := reader.Read()
	if err != nil {
		return fmt.Errorf("reading CSV header: %w", err)
	}

	// Process rows based on data type
	for {
		row, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("reading CSV row: %w", err)
		}

		// Create market data based on type
		var baseData *data.MarketData
		switch dataType {
		case data.DataTypeInsiderTrade:
			insiderData := parseInsiderTradeData(header, row, source)
			baseData, err = data.NewMarketData(
				insiderData.Symbol,
				insiderData.PricePerShare,
				float64(insiderData.Shares),
				source,
				data.DataTypeInsiderTrade,
			)
			if err != nil {
				return fmt.Errorf("creating market data: %w", err)
			}
			// Add insider specific metadata
			baseData.MetaData["insider_name"] = insiderData.InsiderName
			baseData.MetaData["position"] = insiderData.Position
			baseData.MetaData["transaction_type"] = insiderData.TransactionType
			// ... other cases ...
		}

		if err := a.repository.SaveMarketData(a.ctx, baseData); err != nil {
			return fmt.Errorf("saving market data: %w", err)
		}
	}

	return nil
}

func parseEODData(header, row []string, source string) *data.EODData {
	fmt.Println(header)

	now := time.Now().UTC()
	base := data.MarketDataBase{
		ID:        uuid.New().String(),
		Symbol:    row[0],
		Source:    source,
		DataType:  data.DataTypeEOD,
		Timestamp: now,
	}

	return &data.EODData{
		MarketDataBase: base,
		Open:           parseFloat(row[1]),
		High:           parseFloat(row[2]),
		Low:            parseFloat(row[3]),
		Close:          parseFloat(row[4]),
		Volume:         parseFloat(row[5]),
		Date:           now,
	}
}

// Helper function to parse float values
func parseFloat(s string) float64 {
	f, _ := strconv.ParseFloat(s, 64)
	return f
}

func parseInsiderTradeData(header, row []string, source string) *data.InsiderTrade {
	fmt.Println(header)

	now := time.Now().UTC()
	base := data.MarketDataBase{
		ID:        uuid.New().String(),
		Symbol:    row[0],
		Source:    source,
		DataType:  data.DataTypeInsiderTrade,
		Timestamp: now,
	}

	return &data.InsiderTrade{
		MarketDataBase:  base,
		InsiderName:     row[1],
		InsiderTitle:    row[2],
		Position:        row[3],
		Shares:          parseInt(row[4]),
		PricePerShare:   parseFloat(row[5]),
		Value:           parseFloat(row[6]),
		TransactionType: row[7],
		TradeDate:       parseDate(row[8]),
	}
}

func parseInt(s string) int64 {
	i, _ := strconv.ParseInt(s, 10, 64)
	return i
}

func parseDate(s string) time.Time {
	t, _ := time.Parse("2006-01-02", s)
	return t
}

// App struct method
func (a *App) UploadFileData(ctx context.Context, formData map[string]interface{}) error {
	// Extract file, source, and dataType from formData
	fileHeader := formData["file"].(*multipart.FileHeader)
	source := formData["source"].(string)
	dataType := formData["type"].(string)

	// Open the uploaded file
	file, err := fileHeader.Open()
	if err != nil {
		return fmt.Errorf("opening uploaded file: %w", err)
	}
	defer file.Close()

	// Parse the file based on dataType
	switch dataType {
	case "eod":
		if err := a.processEODFile(ctx, file, source); err != nil {
			return err
		}
	case "insider_trades":
		if err := a.processInsiderTradesFile(ctx, file, source); err != nil {
			return err
		}
	default:
		return fmt.Errorf("unsupported data type: %s", dataType)
	}

	return nil
}

// processEODFile parses EOD data from CSV file and stores it in the repository
func (a *App) processEODFile(ctx context.Context, file io.Reader, source string) error {
	reader := csv.NewReader(file)

	// Read CSV header
	header, err := reader.Read()
	if err != nil {
		return fmt.Errorf("reading CSV header: %w", err)
	}

	// Read and process each record
	for {
		row, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("reading CSV row: %w", err)
		}

		// Parse the EOD data
		eodData := parseEODData(header, row, source)

		// Convert to MarketData
		marketData := &data.MarketData{
			ID:        eodData.ID,
			Symbol:    eodData.Symbol,
			Source:    eodData.Source,
			DataType:  eodData.DataType,
			Timestamp: eodData.Timestamp,
			MetaData: map[string]string{
				"open":   fmt.Sprintf("%.4f", eodData.Open),
				"high":   fmt.Sprintf("%.4f", eodData.High),
				"low":    fmt.Sprintf("%.4f", eodData.Low),
				"close":  fmt.Sprintf("%.4f", eodData.Close),
				"volume": fmt.Sprintf("%.0f", eodData.Volume),
				"date":   eodData.Date.Format(time.RFC3339),
			},
		}

		// Save to repository
		if err := a.repository.SaveMarketData(ctx, marketData); err != nil {
			a.logger.Error("Failed to save market data", zap.Error(err))
		}
	}

	return nil
}

// processInsiderTradesFile parses insider trades data and stores it
func (a *App) processInsiderTradesFile(ctx context.Context, file io.Reader, source string) error {
	reader := csv.NewReader(file)

	// Read CSV header
	header, err := reader.Read()
	if err != nil {
		return fmt.Errorf("reading CSV header: %w", err)
	}

	// Read and process each record
	for {
		row, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("reading CSV row: %w", err)
		}

		// Parse the insider trade data
		tradeData := parseInsiderTradeData(header, row, source)

		// Convert to MarketData and save to repository
		marketData := &data.MarketData{
			ID:        tradeData.ID,
			Symbol:    tradeData.Symbol,
			Source:    tradeData.Source,
			DataType:  tradeData.DataType,
			Timestamp: tradeData.Timestamp,
			MetaData: map[string]string{
				"insider_name":     tradeData.InsiderName,
				"insider_title":    tradeData.InsiderTitle,
				"position":         tradeData.Position,
				"shares":           fmt.Sprintf("%d", tradeData.Shares),
				"price_per_share":  fmt.Sprintf("%.2f", tradeData.PricePerShare),
				"value":            fmt.Sprintf("%.2f", tradeData.Value),
				"transaction_type": tradeData.TransactionType,
				"trade_date":       tradeData.TradeDate.Format(time.RFC3339),
			},
		}
		if err := a.repository.SaveMarketData(ctx, marketData); err != nil {
			a.logger.Error("Failed to save insider trade data", zap.Error(err))
		}
	}

	return nil
}
