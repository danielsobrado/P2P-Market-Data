package p2p

import (
	"time"

	"p2p_market_data/pkg/data"

	"github.com/libp2p/go-libp2p-core/peer"
)

// validateMarketData performs validation on the provided market data
func (h *Host) validateMarketData(marketData *data.MarketData) *ValidationResult {
	// Use the validator to validate the data
	isValid, score := h.validator.Validate(marketData)

	result := &ValidationResult{
		MarketDataID: marketData.ID,
		IsValid:      isValid,
		Score:        score,
		ValidatedBy:  []peer.ID{h.host.ID()},
		CompletedAt:  time.Now(),
	}

	return result
}
