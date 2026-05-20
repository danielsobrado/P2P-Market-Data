package host

import (
	"context"
	"net"
	"path/filepath"
	"testing"
	"time"

	"p2p_market_data/pkg/config"
	"p2p_market_data/pkg/data"

	libp2pNetwork "github.com/libp2p/go-libp2p/core/network"
	libp2pPeer "github.com/libp2p/go-libp2p/core/peer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

func freeTCPPort(t *testing.T) int {
	t.Helper()

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer listener.Close()

	return listener.Addr().(*net.TCPAddr).Port
}

func newIntegrationHost(t *testing.T, ctx context.Context) *Host {
	t.Helper()

	cfg := &config.Config{
		P2P: config.P2PConfig{
			Port:             freeTCPPort(t),
			VotingTimeout:    time.Second,
			MinVoters:        1,
			ValidationQuorum: 0.5,
		},
		Security: config.SecurityConfig{
			KeyFile:       filepath.Join(t.TempDir(), "host.key"),
			MaxPenalty:    0.5,
			MinConfidence: 0.5,
		},
	}

	h, err := NewHost(ctx, cfg, zaptest.NewLogger(t), data.NewMockRepository())
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = h.Stop()
	})

	return h
}

func TestHostP2PShareDataBetweenTwoLocalHosts(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	sender := newIntegrationHost(t, ctx)
	receiver := newIntegrationHost(t, ctx)

	require.NoError(t, sender.Start(ctx))
	require.NoError(t, receiver.Start(ctx))

	receiverInfo := libp2pPeer.AddrInfo{
		ID:    receiver.host.ID(),
		Addrs: receiver.host.Addrs(),
	}
	require.NoError(t, sender.networkMgr.ConnectToPeer(receiverInfo))

	assert.Eventually(t, func() bool {
		return sender.host.Network().Connectedness(receiver.host.ID()) == libp2pNetwork.Connected
	}, 5*time.Second, 100*time.Millisecond)

	marketData, err := data.NewMarketData("BTCUSD", 50000, 12, "integration", data.DataTypeEOD)
	require.NoError(t, err)
	require.NoError(t, sender.ShareData(ctx, marketData))

	assert.Eventually(t, func() bool {
		return receiver.metrics.GetMetrics().MessagesProcessed > 0
	}, 10*time.Second, 100*time.Millisecond)
}
