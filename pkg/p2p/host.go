package p2p

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"go.uber.org/zap"

	"p2p_market_data/pkg/config"
	"p2p_market_data/pkg/data"
	"p2p_market_data/pkg/security"
)

const (
	// Protocol IDs
	ProtocolID             = "/p2p/market-data/1.0.0"
	DataValidationProtocol = "/p2p/validation/1.0.0"
	PeerDiscoveryProtocol  = "/p2p/discovery/1.0.0"

	// Topic names
	MarketDataTopic    = "market-data"
	ValidationTopic    = "validation"
	PeerDiscoveryTopic = "peer-discovery"

	// Timeouts
	connectionTimeout = 30 * time.Second
	handshakeTimeout  = 10 * time.Second
	validationTimeout = 20 * time.Second
)

// Host manages the P2P network functionality
type Host struct {
	cfg        *config.P2PConfig
	host       host.Host
	pubsub     *pubsub.PubSub
	topics     map[string]*pubsub.Topic
	subs       map[string]*pubsub.Subscription
	peerStore  *PeerStore
	validators map[string]*security.Validator
	logger     *zap.Logger

	// Channels for coordination
	shutdown   chan struct{}
	msgQueue   chan *Message
	validation chan *ValidationRequest

	// Metrics and state
	metrics *Metrics
	status  *Status
	mu      sync.RWMutex
}

// PeerStore manages peer information and reputation
type PeerStore struct {
	peers map[peer.ID]*data.Peer
	repo  data.Repository
	mu    sync.RWMutex
}

// Metrics tracks P2P network performance
type Metrics struct {
	ConnectedPeers    int64
	MessagesProcessed int64
	ValidationLatency time.Duration
	NetworkLatency    time.Duration
	FailedValidations int64
	mu                sync.RWMutex
}

// Status represents the current state of the P2P host
type Status struct {
	IsReady      bool
	IsValidating bool
	LastError    error
	StartTime    time.Time
	UpdatedAt    time.Time
}

// NewHost creates a new P2P host
func NewHost(ctx context.Context, cfg *config.P2PConfig, repo data.Repository, logger *zap.Logger) (*Host, error) {
	// Generate or load host key
	priv, err := loadOrGenerateKey(cfg.KeyPath)
	if err != nil {
		return nil, fmt.Errorf("key management error: %w", err)
	}

	// Create libp2p host
	opts := []libp2p.Option{
		libp2p.Identity(priv),
		libp2p.ListenAddrStrings(fmt.Sprintf("/ip4/0.0.0.0/tcp/%d", cfg.Port)),
		libp2p.EnableRelay(),
		libp2p.EnableAutoRelay(),
		libp2p.EnableNATService(),
	}

	h, err := libp2p.New(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create libp2p host: %w", err)
	}

	// Initialize pubsub
	ps, err := pubsub.NewGossipSub(ctx, h)
	if err != nil {
		h.Close()
		return nil, fmt.Errorf("failed to create pubsub: %w", err)
	}

	host := &Host{
		cfg:        cfg,
		host:       h,
		pubsub:     ps,
		topics:     make(map[string]*pubsub.Topic),
		subs:       make(map[string]*pubsub.Subscription),
		peerStore:  newPeerStore(repo),
		validators: make(map[string]*security.Validator),
		logger:     logger,
		shutdown:   make(chan struct{}),
		msgQueue:   make(chan *Message, 1000),
		validation: make(chan *ValidationRequest, 100),
		metrics:    &Metrics{},
		status: &Status{
			StartTime: time.Now(),
			UpdatedAt: time.Now(),
		},
	}

	// Initialize topics and subscriptions
	if err := host.initializeTopics(ctx); err != nil {
		h.Close()
		return nil, fmt.Errorf("failed to initialize topics: %w", err)
	}

	// Set up protocol handlers
	host.setupProtocolHandlers()

	return host, nil
}

// Start begins P2P network operations
func (h *Host) Start(ctx context.Context) error {
	h.logger.Info("Starting P2P host",
		zap.String("peerID", h.host.ID().String()),
		zap.Any("listenAddrs", h.host.Addrs()))

	// Start peer discovery
	go h.startPeerDiscovery(ctx)

	// Start message processor
	go h.processMessages(ctx)

	// Start validation worker
	go h.processValidationRequests(ctx)

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
		topic.Close()
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
	msg := &Message{
		Type:      MarketDataMessage,
		Data:      marketData,
		Timestamp: time.Now(),
		SenderID:  h.host.ID(),
	}

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

	h.metrics.mu.Lock()
	h.metrics.MessagesProcessed++
	h.metrics.mu.Unlock()

	h.logger.Debug("Market data shared successfully",
		zap.String("symbol", marketData.Symbol),
		zap.Float64("price", marketData.Price))

	return nil
}

// RequestValidation initiates data validation process
func (h *Host) RequestValidation(ctx context.Context, marketData *data.MarketData) (*ValidationResult, error) {
	req := &ValidationRequest{
		MarketData: marketData,
		ResponseCh: make(chan *ValidationResult, 1),
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

// Private methods

func (h *Host) initializeTopics(ctx context.Context) error {
	topics := []string{MarketDataTopic, ValidationTopic, PeerDiscoveryTopic}

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

		// Start topic handler
		go h.handleTopicMessages(ctx, topicName, sub)
	}

	return nil
}

func (h *Host) setupProtocolHandlers() {
	h.host.SetStreamHandler(ProtocolID, h.handleStream)
	h.host.SetStreamHandler(DataValidationProtocol, h.handleValidationStream)
	h.host.SetStreamHandler(PeerDiscoveryProtocol, h.handleDiscoveryStream)
}

func (h *Host) handleStream(stream network.Stream) {
	// Basic stream handling implementation
	defer stream.Close()

	peer := stream.Conn().RemotePeer()
	h.logger.Debug("Received stream",
		zap.String("protocol", stream.Protocol()),
		zap.String("peer", peer.String()))

	// Handle stream based on protocol
	// Implementation details...
}

func (h *Host) processMessages(ctx context.Context) {
	for {
		select {
		case msg := <-h.msgQueue:
			if err := h.handleMessage(ctx, msg); err != nil {
				h.logger.Error("Failed to handle message",
					zap.Error(err),
					zap.String("type", string(msg.Type)))
			}
		case <-h.shutdown:
			return
		case <-ctx.Done():
			return
		}
	}
}

func (h *Host) handleMessage(ctx context.Context, msg *Message) error {
	// Verify message signature
	if err := h.verifyMessage(msg); err != nil {
		return fmt.Errorf("message verification failed: %w", err)
	}

	switch msg.Type {
	case MarketDataMessage:
		return h.handleMarketDataMessage(ctx, msg)
	case ValidationMessage:
		return h.handleValidationMessage(ctx, msg)
	default:
		return fmt.Errorf("unknown message type: %s", msg.Type)
	}
}
