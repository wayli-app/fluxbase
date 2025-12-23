package database

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/fluxbase-eu/fluxbase/internal/pubsub"
	"github.com/rs/zerolog/log"
)

// SchemaCache provides a thread-safe cache for database schema information
// with TTL-based expiration and manual invalidation support.
// When PubSub is configured, invalidation is broadcast to all instances.
type SchemaCache struct {
	mu          sync.RWMutex
	tables      map[string]*TableInfo // key: "schema.table"
	views       map[string]*TableInfo
	matViews    map[string]*TableInfo
	allTables   []TableInfo
	allViews    []TableInfo
	allMatViews []TableInfo
	ttl         time.Duration
	lastRefresh time.Time
	inspector   *SchemaInspector
	stale       bool // Force refresh on next access
	schemas     []string

	// PubSub for cross-instance cache invalidation
	ps         pubsub.PubSub
	ctx        context.Context
	cancelFunc context.CancelFunc
}

// NewSchemaCache creates a new schema cache with the given TTL
func NewSchemaCache(inspector *SchemaInspector, ttl time.Duration) *SchemaCache {
	return &SchemaCache{
		tables:    make(map[string]*TableInfo),
		views:     make(map[string]*TableInfo),
		matViews:  make(map[string]*TableInfo),
		ttl:       ttl,
		inspector: inspector,
		stale:     true, // Start stale to force initial load
		schemas:   []string{},
	}
}

// makeKey creates a cache key from schema and table name
func makeKey(schema, table string) string {
	return fmt.Sprintf("%s.%s", schema, table)
}

// isExpired checks if the cache has expired based on TTL
func (c *SchemaCache) isExpired() bool {
	return time.Since(c.lastRefresh) > c.ttl
}

// needsRefresh checks if the cache needs to be refreshed
func (c *SchemaCache) needsRefresh() bool {
	return c.stale || c.isExpired()
}

// GetTable retrieves table info from the cache, refreshing if necessary.
// Returns (TableInfo, exists, error)
func (c *SchemaCache) GetTable(ctx context.Context, schema, table string) (*TableInfo, bool, error) {
	// First try to get from cache with read lock
	c.mu.RLock()
	if !c.needsRefresh() {
		key := makeKey(schema, table)
		if info, ok := c.tables[key]; ok {
			c.mu.RUnlock()
			return info, true, nil
		}
		// Also check views and materialized views
		if info, ok := c.views[key]; ok {
			c.mu.RUnlock()
			return info, true, nil
		}
		if info, ok := c.matViews[key]; ok {
			c.mu.RUnlock()
			return info, true, nil
		}
		c.mu.RUnlock()
		return nil, false, nil
	}
	c.mu.RUnlock()

	// Cache needs refresh - do it with write lock
	if err := c.Refresh(ctx); err != nil {
		return nil, false, err
	}

	// Now try again with read lock
	c.mu.RLock()
	defer c.mu.RUnlock()

	key := makeKey(schema, table)
	if info, ok := c.tables[key]; ok {
		return info, true, nil
	}
	if info, ok := c.views[key]; ok {
		return info, true, nil
	}
	if info, ok := c.matViews[key]; ok {
		return info, true, nil
	}

	return nil, false, nil
}

// GetAllTables returns all cached tables, refreshing if necessary
func (c *SchemaCache) GetAllTables(ctx context.Context) ([]TableInfo, error) {
	c.mu.RLock()
	if !c.needsRefresh() {
		result := make([]TableInfo, len(c.allTables))
		copy(result, c.allTables)
		c.mu.RUnlock()
		return result, nil
	}
	c.mu.RUnlock()

	if err := c.Refresh(ctx); err != nil {
		return nil, err
	}

	c.mu.RLock()
	defer c.mu.RUnlock()
	result := make([]TableInfo, len(c.allTables))
	copy(result, c.allTables)
	return result, nil
}

