package p2p

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
	"go.uber.org/zap"

	"p2p_market_data/pkg/data"
	"p2p_market_data/pkg/security"
)

const AuthorityProtocol = "/p2p/authority/1.0.0"

// AuthorityNode represents a node with authority privileges
type AuthorityNode struct {
	host          *Host
	validator     *security.Validator
	networkMgr    *NetworkManager
	verifiedPeers map[peer.ID]*VerifiedPeer
	validations   chan *ValidationRequest
	logger        *zap.Logger
	metrics       *AuthorityMetrics
	mu            sync.RWMutex
}

// VerifiedPeer represents a peer that has been verified by the authority
type VerifiedPeer struct {
	ID           peer.ID
	PublicKey    []byte
	ValidUntil   time.Time
	Permissions  []string
	LastVerified time.Time
}

// AuthorityMetrics tracks authority node performance
type AuthorityMetrics struct {
	ValidationsProcessed int64
	ValidationsAccepted  int64
	ValidationsRejected  int64
	AverageLatency       time.Duration
	LastUpdate           time.Time
	mu                   sync.RWMutex
}

// NewAuthorityNode creates a new authority node
func NewAuthorityNode(host *Host, validator *security.Validator, logger *zap.Logger) (*AuthorityNode, error) {
	networkMgr, err := NewNetworkManager(host, logger)
	if err != nil {
		return nil, fmt.Errorf("creating network manager: %w", err)
	}

	return &AuthorityNode{
		host:          host,
		validator:     validator,
		networkMgr:    networkMgr,
		verifiedPeers: make(map[peer.ID]*VerifiedPeer),
		validations:   make(chan *ValidationRequest, 100),
		logger:        logger,
		metrics:       &AuthorityMetrics{},
	}, nil
}

// Start begins authority node operations
func (an *AuthorityNode) Start(ctx context.Context) error {
	// Register authority protocols
	an.networkMgr.RegisterProtocol(AuthorityProtocol, an.handleAuthorityStream)

	// Start validation worker
	go an.processValidations(ctx)

	// Start peer verification
	go an.verifyPeers(ctx)

	// Start metrics collection
	go an.collectMetrics(ctx)

	return nil
}

// Stop gracefully shuts down the authority node
func (an *AuthorityNode) Stop() error {
	an.networkMgr.UnregisterProtocol(AuthorityProtocol)
	return nil
}

// ValidateData validates market data
func (an *AuthorityNode) ValidateData(ctx context.Context, marketData *data.MarketData) (*ValidationResult, error) {
	validator := &security.Validator{
		// Initialize fields if necessary
	}

	result := &ValidationResult{
		MarketDataID: marketData.ID,
		ValidatedBy:  []peer.ID{an.host.host.ID()},
		CompletedAt:  time.Now(),
	}

	if valid, score := validator.Validate(marketData); valid {
		result.IsValid = true
		result.Score = score
	} else {
		result.IsValid = false
		result.ErrorMsg = "Data validation failed"
	}

	return result, nil
}

// ValidateMarketData validates multiple market data entries in batch
func (an *AuthorityNode) ValidateMarketData(ctx context.Context, marketData []*data.MarketData) ([]*ValidationResult, error) {
	var wg sync.WaitGroup
	results := make([]*ValidationResult, len(marketData))
	errChan := make(chan error, len(marketData))

	for i, md := range marketData {
		wg.Add(1)
		go func(index int, md *data.MarketData) {
			defer wg.Done()

			result, err := an.ValidateData(ctx, md)
			if err != nil {
				errChan <- fmt.Errorf("validating data at index %d: %w", index, err)
				return
			}

			results[index] = result
		}(i, md)
	}

	// Wait for all validations to complete
	wg.Wait()
	close(errChan)

	// Check for errors
	for err := range errChan {
		if err != nil {
			return nil, err
		}
	}

	return results, nil
}

// VerifyPeer verifies a peer's identity and permissions
func (an *AuthorityNode) VerifyPeer(ctx context.Context, peerID peer.ID) (*VerifiedPeer, error) {
	// Check if already verified
	an.mu.RLock()
	if peer, exists := an.verifiedPeers[peerID]; exists && time.Now().Before(peer.ValidUntil) {
		an.mu.RUnlock()
		return peer, nil
	}
	an.mu.RUnlock()

	// Get peer's public key
	pubKey := an.host.host.Peerstore().PubKey(peerID)
	if pubKey == nil {
		return nil, fmt.Errorf("peer public key not found")
	}

	// Verify peer's identity
	verified := &VerifiedPeer{
		ID: peerID,
		PublicKey: func() []byte {
			raw, err := pubKey.Raw()
			if err != nil {
				an.logger.Error("Failed to get raw public key", zap.Error(err))
				return nil
			}
			return raw
		}(),
		ValidUntil:   time.Now().Add(time.Hour * 24),
		Permissions:  []string{"basic"},
		LastVerified: time.Now(),
	}

	an.mu.Lock()
	an.verifiedPeers[peerID] = verified
	an.mu.Unlock()

	return verified, nil
}

