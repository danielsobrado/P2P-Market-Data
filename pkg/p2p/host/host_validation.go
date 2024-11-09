package host

import (
	"context"
	"time"

	libp2pPeer "github.com/libp2p/go-libp2p-core/peer"
)

// processValidationRequests processes validation requests from the validation channel
func (h *Host) processValidationRequests(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-h.shutdown:
			return
		case req := <-h.validation:
			h.handleValidationRequest(ctx, req)
		}
	}
}

// handleValidationRequest handles a single validation request
func (h *Host) handleValidationRequest(ctx context.Context, req *ValidationRequest) {
	startTime := time.Now()

	// Perform validation using the validator
	isValid, score := h.validator.Validate(req.MarketData)

	result := &ValidationResult{
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
