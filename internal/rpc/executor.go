package rpc

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/fluxbase-eu/fluxbase/internal/auth"
	"github.com/fluxbase-eu/fluxbase/internal/config"
	"github.com/fluxbase-eu/fluxbase/internal/database"
	"github.com/fluxbase-eu/fluxbase/internal/middleware"
	"github.com/fluxbase-eu/fluxbase/internal/observability"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/rs/zerolog/log"
)

// Executor handles RPC procedure execution
type Executor struct {
	db        *database.Connection
	storage   *Storage
	validator *Validator
	metrics   *observability.Metrics
	config    *config.RPCConfig
}

// NewExecutor creates a new RPC executor
func NewExecutor(db *database.Connection, storage *Storage, metrics *observability.Metrics, cfg *config.RPCConfig) *Executor {
	return &Executor{
		db:        db,
		storage:   storage,
		validator: NewValidator(),
		metrics:   metrics,
		config:    cfg,
	}
}

// ExecuteContext contains the context for an RPC execution
type ExecuteContext struct {
	Procedure            *Procedure
	Params               map[string]interface{}
	UserID               string
	UserRole             string
	UserEmail            string
	Claims               *auth.TokenClaims
	IsAsync              bool
	ExecutionID          string // If set, reuse this execution record instead of creating a new one
	DisableExecutionLogs bool   // If true, skip creating execution records and logs
}

// ExecuteResult represents the result of an RPC execution
type ExecuteResult struct {
	ExecutionID  string          `json:"execution_id"`
	Status       ExecutionStatus `json:"status"`
	Result       json.RawMessage `json:"result,omitempty"`
	RowsReturned *int            `json:"rows_returned,omitempty"`
	DurationMs   *int            `json:"duration_ms,omitempty"`
	Error        *string         `json:"error,omitempty"`
}

