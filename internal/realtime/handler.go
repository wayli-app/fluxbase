package realtime

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync/atomic"
	"time"

	"github.com/gofiber/contrib/v3/websocket"
	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"

	"github.com/nimbleflux/fluxbase/internal/database"
)

// MessageType represents the type of WebSocket message
type MessageType string

// pingInterval is how often the server sends WebSocket pings to check connection health.
const pingInterval = 15 * time.Second

// pongTimeout is how long to wait for a pong response before considering the connection dead.
const pongTimeout = 60 * time.Second

const (
	MessageTypeSubscribe        MessageType = "subscribe"
	MessageTypeUnsubscribe      MessageType = "unsubscribe"
	MessageTypeHeartbeat        MessageType = "heartbeat"
	MessageTypeBroadcast        MessageType = "broadcast"
	MessageTypePresence         MessageType = "presence"
	MessageTypeError            MessageType = "error"
	MessageTypeAck              MessageType = "ack"
	MessageTypeChange           MessageType = "postgres_changes"
	MessageTypeAccessToken      MessageType = "access_token"
	MessageTypeSubscribeLogs    MessageType = "subscribe_logs"     // Subscribe to execution logs
	MessageTypeExecutionLog     MessageType = "execution_log"      // Execution log event from server
	MessageTypeSubscribeAllLogs MessageType = "subscribe_all_logs" // Subscribe to all logs (admin only)
	MessageTypeLogEntry         MessageType = "log_entry"          // Log entry event from server (all categories)
)

// ClientMessage represents a message from the client
type ClientMessage struct {
	Type           MessageType     `json:"type"`
	Channel        string          `json:"channel,omitempty"`
	Event          string          `json:"event,omitempty"` // INSERT, UPDATE, DELETE, or *
	Schema         string          `json:"schema,omitempty"`
	Table          string          `json:"table,omitempty"`
	Filter         string          `json:"filter,omitempty"` // Supabase-compatible filter: column=operator.value
	Payload        json.RawMessage `json:"payload,omitempty"`
	Config         json.RawMessage `json:"config,omitempty"` // Raw config - can be PostgresChangesConfig or LogSubscriptionConfig
	SubscriptionID string          `json:"subscription_id,omitempty"`
	MessageID      string          `json:"messageId,omitempty"` // Optional message ID for broadcast acknowledgements
	Token          string          `json:"token,omitempty"`     // JWT token for access_token message type
}

// LogSubscriptionConfig represents the config for subscribe_logs messages
type LogSubscriptionConfig struct {
	ExecutionID string `json:"execution_id"`
	Type        string `json:"type"` // function, job, rpc
}

// PostgresChangesConfig represents the config object in postgres_changes subscriptions
type PostgresChangesConfig struct {
	Event  string `json:"event"`            // INSERT, UPDATE, DELETE, or *
	Schema string `json:"schema"`           // Database schema
	Table  string `json:"table"`            // Table name
	Filter string `json:"filter,omitempty"` // Optional filter: column=operator.value
}

// ServerMessage represents a message to the client
type ServerMessage struct {
	Type    MessageType `json:"type"`
	Channel string      `json:"channel,omitempty"`
	Payload interface{} `json:"payload,omitempty"`
	Error   string      `json:"error,omitempty"`
}

// AuthService interface for JWT validation (allows mocking in tests)
type AuthService interface {
	ValidateToken(token string) (*TokenClaims, error)
}

// TokenClaims represents JWT claims
type TokenClaims struct {
	UserID    string
	Email     string
	Role      string
	SessionID string
	RawClaims map[string]interface{} // Full claims map for RLS (includes custom claims like meeting_id, player_id)
}

// RealtimeHandler handles WebSocket connections
type RealtimeHandler struct {
	manager         *Manager
	authService     AuthService
	subManager      *SubscriptionManager
	presenceManager *PresenceManager
}

