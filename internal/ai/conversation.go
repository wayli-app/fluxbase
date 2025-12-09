package ai

import (
	"context"
	"strings"
	"sync"
	"time"

	"github.com/fluxbase-eu/fluxbase/internal/database"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/rs/zerolog/log"
)

// ConversationManager handles conversation state management
// Uses a hybrid approach: in-memory cache for active conversations,
// database persistence for long-term storage
type ConversationManager struct {
	db          *database.Connection
	cache       map[string]*ConversationState
	cacheMu     sync.RWMutex
	cacheTTL    time.Duration
	maxTurns    int
	cleanupDone chan struct{}
}

// ConversationState represents the in-memory state of a conversation
type ConversationState struct {
	ID                    string
	ChatbotID             string
	ChatbotName           string
	UserID                *string
	SessionID             *string
	Messages              []Message
	TotalPromptTokens     int
	TotalCompletionTokens int
	TurnCount             int
	LastAccess            time.Time
	PersistToDatabase     bool
	ExpiresAt             *time.Time
}

// Conversation represents a conversation in the database
type Conversation struct {
	ID                    string     `json:"id"`
	ChatbotID             string     `json:"chatbot_id"`
	UserID                *string    `json:"user_id"`
	SessionID             *string    `json:"session_id"`
	Title                 *string    `json:"title"`
	Status                string     `json:"status"`
	TurnCount             int        `json:"turn_count"`
	TotalPromptTokens     int        `json:"total_prompt_tokens"`
	TotalCompletionTokens int        `json:"total_completion_tokens"`
	CreatedAt             time.Time  `json:"created_at"`
	UpdatedAt             time.Time  `json:"updated_at"`
	LastMessageAt         time.Time  `json:"last_message_at"`
	ExpiresAt             *time.Time `json:"expires_at"`
}

// ConversationMessage represents a message in a conversation
type ConversationMessage struct {
	ID               string                   `json:"id"`
	ConversationID   string                   `json:"conversation_id"`
	Role             string                   `json:"role"`
	Content          string                   `json:"content"`
	ToolCallID       *string                  `json:"tool_call_id,omitempty"`
	ToolName         *string                  `json:"tool_name,omitempty"`
	ToolInput        map[string]interface{}   `json:"tool_input,omitempty"`
	ToolOutput       map[string]interface{}   `json:"tool_output,omitempty"`
	ExecutedSQL      *string                  `json:"executed_sql,omitempty"`
	SQLResultSummary *string                  `json:"sql_result_summary,omitempty"`
	SQLRowCount      *int                     `json:"sql_row_count,omitempty"`
	SQLError         *string                  `json:"sql_error,omitempty"`
	SQLDurationMs    *int                     `json:"sql_duration_ms,omitempty"`
	QueryResults     []map[string]interface{} `json:"query_results,omitempty"` // Full query results for assistant messages
	PromptTokens     *int                     `json:"prompt_tokens,omitempty"`
	CompletionTokens *int                     `json:"completion_tokens,omitempty"`
	CreatedAt        time.Time                `json:"created_at"`
	SequenceNumber   int                      `json:"sequence_number"`
}

// NewConversationManager creates a new conversation manager
func NewConversationManager(db *database.Connection, cacheTTL time.Duration, maxTurns int) *ConversationManager {
	cm := &ConversationManager{
		db:          db,
		cache:       make(map[string]*ConversationState),
		cacheTTL:    cacheTTL,
		maxTurns:    maxTurns,
		cleanupDone: make(chan struct{}),
	}

	// Start cleanup goroutine
	go cm.cleanupLoop()

	return cm
}

