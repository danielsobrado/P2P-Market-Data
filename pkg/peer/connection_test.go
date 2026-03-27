package peer

import (
	"context"
	"testing"
	"time"

	ic "github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	ma "github.com/multiformats/go-multiaddr"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap/zaptest"
)

// stubConn is a minimal network.Conn stub that only implements RemotePeer.
// All other methods panic if called, ensuring tests do not rely on them.
type stubConn struct {
	remote peer.ID
}

func (s *stubConn) RemotePeer() peer.ID        { return s.remote }
func (s *stubConn) LocalPeer() peer.ID         { return peer.ID("local") }
func (s *stubConn) RemotePublicKey() ic.PubKey    { return nil }
func (s *stubConn) ConnState() network.ConnectionState {
	return network.ConnectionState{}
}
func (s *stubConn) LocalMultiaddr() ma.Multiaddr  { return nil }
func (s *stubConn) RemoteMultiaddr() ma.Multiaddr { return nil }
func (s *stubConn) Stat() network.ConnStats       { return network.ConnStats{} }
func (s *stubConn) Scope() network.ConnScope      { return nil }
func (s *stubConn) ID() string                    { return string(s.remote) }
func (s *stubConn) NewStream(context.Context) (network.Stream, error) {
	panic("not implemented")
}
func (s *stubConn) GetStreams() []network.Stream { return nil }
func (s *stubConn) IsClosed() bool               { return false }
func (s *stubConn) Close() error                 { return nil }

// newTestConnectionManager creates a ConnectionManager wired to a test
// PeerStore without requiring a real libp2p host. The host field is nil;
// tests must not call methods that dereference it.
func newTestConnectionManager(t *testing.T) *ConnectionManager {
	t.Helper()
	logger := zaptest.NewLogger(t)
	store := NewPeerStore(logger)
	// A nil host is acceptable for handler-only tests because the handlers
	// do not call into cm.host.
	return &ConnectionManager{
		host:        nil,
		store:       store,
		logger:      logger,
		activeConns: make(map[peer.ID]time.Time),
	}
}

// TestConnectionAccounting_SingleConnectDisconnect verifies the basic case:
// one connect followed by one disconnect produces a zero connection count.
func TestConnectionAccounting_SingleConnectDisconnect(t *testing.T) {
	cm := newTestConnectionManager(t)
	peerA := peer.ID("peerA")
	conn := &stubConn{remote: peerA}

	cm.handleConnected(nil, conn)
	assert.Equal(t, 1, cm.ConnectionCount())
	assert.True(t, cm.IsConnected(peerA))

	cm.handleDisconnected(nil, conn)
	assert.Equal(t, 0, cm.ConnectionCount())
	assert.False(t, cm.IsConnected(peerA))
}

// TestConnectionAccounting_RepeatedConnectDoesNotDrift verifies that firing
// multiple connected events for the same peer does not increment the count
// beyond 1 (idempotent connect).
func TestConnectionAccounting_RepeatedConnectDoesNotDrift(t *testing.T) {
	cm := newTestConnectionManager(t)
	peerA := peer.ID("peerA")
	conn := &stubConn{remote: peerA}

	cm.handleConnected(nil, conn)
	cm.handleConnected(nil, conn)
	cm.handleConnected(nil, conn)

	assert.Equal(t, 1, cm.ConnectionCount(), "repeated connected events must not drift the count above 1")
	assert.True(t, cm.IsConnected(peerA))
}

// TestConnectionAccounting_RepeatedDisconnectDoesNotDoubleDecrement verifies
// that multiple disconnect events for the same peer do not decrement the count
// below zero (idempotent disconnect).
func TestConnectionAccounting_RepeatedDisconnectDoesNotDoubleDecrement(t *testing.T) {
	cm := newTestConnectionManager(t)
	peerA := peer.ID("peerA")
	conn := &stubConn{remote: peerA}

	cm.handleConnected(nil, conn)
	assert.Equal(t, 1, cm.ConnectionCount())

	cm.handleDisconnected(nil, conn)
	cm.handleDisconnected(nil, conn) // second disconnect must be a no-op
	cm.handleDisconnected(nil, conn) // third disconnect must also be a no-op

	assert.Equal(t, 0, cm.ConnectionCount(), "repeated disconnected events must not decrement the count below 0")
	assert.False(t, cm.IsConnected(peerA))
}

// TestConnectionAccounting_MultiplePeers verifies that connect/disconnect
// events for different peers are tracked independently.
func TestConnectionAccounting_MultiplePeers(t *testing.T) {
	cm := newTestConnectionManager(t)

	peers := []peer.ID{"peerA", "peerB", "peerC"}
	conns := make([]*stubConn, len(peers))
	for i, p := range peers {
		conns[i] = &stubConn{remote: p}
		cm.handleConnected(nil, conns[i])
	}

	assert.Equal(t, 3, cm.ConnectionCount())
	for _, p := range peers {
		assert.True(t, cm.IsConnected(p))
	}

	// Disconnect one peer and verify others remain.
	cm.handleDisconnected(nil, conns[1])
	assert.Equal(t, 2, cm.ConnectionCount())
	assert.True(t, cm.IsConnected(peers[0]))
	assert.False(t, cm.IsConnected(peers[1]))
	assert.True(t, cm.IsConnected(peers[2]))
}

// TestConnectionAccounting_ConnectDisconnectReconnect ensures that a peer can
// be connected, disconnected, and connected again without leaving stale state.
func TestConnectionAccounting_ConnectDisconnectReconnect(t *testing.T) {
	cm := newTestConnectionManager(t)
	peerA := peer.ID("peerA")
	conn := &stubConn{remote: peerA}

	cm.handleConnected(nil, conn)
	cm.handleDisconnected(nil, conn)
	cm.handleConnected(nil, conn) // reconnect

	assert.Equal(t, 1, cm.ConnectionCount(), "reconnected peer must count as exactly one connection")
	assert.True(t, cm.IsConnected(peerA))
}

// TestConnectionAccounting_GetConnectedPeers verifies that GetConnectedPeers
// reflects the same set of peers as IsConnected.
func TestConnectionAccounting_GetConnectedPeers(t *testing.T) {
	cm := newTestConnectionManager(t)
	peerA, peerB := peer.ID("peerA"), peer.ID("peerB")

	cm.handleConnected(nil, &stubConn{remote: peerA})
	cm.handleConnected(nil, &stubConn{remote: peerB})

	got := cm.GetConnectedPeers()
	assert.Len(t, got, 2)
	gotSet := map[peer.ID]bool{}
	for _, p := range got {
		gotSet[p] = true
	}
	assert.True(t, gotSet[peerA])
	assert.True(t, gotSet[peerB])

	cm.handleDisconnected(nil, &stubConn{remote: peerA})
	got = cm.GetConnectedPeers()
	assert.Len(t, got, 1)
	assert.Equal(t, peerB, got[0])
}
