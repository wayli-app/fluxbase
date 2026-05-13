package jobs

import (
	"context"

	"github.com/jackc/pgx/v5"

	"github.com/nimbleflux/fluxbase/internal/database"
)

// ========== Namespace Functions ==========

// ListJobNamespaces returns all unique namespaces that have job functions (admin access, bypasses RLS)
func (s *Storage) ListJobNamespaces(ctx context.Context) ([]string, error) {
	query := `SELECT DISTINCT namespace FROM jobs.functions WHERE (tenant_id = $1 OR ($1 IS NULL AND tenant_id IS NULL)) ORDER BY namespace`

	var namespaces []string

	err := s.WithTenant(ctx, func(tx pgx.Tx) error {
		rows, err := tx.Query(ctx, query, database.TenantOrNil(database.TenantFromContext(ctx)))
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

// ListAllScheduledJobFunctions lists all enabled scheduled job functions across all tenants.
// Used by the scheduler which runs cross-tenant.
func (s *Storage) ListAllScheduledJobFunctions(ctx context.Context) ([]*JobFunctionSummary, error) {
	query := `
		SELECT id, name, namespace, description, is_bundled, bundle_error,
			enabled, schedule, timeout_seconds, memory_limit_mb, max_retries,
			progress_timeout_seconds, allow_net, allow_env, allow_read, allow_write, require_roles, disable_execution_logs,
			version, created_by, source, COALESCE(tenant_id::text, ''), created_at, updated_at
		FROM jobs.functions
		WHERE enabled = true AND schedule IS NOT NULL AND schedule != ''
		ORDER BY namespace, name
	`

	var functions []*JobFunctionSummary
	err := database.WrapWithServiceRole(ctx, s.DB, func(tx pgx.Tx) error {
		rows, err := tx.Query(ctx, query)
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
