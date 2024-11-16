// pkg/peer/discovery/bootstrap.go
package discovery

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/libp2p/go-libp2p/core/host"
	libp2ppeer "github.com/libp2p/go-libp2p/core/peer"
	"github.com/multiformats/go-multiaddr"
	"go.uber.org/zap"
)

const (
	bootstrapTimeout  = 30 * time.Second
	reconnectInterval = 10 * time.Minute
	maxRetries        = 3
)

// BootstrapDiscovery handles peer discovery through bootstrap nodes
type BootstrapDiscovery struct {
	host           host.Host
	logger         *zap.Logger
	bootstrapPeers []libp2ppeer.AddrInfo
	connected      map[libp2ppeer.ID]time.Time
	mu             sync.RWMutex
	ctx            context.Context
	cancel         context.CancelFunc
	running        bool
}

// NewBootstrapDiscovery creates a new bootstrap discovery service
func NewBootstrapDiscovery(h host.Host, bootstrapAddrs []string, logger *zap.Logger) (*BootstrapDiscovery, error) {
	ctx, cancel := context.WithCancel(context.Background())

	bd := &BootstrapDiscovery{
		host:      h,
		logger:    logger,
		connected: make(map[libp2ppeer.ID]time.Time),
		ctx:       ctx,
		cancel:    cancel,
	}

	// Parse bootstrap addresses
	if err := bd.parseBootstrapAddrs(bootstrapAddrs); err != nil {
		cancel()
		return nil, err
	}

	return bd, nil
}

// parseBootstrapAddrs converts multiaddr strings to peer info
func (bd *BootstrapDiscovery) parseBootstrapAddrs(addrs []string) error {
	bd.bootstrapPeers = make([]libp2ppeer.AddrInfo, 0, len(addrs))

	for _, addr := range addrs {
		ma, err := multiaddr.NewMultiaddr(addr)
		if err != nil {
			return fmt.Errorf("invalid bootstrap address %s: %w", addr, err)
		}

		peerInfo, err := libp2ppeer.AddrInfoFromP2pAddr(ma)
		if err != nil {
			return fmt.Errorf("invalid peer address %s: %w", addr, err)
		}

		bd.bootstrapPeers = append(bd.bootstrapPeers, *peerInfo)
	}

	return nil
}

// Start begins bootstrap peer discovery
func (bd *BootstrapDiscovery) Start() error {
	bd.mu.Lock()
	defer bd.mu.Unlock()

	if bd.running {
		return nil
	}

	// Start initial bootstrap
	if err := bd.bootstrap(); err != nil {
		return fmt.Errorf("initial bootstrap failed: %w", err)
	}

	// Start reconnection loop
	go bd.reconnectLoop()

	bd.running = true
	bd.logger.Info("Bootstrap discovery started",
		zap.Int("bootstrap_peers", len(bd.bootstrapPeers)))
	return nil
}

// Stop halts bootstrap peer discovery
func (bd *BootstrapDiscovery) Stop() error {
	bd.mu.Lock()
	defer bd.mu.Unlock()

	if !bd.running {
		return nil
	}

	bd.cancel()
	bd.running = false

	// Disconnect from bootstrap peers
	for id := range bd.connected {
		if err := bd.host.Network().ClosePeer(id); err != nil {
			bd.logger.Debug("Error disconnecting from peer",
				zap.String("peer", id.String()),
				zap.Error(err))
		}
	}

	bd.logger.Info("Bootstrap discovery stopped")
	return nil
}

// bootstrap attempts to connect to bootstrap peers
func (bd *BootstrapDiscovery) bootstrap() error {
	if len(bd.bootstrapPeers) == 0 {
		return fmt.Errorf("no bootstrap peers configured")
	}

	ctx, cancel := context.WithTimeout(bd.ctx, bootstrapTimeout)
	defer cancel()

	var wg sync.WaitGroup
	for _, peer := range bd.bootstrapPeers {
		wg.Add(1)
		go func(p libp2ppeer.AddrInfo) {
			defer wg.Done()
			if err := bd.connectToBootstrapPeer(ctx, p); err != nil {
				bd.logger.Debug("Failed to connect to bootstrap peer",
					zap.String("peer", p.ID.String()),
					zap.Error(err))
			}
		}(peer)
	}

	// Wait for connection attempts
	wg.Wait()

	// Check if we connected to any peers
	bd.mu.RLock()
	connectedCount := len(bd.connected)
	bd.mu.RUnlock()

	if connectedCount == 0 {
		return fmt.Errorf("failed to connect to any bootstrap peers")
	}

	return nil
}

// connectToBootstrapPeer attempts to connect to a single bootstrap peer
func (bd *BootstrapDiscovery) connectToBootstrapPeer(ctx context.Context, peerInfo libp2ppeer.AddrInfo) error {
	// Don't connect to self
	if peerInfo.ID == bd.host.ID() {
		return nil
	}

	// Try to connect with retries
	var err error
	for retry := 0; retry < maxRetries; retry++ {
		if err = bd.host.Connect(ctx, peerInfo); err == nil {
			bd.mu.Lock()
			bd.connected[peerInfo.ID] = time.Now()
			bd.mu.Unlock()

			bd.logger.Debug("Connected to bootstrap peer",
				zap.String("peer", peerInfo.ID.String()))
			return nil
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(time.Second * time.Duration(retry+1)):
			continue
		}
	}

	return fmt.Errorf("failed to connect after %d retries: %w", maxRetries, err)
}

// reconnectLoop periodically attempts to reconnect to disconnected bootstrap peers
func (bd *BootstrapDiscovery) reconnectLoop() {
	ticker := time.NewTicker(reconnectInterval)
	defer ticker.Stop()

	for {
		select {
		case <-bd.ctx.Done():
			return
		case <-ticker.C:
			if err := bd.bootstrap(); err != nil {
				bd.logger.Debug("Failed to reconnect to bootstrap peers",
					zap.Error(err))
			}
		}
	}
}

// GetConnectedPeers returns currently connected bootstrap peers
func (bd *BootstrapDiscovery) GetConnectedPeers() []libp2ppeer.ID {
	bd.mu.RLock()
	defer bd.mu.RUnlock()

	peers := make([]libp2ppeer.ID, 0, len(bd.connected))
	for id := range bd.connected {
		peers = append(peers, id)
	}
	return peers
}

// IsConnected checks if a specific bootstrap peer is connected
func (bd *BootstrapDiscovery) IsConnected(id libp2ppeer.ID) bool {
	bd.mu.RLock()
	defer bd.mu.RUnlock()
	_, exists := bd.connected[id]
	return exists
}
