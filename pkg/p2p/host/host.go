package host

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"p2p_market_data/pkg/config"
	"p2p_market_data/pkg/data"
	"p2p_market_data/pkg/p2p/message"
	"p2p_market_data/pkg/security"

	libp2p "github.com/libp2p/go-libp2p"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	libp2pHost "github.com/libp2p/go-libp2p/core/host"
	libp2pNetwork "github.com/libp2p/go-libp2p/core/network"
	libp2pPeer "github.com/libp2p/go-libp2p/core/peer"
	libp2pProtocol "github.com/libp2p/go-libp2p/core/protocol"
	"go.uber.org/zap"
)

const (
	// Protocol IDs
	ProtocolID             = "/p2p/market-data/1.0.0"
	DataValidationProtocol = "/p2p/validation/1.0.0"
	PeerDiscoveryProtocol  = "/p2p/discovery/1.0.0"

	// Topics
	MarketDataTopic    = "MarketDataTopic"
	ValidationTopic    = "ValidationTopic"
	PeerDiscoveryTopic = "PeerDiscoveryTopic"

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
	peerStore *PeerStore
	validator *security.Validator
	logger    *zap.Logger

	// Channels for coordination
	shutdown   chan struct{}
	msgQueue   chan *message.Message
	validation chan *message.ValidationRequest

	// Metrics and state
	metrics *Metrics
	status  *Status
	mu      sync.RWMutex

	networkMgr *NetworkManager

	ctx context.Context
}

