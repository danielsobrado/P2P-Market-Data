package host

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"p2p_market_data/pkg/data"
	"sync"
	"time"

	libp2pPeer "github.com/libp2p/go-libp2p/core/peer"
	"go.uber.org/zap"
)

type splitWriter interface {
	SaveSplitData(context.Context, *data.SplitData) error
}

type insiderWriter interface {
	SaveInsiderData(context.Context, *data.InsiderTrade) error
}

// DataRequest represents a data request to a peer
type DataRequest struct {
	Type    string
	Payload []byte
}

// NetworkManager handles P2P network operations
const (
	metricsCollectionInterval = 1 * time.Minute
)

type NetworkManager struct {
	host        *Host
	connManager *ConnectionManager
	peerManager *PeerManager
	metrics     *Metrics
	logger      *zap.Logger
	ctx         context.Context
	cancel      context.CancelFunc
	wg          sync.WaitGroup
}

// NewNetworkManager creates a new NetworkManager
func NewNetworkManager(host *Host, logger *zap.Logger) (*NetworkManager, error) {
	if host == nil {
		return nil, fmt.Errorf("host cannot be nil")
	}

	ctx, cancel := context.WithCancel(context.Background())

	nm := &NetworkManager{
		host:        host,
		peerManager: NewPeerManager(host.host, logger),
		metrics:     NewMetrics(),
		logger:      logger,
		ctx:         ctx,
		cancel:      cancel,
	}
	nm.connManager = NewConnectionManager(host.host, logger, nm)

	// Start background processes
	nm.wg.Add(2)
	go nm.runConnectionManager()
	go nm.runMetricsCollector()

	return nm, nil
}

// Close gracefully shuts down the NetworkManager
func (nm *NetworkManager) Close() error {
	nm.cancel()
	nm.wg.Wait()
	return nil
}

// ConnectToPeer connects to a specific peer
func (nm *NetworkManager) ConnectToPeer(peerInfo libp2pPeer.AddrInfo) error {
	if err := nm.host.host.Connect(nm.ctx, peerInfo); err != nil {
		return fmt.Errorf("failed to connect to peer %s: %w", peerInfo.ID, err)
	}
	nm.logger.Info("Connected to peer", zap.String("peerID", peerInfo.ID.String()))
	return nil
}

// DiscoverPeers discovers peers using mDNS and other discovery mechanisms
func (nm *NetworkManager) DiscoverPeers() error {
	peers, err := nm.peerManager.Discover()
	if err != nil {
		return fmt.Errorf("peer discovery failed: %w", err)
	}

	for _, peerInfo := range peers {
		if err := nm.ConnectToPeer(peerInfo); err != nil {
			nm.logger.Warn("Failed to connect to discovered peer", zap.Error(err))
		}
	}

	return nil
}

// GetConnectedPeers returns a list of currently connected peers
func (nm *NetworkManager) GetConnectedPeers() []libp2pPeer.ID {
	return nm.connManager.GetConnectedPeers()
}

// runConnectionManager manages peer connections in the background
func (nm *NetworkManager) runConnectionManager() {
	defer nm.wg.Done()

	ticker := time.NewTicker(connectionCheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-nm.ctx.Done():
			return
		case <-ticker.C:
			nm.connManager.ManageConnections(nm.ctx)
		}
	}
}

// runMetricsCollector collects network metrics in the background
func (nm *NetworkManager) runMetricsCollector() {
	defer nm.wg.Done()

	ticker := time.NewTicker(metricsCollectionInterval)
	defer ticker.Stop()

	for {
		select {
		case <-nm.ctx.Done():
			return
		case <-ticker.C:
			nm.metrics.Collect(nm.host)
		}
	}
}

// RequestData requests data from a peer
func (nm *NetworkManager) RequestData(ctx context.Context, peerID string, request data.DataRequest) error {
	if peerID == "" {
		return fmt.Errorf("peerID is required")
	}
	if request.Type == "" {
		return fmt.Errorf("request type is required")
	}

	id, err := libp2pPeer.Decode(peerID)
	if err != nil {
		return fmt.Errorf("invalid peer ID: %w", err)
	}

	// Ensure the target peer is currently known and connected before dispatch.
	if nm.host.host.Network().Connectedness(id) == 0 {
		return fmt.Errorf("peer %s is not connected", peerID)
	}

	stream, err := nm.host.host.NewStream(ctx, id, ProtocolID)
	if err != nil {
		return fmt.Errorf("opening data request stream: %w", err)
	}
	defer stream.Close()

	writer := bufio.NewWriter(stream)
	if err := json.NewEncoder(writer).Encode(request); err != nil {
		return fmt.Errorf("encoding data request: %w", err)
	}
	if err := writer.Flush(); err != nil {
		return fmt.Errorf("flushing data request: %w", err)
	}

	var resp dataResponse
	if err := json.NewDecoder(bufio.NewReader(stream)).Decode(&resp); err != nil {
		return fmt.Errorf("decoding data response: %w", err)
	}
	if resp.Error != "" {
		return fmt.Errorf("remote data request failed: %s", resp.Error)
	}
	if err := nm.persistDataResponse(ctx, peerID, resp); err != nil {
		return err
	}

	nm.logger.Info("Data request queued",
		zap.String("peerID", peerID),
		zap.String("type", request.Type),
		zap.String("symbol", request.Symbol))
	return nil
}

