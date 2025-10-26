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
)

// ClientMessage represents a message from the client
type ClientMessage struct {
	Type    MessageType     `json:"type"`
	Channel string          `json:"channel,omitempty"`
	Payload json.RawMessage `json:"payload,omitempty"`
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
}

// NewRealtimeHandler creates a new realtime handler
func NewRealtimeHandler(manager *Manager, authService AuthService) *RealtimeHandler {
	return &RealtimeHandler{
		manager:     manager,
		authService: authService,
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
	defer h.manager.RemoveConnection(connectionID)

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
		if msg.Channel == "" {
			conn.SendMessage(ServerMessage{
				Type:  MessageTypeError,
				Error: "channel is required for subscribe",
			})
			return
		}

		if err := h.manager.Subscribe(conn.ID, msg.Channel); err != nil {
			conn.SendMessage(ServerMessage{
				Type:  MessageTypeError,
				Error: err.Error(),
			})
			return
		}

		conn.SendMessage(ServerMessage{
			Type:    MessageTypeAck,
			Channel: msg.Channel,
			Payload: map[string]interface{}{
				"subscribed": true,
			},
		})

	case MessageTypeUnsubscribe:
		if msg.Channel == "" {
			conn.SendMessage(ServerMessage{
				Type:  MessageTypeError,
				Error: "channel is required for unsubscribe",
			})
			return
		}

		if err := h.manager.Unsubscribe(conn.ID, msg.Channel); err != nil {
			conn.SendMessage(ServerMessage{
				Type:  MessageTypeError,
				Error: err.Error(),
			})
			return
		}

		conn.SendMessage(ServerMessage{
			Type:    MessageTypeAck,
			Channel: msg.Channel,
			Payload: map[string]interface{}{
				"subscribed": false,
			},
		})

	case MessageTypeHeartbeat:
		// Respond to heartbeat
		conn.SendMessage(ServerMessage{
			Type: MessageTypeHeartbeat,
		})

	default:
		conn.SendMessage(ServerMessage{
			Type:  MessageTypeError,
			Error: "unknown message type",
		})
	}
}

// Broadcast sends a message to all subscribers of a channel
func (h *RealtimeHandler) Broadcast(channel string, payload interface{}) {
	h.manager.Broadcast(channel, ServerMessage{
		Type:    MessageTypeBroadcast,
		Channel: channel,
		Payload: payload,
	})
}

// GetStats returns realtime statistics
func (h *RealtimeHandler) GetStats() map[string]interface{} {
	return map[string]interface{}{
		"connections": h.manager.GetConnectionCount(),
		"channels":    h.manager.GetChannelCount(),
	}
}
