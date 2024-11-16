// pkg/peer/discovery/interfaces.go
package discovery

import (
	"context"
	"time"

	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	"go.uber.org/zap"
)

// Discovery defines the basic interface for peer discovery mechanisms
type Discovery interface {
	// Core functionality
	Start() error
	Stop() error

	// Discovery operations
	GetConnectedPeers() []peer.ID
	IsConnected(peer.ID) bool
}

// PeerInfo holds basic information about a discovered peer
type PeerInfo struct {
	// Essential peer information
	ID        peer.ID
	Addrs     []string
	LastSeen  time.Time
	Connected bool
}

// DiscoveryType represents different discovery mechanisms
type DiscoveryType string

const (
	TypeDHT       DiscoveryType = "dht"
	TypeMDNS      DiscoveryType = "mdns"
	TypeBootstrap DiscoveryType = "bootstrap"
)

// DiscoveryFactory creates discovery instances
type DiscoveryFactory interface {
	CreateDiscovery(dtype DiscoveryType, config interface{}) (Discovery, error)
}

// PeerHandler defines how discovered peers should be handled
type PeerHandler interface {
	// HandlePeer is called when a new peer is discovered
	HandlePeer(ctx context.Context, info PeerInfo) error
}

// ConnectionManager defines basic connection management
type ConnectionManager interface {
	// Core connection operations
	Connect(ctx context.Context, peer peer.ID) error
	Disconnect(peer peer.ID) error

	// Status checks
	IsConnected(peer.ID) bool
	ConnectionCount() int
}

// DiscoveryConfig holds basic configuration for discovery mechanisms
type DiscoveryConfig struct {
	// Basic configuration
	Enabled         bool
	TargetPeers     int
	ConnectionLimit int
	RetryInterval   time.Duration

	// Bootstrap specific
	BootstrapPeers []string

	// DHT specific
	DHTNamespace string

	// MDNS specific
	ServiceTag string
}

// DiscoveryStatus represents the current state of discovery
type DiscoveryStatus struct {
	// Basic status information
	Running        bool
	ConnectedPeers int
	LastDiscovery  time.Time
	DiscoveryType  DiscoveryType
}

// Error types for discovery operations
type DiscoveryError struct {
	Type    DiscoveryType
	Message string
	Err     error
}

func (e *DiscoveryError) Error() string {
	return e.Message
}

// Helper functions

// CreateDiscoveryConfig creates a default configuration
func CreateDiscoveryConfig() *DiscoveryConfig {
	return &DiscoveryConfig{
		Enabled:         true,
		TargetPeers:     10,
		ConnectionLimit: 50,
		RetryInterval:   time.Minute,
	}
}

// ValidateDiscoveryConfig checks if a configuration is valid
func ValidateDiscoveryConfig(config *DiscoveryConfig) error {
	if config.ConnectionLimit < config.TargetPeers {
		return &DiscoveryError{
			Message: "connection limit must be greater than target peers",
		}
	}
	return nil
}

// Example of a simple factory implementation
type BasicDiscoveryFactory struct {
	host      host.Host
	logger    Logger
	zapLogger *zap.Logger
	config    *DiscoveryConfig
}

func (f *BasicDiscoveryFactory) CreateDiscovery(dtype DiscoveryType, config interface{}) (Discovery, error) {
	switch dtype {
	case TypeDHT:
		return NewDHTDiscovery(f.host, f.zapLogger)
	case TypeMDNS:
		return NewMDNSDiscovery(f.host, f.zapLogger)
	case TypeBootstrap:
		cfg, ok := config.([]string)
		if !ok {
			return nil, &DiscoveryError{
				Type:    TypeBootstrap,
				Message: "invalid bootstrap configuration",
			}
		}
		return NewBootstrapDiscovery(f.host, cfg, f.zapLogger)
	default:
		return nil, &DiscoveryError{
			Type:    dtype,
			Message: "unsupported discovery type",
		}
	}
}

func NewBasicDiscoveryFactory(host host.Host, logger *zap.Logger, config *DiscoveryConfig) *BasicDiscoveryFactory {
	return &BasicDiscoveryFactory{
		host:      host,
		logger:    NewZapLoggerAdapter(logger),
		zapLogger: logger,
		config:    config,
	}
}

// Example usage:
/*
func NewPeerManager(host Host, config *DiscoveryConfig, logger Logger) (*PeerManager, error) {
    // Create discovery factory
    factory := &BasicDiscoveryFactory{
        host:   host,
        logger: logger,
        config: config,
    }

    // Create discoveries
    discoveries := make(map[DiscoveryType]Discovery)

    // Add DHT discovery
    if dht, err := factory.CreateDiscovery(TypeDHT, nil); err == nil {
        discoveries[TypeDHT] = dht
    }

    // Add MDNS discovery
    if mdns, err := factory.CreateDiscovery(TypeMDNS, nil); err == nil {
        discoveries[TypeMDNS] = mdns
    }

    // Add bootstrap discovery if peers configured
    if len(config.BootstrapPeers) > 0 {
        if bootstrap, err := factory.CreateDiscovery(TypeBootstrap, config.BootstrapPeers); err == nil {
            discoveries[TypeBootstrap] = bootstrap
        }
    }

    return &PeerManager{
        discoveries: discoveries,
        // ... other initialization
    }, nil
}
*/
