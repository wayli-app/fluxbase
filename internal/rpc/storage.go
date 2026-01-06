package rpc

import (
	"context"
	"fmt"
	"time"

	"github.com/fluxbase-eu/fluxbase/internal/database"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/rs/zerolog/log"
)

// Storage handles database operations for RPC entities
type Storage struct {
	db *database.Connection
}

// NewStorage creates a new RPC storage instance
func NewStorage(db *database.Connection) *Storage {
	return &Storage{
		db: db,
	}
}

// ============================================================================
// PROCEDURE OPERATIONS
// ============================================================================

// CreateProcedure creates a new procedure in the database
func (s *Storage) CreateProcedure(ctx context.Context, proc *Procedure) error {
	query := `
		INSERT INTO rpc.procedures (
			id, name, namespace, description, sql_query, original_code,
			input_schema, output_schema, allowed_tables, allowed_schemas,
			max_execution_time_seconds, require_roles, is_public, disable_execution_logs, schedule,
			enabled, version, source, created_by, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6,
			$7, $8, $9, $10,
			$11, $12, $13, $14, $15,
			$16, $17, $18, $19, $20, $21
		)
	`

	if proc.ID == "" {
		proc.ID = uuid.New().String()
	}
	if proc.CreatedAt.IsZero() {
		proc.CreatedAt = time.Now()
	}
	proc.UpdatedAt = time.Now()

	_, err := s.db.Exec(ctx, query,
		proc.ID, proc.Name, proc.Namespace, proc.Description, proc.SQLQuery, proc.OriginalCode,
		proc.InputSchema, proc.OutputSchema, proc.AllowedTables, proc.AllowedSchemas,
		proc.MaxExecutionTimeSeconds, proc.RequireRoles, proc.IsPublic, proc.DisableExecutionLogs, proc.Schedule,
		proc.Enabled, proc.Version, proc.Source, proc.CreatedBy, proc.CreatedAt, proc.UpdatedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to create procedure: %w", err)
	}

	log.Info().
		Str("id", proc.ID).
		Str("name", proc.Name).
		Str("namespace", proc.Namespace).
		Msg("Created RPC procedure")

	return nil
}

// UpdateProcedure updates an existing procedure in the database
func (s *Storage) UpdateProcedure(ctx context.Context, proc *Procedure) error {
	query := `
		UPDATE rpc.procedures SET
			description = $2,
			sql_query = $3,
			original_code = $4,
			input_schema = $5,
			output_schema = $6,
			allowed_tables = $7,
			allowed_schemas = $8,
			max_execution_time_seconds = $9,
			require_roles = $10,
			is_public = $11,
			disable_execution_logs = $12,
			schedule = $13,
			enabled = $14,
			version = version + 1,
			updated_at = $15
		WHERE id = $1
	`

	proc.UpdatedAt = time.Now()

	result, err := s.db.Exec(ctx, query,
		proc.ID,
		proc.Description,
		proc.SQLQuery,
		proc.OriginalCode,
		proc.InputSchema,
		proc.OutputSchema,
		proc.AllowedTables,
		proc.AllowedSchemas,
		proc.MaxExecutionTimeSeconds,
		proc.RequireRoles,
		proc.IsPublic,
		proc.DisableExecutionLogs,
		proc.Schedule,
		proc.Enabled,
		proc.UpdatedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to update procedure: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("procedure not found: %s", proc.ID)
	}

	log.Info().
		Str("id", proc.ID).
		Str("name", proc.Name).
		Msg("Updated RPC procedure")

	return nil
}

// GetProcedure retrieves a procedure by ID
func (s *Storage) GetProcedure(ctx context.Context, id string) (*Procedure, error) {
	query := `
		SELECT id, name, namespace, description, sql_query, original_code,
			input_schema, output_schema, allowed_tables, allowed_schemas,
			max_execution_time_seconds, require_roles, is_public, disable_execution_logs, schedule,
			enabled, version, source, created_by, created_at, updated_at
		FROM rpc.procedures
		WHERE id = $1
	`

	proc := &Procedure{}
	err := s.db.Pool().QueryRow(ctx, query, id).Scan(
		&proc.ID, &proc.Name, &proc.Namespace, &proc.Description, &proc.SQLQuery, &proc.OriginalCode,
		&proc.InputSchema, &proc.OutputSchema, &proc.AllowedTables, &proc.AllowedSchemas,
		&proc.MaxExecutionTimeSeconds, &proc.RequireRoles, &proc.IsPublic, &proc.DisableExecutionLogs, &proc.Schedule,
		&proc.Enabled, &proc.Version, &proc.Source, &proc.CreatedBy, &proc.CreatedAt, &proc.UpdatedAt,
	)

	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get procedure: %w", err)
	}

	return proc, nil
}

