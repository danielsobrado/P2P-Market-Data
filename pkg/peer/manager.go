// pkg/peer/manager.go
package peer

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/ipfs/go-cid"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	libp2pHost "github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	"go.uber.org/zap"
	"github.com/multiformats/go-multihash"

	"p2p_market_data/pkg/config"
)

const (
	discoveryInterval = 1 * time.Minute
	maxPeers          = 50
	minPeers          = 5
)

// PeerManager handles peer discovery and connections
type PeerManager struct {
	host   libp2pHost.Host
	dht    *dht.IpfsDHT
	logger *zap.Logger
	peers  map[peer.ID]*PeerInfo
	config *config.P2PConfig

	mu      sync.RWMutex
	ctx     context.Context
	cancel  context.CancelFunc
	running bool
}

// PeerInfo holds information about a peer
type PeerInfo struct {
	ID        peer.ID
	Addresses []string
	LastSeen  time.Time
	Connected bool
}

// NewPeerManager creates a new peer manager
func NewPeerManager(host libp2pHost.Host, cfg *config.P2PConfig, logger *zap.Logger) (*PeerManager, error) {
	ctx, cancel := context.WithCancel(context.Background())

	// Initialize DHT
	kadDHT, err := dht.New(ctx, host)
	if (err != nil) {
		cancel()
		return nil, fmt.Errorf("creating DHT: %w", err)
	}

	pm := &PeerManager{
		host:   host,
		dht:    kadDHT,
		logger: logger,
		config: cfg,
		peers:  make(map[peer.ID]*PeerInfo),
		ctx:    ctx,
		cancel: cancel,
	}

	return pm, nil
}

// Start begins peer discovery and management
func (pm *PeerManager) Start(ctx context.Context) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	if pm.running {
		return fmt.Errorf("peer manager already running")
	}

	// Bootstrap DHT
	if err := pm.dht.Bootstrap(ctx); err != nil {
		return fmt.Errorf("bootstrapping DHT: %w", err)
	}

	// Start discovery loop
	go pm.discoveryLoop()

	pm.running = true
	pm.logger.Info("Peer manager started")
	return nil
}

// Stop halts peer discovery and management
func (pm *PeerManager) Stop() error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	if !pm.running {
		return nil
	}

	pm.cancel()
	pm.running = false

	if err := pm.dht.Close(); err != nil {
		return fmt.Errorf("closing DHT: %w", err)
	}

	pm.logger.Info("Peer manager stopped")
	return nil
}

// discoveryLoop periodically looks for new peers
func (pm *PeerManager) discoveryLoop() {
	ticker := time.NewTicker(discoveryInterval)
	defer ticker.Stop()

	for {
		select {
		case <-pm.ctx.Done():
			return
		case <-ticker.C:
			if err := pm.findPeers(); err != nil {
				pm.logger.Error("Peer discovery failed", zap.Error(err))
			}
		}
	}
}

// findPeers discovers new peers
func (pm *PeerManager) findPeers() error {
	// Get peers from routing table
	peers := pm.dht.RoutingTable().ListPeers()
	
	// Find additional peers via content routing
	ctx, cancel := context.WithTimeout(pm.ctx, 30*time.Second)
	defer cancel()

	// Create CID from namespace
	nsHash, err := cid.NewPrefixV1(cid.Raw, multihash.SHA2_256).Sum([]byte("p2p-market-data"))
	if err != nil {
		return fmt.Errorf("creating namespace CID: %w", err)
	}

	// Find providers for our namespace and handle errors
	providers, err := pm.dht.FindProviders(ctx, nsHash)
	if err != nil {
		return fmt.Errorf("finding providers: %w", err)
	}

	// Add provider IDs to peers list
	for _, p := range providers {
		peers = append(peers, p.ID)
	}

	// Process discovered peers
	for _, peerID := range peers {
		if peerID == pm.host.ID() {
			continue // Skip self
		}

		// Update peer info
		func() {
			pm.mu.Lock()
			defer pm.mu.Unlock()
			pm.peers[peerID] = &PeerInfo{
				ID:        peerID,
				LastSeen:  time.Now(),
				Connected: false,
			}
		}()

		// Try to connect if under max peers
		if len(pm.GetConnectedPeers()) < maxPeers {
			peerInfo := pm.host.Peerstore().PeerInfo(peerID)
			go pm.connectToPeer(peerInfo)
		}
	}

	return nil
}

// connectToPeer attempts to connect to a peer
func (pm *PeerManager) connectToPeer(p peer.AddrInfo) {
	if err := pm.host.Connect(pm.ctx, p); err != nil {
		pm.logger.Debug("Failed to connect to peer",
			zap.String("peer", p.ID.String()),
			zap.Error(err))
		return
	}

	pm.mu.Lock()
	if peer, exists := pm.peers[p.ID]; exists {
		peer.Connected = true
		peer.LastSeen = time.Now()
	}
	pm.mu.Unlock()
}

// GetConnectedPeers returns the list of connected peers
func (pm *PeerManager) GetConnectedPeers() []peer.ID {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	var connected []peer.ID
	for id, info := range pm.peers {
		if info.Connected {
			connected = append(connected, id)
		}
	}
	return connected
}

// DisconnectPeer disconnects from a specific peer
func (pm *PeerManager) DisconnectPeer(id peer.ID) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	if peer, exists := pm.peers[id]; exists && peer.Connected {
		if err := pm.host.Network().ClosePeer(id); err != nil {
			return fmt.Errorf("closing connection to peer %s: %w", id, err)
		}
		peer.Connected = false
	}

	return nil
}

// IsPeerConnected checks if a peer is connected
func (pm *PeerManager) IsPeerConnected(id peer.ID) bool {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	if peer, exists := pm.peers[id]; exists {
		return peer.Connected
	}
	return false
}
