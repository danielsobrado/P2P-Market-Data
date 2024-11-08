package p2p

import (
	"encoding/json"
	"fmt"
	"time"

	"p2p_market_data/pkg/data"

	"github.com/libp2p/go-libp2p-core/peer"
)

// MessageType defines the type of P2P message
type MessageType string

const (
	MarketDataMessage MessageType = "MARKET_DATA"
	ValidationMessage MessageType = "VALIDATION"
	VoteMessage       MessageType = "VOTE"
	PeerDiscovery     MessageType = "PEER_DISCOVERY"
	AuthorityMessage  MessageType = "AUTHORITY"
	StatusMessage     MessageType = "STATUS"
	ErrorMessage      MessageType = "ERROR"
)

// Message represents a generic P2P message
type Message struct {
	Type      MessageType       `json:"type"`
	Version   string            `json:"version"`
	ID        string            `json:"id"`
	Timestamp time.Time         `json:"timestamp"`
	SenderID  peer.ID           `json:"sender_id"`
	Data      interface{}       `json:"data"`
	Signature []byte            `json:"signature"`
	Metadata  map[string]string `json:"metadata,omitempty"`
}

// ValidationRequest represents a data validation request
type ValidationRequest struct {
	MarketData  *data.MarketData
	ResponseCh  chan *ValidationResult
	Timestamp   time.Time
	RequestedBy peer.ID
	Timeout     time.Duration
}

// ValidationResult represents the outcome of data validation
type ValidationResult struct {
	MarketDataID string
	IsValid      bool
	Score        float64
	Votes        []*data.Vote
	ValidatedBy  []peer.ID
	ErrorMsg     string
	CompletedAt  time.Time
}

// VoteRequest represents a voting request
type VoteRequest struct {
	MarketDataID string
	ValidatorID  peer.ID
	Deadline     time.Time
	MinVotes     int
	ResponseCh   chan *VoteResult
}

// VoteResult represents the outcome of a voting process
type VoteResult struct {
	MarketDataID string
	Accepted     bool
	VoteCount    int
	Score        float64
	CompletedAt  time.Time
}

// PeerInfo represents information about a peer
type PeerInfo struct {
	ID          peer.ID
	Addresses   []string
	Reputation  float64
	LastSeen    time.Time
	IsAuthority bool
	Roles       []string
	Version     string
}

// ErrorResponse represents an error message
type ErrorResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Details string `json:"details,omitempty"`
}

// NewMessage creates a new message with the given type and data
func NewMessage(msgType MessageType, data interface{}) *Message {
	return &Message{
		Type:      msgType,
		Version:   "1.0.0",
		ID:        generateMessageID(),
		Timestamp: time.Now().UTC(),
		Data:      data,
		Metadata:  make(map[string]string),
	}
}

// Marshal serializes the message to bytes
func (m *Message) Marshal() ([]byte, error) {
	// Create a copy with properly typed data based on message type
	msg := struct {
		Type      MessageType       `json:"type"`
		Version   string            `json:"version"`
		ID        string            `json:"id"`
		Timestamp time.Time         `json:"timestamp"`
		SenderID  peer.ID           `json:"sender_id"`
		Data      json.RawMessage   `json:"data"`
		Signature []byte            `json:"signature"`
		Metadata  map[string]string `json:"metadata,omitempty"`
	}{
		Type:      m.Type,
		Version:   m.Version,
		ID:        m.ID,
		Timestamp: m.Timestamp,
		SenderID:  m.SenderID,
		Signature: m.Signature,
		Metadata:  m.Metadata,
	}

	// Marshal the data field separately
	var err error
	msg.Data, err = json.Marshal(m.Data)
	if err != nil {
		return nil, fmt.Errorf("marshaling message data: %w", err)
	}

	return json.Marshal(msg)
}

