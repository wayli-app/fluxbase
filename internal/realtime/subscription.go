package realtime

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog/log"
)

// Subscription represents an RLS-aware subscription to table changes
type Subscription struct {
	ID         string
	UserID     string
	Role       string
	Table      string
	Schema     string
	Event      string  // INSERT, UPDATE, DELETE, or * for all
	Filter     *Filter // Supabase-compatible filter (column=operator.value)
	OldFilters map[string]interface{} // Legacy simple filters (deprecated)
	ConnID     string // Connection ID this subscription belongs to
}

// SubscriptionFilter represents filters for a subscription
type SubscriptionFilter struct {
	Column   string      `json:"column"`
	Operator string      `json:"operator"` // eq, neq, gt, lt, gte, lte, in
	Value    interface{} `json:"value"`
}

// SubscriptionManager manages RLS-aware subscriptions
type SubscriptionManager struct {
	db            *pgxpool.Pool
	subscriptions map[string]*Subscription   // subscription ID -> subscription
	userSubs      map[string]map[string]bool // user ID -> subscription IDs
	tableSubs     map[string]map[string]bool // "schema.table" -> subscription IDs
	mu            sync.RWMutex
}

// NewSubscriptionManager creates a new subscription manager
func NewSubscriptionManager(db *pgxpool.Pool) *SubscriptionManager {
	return &SubscriptionManager{
		db:            db,
		subscriptions: make(map[string]*Subscription),
		userSubs:      make(map[string]map[string]bool),
		tableSubs:     make(map[string]map[string]bool),
	}
}

// CreateSubscription creates a new RLS-aware subscription
func (sm *SubscriptionManager) CreateSubscription(
	subID string,
	connID string,
	userID string,
	role string,
	schema string,
	table string,
	event string,
	filterStr string,
	legacyFilters map[string]interface{},
) (*Subscription, error) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	// Validate table exists and is allowed for realtime
	if !sm.isTableAllowedUnsafe(schema, table) {
		return nil, fmt.Errorf("table %s.%s not enabled for realtime", schema, table)
	}

	// Parse Supabase-compatible filter
	filter, err := ParseFilter(filterStr)
	if err != nil {
		return nil, fmt.Errorf("invalid filter: %w", err)
	}

	// Default event to "*" (all events)
	if event == "" {
		event = "*"
	}

	sub := &Subscription{
		ID:         subID,
		UserID:     userID,
		Role:       role,
		Table:      table,
		Schema:     schema,
		Event:      event,
		Filter:     filter,
		OldFilters: legacyFilters,
		ConnID:     connID,
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

	log.Info().
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

	subsToRemove := make([]string, 0)
	for subID, sub := range sm.subscriptions {
		if sub.ConnID == connID {
			subsToRemove = append(subsToRemove, subID)
		}
	}

	sm.mu.Unlock()
	for _, subID := range subsToRemove {
		_ = sm.RemoveSubscription(subID)
	}
	sm.mu.Lock()
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
		sm.mu.RUnlock()

		if !exists {
			continue
		}

		// Check if event type matches subscription
		if !sm.matchesEvent(event.Type, sub.Event) {
			continue
		}

		// Check RLS access
		if sm.checkRLSAccess(ctx, sub, event) {
			// Check Supabase-compatible filter
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
	// Use new Supabase-compatible filter if present
	if sub.Filter != nil {
		record := event.Record
		if record == nil {
			record = event.OldRecord
		}
		return sub.Filter.Matches(record)
	}

	// Fall back to legacy filters for backwards compatibility
	return sm.matchesFilters(event, sub.OldFilters)
}

// checkRLSAccess verifies if a user can access a record based on RLS policies
func (sm *SubscriptionManager) checkRLSAccess(ctx context.Context, sub *Subscription, event *ChangeEvent) bool {
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

	// Acquire connection from pool
	conn, err := sm.db.Acquire(ctx)
	if err != nil {
		log.Error().Err(err).Msg("Failed to acquire connection for RLS check")
		return false
	}
	defer conn.Release()

	// Set RLS session variables using parameterized set_config
	batch := &pgx.Batch{}
	batch.Queue("SELECT set_config('app.user_id', $1, true)", sub.UserID)
	batch.Queue("SELECT set_config('app.role', $1, true)", sub.Role)

	br := conn.SendBatch(ctx, batch)
	for i := 0; i < batch.Len(); i++ {
		if _, err := br.Exec(); err != nil {
			br.Close()
			log.Error().Err(err).Msg("Failed to set RLS session variables")
			return false
		}
	}
	br.Close()

	// Check if record is visible with RLS
	var count int
	query := fmt.Sprintf("SELECT COUNT(*) FROM %s.%s WHERE id = $1", sub.Schema, sub.Table)
	err = conn.QueryRow(ctx, query, recordID).Scan(&count)

	if err != nil {
		log.Error().
			Err(err).
			Str("table", fmt.Sprintf("%s.%s", sub.Schema, sub.Table)).
			Interface("record_id", recordID).
			Msg("RLS check query failed")
		return false
	}

	visible := count > 0

	log.Debug().
		Str("user_id", sub.UserID).
		Str("table", fmt.Sprintf("%s.%s", sub.Schema, sub.Table)).
		Interface("record_id", recordID).
		Bool("visible", visible).
		Msg("RLS access check")

	return visible
}

// matchesFilters checks if an event matches subscription filters
func (sm *SubscriptionManager) matchesFilters(event *ChangeEvent, filters map[string]interface{}) bool {
	if len(filters) == 0 {
		return true
	}

	// Check each filter against the record
	record := event.Record
	if record == nil {
		record = event.OldRecord
	}

	for key, value := range filters {
		eventValue, exists := record[key]
		if !exists {
			return false
		}

		// Simple equality check for now
		// Could be extended to support operators
		if !sm.valuesEqual(eventValue, value) {
			return false
		}
	}

	return true
}

// valuesEqual compares two values for equality
func (sm *SubscriptionManager) valuesEqual(a, b interface{}) bool {
	// Handle JSON number comparisons
	switch v := a.(type) {
	case float64:
		switch bv := b.(type) {
		case float64:
			return v == bv
		case int:
			return v == float64(bv)
		case int64:
			return v == float64(bv)
		}
	case int:
		switch bv := b.(type) {
		case float64:
			return float64(v) == bv
		case int:
			return v == bv
		case int64:
			return int64(v) == bv
		}
	case int64:
		switch bv := b.(type) {
		case float64:
			return float64(v) == bv
		case int:
			return v == int64(bv)
		case int64:
			return v == bv
		}
	case string:
		if bv, ok := b.(string); ok {
			return v == bv
		}
	case bool:
		if bv, ok := b.(bool); ok {
			return v == bv
		}
	}

	// Default to simple equality
	return a == b
}

// isTableAllowedUnsafe checks if a table is allowed for realtime (must be called with lock held)
func (sm *SubscriptionManager) isTableAllowedUnsafe(schema, table string) bool {
	// For now, allow all tables in auth and public schemas
	// In production, this should check a configuration table
	switch schema {
	case "auth", "public":
		return true
	default:
		return false
	}
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

// ParseChangeEvent parses a JSON payload into a ChangeEvent
func ParseChangeEvent(payload string) (*ChangeEvent, error) {
	var event ChangeEvent
	if err := json.Unmarshal([]byte(payload), &event); err != nil {
		return nil, fmt.Errorf("failed to parse change event: %w", err)
	}
	return &event, nil
}
