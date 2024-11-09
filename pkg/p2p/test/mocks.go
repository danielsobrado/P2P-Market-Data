package p2p

import (
	"context"
	"sync"

	libp2pPubsub "github.com/libp2p/go-libp2p-pubsub"
)

// TopicStub is a stub for libp2p PubSub topics used in testing
type TopicStub struct {
	messages [][]byte
	mu       sync.Mutex
}

func NewTopicStub() *TopicStub {
	return &TopicStub{
		messages: make([][]byte, 0),
	}
}

func (t *TopicStub) Publish(ctx context.Context, msg []byte, opts ...libp2pPubsub.PubOpt) error {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.messages = append(t.messages, msg)
	return nil
}

func (t *TopicStub) GetMessages() [][]byte {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.messages
}

// Host is a mock of the Host struct with minimal implementation for testing
type Host struct {
	topics map[string]*TopicStub
}
