package functions

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// EdgeFunction represents a stored edge function
type EdgeFunction struct {
	ID                   uuid.UUID  `json:"id"`
	Name                 string     `json:"name"`
	Description          *string    `json:"description"`
	Code                 string     `json:"code"`          // Bundled code (for execution)
	OriginalCode         *string    `json:"original_code"` // Original code before bundling (for editing)
	IsBundled            bool       `json:"is_bundled"`    // Whether code is bundled
	BundleError          *string    `json:"bundle_error"`  // Error if bundling failed
	Version              int        `json:"version"`
	CronSchedule         *string    `json:"cron_schedule"`
	Enabled              bool       `json:"enabled"`
	TimeoutSeconds       int        `json:"timeout_seconds"`
	MemoryLimitMB        int        `json:"memory_limit_mb"`
	AllowNet             bool       `json:"allow_net"`
	AllowEnv             bool       `json:"allow_env"`
	AllowRead            bool       `json:"allow_read"`
	AllowWrite           bool       `json:"allow_write"`
	AllowUnauthenticated bool       `json:"allow_unauthenticated"` // NEW: Allow invocation without authentication
	CreatedAt            time.Time  `json:"created_at"`
	UpdatedAt            time.Time  `json:"updated_at"`
	CreatedBy            *uuid.UUID `json:"created_by"`
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

// Storage manages edge function persistence
type Storage struct {
	db *pgxpool.Pool
}

// NewStorage creates a new storage manager
func NewStorage(db *pgxpool.Pool) *Storage {
	return &Storage{db: db}
}

// CreateFunction creates a new edge function
func (s *Storage) CreateFunction(ctx context.Context, fn *EdgeFunction) error {
	query := `
		INSERT INTO functions.edge_functions (
			name, description, code, original_code, is_bundled, bundle_error,
			enabled, timeout_seconds, memory_limit_mb,
			allow_net, allow_env, allow_read, allow_write, allow_unauthenticated, cron_schedule, created_by
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16)
		RETURNING id, version, created_at, updated_at
	`

	err := s.db.QueryRow(ctx, query,
		fn.Name, fn.Description, fn.Code, fn.OriginalCode, fn.IsBundled, fn.BundleError,
		fn.Enabled, fn.TimeoutSeconds, fn.MemoryLimitMB,
		fn.AllowNet, fn.AllowEnv, fn.AllowRead, fn.AllowWrite, fn.AllowUnauthenticated, fn.CronSchedule, fn.CreatedBy,
	).Scan(&fn.ID, &fn.Version, &fn.CreatedAt, &fn.UpdatedAt)

	if err != nil {
		return fmt.Errorf("failed to create function: %w", err)
	}

	return nil
}

// GetFunction retrieves a function by name
func (s *Storage) GetFunction(ctx context.Context, name string) (*EdgeFunction, error) {
	query := `
		SELECT id, name, description, code, original_code, is_bundled, bundle_error, version, cron_schedule, enabled,
		       timeout_seconds, memory_limit_mb, allow_net, allow_env, allow_read, allow_write, allow_unauthenticated,
		       created_at, updated_at, created_by
		FROM functions.edge_functions
		WHERE name = $1
	`

	fn := &EdgeFunction{}
	err := s.db.QueryRow(ctx, query, name).Scan(
		&fn.ID, &fn.Name, &fn.Description, &fn.Code, &fn.OriginalCode, &fn.IsBundled, &fn.BundleError,
		&fn.Version, &fn.CronSchedule, &fn.Enabled,
		&fn.TimeoutSeconds, &fn.MemoryLimitMB, &fn.AllowNet, &fn.AllowEnv, &fn.AllowRead, &fn.AllowWrite, &fn.AllowUnauthenticated,
		&fn.CreatedAt, &fn.UpdatedAt, &fn.CreatedBy,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to get function: %w", err)
	}

	return fn, nil
}

// ListFunctions returns all functions
func (s *Storage) ListFunctions(ctx context.Context) ([]EdgeFunction, error) {
	query := `
		SELECT id, name, description, code, original_code, is_bundled, bundle_error, version, cron_schedule, enabled,
		       timeout_seconds, memory_limit_mb, allow_net, allow_env, allow_read, allow_write, allow_unauthenticated,
		       created_at, updated_at, created_by
		FROM functions.edge_functions
		ORDER BY created_at DESC
	`

	rows, err := s.db.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to list functions: %w", err)
	}
	defer rows.Close()

	var functions []EdgeFunction
	for rows.Next() {
		fn := EdgeFunction{}
		err := rows.Scan(
			&fn.ID, &fn.Name, &fn.Description, &fn.Code, &fn.OriginalCode, &fn.IsBundled, &fn.BundleError,
			&fn.Version, &fn.CronSchedule, &fn.Enabled,
			&fn.TimeoutSeconds, &fn.MemoryLimitMB, &fn.AllowNet, &fn.AllowEnv, &fn.AllowRead, &fn.AllowWrite, &fn.AllowUnauthenticated,
			&fn.CreatedAt, &fn.UpdatedAt, &fn.CreatedBy,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan function: %w", err)
		}
		functions = append(functions, fn)
	}

	return functions, nil
}

// UpdateFunction updates an existing function
func (s *Storage) UpdateFunction(ctx context.Context, name string, updates map[string]interface{}) error {
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

	query += fmt.Sprintf(" WHERE name = $%d", argCount)
	args = append(args, name)

	_, err := s.db.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("failed to update function: %w", err)
	}

	return nil
}

// DeleteFunction deletes a function by name
func (s *Storage) DeleteFunction(ctx context.Context, name string) error {
	query := "DELETE FROM functions.edge_functions WHERE name = $1"
	_, err := s.db.Exec(ctx, query, name)
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

	err := s.db.QueryRow(ctx, query,
		exec.FunctionID, exec.TriggerType, exec.Status, exec.StatusCode,
		exec.DurationMs, exec.Result, exec.Logs, exec.ErrorMessage, exec.CompletedAt,
	).Scan(&exec.ID, &exec.ExecutedAt)

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

	rows, err := s.db.Query(ctx, query, functionName, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get executions: %w", err)
	}
	defer rows.Close()

	var executions []EdgeFunctionExecution
	for rows.Next() {
		exec := EdgeFunctionExecution{}
		err := rows.Scan(
			&exec.ID, &exec.FunctionID, &exec.TriggerType, &exec.Status, &exec.StatusCode,
			&exec.DurationMs, &exec.Result, &exec.Logs, &exec.ErrorMessage,
			&exec.ExecutedAt, &exec.CompletedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan execution: %w", err)
		}
		executions = append(executions, exec)
	}

	return executions, nil
}
