package jobs

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/rs/zerolog/log"

	"github.com/nimbleflux/fluxbase/internal/database"
)

// ========== Workers ==========

// RegisterWorker registers a new worker
func (s *Storage) RegisterWorker(ctx context.Context, worker *WorkerRecord) error {
	query := `
		INSERT INTO jobs.workers (id, name, hostname, status, max_concurrent_jobs, metadata)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING started_at, last_heartbeat_at
	`

	return database.WrapWithServiceRole(ctx, s.DB, func(tx pgx.Tx) error {
		return tx.QueryRow(
			ctx, query,
			worker.ID, worker.Name, worker.Hostname, worker.Status,
			worker.MaxConcurrentJobs, worker.Metadata,
		).Scan(&worker.StartedAt, &worker.LastHeartbeatAt)
	})
}

// UpdateWorkerHeartbeat updates a worker's heartbeat timestamp
func (s *Storage) UpdateWorkerHeartbeat(ctx context.Context, workerID uuid.UUID, currentJobCount int) error {
	query := `
		UPDATE jobs.workers
		SET last_heartbeat_at = NOW(), current_job_count = $1
		WHERE id = $2
	`

	return database.WrapWithServiceRole(ctx, s.DB, func(tx pgx.Tx) error {
		result, err := tx.Exec(ctx, query, currentJobCount, workerID)
		if err != nil {
			return err
		}
		// Surface the silent no-op that happens once a worker row has been
		// reaped by the stale-worker sweep: the UPDATE affects 0 rows and the
		// loop keeps ticking uselessly. Without this, "heartbeats never update"
		// is invisible.
		if result.RowsAffected() == 0 {
			log.Warn().
				Str("worker_id", workerID.String()).
				Msg("Heartbeat updated 0 rows - worker not found (likely already reaped by stale cleanup)")
		}
		return nil
	})
}

func (s *Storage) UpdateWorkerStatus(ctx context.Context, workerID uuid.UUID, status WorkerStatus) error {
	query := `UPDATE jobs.workers SET status = $1 WHERE id = $2`
	return database.WrapWithServiceRole(ctx, s.DB, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx, query, status, workerID)
		return err
	})
}

func (s *Storage) DeregisterWorker(ctx context.Context, workerID uuid.UUID) error {
	query := `DELETE FROM jobs.workers WHERE id = $1`
	return database.WrapWithServiceRole(ctx, s.DB, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx, query, workerID)
		return err
	})
}

// GetWorker retrieves a worker by ID
func (s *Storage) GetWorker(ctx context.Context, workerID uuid.UUID) (*WorkerRecord, error) {
	query := `
		SELECT id, name, hostname, status, max_concurrent_jobs, current_job_count,
		       last_heartbeat_at, started_at, metadata
		FROM jobs.workers
		WHERE id = $1
	`

	var worker WorkerRecord
	err := s.DB.Pool().QueryRow(ctx, query, workerID).Scan(
		&worker.ID, &worker.Name, &worker.Hostname, &worker.Status,
		&worker.MaxConcurrentJobs, &worker.CurrentJobCount,
		&worker.LastHeartbeatAt, &worker.StartedAt, &worker.Metadata,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("worker not found: %s", workerID)
		}
		return nil, err
	}

	return &worker, nil
}

// ListWorkers lists all workers (admin access, bypasses RLS)
func (s *Storage) ListWorkers(ctx context.Context) ([]*WorkerRecord, error) {
	query := `
		SELECT id, name, hostname, status, max_concurrent_jobs, current_job_count,
		       last_heartbeat_at, started_at, metadata
		FROM jobs.workers
		WHERE (tenant_id = $1 OR ($1 IS NULL AND tenant_id IS NULL))
			ORDER BY started_at DESC
	`

	var workers []*WorkerRecord

	err := s.WithTenant(ctx, func(tx pgx.Tx) error {
		rows, err := tx.Query(ctx, query, database.TenantOrNil(database.TenantFromContext(ctx)))
		if err != nil {
			return err
		}
		defer rows.Close()

		for rows.Next() {
			var worker WorkerRecord
			err := rows.Scan(
				&worker.ID, &worker.Name, &worker.Hostname, &worker.Status,
				&worker.MaxConcurrentJobs, &worker.CurrentJobCount,
				&worker.LastHeartbeatAt, &worker.StartedAt, &worker.Metadata,
			)
			if err != nil {
				return err
			}
			workers = append(workers, &worker)
		}

		return rows.Err()
	})
	if err != nil {
		return nil, err
	}

	return workers, nil
}

// CleanupStaleWorkers removes workers that haven't sent a heartbeat in a while.
// A worker is only reaped if BOTH its heartbeat is stale AND it has been
// registered longer than the timeout — this guarantees a freshly-registered
// worker is never swept before its first heartbeat lands.
func (s *Storage) CleanupStaleWorkers(ctx context.Context, timeout time.Duration) (int64, error) {
	query := `
		DELETE FROM jobs.workers
		WHERE last_heartbeat_at < NOW() - $1::INTERVAL
		  AND started_at < NOW() - $1::INTERVAL
	`

	var result pgconn.CommandTag
	err := database.WrapWithServiceRole(ctx, s.DB, func(tx pgx.Tx) error {
		var execErr error
		result, execErr = tx.Exec(ctx, query, timeout.String())
		return execErr
	})
	if err != nil {
		return 0, err
	}

	return result.RowsAffected(), nil
}

func (s *Storage) ResetOrphanedJobs(ctx context.Context) (int64, error) {
	query := `
		UPDATE jobs.queue
		SET status = $1,
		    worker_id = NULL,
		    started_at = NULL,
		    last_progress_at = NULL
		WHERE status = $2
		  AND worker_id IS NULL
	`

	var result pgconn.CommandTag
	err := database.WrapWithServiceRole(ctx, s.DB, func(tx pgx.Tx) error {
		var execErr error
		result, execErr = tx.Exec(ctx, query, JobStatusPending, JobStatusRunning)
		return execErr
	})
	if err != nil {
		return 0, err
	}

	return result.RowsAffected(), nil
}