// NewHost creates a new P2P host
func NewHost(ctx context.Context, cfg *config.Config, logger *zap.Logger, repo data.Repository) (*Host, error) {
	// Validate P2P configuration
	if err := validateP2PConfig(&cfg.P2P); err != nil {
		return nil, fmt.Errorf("invalid P2P configuration: %w", err)
	}

	// Ensure key directory exists
	keyDir := filepath.Dir(cfg.Security.KeyFile)
	if err := os.MkdirAll(keyDir, 0700); err != nil {
		return nil, fmt.Errorf("creating key directory: %w", err)
	}

	// Load or generate host key
	privKey, err := loadOrGenerateKey(cfg.Security.KeyFile)
	if err != nil {
		return nil, fmt.Errorf("key management error: %w", err)
	}

	// Create libp2p host with custom options
	opts := []libp2p.Option{
		libp2p.Identity(privKey),
		libp2p.ListenAddrStrings(fmt.Sprintf("/ip4/0.0.0.0/tcp/%d", cfg.P2P.Port)),
		libp2p.EnableRelay(),
		libp2p.NATPortMap(),
	}

	h, err := libp2p.New(opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create libp2p host: %w", err)
	}

	// Initialize pubsub with custom options
	pubsubOpts := []pubsub.Option{
		pubsub.WithMessageSigning(true),
		pubsub.WithStrictSignatureVerification(true),
	}
	ps, err := pubsub.NewGossipSub(ctx, h, pubsubOpts...)
	if err != nil {
		h.Close()
		return nil, fmt.Errorf("failed to create pubsub: %w", err)
	}

	// Initialize components
	validator, err := security.NewValidator(cfg.Security)
	if err != nil {
		h.Close()
		return nil, fmt.Errorf("failed to initialize validator: %w", err)
	}

	peerStore := NewPeerStore(repo)

	host := &Host{
		cfg:        &cfg.P2P,
		host:       h,
		pubsub:     ps,
		topics:     make(map[string]*pubsub.Topic),
		subs:       make(map[string]*pubsub.Subscription),
		peerStore:  peerStore,
		validator:  validator,
		logger:     logger,
		shutdown:   make(chan struct{}),
		msgQueue:   make(chan *message.Message, 1000),
		validation: make(chan *message.ValidationRequest, 100),
		metrics:    NewMetrics(),
		status:     NewStatus(),
		ctx:        ctx,
	}

	// Initialize host components
	if err := host.initializeTopics(ctx); err != nil {
		h.Close()
		return nil, fmt.Errorf("failed to initialize topics: %w", err)
	}

	host.setupProtocolHandlers()

	networkMgr, err := NewNetworkManager(host, logger)
	if err != nil {
		h.Close()
		return nil, fmt.Errorf("failed to initialize network manager: %w", err)
	}
	host.networkMgr = networkMgr

	logger.Info("P2P host initialized",
		zap.String("peerID", h.ID().String()),
		zap.Int("port", cfg.P2P.Port))

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

	// Start handling topic messages
	for topicName, sub := range h.subs {
		go h.handleTopicMessages(ctx, topicName, sub)
	}

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
	msg := message.NewMessage(message.MarketDataMessage, marketData)
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
func (h *Host) RequestValidation(ctx context.Context, marketData *data.MarketData) (*message.ValidationResult, error) {
	req := &message.ValidationRequest{
		MarketData: marketData,
		ResponseCh: make(chan *message.ValidationResult, 1),
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

// GetPeerInfo retrieves the address info of a peer by its ID
func (h *Host) GetPeerInfo(peerID libp2pPeer.ID) (libp2pPeer.AddrInfo, error) {
	addrs := h.host.Peerstore().Addrs(peerID)
	if len(addrs) == 0 {
		return libp2pPeer.AddrInfo{}, fmt.Errorf("no addresses found for peer %s", peerID)
	}
	return libp2pPeer.AddrInfo{
		ID:    peerID,
		Addrs: addrs,
	}, nil
}

// DisconnectPeer disconnects from a peer by its ID
func (h *Host) DisconnectPeer(peerID libp2pPeer.ID) error {
	return h.host.Network().ClosePeer(peerID)
}

// ID returns the peer ID of the host
func (h *Host) ID() libp2pPeer.ID {
	return h.host.ID()
}

// SetStreamHandler sets a handler for a specific protocol
func (h *Host) SetStreamHandler(pid string, handler libp2pNetwork.StreamHandler) {
	h.host.SetStreamHandler(libp2pProtocol.ID(pid), handler)
}

// RemoveStreamHandler removes a handler for a specific protocol
func (h *Host) RemoveStreamHandler(pid string) {
	h.host.RemoveStreamHandler(libp2pProtocol.ID(pid))
}

// Close shuts down the host
func (h *Host) Close() error {
	return h.Stop()
}

// Helper methods and functions

// initializeTopics initializes necessary pubsub topics
func (h *Host) initializeTopics(ctx context.Context) error {
	// Initialize necessary topics
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
	}

	return nil
}

// setupProtocolHandlers sets up handlers for specific protocols
func (h *Host) setupProtocolHandlers() {
	h.host.SetStreamHandler(libp2pProtocol.ID(PeerDiscoveryProtocol), h.handleDiscoveryStream)
	h.host.SetStreamHandler(libp2pProtocol.ID(DataValidationProtocol), h.handleValidationStream)
	h.host.SetStreamHandler(libp2pProtocol.ID(ProtocolID), h.handleStream)
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

// processMessages processes incoming pubsub messages
func (h *Host) processMessages(ctx context.Context) {
	// The actual message processing is handled in handleTopicMessages
	// This function can be used for additional processing if needed
}

// handleTopicMessages processes incoming messages from a topic
func (h *Host) handleTopicMessages(ctx context.Context, topicName string, sub *pubsub.Subscription) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-h.shutdown:
			return
		default:
			msg, err := sub.Next(ctx)
			if err != nil {
				if ctx.Err() != nil {
					return
				}
				h.logger.Warn("Error reading from subscription", zap.Error(err))
				continue
			}
			// Process the message
			h.processIncomingMessage(msg)
		}
	}
}

// processIncomingMessage deserializes and handles an incoming message
func (h *Host) processIncomingMessage(msg *pubsub.Message) {
	// Deserialize the message
	receivedMsg := &message.Message{}
	if err := receivedMsg.Unmarshal(msg.Data); err != nil {
		h.logger.Warn("Failed to unmarshal message", zap.Error(err))
		return
	}

	// Verify message signature
	if err := h.verifyMessage(receivedMsg); err != nil {
		h.logger.Warn("Failed to verify message signature", zap.Error(err))
		return
	}

	// Handle the message based on its type
	switch receivedMsg.Type {
	case message.MarketDataMessage:
		h.handleMarketDataMessage(receivedMsg)
		h.metrics.IncrementMessagesProcessed()
	case message.ValidationRequestMessage:
		h.handleValidationRequestMessage(receivedMsg)
	case message.ValidationResponseMessage:
		h.handleValidationResponseMessage(receivedMsg)
	default:
		h.logger.Warn("Unknown message type", zap.String("type", string(receivedMsg.Type)))
	}
}

