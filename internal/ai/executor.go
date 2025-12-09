package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/fluxbase-eu/fluxbase/internal/auth"
	"github.com/fluxbase-eu/fluxbase/internal/database"
	"github.com/fluxbase-eu/fluxbase/internal/middleware"
	"github.com/fluxbase-eu/fluxbase/internal/observability"
	"github.com/jackc/pgx/v5"
	"github.com/rs/zerolog/log"
)

// Executor handles SQL query execution with validation and RLS
type Executor struct {
	db      *database.Connection
	metrics *observability.Metrics
	maxRows int
	timeout time.Duration
}

// NewExecutor creates a new SQL executor
func NewExecutor(db *database.Connection, metrics *observability.Metrics, maxRows int, timeout time.Duration) *Executor {
	return &Executor{
		db:      db,
		metrics: metrics,
		maxRows: maxRows,
		timeout: timeout,
	}
}

// ExecuteRequest represents a request to execute SQL
type ExecuteRequest struct {
	ChatbotName       string
	ChatbotID         string
	ConversationID    string
	UserID            string
	Role              string
	Claims            *auth.TokenClaims
	SQL               string
	Description       string
	AllowedSchemas    []string
	AllowedTables     []string
	AllowedOperations []string
}

// ExecuteResult represents the result of SQL execution
type ExecuteResult struct {
	Success          bool              `json:"success"`
	RowCount         int               `json:"row_count"`
	Columns          []string          `json:"columns,omitempty"`
	Rows             []map[string]any  `json:"rows,omitempty"`
	Summary          string            `json:"summary"`
	Error            string            `json:"error,omitempty"`
	DurationMs       int64             `json:"duration_ms"`
	TablesAccessed   []string          `json:"tables_accessed,omitempty"`
	OperationsUsed   []string          `json:"operations_used,omitempty"`
	ValidationResult *ValidationResult `json:"-"`
}