// CreateConversation creates a new conversation
func (cm *ConversationManager) CreateConversation(ctx context.Context, chatbot *Chatbot, userID *string, sessionID *string) (*ConversationState, error) {
	conversationID := uuid.New().String()

	state := &ConversationState{
		ID:                chatbot.ID,
		ChatbotID:         chatbot.ID,
		ChatbotName:       chatbot.Name,
		UserID:            userID,
		SessionID:         sessionID,
		Messages:          []Message{},
		LastAccess:        time.Now(),
		PersistToDatabase: chatbot.PersistConversations,
	}

	// Set expiration if configured
	if chatbot.ConversationTTLHours > 0 {
		expiresAt := time.Now().Add(time.Duration(chatbot.ConversationTTLHours) * time.Hour)
		state.ExpiresAt = &expiresAt
	}

	// Persist to database if required
	if state.PersistToDatabase {
		conversation := &Conversation{
			ID:        conversationID,
			ChatbotID: chatbot.ID,
			UserID:    userID,
			SessionID: sessionID,
			Status:    "active",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
			ExpiresAt: state.ExpiresAt,
		}

		if err := cm.saveConversation(ctx, conversation); err != nil {
			log.Error().Err(err).Msg("Failed to save conversation to database")
			// Continue with in-memory only
		}
	}

	state.ID = conversationID

	// Add to cache
	cm.cacheMu.Lock()
	cm.cache[conversationID] = state
	cm.cacheMu.Unlock()

	log.Debug().
		Str("conversation_id", conversationID).
		Str("chatbot", chatbot.Name).
		Bool("persist", state.PersistToDatabase).
		Msg("Created new conversation")

	return state, nil
}

// GetConversation retrieves a conversation by ID
func (cm *ConversationManager) GetConversation(ctx context.Context, conversationID string) (*ConversationState, error) {
	// Check cache first
	cm.cacheMu.RLock()
	state, exists := cm.cache[conversationID]
	cm.cacheMu.RUnlock()

	if exists {
		state.LastAccess = time.Now()
		return state, nil
	}

	// Try to load from database
	conversation, err := cm.loadConversation(ctx, conversationID)
	if err != nil {
		return nil, err
	}

	if conversation == nil {
		return nil, nil
	}

	// Load messages
	messages, err := cm.loadMessages(ctx, conversationID)
	if err != nil {
		return nil, err
	}

	// Create state from database
	state = &ConversationState{
		ID:                    conversation.ID,
		ChatbotID:             conversation.ChatbotID,
		UserID:                conversation.UserID,
		SessionID:             conversation.SessionID,
		Messages:              messages,
		TotalPromptTokens:     conversation.TotalPromptTokens,
		TotalCompletionTokens: conversation.TotalCompletionTokens,
		TurnCount:             conversation.TurnCount,
		LastAccess:            time.Now(),
		PersistToDatabase:     true,
		ExpiresAt:             conversation.ExpiresAt,
	}

	// Add to cache
	cm.cacheMu.Lock()
	cm.cache[conversationID] = state
	cm.cacheMu.Unlock()

	return state, nil
}

// AddMessage adds a message to a conversation
func (cm *ConversationManager) AddMessage(ctx context.Context, conversationID string, msg Message, promptTokens, completionTokens int) error {
	cm.cacheMu.Lock()
	state, exists := cm.cache[conversationID]
	if !exists {
		cm.cacheMu.Unlock()
		return nil // Conversation not found
	}

	// Add message to state
	state.Messages = append(state.Messages, msg)
	state.TotalPromptTokens += promptTokens
	state.TotalCompletionTokens += completionTokens
	isFirstUserMessage := false
	if msg.Role == RoleUser {
		state.TurnCount++
		isFirstUserMessage = state.TurnCount == 1
	}
	state.LastAccess = time.Now()
	persistToDb := state.PersistToDatabase
	cm.cacheMu.Unlock()

	// Auto-generate title on first user message
	if persistToDb && isFirstUserMessage {
		go cm.autoGenerateTitleIfNeeded(conversationID, msg.Content)
	}

	// Persist to database if required
	if persistToDb {
		dbMsg := &ConversationMessage{
			ID:             uuid.New().String(),
			ConversationID: conversationID,
			Role:           string(msg.Role),
			Content:        msg.Content,
			SequenceNumber: len(state.Messages),
			CreatedAt:      time.Now(),
		}

		if promptTokens > 0 {
			dbMsg.PromptTokens = &promptTokens
		}
		if completionTokens > 0 {
			dbMsg.CompletionTokens = &completionTokens
		}

		// Convert QueryResults to the database format
		if len(msg.QueryResults) > 0 {
			dbQueryResults := make([]map[string]interface{}, len(msg.QueryResults))
			for i, qr := range msg.QueryResults {
				dbQueryResults[i] = map[string]interface{}{
					"query":     qr.Query,
					"summary":   qr.Summary,
					"row_count": qr.RowCount,
					"data":      qr.Data,
				}
			}
			dbMsg.QueryResults = dbQueryResults
		}

		if err := cm.saveMessage(ctx, dbMsg); err != nil {
			log.Error().Err(err).Msg("Failed to save message to database")
		}

		// Update conversation stats
		if err := cm.updateConversationStats(ctx, conversationID, state); err != nil {
			log.Error().Err(err).Msg("Failed to update conversation stats")
		}
	}

	return nil
}

