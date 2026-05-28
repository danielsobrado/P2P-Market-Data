package host

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"p2p_market_data/pkg/data"
	"sync"
	"time"

	"github.com/google/uuid"
	libp2pPeer "github.com/libp2p/go-libp2p/core/peer"
	"go.uber.org/zap"
)

type splitWriter interface {
	SaveSplitData(context.Context, *data.SplitData) error
}

type insiderWriter interface {
	SaveInsiderData(context.Context, *data.InsiderTrade) error
}

type transferWriter interface {
	SaveTransfer(context.Context, *data.DataTransfer) error
}

type transferHistoryStore interface {
	SaveTransfer(context.Context, *data.DataTransfer) error
	ListTransfers(context.Context) ([]data.DataTransfer, error)
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
	if request.RequestID == "" {
		request.RequestID = uuid.New().String()
	}
	if request.TransferID == "" {
		request.TransferID = uuid.New().String()
	}
	request.ChunkSize = normalizeChunkSize(request.ChunkSize)

	id, err := libp2pPeer.Decode(peerID)
	if err != nil {
		nm.saveTransferFailure(ctx, peerID, request, err)
		return fmt.Errorf("invalid peer ID: %w", err)
	}

	// Ensure the target peer is currently known and connected before dispatch.
	if nm.host.host.Network().Connectedness(id) == 0 {
		err := fmt.Errorf("peer %s is not connected", peerID)
		nm.saveTransferFailure(ctx, peerID, request, err)
		return err
	}

	if err := nm.saveTransferProgress(ctx, peerID, request, nil, "transferring", "", 0); err != nil {
		return err
	}
	nm.host.metrics.RecordTransferStarted()

	for {
		resp, err := nm.requestDataChunk(ctx, id, request)
		if err != nil {
			nm.saveTransferFailure(ctx, peerID, request, err)
			return err
		}
		if resp.Error != "" {
			err := fmt.Errorf("remote data request failed: %s", resp.Error)
			nm.saveTransferFailure(ctx, peerID, request, err)
			return err
		}
		if err := validateChunkResponse(resp, request); err != nil {
			nm.saveTransferFailure(ctx, peerID, request, err)
			return err
		}
		if err := verifyResponseChecksum(resp); err != nil {
			nm.saveTransferFailure(ctx, peerID, request, err)
			return err
		}
		if err := nm.persistDataResponse(ctx, peerID, resp); err != nil {
			nm.saveTransferFailure(ctx, peerID, request, err)
			return err
		}
		payloadBytes := chunkPayloadSize(resp)
		nm.host.metrics.RecordChunkReceived(resp, payloadBytes)
		request.Offset = resp.NextOffset
		if err := nm.saveTransferProgress(ctx, peerID, request, &resp, "transferring", "", payloadBytes); err != nil {
			return err
		}
		if !resp.HasMore {
			if err := nm.saveTransferProgress(ctx, peerID, request, &resp, "completed", "", 0); err != nil {
				return err
			}
			nm.host.metrics.RecordTransferCompleted()
			break
		}
		if responseRowCount(resp) == 0 {
			err := fmt.Errorf("remote response returned no rows while indicating more chunks")
			nm.saveTransferFailure(ctx, peerID, request, err)
			return err
		}
	}

	nm.logger.Info("Data request completed",
		zap.String("peerID", peerID),
		zap.String("type", request.Type),
		zap.String("symbol", request.Symbol),
		zap.String("requestID", request.RequestID))
	return nil
}

func (nm *NetworkManager) requestDataChunk(ctx context.Context, id libp2pPeer.ID, request data.DataRequest) (dataResponse, error) {
	if err := nm.host.signDataRequest(&request); err != nil {
		return dataResponse{}, err
	}
	start := time.Now()
	stream, err := nm.host.host.NewStream(ctx, id, ProtocolID)
	if err != nil {
		return dataResponse{}, fmt.Errorf("opening data request stream: %w", err)
	}
	defer stream.Close()

	writer := bufio.NewWriter(stream)
	if err := json.NewEncoder(writer).Encode(request); err != nil {
		return dataResponse{}, fmt.Errorf("encoding data request: %w", err)
	}
	if err := writer.Flush(); err != nil {
		return dataResponse{}, fmt.Errorf("flushing data request: %w", err)
	}

	var resp dataResponse
	if err := json.NewDecoder(bufio.NewReader(stream)).Decode(&resp); err != nil {
		return dataResponse{}, fmt.Errorf("decoding data response: %w", err)
	}
	nm.host.metrics.UpdateNetworkLatency(time.Since(start))
	if resp.RequestID != "" && resp.RequestID != request.RequestID {
		return dataResponse{}, fmt.Errorf("response request id mismatch: got %s want %s", resp.RequestID, request.RequestID)
	}
	if resp.TransferID != "" && resp.TransferID != request.TransferID {
		return dataResponse{}, fmt.Errorf("response transfer id mismatch: got %s want %s", resp.TransferID, request.TransferID)
	}
	if err := nm.host.verifyDataResponse(id, request, &resp); err != nil {
		return dataResponse{}, err
	}
	return resp, nil
}

