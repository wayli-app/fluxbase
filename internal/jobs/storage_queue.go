package jobs

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/nimbleflux/fluxbase/internal/database"
)

// ========== Job Queue ==========

// EnqueueJob adds a new job to the queue
func (s *Storage) EnqueueJob(ctx context.Context, job *Job) error {
	tenantID := database.TenantFromContext(ctx)
	return s.EnqueueJobWithTenant(ctx, tenantID, job)
}

// EnqueueJobWithTenant adds a new job to the queue with tenant context
func (s *Storage) EnqueueJobWithTenant(ctx context.Context, tenantID string, job *Job) error {
	query := `
		INSERT INTO jobs.queue (
			id, namespace, function_id, job_name, status, payload, priority,
			max_duration_seconds, progress_timeout_seconds, max_retries, created_by, user_role, user_email, scheduled_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
		RETURNING created_at
	`

	return database.WrapWithTenantAwareRole(ctx, s.DB, tenantID, func(tx pgx.Tx) error {
		return tx.QueryRow(
			ctx, query,
			job.ID, job.Namespace, job.JobFunctionID, job.JobName, job.Status, job.Payload,
			job.Priority, job.MaxDurationSeconds, job.ProgressTimeoutSeconds,
			job.MaxRetries, job.CreatedBy, job.UserRole, job.UserEmail, job.ScheduledAt,
		).Scan(&job.CreatedAt)
	})
}

// IsDuplicateJob checks if a pending or running job with the same parameters exists
func (s *Storage) IsDuplicateJob(ctx context.Context, namespace, jobName string, payload *string) (bool, *uuid.UUID, error) {
	// Check for pending or running jobs with matching namespace, job_name, and payload
	query := `
		SELECT id FROM jobs.queue
		WHERE namespace = $1
		  AND job_name = $2
		  AND status IN ($3, $4)
		  AND (
		      (payload IS NULL AND $5::text IS NULL) OR
		      (payload IS NOT NULL AND $5::text IS NOT NULL AND payload::text = $5::text)
		  )
		LIMIT 1
	`

	var existingID uuid.UUID
	err := s.WithTenant(ctx, func(tx pgx.Tx) error {
		return tx.QueryRow(ctx, query, namespace, jobName, JobStatusPending, JobStatusRunning, payload, database.TenantOrNil(database.TenantFromContext(ctx))).Scan(&existingID)
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, nil, nil
		}
		return false, nil, err
	}

	return true, &existingID, nil
}