// NewRealtimeHandler creates a new realtime handler
func NewRealtimeHandler(manager *Manager, authService AuthService, subManager *SubscriptionManager) *RealtimeHandler {
	return &RealtimeHandler{
		manager:         manager,
		authService:     authService,
		subManager:      subManager,
		presenceManager: NewPresenceManager(),
	}
}

// HandleWebSocket handles WebSocket upgrade and communication
func (h *RealtimeHandler) HandleWebSocket(c fiber.Ctx) error {
	// Check if WebSocket upgrade
	if !websocket.IsWebSocketUpgrade(c) {
		return fiber.ErrUpgradeRequired
	}

	// Extract JWT token - prefer Authorization header, fall back to query parameter
	// Note: Query parameter tokens are less secure (logged in server logs, browser history, etc.)
	// but are necessary for browser WebSocket connections which don't support custom headers.
	var token string
	var tokenSource string

	// Try Authorization header first (preferred, more secure)
	authHeader := c.Get("Authorization")
	if strings.HasPrefix(authHeader, "Bearer ") {
		token = strings.TrimPrefix(authHeader, "Bearer ")
		tokenSource = "header"
	}

	// Fall back to query parameter (less secure, but needed for browser WebSocket)
	if token == "" {
		token = c.Query("token")
		if token != "" {
			tokenSource = "query"
			// Log warning about using query parameter for auth
			log.Warn().
				Str("ip", c.IP()).
				Msg("WebSocket auth via query parameter (consider using Authorization header for better security)")
		}
	}

	var userID *string
	role := "anon" // Default to anonymous role
	var rawClaims map[string]interface{}

	if token != "" && h.authService != nil {
		claims, err := h.authService.ValidateToken(token)
		if err != nil {
			log.Debug().Err(err).Str("token_source", tokenSource).Msg("Invalid WebSocket token")
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Invalid or expired token",
			})
		}
		userID = &claims.UserID
		rawClaims = claims.RawClaims // Preserve full JWT claims for RLS
		// Extract role from JWT claims for RLS policy enforcement
		// Roles can be: "anon", "authenticated", "admin", "instance_admin", "service_role"
		if claims.Role != "" {
			role = claims.Role
		} else {
			role = "authenticated" // Default authenticated role if JWT doesn't specify one
		}
	}

	// Store user ID, role, tenant and claims in Fiber locals so handleConnection can access them
	c.Locals("user_id", userID)
	c.Locals("user_role", role)
	c.Locals("claims", rawClaims)
	// Preserve tenant_id from TenantMiddleware (already in Locals from middleware chain)

	// Upgrade to WebSocket
	return websocket.New(h.handleConnection)(c)
}

