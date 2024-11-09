package authority

import (
	"time"
)

// AuthorityStats represents authority node statistics.
type AuthorityStats struct {
	ValidationsProcessed int64
	ValidationsAccepted  int64
	ValidationsRejected  int64
	AverageLatency       time.Duration
	VerifiedPeers        int
	LastUpdate           time.Time
}

// ValidationResult represents the result of a validation.
type ValidationResult struct {
	IsValid bool
}