// handleMarketDataMessage handles incoming market data messages
func (h *Host) handleMarketDataMessage(msg *message.Message) {
	marketData := &data.MarketData{}
	if err := msg.DecodeData(marketData); err != nil {
		h.logger.Warn("Invalid market data message payload", zap.Error(err))
		return
	}

	h.logger.Info("Received market data",
		zap.String("symbol", marketData.Symbol),
		zap.Float64("price", marketData.Price))

	// Optionally store or process the market data
}

// handleValidationRequestMessage handles incoming validation requests
func (h *Host) handleValidationRequestMessage(msg *message.Message) {
	// Implement handling of validation requests
}

// handleValidationResponseMessage handles incoming validation responses
func (h *Host) handleValidationResponseMessage(msg *message.Message) {
	// Implement handling of validation responses
}

// processValidationRequests processes validation requests from the channel
func (h *Host) processValidationRequests(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-h.shutdown:
			return
		case req := <-h.validation:
			go h.handleValidationRequest(ctx, req)
		}
	}
}

// handleValidationRequest handles a single validation request
func (h *Host) handleValidationRequest(ctx context.Context, req *message.ValidationRequest) {
	startTime := time.Now()

	// Perform validation using the validator
	isValid, score := h.validator.Validate(req.MarketData)

	result := &message.ValidationResult{
		MarketDataID: req.MarketData.ID,
		IsValid:      isValid,
		Score:        score,
		CompletedAt:  time.Now(),
		ValidatedBy:  []libp2pPeer.ID{h.host.ID()},
	}

	// Send result back to requester
	select {
	case req.ResponseCh <- result:
	case <-ctx.Done():
	}

	// Update metrics
	duration := time.Since(startTime)
	h.metrics.UpdateValidationLatency(duration)
	if !isValid {
		h.metrics.IncrementFailedValidations()
	}
}

// connectToBootstrapPeers connects to bootstrap peers
func (h *Host) connectToBootstrapPeers(ctx context.Context) error {
	// Implement bootstrap peer connection
	// For example, read a list of bootstrap peers from the config and connect to them
	return nil
}

// signMessage signs the message with the host's private key
func (h *Host) signMessage(msg *message.Message) error {
	// Serialize the message without the signature
	dataToSign, err := msg.MarshalWithoutSignature()
	if err != nil {
		return fmt.Errorf("failed to marshal message for signing: %w", err)
	}

	// Sign the data
	privKey := h.host.Peerstore().PrivKey(h.host.ID())
	signature, err := privKey.Sign(dataToSign)
	if err != nil {
		return fmt.Errorf("failed to sign message: %w", err)
	}

	msg.Signature = signature
	return nil
}

// verifyMessage verifies the message signature
func (h *Host) verifyMessage(msg *message.Message) error {
	// Serialize the message without the signature
	dataToVerify, err := msg.MarshalWithoutSignature()
	if err != nil {
		return fmt.Errorf("failed to marshal message for verification: %w", err)
	}

	// Get the public key of the sender
	pubKey := h.host.Peerstore().PubKey(msg.SenderID)
	if pubKey == nil {
		return fmt.Errorf("public key not found for sender: %s", msg.SenderID)
	}

	// Verify the signature
	if ok, err := pubKey.Verify(dataToVerify, msg.Signature); err != nil || !ok {
		return fmt.Errorf("signature verification failed")
	}

	return nil
}

// validateConfig validates the P2P configuration
func validateP2PConfig(cfg *config.P2PConfig) error {
	if cfg.Port <= 0 || cfg.Port > 65535 {
		return fmt.Errorf("invalid port number: %d", cfg.Port)
	}
	return nil
}

// GetTopic returns a pubsub topic by name
func (h *Host) GetTopic(topicName string) (*pubsub.Topic, error) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	topic, exists := h.topics[topicName]
	if !exists {
		return nil, fmt.Errorf("topic not found: %s", topicName)
	}

	return topic, nil
}

// StringToPeerID converts a string to a libp2p PeerID
func (h *Host) StringToPeerID(peerIDStr string) (libp2pPeer.ID, error) {
	return libp2pPeer.Decode(peerIDStr)
}
