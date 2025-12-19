package jobs

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/fluxbase-eu/fluxbase/internal/database"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// Storage provides database operations for jobs
type Storage struct {
	conn *database.Connection
}

// NewStorage creates a new Storage instance
func NewStorage(conn *database.Connection) *Storage {
	return &Storage{conn: conn}
}

// ========== Job Functions ==========

// CreateJobFunction creates a new job function
func (s *Storage) CreateJobFunction(ctx context.Context, fn *JobFunction) error {
	query := `
		INSERT INTO jobs.functions (
			id, name, namespace, description, code, original_code, is_bundled, bundle_error,
			enabled, schedule, timeout_seconds, memory_limit_mb, max_retries,
			progress_timeout_seconds, allow_net, allow_env, allow_read, allow_write,
			require_role, version, created_by, source
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21, $22
		)
		RETURNING created_at, updated_at
	`

	return s.conn.Pool().QueryRow(ctx, query,
		fn.ID, fn.Name, fn.Namespace, fn.Description, fn.Code, fn.OriginalCode,
		fn.IsBundled, fn.BundleError, fn.Enabled, fn.Schedule, fn.TimeoutSeconds,
		fn.MemoryLimitMB, fn.MaxRetries, fn.ProgressTimeoutSeconds,
		fn.AllowNet, fn.AllowEnv, fn.AllowRead, fn.AllowWrite,
		fn.RequireRole, fn.Version, fn.CreatedBy, fn.Source,
	).Scan(&fn.CreatedAt, &fn.UpdatedAt)
}

// UpdateJobFunction updates an existing job function
func (s *Storage) UpdateJobFunction(ctx context.Context, fn *JobFunction) error {
	query := `
		UPDATE jobs.functions SET
			description = $1, code = $2, original_code = $3, is_bundled = $4, bundle_error = $5,
			enabled = $6, schedule = $7, timeout_seconds = $8, memory_limit_mb = $9,
			max_retries = $10, progress_timeout_seconds = $11, allow_net = $12, allow_env = $13,
			allow_read = $14, allow_write = $15, require_role = $16, version = version + 1
		WHERE id = $17
		RETURNING version, updated_at
	`

	return s.conn.Pool().QueryRow(ctx, query,
		fn.Description, fn.Code, fn.OriginalCode, fn.IsBundled, fn.BundleError,
		fn.Enabled, fn.Schedule, fn.TimeoutSeconds, fn.MemoryLimitMB,
		fn.MaxRetries, fn.ProgressTimeoutSeconds, fn.AllowNet, fn.AllowEnv,
		fn.AllowRead, fn.AllowWrite, fn.RequireRole, fn.ID,
	).Scan(&fn.Version, &fn.UpdatedAt)
}

// UpsertJobFunction creates or updates a job function atomically
func (s *Storage) UpsertJobFunction(ctx context.Context, fn *JobFunction) error {
	query := `
		INSERT INTO jobs.functions (
			id, name, namespace, description, code, original_code, is_bundled, bundle_error,
			enabled, schedule, timeout_seconds, memory_limit_mb, max_retries,
			progress_timeout_seconds, allow_net, allow_env, allow_read, allow_write,
			require_role, version, created_by, source
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, 1, $20, $21
		)
		ON CONFLICT (name, namespace) DO UPDATE SET
			description = EXCLUDED.description,
			code = EXCLUDED.code,
			original_code = EXCLUDED.original_code,
			is_bundled = EXCLUDED.is_bundled,
			bundle_error = EXCLUDED.bundle_error,
			enabled = EXCLUDED.enabled,
			schedule = EXCLUDED.schedule,
			timeout_seconds = EXCLUDED.timeout_seconds,
			memory_limit_mb = EXCLUDED.memory_limit_mb,
			max_retries = EXCLUDED.max_retries,
			progress_timeout_seconds = EXCLUDED.progress_timeout_seconds,
			allow_net = EXCLUDED.allow_net,
			allow_env = EXCLUDED.allow_env,
			allow_read = EXCLUDED.allow_read,
			allow_write = EXCLUDED.allow_write,
			require_role = EXCLUDED.require_role,
			version = jobs.functions.version + 1,
			updated_at = NOW()
		RETURNING id, version, created_at, updated_at
	`

	return s.conn.Pool().QueryRow(ctx, query,
		fn.ID, fn.Name, fn.Namespace, fn.Description, fn.Code, fn.OriginalCode,
		fn.IsBundled, fn.BundleError, fn.Enabled, fn.Schedule, fn.TimeoutSeconds,
		fn.MemoryLimitMB, fn.MaxRetries, fn.ProgressTimeoutSeconds,
		fn.AllowNet, fn.AllowEnv, fn.AllowRead, fn.AllowWrite,
		fn.RequireRole, fn.CreatedBy, fn.Source,
	).Scan(&fn.ID, &fn.Version, &fn.CreatedAt, &fn.UpdatedAt)
}

