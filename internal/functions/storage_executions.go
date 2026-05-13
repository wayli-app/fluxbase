package functions

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/nimbleflux/fluxbase/internal/database"
)

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
		return tx.QueryRow(
			ctx, query,
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
