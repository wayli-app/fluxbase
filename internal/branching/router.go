package branching

import (
	"context"
	"fmt"
	"net/url"
	"sync"
	"time"

	"github.com/fluxbase-eu/fluxbase/internal/config"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog/log"
)

// Router manages connection pools for database branches
type Router struct {
	storage     *Storage
	config      config.BranchingConfig
	mainPool    *pgxpool.Pool
	mainDBURL   string
	pools       map[string]*pgxpool.Pool // slug -> pool
	poolsMu     sync.RWMutex
	poolConfigs map[string]*pgxpool.Config // slug -> config (for recreating pools)
}

// NewRouter creates a new branch router
func NewRouter(storage *Storage, cfg config.BranchingConfig, mainPool *pgxpool.Pool, mainDBURL string) *Router {
	return &Router{
		storage:     storage,
		config:      cfg,
		mainPool:    mainPool,
		mainDBURL:   mainDBURL,
		pools:       make(map[string]*pgxpool.Pool),
		poolConfigs: make(map[string]*pgxpool.Config),
	}
}

// GetPool returns the connection pool for a branch
// If the branch is "main" or empty, returns the main pool
func (r *Router) GetPool(ctx context.Context, slug string) (*pgxpool.Pool, error) {
	// Empty or "main" slug uses the main pool
	if slug == "" || slug == "main" {
		return r.mainPool, nil
	}

	// Check if branching is enabled
	if !r.config.Enabled {
		return nil, ErrBranchingDisabled
	}

	// Check if we already have a pool for this branch
	r.poolsMu.RLock()
	pool, exists := r.pools[slug]
	r.poolsMu.RUnlock()

	if exists && pool != nil {
		return pool, nil
	}

	// Need to create a new pool
	return r.createPoolForBranch(ctx, slug)
}

// createPoolForBranch creates a new connection pool for a branch
func (r *Router) createPoolForBranch(ctx context.Context, slug string) (*pgxpool.Pool, error) {
	r.poolsMu.Lock()
	defer r.poolsMu.Unlock()

	// Double-check after acquiring write lock
	if pool, exists := r.pools[slug]; exists && pool != nil {
		return pool, nil
	}

	// Get branch from storage
	branch, err := r.storage.GetBranchBySlug(ctx, slug)
	if err != nil {
		return nil, err
	}

	// Check if branch is ready
	if branch.Status != BranchStatusReady {
		return nil, ErrBranchNotReady
	}

	// Create connection URL for branch database
	connURL, err := r.getBranchConnectionURL(branch)
	if err != nil {
		return nil, fmt.Errorf("failed to get branch connection URL: %w", err)
	}

	// Parse pool config
	poolConfig, err := pgxpool.ParseConfig(connURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse pool config: %w", err)
	}

	// Configure pool settings (smaller pools for branch databases)
	poolConfig.MaxConns = 10
	poolConfig.MinConns = 1
	poolConfig.MaxConnLifetime = 30 * time.Minute
	poolConfig.MaxConnIdleTime = 5 * time.Minute

	// Create the pool
	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create pool: %w", err)
	}

	// Test the connection
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("failed to ping branch database: %w", err)
	}

	// Store the pool
	r.pools[slug] = pool
	r.poolConfigs[slug] = poolConfig

	log.Info().
		Str("branch_slug", slug).
		Str("database", branch.DatabaseName).
		Msg("Created connection pool for branch")

	return pool, nil
}

// getBranchConnectionURL returns the connection URL for a branch database
func (r *Router) getBranchConnectionURL(branch *Branch) (string, error) {
	// Parse the main database URL
	parsedURL, err := url.Parse(r.mainDBURL)
	if err != nil {
		return "", fmt.Errorf("failed to parse main database URL: %w", err)
	}

	// Replace the database name
	parsedURL.Path = "/" + branch.DatabaseName

	return parsedURL.String(), nil
}

