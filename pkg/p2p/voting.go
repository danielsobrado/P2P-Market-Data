package p2p

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/libp2p/go-libp2p-core/peer"
	"go.uber.org/zap"

	"p2p_market_data/pkg/config"
	"p2p_market_data/pkg/data"
)

// VotingSystem manages the consensus voting process
type VotingSystem struct {
	host          *Host
	activeVotes   map[string]*VotingSession
	votingTimeout time.Duration
	minVoters     int
	quorum        float64
	logger        *zap.Logger
	metrics       *VotingMetrics
	mu            sync.RWMutex
}

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
}

// VoteStatus represents the status of a voting session
type VoteStatus string

const (
	VoteStatusPending  VoteStatus = "pending"
	VoteStatusActive   VoteStatus = "active"
	VoteStatusComplete VoteStatus = "complete"
	VoteStatusFailed   VoteStatus = "failed"
)

// VotingMetrics tracks voting system performance
type VotingMetrics struct {
	SessionsStarted  int64
	SessionsComplete int64
	SessionsFailed   int64
	AverageLatency   time.Duration
	LastUpdate       time.Time
	mu               sync.RWMutex
}

// NewVotingSystem creates a new voting system
func NewVotingSystem(host *Host, logger *zap.Logger, config *config.P2PConfig) *VotingSystem {
	return &VotingSystem{
		host:          host,
		activeVotes:   make(map[string]*VotingSession),
		votingTimeout: config.VotingTimeout,
		minVoters:     config.MinVoters,
		quorum:        config.ValidationQuorum,
		logger:        logger,
		metrics:       &VotingMetrics{},
	}
}

// StartVoting initiates a new voting session
func (vs *VotingSystem) StartVoting(ctx context.Context, marketData *data.MarketData) (*VoteResult, error) {
	// Create new voting session
	session := &VotingSession{
		ID:           marketData.ID,
		MarketData:   marketData,
		Votes:        make(map[peer.ID]*data.Vote),
		StartTime:    time.Now(),
		Status:       VoteStatusActive,
		ResponseChan: make(chan *VoteResult, 1),
	}

	// Register session
	vs.mu.Lock()
	if _, exists := vs.activeVotes[session.ID]; exists {
		vs.mu.Unlock()
		return nil, fmt.Errorf("voting session already exists for market data: %s", session.ID)
	}
	vs.activeVotes[session.ID] = session
	vs.mu.Unlock()

	vs.metrics.mu.Lock()
	vs.metrics.SessionsStarted++
	vs.metrics.LastUpdate = time.Now()
	vs.metrics.mu.Unlock()

	// Broadcast vote request
	if err := vs.broadcastVoteRequest(ctx, session); err != nil {
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
	vs.mu.Lock()
	session, exists := vs.activeVotes[vote.MarketDataID]
	if !exists {
		vs.mu.Unlock()
		return fmt.Errorf("voting session not found: %s", vote.MarketDataID)
	}

	// Add vote to session
	session.Votes[vote.ValidatorID] = vote
	vs.mu.Unlock()

	// Check if we have enough votes to conclude
	if vs.shouldConcludeVoting(session) {
		vs.concludeVoting(session)
	}

	return nil
}

// Private methods

func (vs *VotingSystem) broadcastVoteRequest(ctx context.Context, session *VotingSession) error {
	msg := NewMessage(VoteMessage, &VoteRequest{
		MarketDataID: session.MarketData.ID,
		Deadline:     time.Now().Add(vs.votingTimeout),
		MinVotes:     vs.minVoters,
	})

	// Publish to voting topic
	topic, ok := vs.host.topics[ValidationTopic]
	if !ok {
		return fmt.Errorf("validation topic not initialized")
	}

	msgBytes, err := msg.Marshal()
	if err != nil {
		return fmt.Errorf("marshaling vote request: %w", err)
	}

	return topic.Publish(ctx, msgBytes)
}

func (vs *VotingSystem) shouldConcludeVoting(session *VotingSession) bool {
	// Must have minimum number of votes
	if len(session.Votes) < vs.minVoters {
		return false
	}

	// Check if we have reached quorum
	totalWeight := 0.0
	for _, vote := range session.Votes {
		totalWeight += vote.Confidence
	}

	return totalWeight >= vs.quorum
}

func (vs *VotingSystem) concludeVoting(session *VotingSession) {
	vs.mu.Lock()
	defer vs.mu.Unlock()

	if session.Status != VoteStatusActive {
		return
	}

	// Calculate voting result
	totalWeight := 0.0
	weightedAccept := 0.0

	for _, vote := range session.Votes {
		weight := vote.Confidence
		totalWeight += weight
		if vote.IsValid {
			weightedAccept += weight
		}
	}

	// Create result
	result := &VoteResult{
		MarketDataID: session.ID,
		Accepted:     weightedAccept >= (totalWeight * vs.quorum),
		VoteCount:    len(session.Votes),
		Score:        weightedAccept / totalWeight,
		CompletedAt:  time.Now(),
	}

	session.Status = VoteStatusComplete
	session.Result = result
	session.EndTime = result.CompletedAt

	// Update metrics
	vs.metrics.mu.Lock()
	vs.metrics.SessionsComplete++
	vs.metrics.AverageLatency = (vs.metrics.AverageLatency*9 +
		session.EndTime.Sub(session.StartTime)) / 10
	vs.metrics.LastUpdate = time.Now()
	vs.metrics.mu.Unlock()

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
	}

	// Update metrics
	vs.metrics.mu.Lock()
	vs.metrics.SessionsFailed++
	vs.metrics.LastUpdate = time.Now()
	vs.metrics.mu.Unlock()

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
	vs.metrics.mu.RLock()
	defer vs.metrics.mu.RUnlock()

	vs.mu.RLock()
	activeSessions := len(vs.activeVotes)
	vs.mu.RUnlock()

	return VotingStats{
		ActiveSessions:   activeSessions,
		SessionsStarted:  vs.metrics.SessionsStarted,
		SessionsComplete: vs.metrics.SessionsComplete,
		SessionsFailed:   vs.metrics.SessionsFailed,
		AverageLatency:   vs.metrics.AverageLatency,
		LastUpdate:       vs.metrics.LastUpdate,
	}
}

