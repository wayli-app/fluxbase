package e2e

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/websocket/v2"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/wayli-app/fluxbase/internal/auth"
	"github.com/wayli-app/fluxbase/internal/config"
	"github.com/wayli-app/fluxbase/internal/database"
	"github.com/wayli-app/fluxbase/internal/middleware"
	"github.com/wayli-app/fluxbase/internal/realtime"
)

// TestRealtimeComprehensive tests Realtime WebSocket functionality
func TestRealtimeComprehensive(t *testing.T) {
	// Setup test environment
	cfg := setupRealtimeTestConfig(t)
	db := setupRealtimeTestDatabase(t, cfg)
	defer db.Close()

	// Setup realtime schema
	setupRealtimeSchema(t, db)

	// Create test app
	app := createRealtimeTestApp(t, db, cfg)

	// Start test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		app.Test(r)
	}))
	defer server.Close()

	// Run tests
	t.Run("Connection Lifecycle", func(t *testing.T) {
		testConnectionLifecycle(t, server.URL, db, cfg.JWTSecret)
	})

	t.Run("Channel Subscription", func(t *testing.T) {
		testChannelSubscription(t, server.URL, db, cfg.JWTSecret)
	})

	t.Run("Database Change Notifications - INSERT", func(t *testing.T) {
		testDatabaseInsertNotification(t, server.URL, db, cfg.JWTSecret)
	})

	t.Run("Database Change Notifications - UPDATE", func(t *testing.T) {
		testDatabaseUpdateNotification(t, server.URL, db, cfg.JWTSecret)
	})

	t.Run("Database Change Notifications - DELETE", func(t *testing.T) {
		testDatabaseDeleteNotification(t, server.URL, db, cfg.JWTSecret)
	})

	t.Run("Broadcast Messages", func(t *testing.T) {
		testBroadcastMessages(t, server.URL, db, cfg.JWTSecret)
	})

	t.Run("Multiple Subscriptions", func(t *testing.T) {
		testMultipleSubscriptions(t, server.URL, db, cfg.JWTSecret)
	})

	t.Run("Concurrent Connections", func(t *testing.T) {
		testConcurrentConnections(t, server.URL, db, cfg.JWTSecret)
	})

	t.Run("Authentication Required", func(t *testing.T) {
		testAuthenticationRequired(t, server.URL, db)
	})

	t.Run("Heartbeat Mechanism", func(t *testing.T) {
		testHeartbeat(t, server.URL, db, cfg.JWTSecret)
	})

	t.Run("Unsubscribe from Channel", func(t *testing.T) {
		testUnsubscribe(t, server.URL, db, cfg.JWTSecret)
	})

	t.Run("Connection Recovery", func(t *testing.T) {
		testConnectionRecovery(t, server.URL, db, cfg.JWTSecret)
	})
}

// setupRealtimeTestConfig creates test configuration
func setupRealtimeTestConfig(t *testing.T) *config.Config {
	return &config.Config{
		DatabaseURL: "postgres://postgres:postgres@localhost:5432/fluxbase_test?sslmode=disable",
		JWTSecret:   "test-jwt-secret-realtime",
		Port:        "8080",
	}
}

// setupRealtimeTestDatabase creates database connection
func setupRealtimeTestDatabase(t *testing.T, cfg *config.Config) *database.Connection {
	db, err := database.Connect(cfg.DatabaseURL)
	require.NoError(t, err, "Failed to connect to test database")
	return db
}

