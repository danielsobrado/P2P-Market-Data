package p2p

import (
	"context"

	pubsub "github.com/libp2p/go-libp2p-pubsub"
)

// Topic defines the methods required by a topic in the P2P network
type Topic interface {
	Publish(ctx context.Context, data []byte, opts ...pubsub.PubOpt) error
	// Add other methods if needed
}
