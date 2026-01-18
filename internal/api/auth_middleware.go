package api

import (
	"context"
	"strings"

	"github.com/fluxbase-eu/fluxbase/internal/auth"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog/log"
)

// AuthMiddleware creates a middleware for JWT authentication
func AuthMiddleware(authService *auth.Service) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Try to get token from cookie first (httpOnly cookie)
		token := c.Cookies(AccessTokenCookieName)

		// Fall back to Authorization header for API clients
		if token == "" {
			authHeader := c.Get("Authorization")
			if authHeader == "" {
				return SendMissingAuth(c)
			}

			// Extract token from "Bearer <token>" format
			token = authHeader
			if strings.HasPrefix(authHeader, "Bearer ") {
				token = strings.TrimPrefix(authHeader, "Bearer ")
			}
		}

		// Validate token
		claims, err := authService.ValidateToken(token)
		if err != nil {
			log.Debug().Err(err).Msg("Invalid token")
			return SendInvalidToken(c)
		}

		// Check if token has been revoked
		isRevoked, err := authService.IsTokenRevoked(c.Context(), claims.ID)
		if err != nil {
			log.Error().Err(err).Msg("Failed to check token revocation status")
			// Continue anyway - revocation check failure shouldn't block valid tokens
		} else if isRevoked {
			log.Debug().Str("jti", claims.ID).Msg("Token has been revoked")
			return SendTokenRevoked(c)
		}

		// Store user information in context
		c.Locals("user_id", claims.UserID)
		c.Locals("user_email", claims.Email)
		c.Locals("user_role", claims.Role)
		c.Locals("session_id", claims.SessionID)
		c.Locals("jwt_claims", claims) // Store full claims for Supabase compatibility

		// Continue to next handler
		return c.Next()
	}
}

// OptionalAuthMiddleware creates a middleware that validates JWT but doesn't require it
// Useful for endpoints that work both authenticated and unauthenticated
func OptionalAuthMiddleware(authService *auth.Service) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Try to get token from cookie first (httpOnly cookie)
		token := c.Cookies(AccessTokenCookieName)

		// Fall back to Authorization header for API clients
		authHeader := c.Get("Authorization")

		log.Debug().
			Str("path", c.Path()).
			Bool("has_cookie", token != "").
			Bool("has_auth_header", authHeader != "").
			Msg("OptionalAuthMiddleware: Processing request")

		if token == "" && authHeader == "" {
			// No token provided, continue without authentication
			log.Debug().Str("path", c.Path()).Msg("OptionalAuthMiddleware: No auth, continuing")
			return c.Next()
		}

		// If no cookie token, use header token
		if token == "" {
			token = authHeader
			if strings.HasPrefix(authHeader, "Bearer ") {
				token = strings.TrimPrefix(authHeader, "Bearer ")
			}
		}

		// Validate token
		claims, err := authService.ValidateToken(token)
		if err != nil {
			// Invalid token, but continue anyway since auth is optional
			log.Debug().Err(err).Str("path", c.Path()).Msg("Invalid token in optional auth")
			return c.Next()
		}

		// Check if token has been revoked
		isRevoked, err := authService.IsTokenRevoked(c.Context(), claims.ID)
		if err != nil {
			log.Error().Err(err).Msg("Failed to check token revocation status in optional auth")
			// Continue anyway - revocation check failure shouldn't block valid tokens
		} else if isRevoked {
			// Token is revoked, continue without authentication
			log.Debug().Str("jti", claims.ID).Msg("Revoked token in optional auth, continuing unauthenticated")
			return c.Next()
		}

		// Store user information in context
		c.Locals("user_id", claims.UserID)
		c.Locals("user_email", claims.Email)
		c.Locals("user_role", claims.Role)
		c.Locals("session_id", claims.SessionID)
		c.Locals("jwt_claims", claims) // Store full claims for Supabase compatibility

		log.Debug().
			Str("user_id", claims.UserID).
			Str("path", c.Path()).
			Msg("OptionalAuthMiddleware: Set user context")

		return c.Next()
	}
}

// RequireRole creates a middleware that requires a specific role
// Must be used after AuthMiddleware
func RequireRole(allowedRoles ...string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		userRole := c.Locals("user_role")
		if userRole == nil {
			return SendUnauthorized(c, "Unauthorized", ErrCodeAuthRequired)
		}

		// Check if user role is in allowed roles (with safe type assertion)
		role, ok := userRole.(string)
		if !ok {
			return SendUnauthorized(c, "Invalid role type", ErrCodeInvalidRole)
		}
		for _, allowedRole := range allowedRoles {
			if role == allowedRole {
				return c.Next()
			}
		}

		return SendInsufficientPermissions(c)
	}
}

// GetUserID is a helper to extract user ID from context
func GetUserID(c *fiber.Ctx) (string, bool) {
	userID := c.Locals("user_id")
	if userID == nil {
		return "", false
	}
	id, ok := userID.(string)
	return id, ok
}

