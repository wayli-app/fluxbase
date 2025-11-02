package middleware

import (
	"context"
	"fmt"

	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5"
	"github.com/rs/zerolog/log"
	"github.com/wayli-app/fluxbase/internal/database"
)

// RLSConfig holds configuration for RLS middleware
type RLSConfig struct {
	// DB is the database connection pool
	DB *database.Connection

	// Enabled controls whether RLS enforcement is active
	Enabled bool

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
		// Skip if RLS is disabled
		if !config.Enabled {
			return c.Next()
		}

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
func SetRLSContext(ctx context.Context, tx pgx.Tx, userID interface{}, role string) error {
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

	// Convert userID to string and validate if present
	var userIDStr string
	if userID != nil {
		userIDStr = fmt.Sprintf("%v", userID)

		// Validate UUID format if userID is provided
		// This prevents SQL injection through the userID parameter
		if userIDStr != "" {
			// Check if it's a valid UUID (basic validation: 36 chars with hyphens in right places)
			if len(userIDStr) != 36 || userIDStr[8] != '-' || userIDStr[13] != '-' ||
				userIDStr[18] != '-' || userIDStr[23] != '-' {
				log.Warn().Str("user_id", userIDStr).Msg("Invalid UUID format provided to SetRLSContext")
				return fmt.Errorf("invalid user_id format: must be a valid UUID")
			}
		}
	}

	// Use parameterized set_config() function instead of string interpolation
	// This is safe from SQL injection as PostgreSQL treats the values as data, not SQL
	if userIDStr != "" {
		_, err := tx.Exec(ctx, "SELECT set_config('app.user_id', $1, true)", userIDStr)
		if err != nil {
			log.Error().Err(err).Str("user_id", userIDStr).Msg("Failed to set RLS user_id")
			return fmt.Errorf("failed to set RLS user_id: %w", err)
		}
		log.Debug().Str("user_id", userIDStr).Msg("Set RLS user_id using parameterized query")
	} else {
		_, err := tx.Exec(ctx, "SELECT set_config('app.user_id', '', true)")
		if err != nil {
			log.Error().Err(err).Msg("Failed to set empty RLS user_id")
			return fmt.Errorf("failed to set empty RLS user_id: %w", err)
		}
		log.Debug().Msg("Set empty RLS user_id")
	}

	// Set role using parameterized query (role is already validated above)
	_, err := tx.Exec(ctx, "SELECT set_config('app.role', $1, true)", role)
	if err != nil {
		log.Error().Err(err).Str("role", role).Msg("Failed to set RLS role")
		return fmt.Errorf("failed to set RLS role: %w", err)
	}
	log.Debug().Str("role", role).Msg("Set RLS role using parameterized query")

	// Verify the RLS context was set by querying the session variables
	var checkUserID string
	var checkRole string
	var currentUser string
	if err := tx.QueryRow(ctx, "SELECT current_setting('app.user_id', TRUE), current_setting('app.role', TRUE), current_user").Scan(&checkUserID, &checkRole, &currentUser); err != nil {
		log.Warn().Err(err).Msg("Failed to verify RLS session variables")
	} else {
		log.Debug().
			Str("verified_user_id", checkUserID).
			Str("verified_role", checkRole).
			Str("current_pg_user", currentUser).
			Msg("Verified RLS session variables and PostgreSQL user")
	}

	log.Debug().
		Interface("user_id", userID).
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

	log.Debug().
		Interface("user_id_from_locals", userID).
		Interface("role_from_locals", role).
		Str("path", c.Path()).
		Msg("WrapWithRLS: Retrieved RLS context from Fiber locals")

	if err := SetRLSContext(ctx, tx, userID, role.(string)); err != nil {
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
