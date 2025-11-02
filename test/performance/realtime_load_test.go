package performance

import (
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// BenchmarkWebSocketConnections benchmarks WebSocket connection handling
func BenchmarkWebSocketConnections(b *testing.B) {
	server := newMockRealtimeServer()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		conn := server.connect(fmt.Sprintf("conn%d", i))
		server.disconnect(conn.id)
	}
}

// BenchmarkWebSocketMessageDelivery benchmarks message delivery
func BenchmarkWebSocketMessageDelivery(b *testing.B) {
	server := newMockRealtimeServer()

	// Setup connections
	for i := 0; i < 100; i++ {
		server.connect(fmt.Sprintf("conn%d", i))
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		server.broadcast(map[string]interface{}{
			"event": "UPDATE",
			"data":  fmt.Sprintf("message%d", i),
		})
	}
}

// TestWebSocketScalability100Connections tests 100 concurrent WebSocket connections
func TestWebSocketScalability100Connections(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	server := newMockRealtimeServer()
	numConnections := 100

	var wg sync.WaitGroup
	start := time.Now()

	// Establish connections
	for i := 0; i < numConnections; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			conn := server.connect(fmt.Sprintf("conn%d", id))

			// Subscribe to events
			server.subscribe(conn.id, "users", nil)

			// Keep connection alive
			time.Sleep(1 * time.Second)

			server.disconnect(conn.id)
		}(i)
	}

	wg.Wait()
	duration := time.Since(start)

	t.Logf("Handled %d connections in %v", numConnections, duration)
	t.Logf("Average connection time: %v", duration/time.Duration(numConnections))

	// Assert all connections handled within 5 seconds
	if duration > 5*time.Second {
		t.Errorf("Connection handling too slow: %v", duration)
	}
}

// TestWebSocketScalability500Connections tests 500 concurrent WebSocket connections
func TestWebSocketScalability500Connections(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	server := newMockRealtimeServer()
	numConnections := 500

	var wg sync.WaitGroup
	var connected atomic.Int64

	start := time.Now()

	// Establish connections
	for i := 0; i < numConnections; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			conn := server.connect(fmt.Sprintf("conn%d", id))
			connected.Add(1)

			time.Sleep(500 * time.Millisecond)

			server.disconnect(conn.id)
		}(i)
	}

	wg.Wait()
	duration := time.Since(start)

	t.Logf("Handled %d connections concurrently", connected.Load())
	t.Logf("Total time: %v", duration)

	if duration > 10*time.Second {
		t.Errorf("Connection handling too slow for 500 connections: %v", duration)
	}
}

// TestMessageBroadcastThroughput tests message broadcast throughput
func TestMessageBroadcastThroughput(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	server := newMockRealtimeServer()

	// Setup 100 connections
	for i := 0; i < 100; i++ {
		conn := server.connect(fmt.Sprintf("conn%d", i))
		server.subscribe(conn.id, "users", nil)
	}

	numMessages := 1000
	start := time.Now()

	// Broadcast messages
	for i := 0; i < numMessages; i++ {
		server.broadcast(map[string]interface{}{
			"event": "UPDATE",
			"id":    i,
		})
	}

	duration := time.Since(start)
	mps := float64(numMessages) / duration.Seconds()

	t.Logf("Broadcast %d messages in %v", numMessages, duration)
	t.Logf("Throughput: %.2f messages/second", mps)
	t.Logf("Average delivery time per message: %v", duration/time.Duration(numMessages))

	// Assert at least 100 messages per second
	if mps < 100 {
		t.Errorf("Broadcast throughput too low: %.2f MPS", mps)
	}
}

// TestSubscriptionFiltering tests subscription filtering performance
func TestSubscriptionFiltering(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	server := newMockRealtimeServer()

	// Setup 100 connections with different filters
	for i := 0; i < 100; i++ {
		conn := server.connect(fmt.Sprintf("conn%d", i))
		filters := map[string]interface{}{
			"user_id": fmt.Sprintf("user%d", i%10),
		}
		server.subscribe(conn.id, "users", filters)
	}

	numEvents := 1000
	start := time.Now()

	// Send filtered events
	for i := 0; i < numEvents; i++ {
		event := map[string]interface{}{
			"event":   "UPDATE",
			"user_id": fmt.Sprintf("user%d", i%10),
			"data":    fmt.Sprintf("data%d", i),
		}
		server.broadcastFiltered(event)
	}

	duration := time.Since(start)

	t.Logf("Filtered and delivered %d events in %v", numEvents, duration)
	t.Logf("Average filtering time: %v", duration/time.Duration(numEvents))

	if duration > 5*time.Second {
		t.Errorf("Filtering too slow: %v", duration)
	}
}

// mockRealtimeServer simulates a realtime WebSocket server
type mockRealtimeServer struct {
	connections  map[string]*mockConnection
	subscription map[string][]*mockSubscription
	mu           sync.RWMutex
}

type mockConnection struct {
	id string
}

type mockSubscription struct {
	connectionID string
	table        string
	filters      map[string]interface{}
}

func newMockRealtimeServer() *mockRealtimeServer {
	return &mockRealtimeServer{
		connections:  make(map[string]*mockConnection),
		subscription: make(map[string][]*mockSubscription),
	}
}

func (s *mockRealtimeServer) connect(id string) *mockConnection {
	s.mu.Lock()
	defer s.mu.Unlock()

	conn := &mockConnection{id: id}
	s.connections[id] = conn

	// Simulate connection overhead
	time.Sleep(100 * time.Microsecond)

	return conn
}

func (s *mockRealtimeServer) disconnect(id string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.connections, id)
	delete(s.subscription, id)

	// Simulate disconnection overhead
	time.Sleep(50 * time.Microsecond)
}

func (s *mockRealtimeServer) subscribe(connID, table string, filters map[string]interface{}) {
	s.mu.Lock()
	defer s.mu.Unlock()

	sub := &mockSubscription{
		connectionID: connID,
		table:        table,
		filters:      filters,
	}

	s.subscription[connID] = append(s.subscription[connID], sub)

	// Simulate subscription overhead
	time.Sleep(200 * time.Microsecond)
}

func (s *mockRealtimeServer) broadcast(message map[string]interface{}) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Simulate message serialization
	time.Sleep(10 * time.Microsecond)

	// Broadcast to all connections
	for range s.connections {
		// Simulate message delivery
		time.Sleep(1 * time.Microsecond)
	}
}

func (s *mockRealtimeServer) broadcastFiltered(event map[string]interface{}) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Filter and deliver to matching subscriptions
	for _, subs := range s.subscription {
		for _, sub := range subs {
			if s.matchesFilters(event, sub.filters) {
				// Simulate filtered delivery
				time.Sleep(2 * time.Microsecond)
			}
		}
	}
}

func (s *mockRealtimeServer) matchesFilters(event map[string]interface{}, filters map[string]interface{}) bool {
	if len(filters) == 0 {
		return true
	}

	for key, filterValue := range filters {
		eventValue, ok := event[key]
		if !ok || eventValue != filterValue {
			return false
		}
	}

	return true
}
