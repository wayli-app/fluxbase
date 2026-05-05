package api

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	pg_query "github.com/pganalyze/pg_query_go/v6"
	"github.com/rs/zerolog/log"

	"github.com/nimbleflux/fluxbase/internal/auth"
	"github.com/nimbleflux/fluxbase/internal/database"
	"github.com/nimbleflux/fluxbase/internal/middleware"
	"github.com/nimbleflux/fluxbase/internal/util"
)

type SQLHandler struct {
	db          *database.Connection
	authService *auth.Service
}

func NewSQLHandler(db *database.Connection, authService *auth.Service) *SQLHandler {
	return &SQLHandler{
		db:          db,
		authService: authService,
	}
}

type ExecuteSQLRequest struct {
	Query string `json:"query"`
}

type SQLResult struct {
	Columns         []string         `json:"columns,omitempty"`
	Rows            []map[string]any `json:"rows,omitempty"`
	RowCount        int              `json:"row_count"`
	AffectedRows    int64            `json:"affected_rows,omitempty"`
	ExecutionTimeMS float64          `json:"execution_time_ms"`
	Error           *string          `json:"error,omitempty"`
	Statement       string           `json:"statement"`
}

type ExecuteSQLResponse struct {
	Results []SQLResult `json:"results"`
}

const (
	maxRowsPerQuery = 1000
	queryTimeout    = 30 * time.Second
)

func (h *SQLHandler) ExecuteSQL(c fiber.Ctx) error {
	var req ExecuteSQLRequest
	if err := ParseBody(c, &req); err != nil {
		return err
	}

	if strings.TrimSpace(req.Query) == "" {
		return SendBadRequest(c, "Query cannot be empty", ErrCodeInvalidInput)
	}

	userID := middleware.GetUserID(c)
	userEmail, _ := GetUserEmail(c)

	statements := splitSQLStatements(req.Query)

	tenantID := middleware.GetTenantIDFromContext(c)
	tenantRole := middleware.GetTenantRoleFromContext(c)
	isInstanceAdmin := middleware.IsInstanceAdminFromContext(c)
	tenantSource := middleware.GetTenantSourceFromContext(c)

	actingAsTenantAdmin := tenantID != "" && (tenantSource == "header" || tenantSource == "jwt")

	impersonationToken := c.Get("X-Impersonation-Token")

	log.Info().
		Str("user_id", userID).
		Str("user_email", userEmail).
		Bool("is_instance_admin", isInstanceAdmin).
		Bool("acting_as_tenant_admin", actingAsTenantAdmin).
		Str("tenant_id", tenantID).
		Str("tenant_role", tenantRole).
		Str("tenant_source", tenantSource).
		Bool("has_impersonation_token", impersonationToken != "").
		Str("query_preview", util.TruncateString(req.Query, 100)).
		Msg("SQL query execution attempt")

	if impersonationToken != "" {
		impersonationToken = strings.TrimSpace(impersonationToken)

		impersonationClaims, err := h.authService.ValidateToken(impersonationToken)
		if err != nil {
			log.Warn().
				Err(err).
				Str("user_id", userID).
				Msg("Invalid impersonation token in SQL query")
			return SendBadRequest(c, "Invalid impersonation token", ErrCodeInvalidInput)
		}

		log.Info().
			Str("audit_user_id", userID).
			Str("impersonated_user_id", impersonationClaims.UserID).
			Str("impersonated_role", impersonationClaims.Role).
			Msg("SQL query execution with impersonation token")

		pool := h.getPoolForQuery(c, req.Query)
		return h.executeWithRLSContext(c, pool, statements, impersonationClaims, tenantID, userID)
	}

	if isInstanceAdmin && !actingAsTenantAdmin {
		pool := h.getPoolForQuery(c, req.Query)
		return h.executeAsInstanceAdmin(c, pool, statements, userID)
	}

	pool := h.getPoolForQuery(c, req.Query)

	var claims *auth.TokenClaims
	for _, key := range []string{"claims", "jwt_claims"} {
		if c.Locals(key) != nil {
			if jwtClaims, ok := c.Locals(key).(*auth.TokenClaims); ok {
				claims = jwtClaims
				break
			}
		}
	}

	if claims == nil {
		claims = &auth.TokenClaims{
			UserID: userID,
			Role:   "authenticated",
		}
		if tenantRole != "" {
			claims.Role = tenantRole
		}
	}

	return h.executeWithTenantRLS(c, pool, statements, claims, tenantID, userID, isInstanceAdmin)
}

func (h *SQLHandler) getPoolForQuery(c fiber.Ctx, query string) *pgxpool.Pool {
	if pool := middleware.GetBranchPool(c); pool != nil {
		log.Debug().Msg("Using branch pool for SQL execution")
		return pool
	}

	if tenantPool := middleware.GetTenantPool(c); tenantPool != nil {
		log.Debug().Msg("Using tenant pool for SQL execution")
		return tenantPool
	}

	log.Debug().Msg("Using main pool for SQL execution")
	return h.db.Pool()
}

