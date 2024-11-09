package authority

import (
	"time"

	libp2pPeer "github.com/libp2p/go-libp2p/core/peer"
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

// ValidationResult represents the outcome of a market data validation
type ValidationResult struct {
	MarketDataID string          // ID of the validated market data
	IsValid      bool            // Whether the data passed validation
	Score        float64         // Validation score/confidence
	ValidatedBy  []libp2pPeer.ID // List of peers that validated this data
	ErrorMsg     string          // Error message if validation failed
	CompletedAt  time.Time       // When validation completed
}
