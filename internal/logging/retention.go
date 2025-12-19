package logging

import (
	"context"
	"sync"
	"time"

	"github.com/fluxbase-eu/fluxbase/internal/config"
	"github.com/fluxbase-eu/fluxbase/internal/storage"
	"github.com/rs/zerolog/log"
)

// RetentionService handles automatic cleanup of old log entries.
// It runs periodically (daily by default) and deletes logs older than
// the configured retention period for each category.
type RetentionService struct {
	config   *config.LoggingConfig
	storage  storage.LogStorage
	interval time.Duration
	ctx      context.Context
	cancel   context.CancelFunc
	wg       sync.WaitGroup
	running  bool
	mu       sync.Mutex
}

// NewRetentionService creates a new retention cleanup service.
func NewRetentionService(cfg *config.LoggingConfig, logStorage storage.LogStorage) *RetentionService {
	ctx, cancel := context.WithCancel(context.Background())

	// Default to daily cleanup at 3 AM
	interval := 24 * time.Hour
	if cfg.RetentionCheckInterval > 0 {
		interval = cfg.RetentionCheckInterval
	}

	return &RetentionService{
		config:   cfg,
		storage:  logStorage,
		interval: interval,
		ctx:      ctx,
		cancel:   cancel,
	}
}

// Start begins the retention cleanup service.
func (r *RetentionService) Start() {
	r.mu.Lock()
	if r.running {
		r.mu.Unlock()
		return
	}
	r.running = true
	r.mu.Unlock()

	r.wg.Add(1)
	go r.run()

	log.Info().
		Dur("interval", r.interval).
		Int("system_retention_days", r.config.SystemRetentionDays).
		Int("http_retention_days", r.config.HTTPRetentionDays).
		Int("security_retention_days", r.config.SecurityRetentionDays).
		Int("execution_retention_days", r.config.ExecutionRetentionDays).
		Int("ai_retention_days", r.config.AIRetentionDays).
		Int("custom_retention_days", r.config.CustomRetentionDays).
		Msg("Log retention service started")
}

// Stop stops the retention cleanup service.
func (r *RetentionService) Stop() {
	r.mu.Lock()
	if !r.running {
		r.mu.Unlock()
		return
	}
	r.running = false
	r.mu.Unlock()

	r.cancel()
	r.wg.Wait()

	log.Info().Msg("Log retention service stopped")
}

// run is the main loop that periodically cleans up old logs.
func (r *RetentionService) run() {
	defer r.wg.Done()

	// Run immediately on start
	r.cleanup()

	ticker := time.NewTicker(r.interval)
	defer ticker.Stop()

	for {
		select {
		case <-r.ctx.Done():
			return
		case <-ticker.C:
			r.cleanup()
		}
	}
}

// cleanup deletes old logs for each category based on retention policies.
func (r *RetentionService) cleanup() {
	log.Debug().Msg("Starting log retention cleanup")

	categories := []struct {
		category      storage.LogCategory
		retentionDays int
	}{
		{storage.LogCategorySystem, r.config.SystemRetentionDays},
		{storage.LogCategoryHTTP, r.config.HTTPRetentionDays},
		{storage.LogCategorySecurity, r.config.SecurityRetentionDays},
		{storage.LogCategoryExecution, r.config.ExecutionRetentionDays},
		{storage.LogCategoryAI, r.config.AIRetentionDays},
		{storage.LogCategoryCustom, r.config.CustomRetentionDays},
	}

	var totalDeleted int64

	for _, cat := range categories {
		if cat.retentionDays <= 0 {
			// Skip categories with no retention policy (keep forever)
			continue
		}

		cutoff := time.Now().AddDate(0, 0, -cat.retentionDays)

		opts := storage.LogQueryOptions{
			Category: cat.category,
			EndTime:  cutoff,
		}

		ctx, cancel := context.WithTimeout(r.ctx, 5*time.Minute)
		deleted, err := r.storage.Delete(ctx, opts)
		cancel()

		if err != nil {
			log.Error().
				Err(err).
				Str("category", string(cat.category)).
				Int("retention_days", cat.retentionDays).
				Msg("Failed to delete old logs")
			continue
		}

		if deleted > 0 {
			log.Info().
				Str("category", string(cat.category)).
				Int64("deleted", deleted).
				Time("cutoff", cutoff).
				Msg("Deleted old log entries")
			totalDeleted += deleted
		}
	}

	if totalDeleted > 0 {
		log.Info().
			Int64("total_deleted", totalDeleted).
			Msg("Log retention cleanup completed")
	} else {
		log.Debug().Msg("Log retention cleanup completed - no entries to delete")
	}
}

// RunOnce runs the cleanup once immediately (for testing or manual triggers).
func (r *RetentionService) RunOnce() {
	r.cleanup()
}
