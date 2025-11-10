package realtime

import (
	"encoding/json"
	"time"

	"github.com/gofiber/contrib/websocket"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
)

// MessageType represents the type of WebSocket message
type MessageType string

const (
	MessageTypeSubscribe   MessageType = "subscribe"
	MessageTypeUnsubscribe MessageType = "unsubscribe"
	MessageTypeHeartbeat   MessageType = "heartbeat"
	MessageTypeBroadcast   MessageType = "broadcast"
	MessageTypeError       MessageType = "error"
	MessageTypeAck         MessageType = "ack"
	MessageTypeChange      MessageType = "change"
)

// ClientMessage represents a message from the client
type ClientMessage struct {
	Type    MessageType            `json:"type"`
	Channel string                 `json:"channel,omitempty"`
	Event   string                 `json:"event,omitempty"` // INSERT, UPDATE, DELETE, or *
	Schema  string                 `json:"schema,omitempty"`
	Table   string                 `json:"table,omitempty"`
	Filter  string                 `json:"filter,omitempty"` // Supabase-compatible filter: column=operator.value
	Payload json.RawMessage        `json:"payload,omitempty"`
	Config  *PostgresChangesConfig `json:"config,omitempty"` // Alternative format for postgres_changes
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
	manager     *Manager
	authService AuthService
	subManager  *SubscriptionManager
}

// NewRealtimeHandler creates a new realtime handler
func NewRealtimeHandler(manager *Manager, authService AuthService, subManager *SubscriptionManager) *RealtimeHandler {
	return &RealtimeHandler{
		manager:     manager,
		authService: authService,
		subManager:  subManager,
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

	if token != "" && h.authService != nil {
		claims, err := h.authService.ValidateToken(token)
		if err != nil {
			log.Debug().Err(err).Msg("Invalid WebSocket token")
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Invalid or expired token",
			})
		}
		userID = &claims.UserID
		log.Debug().Str("user_id", claims.UserID).Msg("WebSocket authenticated")
	}

	// Store user ID in Fiber locals so handleConnection can access it
	c.Locals("user_id", userID)

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
		userID = uid.(*string)
	}

	// Add connection to manager
	connection := h.manager.AddConnection(connectionID, c, userID)
	defer func() {
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
		// Extract subscription details from either direct fields or config object
		var event, schema, table, filter string

		if msg.Config != nil {
			// New format: { type: "subscribe", channel: "...", config: { event, schema, table, filter } }
			event = msg.Config.Event
			schema = msg.Config.Schema
			table = msg.Config.Table
			filter = msg.Config.Filter
		} else {
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

		// Get user's role from connection or default to "authenticated"
		role := "authenticated"
		// TODO: Fetch actual role from token claims or connection metadata

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
		// Unsubscribe is handled automatically when connection closes
		// We don't support manual unsubscribe for individual subscriptions yet
		_ = conn.SendMessage(ServerMessage{
			Type:  MessageTypeError,
			Error: "unsubscribe not supported - subscriptions are removed on disconnect",
		})

	case MessageTypeHeartbeat:
		// Respond to heartbeat
		_ = conn.SendMessage(ServerMessage{
			Type: MessageTypeHeartbeat,
		})

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
