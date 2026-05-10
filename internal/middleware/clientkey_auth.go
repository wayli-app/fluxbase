package middleware

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog/log"
	"golang.org/x/crypto/bcrypt"

	"github.com/nimbleflux/fluxbase/internal/auth"
	"github.com/nimbleflux/fluxbase/internal/config"
	"github.com/nimbleflux/fluxbase/internal/keys"
)

// ClientKeyAuth creates middleware that authenticates requests using client keys
// Client key must be provided via X-Client-Key header (query parameter removed for security)
func ClientKeyAuth(clientKeyService *auth.ClientKeyService) fiber.Handler {
	return func(c fiber.Ctx) error {
		// Get client key from X-Client-Key header only (query parameter removed for security)
		clientKey := c.Get("X-Client-Key")

		// If no client key provided, return unauthorized
		if clientKey == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Missing client key. Provide via X-Client-Key header",
			})
		}

		// Validate the client key
		validatedKey, err := clientKeyService.ValidateClientKey(c.RequestCtx(), clientKey)
		if err != nil {
			log.Debug().Err(err).Msg("Invalid client key")

			// Return specific error messages using errors.Is for proper error wrapping
			if errors.Is(err, auth.ErrClientKeyRevoked) {
				return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
					"error": "Client key has been revoked",
				})
			}
			if errors.Is(err, auth.ErrClientKeyExpired) {
				return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
					"error": "Client key has been expired",
				})
			}
			if errors.Is(err, auth.ErrUserClientKeysDisabled) {
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
	return func(c fiber.Ctx) error {
		// Try JWT authentication first
		authHeader := c.Get("Authorization")
		if authHeader != "" && strings.HasPrefix(authHeader, "Bearer ") {
			token := strings.TrimPrefix(authHeader, "Bearer ")

			// Validate JWT token
			claims, err := authService.ValidateToken(token)
			if err == nil {
				// Check if token has been revoked
				// SECURITY: Fail-closed behavior - reject if we can't verify revocation status
				isRevoked, err := authService.IsTokenRevokedWithClaims(c.RequestCtx(), claims.ID, claims.UserID, claims.IssuedAt.Time)
				if err != nil {
					log.Error().Err(err).Str("jti", claims.ID).Msg("Token revocation check failed")
					return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{
						"error":   "service_unavailable",
						"message": "Unable to verify token status",
					})
				}
				if isRevoked {
					return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
						"error":   "token_revoked",
						"message": "Token has been revoked",
					})
				}

				// Valid JWT token
				c.Locals("user_id", claims.UserID)
				c.Locals("user_email", claims.Email)
				c.Locals("user_name", claims.Name)
				c.Locals("user_role", claims.Role)
				c.Locals("session_id", claims.SessionID)
				c.Locals("auth_type", "jwt")
				c.Locals("claims", claims)
				c.Locals("jwt_claims", claims)
				return c.Next()
			}
		}

		// Try client key authentication (header only, query parameter removed for security)
		clientKey := c.Get("X-Client-Key")

		if clientKey != "" {
			validatedKey, err := clientKeyService.ValidateClientKey(c.RequestCtx(), clientKey)
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
	return func(c fiber.Ctx) error {
		// Try JWT authentication first
		authHeader := c.Get("Authorization")
		if authHeader != "" && strings.HasPrefix(authHeader, "Bearer ") {
			token := strings.TrimPrefix(authHeader, "Bearer ")

			// Validate JWT token
			claims, err := authService.ValidateToken(token)
			if err == nil {
				// Check if token has been revoked
				// SECURITY: Fail-closed behavior - reject if we can't verify revocation status
				isRevoked, err := authService.IsTokenRevokedWithClaims(c.RequestCtx(), claims.ID, claims.UserID, claims.IssuedAt.Time)
				if err != nil {
					log.Error().Err(err).Str("jti", claims.ID).Msg("Token revocation check failed")
					return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{
						"error":   "service_unavailable",
						"message": "Unable to verify token status",
					})
				}
				if isRevoked {
					return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
						"error":   "token_revoked",
						"message": "Token has been revoked",
					})
				}

				// Valid JWT token
				c.Locals("user_id", claims.UserID)
				c.Locals("user_email", claims.Email)
				c.Locals("user_name", claims.Name)
				c.Locals("user_role", claims.Role)
				c.Locals("session_id", claims.SessionID)
				c.Locals("auth_type", "jwt")
				c.Locals("claims", claims)
				c.Locals("jwt_claims", claims)
				return c.Next()
			}
		}

		// Try client key authentication (header only, query parameter removed for security)
		clientKey := c.Get("X-Client-Key")

		if clientKey != "" {
			validatedKey, err := clientKeyService.ValidateClientKey(c.RequestCtx(), clientKey)
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
			if errors.Is(err, auth.ErrUserClientKeysDisabled) {
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
	return func(c fiber.Ctx) error {
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

		// JWT auth bypasses scope checking — user authorization is handled by RLS

		return c.Next()
	}
}

// RequireAuthOrServiceKey requires either JWT, client key, OR service key authentication
// This is the most comprehensive auth middleware that accepts all authentication methods
func RequireAuthOrServiceKey(authService *auth.Service, clientKeyService *auth.ClientKeyService, db *pgxpool.Pool, securityCfg *config.SecurityConfig, jwtManager ...*auth.JWTManager) fiber.Handler {
	return authOrServiceKey(authService, clientKeyService, db, securityCfg, true, jwtManager...)
}

// OptionalAuthOrServiceKey allows either JWT, client key, OR service key authentication
// If no authentication is provided, the request continues (for anonymous access with RLS)
// IMPORTANT: If invalid credentials are provided, returns 401 (does not fall back to anonymous)
//
// Supports authentication via:
// - clientkey header containing a JWT with role claim (anon, service_role, authenticated)
// - Authorization: Bearer <jwt> with role claim
// - X-Service-Key header with hashed service key or service role JWT
// - Dashboard admin JWT tokens (when jwtManager is provided)
func OptionalAuthOrServiceKey(authService *auth.Service, clientKeyService *auth.ClientKeyService, db *pgxpool.Pool, securityCfg *config.SecurityConfig, jwtManager ...*auth.JWTManager) fiber.Handler {
	return authOrServiceKey(authService, clientKeyService, db, securityCfg, false, jwtManager...)
}

func authOrServiceKey(
	authService *auth.Service,
	clientKeyService *auth.ClientKeyService,
	db *pgxpool.Pool,
	securityCfg *config.SecurityConfig,
	required bool,
	jwtManager ...*auth.JWTManager,
) fiber.Handler {
	return func(c fiber.Ctx) error {
		if required {
			log.Debug().
				Str("path", c.Path()).
				Str("method", c.Method()).
				Bool("has_auth_header", c.Get("Authorization") != "").
				Bool("has_clientkey_header", c.Get("X-Client-Key") != "").
				Bool("has_service_key_header", c.Get("X-Service-Key") != "").
				Msg("RequireAuthOrServiceKey: Incoming request")
		}

		// 1. Service key authentication (highest privilege)
		serviceKey := c.Get("X-Service-Key")
		authHeader := c.Get("Authorization")

		if serviceKey == "" && strings.HasPrefix(authHeader, "ServiceKey ") {
			serviceKey = strings.TrimPrefix(authHeader, "ServiceKey ")
		}

		if serviceKey != "" {
			if strings.HasPrefix(serviceKey, "eyJ") {
				claims, err := authService.ValidateServiceRoleToken(serviceKey)
				if err == nil {
					c.Locals("user_role", claims.Role)
					c.Locals("auth_type", "service_role_jwt")
					c.Locals("jwt_claims", claims)
					c.Locals("rls_role", claims.Role)
					c.Locals("claims", claims)

					log.Debug().
						Str("role", claims.Role).
						Str("issuer", claims.Issuer).
						Msg("Authenticated with service role JWT via X-Service-Key header")

					return c.Next()
				}
				log.Debug().Err(err).Msg("X-Service-Key JWT validation failed")
				return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
					"error": "Invalid service role token",
				})
			}

			if validateServiceKey(c, db, serviceKey) {
				return c.Next()
			}
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Invalid service key",
			})
		}

		// 2. JWT authentication via Authorization Bearer header or token query param
		// The token query param is used by WebSocket connections (browsers can't set headers)
		token := ""
		if authHeader != "" && strings.HasPrefix(authHeader, "Bearer ") {
			token = strings.TrimPrefix(authHeader, "Bearer ")
		} else if queryToken := c.Query("token"); queryToken != "" {
			token = queryToken
		}

		if token != "" {
			// First, try to validate as auth.users token (app users)
			claims, err := authService.ValidateToken(token)
			if err != nil {
				log.Debug().
					Err(err).
					Msg("authOrServiceKey: authService.ValidateToken failed")
			}
			if err == nil {
				log.Debug().
					Str("role", claims.Role).
					Str("user_id", claims.UserID).
					Str("subject", claims.Subject).
					Msg("authOrServiceKey: JWT validated, checking role")

				// Check if this is a platform admin token (platform.users)
				// Platform tokens use the same JWT secret but have role="instance_admin"
				// and store the user ID in Subject instead of UserID
				if claims.Role == "instance_admin" {
					log.Debug().
						Str("user_id", claims.Subject).
						Str("role", claims.Role).
						Msg("authOrServiceKey: Detected instance_admin token")

					c.Locals("user_id", claims.Subject)
					c.Locals("user_email", claims.Email)
					c.Locals("user_name", claims.Name)
					c.Locals("user_role", claims.Role)
					c.Locals("auth_type", "jwt")
					c.Locals("is_anonymous", false)
					c.Locals("is_instance_admin", true)

					c.Locals("rls_user_id", claims.Subject)
					c.Locals("rls_role", claims.Role)
					c.Locals("claims", claims)
					c.Locals("jwt_claims", claims)

					return c.Next()
				}

				// Check if token has been revoked
				// SECURITY: Fail-closed behavior - reject if we can't verify revocation status
				isRevoked, err := authService.IsTokenRevokedWithClaims(c.RequestCtx(), claims.ID, claims.UserID, claims.IssuedAt.Time)
				if err != nil {
					log.Error().Err(err).Str("jti", claims.ID).Msg("Token revocation check failed")
					return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{
						"error":   "service_unavailable",
						"message": "Unable to verify token status",
					})
				}
				if isRevoked {
					return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
						"error":   "token_revoked",
						"message": "Token has been revoked",
					})
				}

				// Valid JWT token
				c.Locals("user_id", claims.UserID)
				c.Locals("user_email", claims.Email)
				c.Locals("user_role", claims.Role)
				c.Locals("session_id", claims.SessionID)
				c.Locals("auth_type", "jwt")
				c.Locals("is_anonymous", claims.IsAnonymous)
				c.Locals("jwt_claims", claims)

				c.Locals("rls_user_id", claims.UserID)
				c.Locals("rls_role", claims.Role)
				c.Locals("claims", claims)

				// SECURITY: Log audit entry for impersonation tokens
				// Impersonation tokens have an impersonated_by claim indicating the admin who issued them
				if claims.ImpersonatedBy != "" {
					log.Info().
						Str("user_id", claims.UserID).
						Str("impersonated_by", claims.ImpersonatedBy).
						Str("path", c.Path()).
						Str("method", c.Method()).
						Msg("Impersonated request")
				}

				return c.Next()
			}

			// If auth.users validation failed and jwtManager is provided, try platform.users token
			if len(jwtManager) > 0 && jwtManager[0] != nil {
				dashboardClaims, err := jwtManager[0].ValidateAccessToken(token)
				if err == nil {
					c.Locals("user_id", dashboardClaims.Subject)
					c.Locals("user_email", dashboardClaims.Email)
					c.Locals("user_name", dashboardClaims.Name)
					c.Locals("user_role", dashboardClaims.Role)
					c.Locals("auth_type", "jwt")
					c.Locals("is_anonymous", false)
					c.Locals("jwt_claims", dashboardClaims)

					c.Locals("rls_user_id", dashboardClaims.Subject)
					c.Locals("rls_role", dashboardClaims.Role)
					c.Locals("claims", dashboardClaims)

					log.Debug().
						Str("role", dashboardClaims.Role).
						Msg("Authenticated as platform.users via Bearer header")

					return c.Next()
				}
			}

			// User JWT and platform JWT validation failed, try service role JWT (anon/service_role)
			// JWTs with role claims instead of user claims
			if strings.HasPrefix(token, "eyJ") {
				claims, err := authService.ValidateServiceRoleToken(token)
				if err == nil {
					if claims.Role == "service_role" || claims.Role == "anon" {
						// SECURITY: Check emergency revocation for service_role tokens
						// This provides a mechanism to revoke compromised service_role tokens immediately
						if claims.Role == "service_role" {
							isRevoked, err := authService.IsServiceRoleTokenRevoked(c.RequestCtx(), claims.ID)
							if err != nil {
								log.Error().Err(err).Str("jti", claims.ID).Msg("Failed to check service_role token emergency revocation status")
								if securityCfg == nil || !securityCfg.ServiceRoleFailOpen {
									return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{
										"error":   "service_unavailable",
										"message": "Unable to verify token status",
									})
								}
								log.Warn().Str("jti", claims.ID).Msg("Service role revocation check failed - allowing request due to fail-open configuration")
							} else if isRevoked {
								log.Warn().Str("jti", claims.ID).Msg("Service role token has been emergency revoked")
								return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
									"error":             "Service role token has been revoked",
									"error_code":        "token_revoked",
									"revocation_reason": "emergency_revocation",
								})
							}
						}

						c.Locals("user_role", claims.Role)
						c.Locals("auth_type", "service_role_jwt")
						c.Locals("jwt_claims", claims)
						c.Locals("rls_role", claims.Role)
						c.Locals("claims", claims)

						log.Debug().
							Str("role", claims.Role).
							Str("issuer", claims.Issuer).
							Msg("Authenticated with service role JWT via Bearer header")

						return c.Next()
					}
				} else {
					log.Debug().
						Err(err).
						Msg("Bearer token validation failed (tried user JWT then service role JWT)")
				}
			}

			// Bearer token was provided but invalid - return specific error
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Invalid or expired Bearer token",
			})
		}

		// 3. Check for clientkey header (lowercase)
		// This header may contain a JWT with role claim (anon, service_role, authenticated)
		fluxbaseClientKey := c.Get("clientkey")
		if fluxbaseClientKey != "" && strings.HasPrefix(fluxbaseClientKey, "eyJ") {
			// Looks like a JWT - first try user JWT (most common), then service role
			claims, err := authService.ValidateToken(fluxbaseClientKey)
			if err == nil {
				// Check if token has been revoked
				// SECURITY: Fail-closed behavior - reject if we can't verify revocation status
				isRevoked, err := authService.IsTokenRevokedWithClaims(c.RequestCtx(), claims.ID, claims.UserID, claims.IssuedAt.Time)
				if err != nil {
					log.Error().Err(err).Str("jti", claims.ID).Msg("Token revocation check failed")
					return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{
						"error":   "service_unavailable",
						"message": "Unable to verify token status",
					})
				}
				if isRevoked {
					return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
						"error":   "token_revoked",
						"message": "Token has been revoked",
					})
				}

				// Valid user JWT token via clientkey header
				c.Locals("user_id", claims.UserID)
				c.Locals("user_email", claims.Email)
				c.Locals("user_role", claims.Role)
				c.Locals("session_id", claims.SessionID)
				c.Locals("auth_type", "jwt")
				c.Locals("is_anonymous", claims.IsAnonymous)
				c.Locals("jwt_claims", claims)

				c.Locals("rls_user_id", claims.UserID)
				c.Locals("rls_role", claims.Role)
				c.Locals("claims", claims)
				return c.Next()
			}

			// User JWT failed, try service role JWT
			srClaims, err := authService.ValidateServiceRoleToken(fluxbaseClientKey)
			if err == nil {
				// SECURITY: Check emergency revocation for service_role tokens
				// This provides a mechanism to revoke compromised service_role tokens immediately
				if srClaims.Role == "service_role" {
					isRevoked, err := authService.IsServiceRoleTokenRevoked(c.RequestCtx(), srClaims.ID)
					if err != nil {
						log.Error().Err(err).Str("jti", srClaims.ID).Msg("Failed to check service_role token emergency revocation status")
						if securityCfg == nil || !securityCfg.ServiceRoleFailOpen {
							return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{
								"error":   "service_unavailable",
								"message": "Unable to verify token status",
							})
						}
						log.Warn().Str("jti", srClaims.ID).Msg("Service role revocation check failed - allowing request due to fail-open configuration")
					} else if isRevoked {
						log.Warn().Str("jti", srClaims.ID).Msg("Service role token has been emergency revoked")
						return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
							"error":             "Service role token has been revoked",
							"error_code":        "token_revoked",
							"revocation_reason": "emergency_revocation",
						})
					}
				}

				// Valid service role JWT
				c.Locals("user_role", srClaims.Role)
				c.Locals("auth_type", "service_role_jwt")
				c.Locals("jwt_claims", srClaims)

				c.Locals("rls_role", srClaims.Role)
				if srClaims.UserID != "" {
					c.Locals("claims", srClaims)
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

		// 4. Client key authentication (X-Client-Key header, clientkey query param, or clientkey header fallback)
		clientKey := c.Get("X-Client-Key")
		if clientKey == "" {
			clientKey = c.Query("clientkey")
		}
		// Also check lowercase clientkey header if it wasn't a JWT
		if clientKey == "" && fluxbaseClientKey != "" {
			clientKey = fluxbaseClientKey
		}

		if clientKey != "" {
			validatedKey, err := clientKeyService.ValidateClientKey(c.RequestCtx(), clientKey)
			if err == nil {
				c.Locals("client_key_id", validatedKey.ID)
				c.Locals("client_key_name", validatedKey.Name)
				c.Locals("client_key_scopes", validatedKey.Scopes)
				c.Locals("auth_type", "clientkey")
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
			if errors.Is(err, auth.ErrUserClientKeysDisabled) {
				return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
					"error": "User client keys are disabled. Contact an administrator.",
				})
			}
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Invalid client key",
			})
		}

		// 5. No authentication provided
		if required {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Authentication required. Provide Bearer token, X-Client-Key, or X-Service-Key",
			})
		}
		return c.Next()
	}
}

