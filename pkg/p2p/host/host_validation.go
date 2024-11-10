package host

import (
	"context"
	"p2p_market_data/pkg/data"
	"p2p_market_data/pkg/p2p/message"
	"time"

	libp2pPeer "github.com/libp2p/go-libp2p/core/peer"
)

// processValidationRequest handles a single validation request
func (h *Host) processValidationRequest(ctx context.Context, req *message.ValidationRequest) {
	startTime := time.Now()

	// Perform validation using the validator
	isValid, score := h.validator.Validate(req.MarketData)

	result := &message.ValidationResult{
		MarketDataID: req.MarketData.ID,
		IsValid:      isValid,
		Score:        score,
		CompletedAt:  time.Now(),
		ValidatedBy:  []libp2pPeer.ID{h.host.ID()},
	}

	// Send result back to requester
	select {
	case req.ResponseCh <- result:
	case <-ctx.Done():
	}

	// Update metrics
	duration := time.Since(startTime)
	h.metrics.UpdateValidationLatency(duration)
	if !isValid {
		h.metrics.IncrementFailedValidations()
	}
}

// validateMarketData performs validation on the provided market data
func (h *Host) validateMarketData(marketData *data.MarketData) *message.ValidationResult {
	// Use the validator to validate the data
	isValid, score := h.validator.Validate(marketData)

	result := &message.ValidationResult{
		MarketDataID: marketData.ID,
		IsValid:      isValid,
		Score:        score,
		ValidatedBy:  []libp2pPeer.ID{h.host.ID()},
		CompletedAt:  time.Now(),
	}

	return result
}
