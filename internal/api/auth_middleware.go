package api

import (
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
	"github.com/wayli-app/fluxbase/internal/auth"
)

// AuthMiddleware creates a middleware for JWT authentication
func AuthMiddleware(authService *auth.Service) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Get token from Authorization header
		authHeader := c.Get("Authorization")
		if authHeader == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Missing authorization header",
			})
		}

		// Extract token from "Bearer <token>" format
		token := authHeader
		if strings.HasPrefix(authHeader, "Bearer ") {
			token = strings.TrimPrefix(authHeader, "Bearer ")
		}

		// Validate token
		claims, err := authService.ValidateToken(token)
		if err != nil {
			log.Debug().Err(err).Msg("Invalid token")
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Invalid or expired token",
			})
		}

		// Check if token has been revoked
		isRevoked, err := authService.IsTokenRevoked(c.Context(), claims.ID)
		if err != nil {
			log.Error().Err(err).Msg("Failed to check token revocation status")
			// Continue anyway - revocation check failure shouldn't block valid tokens
		} else if isRevoked {
			log.Debug().Str("jti", claims.ID).Msg("Token has been revoked")
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Token has been revoked",
			})
		}

		// Store user information in context
		c.Locals("user_id", claims.UserID)
		c.Locals("user_email", claims.Email)
		c.Locals("user_role", claims.Role)
		c.Locals("session_id", claims.SessionID)

		// Continue to next handler
		return c.Next()
	}
}

// OptionalAuthMiddleware creates a middleware that validates JWT but doesn't require it
// Useful for endpoints that work both authenticated and unauthenticated
func OptionalAuthMiddleware(authService *auth.Service) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Get token from Authorization header
		authHeader := c.Get("Authorization")

		log.Debug().
			Str("path", c.Path()).
			Bool("has_auth", authHeader != "").
			Msg("OptionalAuthMiddleware: Processing request")

		if authHeader == "" {
			// No token provided, continue without authentication
			log.Debug().Str("path", c.Path()).Msg("OptionalAuthMiddleware: No auth header, continuing")
			return c.Next()
		}

		// Extract token from "Bearer <token>" format
		token := authHeader
		if strings.HasPrefix(authHeader, "Bearer ") {
			token = strings.TrimPrefix(authHeader, "Bearer ")
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
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Unauthorized",
			})
		}

		// Check if user role is in allowed roles
		role := userRole.(string)
		for _, allowedRole := range allowedRoles {
			if role == allowedRole {
				return c.Next()
			}
		}

		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "Insufficient permissions",
		})
	}
}

// GetUserID is a helper to extract user ID from context
func GetUserID(c *fiber.Ctx) (string, bool) {
	userID := c.Locals("user_id")
	if userID == nil {
		return "", false
	}
	return userID.(string), true
}

// GetUserEmail is a helper to extract user email from context
func GetUserEmail(c *fiber.Ctx) (string, bool) {
	email := c.Locals("user_email")
	if email == nil {
		return "", false
	}
	return email.(string), true
}

// GetUserRole is a helper to extract user role from context
func GetUserRole(c *fiber.Ctx) (string, bool) {
	role := c.Locals("user_role")
	if role == nil {
		return "", false
	}
	return role.(string), true
}

// UnifiedAuthMiddleware creates a middleware that accepts both auth.users and dashboard.users authentication
// This allows both application users with admin role AND dashboard admins to access admin endpoints
func UnifiedAuthMiddleware(authService *auth.Service, jwtManager *auth.JWTManager) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Get token from Authorization header
		authHeader := c.Get("Authorization")
		if authHeader == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Missing authorization header",
			})
		}

		// Extract token from "Bearer <token>" format
		token := authHeader
		if strings.HasPrefix(authHeader, "Bearer ") {
			token = strings.TrimPrefix(authHeader, "Bearer ")
		}

		// First, try to validate as auth.users token
		claims, err := authService.ValidateToken(token)
		if err == nil {
			// Successfully validated as auth.users token
			// Check if token has been revoked
			isRevoked, err := authService.IsTokenRevoked(c.Context(), claims.ID)
			if err != nil {
				log.Error().Err(err).Msg("Failed to check token revocation status")
				// Continue anyway - revocation check failure shouldn't block valid tokens
			} else if isRevoked {
				log.Debug().Str("jti", claims.ID).Msg("Token has been revoked")
				return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
					"error": "Token has been revoked",
				})
			}

			// Store user information in context
			c.Locals("user_id", claims.UserID)
			c.Locals("user_email", claims.Email)
			c.Locals("user_role", claims.Role)
			c.Locals("session_id", claims.SessionID)

			log.Debug().
				Str("user_id", claims.UserID).
				Str("role", claims.Role).
				Msg("Authenticated as auth.users")

			return c.Next()
		}

		// If auth.users validation failed, try dashboard.users token
		dashboardClaims, err := jwtManager.ValidateAccessToken(token)
		if err != nil {
			// Both validations failed
			log.Debug().Err(err).Msg("Invalid token for both auth types")
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Invalid or expired token",
			})
		}

		// Successfully validated as dashboard.users token
		userID, err := uuid.Parse(dashboardClaims.Subject)
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Invalid user ID in token",
			})
		}

		// Store user information in context
		c.Locals("user_id", userID.String())
		c.Locals("user_email", dashboardClaims.Email)
		c.Locals("user_role", dashboardClaims.Role)

		log.Debug().
			Str("user_id", userID.String()).
			Str("role", dashboardClaims.Role).
			Msg("Authenticated as dashboard.users")

		return c.Next()
	}
}