// serviceKeyInfo holds the result of a service key lookup from either key table.
type serviceKeyInfo struct {
	keyID              string
	keyName            string
	keyType            string
	scopes             []string
	allowedNamespaces  *[]string
	isActive           bool
	expiresAt          *time.Time
	rateLimitPerMinute *int
	tenantID           *string
	lookupTable        string // "auth" or "platform"
}

// validateServiceKey validates a service key against auth.service_keys or platform.service_keys.
// It routes to the correct table based on key prefix and uses SET LOCAL ROLE service_role
// to bypass RLS during key lookup (necessary since no auth context exists yet).
// Returns true if valid, false otherwise.
func validateServiceKey(c fiber.Ctx, db *pgxpool.Pool, serviceKey string) bool {
	if len(serviceKey) < 8 {
		return false
	}

	// Determine which table and prefix to use based on key format
	prefix := keys.ExtractPrefix(serviceKey)
	var keyInfo *serviceKeyInfo
	var keyHash string

	if prefix != "" {
		// New-style key (fb_tsk_, fb_anon_, fb_gsk_, fb_pk_) -> platform.service_keys
		info, hash, err := lookupPlatformServiceKey(c, db, serviceKey)
		if err != nil {
			log.Debug().Err(err).Str("prefix", prefix).Msg("Platform service key lookup failed")
			return false
		}
		keyInfo = info
		keyHash = hash
	} else if strings.HasPrefix(serviceKey, "sk_") || strings.HasPrefix(serviceKey, "pk_") {
		// Legacy key (sk_, pk_) -> auth.service_keys
		info, hash, err := lookupAuthServiceKey(c, db, serviceKey)
		if err != nil {
			log.Debug().Err(err).Str("key_prefix", serviceKey[:min(16, len(serviceKey))]).Msg("Auth service key lookup failed")
			return false
		}
		keyInfo = info
		keyHash = hash
	} else {
		return false
	}

	// Verify the key is active
	if !keyInfo.isActive {
		log.Debug().Str("key_id", keyInfo.keyID).Msg("Service key is disabled")
		return false
	}

	// Verify not expired
	if keyInfo.expiresAt != nil && keyInfo.expiresAt.Before(time.Now()) {
		log.Debug().Str("key_id", keyInfo.keyID).Msg("Service key has expired")
		return false
	}

	// Verify the key hash
	if err := bcrypt.CompareHashAndPassword([]byte(keyHash), []byte(serviceKey)); err != nil {
		log.Debug().Err(err).Str("key_id", keyInfo.keyID).Msg("Invalid service key hash")
		return false
	}

	// Map key type to application role
	role := mapKeyTypetoRole(keyInfo.keyType)

	// Fire-and-forget last_used_at update
	updateLastUsedAt(c, db, keyInfo)

	// Store service key information in context
	c.Locals("service_key_id", keyInfo.keyID)
	c.Locals("service_key_name", keyInfo.keyName)
	c.Locals("service_key_scopes", keyInfo.scopes)
	c.Locals("service_key_type", keyInfo.keyType)
	c.Locals("auth_type", "service_key")
	c.Locals("user_role", role)

	// Store rate limits in context
	c.Locals("service_key_rate_limit_per_minute", keyInfo.rateLimitPerMinute)

	// Store allowed namespaces if present
	if keyInfo.allowedNamespaces != nil {
		c.Locals("allowed_namespaces", *keyInfo.allowedNamespaces)
	}

	// Store tenant_id from key if present (needed for tenant_service keys)
	if keyInfo.tenantID != nil {
		c.Locals("service_key_tenant_id", *keyInfo.tenantID)
	}

	// For RLS context
	c.Locals("rls_role", role)
	c.Locals("rls_user_id", nil)

	log.Debug().
		Str("key_id", keyInfo.keyID).
		Str("key_name", keyInfo.keyName).
		Str("key_type", keyInfo.keyType).
		Str("role", role).
		Str("lookup_table", keyInfo.lookupTable).
		Interface("rate_limit_per_minute", keyInfo.rateLimitPerMinute).
		Msg("Authenticated with service key")

	return true
}

