package security

import (
	"context"
	"fmt"
	"math"
	"sort"
	"sync"
	"time"

	"github.com/libp2p/go-libp2p/core/peer"
	"go.uber.org/zap"

	"p2p_market_data/pkg/data"
)

const (
	// Reputation score bounds
	MinReputationScore = 0.0
	MaxReputationScore = 1.0
	InitialScore       = 0.5

	// Score adjustments
	ValidDataBonus     = 0.05
	InvalidDataPenalty = 0.1
	InactivityPenalty  = 0.01
)

// ReputationManager handles peer reputation tracking
type ReputationManager struct {
	scores        map[peer.ID]*PeerScore
	repo          data.Repository
	logger        *zap.Logger
	metrics       *ReputationMetrics
	updatePeriod  time.Duration
	minReputation float64
	mu            sync.RWMutex
}

// PeerScore tracks a peer's reputation
type PeerScore struct {
	ID           peer.ID
	Score        float64
	UpdatedAt    time.Time
	ValidData    uint64
	InvalidData  uint64
	TotalActions uint64
	LastAction   time.Time
}

// ReputationMetrics tracks reputation system metrics
type ReputationMetrics struct {
	HighRepPeers     int
	LowRepPeers      int
	AverageScore     float64
	UpdatesProcessed uint64
	LastUpdate       time.Time
	mu               sync.RWMutex
}

// NewReputationManager creates a new reputation manager
func NewReputationManager(repo data.Repository, logger *zap.Logger, minReputation float64) *ReputationManager {
	return &ReputationManager{
		scores:        make(map[peer.ID]*PeerScore),
		repo:          repo,
		logger:        logger,
		metrics:       &ReputationMetrics{},
		updatePeriod:  time.Hour,
		minReputation: minReputation,
	}
}

// Start begins reputation management operations
func (rm *ReputationManager) Start(ctx context.Context) error {
	// Load existing scores from repository
	if err := rm.loadScores(); err != nil {
		return fmt.Errorf("loading scores: %w", err)
	}

	// Start periodic updates
	go rm.periodicUpdate(ctx)

	return nil
}

// UpdatePeerReputation updates a peer's reputation score
func (rm *ReputationManager) UpdatePeerReputation(peerID peer.ID, action ReputationAction, value float64) error {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	score, exists := rm.scores[peerID]
	if !exists {
		score = &PeerScore{
			ID:        peerID,
			Score:     InitialScore,
			UpdatedAt: time.Now(),
		}
		rm.scores[peerID] = score
	}

	// Apply score adjustment
	newScore := score.Score
	switch action {
	case ValidData:
		newScore += ValidDataBonus * value
		score.ValidData++
	case InvalidData:
		newScore -= InvalidDataPenalty * value
		score.InvalidData++
	case Inactivity:
		newScore -= InactivityPenalty * value
	}

	// Ensure score stays within bounds
	score.Score = math.Max(MinReputationScore, math.Min(MaxReputationScore, newScore))
	score.UpdatedAt = time.Now()
	score.TotalActions++
	score.LastAction = time.Now()

	// Log significant changes
	if math.Abs(newScore-score.Score) > 0.1 {
		rm.logger.Info("Significant reputation change",
			zap.String("peerID", peerID.String()),
			zap.Float64("oldScore", score.Score),
			zap.Float64("newScore", newScore),
			zap.String("action", string(action)))
	}

	// Save updated score
	if err := rm.saveScore(score); err != nil {
		return fmt.Errorf("saving score: %w", err)
	}

	return nil
}

// GetPeerReputation retrieves a peer's current reputation score
func (rm *ReputationManager) GetPeerReputation(peerID peer.ID) (float64, error) {
	rm.mu.RLock()
	defer rm.mu.RUnlock()

	score, exists := rm.scores[peerID]
	if !exists {
		return InitialScore, nil
	}

	return score.Score, nil
}

// IsPeerTrusted checks if a peer's reputation is above the minimum threshold
func (rm *ReputationManager) IsPeerTrusted(peerID peer.ID) bool {
	score, _ := rm.GetPeerReputation(peerID)
	return score >= rm.minReputation
}

