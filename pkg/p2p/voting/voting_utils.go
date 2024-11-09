package voting

import (
	"context"
	"fmt"
	"sync"
	"time"

	"p2p_market_data/pkg/data"

	"github.com/libp2p/go-libp2p-core/peer"
)

// BatchVoting handles multiple voting sessions in parallel
func (vs *VotingSystem) BatchVoting(ctx context.Context, marketDataBatch []*data.MarketData) ([]*VoteResult, error) {
	results := make([]*VoteResult, len(marketDataBatch))
	var wg sync.WaitGroup
	errChan := make(chan error, len(marketDataBatch))

	for i, md := range marketDataBatch {
		wg.Add(1)
		go func(index int, marketData *data.MarketData) {
			defer wg.Done()

			result, err := vs.StartVoting(ctx, marketData)
			if err != nil {
				errChan <- fmt.Errorf("voting for data at index %d: %w", index, err)
				return
			}

			results[index] = result
		}(i, md)
	}

	// Wait for all voting sessions to complete
	wg.Wait()
	close(errChan)

	// Check for errors
	for err := range errChan {
		if err != nil {
			return nil, err
		}
	}

	return results, nil
}

// GetSessionDetails retrieves detailed information about a voting session
func (vs *VotingSystem) GetSessionDetails(sessionID string) (*VotingSessionDetails, error) {
	vs.mu.RLock()
	defer vs.mu.RUnlock()

	session, exists := vs.activeVotes[sessionID]
	if !exists {
		return nil, fmt.Errorf("voting session not found: %s", sessionID)
	}

	details := session.GetDetails()
	return details, nil
}

// VotingSessionDetails represents detailed information about a voting session
type VotingSessionDetails struct {
	ID           string
	StartTime    time.Time
	EndTime      time.Time
	Status       VoteStatus
	VoteCount    int
	MarketDataID string
	Votes        []*VoteDetails
}

// VoteDetails represents details about an individual vote
type VoteDetails struct {
	ValidatorID peer.ID
	IsValid     bool
	Confidence  float64
	Timestamp   time.Time
}

// GetDetails returns detailed information about the voting session
func (vs *VotingSession) GetDetails() *VotingSessionDetails {
	vs.mu.RLock()
	defer vs.mu.RUnlock()

	votes := make([]*VoteDetails, 0, len(vs.Votes))
	for validatorID, vote := range vs.Votes {
		votes = append(votes, &VoteDetails{
			ValidatorID: validatorID,
			IsValid:     vote.IsValid,
			Confidence:  vote.Confidence,
			Timestamp:   vote.Timestamp,
		})
	}

	return &VotingSessionDetails{
		ID:           vs.ID,
		StartTime:    vs.StartTime,
		EndTime:      vs.EndTime,
		Status:       vs.Status,
		VoteCount:    len(vs.Votes),
		MarketDataID: vs.MarketData.ID,
		Votes:        votes,
	}
}
