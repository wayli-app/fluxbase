package middleware

import (
	"context"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog/log"
	"github.com/wayli-app/fluxbase/internal/auth"
	"golang.org/x/crypto/bcrypt"
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

// RequireAuthOrServiceKey requires either JWT, API key, OR service key authentication
// This is the most comprehensive auth middleware that accepts all authentication methods
func RequireAuthOrServiceKey(authService *auth.Service, apiKeyService *auth.APIKeyService, db *pgxpool.Pool, jwtManager ...*auth.JWTManager) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// First, try service key authentication (highest privilege)
		serviceKey := c.Get("X-Service-Key")
		authHeader := c.Get("Authorization")

		if serviceKey == "" && strings.HasPrefix(authHeader, "ServiceKey ") {
			serviceKey = strings.TrimPrefix(authHeader, "ServiceKey ")
		}

		if serviceKey != "" {
			// Validate service key
			if validateServiceKey(c, db, serviceKey) {
				return c.Next()
			}
			// If service key validation failed, don't try other methods
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Invalid service key",
			})
		}

		// Try JWT authentication
		if authHeader != "" && strings.HasPrefix(authHeader, "Bearer ") {
			token := strings.TrimPrefix(authHeader, "Bearer ")

			// First, try to validate as auth.users token (app users)
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
					c.Locals("is_anonymous", claims.IsAnonymous)

					// Set RLS context
					c.Locals("rls_user_id", claims.UserID)
					c.Locals("rls_role", claims.Role)

					return c.Next()
				}
			}

			// If auth.users validation failed and jwtManager is provided, try dashboard.users token
			if len(jwtManager) > 0 && jwtManager[0] != nil {
				dashboardClaims, err := jwtManager[0].ValidateAccessToken(token)
				if err == nil {
					// Successfully validated as dashboard.users token
					c.Locals("user_id", dashboardClaims.Subject)
					c.Locals("user_email", dashboardClaims.Email)
					c.Locals("user_role", dashboardClaims.Role)
					c.Locals("auth_type", "jwt")
					c.Locals("is_anonymous", false)

					// Set RLS context for dashboard admin
					c.Locals("rls_user_id", dashboardClaims.Subject)
					c.Locals("rls_role", dashboardClaims.Role)

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
					c.Locals("rls_user_id", *validatedKey.UserID)
					c.Locals("rls_role", "authenticated")
				}

				return c.Next()
			}
		}

		// No valid authentication provided
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Authentication required. Provide Bearer token, X-API-Key, or X-Service-Key",
		})
	}
}

// OptionalAuthOrServiceKey allows either JWT, API key, OR service key authentication
// If no authentication is provided, the request continues (for anonymous access with RLS)
// IMPORTANT: If invalid credentials are provided, returns 401 (does not fall back to anonymous)
func OptionalAuthOrServiceKey(authService *auth.Service, apiKeyService *auth.APIKeyService, db *pgxpool.Pool) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// First, try service key authentication (highest privilege)
		serviceKey := c.Get("X-Service-Key")
		authHeader := c.Get("Authorization")

		if serviceKey == "" && strings.HasPrefix(authHeader, "ServiceKey ") {
			serviceKey = strings.TrimPrefix(authHeader, "ServiceKey ")
		}

		if serviceKey != "" {
			// Validate service key
			if validateServiceKey(c, db, serviceKey) {
				return c.Next()
			}
			// If service key validation failed, don't try other methods
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Invalid service key",
			})
		}

		// Try JWT authentication
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
					c.Locals("is_anonymous", claims.IsAnonymous)

					// Set RLS context
					c.Locals("rls_user_id", claims.UserID)
					c.Locals("rls_role", claims.Role)

					return c.Next()
				}
			}
			// If Bearer token was provided but invalid, return 401
			// Don't fall back to anonymous access when invalid credentials are provided
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Invalid or expired Bearer token",
			})
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

				// Set RLS context if API key has user association
				if validatedKey.UserID != nil {
					c.Locals("user_id", *validatedKey.UserID)
					c.Locals("rls_user_id", *validatedKey.UserID)
					c.Locals("rls_role", "authenticated")
				}

				return c.Next()
			}
			// If API key was provided but invalid, return 401
			// Don't fall back to anonymous access when invalid credentials are provided
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Invalid API key",
			})
		}

		// No authentication provided - allow anonymous access with RLS
		// The RLS middleware will set role to 'anon' if no auth is present
		return c.Next()
	}
}

// validateServiceKey validates a service key and sets context if valid
// Returns true if valid, false otherwise
func validateServiceKey(c *fiber.Ctx, db *pgxpool.Pool, serviceKey string) bool {
	// Extract key prefix (first 16 chars for identification)
	// This includes "sk_test_" (8 chars) plus some random chars to ensure uniqueness
	if len(serviceKey) < 16 || !strings.HasPrefix(serviceKey, "sk_") {
		return false
	}
	keyPrefix := serviceKey[:16]

	// Look up service key in database by prefix
	var keyHash string
	var keyID string
	var keyName string
	var scopes []string
	var enabled bool
	var expiresAt *time.Time

	err := db.QueryRow(c.Context(),
		`SELECT id, name, key_hash, scopes, enabled, expires_at
		 FROM auth.service_keys
		 WHERE key_prefix = $1`,
		keyPrefix,
	).Scan(&keyID, &keyName, &keyHash, &scopes, &enabled, &expiresAt)

	if err != nil {
		log.Debug().Err(err).Str("prefix", keyPrefix).Msg("Service key not found")
		return false
	}

	// Check if key is enabled
	if !enabled {
		log.Debug().Str("key_id", keyID).Msg("Service key is disabled")
		return false
	}

	// Check if key has expired
	if expiresAt != nil && expiresAt.Before(time.Now()) {
		log.Debug().Str("key_id", keyID).Msg("Service key has expired")
		return false
	}

	// Verify the key hash
	err = bcrypt.CompareHashAndPassword([]byte(keyHash), []byte(serviceKey))
	if err != nil {
		log.Debug().Err(err).Str("prefix", keyPrefix).Msg("Invalid service key hash")
		return false
	}

	// Update last_used_at timestamp (fire and forget)
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_, _ = db.Exec(ctx,
			`UPDATE auth.service_keys SET last_used_at = NOW() WHERE id = $1`,
			keyID,
		)
	}()

	// Store service key information in context
	c.Locals("service_key_id", keyID)
	c.Locals("service_key_name", keyName)
	c.Locals("service_key_scopes", scopes)
	c.Locals("auth_type", "service_key")
	c.Locals("user_role", "service_role") // Elevated role

	// For RLS context
	c.Locals("rls_role", "service_role")
	c.Locals("rls_user_id", nil) // Service keys don't have user IDs

	log.Debug().
		Str("key_id", keyID).
		Str("key_name", keyName).
		Msg("Authenticated with service key")

	return true
}