// handleConnection handles an individual WebSocket connection
func (h *RealtimeHandler) handleConnection(c *websocket.Conn) {
	// Generate connection ID
	connectionID := uuid.New().String()

	// Get user ID from Fiber locals (set in HandleWebSocket)
	var userID *string
	if uid := c.Locals("user_id"); uid != nil {
		if uidPtr, ok := uid.(*string); ok {
			userID = uidPtr
		}
	}

	// Get role from Fiber locals (set in HandleWebSocket)
	role := "anon" // Default to anonymous
	if r := c.Locals("user_role"); r != nil {
		if roleStr, ok := r.(string); ok {
			role = roleStr
		}
	}

	// Get full JWT claims from Fiber locals (set in HandleWebSocket)
	var claims map[string]interface{}
	if cl := c.Locals("claims"); cl != nil {
		if claimsMap, ok := cl.(map[string]interface{}); ok {
			claims = claimsMap
		}
	}
	if claims == nil {
		if cl := c.Locals("jwt_claims"); cl != nil {
			if claimsMap, ok := cl.(map[string]interface{}); ok {
				claims = claimsMap
			}
		}
	}

	// Get tenant ID from Fiber locals (set by TenantMiddleware)
	var tenantID string
	if tid := c.Locals("tenant_id"); tid != nil {
		if id, ok := tid.(string); ok {
			tenantID = id
		}
	}

	// Add connection to manager (checks connection limit)
	connection, err := h.manager.AddConnection(connectionID, c, userID, role, claims, tenantID)
	if err != nil {
		// Connection limit reached - send error and close
		_ = c.WriteJSON(ServerMessage{
			Type:    MessageTypeError,
			Error:   "max_connections_reached",
			Payload: map[string]interface{}{"message": "Server connection limit reached. Please try again later."},
		})
		return // Close the WebSocket
	}
	defer func() {
		// Clean up presence for this connection
		if h.presenceManager != nil {
			removed := h.presenceManager.CleanupConnection(connectionID)
			// Notify other clients about presence leaving
			for channel, info := range removed {
				h.notifyPresenceLeave(channel, info)
			}
		}
		// Clean up RLS-aware subscriptions
		if h.subManager != nil {
			h.subManager.RemoveConnectionSubscriptions(connectionID)
		}
		// Close the connection to cancel its context and clean up goroutines
		// This must be called BEFORE RemoveConnection to ensure the context
		// watcher goroutine can exit (it's blocked on <-connection.Context().Done())
		_ = connection.Close()
		// Remove from manager tracking
		h.manager.RemoveConnection(connectionID)
	}()

	// Channel for incoming messages - read in background goroutine
	msgChan := make(chan ClientMessage, 10)
	errChan := make(chan error, 1)

	// Goroutine to close connection when context is cancelled
	go func() {
		defer func() {
			if rec := recover(); rec != nil {
				log.Error().
					Interface("panic", rec).
					Str("connection_id", connectionID).
					Str("goroutine", "context_watcher").
					Msg("Panic in context watcher goroutine - recovered")
			}
		}()
		<-connection.Context().Done()
		// Close the WebSocket to unblock any pending reads
		_ = c.Close()
	}()

	go func() {
		defer func() {
			if rec := recover(); rec != nil {
				log.Error().
					Interface("panic", rec).
					Str("connection_id", connectionID).
					Str("goroutine", "message_reader").
					Msg("Panic in message reader goroutine - recovered")
			}
		}()
		for {
			var msg ClientMessage
			if err := c.ReadJSON(&msg); err != nil {
				errChan <- err
				return
			}
			msgChan <- msg
		}
	}()

	// Set up ping/pong for connection health monitoring
	// Track last pong time using atomic int64 (Unix timestamp)
	var lastPongNano atomic.Int64
	lastPongNano.Store(time.Now().UnixNano())

	// Set pong handler to update last pong time
	c.SetPongHandler(func(appData string) error {
		lastPongNano.Store(time.Now().UnixNano())
		return nil
	})

	// Start ping goroutine to detect dead connections
	pingTicker := time.NewTicker(pingInterval)
	defer pingTicker.Stop()

	go func() {
		defer func() {
			if rec := recover(); rec != nil {
				log.Error().
					Interface("panic", rec).
					Str("connection_id", connectionID).
					Str("goroutine", "ping_monitor").
					Msg("Panic in ping monitor goroutine - recovered")
			}
		}()
		for {
			select {
			case <-pingTicker.C:
				// Send ping message
				if err := c.WriteControl(websocket.PingMessage, []byte{}, time.Now().Add(10*time.Second)); err != nil {
					// Write failed, connection is likely dead
					return
				}

				// Check if we haven't received a pong in too long
				lastPong := time.Unix(0, lastPongNano.Load())
				if time.Since(lastPong) > pongTimeout {
					log.Warn().
						Str("connection_id", connectionID).
						Dur("since_pong", time.Since(lastPong)).
						Msg("Connection unresponsive to pings, closing")
					// Connection is dead, trigger close
					_ = c.Close()
					return
				}

			case <-connection.Context().Done():
				// Connection is being closed
				return
			}
		}
	}()

	// Start heartbeat ticker
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	// Non-blocking main loop
	for {
		select {
		case <-ticker.C:
			// Send heartbeat
			if err := connection.SendMessage(ServerMessage{
				Type: MessageTypeHeartbeat,
			}); err != nil {
				log.Error().Err(err).Str("connection_id", connectionID).Msg("Heartbeat failed")
				return
			}

		case msg := <-msgChan:
			// Handle message from read goroutine
			h.handleMessage(connection, msg)

		case err := <-errChan:
			// Handle read error
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Error().Err(err).Str("connection_id", connectionID).Msg("WebSocket error")
			}
			return
		}
	}
}

