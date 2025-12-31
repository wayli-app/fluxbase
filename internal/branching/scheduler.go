package branching

import (
	"context"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
)

// CleanupScheduler manages scheduled cleanup of expired branches
type CleanupScheduler struct {
	manager  *Manager
	router   *Router
	interval time.Duration
	ctx      context.Context
	cancel   context.CancelFunc
	wg       sync.WaitGroup
	running  bool
	mu       sync.Mutex
}

// NewCleanupScheduler creates a new cleanup scheduler
func NewCleanupScheduler(manager *Manager, router *Router, interval time.Duration) *CleanupScheduler {
	if interval <= 0 {
		interval = 1 * time.Hour // Default to hourly cleanup
	}

	ctx, cancel := context.WithCancel(context.Background())

	return &CleanupScheduler{
		manager:  manager,
		router:   router,
		interval: interval,
		ctx:      ctx,
		cancel:   cancel,
	}
}

// Start begins the cleanup scheduler
func (s *CleanupScheduler) Start() {
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		return
	}
	s.running = true
	s.mu.Unlock()

	s.wg.Add(1)
	go s.run()

	log.Info().
		Dur("interval", s.interval).
		Msg("Branch cleanup scheduler started")
}

// Stop gracefully stops the cleanup scheduler
func (s *CleanupScheduler) Stop() {
	s.mu.Lock()
	if !s.running {
		s.mu.Unlock()
		return
	}
	s.running = false
	s.mu.Unlock()

	s.cancel()
	s.wg.Wait()

	log.Info().Msg("Branch cleanup scheduler stopped")
}

// run is the main scheduler loop
func (s *CleanupScheduler) run() {
	defer s.wg.Done()

	// Run initial cleanup after a short delay to allow server to fully start
	select {
	case <-time.After(30 * time.Second):
		s.cleanup()
	case <-s.ctx.Done():
		return
	}

	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			s.cleanup()
		case <-s.ctx.Done():
			return
		}
	}
}

// cleanup performs the actual cleanup of expired branches
func (s *CleanupScheduler) cleanup() {
	log.Debug().Msg("Starting branch cleanup cycle")
	startTime := time.Now()

	// Get expired branches
	expired, err := s.manager.GetStorage().GetExpiredBranches(s.ctx)
	if err != nil {
		log.Error().Err(err).Msg("Failed to get expired branches for cleanup")
		return
	}

	if len(expired) == 0 {
		log.Debug().Msg("No expired branches to clean up")
		return
	}

	log.Info().
		Int("count", len(expired)).
		Msg("Found expired branches to clean up")

	deleted := 0
	failed := 0

	for _, branch := range expired {
		// Check if context is cancelled
		select {
		case <-s.ctx.Done():
			log.Warn().Msg("Cleanup interrupted by shutdown")
			return
		default:
		}

		log.Info().
			Str("branch_id", branch.ID.String()).
			Str("slug", branch.Slug).
			Time("expires_at", *branch.ExpiresAt).
			Msg("Deleting expired branch")

		// Close the connection pool first
		if s.router != nil {
			s.router.ClosePool(branch.Slug)
		}

		// Delete the branch
		if err := s.manager.DeleteBranch(s.ctx, branch.ID, nil); err != nil {
			log.Error().Err(err).
				Str("branch_id", branch.ID.String()).
				Str("slug", branch.Slug).
				Msg("Failed to delete expired branch")
			failed++
		} else {
			deleted++
		}
	}

	duration := time.Since(startTime)
	log.Info().
		Int("deleted", deleted).
		Int("failed", failed).
		Dur("duration", duration).
		Msg("Branch cleanup cycle completed")
}

// RunNow triggers an immediate cleanup cycle (useful for testing or manual cleanup)
func (s *CleanupScheduler) RunNow() {
	go s.cleanup()
}

// IsRunning returns whether the scheduler is currently running
func (s *CleanupScheduler) IsRunning() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.running
}
