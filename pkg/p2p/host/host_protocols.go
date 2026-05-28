package host

import (
	"bufio"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math"
	"time"

	"p2p_market_data/pkg/data"

	libp2pNetwork "github.com/libp2p/go-libp2p/core/network"
	libp2pPeer "github.com/libp2p/go-libp2p/core/peer"
	libp2pMultiaddr "github.com/multiformats/go-multiaddr"
	"go.uber.org/zap"
)

// PeerDiscoveryRequest represents a peer discovery request
type PeerDiscoveryRequest struct {
	// Define fields if needed
}

// PeerDiscoveryResponse represents a peer discovery response
type PeerDiscoveryResponse struct {
	PeerID libp2pPeer.ID
	Addrs  []libp2pMultiaddr.Multiaddr
}

type dataResponse struct {
	RequestID       string               `json:"request_id,omitempty"`
	TransferID      string               `json:"transfer_id,omitempty"`
	ResponderPeerID string               `json:"responder_peer_id,omitempty"`
	ResponderPubKey []byte               `json:"responder_public_key,omitempty"`
	RespondedAt     int64                `json:"responded_at,omitempty"`
	Nonce           string               `json:"nonce,omitempty"`
	Signature       []byte               `json:"signature,omitempty"`
	Type            string               `json:"type"`
	Offset          int                  `json:"offset,omitempty"`
	NextOffset      int                  `json:"next_offset,omitempty"`
	ChunkIndex      int                  `json:"chunk_index,omitempty"`
	ChunkSize       int                  `json:"chunk_size,omitempty"`
	TotalRows       int                  `json:"total_rows,omitempty"`
	TotalChunks     int                  `json:"total_chunks,omitempty"`
	HasMore         bool                 `json:"has_more,omitempty"`
	Checksum        string               `json:"checksum,omitempty"`
	EOD             []data.EODData       `json:"eod,omitempty"`
	Dividends       []*data.DividendData `json:"dividends,omitempty"`
	Insiders        []data.InsiderTrade  `json:"insiders,omitempty"`
	Splits          []data.SplitData     `json:"splits,omitempty"`
	Error           string               `json:"error,omitempty"`
}

type splitReader interface {
	GetSplitData(ctx context.Context, symbol, startDate, endDate string) ([]data.SplitData, error)
}

// handleDiscoveryStream handles peer discovery protocol streams
func (h *Host) handleDiscoveryStream(stream libp2pNetwork.Stream) {
	defer stream.Close()

	peerID := stream.Conn().RemotePeer()
	h.logger.Debug("Received discovery stream",
		zap.String("protocol", string(stream.Protocol())),
		zap.String("peer", peerID.String()))

	// Read discovery request
	var req PeerDiscoveryRequest
	if err := json.NewDecoder(bufio.NewReader(stream)).Decode(&req); err != nil {
		h.logger.Error("Failed to decode discovery request", zap.Error(err))
		return
	}

	// Respond with peer info
	resp := PeerDiscoveryResponse{
		PeerID: h.host.ID(),
		Addrs:  h.host.Addrs(),
	}

	writer := bufio.NewWriter(stream)
	if err := json.NewEncoder(writer).Encode(resp); err != nil {
		h.logger.Error("Failed to encode discovery response", zap.Error(err))
		return
	}

	if err := writer.Flush(); err != nil {
		h.logger.Error("Failed to flush discovery response", zap.Error(err))
		return
	}

	h.logger.Debug("Sent discovery response", zap.String("peer", peerID.String()))
}

// handleValidationStream handles validation protocol streams
func (h *Host) handleValidationStream(stream libp2pNetwork.Stream) {
	defer stream.Close()
	// Implement validation stream handling logic
}

// handleStream handles generic protocol streams
func (h *Host) handleStream(stream libp2pNetwork.Stream) {
	defer stream.Close()

	var req data.DataRequest
	if err := json.NewDecoder(bufio.NewReader(stream)).Decode(&req); err != nil {
		h.logger.Warn("Failed to decode data request stream", zap.Error(err))
		h.metrics.RecordRequestRejected(err)
		return
	}

	ctx := h.ctx
	if ctx == nil {
		ctx = context.Background()
	}
	h.metrics.RecordRequestReceived()
	if err := h.verifyDataRequest(stream.Conn().RemotePeer(), &req); err != nil {
		h.logger.Warn("Rejected unauthenticated data request",
			zap.String("peer", stream.Conn().RemotePeer().String()),
			zap.Error(err))
		h.metrics.RecordAuthFailure(err)
		resp := dataResponse{
			RequestID:  req.RequestID,
			TransferID: req.TransferID,
			Type:       req.Type,
			Error:      fmt.Sprintf("authentication failed: %v", err),
		}
		_ = h.signDataResponse(&resp, req)
		writeDataResponse(h, stream, resp)
		return
	}

	resp, err := h.lookupData(ctx, req)
	if err != nil {
		h.metrics.RecordRequestRejected(err)
		resp = dataResponse{
			RequestID:  req.RequestID,
			TransferID: req.TransferID,
			Type:       req.Type,
			Error:      err.Error(),
		}
	}
	if err := h.signDataResponse(&resp, req); err != nil {
		h.logger.Warn("Failed to sign data response", zap.Error(err))
		h.metrics.RecordRequestRejected(err)
		resp = dataResponse{
			RequestID:  req.RequestID,
			TransferID: req.TransferID,
			Type:       req.Type,
			Error:      fmt.Sprintf("response signing failed: %v", err),
		}
		_ = h.signDataResponse(&resp, req)
	}

	writeDataResponse(h, stream, resp)
}