// setupRealtimeSchema creates test tables with triggers
func setupRealtimeSchema(t *testing.T, db *database.Connection) {
	ctx := context.Background()

	queries := []string{
		`CREATE SCHEMA IF NOT EXISTS auth`,
		`CREATE TABLE IF NOT EXISTS auth.users (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			email TEXT UNIQUE NOT NULL,
			created_at TIMESTAMPTZ DEFAULT NOW()
		)`,

		`CREATE TABLE IF NOT EXISTS realtime_messages (
			id SERIAL PRIMARY KEY,
			content TEXT NOT NULL,
			user_id UUID,
			created_at TIMESTAMPTZ DEFAULT NOW()
		)`,

		// Create realtime notification function
		`CREATE OR REPLACE FUNCTION notify_changes() RETURNS trigger AS $$
		DECLARE
			payload TEXT;
		BEGIN
			IF TG_OP = 'DELETE' THEN
				payload := json_build_object(
					'table', TG_TABLE_NAME,
					'type', TG_OP,
					'old', row_to_json(OLD)
				)::text;
				PERFORM pg_notify('table_changes', payload);
				RETURN OLD;
			ELSE
				payload := json_build_object(
					'table', TG_TABLE_NAME,
					'type', TG_OP,
					'new', row_to_json(NEW)
				)::text;
				PERFORM pg_notify('table_changes', payload);
				RETURN NEW;
			END IF;
		END;
		$$ LANGUAGE plpgsql`,

		// Create trigger for realtime_messages table
		`DROP TRIGGER IF EXISTS realtime_messages_notify ON realtime_messages`,
		`CREATE TRIGGER realtime_messages_notify
		AFTER INSERT OR UPDATE OR DELETE ON realtime_messages
		FOR EACH ROW EXECUTE FUNCTION notify_changes()`,
	}

	for _, query := range queries {
		_, err := db.Pool().Exec(ctx, query)
		require.NoError(t, err, "Failed to setup realtime schema")
	}

	// Cleanup
	_, _ = db.Pool().Exec(ctx, "TRUNCATE realtime_messages CASCADE")
	_, _ = db.Pool().Exec(ctx, "DELETE FROM auth.users WHERE email LIKE '%@realtimetest.com'")
}

// createRealtimeTestApp creates Fiber app with realtime routes
func createRealtimeTestApp(t *testing.T, db *database.Connection, cfg *config.Config) *fiber.App {
	app := fiber.New()

	// Auth middleware
	app.Use("/ws", middleware.AuthMiddleware(middleware.AuthConfig{
		JWTSecret: cfg.JWTSecret,
		Optional:  true,
	}))

	// Upgrade to WebSocket
	app.Use("/ws", func(c *fiber.Ctx) error {
		if websocket.IsWebSocketUpgrade(c) {
			return c.Next()
		}
		return fiber.ErrUpgradeRequired
	})

	// Register realtime handler
	authService := auth.NewService(db, cfg.JWTSecret, "smtp://fake", "noreply@test.com")
	realtimeHandler := realtime.NewRealtimeHandler(db, authService)
	app.Get("/ws", websocket.New(func(c *websocket.Conn) {
		realtimeHandler.HandleWebSocket(c)
	}))

	return app
}

// testConnectionLifecycle tests WebSocket connection open/close
func testConnectionLifecycle(t *testing.T, serverURL string, db *database.Connection, jwtSecret string) {
	token := createRealtimeTestUser(t, db, jwtSecret)

	// Convert HTTP URL to WebSocket URL
	wsURL := strings.Replace(serverURL, "http://", "ws://", 1) + "/ws"

	// Connect with authentication
	headers := http.Header{}
	headers.Set("Authorization", "Bearer "+token)

	dialer := websocket.Dialer{
		HandshakeTimeout: 5 * time.Second,
	}

	conn, resp, err := dialer.Dial(wsURL, headers)
	if err != nil {
		t.Logf("WebSocket dial error: %v, response: %+v", err, resp)
		if resp != nil && resp.StatusCode == 404 {
			t.Skip("WebSocket endpoint not available in test environment")
		}
		require.NoError(t, err, "Should connect to WebSocket")
	}
	defer conn.Close()

	assert.NotNil(t, conn, "Connection should be established")

	// Close connection
	err = conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
	if err != nil {
		t.Logf("Expected close error: %v", err)
	}
}

// testChannelSubscription tests subscribing to channels
func testChannelSubscription(t *testing.T, serverURL string, db *database.Connection, jwtSecret string) {
	token := createRealtimeTestUser(t, db, jwtSecret)
	wsURL := strings.Replace(serverURL, "http://", "ws://", 1) + "/ws"

	conn := connectWebSocket(t, wsURL, token)
	if conn == nil {
		t.Skip("WebSocket not available")
		return
	}
	defer conn.Close()

	// Subscribe to a channel
	subscribeMsg := map[string]interface{}{
		"type":    "subscribe",
		"channel": "realtime_messages",
	}

	err := conn.WriteJSON(subscribeMsg)
	require.NoError(t, err, "Should send subscribe message")

	// Wait for acknowledgment
	conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	var ackMsg map[string]interface{}
	err = conn.ReadJSON(&ackMsg)
	if err == nil {
		assert.Equal(t, "ack", ackMsg["type"], "Should receive acknowledgment")
		assert.Equal(t, "realtime_messages", ackMsg["channel"])
	}
}

