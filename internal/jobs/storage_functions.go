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

// ========== Job Functions ==========

// CreateJobFunction creates a new job function
func (s *Storage) CreateJobFunction(ctx context.Context, fn *JobFunction) error {
	tenantID := database.TenantFromContext(ctx)
	return s.CreateJobFunctionWithTenant(ctx, tenantID, fn)
}

// CreateJobFunctionWithTenant creates a new job function with tenant context
func (s *Storage) CreateJobFunctionWithTenant(ctx context.Context, tenantID string, fn *JobFunction) error {
	query := `
		INSERT INTO jobs.functions (
			id, name, namespace, description, code, original_code, is_bundled, bundle_error,
			enabled, schedule, timeout_seconds, memory_limit_mb, max_retries,
			progress_timeout_seconds, allow_net, allow_env, allow_read, allow_write,
			require_roles, disable_execution_logs, version, created_by, source
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21, $22, $23
		)
		RETURNING created_at, updated_at
	`

	return database.WrapWithTenantAwareRole(ctx, s.DB, tenantID, func(tx pgx.Tx) error {
		return tx.QueryRow(
			ctx, query,
			fn.ID, fn.Name, fn.Namespace, fn.Description, fn.Code, fn.OriginalCode,
			fn.IsBundled, fn.BundleError, fn.Enabled, fn.Schedule, fn.TimeoutSeconds,
			fn.MemoryLimitMB, fn.MaxRetries, fn.ProgressTimeoutSeconds,
			fn.AllowNet, fn.AllowEnv, fn.AllowRead, fn.AllowWrite,
			fn.RequireRoles, fn.DisableExecutionLogs, fn.Version, fn.CreatedBy, fn.Source,
		).Scan(&fn.CreatedAt, &fn.UpdatedAt)
	})
}

// UpdateJobFunction updates an existing job function
func (s *Storage) UpdateJobFunction(ctx context.Context, fn *JobFunction) error {
	tenantID := database.TenantFromContext(ctx)
	return s.UpdateJobFunctionWithTenant(ctx, tenantID, fn)
}

// UpdateJobFunctionWithTenant updates an existing job function with tenant context
func (s *Storage) UpdateJobFunctionWithTenant(ctx context.Context, tenantID string, fn *JobFunction) error {
	query := `
		UPDATE jobs.functions SET
			description = $1, code = $2, original_code = $3, is_bundled = $4, bundle_error = $5,
			enabled = $6, schedule = $7, timeout_seconds = $8, memory_limit_mb = $9,
			max_retries = $10, progress_timeout_seconds = $11, allow_net = $12, allow_env = $13,
			allow_read = $14, allow_write = $15, require_roles = $16, disable_execution_logs = $17, version = version + 1
		WHERE id = $18
		RETURNING version, updated_at
	`

	return database.WrapWithTenantAwareRole(ctx, s.DB, tenantID, func(tx pgx.Tx) error {
		return tx.QueryRow(
			ctx, query,
			fn.Description, fn.Code, fn.OriginalCode, fn.IsBundled, fn.BundleError,
			fn.Enabled, fn.Schedule, fn.TimeoutSeconds, fn.MemoryLimitMB,
			fn.MaxRetries, fn.ProgressTimeoutSeconds, fn.AllowNet, fn.AllowEnv,
			fn.AllowRead, fn.AllowWrite, fn.RequireRoles, fn.DisableExecutionLogs, fn.ID,
		).Scan(&fn.Version, &fn.UpdatedAt)
	})
}

func (s *Storage) UpdateJobFunctionForSync(ctx context.Context, tenantID string, fn *JobFunction) error {
	query := `
		UPDATE jobs.functions SET
			description = $1, code = $2, original_code = $3, is_bundled = $4, bundle_error = $5,
			enabled = $6, schedule = $7, timeout_seconds = $8, memory_limit_mb = $9,
			max_retries = $10, progress_timeout_seconds = $11, allow_net = $12, allow_env = $13,
			allow_read = $14, allow_write = $15, require_roles = $16, disable_execution_logs = $17, version = version + 1
		WHERE id = $18
		RETURNING version, updated_at
	`

	return database.WrapWithTenantAwareRole(ctx, s.DB, tenantID, func(tx pgx.Tx) error {
		return tx.QueryRow(
			ctx, query,
			fn.Description, fn.Code, fn.OriginalCode, fn.IsBundled, fn.BundleError,
			fn.Enabled, fn.Schedule, fn.TimeoutSeconds, fn.MemoryLimitMB,
			fn.MaxRetries, fn.ProgressTimeoutSeconds, fn.AllowNet, fn.AllowEnv,
			fn.AllowRead, fn.AllowWrite, fn.RequireRoles, fn.DisableExecutionLogs, fn.ID,
		).Scan(&fn.Version, &fn.UpdatedAt)
	})
}

