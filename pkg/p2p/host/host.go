package host

import (
	"context"
	"fmt"
	"sync"
	"time"

	"p2p_market_data/pkg/config"
	"p2p_market_data/pkg/data"
	"p2p_market_data/pkg/p2p"
	"p2p_market_data/pkg/security"

	libp2p "github.com/libp2p/go-libp2p"
	libp2pHost "github.com/libp2p/go-libp2p-core/host"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"go.uber.org/zap"
)

const (
	// Protocol IDs
	ProtocolID             = "/p2p/market-data/1.0.0"
	DataValidationProtocol = "/p2p/validation/1.0.0"
	PeerDiscoveryProtocol  = "/p2p/discovery/1.0.0"

	// Topics
	MarketDataTopic = "MarketDataTopic"

	// Timeouts
	connectionTimeout = 30 * time.Second
	validationTimeout = 20 * time.Second
)

// Host manages the P2P network functionality
type Host struct {
	cfg       *config.P2PConfig
	host      libp2pHost.Host
	pubsub    *pubsub.PubSub
	topics    map[string]*pubsub.Topic
	subs      map[string]*pubsub.Subscription
	peerStore *p2p.PeerStore
	validator *security.Validator
	logger    *zap.Logger

	// Channels for coordination
	shutdown   chan struct{}
	msgQueue   chan *p2p.Message
	validation chan *p2p.ValidationRequest

	// Metrics and state
	metrics *p2p.Metrics
	status  *p2p.Status
	mu      sync.RWMutex

	// Add the networkMgr field
	networkMgr *p2p.NetworkManager

	ctx context.Context
}

// NewHost creates a new P2P host
func NewHost(ctx context.Context, cfg *config.Config, logger *zap.Logger, repo data.Repository) (*Host, error) {
	if err := validateConfig(&cfg.P2P); err != nil {
		return nil, fmt.Errorf("invalid P2P configuration: %w", err)
	}

	// Generate or load host key
	privKey, err := loadOrGenerateKey(cfg.Security.KeyFile)
	if err != nil {
		return nil, fmt.Errorf("key management error: %w", err)
	}

	// Create libp2p host
	h, err := libp2p.New(
		libp2p.Identity(privKey),
		libp2p.ListenAddrStrings(fmt.Sprintf("/ip4/0.0.0.0/tcp/%d", cfg.P2P.Port)),
		libp2p.EnableRelay(),
		libp2p.EnableAutoRelay(),
		libp2p.NATPortMap(),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create libp2p host: %w", err)
	}

	// Initialize pubsub
	ps, err := pubsub.NewGossipSub(ctx, h)
	if err != nil {
		h.Close()
		return nil, fmt.Errorf("failed to create pubsub: %w", err)
	}

	// Initialize validator
	validator, err := security.NewValidator(cfg.Security)
	if err != nil {
		h.Close()
		return nil, fmt.Errorf("failed to initialize validator: %w", err)
	}

	host := &Host{
		cfg:        &cfg.P2P,
		host:       h,
		pubsub:     ps,
		topics:     make(map[string]*pubsub.Topic),
		subs:       make(map[string]*pubsub.Subscription),
		peerStore:  p2p.NewPeerStore(repo),
		validator:  validator,
		logger:     logger,
		shutdown:   make(chan struct{}),
		msgQueue:   make(chan *p2p.Message, 1000),
		validation: make(chan *p2p.ValidationRequest, 100),
		metrics:    p2p.NewMetrics(),
		status:     p2p.NewStatus(),
		ctx:        ctx,
	}

	// Initialize topics and subscriptions
	if err := host.initializeTopics(ctx); err != nil {
		h.Close()
		return nil, fmt.Errorf("failed to initialize topics: %w", err)
	}

	// Set up protocol handlers
	host.setupProtocolHandlers()

	// Initialize NetworkManager
	networkMgr, err := p2p.NewNetworkManager(host, logger)
	if err != nil {
		h.Close()
		return nil, fmt.Errorf("failed to initialize network manager: %w", err)
	}
	host.networkMgr = networkMgr

	return host, nil
}

// Start begins P2P network operations
func (h *Host) Start(ctx context.Context) error {
	h.logger.Info("Starting P2P host",
		zap.String("peerID", h.host.ID().String()),
		zap.Any("listenAddrs", h.host.Addrs()))

	// Start background processes
	go h.processMessages(ctx)
	go h.processValidationRequests(ctx)
	go h.collectMetrics(ctx)

	// Connect to bootstrap peers
	if err := h.connectToBootstrapPeers(ctx); err != nil {
		h.logger.Warn("Failed to connect to some bootstrap peers", zap.Error(err))
	}

	h.mu.Lock()
	h.status.IsReady = true
	h.status.UpdatedAt = time.Now()
	h.mu.Unlock()

	return nil
}