// handleMessage processes a client message
func (h *RealtimeHandler) handleMessage(conn *Connection, msg ClientMessage) {
	switch msg.Type {
	case MessageTypeSubscribe:
		// Check if this is a broadcast-only channel (no table required)
		isAdminChannel := len(msg.Channel) >= 15 && msg.Channel[:15] == "realtime:admin:"
		isFluxbaseChannel := len(msg.Channel) >= 9 && msg.Channel[:9] == "fluxbase:"

		if isAdminChannel || isFluxbaseChannel {
			// Admin channels require admin role
			if isAdminChannel {
				if conn.Role != "admin" && conn.Role != "instance_admin" && conn.Role != "service_role" {
					_ = conn.SendMessage(ServerMessage{
						Type:  MessageTypeError,
						Error: "admin access required to subscribe to admin channels",
					})
					return
				}
			}

			// Subscribe connection to channel (broadcast-only, no database subscription)
			if !conn.IsSubscribed(msg.Channel) {
				conn.Subscribe(msg.Channel)
			}

			// Send acknowledgment
			_ = conn.SendMessage(ServerMessage{
				Type: MessageTypeAck,
				Payload: map[string]interface{}{
					"subscribed": true,
					"channel":    msg.Channel,
				},
			})
			return
		}

		// Extract subscription details from either direct fields or config object
		var event, schema, table, filter string

		if len(msg.Config) > 0 {
			// New format: { type: "subscribe", channel: "...", config: { event, schema, table, filter } }
			var config PostgresChangesConfig
			if err := json.Unmarshal(msg.Config, &config); err == nil {
				event = config.Event
				schema = config.Schema
				table = config.Table
				filter = config.Filter
			}
		}
		// Fall back to legacy format fields if config wasn't parsed
		if table == "" {
			// Legacy format: { type: "subscribe", event, schema, table, filter }
			event = msg.Event
			schema = msg.Schema
			table = msg.Table
			filter = msg.Filter
		}

		// Validate table is provided
		if table == "" {
			_ = conn.SendMessage(ServerMessage{
				Type:  MessageTypeError,
				Error: "table is required for subscribe",
			})
			return
		}

		// Authentication required for all subscriptions
		if conn.UserID == nil {
			_ = conn.SendMessage(ServerMessage{
				Type:  MessageTypeError,
				Error: "authentication required for subscriptions",
			})
			return
		}

		// Default schema to "public"
		if schema == "" {
			schema = "public"
		}

		// Default event to "*" (all events)
		if event == "" {
			event = "*"
		}

		// Get user's role and claims from connection
		role := conn.Role
		claims := conn.Claims

		// Create RLS-aware subscription
		subID := uuid.New().String()
		_, err := h.subManager.CreateSubscription(
			subID,
			conn.ID,
			*conn.UserID,
			role,
			claims,
			schema,
			table,
			event,
			filter,
		)
		if err != nil {
			_ = conn.SendMessage(ServerMessage{
				Type:  MessageTypeError,
				Error: err.Error(),
			})
			return
		}

		// Send acknowledgment
		ackPayload := map[string]interface{}{
			"subscribed":      true,
			"subscription_id": subID,
			"schema":          schema,
			"table":           table,
			"event":           event,
		}
		if filter != "" {
			ackPayload["filter"] = filter
		}

		_ = conn.SendMessage(ServerMessage{
			Type:    MessageTypeAck,
			Payload: ackPayload,
		})

	case MessageTypeUnsubscribe:
		// Handle unsubscribe with subscription_id
		if msg.SubscriptionID != "" {
			// Remove the specific subscription
			err := h.subManager.RemoveSubscription(msg.SubscriptionID)
			if err != nil {
				// If subscription not found, it's already unsubscribed - treat as success
				// This handles race conditions during cleanup gracefully
				if err.Error() == "subscription not found" {
					_ = conn.SendMessage(ServerMessage{
						Type: MessageTypeAck,
						Payload: map[string]interface{}{
							"unsubscribed":    true,
							"subscription_id": msg.SubscriptionID,
						},
					})
					return
				}

				// For other errors, still report them
				_ = conn.SendMessage(ServerMessage{
					Type:  MessageTypeError,
					Error: fmt.Sprintf("failed to unsubscribe: %s", err.Error()),
				})
				return
			}

			// Send acknowledgment
			_ = conn.SendMessage(ServerMessage{
				Type: MessageTypeAck,
				Payload: map[string]interface{}{
					"unsubscribed":    true,
					"subscription_id": msg.SubscriptionID,
				},
			})
		} else {
			// Fallback: Remove all subscriptions for this connection
			// This provides graceful cleanup similar to connection close
			if h.subManager != nil {
				h.subManager.RemoveConnectionSubscriptions(conn.ID)
			}

			// Send acknowledgment
			_ = conn.SendMessage(ServerMessage{
				Type: MessageTypeAck,
				Payload: map[string]interface{}{
					"unsubscribed": true,
				},
			})
		}

	case MessageTypeHeartbeat:
		// Client heartbeat received - no echo needed
		// Server sends its own heartbeats on interval (line 172)

	case MessageTypeBroadcast:
		h.handleBroadcast(conn, msg)

	case MessageTypePresence:
		h.handlePresence(conn, msg)

	case MessageTypeAccessToken:
		h.handleAccessToken(conn, msg)

	case MessageTypeSubscribeLogs:
		h.handleSubscribeLogs(conn, msg)

	case MessageTypeSubscribeAllLogs:
		h.handleSubscribeAllLogs(conn, msg)

	default:
		_ = conn.SendMessage(ServerMessage{
			Type:  MessageTypeError,
			Error: "unknown message type",
		})
	}
}

