// Package pubsub provides pub/sub interfaces for cross-instance communication.
// This enables realtime broadcasts to reach all connected clients regardless of
// which Fluxbase instance they're connected to.
package pubsub

import (
	"context"
)

// Message represents a pub/sub message
type Message struct {
	// Channel is the channel the message was published to
	Channel string `json:"channel"`

	// Payload is the message content
	Payload []byte `json:"payload"`
}

// PubSub is the interface for pub/sub backends.
// Implementations should handle concurrent access safely.
type PubSub interface {
	// Publish sends a message to all subscribers of a channel.
	// The channel is a logical grouping (e.g., "broadcast", "presence").
	Publish(ctx context.Context, channel string, payload []byte) error

	// Subscribe returns a channel that receives messages published to the given channel.
	// The returned channel is closed when the context is cancelled or Close is called.
	// Multiple calls to Subscribe with the same channel create independent subscriptions.
	Subscribe(ctx context.Context, channel string) (<-chan Message, error)

	// Close releases all resources and closes all subscriptions.
	Close() error
}

// BroadcastChannel is the channel used for cross-instance broadcasts
const BroadcastChannel = "fluxbase:broadcast"

// PresenceChannel is the channel used for presence synchronization
const PresenceChannel = "fluxbase:presence"
