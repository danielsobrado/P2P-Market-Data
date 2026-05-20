package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"p2p_market_data/pkg/config"
	"p2p_market_data/pkg/data"
	"p2p_market_data/pkg/p2p/host"

	"go.uber.org/zap"
)

type server struct {
	name   string
	host   *host.Host
	repo   *data.MemoryRepository
	logger *zap.Logger
}

type statusResponse struct {
	Name          string   `json:"name"`
	PeerID        string   `json:"peer_id"`
	ListenAddrs   []string `json:"listen_addrs"`
	DialAddrs     []string `json:"dial_addrs"`
	Connected     []string `json:"connected_peers"`
	MarketRecords int      `json:"market_records"`
}

type connectRequest struct {
	Addr  string   `json:"addr"`
	Addrs []string `json:"addrs"`
}

type publishRequest struct {
	Symbol   string  `json:"symbol"`
	Price    float64 `json:"price"`
	Volume   float64 `json:"volume"`
	Source   string  `json:"source"`
	DataType string  `json:"data_type"`
}

func main() {
	logger, err := zap.NewProduction()
	if env("LOG_LEVEL", "info") == "debug" {
		logger, err = zap.NewDevelopment()
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "creating logger: %v\n", err)
		os.Exit(1)
	}
	defer logger.Sync()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	cfg := nodeConfig()
	repo := data.NewMemoryRepository()
	p2pHost, err := host.NewHost(ctx, cfg, logger, repo)
	if err != nil {
		logger.Fatal("Failed to create P2P host", zap.Error(err))
	}
	if err := p2pHost.Start(ctx); err != nil {
		logger.Fatal("Failed to start P2P host", zap.Error(err))
	}

	srv := &server{
		name:   env("NODE_NAME", "p2p-node"),
		host:   p2pHost,
		repo:   repo,
		logger: logger,
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", srv.health)
	mux.HandleFunc("GET /status", srv.status)
	mux.HandleFunc("POST /connect", srv.connect)
	mux.HandleFunc("POST /market-data", srv.publishMarketData)
	mux.HandleFunc("GET /market-data", srv.listMarketData)

	httpServer := &http.Server{
		Addr:              env("NODE_HTTP_ADDR", ":8080"),
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		logger.Info("HTTP API listening", zap.String("addr", httpServer.Addr))
		if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Fatal("HTTP API failed", zap.Error(err))
		}
	}()

	<-ctx.Done()
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = httpServer.Shutdown(shutdownCtx)
	_ = p2pHost.Stop()
}

func nodeConfig() *config.Config {
	p2pPort := envInt("P2P_PORT", 9000)
	keyFile := env("P2P_KEY_FILE", "/data/host.key")
	_ = os.MkdirAll(filepath.Dir(keyFile), 0700)

	return &config.Config{
		Environment: env("ENVIRONMENT", "production"),
		LogLevel:    env("LOG_LEVEL", "info"),
		P2P: config.P2PConfig{
			Port:             p2pPort,
			BootstrapPeers:   splitCSV(env("P2P_BOOTSTRAP_PEERS", "")),
			MaxPeers:         50,
			MinPeers:         1,
			PeerTimeout:      30 * time.Second,
			MinVoters:        1,
			ValidationQuorum: 0.5,
			VotingTimeout:    10 * time.Second,
		},
		Security: config.SecurityConfig{
			KeyFile:       keyFile,
			MaxPenalty:    0.5,
			MinConfidence: 0.5,
			TokenExpiry:   24 * time.Hour,
		},
	}
}

func (s *server) health(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *server) status(w http.ResponseWriter, r *http.Request) {
	items, err := s.repo.ListMarketData(r.Context(), data.MarketDataFilter{})
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	writeJSON(w, http.StatusOK, statusResponse{
		Name:          s.name,
		PeerID:        s.host.ID().String(),
		ListenAddrs:   s.host.Addrs(),
		DialAddrs:     s.host.FullAddrs(),
		Connected:     s.host.ConnectedPeers(),
		MarketRecords: len(items),
	})
}

func (s *server) connect(w http.ResponseWriter, r *http.Request) {
	var req connectRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	addrs := append([]string{}, req.Addrs...)
	if req.Addr != "" {
		addrs = append(addrs, req.Addr)
	}
	if len(addrs) == 0 {
		writeError(w, http.StatusBadRequest, errors.New("addr or addrs is required"))
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()
	for _, addr := range addrs {
		if err := s.host.Connect(ctx, addr); err != nil {
			writeError(w, http.StatusBadGateway, fmt.Errorf("connecting %s: %w", addr, err))
			return
		}
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"connected": s.host.ConnectedPeers(),
	})
}

func (s *server) publishMarketData(w http.ResponseWriter, r *http.Request) {
	var req publishRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if req.Source == "" {
		req.Source = s.name
	}
	if req.DataType == "" {
		req.DataType = data.DataTypeEOD
	}
	if req.Volume == 0 {
		req.Volume = 1
	}

	item, err := data.NewMarketData(req.Symbol, req.Price, req.Volume, req.Source, req.DataType)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if err := s.repo.SaveMarketData(r.Context(), item); err != nil && err != data.ErrDuplicate {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	if err := s.host.ShareData(r.Context(), item); err != nil {
		writeError(w, http.StatusBadGateway, err)
		return
	}

	writeJSON(w, http.StatusAccepted, item)
}

func (s *server) listMarketData(w http.ResponseWriter, r *http.Request) {
	limit := envIntFromString(r.URL.Query().Get("limit"), 100)
	items, err := s.repo.ListMarketData(r.Context(), data.MarketDataFilter{
		Symbol:   r.URL.Query().Get("symbol"),
		DataType: r.URL.Query().Get("data_type"),
		Limit:    limit,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, items)
}

func writeJSON(w http.ResponseWriter, status int, value interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func writeError(w http.ResponseWriter, status int, err error) {
	writeJSON(w, status, map[string]string{"error": err.Error()})
}

func env(key, fallback string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	return value
}

func envInt(key string, fallback int) int {
	return envIntFromString(os.Getenv(key), fallback)
}

func envIntFromString(value string, fallback int) int {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}
	return parsed
}

func splitCSV(value string) []string {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	parts := strings.Split(value, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			out = append(out, part)
		}
	}
	return out
}
