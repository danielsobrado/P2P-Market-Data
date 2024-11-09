package p2p

import (
	"bufio"
	"context"
	"encoding/json"
	"time"

	libp2pNetwork "github.com/libp2p/go-libp2p-core/network"
	libp2pPeer "github.com/libp2p/go-libp2p-core/peer"
	"go.uber.org/zap"
)

// handleStream handles incoming streams for the default protocol
func (h *Host) handleStream(stream libp2pNetwork.Stream) {
	defer stream.Close()

	peerID := stream.Conn().RemotePeer()
	h.logger.Debug("Received stream",
		zap.String("protocol", string(stream.Protocol())),
		zap.String("peer", peerID.String()))

	// Set read/write deadlines
	if err := stream.SetDeadline(time.Now().Add(streamReadTimeout)); err != nil {
		h.logger.Error("Failed to set stream deadline", zap.Error(err))
		return
	}

	// Read message from the stream
	var msg Message
	reader := bufio.NewReader(stream)
	decoder := json.NewDecoder(reader)

	if err := decoder.Decode(&msg); err != nil {
		h.logger.Error("Failed to decode message from stream", zap.Error(err))
		return
	}

	// Handle the message
	if err := h.handleMessage(context.Background(), &msg); err != nil {
		h.logger.Error("Failed to handle message from stream", zap.Error(err))
		return
	}

	h.logger.Debug("Successfully handled message from stream",
		zap.String("peer", peerID.String()))
}

// handleValidationStream handles incoming validation requests from peers
func (h *Host) handleValidationStream(stream libp2pNetwork.Stream) {
	defer stream.Close()

	peerID := stream.Conn().RemotePeer()
	h.logger.Debug("Received validation stream",
		zap.String("protocol", string(stream.Protocol())),
		zap.String("peer", peerID.String()))

	// Set read/write deadlines
	if err := stream.SetDeadline(time.Now().Add(streamReadTimeout)); err != nil {
		h.logger.Error("Failed to set stream deadline", zap.Error(err))
		return
	}

	// Read validation request from the stream
	var req ValidationRequest
	reader := bufio.NewReader(stream)
	decoder := json.NewDecoder(reader)

	if err := decoder.Decode(&req); err != nil {
		h.logger.Error("Failed to decode validation request", zap.Error(err))
		return
	}

	// Validate the market data
	result := h.validateMarketData(req.MarketData)

	// Send validation result back to the requester
	writer := bufio.NewWriter(stream)
	encoder := json.NewEncoder(writer)

	if err := encoder.Encode(result); err != nil {
		h.logger.Error("Failed to encode validation result", zap.Error(err))
		return
	}
	if err := writer.Flush(); err != nil {
		h.logger.Error("Failed to flush validation result", zap.Error(err))
		return
	}

	h.logger.Debug("Sent validation result",
		zap.String("peer", peerID.String()),
		zap.String("marketDataID", req.MarketData.ID))
}

// handleDiscoveryStream handles incoming peer discovery requests
func (h *Host) handleDiscoveryStream(stream libp2pNetwork.Stream) {
	defer stream.Close()

	peerID := stream.Conn().RemotePeer()
	h.logger.Debug("Received discovery stream",
		zap.String("protocol", string(stream.Protocol())),
		zap.String("peer", peerID.String()))

	// Read discovery request
	var req PeerDiscoveryRequest
	reader := bufio.NewReader(stream)
	decoder := json.NewDecoder(reader)

	if err := decoder.Decode(&req); err != nil {
		h.logger.Error("Failed to decode discovery request", zap.Error(err))
		return
	}

	// Respond with peer info
	resp := PeerDiscoveryResponse{
		PeerID: h.host.ID(),
		Addrs:  h.host.Addrs(),
	}

	writer := bufio.NewWriter(stream)
	encoder := json.NewEncoder(writer)

	if err := encoder.Encode(resp); err != nil {
		h.logger.Error("Failed to encode discovery response", zap.Error(err))
		return
	}
	if err := writer.Flush(); err != nil {
		h.logger.Error("Failed to flush discovery response", zap.Error(err))
		return
	}

	h.logger.Debug("Sent discovery response",
		zap.String("peer", peerID.String()))
}

// Additional data structures for discovery
type PeerDiscoveryRequest struct {
	// Define fields if needed
}

type PeerDiscoveryResponse struct {
	PeerID libp2pPeer.ID
	Addrs  []libp2pMultiaddr.Multiaddr
}