// collectMetrics periodically collects metrics from the host
func (h *Host) collectMetrics(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-h.shutdown:
			return
		case <-ticker.C:
			h.metrics.Collect(h)
		}
	}
}

// Stop gracefully shuts down the P2P host
func (h *Host) Stop() error {
	h.logger.Info("Stopping P2P host")

	// Signal shutdown
	close(h.shutdown)

	// Close subscriptions
	for _, sub := range h.subs {
		sub.Cancel()
	}

	// Close topics
	for _, topic := range h.topics {
		if err := topic.Close(); err != nil {
			h.logger.Warn("Failed to close topic", zap.Error(err))
		}
	}

	// Close network manager
	if h.networkMgr != nil {
		if err := h.networkMgr.Close(); err != nil {
			h.logger.Warn("Failed to close network manager", zap.Error(err))
		}
	}

	// Close host
	if err := h.host.Close(); err != nil {
		return fmt.Errorf("failed to close libp2p host: %w", err)
	}

	h.logger.Info("P2P host stopped")
	return nil
}

// ShareData publishes market data to the network
func (h *Host) ShareData(ctx context.Context, marketData *data.MarketData) error {
	// Validate data
	if err := marketData.Validate(); err != nil {
		return fmt.Errorf("invalid market data: %w", err)
	}

	// Create message
	msg := p2p.NewMessage(p2p.MarketDataMessage, marketData)
	msg.SenderID = h.host.ID()

	// Sign message
	if err := h.signMessage(msg); err != nil {
		return fmt.Errorf("failed to sign message: %w", err)
	}

	// Publish to topic
	topic, ok := h.topics[MarketDataTopic]
	if !ok {
		return fmt.Errorf("market data topic not initialized")
	}

	msgBytes, err := msg.Marshal()
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	if err := topic.Publish(ctx, msgBytes); err != nil {
		h.logger.Error("Failed to publish market data",
			zap.Error(err),
			zap.String("symbol", marketData.Symbol))
		return fmt.Errorf("failed to publish message: %w", err)
	}

	h.metrics.IncrementMessagesProcessed()

	h.logger.Debug("Market data shared successfully",
		zap.String("symbol", marketData.Symbol),
		zap.Float64("price", marketData.Price))

	return nil
}

// RequestValidation initiates data validation process
func (h *Host) RequestValidation(ctx context.Context, marketData *data.MarketData) (*p2p.ValidationResult, error) {
	req := &p2p.ValidationRequest{
		MarketData: marketData,
		ResponseCh: make(chan *p2p.ValidationResult, 1),
		Timestamp:  time.Now(),
	}

	// Send validation request
	select {
	case h.validation <- req:
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-h.shutdown:
		return nil, fmt.Errorf("host is shutting down")
	}

	// Wait for response with timeout
	select {
	case result := <-req.ResponseCh:
		return result, nil
	case <-time.After(validationTimeout):
		return nil, fmt.Errorf("validation timeout")
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

func (h *Host) initializeTopics(ctx context.Context) error {
	// Initialize necessary topics
	topics := []string{"MarketDataTopic", "ValidationTopic"}
	for _, topicName := range topics {
		topic, err := h.pubsub.Join(topicName)
		if err != nil {
			return fmt.Errorf("failed to join topic %s: %w", topicName, err)
		}
		h.topics[topicName] = topic

		sub, err := topic.Subscribe()
		if err != nil {
			return fmt.Errorf("failed to subscribe to topic %s: %w", topicName, err)
		}
		h.subs[topicName] = sub
	}

	return nil
}

func (h *Host) setupProtocolHandlers() {
	// Implement protocol handlers as needed
}

func (h *Host) signMessage(msg *p2p.Message) error {
	// Implement message signing
	return nil
}

func (h *Host) processMessages(ctx context.Context) {
	// Implement message processing
}

func (h *Host) processValidationRequests(ctx context.Context) {
	// Implement validation request processing
}

func (h *Host) connectToBootstrapPeers(ctx context.Context) error {
	// Implement bootstrap peer connection
	return nil
}

// GetTopic returns a pubsub topic by name, creating it if it doesn't exist
func (h *Host) GetTopic(name string) (*pubsub.Topic, error) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if topic, exists := h.topics[name]; exists {
		return topic, nil
	}

	// Create new topic if it doesn't exist
	topic, err := h.pubsub.Join(name)
	if err != nil {
		return nil, fmt.Errorf("failed to join topic %s: %w", name, err)
	}

	h.topics[name] = topic
	return topic, nil
}