// Execute executes an RPC procedure synchronously
func (e *Executor) Execute(ctx context.Context, execCtx *ExecuteContext) (*ExecuteResult, error) {
	start := time.Now()

	var exec *Execution

	// Check if we're continuing an existing execution (async case)
	if execCtx.ExecutionID != "" {
		// Reuse existing execution record - update it to running status
		exec = &Execution{
			ID:            execCtx.ExecutionID,
			ProcedureID:   &execCtx.Procedure.ID,
			ProcedureName: execCtx.Procedure.Name,
			Namespace:     execCtx.Procedure.Namespace,
			Status:        StatusRunning,
			IsAsync:       execCtx.IsAsync,
		}

		// Set started time
		now := time.Now()
		exec.StartedAt = &now

		// For async executions (ExecutionID is set), always update status to running
		// since the record was created in ExecuteAsync and user expects to poll for status
		if err := e.storage.UpdateExecution(ctx, exec); err != nil {
			log.Error().Err(err).Msg("Failed to update execution record to running")
		}
	} else {
		// Create new execution record (sync case)
		exec = &Execution{
			ID:            uuid.New().String(),
			ProcedureID:   &execCtx.Procedure.ID,
			ProcedureName: execCtx.Procedure.Name,
			Namespace:     execCtx.Procedure.Namespace,
			Status:        StatusRunning,
			IsAsync:       execCtx.IsAsync,
			CreatedAt:     time.Now(),
		}

		// Set optional user fields (nil if empty, to store as NULL in database)
		if execCtx.UserID != "" {
			exec.UserID = &execCtx.UserID
		}
		if execCtx.UserRole != "" {
			exec.UserRole = &execCtx.UserRole
		}
		if execCtx.UserEmail != "" {
			exec.UserEmail = &execCtx.UserEmail
		}

		// Encode input params
		if execCtx.Params != nil {
			paramsJSON, _ := json.Marshal(execCtx.Params)
			exec.InputParams = paramsJSON
		}

		// Set started time
		now := time.Now()
		exec.StartedAt = &now

		// Save execution record (unless logging is disabled)
		if !execCtx.DisableExecutionLogs {
			if err := e.storage.CreateExecution(ctx, exec); err != nil {
				log.Error().Err(err).Msg("Failed to create execution record")
			}
		}
	}

	// Log start (unless logging is disabled)
	if !execCtx.DisableExecutionLogs {
		e.appendLog(ctx, exec.ID, 1, "info", fmt.Sprintf("Starting RPC execution: %s/%s", execCtx.Procedure.Namespace, execCtx.Procedure.Name))
	}

	// Validate input parameters
	if err := e.validator.ValidateInput(execCtx.Params, execCtx.Procedure.InputSchema); err != nil {
		return e.failExecutionWithContext(ctx, exec, execCtx, start, fmt.Sprintf("Input validation failed: %s", err.Error()))
	}

	if !execCtx.DisableExecutionLogs {
		e.appendLog(ctx, exec.ID, 2, "info", "Input validation passed")
	}

	// Validate SQL
	validationResult := e.validator.ValidateSQL(
		execCtx.Procedure.SQLQuery,
		execCtx.Procedure.AllowedTables,
		execCtx.Procedure.AllowedSchemas,
	)

	if !validationResult.Valid {
		return e.failExecutionWithContext(ctx, exec, execCtx, start, fmt.Sprintf("SQL validation failed: %v", validationResult.Errors))
	}

	if !execCtx.DisableExecutionLogs {
		e.appendLog(ctx, exec.ID, 3, "info", fmt.Sprintf("SQL validation passed. Tables: %v, Operations: %v",
			validationResult.TablesAccessed, validationResult.OperationsUsed))
	}

	// Build SQL with parameter substitution
	sql, err := e.buildSQL(execCtx.Procedure.SQLQuery, execCtx.Params, execCtx)
	if err != nil {
		return e.failExecutionWithContext(ctx, exec, execCtx, start, fmt.Sprintf("Failed to build SQL: %s", err.Error()))
	}

	if !execCtx.DisableExecutionLogs {
		e.appendLog(ctx, exec.ID, 4, "info", "SQL prepared with parameters")
	}

	// Create a context with timeout
	timeout := time.Duration(execCtx.Procedure.MaxExecutionTimeSeconds) * time.Second
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	queryCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// Execute with RLS context
	result, rowCount, err := e.executeWithRLS(queryCtx, sql, execCtx)
	if err != nil {
		// Check for timeout
		if ctx.Err() == context.DeadlineExceeded {
			exec.Status = StatusTimeout
			return e.failExecutionWithContext(ctx, exec, execCtx, start, "Query execution timed out")
		}
		return e.failExecutionWithContext(ctx, exec, execCtx, start, fmt.Sprintf("Query execution failed: %s", err.Error()))
	}

	if !execCtx.DisableExecutionLogs {
		e.appendLog(ctx, exec.ID, 5, "info", fmt.Sprintf("Query executed successfully. Rows returned: %d", rowCount))
	}

	// Complete execution
	duration := int(time.Since(start).Milliseconds())
	completedAt := time.Now()

	exec.Status = StatusCompleted
	exec.Result = result
	exec.RowsReturned = &rowCount
	exec.DurationMs = &duration
	exec.CompletedAt = &completedAt

	// Always update execution record for async (or if logs are enabled)
	if execCtx.IsAsync || !execCtx.DisableExecutionLogs {
		if err := e.storage.UpdateExecution(ctx, exec); err != nil {
			log.Error().Err(err).Msg("Failed to update execution record")
		}
	}
	// Only append verbose logs if logging is enabled
	if !execCtx.DisableExecutionLogs {
		e.appendLog(ctx, exec.ID, 6, "info", fmt.Sprintf("Execution completed in %dms", duration))
	}

	// Record metrics
	if e.metrics != nil {
		e.metrics.RecordRPCExecution(execCtx.Procedure.Name, "success", time.Since(start))
	}

	return &ExecuteResult{
		ExecutionID:  exec.ID,
		Status:       StatusCompleted,
		Result:       result,
		RowsReturned: &rowCount,
		DurationMs:   &duration,
	}, nil
}

// ExecuteAsync executes an RPC procedure asynchronously
func (e *Executor) ExecuteAsync(ctx context.Context, execCtx *ExecuteContext) (*ExecuteResult, error) {
	execCtx.IsAsync = true

	// Create execution record with pending status
	exec := &Execution{
		ID:            uuid.New().String(),
		ProcedureID:   &execCtx.Procedure.ID,
		ProcedureName: execCtx.Procedure.Name,
		Namespace:     execCtx.Procedure.Namespace,
		Status:        StatusPending,
		IsAsync:       true,
		CreatedAt:     time.Now(),
	}

	// Set optional user fields (nil if empty, to store as NULL in database)
	if execCtx.UserID != "" {
		exec.UserID = &execCtx.UserID
	}
	if execCtx.UserRole != "" {
		exec.UserRole = &execCtx.UserRole
	}
	if execCtx.UserEmail != "" {
		exec.UserEmail = &execCtx.UserEmail
	}

	// Encode input params
	if execCtx.Params != nil {
		paramsJSON, _ := json.Marshal(execCtx.Params)
		exec.InputParams = paramsJSON
	}

	// For async executions, ALWAYS create the execution record so getStatus() works.
	// The DisableExecutionLogs flag only controls verbose log messages, not execution tracking.
	if err := e.storage.CreateExecution(ctx, exec); err != nil {
		return nil, fmt.Errorf("failed to create execution record: %w", err)
	}

	// Pass the execution ID so the background worker updates this record
	execCtx.ExecutionID = exec.ID

	// Start async execution in goroutine
	go func() {
		// Create new context for background execution
		bgCtx := context.Background()

		// Execute - will update the existing record instead of creating a new one
		_, _ = e.Execute(bgCtx, execCtx)
	}()

	return &ExecuteResult{
		ExecutionID: exec.ID,
		Status:      StatusPending,
	}, nil
}

