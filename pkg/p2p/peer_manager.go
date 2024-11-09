package p2p

import (
	"context"
	"sync"
	"time"

	libp2pPeer "github.com/libp2p/go-libp2p-core/peer"
	libp2pDiscovery "github.com/libp2p/go-libp2p-discovery"
	"go.uber.org/zap"
)

// PeerManager handles peer discovery and management
type PeerManager struct {
	host        *Host
	logger      *zap.Logger
	discovery   *libp2pDiscovery.RoutingDiscovery
	mu          sync.Mutex
	discoverCtx context.Context
}

// NewPeerManager creates a new PeerManager
func NewPeerManager(host *Host, logger *zap.Logger) *PeerManager {
	routingDiscovery := libp2pDiscovery.NewRoutingDiscovery(host.host.Peerstore().Peerstore())
	return &PeerManager{
		host:      host,
		logger:    logger,
		discovery: routingDiscovery,
	}
}

// Discover uses mDNS and other mechanisms to find new peers
func (pm *PeerManager) Discover() ([]libp2pPeer.AddrInfo, error) {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	// Use a context with timeout for discovery
	ctx, cancel := context.WithTimeout(pm.host.ctx, discoveryTimeout)
	defer cancel()

	peerChan, err := pm.discovery.FindPeers(ctx, discoveryNamespace)
	if err != nil {
		return nil, err
	}

	var peers []libp2pPeer.AddrInfo
	for peerInfo := range peerChan {
		if peerInfo.ID == pm.host.host.ID() {
			continue // Skip self
		}
		peers = append(peers, peerInfo)
	}

	return peers, nil
}

// AddPeer adds a peer to the peer store
func (pm *PeerManager) AddPeer(peerInfo libp2pPeer.AddrInfo) {
	peer := &Peer{
		ID:        peerInfo.ID,
		Addresses: peerInfo.Addrs,
	}
	pm.host.peerStore.AddPeer(peer)
}

// RemovePeer removes a peer from the peer store
func (pm *PeerManager) RemovePeer(peerID libp2pPeer.ID) {
	pm.host.peerStore.RemovePeer(peerID)
}

// GetPeer retrieves a peer from the peer store
func (pm *PeerManager) GetPeer(peerID libp2pPeer.ID) (*Peer, bool) {
	return pm.host.peerStore.GetPeer(peerID)
}

const (
	discoveryTimeout   = 10 * time.Second
	discoveryNamespace = "p2p-market-data"
)