// ClaimNextJob claims the next available job for a worker (using SELECT FOR UPDATE SKIP LOCKED)
func (s *Storage) ClaimNextJob(ctx context.Context, workerID uuid.UUID) (*Job, error) {
	query := `
		UPDATE jobs.queue
		SET status = $1,
		    worker_id = $2,
		    started_at = NOW(),
		    last_progress_at = NOW()
		WHERE id = (
			SELECT id FROM jobs.queue
			WHERE status = $3
			  AND (scheduled_at IS NULL OR scheduled_at <= NOW())
			ORDER BY priority DESC, created_at ASC
			LIMIT 1
			FOR UPDATE SKIP LOCKED
		)
		  AND EXISTS (SELECT 1 FROM jobs.workers WHERE id = $2)
		RETURNING id, namespace, function_id, job_name, status, payload, result, progress,
		          priority, max_duration_seconds, progress_timeout_seconds, max_retries,
		          retry_count, error_message, worker_id, created_by, user_role, user_email, created_at,
		          scheduled_at, started_at, last_progress_at, completed_at,
		          COALESCE(tenant_id::text, '')
	`

	var job Job
	var tenantID string
	err := database.WrapWithServiceRole(ctx, s.DB, func(tx pgx.Tx) error {
		return tx.QueryRow(ctx, query, JobStatusRunning, workerID, JobStatusPending).Scan(
			&job.ID, &job.Namespace, &job.JobFunctionID, &job.JobName, &job.Status,
			&job.Payload, &job.Result, &job.Progress, &job.Priority,
			&job.MaxDurationSeconds, &job.ProgressTimeoutSeconds, &job.MaxRetries,
			&job.RetryCount, &job.ErrorMessage, &job.WorkerID, &job.CreatedBy, &job.UserRole, &job.UserEmail,
			&job.CreatedAt, &job.ScheduledAt, &job.StartedAt, &job.LastProgressAt, &job.CompletedAt,
			&tenantID,
		)
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	job.TenantID = tenantID
	return &job, nil
}

// UpdateJobProgress updates job progress
func (s *Storage) UpdateJobProgress(ctx context.Context, jobID uuid.UUID, progress string) error {
	query := `
		UPDATE jobs.queue
		SET progress = $1, last_progress_at = NOW()
		WHERE id = $2 AND status = $3
	`

	var result pgconn.CommandTag
	err := s.WithTenant(ctx, func(tx pgx.Tx) error {
		var execErr error
		result, execErr = tx.Exec(ctx, query, progress, jobID, JobStatusRunning)
		return execErr
	})
	if err != nil {
		return err
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("job not found or not running: %s", jobID)
	}

	return nil
}

// Note: Execution logs are now stored in the central logging schema (logging.entries)

// CompleteJob marks a job as completed
func (s *Storage) CompleteJob(ctx context.Context, jobID uuid.UUID, result string) error {
	query := `
		UPDATE jobs.queue
		SET status = $1, result = $2, completed_at = NOW()
		WHERE id = $3 AND status = $4
	`

	var cmdTag pgconn.CommandTag
	err := s.WithTenant(ctx, func(tx pgx.Tx) error {
		var execErr error
		cmdTag, execErr = tx.Exec(ctx, query, JobStatusCompleted, result, jobID, JobStatusRunning)
		return execErr
	})
	if err != nil {
		return err
	}

	if cmdTag.RowsAffected() == 0 {
		return fmt.Errorf("job not found or not running: %s", jobID)
	}

	return nil
}

// FailJob marks a job as failed
func (s *Storage) FailJob(ctx context.Context, jobID uuid.UUID, errorMessage string) error {
	query := `
		UPDATE jobs.queue
		SET status = $1, error_message = $2, completed_at = NOW()
		WHERE id = $3 AND status = $4
	`

	var cmdTag pgconn.CommandTag
	err := s.WithTenant(ctx, func(tx pgx.Tx) error {
		var execErr error
		cmdTag, execErr = tx.Exec(ctx, query, JobStatusFailed, errorMessage, jobID, JobStatusRunning)
		return execErr
	})
	if err != nil {
		return err
	}

	if cmdTag.RowsAffected() == 0 {
		return fmt.Errorf("job not found or not running: %s", jobID)
	}

	return nil
}

// CancelJob marks a job as cancelled
func (s *Storage) CancelJob(ctx context.Context, jobID uuid.UUID) error {
	query := `
		UPDATE jobs.queue
		SET status = $1, completed_at = NOW()
		WHERE id = $2 AND status IN ($3, $4) AND (tenant_id = $5 OR ($5 IS NULL AND tenant_id IS NULL))
	`

	var result pgconn.CommandTag
	err := s.WithTenant(ctx, func(tx pgx.Tx) error {
		var execErr error
		result, execErr = tx.Exec(ctx, query, JobStatusCancelled, jobID, JobStatusPending, JobStatusRunning, database.TenantOrNil(database.TenantFromContext(ctx)))
		return execErr
	})
	if err != nil {
		return err
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("job not found or cannot be cancelled: %s", jobID)
	}

	return nil
}

// InterruptJob marks a running job as interrupted (used during graceful shutdown)
func (s *Storage) InterruptJob(ctx context.Context, jobID uuid.UUID, reason string) error {
	query := `
		UPDATE jobs.queue
		SET status = $1, error_message = $2, completed_at = NOW()
		WHERE id = $3 AND status = $4
	`

	var result pgconn.CommandTag
	err := s.WithTenant(ctx, func(tx pgx.Tx) error {
		var execErr error
		result, execErr = tx.Exec(ctx, query, JobStatusInterrupted, reason, jobID, JobStatusRunning)
		return execErr
	})
	if err != nil {
		return err
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("job not found or not running: %s", jobID)
	}

	return nil
}

func (s *Storage) RequeueJob(ctx context.Context, jobID uuid.UUID, errorMsg string) error {
	return s.requeueJobWithStatus(ctx, jobID, JobStatusRunning, errorMsg)
}

func (s *Storage) RequeueFailedJob(ctx context.Context, jobID uuid.UUID) error {
	return s.requeueJobWithStatus(ctx, jobID, JobStatusFailed, "")
}

func (s *Storage) requeueJobWithStatus(ctx context.Context, jobID uuid.UUID, currentStatus JobStatus, errorMsg string) error {
	query := `
		UPDATE jobs.queue
		SET status = $1, retry_count = retry_count + 1, worker_id = NULL,
		    started_at = NULL, last_progress_at = NULL, completed_at = NULL,
		    error_message = CASE WHEN $5 != '' THEN $5 ELSE error_message END,
		    scheduled_at = NOW() + make_interval(secs => 5.0 * POWER(2::float8, LEAST(retry_count, 6)))
		WHERE id = $2 AND status = $3 AND retry_count < max_retries AND (tenant_id = $4 OR ($4 IS NULL AND tenant_id IS NULL))
	`

	var result pgconn.CommandTag
	err := s.WithTenant(ctx, func(tx pgx.Tx) error {
		var execErr error
		result, execErr = tx.Exec(
			ctx, query,
			JobStatusPending, jobID, currentStatus,
			database.TenantOrNil(database.TenantFromContext(ctx)),
			errorMsg,
		)
		return execErr
	})
	if err != nil {
		return err
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("job not found, not %s, or max retries reached: %s", string(currentStatus), jobID)
	}

	return nil
}

// ResubmitJob creates a new job based on an existing job (works for any status)
func (s *Storage) ResubmitJob(ctx context.Context, originalJobID uuid.UUID) (*Job, error) {
	// First get the original job
	originalJob, err := s.GetJobByIDAdmin(ctx, originalJobID)
	if err != nil {
		return nil, fmt.Errorf("original job not found: %w", err)
	}

	// Create a new job with the same parameters
	newJob := &Job{
		ID:                     uuid.New(),
		Namespace:              originalJob.Namespace,
		JobFunctionID:          originalJob.JobFunctionID,
		JobName:                originalJob.JobName,
		Status:                 JobStatusPending,
		Payload:                originalJob.Payload,
		Priority:               originalJob.Priority,
		MaxDurationSeconds:     originalJob.MaxDurationSeconds,
		ProgressTimeoutSeconds: originalJob.ProgressTimeoutSeconds,
		MaxRetries:             originalJob.MaxRetries,
		RetryCount:             0,
		CreatedBy:              originalJob.CreatedBy,
		UserRole:               originalJob.UserRole,
		UserEmail:              originalJob.UserEmail,
	}

	// Insert the new job
	query := `
		INSERT INTO jobs.queue (
			id, namespace, function_id, job_name, status, payload, priority,
			max_duration_seconds, progress_timeout_seconds, max_retries, created_by, user_role, user_email
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
		RETURNING created_at
	`

	err = s.WithTenant(ctx, func(tx pgx.Tx) error {
		return tx.QueryRow(
			ctx, query,
			newJob.ID, newJob.Namespace, newJob.JobFunctionID, newJob.JobName, newJob.Status,
			newJob.Payload, newJob.Priority, newJob.MaxDurationSeconds, newJob.ProgressTimeoutSeconds,
			newJob.MaxRetries, newJob.CreatedBy, newJob.UserRole, newJob.UserEmail,
		).Scan(&newJob.CreatedAt)
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create new job: %w", err)
	}

	return newJob, nil
}

// CreateJob creates a new job in the queue (alias for EnqueueJob for consistency)
func (s *Storage) CreateJob(ctx context.Context, job *Job) error {
	return s.EnqueueJob(ctx, job)
}
