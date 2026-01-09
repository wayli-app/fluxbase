package middleware

import (
	"context"
	"strings"
	"time"

	"github.com/fluxbase-eu/fluxbase/internal/auth"
	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog/log"
	"golang.org/x/crypto/bcrypt"
)

// ClientKeyAuth creates middleware that authenticates requests using client keys
// Client key must be provided via X-Client-Key header (query parameter removed for security)
func ClientKeyAuth(clientKeyService *auth.ClientKeyService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Get client key from X-Client-Key header only (query parameter removed for security)
		clientKey := c.Get("X-Client-Key")

		// If no client key provided, return unauthorized
		if clientKey == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Missing client key. Provide via X-Client-Key header",
			})
		}

		// Validate the client key
		validatedKey, err := clientKeyService.ValidateClientKey(c.Context(), clientKey)
		if err != nil {
			log.Debug().Err(err).Msg("Invalid client key")

			// Return specific error messages
			switch err {
			case auth.ErrClientKeyRevoked:
				return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
					"error": "Client key has been revoked",
				})
			case auth.ErrClientKeyExpired:
				return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
					"error": "Client key has expired",
				})
			case auth.ErrUserClientKeysDisabled:
				return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
					"error": "User client keys are disabled. Contact an administrator.",
				})
			}

			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Invalid client key",
			})
		}

		// Store client key information in context
		c.Locals("client_key_id", validatedKey.ID)
		c.Locals("client_key_name", validatedKey.Name)
		c.Locals("client_key_scopes", validatedKey.Scopes)

		// Store allowed namespaces (nil = all allowed, empty = default only)
		if validatedKey.AllowedNamespaces != nil {
			c.Locals("allowed_namespaces", validatedKey.AllowedNamespaces)
		}

		// If client key is associated with a user, store user ID
		if validatedKey.UserID != nil {
			c.Locals("user_id", *validatedKey.UserID)
		}

		// Continue to next handler
		return c.Next()
	}
}

// OptionalClientKeyAuth allows both JWT and client key authentication
// Tries JWT first, then client key
func OptionalClientKeyAuth(authService *auth.Service, clientKeyService *auth.ClientKeyService) fiber.Handler {
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
					c.Locals("user_name", claims.Name)
					c.Locals("user_role", claims.Role)
					c.Locals("session_id", claims.SessionID)
					c.Locals("auth_type", "jwt")
					return c.Next()
				}
			}
		}

		// Try client key authentication (header only, query parameter removed for security)
		clientKey := c.Get("X-Client-Key")

		if clientKey != "" {
			validatedKey, err := clientKeyService.ValidateClientKey(c.Context(), clientKey)
			if err == nil {
				// Valid client key
				c.Locals("client_key_id", validatedKey.ID)
				c.Locals("client_key_name", validatedKey.Name)
				c.Locals("client_key_scopes", validatedKey.Scopes)
				c.Locals("auth_type", "clientkey")

				// Store allowed namespaces
				if validatedKey.AllowedNamespaces != nil {
					c.Locals("allowed_namespaces", validatedKey.AllowedNamespaces)
				}

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

// RequireEitherAuth requires either JWT or client key authentication
// This is the recommended middleware for protecting API endpoints
func RequireEitherAuth(authService *auth.Service, clientKeyService *auth.ClientKeyService) fiber.Handler {
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
					c.Locals("user_name", claims.Name)
					c.Locals("user_role", claims.Role)
					c.Locals("session_id", claims.SessionID)
					c.Locals("auth_type", "jwt")
					return c.Next()
				}
			}
		}

		// Try client key authentication (header only, query parameter removed for security)
		clientKey := c.Get("X-Client-Key")

		if clientKey != "" {
			validatedKey, err := clientKeyService.ValidateClientKey(c.Context(), clientKey)
			if err == nil {
				// Valid client key
				c.Locals("client_key_id", validatedKey.ID)
				c.Locals("client_key_name", validatedKey.Name)
				c.Locals("client_key_scopes", validatedKey.Scopes)
				// Store allowed namespaces
				if validatedKey.AllowedNamespaces != nil {
					c.Locals("allowed_namespaces", validatedKey.AllowedNamespaces)
				}

				c.Locals("auth_type", "clientkey")

				if validatedKey.UserID != nil {
					c.Locals("user_id", *validatedKey.UserID)
				}

				return c.Next()
			}

			// Return specific error for disabled user keys
			if err == auth.ErrUserClientKeysDisabled {
				return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
					"error": "User client keys are disabled. Contact an administrator.",
				})
			}
		}

		// No valid authentication provided
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Authentication required. Provide either a Bearer token or X-Client-Key header",
		})
	}
}