// GetJobFunction retrieves a job function by namespace and name
func (s *Storage) GetJobFunction(ctx context.Context, namespace, name string) (*JobFunction, error) {
	query := `
		SELECT id, name, namespace, description, code, original_code, is_bundled, bundle_error,
			enabled, schedule, timeout_seconds, memory_limit_mb, max_retries,
			progress_timeout_seconds, allow_net, allow_env, allow_read, allow_write, require_role,
			version, created_by, source, created_at, updated_at
		FROM jobs.functions
		WHERE namespace = $1 AND name = $2
	`

	var fn JobFunction
	err := s.conn.Pool().QueryRow(ctx, query, namespace, name).Scan(
		&fn.ID, &fn.Name, &fn.Namespace, &fn.Description, &fn.Code, &fn.OriginalCode,
		&fn.IsBundled, &fn.BundleError, &fn.Enabled, &fn.Schedule, &fn.TimeoutSeconds,
		&fn.MemoryLimitMB, &fn.MaxRetries, &fn.ProgressTimeoutSeconds,
		&fn.AllowNet, &fn.AllowEnv, &fn.AllowRead, &fn.AllowWrite, &fn.RequireRole,
		&fn.Version, &fn.CreatedBy, &fn.Source, &fn.CreatedAt, &fn.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("job function not found: %s/%s", namespace, name)
		}
		return nil, err
	}

	return &fn, nil
}

// GetJobFunctionByName retrieves the first job function matching the name (any namespace)
// Results are ordered alphabetically by namespace, so "default" is preferred if it exists
func (s *Storage) GetJobFunctionByName(ctx context.Context, name string) (*JobFunction, error) {
	query := `
		SELECT id, name, namespace, description, code, original_code, is_bundled, bundle_error,
			enabled, schedule, timeout_seconds, memory_limit_mb, max_retries,
			progress_timeout_seconds, allow_net, allow_env, allow_read, allow_write, require_role,
			version, created_by, source, created_at, updated_at
		FROM jobs.functions
		WHERE name = $1
		ORDER BY namespace
		LIMIT 1
	`

	var fn JobFunction
	err := s.conn.Pool().QueryRow(ctx, query, name).Scan(
		&fn.ID, &fn.Name, &fn.Namespace, &fn.Description, &fn.Code, &fn.OriginalCode,
		&fn.IsBundled, &fn.BundleError, &fn.Enabled, &fn.Schedule, &fn.TimeoutSeconds,
		&fn.MemoryLimitMB, &fn.MaxRetries, &fn.ProgressTimeoutSeconds,
		&fn.AllowNet, &fn.AllowEnv, &fn.AllowRead, &fn.AllowWrite, &fn.RequireRole,
		&fn.Version, &fn.CreatedBy, &fn.Source, &fn.CreatedAt, &fn.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("job function not found: %s", name)
		}
		return nil, err
	}

	return &fn, nil
}

// GetJobFunctionByID retrieves a job function by ID
func (s *Storage) GetJobFunctionByID(ctx context.Context, id uuid.UUID) (*JobFunction, error) {
	query := `
		SELECT id, name, namespace, description, code, original_code, is_bundled, bundle_error,
			enabled, schedule, timeout_seconds, memory_limit_mb, max_retries,
			progress_timeout_seconds, allow_net, allow_env, allow_read, allow_write, require_role,
			version, created_by, source, created_at, updated_at
		FROM jobs.functions
		WHERE id = $1
	`

	var fn JobFunction
	err := s.conn.Pool().QueryRow(ctx, query, id).Scan(
		&fn.ID, &fn.Name, &fn.Namespace, &fn.Description, &fn.Code, &fn.OriginalCode,
		&fn.IsBundled, &fn.BundleError, &fn.Enabled, &fn.Schedule, &fn.TimeoutSeconds,
		&fn.MemoryLimitMB, &fn.MaxRetries, &fn.ProgressTimeoutSeconds,
		&fn.AllowNet, &fn.AllowEnv, &fn.AllowRead, &fn.AllowWrite, &fn.RequireRole,
		&fn.Version, &fn.CreatedBy, &fn.Source, &fn.CreatedAt, &fn.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("job function not found: %s", id)
		}
		return nil, err
	}

	return &fn, nil
}

