package voting

import (
	"context"
	"fmt"
	"sync"
	"time"

	"p2p_market_data/pkg/config"
	"p2p_market_data/pkg/data"
	"p2p_market_data/pkg/p2p/message"
	"p2p_market_data/pkg/p2p/host"

	"go.uber.org/zap"
)

const (
	ValidationTopic = "validation"
)

// VotingSystem manages the consensus voting process
type VotingSystem struct {
	host          *host.Host
	activeVotes   map[string]*VotingSession
	votingTimeout time.Duration
	minVoters     int
	quorum        float64
	logger        *zap.Logger
	metrics       *VotingMetrics
	mu            sync.RWMutex
}

// NewVotingSystem creates a new voting system
func NewVotingSystem(host *host.Host, logger *zap.Logger, cfg *config.P2PConfig) *VotingSystem {
	return &VotingSystem{
		host:          host,
		activeVotes:   make(map[string]*VotingSession),
		votingTimeout: cfg.VotingTimeout,
		minVoters:     cfg.MinVoters,
		quorum:        cfg.ValidationQuorum,
		logger:        logger,
		metrics:       NewVotingMetrics(),
	}
}

// StartVoting initiates a new voting session
func (vs *VotingSystem) StartVoting(ctx context.Context, marketData *data.MarketData) (*VoteResult, error) {
	// Create new voting session
	session := NewVotingSession(marketData, vs.votingTimeout, vs.minVoters, vs.quorum)

	// Register session
	vs.mu.Lock()
	if _, exists := vs.activeVotes[session.ID]; exists {
		vs.mu.Unlock()
		return nil, fmt.Errorf("voting session already exists for market data: %s", session.ID)
	}
	vs.activeVotes[session.ID] = session
	vs.mu.Unlock()

	vs.metrics.IncrementSessionsStarted()

	// Broadcast vote request
	if err := vs.broadcastVoteRequest(ctx, session); err != nil {
		vs.failSession(session.ID, fmt.Sprintf("broadcasting vote request: %v", err))
		return nil, fmt.Errorf("broadcasting vote request: %w", err)
	}

	// Wait for result or timeout
	select {
	case result := <-session.ResponseChan:
		return result, nil
	case <-time.After(vs.votingTimeout):
		vs.failSession(session.ID, "voting timeout")
		return nil, fmt.Errorf("voting timeout")
	case <-ctx.Done():
		vs.failSession(session.ID, "context cancelled")
		return nil, ctx.Err()
	}
}

// SubmitVote submits a vote for a market data validation
func (vs *VotingSystem) SubmitVote(vote *data.Vote) error {
	if err := vs.validateVote(vote); err != nil {
		return err
	}

	vs.mu.RLock()
	session, exists := vs.activeVotes[vote.MarketDataID]
	vs.mu.RUnlock()
	if !exists {
		return fmt.Errorf("voting session not found: %s", vote.MarketDataID)
	}

	// Add vote to session
	if err := session.AddVote(vote); err != nil {
		return fmt.Errorf("adding vote: %w", err)
	}

	// Check if we have enough votes to conclude
	if session.ShouldConclude() {
		vs.concludeVoting(session)
	}

	return nil
}

// Private methods

func (vs *VotingSystem) broadcastVoteRequest(ctx context.Context, session *VotingSession) error {
	msg := message.NewMessage(message.MessageType(VoteRequestMessage), &VoteRequest{
		MarketDataID: session.MarketData.ID,
		Deadline:     time.Now().Add(vs.votingTimeout),
		MinVotes:     vs.minVoters,
	})

	// Publish to validation topic
	topic, err := vs.host.GetTopic(ValidationTopic)
	if err != nil {
		return fmt.Errorf("validation topic not initialized: %w", err)
	}

	msgBytes, err := msg.Marshal()
	if err != nil {
		return fmt.Errorf("marshaling vote request: %w", err)
	}

	return topic.Publish(ctx, msgBytes)
}