// lookupPlatformServiceKey looks up a key in platform.service_keys using SET LOCAL ROLE service_role
// to bypass RLS (key validation happens before auth context is set).
func lookupPlatformServiceKey(c fiber.Ctx, db *pgxpool.Pool, serviceKey string) (*serviceKeyInfo, string, error) {
	keyPrefix := keys.ExtractPrefix(serviceKey)
	if keyPrefix == "" || len(serviceKey) < len(keyPrefix)+8 {
		return nil, "", errors.New("invalid platform key format")
	}
	prefixForLookup := serviceKey[:len(keyPrefix)+8]

	// Use tenant pool if available (database-per-tenant), otherwise use main pool
	pool := GetTenantPool(c)
	if pool == nil {
		pool = db
	}

	var info serviceKeyInfo
	var keyHash string

	err := queryWithServiceRole(c.RequestCtx(), pool, func(tx pgx.Tx) error {
		return tx.QueryRow(
			c.RequestCtx(),
			`SELECT id, name, key_hash, key_type, scopes, allowed_namespaces,
			        is_active, expires_at, rate_limit_per_minute, tenant_id
			 FROM platform.service_keys
			 WHERE key_prefix = $1`,
			prefixForLookup,
		).Scan(&info.keyID, &info.keyName, &keyHash, &info.keyType, &info.scopes,
			&info.allowedNamespaces, &info.isActive, &info.expiresAt,
			&info.rateLimitPerMinute, &info.tenantID)
	})
	if err != nil {
		return nil, "", fmt.Errorf("platform key lookup failed: %w", err)
	}

	info.lookupTable = "platform"
	return &info, keyHash, nil
}

