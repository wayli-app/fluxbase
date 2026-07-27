package jobs

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"

	"github.com/nimbleflux/fluxbase/internal/config"
	"github.com/nimbleflux/fluxbase/internal/database"
	"github.com/nimbleflux/fluxbase/internal/secrets"
	"github.com/nimbleflux/fluxbase/internal/settings"
)

// workerError represents an error from a worker
type workerError struct {
	workerID uuid.UUID
	err      error
}

// Manager manages multiple workers with automatic restart on failure
type Manager struct {
	Config                 *config.JobsConfig
	BaseConfig             *config.Config
	db                     *database.Connection
	Storage                *Storage
	SecretsStorage         *secrets.Storage
	SettingsSecretsService *settings.SecretsService
	Workers                []*Worker
	jwtSecret              string
	publicURL              string
	wg                     sync.WaitGroup
	stopCh                 chan struct{}

	// Worker supervision
	workerErrors   chan workerError
	workersMutex   sync.RWMutex
	activeWorkers  map[uuid.UUID]bool
	restartCounts  map[uuid.UUID]int
	restartTimes   map[uuid.UUID]time.Time
	restartMutex   sync.Mutex
	targetCount    int
	supervisorCtx  context.Context
	supervisorStop context.CancelFunc
}

// NewManager creates a new worker manager
func NewManager(cfg *config.JobsConfig, conn *database.Connection, jwtSecret, publicURL string, secretsStorage *secrets.Storage, baseConfig *config.Config) *Manager {
	return &Manager{
		Config:         cfg,
		BaseConfig:     baseConfig,
		db:             conn,
		Storage:        NewStorage(conn),
		SecretsStorage: secretsStorage,
		Workers:        make([]*Worker, 0),
		jwtSecret:      jwtSecret,
		publicURL:      publicURL,
		stopCh:         make(chan struct{}),
		workerErrors:   make(chan workerError, 100),
		activeWorkers:  make(map[uuid.UUID]bool),
		restartCounts:  make(map[uuid.UUID]int),
		restartTimes:   make(map[uuid.UUID]time.Time),
	}
}

// Start starts the specified number of workers with automatic restart on failure
func (m *Manager) Start(ctx context.Context, workerCount int) error {
	if workerCount <= 0 {
		return fmt.Errorf("worker count must be positive, got: %d", workerCount)
	}

	log.Info().
		Int("worker_count", workerCount).
		Str("mode", m.Config.WorkerMode).
		Msg("Starting job worker manager")

	// Recover stale jobs from any previous process. On restart, all
	// previous workers are dead — their running jobs are orphaned.
	// Reset them to pending immediately instead of waiting 15-45s for
	// the staleWorkerCleanupLoop (which only runs from a live worker
	// and only resets jobs where worker_id IS NULL).
	recovered, err := m.Storage.RecoverStaleJobs(ctx)
	if err != nil {
		log.Warn().Err(err).Msg("Failed to recover stale jobs on startup")
	} else if recovered > 0 {
		log.Info().Int64("jobs_recovered", recovered).Msg("Recovered stale running jobs from previous process")
	}

	// Clean up stale worker registrations. timeout=0 means clean ALL rows
	// (they're all stale — no previous worker is alive on a fresh start).
	cleaned, err := m.Storage.CleanupStaleWorkers(ctx, 0)
	if err != nil {
		log.Warn().Err(err).Msg("Failed to clean up stale workers on startup")
	} else if cleaned > 0 {
		log.Info().Int64("workers_cleaned", cleaned).Msg("Cleaned up stale worker registrations from previous process")
	}

	m.targetCount = workerCount
	m.supervisorCtx, m.supervisorStop = context.WithCancel(context.Background())

	// Start supervisor goroutine to monitor worker health
	go m.superviseWorkers()

	// Start initial workers
	for i := 0; i < workerCount; i++ {
		m.startWorker(ctx)
	}

	log.Info().
		Int("worker_count", len(m.Workers)).
		Msg("All workers started")

	return nil
}

