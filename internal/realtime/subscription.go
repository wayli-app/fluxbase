package realtime

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog/log"
)

// validIdentifierRegex ensures identifier names are safe PostgreSQL identifiers
var validIdentifierRegex = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*$`)

// quoteIdentifier safely quotes a PostgreSQL identifier to prevent SQL injection.
func quoteIdentifier(identifier string) string {
	return `"` + strings.ReplaceAll(identifier, `"`, `""`) + `"`
}

// isValidIdentifier checks if a string is a valid PostgreSQL identifier
func isValidIdentifier(s string) bool {
	return validIdentifierRegex.MatchString(s)
}

// Default RLS cache settings (used when no config provided)
const (
	DefaultRLSCacheTTL     = 30 * time.Second // 30 seconds default
	DefaultRLSCacheMaxSize = 100000           // 100K entries default
)

// RLSCacheConfig holds configuration for the RLS cache
type RLSCacheConfig struct {
	MaxSize int           // Maximum number of entries (0 = use default)
	TTL     time.Duration // Cache entry TTL (0 = use default)
}

// rlsCacheEntry represents a cached RLS check result
type rlsCacheEntry struct {
	allowed   bool
	expiresAt time.Time
}

// rlsCache provides a simple time-based cache for RLS check results
type rlsCache struct {
	mu      sync.RWMutex
	entries map[string]*rlsCacheEntry
	maxSize int
	ttl     time.Duration
}

// newRLSCache creates a new RLS cache with default settings
func newRLSCache() *rlsCache {
	return newRLSCacheWithConfig(RLSCacheConfig{})
}

// newRLSCacheWithConfig creates a new RLS cache with custom configuration
func newRLSCacheWithConfig(config RLSCacheConfig) *rlsCache {
	maxSize := config.MaxSize
	if maxSize <= 0 {
		maxSize = DefaultRLSCacheMaxSize
	}

	ttl := config.TTL
	if ttl <= 0 {
		ttl = DefaultRLSCacheTTL
	}

	cache := &rlsCache{
		entries: make(map[string]*rlsCacheEntry),
		maxSize: maxSize,
		ttl:     ttl,
	}
	// Start cleanup goroutine
	go cache.cleanup()
	return cache
}

// generateCacheKey creates a unique cache key for an RLS check
func (c *rlsCache) generateCacheKey(schema, table, role string, recordID interface{}, claims map[string]interface{}) string {
	// Create a deterministic key from all parameters
	data := fmt.Sprintf("%s:%s:%s:%v", schema, table, role, recordID)
	// Include a hash of the claims to handle custom claims
	if claims != nil {
		claimsJSON, _ := json.Marshal(claims)
		hash := sha256.Sum256(claimsJSON)
		data += ":" + hex.EncodeToString(hash[:8]) // Use first 8 bytes of hash for brevity
	}
	return data
}

// get retrieves a cached result, returning (allowed, found)
func (c *rlsCache) get(key string) (bool, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	entry, exists := c.entries[key]
	if !exists {
		return false, false
	}

	if time.Now().After(entry.expiresAt) {
		return false, false // Entry expired
	}

	return entry.allowed, true
}

// set stores a result in the cache
func (c *rlsCache) set(key string, allowed bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Evict old entries if cache is too large
	if len(c.entries) >= c.maxSize {
		c.evictExpiredLocked()
	}

	c.entries[key] = &rlsCacheEntry{
		allowed:   allowed,
		expiresAt: time.Now().Add(c.ttl),
	}
}

// evictExpiredLocked removes expired entries (must be called with lock held)
func (c *rlsCache) evictExpiredLocked() {
	now := time.Now()
	for key, entry := range c.entries {
		if now.After(entry.expiresAt) {
			delete(c.entries, key)
		}
	}
}

// cleanup periodically removes expired entries
func (c *rlsCache) cleanup() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		c.mu.Lock()
		c.evictExpiredLocked()
		c.mu.Unlock()
	}
}