func (vs *VotingSystem) concludeVoting(session *VotingSession) {
	vs.mu.Lock()
	defer vs.mu.Unlock()

	if session.Status != VoteStatusActive {
		return
	}

	// Calculate voting result
	result := session.CalculateResult()

	session.Status = VoteStatusComplete
	session.Result = result
	session.EndTime = result.CompletedAt

	// Update metrics
	vs.metrics.IncrementSessionsComplete()
	vs.metrics.UpdateAverageLatency(session.EndTime.Sub(session.StartTime))

	// Send result
	session.ResponseChan <- result

	// Cleanup session
	go vs.cleanupSession(session.ID)
}

func (vs *VotingSystem) failSession(sessionID string, reason string) {
	vs.mu.Lock()
	defer vs.mu.Unlock()

	session, exists := vs.activeVotes[sessionID]
	if !exists {
		return
	}

	session.Status = VoteStatusFailed
	session.EndTime = time.Now()

	result := &VoteResult{
		MarketDataID: sessionID,
		Accepted:     false,
		VoteCount:    len(session.Votes),
		Score:        0,
		CompletedAt:  session.EndTime,
		ErrorMsg:     reason,
	}

	// Update metrics
	vs.metrics.IncrementSessionsFailed()

	// Send result
	session.ResponseChan <- result

	// Cleanup session
	go vs.cleanupSession(sessionID)

	vs.logger.Warn("Voting session failed",
		zap.String("sessionID", sessionID),
		zap.String("reason", reason))
}

func (vs *VotingSystem) cleanupSession(sessionID string) {
	// Wait some time before cleanup to allow for late votes
	time.Sleep(vs.votingTimeout)

	vs.mu.Lock()
	defer vs.mu.Unlock()

	delete(vs.activeVotes, sessionID)
}

// GetVotingStats returns current voting system statistics
func (vs *VotingSystem) GetVotingStats() VotingStats {
	return vs.metrics.GetStats(len(vs.activeVotes))
}

// GetActiveVotingSessions returns all active voting sessions
func (vs *VotingSystem) GetActiveVotingSessions() []*VotingSessionInfo {
	vs.mu.RLock()
	defer vs.mu.RUnlock()

	sessions := make([]*VotingSessionInfo, 0, len(vs.activeVotes))
	for _, session := range vs.activeVotes {
		sessions = append(sessions, session.GetInfo())
	}

	return sessions
}

// validateVote checks if a vote is valid
func (vs *VotingSystem) validateVote(vote *data.Vote) error {
	if vote == nil {
		return fmt.Errorf("vote cannot be nil")
	}
	if vote.MarketDataID == "" {
		return fmt.Errorf("market data ID cannot be empty")
	}
	if vote.ValidatorID == "" {
		return fmt.Errorf("validator ID cannot be empty")
	}
	if vote.Confidence < 0 || vote.Confidence > 1 {
		return fmt.Errorf("confidence must be between 0 and 1")
	}
	if vote.Signature == nil {
		return fmt.Errorf("vote must be signed")
	}
	return nil
}

// GetVoteResult retrieves the result of a completed voting session
func (vs *VotingSystem) GetVoteResult(sessionID string) (*VoteResult, error) {
	vs.mu.RLock()
	defer vs.mu.RUnlock()

	session, exists := vs.activeVotes[sessionID]
	if !exists {
		return nil, fmt.Errorf("voting session not found: %s", sessionID)
	}

	if session.Status != VoteStatusComplete {
		return nil, fmt.Errorf("voting session not complete: %s", sessionID)
	}

	return session.Result, nil
}

// UpdateConfig updates the voting system configuration
func (vs *VotingSystem) UpdateConfig(cfg VotingSystemConfig) error {
	if cfg.VotingTimeout <= 0 {
		return fmt.Errorf("voting timeout must be positive")
	}
	if cfg.MinVoters <= 0 {
		return fmt.Errorf("minimum voters must be positive")
	}
	if cfg.Quorum <= 0 || cfg.Quorum > 1 {
		return fmt.Errorf("quorum must be between 0 and 1")
	}

	vs.mu.Lock()
	defer vs.mu.Unlock()

	vs.votingTimeout = cfg.VotingTimeout
	vs.minVoters = cfg.MinVoters
	vs.quorum = cfg.Quorum

	vs.logger.Info("Voting system configuration updated",
		zap.Duration("timeout", cfg.VotingTimeout),
		zap.Int("minVoters", cfg.MinVoters),
		zap.Float64("quorum", cfg.Quorum))

	return nil
}