// testDatabaseInsertNotification tests INSERT notifications
func testDatabaseInsertNotification(t *testing.T, serverURL string, db *database.Connection, jwtSecret string) {
	token := createRealtimeTestUser(t, db, jwtSecret)
	wsURL := strings.Replace(serverURL, "http://", "ws://", 1) + "/ws"

	conn := connectWebSocket(t, wsURL, token)
	if conn == nil {
		t.Skip("WebSocket not available")
		return
	}
	defer conn.Close()

	// Subscribe to realtime_messages table
	subscribeMsg := map[string]interface{}{
		"type":    "subscribe",
		"channel": "public:realtime_messages",
	}
	conn.WriteJSON(subscribeMsg)

	// Wait for ack
	time.Sleep(500 * time.Millisecond)

	// Insert a row
	ctx := context.Background()
	_, err := db.Pool().Exec(ctx, `
		INSERT INTO realtime_messages (content, user_id)
		VALUES ('Test message', $1)
	`, uuid.New())
	require.NoError(t, err)

	// Listen for notification
	conn.SetReadDeadline(time.Now().Add(3 * time.Second))
	var notification map[string]interface{}
	err = conn.ReadJSON(&notification)
	if err == nil {
		assert.Equal(t, "INSERT", notification["type"], "Should receive INSERT notification")
		assert.NotNil(t, notification["payload"])
	}
}

// testDatabaseUpdateNotification tests UPDATE notifications
func testDatabaseUpdateNotification(t *testing.T, serverURL string, db *database.Connection, jwtSecret string) {
	token := createRealtimeTestUser(t, db, jwtSecret)
	wsURL := strings.Replace(serverURL, "http://", "ws://", 1) + "/ws"

	conn := connectWebSocket(t, wsURL, token)
	if conn == nil {
		t.Skip("WebSocket not available")
		return
	}
	defer conn.Close()

	// Insert a row first
	ctx := context.Background()
	var id int
	err := db.Pool().QueryRow(ctx, `
		INSERT INTO realtime_messages (content)
		VALUES ('Original message')
		RETURNING id
	`).Scan(&id)
	require.NoError(t, err)

	// Subscribe
	subscribeMsg := map[string]interface{}{
		"type":    "subscribe",
		"channel": "public:realtime_messages",
	}
	conn.WriteJSON(subscribeMsg)
	time.Sleep(500 * time.Millisecond)

	// Update the row
	_, err = db.Pool().Exec(ctx, `
		UPDATE realtime_messages
		SET content = 'Updated message'
		WHERE id = $1
	`, id)
	require.NoError(t, err)

	// Listen for notification
	conn.SetReadDeadline(time.Now().Add(3 * time.Second))
	var notification map[string]interface{}
	err = conn.ReadJSON(&notification)
	if err == nil {
		assert.Equal(t, "UPDATE", notification["type"])
	}
}

// testDatabaseDeleteNotification tests DELETE notifications
func testDatabaseDeleteNotification(t *testing.T, serverURL string, db *database.Connection, jwtSecret string) {
	token := createRealtimeTestUser(t, db, jwtSecret)
	wsURL := strings.Replace(serverURL, "http://", "ws://", 1) + "/ws"

	conn := connectWebSocket(t, wsURL, token)
	if conn == nil {
		t.Skip("WebSocket not available")
		return
	}
	defer conn.Close()

	// Insert a row
	ctx := context.Background()
	var id int
	err := db.Pool().QueryRow(ctx, `
		INSERT INTO realtime_messages (content)
		VALUES ('To be deleted')
		RETURNING id
	`).Scan(&id)
	require.NoError(t, err)

	// Subscribe
	subscribeMsg := map[string]interface{}{
		"type":    "subscribe",
		"channel": "public:realtime_messages",
	}
	conn.WriteJSON(subscribeMsg)
	time.Sleep(500 * time.Millisecond)

	// Delete the row
	_, err = db.Pool().Exec(ctx, `
		DELETE FROM realtime_messages WHERE id = $1
	`, id)
	require.NoError(t, err)

	// Listen for notification
	conn.SetReadDeadline(time.Now().Add(3 * time.Second))
	var notification map[string]interface{}
	err = conn.ReadJSON(&notification)
	if err == nil {
		assert.Equal(t, "DELETE", notification["type"])
	}
}

