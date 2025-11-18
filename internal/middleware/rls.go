package middleware

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5"
	"github.com/rs/zerolog/log"
	"github.com/wayli-app/fluxbase/internal/auth"
	"github.com/wayli-app/fluxbase/internal/database"
)

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

	return func(c *fiber.Ctx) error {
		// Get user ID from context (set by auth middleware)
		userID := c.Locals("user_id")

		// Debug logging
		log.Debug().
			Interface("user_id", userID).
			Str("path", c.Path()).
			Msg("RLSMiddleware: checking user_id from context")

		// If no user is authenticated, set empty session variables
		// This allows RLS policies to restrict access appropriately
		if userID == nil {
			// Store in context that this is an anonymous request
			c.Locals("rls_user_id", nil)
			c.Locals("rls_role", "anon")
			log.Debug().Str("path", c.Path()).Msg("RLSMiddleware: No user_id, setting anonymous")
			return c.Next()
		}

		// Store RLS context for use in query execution
		c.Locals("rls_user_id", userID)
		c.Locals("rls_role", "authenticated")

		// Also check for admin/superuser role if present
		if role := c.Locals("user_role"); role != nil {
			c.Locals("rls_role", role)
		}

		log.Debug().
			Interface("rls_user_id", userID).
			Interface("rls_role", c.Locals("rls_role")).
			Str("path", c.Path()).
			Msg("RLSMiddleware: Set RLS context")

		return c.Next()
	}
}

// SetRLSContext sets PostgreSQL session variables for RLS enforcement
// This should be called at the beginning of each database transaction
// Sets Supabase-compatible request.jwt.claims format
func SetRLSContext(ctx context.Context, tx pgx.Tx, userID string, role string, claims *auth.TokenClaims) error {
	// Validate role to prevent injection
	validRoles := map[string]bool{
		"anon":            true,
		"authenticated":   true,
		"admin":           true,
		"dashboard_admin": true,
		"service_role":    true,
	}

	if !validRoles[role] {
		log.Warn().Str("role", role).Msg("Invalid role provided to SetRLSContext")
		return fmt.Errorf("invalid role: %s", role)
	}

	// Build Supabase-compatible JWT claims JSON
	jwtClaims := map[string]interface{}{
		"sub":  userID, // Supabase uses 'sub' for user ID
		"role": role,
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
	}

	// Marshal to JSON
	jwtClaimsJSON, err := json.Marshal(jwtClaims)
	if err != nil {
		log.Error().Err(err).Msg("Failed to marshal JWT claims")
		return fmt.Errorf("failed to marshal JWT claims: %w", err)
	}

	// Set request.jwt.claims session variable (Supabase format)
	_, err = tx.Exec(ctx, "SELECT set_config('request.jwt.claims', $1, true)", string(jwtClaimsJSON))
	if err != nil {
		log.Error().Err(err).Msg("Failed to set request.jwt.claims")
		return fmt.Errorf("failed to set request.jwt.claims: %w", err)
	}
	log.Debug().Str("jwt_claims", string(jwtClaimsJSON)).Msg("Set request.jwt.claims using parameterized query")

	log.Debug().
		Str("user_id", userID).
		Str("role", role).
		Msg("RLS context set for transaction")

	return nil
}

// WrapWithRLS wraps a database operation with RLS context
// This is a helper function for setting RLS context in queries
func WrapWithRLS(ctx context.Context, conn *database.Connection, c *fiber.Ctx, fn func(tx pgx.Tx) error) error {
	// Start transaction
	tx, err := conn.Pool().Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	// Set RLS context from Fiber context
	userID := c.Locals("rls_user_id")
	role := c.Locals("rls_role")
	if role == nil {
		role = "anon"
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

	log.Debug().
		Str("user_id", userIDStr).
		Str("role", role.(string)).
		Bool("has_jwt_claims", claims != nil).
		Str("path", c.Path()).
		Msg("WrapWithRLS: Retrieved RLS context from Fiber locals")

	if err := SetRLSContext(ctx, tx, userIDStr, role.(string), claims); err != nil {
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

func GetRLSContext(c *fiber.Ctx) RLSContext {
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
func LogRLSViolation(ctx context.Context, db *database.Connection, c *fiber.Ctx, operation string, tableName string) {
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

	_, err = db.Exec(ctx, query,
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