// GetAllViews returns all cached views, refreshing if necessary
func (c *SchemaCache) GetAllViews(ctx context.Context) ([]TableInfo, error) {
	c.mu.RLock()
	if !c.needsRefresh() {
		result := make([]TableInfo, len(c.allViews))
		copy(result, c.allViews)
		c.mu.RUnlock()
		return result, nil
	}
	c.mu.RUnlock()

	if err := c.Refresh(ctx); err != nil {
		return nil, err
	}

	c.mu.RLock()
	defer c.mu.RUnlock()
	result := make([]TableInfo, len(c.allViews))
	copy(result, c.allViews)
	return result, nil
}

// GetAllMaterializedViews returns all cached materialized views, refreshing if necessary
func (c *SchemaCache) GetAllMaterializedViews(ctx context.Context) ([]TableInfo, error) {
	c.mu.RLock()
	if !c.needsRefresh() {
		result := make([]TableInfo, len(c.allMatViews))
		copy(result, c.allMatViews)
		c.mu.RUnlock()
		return result, nil
	}
	c.mu.RUnlock()

	if err := c.Refresh(ctx); err != nil {
		return nil, err
	}

	c.mu.RLock()
	defer c.mu.RUnlock()
	result := make([]TableInfo, len(c.allMatViews))
	copy(result, c.allMatViews)
	return result, nil
}

// GetSchemas returns cached schemas
func (c *SchemaCache) GetSchemas(ctx context.Context) ([]string, error) {
	c.mu.RLock()
	if !c.needsRefresh() {
		result := make([]string, len(c.schemas))
		copy(result, c.schemas)
		c.mu.RUnlock()
		return result, nil
	}
	c.mu.RUnlock()

	if err := c.Refresh(ctx); err != nil {
		return nil, err
	}

	c.mu.RLock()
	defer c.mu.RUnlock()
	result := make([]string, len(c.schemas))
	copy(result, c.schemas)
	return result, nil
}

// Invalidate marks the cache as stale, forcing a refresh on next access.
// This only invalidates the local cache. Use InvalidateAll to broadcast
// invalidation to all instances.
func (c *SchemaCache) Invalidate() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.stale = true
	log.Debug().Msg("Schema cache invalidated (local)")
}

// InvalidateAll marks the cache as stale and broadcasts the invalidation
// to all other instances via PubSub. Use this when schema changes occur
// (e.g., after migrations) to ensure all instances refresh their caches.
func (c *SchemaCache) InvalidateAll(ctx context.Context) {
	// First invalidate locally
	c.Invalidate()

	// Then broadcast to other instances if PubSub is configured
	if c.ps != nil {
		if err := c.ps.Publish(ctx, pubsub.SchemaCacheChannel, []byte("invalidate")); err != nil {
			log.Error().Err(err).Msg("Failed to broadcast schema cache invalidation")
		} else {
			log.Debug().Msg("Schema cache invalidation broadcast sent")
		}
	}
}

// SetPubSub configures the PubSub backend for cross-instance cache invalidation.
// When set, InvalidateAll will broadcast invalidation messages to all instances,
// and this instance will listen for invalidation messages from others.
func (c *SchemaCache) SetPubSub(ps pubsub.PubSub) {
	c.mu.Lock()
	c.ps = ps
	c.mu.Unlock()

	if ps != nil {
		c.startInvalidationListener()
	}
}

// startInvalidationListener subscribes to schema cache invalidation messages
// from other instances and invalidates the local cache when received.
func (c *SchemaCache) startInvalidationListener() {
	c.mu.Lock()
	// Cancel any existing listener
	if c.cancelFunc != nil {
		c.cancelFunc()
	}
	c.ctx, c.cancelFunc = context.WithCancel(context.Background())
	ctx := c.ctx
	ps := c.ps
	c.mu.Unlock()

	go func() {
		msgCh, err := ps.Subscribe(ctx, pubsub.SchemaCacheChannel)
		if err != nil {
			log.Error().Err(err).Msg("Failed to subscribe to schema cache invalidation channel")
			return
		}

		log.Info().Msg("Schema cache listening for cross-instance invalidation messages")

		for {
			select {
			case <-ctx.Done():
				log.Debug().Msg("Schema cache invalidation listener stopped")
				return
			case msg, ok := <-msgCh:
				if !ok {
					log.Debug().Msg("Schema cache invalidation channel closed")
					return
				}
				log.Debug().Str("payload", string(msg.Payload)).Msg("Received schema cache invalidation from another instance")
				c.Invalidate()
			}
		}
	}()
}

