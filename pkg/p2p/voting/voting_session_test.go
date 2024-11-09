package voting

import (
	"testing"
	"time"

	"p2p_market_data/pkg/data"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestVotingSession_AddVote(t *testing.T) {
	// Setup
	marketData := &data.MarketData{ID: "md1"}
	session := NewVotingSession(marketData, 5*time.Second, 1, 0.5)

	// Create a vote
	vote := &data.Vote{
		MarketDataID: "md1",
		ValidatorID:  "validator1",
		IsValid:      true,
		Confidence:   0.8,
		Timestamp:    time.Now(),
	}

	// Add vote
	err := session.AddVote(vote)
	require.NoError(t, err)
	assert.Equal(t, 1, len(session.Votes))

	// Attempt to add a vote after session completion
	session.Status = VoteStatusComplete
	vote2 := &data.Vote{
		MarketDataID: "md1",
		ValidatorID:  "validator2",
		IsValid:      false,
		Confidence:   0.9,
		Timestamp:    time.Now(),
	}
	err = session.AddVote(vote2)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "voting session not active")
}

func TestVotingSession_ShouldConclude(t *testing.T) {
	// Setup
	marketData := &data.MarketData{ID: "md1"}
	session := NewVotingSession(marketData, 5*time.Second, 2, 0.6)

	// Add votes
	vote1 := &data.Vote{
		MarketDataID: "md1",
		ValidatorID:  "validator1",
		IsValid:      true,
		Confidence:   0.5,
		Timestamp:    time.Now(),
	}
	vote2 := &data.Vote{
		MarketDataID: "md1",
		ValidatorID:  "validator2",
		IsValid:      true,
		Confidence:   0.4,
		Timestamp:    time.Now(),
	}

	session.AddVote(vote1)
	assert.False(t, session.ShouldConclude())

	session.AddVote(vote2)
	assert.True(t, session.ShouldConclude())
}

func TestVotingSession_CalculateResult(t *testing.T) {
	// Setup
	marketData := &data.MarketData{ID: "md1"}
	session := NewVotingSession(marketData, 5*time.Second, 1, 0.5)

	// Add votes
	vote1 := &data.Vote{
		MarketDataID: "md1",
		ValidatorID:  "validator1",
		IsValid:      true,
		Confidence:   0.6,
		Timestamp:    time.Now(),
	}
	vote2 := &data.Vote{
		MarketDataID: "md1",
		ValidatorID:  "validator2",
		IsValid:      false,
		Confidence:   0.4,
		Timestamp:    time.Now(),
	}

	session.AddVote(vote1)
	session.AddVote(vote2)

	result := session.CalculateResult()
	assert.Equal(t, 2, result.VoteCount)
	assert.True(t, result.Accepted)
	assert.InDelta(t, 0.6, result.Score, 0.01)
}

func TestVotingSession_GetInfo(t *testing.T) {
	// Setup
	marketData := &data.MarketData{ID: "md1"}
	session := NewVotingSession(marketData, 5*time.Second, 1, 0.5)

	// Add a vote
	vote := &data.Vote{
		MarketDataID: "md1",
		ValidatorID:  "validator1",
		IsValid:      true,
		Confidence:   0.8,
		Timestamp:    time.Now(),
	}
	session.AddVote(vote)

	info := session.GetInfo()
	assert.Equal(t, "md1", info.ID)
	assert.Equal(t, VoteStatusActive, info.Status)
	assert.Equal(t, 1, info.VoteCount)
}
