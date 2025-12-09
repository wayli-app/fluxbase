package functions

import (
	"context"
	"fmt"
	"time"

	"github.com/fluxbase-eu/fluxbase/internal/database"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

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
	AllowUnauthenticated bool      `json:"allow_unauthenticated"` // Allow invocation without authentication
	IsPublic             bool      `json:"is_public"`             // Whether function is publicly listed
	// CORS configuration (nil means use global defaults from FLUXBASE_CORS_* env vars)
	CorsOrigins     *string    `json:"cors_origins"`
	CorsMethods     *string    `json:"cors_methods"`
	CorsHeaders     *string    `json:"cors_headers"`
	CorsCredentials *bool      `json:"cors_credentials"`
	CorsMaxAge      *int       `json:"cors_max_age"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
	CreatedBy       *uuid.UUID `json:"created_by"`
	Source          string     `json:"source"` // "filesystem" or "api"
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
	CorsOrigins          *string    `json:"cors_origins"`
	CorsMethods          *string    `json:"cors_methods"`
	CorsHeaders          *string    `json:"cors_headers"`
	CorsCredentials      *bool      `json:"cors_credentials"`
	CorsMaxAge           *int       `json:"cors_max_age"`
	CreatedAt            time.Time  `json:"created_at"`
	UpdatedAt            time.Time  `json:"updated_at"`
	CreatedBy            *uuid.UUID `json:"created_by"`
	Source               string     `json:"source"` // "filesystem" or "api"
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

// ExecutionLog represents a single log entry for a function execution
type ExecutionLog struct {
	ID          int64     `json:"id"`
	ExecutionID uuid.UUID `json:"execution_id"`
	LineNumber  int       `json:"line_number"`
	Level       string    `json:"level"` // debug, info, warn, error
	Message     string    `json:"message"`
	CreatedAt   time.Time `json:"created_at"`
}

// Storage manages edge function persistence
type Storage struct {
	db *database.Connection
}

// NewStorage creates a new storage manager
func NewStorage(db *database.Connection) *Storage {
	return &Storage{db: db}
}

// CreateFunction creates a new edge function
func (s *Storage) CreateFunction(ctx context.Context, fn *EdgeFunction) error {
	query := `
		INSERT INTO functions.edge_functions (
			name, namespace, description, code, original_code, is_bundled, bundle_error,
			enabled, timeout_seconds, memory_limit_mb,
			allow_net, allow_env, allow_read, allow_write, allow_unauthenticated, is_public,
			cors_origins, cors_methods, cors_headers, cors_credentials, cors_max_age,
			cron_schedule, created_by, source
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21, $22, $23, $24)
		RETURNING id, version, created_at, updated_at
	`

	err := database.WrapWithServiceRole(ctx, s.db, func(tx pgx.Tx) error {
		return tx.QueryRow(ctx, query,
			fn.Name, fn.Namespace, fn.Description, fn.Code, fn.OriginalCode, fn.IsBundled, fn.BundleError,
			fn.Enabled, fn.TimeoutSeconds, fn.MemoryLimitMB,
			fn.AllowNet, fn.AllowEnv, fn.AllowRead, fn.AllowWrite, fn.AllowUnauthenticated, fn.IsPublic,
			fn.CorsOrigins, fn.CorsMethods, fn.CorsHeaders, fn.CorsCredentials, fn.CorsMaxAge,
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
	query := `
		SELECT id, name, namespace, description, code, original_code, is_bundled, bundle_error, version, cron_schedule, enabled,
		       timeout_seconds, memory_limit_mb, allow_net, allow_env, allow_read, allow_write, allow_unauthenticated, is_public,
		       cors_origins, cors_methods, cors_headers, cors_credentials, cors_max_age,
		       created_at, updated_at, created_by, source
		FROM functions.edge_functions
		WHERE name = $1
		ORDER BY namespace
		LIMIT 1
	`

	fn := &EdgeFunction{}
	err := database.WrapWithServiceRole(ctx, s.db, func(tx pgx.Tx) error {
		return tx.QueryRow(ctx, query, name).Scan(
			&fn.ID, &fn.Name, &fn.Namespace, &fn.Description, &fn.Code, &fn.OriginalCode, &fn.IsBundled, &fn.BundleError,
			&fn.Version, &fn.CronSchedule, &fn.Enabled,
			&fn.TimeoutSeconds, &fn.MemoryLimitMB, &fn.AllowNet, &fn.AllowEnv, &fn.AllowRead, &fn.AllowWrite, &fn.AllowUnauthenticated, &fn.IsPublic,
			&fn.CorsOrigins, &fn.CorsMethods, &fn.CorsHeaders, &fn.CorsCredentials, &fn.CorsMaxAge,
			&fn.CreatedAt, &fn.UpdatedAt, &fn.CreatedBy, &fn.Source,
		)
	})

	if err != nil {
		return nil, fmt.Errorf("failed to get function: %w", err)
	}

	return fn, nil
}

// GetFunctionByNamespace retrieves a function by name and namespace
func (s *Storage) GetFunctionByNamespace(ctx context.Context, name string, namespace string) (*EdgeFunction, error) {
	query := `
		SELECT id, name, namespace, description, code, original_code, is_bundled, bundle_error, version, cron_schedule, enabled,
		       timeout_seconds, memory_limit_mb, allow_net, allow_env, allow_read, allow_write, allow_unauthenticated, is_public,
		       cors_origins, cors_methods, cors_headers, cors_credentials, cors_max_age,
		       created_at, updated_at, created_by, source
		FROM functions.edge_functions
		WHERE name = $1 AND namespace = $2
	`

	fn := &EdgeFunction{}
	err := database.WrapWithServiceRole(ctx, s.db, func(tx pgx.Tx) error {
		return tx.QueryRow(ctx, query, name, namespace).Scan(
			&fn.ID, &fn.Name, &fn.Namespace, &fn.Description, &fn.Code, &fn.OriginalCode, &fn.IsBundled, &fn.BundleError,
			&fn.Version, &fn.CronSchedule, &fn.Enabled,
			&fn.TimeoutSeconds, &fn.MemoryLimitMB, &fn.AllowNet, &fn.AllowEnv, &fn.AllowRead, &fn.AllowWrite, &fn.AllowUnauthenticated, &fn.IsPublic,
			&fn.CorsOrigins, &fn.CorsMethods, &fn.CorsHeaders, &fn.CorsCredentials, &fn.CorsMaxAge,
			&fn.CreatedAt, &fn.UpdatedAt, &fn.CreatedBy, &fn.Source,
		)
	})

	if err != nil {
		return nil, fmt.Errorf("failed to get function: %w", err)
	}

	return fn, nil
}

// ListFunctions returns all public functions (is_public=true), excludes code for performance
func (s *Storage) ListFunctions(ctx context.Context) ([]EdgeFunctionSummary, error) {
	query := `
		SELECT id, name, namespace, description, is_bundled, bundle_error, version, cron_schedule, enabled,
		       timeout_seconds, memory_limit_mb, allow_net, allow_env, allow_read, allow_write, allow_unauthenticated, is_public,
		       cors_origins, cors_methods, cors_headers, cors_credentials, cors_max_age,
		       created_at, updated_at, created_by, source
		FROM functions.edge_functions
		WHERE is_public = true
		ORDER BY created_at DESC
	`

	var functions []EdgeFunctionSummary
	err := database.WrapWithServiceRole(ctx, s.db, func(tx pgx.Tx) error {
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
				&fn.TimeoutSeconds, &fn.MemoryLimitMB, &fn.AllowNet, &fn.AllowEnv, &fn.AllowRead, &fn.AllowWrite, &fn.AllowUnauthenticated, &fn.IsPublic,
				&fn.CorsOrigins, &fn.CorsMethods, &fn.CorsHeaders, &fn.CorsCredentials, &fn.CorsMaxAge,
				&fn.CreatedAt, &fn.UpdatedAt, &fn.CreatedBy, &fn.Source,
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

// ListFunctionNamespaces returns all unique namespaces that have edge functions
func (s *Storage) ListFunctionNamespaces(ctx context.Context) ([]string, error) {
	query := `SELECT DISTINCT namespace FROM functions.edge_functions ORDER BY namespace`

	var namespaces []string
	err := database.WrapWithServiceRole(ctx, s.db, func(tx pgx.Tx) error {
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
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to list function namespaces: %w", err)
	}

	return namespaces, nil
}

// ListFunctionsByNamespace returns all functions in a specific namespace, excludes code for performance
func (s *Storage) ListFunctionsByNamespace(ctx context.Context, namespace string) ([]EdgeFunctionSummary, error) {
	query := `
		SELECT id, name, namespace, description, is_bundled, bundle_error, version, cron_schedule, enabled,
		       timeout_seconds, memory_limit_mb, allow_net, allow_env, allow_read, allow_write, allow_unauthenticated, is_public,
		       cors_origins, cors_methods, cors_headers, cors_credentials, cors_max_age,
		       created_at, updated_at, created_by, source
		FROM functions.edge_functions
		WHERE namespace = $1
		ORDER BY created_at DESC
	`

	var functions []EdgeFunctionSummary
	err := database.WrapWithServiceRole(ctx, s.db, func(tx pgx.Tx) error {
		rows, err := tx.Query(ctx, query, namespace)
		if err != nil {
			return err
		}
		defer rows.Close()

		for rows.Next() {
			fn := EdgeFunctionSummary{}
			err := rows.Scan(
				&fn.ID, &fn.Name, &fn.Namespace, &fn.Description, &fn.IsBundled, &fn.BundleError,
				&fn.Version, &fn.CronSchedule, &fn.Enabled,
				&fn.TimeoutSeconds, &fn.MemoryLimitMB, &fn.AllowNet, &fn.AllowEnv, &fn.AllowRead, &fn.AllowWrite, &fn.AllowUnauthenticated, &fn.IsPublic,
				&fn.CorsOrigins, &fn.CorsMethods, &fn.CorsHeaders, &fn.CorsCredentials, &fn.CorsMaxAge,
				&fn.CreatedAt, &fn.UpdatedAt, &fn.CreatedBy, &fn.Source,
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
	// Build dynamic UPDATE query
	query := "UPDATE functions.edge_functions SET "
	args := []interface{}{}
	argCount := 1

	for key, value := range updates {
		if argCount > 1 {
			query += ", "
		}
		query += fmt.Sprintf("%s = $%d", key, argCount)
		args = append(args, value)
		argCount++
	}

	query += fmt.Sprintf(" WHERE name = $%d AND namespace = $%d", argCount, argCount+1)
	args = append(args, name, namespace)

	err := database.WrapWithServiceRole(ctx, s.db, func(tx pgx.Tx) error {
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

// DeleteFunctionByNamespace deletes a function by name and namespace
func (s *Storage) DeleteFunctionByNamespace(ctx context.Context, name string, namespace string) error {
	query := "DELETE FROM functions.edge_functions WHERE name = $1 AND namespace = $2"
	err := database.WrapWithServiceRole(ctx, s.db, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx, query, name, namespace)
		return err
	})
	if err != nil {
		return fmt.Errorf("failed to delete function: %w", err)
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

	err := database.WrapWithServiceRole(ctx, s.db, func(tx pgx.Tx) error {
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

	err := database.WrapWithServiceRole(ctx, s.db, func(tx pgx.Tx) error {
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
	query := `
		UPDATE functions.edge_executions
		SET status = $2, status_code = $3, duration_ms = $4, result = $5, logs = $6, error_message = $7, completed_at = NOW()
		WHERE id = $1
	`

	err := database.WrapWithServiceRole(ctx, s.db, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx, query, id, status, statusCode, durationMs, result, logs, errorMessage)
		return err
	})

	if err != nil {
		return fmt.Errorf("failed to complete execution: %w", err)
	}

	return nil
}

// GetExecutions returns execution history for a function
func (s *Storage) GetExecutions(ctx context.Context, functionName string, limit int) ([]EdgeFunctionExecution, error) {
	query := `
		SELECT e.id, e.function_id, e.trigger_type, e.status, e.status_code,
		       e.duration_ms, e.result, e.logs, e.error_message,
		       e.started_at, e.completed_at
		FROM functions.edge_executions e
		JOIN functions.edge_functions f ON e.function_id = f.id
		WHERE f.name = $1
		ORDER BY e.started_at DESC
		LIMIT $2
	`

	var executions []EdgeFunctionExecution
	err := database.WrapWithServiceRole(ctx, s.db, func(tx pgx.Tx) error {
		rows, err := tx.Query(ctx, query, functionName, limit)
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
	// Build count query
	countQuery := `
		SELECT COUNT(*)
		FROM functions.edge_executions e
		JOIN functions.edge_functions f ON e.function_id = f.id
		WHERE 1=1
	`
	countArgs := []interface{}{}
	argIdx := 1

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
		argIdx++
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

	err := database.WrapWithServiceRole(ctx, s.db, func(tx pgx.Tx) error {
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

// CreateSharedModule creates a new shared module
func (s *Storage) CreateSharedModule(ctx context.Context, module *SharedModule) error {
	query := `
		INSERT INTO functions.shared_modules (
			module_path, content, description, created_by
		) VALUES ($1, $2, $3, $4)
		RETURNING id, version, created_at, updated_at
	`

	err := database.WrapWithServiceRole(ctx, s.db, func(tx pgx.Tx) error {
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
	query := `
		SELECT id, module_path, content, description, version, created_at, updated_at, created_by
		FROM functions.shared_modules
		WHERE module_path = $1
	`

	module := &SharedModule{}
	err := database.WrapWithServiceRole(ctx, s.db, func(tx pgx.Tx) error {
		return tx.QueryRow(ctx, query, modulePath).Scan(
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
	query := `
		SELECT id, module_path, content, description, version, created_at, updated_at, created_by
		FROM functions.shared_modules
		ORDER BY module_path
	`

	var modules []SharedModule
	err := database.WrapWithServiceRole(ctx, s.db, func(tx pgx.Tx) error {
		rows, err := tx.Query(ctx, query)
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
	query := `
		UPDATE functions.shared_modules
		SET content = $1, description = $2, version = version + 1, updated_at = NOW()
		WHERE module_path = $3
	`

	err := database.WrapWithServiceRole(ctx, s.db, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx, query, content, description, modulePath)
		return err
	})

	if err != nil {
		return fmt.Errorf("failed to update shared module: %w", err)
	}

	return nil
}

// DeleteSharedModule deletes a shared module by path
func (s *Storage) DeleteSharedModule(ctx context.Context, modulePath string) error {
	query := "DELETE FROM functions.shared_modules WHERE module_path = $1"

	err := database.WrapWithServiceRole(ctx, s.db, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx, query, modulePath)
		return err
	})

	if err != nil {
		return fmt.Errorf("failed to delete shared module: %w", err)
	}

	return nil
}

// SaveFunctionFiles stores supporting files for a function
func (s *Storage) SaveFunctionFiles(ctx context.Context, functionID uuid.UUID, files []FunctionFile) error {
	// First, delete existing files for this function
	deleteQuery := "DELETE FROM functions.edge_files WHERE function_id = $1"

	err := database.WrapWithServiceRole(ctx, s.db, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx, deleteQuery, functionID)
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
	query := `
		SELECT id, function_id, file_path, content, created_at, updated_at
		FROM functions.edge_files
		WHERE function_id = $1
		ORDER BY file_path
	`

	var files []FunctionFile
	err := database.WrapWithServiceRole(ctx, s.db, func(tx pgx.Tx) error {
		rows, err := tx.Query(ctx, query, functionID)
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

// ============================================================================
// EXECUTION LOG OPERATIONS
// ============================================================================

// AppendExecutionLog adds a single log entry for a function execution
func (s *Storage) AppendExecutionLog(ctx context.Context, executionID uuid.UUID, lineNumber int, level, message string) error {
	query := `
		INSERT INTO functions.execution_logs (execution_id, line_number, level, message, created_at)
		VALUES ($1, $2, $3, $4, $5)
	`

	err := database.WrapWithServiceRole(ctx, s.db, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx, query, executionID, lineNumber, level, message, time.Now())
		return err
	})

	if err != nil {
		return fmt.Errorf("failed to append execution log: %w", err)
	}

	return nil
}

// AppendExecutionLogs adds multiple log entries for a function execution (batch insert)
func (s *Storage) AppendExecutionLogs(ctx context.Context, executionID uuid.UUID, logs []ExecutionLog) error {
	if len(logs) == 0 {
		return nil
	}

	query := `
		INSERT INTO functions.execution_logs (execution_id, line_number, level, message, created_at)
		VALUES ($1, $2, $3, $4, $5)
	`

	err := database.WrapWithServiceRole(ctx, s.db, func(tx pgx.Tx) error {
		for _, log := range logs {
			_, err := tx.Exec(ctx, query, executionID, log.LineNumber, log.Level, log.Message, time.Now())
			if err != nil {
				return err
			}
		}
		return nil
	})

	if err != nil {
		return fmt.Errorf("failed to append execution logs: %w", err)
	}

	return nil
}

// GetExecutionLogs retrieves all logs for a function execution
func (s *Storage) GetExecutionLogs(ctx context.Context, executionID uuid.UUID) ([]ExecutionLog, error) {
	query := `
		SELECT id, execution_id, line_number, level, message, created_at
		FROM functions.execution_logs
		WHERE execution_id = $1
		ORDER BY line_number ASC
	`

	var logs []ExecutionLog
	err := database.WrapWithServiceRole(ctx, s.db, func(tx pgx.Tx) error {
		rows, err := tx.Query(ctx, query, executionID)
		if err != nil {
			return err
		}
		defer rows.Close()

		for rows.Next() {
			log := ExecutionLog{}
			err := rows.Scan(
				&log.ID, &log.ExecutionID, &log.LineNumber,
				&log.Level, &log.Message, &log.CreatedAt,
			)
			if err != nil {
				return err
			}
			logs = append(logs, log)
		}
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to get execution logs: %w", err)
	}

	return logs, nil
}

// GetExecutionLogsSince retrieves logs for an execution after a specific line number (for streaming/pagination)
func (s *Storage) GetExecutionLogsSince(ctx context.Context, executionID uuid.UUID, afterLine int) ([]ExecutionLog, error) {
	query := `
		SELECT id, execution_id, line_number, level, message, created_at
		FROM functions.execution_logs
		WHERE execution_id = $1 AND line_number > $2
		ORDER BY line_number ASC
	`

	var logs []ExecutionLog
	err := database.WrapWithServiceRole(ctx, s.db, func(tx pgx.Tx) error {
		rows, err := tx.Query(ctx, query, executionID, afterLine)
		if err != nil {
			return err
		}
		defer rows.Close()

		for rows.Next() {
			log := ExecutionLog{}
			err := rows.Scan(
				&log.ID, &log.ExecutionID, &log.LineNumber,
				&log.Level, &log.Message, &log.CreatedAt,
			)
			if err != nil {
				return err
			}
			logs = append(logs, log)
		}
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to get execution logs: %w", err)
	}

	return logs, nil
}