// RequireScope checks if the authenticated user/client key/service key has required scopes
func RequireScope(requiredScopes ...string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		authType := c.Locals("auth_type")

		// If authenticated via client key, check scopes
		if authType == "clientkey" {
			scopes, ok := c.Locals("client_key_scopes").([]string)
			if !ok {
				return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
					"error": "No scopes found for client key",
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

		// If authenticated via service key, check scopes
		if authType == "service_key" {
			scopes, ok := c.Locals("service_key_scopes").([]string)
			if !ok {
				return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
					"error": "No scopes found for service key",
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

// RequireAuthOrServiceKey requires either JWT, client key, OR service key authentication
// This is the most comprehensive auth middleware that accepts all authentication methods
func RequireAuthOrServiceKey(authService *auth.Service, clientKeyService *auth.ClientKeyService, db *pgxpool.Pool, jwtManager ...*auth.JWTManager) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Debug logging for service_role troubleshooting
		log.Debug().
			Str("path", c.Path()).
			Str("method", c.Method()).
			Bool("has_auth_header", c.Get("Authorization") != "").
			Bool("has_clientkey_header", c.Get("X-Client-Key") != "").
			Bool("has_service_key_header", c.Get("X-Service-Key") != "").
			Msg("RequireAuthOrServiceKey: Incoming request")

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
			if err != nil {
				log.Debug().
					Err(err).
					Msg("RequireAuthOrServiceKey: authService.ValidateToken failed")
			}
			if err == nil {
				// DEBUG: Log what we got from validation
				log.Debug().
					Str("role", claims.Role).
					Str("user_id", claims.UserID).
					Str("subject", claims.Subject).
					Msg("RequireAuthOrServiceKey: JWT validated, checking role")

				// Check if this is a dashboard admin token (dashboard.users)
				// Dashboard tokens use the same JWT secret but have role="dashboard_admin"
				// and store the user ID in Subject instead of UserID
				if claims.Role == "dashboard_admin" {
					log.Debug().
						Str("user_id", claims.Subject).
						Str("role", claims.Role).
						Msg("RequireAuthOrServiceKey: Detected dashboard_admin token")

					c.Locals("user_id", claims.Subject)
					c.Locals("user_email", claims.Email)
					c.Locals("user_name", claims.Name)
					c.Locals("user_role", claims.Role)
					c.Locals("auth_type", "jwt")
					c.Locals("is_anonymous", false)

					// Set RLS context for dashboard admin
					c.Locals("rls_user_id", claims.Subject)
					c.Locals("rls_role", claims.Role)

					return c.Next()
				}

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
					c.Locals("user_name", dashboardClaims.Name)
					c.Locals("user_role", dashboardClaims.Role)
					c.Locals("auth_type", "jwt")
					c.Locals("is_anonymous", false)

					// Set RLS context for dashboard admin
					c.Locals("rls_user_id", dashboardClaims.Subject)
					c.Locals("rls_role", dashboardClaims.Role)

					return c.Next()
				}
			}

			// User JWT and dashboard JWT validation failed, try service role JWT (anon/service_role)
			// This handles the Supabase pattern where JWTs have role claims instead of user claims
			if strings.HasPrefix(token, "eyJ") {
				claims, err := authService.ValidateServiceRoleToken(token)
				if err == nil {
					// Check if this is a service_role or anon token
					if claims.Role == "service_role" || claims.Role == "anon" {
						// Valid service role JWT - intentionally skip revocation check.
						// Service role tokens are system-level credentials that should never be blacklisted.
						c.Locals("user_role", claims.Role)
						c.Locals("auth_type", "service_role_jwt")
						c.Locals("jwt_claims", claims)
						c.Locals("rls_role", claims.Role)

						log.Debug().
							Str("role", claims.Role).
							Str("issuer", claims.Issuer).
							Msg("Authenticated with service role JWT via Bearer header")

						return c.Next()
					}
				}
			}

			// Bearer token was provided but invalid - return specific error
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Invalid or expired Bearer token",
			})
		}

		// Try client key authentication (header only, query parameter removed for security)
		clientKey := c.Get("X-Client-Key")

		if clientKey != "" {
			validatedKey, err := clientKeyService.ValidateClientKey(c.Context(), clientKey)
			if err == nil {
				// Valid client key
				c.Locals("client_key_id", validatedKey.ID)
				c.Locals("client_key_name", validatedKey.Name)
				c.Locals("client_key_scopes", validatedKey.Scopes)
				c.Locals("auth_type", "clientkey")
				// Store allowed namespaces
				if validatedKey.AllowedNamespaces != nil {
					c.Locals("allowed_namespaces", validatedKey.AllowedNamespaces)
				}

				if validatedKey.UserID != nil {
					c.Locals("user_id", *validatedKey.UserID)
					c.Locals("rls_user_id", *validatedKey.UserID)
					c.Locals("rls_role", "authenticated")
				}

				return c.Next()
			}
			// Client key was provided but invalid - return specific error
			if err == auth.ErrUserClientKeysDisabled {
				return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
					"error": "User client keys are disabled. Contact an administrator.",
				})
			}
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Invalid client key",
			})
		}

		// No authentication provided at all
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Authentication required. Provide Bearer token, X-Client-Key, or X-Service-Key",
		})
	}
}

