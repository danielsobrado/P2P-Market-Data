package security

import (
	"fmt"
	"p2p_market_data/pkg/config"
	"p2p_market_data/pkg/data"
)

// Validator performs validation on market data.
type Validator struct {
	minReputationScore float64
	maxPenalty         float64
	minConfidence      float64
}

// NewValidator creates a new Validator instance with the provided configuration.
func NewValidator(cfg config.SecurityConfig) (*Validator, error) {
	if cfg.MaxPenalty <= 0 || cfg.MaxPenalty > 1 {
		return nil, fmt.Errorf("max_penalty must be between 0 and 1")
	}
	if cfg.MinConfidence <= 0 || cfg.MinConfidence > 1 {
		return nil, fmt.Errorf("min_confidence must be between 0 and 1")
	}
	// Initialize other fields and return the Validator
	return &Validator{
		maxPenalty:    cfg.MaxPenalty,
		minConfidence: cfg.MinConfidence,
	}, nil
}

// Validate runs the market data through the validation process.
func (v *Validator) Validate(marketData *data.MarketData) (bool, float64) {
	// Implement your validation logic here
	// For example, check marketData against certain criteria, use external data sources, etc.

	// Placeholder implementation:
	// Let's assume the market data is valid if the price is within an acceptable range
	isValid := v.isPriceValid(marketData.Price)
	score := v.calculateScore(isValid)

	return isValid, score
}

// isPriceValid checks if the price is within an acceptable range.
func (v *Validator) isPriceValid(price float64) bool {
	// Placeholder logic
	return price > 0
}

// calculateScore calculates a validation score based on the validity.
func (v *Validator) calculateScore(isValid bool) float64 {
	if isValid {
		// Return a high score if valid
		return v.minReputationScore + (1.0-v.minReputationScore)*(1.0-v.maxPenalty)
	}
	// Return a low score if not valid
	return v.minReputationScore * v.maxPenalty
}