// GetMessages returns all messages in a conversation
func (cm *ConversationManager) GetMessages(conversationID string) []Message {
	cm.cacheMu.RLock()
	defer cm.cacheMu.RUnlock()

	state, exists := cm.cache[conversationID]
	if !exists {
		return nil
	}

	return state.Messages
}

// GetActiveConversationsCount returns the count of active conversations
func (cm *ConversationManager) GetActiveConversationsCount() int {
	cm.cacheMu.RLock()
	defer cm.cacheMu.RUnlock()
	return len(cm.cache)
}

// CloseConversation closes a conversation
func (cm *ConversationManager) CloseConversation(ctx context.Context, conversationID string) error {
	cm.cacheMu.Lock()
	state, exists := cm.cache[conversationID]
	if exists {
		delete(cm.cache, conversationID)
	}
	cm.cacheMu.Unlock()

	if !exists {
		return nil
	}

	// Update database status if persisted
	if state.PersistToDatabase {
		query := `UPDATE ai.conversations SET status = 'archived', updated_at = NOW() WHERE id = $1`
		_, err := cm.db.Exec(ctx, query, conversationID)
		if err != nil {
			log.Error().Err(err).Str("id", conversationID).Msg("Failed to archive conversation")
			return err
		}
	}

	return nil
}

// Database operations

func (cm *ConversationManager) saveConversation(ctx context.Context, conv *Conversation) error {
	// Validate user_id exists in auth.users before inserting
	// Admin users (from dashboard.users) won't have entries in auth.users
	validUserID := conv.UserID
	if validUserID != nil {
		var exists bool
		err := cm.db.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM auth.users WHERE id = $1)", *validUserID).Scan(&exists)
		if err != nil {
			log.Warn().Err(err).Str("user_id", *validUserID).Msg("Failed to check if user exists, setting user_id to NULL")
			validUserID = nil
		} else if !exists {
			log.Debug().Str("user_id", *validUserID).Msg("User not found in auth.users (likely admin user), setting user_id to NULL")
			validUserID = nil
		}
	}

	query := `
		INSERT INTO ai.conversations (
			id, chatbot_id, user_id, session_id, title, status,
			turn_count, total_prompt_tokens, total_completion_tokens,
			created_at, updated_at, last_message_at, expires_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13
		)
	`

	_, err := cm.db.Exec(ctx, query,
		conv.ID, conv.ChatbotID, validUserID, conv.SessionID, conv.Title, conv.Status,
		conv.TurnCount, conv.TotalPromptTokens, conv.TotalCompletionTokens,
		conv.CreatedAt, conv.UpdatedAt, conv.LastMessageAt, conv.ExpiresAt,
	)

	return err
}

func (cm *ConversationManager) loadConversation(ctx context.Context, id string) (*Conversation, error) {
	query := `
		SELECT id, chatbot_id, user_id, session_id, title, status,
			turn_count, total_prompt_tokens, total_completion_tokens,
			created_at, updated_at, last_message_at, expires_at
		FROM ai.conversations
		WHERE id = $1 AND status = 'active'
	`

	conv := &Conversation{}
	err := cm.db.QueryRow(ctx, query, id).Scan(
		&conv.ID, &conv.ChatbotID, &conv.UserID, &conv.SessionID, &conv.Title, &conv.Status,
		&conv.TurnCount, &conv.TotalPromptTokens, &conv.TotalCompletionTokens,
		&conv.CreatedAt, &conv.UpdatedAt, &conv.LastMessageAt, &conv.ExpiresAt,
	)

	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return conv, nil
}

