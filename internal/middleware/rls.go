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
			Str("auth_header", c.Get("Authorization")).
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
	var queries []string

	// Set user ID if present
	if userID != nil {
		queries = append(queries, fmt.Sprintf("SET LOCAL app.user_id = '%v'", userID))
	} else {
		queries = append(queries, "SET LOCAL app.user_id = ''")
	}

	// Set role (anon, authenticated, admin, etc.)
	queries = append(queries, fmt.Sprintf("SET LOCAL app.role = '%s'", role))

	// Execute all session variable sets
	for _, query := range queries {
		if _, err := tx.Exec(ctx, query); err != nil {
			log.Error().Err(err).Str("query", query).Msg("Failed to set RLS session variable")
			return fmt.Errorf("failed to set RLS context: %w", err)
		}
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