// lookupAuthServiceKey looks up a key in auth.service_keys using SET LOCAL ROLE service_role
// to bypass RLS (key validation happens before auth context is set).
func lookupAuthServiceKey(c fiber.Ctx, db *pgxpool.Pool, serviceKey string) (*serviceKeyInfo, string, error) {
	if len(serviceKey) < 16 {
		return nil, "", errors.New("auth key too short")
	}
	keyPrefix := serviceKey[:16]

	// Use tenant pool if available (database-per-tenant), otherwise use main pool
	pool := GetTenantPool(c)
	if pool == nil {
		pool = db
	}

	var info serviceKeyInfo
	var keyHash string
	var revokedAt *time.Time
	var rateLimitPerHour *int

	err := queryWithServiceRole(c.RequestCtx(), pool, func(tx pgx.Tx) error {
		return tx.QueryRow(
			c.RequestCtx(),
			`SELECT id, name, key_hash, COALESCE(key_type, 'service'), scopes, allowed_namespaces,
			        enabled, expires_at, rate_limit_per_minute, rate_limit_per_hour,
			        revoked_at, tenant_id
			 FROM auth.service_keys
			 WHERE key_prefix = $1`,
			keyPrefix,
		).Scan(&info.keyID, &info.keyName, &keyHash, &info.keyType, &info.scopes,
			&info.allowedNamespaces, &info.isActive, &info.expiresAt,
			&info.rateLimitPerMinute, &rateLimitPerHour, &revokedAt, &info.tenantID)
	})
	if err != nil {
		return nil, "", fmt.Errorf("auth key lookup failed: %w", err)
	}

	// Check revoked (handled inline for legacy table)
	if revokedAt != nil {
		info.isActive = false
	}

	info.lookupTable = "auth"
	return &info, keyHash, nil
}

