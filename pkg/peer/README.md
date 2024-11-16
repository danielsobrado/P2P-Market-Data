# Peer Discovery and Management Components

## Overview
This package provides a simple, modular peer discovery and management system for p2p applications. It includes several discovery mechanisms and basic peer management functionality.

## Components

### Core Components
```
pkg/peer/
├── manager.go        # Main peer manager
├── store.go         # Peer information storage
├── connection.go    # Connection management
└── discovery/       # Discovery mechanisms
    ├── interfaces.go  # Common interfaces
    ├── dht.go        # DHT-based discovery
    ├── mdns.go       # Local network discovery
    ├── bootstrap.go  # Bootstrap node discovery
    └── validator.go  # DHT record validation
```

## Quick Start

### Basic Usage
```go
// Initialize peer manager with discovery
func main() {
    // Create host and logger
    host, _ := libp2p.New()
    logger, _ := zap.NewProduction()

    // Create peer manager
    peerMgr, err := peer.NewManager(host, &peer.Config{
        MaxPeers: 50,
        MinPeers: 5,
        BootstrapPeers: []string{
            "/ip4/104.131.131.82/tcp/4001/p2p/QmaCpDMGvV2BGHeYERUEnRQAwe3N8SzbUtfsmvsqQLuvuJ",
        },
    }, logger)
    if err != nil {
        log.Fatal(err)
    }

    // Start peer manager
    if err := peerMgr.Start(context.Background()); err != nil {
        log.Fatal(err)
    }
    defer peerMgr.Stop()

    // Use peer manager...
}
```

## Component Details

### 1. Peer Manager (manager.go)
Primary coordinator for peer discovery and connections.

```go
// Example: Initialize and use peer manager
peerMgr, err := peer.NewManager(host, config, logger)
if err != nil {
    return err
}

// Get connected peers
peers := peerMgr.GetPeers()

// Connect to a specific peer
err = peerMgr.ConnectToPeer(peerID)

// Disconnect from a peer
err = peerMgr.DisconnectPeer(peerID)
```

### 2. Peer Store (store.go)
Manages peer information and basic reputation.

```go
// Example: Using peer store
store := peer.NewPeerStore(logger)

// Add or update peer
store.AddPeer(peerID, addresses)

// Get peer information
peerInfo, exists := store.GetPeer(peerID)

// Update peer status
store.UpdatePeerStatus(peerID, connected)

// Get connected peers
connectedPeers := store.GetConnectedPeers()
```

### 3. Connection Manager (connection.go)
Handles peer connections and limits.

```go
// Example: Using connection manager
connMgr := peer.NewConnectionManager(host, store, logger)

// Start connection management
err := connMgr.Start()

// Connect to peer
err = connMgr.ConnectToPeer(peerInfo)

// Get connection count
count := connMgr.ConnectionCount()
```

### 4. Discovery Components

#### DHT Discovery (discovery/dht.go)
Distributed Hash Table based peer discovery.

```go
// Example: Using DHT discovery
dht, err := discovery.NewDHTDiscovery(host, logger)
if err != nil {
    return err
}

// Start DHT discovery
err = dht.Start()

// Find peers
peers, err := dht.FindPeers("namespace")
```

#### MDNS Discovery (discovery/mdns.go)
Local network peer discovery.

```go
// Example: Using MDNS discovery
mdns, err := discovery.NewMDNSDiscovery(host, logger)
if err != nil {
    return err
}

// Start MDNS discovery
err = mdns.Start()

// Get local peers
localPeers := mdns.GetLocalPeers()
```

#### Bootstrap Discovery (discovery/bootstrap.go)
Bootstrap node based peer discovery.

```go
// Example: Using bootstrap discovery
bootstrap, err := discovery.NewBootstrapDiscovery(host, bootstrapPeers, logger)
if err != nil {
    return err
}

// Start bootstrap discovery
err = bootstrap.Start()

// Get connected bootstrap peers
peers := bootstrap.GetConnectedPeers()
```

## Integration Examples

### 1. Complete Peer Discovery Setup
```go
func setupPeerDiscovery(host libp2p.Host, logger *zap.Logger) error {
    // Create peer store
    store := peer.NewPeerStore(logger)

    // Create connection manager
    connMgr := peer.NewConnectionManager(host, store, logger)
    
    // Create discovery factory
    factory := &discovery.BasicDiscoveryFactory{
        host:   host,
        logger: logger,
    }

    // Initialize discoveries
    discoveries := make(map[discovery.DiscoveryType]discovery.Discovery)

    // Add DHT discovery
    if dht, err := factory.CreateDiscovery(discovery.TypeDHT, nil); err == nil {
        discoveries[discovery.TypeDHT] = dht
    }

    // Add MDNS discovery
    if mdns, err := factory.CreateDiscovery(discovery.TypeMDNS, nil); err == nil {
        discoveries[discovery.TypeMDNS] = mdns
    }

    // Create peer manager with all components
    peerMgr, err := peer.NewManager(
        host,
        &peer.Config{
            Store:       store,
            ConnMgr:    connMgr,
            Discoveries: discoveries,
        },
        logger,
    )
    if err != nil {
        return err
    }

    return peerMgr.Start(context.Background())
}
```

### 2. Custom Discovery Integration
```go
// Implement custom discovery
type CustomDiscovery struct {
    discovery.Discovery
    // Custom fields
}

func (cd *CustomDiscovery) Start() error {
    // Custom discovery logic
    return nil
}

// Add to peer manager
peerMgr.AddDiscovery("custom", customDiscovery)
```

### 3. Event Handling
```go
// Handle peer events
peerMgr.OnPeerConnected(func(p peer.ID) {
    log.Printf("Peer connected: %s", p)
})

peerMgr.OnPeerDisconnected(func(p peer.ID) {
    log.Printf("Peer disconnected: %s", p)
})
```

## Best Practices

1. **Configuration**
   - Set reasonable connection limits
   - Use appropriate discovery mechanisms
   - Configure timeouts properly

```go
config := &peer.Config{
    MaxPeers: 50,
    MinPeers: 5,
    ConnectionTimeout: 30 * time.Second,
    RetryInterval: 1 * time.Minute,
}
```

2. **Error Handling**
   - Handle discovery errors gracefully
   - Implement retry mechanisms
   - Log important events

```go
if err := peerMgr.ConnectToPeer(peerID); err != nil {
    switch {
    case errors.Is(err, peer.ErrPeerLimitReached):
        logger.Warn("Peer limit reached")
    case errors.Is(err, peer.ErrPeerNotFound):
        logger.Debug("Peer not found")
    default:
        logger.Error("Connection error", zap.Error(err))
    }
}
```

3. **Resource Management**
   - Always close/cleanup resources
   - Use context for cancellation
   - Monitor connection counts

```go
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()

if err := peerMgr.Start(ctx); err != nil {
    logger.Fatal("Failed to start peer manager", zap.Error(err))
}
defer peerMgr.Stop()
```

## Troubleshooting

Common issues and solutions:

1. **No peers found**
   - Check bootstrap peers are accessible
   - Verify network connectivity
   - Enable multiple discovery mechanisms

2. **Connection issues**
   - Verify network/firewall settings
   - Check connection limits
   - Review peer addresses

3. **High resource usage**
   - Adjust connection limits
   - Increase cleanup intervals
   - Monitor peer counts

## Testing

Run the test suite:
```bash
go test ./pkg/peer/... -v
```

Run specific tests:
```bash
go test ./pkg/peer/discovery -run TestDHT
```