// Private methods

func (rm *ReputationManager) loadScores() error {
	// Load scores from repository
	// Implementation depends on your repository interface
	return nil
}

func (rm *ReputationManager) saveScore(score *PeerScore) error {
	// Save score to repository
	// Implementation depends on your repository interface
	return nil
}

func (rm *ReputationManager) periodicUpdate(ctx context.Context) {
	ticker := time.NewTicker(rm.updatePeriod)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			rm.updateAllScores()
			rm.updateMetrics()
		}
	}
}

func (rm *ReputationManager) updateAllScores() {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	now := time.Now()
	for _, score := range rm.scores {
		// Apply inactivity penalty
		if now.Sub(score.LastAction) > 24*time.Hour {
			score.Score = math.Max(MinReputationScore, score.Score-InactivityPenalty)
			score.UpdatedAt = now
		}
	}
}

func (rm *ReputationManager) updateMetrics() {
	rm.metrics.mu.Lock()
	defer rm.metrics.mu.Unlock()

	rm.mu.RLock()
	defer rm.mu.RUnlock()

	var totalScore float64
	highRep := 0
	lowRep := 0

	for _, score := range rm.scores {
		totalScore += score.Score
		if score.Score >= rm.minReputation {
			highRep++
		} else {
			lowRep++
		}
	}

	rm.metrics.HighRepPeers = highRep
	rm.metrics.LowRepPeers = lowRep
	if len(rm.scores) > 0 {
		rm.metrics.AverageScore = totalScore / float64(len(rm.scores))
	}
	rm.metrics.UpdatesProcessed++
	rm.metrics.LastUpdate = time.Now()
}

// ReputationAction represents types of actions that affect reputation
type ReputationAction string

const (
	ValidData   ReputationAction = "VALID_DATA"
	InvalidData ReputationAction = "INVALID_DATA"
	Inactivity  ReputationAction = "INACTIVITY"
)

// Additional reputation management functions

// GetPeerStats retrieves detailed statistics for a peer
func (rm *ReputationManager) GetPeerStats(peerID peer.ID) (*PeerStats, error) {
	rm.mu.RLock()
	defer rm.mu.RUnlock()

	score, exists := rm.scores[peerID]
	if !exists {
		return nil, fmt.Errorf("peer not found: %s", peerID)
	}

	return &PeerStats{
		ID:             peerID,
		CurrentScore:   score.Score,
		ValidData:      score.ValidData,
		InvalidData:    score.InvalidData,
		TotalActions:   score.TotalActions,
		LastAction:     score.LastAction,
		ScoreUpdatedAt: score.UpdatedAt,
	}, nil
}

// PeerStats represents detailed peer statistics
type PeerStats struct {
	ID             peer.ID
	CurrentScore   float64
	ValidData      uint64
	InvalidData    uint64
	TotalActions   uint64
	LastAction     time.Time
	ScoreUpdatedAt time.Time
}

// BatchUpdateReputations updates multiple peer reputations
func (rm *ReputationManager) BatchUpdateReputations(updates map[peer.ID]ReputationUpdate) error {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	for peerID, update := range updates {
		score, exists := rm.scores[peerID]
		if !exists {
			score = &PeerScore{
				ID:        peerID,
				Score:     InitialScore,
				UpdatedAt: time.Now(),
			}
			rm.scores[peerID] = score
		}

		newScore := calculateNewScore(score.Score, update)
		score.Score = newScore
		score.UpdatedAt = time.Now()
		score.TotalActions++
		score.LastAction = time.Now()

		switch update.Action {
		case ValidData:
			score.ValidData++
		case InvalidData:
			score.InvalidData++
		}
	}

	return rm.saveScores()
}

// ReputationUpdate represents a reputation score update
type ReputationUpdate struct {
	Action ReputationAction
	Value  float64
	Reason string
}

// GetReputationStats returns current reputation system statistics
func (rm *ReputationManager) GetReputationStats() ReputationStats {
	rm.metrics.mu.RLock()
	defer rm.metrics.mu.RUnlock()

	return ReputationStats{
		HighRepPeers:     rm.metrics.HighRepPeers,
		LowRepPeers:      rm.metrics.LowRepPeers,
		AverageScore:     rm.metrics.AverageScore,
		UpdatesProcessed: rm.metrics.UpdatesProcessed,
		LastUpdate:       rm.metrics.LastUpdate,
	}
}

