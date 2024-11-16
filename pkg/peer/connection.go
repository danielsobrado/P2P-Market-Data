// pkg/peer/connection.go
package peer

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"go.uber.org/zap"
)

const (
	// Connection limits
	maxConnections = 50
	minConnections = 5

	// Timeouts
	connectTimeout = 30 * time.Second
	pruneInterval  = 5 * time.Minute
)

// ConnectionManager handles peer connections
type ConnectionManager struct {
	host   host.Host
	store  *PeerStore
	logger *zap.Logger

	// Connection tracking
	connCount   int
	activeConns map[peer.ID]time.Time

	// Control
	ctx     context.Context
	cancel  context.CancelFunc
	mu      sync.RWMutex
	running bool
}

// NewConnectionManager creates a new connection manager
func NewConnectionManager(h host.Host, store *PeerStore, logger *zap.Logger) *ConnectionManager {
	ctx, cancel := context.WithCancel(context.Background())

	return &ConnectionManager{
		host:        h,
		store:       store,
		logger:      logger,
		activeConns: make(map[peer.ID]time.Time),
		ctx:         ctx,
		cancel:      cancel,
	}
}

// Start begins connection management
func (cm *ConnectionManager) Start() error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	if cm.running {
		return fmt.Errorf("connection manager already running")
	}

	// Register connection notifier
	cm.host.Network().Notify(&network.NotifyBundle{
		ConnectedF:    cm.handleConnected,
		DisconnectedF: cm.handleDisconnected,
	})

	// Start maintenance loop
	go cm.maintenanceLoop()

	cm.running = true
	cm.logger.Info("Connection manager started")
	return nil
}

// Stop halts connection management
func (cm *ConnectionManager) Stop() error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	if !cm.running {
		return nil
	}

	cm.cancel()
	cm.running = false
	cm.host.Network().StopNotify(&network.NotifyBundle{})
	cm.logger.Info("Connection manager stopped")
	return nil
}

// ConnectToPeer attempts to connect to a peer
func (cm *ConnectionManager) ConnectToPeer(p peer.AddrInfo) error {
	// Check connection limits
	if !cm.canAddConnection() {
		return fmt.Errorf("max connections reached")
	}

	// Create timeout context
	ctx, cancel := context.WithTimeout(cm.ctx, connectTimeout)
	defer cancel()

	// Attempt connection
	if err := cm.host.Connect(ctx, p); err != nil {
		return fmt.Errorf("connecting to peer %s: %w", p.ID, err)
	}

	return nil
}

// DisconnectPeer disconnects from a peer
func (cm *ConnectionManager) DisconnectPeer(id peer.ID) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	if _, exists := cm.activeConns[id]; !exists {
		return fmt.Errorf("peer not connected: %s", id)
	}

	if err := cm.host.Network().ClosePeer(id); err != nil {
		return fmt.Errorf("closing connection to peer %s: %w", id, err)
	}

	delete(cm.activeConns, id)
	cm.connCount--

	return nil
}

// Connection event handlers
func (cm *ConnectionManager) handleConnected(_ network.Network, conn network.Conn) {
	peerID := conn.RemotePeer()

	cm.mu.Lock()
	defer cm.mu.Unlock()

	cm.activeConns[peerID] = time.Now()
	cm.connCount++

	cm.logger.Debug("Peer connected",
		zap.String("peer", peerID.String()),
		zap.Int("total_connections", cm.connCount))

	// Update peer store
	cm.store.UpdatePeerStatus(peerID, true)
}

func (cm *ConnectionManager) handleDisconnected(_ network.Network, conn network.Conn) {
	peerID := conn.RemotePeer()

	cm.mu.Lock()
	defer cm.mu.Unlock()

	delete(cm.activeConns, peerID)
	cm.connCount--

	cm.logger.Debug("Peer disconnected",
		zap.String("peer", peerID.String()),
		zap.Int("total_connections", cm.connCount))

	// Update peer store
	cm.store.UpdatePeerStatus(peerID, false)
}

// Maintenance functions
func (cm *ConnectionManager) maintenanceLoop() {
	ticker := time.NewTicker(pruneInterval)
	defer ticker.Stop()

	for {
		select {
		case <-cm.ctx.Done():
			return
		case <-ticker.C:
			cm.performMaintenance()
		}
	}
}

func (cm *ConnectionManager) performMaintenance() {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	// If we're below minimum connections, don't prune
	if cm.connCount <= minConnections {
		return
	}

	// Find and remove stale connections
	now := time.Now()
	for id, connTime := range cm.activeConns {
		// If connection is older than 1 hour and we're above minimum
		if now.Sub(connTime) > time.Hour && cm.connCount > minConnections {
			if err := cm.host.Network().ClosePeer(id); err != nil {
				cm.logger.Debug("Error closing stale connection",
					zap.String("peer", id.String()),
					zap.Error(err))
				continue
			}

			delete(cm.activeConns, id)
			cm.connCount--
		}
	}
}

// Helper functions
func (cm *ConnectionManager) canAddConnection() bool {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	return cm.connCount < maxConnections
}

// GetConnectedPeers returns currently connected peers
func (cm *ConnectionManager) GetConnectedPeers() []peer.ID {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	peers := make([]peer.ID, 0, len(cm.activeConns))
	for id := range cm.activeConns {
		peers = append(peers, id)
	}
	return peers
}

// ConnectionCount returns the current number of connections
func (cm *ConnectionManager) ConnectionCount() int {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	return cm.connCount
}

// IsConnected checks if a peer is connected
func (cm *ConnectionManager) IsConnected(id peer.ID) bool {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	_, exists := cm.activeConns[id]
	return exists
}