// GetStats returns realtime statistics
func (h *RealtimeHandler) GetStats() map[string]interface{} {
	return map[string]interface{}{
		"connections": h.manager.GetConnectionCount(),
	}
}

// GetDetailedStats returns detailed realtime statistics
func (h *RealtimeHandler) GetDetailedStats() map[string]interface{} {
	return h.manager.GetDetailedStats()
}

// GetManager returns the realtime manager
func (h *RealtimeHandler) GetManager() *Manager {
	return h.manager
}

// handleBroadcast processes broadcast messages
func (h *RealtimeHandler) handleBroadcast(conn *Connection, msg ClientMessage) {
	if msg.Channel == "" {
		_ = conn.SendMessage(ServerMessage{
			Type:  MessageTypeError,
			Error: "channel is required for broadcast",
		})
		return
	}

	// Check if this is an admin channel subscription (read-only)
	if len(msg.Channel) >= 15 && msg.Channel[:15] == "realtime:admin:" {
		// Admin channels require admin role and are subscribe-only (no broadcasting)
		if conn.Role != "admin" && conn.Role != "instance_admin" && conn.Role != "service_role" {
			_ = conn.SendMessage(ServerMessage{
				Type:  MessageTypeError,
				Error: "admin access required to subscribe to admin channels",
			})
			return
		}

		// For admin channels, only allow subscription, not broadcasting
		// Subscribe connection to channel if not already subscribed
		if !conn.IsSubscribed(msg.Channel) {
			conn.Subscribe(msg.Channel)
		}

		// Send acknowledgment for subscription
		_ = conn.SendMessage(ServerMessage{
			Type: MessageTypeAck,
			Payload: map[string]interface{}{
				"subscribed": true,
				"channel":    msg.Channel,
			},
		})
		return
	}

	// Subscribe connection to channel if not already subscribed
	if !conn.IsSubscribed(msg.Channel) {
		conn.Subscribe(msg.Channel)
	}

	// Build broadcast payload
	broadcastPayload := map[string]interface{}{
		"event":   msg.Event,
		"payload": msg.Payload,
	}

	// Broadcast to all connections subscribed to this channel
	h.manager.BroadcastToChannel(msg.Channel, ServerMessage{
		Type:    MessageTypeBroadcast,
		Channel: msg.Channel,
		Payload: map[string]interface{}{
			"broadcast": broadcastPayload,
		},
	})

	// Send acknowledgment if messageId is present (Supabase-compatible broadcast acks)
	if msg.MessageID != "" {
		_ = conn.SendMessage(ServerMessage{
			Type: MessageTypeAck,
			Payload: map[string]interface{}{
				"messageId": msg.MessageID,
				"status":    "ok",
			},
		})
		log.Debug().
			Str("channel", msg.Channel).
			Str("messageId", msg.MessageID).
			Msg("Sent broadcast acknowledgment")
	}
}