// UpsertJobFunction creates or updates a job function atomically
func (s *Storage) UpsertJobFunction(ctx context.Context, fn *JobFunction) error {
	tenantID := database.TenantFromContext(ctx)
	return s.UpsertJobFunctionWithTenant(ctx, tenantID, fn)
}

// UpsertJobFunctionWithTenant creates or updates a job function atomically with tenant context
func (s *Storage) UpsertJobFunctionWithTenant(ctx context.Context, tenantID string, fn *JobFunction) error {
	query := `
		INSERT INTO jobs.functions (
			id, name, namespace, description, code, original_code, is_bundled, bundle_error,
			enabled, schedule, timeout_seconds, memory_limit_mb, max_retries,
			progress_timeout_seconds, allow_net, allow_env, allow_read, allow_write,
			require_roles, disable_execution_logs, version, created_by, source
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, 1, $21, $22
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
			require_roles = EXCLUDED.require_roles,
			disable_execution_logs = EXCLUDED.disable_execution_logs,
			version = jobs.functions.version + 1,
			updated_at = NOW()
		RETURNING id, version, created_at, updated_at
	`

	return database.WrapWithTenantAwareRole(ctx, s.DB, tenantID, func(tx pgx.Tx) error {
		return tx.QueryRow(
			ctx, query,
			fn.ID, fn.Name, fn.Namespace, fn.Description, fn.Code, fn.OriginalCode,
			fn.IsBundled, fn.BundleError, fn.Enabled, fn.Schedule, fn.TimeoutSeconds,
			fn.MemoryLimitMB, fn.MaxRetries, fn.ProgressTimeoutSeconds,
			fn.AllowNet, fn.AllowEnv, fn.AllowRead, fn.AllowWrite,
			fn.RequireRoles, fn.DisableExecutionLogs, fn.CreatedBy, fn.Source,
		).Scan(&fn.ID, &fn.Version, &fn.CreatedAt, &fn.UpdatedAt)
	})
}

