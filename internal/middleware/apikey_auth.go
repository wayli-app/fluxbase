package middleware

import (
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog/log"
	"github.com/wayli-app/fluxbase/internal/auth"
)

// APIKeyAuth creates middleware that authenticates requests using API keys
// Checks for API key in X-API-Key header or apikey query parameter
func APIKeyAuth(apiKeyService *auth.APIKeyService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Try to get API key from X-API-Key header first
		apiKey := c.Get("X-API-Key")

		// If not in header, try query parameter (less secure, but convenient for testing)
		if apiKey == "" {
			apiKey = c.Query("apikey")
		}

		// If no API key provided, return unauthorized
		if apiKey == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Missing API key. Provide via X-API-Key header or apikey query parameter",
			})
		}

		// Validate the API key
		validatedKey, err := apiKeyService.ValidateAPIKey(c.Context(), apiKey)
		if err != nil {
			log.Debug().Err(err).Msg("Invalid API key")

			// Return specific error messages
			if err == auth.ErrAPIKeyRevoked {
				return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
					"error": "API key has been revoked",
				})
			} else if err == auth.ErrAPIKeyExpired {
				return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
					"error": "API key has expired",
				})
			}

			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Invalid API key",
			})
		}

		// Store API key information in context
		c.Locals("api_key_id", validatedKey.ID)
		c.Locals("api_key_name", validatedKey.Name)
		c.Locals("api_key_scopes", validatedKey.Scopes)

		// If API key is associated with a user, store user ID
		if validatedKey.UserID != nil {
			c.Locals("user_id", *validatedKey.UserID)
		}

		// Continue to next handler
		return c.Next()
	}
}

// OptionalAPIKeyAuth allows both JWT and API key authentication
// Tries JWT first, then API key
func OptionalAPIKeyAuth(authService *auth.Service, apiKeyService *auth.APIKeyService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Try JWT authentication first
		authHeader := c.Get("Authorization")
		if authHeader != "" && strings.HasPrefix(authHeader, "Bearer ") {
			token := strings.TrimPrefix(authHeader, "Bearer ")

			// Validate JWT token
			claims, err := authService.ValidateToken(token)
			if err == nil {
				// Check if token has been revoked
				isRevoked, err := authService.IsTokenRevoked(c.Context(), claims.ID)
				if err == nil && !isRevoked {
					// Valid JWT token
					c.Locals("user_id", claims.UserID)
					c.Locals("user_email", claims.Email)
					c.Locals("user_role", claims.Role)
					c.Locals("session_id", claims.SessionID)
					c.Locals("auth_type", "jwt")
					return c.Next()
				}
			}
		}

		// Try API key authentication
		apiKey := c.Get("X-API-Key")
		if apiKey == "" {
			apiKey = c.Query("apikey")
		}

		if apiKey != "" {
			validatedKey, err := apiKeyService.ValidateAPIKey(c.Context(), apiKey)
			if err == nil {
				// Valid API key
				c.Locals("api_key_id", validatedKey.ID)
				c.Locals("api_key_name", validatedKey.Name)
				c.Locals("api_key_scopes", validatedKey.Scopes)
				c.Locals("auth_type", "apikey")

				if validatedKey.UserID != nil {
					c.Locals("user_id", *validatedKey.UserID)
				}

				return c.Next()
			}
		}

		// No valid authentication provided, continue anyway (optional auth)
		return c.Next()
	}
}

// RequireEitherAuth requires either JWT or API key authentication
// This is the recommended middleware for protecting API endpoints
func RequireEitherAuth(authService *auth.Service, apiKeyService *auth.APIKeyService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Try JWT authentication first
		authHeader := c.Get("Authorization")
		if authHeader != "" && strings.HasPrefix(authHeader, "Bearer ") {
			token := strings.TrimPrefix(authHeader, "Bearer ")

			// Validate JWT token
			claims, err := authService.ValidateToken(token)
			if err == nil {
				// Check if token has been revoked
				isRevoked, err := authService.IsTokenRevoked(c.Context(), claims.ID)
				if err == nil && !isRevoked {
					// Valid JWT token
					c.Locals("user_id", claims.UserID)
					c.Locals("user_email", claims.Email)
					c.Locals("user_role", claims.Role)
					c.Locals("session_id", claims.SessionID)
					c.Locals("auth_type", "jwt")
					return c.Next()
				}
			}
		}

		// Try API key authentication
		apiKey := c.Get("X-API-Key")
		if apiKey == "" {
			apiKey = c.Query("apikey")
		}

		if apiKey != "" {
			validatedKey, err := apiKeyService.ValidateAPIKey(c.Context(), apiKey)
			if err == nil {
				// Valid API key
				c.Locals("api_key_id", validatedKey.ID)
				c.Locals("api_key_name", validatedKey.Name)
				c.Locals("api_key_scopes", validatedKey.Scopes)
				c.Locals("auth_type", "apikey")

				if validatedKey.UserID != nil {
					c.Locals("user_id", *validatedKey.UserID)
				}

				return c.Next()
			}
		}

		// No valid authentication provided
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Authentication required. Provide either a Bearer token or X-API-Key header",
		})
	}
}

// RequireScope checks if the authenticated user/API key has required scopes
func RequireScope(requiredScopes ...string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		authType := c.Locals("auth_type")

		// If authenticated via API key, check scopes
		if authType == "apikey" {
			scopes, ok := c.Locals("api_key_scopes").([]string)
			if !ok {
				return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
					"error": "No scopes found for API key",
				})
			}

			// Check if all required scopes are present
			for _, required := range requiredScopes {
				found := false
				for _, scope := range scopes {
					if scope == required || scope == "*" {
						found = true
						break
					}
				}
				if !found {
					return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
						"error":          "Insufficient permissions",
						"required_scope": required,
					})
				}
			}
		}

		// JWT auth doesn't use scopes yet, so just allow
		// (could be extended in the future to check user roles)

		return c.Next()
	}
}
