package host

import (
	"context"
	"sync"
	"time"

	"github.com/libp2p/go-libp2p/core/host"
	libp2pPeer "github.com/libp2p/go-libp2p/core/peer"
	"go.uber.org/zap"
)

const (
	connectionCheckInterval = 30 * time.Second
	minPeers                = 5
	maxPeers                = 50
)

// ConnectionManager handles peer connections
type ConnectionManager struct {
	host       host.Host
	logger     *zap.Logger
	mu         sync.RWMutex
	networkMgr *NetworkManager
}

// NewConnectionManager creates a new ConnectionManager
func NewConnectionManager(host host.Host, logger *zap.Logger, networkMgr *NetworkManager) *ConnectionManager {
	return &ConnectionManager{
		host:       host,
		logger:     logger,
		networkMgr: networkMgr,
	}
}

// ManageConnections ensures that the number of connections stays within desired limits
func (cm *ConnectionManager) ManageConnections(ctx context.Context) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	connectedPeers := cm.GetConnectedPeers()
	numPeers := len(connectedPeers)

	if numPeers < minPeers {
		cm.logger.Info("Peer count below minimum, discovering new peers")
		if err := cm.networkMgr.DiscoverPeers(); err != nil {
			cm.logger.Warn("Peer discovery failed", zap.Error(err))
		}
	} else if numPeers > maxPeers {
		cm.logger.Info("Peer count above maximum, pruning peers",
			zap.Int("connectedPeers", numPeers))
		cm.prunePeers(numPeers - maxPeers)
	}
}

// GetConnectedPeers returns a list of currently connected peers
func (cm *ConnectionManager) GetConnectedPeers() []libp2pPeer.ID {
	conns := cm.host.Network().Conns()
	peers := make([]libp2pPeer.ID, 0, len(conns))
	for _, conn := range conns {
		peers = append(peers, conn.RemotePeer())
	}
	return peers
}

// prunePeers disconnects from excess peers
func (cm *ConnectionManager) prunePeers(excess int) {
	conns := cm.host.Network().Conns()
	for i := 0; i < excess && i < len(conns); i++ {
		conn := conns[i]
		if err := conn.Close(); err != nil {
			cm.logger.Warn("Failed to close connection", zap.Error(err))
		} else {
			cm.logger.Info("Pruned peer connection", zap.String("peerID", conn.RemotePeer().String()))
		}
	}
}