// SubscriptionDB defines the database operations needed by SubscriptionManager.
// This interface allows for easier testing with mocks.
type SubscriptionDB interface {
	// IsTableRealtimeEnabled checks if a table is enabled for realtime in the schema registry.
	IsTableRealtimeEnabled(ctx context.Context, schema, table string) (bool, error)
	// CheckRLSAccess verifies if a user can access a record based on RLS policies.
	// The claims map contains the full JWT claims to be passed to PostgreSQL for RLS evaluation.
	CheckRLSAccess(ctx context.Context, schema, table, role string, claims map[string]interface{}, recordID interface{}) (bool, error)
	// CheckRPCOwnership checks if a user owns an RPC execution.
	CheckRPCOwnership(ctx context.Context, execID, userID uuid.UUID) (isOwner bool, exists bool, err error)
	// CheckJobOwnership checks if a user owns a job execution.
	CheckJobOwnership(ctx context.Context, execID, userID uuid.UUID) (isOwner bool, exists bool, err error)
	// CheckFunctionOwnership checks if a user owns a function execution.
	CheckFunctionOwnership(ctx context.Context, execID, userID uuid.UUID) (isOwner bool, exists bool, err error)
}

// pgxSubscriptionDB implements SubscriptionDB using a pgxpool.Pool.
type pgxSubscriptionDB struct {
	pool *pgxpool.Pool
}

// NewPgxSubscriptionDB creates a SubscriptionDB backed by a pgx pool.
func NewPgxSubscriptionDB(pool *pgxpool.Pool) SubscriptionDB {
	return &pgxSubscriptionDB{pool: pool}
}

func (db *pgxSubscriptionDB) IsTableRealtimeEnabled(ctx context.Context, schema, table string) (bool, error) {
	var enabled bool
	err := db.pool.QueryRow(ctx, `
		SELECT realtime_enabled FROM realtime.schema_registry
		WHERE schema_name = $1 AND table_name = $2
	`, schema, table).Scan(&enabled)
	if err == pgx.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return enabled, nil
}

