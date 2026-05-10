package middleware

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/gofiber/fiber/v3"
	"github.com/jackc/pgx/v5"
	"github.com/rs/zerolog/log"

	"github.com/nimbleflux/fluxbase/internal/auth"
	"github.com/nimbleflux/fluxbase/internal/database"
)

// TestTransactionMiddleware returns a Fiber middleware that injects a test transaction
// into the request context. When set, WrapWithRLS reuses this transaction instead of
// beginning a new one, enabling HTTP-level tests with transaction isolation.
func TestTransactionMiddleware(tx pgx.Tx) fiber.Handler {
	return func(c fiber.Ctx) error {
		c.Locals("test_tx", tx)
		return c.Next()
	}
}

// quoteIdentifier safely quotes a PostgreSQL identifier to prevent SQL injection.
// It wraps the identifier in double quotes and escapes any embedded double quotes.
func quoteIdentifier(identifier string) string {
	return `"` + strings.ReplaceAll(identifier, `"`, `""`) + `"`
}

// RLSConfig holds configuration for RLS middleware
type RLSConfig struct {
	// DB is the database connection pool
	DB *database.Connection

	// SessionVarPrefix is the prefix for PostgreSQL session variables
	// Default: "app"
	SessionVarPrefix string
}

// RLSMiddleware enforces Row Level Security by setting PostgreSQL session variables
// based on the authenticated user context
func RLSMiddleware(config RLSConfig) fiber.Handler {
	// Set default prefix if not provided
	if config.SessionVarPrefix == "" {
		config.SessionVarPrefix = "app"
	}

	return func(c fiber.Ctx) error {
		// Get user ID from context (set by auth middleware)
		userID := c.Locals("user_id")

		// Check if role is already set (e.g., by service key authentication)
		existingRole := c.Locals("rls_role")

		// Debug logging
		log.Debug().
			Interface("user_id", userID).
			Interface("existing_rls_role", existingRole).
			Str("path", c.Path()).
			Msg("RLSMiddleware: checking user_id from context")

		// If no user is authenticated AND no role is already set, treat as anonymous
		// This allows service key auth to set service_role even without a user_id
		if userID == nil && existingRole == nil {
			// Store in context that this is an anonymous request
			c.Locals("rls_user_id", nil)
			c.Locals("rls_role", "anon")
			log.Debug().Str("path", c.Path()).Msg("RLSMiddleware: No user_id and no existing role, setting anonymous")
			return c.Next()
		}

		// If role is already set (e.g., service_role from service key auth), preserve it
		if existingRole != nil {
			log.Debug().
				Interface("existing_rls_role", existingRole).
				Str("path", c.Path()).
				Msg("RLSMiddleware: Preserving existing RLS role")
			return c.Next()
		}

		// Store RLS context for use in query execution
		c.Locals("rls_user_id", userID)

		// Map application role to database role (handles instance_admin, tenant_service -> service_role)
		// Important: Check for both non-nil AND non-empty string to avoid
		// overwriting "authenticated" default with empty string (which maps to "anon")
		if role := c.Locals("user_role"); role != nil {
			if roleStr, ok := role.(string); ok && roleStr != "" {
				c.Locals("rls_role", mapAppRoleToDatabaseRole(roleStr))
			} else {
				c.Locals("rls_role", "authenticated")
			}
		} else {
			c.Locals("rls_role", "authenticated")
		}

		log.Debug().
			Interface("rls_user_id", userID).
			Interface("rls_role", c.Locals("rls_role")).
			Str("path", c.Path()).
			Msg("RLSMiddleware: Set RLS context")

		return c.Next()
	}
}

// mapAppRoleToDatabaseRole maps application-level roles to database-level roles
// This separates concerns: database roles (anon, authenticated, service_role) for
// PostgreSQL-level security, and application roles (admin, user, etc.) for business logic
func mapAppRoleToDatabaseRole(appRole string) string {
	switch appRole {
	case "service_role", "instance_admin":
		// Service role and instance_admin map to service_role - has BYPASSRLS privilege
		// Instance admins are Fluxbase platform admins and need full data access
		return "service_role"
	case "tenant_service":
		return "tenant_service"
	case "tenant_admin":
		// Tenant admins map to authenticated - they respect RLS but have tenant context
		// The tenant_id is set separately in the JWT claims for tenant-scoped access
		return "authenticated"
	case "anon", "":
		// Anonymous or empty role maps to anon
		return "anon"
	default:
		// All other roles (admin, user, authenticated, moderator, etc.)
		// map to the authenticated database role
		return "authenticated"
	}
}

