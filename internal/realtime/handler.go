package realtime

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/gofiber/contrib/websocket"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
)

// MessageType represents the type of WebSocket message
type MessageType string

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
func (h *RealtimeHandler) HandleWebSocket(c *fiber.Ctx) error {
	// Check if WebSocket upgrade
	if !websocket.IsWebSocketUpgrade(c) {
		return fiber.ErrUpgradeRequired
	}

	// Extract and validate JWT token from query parameter
	token := c.Query("token")
	var userID *string
	var role = "anon" // Default to anonymous role

	if token != "" && h.authService != nil {
		claims, err := h.authService.ValidateToken(token)
		if err != nil {
			log.Debug().Err(err).Msg("Invalid WebSocket token")
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Invalid or expired token",
			})
		}
		userID = &claims.UserID
		// Extract role from JWT claims for RLS policy enforcement
		// Roles can be: "anon", "authenticated", "admin", "dashboard_admin", "service_role"
		if claims.Role != "" {
			role = claims.Role
		} else {
			role = "authenticated" // Default authenticated role if JWT doesn't specify one
		}
	}

	// Store user ID and role in Fiber locals so handleConnection can access them
	c.Locals("user_id", userID)
	c.Locals("role", role)

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
	if r := c.Locals("role"); r != nil {
		if roleStr, ok := r.(string); ok {
			role = roleStr
		}
	}

	// Add connection to manager
	connection := h.manager.AddConnection(connectionID, c, userID, role)
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
		h.manager.RemoveConnection(connectionID)
	}()

	// Start heartbeat ticker
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	// Handle incoming messages
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

		default:
			// Read message
			var msg ClientMessage
			if err := c.ReadJSON(&msg); err != nil {
				// Check if connection was closed
				if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
					log.Error().Err(err).Str("connection_id", connectionID).Msg("WebSocket error")
				}
				return
			}

			// Handle message
			h.handleMessage(connection, msg)
		}
	}
}

// handleMessage processes a client message
func (h *RealtimeHandler) handleMessage(conn *Connection, msg ClientMessage) {
	switch msg.Type {
	case MessageTypeSubscribe:
		// Check if this is an admin channel subscription (broadcast-only, no table required)
		if len(msg.Channel) >= 15 && msg.Channel[:15] == "realtime:admin:" {
			// Admin channels require admin role
			if conn.Role != "admin" && conn.Role != "dashboard_admin" && conn.Role != "service_role" {
				_ = conn.SendMessage(ServerMessage{
					Type:  MessageTypeError,
					Error: "admin access required to subscribe to admin channels",
				})
				return
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

		// Get user's role from connection
		role := conn.Role

		// Create RLS-aware subscription
		subID := uuid.New().String()
		_, err := h.subManager.CreateSubscription(
			subID,
			conn.ID,
			*conn.UserID,
			role,
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
		if conn.Role != "admin" && conn.Role != "dashboard_admin" && conn.Role != "service_role" {
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
	conn.UpdateAuth(userID, claims.Role)

	// If role changed, update subscriptions in the subscription manager
	if oldRole != claims.Role && h.subManager != nil {
		h.subManager.UpdateConnectionRole(conn.ID, claims.Role)
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
	if conn.Role != "admin" && conn.Role != "dashboard_admin" && conn.Role != "service_role" {
		if conn.UserID == nil {
			_ = conn.SendMessage(ServerMessage{
				Type:  MessageTypeError,
				Error: "authentication required",
			})
			return
		}

		isOwner, exists, err := h.subManager.CheckExecutionOwnership(
			context.Background(), executionID, *conn.UserID, executionType,
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
	if conn.Role != "admin" && conn.Role != "dashboard_admin" && conn.Role != "service_role" {
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
