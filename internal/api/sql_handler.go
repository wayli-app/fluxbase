package api

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/fluxbase-eu/fluxbase/internal/auth"
	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog/log"
)

// SQLHandler handles SQL query execution for the admin SQL editor
type SQLHandler struct {
	db          *pgxpool.Pool
	authService *auth.Service
}

// NewSQLHandler creates a new SQL handler
func NewSQLHandler(db *pgxpool.Pool, authService *auth.Service) *SQLHandler {
	return &SQLHandler{
		db:          db,
		authService: authService,
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

	// Split query into statements (basic split by semicolon)
	statements := splitSQLStatements(req.Query)

	// Check for impersonation token in custom header
	// This allows the admin to stay authenticated while executing queries as another user
	impersonationToken := c.Get("X-Impersonation-Token")

	// Log whether impersonation header was received (INFO level for visibility)
	log.Info().
		Str("user_id", userID).
		Bool("has_impersonation_token", impersonationToken != "").
		Int("token_length", len(impersonationToken)).
		Msg("SQL query execution - checking impersonation")

	if impersonationToken != "" {
		// Trim any whitespace
		impersonationToken = strings.TrimSpace(impersonationToken)

		// Debug: Log token preview
		tokenPreview := impersonationToken
		if len(tokenPreview) > 30 {
			tokenPreview = tokenPreview[:30] + "..."
		}
		log.Debug().
			Str("token_preview", tokenPreview).
			Bool("starts_with_ey", strings.HasPrefix(impersonationToken, "ey")).
			Msg("Validating SQL impersonation token")

		// Validate the impersonation token
		impersonationClaims, err := h.authService.ValidateToken(impersonationToken)
		if err != nil {
			log.Warn().
				Err(err).
				Str("user_id", userID).
				Str("token_preview", tokenPreview).
				Msg("Invalid impersonation token in SQL query")
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Invalid impersonation token",
			})
		}

		log.Info().
			Str("audit_user_id", userID).
			Str("audit_user_email", userEmail).
			Str("impersonated_user_id", impersonationClaims.UserID).
			Str("impersonated_role", impersonationClaims.Role).
			Str("query_preview", truncateString(req.Query, 100)).
			Msg("SQL query execution with impersonation token")

		// Use impersonation claims for RLS context
		return h.executeWithRLSContext(c, req.Query, statements, impersonationClaims, userID)
	}

	// No impersonation - check if JWT claims indicate a known database role
	claims, hasClaims := c.Locals("jwt_claims").(*auth.TokenClaims)

	// Log query execution attempt
	log.Info().
		Str("user_id", userID).
		Str("user_email", userEmail).
		Bool("has_claims", hasClaims).
		Str("query_preview", truncateString(req.Query, 100)).
		Msg("SQL query execution attempt")

	// Only use RLS context for known database roles (direct token, not impersonation)
	// Dashboard admins (role like "dashboard_admin") get service_role access
	if hasClaims && claims != nil && isKnownDatabaseRole(claims.Role) {
		return h.executeWithRLSContext(c, req.Query, statements, claims, userID)
	}

	// Admin mode: execute with service_role for full access
	return h.executeAsServiceRole(c, req.Query, statements, userID)
}

