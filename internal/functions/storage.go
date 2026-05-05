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
	"allow_write": true, "allow_unauthenticated": true, "is_public": true,
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
			allow_net, allow_env, allow_read, allow_write, allow_unauthenticated, is_public, disable_execution_logs,
			cors_origins, cors_methods, cors_headers, cors_credentials, cors_max_age,
			rate_limit_per_minute, rate_limit_per_hour, rate_limit_per_day,
			cron_schedule, created_by, source
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21, $22, $23, $24, $25, $26, $27, $28)
		RETURNING id, version, created_at, updated_at
	`

	err := s.WithTenant(ctx, func(tx pgx.Tx) error {
		return tx.QueryRow(ctx, query,
			fn.Name, fn.Namespace, fn.Description, fn.Code, fn.OriginalCode, fn.IsBundled, fn.BundleError,
			fn.Enabled, fn.TimeoutSeconds, fn.MemoryLimitMB,
			fn.AllowNet, fn.AllowEnv, fn.AllowRead, fn.AllowWrite, fn.AllowUnauthenticated, fn.IsPublic, fn.DisableExecutionLogs,
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
		       timeout_seconds, memory_limit_mb, allow_net, allow_env, allow_read, allow_write, allow_unauthenticated, is_public, disable_execution_logs,
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
			&fn.TimeoutSeconds, &fn.MemoryLimitMB, &fn.AllowNet, &fn.AllowEnv, &fn.AllowRead, &fn.AllowWrite, &fn.AllowUnauthenticated, &fn.IsPublic, &fn.DisableExecutionLogs,
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

// GetFunctionForSync retrieves a function by name, matching either the given tenant or NULL tenant_id.
// Used by sync/reload flows to find existing functions regardless of backfill state.
func (s *Storage) GetFunctionForSync(ctx context.Context, name string, tenantID string) (*EdgeFunction, error) {
	query := `
		SELECT id, name, namespace, description, code, original_code, is_bundled, bundle_error, version, cron_schedule, enabled,
		       timeout_seconds, memory_limit_mb, allow_net, allow_env, allow_read, allow_write, allow_unauthenticated, is_public, disable_execution_logs,
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
	err := database.WrapWithServiceRole(ctx, s.DB, func(tx pgx.Tx) error {
		return tx.QueryRow(ctx, query, name, database.TenantOrNil(tenantID)).Scan(
			&fn.ID, &fn.Name, &fn.Namespace, &fn.Description, &fn.Code, &fn.OriginalCode, &fn.IsBundled, &fn.BundleError,
			&fn.Version, &fn.CronSchedule, &fn.Enabled,
			&fn.TimeoutSeconds, &fn.MemoryLimitMB, &fn.AllowNet, &fn.AllowEnv, &fn.AllowRead, &fn.AllowWrite, &fn.AllowUnauthenticated, &fn.IsPublic, &fn.DisableExecutionLogs,
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
		       timeout_seconds, memory_limit_mb, allow_net, allow_env, allow_read, allow_write, allow_unauthenticated, is_public, disable_execution_logs,
		       cors_origins, cors_methods, cors_headers, cors_credentials, cors_max_age,
		       rate_limit_per_minute, rate_limit_per_hour, rate_limit_per_day,
		       created_at, updated_at, created_by, source, tenant_id
		FROM functions.edge_functions
		WHERE is_public = true
		  AND (tenant_id = $1 OR tenant_id IS NULL)
		ORDER BY created_at DESC
	`

	var functions []EdgeFunctionSummary
	err := database.WrapWithServiceRole(ctx, s.DB, func(tx pgx.Tx) error {
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
				&fn.TimeoutSeconds, &fn.MemoryLimitMB, &fn.AllowNet, &fn.AllowEnv, &fn.AllowRead, &fn.AllowWrite, &fn.AllowUnauthenticated, &fn.IsPublic, &fn.DisableExecutionLogs,
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

// GetFunctionByNamespace retrieves a function by name and namespace
func (s *Storage) GetFunctionByNamespace(ctx context.Context, name string, namespace string) (*EdgeFunction, error) {
	tenantID := database.TenantFromContext(ctx)

	query := `
		SELECT id, name, namespace, description, code, original_code, is_bundled, bundle_error, version, cron_schedule, enabled,
		       timeout_seconds, memory_limit_mb, allow_net, allow_env, allow_read, allow_write, allow_unauthenticated, is_public, disable_execution_logs,
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
			&fn.TimeoutSeconds, &fn.MemoryLimitMB, &fn.AllowNet, &fn.AllowEnv, &fn.AllowRead, &fn.AllowWrite, &fn.AllowUnauthenticated, &fn.IsPublic, &fn.DisableExecutionLogs,
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
		       timeout_seconds, memory_limit_mb, allow_net, allow_env, allow_read, allow_write, allow_unauthenticated, is_public, disable_execution_logs,
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
				&fn.TimeoutSeconds, &fn.MemoryLimitMB, &fn.AllowNet, &fn.AllowEnv, &fn.AllowRead, &fn.AllowWrite, &fn.AllowUnauthenticated, &fn.IsPublic, &fn.DisableExecutionLogs,
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

// ListFunctionsByNamespaceForSync returns all functions matching the given tenant OR with NULL tenant_id.
// This is used by the sync flow to find existing functions regardless of whether they
// have been backfilled to the current tenant or still have NULL tenant_id from pre-tenancy.
func (s *Storage) ListFunctionsByNamespaceForSync(ctx context.Context, namespace string, tenantID string) ([]EdgeFunctionSummary, error) {
	query := `
		SELECT id, name, namespace, description, is_bundled, bundle_error, version, cron_schedule, enabled,
		       timeout_seconds, memory_limit_mb, allow_net, allow_env, allow_read, allow_write, allow_unauthenticated, is_public, disable_execution_logs,
		       cors_origins, cors_methods, cors_headers, cors_credentials, cors_max_age,
		       rate_limit_per_minute, rate_limit_per_hour, rate_limit_per_day,
		       created_at, updated_at, created_by, source, tenant_id
		FROM functions.edge_functions
		WHERE namespace = $1
		  AND (tenant_id = $2 OR tenant_id IS NULL)
		ORDER BY created_at DESC
	`

	var functions []EdgeFunctionSummary
	err := database.WrapWithServiceRole(ctx, s.DB, func(tx pgx.Tx) error {
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
				&fn.TimeoutSeconds, &fn.MemoryLimitMB, &fn.AllowNet, &fn.AllowEnv, &fn.AllowRead, &fn.AllowWrite, &fn.AllowUnauthenticated, &fn.IsPublic, &fn.DisableExecutionLogs,
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

// ListAllFunctions returns all functions regardless of is_public setting (admin use)
func (s *Storage) ListAllFunctions(ctx context.Context) ([]EdgeFunctionSummary, error) {
	tenantID := database.TenantFromContext(ctx)

	query := `
		SELECT id, name, namespace, description, is_bundled, bundle_error, version, cron_schedule, enabled,
		       timeout_seconds, memory_limit_mb, allow_net, allow_env, allow_read, allow_write, allow_unauthenticated, is_public, disable_execution_logs,
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
				&fn.TimeoutSeconds, &fn.MemoryLimitMB, &fn.AllowNet, &fn.AllowEnv, &fn.AllowRead, &fn.AllowWrite, &fn.AllowUnauthenticated, &fn.IsPublic, &fn.DisableExecutionLogs,
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
		       timeout_seconds, memory_limit_mb, allow_net, allow_env, allow_read, allow_write, allow_unauthenticated, is_public, disable_execution_logs,
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
				&fn.TimeoutSeconds, &fn.MemoryLimitMB, &fn.AllowNet, &fn.AllowEnv, &fn.AllowRead, &fn.AllowWrite, &fn.AllowUnauthenticated, &fn.IsPublic, &fn.DisableExecutionLogs,
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

// ListAllFunctionsAllTenants returns all functions with cron schedules across all tenants.
// Used by the scheduler to load cron-enabled functions without tenant filtering.
func (s *Storage) ListAllFunctionsAllTenants(ctx context.Context) ([]EdgeFunctionSummary, error) {
	query := `
		SELECT id, name, namespace, description, is_bundled, bundle_error, version, cron_schedule, enabled,
		       timeout_seconds, memory_limit_mb, allow_net, allow_env, allow_read, allow_write, allow_unauthenticated, is_public, disable_execution_logs,
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
				&fn.TimeoutSeconds, &fn.MemoryLimitMB, &fn.AllowNet, &fn.AllowEnv, &fn.AllowRead, &fn.AllowWrite, &fn.AllowUnauthenticated, &fn.IsPublic, &fn.DisableExecutionLogs,
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

	err := database.WrapWithServiceRole(ctx, s.DB, func(tx pgx.Tx) error {
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

	err := database.WrapWithServiceRole(ctx, s.DB, func(tx pgx.Tx) error {
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
	err := database.WrapWithServiceRole(ctx, s.DB, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx, query, name, namespace, database.TenantOrNil(tenantID))
		return err
	})
	if err != nil {
		return fmt.Errorf("failed to delete function for sync: %w", err)
	}
	return nil
}

// LogExecution logs a function execution
func (s *Storage) LogExecution(ctx context.Context, exec *EdgeFunctionExecution) error {
	query := `
		INSERT INTO functions.edge_executions (
			function_id, trigger_type, status, status_code,
			duration_ms, result, logs, error_message, completed_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id, started_at
	`

	err := s.WithTenant(ctx, func(tx pgx.Tx) error {
		return tx.QueryRow(ctx, query,
			exec.FunctionID, exec.TriggerType, exec.Status, exec.StatusCode,
			exec.DurationMs, exec.Result, exec.Logs, exec.ErrorMessage, exec.CompletedAt,
		).Scan(&exec.ID, &exec.ExecutedAt)
	})
	if err != nil {
		return fmt.Errorf("failed to log execution: %w", err)
	}

	return nil
}

// CreateExecution creates a new execution record with "running" status
// This should be called BEFORE execution to enable real-time logging
func (s *Storage) CreateExecution(ctx context.Context, id uuid.UUID, functionID uuid.UUID, triggerType string) error {
	query := `
		INSERT INTO functions.edge_executions (id, function_id, trigger_type, status)
		VALUES ($1, $2, $3, 'running')
	`

	err := s.WithTenant(ctx, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx, query, id, functionID, triggerType)
		return err
	})
	if err != nil {
		return fmt.Errorf("failed to create execution: %w", err)
	}

	return nil
}

// CompleteExecution updates an execution record when finished
func (s *Storage) CompleteExecution(ctx context.Context, id uuid.UUID, status string, statusCode *int, durationMs *int, result *string, logs *string, errorMessage *string) error {
	tenantID := database.TenantFromContext(ctx)

	query := `
		UPDATE functions.edge_executions
		SET status = $2, status_code = $3, duration_ms = $4, result = $5, logs = $6, error_message = $7, completed_at = NOW()
		WHERE id = $1
		  AND (tenant_id = $8 OR ($8 IS NULL AND tenant_id IS NULL))
	`

	err := s.WithTenant(ctx, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx, query, id, status, statusCode, durationMs, result, logs, errorMessage, database.TenantOrNil(tenantID))
		return err
	})
	if err != nil {
		return fmt.Errorf("failed to complete execution: %w", err)
	}

	return nil
}

// GetExecutions returns execution history for a function
func (s *Storage) GetExecutions(ctx context.Context, functionName string, limit int) ([]EdgeFunctionExecution, error) {
	tenantID := database.TenantFromContext(ctx)

	query := `
		SELECT e.id, e.function_id, e.trigger_type, e.status, e.status_code,
		       e.duration_ms, e.result, e.logs, e.error_message,
		       e.started_at, e.completed_at
		FROM functions.edge_executions e
		JOIN functions.edge_functions f ON e.function_id = f.id
		WHERE f.name = $1
		  AND (f.tenant_id = $2 OR ($2 IS NULL AND f.tenant_id IS NULL))
		ORDER BY e.started_at DESC
		LIMIT $3
	`

	var executions []EdgeFunctionExecution
	err := s.WithTenant(ctx, func(tx pgx.Tx) error {
		rows, err := tx.Query(ctx, query, functionName, database.TenantOrNil(tenantID), limit)
		if err != nil {
			return err
		}
		defer rows.Close()

		for rows.Next() {
			exec := EdgeFunctionExecution{}
			err := rows.Scan(
				&exec.ID, &exec.FunctionID, &exec.TriggerType, &exec.Status, &exec.StatusCode,
				&exec.DurationMs, &exec.Result, &exec.Logs, &exec.ErrorMessage,
				&exec.ExecutedAt, &exec.CompletedAt,
			)
			if err != nil {
				return err
			}
			executions = append(executions, exec)
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get executions: %w", err)
	}

	return executions, nil
}

// AdminExecution extends EdgeFunctionExecution with function name for admin listings
type AdminExecution struct {
	EdgeFunctionExecution
	FunctionName string `json:"function_name"`
	Namespace    string `json:"namespace"`
}

// AdminExecutionFilters defines filter parameters for listing all executions
type AdminExecutionFilters struct {
	Namespace    string
	FunctionName string
	Status       string
	Limit        int
	Offset       int
}

// ListAllExecutions returns execution history across all functions with filters (admin only)
func (s *Storage) ListAllExecutions(ctx context.Context, filters AdminExecutionFilters) ([]AdminExecution, int, error) {
	tenantID := database.TenantFromContext(ctx)
	tenantFilter := " AND (f.tenant_id = $%d OR ($%d IS NULL AND f.tenant_id IS NULL))"

	// Build count query
	countQuery := `
		SELECT COUNT(*)
		FROM functions.edge_executions e
		JOIN functions.edge_functions f ON e.function_id = f.id
		WHERE 1=1
	`
	countArgs := []interface{}{}
	argIdx := 1

	// Add tenant filter as first argument
	countQuery += fmt.Sprintf(tenantFilter, argIdx, argIdx)
	countArgs = append(countArgs, database.TenantOrNil(tenantID))
	argIdx++

	if filters.Namespace != "" {
		countQuery += fmt.Sprintf(" AND f.namespace = $%d", argIdx)
		countArgs = append(countArgs, filters.Namespace)
		argIdx++
	}
	if filters.FunctionName != "" {
		countQuery += fmt.Sprintf(" AND f.name ILIKE $%d", argIdx)
		countArgs = append(countArgs, "%"+filters.FunctionName+"%")
		argIdx++
	}
	if filters.Status != "" {
		countQuery += fmt.Sprintf(" AND e.status = $%d", argIdx)
		countArgs = append(countArgs, filters.Status)
	}

	// Build main query
	query := `
		SELECT e.id, e.function_id, e.trigger_type, e.status, e.status_code,
		       e.duration_ms, e.result, e.logs, e.error_message,
		       e.started_at, e.completed_at, f.name, f.namespace
		FROM functions.edge_executions e
		JOIN functions.edge_functions f ON e.function_id = f.id
		WHERE 1=1
	`
	args := []interface{}{}
	argIdx = 1

	// Add tenant filter as first argument
	query += fmt.Sprintf(tenantFilter, argIdx, argIdx)
	args = append(args, database.TenantOrNil(tenantID))
	argIdx++

	if filters.Namespace != "" {
		query += fmt.Sprintf(" AND f.namespace = $%d", argIdx)
		args = append(args, filters.Namespace)
		argIdx++
	}
	if filters.FunctionName != "" {
		query += fmt.Sprintf(" AND f.name ILIKE $%d", argIdx)
		args = append(args, "%"+filters.FunctionName+"%")
		argIdx++
	}
	if filters.Status != "" {
		query += fmt.Sprintf(" AND e.status = $%d", argIdx)
		args = append(args, filters.Status)
		argIdx++
	}

	query += " ORDER BY e.started_at DESC"
	query += fmt.Sprintf(" LIMIT $%d OFFSET $%d", argIdx, argIdx+1)
	args = append(args, filters.Limit, filters.Offset)

	var executions []AdminExecution
	var total int

	err := s.WithTenant(ctx, func(tx pgx.Tx) error {
		// Get total count
		if err := tx.QueryRow(ctx, countQuery, countArgs...).Scan(&total); err != nil {
			return fmt.Errorf("failed to count executions: %w", err)
		}

		// Get executions
		rows, err := tx.Query(ctx, query, args...)
		if err != nil {
			return err
		}
		defer rows.Close()

		for rows.Next() {
			exec := AdminExecution{}
			err := rows.Scan(
				&exec.ID, &exec.FunctionID, &exec.TriggerType, &exec.Status, &exec.StatusCode,
				&exec.DurationMs, &exec.Result, &exec.Logs, &exec.ErrorMessage,
				&exec.ExecutedAt, &exec.CompletedAt, &exec.FunctionName, &exec.Namespace,
			)
			if err != nil {
				return err
			}
			executions = append(executions, exec)
		}
		return nil
	})
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list executions: %w", err)
	}

	return executions, total, nil
}

// CreateSharedModule creates a new shared module or updates it if it already exists (upsert)
func (s *Storage) CreateSharedModule(ctx context.Context, module *SharedModule) error {
	query := `
		INSERT INTO functions.shared_modules (
			module_path, content, description, created_by
		) VALUES ($1, $2, $3, $4)
		ON CONFLICT (module_path) DO UPDATE SET
			content = EXCLUDED.content,
			description = EXCLUDED.description,
			version = functions.shared_modules.version + 1,
			updated_at = NOW()
		RETURNING id, version, created_at, updated_at
	`

	err := s.WithTenant(ctx, func(tx pgx.Tx) error {
		return tx.QueryRow(ctx, query,
			module.ModulePath, module.Content, module.Description, module.CreatedBy,
		).Scan(&module.ID, &module.Version, &module.CreatedAt, &module.UpdatedAt)
	})
	if err != nil {
		return fmt.Errorf("failed to create shared module: %w", err)
	}

	return nil
}

// GetSharedModule retrieves a shared module by path
func (s *Storage) GetSharedModule(ctx context.Context, modulePath string) (*SharedModule, error) {
	tenantID := database.TenantFromContext(ctx)

	query := `
		SELECT id, module_path, content, description, version, created_at, updated_at, created_by
		FROM functions.shared_modules
		WHERE module_path = $1
		  AND (tenant_id = $2 OR ($2 IS NULL AND tenant_id IS NULL))
	`

	module := &SharedModule{}
	err := s.WithTenant(ctx, func(tx pgx.Tx) error {
		return tx.QueryRow(ctx, query, modulePath, database.TenantOrNil(tenantID)).Scan(
			&module.ID, &module.ModulePath, &module.Content, &module.Description,
			&module.Version, &module.CreatedAt, &module.UpdatedAt, &module.CreatedBy,
		)
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get shared module: %w", err)
	}

	return module, nil
}

// ListSharedModules returns all shared modules
func (s *Storage) ListSharedModules(ctx context.Context) ([]SharedModule, error) {
	tenantID := database.TenantFromContext(ctx)

	query := `
		SELECT id, module_path, content, description, version, created_at, updated_at, created_by
		FROM functions.shared_modules
		WHERE (tenant_id = $1 OR ($1 IS NULL AND tenant_id IS NULL))
		ORDER BY module_path
	`

	var modules []SharedModule
	err := s.WithTenant(ctx, func(tx pgx.Tx) error {
		rows, err := tx.Query(ctx, query, database.TenantOrNil(tenantID))
		if err != nil {
			return err
		}
		defer rows.Close()

		for rows.Next() {
			module := SharedModule{}
			err := rows.Scan(
				&module.ID, &module.ModulePath, &module.Content, &module.Description,
				&module.Version, &module.CreatedAt, &module.UpdatedAt, &module.CreatedBy,
			)
			if err != nil {
				return err
			}
			modules = append(modules, module)
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list shared modules: %w", err)
	}

	return modules, nil
}

// UpdateSharedModule updates an existing shared module
func (s *Storage) UpdateSharedModule(ctx context.Context, modulePath string, content string, description *string) error {
	tenantID := database.TenantFromContext(ctx)

	query := `
		UPDATE functions.shared_modules
		SET content = $1, description = $2, version = version + 1, updated_at = NOW()
		WHERE module_path = $3
		  AND (tenant_id = $4 OR ($4 IS NULL AND tenant_id IS NULL))
	`

	err := s.WithTenant(ctx, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx, query, content, description, modulePath, database.TenantOrNil(tenantID))
		return err
	})
	if err != nil {
		return fmt.Errorf("failed to update shared module: %w", err)
	}

	return nil
}

// DeleteSharedModule deletes a shared module
func (s *Storage) DeleteSharedModule(ctx context.Context, modulePath string) error {
	tenantID := database.TenantFromContext(ctx)

	query := "DELETE FROM functions.shared_modules WHERE module_path = $1 AND (tenant_id = $2 OR ($2 IS NULL AND tenant_id IS NULL))"

	err := s.WithTenant(ctx, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx, query, modulePath, database.TenantOrNil(tenantID))
		return err
	})
	if err != nil {
		return fmt.Errorf("failed to delete shared module: %w", err)
	}

	return nil
}

// SaveFunctionFiles stores supporting files for a function
func (s *Storage) SaveFunctionFiles(ctx context.Context, functionID uuid.UUID, files []FunctionFile) error {
	tenantID := database.TenantFromContext(ctx)

	// First, delete existing files for this function
	deleteQuery := "DELETE FROM functions.edge_files WHERE function_id = $1 AND (tenant_id = $2 OR ($2 IS NULL AND tenant_id IS NULL))"

	err := s.WithTenant(ctx, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx, deleteQuery, functionID, database.TenantOrNil(tenantID))
		if err != nil {
			return err
		}

		// Insert new files
		insertQuery := `
			INSERT INTO functions.edge_files (
				function_id, file_path, content
			) VALUES ($1, $2, $3)
		`

		for _, file := range files {
			_, err := tx.Exec(ctx, insertQuery, functionID, file.FilePath, file.Content)
			if err != nil {
				return err
			}
		}

		return nil
	})
	if err != nil {
		return fmt.Errorf("failed to save function files: %w", err)
	}

	return nil
}

// GetFunctionFiles retrieves all supporting files for a function
func (s *Storage) GetFunctionFiles(ctx context.Context, functionID uuid.UUID) ([]FunctionFile, error) {
	tenantID := database.TenantFromContext(ctx)

	query := `
		SELECT id, function_id, file_path, content, created_at, updated_at
		FROM functions.edge_files
		WHERE function_id = $1
		  AND (tenant_id = $2 OR ($2 IS NULL AND tenant_id IS NULL))
		ORDER BY file_path
	`

	var files []FunctionFile
	err := s.WithTenant(ctx, func(tx pgx.Tx) error {
		rows, err := tx.Query(ctx, query, functionID, database.TenantOrNil(tenantID))
		if err != nil {
			return err
		}
		defer rows.Close()

		for rows.Next() {
			file := FunctionFile{}
			err := rows.Scan(
				&file.ID, &file.FunctionID, &file.FilePath, &file.Content,
				&file.CreatedAt, &file.UpdatedAt,
			)
			if err != nil {
				return err
			}
			files = append(files, file)
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get function files: %w", err)
	}

	return files, nil
}

// Note: Execution logs are now stored in the central logging schema (logging.entries)
