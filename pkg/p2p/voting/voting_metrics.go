package voting

import (
	"sync"
	"time"
)

// VotingMetrics tracks voting system performance
type VotingMetrics struct {
	sessionsStarted  int64
	sessionsComplete int64
	sessionsFailed   int64
	averageLatency   time.Duration
	lastUpdate       time.Time
	mu               sync.RWMutex
}

// NewVotingMetrics creates a new VotingMetrics instance
func NewVotingMetrics() *VotingMetrics {
	return &VotingMetrics{}
}

// IncrementSessionsStarted increments the sessionsStarted counter
func (vm *VotingMetrics) IncrementSessionsStarted() {
	vm.mu.Lock()
	defer vm.mu.Unlock()
	vm.sessionsStarted++
	vm.lastUpdate = time.Now()
}

// IncrementSessionsComplete increments the sessionsComplete counter
func (vm *VotingMetrics) IncrementSessionsComplete() {
	vm.mu.Lock()
	defer vm.mu.Unlock()
	vm.sessionsComplete++
	vm.lastUpdate = time.Now()
}

// IncrementSessionsFailed increments the sessionsFailed counter
func (vm *VotingMetrics) IncrementSessionsFailed() {
	vm.mu.Lock()
	defer vm.mu.Unlock()
	vm.sessionsFailed++
	vm.lastUpdate = time.Now()
}

// UpdateAverageLatency updates the average latency
func (vm *VotingMetrics) UpdateAverageLatency(latency time.Duration) {
	vm.mu.Lock()
	defer vm.mu.Unlock()
	alpha := 0.1
	vm.averageLatency = time.Duration(float64(vm.averageLatency)*(1-alpha) + float64(latency)*alpha)
	vm.lastUpdate = time.Now()
}

// GetStats returns the current voting statistics
func (vm *VotingMetrics) GetStats(activeSessions int) VotingStats {
	vm.mu.RLock()
	defer vm.mu.RUnlock()

	return VotingStats{
		ActiveSessions:   activeSessions,
		SessionsStarted:  vm.sessionsStarted,
		SessionsComplete: vm.sessionsComplete,
		SessionsFailed:   vm.sessionsFailed,
		AverageLatency:   vm.averageLatency,
		LastUpdate:       vm.lastUpdate,
	}
}