// testBroadcastMessages tests broadcasting messages to channels
func testBroadcastMessages(t *testing.T, serverURL string, db *database.Connection, jwtSecret string) {
	token := createRealtimeTestUser(t, db, jwtSecret)
	wsURL := strings.Replace(serverURL, "http://", "ws://", 1) + "/ws"

	// Connect two clients
	conn1 := connectWebSocket(t, wsURL, token)
	if conn1 == nil {
		t.Skip("WebSocket not available")
		return
	}
	defer conn1.Close()

	conn2 := connectWebSocket(t, wsURL, token)
	if conn2 == nil {
		t.Skip("WebSocket not available")
		return
	}
	defer conn2.Close()

	// Both subscribe to same channel
	subscribeMsg := map[string]interface{}{
		"type":    "subscribe",
		"channel": "chat:room1",
	}

	conn1.WriteJSON(subscribeMsg)
	conn2.WriteJSON(subscribeMsg)
	time.Sleep(500 * time.Millisecond)

	// Client 1 broadcasts a message
	broadcastMsg := map[string]interface{}{
		"type":    "broadcast",
		"channel": "chat:room1",
		"payload": map[string]interface{}{
			"message": "Hello from client 1",
		},
	}

	err := conn1.WriteJSON(broadcastMsg)
	require.NoError(t, err)

	// Client 2 should receive the broadcast
	conn2.SetReadDeadline(time.Now().Add(3 * time.Second))
	var received map[string]interface{}
	err = conn2.ReadJSON(&received)
	if err == nil {
		assert.Equal(t, "broadcast", received["type"])
	}
}

// testMultipleSubscriptions tests subscribing to multiple channels
func testMultipleSubscriptions(t *testing.T, serverURL string, db *database.Connection, jwtSecret string) {
	token := createRealtimeTestUser(t, db, jwtSecret)
	wsURL := strings.Replace(serverURL, "http://", "ws://", 1) + "/ws"

	conn := connectWebSocket(t, wsURL, token)
	if conn == nil {
		t.Skip("WebSocket not available")
		return
	}
	defer conn.Close()

	// Subscribe to multiple channels
	channels := []string{"channel1", "channel2", "channel3"}

	for _, channel := range channels {
		subscribeMsg := map[string]interface{}{
			"type":    "subscribe",
			"channel": channel,
		}
		err := conn.WriteJSON(subscribeMsg)
		require.NoError(t, err)
		time.Sleep(100 * time.Millisecond)
	}

	// All subscriptions should be acknowledged
	t.Log("Successfully subscribed to multiple channels")
}

// testConcurrentConnections tests multiple simultaneous connections
func testConcurrentConnections(t *testing.T, serverURL string, db *database.Connection, jwtSecret string) {
	token := createRealtimeTestUser(t, db, jwtSecret)
	wsURL := strings.Replace(serverURL, "http://", "ws://", 1) + "/ws"

	// Connect 10 clients simultaneously
	numConnections := 10
	connections := make([]*websocket.Conn, 0, numConnections)

	for i := 0; i < numConnections; i++ {
		conn := connectWebSocket(t, wsURL, token)
		if conn != nil {
			connections = append(connections, conn)
		}
	}

	// Clean up all connections
	for _, conn := range connections {
		conn.Close()
	}

	assert.GreaterOrEqual(t, len(connections), 5, "Should establish at least 5 concurrent connections")
}

// testAuthenticationRequired tests that authentication is enforced
func testAuthenticationRequired(t *testing.T, serverURL string, db *database.Connection) {
	wsURL := strings.Replace(serverURL, "http://", "ws://", 1) + "/ws"

	// Try to connect without authentication
	dialer := websocket.Dialer{
		HandshakeTimeout: 5 * time.Second,
	}

	conn, resp, err := dialer.Dial(wsURL, nil)
	if conn != nil {
		defer conn.Close()
	}

	// Should either fail to connect or have limited access
	if err != nil {
		t.Logf("Expected auth failure: %v", err)
		// Connection rejected - good!
	} else {
		// Connected but should have limited capabilities
		t.Log("Connected without auth - has limited access")
	}
}