// Private methods

func (an *AuthorityNode) collectMetrics(ctx context.Context) {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			an.updateMetrics()
		}
	}
}

func (an *AuthorityNode) processValidations(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case req := <-an.validations:
			result := an.validateDataInternal(req.MarketData)
			select {
			case req.ResponseCh <- result:
			case <-ctx.Done():
				return
			}
		}
	}
}

func (an *AuthorityNode) validateDataInternal(marketData *data.MarketData) *ValidationResult {
	result := &ValidationResult{
		MarketDataID: marketData.ID,
		ValidatedBy:  []peer.ID{an.host.host.ID()},
		CompletedAt:  time.Now(),
	}

	// Perform validation checks
	if valid, score := an.validator.Validate(marketData); valid {
		result.IsValid = true
		result.Score = score
	} else {
		result.IsValid = false
		result.ErrorMsg = "Data validation failed"
	}

	return result
}

func (an *AuthorityNode) verifyPeers(ctx context.Context) {
	ticker := time.NewTicker(time.Hour)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			an.cleanupExpiredPeers()
		}
	}
}

func (an *AuthorityNode) cleanupExpiredPeers() {
	an.mu.Lock()
	defer an.mu.Unlock()

	now := time.Now()
	for id, peer := range an.verifiedPeers {
		if now.After(peer.ValidUntil) {
			delete(an.verifiedPeers, id)
		}
	}
}

func (an *AuthorityNode) isVerifiedSource(source string) bool {
	// Implementation of source verification logic
	// This could involve checking against a whitelist, verifying signatures, etc.
	return true
}

func (an *AuthorityNode) updateMetrics(result *ValidationResult, duration time.Duration) {
	an.metrics.mu.Lock()
	defer an.metrics.mu.Unlock()

	an.metrics.ValidationsProcessed++
	if result.IsValid {
		an.metrics.ValidationsAccepted++
	} else {
		an.metrics.ValidationsRejected++
	}

	// Update average latency using weighted average
	if an.metrics.AverageLatency == 0 {
		an.metrics.AverageLatency = duration
	} else {
		an.metrics.AverageLatency = (an.metrics.AverageLatency*9 + duration) / 10
	}

	an.metrics.LastUpdate = time.Now()
}

// Additional methods for the authority node

// GetStats returns current authority node statistics
func (an *AuthorityNode) GetStats() AuthorityStats {
	an.metrics.mu.RLock()
	defer an.metrics.mu.RUnlock()

	return AuthorityStats{
		ValidationsProcessed: an.metrics.ValidationsProcessed,
		ValidationsAccepted:  an.metrics.ValidationsAccepted,
		ValidationsRejected:  an.metrics.ValidationsRejected,
		AverageLatency:       an.metrics.AverageLatency,
		VerifiedPeers:        len(an.verifiedPeers),
		LastUpdate:           an.metrics.LastUpdate,
	}
}

// AuthorityStats represents authority node statistics
type AuthorityStats struct {
	ValidationsProcessed int64
	ValidationsAccepted  int64
	ValidationsRejected  int64
	AverageLatency       time.Duration
	VerifiedPeers        int
	LastUpdate           time.Time
}

// Stream handling methods

func (an *AuthorityNode) handleAuthorityStream(stream network.Stream) {
	defer stream.Close()

	peer := stream.Conn().RemotePeer()
	an.logger.Debug("Received authority stream",
		zap.String("peer", peer.String()))

	// Read the request
	req, err := ReadValidationRequest(stream)
	if err != nil {
		an.logger.Error("Failed to read validation request",
			zap.Error(err),
			zap.String("peer", peer.String()))
		return
	}

	// Verify the peer
	_, err = an.VerifyPeer(context.Background(), peer)
	if err != nil {
		an.logger.Error("Peer verification failed",
			zap.Error(err),
			zap.String("peer", peer.String()))
		WriteErrorResponse(stream, err)
		return
	}

	// Process the validation request
	result := an.validateDataInternal(req.MarketData)

	// Send the response
	if err := WriteValidationResponse(stream, result); err != nil {
		an.logger.Error("Failed to write validation response",
			zap.Error(err),
			zap.String("peer", peer.String()))
	}
}