// SetRLSContext sets PostgreSQL session variables for RLS enforcement
// This should be called at the beginning of each database transaction
// Uses hybrid approach: SET ROLE for database-level security + request.jwt.claims for app-level authorization
func SetRLSContext(ctx context.Context, tx pgx.Tx, userID string, role string, claims *auth.TokenClaims) error {
	// Map application role to database role for SET ROLE
	dbRole := mapAppRoleToDatabaseRole(role)

	// Enhanced debug logging for service_role troubleshooting
	log.Debug().
		Str("input_user_id", userID).
		Str("input_role", role).
		Str("mapped_db_role", dbRole).
		Bool("has_claims", claims != nil).
		Msg("SetRLSContext: Starting RLS context setup")

	// Validate database role (defense in depth - should always be one of these)
	validDBRoles := map[string]bool{
		"anon":           true,
		"authenticated":  true,
		"service_role":   true,
		"tenant_service": true,
	}

	if !validDBRoles[dbRole] {
		log.Error().Str("app_role", role).Str("db_role", dbRole).Msg("Invalid database role after mapping")
		return fmt.Errorf("invalid database role: %s (mapped from app role: %s)", dbRole, role)
	}

	// SET LOCAL ROLE for database-level security
	// This provides defense-in-depth: connection runs with minimal privileges
	// Using quoteIdentifier for proper PostgreSQL identifier escaping (defense in depth)
	setRoleQuery := fmt.Sprintf("SET LOCAL ROLE %s", quoteIdentifier(dbRole))
	_, err := tx.Exec(ctx, setRoleQuery)
	if err != nil {
		log.Error().Err(err).Str("db_role", dbRole).Msg("Failed to SET LOCAL ROLE")
		return fmt.Errorf("failed to SET LOCAL ROLE %s: %w", dbRole, err)
	}
	log.Debug().
		Str("app_role", role).
		Str("db_role", dbRole).
		Msg("SET LOCAL ROLE executed successfully")

	// Build JWT claims JSON
	// IMPORTANT: Store the ORIGINAL application role (not the mapped database role)
	// This allows RLS policies to check fine-grained application roles like 'admin', 'moderator', etc.
	jwtClaims := map[string]interface{}{
		"sub":  userID,
		"role": role,   // Original application role (admin, user, etc.) - NOT the database role
	}

	// Add optional fields if claims are provided
	if claims != nil {
		if claims.Email != "" {
			jwtClaims["email"] = claims.Email
		}
		if claims.SessionID != "" {
			jwtClaims["session_id"] = claims.SessionID
		}
		if claims.UserMetadata != nil {
			jwtClaims["user_metadata"] = claims.UserMetadata
		}
		if claims.AppMetadata != nil {
			jwtClaims["app_metadata"] = claims.AppMetadata
		}
		if claims.IsAnonymous {
			jwtClaims["is_anonymous"] = claims.IsAnonymous
		}
		// Multi-tenancy fields
		if claims.TenantID != nil {
			jwtClaims["tenant_id"] = *claims.TenantID
		}
		if claims.TenantRole != "" {
			jwtClaims["tenant_role"] = claims.TenantRole
		}
		if claims.IsInstanceAdmin {
			jwtClaims["is_instance_admin"] = true
		}
	}

	// Marshal to JSON
	jwtClaimsJSON, err := json.Marshal(jwtClaims)
	if err != nil {
		log.Error().Err(err).Msg("Failed to marshal JWT claims")
		return fmt.Errorf("failed to marshal JWT claims: %w", err)
	}

	// Set request.jwt.claims session variable
	_, err = tx.Exec(ctx, "SELECT set_config('request.jwt.claims', $1, true)", string(jwtClaimsJSON))
	if err != nil {
		log.Error().Err(err).Msg("Failed to set request.jwt.claims")
		return fmt.Errorf("failed to set request.jwt.claims: %w", err)
	}
	// Log redacted claims (avoid leaking PII/metadata in debug logs)
	log.Debug().
		Str("user_id", userID).
		Str("role", role).
		Bool("has_tenant", claims != nil && claims.TenantID != nil).
		Msg("Set request.jwt.claims using parameterized query")

	// Set tenant context for multi-tenancy (app.current_tenant_id)
	// Priority: claims.TenantID > Go context tenant_id (from TenantMiddleware)
	if claims != nil && claims.TenantID != nil {
		tenantIDStr := *claims.TenantID
		_, err = tx.Exec(ctx, "SELECT set_config('app.current_tenant_id', $1, true)", tenantIDStr)
		if err != nil {
			log.Error().Err(err).Str("tenant_id", tenantIDStr).Msg("Failed to set app.current_tenant_id")
			return fmt.Errorf("failed to set app.current_tenant_id: %w", err)
		}
		log.Debug().Str("tenant_id", tenantIDStr).Msg("Set app.current_tenant_id from claims")
	} else if tenantID := database.TenantFromContext(ctx); tenantID != "" {
		_, err = tx.Exec(ctx, "SELECT set_config('app.current_tenant_id', $1, true)", tenantID)
		if err != nil {
			log.Error().Err(err).Str("tenant_id", tenantID).Msg("Failed to set app.current_tenant_id")
			return fmt.Errorf("failed to set app.current_tenant_id: %w", err)
		}
		log.Debug().Str("tenant_id", tenantID).Msg("Set app.current_tenant_id from context")
	}

	log.Debug().
		Str("user_id", userID).
		Str("role", role).
		Bool("has_tenant", claims != nil && claims.TenantID != nil).
		Msg("RLS context set for transaction")

	return nil
}

