package realtime

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
)

// Subscription represents an RLS-aware subscription to table changes
type Subscription struct {
	ID     string
	UserID string
	Role   string
	Claims map[string]interface{} // Full JWT claims for RLS (includes custom claims like meeting_id, player_id)
	Table  string
	Schema string
	Event  string  // INSERT, UPDATE, DELETE, or * for all
	Filter *Filter // Filter (column=operator.value)
	ConnID string  // Connection ID this subscription belongs to
}

// LogSubscription represents a subscription to execution logs
type LogSubscription struct {
	ID            string
	ConnID        string
	ExecutionID   string
	ExecutionType string // "function", "job", "rpc"
}

// AllLogsSubscription represents a subscription to all logs (admin only)
type AllLogsSubscription struct {
	ID       string
	ConnID   string
	Category string   // Optional filter by category
	Levels   []string // Optional filter by levels
}

// SubscriptionManager manages RLS-aware subscriptions
type SubscriptionManager struct {
	db            SubscriptionDB
	subscriptions map[string]*Subscription        // subscription ID -> subscription
	userSubs      map[string]map[string]bool      // user ID -> subscription IDs
	tableSubs     map[string]map[string]bool      // "schema.table" -> subscription IDs
	logSubs       map[string]*LogSubscription     // subscription ID -> log subscription
	execLogSubs   map[string]map[string]bool      // execution ID -> subscription IDs
	allLogsSubs   map[string]*AllLogsSubscription // subscription ID -> all-logs subscription
	rlsCache      *rlsCache                       // RLS check result cache
	mu            sync.RWMutex
}

// NewSubscriptionManager creates a new subscription manager with default RLS cache settings.
// For production use, pass NewPgxSubscriptionDB(pool).
// For testing, pass a mock implementation of SubscriptionDB.
func NewSubscriptionManager(db SubscriptionDB) *SubscriptionManager {
	return NewSubscriptionManagerWithConfig(db, RLSCacheConfig{})
}

// NewSubscriptionManagerWithConfig creates a new subscription manager with custom RLS cache settings.
func NewSubscriptionManagerWithConfig(db SubscriptionDB, cacheConfig RLSCacheConfig) *SubscriptionManager {
	return &SubscriptionManager{
		db:            db,
		subscriptions: make(map[string]*Subscription),
		userSubs:      make(map[string]map[string]bool),
		tableSubs:     make(map[string]map[string]bool),
		logSubs:       make(map[string]*LogSubscription),
		execLogSubs:   make(map[string]map[string]bool),
		allLogsSubs:   make(map[string]*AllLogsSubscription),
		rlsCache:      newRLSCacheWithConfig(cacheConfig),
	}
}

// CreateSubscription creates a new RLS-aware subscription
func (sm *SubscriptionManager) CreateSubscription(
	subID string,
	connID string,
	userID string,
	role string,
	claims map[string]interface{},
	schema string,
	table string,
	event string,
	filterStr string,
) (*Subscription, error) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	// Validate table exists and is allowed for realtime
	if !sm.isTableAllowedUnsafe(schema, table) {
		return nil, fmt.Errorf("table %s.%s not enabled for realtime", schema, table)
	}

	// Parse filter
	filter, err := ParseFilter(filterStr)
	if err != nil {
		return nil, fmt.Errorf("invalid filter: %w", err)
	}

	// Default event to "*" (all events)
	if event == "" {
		event = "*"
	}

	sub := &Subscription{
		ID:     subID,
		UserID: userID,
		Role:   role,
		Claims: claims,
		Table:  table,
		Schema: schema,
		Event:  event,
		Filter: filter,
		ConnID: connID,
	}

	// Store subscription
	sm.subscriptions[subID] = sub

	// Track by user
	if _, exists := sm.userSubs[userID]; !exists {
		sm.userSubs[userID] = make(map[string]bool)
	}
	sm.userSubs[userID][subID] = true

	// Track by table
	tableKey := fmt.Sprintf("%s.%s", schema, table)
	if _, exists := sm.tableSubs[tableKey]; !exists {
		sm.tableSubs[tableKey] = make(map[string]bool)
	}
	sm.tableSubs[tableKey][subID] = true

	log.Debug().
		Str("sub_id", subID).
		Str("user_id", userID).
		Str("table", tableKey).
		Msg("Created RLS-aware subscription")

	return sub, nil
}

