package voting

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestVotingMetrics_IncrementSessionsStarted(t *testing.T) {
	metrics := NewVotingMetrics()
	metrics.IncrementSessionsStarted()
	assert.Equal(t, int64(1), metrics.sessionsStarted)
}

func TestVotingMetrics_IncrementSessionsComplete(t *testing.T) {
	metrics := NewVotingMetrics()
	metrics.IncrementSessionsComplete()
	assert.Equal(t, int64(1), metrics.sessionsComplete)
}

func TestVotingMetrics_UpdateAverageLatency(t *testing.T) {
	metrics := NewVotingMetrics()
	metrics.UpdateAverageLatency(2 * time.Second)
	assert.Equal(t, 2*time.Second, metrics.averageLatency)
}

func TestVotingMetrics_GetStats(t *testing.T) {
	metrics := NewVotingMetrics()
	metrics.sessionsStarted = 10
	metrics.sessionsComplete = 7
	metrics.sessionsFailed = 3
	metrics.averageLatency = 2 * time.Second
	metrics.lastUpdate = time.Now()

	stats := metrics.GetStats(5)
	assert.Equal(t, 5, stats.ActiveSessions)
	assert.Equal(t, int64(10), stats.SessionsStarted)
	assert.Equal(t, int64(7), stats.SessionsComplete)
	assert.Equal(t, int64(3), stats.SessionsFailed)
	assert.Equal(t, 2*time.Second, stats.AverageLatency)
}