func validateChunkResponse(resp dataResponse, request data.DataRequest) error {
	if resp.Checksum == "" {
		return fmt.Errorf("response checksum is required")
	}
	if resp.ChunkSize != normalizeChunkSize(request.ChunkSize) {
		return fmt.Errorf("response chunk size mismatch: got %d want %d", resp.ChunkSize, normalizeChunkSize(request.ChunkSize))
	}
	if resp.Offset != request.Offset {
		return fmt.Errorf("response offset mismatch: got %d want %d", resp.Offset, request.Offset)
	}
	if resp.NextOffset < resp.Offset {
		return fmt.Errorf("response next offset %d is before offset %d", resp.NextOffset, resp.Offset)
	}
	rowCount := responseRowCount(resp)
	if resp.NextOffset-resp.Offset != rowCount {
		return fmt.Errorf("response row count mismatch: got %d row(s) for offset range %d", rowCount, resp.NextOffset-resp.Offset)
	}
	if resp.TotalRows < resp.NextOffset {
		return fmt.Errorf("response total rows %d is before next offset %d", resp.TotalRows, resp.NextOffset)
	}
	if resp.TotalRows > 0 && resp.TotalChunks <= 0 {
		return fmt.Errorf("response total chunks is required when total rows is non-zero")
	}
	if resp.HasMore && resp.NextOffset >= resp.TotalRows {
		return fmt.Errorf("response indicates more chunks after final offset")
	}
	if !resp.HasMore && resp.NextOffset != resp.TotalRows {
		return fmt.Errorf("response final offset %d does not match total rows %d", resp.NextOffset, resp.TotalRows)
	}
	return nil
}

func verifyResponseChecksum(resp dataResponse) error {
	if resp.Checksum == "" {
		return nil
	}
	checksum, err := responseChecksum(resp)
	if err != nil {
		return err
	}
	if checksum != resp.Checksum {
		return fmt.Errorf("response checksum mismatch for chunk %d", resp.ChunkIndex)
	}
	return nil
}

func chunkPayloadSize(resp dataResponse) int64 {
	content, err := json.Marshal(resp)
	if err != nil {
		return 0
	}
	return int64(len(content))
}

func (nm *NetworkManager) saveTransferProgress(ctx context.Context, peerID string, request data.DataRequest, resp *dataResponse, status, errorMessage string, bytesReceived int64) error {
	repo, ok := nm.host.repo.(transferWriter)
	if !ok {
		return nil
	}

	now := time.Now().UTC()
	var existing *data.DataTransfer
	if history, ok := nm.host.repo.(transferHistoryStore); ok {
		existing = findTransferByID(ctx, history, request.TransferID)
	}
	transfer := &data.DataTransfer{
		ID:          request.TransferID,
		RequestID:   request.RequestID,
		Type:        request.Type,
		Symbol:      request.Symbol,
		Source:      peerID,
		Destination: "local",
		StartDate:   request.StartDate,
		EndDate:     request.EndDate,
		Granularity: request.Granularity,
		Status:      status,
		StartTime:   now,
		ChunkSize:   request.ChunkSize,
		Error:       errorMessage,
	}
	if existing != nil {
		transfer.StartTime = existing.StartTime
		transfer.TotalRows = existing.TotalRows
		transfer.TotalChunks = existing.TotalChunks
		transfer.CompletedChunks = existing.CompletedChunks
		transfer.ResumeOffset = existing.ResumeOffset
		transfer.Size = existing.Size
		transfer.Speed = existing.Speed
		transfer.Progress = existing.Progress
	}
	if resp != nil {
		transfer.TotalRows = resp.TotalRows
		transfer.TotalChunks = resp.TotalChunks
		transfer.CompletedChunks = resp.TotalChunks
		if resp.TotalRows > 0 {
			transfer.CompletedChunks = resp.ChunkIndex + 1
		}
		transfer.ResumeOffset = resp.NextOffset
		rowSize := int64(512)
		rowCount := responseRowCount(*resp)
		if bytesReceived > 0 && rowCount > 0 {
			rowSize = bytesReceived / int64(rowCount)
			if rowSize <= 0 {
				rowSize = 512
			}
		}
		transfer.Size = int64(resp.NextOffset) * rowSize
		if transfer.Size < bytesReceived {
			transfer.Size = bytesReceived
		}
		transfer.Speed = float64(bytesReceived)
		elapsed := now.Sub(transfer.StartTime).Seconds()
		if elapsed > 0 && transfer.Size > 0 {
			transfer.Speed = float64(transfer.Size) / elapsed
		}
		if resp.TotalRows > 0 {
			transfer.Progress = float64(resp.NextOffset) / float64(resp.TotalRows) * 100
		} else if status == "completed" {
			transfer.Progress = 100
		}
	} else if request.Offset > 0 {
		transfer.ResumeOffset = request.Offset
		transfer.CompletedChunks = (request.Offset + request.ChunkSize - 1) / request.ChunkSize
	}
	if status == "completed" {
		transfer.Progress = 100
		transfer.EndTime = now
	}
	if status == "failed" {
		transfer.EndTime = now
	}
	return repo.SaveTransfer(ctx, transfer)
}

func findTransferByID(ctx context.Context, repo transferHistoryStore, id string) *data.DataTransfer {
	if id == "" {
		return nil
	}
	transfers, err := repo.ListTransfers(ctx)
	if err != nil {
		return nil
	}
	for _, transfer := range transfers {
		if transfer.ID == id {
			copyTransfer := transfer
			return &copyTransfer
		}
	}
	return nil
}

func (nm *NetworkManager) saveTransferFailure(ctx context.Context, peerID string, request data.DataRequest, err error) {
	if request.TransferID == "" {
		return
	}
	nm.host.metrics.RecordTransferFailed(err)
	_ = nm.saveTransferProgress(ctx, peerID, request, nil, "failed", err.Error(), 0)
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
