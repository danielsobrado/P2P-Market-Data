package authority

import (
	"context"
	"encoding/json"
	"fmt"
	"p2p_market_data/pkg/p2p/message"

	libp2pNetwork "github.com/libp2p/go-libp2p-core/network"
	"go.uber.org/zap"
)

const AuthorityProtocolID = "/p2p/authority/1.0.0"

// handleAuthorityStream handles incoming authority protocol streams.
func (an *AuthorityNode) handleAuthorityStream(stream libp2pNetwork.Stream) {
	defer stream.Close()

	peerID := stream.Conn().RemotePeer()
	an.logger.Debug("Received authority stream", zap.String("peer", peerID.String()))

	req, err := an.readValidationRequest(stream)
	if err != nil {
		an.logger.Error("Failed to read validation request", zap.Error(err), zap.String("peer", peerID.String()))
		an.writeErrorResponse(stream, err)
		return
	}

	if _, err := an.VerifyPeer(context.Background(), peerID); err != nil {
		an.logger.Error("Peer verification failed", zap.Error(err), zap.String("peer", peerID.String()))
		an.writeErrorResponse(stream, err)
		return
	}

	result := an.validateDataInternal(req.MarketData)

	if err := an.writeValidationResponse(stream, result); err != nil {
		an.logger.Error("Failed to write validation response", zap.Error(err), zap.String("peer", peerID.String()))
	}
}

// Helper functions for stream reading/writing.

func (an *AuthorityNode) readValidationRequest(stream libp2pNetwork.Stream) (*message.ValidationRequest, error) {
	var req message.ValidationRequest
	decoder := json.NewDecoder(stream)
	if err := decoder.Decode(&req); err != nil {
		return nil, fmt.Errorf("decoding validation request: %w", err)
	}
	return &req, nil
}

func (an *AuthorityNode) writeValidationResponse(stream libp2pNetwork.Stream, result *ValidationResult) error {
	encoder := json.NewEncoder(stream)
	if err := encoder.Encode(result); err != nil {
		return fmt.Errorf("encoding validation response: %w", err)
	}
	return nil
}

func (an *AuthorityNode) writeErrorResponse(stream libp2pNetwork.Stream, err error) error {
	resp := message.ErrorResponse{
		Code:    500,
		Message: "Internal Error",
		Details: err.Error(),
	}
	encoder := json.NewEncoder(stream)
	if err := encoder.Encode(resp); err != nil {
		return fmt.Errorf("encoding error response: %w", err)
	}
	return nil
}