// GetJobFunction retrieves a job function by namespace and name
func (s *Storage) GetJobFunction(ctx context.Context, namespace, name string) (*JobFunction, error) {
	query := `
		SELECT id, name, namespace, description, code, original_code, is_bundled, bundle_error,
			enabled, schedule, timeout_seconds, memory_limit_mb, max_retries,
			progress_timeout_seconds, allow_net, allow_env, allow_read, allow_write, require_roles, disable_execution_logs,
			version, created_by, source, created_at, updated_at
		FROM jobs.functions
		WHERE namespace = $1 AND name = $2 AND (tenant_id = $3 OR ($3 IS NULL AND tenant_id IS NULL))
	`

	var fn JobFunction
	err := s.WithTenant(ctx, func(tx pgx.Tx) error {
		return tx.QueryRow(ctx, query, namespace, name, database.TenantOrNil(database.TenantFromContext(ctx))).Scan(
			&fn.ID, &fn.Name, &fn.Namespace, &fn.Description, &fn.Code, &fn.OriginalCode,
			&fn.IsBundled, &fn.BundleError, &fn.Enabled, &fn.Schedule, &fn.TimeoutSeconds,
			&fn.MemoryLimitMB, &fn.MaxRetries, &fn.ProgressTimeoutSeconds,
			&fn.AllowNet, &fn.AllowEnv, &fn.AllowRead, &fn.AllowWrite, &fn.RequireRoles, &fn.DisableExecutionLogs,
			&fn.Version, &fn.CreatedBy, &fn.Source, &fn.CreatedAt, &fn.UpdatedAt,
		)
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
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
			progress_timeout_seconds, allow_net, allow_env, allow_read, allow_write, require_roles, disable_execution_logs,
			version, created_by, source, created_at, updated_at
		FROM jobs.functions
		WHERE name = $1 AND (tenant_id = $2 OR ($2 IS NULL AND tenant_id IS NULL))
		ORDER BY namespace
		LIMIT 1
	`

	var fn JobFunction
	err := s.WithTenant(ctx, func(tx pgx.Tx) error {
		return tx.QueryRow(ctx, query, name, database.TenantOrNil(database.TenantFromContext(ctx))).Scan(
			&fn.ID, &fn.Name, &fn.Namespace, &fn.Description, &fn.Code, &fn.OriginalCode,
			&fn.IsBundled, &fn.BundleError, &fn.Enabled, &fn.Schedule, &fn.TimeoutSeconds,
			&fn.MemoryLimitMB, &fn.MaxRetries, &fn.ProgressTimeoutSeconds,
			&fn.AllowNet, &fn.AllowEnv, &fn.AllowRead, &fn.AllowWrite, &fn.RequireRoles, &fn.DisableExecutionLogs,
			&fn.Version, &fn.CreatedBy, &fn.Source, &fn.CreatedAt, &fn.UpdatedAt,
		)
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
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
			progress_timeout_seconds, allow_net, allow_env, allow_read, allow_write, require_roles, disable_execution_logs,
			version, created_by, source, created_at, updated_at
		FROM jobs.functions
		WHERE id = $1 AND (tenant_id = $2 OR ($2 IS NULL AND tenant_id IS NULL))
	`

	var fn JobFunction
	err := s.WithTenant(ctx, func(tx pgx.Tx) error {
		return tx.QueryRow(ctx, query, id, database.TenantOrNil(database.TenantFromContext(ctx))).Scan(
			&fn.ID, &fn.Name, &fn.Namespace, &fn.Description, &fn.Code, &fn.OriginalCode,
			&fn.IsBundled, &fn.BundleError, &fn.Enabled, &fn.Schedule, &fn.TimeoutSeconds,
			&fn.MemoryLimitMB, &fn.MaxRetries, &fn.ProgressTimeoutSeconds,
			&fn.AllowNet, &fn.AllowEnv, &fn.AllowRead, &fn.AllowWrite, &fn.RequireRoles, &fn.DisableExecutionLogs,
			&fn.Version, &fn.CreatedBy, &fn.Source, &fn.CreatedAt, &fn.UpdatedAt,
		)
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
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
			progress_timeout_seconds, allow_net, allow_env, allow_read, allow_write, require_roles, disable_execution_logs,
			version, created_by, source, COALESCE(tenant_id::text, ''), created_at, updated_at
		FROM jobs.functions
		WHERE namespace = $1 AND (tenant_id = $2 OR ($2 IS NULL AND tenant_id IS NULL))
		ORDER BY name
	`

	var functions []*JobFunctionSummary
	err := s.WithTenant(ctx, func(tx pgx.Tx) error {
		rows, err := tx.Query(ctx, query, namespace, database.TenantOrNil(database.TenantFromContext(ctx)))
		if err != nil {
			return err
		}
		defer rows.Close()

		for rows.Next() {
			var fn JobFunctionSummary
			if err := rows.Scan(
				&fn.ID, &fn.Name, &fn.Namespace, &fn.Description,
				&fn.IsBundled, &fn.BundleError, &fn.Enabled, &fn.Schedule, &fn.TimeoutSeconds,
				&fn.MemoryLimitMB, &fn.MaxRetries, &fn.ProgressTimeoutSeconds,
				&fn.AllowNet, &fn.AllowEnv, &fn.AllowRead, &fn.AllowWrite, &fn.RequireRoles, &fn.DisableExecutionLogs,
				&fn.Version, &fn.CreatedBy, &fn.Source, &fn.TenantID, &fn.CreatedAt, &fn.UpdatedAt,
			); err != nil {
				return err
			}
			functions = append(functions, &fn)
		}

		return rows.Err()
	})
	if err != nil {
		return nil, err
	}

	return functions, nil
}

// ListJobFunctionsForSync lists job functions matching the given tenant OR with NULL tenant_id.
// Used by sync flows to find existing functions regardless of backfill state.
func (s *Storage) ListJobFunctionsForSync(ctx context.Context, namespace string, tenantID string) ([]*JobFunctionSummary, error) {
	query := `
		SELECT id, name, namespace, description, is_bundled, bundle_error,
			enabled, schedule, timeout_seconds, memory_limit_mb, max_retries,
			progress_timeout_seconds, allow_net, allow_env, allow_read, allow_write, require_roles, disable_execution_logs,
			version, created_by, source, COALESCE(tenant_id::text, ''), created_at, updated_at
		FROM jobs.functions
		WHERE namespace = $1 AND (tenant_id = $2 OR tenant_id IS NULL)
		ORDER BY name
	`

	var functions []*JobFunctionSummary
	err := database.WrapWithServiceRoleAndTenant(ctx, s.DB, tenantID, func(tx pgx.Tx) error {
		rows, err := tx.Query(ctx, query, namespace, database.TenantOrNil(tenantID))
		if err != nil {
			return err
		}
		defer rows.Close()

		for rows.Next() {
			var fn JobFunctionSummary
			if err := rows.Scan(
				&fn.ID, &fn.Name, &fn.Namespace, &fn.Description,
				&fn.IsBundled, &fn.BundleError, &fn.Enabled, &fn.Schedule, &fn.TimeoutSeconds,
				&fn.MemoryLimitMB, &fn.MaxRetries, &fn.ProgressTimeoutSeconds,
				&fn.AllowNet, &fn.AllowEnv, &fn.AllowRead, &fn.AllowWrite, &fn.RequireRoles, &fn.DisableExecutionLogs,
				&fn.Version, &fn.CreatedBy, &fn.Source, &fn.TenantID, &fn.CreatedAt, &fn.UpdatedAt,
			); err != nil {
				return err
			}
			functions = append(functions, &fn)
		}

		return rows.Err()
	})
	if err != nil {
		return nil, err
	}

	return functions, nil
}

// ListAllJobFunctions lists all job functions across all namespaces (admin use)
func (s *Storage) ListAllJobFunctions(ctx context.Context) ([]*JobFunctionSummary, error) {
	query := `
		SELECT id, name, namespace, description, is_bundled, bundle_error,
			enabled, schedule, timeout_seconds, memory_limit_mb, max_retries,
			progress_timeout_seconds, allow_net, allow_env, allow_read, allow_write, require_roles, disable_execution_logs,
			version, created_by, source, COALESCE(tenant_id::text, ''), created_at, updated_at
		FROM jobs.functions
		WHERE (tenant_id = $1 OR ($1 IS NULL AND tenant_id IS NULL))
			ORDER BY namespace, name
	`

	var functions []*JobFunctionSummary
	err := s.WithTenant(ctx, func(tx pgx.Tx) error {
		rows, err := tx.Query(ctx, query, database.TenantOrNil(database.TenantFromContext(ctx)))
		if err != nil {
			return err
		}
		defer rows.Close()

		for rows.Next() {
			var fn JobFunctionSummary
			if err := rows.Scan(
				&fn.ID, &fn.Name, &fn.Namespace, &fn.Description,
				&fn.IsBundled, &fn.BundleError, &fn.Enabled, &fn.Schedule, &fn.TimeoutSeconds,
				&fn.MemoryLimitMB, &fn.MaxRetries, &fn.ProgressTimeoutSeconds,
				&fn.AllowNet, &fn.AllowEnv, &fn.AllowRead, &fn.AllowWrite, &fn.RequireRoles, &fn.DisableExecutionLogs,
				&fn.Version, &fn.CreatedBy, &fn.Source, &fn.TenantID, &fn.CreatedAt, &fn.UpdatedAt,
			); err != nil {
				return err
			}
			functions = append(functions, &fn)
		}

		return rows.Err()
	})
	if err != nil {
		return nil, err
	}

	return functions, nil
}

// DeleteJobFunction deletes a job function
func (s *Storage) DeleteJobFunction(ctx context.Context, namespace, name string) error {
	tenantID := database.TenantFromContext(ctx)
	return s.DeleteJobFunctionWithTenant(ctx, tenantID, namespace, name)
}

// DeleteJobFunctionWithTenant deletes a job function with tenant context
func (s *Storage) DeleteJobFunctionWithTenant(ctx context.Context, tenantID string, namespace, name string) error {
	query := `DELETE FROM jobs.functions WHERE namespace = $1 AND name = $2 AND (tenant_id = $3 OR ($3 IS NULL AND tenant_id IS NULL))`

	var result pgconn.CommandTag
	err := database.WrapWithTenantAwareRole(ctx, s.DB, tenantID, func(tx pgx.Tx) error {
		var execErr error
		result, execErr = tx.Exec(ctx, query, namespace, name, database.TenantOrNil(tenantID))
		return execErr
	})
	if err != nil {
		return err
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("job function not found: %s/%s", namespace, name)
	}

	return nil
}

func (s *Storage) DeleteJobFunctionForSync(ctx context.Context, tenantID string, namespace, name string) error {
	query := `DELETE FROM jobs.functions WHERE namespace = $1 AND name = $2 AND (tenant_id = $3 OR tenant_id IS NULL)`

	var result pgconn.CommandTag
	err := database.WrapWithTenantAwareRole(ctx, s.DB, tenantID, func(tx pgx.Tx) error {
		var execErr error
		result, execErr = tx.Exec(ctx, query, namespace, name, database.TenantOrNil(tenantID))
		return execErr
	})
	if err != nil {
		return err
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("job function not found: %s/%s", namespace, name)
	}

	return nil
}
