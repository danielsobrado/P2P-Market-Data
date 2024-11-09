package voting

import (
	"fmt"
	"sync"
	"time"

	"p2p_market_data/pkg/data"

	"github.com/libp2p/go-libp2p-core/peer"
)

// VotingSession represents an active voting session
type VotingSession struct {
	ID           string
	MarketData   *data.MarketData
	Votes        map[peer.ID]*data.Vote
	StartTime    time.Time
	EndTime      time.Time
	Status       VoteStatus
	Result       *VoteResult
	ResponseChan chan *VoteResult
	mu           sync.RWMutex

	votingTimeout time.Duration
	minVoters     int
	quorum        float64
}

// NewVotingSession creates a new voting session
func NewVotingSession(marketData *data.MarketData, votingTimeout time.Duration, minVoters int, quorum float64) *VotingSession {
	return &VotingSession{
		ID:            marketData.ID,
		MarketData:    marketData,
		Votes:         make(map[peer.ID]*data.Vote),
		StartTime:     time.Now(),
		Status:        VoteStatusActive,
		ResponseChan:  make(chan *VoteResult, 1),
		votingTimeout: votingTimeout,
		minVoters:     minVoters,
		quorum:        quorum,
	}
}

// AddVote adds a vote to the session
func (vs *VotingSession) AddVote(vote *data.Vote) error {
	vs.mu.Lock()
	defer vs.mu.Unlock()

	if vs.Status != VoteStatusActive {
		return fmt.Errorf("voting session not active")
	}

	vs.Votes[peer.ID(vote.ValidatorID)] = vote
	return nil
}

// ShouldConclude checks if the voting session should conclude
func (vs *VotingSession) ShouldConclude() bool {
	vs.mu.RLock()
	defer vs.mu.RUnlock()

	// Must have minimum number of votes
	if len(vs.Votes) < vs.minVoters {
		return false
	}

	// Check if we have reached quorum
	totalWeight := 0.0
	for _, vote := range vs.Votes {
		totalWeight += vote.Confidence
	}

	return totalWeight >= vs.quorum
}

// CalculateResult calculates the result of the voting session
func (vs *VotingSession) CalculateResult() *VoteResult {
	vs.mu.RLock()
	defer vs.mu.RUnlock()

	totalWeight := 0.0
	weightedAccept := 0.0

	for _, vote := range vs.Votes {
		weight := vote.Confidence
		totalWeight += weight
		if vote.IsValid {
			weightedAccept += weight
		}
	}

	score := 0.0
	if totalWeight > 0 {
		score = weightedAccept / totalWeight
	}

	result := &VoteResult{
		MarketDataID: vs.ID,
		Accepted:     score >= vs.quorum,
		VoteCount:    len(vs.Votes),
		Score:        score,
		CompletedAt:  time.Now(),
	}

	return result
}

// GetInfo returns summary information about the voting session
func (vs *VotingSession) GetInfo() *VotingSessionInfo {
	vs.mu.RLock()
	defer vs.mu.RUnlock()

	return &VotingSessionInfo{
		ID:        vs.ID,
		StartTime: vs.StartTime,
		Status:    vs.Status,
		VoteCount: len(vs.Votes),
	}
}
