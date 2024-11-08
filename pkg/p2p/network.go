 
package p2p

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/protocol"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/multiformats/go-multiaddr"
	"go.uber.org/zap"
)

// NetworkManager handles P2P network operations
type NetworkManager struct {
	host     *Host
	dht      *dht.IpfsDHT
	logger   *zap.Logger
	metrics  *NetworkMetrics
	mu       sync.RWMutex
}

// NetworkMetrics tracks network performance
type NetworkMetrics struct {
	ConnectedPeers     int
	DiscoveredPeers    int
	FailedConnections  int
	AverageLatency     time.Duration
	BandwidthIn        int64
	BandwidthOut       int64
	LastRefresh        time.Time
	mu                 sync.RWMutex
}

// NewNetworkManager creates a new network manager
func NewNetworkManager(host *Host, logger *zap.Logger) (*NetworkManager, error) {
	// Create DHT
	ctx := context.Background()
	dht, err := dht.New(ctx, host.host)
	if err != nil {
		return nil, fmt.Errorf("creating DHT: %w", err)
	}

	return &NetworkManager{
		host:    host,
		dht:     dht,
		logger:  logger,
		metrics: &NetworkMetrics{
			LastRefresh: time.Now(),
		},
	}, nil
}

// Start begins network operations
func (nm *NetworkManager) Start(ctx context.Context) error {
	// Bootstrap DHT
	if err := nm.dht.Bootstrap(ctx); err != nil {
		return fmt.Errorf("bootstrapping DHT: %w", err)
	}

	// Start peer discovery
	go nm.startPeerDiscovery(ctx)

	// Start metrics collection
	go nm.collectMetrics(ctx)

	// Start connection manager
	go nm.manageConnections(ctx)

	return nil
}

// Stop gracefully shuts down network operations
func (nm *NetworkManager) Stop() error {
	if err := nm.dht.Close(); err != nil {
		return fmt.Errorf("closing DHT: %w", err)
	}
	return nil
}

// ConnectToPeer establishes a connection to a peer
func (nm *NetworkManager) ConnectToPeer(ctx context.Context, peerInfo peer.AddrInfo) error {
	// Check if already connected
	if nm.host.host.Network().Connectedness(peerInfo.ID) == network.Connected {
		return nil
	}

	// Set connection timeout
	ctx, cancel := context.WithTimeout(ctx, connectionTimeout)
	defer cancel()

	// Attempt connection
	if err := nm.host.host.Connect(ctx, peerInfo); err != nil {
		nm.metrics.mu.Lock()
		nm.metrics.FailedConnections++
		nm.metrics.mu.Unlock()
		return fmt.Errorf("connecting to peer %s: %w", peerInfo.ID, err)
	}

	nm.metrics.mu.Lock()
	nm.metrics.ConnectedPeers++
	nm.metrics.mu.Unlock()

	nm.logger.Debug("Connected to peer",
		zap.String("peerID", peerInfo.ID.String()),
		zap.Any("addresses", peerInfo.Addrs))

	return nil
}

// DisconnectPeer closes the connection to a peer
func (nm *NetworkManager) DisconnectPeer(peerID peer.ID) error {
	if err := nm.host.host.Network().ClosePeer(peerID); err != nil {
		return fmt.Errorf("closing connection to peer %s: %w", peerID, err)
	}

	nm.metrics.mu.Lock()
	nm.metrics.ConnectedPeers--
	nm.metrics.mu.Unlock()

	return nil
}

// FindPeers searches for peers providing a specific service
func (nm *NetworkManager) FindPeers(ctx context.Context, serviceTag string) ([]peer.AddrInfo, error) {
	// Create context with timeout
	ctx, cancel := context.WithTimeout(ctx, time.Second*30)
	defer cancel()

	// Find peers
	peerChan, err := nm.dht.FindProviders(ctx, []byte(serviceTag))
	if err != nil {
		return nil, fmt.Errorf("finding providers: %w", err)
	}

	var peers []peer.AddrInfo
	for p := range peerChan {
		peers = append(peers, p)
	}

	nm.metrics.mu.Lock()
	nm.metrics.DiscoveredPeers = len(peers)
	nm.metrics.mu.Unlock()

	return peers, nil
}

// Private methods

func (nm *NetworkManager) startPeerDiscovery(ctx context.Context) {
	ticker := time.NewTicker(time.Minute * 5)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := nm.discoveryRound(ctx); err != nil {
				nm.logger.Error("Peer discovery failed",
					zap.Error(err))
			}
		}
	}
}

func (nm *NetworkManager) discoveryRound(ctx context.Context) error {
	// Find peers
	peers, err := nm.FindPeers(ctx, string(ProtocolID))
	if err != nil {
		return fmt.Errorf("finding peers: %w", err)
	}

	// Connect to new peers
	for _, p := range peers {
		if nm.shouldConnectToPeer(p.ID) {
			go func(peer peer.AddrInfo) {
				if err := nm.ConnectToPeer(ctx, peer); err != nil {
					nm.logger.Debug("Failed to connect to discovered peer",
						zap.String("peerID", peer.ID.String()),
						zap.Error(err))
				}
			}(p)
		}
	}

	return nil
}

func (nm *NetworkManager) shouldConnectToPeer(peerID peer.ID) bool {
	// Don't connect to self
	if peerID == nm.host.host.ID() {
		return false
	}

	// Check if already connected
	if nm.host.host.Network().Connectedness(peerID) == network.Connected {
		return false
	}

	// Check connection limit
	nm.metrics.mu.RLock()
	defer nm.metrics.mu.RUnlock()
	return nm.metrics.ConnectedPeers < nm.host.cfg.MaxPeers
}