// ReputationStats represents reputation system statistics
type ReputationStats struct {
	HighRepPeers     int
	LowRepPeers      int
	AverageScore     float64
	UpdatesProcessed uint64
	LastUpdate       time.Time
}

// Reset reputation score for a peer
func (rm *ReputationManager) ResetPeerReputation(peerID peer.ID) error {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	score, exists := rm.scores[peerID]
	if !exists {
		return fmt.Errorf("peer not found: %s", peerID)
	}

	score.Score = InitialScore
	score.ValidData = 0
	score.InvalidData = 0
	score.TotalActions = 0
	score.UpdatedAt = time.Now()
	score.LastAction = time.Now()

	return rm.saveScore(score)
}

// ExportReputationData exports all reputation data
func (rm *ReputationManager) ExportReputationData() ([]*PeerScore, error) {
	rm.mu.RLock()
	defer rm.mu.RUnlock()

	scores := make([]*PeerScore, 0, len(rm.scores))
	for _, score := range rm.scores {
		scores = append(scores, &PeerScore{
			ID:           score.ID,
			Score:        score.Score,
			UpdatedAt:    score.UpdatedAt,
			ValidData:    score.ValidData,
			InvalidData:  score.InvalidData,
			TotalActions: score.TotalActions,
			LastAction:   score.LastAction,
		})
	}

	return scores, nil
}

// ImportReputationData imports reputation data
func (rm *ReputationManager) ImportReputationData(scores []*PeerScore) error {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	for _, score := range scores {
		rm.scores[score.ID] = score
	}

	return rm.saveScores()
}

// Helper functions

func calculateNewScore(currentScore float64, update ReputationUpdate) float64 {
	var delta float64
	switch update.Action {
	case ValidData:
		delta = ValidDataBonus * update.Value
	case InvalidData:
		delta = -InvalidDataPenalty * update.Value
	case Inactivity:
		delta = -InactivityPenalty * update.Value
	}

	newScore := currentScore + delta
	return math.Max(MinReputationScore, math.Min(MaxReputationScore, newScore))
}

func (rm *ReputationManager) saveScores() error {
	// Save all scores to repository
	for _, score := range rm.scores {
		if err := rm.saveScore(score); err != nil {
			return fmt.Errorf("saving scores: %w", err)
		}
	}
	return nil
}

// GetTopPeers returns the top N peers by reputation
func (rm *ReputationManager) GetTopPeers(n int) []*PeerScore {
	rm.mu.RLock()
	defer rm.mu.RUnlock()

	// Convert map to slice for sorting
	scores := make([]*PeerScore, 0, len(rm.scores))
	for _, score := range rm.scores {
		scores = append(scores, score)
	}

	// Sort by score descending
	sort.Slice(scores, func(i, j int) bool {
		return scores[i].Score > scores[j].Score
	})

	// Return top N scores
	if n > len(scores) {
		n = len(scores)
	}
	return scores[:n]
}

// AdjustReputationThresholds updates reputation thresholds based on network health
func (rm *ReputationManager) AdjustReputationThresholds(ctx context.Context) {
	ticker := time.NewTicker(24 * time.Hour)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			rm.adjustThresholds()
		}
	}
}

func (rm *ReputationManager) adjustThresholds() {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	stats := rm.GetReputationStats()

	// Adjust minimum reputation based on network health
	if stats.AverageScore > 0.7 && stats.HighRepPeers > stats.LowRepPeers*2 {
		rm.minReputation = math.Min(rm.minReputation+0.05, 0.8)
	} else if stats.AverageScore < 0.3 || stats.LowRepPeers > stats.HighRepPeers {
		rm.minReputation = math.Max(rm.minReputation-0.05, 0.2)
	}

	rm.logger.Info("Adjusted reputation thresholds",
		zap.Float64("minReputation", rm.minReputation),
		zap.Float64("averageScore", stats.AverageScore))
}