// buildSQL builds the SQL query with parameter substitution
func (e *Executor) buildSQL(sqlTemplate string, params map[string]interface{}, execCtx *ExecuteContext) (string, error) {
	sql := sqlTemplate

	// Add caller context parameters
	callerParams := map[string]interface{}{
		"caller_id":    execCtx.UserID,
		"caller_role":  execCtx.UserRole,
		"caller_email": execCtx.UserEmail,
	}

	// Merge caller params with user params (user params take precedence)
	allParams := make(map[string]interface{})
	for k, v := range callerParams {
		allParams[k] = v
	}
	for k, v := range params {
		allParams[k] = v
	}

	// Replace $param_name with actual values
	// Use a regex to find all parameter placeholders
	paramPattern := regexp.MustCompile(`\$([a-zA-Z_][a-zA-Z0-9_]*)`)

	var missingParams []string
	sql = paramPattern.ReplaceAllStringFunc(sql, func(match string) string {
		paramName := strings.TrimPrefix(match, "$")
		value, exists := allParams[paramName]
		if !exists {
			missingParams = append(missingParams, paramName)
			return match
		}
		return e.formatValue(value)
	})

	if len(missingParams) > 0 {
		return "", fmt.Errorf("missing required parameters: %v", missingParams)
	}

	return sql, nil
}

// formatValue formats a Go value for use in SQL
func (e *Executor) formatValue(value interface{}) string {
	if value == nil {
		return "NULL"
	}

	switch v := value.(type) {
	case string:
		// Escape single quotes
		escaped := strings.ReplaceAll(v, "'", "''")
		return fmt.Sprintf("'%s'", escaped)
	case int, int32, int64, float32, float64:
		return fmt.Sprintf("%v", v)
	case bool:
		if v {
			return "TRUE"
		}
		return "FALSE"
	case json.Number:
		return v.String()
	case []float32:
		// Format as PostgreSQL vector literal: '[0.1, 0.2, ...]'::vector
		return formatVectorLiteral32(v)
	case []float64:
		// Format as PostgreSQL vector literal: '[0.1, 0.2, ...]'::vector
		return formatVectorLiteral64(v)
	case []interface{}:
		// Check if it's a numeric array (potential vector from JSON)
		if isNumericArray(v) {
			return formatVectorLiteralInterface(v)
		}
		// Format as PostgreSQL array
		var items []string
		for _, item := range v {
			items = append(items, e.formatValue(item))
		}
		return fmt.Sprintf("ARRAY[%s]", strings.Join(items, ", "))
	case map[string]interface{}:
		// Format as JSONB
		jsonBytes, _ := json.Marshal(v)
		escaped := strings.ReplaceAll(string(jsonBytes), "'", "''")
		return fmt.Sprintf("'%s'::jsonb", escaped)
	default:
		// Convert to string
		str := fmt.Sprintf("%v", v)
		escaped := strings.ReplaceAll(str, "'", "''")
		return fmt.Sprintf("'%s'", escaped)
	}
}

// isNumericArray checks if a []interface{} contains only numeric values
func isNumericArray(arr []interface{}) bool {
	if len(arr) == 0 {
		return false
	}
	for _, item := range arr {
		switch item.(type) {
		case float64, float32, int, int64, int32, json.Number:
			// Numeric type - continue
		default:
			return false
		}
	}
	return true
}

// formatVectorLiteral32 formats a []float32 as PostgreSQL vector literal
func formatVectorLiteral32(v []float32) string {
	parts := make([]string, len(v))
	for i, f := range v {
		parts[i] = fmt.Sprintf("%g", f)
	}
	return fmt.Sprintf("'[%s]'::vector", strings.Join(parts, ","))
}

// formatVectorLiteral64 formats a []float64 as PostgreSQL vector literal
func formatVectorLiteral64(v []float64) string {
	parts := make([]string, len(v))
	for i, f := range v {
		parts[i] = fmt.Sprintf("%g", f)
	}
	return fmt.Sprintf("'[%s]'::vector", strings.Join(parts, ","))
}