// GetProcedureByName retrieves a procedure by namespace and name
func (s *Storage) GetProcedureByName(ctx context.Context, namespace, name string) (*Procedure, error) {
	query := `
		SELECT id, name, namespace, description, sql_query, original_code,
			input_schema, output_schema, allowed_tables, allowed_schemas,
			max_execution_time_seconds, require_roles, is_public, disable_execution_logs, schedule,
			enabled, version, source, created_by, created_at, updated_at
		FROM rpc.procedures
		WHERE namespace = $1 AND name = $2
	`

	proc := &Procedure{}
	err := s.db.Pool().QueryRow(ctx, query, namespace, name).Scan(
		&proc.ID, &proc.Name, &proc.Namespace, &proc.Description, &proc.SQLQuery, &proc.OriginalCode,
		&proc.InputSchema, &proc.OutputSchema, &proc.AllowedTables, &proc.AllowedSchemas,
		&proc.MaxExecutionTimeSeconds, &proc.RequireRoles, &proc.IsPublic, &proc.DisableExecutionLogs, &proc.Schedule,
		&proc.Enabled, &proc.Version, &proc.Source, &proc.CreatedBy, &proc.CreatedAt, &proc.UpdatedAt,
	)

	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get procedure by name: %w", err)
	}

	return proc, nil
}

// ListProcedures lists all procedures, optionally filtered by namespace
func (s *Storage) ListProcedures(ctx context.Context, namespace string) ([]*Procedure, error) {
	var query string
	var args []interface{}

	if namespace != "" {
		query = `
			SELECT id, name, namespace, description, sql_query, original_code,
				input_schema, output_schema, allowed_tables, allowed_schemas,
				max_execution_time_seconds, require_roles, is_public, disable_execution_logs, schedule,
				enabled, version, source, created_by, created_at, updated_at
			FROM rpc.procedures
			WHERE namespace = $1
			ORDER BY name ASC
		`
		args = []interface{}{namespace}
	} else {
		query = `
			SELECT id, name, namespace, description, sql_query, original_code,
				input_schema, output_schema, allowed_tables, allowed_schemas,
				max_execution_time_seconds, require_roles, is_public, disable_execution_logs, schedule,
				enabled, version, source, created_by, created_at, updated_at
			FROM rpc.procedures
			ORDER BY namespace ASC, name ASC
		`
	}

	rows, err := s.db.Pool().Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list procedures: %w", err)
	}
	defer rows.Close()

	var procedures []*Procedure
	for rows.Next() {
		proc := &Procedure{}
		err := rows.Scan(
			&proc.ID, &proc.Name, &proc.Namespace, &proc.Description, &proc.SQLQuery, &proc.OriginalCode,
			&proc.InputSchema, &proc.OutputSchema, &proc.AllowedTables, &proc.AllowedSchemas,
			&proc.MaxExecutionTimeSeconds, &proc.RequireRoles, &proc.IsPublic, &proc.DisableExecutionLogs, &proc.Schedule,
			&proc.Enabled, &proc.Version, &proc.Source, &proc.CreatedBy, &proc.CreatedAt, &proc.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan procedure: %w", err)
		}
		procedures = append(procedures, proc)
	}

	return procedures, nil
}

// ListPublicProcedures lists all public and enabled procedures
func (s *Storage) ListPublicProcedures(ctx context.Context, namespace string) ([]*ProcedureSummary, error) {
	var query string
	var args []interface{}

	if namespace != "" {
		query = `
			SELECT id, name, namespace, description, allowed_tables, allowed_schemas,
				max_execution_time_seconds, require_roles, is_public, disable_execution_logs, schedule,
				enabled, version, source, created_at, updated_at
			FROM rpc.procedures
			WHERE namespace = $1 AND enabled = true AND is_public = true
			ORDER BY name ASC
		`
		args = []interface{}{namespace}
	} else {
		query = `
			SELECT id, name, namespace, description, allowed_tables, allowed_schemas,
				max_execution_time_seconds, require_roles, is_public, disable_execution_logs, schedule,
				enabled, version, source, created_at, updated_at
			FROM rpc.procedures
			WHERE enabled = true AND is_public = true
			ORDER BY namespace ASC, name ASC
		`
	}

	rows, err := s.db.Pool().Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list public procedures: %w", err)
	}
	defer rows.Close()

	var procedures []*ProcedureSummary
	for rows.Next() {
		proc := &ProcedureSummary{}
		err := rows.Scan(
			&proc.ID, &proc.Name, &proc.Namespace, &proc.Description, &proc.AllowedTables, &proc.AllowedSchemas,
			&proc.MaxExecutionTimeSeconds, &proc.RequireRoles, &proc.IsPublic, &proc.DisableExecutionLogs, &proc.Schedule,
			&proc.Enabled, &proc.Version, &proc.Source, &proc.CreatedAt, &proc.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan procedure: %w", err)
		}
		procedures = append(procedures, proc)
	}

	return procedures, nil
}