// ClosePool closes and removes the pool for a branch
func (r *Router) ClosePool(slug string) {
	r.poolsMu.Lock()
	defer r.poolsMu.Unlock()

	if pool, exists := r.pools[slug]; exists {
		pool.Close()
		delete(r.pools, slug)
		delete(r.poolConfigs, slug)

		log.Info().
			Str("branch_slug", slug).
			Msg("Closed connection pool for branch")
	}
}

// CloseAllPools closes all branch pools (called during shutdown)
func (r *Router) CloseAllPools() {
	r.poolsMu.Lock()
	defer r.poolsMu.Unlock()

	for slug, pool := range r.pools {
		pool.Close()
		log.Debug().
			Str("branch_slug", slug).
			Msg("Closed connection pool for branch")
	}

	r.pools = make(map[string]*pgxpool.Pool)
	r.poolConfigs = make(map[string]*pgxpool.Config)
}

// RefreshPool recreates the pool for a branch (e.g., after migration)
func (r *Router) RefreshPool(ctx context.Context, slug string) error {
	// Close existing pool
	r.ClosePool(slug)

	// Create new pool
	_, err := r.createPoolForBranch(ctx, slug)
	return err
}

// GetActivePools returns the list of active branch slugs
func (r *Router) GetActivePools() []string {
	r.poolsMu.RLock()
	defer r.poolsMu.RUnlock()

	slugs := make([]string, 0, len(r.pools))
	for slug := range r.pools {
		slugs = append(slugs, slug)
	}
	return slugs
}

// GetPoolStats returns statistics for all pools
func (r *Router) GetPoolStats() map[string]PoolStats {
	r.poolsMu.RLock()
	defer r.poolsMu.RUnlock()

	stats := make(map[string]PoolStats)

	// Add main pool stats
	mainStat := r.mainPool.Stat()
	stats["main"] = PoolStats{
		TotalConns:      mainStat.TotalConns(),
		IdleConns:       mainStat.IdleConns(),
		AcquiredConns:   mainStat.AcquiredConns(),
		MaxConns:        mainStat.MaxConns(),
		AcquireCount:    mainStat.AcquireCount(),
		AcquireDuration: mainStat.AcquireDuration(),
	}

	// Add branch pool stats
	for slug, pool := range r.pools {
		stat := pool.Stat()
		stats[slug] = PoolStats{
			TotalConns:      stat.TotalConns(),
			IdleConns:       stat.IdleConns(),
			AcquiredConns:   stat.AcquiredConns(),
			MaxConns:        stat.MaxConns(),
			AcquireCount:    stat.AcquireCount(),
			AcquireDuration: stat.AcquireDuration(),
		}
	}

	return stats
}

// PoolStats contains connection pool statistics
type PoolStats struct {
	TotalConns      int32         `json:"total_conns"`
	IdleConns       int32         `json:"idle_conns"`
	AcquiredConns   int32         `json:"acquired_conns"`
	MaxConns        int32         `json:"max_conns"`
	AcquireCount    int64         `json:"acquire_count"`
	AcquireDuration time.Duration `json:"acquire_duration"`
}

// IsMainBranch checks if a slug refers to the main branch
func IsMainBranch(slug string) bool {
	return slug == "" || slug == "main"
}

// GetMainPool returns the main database pool
func (r *Router) GetMainPool() *pgxpool.Pool {
	return r.mainPool
}

// HasPool checks if a pool exists for the given branch
func (r *Router) HasPool(slug string) bool {
	if IsMainBranch(slug) {
		return true
	}

	r.poolsMu.RLock()
	defer r.poolsMu.RUnlock()

	_, exists := r.pools[slug]
	return exists
}

// WarmupPool pre-creates a connection pool for a branch
// This is useful after branch creation to ensure the pool is ready
func (r *Router) WarmupPool(ctx context.Context, slug string) error {
	_, err := r.GetPool(ctx, slug)
	return err
}

// GetStorage returns the storage instance
func (r *Router) GetStorage() *Storage {
	return r.storage
}
