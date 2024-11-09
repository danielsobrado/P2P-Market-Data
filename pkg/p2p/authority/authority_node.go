package authority

import (
	"context"
	"sync"

	"p2p_market_data/pkg/p2p/host"
	"p2p_market_data/pkg/security"

	libp2pPeer "github.com/libp2p/go-libp2p-core/peer"
	"go.uber.org/zap"
)

// AuthorityNode represents a node with authority privileges.
type AuthorityNode struct {
	host          *Host
	validator     *security.Validator
	networkMgr    *NetworkManager
	verifiedPeers map[libp2pPeer.ID]*VerifiedPeer
	validations   chan *ValidationRequest
	logger        *zap.Logger
	metrics       *AuthorityMetrics
	mu            sync.RWMutex
}

// Start begins authority node operations.
func (an *AuthorityNode) Start(ctx context.Context) error {
	// Register authority protocol handler.
	an.host.host.SetStreamHandler(AuthorityProtocolID, an.handleAuthorityStream)

	// Start background processes.
	go an.processValidations(ctx)
	go an.verifyPeers(ctx)
	go an.collectMetrics(ctx)

	an.logger.Info("Authority node started")
	return nil
}

// Stop gracefully shuts down the authority node.
func (an *AuthorityNode) Stop() error {
	an.host.host.RemoveStreamHandler(AuthorityProtocolID)
	close(an.validations)
	an.logger.Info("Authority node stopped")
	return nil
}
