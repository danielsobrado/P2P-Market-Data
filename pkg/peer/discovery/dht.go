// pkg/peer/discovery/dht.go
package discovery

import (
	"context"
	"fmt"
	"time"

	"github.com/ipfs/go-cid"
	"github.com/libp2p/go-libp2p-core/network"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	"go.uber.org/zap"
)

const (
	// Basic timeouts
	dhtLookupTimeout = 30 * time.Second
	retryInterval    = 10 * time.Second
)

// DHTDiscovery implements peer discovery using Kademlia DHT
type DHTDiscovery struct {
	host    host.Host
	dht     *dht.IpfsDHT
	logger  *zap.Logger
	ctx     context.Context
	cancel  context.CancelFunc
	running bool
}

// NewDHTDiscovery creates a new DHT-based discovery service
func NewDHTDiscovery(h host.Host, logger *zap.Logger) (*DHTDiscovery, error) {
	ctx, cancel := context.WithCancel(context.Background())

	// Initialize DHT in server mode
	kadDHT, err := dht.New(ctx, h, dht.Mode(dht.ModeServer))
	if err != nil {
		cancel()
		return nil, fmt.Errorf("creating DHT: %w", err)
	}

	return &DHTDiscovery{
		host:   h,
		dht:    kadDHT,
		logger: logger,
		ctx:    ctx,
		cancel: cancel,
	}, nil
}

// Start initializes DHT and bootstraps
func (d *DHTDiscovery) Start() error {
	if d.running {
		return fmt.Errorf("DHT discovery already running")
	}

	// Bootstrap the DHT
	if err := d.dht.Bootstrap(d.ctx); err != nil {
		return fmt.Errorf("bootstrapping DHT: %w", err)
	}

	d.running = true
	d.logger.Info("DHT discovery started")
	return nil
}

// Stop halts DHT discovery
func (d *DHTDiscovery) Stop() error {
	if !d.running {
		return nil
	}

	d.cancel()
	if err := d.dht.Close(); err != nil {
		return fmt.Errorf("closing DHT: %w", err)
	}

	d.running = false
	d.logger.Info("DHT discovery stopped")
	return nil
}

// FindPeers looks for peers in the DHT
func (d *DHTDiscovery) FindPeers(namespace string) ([]peer.AddrInfo, error) {
	if !d.running {
		return nil, fmt.Errorf("DHT discovery not running")
	}

	ctx, cancel := context.WithTimeout(d.ctx, dhtLookupTimeout)
	defer cancel()

	// Convert namespace to CID for DHT lookup
	nsID, err := cid.Decode(namespace)
	if err != nil {
		return nil, fmt.Errorf("invalid namespace: %w", err)
	}

	// Find providers for the namespace
	peerChan := d.dht.FindProvidersAsync(ctx, nsID, 0)
	if err != nil {
		return nil, fmt.Errorf("finding providers: %w", err)
	}

	var peers []peer.AddrInfo
	for p := range peerChan {
		if p.ID == d.host.ID() {
			continue // Skip self
		}
		peers = append(peers, p)
	}

	return peers, nil
}

// Provide announces this node can provide data for namespace
func (d *DHTDiscovery) Provide(namespace string) error {
	if !d.running {
		return fmt.Errorf("DHT discovery not running")
	}

	ctx, cancel := context.WithTimeout(d.ctx, dhtLookupTimeout)
	defer cancel()

	// Decode the namespace into a CID
	cid, err := cid.Decode(namespace)
	if err != nil {
		return fmt.Errorf("invalid namespace CID: %w", err)
	}

	// Provide the CID to the DHT
	if err := d.dht.Provide(ctx, cid, true); err != nil {
		return fmt.Errorf("providing namespace: %w", err)
	}

	return nil
}

// GetClosestPeers returns the closest peers to a given key
func (d *DHTDiscovery) GetClosestPeers(key string) ([]peer.ID, error) {
	if !d.running {
		return nil, fmt.Errorf("DHT discovery not running")
	}

	ctx, cancel := context.WithTimeout(d.ctx, dhtLookupTimeout)
	defer cancel()

	peers, err := d.dht.GetClosestPeers(ctx, key)
	if err != nil {
		return nil, fmt.Errorf("finding closest peers: %w", err)
	}

	return peers, nil
}

// GetConnectedPeers returns a list of currently connected peers via DHT
func (d *DHTDiscovery) GetConnectedPeers() []peer.ID {
	if !d.running {
		return nil
	}

	// Get peers from DHT routing table
	peers := d.dht.RoutingTable().ListPeers()

	// Filter out self and create result slice
	result := make([]peer.ID, 0, len(peers))
	for _, p := range peers {
		if p == d.host.ID() {
			continue // Skip self
		}
		if d.host.Network().Connectedness(p) == network.Connected {
			result = append(result, p)
		}
	}

	return result
}

// IsConnected checks if a specific peer is connected
func (d *DHTDiscovery) IsConnected(id peer.ID) bool {
	if !d.running {
		return false
	}
	return d.host.Network().Connectedness(id) == network.Connected
}