// formatVectorLiteralInterface formats a []interface{} (numeric) as PostgreSQL vector literal
func formatVectorLiteralInterface(v []interface{}) string {
	parts := make([]string, len(v))
	for i, item := range v {
		switch num := item.(type) {
		case float64:
			parts[i] = fmt.Sprintf("%g", num)
		case float32:
			parts[i] = fmt.Sprintf("%g", num)
		case int:
			parts[i] = fmt.Sprintf("%d", num)
		case int64:
			parts[i] = fmt.Sprintf("%d", num)
		case int32:
			parts[i] = fmt.Sprintf("%d", num)
		case json.Number:
			parts[i] = num.String()
		default:
			parts[i] = fmt.Sprintf("%v", num)
		}
	}
	return fmt.Sprintf("'[%s]'::vector", strings.Join(parts, ","))
}

// executeWithRLS executes the SQL query with RLS context set
func (e *Executor) executeWithRLS(ctx context.Context, sql string, execCtx *ExecuteContext) (json.RawMessage, int, error) {
	// Start transaction with RLS
	tx, err := e.db.Pool().Begin(ctx)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	// Set RLS context
	if err := middleware.SetRLSContext(ctx, tx, execCtx.UserID, execCtx.UserRole, execCtx.Claims); err != nil {
		return nil, 0, fmt.Errorf("failed to set RLS context: %w", err)
	}

	// Execute the query
	rows, err := tx.Query(ctx, sql)
	if err != nil {
		return nil, 0, fmt.Errorf("query execution failed: %w", err)
	}
	defer rows.Close()

	// Get column names
	fieldDescs := rows.FieldDescriptions()
	columns := make([]string, len(fieldDescs))
	for i, fd := range fieldDescs {
		columns[i] = string(fd.Name)
	}

	// Collect rows
	var resultRows []map[string]interface{}
	maxRows := 1000 // Default limit
	if e.config != nil && e.config.DefaultMaxRows > 0 {
		maxRows = e.config.DefaultMaxRows
	}

	rowCount := 0
	for rows.Next() {
		if rowCount >= maxRows {
			break
		}

		values, err := rows.Values()
		if err != nil {
			log.Warn().Err(err).Msg("Failed to scan row values")
			continue
		}

		row := make(map[string]interface{})
		for i, col := range columns {
			row[col] = convertValue(values[i])
		}
		resultRows = append(resultRows, row)
		rowCount++
	}

	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("error reading results: %w", err)
	}

	// Commit transaction
	if err := tx.Commit(ctx); err != nil {
		return nil, 0, fmt.Errorf("failed to commit: %w", err)
	}

	// Marshal result
	resultJSON, err := json.Marshal(resultRows)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to marshal result: %w", err)
	}

	return resultJSON, rowCount, nil
}

// failExecutionWithContext marks an execution as failed and returns the error result
// When execCtx is provided and DisableExecutionLogs is true, skips updating execution record and appending logs
func (e *Executor) failExecutionWithContext(ctx context.Context, exec *Execution, execCtx *ExecuteContext, start time.Time, errorMsg string) (*ExecuteResult, error) {
	duration := int(time.Since(start).Milliseconds())
	completedAt := time.Now()

	exec.Status = StatusFailed
	exec.ErrorMessage = &errorMsg
	exec.DurationMs = &duration
	exec.CompletedAt = &completedAt

	disableLogs := execCtx != nil && execCtx.DisableExecutionLogs
	isAsync := execCtx != nil && execCtx.IsAsync

	// Always update execution record for async (or if logs are enabled)
	if isAsync || !disableLogs {
		if err := e.storage.UpdateExecution(ctx, exec); err != nil {
			log.Error().Err(err).Msg("Failed to update execution record")
		}
	}
	// Only append verbose logs if logging is enabled
	if !disableLogs {
		e.appendLog(ctx, exec.ID, 99, "error", errorMsg)
	}

	// Record metrics
	if e.metrics != nil {
		e.metrics.RecordRPCExecution(exec.ProcedureName, "error", time.Since(start))
	}

	return &ExecuteResult{
		ExecutionID: exec.ID,
		Status:      StatusFailed,
		DurationMs:  &duration,
		Error:       &errorMsg,
	}, nil
}

// appendLog appends a log entry to the execution
// Note: Execution logs are now stored in the central logging schema (logging.entries)
func (e *Executor) appendLog(ctx context.Context, executionID string, lineNumber int, level, message string) {
	// Log to zerolog - central logging service will capture this via execution_id field
	log.Info().
		Str("execution_id", executionID).
		Str("execution_type", "rpc").
		Str("level", level).
		Int("line_number", lineNumber).
		Msg(message)
}

// convertValue converts database values to JSON-safe types
func convertValue(v interface{}) interface{} {
	if v == nil {
		return nil
	}

	switch val := v.(type) {
	case []byte:
		// Try to parse as JSON
		var jsonVal interface{}
		if err := json.Unmarshal(val, &jsonVal); err == nil {
			return jsonVal
		}
		return string(val)
	case time.Time:
		return val.Format(time.RFC3339)
	case pgx.Rows:
		return nil // Skip complex types
	default:
		return val
	}
}
