package pubsub

import (
	"context"
	"sync"

	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog/log"
)

// RedisPubSub implements PubSub using Redis pub/sub.
// This is the recommended backend for high-scale deployments.
//
// Supported backends (all use the same go-redis library):
// - Dragonfly (recommended): 25x faster than Redis
// - Redis: The original Redis server
// - Valkey: Redis fork by Linux Foundation
// - KeyDB: Multi-threaded Redis fork
//
// Performance characteristics:
// - Designed for high throughput (100,000+ messages/second)
// - Messages are not persisted (pub/sub only)
// - No payload size limit (beyond available memory)
type RedisPubSub struct {
	client      *redis.Client
	subscribers map[string][]chan Message
	mu          sync.RWMutex
	ctx         context.Context
	cancel      context.CancelFunc
	wg          sync.WaitGroup
}

// NewRedisPubSub creates a new Redis-backed pub/sub.
// url should be in the format: redis://[password@]host:port[/db]
func NewRedisPubSub(url string) (*RedisPubSub, error) {
	opts, err := redis.ParseURL(url)
	if err != nil {
		return nil, err
	}

	client := redis.NewClient(opts)

	// Test connection
	ctx := context.Background()
	if err := client.Ping(ctx).Err(); err != nil {
		return nil, err
	}

	log.Info().Str("addr", opts.Addr).Msg("Connected to Redis-compatible backend for pub/sub")

	ctx, cancel := context.WithCancel(context.Background())
	return &RedisPubSub{
		client:      client,
		subscribers: make(map[string][]chan Message),
		ctx:         ctx,
		cancel:      cancel,
	}, nil
}

// Publish sends a message to all subscribers of a channel.
func (r *RedisPubSub) Publish(ctx context.Context, channel string, payload []byte) error {
	return r.client.Publish(ctx, channel, payload).Err()
}

// Subscribe returns a channel that receives messages published to the given channel.
func (r *RedisPubSub) Subscribe(ctx context.Context, channel string) (<-chan Message, error) {
	ch := make(chan Message, 100)

	// Subscribe to Redis channel
	pubsub := r.client.Subscribe(r.ctx, channel)

	// Wait for subscription to be ready
	_, err := pubsub.Receive(r.ctx)
	if err != nil {
		close(ch)
		return nil, err
	}

	// Store the subscription
	r.mu.Lock()
	r.subscribers[channel] = append(r.subscribers[channel], ch)
	r.mu.Unlock()

	// Process messages in a goroutine
	r.wg.Add(1)
	go func() {
		defer r.wg.Done()
		defer func() {
			r.unsubscribe(channel, ch)
			_ = pubsub.Close()
		}()

		msgCh := pubsub.Channel()
		for {
			select {
			case <-ctx.Done():
				return
			case <-r.ctx.Done():
				return
			case msg, ok := <-msgCh:
				if !ok {
					return
				}
				select {
				case ch <- Message{Channel: msg.Channel, Payload: []byte(msg.Payload)}:
				default:
					log.Warn().Str("channel", channel).Msg("Pub/sub subscriber channel full, dropping message")
				}
			}
		}
	}()

	return ch, nil
}

// unsubscribe removes a subscriber channel
func (r *RedisPubSub) unsubscribe(channel string, ch chan Message) {
	r.mu.Lock()
	defer r.mu.Unlock()

	subs := r.subscribers[channel]
	for i, sub := range subs {
		if sub == ch {
			r.subscribers[channel] = append(subs[:i], subs[i+1:]...)
			close(ch)
			break
		}
	}
}

// Close releases all resources and closes all subscriptions.
func (r *RedisPubSub) Close() error {
	r.cancel()
	r.wg.Wait()

	r.mu.Lock()
	for _, subs := range r.subscribers {
		for _, ch := range subs {
			close(ch)
		}
	}
	r.subscribers = make(map[string][]chan Message)
	r.mu.Unlock()

	err := r.client.Close()
	log.Info().Msg("Redis pub/sub closed")
	return err
}
