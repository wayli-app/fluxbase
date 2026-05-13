package functions

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/nimbleflux/fluxbase/internal/database"
)

var allowedFunctionColumns = map[string]bool{
	"name": true, "namespace": true, "description": true, "code": true,
	"original_code": true, "is_bundled": true, "bundle_error": true,
	"enabled": true, "timeout_seconds": true, "memory_limit_mb": true,
	"allow_net": true, "allow_env": true, "allow_read": true,
	"allow_write": true, "allowed_domains": true, "allow_unauthenticated": true, "is_public": true,
	"cron_schedule": true, "version": true, "created_by": true,
	"source": true, "needs_rebundle": true, "cors_origins": true,
	"cors_methods": true, "cors_headers": true, "cors_credentials": true,
	"cors_max_age": true, "disable_execution_logs": true,
	"rate_limit_per_minute": true, "rate_limit_per_hour": true,
	"rate_limit_per_day": true,
}

// EdgeFunction represents a stored edge function
type EdgeFunction struct {
	ID                   uuid.UUID `json:"id"`
	Name                 string    `json:"name"`
	Namespace            string    `json:"namespace"` // Namespace for isolating functions across apps/deployments
	Description          *string   `json:"description"`
	Code                 string    `json:"code"`          // Bundled code (for execution)
	OriginalCode         *string   `json:"original_code"` // Original code before bundling (for editing)
	IsBundled            bool      `json:"is_bundled"`    // Whether code is bundled
	BundleError          *string   `json:"bundle_error"`  // Error if bundling failed
	Version              int       `json:"version"`
	CronSchedule         *string   `json:"cron_schedule"`
	Enabled              bool      `json:"enabled"`
	TimeoutSeconds       int       `json:"timeout_seconds"`
	MemoryLimitMB        int       `json:"memory_limit_mb"`
	AllowNet             bool      `json:"allow_net"`
	AllowEnv             bool      `json:"allow_env"`
	AllowRead            bool      `json:"allow_read"`
	AllowWrite           bool      `json:"allow_write"`
	AllowedDomains       *string   `json:"allowed_domains"`
	AllowUnauthenticated bool      `json:"allow_unauthenticated"`  // Allow invocation without authentication
	IsPublic             bool      `json:"is_public"`              // Whether function is publicly listed
	DisableExecutionLogs bool      `json:"disable_execution_logs"` // Disable execution log creation
	// CORS configuration (nil means use global defaults from FLUXBASE_CORS_* env vars)
	CorsOrigins     *string `json:"cors_origins"`
	CorsMethods     *string `json:"cors_methods"`
	CorsHeaders     *string `json:"cors_headers"`
	CorsCredentials *bool   `json:"cors_credentials"`
	CorsMaxAge      *int    `json:"cors_max_age"`
	// Rate limiting configuration (nil means unlimited)
	RateLimitPerMinute *int       `json:"rate_limit_per_minute"`
	RateLimitPerHour   *int       `json:"rate_limit_per_hour"`
	RateLimitPerDay    *int       `json:"rate_limit_per_day"`
	CreatedAt          time.Time  `json:"created_at"`
	UpdatedAt          time.Time  `json:"updated_at"`
	CreatedBy          *uuid.UUID `json:"created_by"`
	Source             string     `json:"source"` // "filesystem" or "api"
	TenantID           *string    `json:"tenant_id"`
}

// EdgeFunctionSummary is a lightweight version of EdgeFunction for list responses (excludes code fields)
type EdgeFunctionSummary struct {
	ID                   uuid.UUID  `json:"id"`
	Name                 string     `json:"name"`
	Namespace            string     `json:"namespace"`
	Description          *string    `json:"description"`
	IsBundled            bool       `json:"is_bundled"`
	BundleError          *string    `json:"bundle_error"`
	Version              int        `json:"version"`
	CronSchedule         *string    `json:"cron_schedule"`
	Enabled              bool       `json:"enabled"`
	TimeoutSeconds       int        `json:"timeout_seconds"`
	MemoryLimitMB        int        `json:"memory_limit_mb"`
	AllowNet             bool       `json:"allow_net"`
	AllowEnv             bool       `json:"allow_env"`
	AllowRead            bool       `json:"allow_read"`
	AllowWrite           bool       `json:"allow_write"`
	AllowedDomains       *string    `json:"allowed_domains"`
	AllowUnauthenticated bool       `json:"allow_unauthenticated"`
	IsPublic             bool       `json:"is_public"`
	DisableExecutionLogs bool       `json:"disable_execution_logs"`
	CorsOrigins          *string    `json:"cors_origins"`
	CorsMethods          *string    `json:"cors_methods"`
	CorsHeaders          *string    `json:"cors_headers"`
	CorsCredentials      *bool      `json:"cors_credentials"`
	CorsMaxAge           *int       `json:"cors_max_age"`
	RateLimitPerMinute   *int       `json:"rate_limit_per_minute"`
	RateLimitPerHour     *int       `json:"rate_limit_per_hour"`
	RateLimitPerDay      *int       `json:"rate_limit_per_day"`
	CreatedAt            time.Time  `json:"created_at"`
	UpdatedAt            time.Time  `json:"updated_at"`
	CreatedBy            *uuid.UUID `json:"created_by"`
	Source               string     `json:"source"` // "filesystem" or "api"
	TenantID             *string    `json:"tenant_id"`
}