// startWorker creates and starts a single worker
func (m *Manager) startWorker(ctx context.Context) *Worker {
	worker := NewWorker(m.Config, m.Storage, m.jwtSecret, m.publicURL, m.SecretsStorage, m.BaseConfig, m.db)
	worker.SettingsSecretsService = m.SettingsSecretsService

	m.workersMutex.Lock()
	m.Workers = append(m.Workers, worker)
	m.activeWorkers[worker.ID] = true
	m.workersMutex.Unlock()

	m.wg.Add(1)
	go func(w *Worker) {
		defer m.wg.Done()
		defer func() {
			m.workersMutex.Lock()
			delete(m.activeWorkers, w.ID)
			m.workersMutex.Unlock()
		}()

		if err := w.Start(ctx); err != nil {
			log.Error().
				Err(err).
				Str("worker_id", w.ID.String()).
				Msg("Worker failed")
			// Notify supervisor about the failure
			select {
			case m.workerErrors <- workerError{workerID: w.ID, err: err}:
			default:
				// Channel full, log and continue
				log.Warn().Str("worker_id", w.ID.String()).Msg("Worker error channel full, cannot notify supervisor")
			}
		}
	}(worker)

	return worker
}

// superviseWorkers monitors worker health and restarts failed workers
func (m *Manager) superviseWorkers() {
	for {
		select {
		case err := <-m.workerErrors:
			log.Warn().
				Err(err.err).
				Str("worker_id", err.workerID.String()).
				Msg("Worker failed, checking restart eligibility")

			// Check if we should restart
			m.restartMutex.Lock()
			// Reset restart count after 5 minutes of stability
			if lastTime, ok := m.restartTimes[err.workerID]; ok && time.Since(lastTime) >= 5*time.Minute {
				delete(m.restartCounts, err.workerID)
				delete(m.restartTimes, err.workerID)
			}
			restartCount := m.restartCounts[err.workerID]
			maxRestarts := 5
			shouldRestart := restartCount < maxRestarts
			if shouldRestart {
				m.restartCounts[err.workerID] = restartCount + 1
				m.restartTimes[err.workerID] = time.Now()
			}
			m.restartMutex.Unlock()

			if shouldRestart {
				// Exponential backoff: 1s, 2s, 4s, 8s, 16s
				backoff := time.Second << time.Duration(restartCount)
				if backoff > 30*time.Second {
					backoff = 30 * time.Second
				}
				log.Info().
					Str("failed_worker_id", err.workerID.String()).
					Int("restart_count", restartCount+1).
					Dur("backoff", backoff).
					Msg("Scheduling worker restart with backoff")

				time.Sleep(backoff)

				// Check current worker count
				m.workersMutex.RLock()
				currentCount := len(m.activeWorkers)
				m.workersMutex.RUnlock()

				if currentCount < m.targetCount {
					log.Info().
						Int("current_workers", currentCount).
						Int("target_workers", m.targetCount).
						Msg("Starting replacement worker")
					m.startWorker(m.supervisorCtx)
				} else {
					log.Info().
						Int("current_workers", currentCount).
						Msg("Worker count at target, not starting replacement")
				}
			} else {
				log.Error().
					Str("worker_id", err.workerID.String()).
					Int("restart_count", restartCount).
					Msg("Worker exceeded max restarts, not restarting")
			}

		case <-m.supervisorCtx.Done():
			log.Info().Msg("Worker supervisor stopped")
			return
		}
	}
}

// Stop stops all workers gracefully
func (m *Manager) Stop() {
	log.Info().Msg("Stopping job worker manager")

	// Stop the supervisor first
	if m.supervisorStop != nil {
		m.supervisorStop()
	}

	// Signal all workers to stop
	m.workersMutex.RLock()
	for _, worker := range m.Workers {
		worker.Stop()
	}
	m.workersMutex.RUnlock()

	// Wait for all workers to complete
	m.wg.Wait()

	log.Info().Msg("All workers stopped")
}

// GetWorkerCount returns the number of active workers
// It returns the count from activeWorkers map when using supervisor,
// or falls back to Workers slice length for backward compatibility
func (m *Manager) GetWorkerCount() int {
	m.workersMutex.RLock()
	activeCount := len(m.activeWorkers)
	m.workersMutex.RUnlock()

	// If activeWorkers is populated, use it (supervisor mode)
	if activeCount > 0 {
		return activeCount
	}

	// Fall back to Workers slice for backward compatibility with tests
	m.workersMutex.RLock()
	defer m.workersMutex.RUnlock()
	return len(m.Workers)
}

// SetSettingsSecretsService sets the settings secrets service for accessing user/system secrets
func (m *Manager) SetSettingsSecretsService(svc *settings.SecretsService) {
	m.SettingsSecretsService = svc
}

// CancelJob cancels a running job by signaling all workers
// This immediately kills the Deno process if the job is running on any worker
func (m *Manager) CancelJob(jobID uuid.UUID) {
	m.workersMutex.RLock()
	workers := make([]*Worker, len(m.Workers))
	copy(workers, m.Workers)
	m.workersMutex.RUnlock()

	for _, worker := range workers {
		worker.cancelJob(jobID)
	}
}