// executeWithRLSContext executes SQL statements with Row Level Security context
// This is used when impersonating a user to test RLS policies
func (h *SQLHandler) executeWithRLSContext(c *fiber.Ctx, fullQuery string, statements []string, claims *auth.TokenClaims, auditUserID string) error {
	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), queryTimeout)
	defer cancel()

	// Acquire a dedicated connection for setting session variables
	conn, err := h.db.Acquire(ctx)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": fmt.Sprintf("Failed to acquire database connection: %v", err),
		})
	}
	defer conn.Release()

	// Build JWT claims JSON for request.jwt.claims setting
	// This mirrors what Supabase/PostgREST does
	claimsMap := map[string]any{
		"sub":          claims.UserID,
		"role":         claims.Role,
		"email":        claims.Email,
		"is_anonymous": claims.IsAnonymous,
	}
	if claims.SessionID != "" {
		claimsMap["session_id"] = claims.SessionID
	}
	if claims.UserMetadata != nil {
		claimsMap["user_metadata"] = claims.UserMetadata
	}
	if claims.AppMetadata != nil {
		claimsMap["app_metadata"] = claims.AppMetadata
	}

	claimsJSON, err := json.Marshal(claimsMap)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": fmt.Sprintf("Failed to serialize JWT claims: %v", err),
		})
	}

	// Determine the database role to use
	// Map application roles to PostgreSQL database roles
	// Valid database roles: authenticated, anon, service_role
	// Any other role (like "admin") maps to "authenticated" since they're authenticated users
	dbRole := claims.Role
	if !isKnownDatabaseRole(dbRole) {
		log.Debug().
			Str("app_role", claims.Role).
			Str("db_role", "authenticated").
			Msg("Mapping application role to database role")
		dbRole = "authenticated"
	}
	if dbRole == "" {
		dbRole = "authenticated"
	}

	// Log the exact claims being set (INFO level for visibility)
	log.Info().
		Str("claims_json", string(claimsJSON)).
		Str("db_role", dbRole).
		Str("user_id", claims.UserID).
		Msg("Setting RLS context for SQL execution")

	log.Info().
		Str("audit_user_id", auditUserID).
		Str("impersonated_user_id", claims.UserID).
		Str("impersonated_role", dbRole).
		Str("query_preview", truncateString(fullQuery, 100)).
		Msg("SQL query execution with RLS context")

	// Execute all statements within a transaction to maintain RLS context
	results := make([]SQLResult, 0, len(statements))

	tx, err := conn.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": fmt.Sprintf("Failed to begin transaction: %v", err),
		})
	}
	defer tx.Rollback(ctx) // Will be no-op if committed

	// Set RLS session variables
	_, err = tx.Exec(ctx, "SELECT set_config('request.jwt.claims', $1, true)", string(claimsJSON))
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": fmt.Sprintf("Failed to set JWT claims: %v", err),
		})
	}

	// Set the role (use SET LOCAL to limit to current transaction)
	// Note: We quote the role identifier to prevent SQL injection
	_, err = tx.Exec(ctx, fmt.Sprintf("SET LOCAL ROLE %q", dbRole))
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": fmt.Sprintf("Failed to set role '%s': %v", dbRole, err),
		})
	}

	// Execute each statement
	for _, stmt := range statements {
		stmt = strings.TrimSpace(stmt)
		if stmt == "" {
			continue
		}

		result := h.executeStatementInTx(ctx, tx, stmt)
		results = append(results, result)

		// Log each query execution
		if result.Error != nil {
			log.Warn().
				Str("audit_user_id", auditUserID).
				Str("impersonated_user_id", claims.UserID).
				Str("statement", truncateString(stmt, 100)).
				Str("error", *result.Error).
				Msg("SQL query execution failed (RLS context)")
		} else {
			log.Info().
				Str("audit_user_id", auditUserID).
				Str("impersonated_user_id", claims.UserID).
				Str("statement", truncateString(stmt, 100)).
				Int("row_count", result.RowCount).
				Float64("execution_time_ms", result.ExecutionTimeMS).
				Msg("SQL query executed successfully (RLS context)")
		}
	}

	// Commit the transaction (for read-only queries, this is just cleanup)
	if err := tx.Commit(ctx); err != nil {
		log.Warn().Err(err).Msg("Failed to commit RLS transaction (may be read-only)")
	}

	return c.JSON(ExecuteSQLResponse{
		Results: results,
	})
}

// isKnownDatabaseRole checks if a role is a PostgreSQL database role
// that can be used with SET ROLE for RLS policies
func isKnownDatabaseRole(role string) bool {
	switch role {
	case "authenticated", "anon", "service_role":
		return true
	default:
		return false
	}
}

// executeAsServiceRole executes SQL with service_role (full admin access)
// This is used for dashboard admins who are not impersonating anyone
func (h *SQLHandler) executeAsServiceRole(c *fiber.Ctx, fullQuery string, statements []string, auditUserID string) error {
	ctx, cancel := context.WithTimeout(context.Background(), queryTimeout)
	defer cancel()

	conn, err := h.db.Acquire(ctx)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": fmt.Sprintf("Failed to acquire database connection: %v", err),
		})
	}
	defer conn.Release()

	log.Info().
		Str("audit_user_id", auditUserID).
		Str("role", "service_role").
		Str("query_preview", truncateString(fullQuery, 100)).
		Msg("SQL query execution with service_role")

	results := make([]SQLResult, 0, len(statements))

	tx, err := conn.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": fmt.Sprintf("Failed to begin transaction: %v", err),
		})
	}
	defer tx.Rollback(ctx)

	// Set service_role for full admin access
	_, err = tx.Exec(ctx, "SET LOCAL ROLE service_role")
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": fmt.Sprintf("Failed to set service_role: %v", err),
		})
	}

	// Execute each statement
	for _, stmt := range statements {
		stmt = strings.TrimSpace(stmt)
		if stmt == "" {
			continue
		}

		result := h.executeStatementInTx(ctx, tx, stmt)
		results = append(results, result)

		// Log each query execution
		if result.Error != nil {
			log.Warn().
				Str("audit_user_id", auditUserID).
				Str("statement", truncateString(stmt, 100)).
				Str("error", *result.Error).
				Msg("SQL query execution failed (service_role)")
		} else {
			log.Info().
				Str("audit_user_id", auditUserID).
				Str("statement", truncateString(stmt, 100)).
				Int("row_count", result.RowCount).
				Float64("execution_time_ms", result.ExecutionTimeMS).
				Msg("SQL query executed successfully (service_role)")
		}
	}

	if err := tx.Commit(ctx); err != nil {
		log.Warn().Err(err).Msg("Failed to commit service_role transaction")
	}

	return c.JSON(ExecuteSQLResponse{Results: results})
}

// executeStatementInTx executes a single SQL statement within a transaction
func (h *SQLHandler) executeStatementInTx(ctx context.Context, tx pgx.Tx, statement string) SQLResult {
	startTime := time.Now()

	result := SQLResult{
		Statement: statement,
	}

	// Execute query
	rows, err := tx.Query(ctx, statement)
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
		for rows.Next() {
			// Should not happen for non-SELECT, but drain just in case
		}
	}

	// Check for errors during iteration
	if err := rows.Err(); err != nil {
		errorMsg := err.Error()
		result.Error = &errorMsg
	}

	// Get command tag for affected rows
	commandTag := rows.CommandTag()
	if len(columns) == 0 {
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
