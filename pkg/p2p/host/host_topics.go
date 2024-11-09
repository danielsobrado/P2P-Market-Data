package host

import (
	"context"
	"p2p_market_data/pkg/data"

	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"go.uber.org/zap"
)

type MessageType string

const (
	MarketDataMessage         MessageType = "MARKET_DATA"
	ValidationRequestMessage  MessageType = "VALIDATION_REQUEST"
	ValidationResponseMessage MessageType = "VALIDATION_RESPONSE"
)

// handleTopicMessages processes incoming messages from a topic
func (h *Host) handleTopicMessages(ctx context.Context, topicName string, sub *pubsub.Subscription) {
	for {
		msg, err := sub.Next(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			h.logger.Warn("Error reading from subscription", zap.Error(err))
			continue
		}

		// Process the message
		h.processIncomingMessage(msg)
	}
}

// processIncomingMessage deserializes and handles an incoming message
func (h *Host) processIncomingMessage(msg *pubsub.Message) {
	// Deserialize the message
	receivedMsg := &Message{}
	if err := receivedMsg.Unmarshal(msg.Data); err != nil {
		h.logger.Warn("Failed to unmarshal message", zap.Error(err))
		return
	}

	// Verify message signature
	if err := h.verifyMessage(receivedMsg); err != nil {
		h.logger.Warn("Failed to verify message signature", zap.Error(err))
		return
	}

	// Handle the message based on its type
	switch receivedMsg.Type {
	case MarketDataMessage:
		h.handleMarketDataMessage(receivedMsg)
		h.metrics.IncrementMessagesProcessed()
	case ValidationRequestMessage:
		h.handleValidationRequestMessage(receivedMsg)
	case ValidationResponseMessage:
		h.handleValidationResponseMessage(receivedMsg)
	default:
		h.logger.Warn("Unknown message type", zap.String("type", string(receivedMsg.Type)))
	}
}

// handleMarketDataMessage handles incoming market data messages
func (h *Host) handleMarketDataMessage(msg *Message) {
	marketData, ok := msg.Payload.(*data.MarketData)
	if !ok {
		h.logger.Warn("Invalid market data message payload")
		return
	}

	h.logger.Info("Received market data",
		zap.String("symbol", marketData.Symbol),
		zap.Float64("price", marketData.Price))

	// Optionally store or process the market data
}

// handleValidationRequestMessage handles incoming validation requests
func (h *Host) handleValidationRequestMessage(msg *Message) {
	// Implement handling of validation requests
}

// handleValidationResponseMessage handles incoming validation responses
func (h *Host) handleValidationResponseMessage(msg *Message) {
	// Implement handling of validation responses
}