// Execute validates and executes a SQL query with RLS context
func (e *Executor) Execute(ctx context.Context, req *ExecuteRequest) (*ExecuteResult, error) {
	start := time.Now()
	result := &ExecuteResult{
		Success: false,
	}

	// Create a context with timeout
	queryCtx, cancel := context.WithTimeout(ctx, e.timeout)
	defer cancel()

	// Create validator with chatbot's allowed resources
	validator := NewSQLValidator(req.AllowedSchemas, req.AllowedTables, req.AllowedOperations)

	// Validate the SQL
	validationResult, normalizedSQL, err := validator.ValidateAndNormalize(req.SQL)
	result.ValidationResult = validationResult
	result.TablesAccessed = validationResult.TablesAccessed
	result.OperationsUsed = validationResult.OperationsUsed

	if err != nil {
		result.Error = fmt.Sprintf("Query validation failed: %s", err.Error())
		result.Summary = "Query was rejected due to security validation"

		// Record metrics
		if e.metrics != nil {
			e.metrics.RecordAISQLQuery(req.ChatbotName, "rejected", time.Since(start))
		}

		log.Warn().
			Str("chatbot", req.ChatbotName).
			Str("user_id", req.UserID).
			Str("sql", req.SQL).
			Strs("errors", validationResult.Errors).
			Msg("SQL validation failed")

		return result, nil
	}

	// Execute with RLS context
	var rows pgx.Rows
	var queryErr error

	// Start transaction with RLS
	tx, err := e.db.Pool().Begin(queryCtx)
	if err != nil {
		result.Error = fmt.Sprintf("Failed to begin transaction: %s", err.Error())
		result.Summary = "Database error"
		return result, nil
	}
	defer func() { _ = tx.Rollback(queryCtx) }()

	// Set RLS context
	if err := middleware.SetRLSContext(queryCtx, tx, req.UserID, req.Role, req.Claims); err != nil {
		result.Error = fmt.Sprintf("Failed to set RLS context: %s", err.Error())
		result.Summary = "Security context error"
		return result, nil
	}

	// Set search_path to include allowed schemas so unqualified table names resolve correctly
	if len(req.AllowedSchemas) > 0 {
		quotedSchemas := make([]string, len(req.AllowedSchemas))
		for i, schema := range req.AllowedSchemas {
			quotedSchemas[i] = pgx.Identifier{schema}.Sanitize()
		}
		searchPathSQL := fmt.Sprintf("SET LOCAL search_path TO %s", strings.Join(quotedSchemas, ", "))
		log.Debug().
			Strs("allowed_schemas", req.AllowedSchemas).
			Str("search_path_sql", searchPathSQL).
			Msg("Setting search_path for AI query")
		if _, err := tx.Exec(queryCtx, searchPathSQL); err != nil {
			result.Error = fmt.Sprintf("Failed to set search_path: %s", err.Error())
			result.Summary = "Schema configuration error"
			return result, nil
		}
	} else {
		log.Debug().
			Str("chatbot", req.ChatbotName).
			Msg("No allowed_schemas configured, using default search_path")
	}

	// Execute the query
	rows, queryErr = tx.Query(queryCtx, normalizedSQL)
	if queryErr != nil {
		result.Error = fmt.Sprintf("Query execution failed: %s", queryErr.Error())
		result.Summary = "Query failed to execute"
		result.DurationMs = time.Since(start).Milliseconds()

		// Record metrics
		if e.metrics != nil {
			e.metrics.RecordAISQLQuery(req.ChatbotName, "error", time.Since(start))
		}

		log.Error().
			Err(queryErr).
			Str("chatbot", req.ChatbotName).
			Str("user_id", req.UserID).
			Str("sql", normalizedSQL).
			Msg("SQL execution failed")

		return result, nil
	}
	defer rows.Close()

	// Get column names
	fieldDescs := rows.FieldDescriptions()
	result.Columns = make([]string, len(fieldDescs))
	for i, fd := range fieldDescs {
		result.Columns[i] = string(fd.Name)
	}

	// Collect rows
	result.Rows = []map[string]any{}
	for rows.Next() {
		if result.RowCount >= e.maxRows {
			// Limit reached
			break
		}

		values, err := rows.Values()
		if err != nil {
			log.Warn().Err(err).Msg("Failed to scan row values")
			continue
		}

		row := make(map[string]any)
		for i, col := range result.Columns {
			row[col] = convertValue(values[i])
		}
		result.Rows = append(result.Rows, row)
		result.RowCount++
	}

	if err := rows.Err(); err != nil {
		result.Error = fmt.Sprintf("Error reading results: %s", err.Error())
		result.Summary = "Error reading query results"
		return result, nil
	}

	// Commit transaction
	if err := tx.Commit(queryCtx); err != nil {
		result.Error = fmt.Sprintf("Failed to commit: %s", err.Error())
		result.Summary = "Transaction error"
		return result, nil
	}

	// Build summary for LLM
	result.Success = true
	result.DurationMs = time.Since(start).Milliseconds()
	result.Summary = e.buildSummary(req, result)

	// Record metrics
	if e.metrics != nil {
		e.metrics.RecordAISQLQuery(req.ChatbotName, "executed", time.Since(start))
	}

	log.Debug().
		Str("chatbot", req.ChatbotName).
		Str("user_id", req.UserID).
		Int("row_count", result.RowCount).
		Int64("duration_ms", result.DurationMs).
		Strs("tables", result.TablesAccessed).
		Msg("SQL executed successfully")

	return result, nil
}

// buildSummary creates a summary for the LLM (not the raw data)
func (e *Executor) buildSummary(req *ExecuteRequest, result *ExecuteResult) string {
	if result.RowCount == 0 {
		return fmt.Sprintf("Query returned no results. Tables accessed: %v",
			result.TablesAccessed)
	}

	summary := fmt.Sprintf("Query returned %d row(s)", result.RowCount)
	if result.RowCount >= e.maxRows {
		summary += fmt.Sprintf(" (limited to %d)", e.maxRows)
	}
	summary += fmt.Sprintf(". Tables accessed: %v", result.TablesAccessed)

	// Add sample of first few values for context
	if len(result.Rows) > 0 && len(result.Columns) > 0 {
		// Show first column sample
		firstCol := result.Columns[0]
		var samples []string
		for i := 0; i < min(3, len(result.Rows)); i++ {
			if v, ok := result.Rows[i][firstCol]; ok {
				samples = append(samples, fmt.Sprintf("%v", v))
			}
		}
		if len(samples) > 0 {
			summary += fmt.Sprintf(". Sample %s values: %v", firstCol, samples)
		}
	}

	return summary
}

// convertValue converts database values to JSON-safe types
func convertValue(v any) any {
	if v == nil {
		return nil
	}

	switch val := v.(type) {
	case []byte:
		// Try to parse as JSON
		var jsonVal any
		if err := json.Unmarshal(val, &jsonVal); err == nil {
			return jsonVal
		}
		return string(val)
	case time.Time:
		return val.Format(time.RFC3339)
	default:
		return val
	}
}

// min returns the minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