// GetUserEmail is a helper to extract user email from context
func GetUserEmail(c *fiber.Ctx) (string, bool) {
	email := c.Locals("user_email")
	if email == nil {
		return "", false
	}
	e, ok := email.(string)
	return e, ok
}

// GetUserRole is a helper to extract user role from context
func GetUserRole(c *fiber.Ctx) (string, bool) {
	role := c.Locals("user_role")
	if role == nil {
		return "", false
	}
	r, ok := role.(string)
	return r, ok
}

// UnifiedAuthMiddleware creates a middleware that accepts both auth.users and dashboard.users authentication
// This allows both application users with admin role AND dashboard admins to access admin endpoints.
// The db parameter is used to check the actual role from auth.users when JWT role is "authenticated",
// allowing role changes to take effect immediately without requiring re-login.
func UnifiedAuthMiddleware(authService *auth.Service, jwtManager *auth.JWTManager, db *pgxpool.Pool) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Try to get token from cookie first (httpOnly cookie)
		token := c.Cookies(AccessTokenCookieName)

		// Fall back to Authorization header for API clients
		if token == "" {
			authHeader := c.Get("Authorization")
			if authHeader == "" {
				return SendMissingAuth(c)
			}

			// Extract token from "Bearer <token>" format
			token = authHeader
			if strings.HasPrefix(authHeader, "Bearer ") {
				token = strings.TrimPrefix(authHeader, "Bearer ")
			}
		}

		// First, try to validate as auth.users token
		claims, err := authService.ValidateToken(token)
		if err == nil {
			// Check if this is a dashboard admin token (dashboard.users)
			// Dashboard tokens use the same JWT secret but have role="dashboard_admin"
			// and store the user ID in Subject instead of UserID
			if claims.Role == "dashboard_admin" {
				c.Locals("user_id", claims.Subject)
				c.Locals("user_email", claims.Email)
				c.Locals("user_role", claims.Role)
				c.Locals("jwt_claims", claims)

				log.Debug().
					Str("user_id", claims.Subject).
					Str("role", claims.Role).
					Msg("Authenticated as dashboard.users via role check")

				return c.Next()
			}

			// Successfully validated as auth.users token
			// Check if token has been revoked
			isRevoked, err := authService.IsTokenRevoked(c.Context(), claims.ID)
			if err != nil {
				log.Error().Err(err).Msg("Failed to check token revocation status")
				// Continue anyway - revocation check failure shouldn't block valid tokens
			} else if isRevoked {
				log.Debug().Str("jti", claims.ID).Msg("Token has been revoked")
				return SendTokenRevoked(c)
			}

			// Store user information in context
			c.Locals("user_id", claims.UserID)
			c.Locals("user_email", claims.Email)
			c.Locals("session_id", claims.SessionID)
			c.Locals("jwt_claims", claims) // Store full claims for Supabase compatibility

			// Check actual role from database if JWT role is "authenticated"
			// This allows role changes to take effect immediately without re-login
			effectiveRole := claims.Role
			if claims.Role == "authenticated" && db != nil {
				dbRole, err := getUserRoleFromDB(c.Context(), db, claims.UserID)
				if err == nil && (dbRole == "admin" || dbRole == "service_role") {
					effectiveRole = dbRole
					log.Debug().
						Str("user_id", claims.UserID).
						Str("jwt_role", claims.Role).
						Str("db_role", dbRole).
						Msg("Elevated role from database")
				}
			}
			c.Locals("user_role", effectiveRole)

			return c.Next()
		}

		// If auth.users validation failed, try dashboard.users token
		dashboardClaims, err := jwtManager.ValidateAccessToken(token)
		if err != nil {
			// Both validations failed
			log.Debug().Err(err).Msg("Invalid token for both auth types")
			return SendInvalidToken(c)
		}

		// Successfully validated as dashboard.users token
		userID, err := uuid.Parse(dashboardClaims.Subject)
		if err != nil {
			return SendUnauthorized(c, "Invalid user ID in token", ErrCodeInvalidUserID)
		}

		// Store user information in context
		c.Locals("user_id", userID.String())
		c.Locals("user_email", dashboardClaims.Email)
		c.Locals("user_role", dashboardClaims.Role)
		c.Locals("jwt_claims", dashboardClaims) // Store full claims for Supabase compatibility

		log.Debug().
			Str("user_id", userID.String()).
			Str("role", dashboardClaims.Role).
			Msg("Authenticated as dashboard.users")

		return c.Next()
	}
}

// getUserRoleFromDB fetches user's role from auth.users table.
// Also checks app_metadata.role as fallback.
// This allows role changes to take effect immediately without re-login.
func getUserRoleFromDB(ctx context.Context, db *pgxpool.Pool, userID string) (string, error) {
	var role string
	var appMetadata []byte

	err := db.QueryRow(ctx, `
		SELECT role, app_metadata
		FROM auth.users
		WHERE id = $1
	`, userID).Scan(&role, &appMetadata)

	if err != nil {
		return "", err
	}

	// Only use the explicit database role column for authorization
	// app_metadata.role is NOT used for privilege elevation as it could be user-editable
	return role, nil
}