// handlePresence processes presence messages
func (h *RealtimeHandler) handlePresence(conn *Connection, msg ClientMessage) {
	if msg.Channel == "" {
		_ = conn.SendMessage(ServerMessage{
			Type:  MessageTypeError,
			Error: "channel is required for presence",
		})
		return
	}

	// Subscribe connection to channel if not already subscribed
	if !conn.IsSubscribed(msg.Channel) {
		conn.Subscribe(msg.Channel)
	}

	// Parse payload to get presence event and data
	var presencePayload struct {
		Event string                 `json:"event"`
		Key   string                 `json:"key"`
		State map[string]interface{} `json:"state"`
	}

	if msg.Payload != nil {
		if err := json.Unmarshal(msg.Payload, &presencePayload); err != nil {
			_ = conn.SendMessage(ServerMessage{
				Type:  MessageTypeError,
				Error: "invalid presence payload",
			})
			return
		}
	}

	// Handle different presence events
	switch presencePayload.Event {
	case "track":
		if presencePayload.Key == "" {
			_ = conn.SendMessage(ServerMessage{
				Type:  MessageTypeError,
				Error: "key is required for presence track",
			})
			return
		}

		// Track presence
		info, isNew := h.presenceManager.Track(
			msg.Channel,
			presencePayload.Key,
			presencePayload.State,
			conn.UserID,
			conn.ID,
		)

		// Notify all clients in the channel about the new/updated presence
		if isNew {
			h.notifyPresenceJoin(msg.Channel, info)
		} else {
			// For updates, send sync event
			h.notifyPresenceSync(msg.Channel)
		}

	case "untrack":
		if presencePayload.Key == "" {
			_ = conn.SendMessage(ServerMessage{
				Type:  MessageTypeError,
				Error: "key is required for presence untrack",
			})
			return
		}

		// Untrack presence
		info := h.presenceManager.Untrack(msg.Channel, presencePayload.Key, conn.ID)
		if info != nil {
			h.notifyPresenceLeave(msg.Channel, info)
		}

	default:
		_ = conn.SendMessage(ServerMessage{
			Type:  MessageTypeError,
			Error: "unknown presence event",
		})
	}
}

// notifyPresenceJoin broadcasts a presence join event to all clients in the channel
func (h *RealtimeHandler) notifyPresenceJoin(channel string, info *PresenceInfo) {
	presenceState := h.presenceManager.GetPresenceState(channel)

	payload := map[string]interface{}{
		"event":            "join",
		"key":              info.Key,
		"newPresences":     []PresenceState{info.State},
		"currentPresences": presenceState,
	}

	h.manager.BroadcastToChannel(channel, ServerMessage{
		Type:    MessageTypePresence,
		Channel: channel,
		Payload: map[string]interface{}{
			"presence": payload,
		},
	})
}

