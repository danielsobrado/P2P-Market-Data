// pkg/peer/discovery/validator.go
package discovery

import (
	"encoding/json"
	"fmt"
	"time"

	"go.uber.org/zap"
)

// Basic validation constants
const (
	maxRecordSize = 1024 * 1024 // 1MB
	maxRecordAge  = 24 * time.Hour
)

// Record represents the basic DHT record structure
type Record struct {
	// Only essential fields
	PeerID    string    `json:"peer_id"`           // Who created the record
	Data      []byte    `json:"data"`              // Actual content
	Timestamp time.Time `json:"timestamp"`         // When it was created
	Version   int       `json:"version,omitempty"` // Optional version number
}

// Validator provides simple DHT record validation
type Validator struct {
	logger *zap.Logger
}

// NewValidator creates a new validator instance
func NewValidator(logger *zap.Logger) *Validator {
	return &Validator{
		logger: logger,
	}
}

// Validate checks if a DHT record is valid
func (v *Validator) Validate(key string, value []byte) error {
	// Size check
	if len(value) > maxRecordSize {
		return fmt.Errorf("record too large: %d > %d bytes", len(value), maxRecordSize)
	}

	// Parse record
	var record Record
	if err := json.Unmarshal(value, &record); err != nil {
		return fmt.Errorf("invalid record format: %w", err)
	}

	// Required fields
	if record.PeerID == "" {
		return fmt.Errorf("missing peer ID")
	}

	// Time validation
	now := time.Now()
	recordAge := now.Sub(record.Timestamp)

	if recordAge > maxRecordAge {
		return fmt.Errorf("record too old: %v", recordAge)
	}

	if record.Timestamp.After(now.Add(time.Minute)) {
		return fmt.Errorf("record timestamp in future")
	}

	return nil
}

// Select chooses the best record when multiple values exist
func (v *Validator) Select(key string, values [][]byte) (int, error) {
	if len(values) == 0 {
		return 0, fmt.Errorf("no values to select from")
	}

	bestIndex := 0
	var bestTime time.Time
	var bestVersion int

	for i, value := range values {
		// Skip invalid records
		if err := v.Validate(key, value); err != nil {
			continue
		}

		var record Record
		if err := json.Unmarshal(value, &record); err != nil {
			continue
		}

		// Prefer higher versions first
		if record.Version > bestVersion {
			bestIndex = i
			bestVersion = record.Version
			bestTime = record.Timestamp
			continue
		}

		// For same version, prefer most recent
		if record.Version == bestVersion && record.Timestamp.After(bestTime) {
			bestIndex = i
			bestTime = record.Timestamp
		}
	}

	return bestIndex, nil
}

// CreateRecord creates a new valid record
func CreateRecord(peerID string, data []byte, version int) (*Record, error) {
	record := &Record{
		PeerID:    peerID,
		Data:      data,
		Timestamp: time.Now(),
		Version:   version,
	}

	// Validate before returning
	recordBytes, err := json.Marshal(record)
	if err != nil {
		return nil, fmt.Errorf("marshaling record: %w", err)
	}

	validator := NewValidator(nil)
	if err := validator.Validate("", recordBytes); err != nil {
		return nil, fmt.Errorf("invalid record: %w", err)
	}

	return record, nil
}