func (h *SQLHandler) executeWithRLSContext(c fiber.Ctx, pool *pgxpool.Pool, statements []string, claims *auth.TokenClaims, tenantID string, auditUserID string) error {
	ctx, cancel := context.WithTimeout(context.Background(), queryTimeout)
	defer cancel()

	conn, err := pool.Acquire(ctx)
	if err != nil {
		return SendInternalError(c, "Failed to acquire database connection")
	}
	defer conn.Release()

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
	if tenantID != "" {
		claimsMap["tenant_id"] = tenantID
	}

	claimsJSON, err := json.Marshal(claimsMap)
	if err != nil {
		return SendInternalError(c, "Failed to serialize JWT claims")
	}

	dbRole := "authenticated"
	if claims.Role == "anon" {
		dbRole = "anon"
	}

	log.Info().
		Str("audit_user_id", auditUserID).
		Str("impersonated_user_id", claims.UserID).
		Str("impersonated_role", dbRole).
		Str("tenant_id", tenantID).
		Msg("SQL query execution with RLS context (impersonation)")

	results := make([]SQLResult, 0, len(statements))

	tx, err := conn.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return SendInternalError(c, "Failed to begin transaction")
	}
	defer func() { _ = tx.Rollback(ctx) }()

	_, err = tx.Exec(ctx, "SELECT set_config('request.jwt.claims', $1, true)", string(claimsJSON))
	if err != nil {
		return SendInternalError(c, "Failed to set JWT claims")
	}

	if tenantID != "" {
		_, err = tx.Exec(ctx, "SELECT set_config('app.current_tenant_id', $1, true)", tenantID)
		if err != nil {
			return SendInternalError(c, "Failed to set tenant context")
		}
	}

	_, err = tx.Exec(ctx, fmt.Sprintf("SET LOCAL ROLE %q", dbRole))
	if err != nil {
		return SendInternalError(c, "Failed to set database role")
	}

	for _, stmt := range statements {
		stmt = strings.TrimSpace(stmt)
		if stmt == "" {
			continue
		}

		result := h.executeStatementInTx(ctx, tx, stmt)
		results = append(results, result)

		if result.Error != nil {
			log.Warn().
				Str("audit_user_id", auditUserID).
				Str("statement", util.TruncateString(stmt, 100)).
				Str("error", *result.Error).
				Msg("SQL query execution failed (impersonation RLS)")
		} else {
			log.Info().
				Str("audit_user_id", auditUserID).
				Str("statement", util.TruncateString(stmt, 100)).
				Int("row_count", result.RowCount).
				Msg("SQL query executed successfully (impersonation RLS)")
		}
	}

	if err := tx.Commit(ctx); err != nil {
		log.Warn().Err(err).Msg("Failed to commit RLS transaction")
	}

	return c.JSON(ExecuteSQLResponse{
		Results: results,
	})
}

func (h *SQLHandler) executeAsInstanceAdmin(c fiber.Ctx, pool *pgxpool.Pool, statements []string, auditUserID string) error {
	ctx, cancel := context.WithTimeout(context.Background(), queryTimeout)
	defer cancel()

	conn, err := pool.Acquire(ctx)
	if err != nil {
		return SendInternalError(c, "Failed to acquire database connection")
	}
	defer conn.Release()

	log.Info().
		Str("audit_user_id", auditUserID).
		Str("role", "service_role").
		Msg("SQL query execution as instance admin (full access)")

	results := make([]SQLResult, 0, len(statements))

	tx, err := conn.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return SendInternalError(c, "Failed to begin transaction")
	}
	defer func() { _ = tx.Rollback(ctx) }()

	_, err = tx.Exec(ctx, "SET LOCAL ROLE service_role")
	if err != nil {
		return SendInternalError(c, "Failed to set service_role")
	}

	for _, stmt := range statements {
		stmt = strings.TrimSpace(stmt)
		if stmt == "" {
			continue
		}

		result := h.executeStatementInTx(ctx, tx, stmt)
		results = append(results, result)

		if result.Error != nil {
			log.Warn().
				Str("audit_user_id", auditUserID).
				Str("statement", util.TruncateString(stmt, 100)).
				Str("error", *result.Error).
				Msg("SQL query execution failed (instance admin)")
		} else {
			log.Info().
				Str("audit_user_id", auditUserID).
				Str("statement", util.TruncateString(stmt, 100)).
				Int("row_count", result.RowCount).
				Msg("SQL query executed successfully (instance admin)")
		}
	}

	if err := tx.Commit(ctx); err != nil {
		log.Warn().Err(err).Msg("Failed to commit instance admin transaction")
	}

	return c.JSON(ExecuteSQLResponse{Results: results})
}

