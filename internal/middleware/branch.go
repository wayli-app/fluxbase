package middleware

import (
	"github.com/fluxbase-eu/fluxbase/internal/branching"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog/log"
)

const (
	// BranchHeader is the HTTP header for specifying the branch
	BranchHeader = "X-Fluxbase-Branch"

	// BranchQueryParam is the query parameter for specifying the branch
	BranchQueryParam = "branch"

	// LocalsBranchSlug is the Fiber locals key for the branch slug
	LocalsBranchSlug = "branch_slug"

	// LocalsBranchPool is the Fiber locals key for the branch connection pool
	LocalsBranchPool = "branch_pool"

	// LocalsBranch is the Fiber locals key for the branch object
	LocalsBranch = "branch"
)

// BranchContextConfig holds configuration for the branch context middleware
type BranchContextConfig struct {
	// Router is the branch router for getting connection pools
	Router *branching.Router

	// RequireAccess determines if access checks should be performed
	// When true, authenticated users must have explicit access to non-main branches
	RequireAccess bool

	// AllowAnonymous determines if anonymous users can access branches
	// When true, anonymous users can only access the main branch
	AllowAnonymous bool
}

// BranchContext creates a middleware that extracts branch context from requests
// and sets up the appropriate connection pool
func BranchContext(config BranchContextConfig) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Extract branch slug from header or query param
		branchSlug := c.Get(BranchHeader)
		if branchSlug == "" {
			branchSlug = c.Query(BranchQueryParam)
		}

		// Default to main branch
		if branchSlug == "" {
			branchSlug = "main"
		}

		// Get user ID from context (if authenticated)
		var userID *uuid.UUID
		if uid, ok := c.Locals("user_id").(string); ok && uid != "" {
			if id, err := uuid.Parse(uid); err == nil {
				userID = &id
			}
		}

		// For non-main branches, check access
		if !branching.IsMainBranch(branchSlug) {
			// Check if branching is enabled
			if config.Router == nil {
				return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{
					"error":   "branching_disabled",
					"message": "Database branching is not enabled",
				})
			}

			// Check authentication for non-main branches
			if config.RequireAccess && userID == nil && !config.AllowAnonymous {
				return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
					"error":   "authentication_required",
					"message": "Authentication is required to access branches",
				})
			}

			// Check access if required and user is authenticated
			if config.RequireAccess && userID != nil {
				hasAccess, err := config.Router.GetStorage().UserHasAccess(c.Context(), branchSlug, *userID)
				if err != nil {
					log.Error().Err(err).
						Str("branch", branchSlug).
						Str("user_id", userID.String()).
						Msg("Failed to check branch access")
					return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
						"error":   "access_check_failed",
						"message": "Failed to verify branch access",
					})
				}

				if !hasAccess {
					return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
						"error":   "access_denied",
						"message": "You do not have access to this branch",
					})
				}
			}
		}

		// Get connection pool for the branch
		var pool *pgxpool.Pool
		var err error

		if config.Router != nil {
			pool, err = config.Router.GetPool(c.Context(), branchSlug)
			if err != nil {
				if err == branching.ErrBranchNotFound {
					return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
						"error":   "branch_not_found",
						"message": "Branch not found: " + branchSlug,
					})
				}
				if err == branching.ErrBranchNotReady {
					return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{
						"error":   "branch_not_ready",
						"message": "Branch is not ready: " + branchSlug,
					})
				}
				if err == branching.ErrBranchingDisabled {
					// For main branch, we should still work
					if branching.IsMainBranch(branchSlug) {
						pool = config.Router.GetMainPool()
					} else {
						return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{
							"error":   "branching_disabled",
							"message": "Database branching is not enabled",
						})
					}
				} else {
					log.Error().Err(err).
						Str("branch", branchSlug).
						Msg("Failed to get branch pool")
					return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
						"error":   "pool_error",
						"message": "Failed to get database connection for branch",
					})
				}
			}
		}

		// Store branch context in locals
		c.Locals(LocalsBranchSlug, branchSlug)
		if pool != nil {
			c.Locals(LocalsBranchPool, pool)
		}

		// Log branch context for debugging
		if branchSlug != "main" {
			log.Debug().
				Str("branch", branchSlug).
				Str("path", c.Path()).
				Msg("Request using branch database")
		}

		return c.Next()
	}
}

// GetBranchSlug extracts the branch slug from Fiber context
func GetBranchSlug(c *fiber.Ctx) string {
	if slug, ok := c.Locals(LocalsBranchSlug).(string); ok {
		return slug
	}
	return "main"
}

// GetBranchPool extracts the branch connection pool from Fiber context
func GetBranchPool(c *fiber.Ctx) *pgxpool.Pool {
	if pool, ok := c.Locals(LocalsBranchPool).(*pgxpool.Pool); ok {
		return pool
	}
	return nil
}

// IsUsingBranch checks if the request is using a non-main branch
func IsUsingBranch(c *fiber.Ctx) bool {
	slug := GetBranchSlug(c)
	return !branching.IsMainBranch(slug)
}

// BranchContextSimple creates a simple middleware that only sets branch context
// without access checks (useful for internal routes)
func BranchContextSimple(router *branching.Router) fiber.Handler {
	return BranchContext(BranchContextConfig{
		Router:         router,
		RequireAccess:  false,
		AllowAnonymous: true,
	})
}

// RequireBranchAccess creates a middleware that requires branch access
// This should be used after authentication middleware
func RequireBranchAccess(router *branching.Router) fiber.Handler {
	return BranchContext(BranchContextConfig{
		Router:         router,
		RequireAccess:  true,
		AllowAnonymous: false,
	})
}
