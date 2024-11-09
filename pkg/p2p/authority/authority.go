package authority

import (
	"fmt"
	"p2p_market_data/pkg/config"
	"p2p_market_data/pkg/security"

	libp2pPeer "github.com/libp2p/go-libp2p/core/peer"
	"go.uber.org/zap"
)

// NewAuthorityNode creates and initializes a new authority node.
func NewAuthorityNode(cfg *config.P2PConfig, host *Host, logger *zap.Logger) (*AuthorityNode, error) {
	validator := security.NewValidator() // Assuming a validator is created here

	networkMgr, err := NewNetworkManager(host, logger)
	if err != nil {
		return nil, fmt.Errorf("creating network manager: %w", err)
	}

	authorityNode := &AuthorityNode{
		host:          host,
		validator:     validator,
		networkMgr:    networkMgr,
		verifiedPeers: make(map[libp2pPeer.ID]*VerifiedPeer),
		validations:   make(chan *ValidationRequest, 100),
		logger:        logger,
		metrics:       NewAuthorityMetrics(),
	}

	return authorityNode, nil
}