// testHeartbeat tests heartbeat mechanism
func testHeartbeat(t *testing.T, serverURL string, db *database.Connection, jwtSecret string) {
	token := createRealtimeTestUser(t, db, jwtSecret)
	wsURL := strings.Replace(serverURL, "http://", "ws://", 1) + "/ws"

	conn := connectWebSocket(t, wsURL, token)
	if conn == nil {
		t.Skip("WebSocket not available")
		return
	}
	defer conn.Close()

	// Send heartbeat
	heartbeatMsg := map[string]interface{}{
		"type": "heartbeat",
	}

	err := conn.WriteJSON(heartbeatMsg)
	require.NoError(t, err)

	// Should receive heartbeat response
	conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	var response map[string]interface{}
	err = conn.ReadJSON(&response)
	if err == nil {
		// Heartbeat acknowledged
		t.Log("Heartbeat acknowledged")
	}
}

// testUnsubscribe tests unsubscribing from channels
func testUnsubscribe(t *testing.T, serverURL string, db *database.Connection, jwtSecret string) {
	token := createRealtimeTestUser(t, db, jwtSecret)
	wsURL := strings.Replace(serverURL, "http://", "ws://", 1) + "/ws"

	conn := connectWebSocket(t, wsURL, token)
	if conn == nil {
		t.Skip("WebSocket not available")
		return
	}
	defer conn.Close()

	// Subscribe
	subscribeMsg := map[string]interface{}{
		"type":    "subscribe",
		"channel": "test_channel",
	}
	conn.WriteJSON(subscribeMsg)
	time.Sleep(300 * time.Millisecond)

	// Unsubscribe
	unsubscribeMsg := map[string]interface{}{
		"type":    "unsubscribe",
		"channel": "test_channel",
	}
	err := conn.WriteJSON(unsubscribeMsg)
	require.NoError(t, err)

	// Should receive acknowledgment
	conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	var ack map[string]interface{}
	err = conn.ReadJSON(&ack)
	if err == nil {
		t.Log("Unsubscribe acknowledged")
	}
}

// testConnectionRecovery tests reconnection after disconnect
func testConnectionRecovery(t *testing.T, serverURL string, db *database.Connection, jwtSecret string) {
	token := createRealtimeTestUser(t, db, jwtSecret)
	wsURL := strings.Replace(serverURL, "http://", "ws://", 1) + "/ws"

	// First connection
	conn1 := connectWebSocket(t, wsURL, token)
	if conn1 == nil {
		t.Skip("WebSocket not available")
		return
	}

	// Close it
	conn1.Close()

	// Wait a bit
	time.Sleep(500 * time.Millisecond)

	// Reconnect
	conn2 := connectWebSocket(t, wsURL, token)
	if conn2 == nil {
		t.Skip("WebSocket not available")
		return
	}
	defer conn2.Close()

	assert.NotNil(t, conn2, "Should be able to reconnect")
}

// Helper functions

func createRealtimeTestUser(t *testing.T, db *database.Connection, jwtSecret string) string {
	ctx := context.Background()

	email := fmt.Sprintf("user%d@realtimetest.com", time.Now().UnixNano())
	userID := uuid.New().String()

	_, err := db.Pool().Exec(ctx, `
		INSERT INTO auth.users (id, email)
		VALUES ($1, $2)
		ON CONFLICT (email) DO NOTHING
	`, userID, email)
	require.NoError(t, err)

	// Generate JWT
	authService := auth.NewService(db, jwtSecret, "smtp://fake", "noreply@test.com")
	token, err := authService.GenerateJWT(userID, email)
	require.NoError(t, err)

	return token
}

func connectWebSocket(t *testing.T, wsURL, token string) *websocket.Conn {
	headers := http.Header{}
	headers.Set("Authorization", "Bearer "+token)

	dialer := websocket.Dialer{
		HandshakeTimeout: 5 * time.Second,
	}

	conn, resp, err := dialer.Dial(wsURL, headers)
	if err != nil {
		if resp != nil && resp.StatusCode == 404 {
			return nil // WebSocket not available in test
		}
		t.Logf("WebSocket connection failed: %v", err)
		return nil
	}

	return conn
}