func (cm *ConversationManager) loadMessages(ctx context.Context, conversationID string) ([]Message, error) {
	query := `
		SELECT role, content, tool_call_id
		FROM ai.messages
		WHERE conversation_id = $1
		ORDER BY sequence_number
	`

	rows, err := cm.db.Query(ctx, query, conversationID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var messages []Message
	for rows.Next() {
		var role string
		var content string
		var toolCallID *string

		if err := rows.Scan(&role, &content, &toolCallID); err != nil {
			continue
		}

		msg := Message{
			Role:    Role(role),
			Content: content,
		}
		if toolCallID != nil {
			msg.ToolCallID = *toolCallID
		}
		messages = append(messages, msg)
	}

	return messages, nil
}

func (cm *ConversationManager) saveMessage(ctx context.Context, msg *ConversationMessage) error {
	query := `
		INSERT INTO ai.messages (
			id, conversation_id, role, content, tool_call_id, tool_name,
			executed_sql, sql_result_summary, sql_row_count, sql_error, sql_duration_ms,
			query_results, prompt_tokens, completion_tokens, created_at, sequence_number
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16
		)
	`

	_, err := cm.db.Exec(ctx, query,
		msg.ID, msg.ConversationID, msg.Role, msg.Content, msg.ToolCallID, msg.ToolName,
		msg.ExecutedSQL, msg.SQLResultSummary, msg.SQLRowCount, msg.SQLError, msg.SQLDurationMs,
		msg.QueryResults, msg.PromptTokens, msg.CompletionTokens, msg.CreatedAt, msg.SequenceNumber,
	)

	return err
}

func (cm *ConversationManager) updateConversationStats(ctx context.Context, conversationID string, state *ConversationState) error {
	query := `
		UPDATE ai.conversations SET
			turn_count = $2,
			total_prompt_tokens = $3,
			total_completion_tokens = $4,
			updated_at = NOW(),
			last_message_at = NOW()
		WHERE id = $1
	`

	_, err := cm.db.Exec(ctx, query,
		conversationID,
		state.TurnCount,
		state.TotalPromptTokens,
		state.TotalCompletionTokens,
	)

	return err
}

// cleanupLoop periodically cleans up expired conversations from cache
func (cm *ConversationManager) cleanupLoop() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			cm.cleanup()
		case <-cm.cleanupDone:
			return
		}
	}
}

func (cm *ConversationManager) cleanup() {
	cm.cacheMu.Lock()
	defer cm.cacheMu.Unlock()

	now := time.Now()
	for id, state := range cm.cache {
		// Remove if expired
		if state.ExpiresAt != nil && now.After(*state.ExpiresAt) {
			delete(cm.cache, id)
			log.Debug().Str("id", id).Msg("Removed expired conversation from cache")
			continue
		}

		// Remove if inactive for too long
		if now.Sub(state.LastAccess) > cm.cacheTTL {
			delete(cm.cache, id)
			log.Debug().Str("id", id).Msg("Removed inactive conversation from cache")
		}
	}
}

// Close shuts down the conversation manager
func (cm *ConversationManager) Close() {
	close(cm.cleanupDone)
}

// generateTitle creates an auto-generated title from the first user message
func generateTitle(content string) string {
	// Trim whitespace and normalize
	content = strings.TrimSpace(content)
	if content == "" {
		return "New conversation"
	}

	// If short enough, return as-is
	if len(content) <= 50 {
		return content
	}

	// Find the last space before 50 chars to preserve word boundaries
	truncated := content[:50]
	lastSpace := strings.LastIndex(truncated, " ")
	if lastSpace > 30 {
		return truncated[:lastSpace] + "..."
	}
	return truncated + "..."
}

// autoGenerateTitleIfNeeded generates a title for a conversation if not already set
func (cm *ConversationManager) autoGenerateTitleIfNeeded(conversationID, content string) {
	// Use a separate context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Check if title is already set
	var existingTitle *string
	query := `SELECT title FROM ai.conversations WHERE id = $1`
	err := cm.db.QueryRow(ctx, query, conversationID).Scan(&existingTitle)
	if err != nil {
		log.Warn().Err(err).Str("conversation_id", conversationID).Msg("Failed to check existing title")
		return
	}

	if existingTitle != nil && *existingTitle != "" {
		return // Title already set, don't override
	}

	// Generate and set title
	title := generateTitle(content)
	updateQuery := `UPDATE ai.conversations SET title = $2, updated_at = NOW() WHERE id = $1`
	_, err = cm.db.Exec(ctx, updateQuery, conversationID, title)
	if err != nil {
		log.Warn().Err(err).Str("conversation_id", conversationID).Msg("Failed to set auto-generated title")
		return
	}

	log.Debug().
		Str("conversation_id", conversationID).
		Str("title", title).
		Msg("Auto-generated conversation title")
}
