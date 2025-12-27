package jobs

import (
	"context"
	"fmt"
	"sync"

	"github.com/fluxbase-eu/fluxbase/internal/config"
	"github.com/fluxbase-eu/fluxbase/internal/database"
	"github.com/fluxbase-eu/fluxbase/internal/secrets"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
)

// Manager manages multiple workers
type Manager struct {
	Config         *config.JobsConfig
	Storage        *Storage
	SecretsStorage *secrets.Storage
	Workers        []*Worker
	jwtSecret      string
	publicURL      string
	wg             sync.WaitGroup
	stopCh         chan struct{}
}

// NewManager creates a new worker manager
func NewManager(cfg *config.JobsConfig, conn *database.Connection, jwtSecret, publicURL string, secretsStorage *secrets.Storage) *Manager {
	return &Manager{
		Config:         cfg,
		Storage:        NewStorage(conn),
		SecretsStorage: secretsStorage,
		Workers:        make([]*Worker, 0),
		jwtSecret:      jwtSecret,
		publicURL:      publicURL,
		stopCh:         make(chan struct{}),
	}
}

// Start starts the specified number of workers
func (m *Manager) Start(ctx context.Context, workerCount int) error {
	if workerCount <= 0 {
		return fmt.Errorf("worker count must be positive, got: %d", workerCount)
	}

	log.Info().
		Int("worker_count", workerCount).
		Str("mode", m.Config.WorkerMode).
		Msg("Starting job worker manager")

	// Start workers
	for i := 0; i < workerCount; i++ {
		worker := NewWorker(m.Config, m.Storage, m.jwtSecret, m.publicURL, m.SecretsStorage)
		m.Workers = append(m.Workers, worker)

		m.wg.Add(1)
		go func(w *Worker) {
			defer m.wg.Done()
			if err := w.Start(ctx); err != nil {
				log.Error().
					Err(err).
					Str("worker_id", w.ID.String()).
					Msg("Worker failed")
			}
		}(worker)
	}

	log.Info().
		Int("worker_count", len(m.Workers)).
		Msg("All workers started")

	return nil
}

// Stop stops all workers gracefully
func (m *Manager) Stop() {
	log.Info().Msg("Stopping job worker manager")

	// Signal all workers to stop
	for _, worker := range m.Workers {
		worker.Stop()
	}

	// Wait for all workers to complete
	m.wg.Wait()

	log.Info().Msg("All workers stopped")
}

// GetWorkerCount returns the number of active workers
func (m *Manager) GetWorkerCount() int {
	return len(m.Workers)
}

// CancelJob cancels a running job by signaling all workers
// This immediately kills the Deno process if the job is running on any worker
func (m *Manager) CancelJob(jobID uuid.UUID) {
	for _, worker := range m.Workers {
		worker.cancelJob(jobID)
	}
}
