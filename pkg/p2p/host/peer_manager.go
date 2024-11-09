package host

import (
	"context"
	"sync"
	"time"

	"github.com/libp2p/go-libp2p-core/host"
	libp2pPeer "github.com/libp2p/go-libp2p-core/peer"
	libp2pDiscovery "github.com/libp2p/go-libp2p-discovery"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/multiformats/go-multiaddr"
	"go.uber.org/zap"
)

// Peer represents a peer in the network
type Peer struct {
	ID        libp2pPeer.ID
	Addresses []multiaddr.Multiaddr
}

// PeerManager handles peer discovery and management
type PeerManager struct {
	host      *host.Host
	logger    *zap.Logger
	discovery *libp2pDiscovery.RoutingDiscovery
	mu        sync.Mutex
	ctx       context.Context
}

// NewPeerManager creates a new PeerManager
func NewPeerManager(host *host.Host, logger *zap.Logger) *PeerManager {
	kadDHT, _ := dht.New(context.Background(), *host)
	routingDiscovery := libp2pDiscovery.NewRoutingDiscovery(kadDHT)
	return &PeerManager{
		host:      host,
		logger:    logger,
		discovery: routingDiscovery,
		ctx:       context.Background(),
	}
}

// Discover uses mDNS and other mechanisms to find new peers
func (pm *PeerManager) Discover() ([]libp2pPeer.AddrInfo, error) {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	ctx, cancel := context.WithTimeout(pm.ctx, discoveryTimeout)
	defer cancel()

	peerChan, err := pm.discovery.FindPeers(ctx, discoveryNamespace)
	if err != nil {
		return nil, err
	}

	var peers []libp2pPeer.AddrInfo
	for peerInfo := range peerChan {
		if peerInfo.ID == (*pm.host).ID() {
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
	(*pm.host).Peerstore().AddAddrs(peer.ID, peer.Addresses, time.Hour)
}

// RemovePeer removes a peer from the peer store
func (pm *PeerManager) RemovePeer(peerID libp2pPeer.ID) {
	(*pm.host).Peerstore().RemovePeer(peerID)
}

// GetPeer retrieves a peer from the peer store
func (pm *PeerManager) GetPeer(peerID libp2pPeer.ID) (*Peer, bool) {
	addrs := (*pm.host).Peerstore().Addrs(peerID)
	if len(addrs) == 0 {
		return nil, false
	}
	return &Peer{
		ID:        peerID,
		Addresses: addrs,
	}, true
}

const (
	discoveryTimeout   = 10 * time.Second
	discoveryNamespace = "p2p-market-data"
)
