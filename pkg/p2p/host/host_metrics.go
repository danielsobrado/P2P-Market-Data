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
	RequestsReceived  int64
	RequestsRejected  int64
	AuthFailures      int64
	TransfersStarted  int64
	TransfersComplete int64
	TransfersFailed   int64
	ChunksSent        int64
	ChunksReceived    int64
	RowsSent          int64
	RowsReceived      int64
	BytesSent         int64
	BytesReceived     int64
	LastError         string
	LastRequestAt     time.Time
	LastTransferAt    time.Time
	LastUpdated       time.Time
	mu                sync.RWMutex
}

// MetricsSnapshot is a JSON-friendly snapshot of host operational counters.
type MetricsSnapshot struct {
	ConnectedPeers      int    `json:"connectedPeers"`
	TotalPeers          int    `json:"totalPeers"`
	MessagesProcessed   int64  `json:"messagesProcessed"`
	ValidationLatencyMs int64  `json:"validationLatencyMs"`
	NetworkLatencyMs    int64  `json:"networkLatencyMs"`
	FailedValidations   int64  `json:"failedValidations"`
	RequestsReceived    int64  `json:"requestsReceived"`
	RequestsRejected    int64  `json:"requestsRejected"`
	AuthFailures        int64  `json:"authFailures"`
	TransfersStarted    int64  `json:"transfersStarted"`
	TransfersComplete   int64  `json:"transfersComplete"`
	TransfersFailed     int64  `json:"transfersFailed"`
	ChunksSent          int64  `json:"chunksSent"`
	ChunksReceived      int64  `json:"chunksReceived"`
	RowsSent            int64  `json:"rowsSent"`
	RowsReceived        int64  `json:"rowsReceived"`
	BytesSent           int64  `json:"bytesSent"`
	BytesReceived       int64  `json:"bytesReceived"`
	LastError           string `json:"lastError,omitempty"`
	LastRequestAt       string `json:"lastRequestAt,omitempty"`
	LastTransferAt      string `json:"lastTransferAt,omitempty"`
	LastUpdated         string `json:"lastUpdated"`
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

func (m *Metrics) RecordRequestReceived() {
	m.mu.Lock()
	defer m.mu.Unlock()
	now := time.Now()
	m.RequestsReceived++
	m.LastRequestAt = now
	m.LastUpdated = now
}

func (m *Metrics) RecordRequestRejected(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	now := time.Now()
	m.RequestsRejected++
	if err != nil {
		m.LastError = err.Error()
	}
	m.LastRequestAt = now
	m.LastUpdated = now
}

func (m *Metrics) RecordAuthFailure(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	now := time.Now()
	m.AuthFailures++
	m.RequestsRejected++
	if err != nil {
		m.LastError = err.Error()
	}
	m.LastRequestAt = now
	m.LastUpdated = now
}

func (m *Metrics) RecordTransferStarted() {
	m.mu.Lock()
	defer m.mu.Unlock()
	now := time.Now()
	m.TransfersStarted++
	m.LastTransferAt = now
	m.LastUpdated = now
}

func (m *Metrics) RecordTransferCompleted() {
	m.mu.Lock()
	defer m.mu.Unlock()
	now := time.Now()
	m.TransfersComplete++
	m.LastTransferAt = now
	m.LastUpdated = now
}

func (m *Metrics) RecordTransferFailed(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	now := time.Now()
	m.TransfersFailed++
	if err != nil {
		m.LastError = err.Error()
	}
	m.LastTransferAt = now
	m.LastUpdated = now
}

func (m *Metrics) RecordResponseSent(resp dataResponse, bytes int64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	now := time.Now()
	m.ChunksSent++
	m.RowsSent += int64(responseRowCount(resp))
	m.BytesSent += bytes
	m.LastRequestAt = now
	m.LastUpdated = now
}

func (m *Metrics) RecordChunkReceived(resp dataResponse, bytes int64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	now := time.Now()
	m.ChunksReceived++
	m.RowsReceived += int64(responseRowCount(resp))
	m.BytesReceived += bytes
	m.LastTransferAt = now
	m.LastUpdated = now
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
		AvgLatency:        m.AvgLatency,
		RequestsReceived:  m.RequestsReceived,
		RequestsRejected:  m.RequestsRejected,
		AuthFailures:      m.AuthFailures,
		TransfersStarted:  m.TransfersStarted,
		TransfersComplete: m.TransfersComplete,
		TransfersFailed:   m.TransfersFailed,
		ChunksSent:        m.ChunksSent,
		ChunksReceived:    m.ChunksReceived,
		RowsSent:          m.RowsSent,
		RowsReceived:      m.RowsReceived,
		BytesSent:         m.BytesSent,
		BytesReceived:     m.BytesReceived,
		LastError:         m.LastError,
		LastRequestAt:     m.LastRequestAt,
		LastTransferAt:    m.LastTransferAt,
		LastUpdated:       m.LastUpdated,
	}
}

func (m *Metrics) Snapshot() MetricsSnapshot {
	current := m.GetMetrics()
	snapshot := MetricsSnapshot{
		ConnectedPeers:      current.ConnectedPeers,
		TotalPeers:          current.TotalPeers,
		MessagesProcessed:   current.MessagesProcessed,
		ValidationLatencyMs: current.ValidationLatency.Milliseconds(),
		NetworkLatencyMs:    current.NetworkLatency.Milliseconds(),
		FailedValidations:   current.FailedValidations,
		RequestsReceived:    current.RequestsReceived,
		RequestsRejected:    current.RequestsRejected,
		AuthFailures:        current.AuthFailures,
		TransfersStarted:    current.TransfersStarted,
		TransfersComplete:   current.TransfersComplete,
		TransfersFailed:     current.TransfersFailed,
		ChunksSent:          current.ChunksSent,
		ChunksReceived:      current.ChunksReceived,
		RowsSent:            current.RowsSent,
		RowsReceived:        current.RowsReceived,
		BytesSent:           current.BytesSent,
		BytesReceived:       current.BytesReceived,
		LastError:           current.LastError,
	}
	if !current.LastRequestAt.IsZero() {
		snapshot.LastRequestAt = current.LastRequestAt.UTC().Format(time.RFC3339)
	}
	if !current.LastTransferAt.IsZero() {
		snapshot.LastTransferAt = current.LastTransferAt.UTC().Format(time.RFC3339)
	}
	if !current.LastUpdated.IsZero() {
		snapshot.LastUpdated = current.LastUpdated.UTC().Format(time.RFC3339)
	}
	return snapshot
}
