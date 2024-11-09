package host

import (
	"sync"
	"time"
)

// Metrics tracks P2P network performance
type Metrics struct {
	ConnectedPeers    int
	TotalPeers        int
	MessagesProcessed int64
	ValidationLatency time.Duration
	NetworkLatency    time.Duration
	FailedValidations int64
	AvgLatency        time.Duration
	LastUpdated       time.Time
	mu                sync.RWMutex
}

// NewMetrics creates a new Metrics instance
func NewMetrics() *Metrics {
	return &Metrics{
		LastUpdated: time.Now(),
	}
}

// Collect gathers metrics from the host
func (m *Metrics) Collect(h *Host) {
	m.mu.Lock()
	defer m.mu.Unlock()

	conns := h.host.Network().Conns()
	m.ConnectedPeers = len(conns)
	m.TotalPeers = len(h.host.Peerstore().Peers())
	m.LastUpdated = time.Now()
	// Additional metrics can be collected here
}

// IncrementMessagesProcessed increments the messages processed count
func (m *Metrics) IncrementMessagesProcessed() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.MessagesProcessed++
	m.LastUpdated = time.Now()
}

// UpdateValidationLatency updates the average validation latency
func (m *Metrics) UpdateValidationLatency(duration time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()
	alpha := 0.1
	m.ValidationLatency = time.Duration(float64(m.ValidationLatency)*(1-alpha) + float64(duration)*alpha)
	m.LastUpdated = time.Now()
}

// IncrementFailedValidations increments the failed validations count
func (m *Metrics) IncrementFailedValidations() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.FailedValidations++
	m.LastUpdated = time.Now()
}

// UpdateNetworkLatency updates the average network latency
func (m *Metrics) UpdateNetworkLatency(duration time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()
	alpha := 0.1
	m.NetworkLatency = time.Duration(float64(m.NetworkLatency)*(1-alpha) + float64(duration)*alpha)
	m.LastUpdated = time.Now()
}

// GetMetrics returns a snapshot of the current metrics
func (m *Metrics) GetMetrics() Metrics {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return Metrics{
		ConnectedPeers:    m.ConnectedPeers,
		TotalPeers:        m.TotalPeers,
		MessagesProcessed: m.MessagesProcessed,
		ValidationLatency: m.ValidationLatency,
		NetworkLatency:    m.NetworkLatency,
		FailedValidations: m.FailedValidations,
		AvgLatency:       m.AvgLatency,
		LastUpdated:      m.LastUpdated,
	}
}