// DeleteProcedure deletes a procedure by ID
func (s *Storage) DeleteProcedure(ctx context.Context, id string) error {
	query := `DELETE FROM rpc.procedures WHERE id = $1`

	result, err := s.db.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete procedure: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("procedure not found: %s", id)
	}

	log.Info().Str("id", id).Msg("Deleted RPC procedure")
	return nil
}

// DeleteProcedureByName deletes a procedure by namespace and name
func (s *Storage) DeleteProcedureByName(ctx context.Context, namespace, name string) error {
	query := `DELETE FROM rpc.procedures WHERE namespace = $1 AND name = $2`

	result, err := s.db.Exec(ctx, query, namespace, name)
	if err != nil {
		return fmt.Errorf("failed to delete procedure: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("procedure not found: %s/%s", namespace, name)
	}

	log.Info().
		Str("namespace", namespace).
		Str("name", name).
		Msg("Deleted RPC procedure")

	return nil
}

// ListNamespaces lists all unique namespaces
func (s *Storage) ListNamespaces(ctx context.Context) ([]string, error) {
	query := `SELECT DISTINCT namespace FROM rpc.procedures ORDER BY namespace ASC`

	rows, err := s.db.Pool().Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to list namespaces: %w", err)
	}
	defer rows.Close()

	var namespaces []string
	for rows.Next() {
		var ns string
		if err := rows.Scan(&ns); err != nil {
			return nil, fmt.Errorf("failed to scan namespace: %w", err)
		}
		namespaces = append(namespaces, ns)
	}

	return namespaces, nil
}

// ListScheduledProcedures returns all enabled procedures with a schedule
func (s *Storage) ListScheduledProcedures(ctx context.Context) ([]*Procedure, error) {
	query := `
		SELECT id, name, namespace, description, sql_query, original_code,
			input_schema, output_schema, allowed_tables, allowed_schemas,
			max_execution_time_seconds, require_roles, is_public, disable_execution_logs, schedule,
			enabled, version, source, created_by, created_at, updated_at
		FROM rpc.procedures
		WHERE enabled = true AND schedule IS NOT NULL AND schedule != ''
		ORDER BY namespace ASC, name ASC
	`

	rows, err := s.db.Pool().Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to list scheduled procedures: %w", err)
	}
	defer rows.Close()

	var procedures []*Procedure
	for rows.Next() {
		proc := &Procedure{}
		err := rows.Scan(
			&proc.ID, &proc.Name, &proc.Namespace, &proc.Description, &proc.SQLQuery, &proc.OriginalCode,
			&proc.InputSchema, &proc.OutputSchema, &proc.AllowedTables, &proc.AllowedSchemas,
			&proc.MaxExecutionTimeSeconds, &proc.RequireRoles, &proc.IsPublic, &proc.DisableExecutionLogs, &proc.Schedule,
			&proc.Enabled, &proc.Version, &proc.Source, &proc.CreatedBy, &proc.CreatedAt, &proc.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan procedure: %w", err)
		}
		procedures = append(procedures, proc)
	}

	return procedures, nil
}

// ============================================================================
// EXECUTION OPERATIONS
// ============================================================================

// CreateExecution creates a new execution record
func (s *Storage) CreateExecution(ctx context.Context, exec *Execution) error {
	query := `
		INSERT INTO rpc.executions (
			id, procedure_id, procedure_name, namespace, status,
			input_params, result, error_message, rows_returned, duration_ms,
			user_id, user_role, user_email, is_async,
			created_at, started_at, completed_at
		) VALUES (
			$1, $2, $3, $4, $5,
			$6, $7, $8, $9, $10,
			$11, $12, $13, $14,
			$15, $16, $17
		)
	`

	if exec.ID == "" {
		exec.ID = uuid.New().String()
	}
	if exec.CreatedAt.IsZero() {
		exec.CreatedAt = time.Now()
	}

	_, err := s.db.Exec(ctx, query,
		exec.ID, exec.ProcedureID, exec.ProcedureName, exec.Namespace, exec.Status,
		exec.InputParams, exec.Result, exec.ErrorMessage, exec.RowsReturned, exec.DurationMs,
		exec.UserID, exec.UserRole, exec.UserEmail, exec.IsAsync,
		exec.CreatedAt, exec.StartedAt, exec.CompletedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to create execution: %w", err)
	}

	return nil
}