// queryWithServiceRole executes a read query within a transaction with SET LOCAL ROLE service_role.
// This bypasses RLS for key lookup operations that happen before auth context is established.
func queryWithServiceRole(ctx context.Context, pool *pgxpool.Pool, fn func(tx pgx.Tx) error) error {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	_, err = tx.Exec(ctx, `SET LOCAL ROLE "service_role"`)
	if err != nil {
		return fmt.Errorf("failed to SET LOCAL ROLE service_role: %w", err)
	}

	if err := fn(tx); err != nil {
		return err
	}

	return tx.Commit(ctx)
}

// mapKeyTypetoRole maps a service key type to the application role used for RLS.
func mapKeyTypetoRole(keyType string) string {
	switch keyType {
	case keys.KeyTypeAnon:
		return "anon"
	case keys.KeyTypeTenantService:
		return "tenant_service"
	case keys.KeyTypeGlobalService, "service":
		return "service_role"
	case keys.KeyTypePublishable:
		return "authenticated"
	default:
		log.Warn().Str("key_type", keyType).Msg("mapKeyTypetoRole: unrecognized key type, defaulting to anon")
		return "anon"
	}
}

// updateLastUsedAt fires a background goroutine to update the last_used_at timestamp.
// Uses SET LOCAL ROLE service_role to bypass RLS on both key tables.
func updateLastUsedAt(c fiber.Ctx, db *pgxpool.Pool, info *serviceKeyInfo) {
	table := "auth.service_keys"
	if info.lookupTable == "platform" {
		table = "platform.service_keys"
	}

	keyID := info.keyID
	// Capture pool reference before spawning goroutine to avoid data race
	// on fiber.Ctx (the context is not safe for concurrent access).
	pool := GetTenantPool(c)
	if pool == nil {
		pool = db
	}
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		tx, err := pool.Begin(ctx)
		if err != nil {
			return
		}
		defer func() { _ = tx.Rollback(ctx) }()

		_, err = tx.Exec(ctx, `SET LOCAL ROLE "service_role"`)
		if err != nil {
			return
		}

		_, _ = tx.Exec(
			ctx,
			`UPDATE `+table+` SET last_used_at = NOW() WHERE id = $1`,
			keyID,
		)
		_ = tx.Commit(ctx)
	}()
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// RequireAdmin middleware restricts access to admin users only
// Allows: service_role (from service keys or service_role JWT) and instance_admin users
// This should be used after authentication middleware (RequireAuthOrServiceKey)
func RequireAdmin() fiber.Handler {
	return func(c fiber.Ctx) error {
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
		if role == "service_role" || role == "instance_admin" || role == "tenant_service" {
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
			Msg("Admin access denied - requires service_role or instance_admin")

		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "Admin access required. Only service_role and instance_admin can access this endpoint.",
		})
	}
}

// RequireAdminIfClientKeysDisabled middleware conditionally requires admin access
// when the 'app.auth.allow_user_client_keys' setting is disabled.
// If the setting is enabled (default), allows regular users through.
// If the setting is disabled, requires admin access (service_role or instance_admin).
func RequireAdminIfClientKeysDisabled(settingsCache *auth.SettingsCache) fiber.Handler {
	return func(c fiber.Ctx) error {
		// Check if user client keys are allowed
		allowUserKeys := settingsCache.GetBool(c.RequestCtx(), "app.auth.allow_user_client_keys", true)

		if allowUserKeys {
			// Setting is enabled - allow regular users to manage their own keys
			return c.Next()
		}

		// Setting is disabled - require admin access
		return RequireAdmin()(c)
	}
}

// fiber:context-methods migrated