// EdgeFunctionExecution represents a function execution log
type EdgeFunctionExecution struct {
	ID             uuid.UUID  `json:"id"`
	FunctionID     uuid.UUID  `json:"function_id"`
	TriggerType    string     `json:"trigger_type"`
	TriggerPayload *string    `json:"trigger_payload"`
	Status         string     `json:"status"`
	StatusCode     *int       `json:"status_code"`
	DurationMs     *int       `json:"duration_ms"`
	Result         *string    `json:"result"`
	Logs           *string    `json:"logs"`
	ErrorMessage   *string    `json:"error_message"`
	ErrorStack     *string    `json:"error_stack"`
	ExecutedAt     time.Time  `json:"executed_at"`
	CompletedAt    *time.Time `json:"completed_at"`
}

// FunctionFile represents a supporting file for an edge function
type FunctionFile struct {
	ID         uuid.UUID `json:"id"`
	FunctionID uuid.UUID `json:"function_id"`
	FilePath   string    `json:"file_path"` // e.g., "utils.ts", "helpers/db.ts"
	Content    string    `json:"content"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

// SharedModule represents a shared module accessible by all edge functions
type SharedModule struct {
	ID          uuid.UUID  `json:"id"`
	ModulePath  string     `json:"module_path"` // e.g., "_shared/cors.ts"
	Content     string     `json:"content"`
	Description *string    `json:"description"`
	Version     int        `json:"version"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	CreatedBy   *uuid.UUID `json:"created_by"`
}

// Storage manages edge function persistence
type Storage struct {
	database.TenantAware
}

// NewStorage creates a new storage manager
func NewStorage(db *database.Connection) *Storage {
	return &Storage{TenantAware: database.TenantAware{DB: db}}
}

// CreateFunction creates a new edge function
func (s *Storage) CreateFunction(ctx context.Context, fn *EdgeFunction) error {
	query := `
		INSERT INTO functions.edge_functions (
			name, namespace, description, code, original_code, is_bundled, bundle_error,
			enabled, timeout_seconds, memory_limit_mb,
			allow_net, allow_env, allow_read, allow_write, allowed_domains, allow_unauthenticated, is_public, disable_execution_logs,
			cors_origins, cors_methods, cors_headers, cors_credentials, cors_max_age,
			rate_limit_per_minute, rate_limit_per_hour, rate_limit_per_day,
			cron_schedule, created_by, source
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21, $22, $23, $24, $25, $26, $27, $28, $29)
		RETURNING id, version, created_at, updated_at
	`

	err := s.WithTenant(ctx, func(tx pgx.Tx) error {
		return tx.QueryRow(
			ctx, query,
			fn.Name, fn.Namespace, fn.Description, fn.Code, fn.OriginalCode, fn.IsBundled, fn.BundleError,
			fn.Enabled, fn.TimeoutSeconds, fn.MemoryLimitMB,
			fn.AllowNet, fn.AllowEnv, fn.AllowRead, fn.AllowWrite, fn.AllowedDomains, fn.AllowUnauthenticated, fn.IsPublic, fn.DisableExecutionLogs,
			fn.CorsOrigins, fn.CorsMethods, fn.CorsHeaders, fn.CorsCredentials, fn.CorsMaxAge,
			fn.RateLimitPerMinute, fn.RateLimitPerHour, fn.RateLimitPerDay,
			fn.CronSchedule, fn.CreatedBy, fn.Source,
		).Scan(&fn.ID, &fn.Version, &fn.CreatedAt, &fn.UpdatedAt)
	})
	if err != nil {
		return fmt.Errorf("failed to create function: %w", err)
	}

	return nil
}

