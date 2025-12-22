package pubsub

import (
	"context"
	"sync"
)

// localSubscriber represents a single subscriber with its channel and closed state.
type localSubscriber struct {
	ch     chan Message
	closed bool
	mu     sync.Mutex
}

// send attempts to send a message to the subscriber.
// Returns false if the subscriber is closed or the channel is full.
func (s *localSubscriber) send(msg Message) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return false
	}

	select {
	case s.ch <- msg:
		return true
	default:
		// Channel full, skip
		return false
	}
}

// close marks the subscriber as closed and closes the channel.
func (s *localSubscriber) close() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.closed {
		s.closed = true
		close(s.ch)
	}
}

// LocalPubSub implements PubSub for single-instance deployments.
// Messages are only delivered within the same process.
// This has zero overhead for single-instance deployments.
type LocalPubSub struct {
	subscribers map[string][]*localSubscriber
	mu          sync.RWMutex
}

// NewLocalPubSub creates a new local pub/sub.
func NewLocalPubSub() *LocalPubSub {
	return &LocalPubSub{
		subscribers: make(map[string][]*localSubscriber),
	}
}

// Publish sends a message to all local subscribers of a channel.
func (l *LocalPubSub) Publish(ctx context.Context, channel string, payload []byte) error {
	l.mu.RLock()
	// Copy the slice to avoid holding the lock during sends
	subs := make([]*localSubscriber, len(l.subscribers[channel]))
	copy(subs, l.subscribers[channel])
	l.mu.RUnlock()

	msg := Message{
		Channel: channel,
		Payload: payload,
	}

	for _, sub := range subs {
		sub.send(msg)
	}

	return nil
}

// Subscribe returns a channel that receives messages published to the given channel.
func (l *LocalPubSub) Subscribe(ctx context.Context, channel string) (<-chan Message, error) {
	sub := &localSubscriber{
		ch: make(chan Message, 100),
	}

	l.mu.Lock()
	l.subscribers[channel] = append(l.subscribers[channel], sub)
	l.mu.Unlock()

	// Remove subscription when context is cancelled
	go func() {
		<-ctx.Done()
		l.unsubscribe(channel, sub)
	}()

	return sub.ch, nil
}

// unsubscribe removes a subscriber
func (l *LocalPubSub) unsubscribe(channel string, sub *localSubscriber) {
	l.mu.Lock()
	subs := l.subscribers[channel]
	for i, s := range subs {
		if s == sub {
			l.subscribers[channel] = append(subs[:i], subs[i+1:]...)
			break
		}
	}
	l.mu.Unlock()

	// Close outside the lock to avoid potential deadlock
	sub.close()
}

// Close releases all resources.
func (l *LocalPubSub) Close() error {
	l.mu.Lock()
	allSubs := make([]*localSubscriber, 0)
	for _, subs := range l.subscribers {
		allSubs = append(allSubs, subs...)
	}
	l.subscribers = make(map[string][]*localSubscriber)
	l.mu.Unlock()

	// Close all subscribers outside the lock
	for _, sub := range allSubs {
		sub.close()
	}

	return nil
}