func (nm *NetworkManager) manageConnections(ctx context.Context) {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			nm.pruneConnections()
			nm.ensureMinimumPeers(ctx)
		}
	}
}

func (nm *NetworkManager) pruneConnections() {
	connectedPeers := nm.host.host.Network().Peers()
	for _, peerID := range connectedPeers {
		// Check peer health
		if !nm.isPeerHealthy(peerID) {
			nm.logger.Debug("Pruning unhealthy peer connection",
				zap.String("peerID", peerID.String()))
			nm.DisconnectPeer(peerID)
		}
	}
}

func (nm *NetworkManager) ensureMinimumPeers(ctx context.Context) {
	connectedPeers := nm.host.host.Network().Peers()
	if len(connectedPeers) < nm.host.cfg.MinPeers {
		nm.logger.Debug("Below minimum peer threshold, discovering new peers",
			zap.Int("connected", len(connectedPeers)),
			zap.Int("minimum", nm.host.cfg.MinPeers))
		
		if err := nm.discoveryRound(ctx); err != nil {
			nm.logger.Error("Failed to discover new peers",
				zap.Error(err))
		}
	}
}

func (nm *NetworkManager) isPeerHealthy(peerID peer.ID) bool {
	// Get peer connection stats
	stats := nm.host.host.Network().ConnectionsToPeer(peerID)
	if len(stats) == 0 {
		return false
	}

	// Check last activity
	for _, conn := range stats {
		if time.Since(conn.Stat().LastActivity) > nm.host.cfg.PeerTimeout {
			return false
		}
	}

	// Check peer reputation
	if peer := nm.host.peerStore.GetPeer(peerID); peer != nil {
		if peer.Reputation < nm.host.cfg.MinReputation {
			return false
		}
	}

	return true
}

func (nm *NetworkManager) collectMetrics(ctx context.Context) {
	ticker := time.NewTicker(time.Second * 30)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			nm.updateMetrics()
		}
	}
}

func (nm *NetworkManager) updateMetrics() {
	nm.metrics.mu.Lock()
	defer nm.metrics.mu.Unlock()

	// Update connected peers count
	nm.metrics.ConnectedPeers = len(nm.host.host.Network().Peers())

	// Calculate average latency
	var totalLatency time.Duration
	var peerCount int
	for _, peer := range nm.host.host.Network().Peers() {
		latency := nm.host.host.Network().Latency(peer)
		if latency > 0 {
			totalLatency += latency
			peerCount++
		}
	}
	if peerCount > 0 {
		nm.metrics.AverageLatency = totalLatency / time.Duration(peerCount)
	}

	// Update bandwidth metrics
	stats := nm.host.host.Network().GetBandwidthStats()
	nm.metrics.BandwidthIn = stats.TotalIn
	nm.metrics.BandwidthOut = stats.TotalOut

	nm.metrics.LastRefresh = time.Now()
}

// GetNetworkStats returns current network statistics
func (nm *NetworkManager) GetNetworkStats() NetworkStats {
	nm.metrics.mu.RLock()
	defer nm.metrics.mu.RUnlock()

	return NetworkStats{
		ConnectedPeers:    nm.metrics.ConnectedPeers,
		DiscoveredPeers:   nm.metrics.DiscoveredPeers,
		FailedConnections: nm.metrics.FailedConnections,
		AverageLatency:    nm.metrics.AverageLatency,
		BandwidthIn:       nm.metrics.BandwidthIn,
		BandwidthOut:      nm.metrics.BandwidthOut,
		LastRefresh:       nm.metrics.LastRefresh,
	}
}

// NetworkStats represents network statistics
type NetworkStats struct {
	ConnectedPeers    int
	DiscoveredPeers   int
	FailedConnections int
	AverageLatency    time.Duration
	BandwidthIn       int64
	BandwidthOut      int64
	LastRefresh       time.Time
}

// DialPeer attempts to establish a connection with retries
func (nm *NetworkManager) DialPeer(ctx context.Context, peerInfo peer.AddrInfo) error {
	var lastErr error
	for i := 0; i < 3; i++ {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			if err := nm.ConnectToPeer(ctx, peerInfo); err != nil {
				lastErr = err
				time.Sleep(time.Second * time.Duration(i+1))
				continue
			}
			return nil
		}
	}
	return fmt.Errorf("failed to dial peer after retries: %w", lastErr)
}

// RegisterProtocol registers a new protocol handler
func (nm *NetworkManager) RegisterProtocol(protocolID protocol.ID, handler network.StreamHandler) {
	nm.host.host.SetStreamHandler(protocolID, handler)
}

// UnregisterProtocol removes a protocol handler
func (nm *NetworkManager) UnregisterProtocol(protocolID protocol.ID) {
	nm.host.host.RemoveStreamHandler(protocolID)
}

// GetPeerAddresses returns all known addresses for a peer
func (nm *NetworkManager) GetPeerAddresses(peerID peer.ID) []multiaddr.Multiaddr {
	return nm.host.host.Peerstore().Addrs(peerID)
}

// UpdatePeerAddresses updates the known addresses for a peer
func (nm *NetworkManager) UpdatePeerAddresses(peerID peer.ID, addrs []multiaddr.Multiaddr) {
	nm.host.host.Peerstore().AddAddrs(peerID, addrs, time.Hour*24)
}

// PeerLatency returns the latency to a specific peer
func (nm *NetworkManager) PeerLatency(peerID peer.ID) time.Duration {
	return nm.host.host.Network().Latency(peerID)
}

// IsConnected checks if a peer is currently connected
func (nm *NetworkManager) IsConnected(peerID peer.ID) bool {
	return nm.host.host.Network().Connectedness(peerID) == network.Connected
}