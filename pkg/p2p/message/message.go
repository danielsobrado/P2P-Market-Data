package message

import (
	"encoding/json"
	"time"

	"p2p_market_data/pkg/data"
	"p2p_market_data/pkg/p2p/utils"

	"github.com/libp2p/go-libp2p-core/peer"
)

// MessageType represents the type of message
type MessageType string

const (
	MarketDataMessage         MessageType = "MarketData"
	ValidationRequestMessage  MessageType = "ValidationRequest"
	ValidationResponseMessage MessageType = "ValidationResponse"
)

// Message represents a P2P network message
type Message struct {
	Type      MessageType `json:"type"`
	Version   string      `json:"version"`
	ID        string      `json:"id"`
	Timestamp time.Time   `json:"timestamp"`
	SenderID  peer.ID     `json:"sender_id"`
	Data      interface{} `json:"data"`
	Signature []byte      `json:"signature,omitempty"`
}

// NewMessage creates a new message
func NewMessage(msgType MessageType, data interface{}) *Message {
	return &Message{
		Type:      msgType,
		Version:   "1.0.0",
		ID:        utils.GenerateMessageID(),
		Timestamp: time.Now(),
		Data:      data,
	}
}

// Marshal serializes the message
func (m *Message) Marshal() ([]byte, error) {
	return json.Marshal(m)
}

// MarshalWithoutSignature serializes the message without the signature field
func (m *Message) MarshalWithoutSignature() ([]byte, error) {
	temp := &Message{
		Type:     m.Type,
		Data:     m.Data,
		SenderID: m.SenderID,
	}
	return json.Marshal(temp)
}

// Unmarshal deserializes the message
func (m *Message) Unmarshal(data []byte) error {
	return json.Unmarshal(data, m)
}

// ValidationRequest represents a validation request
type ValidationRequest struct {
	MarketData *data.MarketData `json:"market_data"`
	ResponseCh chan *ValidationResult
	Timestamp  time.Time `json:"timestamp"`
}

// ValidationResult represents the result of a validation
type ValidationResult struct {
	MarketDataID string    `json:"market_data_id"`
	IsValid      bool      `json:"is_valid"`
	Score        float64   `json:"score"`
	ValidatedBy  []peer.ID `json:"validated_by"`
	CompletedAt  time.Time `json:"completed_at"`
	ErrorMsg     string    `json:"error_msg,omitempty"`
}

// ErrorResponse represents an error response
type ErrorResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Details string `json:"details,omitempty"`
}