// WrapWithServiceRole wraps a database operation with service_role context
// Used for privileged operations like auth, admin tasks, and webhooks
func WrapWithServiceRole(ctx context.Context, conn *database.Connection, fn func(tx pgx.Tx) error) error {
	// Start transaction
	tx, err := conn.Pool().Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	// SET LOCAL ROLE service_role - bypasses RLS for privileged operations
	// This provides the same security model as separate admin connections
	// Using quoteIdentifier for consistent security practices
	_, err = tx.Exec(ctx, fmt.Sprintf("SET LOCAL ROLE %s", quoteIdentifier("service_role")))
	if err != nil {
		log.Error().Err(err).Msg("Failed to SET LOCAL ROLE service_role")
		return fmt.Errorf("failed to SET LOCAL ROLE service_role: %w", err)
	}

	log.Debug().Msg("SET LOCAL ROLE service_role - running privileged operation")

	// Execute the wrapped function
	if err := fn(tx); err != nil {
		return err
	}

	// Commit transaction
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// WrapWithRLS wraps a database operation with RLS context.
// Pool selection priority: branch pool > tenant pool (for public schema) > main pool.
// The target schema is determined by GetTargetSchema(c), which callers can set
// via SetTargetSchema before calling this function.
func WrapWithRLS(ctx context.Context, conn *database.Connection, c fiber.Ctx, fn func(tx pgx.Tx) error) error {
	if testTx, ok := c.Locals("test_tx").(pgx.Tx); ok && testTx != nil {
		role := c.Locals("rls_role")
		if role == nil {
			role = "anon"
		}
		var userIDStr string
		if userID := c.Locals("rls_user_id"); userID != nil {
			userIDStr = fmt.Sprintf("%v", userID)
		}
		var claims *auth.TokenClaims
		if jwtClaims := c.Locals("jwt_claims"); jwtClaims != nil {
			if tc, ok := jwtClaims.(*auth.TokenClaims); ok {
				claims = tc
			}
		}
		roleStr := role.(string)
		tenantID, hasTenant := c.Locals("tenant_id").(string)
		isDefaultTenant, _ := c.Locals("is_default_tenant").(bool)
		if hasTenant && tenantID != "" && !isDefaultTenant {
			if roleStr == "instance_admin" || roleStr == "service_role" {
				roleStr = "tenant_service"
			}
		}
		if err := SetRLSContext(ctx, testTx, userIDStr, roleStr, claims); err != nil {
			return err
		}
		return fn(testTx)
	}

	pool := GetPoolForSchema(c, GetTargetSchema(c), conn.Pool())

	// Start transaction
	tx, err := pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	// Set RLS context from Fiber context
	userID := c.Locals("rls_user_id")
	role := c.Locals("rls_role")

	// Enhanced debug logging for service_role troubleshooting
	log.Debug().
		Interface("raw_rls_user_id", userID).
		Interface("raw_rls_role", role).
		Str("auth_type", fmt.Sprintf("%v", c.Locals("auth_type"))).
		Str("user_role", fmt.Sprintf("%v", c.Locals("user_role"))).
		Str("path", c.Path()).
		Msg("WrapWithRLS: Raw Fiber locals before processing")

	if role == nil {
		role = "anon"
		log.Debug().Msg("WrapWithRLS: rls_role was nil, defaulting to 'anon'")
	}

	// Extract JWT claims if available
	var claims *auth.TokenClaims
	if jwtClaims := c.Locals("jwt_claims"); jwtClaims != nil {
		if tc, ok := jwtClaims.(*auth.TokenClaims); ok {
			claims = tc
		}
	}

	// Convert userID to string
	var userIDStr string
	if userID != nil {
		userIDStr = fmt.Sprintf("%v", userID)
	}

	// When a tenant context is active, override instance_admin/service_role to tenant_service.
	// This ensures FDW queries work (no user mapping for service_role on tenant DBs)
	// and RLS enforces tenant isolation for the selected tenant.
	roleStr := role.(string)
	tenantID, hasTenant := c.Locals("tenant_id").(string)
	isDefaultTenant, _ := c.Locals("is_default_tenant").(bool)
	if hasTenant && tenantID != "" && !isDefaultTenant {
		if roleStr == "instance_admin" || roleStr == "service_role" {
			roleStr = "tenant_service"
		}
	}

	log.Debug().
		Str("user_id", userIDStr).
		Str("role", roleStr).
		Bool("has_jwt_claims", claims != nil).
		Bool("tenant_context", hasTenant && tenantID != "").
		Str("path", c.Path()).
		Msg("WrapWithRLS: Retrieved RLS context from Fiber locals")

	if err := SetRLSContext(ctx, tx, userIDStr, roleStr, claims); err != nil {
		return err
	}

	// Execute the wrapped function
	if err := fn(tx); err != nil {
		return err
	}

	// Commit transaction
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// GetRLSContext extracts RLS context from Fiber context
type RLSContext struct {
	UserID interface{}
	Role   string
}

func GetRLSContext(c fiber.Ctx) RLSContext {
	role := c.Locals("rls_role")
	if role == nil {
		role = "anon"
	}

	return RLSContext{
		UserID: c.Locals("rls_user_id"),
		Role:   role.(string),
	}
}

// LogRLSViolation logs an RLS policy violation to the audit log table
// This should be called when an operation is blocked by RLS policies
func LogRLSViolation(ctx context.Context, db *database.Connection, c fiber.Ctx, operation string, tableName string) {
	// Extract schema and table name if combined
	schema := "public"
	table := tableName
	if len(tableName) > 0 && tableName[0] != '$' {
		// Parse schema.table format
		parts := splitTableName(tableName)
		if len(parts) == 2 {
			schema = parts[0]
			table = parts[1]
		}
	}

	// Get RLS context
	rlsCtx := GetRLSContext(c)
	role := rlsCtx.Role
	if role == "" {
		role = "anon"
	}

	// Get request context
	ip := c.IP()
	userAgent := c.Get("User-Agent")
	requestID := c.Get("X-Request-ID")
	if requestID == "" {
		if reqID := c.Locals("request_id"); reqID != nil {
			if reqIDStr, ok := reqID.(string); ok {
				requestID = reqIDStr
			}
		}
	}

	// Convert user_id to string for logging
	var userIDStr *string
	if rlsCtx.UserID != nil {
		userIDVal := fmt.Sprintf("%v", rlsCtx.UserID)
		userIDStr = &userIDVal
	}

	// Build details JSONB
	details := map[string]interface{}{
		"path":   c.Path(),
		"method": c.Method(),
	}

	detailsJSON, err := json.Marshal(details)
	if err != nil {
		log.Error().Err(err).Msg("Failed to marshal RLS audit details")
		detailsJSON = []byte("{}")
	}

	// Insert audit log entry
	// We use a separate connection without RLS context to avoid infinite loops
	query := `
		INSERT INTO auth.rls_audit_log (
			user_id, role, operation, table_schema, table_name,
			allowed, ip_address, user_agent, request_id, details
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10
		)
	`

	_, err = db.Exec(
		ctx, query,
		userIDStr,   // user_id
		role,        // role
		operation,   // operation (INSERT, UPDATE, DELETE, SELECT)
		schema,      // table_schema
		table,       // table_name
		false,       // allowed (false = violation)
		ip,          // ip_address
		userAgent,   // user_agent
		requestID,   // request_id
		detailsJSON, // details
	)
	if err != nil {
		// Log error but don't fail the request
		log.Error().
			Err(err).
			Str("operation", operation).
			Str("table", tableName).
			Interface("user_id", rlsCtx.UserID).
			Msg("Failed to log RLS violation to audit table")
	}
}

// splitTableName splits a "schema.table" string into [schema, table]
func splitTableName(fullName string) []string {
	parts := make([]string, 0, 2)
	dotIndex := -1
	for i, char := range fullName {
		if char == '.' {
			dotIndex = i
			break
		}
	}

	if dotIndex > 0 {
		parts = append(parts, fullName[:dotIndex])
		parts = append(parts, fullName[dotIndex+1:])
	} else {
		parts = append(parts, fullName)
	}

	return parts
}