// notifyPresenceLeave broadcasts a presence leave event to all clients in the channel
func (h *RealtimeHandler) notifyPresenceLeave(channel string, info *PresenceInfo) {
	presenceState := h.presenceManager.GetPresenceState(channel)

	payload := map[string]interface{}{
		"event":            "leave",
		"key":              info.Key,
		"leftPresences":    []PresenceState{info.State},
		"currentPresences": presenceState,
	}

	h.manager.BroadcastToChannel(channel, ServerMessage{
		Type:    MessageTypePresence,
		Channel: channel,
		Payload: map[string]interface{}{
			"presence": payload,
		},
	})
}

// notifyPresenceSync broadcasts a presence sync event to all clients in the channel
func (h *RealtimeHandler) notifyPresenceSync(channel string) {
	presenceState := h.presenceManager.GetPresenceState(channel)

	payload := map[string]interface{}{
		"event":            "sync",
		"currentPresences": presenceState,
	}

	h.manager.BroadcastToChannel(channel, ServerMessage{
		Type:    MessageTypePresence,
		Channel: channel,
		Payload: map[string]interface{}{
			"presence": payload,
		},
	})
}

// handleAccessToken processes access token update messages
func (h *RealtimeHandler) handleAccessToken(conn *Connection, msg ClientMessage) {
	if msg.Token == "" {
		_ = conn.SendMessage(ServerMessage{
			Type:  MessageTypeError,
			Error: "token is required for access_token message",
		})
		return
	}

	// Validate the new token
	claims, err := h.authService.ValidateToken(msg.Token)
	if err != nil {
		log.Warn().
			Err(err).
			Str("connection_id", conn.ID).
			Msg("Invalid token in access_token message")
		_ = conn.SendMessage(ServerMessage{
			Type:  MessageTypeError,
			Error: "invalid token",
		})
		return
	}

	// Update the connection's auth context
	var userID *string
	if claims.UserID != "" {
		userID = &claims.UserID
	}

	oldRole := conn.Role
	conn.UpdateAuth(userID, claims.Role, claims.RawClaims)

	// If role or claims changed, update subscriptions in the subscription manager
	if oldRole != claims.Role && h.subManager != nil {
		h.subManager.UpdateConnectionRole(conn.ID, claims.Role)
	}
	// Also update claims in subscriptions for RLS
	if h.subManager != nil {
		h.subManager.UpdateConnectionClaims(conn.ID, claims.RawClaims)
	}

	log.Info().
		Str("connection_id", conn.ID).
		Str("user_id", claims.UserID).
		Str("old_role", oldRole).
		Str("new_role", claims.Role).
		Msg("Access token updated on connection")

	// Send acknowledgment
	_ = conn.SendMessage(ServerMessage{
		Type: MessageTypeAck,
		Payload: map[string]interface{}{
			"type":    "access_token",
			"updated": true,
		},
	})
}

