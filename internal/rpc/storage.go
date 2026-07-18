package rpc

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/rs/zerolog/log"

	"github.com/nimbleflux/fluxbase/internal/database"
)

// Storage handles database operations for RPC entities
type Storage struct {
	database.TenantAware
}

// NewStorage creates a new RPC storage instance
func NewStorage(db *database.Connection) *Storage {
	return &Storage{
		TenantAware: database.TenantAware{DB: db},
	}
}

// ============================================================================
// PROCEDURE OPERATIONS
// ============================================================================

// CreateProcedure creates a new procedure in the database
func (s *Storage) CreateProcedure(ctx context.Context, proc *Procedure) error {
	tenantID := database.TenantFromContext(ctx)
	return s.CreateProcedureWithTenant(ctx, tenantID, proc)
}

// CreateProcedureWithTenant creates a new procedure in the database with tenant context
func (s *Storage) CreateProcedureWithTenant(ctx context.Context, tenantID string, proc *Procedure) error {
	// IMPORTANT: include tenant_id in the INSERT column list explicitly.
	// The chatbots/procedures tables have a BEFORE INSERT trigger
	// (auth.set_tenant_id_from_context) that auto-populates tenant_id from
	// the app.current_tenant_id GUC when NEW.tenant_id IS NULL. But that
	// GUC is only set by WrapWithTenantAwareRole when tenantID != "" — and
	// some call paths (e.g. CreateChatbot wrapper) pass empty string while
	// relying on context propagation. Setting tenant_id explicitly here
	// removes the dependency on the GUC being set and prevents the
	// silent-corruption bug where rows are created with NULL tenant_id and
	// then become invisible to tenant-scoped list queries.
	query := `
		INSERT INTO rpc.procedures (
			id, name, namespace, description, sql_query, original_code,
			input_schema, output_schema, allowed_tables, allowed_schemas,
			max_execution_time_seconds, require_roles, is_public, disable_execution_logs, schedule,
			enabled, version, source, created_by, created_at, updated_at, tenant_id
		) VALUES (
			$1, $2, $3, $4, $5, $6,
			$7, $8, $9, $10,
			$11, $12, $13, $14, $15,
			$16, $17, $18, $19, $20, $21, $22
		)
	`

	if proc.ID == "" {
		proc.ID = uuid.New().String()
	}
	if proc.CreatedAt.IsZero() {
		proc.CreatedAt = time.Now()
	}
	proc.UpdatedAt = time.Now()

	// Resolve tenant ID: prefer the explicit parameter; fall back to context.
	// This matches the resolution done inside WrapWithTenantAwareRole and
	// keeps the INSERT correct regardless of which wrapper the caller used.
	resolvedTenant := tenantID
	if resolvedTenant == "" {
		resolvedTenant = database.TenantFromContext(ctx)
	}

	err := database.WrapWithTenantAwareRole(ctx, s.DB, resolvedTenant, func(tx pgx.Tx) error {
		_, err := tx.Exec(
			ctx, query,
			proc.ID, proc.Name, proc.Namespace, proc.Description, proc.SQLQuery, proc.OriginalCode,
			proc.InputSchema, proc.OutputSchema, proc.AllowedTables, proc.AllowedSchemas,
			proc.MaxExecutionTimeSeconds, proc.RequireRoles, proc.IsPublic, proc.DisableExecutionLogs, proc.Schedule,
			proc.Enabled, proc.Version, proc.Source, proc.CreatedBy, proc.CreatedAt, proc.UpdatedAt,
			database.TenantOrNil(resolvedTenant),
		)
		return err
	})
	if err != nil {
		return fmt.Errorf("failed to create procedure: %w", err)
	}

	log.Info().
		Str("id", proc.ID).
		Str("name", proc.Name).
		Str("namespace", proc.Namespace).
		Str("tenant_id", resolvedTenant).
		Msg("Created RPC procedure")

	return nil
}