// UpdateExecution updates an existing execution record
func (s *Storage) UpdateExecution(ctx context.Context, exec *Execution) error {
	query := `
		UPDATE rpc.executions SET
			status = $2,
			result = $3,
			error_message = $4,
			rows_returned = $5,
			duration_ms = $6,
			started_at = $7,
			completed_at = $8
		WHERE id = $1
	`

	result, err := s.db.Exec(ctx, query,
		exec.ID,
		exec.Status,
		exec.Result,
		exec.ErrorMessage,
		exec.RowsReturned,
		exec.DurationMs,
		exec.StartedAt,
		exec.CompletedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to update execution: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("execution not found: %s", exec.ID)
	}

	return nil
}

// CancelExecution cancels a pending or running execution
func (s *Storage) CancelExecution(ctx context.Context, id string) error {
	query := `
		UPDATE rpc.executions SET
			status = $2,
			completed_at = NOW()
		WHERE id = $1 AND status IN ($3, $4)
	`

	result, err := s.db.Exec(ctx, query, id, StatusCancelled, StatusPending, StatusRunning)
	if err != nil {
		return fmt.Errorf("failed to cancel execution: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("execution not found or cannot be cancelled: %s", id)
	}

	return nil
}

// GetExecution retrieves an execution by ID
func (s *Storage) GetExecution(ctx context.Context, id string) (*Execution, error) {
	query := `
		SELECT id, procedure_id, procedure_name, namespace, status,
			input_params, result, error_message, rows_returned, duration_ms,
			user_id, user_role, user_email, is_async,
			created_at, started_at, completed_at
		FROM rpc.executions
		WHERE id = $1
	`

	exec := &Execution{}
	err := s.db.Pool().QueryRow(ctx, query, id).Scan(
		&exec.ID, &exec.ProcedureID, &exec.ProcedureName, &exec.Namespace, &exec.Status,
		&exec.InputParams, &exec.Result, &exec.ErrorMessage, &exec.RowsReturned, &exec.DurationMs,
		&exec.UserID, &exec.UserRole, &exec.UserEmail, &exec.IsAsync,
		&exec.CreatedAt, &exec.StartedAt, &exec.CompletedAt,
	)

	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get execution: %w", err)
	}

	return exec, nil
}

// ListExecutions lists executions with optional filters
func (s *Storage) ListExecutions(ctx context.Context, opts ListExecutionsOptions) ([]*Execution, error) {
	query := `
		SELECT id, procedure_id, procedure_name, namespace, status,
			input_params, result, error_message, rows_returned, duration_ms,
			user_id, user_role, user_email, is_async,
			created_at, started_at, completed_at
		FROM rpc.executions
		WHERE 1=1
	`
	args := []interface{}{}
	argIndex := 1

	if opts.Namespace != "" {
		query += fmt.Sprintf(" AND namespace = $%d", argIndex)
		args = append(args, opts.Namespace)
		argIndex++
	}

	if opts.ProcedureName != "" {
		query += fmt.Sprintf(" AND procedure_name = $%d", argIndex)
		args = append(args, opts.ProcedureName)
		argIndex++
	}

	if opts.Status != "" {
		query += fmt.Sprintf(" AND status = $%d", argIndex)
		args = append(args, opts.Status)
		argIndex++
	}

	if opts.UserID != "" {
		query += fmt.Sprintf(" AND user_id = $%d", argIndex)
		args = append(args, opts.UserID)
		argIndex++
	}

	query += " ORDER BY created_at DESC"

	if opts.Limit > 0 {
		query += fmt.Sprintf(" LIMIT $%d", argIndex)
		args = append(args, opts.Limit)
		argIndex++
	} else {
		query += " LIMIT 100" // Default limit
	}

	if opts.Offset > 0 {
		query += fmt.Sprintf(" OFFSET $%d", argIndex)
		args = append(args, opts.Offset)
	}

	rows, err := s.db.Pool().Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list executions: %w", err)
	}
	defer rows.Close()

	var executions []*Execution
	for rows.Next() {
		exec := &Execution{}
		err := rows.Scan(
			&exec.ID, &exec.ProcedureID, &exec.ProcedureName, &exec.Namespace, &exec.Status,
			&exec.InputParams, &exec.Result, &exec.ErrorMessage, &exec.RowsReturned, &exec.DurationMs,
			&exec.UserID, &exec.UserRole, &exec.UserEmail, &exec.IsAsync,
			&exec.CreatedAt, &exec.StartedAt, &exec.CompletedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan execution: %w", err)
		}
		executions = append(executions, exec)
	}

	return executions, nil
}

// Note: Execution logs are now stored in the central logging schema (logging.entries)