func writeDataResponse(h *Host, stream libp2pNetwork.Stream, resp dataResponse) {
	writer := bufio.NewWriter(stream)
	if err := json.NewEncoder(writer).Encode(resp); err != nil {
		h.logger.Warn("Failed to encode data response stream", zap.Error(err))
		return
	}
	if err := writer.Flush(); err != nil {
		h.logger.Warn("Failed to flush data response stream", zap.Error(err))
	}
	if content, err := json.Marshal(resp); err == nil {
		h.metrics.RecordResponseSent(resp, int64(len(content)))
	}
}

func (h *Host) lookupData(ctx context.Context, req data.DataRequest) (dataResponse, error) {
	if h.repo == nil {
		return dataResponse{}, fmt.Errorf("repository not initialized")
	}

	resp := dataResponse{
		RequestID:  req.RequestID,
		TransferID: req.TransferID,
		Type:       req.Type,
	}
	switch req.Type {
	case data.DataTypeEOD:
		rows, err := h.repo.GetEODData(ctx, req.Symbol, req.StartDate, req.EndDate)
		if err != nil {
			return resp, err
		}
		resp.EOD = rows
	case data.DataTypeDividend:
		start, err := time.Parse("2006-01-02", req.StartDate)
		if err != nil {
			return resp, err
		}
		end, err := time.Parse("2006-01-02", req.EndDate)
		if err != nil {
			return resp, err
		}
		rows, err := h.repo.GetDividendData(ctx, req.Symbol, start, end)
		if err != nil {
			return resp, err
		}
		resp.Dividends = rows
	case data.DataTypeInsiderTrade:
		rows, err := h.repo.GetInsiderData(ctx, req.Symbol, req.StartDate, req.EndDate)
		if err != nil {
			return resp, err
		}
		resp.Insiders = rows
	case data.DataTypeSplit:
		repo, ok := h.repo.(splitReader)
		if !ok {
			return resp, fmt.Errorf("split repository not available")
		}
		rows, err := repo.GetSplitData(ctx, req.Symbol, req.StartDate, req.EndDate)
		if err != nil {
			return resp, err
		}
		resp.Splits = rows
	default:
		return resp, fmt.Errorf("unsupported data type: %s", req.Type)
	}
	return chunkDataResponse(resp, req)
}

func chunkDataResponse(resp dataResponse, req data.DataRequest) (dataResponse, error) {
	totalRows := responseRowCount(resp)
	chunkSize := normalizeChunkSize(req.ChunkSize)
	offset := req.Offset
	if offset < 0 {
		offset = 0
	}
	if offset > totalRows {
		offset = totalRows
	}
	nextOffset := offset + chunkSize
	if nextOffset > totalRows {
		nextOffset = totalRows
	}

	resp.Offset = offset
	resp.NextOffset = nextOffset
	resp.ChunkSize = chunkSize
	resp.TotalRows = totalRows
	if totalRows > 0 {
		resp.TotalChunks = int(math.Ceil(float64(totalRows) / float64(chunkSize)))
		resp.ChunkIndex = offset / chunkSize
	}
	resp.HasMore = nextOffset < totalRows
	sliceResponseRows(&resp, offset, nextOffset)
	checksum, err := responseChecksum(resp)
	if err != nil {
		return resp, err
	}
	resp.Checksum = checksum
	return resp, nil
}

func normalizeChunkSize(size int) int {
	switch {
	case size <= 0:
		return 100
	case size > 1000:
		return 1000
	default:
		return size
	}
}

func responseRowCount(resp dataResponse) int {
	switch resp.Type {
	case data.DataTypeEOD:
		return len(resp.EOD)
	case data.DataTypeDividend:
		return len(resp.Dividends)
	case data.DataTypeInsiderTrade:
		return len(resp.Insiders)
	case data.DataTypeSplit:
		return len(resp.Splits)
	default:
		return 0
	}
}

func sliceResponseRows(resp *dataResponse, start, end int) {
	switch resp.Type {
	case data.DataTypeEOD:
		resp.EOD = resp.EOD[start:end]
	case data.DataTypeDividend:
		resp.Dividends = resp.Dividends[start:end]
	case data.DataTypeInsiderTrade:
		resp.Insiders = resp.Insiders[start:end]
	case data.DataTypeSplit:
		resp.Splits = resp.Splits[start:end]
	}
}

func responseChecksum(resp dataResponse) (string, error) {
	resp.Checksum = ""
	resp.Signature = nil
	content, err := json.Marshal(resp)
	if err != nil {
		return "", fmt.Errorf("marshaling response checksum payload: %w", err)
	}
	sum := sha256.Sum256(content)
	return hex.EncodeToString(sum[:]), nil
}
