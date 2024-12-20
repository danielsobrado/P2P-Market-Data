package host

import (
	"fmt"
	"sync"

	"p2p_market_data/pkg/data"

	"github.com/libp2p/go-libp2p/core/crypto"
	libp2pPeer "github.com/libp2p/go-libp2p/core/peer"
)

// PeerStore manages peer information and reputation
type PeerStore struct {
	peers map[libp2pPeer.ID]*data.Peer
	repo  data.Repository
	mu    sync.RWMutex
}

// NewPeerStore creates a new PeerStore
func NewPeerStore(repo data.Repository) *PeerStore {
	return &PeerStore{
		peers: make(map[libp2pPeer.ID]*data.Peer),
		repo:  repo,
	}
}

// GetPeerstore returns the peer store of the host
func (h *Host) GetPeerstore() *PeerStore {
	return h.peerStore
}

// AddPeer adds or updates a peer in the store
func (ps *PeerStore) AddPeer(peer *data.Peer) {
	ps.mu.Lock()
	defer ps.mu.Unlock()
	ps.peers[libp2pPeer.ID(peer.ID)] = peer
}

// GetPeer retrieves a peer from the store
func (ps *PeerStore) GetPeer(peerID libp2pPeer.ID) (*data.Peer, bool) {
	ps.mu.RLock()
	defer ps.mu.RUnlock()
	peer, exists := ps.peers[peerID]
	return peer, exists
}

// RemovePeer removes a peer from the store
func (ps *PeerStore) RemovePeer(peerID libp2pPeer.ID) {
	ps.mu.Lock()
	defer ps.mu.Unlock()
	delete(ps.peers, peerID)
}

// ListPeers returns all peers in the store
func (ps *PeerStore) ListPeers() []*data.Peer {
	ps.mu.RLock()
	defer ps.mu.RUnlock()
	peers := make([]*data.Peer, 0, len(ps.peers))
	for _, peer := range ps.peers {
		peers = append(peers, peer)
	}
	return peers
}

func (ps *PeerStore) PubKey(peerID libp2pPeer.ID) (crypto.PubKey, error) {
	ps.mu.RLock()
	defer ps.mu.RUnlock()

	peer, exists := ps.peers[peerID]
	if !exists {
		return nil, fmt.Errorf("peer not found")
	}

	pubKey, err := crypto.UnmarshalPublicKey(peer.PublicKey)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal public key: %w", err)
	}
	return pubKey, nil
}