// UpdateProcedure updates an existing procedure in the database
func (s *Storage) UpdateProcedure(ctx context.Context, proc *Procedure) error {
	tenantID := database.TenantFromContext(ctx)
	return s.UpdateProcedureWithTenant(ctx, tenantID, proc)
}

func (s *Storage) UpdateProcedureForSync(ctx context.Context, tenantID string, proc *Procedure) error {
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

	return database.WrapWithTenantAwareRole(ctx, s.DB, tenantID, func(tx pgx.Tx) error {
		_, err := tx.Exec(
			ctx, query,
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
		return err
	})
}

// UpdateProcedureWithTenant updates an existing procedure in the database with tenant context
func (s *Storage) UpdateProcedureWithTenant(ctx context.Context, tenantID string, proc *Procedure) error {
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
		  AND (tenant_id = $16 OR ($16 IS NULL AND tenant_id IS NULL))
	`

	proc.UpdatedAt = time.Now()

	var result pgconn.CommandTag
	err := database.WrapWithTenantAwareRole(ctx, s.DB, tenantID, func(tx pgx.Tx) error {
		var execErr error
		result, execErr = tx.Exec(
			ctx, query,
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
			database.TenantOrNil(tenantID),
		)
		return execErr
	})
	if err != nil {
		return fmt.Errorf("failed to update procedure: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("procedure not found: %s", proc.ID)
	}

	log.Info().
		Str("id", proc.ID).
		Str("name", proc.Name).
		Str("tenant_id", tenantID).
		Msg("Updated RPC procedure")

	return nil
}

// GetProcedure retrieves a procedure by ID
func (s *Storage) GetProcedure(ctx context.Context, id string) (*Procedure, error) {
	tenantID := database.TenantFromContext(ctx)
	query := `
		SELECT id, name, namespace, description, sql_query, original_code,
			input_schema, output_schema, allowed_tables, allowed_schemas,
			max_execution_time_seconds, require_roles, is_public, disable_execution_logs, schedule,
			enabled, version, source, created_by, created_at, updated_at
		FROM rpc.procedures
		WHERE id = $1
		  AND (tenant_id = $2 OR ($2 IS NULL AND tenant_id IS NULL))
	`

	proc := &Procedure{}
	err := s.WithTenant(ctx, func(tx pgx.Tx) error {
		return tx.QueryRow(ctx, query, id, database.TenantOrNil(tenantID)).Scan(
			&proc.ID, &proc.Name, &proc.Namespace, &proc.Description, &proc.SQLQuery, &proc.OriginalCode,
			&proc.InputSchema, &proc.OutputSchema, &proc.AllowedTables, &proc.AllowedSchemas,
			&proc.MaxExecutionTimeSeconds, &proc.RequireRoles, &proc.IsPublic, &proc.DisableExecutionLogs, &proc.Schedule,
			&proc.Enabled, &proc.Version, &proc.Source, &proc.CreatedBy, &proc.CreatedAt, &proc.UpdatedAt,
		)
	})

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get procedure: %w", err)
	}

	return proc, nil
}

// GetProcedureByName retrieves a procedure by namespace and name
func (s *Storage) GetProcedureByName(ctx context.Context, namespace, name string) (*Procedure, error) {
	tenantID := database.TenantFromContext(ctx)
	query := `
		SELECT id, name, namespace, description, sql_query, original_code,
			input_schema, output_schema, allowed_tables, allowed_schemas,
			max_execution_time_seconds, require_roles, is_public, disable_execution_logs, schedule,
			enabled, version, source, created_by, created_at, updated_at
		FROM rpc.procedures
		WHERE namespace = $1 AND name = $2
		  AND (tenant_id = $3 OR ($3 IS NULL AND tenant_id IS NULL))
	`

	proc := &Procedure{}
	err := s.WithTenant(ctx, func(tx pgx.Tx) error {
		return tx.QueryRow(ctx, query, namespace, name, database.TenantOrNil(tenantID)).Scan(
			&proc.ID, &proc.Name, &proc.Namespace, &proc.Description, &proc.SQLQuery, &proc.OriginalCode,
			&proc.InputSchema, &proc.OutputSchema, &proc.AllowedTables, &proc.AllowedSchemas,
			&proc.MaxExecutionTimeSeconds, &proc.RequireRoles, &proc.IsPublic, &proc.DisableExecutionLogs, &proc.Schedule,
			&proc.Enabled, &proc.Version, &proc.Source, &proc.CreatedBy, &proc.CreatedAt, &proc.UpdatedAt,
		)
	})

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get procedure by name: %w", err)
	}

	return proc, nil
}

// ListProcedures lists all procedures, optionally filtered by namespace
func (s *Storage) ListProcedures(ctx context.Context, namespace string) ([]*Procedure, error) {
	tenantID := database.TenantFromContext(ctx)
	var query string
	var args []interface{}

	if namespace != "" {
		query = `
			SELECT id, name, namespace, description, sql_query, original_code,
				input_schema, output_schema, allowed_tables, allowed_schemas,
				max_execution_time_seconds, require_roles, is_public, disable_execution_logs, schedule,
				enabled, version, source, created_by, created_at, updated_at
			FROM rpc.procedures
			WHERE namespace = $1 AND (tenant_id = $2 OR ($2 IS NULL AND tenant_id IS NULL))
			ORDER BY name ASC
		`
		args = []interface{}{namespace, database.TenantOrNil(tenantID)}
	} else {
		query = `
			SELECT id, name, namespace, description, sql_query, original_code,
				input_schema, output_schema, allowed_tables, allowed_schemas,
				max_execution_time_seconds, require_roles, is_public, disable_execution_logs, schedule,
				enabled, version, source, created_by, created_at, updated_at
			FROM rpc.procedures
			WHERE (tenant_id = $1 OR ($1 IS NULL AND tenant_id IS NULL))
			ORDER BY namespace ASC, name ASC
		`
		args = []interface{}{database.TenantOrNil(tenantID)}
	}
	var procedures []*Procedure
	err := s.WithTenant(ctx, func(tx pgx.Tx) error {
		rows, queryErr := tx.Query(ctx, query, args...)
		if queryErr != nil {
			return queryErr
		}
		defer rows.Close()

		for rows.Next() {
			proc := &Procedure{}
			if scanErr := rows.Scan(
				&proc.ID, &proc.Name, &proc.Namespace, &proc.Description, &proc.SQLQuery, &proc.OriginalCode,
				&proc.InputSchema, &proc.OutputSchema, &proc.AllowedTables, &proc.AllowedSchemas,
				&proc.MaxExecutionTimeSeconds, &proc.RequireRoles, &proc.IsPublic, &proc.DisableExecutionLogs, &proc.Schedule,
				&proc.Enabled, &proc.Version, &proc.Source, &proc.CreatedBy, &proc.CreatedAt, &proc.UpdatedAt,
			); scanErr != nil {
				return scanErr
			}
			procedures = append(procedures, proc)
		}
		return rows.Err()
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list procedures: %w", err)
	}

	return procedures, nil
}

// ListProceduresForSync lists procedures matching the given tenant OR with NULL tenant_id.
// Used by sync flows to find existing procedures regardless of backfill state.
func (s *Storage) ListProceduresForSync(ctx context.Context, namespace string, tenantID string) ([]*Procedure, error) {
	query := `
		SELECT id, name, namespace, description, sql_query, original_code,
			input_schema, output_schema, allowed_tables, allowed_schemas,
			max_execution_time_seconds, require_roles, is_public, disable_execution_logs, schedule,
			enabled, version, source, created_by, created_at, updated_at
		FROM rpc.procedures
		WHERE namespace = $1 AND (tenant_id = $2 OR tenant_id IS NULL)
		ORDER BY name ASC
	`
	var procedures []*Procedure
	err := database.WrapWithServiceRoleAndTenant(ctx, s.DB, tenantID, func(tx pgx.Tx) error {
		rows, queryErr := tx.Query(ctx, query, namespace, database.TenantOrNil(tenantID))
		if queryErr != nil {
			return queryErr
		}
		defer rows.Close()

		for rows.Next() {
			proc := &Procedure{}
			if scanErr := rows.Scan(
				&proc.ID, &proc.Name, &proc.Namespace, &proc.Description, &proc.SQLQuery, &proc.OriginalCode,
				&proc.InputSchema, &proc.OutputSchema, &proc.AllowedTables, &proc.AllowedSchemas,
				&proc.MaxExecutionTimeSeconds, &proc.RequireRoles, &proc.IsPublic, &proc.DisableExecutionLogs, &proc.Schedule,
				&proc.Enabled, &proc.Version, &proc.Source, &proc.CreatedBy, &proc.CreatedAt, &proc.UpdatedAt,
			); scanErr != nil {
				return scanErr
			}
			procedures = append(procedures, proc)
		}
		return rows.Err()
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list procedures for sync: %w", err)
	}

	return procedures, nil
}

// ListPublicProcedures lists all public and enabled procedures
func (s *Storage) ListPublicProcedures(ctx context.Context, namespace string) ([]*ProcedureSummary, error) {
	tenantID := database.TenantFromContext(ctx)
	var query string
	var args []interface{}

	if namespace != "" {
		query = `
			SELECT id, name, namespace, description, allowed_tables, allowed_schemas,
				max_execution_time_seconds, require_roles, is_public, disable_execution_logs, schedule,
				enabled, version, source, created_at, updated_at
			FROM rpc.procedures
			WHERE namespace = $1 AND enabled = true AND is_public = true AND (tenant_id = $2 OR ($2 IS NULL AND tenant_id IS NULL))
			ORDER BY name ASC
		`
		args = []interface{}{namespace, database.TenantOrNil(tenantID)}
	} else {
		query = `
			SELECT id, name, namespace, description, allowed_tables, allowed_schemas,
				max_execution_time_seconds, require_roles, is_public, disable_execution_logs, schedule,
				enabled, version, source, created_at, updated_at
			FROM rpc.procedures
			WHERE enabled = true AND is_public = true AND (tenant_id = $1 OR ($1 IS NULL AND tenant_id IS NULL))
			ORDER BY namespace ASC, name ASC
		`
		args = []interface{}{database.TenantOrNil(tenantID)}
	}
	var procedures []*ProcedureSummary
	err := s.WithTenant(ctx, func(tx pgx.Tx) error {
		rows, queryErr := tx.Query(ctx, query, args...)
		if queryErr != nil {
			return queryErr
		}
		defer rows.Close()

		for rows.Next() {
			proc := &ProcedureSummary{}
			if scanErr := rows.Scan(
				&proc.ID, &proc.Name, &proc.Namespace, &proc.Description, &proc.AllowedTables, &proc.AllowedSchemas,
				&proc.MaxExecutionTimeSeconds, &proc.RequireRoles, &proc.IsPublic, &proc.DisableExecutionLogs, &proc.Schedule,
				&proc.Enabled, &proc.Version, &proc.Source, &proc.CreatedAt, &proc.UpdatedAt,
			); scanErr != nil {
				return scanErr
			}
			procedures = append(procedures, proc)
		}
		return rows.Err()
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list public procedures: %w", err)
	}

	return procedures, nil
}

// DeleteProcedure deletes a procedure by ID
func (s *Storage) DeleteProcedure(ctx context.Context, id string) error {
	tenantID := database.TenantFromContext(ctx)
	return s.DeleteProcedureWithTenant(ctx, tenantID, id)
}

// DeleteProcedureWithTenant deletes a procedure by ID with tenant context
func (s *Storage) DeleteProcedureWithTenant(ctx context.Context, tenantID string, id string) error {
	query := `DELETE FROM rpc.procedures WHERE id = $1 AND (tenant_id = $2 OR ($2 IS NULL AND tenant_id IS NULL))`

	var result pgconn.CommandTag
	err := database.WrapWithTenantAwareRole(ctx, s.DB, tenantID, func(tx pgx.Tx) error {
		var execErr error
		result, execErr = tx.Exec(ctx, query, id, database.TenantOrNil(tenantID))
		return execErr
	})
	if err != nil {
		return fmt.Errorf("failed to delete procedure: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("procedure not found: %s", id)
	}

	log.Info().Str("id", id).Str("tenant_id", tenantID).Msg("Deleted RPC procedure")
	return nil
}

func (s *Storage) DeleteProcedureForSync(ctx context.Context, tenantID string, id string) error {
	query := `DELETE FROM rpc.procedures WHERE id = $1 AND (tenant_id = $2 OR tenant_id IS NULL)`

	var result pgconn.CommandTag
	err := database.WrapWithTenantAwareRole(ctx, s.DB, tenantID, func(tx pgx.Tx) error {
		var execErr error
		result, execErr = tx.Exec(ctx, query, id, database.TenantOrNil(tenantID))
		return execErr
	})
	if err != nil {
		return fmt.Errorf("failed to delete procedure: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("procedure not found: %s", id)
	}

	return nil
}

// DeleteProcedureByName deletes a procedure by namespace and name
func (s *Storage) DeleteProcedureByName(ctx context.Context, namespace, name string) error {
	tenantID := database.TenantFromContext(ctx)
	return s.DeleteProcedureByNameWithTenant(ctx, tenantID, namespace, name)
}

// DeleteProcedureByNameWithTenant deletes a procedure by namespace and name with tenant context
func (s *Storage) DeleteProcedureByNameWithTenant(ctx context.Context, tenantID string, namespace, name string) error {
	query := `DELETE FROM rpc.procedures WHERE namespace = $1 AND name = $2 AND (tenant_id = $3 OR ($3 IS NULL AND tenant_id IS NULL))`

	var result pgconn.CommandTag
	err := database.WrapWithTenantAwareRole(ctx, s.DB, tenantID, func(tx pgx.Tx) error {
		var execErr error
		result, execErr = tx.Exec(ctx, query, namespace, name, database.TenantOrNil(tenantID))
		return execErr
	})
	if err != nil {
		return fmt.Errorf("failed to delete procedure: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("procedure not found: %s/%s", namespace, name)
	}

	log.Info().
		Str("namespace", namespace).
		Str("name", name).
		Str("tenant_id", tenantID).
		Msg("Deleted RPC procedure")

	return nil
}

// ListNamespaces lists all unique namespaces
func (s *Storage) ListNamespaces(ctx context.Context) ([]string, error) {
	query := `SELECT DISTINCT namespace FROM rpc.procedures WHERE (tenant_id = $1 OR ($1 IS NULL AND tenant_id IS NULL)) ORDER BY namespace ASC`

	tenantID := database.TenantFromContext(ctx)
	var namespaces []string
	err := s.WithTenant(ctx, func(tx pgx.Tx) error {
		rows, queryErr := tx.Query(ctx, query, database.TenantOrNil(tenantID))
		if queryErr != nil {
			return queryErr
		}
		defer rows.Close()

		for rows.Next() {
			var ns string
			if scanErr := rows.Scan(&ns); scanErr != nil {
				return scanErr
			}
			namespaces = append(namespaces, ns)
		}
		return rows.Err()
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list namespaces: %w", err)
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

	var procedures []*Procedure
	err := s.WithTenant(ctx, func(tx pgx.Tx) error {
		rows, queryErr := tx.Query(ctx, query)
		if queryErr != nil {
			return queryErr
		}
		defer rows.Close()

		for rows.Next() {
			proc := &Procedure{}
			if scanErr := rows.Scan(
				&proc.ID, &proc.Name, &proc.Namespace, &proc.Description, &proc.SQLQuery, &proc.OriginalCode,
				&proc.InputSchema, &proc.OutputSchema, &proc.AllowedTables, &proc.AllowedSchemas,
				&proc.MaxExecutionTimeSeconds, &proc.RequireRoles, &proc.IsPublic, &proc.DisableExecutionLogs, &proc.Schedule,
				&proc.Enabled, &proc.Version, &proc.Source, &proc.CreatedBy, &proc.CreatedAt, &proc.UpdatedAt,
			); scanErr != nil {
				return scanErr
			}
			procedures = append(procedures, proc)
		}
		return rows.Err()
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list scheduled procedures: %w", err)
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

	err := s.WithTenant(ctx, func(tx pgx.Tx) error {
		_, execErr := tx.Exec(
			ctx, query,
			exec.ID, exec.ProcedureID, exec.ProcedureName, exec.Namespace, exec.Status,
			exec.InputParams, exec.Result, exec.ErrorMessage, exec.RowsReturned, exec.DurationMs,
			exec.UserID, exec.UserRole, exec.UserEmail, exec.IsAsync,
			exec.CreatedAt, exec.StartedAt, exec.CompletedAt,
		)
		return execErr
	})
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

	var result pgconn.CommandTag
	err := s.WithTenant(ctx, func(tx pgx.Tx) error {
		var execErr error
		result, execErr = tx.Exec(
			ctx, query,
			exec.ID,
			exec.Status,
			exec.Result,
			exec.ErrorMessage,
			exec.RowsReturned,
			exec.DurationMs,
			exec.StartedAt,
			exec.CompletedAt,
		)
		return execErr
	})
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

	var result pgconn.CommandTag
	err := s.WithTenant(ctx, func(tx pgx.Tx) error {
		var execErr error
		result, execErr = tx.Exec(ctx, query, id, StatusCancelled, StatusPending, StatusRunning)
		return execErr
	})
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
	tenantID := database.TenantFromContext(ctx)
	query := `
		SELECT id, procedure_id, procedure_name, namespace, status,
			input_params, result, error_message, rows_returned, duration_ms,
			user_id, user_role, user_email, is_async,
			created_at, started_at, completed_at
		FROM rpc.executions
		WHERE id = $1
		  AND (tenant_id = $2 OR ($2 IS NULL AND tenant_id IS NULL))
	`

	exec := &Execution{}
	err := s.WithTenant(ctx, func(tx pgx.Tx) error {
		return tx.QueryRow(ctx, query, id, database.TenantOrNil(tenantID)).Scan(
			&exec.ID, &exec.ProcedureID, &exec.ProcedureName, &exec.Namespace, &exec.Status,
			&exec.InputParams, &exec.Result, &exec.ErrorMessage, &exec.RowsReturned, &exec.DurationMs,
			&exec.UserID, &exec.UserRole, &exec.UserEmail, &exec.IsAsync,
			&exec.CreatedAt, &exec.StartedAt, &exec.CompletedAt,
		)
	})

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get execution: %w", err)
	}

	return exec, nil
}

// ListExecutions lists executions with optional filters
func (s *Storage) ListExecutions(ctx context.Context, opts ListExecutionsOptions) ([]*Execution, error) {
	tenantID := database.TenantFromContext(ctx)
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

	// Tenant filter (first dynamic filter)
	query += fmt.Sprintf(" AND (tenant_id = $%d OR ($%d IS NULL AND tenant_id IS NULL))", argIndex, argIndex)
	args = append(args, database.TenantOrNil(tenantID))
	argIndex++

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

	var executions []*Execution
	err := s.WithTenant(ctx, func(tx pgx.Tx) error {
		rows, queryErr := tx.Query(ctx, query, args...)
		if queryErr != nil {
			return queryErr
		}
		defer rows.Close()

		for rows.Next() {
			exec := &Execution{}
			if scanErr := rows.Scan(
				&exec.ID, &exec.ProcedureID, &exec.ProcedureName, &exec.Namespace, &exec.Status,
				&exec.InputParams, &exec.Result, &exec.ErrorMessage, &exec.RowsReturned, &exec.DurationMs,
				&exec.UserID, &exec.UserRole, &exec.UserEmail, &exec.IsAsync,
				&exec.CreatedAt, &exec.StartedAt, &exec.CompletedAt,
			); scanErr != nil {
				return scanErr
			}
			executions = append(executions, exec)
		}
		return rows.Err()
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list executions: %w", err)
	}

	return executions, nil
}

// Note: Execution logs are now stored in the central logging schema (logging.entries)
