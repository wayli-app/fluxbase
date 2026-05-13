package jobs

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/nimbleflux/fluxbase/internal/database"
)

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
		WHERE q.id = $1 AND (q.tenant_id = $2 OR ($2 IS NULL AND q.tenant_id IS NULL))
	`

	tenantID := database.TenantFromContext(ctx)

	var job Job
	err := database.WrapWithServiceRoleAndTenant(ctx, s.DB, tenantID, func(tx pgx.Tx) error {
		return tx.QueryRow(ctx, query, jobID, database.TenantOrNil(tenantID)).Scan(
			&job.ID, &job.Namespace, &job.JobFunctionID, &job.JobName, &job.Status,
			&job.Payload, &job.Result, &job.Progress, &job.Priority,
			&job.MaxDurationSeconds, &job.ProgressTimeoutSeconds, &job.MaxRetries,
			&job.RetryCount, &job.ErrorMessage, &job.WorkerID, &job.CreatedBy, &job.UserRole, &job.UserEmail, &job.UserName,
			&job.CreatedAt, &job.ScheduledAt, &job.StartedAt, &job.LastProgressAt, &job.CompletedAt,
		)
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
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
	tenantID := database.TenantFromContext(ctx)

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

	// Tenant filter (first dynamic filter)
	query += fmt.Sprintf(" AND (q.tenant_id = $%d OR ($%d IS NULL AND q.tenant_id IS NULL))", argCount, argCount)
	args = append(args, database.TenantOrNil(tenantID))
	argCount++

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

	var jobs []*Job
	err := database.WrapWithServiceRoleAndTenant(ctx, s.DB, tenantID, func(tx pgx.Tx) error {
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

func (s *Storage) GetJobByIDAdmin(ctx context.Context, jobID uuid.UUID) (*Job, error) {
	return s.GetJob(ctx, jobID)
}

func (s *Storage) ListJobsAdmin(ctx context.Context, filters *JobFilters) ([]*Job, error) {
	return s.ListJobs(ctx, filters)
}

// GetJobStats retrieves aggregate statistics about jobs (admin access, bypasses RLS)
func (s *Storage) GetJobStats(ctx context.Context, namespace *string) (*JobStats, error) {
	stats := &JobStats{}

	tenantID := database.TenantFromContext(ctx)

	var args []interface{}
	args = append(args, database.TenantOrNil(tenantID))
	if namespace != nil {
		args = append(args, *namespace)
	}

	err := s.WithTenant(ctx, func(tx pgx.Tx) error {
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
			WHERE (tenant_id = $1 OR ($1 IS NULL AND tenant_id IS NULL))
		`
		if namespace != nil {
			countQuery += " AND namespace = $2"
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
				WHERE (tenant_id = $1 OR ($1 IS NULL AND tenant_id IS NULL))
			`
		if namespace != nil {
			statusQuery += " AND namespace = $2"
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
				AND (tenant_id = $1 OR ($1 IS NULL AND tenant_id IS NULL))
			`
		if namespace != nil {
			dayQuery += " AND namespace = $2"
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
				WHERE (tenant_id = $1 OR ($1 IS NULL AND tenant_id IS NULL))
			`
		if namespace != nil {
			funcQuery += " AND namespace = $2"
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