func (nm *NetworkManager) persistDataResponse(ctx context.Context, peerID string, resp dataResponse) error {
	switch resp.Type {
	case data.DataTypeEOD:
		for _, row := range resp.EOD {
			if err := validateEODResponse(row); err != nil {
				return err
			}
			price := row.Close
			if price <= 0 {
				price = 1
			}
			volume := row.Volume
			if volume < 0 {
				volume = 0
			}
			item, err := data.NewMarketData(row.Symbol, price, volume, peerID, data.DataTypeEOD)
			if err != nil {
				return err
			}
			item.Timestamp = row.Date
			if item.Timestamp.IsZero() {
				item.Timestamp = row.Timestamp
			}
			item.MetaData["open"] = fmt.Sprintf("%f", row.Open)
			item.MetaData["high"] = fmt.Sprintf("%f", row.High)
			item.MetaData["low"] = fmt.Sprintf("%f", row.Low)
			item.MetaData["close"] = fmt.Sprintf("%f", row.Close)
			item.MetaData["adjusted_close"] = fmt.Sprintf("%f", row.AdjustedClose)
			item.MetaData["volume"] = fmt.Sprintf("%f", row.Volume)
			item.UpdateHash()
			if err := nm.host.repo.SaveMarketData(ctx, item); err != nil && err != data.ErrDuplicate {
				return err
			}
		}
	case data.DataTypeDividend:
		for _, row := range resp.Dividends {
			if err := validateDividendResponse(row); err != nil {
				return err
			}
			if row.Source == "" {
				row.Source = peerID
			}
			if err := nm.host.repo.SaveDividendData(ctx, row); err != nil && err != data.ErrDuplicate {
				return err
			}
		}
	case data.DataTypeInsiderTrade:
		repo, ok := nm.host.repo.(insiderWriter)
		if !ok {
			return fmt.Errorf("insider repository not available")
		}
		for _, row := range resp.Insiders {
			if err := validateInsiderResponse(row); err != nil {
				return err
			}
			if row.Source == "" {
				row.Source = peerID
			}
			if err := repo.SaveInsiderData(ctx, &row); err != nil && err != data.ErrDuplicate {
				return err
			}
		}
	case data.DataTypeSplit:
		repo, ok := nm.host.repo.(splitWriter)
		if !ok {
			return fmt.Errorf("split repository not available")
		}
		for _, row := range resp.Splits {
			if err := validateSplitResponse(row); err != nil {
				return err
			}
			if row.Source == "" {
				row.Source = peerID
			}
			if err := repo.SaveSplitData(ctx, &row); err != nil && err != data.ErrDuplicate {
				return err
			}
		}
	default:
		return fmt.Errorf("unsupported response type: %s", resp.Type)
	}
	return nil
}

func validateEODResponse(row data.EODData) error {
	if row.Symbol == "" {
		return fmt.Errorf("invalid EOD response row: symbol is required")
	}
	if row.Date.IsZero() && row.Timestamp.IsZero() {
		return fmt.Errorf("invalid EOD response row for %s: date is required", row.Symbol)
	}
	if row.Close <= 0 {
		return fmt.Errorf("invalid EOD response row for %s: close must be positive", row.Symbol)
	}
	if row.Volume < 0 {
		return fmt.Errorf("invalid EOD response row for %s: volume cannot be negative", row.Symbol)
	}
	return nil
}

func validateDividendResponse(row *data.DividendData) error {
	if row == nil {
		return fmt.Errorf("invalid dividend response row: row is nil")
	}
	if row.Symbol == "" {
		return fmt.Errorf("invalid dividend response row: symbol is required")
	}
	if row.ExDate.IsZero() {
		return fmt.Errorf("invalid dividend response row for %s: ex-date is required", row.Symbol)
	}
	if row.Amount <= 0 {
		return fmt.Errorf("invalid dividend response row for %s: amount must be positive", row.Symbol)
	}
	return nil
}

func validateInsiderResponse(row data.InsiderTrade) error {
	if row.Symbol == "" {
		return fmt.Errorf("invalid insider response row: symbol is required")
	}
	if row.TradeDate.IsZero() {
		return fmt.Errorf("invalid insider response row for %s: trade date is required", row.Symbol)
	}
	if row.Shares <= 0 {
		return fmt.Errorf("invalid insider response row for %s: shares must be positive", row.Symbol)
	}
	return nil
}

func validateSplitResponse(row data.SplitData) error {
	if row.Symbol == "" {
		return fmt.Errorf("invalid split response row: symbol is required")
	}
	if row.ExDate.IsZero() {
		return fmt.Errorf("invalid split response row for %s: ex-date is required", row.Symbol)
	}
	if row.SplitRatio <= 0 {
		return fmt.Errorf("invalid split response row for %s: split ratio must be positive", row.Symbol)
	}
	if row.OldShares <= 0 || row.NewShares <= 0 {
		return fmt.Errorf("invalid split response row for %s: share counts must be positive", row.Symbol)
	}
	return nil
}

// ResetConnection resets the network connections
func (nm *NetworkManager) ResetConnection() error {
	for _, peerID := range nm.connManager.GetConnectedPeers() {
		if err := nm.host.DisconnectPeer(peerID); err != nil {
			nm.logger.Warn("Failed to disconnect peer during reset",
				zap.String("peerID", peerID.String()),
				zap.Error(err))
		}
	}

	if err := nm.DiscoverPeers(); err != nil {
		return fmt.Errorf("discovering peers after reset: %w", err)
	}
	return nil
}

// ResetProcessing resets data processing state
func (nm *NetworkManager) ResetProcessing() error {
	nm.logger.Info("Reset processing requested")
	return nil
}

// RetryConnection attempts to reconnect to peers
func (nm *NetworkManager) RetryConnection() error {
	return nm.DiscoverPeers()
}