// OptionalAuthOrServiceKey allows either JWT, client key, OR service key authentication
// If no authentication is provided, the request continues (for anonymous access with RLS)
// IMPORTANT: If invalid credentials are provided, returns 401 (does not fall back to anonymous)
//
// Supports Supabase-compatible authentication:
// - clientkey header containing a JWT with role claim (anon, service_role, authenticated)
// - Authorization: Bearer <jwt> with role claim
// - X-Service-Key header with hashed service key
// - Dashboard admin JWT tokens (when jwtManager is provided)
func OptionalAuthOrServiceKey(authService *auth.Service, clientKeyService *auth.ClientKeyService, db *pgxpool.Pool, jwtManager ...*auth.JWTManager) fiber.Handler {
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

		// Try JWT authentication via Authorization Bearer header or token query param
		// The token query param is used by WebSocket connections (browsers can't set headers)
		// Check user JWT first (most common case), then service role JWT
		token := ""
		if authHeader != "" && strings.HasPrefix(authHeader, "Bearer ") {
			token = strings.TrimPrefix(authHeader, "Bearer ")
		} else if queryToken := c.Query("token"); queryToken != "" {
			token = queryToken
		}

		if token != "" {

			// First, try to validate as a user JWT token (most common case)
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
					c.Locals("jwt_claims", claims)

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
					c.Locals("user_name", dashboardClaims.Name)
					c.Locals("user_role", dashboardClaims.Role)
					c.Locals("auth_type", "jwt")
					c.Locals("is_anonymous", false)
					c.Locals("jwt_claims", dashboardClaims)

					// Set RLS context for dashboard admin (maps to service_role in RLS middleware)
					c.Locals("rls_user_id", dashboardClaims.Subject)
					c.Locals("rls_role", dashboardClaims.Role)

					log.Debug().
						Str("user_id", dashboardClaims.Subject).
						Str("role", dashboardClaims.Role).
						Msg("Authenticated as dashboard.users via Bearer header")

					return c.Next()
				}
			}

			// User JWT validation failed, try service role JWT (anon/service_role)
			// This handles the Supabase pattern where the same JWT is sent as both clientkey and Bearer
			if strings.HasPrefix(token, "eyJ") {
				claims, err := authService.ValidateServiceRoleToken(token)
				if err == nil {
					// Check if this is a service_role or anon token (not a user token)
					if claims.Role == "service_role" || claims.Role == "anon" {
						// Valid service role JWT - intentionally skip revocation check.
						// Service role tokens are system-level credentials that should never be blacklisted.
						c.Locals("user_role", claims.Role)
						c.Locals("auth_type", "service_role_jwt")
						c.Locals("jwt_claims", claims)
						c.Locals("rls_role", claims.Role)

						log.Debug().
							Str("role", claims.Role).
							Str("issuer", claims.Issuer).
							Msg("Authenticated with service role JWT via Bearer header")

						return c.Next()
					}
				} else {
					// Both user JWT and service role JWT validation failed
					log.Debug().
						Err(err).
						Msg("Bearer token validation failed (tried user JWT then service role JWT)")
				}
			}

			// If Bearer token was provided but invalid, return 401
			// Don't fall back to anonymous access when invalid credentials are provided
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Invalid or expired Bearer token",
			})
		}

		// Check for Supabase-style clientkey header (lowercase)
		// This header may contain a JWT with role claim (anon, service_role, authenticated)
		fluxbaseClientKey := c.Get("clientkey")
		if fluxbaseClientKey != "" && strings.HasPrefix(fluxbaseClientKey, "eyJ") {
			// Looks like a JWT - first try user JWT (most common), then service role
			claims, err := authService.ValidateToken(fluxbaseClientKey)
			if err == nil {
				// Check if token has been revoked
				isRevoked, err := authService.IsTokenRevoked(c.Context(), claims.ID)
				if err == nil && !isRevoked {
					// Valid user JWT token via clientkey header
					c.Locals("user_id", claims.UserID)
					c.Locals("user_email", claims.Email)
					c.Locals("user_role", claims.Role)
					c.Locals("session_id", claims.SessionID)
					c.Locals("auth_type", "jwt")
					c.Locals("is_anonymous", claims.IsAnonymous)
					c.Locals("jwt_claims", claims)

					// Set RLS context
					c.Locals("rls_user_id", claims.UserID)
					c.Locals("rls_role", claims.Role)

					return c.Next()
				}
			}

			// User JWT failed, try service role JWT
			srClaims, err := authService.ValidateServiceRoleToken(fluxbaseClientKey)
			if err == nil {
				// Valid service role JWT
				c.Locals("user_role", srClaims.Role)
				c.Locals("auth_type", "service_role_jwt")
				c.Locals("jwt_claims", srClaims)

				// Set RLS context based on role claim
				c.Locals("rls_role", srClaims.Role)
				if srClaims.UserID != "" {
					c.Locals("user_id", srClaims.UserID)
					c.Locals("rls_user_id", srClaims.UserID)
				}

				log.Debug().
					Str("role", srClaims.Role).
					Str("issuer", srClaims.Issuer).
					Msg("Authenticated with service role JWT via clientkey header")

				return c.Next()
			}
			// If clientkey JWT was provided but invalid, log and fall through to try client key auth
			log.Debug().
				Err(err).
				Msg("clientkey header JWT validation failed (tried user JWT then service role JWT)")
		}

		// Try client key authentication (X-Client-Key header or clientkey query param)
		clientKey := c.Get("X-Client-Key")
		if clientKey == "" {
			clientKey = c.Query("clientkey")
		}
		// Also check lowercase clientkey header if it wasn't a JWT
		if clientKey == "" && fluxbaseClientKey != "" {
			clientKey = fluxbaseClientKey
		}

		if clientKey != "" {
			validatedKey, err := clientKeyService.ValidateClientKey(c.Context(), clientKey)
			if err == nil {
				// Valid client key
				c.Locals("client_key_id", validatedKey.ID)
				c.Locals("client_key_name", validatedKey.Name)
				c.Locals("client_key_scopes", validatedKey.Scopes)
				// Store allowed namespaces
				if validatedKey.AllowedNamespaces != nil {
					c.Locals("allowed_namespaces", validatedKey.AllowedNamespaces)
				}

				c.Locals("auth_type", "clientkey")

				// Set RLS context if client key has user association
				if validatedKey.UserID != nil {
					c.Locals("user_id", *validatedKey.UserID)
					c.Locals("rls_user_id", *validatedKey.UserID)
					c.Locals("rls_role", "authenticated")
				}

				return c.Next()
			}
			// If client key was provided but invalid, return specific error
			// Don't fall back to anonymous access when invalid credentials are provided
			if err == auth.ErrUserClientKeysDisabled {
				return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
					"error": "User client keys are disabled. Contact an administrator.",
				})
			}
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Invalid client key",
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
	var rateLimitPerMinute *int
	var rateLimitPerHour *int

	err := db.QueryRow(c.Context(),
		`SELECT id, name, key_hash, scopes, enabled, expires_at,
		        rate_limit_per_minute, rate_limit_per_hour
		 FROM auth.service_keys
		 WHERE key_prefix = $1`,
		keyPrefix,
	).Scan(&keyID, &keyName, &keyHash, &scopes, &enabled, &expiresAt,
		&rateLimitPerMinute, &rateLimitPerHour)

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

	// Store rate limits in context (nil means unlimited)
	c.Locals("service_key_rate_limit_per_minute", rateLimitPerMinute)
	c.Locals("service_key_rate_limit_per_hour", rateLimitPerHour)

	// For RLS context
	c.Locals("rls_role", "service_role")
	c.Locals("rls_user_id", nil) // Service keys don't have user IDs

	log.Debug().
		Str("key_id", keyID).
		Str("key_name", keyName).
		Interface("rate_limit_per_minute", rateLimitPerMinute).
		Interface("rate_limit_per_hour", rateLimitPerHour).
		Msg("Authenticated with service key")

	return true
}

