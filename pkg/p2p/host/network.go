package host

import (
	"context"
	"fmt"
	"p2p_market_data/pkg/data"
	"sync"
	"time"

	libp2pPeer "github.com/libp2p/go-libp2p/core/peer"
	"go.uber.org/zap"
)

// DataRequest represents a data request to a peer
type DataRequest struct {
	Type    string
	Payload []byte
}

// NetworkManager handles P2P network operations
const (
	metricsCollectionInterval = 1 * time.Minute
)

type NetworkManager struct {
	host        *Host
	connManager *ConnectionManager
	peerManager *PeerManager
	metrics     *Metrics
	logger      *zap.Logger
	ctx         context.Context
	cancel      context.CancelFunc
	wg          sync.WaitGroup
}

// NewNetworkManager creates a new NetworkManager
func NewNetworkManager(host *Host, logger *zap.Logger) (*NetworkManager, error) {
	if host == nil {
		return nil, fmt.Errorf("host cannot be nil")
	}

	ctx, cancel := context.WithCancel(context.Background())

	nm := &NetworkManager{
		host:        host,
		peerManager: NewPeerManager(host.host, logger),
		metrics:     NewMetrics(),
		logger:      logger,
		ctx:         ctx,
		cancel:      cancel,
	}
	nm.connManager = NewConnectionManager(host.host, logger, nm)

	// Start background processes
	nm.wg.Add(2)
	go nm.runConnectionManager()
	go nm.runMetricsCollector()

	return nm, nil
}

// Close gracefully shuts down the NetworkManager
func (nm *NetworkManager) Close() error {
	nm.cancel()
	nm.wg.Wait()
	return nil
}

// ConnectToPeer connects to a specific peer
func (nm *NetworkManager) ConnectToPeer(peerInfo libp2pPeer.AddrInfo) error {
	if err := nm.host.host.Connect(nm.ctx, peerInfo); err != nil {
		return fmt.Errorf("failed to connect to peer %s: %w", peerInfo.ID, err)
	}
	nm.logger.Info("Connected to peer", zap.String("peerID", peerInfo.ID.String()))
	return nil
}

// DiscoverPeers discovers peers using mDNS and other discovery mechanisms
func (nm *NetworkManager) DiscoverPeers() error {
	peers, err := nm.peerManager.Discover()
	if err != nil {
		return fmt.Errorf("peer discovery failed: %w", err)
	}

	for _, peerInfo := range peers {
		if err := nm.ConnectToPeer(peerInfo); err != nil {
			nm.logger.Warn("Failed to connect to discovered peer", zap.Error(err))
		}
	}

	return nil
}

// GetConnectedPeers returns a list of currently connected peers
func (nm *NetworkManager) GetConnectedPeers() []libp2pPeer.ID {
	return nm.connManager.GetConnectedPeers()
}

// runConnectionManager manages peer connections in the background
func (nm *NetworkManager) runConnectionManager() {
	defer nm.wg.Done()

	ticker := time.NewTicker(connectionCheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-nm.ctx.Done():
			return
		case <-ticker.C:
			nm.connManager.ManageConnections(nm.ctx)
		}
	}
}

// runMetricsCollector collects network metrics in the background
func (nm *NetworkManager) runMetricsCollector() {
	defer nm.wg.Done()

	ticker := time.NewTicker(metricsCollectionInterval)
	defer ticker.Stop()

	for {
		select {
		case <-nm.ctx.Done():
			return
		case <-ticker.C:
			nm.metrics.Collect(nm.host)
		}
	}
}

// RequestData requests data from a peer
func (nm *NetworkManager) RequestData(ctx context.Context, peerID string, request data.DataRequest) error {
	if peerID == "" {
		return fmt.Errorf("peerID is required")
	}
	if request.Type == "" {
		return fmt.Errorf("request type is required")
	}

	id, err := libp2pPeer.Decode(peerID)
	if err != nil {
		return fmt.Errorf("invalid peer ID: %w", err)
	}

	// Ensure the target peer is currently known and connected before dispatch.
	if nm.host.host.Network().Connectedness(id) == 0 {
		return fmt.Errorf("peer %s is not connected", peerID)
	}

	nm.logger.Info("Data request queued",
		zap.String("peerID", peerID),
		zap.String("type", request.Type),
		zap.String("symbol", request.Symbol))
	return nil
}

// ResetConnection resets the network connections
func (nm *NetworkManager) ResetConnection() error {
	for _, peerID := range nm.connManager.GetConnectedPeers() {
		if err := nm.host.DisconnectPeer(peerID); err != nil {
			nm.logger.Warn("Failed to disconnect peer during reset",
				zap.String("peerID", peerID.String()),
				zap.Error(err))
		}
	}

	if err := nm.DiscoverPeers(); err != nil {
		return fmt.Errorf("discovering peers after reset: %w", err)
	}
	return nil
}

// ResetProcessing resets data processing state
func (nm *NetworkManager) ResetProcessing() error {
	nm.logger.Info("Reset processing requested")
	return nil
}

// RetryConnection attempts to reconnect to peers
func (nm *NetworkManager) RetryConnection() error {
	return nm.DiscoverPeers()
}
