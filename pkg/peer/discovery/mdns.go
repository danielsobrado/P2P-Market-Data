// pkg/peer/discovery/mdns.go
package discovery

import (
	"context"
	"sync"
	"time"

	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/p2p/discovery/mdns"
	"go.uber.org/zap"
)

const (
	discoveryInterval = 1 * time.Minute
	serviceTag        = "_p2p-market-data._udp"
)

// MDNSDiscovery handles local network peer discovery using mDNS
type MDNSDiscovery struct {
	host    host.Host
	logger  *zap.Logger
	service mdns.Service

	// Discovery state
	localPeers map[peer.ID]time.Time
	peerCh     chan peer.AddrInfo
	mu         sync.RWMutex

	// Control
	ctx     context.Context
	cancel  context.CancelFunc
	running bool
}

// NewMDNSDiscovery creates a new MDNS discovery service
func NewMDNSDiscovery(h host.Host, logger *zap.Logger) (*MDNSDiscovery, error) {
	ctx, cancel := context.WithCancel(context.Background())

	md := &MDNSDiscovery{
		host:       h,
		logger:     logger,
		localPeers: make(map[peer.ID]time.Time),
		peerCh:     make(chan peer.AddrInfo),
		ctx:        ctx,
		cancel:     cancel,
	}

	return md, nil
}

// Start begins local peer discovery
func (m *MDNSDiscovery) Start() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.running {
		return nil
	}

	// Create MDNS service
	m.service = mdns.NewMdnsService(m.host, serviceTag, m)

	// Start discovery loop
	go m.discoveryLoop()

	m.running = true
	m.logger.Info("MDNS discovery started",
		zap.String("service_tag", serviceTag))
	return nil
}

// Stop halts local peer discovery
func (m *MDNSDiscovery) Stop() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.running {
		return nil
	}

	if m.service != nil {
		m.service.Close()
	}

	m.cancel()
	m.running = false
	m.logger.Info("MDNS discovery stopped")
	return nil
}

// HandlePeerFound implements the mdns.Notifee interface
func (m *MDNSDiscovery) HandlePeerFound(peerInfo peer.AddrInfo) {
	// Skip self-discovery
	if peerInfo.ID == m.host.ID() {
		return
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	// Update peer last seen time
	m.localPeers[peerInfo.ID] = time.Now()

	// Try to send peer info to channel
	select {
	case m.peerCh <- peerInfo:
	default:
		m.logger.Debug("Peer channel full, skipping peer",
			zap.String("peer", peerInfo.ID.String()))
	}
}

// discoveryLoop periodically checks for discovered peers
func (m *MDNSDiscovery) discoveryLoop() {
	ticker := time.NewTicker(discoveryInterval)
	defer ticker.Stop()

	for {
		select {
		case <-m.ctx.Done():
			return
		case <-ticker.C:
			m.cleanupStalePeers()
		case peer := <-m.peerCh:
			// Try to connect to discovered peer
			go m.connectToPeer(peer)
		}
	}
}

// cleanupStalePeers removes peers not seen recently
func (m *MDNSDiscovery) cleanupStalePeers() {
	m.mu.Lock()
	defer m.mu.Unlock()

	now := time.Now()
	staleTimeout := 10 * time.Minute

	for id, lastSeen := range m.localPeers {
		if now.Sub(lastSeen) > staleTimeout {
			delete(m.localPeers, id)
			m.logger.Debug("Removed stale peer",
				zap.String("peer", id.String()))
		}
	}
}

// connectToPeer attempts to connect to a discovered peer
func (m *MDNSDiscovery) connectToPeer(peerInfo peer.AddrInfo) {
	ctx, cancel := context.WithTimeout(m.ctx, 10*time.Second)
	defer cancel()

	if err := m.host.Connect(ctx, peerInfo); err != nil {
		m.logger.Debug("Failed to connect to discovered peer",
			zap.String("peer", peerInfo.ID.String()),
			zap.Error(err))
		return
	}

	m.logger.Debug("Connected to discovered peer",
		zap.String("peer", peerInfo.ID.String()))
}

// GetLocalPeers returns all currently known local peers
func (m *MDNSDiscovery) GetLocalPeers() []peer.ID {
	m.mu.RLock()
	defer m.mu.RUnlock()

	peers := make([]peer.ID, 0, len(m.localPeers))
	for id := range m.localPeers {
		peers = append(peers, id)
	}
	return peers
}

// IsRunning returns the current state of the discovery service
func (m *MDNSDiscovery) IsRunning() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.running
}

// GetConnectedPeers returns a list of currently connected peers discovered via mDNS
func (md *MDNSDiscovery) GetConnectedPeers() []peer.ID {
    if (!md.running) {
        return nil
    }

    md.mu.RLock()
    defer md.mu.RUnlock()

    result := make([]peer.ID, 0, len(md.localPeers))
    for peerID := range md.localPeers {
        // Skip self
        if (peerID == md.host.ID()) {
            continue
        }
        // Only include if still connected
        if (md.host.Network().Connectedness(peerID) == network.Connected) {
            result = append(result, peerID)
        }
    }

    return result
}

// IsConnected checks if a specific peer is connected
func (md *MDNSDiscovery) IsConnected(id peer.ID) bool {
    if (!md.running) {
        return false
    }
    md.mu.RLock()
    _, exists := md.localPeers[id]
    md.mu.RUnlock()
    return exists && md.host.Network().Connectedness(id) == network.Connected
}
