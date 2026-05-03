package api

import (
	"context"
	"strings"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"

	"github.com/nimbleflux/fluxbase/internal/auth"
	"github.com/nimbleflux/fluxbase/internal/config"
	"github.com/nimbleflux/fluxbase/internal/database"
)

// getTenantJWTSecret extracts the tenant-specific JWT secret from context if available
// Returns empty string if no tenant config or no tenant-specific secret
func getTenantJWTSecret(c fiber.Ctx) string {
	tenantConfig, ok := c.Locals("tenant_config").(*config.Config)
	if !ok || tenantConfig == nil {
		return ""
	}
	// Return the tenant's JWT secret if it differs from empty (meaning it was overridden)
	if tenantConfig.Auth.JWTSecret != "" {
		return tenantConfig.Auth.JWTSecret
	}
	return ""
}

// AuthMiddleware creates a middleware for JWT authentication
func AuthMiddleware(authService *auth.Service) fiber.Handler {
	return func(c fiber.Ctx) error {
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

		// Validate token - use tenant-specific secret if available
		var claims *auth.TokenClaims
		var err error
		tenantSecret := getTenantJWTSecret(c)
		if tenantSecret != "" {
			claims, err = authService.ValidateTokenWithSecret(token, tenantSecret)
		} else {
			claims, err = authService.ValidateToken(token)
		}
		if err != nil {
			log.Debug().Err(err).Msg("Invalid token")
			return SendInvalidToken(c)
		}

		// Check if token has been revoked
		isRevoked, err := authService.IsTokenRevoked(c.RequestCtx(), claims.ID)
		if err != nil {
			// SECURITY: Fail-closed for sensitive operations
			// If we cannot verify token revocation status, deny access to sensitive operations
			if isSensitiveOperation(c) {
				log.Error().Err(err).Str("jti", claims.ID).Str("method", c.Method()).Str("path", c.Path()).Msg("Failed to check token revocation status for sensitive operation - denying access")
				return SendUnauthorized(c, "Unable to verify token status", ErrCodeInvalidToken)
			}
			// For non-sensitive operations, continue (fail-open)
			log.Error().Err(err).Msg("Failed to check token revocation status")
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
	return func(c fiber.Ctx) error {
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
		isRevoked, err := authService.IsTokenRevoked(c.RequestCtx(), claims.ID)
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
	tenantAdminAllowed := false
	for _, r := range allowedRoles {
		if r == "tenant_admin" {
			tenantAdminAllowed = true
			break
		}
	}

	return func(c fiber.Ctx) error {
		// Service key auth: bypass role check for global keys.
		// Tenant-scoped keys (tenant_service) bypass on tenant-scoped routes
		// and check the explicit allowedRoles list for other routes.
		if authType, _ := c.Locals("auth_type").(string); authType == "service_key" {
			keyType, _ := c.Locals("service_key_type").(string)
			if keyType != "tenant_service" {
				return c.Next()
			}
			if tenantAdminAllowed {
				return c.Next()
			}
			// Fall through to allowedRoles check for tenant_service service keys
		}

		userRole := c.Locals("user_role")
		if userRole == nil {
			return SendUnauthorized(c, "Unauthorized", ErrCodeAuthRequired)
		}

		// Check if user role is in allowed roles (with safe type assertion)
		role, ok := userRole.(string)
		if !ok {
			return SendUnauthorized(c, "Invalid role type", ErrCodeInvalidRole)
		}

		// service_role JWT bypasses role checks — full access by design.
		// This handles CLI clients that authenticate via service_role JWT.
		if role == "service_role" {
			return c.Next()
		}

		// tenant_service JWT: same logic as tenant_service service keys —
		// only bypass on tenant-scoped routes.
		if role == "tenant_service" {
			if tenantAdminAllowed {
				return c.Next()
			}
		}

		for _, allowedRole := range allowedRoles {
			if role == allowedRole {
				return c.Next()
			}
		}

		return SendInsufficientPermissions(c)
	}
}

// GetUserEmail is a helper to extract user email from context
func GetUserEmail(c fiber.Ctx) (string, bool) {
	email := c.Locals("user_email")
	if email == nil {
		return "", false
	}
	e, ok := email.(string)
	return e, ok
}

// GetUserRole is a helper to extract user role from context
func GetUserRole(c fiber.Ctx) (string, bool) {
	role := c.Locals("user_role")
	if role == nil {
		return "", false
	}
	r, ok := role.(string)
	return r, ok
}

// UnifiedAuthMiddleware creates a middleware that accepts both auth.users and platform.users authentication
// This allows both application users with admin role AND dashboard admins to access admin endpoints.
// The db parameter is used to check the actual role from auth.users when JWT role is "authenticated",
// allowing role changes to take effect immediately without requiring re-login.
func UnifiedAuthMiddleware(authService *auth.Service, jwtManager *auth.JWTManager, db *database.Connection) fiber.Handler {
	return func(c fiber.Ctx) error {
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
			// Check if this is a platform admin token (platform.users)
			// Platform tokens use the same JWT secret but have role="instance_admin"
			// and store the user ID in Subject instead of UserID
			if claims.Role == "instance_admin" {
				c.Locals("user_id", claims.Subject)
				c.Locals("user_email", claims.Email)
				c.Locals("user_role", claims.Role)
				c.Locals("is_instance_admin", true)
				c.Locals("jwt_claims", claims)

				log.Debug().
					Str("user_id", claims.Subject).
					Str("role", claims.Role).
					Msg("Authenticated as platform.users via role check")

				return c.Next()
			}

			// Successfully validated as auth.users token
			// Check if token has been revoked
			isRevoked, err := authService.IsTokenRevoked(c.RequestCtx(), claims.ID)
			if err != nil {
				// SECURITY: Fail-closed for sensitive operations
				// If we cannot verify token revocation status, deny access to sensitive operations
				if isSensitiveOperation(c) {
					log.Error().Err(err).Str("jti", claims.ID).Str("method", c.Method()).Str("path", c.Path()).Msg("Failed to check token revocation status for sensitive operation - denying access")
					return SendUnauthorized(c, "Unable to verify token status", ErrCodeInvalidToken)
				}
				// For non-sensitive operations, continue (fail-open)
				log.Error().Err(err).Msg("Failed to check token revocation status")
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
				dbRole, err := getUserRoleFromDB(c.RequestCtx(), db, claims.UserID)
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

		// If auth.users validation failed, try platform.users token
		dashboardClaims, err := jwtManager.ValidateAccessToken(token)
		if err != nil {
			// Both validations failed
			log.Debug().Err(err).Msg("Invalid token for both auth types")
			return SendInvalidToken(c)
		}

		// Successfully validated as platform.users token
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
			Msg("Authenticated as platform.users")

		return c.Next()
	}
}

// getUserRoleFromDB fetches user's role from auth.users table.
// Also checks app_metadata.role as fallback.
// This allows role changes to take effect immediately without re-login.
func getUserRoleFromDB(ctx context.Context, db *database.Connection, userID string) (string, error) {
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

// isSensitiveOperation determines if an HTTP request is a sensitive operation that should
// fail-closed when token revocation status cannot be verified. This provides defense-in-depth
// by ensuring that potentially dangerous operations cannot proceed if we cannot verify the
// token's validity.
//
// Sensitive operations include:
// - DELETE requests (data destruction)
// - PATCH requests (data modification)
// - POST requests to /graphql ( GraphQL mutations can modify data)
// - POST requests to /rest/* (REST mutations)
func isSensitiveOperation(c fiber.Ctx) bool {
	method := c.Method()
	path := c.Path()

	// All DELETE and PATCH operations are sensitive
	if method == "DELETE" || method == "PATCH" {
		return true
	}

	// POST requests can be sensitive depending on the endpoint
	if method == "POST" {
		// GraphQL mutations use POST
		if strings.HasPrefix(path, "/graphql") {
			return true
		}
		// REST CRUD operations use POST
		if strings.HasPrefix(path, "/rest/") {
			return true
		}
	}

	return false
}

// fiber:context-methods migrated