func (h *SQLHandler) executeWithTenantRLS(c fiber.Ctx, pool *pgxpool.Pool, statements []string, claims *auth.TokenClaims, tenantID string, auditUserID string, isInstanceAdmin bool) error {
	ctx, cancel := context.WithTimeout(context.Background(), queryTimeout)
	defer cancel()

	conn, err := pool.Acquire(ctx)
	if err != nil {
		return SendInternalError(c, "Failed to acquire database connection")
	}
	defer conn.Release()

	claimsMap := map[string]any{
		"sub":  claims.UserID,
		"role": claims.Role,
	}
	if claims.Email != "" {
		claimsMap["email"] = claims.Email
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
	if tenantID != "" {
		claimsMap["tenant_id"] = tenantID
	}

	claimsJSON, err := json.Marshal(claimsMap)
	if err != nil {
		return SendInternalError(c, "Failed to serialize JWT claims")
	}

	dbRole := "authenticated"
	if claims.Role == "anon" {
		dbRole = "anon"
	}

	log.Info().
		Str("audit_user_id", auditUserID).
		Str("db_role", dbRole).
		Str("tenant_id", tenantID).
		Bool("is_instance_admin", isInstanceAdmin).
		Msg("SQL query execution with tenant RLS enforcement")

	results := make([]SQLResult, 0, len(statements))

	tx, err := conn.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return SendInternalError(c, "Failed to begin transaction")
	}
	defer func() { _ = tx.Rollback(ctx) }()

	_, err = tx.Exec(ctx, "SELECT set_config('request.jwt.claims', $1, true)", string(claimsJSON))
	if err != nil {
		return SendInternalError(c, "Failed to set JWT claims")
	}

	if tenantID != "" {
		_, err = tx.Exec(ctx, "SELECT set_config('app.current_tenant_id', $1, true)", tenantID)
		if err != nil {
			return SendInternalError(c, "Failed to set tenant context")
		}
	}

	_, err = tx.Exec(ctx, fmt.Sprintf("SET LOCAL ROLE %q", dbRole))
	if err != nil {
		return SendInternalError(c, "Failed to set database role")
	}

	for _, stmt := range statements {
		stmt = strings.TrimSpace(stmt)
		if stmt == "" {
			continue
		}

		result := h.executeStatementInTx(ctx, tx, stmt)
		results = append(results, result)

		if result.Error != nil {
			log.Warn().
				Str("audit_user_id", auditUserID).
				Str("tenant_id", tenantID).
				Str("statement", util.TruncateString(stmt, 100)).
				Str("error", *result.Error).
				Msg("SQL query execution failed (tenant RLS)")
		} else {
			log.Info().
				Str("audit_user_id", auditUserID).
				Str("tenant_id", tenantID).
				Str("statement", util.TruncateString(stmt, 100)).
				Int("row_count", result.RowCount).
				Msg("SQL query executed successfully (tenant RLS)")
		}
	}

	if err := tx.Commit(ctx); err != nil {
		log.Warn().Err(err).Msg("Failed to commit tenant RLS transaction")
	}

	return c.JSON(ExecuteSQLResponse{Results: results})
}

func (h *SQLHandler) executeStatementInTx(ctx context.Context, tx pgx.Tx, statement string) SQLResult {
	startTime := time.Now()

	result := SQLResult{
		Statement: statement,
	}

	rows, err := tx.Query(ctx, statement)
	if err != nil {
		errorMsg := err.Error()
		result.Error = &errorMsg
		result.ExecutionTimeMS = float64(time.Since(startTime).Milliseconds())
		return result
	}
	defer rows.Close()

	fieldDescriptions := rows.FieldDescriptions()
	columns := make([]string, len(fieldDescriptions))
	for i, fd := range fieldDescriptions {
		columns[i] = string(fd.Name)
	}
	result.Columns = columns

	if len(columns) > 0 {
		resultRows := make([]map[string]any, 0)
		rowCount := 0

		for rows.Next() {
			if rowCount >= maxRowsPerQuery {
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
		for rows.Next() {
		}
	}

	if err := rows.Err(); err != nil {
		errorMsg := err.Error()
		result.Error = &errorMsg
	}

	commandTag := rows.CommandTag()
	if len(columns) == 0 {
		result.AffectedRows = commandTag.RowsAffected()
		result.RowCount = int(commandTag.RowsAffected())
	}

	result.ExecutionTimeMS = float64(time.Since(startTime).Milliseconds())
	return result
}

func splitSQLStatements(query string) []string {
	stmts, err := pg_query.SplitWithParser(query, true)
	if err != nil {
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

	if len(stmts) == 0 {
		return []string{}
	}

	return stmts
}

func convertValue(v any) any {
	if v == nil {
		return nil
	}

	if b, ok := v.([16]byte); ok {
		return formatUUID(b[:])
	}

	if b, ok := v.([]byte); ok && len(b) == 16 {
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

func formatUUID(b []byte) string {
	return fmt.Sprintf(
		"%s-%s-%s-%s-%s",
		hex.EncodeToString(b[0:4]),
		hex.EncodeToString(b[4:6]),
		hex.EncodeToString(b[6:8]),
		hex.EncodeToString(b[8:10]),
		hex.EncodeToString(b[10:16]),
	)
}