// GetFunction retrieves the first function matching the name (any namespace)
// Results are ordered alphabetically by namespace, so "default" is preferred if it exists
func (s *Storage) GetFunction(ctx context.Context, name string) (*EdgeFunction, error) {
	tenantID := database.TenantFromContext(ctx)

	query := `
		SELECT id, name, namespace, description, code, original_code, is_bundled, bundle_error, version, cron_schedule, enabled,
		       timeout_seconds, memory_limit_mb, allow_net, allow_env, allow_read, allow_write, allowed_domains, allow_unauthenticated, is_public, disable_execution_logs,
		       cors_origins, cors_methods, cors_headers, cors_credentials, cors_max_age,
		       rate_limit_per_minute, rate_limit_per_hour, rate_limit_per_day,
		       created_at, updated_at, created_by, source, tenant_id
		FROM functions.edge_functions
		WHERE name = $1
		  AND (tenant_id = $2 OR ($2 IS NULL AND tenant_id IS NULL))
		ORDER BY namespace
		LIMIT 1
	`

	fn := &EdgeFunction{}
	err := s.WithTenant(ctx, func(tx pgx.Tx) error {
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

// GetFunctionByNamespace retrieves a function by name and namespace
func (s *Storage) GetFunctionByNamespace(ctx context.Context, name string, namespace string) (*EdgeFunction, error) {
	tenantID := database.TenantFromContext(ctx)

	query := `
		SELECT id, name, namespace, description, code, original_code, is_bundled, bundle_error, version, cron_schedule, enabled,
		       timeout_seconds, memory_limit_mb, allow_net, allow_env, allow_read, allow_write, allowed_domains, allow_unauthenticated, is_public, disable_execution_logs,
		       cors_origins, cors_methods, cors_headers, cors_credentials, cors_max_age,
		       rate_limit_per_minute, rate_limit_per_hour, rate_limit_per_day,
		       created_at, updated_at, created_by, source, tenant_id
		FROM functions.edge_functions
		WHERE name = $1 AND namespace = $2
		  AND (tenant_id = $3 OR ($3 IS NULL AND tenant_id IS NULL))
	`

	fn := &EdgeFunction{}
	err := s.WithTenant(ctx, func(tx pgx.Tx) error {
		return tx.QueryRow(ctx, query, name, namespace, database.TenantOrNil(tenantID)).Scan(
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

// ListFunctions returns all public functions (is_public=true), excludes code for performance
func (s *Storage) ListFunctions(ctx context.Context) ([]EdgeFunctionSummary, error) {
	tenantID := database.TenantFromContext(ctx)

	query := `
		SELECT id, name, namespace, description, is_bundled, bundle_error, version, cron_schedule, enabled,
		       timeout_seconds, memory_limit_mb, allow_net, allow_env, allow_read, allow_write, allowed_domains, allow_unauthenticated, is_public, disable_execution_logs,
		       cors_origins, cors_methods, cors_headers, cors_credentials, cors_max_age,
		       rate_limit_per_minute, rate_limit_per_hour, rate_limit_per_day,
		       created_at, updated_at, created_by, source, tenant_id
		FROM functions.edge_functions
		WHERE is_public = true
		  AND (tenant_id = $1 OR ($1 IS NULL AND tenant_id IS NULL))
		ORDER BY created_at DESC
	`

	var functions []EdgeFunctionSummary
	err := s.WithTenant(ctx, func(tx pgx.Tx) error {
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
		return nil, fmt.Errorf("failed to list functions: %w", err)
	}

	return functions, nil
}

// ListAllFunctions returns all functions regardless of is_public setting (admin use)
func (s *Storage) ListAllFunctions(ctx context.Context) ([]EdgeFunctionSummary, error) {
	tenantID := database.TenantFromContext(ctx)

	query := `
		SELECT id, name, namespace, description, is_bundled, bundle_error, version, cron_schedule, enabled,
		       timeout_seconds, memory_limit_mb, allow_net, allow_env, allow_read, allow_write, allowed_domains, allow_unauthenticated, is_public, disable_execution_logs,
		       cors_origins, cors_methods, cors_headers, cors_credentials, cors_max_age,
		       rate_limit_per_minute, rate_limit_per_hour, rate_limit_per_day,
		       created_at, updated_at, created_by, source, tenant_id
		FROM functions.edge_functions
		WHERE (tenant_id = $1 OR ($1 IS NULL AND tenant_id IS NULL))
		ORDER BY namespace, name
	`

	var functions []EdgeFunctionSummary
	err := s.WithTenant(ctx, func(tx pgx.Tx) error {
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
		return nil, fmt.Errorf("failed to list all functions: %w", err)
	}

	return functions, nil
}

// ListFunctionNamespaces returns all unique namespaces that have edge functions
func (s *Storage) ListFunctionNamespaces(ctx context.Context) ([]string, error) {
	tenantID := database.TenantFromContext(ctx)

	query := `SELECT DISTINCT namespace FROM functions.edge_functions WHERE (tenant_id = $1 OR ($1 IS NULL AND tenant_id IS NULL)) ORDER BY namespace`

	var namespaces []string
	err := s.WithTenant(ctx, func(tx pgx.Tx) error {
		rows, err := tx.Query(ctx, query, database.TenantOrNil(tenantID))
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
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list function namespaces: %w", err)
	}

	return namespaces, nil
}

// ListFunctionsByNamespace returns all functions in a specific namespace, excludes code for performance
func (s *Storage) ListFunctionsByNamespace(ctx context.Context, namespace string) ([]EdgeFunctionSummary, error) {
	tenantID := database.TenantFromContext(ctx)

	query := `
		SELECT id, name, namespace, description, is_bundled, bundle_error, version, cron_schedule, enabled,
		       timeout_seconds, memory_limit_mb, allow_net, allow_env, allow_read, allow_write, allowed_domains, allow_unauthenticated, is_public, disable_execution_logs,
		       cors_origins, cors_methods, cors_headers, cors_credentials, cors_max_age,
		       rate_limit_per_minute, rate_limit_per_hour, rate_limit_per_day,
		       created_at, updated_at, created_by, source, tenant_id
		FROM functions.edge_functions
		WHERE namespace = $1
		  AND (tenant_id = $2 OR ($2 IS NULL AND tenant_id IS NULL))
		ORDER BY created_at DESC
	`

	var functions []EdgeFunctionSummary
	err := s.WithTenant(ctx, func(tx pgx.Tx) error {
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
		return nil, fmt.Errorf("failed to list functions by namespace: %w", err)
	}

	return functions, nil
}

// UpdateFunction updates an existing function (uses default namespace for backwards compatibility)
func (s *Storage) UpdateFunction(ctx context.Context, name string, updates map[string]interface{}) error {
	return s.UpdateFunctionByNamespace(ctx, name, "default", updates)
}

// UpdateFunctionByNamespace updates an existing function in a specific namespace
func (s *Storage) UpdateFunctionByNamespace(ctx context.Context, name string, namespace string, updates map[string]interface{}) error {
	tenantID := database.TenantFromContext(ctx)

	// Build dynamic UPDATE query
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

	// Add tenant filter to prevent cross-tenant updates
	query += fmt.Sprintf(" AND (tenant_id = $%d OR ($%d IS NULL AND tenant_id IS NULL))", argCount+2, argCount+2)
	args = append(args, database.TenantOrNil(tenantID))

	err := s.WithTenant(ctx, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx, query, args...)
		return err
	})
	if err != nil {
		return fmt.Errorf("failed to update function: %w", err)
	}

	return nil
}

// DeleteFunction deletes a function by name (uses default namespace for backwards compatibility)
func (s *Storage) DeleteFunction(ctx context.Context, name string) error {
	return s.DeleteFunctionByNamespace(ctx, name, "default")
}

// DeleteFunctionByNamespace deletes an existing function in a specific namespace
func (s *Storage) DeleteFunctionByNamespace(ctx context.Context, name string, namespace string) error {
	tenantID := database.TenantFromContext(ctx)

	query := "DELETE FROM functions.edge_functions WHERE name = $1 AND namespace = $2 AND (tenant_id = $3 OR ($3 IS NULL AND tenant_id IS NULL))"
	err := s.WithTenant(ctx, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx, query, name, namespace, database.TenantOrNil(tenantID))
		return err
	})
	if err != nil {
		return fmt.Errorf("failed to delete function: %w", err)
	}
	return nil
}