// RequireAdmin middleware restricts access to admin users only
// Allows: service_role (from service keys or service_role JWT) and dashboard_admin users
// This should be used after authentication middleware (RequireAuthOrServiceKey)
func RequireAdmin() fiber.Handler {
	return func(c *fiber.Ctx) error {
		authType, _ := c.Locals("auth_type").(string)
		role, _ := c.Locals("user_role").(string)

		// Service keys always have service_role
		if authType == "service_key" {
			log.Debug().
				Str("auth_type", authType).
				Msg("Admin access granted - service key")
			return c.Next()
		}

		// Check for admin roles
		if role == "service_role" || role == "dashboard_admin" {
			log.Debug().
				Str("auth_type", authType).
				Str("role", role).
				Msg("Admin access granted")
			return c.Next()
		}

		log.Warn().
			Str("auth_type", authType).
			Str("role", role).
			Str("path", c.Path()).
			Msg("Admin access denied - requires service_role or dashboard_admin")

		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "Admin access required. Only service_role and dashboard_admin can access this endpoint.",
		})
	}
}

// RequireAdminIfClientKeysDisabled middleware conditionally requires admin access
// when the 'app.auth.allow_user_client_keys' setting is disabled.
// If the setting is enabled (default), allows regular users through.
// If the setting is disabled, requires admin access (service_role or dashboard_admin).
func RequireAdminIfClientKeysDisabled(settingsCache *auth.SettingsCache) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Check if user client keys are allowed
		allowUserKeys := settingsCache.GetBool(c.Context(), "app.auth.allow_user_client_keys", true)

		if allowUserKeys {
			// Setting is enabled - allow regular users to manage their own keys
			return c.Next()
		}

		// Setting is disabled - require admin access
		return RequireAdmin()(c)
	}
}
