package authority

import (
	"fmt"
	"p2p_market_data/pkg/config"
	"p2p_market_data/pkg/security"

	"p2p_market_data/pkg/p2p/host"
	"p2p_market_data/pkg/p2p/message"

	libp2pPeer "github.com/libp2p/go-libp2p/core/peer"
	"go.uber.org/zap"
)

// NewAuthorityNode creates and initializes a new authority node.
func NewAuthorityNode(cfg *config.P2PConfig, hostStruct *host.Host, logger *zap.Logger) (*AuthorityNode, error) {
	validator, err := security.NewValidator(*cfg.Security)
	if err != nil {
		return nil, fmt.Errorf("creating validator: %w", err)
	}

	networkMgr, err := host.NewNetworkManager(hostStruct, logger)
	if err != nil {
		return nil, fmt.Errorf("creating network manager: %w", err)
	}

	authorityNode := &AuthorityNode{
		host:          hostStruct,
		validator:     validator,
		networkMgr:    networkMgr,
		verifiedPeers: make(map[libp2pPeer.ID]*VerifiedPeer),
		validations:   make(chan *message.ValidationRequest, 100),
		logger:        logger,
		metrics:       NewAuthorityMetrics(),
	}

	return authorityNode, nil
}