// RemoveSubscription removes a subscription
func (sm *SubscriptionManager) RemoveSubscription(subID string) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	sub, exists := sm.subscriptions[subID]
	if !exists {
		return fmt.Errorf("subscription not found")
	}

	// Remove from user subscriptions
	if userSubs, exists := sm.userSubs[sub.UserID]; exists {
		delete(userSubs, subID)
		if len(userSubs) == 0 {
			delete(sm.userSubs, sub.UserID)
		}
	}

	// Remove from table subscriptions
	tableKey := fmt.Sprintf("%s.%s", sub.Schema, sub.Table)
	if tableSubs, exists := sm.tableSubs[tableKey]; exists {
		delete(tableSubs, subID)
		if len(tableSubs) == 0 {
			delete(sm.tableSubs, tableKey)
		}
	}

	delete(sm.subscriptions, subID)

	log.Info().
		Str("sub_id", subID).
		Str("user_id", sub.UserID).
		Msg("Removed subscription")

	return nil
}

// RemoveConnectionSubscriptions removes all subscriptions for a connection
func (sm *SubscriptionManager) RemoveConnectionSubscriptions(connID string) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	for subID, sub := range sm.subscriptions {
		if sub.ConnID == connID {
			// Remove from user subscriptions
			if userSubs, exists := sm.userSubs[sub.UserID]; exists {
				delete(userSubs, subID)
				if len(userSubs) == 0 {
					delete(sm.userSubs, sub.UserID)
				}
			}

			// Remove from table subscriptions
			tableKey := fmt.Sprintf("%s.%s", sub.Schema, sub.Table)
			if tableSubs, exists := sm.tableSubs[tableKey]; exists {
				delete(tableSubs, subID)
				if len(tableSubs) == 0 {
					delete(sm.tableSubs, tableKey)
				}
			}

			delete(sm.subscriptions, subID)

			log.Info().
				Str("sub_id", subID).
				Str("user_id", sub.UserID).
				Msg("Removed subscription")
		}
	}
}

// FilterEventForSubscribers filters a change event for all subscribers with RLS
func (sm *SubscriptionManager) FilterEventForSubscribers(ctx context.Context, event *ChangeEvent) map[string]*ChangeEvent {
	sm.mu.RLock()

	tableKey := fmt.Sprintf("%s.%s", event.Schema, event.Table)
	subIDs, exists := sm.tableSubs[tableKey]
	if !exists || len(subIDs) == 0 {
		sm.mu.RUnlock()
		return nil
	}

	// Get copy of subscription IDs
	subIDsCopy := make([]string, 0, len(subIDs))
	for subID := range subIDs {
		subIDsCopy = append(subIDsCopy, subID)
	}
	sm.mu.RUnlock()

	// Filter for each subscription
	result := make(map[string]*ChangeEvent)
	for _, subID := range subIDsCopy {
		sm.mu.RLock()
		sub, exists := sm.subscriptions[subID]
		if !exists {
			sm.mu.RUnlock()
			continue
		}
		// Copy claims while holding lock to prevent concurrent map access
		claims := copyClaims(sub.Claims)
		sm.mu.RUnlock()

		// Check if event type matches subscription
		if !sm.matchesEvent(event.Type, sub.Event) {
			continue
		}

		// Check RLS access with copied claims
		if sm.checkRLSAccess(ctx, sub, event, claims) {
			// Check filter
			if sm.matchesFilter(event, sub) {
				result[sub.ConnID] = event
			}
		}
	}

	return result
}

// matchesEvent checks if an event type matches the subscription event filter
func (sm *SubscriptionManager) matchesEvent(eventType, subEvent string) bool {
	if subEvent == "*" {
		return true
	}
	return eventType == subEvent
}

// matchesFilter checks if an event matches the subscription filter
func (sm *SubscriptionManager) matchesFilter(event *ChangeEvent, sub *Subscription) bool {
	// Use filter if present
	if sub.Filter != nil {
		record := event.Record
		if record == nil {
			record = event.OldRecord
		}
		return sub.Filter.Matches(record)
	}

	// No filter specified - match all
	return true
}

