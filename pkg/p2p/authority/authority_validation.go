package authority

import (
	"context"
	"fmt"
	"sync"
	"time"

	"p2p_market_data/pkg/data"
	"p2p_market_data/pkg/p2p/message"

	libp2pPeer "github.com/libp2p/go-libp2p/core/peer"
)

// ValidateData validates a single market data entry.
func (an *AuthorityNode) ValidateData(ctx context.Context, marketData *data.MarketData) (*ValidationResult, error) {
	startTime := time.Now()
	result := an.validateDataInternal(marketData)
	duration := time.Since(startTime)

	an.updateMetrics(result, duration)
	return result, nil
}

// ValidateMarketData validates multiple market data entries in batch.
func (an *AuthorityNode) ValidateMarketData(ctx context.Context, marketDataList []*data.MarketData) ([]*ValidationResult, error) {
	var wg sync.WaitGroup
	results := make([]*ValidationResult, len(marketDataList))
	errChan := make(chan error, len(marketDataList))

	for i, md := range marketDataList {
		wg.Add(1)
		go func(index int, md *data.MarketData) {
			defer wg.Done()
			result, err := an.ValidateData(ctx, md)
			if err != nil {
				errChan <- fmt.Errorf("validation error at index %d: %w", index, err)
				return
			}
			results[index] = result
		}(i, md)
	}

	wg.Wait()
	close(errChan)

	for err := range errChan {
		if err != nil {
			return nil, err
		}
	}

	return results, nil
}

// validateDataInternal performs the actual validation logic.
func (an *AuthorityNode) validateDataInternal(marketData *data.MarketData) *ValidationResult {
	startTime := time.Now()
	result := &ValidationResult{
		MarketDataID: marketData.ID,
		ValidatedBy:  []libp2pPeer.ID{an.host.ID()},
		CompletedAt:  time.Now(),
	}

	if valid, score := an.validator.Validate(marketData); valid {
		result.IsValid = true
		result.Score = score
	} else {
		result.IsValid = false
		result.ErrorMsg = "Data validation failed"
	}

	duration := time.Since(startTime)
	an.updateMetrics(result, duration)
	return result
}

// processValidations processes validation requests from the validations channel.
func (an *AuthorityNode) processValidations(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case req, ok := <-an.validations:
			if !ok {
				return
			}
			result := an.validateDataInternal(req.MarketData)
			convertedResult := &message.ValidationResult{
				// Populate necessary fields from result
				IsValid:  result.IsValid,
				Score:    result.Score,
				ErrorMsg: result.ErrorMsg,
				// Add other required fields
			}
			select {
			case req.ResponseCh <- convertedResult:
			case <-ctx.Done():
				return
			}
		}
	}
}