func (db *pgxSubscriptionDB) CheckRLSAccess(ctx context.Context, schema, table, role string, claims map[string]interface{}, recordID interface{}) (bool, error) {
	// Validate schema and table names to prevent SQL injection
	if !isValidIdentifier(schema) {
		return false, fmt.Errorf("invalid schema name: %s", schema)
	}
	if !isValidIdentifier(table) {
		return false, fmt.Errorf("invalid table name: %s", table)
	}

	conn, err := db.pool.Acquire(ctx)
	if err != nil {
		return false, err
	}
	defer conn.Release()

	// Start a transaction for SET LOCAL (required by PostgreSQL)
	tx, err := conn.Begin(ctx)
	if err != nil {
		return false, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	// Use provided claims, ensuring role is set
	jwtClaims := claims
	if jwtClaims == nil {
		jwtClaims = make(map[string]interface{})
	}
	// Ensure role is set in claims for RLS policies that use it
	jwtClaims["role"] = role

	jwtClaimsJSON, err := json.Marshal(jwtClaims)
	if err != nil {
		return false, err
	}

	// Map application role to database role (hardcoded values - safe)
	// Using quoteIdentifier for defense in depth
	dbRole := "authenticated"
	switch role {
	case "service_role":
		dbRole = "service_role"
	case "anon", "":
		dbRole = "anon"
	}

	_, err = tx.Exec(ctx, fmt.Sprintf("SET LOCAL ROLE %s", quoteIdentifier(dbRole)))
	if err != nil {
		return false, err
	}

	_, err = tx.Exec(ctx, "SELECT set_config('request.jwt.claims', $1, true)", string(jwtClaimsJSON))
	if err != nil {
		return false, err
	}

	var count int
	// Use quoteIdentifier to prevent SQL injection even though we validated above
	query := fmt.Sprintf("SELECT COUNT(*) FROM %s.%s WHERE id = $1", quoteIdentifier(schema), quoteIdentifier(table))
	err = tx.QueryRow(ctx, query, recordID).Scan(&count)
	if err != nil {
		return false, err
	}

	return count > 0, nil
}

func (db *pgxSubscriptionDB) CheckRPCOwnership(ctx context.Context, execID, userID uuid.UUID) (bool, bool, error) {
	var ownerID *uuid.UUID
	err := db.pool.QueryRow(ctx, "SELECT user_id FROM rpc.executions WHERE id = $1", execID).Scan(&ownerID)
	if err == pgx.ErrNoRows {
		return false, false, nil
	}
	if err != nil {
		return false, false, err
	}
	if ownerID == nil {
		return true, true, nil
	}
	return *ownerID == userID, true, nil
}

func (db *pgxSubscriptionDB) CheckJobOwnership(ctx context.Context, execID, userID uuid.UUID) (bool, bool, error) {
	var ownerID *uuid.UUID
	err := db.pool.QueryRow(ctx, "SELECT created_by FROM jobs.queue WHERE id = $1", execID).Scan(&ownerID)
	if err == pgx.ErrNoRows {
		return false, false, nil
	}
	if err != nil {
		return false, false, err
	}
	if ownerID == nil {
		return true, true, nil
	}
	return *ownerID == userID, true, nil
}

func (db *pgxSubscriptionDB) CheckFunctionOwnership(ctx context.Context, execID, userID uuid.UUID) (bool, bool, error) {
	var ownerID *uuid.UUID
	err := db.pool.QueryRow(ctx, `
		SELECT ef.created_by
		FROM functions.edge_executions ee
		JOIN functions.edge_functions ef ON ee.function_id = ef.id
		WHERE ee.id = $1
	`, execID).Scan(&ownerID)
	if err == pgx.ErrNoRows {
		return false, false, nil
	}
	if err != nil {
		return false, false, err
	}
	if ownerID == nil {
		return true, true, nil
	}
	return *ownerID == userID, true, nil
}

// Subscription represents an RLS-aware subscription to table changes
type Subscription struct {
	ID     string
	UserID string
	Role   string
	Claims map[string]interface{} // Full JWT claims for RLS (includes custom claims like meeting_id, player_id)
	Table  string
	Schema string
	Event  string  // INSERT, UPDATE, DELETE, or * for all
	Filter *Filter // Supabase-compatible filter (column=operator.value)
	ConnID string  // Connection ID this subscription belongs to
}

// copyClaims creates a shallow copy of claims map to prevent concurrent map access during logging.
// This is necessary because zerolog's Interface() iterates over the map, which can race with
// concurrent modifications to the claims map from other goroutines.
func copyClaims(claims map[string]interface{}) map[string]interface{} {
	if claims == nil {
		return nil
	}
	copied := make(map[string]interface{}, len(claims))
	for k, v := range claims {
		copied[k] = v
	}
	return copied
}

// SubscriptionFilter represents filters for a subscription
type SubscriptionFilter struct {
	Column   string      `json:"column"`
	Operator string      `json:"operator"` // eq, neq, gt, lt, gte, lte, in
	Value    interface{} `json:"value"`
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

	// No filter specified - match all
	return true
}

// checkRLSAccess verifies if a user can access a record based on RLS policies
// Uses a short-lived cache to reduce database load for repeated checks
func (sm *SubscriptionManager) checkRLSAccess(ctx context.Context, sub *Subscription, event *ChangeEvent) bool {
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
	cacheKey := sm.rlsCache.generateCacheKey(sub.Schema, sub.Table, sub.Role, recordID, sub.Claims)
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
		Interface("claims", copyClaims(sub.Claims)).
		Msg("Starting RLS access check")

	visible, err := sm.db.CheckRLSAccess(ctx, sub.Schema, sub.Table, sub.Role, sub.Claims, recordID)
	if err != nil {
		log.Error().
			Err(err).
			Str("table", fmt.Sprintf("%s.%s", sub.Schema, sub.Table)).
			Interface("record_id", recordID).
			Interface("claims", copyClaims(sub.Claims)).
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

	subsToRemove := make([]string, 0)
	for subID, sub := range sm.logSubs {
		if sub.ConnID == connID {
			subsToRemove = append(subsToRemove, subID)
		}
	}

	sm.mu.Unlock()

	for _, subID := range subsToRemove {
		_ = sm.RemoveLogSubscription(subID)
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

	subsToRemove := make([]string, 0)
	for subID, sub := range sm.allLogsSubs {
		if sub.ConnID == connID {
			subsToRemove = append(subsToRemove, subID)
		}
	}

	sm.mu.Unlock()

	for _, subID := range subsToRemove {
		_ = sm.RemoveAllLogsSubscription(subID)
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