// Close stops the invalidation listener if running
func (c *SchemaCache) Close() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.cancelFunc != nil {
		c.cancelFunc()
		c.cancelFunc = nil
	}
}

// Refresh forces an immediate cache refresh
func (c *SchemaCache) Refresh(ctx context.Context) error {
	// Fetch all data without holding the lock
	schemas, err := c.inspector.GetSchemas(ctx)
	if err != nil {
		return fmt.Errorf("failed to get schemas: %w", err)
	}

	// Filter out system schemas
	var userSchemas []string
	for _, schema := range schemas {
		if schema != "information_schema" && schema != "pg_catalog" && schema != "pg_toast" && schema != "_fluxbase" {
			userSchemas = append(userSchemas, schema)
		}
	}

	// Collect all tables, views, and materialized views
	newTables := make(map[string]*TableInfo)
	newViews := make(map[string]*TableInfo)
	newMatViews := make(map[string]*TableInfo)
	var allTables []TableInfo
	var allViews []TableInfo
	var allMatViews []TableInfo

	for _, schema := range userSchemas {
		// Get tables
		tables, err := c.inspector.GetAllTables(ctx, schema)
		if err != nil {
			log.Warn().Err(err).Str("schema", schema).Msg("Failed to get tables from schema")
			continue
		}
		for i := range tables {
			table := tables[i]
			key := makeKey(table.Schema, table.Name)
			newTables[key] = &table
			allTables = append(allTables, table)
		}

		// Get views
		views, err := c.inspector.GetAllViews(ctx, schema)
		if err != nil {
			log.Warn().Err(err).Str("schema", schema).Msg("Failed to get views from schema")
		} else {
			for i := range views {
				view := views[i]
				key := makeKey(view.Schema, view.Name)
				newViews[key] = &view
				allViews = append(allViews, view)
			}
		}

		// Get materialized views
		matViews, err := c.inspector.GetAllMaterializedViews(ctx, schema)
		if err != nil {
			log.Warn().Err(err).Str("schema", schema).Msg("Failed to get materialized views from schema")
		} else {
			for i := range matViews {
				matView := matViews[i]
				key := makeKey(matView.Schema, matView.Name)
				newMatViews[key] = &matView
				allMatViews = append(allMatViews, matView)
			}
		}
	}

	// Atomically swap the cache
	c.mu.Lock()
	defer c.mu.Unlock()

	c.tables = newTables
	c.views = newViews
	c.matViews = newMatViews
	c.allTables = allTables
	c.allViews = allViews
	c.allMatViews = allMatViews
	c.schemas = userSchemas
	c.lastRefresh = time.Now()
	c.stale = false

	log.Debug().
		Int("tables", len(allTables)).
		Int("views", len(allViews)).
		Int("matViews", len(allMatViews)).
		Int("schemas", len(userSchemas)).
		Msg("Schema cache refreshed")

	return nil
}

// TableCount returns the number of cached tables
func (c *SchemaCache) TableCount() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.tables)
}

// ViewCount returns the number of cached views
func (c *SchemaCache) ViewCount() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.views)
}

// IsTableWritable checks if a table is writable (not a view or materialized view)
func (c *SchemaCache) IsTableWritable(ctx context.Context, schema, table string) (bool, error) {
	c.mu.RLock()
	if !c.needsRefresh() {
		key := makeKey(schema, table)
		// Check if it's a regular table (writable)
		if _, ok := c.tables[key]; ok {
			c.mu.RUnlock()
			return true, nil
		}
		// Check if it's a view or materialized view (read-only)
		if _, ok := c.views[key]; ok {
			c.mu.RUnlock()
			return false, nil
		}
		if _, ok := c.matViews[key]; ok {
			c.mu.RUnlock()
			return false, nil
		}
		c.mu.RUnlock()
		return false, nil // Not found
	}
	c.mu.RUnlock()

	// Refresh and try again
	if err := c.Refresh(ctx); err != nil {
		return false, err
	}

	c.mu.RLock()
	defer c.mu.RUnlock()

	key := makeKey(schema, table)
	if _, ok := c.tables[key]; ok {
		return true, nil
	}
	return false, nil
}