// checkRLSAccess verifies if a user can access a record based on RLS policies
// Uses a short-lived cache to reduce database load for repeated checks
// Claims are passed as a parameter to avoid concurrent map access - they must be copied
// while holding the lock before calling this function.
func (sm *SubscriptionManager) checkRLSAccess(ctx context.Context, sub *Subscription, event *ChangeEvent, claims map[string]interface{}) bool {
	if sm.db == nil {
		return true // No DB means no RLS check (test mode)
	}

	// Service role users bypass RLS
	if sub.Role == "service_role" {
		return true
	}

	// Get record ID from event
	recordID, ok := event.Record["id"]
	if !ok {
		// If no ID, check old_record for DELETE events
		if event.OldRecord != nil {
			recordID, ok = event.OldRecord["id"]
			if !ok {
				return false
			}
		} else {
			return false
		}
	}

	// Generate cache key and check cache first
	cacheKey := sm.rlsCache.generateCacheKey(sub.Schema, sub.Table, sub.Role, recordID, claims)
	if allowed, found := sm.rlsCache.get(cacheKey); found {
		log.Debug().
			Str("user_id", sub.UserID).
			Str("table", fmt.Sprintf("%s.%s", sub.Schema, sub.Table)).
			Interface("record_id", recordID).
			Bool("visible", allowed).
			Bool("cached", true).
			Msg("RLS access check (cached)")
		return allowed
	}

	// Cache miss - perform actual RLS check
	log.Debug().
		Str("user_id", sub.UserID).
		Str("role", sub.Role).
		Str("table", fmt.Sprintf("%s.%s", sub.Schema, sub.Table)).
		Interface("record_id", recordID).
		Interface("claims", copyClaims(claims)).
		Msg("Starting RLS access check")

	visible, err := sm.db.CheckRLSAccess(ctx, sub.Schema, sub.Table, sub.Role, claims, recordID)
	if err != nil {
		log.Error().
			Err(err).
			Str("table", fmt.Sprintf("%s.%s", sub.Schema, sub.Table)).
			Interface("record_id", recordID).
			Interface("claims", copyClaims(claims)).
			Msg("RLS check failed")
		return false
	}

	// Cache the result
	sm.rlsCache.set(cacheKey, visible)

	log.Debug().
		Str("user_id", sub.UserID).
		Str("table", fmt.Sprintf("%s.%s", sub.Schema, sub.Table)).
		Interface("record_id", recordID).
		Bool("visible", visible).
		Bool("cached", false).
		Msg("RLS access check completed")

	return visible
}

// isTableAllowedUnsafe checks if a table is allowed for realtime (must be called with lock held)
// It checks the realtime.schema_registry table to see if the table is enabled for realtime.
func (sm *SubscriptionManager) isTableAllowedUnsafe(schema, table string) bool {
	if sm.db == nil {
		return true // No DB means all tables allowed (test mode)
	}

	enabled, err := sm.db.IsTableRealtimeEnabled(context.Background(), schema, table)
	if err != nil {
		// Table not registered in schema_registry - not enabled for realtime
		return false
	}

	return enabled
}

// GetSubscriptionsByConnection returns all subscriptions for a connection
func (sm *SubscriptionManager) GetSubscriptionsByConnection(connID string) []*Subscription {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	subs := make([]*Subscription, 0)
	for _, sub := range sm.subscriptions {
		if sub.ConnID == connID {
			subs = append(subs, sub)
		}
	}

	return subs
}

// GetStats returns subscription statistics
func (sm *SubscriptionManager) GetStats() map[string]interface{} {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	return map[string]interface{}{
		"total_subscriptions": len(sm.subscriptions),
		"users_with_subs":     len(sm.userSubs),
		"tables_with_subs":    len(sm.tableSubs),
	}
}

// UpdateConnectionRole updates the role for all subscriptions belonging to a connection
func (sm *SubscriptionManager) UpdateConnectionRole(connID string, newRole string) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	for _, sub := range sm.subscriptions {
		if sub.ConnID == connID {
			sub.Role = newRole
		}
	}

	log.Info().
		Str("connection_id", connID).
		Str("new_role", newRole).
		Msg("Updated role for connection subscriptions")
}

