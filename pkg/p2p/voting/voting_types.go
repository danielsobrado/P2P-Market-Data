package voting

import (
	"time"
)

// VoteStatus represents the status of a voting session
type VoteStatus string

const (
	VoteStatusPending  VoteStatus = "pending"
	VoteStatusActive   VoteStatus = "active"
	VoteStatusComplete VoteStatus = "complete"
	VoteStatusFailed   VoteStatus = "failed"
)

// VotingStats represents voting system statistics
type VotingStats struct {
	ActiveSessions   int
	SessionsStarted  int64
	SessionsComplete int64
	SessionsFailed   int64
	AverageLatency   time.Duration
	LastUpdate       time.Time
}

// VotingSessionInfo represents summary information about a voting session
type VotingSessionInfo struct {
	ID        string
	StartTime time.Time
	Status    VoteStatus
	VoteCount int
}

// VoteResult represents the result of a voting session
type VoteResult struct {
	MarketDataID string
	Accepted     bool
	VoteCount    int
	Score        float64
	CompletedAt  time.Time
	ErrorMsg     string
}

// VoteRequest represents a request for votes
type VoteRequest struct {
	MarketDataID string
	Deadline     time.Time
	MinVotes     int
}

// VotingSystemConfig allows runtime configuration updates
type VotingSystemConfig struct {
	VotingTimeout time.Duration
	MinVoters     int
	Quorum        float64
}

// VoteMessageType represents the type of vote-related messages
type VoteMessageType string

const (
	VoteRequestMessage  VoteMessageType = "VoteRequest"
	VoteResponseMessage VoteMessageType = "VoteResponse"
)