// VotingStats represents voting system statistics
type VotingStats struct {
	ActiveSessions   int
	SessionsStarted  int64
	SessionsComplete int64
	SessionsFailed   int64
	AverageLatency   time.Duration
	LastUpdate       time.Time
}

// GetActiveVotingSessions returns all active voting sessions
func (vs *VotingSystem) GetActiveVotingSessions() []*VotingSessionInfo {
	vs.mu.RLock()
	defer vs.mu.RUnlock()

	sessions := make([]*VotingSessionInfo, 0, len(vs.activeVotes))
	for _, session := range vs.activeVotes {
		sessions = append(sessions, &VotingSessionInfo{
			ID:        session.ID,
			StartTime: session.StartTime,
			Status:    session.Status,
			VoteCount: len(session.Votes),
		})
	}

	return sessions
}

// VotingSessionInfo represents summary information about a voting session
type VotingSessionInfo struct {
	ID        string
	StartTime time.Time
	Status    VoteStatus
	VoteCount int
}

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

// VotingSystemConfig allows runtime configuration updates
type VotingSystemConfig struct {
	VotingTimeout time.Duration
	MinVoters     int
	Quorum        float64
}

// UpdateConfig updates the voting system configuration
func (vs *VotingSystem) UpdateConfig(config VotingSystemConfig) error {
	if config.VotingTimeout <= 0 {
		return fmt.Errorf("voting timeout must be positive")
	}
	if config.MinVoters <= 0 {
		return fmt.Errorf("minimum voters must be positive")
	}
	if config.Quorum <= 0 || config.Quorum > 1 {
		return fmt.Errorf("quorum must be between 0 and 1")
	}

	vs.mu.Lock()
	defer vs.mu.Unlock()

	vs.votingTimeout = config.VotingTimeout
	vs.minVoters = config.MinVoters
	vs.quorum = config.Quorum

	vs.logger.Info("Voting system configuration updated",
		zap.Duration("timeout", config.VotingTimeout),
		zap.Int("minVoters", config.MinVoters),
		zap.Float64("quorum", config.Quorum))

	return nil
}

// Additional helper functions

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

// GetSessionDetails retrieves detailed information about a voting session
func (vs *VotingSystem) GetSessionDetails(sessionID string) (*VotingSessionDetails, error) {
	vs.mu.RLock()
	defer vs.mu.RUnlock()

	session, exists := vs.activeVotes[sessionID]
	if !exists {
		return nil, fmt.Errorf("voting session not found: %s", sessionID)
	}

	details := &VotingSessionDetails{
		ID:           session.ID,
		StartTime:    session.StartTime,
		EndTime:      session.EndTime,
		Status:       session.Status,
		VoteCount:    len(session.Votes),
		MarketDataID: session.MarketData.ID,
		Votes:        make([]*VoteDetails, 0, len(session.Votes)),
	}

	for validatorID, vote := range session.Votes {
		details.Votes = append(details.Votes, &VoteDetails{
			ValidatorID: validatorID,
			IsValid:     vote.IsValid,
			Confidence:  vote.Confidence,
			Timestamp:   vote.Timestamp,
		})
	}

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