// UpdateConnectionClaims updates the claims for all subscriptions belonging to a connection
func (sm *SubscriptionManager) UpdateConnectionClaims(connID string, newClaims map[string]interface{}) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	for _, sub := range sm.subscriptions {
		if sub.ConnID == connID {
			sub.Claims = newClaims
		}
	}

	log.Info().
		Str("connection_id", connID).
		Msg("Updated claims for connection subscriptions")
}

// ParseChangeEvent parses a JSON payload into a ChangeEvent
func ParseChangeEvent(payload string) (*ChangeEvent, error) {
	var event ChangeEvent
	if err := json.Unmarshal([]byte(payload), &event); err != nil {
		return nil, fmt.Errorf("failed to parse change event: %w", err)
	}
	return &event, nil
}

// CreateLogSubscription creates a subscription for execution logs
func (sm *SubscriptionManager) CreateLogSubscription(subID, connID, executionID, executionType string) (*LogSubscription, error) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	sub := &LogSubscription{
		ID:            subID,
		ConnID:        connID,
		ExecutionID:   executionID,
		ExecutionType: executionType,
	}

	// Store subscription
	sm.logSubs[subID] = sub

	// Track by execution ID
	if _, exists := sm.execLogSubs[executionID]; !exists {
		sm.execLogSubs[executionID] = make(map[string]bool)
	}
	sm.execLogSubs[executionID][subID] = true

	return sub, nil
}

// RemoveLogSubscription removes an execution log subscription
func (sm *SubscriptionManager) RemoveLogSubscription(subID string) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	sub, exists := sm.logSubs[subID]
	if !exists {
		return fmt.Errorf("log subscription not found")
	}

	// Remove from execution ID subscriptions
	if execSubs, exists := sm.execLogSubs[sub.ExecutionID]; exists {
		delete(execSubs, subID)
		if len(execSubs) == 0 {
			delete(sm.execLogSubs, sub.ExecutionID)
		}
	}

	delete(sm.logSubs, subID)

	log.Info().
		Str("sub_id", subID).
		Str("execution_id", sub.ExecutionID).
		Msg("Removed execution log subscription")

	return nil
}

// GetLogSubscribers returns all connection IDs subscribed to an execution's logs
func (sm *SubscriptionManager) GetLogSubscribers(executionID string) []string {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	subIDs, exists := sm.execLogSubs[executionID]
	if !exists {
		return nil
	}

	connIDs := make([]string, 0, len(subIDs))
	for subID := range subIDs {
		if sub, exists := sm.logSubs[subID]; exists {
			connIDs = append(connIDs, sub.ConnID)
		}
	}

	return connIDs
}

// RemoveConnectionLogSubscriptions removes all log subscriptions for a connection
func (sm *SubscriptionManager) RemoveConnectionLogSubscriptions(connID string) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	for subID, sub := range sm.logSubs {
		if sub.ConnID == connID {
			// Remove from execution ID subscriptions
			if execSubs, exists := sm.execLogSubs[sub.ExecutionID]; exists {
				delete(execSubs, subID)
				if len(execSubs) == 0 {
					delete(sm.execLogSubs, sub.ExecutionID)
				}
			}

			delete(sm.logSubs, subID)

			log.Info().
				Str("sub_id", subID).
				Str("execution_id", sub.ExecutionID).
				Msg("Removed execution log subscription")
		}
	}
}

// GetLogSubscriptionsByConnection returns all log subscriptions for a connection
func (sm *SubscriptionManager) GetLogSubscriptionsByConnection(connID string) []*LogSubscription {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	subs := make([]*LogSubscription, 0)
	for _, sub := range sm.logSubs {
		if sub.ConnID == connID {
			subs = append(subs, sub)
		}
	}

	return subs
}

// CreateAllLogsSubscription creates a subscription for all logs (admin only)
func (sm *SubscriptionManager) CreateAllLogsSubscription(subID, connID, category string, levels []string) (*AllLogsSubscription, error) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	sub := &AllLogsSubscription{
		ID:       subID,
		ConnID:   connID,
		Category: category,
		Levels:   levels,
	}

	// Store subscription
	sm.allLogsSubs[subID] = sub

	// Note: The handler logs this with more context (connection info)

	return sub, nil
}

