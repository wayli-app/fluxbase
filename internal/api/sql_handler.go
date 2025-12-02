package api

import (
	"context"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog/log"
)

// SQLHandler handles SQL query execution for the admin SQL editor
type SQLHandler struct {
	db *pgxpool.Pool
}

// NewSQLHandler creates a new SQL handler
func NewSQLHandler(db *pgxpool.Pool) *SQLHandler {
	return &SQLHandler{
		db: db,
	}
}

// ExecuteSQLRequest represents a SQL execution request
type ExecuteSQLRequest struct {
	Query string `json:"query"`
}

// SQLResult represents the result of a single SQL statement
type SQLResult struct {
	Columns         []string         `json:"columns,omitempty"`
	Rows            []map[string]any `json:"rows,omitempty"`
	RowCount        int              `json:"row_count"`
	AffectedRows    int64            `json:"affected_rows,omitempty"`
	ExecutionTimeMS float64          `json:"execution_time_ms"`
	Error           *string          `json:"error,omitempty"`
	Statement       string           `json:"statement"`
}

// ExecuteSQLResponse represents the response for SQL execution
type ExecuteSQLResponse struct {
	Results []SQLResult `json:"results"`
}

const (
	maxRowsPerQuery = 1000
	queryTimeout    = 30 * time.Second
)

// ExecuteSQL executes SQL queries provided by the user
// @Summary Execute SQL queries
// @Description Executes one or more SQL statements and returns results. Only accessible by dashboard admins.
// @Tags SQL
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param query body ExecuteSQLRequest true "SQL query to execute"
// @Success 200 {object} ExecuteSQLResponse
// @Failure 400 {object} fiber.Map
// @Failure 401 {object} fiber.Map
// @Failure 500 {object} fiber.Map
// @Router /api/v1/sql/execute [post]
func (h *SQLHandler) ExecuteSQL(c *fiber.Ctx) error {
	// Parse request
	var req ExecuteSQLRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	// Validate query
	if strings.TrimSpace(req.Query) == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Query cannot be empty",
		})
	}

	// Get user information for audit logging
	userID, _ := GetUserID(c)
	userEmail, _ := GetUserEmail(c)

	// Log query execution attempt
	log.Info().
		Str("user_id", userID).
		Str("user_email", userEmail).
		Str("query_preview", truncateString(req.Query, 100)).
		Msg("SQL query execution attempt")

	// Split query into statements (basic split by semicolon)
	statements := splitSQLStatements(req.Query)
	results := make([]SQLResult, 0, len(statements))

	// Execute each statement
	for _, stmt := range statements {
		stmt = strings.TrimSpace(stmt)
		if stmt == "" {
			continue
		}

		result := h.executeStatement(c.Context(), stmt)
		results = append(results, result)

		// Log each query execution
		if result.Error != nil {
			log.Warn().
				Str("user_id", userID).
				Str("statement", truncateString(stmt, 100)).
				Str("error", *result.Error).
				Msg("SQL query execution failed")
		} else {
			log.Info().
				Str("user_id", userID).
				Str("statement", truncateString(stmt, 100)).
				Int("row_count", result.RowCount).
				Float64("execution_time_ms", result.ExecutionTimeMS).
				Msg("SQL query executed successfully")
		}
	}

	return c.JSON(ExecuteSQLResponse{
		Results: results,
	})
}

// executeStatement executes a single SQL statement and returns the result
func (h *SQLHandler) executeStatement(ctx context.Context, statement string) SQLResult {
	startTime := time.Now()

	// Create context with timeout
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	result := SQLResult{
		Statement: statement,
	}

	// Execute query
	rows, err := h.db.Query(ctx, statement)
	if err != nil {
		errorMsg := err.Error()
		result.Error = &errorMsg
		result.ExecutionTimeMS = float64(time.Since(startTime).Milliseconds())
		return result
	}
	defer rows.Close()

	// Get column descriptions
	fieldDescriptions := rows.FieldDescriptions()
	columns := make([]string, len(fieldDescriptions))
	for i, fd := range fieldDescriptions {
		columns[i] = string(fd.Name)
	}
	result.Columns = columns

	// Check if this is a SELECT query (has columns)
	if len(columns) > 0 {
		// Read rows
		resultRows := make([]map[string]any, 0)
		rowCount := 0

		for rows.Next() {
			if rowCount >= maxRowsPerQuery {
				// Drain remaining rows but don't include them
				for rows.Next() {
					rowCount++
				}
				errorMsg := fmt.Sprintf("Result limited to %d rows (query returned %d rows)", maxRowsPerQuery, rowCount)
				result.Error = &errorMsg
				break
			}

			values, err := rows.Values()
			if err != nil {
				errorMsg := fmt.Sprintf("Error reading row: %v", err)
				result.Error = &errorMsg
				break
			}

			row := make(map[string]any)
			for i, col := range columns {
				row[col] = convertValue(values[i])
			}
			resultRows = append(resultRows, row)
			rowCount++
		}

		result.Rows = resultRows
		result.RowCount = len(resultRows)
	} else {
		// For non-SELECT queries (INSERT, UPDATE, DELETE, etc.)
		// We need to consume the rows (even though there are none) to get CommandTag
		for rows.Next() {
			// Should not happen for non-SELECT, but drain just in case
		}
	}

	// Check for errors during iteration
	if err := rows.Err(); err != nil {
		errorMsg := err.Error()
		result.Error = &errorMsg
	}

	// Get command tag for affected rows (works for all query types)
	commandTag := rows.CommandTag()
	if len(columns) == 0 {
		// For non-SELECT queries, set affected rows
		result.AffectedRows = commandTag.RowsAffected()
		result.RowCount = int(commandTag.RowsAffected())
	}

	result.ExecutionTimeMS = float64(time.Since(startTime).Milliseconds())
	return result
}

// splitSQLStatements splits a SQL query string into individual statements
// This is a simple implementation that splits by semicolons
func splitSQLStatements(query string) []string {
	// Simple split by semicolon
	// Note: This doesn't handle semicolons inside strings or comments
	// For production use, consider using a proper SQL parser
	statements := strings.Split(query, ";")
	result := make([]string, 0, len(statements))

	for _, stmt := range statements {
		stmt = strings.TrimSpace(stmt)
		if stmt != "" {
			result = append(result, stmt)
		}
	}

	return result
}

// truncateString truncates a string to a maximum length
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

// convertValue converts database values to JSON-friendly formats
// Specifically handles UUID byte arrays which pgx returns as [16]byte
func convertValue(v any) any {
	if v == nil {
		return nil
	}

	// Handle UUID: pgx returns UUIDs as [16]byte arrays
	if b, ok := v.([16]byte); ok {
		return formatUUID(b[:])
	}

	// Handle byte slices that might be UUIDs (some drivers return []byte)
	if b, ok := v.([]byte); ok && len(b) == 16 {
		// Check if it looks like a UUID (not printable ASCII)
		isPrintable := true
		for _, c := range b {
			if c < 32 || c > 126 {
				isPrintable = false
				break
			}
		}
		if !isPrintable {
			return formatUUID(b)
		}
	}

	return v
}

// formatUUID formats a 16-byte slice as a UUID string
func formatUUID(b []byte) string {
	return fmt.Sprintf("%s-%s-%s-%s-%s",
		hex.EncodeToString(b[0:4]),
		hex.EncodeToString(b[4:6]),
		hex.EncodeToString(b[6:8]),
		hex.EncodeToString(b[8:10]),
		hex.EncodeToString(b[10:16]),
	)
}
