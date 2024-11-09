package host

import (
	"sync"
	"time"
)

// Status represents the current state of the P2P host
type Status struct {
	IsReady      bool      `json:"is_ready"`
	IsValidating bool      `json:"is_validating"`
	LastError    error     `json:"last_error,omitempty"`
	StartTime    time.Time `json:"start_time"`
	UpdatedAt    time.Time `json:"updated_at"`
	Version      string    `json:"version"`
	mu           sync.RWMutex
}

// NewStatus creates a new Status instance
func NewStatus() *Status {
	now := time.Now()
	return &Status{
		StartTime: now,
		UpdatedAt: now,
		Version:   "1.0.0",
	}
}

// UpdateStatus updates the host's status
func (s *Status) UpdateStatus(isReady bool, isValidating bool, lastError error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.IsReady = isReady
	s.IsValidating = isValidating
	s.LastError = lastError
	s.UpdatedAt = time.Now()
}

// GetStatus returns a snapshot of the current status
func (s *Status) GetStatus() Status {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return Status{
		IsReady:      s.IsReady,
		IsValidating: s.IsValidating,
		LastError:    s.LastError,
		StartTime:    s.StartTime,
		UpdatedAt:    s.UpdatedAt,
		Version:      s.Version,
	}
}

// IsOnline checks if the host is ready and not in error state
func (s *Status) IsOnline() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.IsReady && s.LastError == nil
}

// GetUptime returns the duration since start
func (s *Status) GetUptime() time.Duration {
	return time.Since(s.StartTime)
}
