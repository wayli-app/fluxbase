package functions

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/wayli-app/fluxbase/internal/database"
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
			cron_schedule, created_by
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21, $22, $23)
		RETURNING id, version, created_at, updated_at
	`

	err := database.WrapWithServiceRole(ctx, s.db, func(tx pgx.Tx) error {
		return tx.QueryRow(ctx, query,
			fn.Name, fn.Namespace, fn.Description, fn.Code, fn.OriginalCode, fn.IsBundled, fn.BundleError,
			fn.Enabled, fn.TimeoutSeconds, fn.MemoryLimitMB,
			fn.AllowNet, fn.AllowEnv, fn.AllowRead, fn.AllowWrite, fn.AllowUnauthenticated, fn.IsPublic,
			fn.CorsOrigins, fn.CorsMethods, fn.CorsHeaders, fn.CorsCredentials, fn.CorsMaxAge,
			fn.CronSchedule, fn.CreatedBy,
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
		       created_at, updated_at, created_by
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
			&fn.CreatedAt, &fn.UpdatedAt, &fn.CreatedBy,
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
		       created_at, updated_at, created_by
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
			&fn.CreatedAt, &fn.UpdatedAt, &fn.CreatedBy,
		)
	})

	if err != nil {
		return nil, fmt.Errorf("failed to get function: %w", err)
	}

	return fn, nil
}

// ListFunctions returns all public functions (is_public=true)
func (s *Storage) ListFunctions(ctx context.Context) ([]EdgeFunction, error) {
	query := `
		SELECT id, name, namespace, description, code, original_code, is_bundled, bundle_error, version, cron_schedule, enabled,
		       timeout_seconds, memory_limit_mb, allow_net, allow_env, allow_read, allow_write, allow_unauthenticated, is_public,
		       cors_origins, cors_methods, cors_headers, cors_credentials, cors_max_age,
		       created_at, updated_at, created_by
		FROM functions.edge_functions
		WHERE is_public = true
		ORDER BY created_at DESC
	`

	var functions []EdgeFunction
	err := database.WrapWithServiceRole(ctx, s.db, func(tx pgx.Tx) error {
		rows, err := tx.Query(ctx, query)
		if err != nil {
			return err
		}
		defer rows.Close()

		for rows.Next() {
			fn := EdgeFunction{}
			err := rows.Scan(
				&fn.ID, &fn.Name, &fn.Namespace, &fn.Description, &fn.Code, &fn.OriginalCode, &fn.IsBundled, &fn.BundleError,
				&fn.Version, &fn.CronSchedule, &fn.Enabled,
				&fn.TimeoutSeconds, &fn.MemoryLimitMB, &fn.AllowNet, &fn.AllowEnv, &fn.AllowRead, &fn.AllowWrite, &fn.AllowUnauthenticated, &fn.IsPublic,
				&fn.CorsOrigins, &fn.CorsMethods, &fn.CorsHeaders, &fn.CorsCredentials, &fn.CorsMaxAge,
				&fn.CreatedAt, &fn.UpdatedAt, &fn.CreatedBy,
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

// ListFunctionsByNamespace returns all functions in a specific namespace
func (s *Storage) ListFunctionsByNamespace(ctx context.Context, namespace string) ([]EdgeFunction, error) {
	query := `
		SELECT id, name, namespace, description, code, original_code, is_bundled, bundle_error, version, cron_schedule, enabled,
		       timeout_seconds, memory_limit_mb, allow_net, allow_env, allow_read, allow_write, allow_unauthenticated, is_public,
		       cors_origins, cors_methods, cors_headers, cors_credentials, cors_max_age,
		       created_at, updated_at, created_by
		FROM functions.edge_functions
		WHERE namespace = $1
		ORDER BY created_at DESC
	`

	var functions []EdgeFunction
	err := database.WrapWithServiceRole(ctx, s.db, func(tx pgx.Tx) error {
		rows, err := tx.Query(ctx, query, namespace)
		if err != nil {
			return err
		}
		defer rows.Close()

		for rows.Next() {
			fn := EdgeFunction{}
			err := rows.Scan(
				&fn.ID, &fn.Name, &fn.Namespace, &fn.Description, &fn.Code, &fn.OriginalCode, &fn.IsBundled, &fn.BundleError,
				&fn.Version, &fn.CronSchedule, &fn.Enabled,
				&fn.TimeoutSeconds, &fn.MemoryLimitMB, &fn.AllowNet, &fn.AllowEnv, &fn.AllowRead, &fn.AllowWrite, &fn.AllowUnauthenticated, &fn.IsPublic,
				&fn.CorsOrigins, &fn.CorsMethods, &fn.CorsHeaders, &fn.CorsCredentials, &fn.CorsMaxAge,
				&fn.CreatedAt, &fn.UpdatedAt, &fn.CreatedBy,
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
		INSERT INTO functions.edge_function_executions (
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

// GetExecutions returns execution history for a function
func (s *Storage) GetExecutions(ctx context.Context, functionName string, limit int) ([]EdgeFunctionExecution, error) {
	query := `
		SELECT e.id, e.function_id, e.trigger_type, e.status, e.status_code,
		       e.duration_ms, e.result, e.logs, e.error_message,
		       e.started_at, e.completed_at
		FROM functions.edge_function_executions e
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
	deleteQuery := "DELETE FROM functions.edge_function_files WHERE function_id = $1"

	err := database.WrapWithServiceRole(ctx, s.db, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx, deleteQuery, functionID)
		if err != nil {
			return err
		}

		// Insert new files
		insertQuery := `
			INSERT INTO functions.edge_function_files (
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
		FROM functions.edge_function_files
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
