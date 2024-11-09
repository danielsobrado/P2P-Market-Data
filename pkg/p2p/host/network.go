package host

import (
	"context"
	"fmt"
	"sync"
	"time"

	libp2pPeer "github.com/libp2p/go-libp2p-core/peer"
	"go.uber.org/zap"
)

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