// Unmarshal deserializes the message from bytes
func (m *Message) Unmarshal(data []byte) error {
	// First unmarshal the message structure
	var msg struct {
		Type      MessageType       `json:"type"`
		Version   string            `json:"version"`
		ID        string            `json:"id"`
		Timestamp time.Time         `json:"timestamp"`
		SenderID  peer.ID           `json:"sender_id"`
		Data      json.RawMessage   `json:"data"`
		Signature []byte            `json:"signature"`
		Metadata  map[string]string `json:"metadata,omitempty"`
	}

	if err := json.Unmarshal(data, &msg); err != nil {
		return fmt.Errorf("unmarshaling message structure: %w", err)
	}

	// Copy basic fields
	m.Type = msg.Type
	m.Version = msg.Version
	m.ID = msg.ID
	m.Timestamp = msg.Timestamp
	m.SenderID = msg.SenderID
	m.Signature = msg.Signature
	m.Metadata = msg.Metadata

	// Unmarshal the data field based on message type
	switch m.Type {
	case MarketDataMessage:
		var marketData data.MarketDataType // Replace with the correct type from the data package
		if err := json.Unmarshal(msg.Data, &marketData); err != nil {
			return fmt.Errorf("unmarshaling market data: %w", err)
		}
		m.Data = &marketData

	case ValidationMessage:
		var validationResult ValidationResult
		if err := json.Unmarshal(msg.Data, &validationResult); err != nil {
			return fmt.Errorf("unmarshaling validation result: %w", err)
		}
		m.Data = &validationResult

	case VoteMessage:
		var vote data.Vote
		if err := json.Unmarshal(msg.Data, &vote); err != nil {
			return fmt.Errorf("unmarshaling vote: %w", err)
		}
		m.Data = &vote

	case PeerDiscovery:
		var peerInfo PeerInfo
		if err := json.Unmarshal(msg.Data, &peerInfo); err != nil {
			return fmt.Errorf("unmarshaling peer info: %w", err)
		}
		m.Data = &peerInfo

	case ErrorMessage:
		var errorResponse ErrorResponse
		if err := json.Unmarshal(msg.Data, &errorResponse); err != nil {
			return fmt.Errorf("unmarshaling error response: %w", err)
		}
		m.Data = &errorResponse

	default:
		return fmt.Errorf("unknown message type: %s", m.Type)
	}

	return nil
}

// Validate checks if the message is valid
func (m *Message) Validate() error {
	if m.Type == "" {
		return fmt.Errorf("message type cannot be empty")
	}
	if m.Version == "" {
		return fmt.Errorf("message version cannot be empty")
	}
	if m.ID == "" {
		return fmt.Errorf("message ID cannot be empty")
	}
	if m.Timestamp.IsZero() {
		return fmt.Errorf("message timestamp cannot be zero")
	}
	if m.SenderID == "" {
		return fmt.Errorf("sender ID cannot be empty")
	}
	if m.Data == nil {
		return fmt.Errorf("message data cannot be nil")
	}
	if m.Signature == nil {
		return fmt.Errorf("message signature cannot be nil")
	}

	// Validate data based on message type
	switch m.Type {
	case MarketDataMessage:
		if marketData, ok := m.Data.(*data.MarketData); ok {
			return marketData.Validate()
		}
		return fmt.Errorf("invalid market data type")

	case ValidationMessage:
		if result, ok := m.Data.(*ValidationResult); ok {
			if result.MarketDataID == "" {
				return fmt.Errorf("validation result must have market data ID")
			}
		}
		return fmt.Errorf("invalid validation result type")

	case VoteMessage:
		if vote, ok := m.Data.(*data.Vote); ok {
			return vote.Validate()
		}
		return fmt.Errorf("invalid vote type")
	}

	return nil
}

// Helper functions

func generateMessageID() string {
	return fmt.Sprintf("%d-%s", time.Now().UnixNano(), randomString(8))
}

func randomString(n int) string {
	const letters = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, n)
	for i := range b {
		b[i] = letters[time.Now().UnixNano()%int64(len(letters))]
	}
	return string(b)
}
