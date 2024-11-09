package authority

import (
	"context"
	"fmt"
	"time"

	libp2pPeer "github.com/libp2p/go-libp2p-core/peer"
	"go.uber.org/zap"
)

// VerifiedPeer represents a peer that has been verified by the authority.
type VerifiedPeer struct {
	ID           libp2pPeer.ID
	PublicKey    []byte
	ValidUntil   time.Time
	Permissions  []string
	LastVerified time.Time
}

// VerifyPeer verifies a peer's identity and permissions.
func (an *AuthorityNode) VerifyPeer(ctx context.Context, peerID libp2pPeer.ID) (*VerifiedPeer, error) {
	an.mu.RLock()
	if peer, exists := an.verifiedPeers[peerID]; exists && time.Now().Before(peer.ValidUntil) {
		an.mu.RUnlock()
		return peer, nil
	}
	an.mu.RUnlock()

	pubKey, err := an.host.GetPeerstore().PubKey(peerID)
	if err != nil {
		return nil, fmt.Errorf("getting peer public key: %w", err)
	}
	if pubKey == nil {
		return nil, fmt.Errorf("peer public key not found")
	}

	rawPubKey, err := pubKey.Raw()
	if err != nil {
		return nil, fmt.Errorf("getting raw public key: %w", err)
	}

	// Verification logic (e.g., check a certificate or perform challenge-response).
	verified := &VerifiedPeer{
		ID:           peerID,
		PublicKey:    rawPubKey,
		ValidUntil:   time.Now().Add(24 * time.Hour),
		Permissions:  []string{"basic"},
		LastVerified: time.Now(),
	}

	an.mu.Lock()
	an.verifiedPeers[peerID] = verified
	an.mu.Unlock()

	return verified, nil
}

func (an *AuthorityNode) verifyPeers(ctx context.Context) {
	ticker := time.NewTicker(time.Hour)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			an.cleanupExpiredPeers()
		}
	}
}

func (an *AuthorityNode) cleanupExpiredPeers() {
	an.mu.Lock()
	defer an.mu.Unlock()

	now := time.Now()
	for id, peer := range an.verifiedPeers {
		if now.After(peer.ValidUntil) {
			delete(an.verifiedPeers, id)
		}
	}
}

func (an *AuthorityNode) RevokeVerification(peerID libp2pPeer.ID) error {
	an.mu.Lock()
	defer an.mu.Unlock()

	if _, exists := an.verifiedPeers[peerID]; !exists {
		return fmt.Errorf("peer not found: %s", peerID)
	}

	delete(an.verifiedPeers, peerID)
	an.logger.Info("Revoked peer verification", zap.String("peerID", peerID.String()))
	return nil
}

func (an *AuthorityNode) UpdatePeerPermissions(peerID libp2pPeer.ID, permissions []string) error {
	an.mu.Lock()
	defer an.mu.Unlock()

	peer, exists := an.verifiedPeers[peerID]
	if !exists {
		return fmt.Errorf("peer not found: %s", peerID)
	}

	peer.Permissions = permissions
	peer.LastVerified = time.Now()
	an.logger.Info("Updated peer permissions", zap.String("peerID", peerID.String()), zap.Strings("permissions", permissions))
	return nil
}

func (an *AuthorityNode) IsAuthorizedForAction(peerID libp2pPeer.ID, action string) bool {
	an.mu.RLock()
	defer an.mu.RUnlock()

	peer, exists := an.verifiedPeers[peerID]
	if !exists || time.Now().After(peer.ValidUntil) {
		return false
	}

	for _, perm := range peer.Permissions {
		if perm == action || perm == "admin" {
			return true
		}
	}

	return false
}

func (an *AuthorityNode) GetVerifiedPeers() []*VerifiedPeer {
	an.mu.RLock()
	defer an.mu.RUnlock()

	peers := make([]*VerifiedPeer, 0, len(an.verifiedPeers))
	for _, peer := range an.verifiedPeers {
		peers = append(peers, peer)
	}

	return peers
}