// ListJobFunctions lists all job functions in a namespace (excludes code for performance)
func (s *Storage) ListJobFunctions(ctx context.Context, namespace string) ([]*JobFunctionSummary, error) {
	query := `
		SELECT id, name, namespace, description, is_bundled, bundle_error,
			enabled, schedule, timeout_seconds, memory_limit_mb, max_retries,
			progress_timeout_seconds, allow_net, allow_env, allow_read, allow_write, require_role,
			version, created_by, source, created_at, updated_at
		FROM jobs.functions
		WHERE namespace = $1
		ORDER BY name
	`

	rows, err := s.conn.Pool().Query(ctx, query, namespace)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var functions []*JobFunctionSummary
	for rows.Next() {
		var fn JobFunctionSummary
		err := rows.Scan(
			&fn.ID, &fn.Name, &fn.Namespace, &fn.Description,
			&fn.IsBundled, &fn.BundleError, &fn.Enabled, &fn.Schedule, &fn.TimeoutSeconds,
			&fn.MemoryLimitMB, &fn.MaxRetries, &fn.ProgressTimeoutSeconds,
			&fn.AllowNet, &fn.AllowEnv, &fn.AllowRead, &fn.AllowWrite, &fn.RequireRole,
			&fn.Version, &fn.CreatedBy, &fn.Source, &fn.CreatedAt, &fn.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		functions = append(functions, &fn)
	}

	return functions, rows.Err()
}

// ListAllJobFunctions lists all job functions across all namespaces (admin use)
func (s *Storage) ListAllJobFunctions(ctx context.Context) ([]*JobFunctionSummary, error) {
	query := `
		SELECT id, name, namespace, description, is_bundled, bundle_error,
			enabled, schedule, timeout_seconds, memory_limit_mb, max_retries,
			progress_timeout_seconds, allow_net, allow_env, allow_read, allow_write, require_role,
			version, created_by, source, created_at, updated_at
		FROM jobs.functions
		ORDER BY namespace, name
	`

	rows, err := s.conn.Pool().Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var functions []*JobFunctionSummary
	for rows.Next() {
		var fn JobFunctionSummary
		err := rows.Scan(
			&fn.ID, &fn.Name, &fn.Namespace, &fn.Description,
			&fn.IsBundled, &fn.BundleError, &fn.Enabled, &fn.Schedule, &fn.TimeoutSeconds,
			&fn.MemoryLimitMB, &fn.MaxRetries, &fn.ProgressTimeoutSeconds,
			&fn.AllowNet, &fn.AllowEnv, &fn.AllowRead, &fn.AllowWrite, &fn.RequireRole,
			&fn.Version, &fn.CreatedBy, &fn.Source, &fn.CreatedAt, &fn.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		functions = append(functions, &fn)
	}

	return functions, rows.Err()
}

// DeleteJobFunction deletes a job function
func (s *Storage) DeleteJobFunction(ctx context.Context, namespace, name string) error {
	query := `DELETE FROM jobs.functions WHERE namespace = $1 AND name = $2`
	result, err := s.conn.Pool().Exec(ctx, query, namespace, name)
	if err != nil {
		return err
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("job function not found: %s/%s", namespace, name)
	}

	return nil
}

// ========== Job Function Files ==========

// CreateJobFunctionFile creates a supporting file for a job function
func (s *Storage) CreateJobFunctionFile(ctx context.Context, file *JobFunctionFile) error {
	query := `
		INSERT INTO jobs.function_files (id, function_id, file_path, content)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (function_id, file_path) DO UPDATE SET content = EXCLUDED.content
		RETURNING created_at
	`

	return s.conn.Pool().QueryRow(ctx, query,
		file.ID, file.JobFunctionID, file.FilePath, file.Content,
	).Scan(&file.CreatedAt)
}

// ListJobFunctionFiles lists all files for a job function
func (s *Storage) ListJobFunctionFiles(ctx context.Context, jobFunctionID uuid.UUID) ([]*JobFunctionFile, error) {
	query := `
		SELECT id, function_id, file_path, content, created_at
		FROM jobs.function_files
		WHERE function_id = $1
		ORDER BY file_path
	`

	rows, err := s.conn.Pool().Query(ctx, query, jobFunctionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var files []*JobFunctionFile
	for rows.Next() {
		var file JobFunctionFile
		if err := rows.Scan(&file.ID, &file.JobFunctionID, &file.FilePath, &file.Content, &file.CreatedAt); err != nil {
			return nil, err
		}
		files = append(files, &file)
	}

	return files, rows.Err()
}

// DeleteJobFunctionFiles deletes all files for a job function
func (s *Storage) DeleteJobFunctionFiles(ctx context.Context, jobFunctionID uuid.UUID) error {
	query := `DELETE FROM jobs.function_files WHERE function_id = $1`
	_, err := s.conn.Pool().Exec(ctx, query, jobFunctionID)
	return err
}

// ========== Job Queue ==========

// EnqueueJob adds a new job to the queue
func (s *Storage) EnqueueJob(ctx context.Context, job *Job) error {
	query := `
		INSERT INTO jobs.queue (
			id, namespace, function_id, job_name, status, payload, priority,
			max_duration_seconds, progress_timeout_seconds, max_retries, created_by, user_role, user_email, scheduled_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
		RETURNING created_at
	`

	return s.conn.Pool().QueryRow(ctx, query,
		job.ID, job.Namespace, job.JobFunctionID, job.JobName, job.Status, job.Payload,
		job.Priority, job.MaxDurationSeconds, job.ProgressTimeoutSeconds,
		job.MaxRetries, job.CreatedBy, job.UserRole, job.UserEmail, job.ScheduledAt,
	).Scan(&job.CreatedAt)
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
		          scheduled_at, started_at, last_progress_at, completed_at
	`

	var job Job
	err := s.conn.Pool().QueryRow(ctx, query, JobStatusRunning, workerID, JobStatusPending).Scan(
		&job.ID, &job.Namespace, &job.JobFunctionID, &job.JobName, &job.Status,
		&job.Payload, &job.Result, &job.Progress, &job.Priority,
		&job.MaxDurationSeconds, &job.ProgressTimeoutSeconds, &job.MaxRetries,
		&job.RetryCount, &job.ErrorMessage, &job.WorkerID, &job.CreatedBy, &job.UserRole, &job.UserEmail,
		&job.CreatedAt, &job.ScheduledAt, &job.StartedAt, &job.LastProgressAt, &job.CompletedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil // No jobs available
		}
		return nil, err
	}

	return &job, nil
}

// UpdateJobProgress updates job progress
func (s *Storage) UpdateJobProgress(ctx context.Context, jobID uuid.UUID, progress string) error {
	query := `
		UPDATE jobs.queue
		SET progress = $1, last_progress_at = NOW()
		WHERE id = $2 AND status = $3
	`

	result, err := s.conn.Pool().Exec(ctx, query, progress, jobID, JobStatusRunning)
	if err != nil {
		return err
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("job not found or not running: %s", jobID)
	}

	return nil
}

// InsertExecutionLog inserts a single log line for a job
func (s *Storage) InsertExecutionLog(ctx context.Context, jobID uuid.UUID, lineNumber int, level string, message string) error {
	query := `
		INSERT INTO jobs.execution_logs (job_id, line_number, level, message)
		VALUES ($1, $2, $3, $4)
	`

	_, err := s.conn.Pool().Exec(ctx, query, jobID, lineNumber, level, message)
	return err
}

// GetExecutionLogs retrieves execution logs for a job, optionally starting after a line number
func (s *Storage) GetExecutionLogs(ctx context.Context, jobID uuid.UUID, afterLine *int) ([]*ExecutionLog, error) {
	query := `
		SELECT id, job_id, line_number, level, message, created_at
		FROM jobs.execution_logs
		WHERE job_id = $1
	`
	args := []interface{}{jobID}

	if afterLine != nil {
		query += " AND line_number > $2"
		args = append(args, *afterLine)
	}

	query += " ORDER BY line_number ASC"

	var logs []*ExecutionLog

	// Use service role to bypass RLS (admin endpoint)
	err := database.WrapWithServiceRole(ctx, s.conn, func(tx pgx.Tx) error {
		rows, err := tx.Query(ctx, query, args...)
		if err != nil {
			return err
		}
		defer rows.Close()

		for rows.Next() {
			var log ExecutionLog
			if err := rows.Scan(&log.ID, &log.JobID, &log.LineNumber, &log.Level, &log.Message, &log.CreatedAt); err != nil {
				return err
			}
			logs = append(logs, &log)
		}

		return rows.Err()
	})

	if err != nil {
		return nil, err
	}

	return logs, nil
}

// CompleteJob marks a job as completed
func (s *Storage) CompleteJob(ctx context.Context, jobID uuid.UUID, result string) error {
	query := `
		UPDATE jobs.queue
		SET status = $1, result = $2, completed_at = NOW()
		WHERE id = $3 AND status = $4
	`

	cmdTag, err := s.conn.Pool().Exec(ctx, query, JobStatusCompleted, result, jobID, JobStatusRunning)
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

	cmdTag, err := s.conn.Pool().Exec(ctx, query, JobStatusFailed, errorMessage, jobID, JobStatusRunning)
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
		WHERE id = $2 AND status IN ($3, $4)
	`

	result, err := s.conn.Pool().Exec(ctx, query, JobStatusCancelled, jobID, JobStatusPending, JobStatusRunning)
	if err != nil {
		return err
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("job not found or cannot be cancelled: %s", jobID)
	}

	return nil
}

// RequeueJob requeues a failed job for retry
func (s *Storage) RequeueJob(ctx context.Context, jobID uuid.UUID) error {
	query := `
		UPDATE jobs.queue
		SET status = $1, retry_count = retry_count + 1, worker_id = NULL,
		    started_at = NULL, last_progress_at = NULL, completed_at = NULL,
		    error_message = NULL
		WHERE id = $2 AND status = $3 AND retry_count < max_retries
	`

	result, err := s.conn.Pool().Exec(ctx, query, JobStatusPending, jobID, JobStatusFailed)
	if err != nil {
		return err
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("job not found, not failed, or max retries reached: %s", jobID)
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

	err = s.conn.Pool().QueryRow(ctx, query,
		newJob.ID, newJob.Namespace, newJob.JobFunctionID, newJob.JobName, newJob.Status,
		newJob.Payload, newJob.Priority, newJob.MaxDurationSeconds, newJob.ProgressTimeoutSeconds,
		newJob.MaxRetries, newJob.CreatedBy, newJob.UserRole, newJob.UserEmail,
	).Scan(&newJob.CreatedAt)

	if err != nil {
		return nil, fmt.Errorf("failed to create new job: %w", err)
	}

	return newJob, nil
}

// GetJob retrieves a job by ID
func (s *Storage) GetJob(ctx context.Context, jobID uuid.UUID) (*Job, error) {
	query := `
		SELECT q.id, q.namespace, q.function_id, q.job_name, q.status, q.payload, q.result, q.progress,
		       q.priority, q.max_duration_seconds, q.progress_timeout_seconds, q.max_retries,
		       q.retry_count, q.error_message, q.worker_id, q.created_by, q.user_role, q.user_email,
		       COALESCE(u.user_metadata->>'name', u.user_metadata->>'full_name') as user_name,
		       q.created_at, q.scheduled_at, q.started_at, q.last_progress_at, q.completed_at
		FROM jobs.queue q
		LEFT JOIN auth.users u ON q.created_by = u.id
		WHERE q.id = $1
	`

	var job Job
	err := s.conn.Pool().QueryRow(ctx, query, jobID).Scan(
		&job.ID, &job.Namespace, &job.JobFunctionID, &job.JobName, &job.Status,
		&job.Payload, &job.Result, &job.Progress, &job.Priority,
		&job.MaxDurationSeconds, &job.ProgressTimeoutSeconds, &job.MaxRetries,
		&job.RetryCount, &job.ErrorMessage, &job.WorkerID, &job.CreatedBy, &job.UserRole, &job.UserEmail, &job.UserName,
		&job.CreatedAt, &job.ScheduledAt, &job.StartedAt, &job.LastProgressAt, &job.CompletedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("job not found: %s", jobID)
		}
		return nil, err
	}

	return &job, nil
}

// ListJobs lists jobs with optional filters
// Note: This query excludes large fields (result, payload) for performance by default.
// Use GetJob to fetch full job details, or set IncludeResult filter to include result field.
func (s *Storage) ListJobs(ctx context.Context, filters *JobFilters) ([]*Job, error) {
	// Conditionally include result field (payload always excluded for list performance)
	includeResult := filters != nil && filters.IncludeResult != nil && *filters.IncludeResult

	var query string
	if includeResult {
		query = `
		SELECT q.id, q.namespace, q.function_id, q.job_name, q.status, q.result, q.progress,
		       q.priority, q.max_duration_seconds, q.progress_timeout_seconds, q.max_retries,
		       q.retry_count, q.error_message, q.worker_id, q.created_by, q.user_role, q.user_email,
		       COALESCE(u.user_metadata->>'name', u.user_metadata->>'full_name') as user_name,
		       q.created_at, q.scheduled_at, q.started_at, q.last_progress_at, q.completed_at
		FROM jobs.queue q
		LEFT JOIN auth.users u ON q.created_by = u.id
		WHERE 1=1
	`
	} else {
		query = `
		SELECT q.id, q.namespace, q.function_id, q.job_name, q.status, q.progress,
		       q.priority, q.max_duration_seconds, q.progress_timeout_seconds, q.max_retries,
		       q.retry_count, q.error_message, q.worker_id, q.created_by, q.user_role, q.user_email,
		       COALESCE(u.user_metadata->>'name', u.user_metadata->>'full_name') as user_name,
		       q.created_at, q.scheduled_at, q.started_at, q.last_progress_at, q.completed_at
		FROM jobs.queue q
		LEFT JOIN auth.users u ON q.created_by = u.id
		WHERE 1=1
	`
	}

	args := []interface{}{}
	argCount := 1

	if filters != nil {
		if filters.Status != nil {
			query += fmt.Sprintf(" AND q.status = $%d", argCount)
			args = append(args, *filters.Status)
			argCount++
		}
		if filters.JobName != nil {
			query += fmt.Sprintf(" AND q.job_name = $%d", argCount)
			args = append(args, *filters.JobName)
			argCount++
		}
		if filters.Namespace != nil {
			query += fmt.Sprintf(" AND q.namespace = $%d", argCount)
			args = append(args, *filters.Namespace)
			argCount++
		}
		if filters.CreatedBy != nil {
			query += fmt.Sprintf(" AND q.created_by = $%d", argCount)
			args = append(args, *filters.CreatedBy)
			argCount++
		}
		if filters.WorkerID != nil {
			query += fmt.Sprintf(" AND q.worker_id = $%d", argCount)
			args = append(args, *filters.WorkerID)
			argCount++
		}
	}

	query += " ORDER BY q.created_at DESC"

	if filters != nil && filters.Limit != nil && *filters.Limit > 0 {
		query += fmt.Sprintf(" LIMIT $%d", argCount)
		args = append(args, *filters.Limit)
		argCount++

		if filters.Offset != nil && *filters.Offset > 0 {
			query += fmt.Sprintf(" OFFSET $%d", argCount)
			args = append(args, *filters.Offset)
		}
	}

	rows, err := s.conn.Pool().Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var jobs []*Job
	for rows.Next() {
		var job Job
		var scanErr error
		if includeResult {
			// Scan with result field included
			scanErr = rows.Scan(
				&job.ID, &job.Namespace, &job.JobFunctionID, &job.JobName, &job.Status,
				&job.Result, &job.Progress, &job.Priority,
				&job.MaxDurationSeconds, &job.ProgressTimeoutSeconds, &job.MaxRetries,
				&job.RetryCount, &job.ErrorMessage, &job.WorkerID, &job.CreatedBy, &job.UserRole, &job.UserEmail, &job.UserName,
				&job.CreatedAt, &job.ScheduledAt, &job.StartedAt, &job.LastProgressAt, &job.CompletedAt,
			)
		} else {
			// Scan without result field (payload, result are nil for performance)
			scanErr = rows.Scan(
				&job.ID, &job.Namespace, &job.JobFunctionID, &job.JobName, &job.Status,
				&job.Progress, &job.Priority,
				&job.MaxDurationSeconds, &job.ProgressTimeoutSeconds, &job.MaxRetries,
				&job.RetryCount, &job.ErrorMessage, &job.WorkerID, &job.CreatedBy, &job.UserRole, &job.UserEmail, &job.UserName,
				&job.CreatedAt, &job.ScheduledAt, &job.StartedAt, &job.LastProgressAt, &job.CompletedAt,
			)
		}
		if scanErr != nil {
			return nil, scanErr
		}
		jobs = append(jobs, &job)
	}

	return jobs, rows.Err()
}

// CreateJob creates a new job in the queue (alias for EnqueueJob for consistency)
func (s *Storage) CreateJob(ctx context.Context, job *Job) error {
	return s.EnqueueJob(ctx, job)
}

// GetJobByIDAdmin retrieves a job by ID (admin access, bypasses RLS)
func (s *Storage) GetJobByIDAdmin(ctx context.Context, jobID uuid.UUID) (*Job, error) {
	query := `
		SELECT q.id, q.namespace, q.function_id, q.job_name, q.status, q.payload, q.result, q.progress,
		       q.priority, q.max_duration_seconds, q.progress_timeout_seconds, q.max_retries,
		       q.retry_count, q.error_message, q.worker_id, q.created_by, q.user_role, q.user_email,
		       COALESCE(u.user_metadata->>'name', u.user_metadata->>'full_name') as user_name,
		       q.created_at, q.scheduled_at, q.started_at, q.last_progress_at, q.completed_at
		FROM jobs.queue q
		LEFT JOIN auth.users u ON q.created_by = u.id
		WHERE q.id = $1
	`

	var job Job

	// Use service role to bypass RLS (admin endpoint)
	err := database.WrapWithServiceRole(ctx, s.conn, func(tx pgx.Tx) error {
		return tx.QueryRow(ctx, query, jobID).Scan(
			&job.ID, &job.Namespace, &job.JobFunctionID, &job.JobName, &job.Status,
			&job.Payload, &job.Result, &job.Progress, &job.Priority,
			&job.MaxDurationSeconds, &job.ProgressTimeoutSeconds, &job.MaxRetries,
			&job.RetryCount, &job.ErrorMessage, &job.WorkerID, &job.CreatedBy, &job.UserRole, &job.UserEmail, &job.UserName,
			&job.CreatedAt, &job.ScheduledAt, &job.StartedAt, &job.LastProgressAt, &job.CompletedAt,
		)
	})

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("job not found: %s", jobID)
		}
		return nil, err
	}

	return &job, nil
}

// ListJobsAdmin lists jobs with optional filters (admin access, bypasses RLS)
// Note: This query excludes large fields (result, payload) for performance by default.
// Use GetJobByIDAdmin to fetch full job details, or set IncludeResult filter to include result field.
func (s *Storage) ListJobsAdmin(ctx context.Context, filters *JobFilters) ([]*Job, error) {
	// Conditionally include result field (payload always excluded for list performance)
	includeResult := filters != nil && filters.IncludeResult != nil && *filters.IncludeResult

	var query string
	if includeResult {
		query = `
		SELECT q.id, q.namespace, q.function_id, q.job_name, q.status, q.result, q.progress,
		       q.priority, q.max_duration_seconds, q.progress_timeout_seconds, q.max_retries,
		       q.retry_count, q.error_message, q.worker_id, q.created_by, q.user_role, q.user_email,
		       COALESCE(u.user_metadata->>'name', u.user_metadata->>'full_name') as user_name,
		       q.created_at, q.scheduled_at, q.started_at, q.last_progress_at, q.completed_at
		FROM jobs.queue q
		LEFT JOIN auth.users u ON q.created_by = u.id
		WHERE 1=1
	`
	} else {
		query = `
		SELECT q.id, q.namespace, q.function_id, q.job_name, q.status, q.progress,
		       q.priority, q.max_duration_seconds, q.progress_timeout_seconds, q.max_retries,
		       q.retry_count, q.error_message, q.worker_id, q.created_by, q.user_role, q.user_email,
		       COALESCE(u.user_metadata->>'name', u.user_metadata->>'full_name') as user_name,
		       q.created_at, q.scheduled_at, q.started_at, q.last_progress_at, q.completed_at
		FROM jobs.queue q
		LEFT JOIN auth.users u ON q.created_by = u.id
		WHERE 1=1
	`
	}

	args := []interface{}{}
	argCount := 1

	if filters != nil {
		if filters.Status != nil {
			query += fmt.Sprintf(" AND q.status = $%d", argCount)
			args = append(args, *filters.Status)
			argCount++
		}
		if filters.Namespace != nil {
			query += fmt.Sprintf(" AND q.namespace = $%d", argCount)
			args = append(args, *filters.Namespace)
			argCount++
		}
		if filters.JobName != nil {
			query += fmt.Sprintf(" AND q.job_name = $%d", argCount)
			args = append(args, *filters.JobName)
			argCount++
		}
		if filters.CreatedBy != nil {
			query += fmt.Sprintf(" AND q.created_by = $%d", argCount)
			args = append(args, *filters.CreatedBy)
			argCount++
		}
		if filters.WorkerID != nil {
			query += fmt.Sprintf(" AND q.worker_id = $%d", argCount)
			args = append(args, *filters.WorkerID)
			argCount++
		}
	}

	query += " ORDER BY q.created_at DESC"

	if filters != nil && filters.Limit != nil && *filters.Limit > 0 {
		query += fmt.Sprintf(" LIMIT $%d", argCount)
		args = append(args, *filters.Limit)
		argCount++

		if filters.Offset != nil && *filters.Offset > 0 {
			query += fmt.Sprintf(" OFFSET $%d", argCount)
			args = append(args, *filters.Offset)
		}
	}

	var jobs []*Job

	// Use service role to bypass RLS (admin endpoint)
	err := database.WrapWithServiceRole(ctx, s.conn, func(tx pgx.Tx) error {
		rows, err := tx.Query(ctx, query, args...)
		if err != nil {
			return err
		}
		defer rows.Close()

		for rows.Next() {
			var job Job
			var scanErr error
			if includeResult {
				// Scan with result field included
				scanErr = rows.Scan(
					&job.ID, &job.Namespace, &job.JobFunctionID, &job.JobName, &job.Status,
					&job.Result, &job.Progress, &job.Priority,
					&job.MaxDurationSeconds, &job.ProgressTimeoutSeconds, &job.MaxRetries,
					&job.RetryCount, &job.ErrorMessage, &job.WorkerID, &job.CreatedBy, &job.UserRole, &job.UserEmail, &job.UserName,
					&job.CreatedAt, &job.ScheduledAt, &job.StartedAt, &job.LastProgressAt, &job.CompletedAt,
				)
			} else {
				// Scan without result field (payload, result are nil for performance)
				scanErr = rows.Scan(
					&job.ID, &job.Namespace, &job.JobFunctionID, &job.JobName, &job.Status,
					&job.Progress, &job.Priority,
					&job.MaxDurationSeconds, &job.ProgressTimeoutSeconds, &job.MaxRetries,
					&job.RetryCount, &job.ErrorMessage, &job.WorkerID, &job.CreatedBy, &job.UserRole, &job.UserEmail, &job.UserName,
					&job.CreatedAt, &job.ScheduledAt, &job.StartedAt, &job.LastProgressAt, &job.CompletedAt,
				)
			}
			if scanErr != nil {
				return scanErr
			}
			jobs = append(jobs, &job)
		}

		return rows.Err()
	})

	if err != nil {
		return nil, err
	}

	return jobs, nil
}

// GetJobStats retrieves aggregate statistics about jobs (admin access, bypasses RLS)
func (s *Storage) GetJobStats(ctx context.Context, namespace *string) (*JobStats, error) {
	stats := &JobStats{}

	// Build args for namespace filter
	var args []interface{}
	if namespace != nil {
		args = append(args, *namespace)
	}

	// Use service role to bypass RLS (admin endpoint)
	err := database.WrapWithServiceRole(ctx, s.conn, func(tx pgx.Tx) error {
		// Basic counts query
		countQuery := `
			SELECT
				COUNT(*) AS total,
				COUNT(*) FILTER (WHERE status = 'pending') AS pending,
				COUNT(*) FILTER (WHERE status = 'running') AS running,
				COUNT(*) FILTER (WHERE status = 'completed') AS completed,
				COUNT(*) FILTER (WHERE status = 'failed') AS failed,
				COUNT(*) FILTER (WHERE status = 'cancelled') AS cancelled,
				COALESCE(AVG(EXTRACT(EPOCH FROM (completed_at - started_at))) FILTER (WHERE completed_at IS NOT NULL AND started_at IS NOT NULL), 0) AS avg_duration
			FROM jobs.queue
		`
		if namespace != nil {
			countQuery += " WHERE namespace = $1"
		}

		err := tx.QueryRow(ctx, countQuery, args...).Scan(
			&stats.TotalJobs, &stats.PendingJobs, &stats.RunningJobs,
			&stats.CompletedJobs, &stats.FailedJobs, &stats.CancelledJobs,
			&stats.AvgDurationSeconds,
		)
		if err != nil {
			return err
		}

		// Jobs by status
		statusQuery := `
			SELECT status, COUNT(*) as count
			FROM jobs.queue
		`
		if namespace != nil {
			statusQuery += " WHERE namespace = $1"
		}
		statusQuery += " GROUP BY status ORDER BY count DESC"

		rows, err := tx.Query(ctx, statusQuery, args...)
		if err != nil {
			return err
		}
		defer rows.Close()

		for rows.Next() {
			var sc JobStatusCount
			if err := rows.Scan(&sc.Status, &sc.Count); err != nil {
				return err
			}
			stats.JobsByStatus = append(stats.JobsByStatus, sc)
		}
		if err := rows.Err(); err != nil {
			return err
		}

		// Jobs by day (last 7 days)
		dayQuery := `
			SELECT DATE(created_at) as date, COUNT(*) as count
			FROM jobs.queue
			WHERE created_at >= NOW() - INTERVAL '7 days'
		`
		if namespace != nil {
			dayQuery += " AND namespace = $1"
		}
		dayQuery += " GROUP BY DATE(created_at) ORDER BY date DESC"

		rows, err = tx.Query(ctx, dayQuery, args...)
		if err != nil {
			return err
		}
		defer rows.Close()

		for rows.Next() {
			var dc JobDayCount
			var date time.Time
			if err := rows.Scan(&date, &dc.Count); err != nil {
				return err
			}
			dc.Date = date.Format("2006-01-02")
			stats.JobsByDay = append(stats.JobsByDay, dc)
		}
		if err := rows.Err(); err != nil {
			return err
		}

		// Jobs by function (top 10)
		funcQuery := `
			SELECT job_name, COUNT(*) as count
			FROM jobs.queue
		`
		if namespace != nil {
			funcQuery += " WHERE namespace = $1"
		}
		funcQuery += " GROUP BY job_name ORDER BY count DESC LIMIT 10"

		rows, err = tx.Query(ctx, funcQuery, args...)
		if err != nil {
			return err
		}
		defer rows.Close()

		for rows.Next() {
			var fc JobFunctionCount
			if err := rows.Scan(&fc.Name, &fc.Count); err != nil {
				return err
			}
			stats.JobsByFunction = append(stats.JobsByFunction, fc)
		}

		return rows.Err()
	})

	if err != nil {
		return nil, err
	}

	return stats, nil
}

// ========== Workers ==========

// RegisterWorker registers a new worker
func (s *Storage) RegisterWorker(ctx context.Context, worker *WorkerRecord) error {
	query := `
		INSERT INTO jobs.workers (id, name, hostname, status, max_concurrent_jobs, metadata)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING started_at, last_heartbeat_at
	`

	return s.conn.Pool().QueryRow(ctx, query,
		worker.ID, worker.Name, worker.Hostname, worker.Status,
		worker.MaxConcurrentJobs, worker.Metadata,
	).Scan(&worker.StartedAt, &worker.LastHeartbeatAt)
}

// UpdateWorkerHeartbeat updates a worker's heartbeat timestamp
func (s *Storage) UpdateWorkerHeartbeat(ctx context.Context, workerID uuid.UUID, currentJobCount int) error {
	query := `
		UPDATE jobs.workers
		SET last_heartbeat_at = NOW(), current_job_count = $1
		WHERE id = $2
	`

	_, err := s.conn.Pool().Exec(ctx, query, currentJobCount, workerID)
	return err
}

// UpdateWorkerStatus updates a worker's status
func (s *Storage) UpdateWorkerStatus(ctx context.Context, workerID uuid.UUID, status WorkerStatus) error {
	query := `UPDATE jobs.workers SET status = $1 WHERE id = $2`
	_, err := s.conn.Pool().Exec(ctx, query, status, workerID)
	return err
}

// DeregisterWorker removes a worker from the registry
func (s *Storage) DeregisterWorker(ctx context.Context, workerID uuid.UUID) error {
	query := `DELETE FROM jobs.workers WHERE id = $1`
	_, err := s.conn.Pool().Exec(ctx, query, workerID)
	return err
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
	err := s.conn.Pool().QueryRow(ctx, query, workerID).Scan(
		&worker.ID, &worker.Name, &worker.Hostname, &worker.Status,
		&worker.MaxConcurrentJobs, &worker.CurrentJobCount,
		&worker.LastHeartbeatAt, &worker.StartedAt, &worker.Metadata,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
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
		ORDER BY started_at DESC
	`

	var workers []*WorkerRecord

	// Use service role to bypass RLS (admin endpoint)
	err := database.WrapWithServiceRole(ctx, s.conn, func(tx pgx.Tx) error {
		rows, err := tx.Query(ctx, query)
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

// CleanupStaleWorkers removes workers that haven't sent a heartbeat in a while
func (s *Storage) CleanupStaleWorkers(ctx context.Context, timeout time.Duration) (int64, error) {
	query := `
		DELETE FROM jobs.workers
		WHERE last_heartbeat_at < NOW() - $1::INTERVAL
	`

	result, err := s.conn.Pool().Exec(ctx, query, timeout.String())
	if err != nil {
		return 0, err
	}

	return result.RowsAffected(), nil
}

// ResetOrphanedJobs resets jobs that are running but have no worker (worker was deleted)
// Returns the number of jobs reset to pending status
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

	result, err := s.conn.Pool().Exec(ctx, query, JobStatusPending, JobStatusRunning)
	if err != nil {
		return 0, err
	}

	return result.RowsAffected(), nil
}

// ========== Namespace Functions ==========

// ListJobNamespaces returns all unique namespaces that have job functions (admin access, bypasses RLS)
func (s *Storage) ListJobNamespaces(ctx context.Context) ([]string, error) {
	query := `SELECT DISTINCT namespace FROM jobs.functions ORDER BY namespace`

	var namespaces []string

	// Use service role to bypass RLS (admin endpoint)
	err := database.WrapWithServiceRole(ctx, s.conn, func(tx pgx.Tx) error {
		rows, err := tx.Query(ctx, query)
		if err != nil {
			return err
		}
		defer rows.Close()

		for rows.Next() {
			var ns string
			if err := rows.Scan(&ns); err != nil {
				return err
			}
			namespaces = append(namespaces, ns)
		}
		return rows.Err()
	})

	if err != nil {
		return nil, err
	}

	return namespaces, nil
}

// ========== Helper Functions ==========

// ProgressToJSON converts a Progress struct to JSON string
func ProgressToJSON(p *Progress) (*string, error) {
	if p == nil {
		return nil, nil
	}

	data, err := json.Marshal(p)
	if err != nil {
		return nil, err
	}

	str := string(data)
	return &str, nil
}

// JSONToProgress converts a JSON string to Progress struct
func JSONToProgress(s *string) (*Progress, error) {
	if s == nil || *s == "" {
		return nil, nil
	}

	var p Progress
	if err := json.Unmarshal([]byte(*s), &p); err != nil {
		return nil, err
	}

	return &p, nil
}