// handleSubscribeLogs processes execution log subscription requests
func (h *RealtimeHandler) handleSubscribeLogs(conn *Connection, msg ClientMessage) {
	// Extract execution_id and type from config or payload
	var executionID, executionType string

	// Try to parse config first (SDK sends { execution_id, type } in config)
	if len(msg.Config) > 0 {
		var logConfig LogSubscriptionConfig
		if err := json.Unmarshal(msg.Config, &logConfig); err == nil {
			executionID = logConfig.ExecutionID
			executionType = logConfig.Type
		}

		// Also try legacy format (schema/table fields)
		if executionID == "" {
			var pgConfig PostgresChangesConfig
			if err := json.Unmarshal(msg.Config, &pgConfig); err == nil {
				executionID = pgConfig.Schema
				executionType = pgConfig.Table
			}
		}
	}

	// Fall back to payload if config didn't have it
	if executionID == "" && len(msg.Payload) > 0 {
		var payload map[string]interface{}
		if err := json.Unmarshal(msg.Payload, &payload); err == nil {
			if v, ok := payload["execution_id"].(string); ok {
				executionID = v
			}
			if v, ok := payload["type"].(string); ok {
				executionType = v
			}
		}
	}

	// Validate execution_id is provided
	if executionID == "" {
		_ = conn.SendMessage(ServerMessage{
			Type:  MessageTypeError,
			Error: "execution_id is required for subscribe_logs",
		})
		return
	}

	// Check ownership (unless admin/service role)
	if conn.Role != "admin" && conn.Role != "instance_admin" && conn.Role != "service_role" {
		if conn.UserID == nil {
			_ = conn.SendMessage(ServerMessage{
				Type:  MessageTypeError,
				Error: "authentication required",
			})
			return
		}

		ctx := context.Background()
		if conn.TenantID != "" {
			ctx = database.ContextWithTenant(ctx, conn.TenantID)
		}
		isOwner, exists, err := h.subManager.CheckExecutionOwnership(
			ctx, executionID, *conn.UserID, executionType,
		)
		if err != nil {
			log.Error().Err(err).Str("execution_id", executionID).Msg("Failed to check execution ownership")
			_ = conn.SendMessage(ServerMessage{
				Type:  MessageTypeError,
				Error: "failed to verify execution access",
			})
			return
		}
		if !exists {
			_ = conn.SendMessage(ServerMessage{
				Type:  MessageTypeError,
				Error: "execution not found",
			})
			return
		}
		if !isOwner {
			// For other user's executions, return same error to avoid information disclosure
			_ = conn.SendMessage(ServerMessage{
				Type:  MessageTypeError,
				Error: "execution not found",
			})
			return
		}
	}

	// Create log subscription
	subID := uuid.New().String()
	_, err := h.subManager.CreateLogSubscription(subID, conn.ID, executionID, executionType)
	if err != nil {
		_ = conn.SendMessage(ServerMessage{
			Type:  MessageTypeError,
			Error: err.Error(),
		})
		return
	}

	// Send acknowledgment
	ackPayload := map[string]interface{}{
		"subscribed":      true,
		"subscription_id": subID,
		"execution_id":    executionID,
	}
	if executionType != "" {
		ackPayload["type"] = executionType
	}

	_ = conn.SendMessage(ServerMessage{
		Type:    MessageTypeAck,
		Payload: ackPayload,
	})

	log.Info().
		Str("connection_id", conn.ID).
		Str("subscription_id", subID).
		Str("execution_id", executionID).
		Str("execution_type", executionType).
		Msg("Created execution log subscription")
}

// handleSubscribeAllLogs processes all-logs subscription requests (admin only).
func (h *RealtimeHandler) handleSubscribeAllLogs(conn *Connection, msg ClientMessage) {
	// Require admin role for all-logs subscription
	if conn.Role != "admin" && conn.Role != "instance_admin" && conn.Role != "service_role" {
		_ = conn.SendMessage(ServerMessage{
			Type:  MessageTypeError,
			Error: "admin role required for all-logs subscription",
		})
		return
	}

	// Extract optional filters from payload
	var filters struct {
		Category string   `json:"category,omitempty"`
		Levels   []string `json:"levels,omitempty"`
	}

	if msg.Payload != nil {
		_ = json.Unmarshal(msg.Payload, &filters)
	}

	// Create all-logs subscription
	subID := uuid.New().String()
	_, err := h.subManager.CreateAllLogsSubscription(subID, conn.ID, filters.Category, filters.Levels)
	if err != nil {
		_ = conn.SendMessage(ServerMessage{
			Type:  MessageTypeError,
			Error: err.Error(),
		})
		return
	}

	// Send acknowledgment
	ackPayload := map[string]interface{}{
		"subscribed":      true,
		"subscription_id": subID,
		"type":            "all_logs",
	}
	if filters.Category != "" {
		ackPayload["category"] = filters.Category
	}
	if len(filters.Levels) > 0 {
		ackPayload["levels"] = filters.Levels
	}

	_ = conn.SendMessage(ServerMessage{
		Type:    MessageTypeAck,
		Payload: ackPayload,
	})

	log.Debug().
		Str("connection_id", conn.ID).
		Str("subscription_id", subID).
		Str("category", filters.Category).
		Strs("levels", filters.Levels).
		Msg("Created all-logs subscription")
}
