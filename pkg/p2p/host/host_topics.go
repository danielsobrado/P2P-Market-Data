package host

import (
	"context"
	"p2p_market_data/pkg/data"
	"p2p_market_data/pkg/p2p/message"

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
func (h *Host) processTopicMessages(ctx context.Context, topicName string, sub *pubsub.Subscription) {
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
		h.processTopicMessage(msg)
	}
}

// processTopicMessage deserializes and handles an incoming message from topics
func (h *Host) processTopicMessage(msg *pubsub.Message) {
	// Deserialize the message
	receivedMsg := &message.Message{}
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
	case message.MarketDataMessage:
		h.processMarketDataMessage(receivedMsg)
		h.metrics.IncrementMessagesProcessed()
	case message.ValidationRequestMessage:
		h.handleValidationRequestMessage(receivedMsg)
	case message.ValidationResponseMessage:
		h.handleValidationResponseMessage(receivedMsg)
	default:
		h.logger.Warn("Unknown message type", zap.String("type", string(receivedMsg.Type)))
	}
}

// processMarketDataMessage handles incoming market data messages
func (h *Host) processMarketDataMessage(msg *message.Message) {
	marketData, ok := msg.Data.(*data.MarketData)
	if !ok {
		h.logger.Warn("Invalid market data message payload")
		return
	}

	h.logger.Info("Received market data",
		zap.String("symbol", marketData.Symbol),
		zap.Float64("price", marketData.Price))

	// Optionally store or process the market data
}


