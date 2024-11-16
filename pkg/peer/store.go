// pkg/peer/store.go
package peer

import (
	"fmt"
	"sync"
	"time"

	"github.com/libp2p/go-libp2p/core/peer"
	"go.uber.org/zap"
)

// PeerStore manages peer information
type PeerStore struct {
	peers  map[peer.ID]*PeerData
	logger *zap.Logger
	mu     sync.RWMutex
}

// PeerData holds essential peer information
type PeerData struct {
	ID        peer.ID
	Addresses []string
	LastSeen  time.Time
	Connected bool
	// Simple reputation score (-100 to 100)
	Score int
}

// NewPeerStore creates a new peer store
func NewPeerStore(logger *zap.Logger) *PeerStore {
	return &PeerStore{
		peers:  make(map[peer.ID]*PeerData),
		logger: logger,
	}
}

// AddPeer adds or updates a peer in the store
func (ps *PeerStore) AddPeer(id peer.ID, addresses []string) {
	ps.mu.Lock()
	defer ps.mu.Unlock()

	if p, exists := ps.peers[id]; exists {
		// Update existing peer
		p.LastSeen = time.Now()
		p.Addresses = addresses
	} else {
		// Add new peer
		ps.peers[id] = &PeerData{
			ID:        id,
			Addresses: addresses,
			LastSeen:  time.Now(),
			Score:     0,
		}
	}
}

// GetPeer retrieves peer information
func (ps *PeerStore) GetPeer(id peer.ID) (*PeerData, bool) {
	ps.mu.RLock()
	defer ps.mu.RUnlock()

	peer, exists := ps.peers[id]
	return peer, exists
}

// RemovePeer removes a peer from the store
func (ps *PeerStore) RemovePeer(id peer.ID) {
	ps.mu.Lock()
	defer ps.mu.Unlock()

	delete(ps.peers, id)
}

// UpdatePeerStatus updates peer connection status
func (ps *PeerStore) UpdatePeerStatus(id peer.ID, connected bool) {
	ps.mu.Lock()
	defer ps.mu.Unlock()

	if peer, exists := ps.peers[id]; exists {
		peer.Connected = connected
		peer.LastSeen = time.Now()
	}
}

// AdjustScore adjusts a peer's reputation score
func (ps *PeerStore) AdjustScore(id peer.ID, adjustment int) error {
	ps.mu.Lock()
	defer ps.mu.Unlock()

	peer, exists := ps.peers[id]
	if !exists {
		return fmt.Errorf("peer not found: %s", id)
	}

	// Adjust score within bounds (-100 to 100)
	newScore := peer.Score + adjustment
	if newScore > 100 {
		peer.Score = 100
	} else if newScore < -100 {
		peer.Score = -100
	} else {
		peer.Score = newScore
	}

	return nil
}

// GetConnectedPeers returns all connected peers
func (ps *PeerStore) GetConnectedPeers() []*PeerData {
	ps.mu.RLock()
	defer ps.mu.RUnlock()

	var connected []*PeerData
	for _, peer := range ps.peers {
		if peer.Connected {
			connected = append(connected, peer)
		}
	}
	return connected
}

// CleanupStalePeers removes peers not seen recently
func (ps *PeerStore) CleanupStalePeers(maxAge time.Duration) {
	ps.mu.Lock()
	defer ps.mu.Unlock()

	now := time.Now()
	for id, peer := range ps.peers {
		if !peer.Connected && now.Sub(peer.LastSeen) > maxAge {
			delete(ps.peers, id)
			ps.logger.Debug("Removed stale peer",
				zap.String("peer", id.String()),
				zap.Time("lastSeen", peer.LastSeen))
		}
	}
}

// GetPeersByScore returns peers above a minimum score
func (ps *PeerStore) GetPeersByScore(minScore int) []*PeerData {
	ps.mu.RLock()
	defer ps.mu.RUnlock()

	var qualified []*PeerData
	for _, peer := range ps.peers {
		if peer.Score >= minScore {
			qualified = append(qualified, peer)
		}
	}
	return qualified
}

// IsTrustedPeer checks if a peer has a good reputation
func (ps *PeerStore) IsTrustedPeer(id peer.ID) bool {
	ps.mu.RLock()
	defer ps.mu.RUnlock()

	if peer, exists := ps.peers[id]; exists {
		return peer.Score >= 50 // Consider peers with score >= 50 trusted
	}
	return false
}
