package tenantdb

import (
	"container/list"
	"context"
	"errors"
	"fmt"
	"net/url"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog/log"
)

var (
	ErrPoolNotFound   = errors.New("pool not found")
	ErrTenantDeleting = errors.New("tenant is being deleted")
)

type Router struct {
	storage   *Repository
	config    Config
	mainPool  *pgxpool.Pool
	adminPool *pgxpool.Pool
	manager   *Manager
	dbURL     string

	mu         sync.RWMutex
	pools      map[string]*poolEntry
	lruList    *list.List
	totalConns int32
}

type poolEntry struct {
	pool     *pgxpool.Pool
	dbName   string
	tenantID string
	lastUsed time.Time
	element  *list.Element
}

func NewRouter(
	storage *Repository,
	config Config,
	mainPool *pgxpool.Pool,
	adminPool *pgxpool.Pool,
	dbURL string,
) *Router {
	return &Router{
		storage:   storage,
		config:    config,
		mainPool:  mainPool,
		adminPool: adminPool,
		dbURL:     dbURL,
		pools:     make(map[string]*poolEntry),
		lruList:   list.New(),
	}
}

func (r *Router) SetManager(manager *Manager) {
	r.manager = manager
}

func (r *Router) GetPool(tenantID string) (*pgxpool.Pool, error) {
	tenant, err := r.storage.GetTenant(context.Background(), tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to get tenant: %w", err)
	}

	if tenant.Status == TenantStatusDeleting {
		return nil, ErrTenantDeleting
	}

	if tenant.UsesMainDatabase() {
		log.Debug().Str("tenant_id", tenantID).Msg("Using main database pool (default tenant)")
		return r.mainPool, nil
	}

	return r.getOrCreatePool(tenant.ID, *tenant.DBName)
}

func (r *Router) getOrCreatePool(tenantID, dbName string) (*pgxpool.Pool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if entry, ok := r.pools[tenantID]; ok {
		entry.lastUsed = time.Now()
		r.lruList.MoveToFront(entry.element)
		return entry.pool, nil
	}

	if r.config.Pool.MaxTotalConnections > 0 && r.totalConns >= r.config.Pool.MaxTotalConnections {
		if err := r.evictLRU(); err != nil {
			log.Warn().Err(err).Msg("Failed to evict LRU pool")
		}
	}

	pool, err := r.createPool(dbName)
	if err != nil {
		return nil, fmt.Errorf("failed to create pool for %s: %w", dbName, err)
	}

	entry := &poolEntry{
		pool:     pool,
		dbName:   dbName,
		tenantID: tenantID,
		lastUsed: time.Now(),
	}
	entry.element = r.lruList.PushFront(tenantID)
	r.pools[tenantID] = entry

	stats := pool.Stat()
	r.totalConns += int32(stats.AcquiredConns())

	log.Info().Str("tenant_id", tenantID).Str("db", dbName).Msg("Created tenant pool")

	if r.config.Migrations.OnAccess && r.manager != nil {
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			if pending, _ := r.manager.hasPendingMigrations(ctx, pool); pending {
				if err := r.manager.runSystemMigrationsForDB(ctx, dbName); err != nil {
					log.Error().Err(err).Str("db", dbName).Msg("Lazy migration failed")
				} else {
					log.Info().Str("db", dbName).Msg("Lazy migration completed")
				}
			}
		}()
	}

	return pool, nil
}

func (r *Router) createPool(dbName string) (*pgxpool.Pool, error) {
	u, err := url.Parse(r.dbURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse database URL: %w", err)
	}

	u.Path = "/" + dbName
	dbURL := u.String()

	poolConfig, err := pgxpool.ParseConfig(dbURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse pool config: %w", err)
	}

	poolConfig.MaxConns = 10
	poolConfig.MinConns = 2
	poolConfig.MaxConnLifetime = 1 * time.Hour
	poolConfig.MaxConnIdleTime = 30 * time.Minute

	return pgxpool.NewWithConfig(context.Background(), poolConfig)
}

func (r *Router) evictLRU() error {
	if r.lruList.Len() == 0 {
		return errors.New("no pools to evict")
	}

	oldest := r.lruList.Back()
	if oldest == nil {
		return errors.New("no pools to evict")
	}

	tenantID := oldest.Value.(string)
	entry, ok := r.pools[tenantID]
	if !ok {
		r.lruList.Remove(oldest)
		return nil
	}

	age := time.Since(entry.lastUsed)
	if age < r.config.Pool.EvictionAge {
		return errors.New("oldest pool not yet eligible for eviction")
	}

	stats := entry.pool.Stat()
	connCount := int32(stats.AcquiredConns())

	entry.pool.Close()
	delete(r.pools, tenantID)
	r.lruList.Remove(entry.element)
	r.totalConns -= connCount

	log.Info().Str("tenant_id", tenantID).Str("db", entry.dbName).Dur("idle", age).Msg("Evicted idle tenant pool")
	return nil
}

func (r *Router) RemovePool(tenantID string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if entry, ok := r.pools[tenantID]; ok {
		stats := entry.pool.Stat()
		connCount := int32(stats.AcquiredConns())

		entry.pool.Close()
		delete(r.pools, tenantID)
		r.lruList.Remove(entry.element)
		r.totalConns -= connCount

		log.Info().Str("tenant_id", tenantID).Str("db", entry.dbName).Msg("Removed tenant pool")
	}
}

func (r *Router) RemovePoolByDBName(dbName string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	for tenantID, entry := range r.pools {
		if entry.dbName == dbName {
			stats := entry.pool.Stat()
			connCount := int32(stats.AcquiredConns())

			entry.pool.Close()
			delete(r.pools, tenantID)
			r.lruList.Remove(entry.element)
			r.totalConns -= connCount

			log.Info().Str("tenant_id", tenantID).Str("db", dbName).Msg("Removed tenant pool by db_name")
			return
		}
	}
}

func (r *Router) Close() {
	r.mu.Lock()
	defer r.mu.Unlock()

	for tenantID, entry := range r.pools {
		entry.pool.Close()
		log.Debug().Str("tenant_id", tenantID).Str("db", entry.dbName).Msg("Closed tenant pool")
	}

	r.pools = make(map[string]*poolEntry)
	r.lruList = list.New()
	r.totalConns = 0

	log.Info().Msg("All tenant pools closed")
}

func (r *Router) Stats() RouterStats {
	r.mu.RLock()
	defer r.mu.RUnlock()

	stats := RouterStats{
		TotalPools:      len(r.pools),
		TotalConns:      r.totalConns,
		ActiveDatabases: make([]string, 0, len(r.pools)),
	}

	for _, entry := range r.pools {
		stats.ActiveDatabases = append(stats.ActiveDatabases, entry.dbName)
	}

	return stats
}

type RouterStats struct {
	TotalPools      int
	TotalConns      int32
	ActiveDatabases []string
}
