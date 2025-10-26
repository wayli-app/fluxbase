package api

import (
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/wayli-app/fluxbase/internal/auth"
	"github.com/rs/zerolog/log"
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
		if authHeader == "" {
			// No token provided, continue without authentication
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
			log.Debug().Err(err).Msg("Invalid token in optional auth")
			return c.Next()
		}

		// Store user information in context
		c.Locals("user_id", claims.UserID)
		c.Locals("user_email", claims.Email)
		c.Locals("user_role", claims.Role)
		c.Locals("session_id", claims.SessionID)

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
