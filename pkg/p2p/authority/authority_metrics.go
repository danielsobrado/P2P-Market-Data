package authority

import (
	"context"
	"sync"
	"time"
)

// AuthorityMetrics tracks authority node performance.
type AuthorityMetrics struct {
	ValidationsProcessed int64
	ValidationsAccepted  int64
	ValidationsRejected  int64
	AverageLatency       time.Duration
	LastUpdate           time.Time
	mu                   sync.RWMutex
}

// NewAuthorityMetrics creates a new AuthorityMetrics instance.
func NewAuthorityMetrics() *AuthorityMetrics {
	return &AuthorityMetrics{}
}

func (an *AuthorityNode) updateMetrics(result *ValidationResult, duration time.Duration) {
	an.metrics.mu.Lock()
	defer an.metrics.mu.Unlock()

	an.metrics.ValidationsProcessed++
	if result.IsValid {
		an.metrics.ValidationsAccepted++
	} else {
		an.metrics.ValidationsRejected++
	}

	alpha := 0.1
	an.metrics.AverageLatency = time.Duration(float64(an.metrics.AverageLatency)*(1-alpha) + float64(duration)*alpha)
	an.metrics.LastUpdate = time.Now()
}

func (an *AuthorityNode) collectMetrics(ctx context.Context) {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			an.metrics.mu.Lock()
			an.metrics.LastUpdate = time.Now()
			an.metrics.mu.Unlock()
		}
	}
}

// GetStats returns the current authority node statistics.
func (an *AuthorityNode) GetStats() AuthorityStats {
	an.metrics.mu.RLock()
	defer an.metrics.mu.RUnlock()

	an.mu.RLock()
	defer an.mu.RUnlock()

	return AuthorityStats{
		ValidationsProcessed: an.metrics.ValidationsProcessed,
		ValidationsAccepted:  an.metrics.ValidationsAccepted,
		ValidationsRejected:  an.metrics.ValidationsRejected,
		AverageLatency:       an.metrics.AverageLatency,
		VerifiedPeers:        len(an.verifiedPeers),
		LastUpdate:           an.metrics.LastUpdate,
	}
}