// RemoveAllLogsSubscription removes an all-logs subscription
func (sm *SubscriptionManager) RemoveAllLogsSubscription(subID string) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if _, exists := sm.allLogsSubs[subID]; !exists {
		return fmt.Errorf("all-logs subscription not found")
	}

	delete(sm.allLogsSubs, subID)

	log.Info().
		Str("sub_id", subID).
		Msg("Removed all-logs subscription")

	return nil
}

// GetAllLogsSubscribers returns all connection IDs subscribed to all logs,
// along with their filter preferences
func (sm *SubscriptionManager) GetAllLogsSubscribers() map[string]*AllLogsSubscription {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	// Return map of connection ID -> subscription
	result := make(map[string]*AllLogsSubscription)
	for _, sub := range sm.allLogsSubs {
		result[sub.ConnID] = sub
	}

	return result
}

// RemoveConnectionAllLogsSubscriptions removes all all-logs subscriptions for a connection
func (sm *SubscriptionManager) RemoveConnectionAllLogsSubscriptions(connID string) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	for subID, sub := range sm.allLogsSubs {
		if sub.ConnID == connID {
			delete(sm.allLogsSubs, subID)

			log.Info().
				Str("sub_id", subID).
				Msg("Removed all-logs subscription")
		}
	}
}

// CheckExecutionOwnership verifies if a user owns the execution.
// Returns (isOwner, exists, error).
// executionType can be "rpc", "job", "function", or empty (will try all).
func (sm *SubscriptionManager) CheckExecutionOwnership(ctx context.Context, executionID, userID, executionType string) (isOwner bool, exists bool, err error) {
	execUUID, err := uuid.Parse(executionID)
	if err != nil {
		return false, false, fmt.Errorf("invalid execution ID: %w", err)
	}

	userUUID, err := uuid.Parse(userID)
	if err != nil {
		return false, false, fmt.Errorf("invalid user ID: %w", err)
	}

	// Check based on execution type
	switch executionType {
	case "rpc":
		return sm.checkRPCOwnership(ctx, execUUID, userUUID)
	case "job":
		return sm.checkJobOwnership(ctx, execUUID, userUUID)
	case "function":
		return sm.checkFunctionOwnership(ctx, execUUID, userUUID)
	default:
		// Unknown type - try all tables
		return sm.checkAnyExecution(ctx, execUUID, userUUID)
	}
}

func (sm *SubscriptionManager) checkRPCOwnership(ctx context.Context, execID, userID uuid.UUID) (bool, bool, error) {
	if sm.db == nil {
		return true, true, nil // No DB means allow all (test mode)
	}
	return sm.db.CheckRPCOwnership(ctx, execID, userID)
}

func (sm *SubscriptionManager) checkJobOwnership(ctx context.Context, execID, userID uuid.UUID) (bool, bool, error) {
	if sm.db == nil {
		return true, true, nil // No DB means allow all (test mode)
	}
	return sm.db.CheckJobOwnership(ctx, execID, userID)
}

func (sm *SubscriptionManager) checkFunctionOwnership(ctx context.Context, execID, userID uuid.UUID) (bool, bool, error) {
	if sm.db == nil {
		return true, true, nil // No DB means allow all (test mode)
	}
	return sm.db.CheckFunctionOwnership(ctx, execID, userID)
}

func (sm *SubscriptionManager) checkAnyExecution(ctx context.Context, execID, userID uuid.UUID) (bool, bool, error) {
	// Try RPC first
	isOwner, exists, err := sm.checkRPCOwnership(ctx, execID, userID)
	if err != nil || exists {
		return isOwner, exists, err
	}
	// Try jobs
	isOwner, exists, err = sm.checkJobOwnership(ctx, execID, userID)
	if err != nil || exists {
		return isOwner, exists, err
	}
	// Try functions
	return sm.checkFunctionOwnership(ctx, execID, userID)
}

func (sm *SubscriptionManager) Close() {
	if sm.rlsCache != nil {
		sm.rlsCache.stop()
	}
}
