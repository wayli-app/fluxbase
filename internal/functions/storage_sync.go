package functions

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"

	"github.com/nimbleflux/fluxbase/internal/database"
)

// GetFunctionForSync retrieves a function by name, matching either the given tenant or NULL tenant_id.
// Used by sync/reload flows to find existing functions regardless of backfill state.
func (s *Storage) GetFunctionForSync(ctx context.Context, name string, tenantID string) (*EdgeFunction, error) {
	query := `
		SELECT id, name, namespace, description, code, original_code, is_bundled, bundle_error, version, cron_schedule, enabled,
		       timeout_seconds, memory_limit_mb, allow_net, allow_env, allow_read, allow_write, allowed_domains, allow_unauthenticated, is_public, disable_execution_logs,
		       cors_origins, cors_methods, cors_headers, cors_credentials, cors_max_age,
		       rate_limit_per_minute, rate_limit_per_hour, rate_limit_per_day,
		       created_at, updated_at, created_by, source, tenant_id
		FROM functions.edge_functions
		WHERE name = $1
		  AND (tenant_id = $2 OR tenant_id IS NULL)
		ORDER BY namespace
		LIMIT 1
	`

	fn := &EdgeFunction{}
	err := database.WrapWithServiceRoleAndTenant(ctx, s.DB, tenantID, func(tx pgx.Tx) error {
		return tx.QueryRow(ctx, query, name, database.TenantOrNil(tenantID)).Scan(
			&fn.ID, &fn.Name, &fn.Namespace, &fn.Description, &fn.Code, &fn.OriginalCode, &fn.IsBundled, &fn.BundleError,
			&fn.Version, &fn.CronSchedule, &fn.Enabled,
			&fn.TimeoutSeconds, &fn.MemoryLimitMB, &fn.AllowNet, &fn.AllowEnv, &fn.AllowRead, &fn.AllowWrite, &fn.AllowedDomains, &fn.AllowUnauthenticated, &fn.IsPublic, &fn.DisableExecutionLogs,
			&fn.CorsOrigins, &fn.CorsMethods, &fn.CorsHeaders, &fn.CorsCredentials, &fn.CorsMaxAge,
			&fn.RateLimitPerMinute, &fn.RateLimitPerHour, &fn.RateLimitPerDay,
			&fn.CreatedAt, &fn.UpdatedAt, &fn.CreatedBy, &fn.Source, &fn.TenantID,
		)
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get function: %w", err)
	}

	return fn, nil
}

// ListFunctionsForSync returns all public functions matching the given tenant OR with NULL tenant_id.
// Used by the reload flow to find existing functions regardless of backfill state.
func (s *Storage) ListFunctionsForSync(ctx context.Context, tenantID string) ([]EdgeFunctionSummary, error) {
	query := `
		SELECT id, name, namespace, description, is_bundled, bundle_error, version, cron_schedule, enabled,
		       timeout_seconds, memory_limit_mb, allow_net, allow_env, allow_read, allow_write, allowed_domains, allow_unauthenticated, is_public, disable_execution_logs,
		       cors_origins, cors_methods, cors_headers, cors_credentials, cors_max_age,
		       rate_limit_per_minute, rate_limit_per_hour, rate_limit_per_day,
		       created_at, updated_at, created_by, source, tenant_id
		FROM functions.edge_functions
		WHERE is_public = true
		  AND (tenant_id = $1 OR tenant_id IS NULL)
		ORDER BY created_at DESC
	`

	var functions []EdgeFunctionSummary
	err := database.WrapWithServiceRoleAndTenant(ctx, s.DB, tenantID, func(tx pgx.Tx) error {
		rows, err := tx.Query(ctx, query, database.TenantOrNil(tenantID))
		if err != nil {
			return err
		}
		defer rows.Close()

		for rows.Next() {
			fn := EdgeFunctionSummary{}
			err := rows.Scan(
				&fn.ID, &fn.Name, &fn.Namespace, &fn.Description, &fn.IsBundled, &fn.BundleError,
				&fn.Version, &fn.CronSchedule, &fn.Enabled,
				&fn.TimeoutSeconds, &fn.MemoryLimitMB, &fn.AllowNet, &fn.AllowEnv, &fn.AllowRead, &fn.AllowWrite, &fn.AllowedDomains, &fn.AllowUnauthenticated, &fn.IsPublic, &fn.DisableExecutionLogs,
				&fn.CorsOrigins, &fn.CorsMethods, &fn.CorsHeaders, &fn.CorsCredentials, &fn.CorsMaxAge,
				&fn.RateLimitPerMinute, &fn.RateLimitPerHour, &fn.RateLimitPerDay,
				&fn.CreatedAt, &fn.UpdatedAt, &fn.CreatedBy, &fn.Source, &fn.TenantID,
			)
			if err != nil {
				return err
			}
			functions = append(functions, fn)
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list functions for sync: %w", err)
	}

	return functions, nil
}

// ListFunctionsByNamespaceForSync returns all functions matching the given tenant OR with NULL tenant_id.
// This is used by the sync flow to find existing functions regardless of whether they
// have been backfilled to the current tenant or still have NULL tenant_id from pre-tenancy.
func (s *Storage) ListFunctionsByNamespaceForSync(ctx context.Context, namespace string, tenantID string) ([]EdgeFunctionSummary, error) {
	query := `
		SELECT id, name, namespace, description, is_bundled, bundle_error, version, cron_schedule, enabled,
		       timeout_seconds, memory_limit_mb, allow_net, allow_env, allow_read, allow_write, allowed_domains, allow_unauthenticated, is_public, disable_execution_logs,
		       cors_origins, cors_methods, cors_headers, cors_credentials, cors_max_age,
		       rate_limit_per_minute, rate_limit_per_hour, rate_limit_per_day,
		       created_at, updated_at, created_by, source, tenant_id
		FROM functions.edge_functions
		WHERE namespace = $1
		  AND (tenant_id = $2 OR tenant_id IS NULL)
		ORDER BY created_at DESC
	`

	var functions []EdgeFunctionSummary
	err := database.WrapWithServiceRoleAndTenant(ctx, s.DB, tenantID, func(tx pgx.Tx) error {
		rows, err := tx.Query(ctx, query, namespace, database.TenantOrNil(tenantID))
		if err != nil {
			return err
		}
		defer rows.Close()

		for rows.Next() {
			fn := EdgeFunctionSummary{}
			err := rows.Scan(
				&fn.ID, &fn.Name, &fn.Namespace, &fn.Description, &fn.IsBundled, &fn.BundleError,
				&fn.Version, &fn.CronSchedule, &fn.Enabled,
				&fn.TimeoutSeconds, &fn.MemoryLimitMB, &fn.AllowNet, &fn.AllowEnv, &fn.AllowRead, &fn.AllowWrite, &fn.AllowedDomains, &fn.AllowUnauthenticated, &fn.IsPublic, &fn.DisableExecutionLogs,
				&fn.CorsOrigins, &fn.CorsMethods, &fn.CorsHeaders, &fn.CorsCredentials, &fn.CorsMaxAge,
				&fn.RateLimitPerMinute, &fn.RateLimitPerHour, &fn.RateLimitPerDay,
				&fn.CreatedAt, &fn.UpdatedAt, &fn.CreatedBy, &fn.Source, &fn.TenantID,
			)
			if err != nil {
				return err
			}
			functions = append(functions, fn)
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list functions for sync: %w", err)
	}

	return functions, nil
}

// ListAllFunctionsAllTenants returns all functions with cron schedules across all tenants.
// Used by the scheduler to load cron-enabled functions without tenant filtering.
func (s *Storage) ListAllFunctionsAllTenants(ctx context.Context) ([]EdgeFunctionSummary, error) {
	query := `
		SELECT id, name, namespace, description, is_bundled, bundle_error, version, cron_schedule, enabled,
		       timeout_seconds, memory_limit_mb, allow_net, allow_env, allow_read, allow_write, allowed_domains, allow_unauthenticated, is_public, disable_execution_logs,
		       cors_origins, cors_methods, cors_headers, cors_credentials, cors_max_age,
		       rate_limit_per_minute, rate_limit_per_hour, rate_limit_per_day,
		       created_at, updated_at, created_by, source, tenant_id
		FROM functions.edge_functions
		WHERE cron_schedule IS NOT NULL AND cron_schedule != ''
		ORDER BY namespace, name
	`

	var functions []EdgeFunctionSummary
	err := database.WrapWithServiceRole(ctx, s.DB, func(tx pgx.Tx) error {
		rows, err := tx.Query(ctx, query)
		if err != nil {
			return err
		}
		defer rows.Close()

		for rows.Next() {
			fn := EdgeFunctionSummary{}
			err := rows.Scan(
				&fn.ID, &fn.Name, &fn.Namespace, &fn.Description, &fn.IsBundled, &fn.BundleError,
				&fn.Version, &fn.CronSchedule, &fn.Enabled,
				&fn.TimeoutSeconds, &fn.MemoryLimitMB, &fn.AllowNet, &fn.AllowEnv, &fn.AllowRead, &fn.AllowWrite, &fn.AllowedDomains, &fn.AllowUnauthenticated, &fn.IsPublic, &fn.DisableExecutionLogs,
				&fn.CorsOrigins, &fn.CorsMethods, &fn.CorsHeaders, &fn.CorsCredentials, &fn.CorsMaxAge,
				&fn.RateLimitPerMinute, &fn.RateLimitPerHour, &fn.RateLimitPerDay,
				&fn.CreatedAt, &fn.UpdatedAt, &fn.CreatedBy, &fn.Source, &fn.TenantID,
			)
			if err != nil {
				return err
			}
			functions = append(functions, fn)
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list all functions across tenants: %w", err)
	}

	return functions, nil
}

// UpdateFunctionForSync updates a function matching the given tenant OR NULL tenant_id.
// Used by sync/reload flows to update functions regardless of backfill state.
func (s *Storage) UpdateFunctionForSync(ctx context.Context, name string, tenantID string, updates map[string]interface{}) error {
	query := "UPDATE functions.edge_functions SET "
	args := []interface{}{}
	argCount := 1

	for key, value := range updates {
		if !allowedFunctionColumns[key] {
			continue
		}
		if argCount > 1 {
			query += ", "
		}
		query += fmt.Sprintf("%s = $%d", key, argCount)
		args = append(args, value)
		argCount++
	}

	query += fmt.Sprintf(" WHERE name = $%d AND namespace = 'default'", argCount)
	args = append(args, name)

	query += fmt.Sprintf(" AND (tenant_id = $%d OR tenant_id IS NULL)", argCount+1)
	args = append(args, database.TenantOrNil(tenantID))

	err := database.WrapWithServiceRoleAndTenant(ctx, s.DB, tenantID, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx, query, args...)
		return err
	})
	if err != nil {
		return fmt.Errorf("failed to update function for sync: %w", err)
	}

	return nil
}

// UpdateFunctionByNamespaceForSync updates a function by name+namespace matching the given tenant OR NULL tenant_id.
func (s *Storage) UpdateFunctionByNamespaceForSync(ctx context.Context, name string, namespace string, tenantID string, updates map[string]interface{}) error {
	query := "UPDATE functions.edge_functions SET "
	args := []interface{}{}
	argCount := 1

	for key, value := range updates {
		if !allowedFunctionColumns[key] {
			continue
		}
		if argCount > 1 {
			query += ", "
		}
		query += fmt.Sprintf("%s = $%d", key, argCount)
		args = append(args, value)
		argCount++
	}

	query += fmt.Sprintf(" WHERE name = $%d AND namespace = $%d", argCount, argCount+1)
	args = append(args, name, namespace)

	query += fmt.Sprintf(" AND (tenant_id = $%d OR tenant_id IS NULL)", argCount+2)
	args = append(args, database.TenantOrNil(tenantID))

	err := database.WrapWithServiceRoleAndTenant(ctx, s.DB, tenantID, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx, query, args...)
		return err
	})
	if err != nil {
		return fmt.Errorf("failed to update function for sync: %w", err)
	}

	return nil
}

// DeleteFunctionForSync deletes a function matching the given tenant OR NULL tenant_id.
func (s *Storage) DeleteFunctionForSync(ctx context.Context, name string, namespace string, tenantID string) error {
	query := "DELETE FROM functions.edge_functions WHERE name = $1 AND namespace = $2 AND (tenant_id = $3 OR tenant_id IS NULL)"
	err := database.WrapWithServiceRoleAndTenant(ctx, s.DB, tenantID, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx, query, name, namespace, database.TenantOrNil(tenantID))
		return err
	})
	if err != nil {
		return fmt.Errorf("failed to delete function for sync: %w", err)
	}
	return nil
}