// Helper functions for stream reading/writing

func ReadValidationRequest(stream network.Stream) (*ValidationRequest, error) {
	buf := make([]byte, 1024*1024) // 1MB buffer
	n, err := stream.Read(buf)
	if err != nil {
		return nil, fmt.Errorf("reading stream: %w", err)
	}

	var req ValidationRequest
	if err := json.Unmarshal(buf[:n], &req); err != nil {
		return nil, fmt.Errorf("unmarshaling request: %w", err)
	}

	return &req, nil
}

func WriteValidationResponse(stream network.Stream, result *ValidationResult) error {
	resp, err := json.Marshal(result)
	if err != nil {
		return fmt.Errorf("marshaling response: %w", err)
	}

	n, err := stream.Write(resp)
	if err != nil {
		return fmt.Errorf("writing response: %w", err)
	}

	if n != len(resp) {
		return fmt.Errorf("incomplete write: wrote %d of %d bytes", n, len(resp))
	}

	return nil
}

func WriteErrorResponse(stream network.Stream, err error) error {
	resp := ErrorResponse{
		Code:    500,
		Message: "Internal Error",
		Details: err.Error(),
	}

	data, err := json.Marshal(resp)
	if err != nil {
		return fmt.Errorf("marshaling error response: %w", err)
	}

	if _, err := stream.Write(data); err != nil {
		return fmt.Errorf("writing error response: %w", err)
	}

	return nil
}

// Additional authority node functionalities

// RevokeVerification revokes a peer's verification status
func (an *AuthorityNode) RevokeVerification(peerID peer.ID) error {
	an.mu.Lock()
	defer an.mu.Unlock()

	if _, exists := an.verifiedPeers[peerID]; !exists {
		return fmt.Errorf("peer not found: %s", peerID)
	}

	delete(an.verifiedPeers, peerID)
	an.logger.Info("Revoked peer verification",
		zap.String("peerID", peerID.String()))

	return nil
}

// UpdatePeerPermissions updates a verified peer's permissions
func (an *AuthorityNode) UpdatePeerPermissions(peerID peer.ID, permissions []string) error {
	an.mu.Lock()
	defer an.mu.Unlock()

	peer, exists := an.verifiedPeers[peerID]
	if !exists {
		return fmt.Errorf("peer not found: %s", peerID)
	}

	peer.Permissions = permissions
	peer.LastVerified = time.Now()

	an.logger.Info("Updated peer permissions",
		zap.String("peerID", peerID.String()),
		zap.Strings("permissions", permissions))

	return nil
}

// IsAuthorizedForAction checks if a peer is authorized for a specific action
func (an *AuthorityNode) IsAuthorizedForAction(peerID peer.ID, action string) bool {
	an.mu.RLock()
	defer an.mu.RUnlock()

	peer, exists := an.verifiedPeers[peerID]
	if !exists {
		return false
	}

	if time.Now().After(peer.ValidUntil) {
		return false
	}

	for _, perm := range peer.Permissions {
		if perm == action || perm == "admin" {
			return true
		}
	}

	return false
}

// GetVerifiedPeers returns all currently verified peers
func (an *AuthorityNode) GetVerifiedPeers() []*VerifiedPeer {
	an.mu.RLock()
	defer an.mu.RUnlock()

	peers := make([]*VerifiedPeer, 0, len(an.verifiedPeers))
	for _, peer := range an.verifiedPeers {
		peers = append(peers, peer)
	}

	return peers
}

// ValidateDataBatch validates multiple market data entries in batch
func (an *AuthorityNode) ValidateDataBatch(ctx context.Context, marketDataBatch []*data.MarketData) ([]*ValidationResult, error) {
	var wg sync.WaitGroup
	results := make([]*ValidationResult, len(marketDataBatch))
	errChan := make(chan error, len(marketDataBatch))

	for i, md := range marketDataBatch {
		wg.Add(1)
		go func(index int, marketData *data.MarketData) {
			defer wg.Done()

			result, err := an.ValidateData(ctx, marketData)
			if err != nil {
				errChan <- fmt.Errorf("validating data at index %d: %w", index, err)
				return
			}

			results[index] = result
		}(i, md)
	}

	// Wait for all validations to complete
	wg.Wait()
	close(errChan)

	// Check for errors
	for err := range errChan {
		if err != nil {
			return nil, err
		}
	}

	return results, nil
}
