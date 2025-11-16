package realtime

import (
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
	MessageTypeSubscribe   MessageType = "subscribe"
	MessageTypeUnsubscribe MessageType = "unsubscribe"
	MessageTypeHeartbeat   MessageType = "heartbeat"
	MessageTypeBroadcast   MessageType = "broadcast"
	MessageTypePresence    MessageType = "presence"
	MessageTypeError       MessageType = "error"
	MessageTypeAck         MessageType = "ack"
	MessageTypeChange      MessageType = "change"
)

// ClientMessage represents a message from the client
type ClientMessage struct {
	Type           MessageType            `json:"type"`
	Channel        string                 `json:"channel,omitempty"`
	Event          string                 `json:"event,omitempty"` // INSERT, UPDATE, DELETE, or *
	Schema         string                 `json:"schema,omitempty"`
	Table          string                 `json:"table,omitempty"`
	Filter         string                 `json:"filter,omitempty"` // Supabase-compatible filter: column=operator.value
	Payload        json.RawMessage        `json:"payload,omitempty"`
	Config         *PostgresChangesConfig `json:"config,omitempty"` // Alternative format for postgres_changes
	SubscriptionID string                 `json:"subscription_id,omitempty"`
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
		// Validate subscription ID is provided
		if msg.SubscriptionID == "" {
			_ = conn.SendMessage(ServerMessage{
				Type:  MessageTypeError,
				Error: "subscription_id is required for unsubscribe",
			})
			return
		}

		// Remove the subscription
		err := h.subManager.RemoveSubscription(msg.SubscriptionID)
		if err != nil {
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

	case MessageTypeHeartbeat:
		// Respond to heartbeat
		_ = conn.SendMessage(ServerMessage{
			Type: MessageTypeHeartbeat,
		})

	case MessageTypeBroadcast:
		h.handleBroadcast(conn, msg)

	case MessageTypePresence:
		h.handlePresence(conn, msg)

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
