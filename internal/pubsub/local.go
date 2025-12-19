package pubsub

import (
	"context"
	"sync"
)

// LocalPubSub implements PubSub for single-instance deployments.
// Messages are only delivered within the same process.
// This has zero overhead for single-instance deployments.
type LocalPubSub struct {
	subscribers map[string][]chan Message
	mu          sync.RWMutex
}

// NewLocalPubSub creates a new local pub/sub.
func NewLocalPubSub() *LocalPubSub {
	return &LocalPubSub{
		subscribers: make(map[string][]chan Message),
	}
}

// Publish sends a message to all local subscribers of a channel.
func (l *LocalPubSub) Publish(ctx context.Context, channel string, payload []byte) error {
	l.mu.RLock()
	subs := l.subscribers[channel]
	l.mu.RUnlock()

	msg := Message{
		Channel: channel,
		Payload: payload,
	}

	for _, ch := range subs {
		select {
		case ch <- msg:
		default:
			// Channel full, skip
		}
	}

	return nil
}

// Subscribe returns a channel that receives messages published to the given channel.
func (l *LocalPubSub) Subscribe(ctx context.Context, channel string) (<-chan Message, error) {
	ch := make(chan Message, 100)

	l.mu.Lock()
	l.subscribers[channel] = append(l.subscribers[channel], ch)
	l.mu.Unlock()

	// Remove subscription when context is cancelled
	go func() {
		<-ctx.Done()
		l.unsubscribe(channel, ch)
	}()

	return ch, nil
}

// unsubscribe removes a subscriber channel
func (l *LocalPubSub) unsubscribe(channel string, ch chan Message) {
	l.mu.Lock()
	defer l.mu.Unlock()

	subs := l.subscribers[channel]
	for i, sub := range subs {
		if sub == ch {
			l.subscribers[channel] = append(subs[:i], subs[i+1:]...)
			close(ch)
			break
		}
	}
}

// Close releases all resources.
func (l *LocalPubSub) Close() error {
	l.mu.Lock()
	defer l.mu.Unlock()

	for _, subs := range l.subscribers {
		for _, ch := range subs {
			close(ch)
		}
	}
	l.subscribers = make(map[string][]chan Message)

	return nil
}
