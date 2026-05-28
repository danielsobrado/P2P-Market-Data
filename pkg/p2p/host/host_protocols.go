package host

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
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
	Type      string               `json:"type"`
	EOD       []data.EODData       `json:"eod,omitempty"`
	Dividends []*data.DividendData `json:"dividends,omitempty"`
	Insiders  []data.InsiderTrade  `json:"insiders,omitempty"`
	Splits    []data.SplitData     `json:"splits,omitempty"`
	Error     string               `json:"error,omitempty"`
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
		return
	}

	ctx := h.ctx
	if ctx == nil {
		ctx = context.Background()
	}
	resp, err := h.lookupData(ctx, req)
	if err != nil {
		resp = dataResponse{Type: req.Type, Error: err.Error()}
	}

	writer := bufio.NewWriter(stream)
	if err := json.NewEncoder(writer).Encode(resp); err != nil {
		h.logger.Warn("Failed to encode data response stream", zap.Error(err))
		return
	}
	if err := writer.Flush(); err != nil {
		h.logger.Warn("Failed to flush data response stream", zap.Error(err))
	}
}

func (h *Host) lookupData(ctx context.Context, req data.DataRequest) (dataResponse, error) {
	if h.repo == nil {
		return dataResponse{}, fmt.Errorf("repository not initialized")
	}

	resp := dataResponse{Type: req.Type}
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
	return resp, nil
}
