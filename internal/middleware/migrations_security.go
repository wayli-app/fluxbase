package middleware

import (
	"context"
	"net"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog/log"
	"github.com/wayli-app/fluxbase/internal/auth"
	"github.com/wayli-app/fluxbase/internal/config"
	"golang.org/x/crypto/bcrypt"
)

// RequireMigrationsEnabled checks if migrations API is enabled
// If disabled, returns HTTP 404 to hide the feature entirely
func RequireMigrationsEnabled(cfg *config.MigrationsConfig) fiber.Handler {
	return func(c *fiber.Ctx) error {
		if !cfg.Enabled {
			log.Warn().
				Str("path", c.Path()).
				Str("ip", c.IP()).
				Msg("Migrations API access denied - feature disabled")
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "Not Found",
			})
		}
		return c.Next()
	}
}

// RequireMigrationsIPAllowlist restricts access to migrations API by IP range
// Only IPs within the configured ranges are allowed
// If no IP ranges are configured, all IPs are allowed
func RequireMigrationsIPAllowlist(cfg *config.MigrationsConfig) fiber.Handler {
	// Parse allowed IP ranges at startup
	var allowedNets []*net.IPNet
	for _, ipRange := range cfg.AllowedIPRanges {
		_, network, err := net.ParseCIDR(ipRange)
		if err != nil {
			log.Error().Err(err).Str("range", ipRange).Msg("Invalid IP range in migrations config")
			continue
		}
		allowedNets = append(allowedNets, network)
	}

	return func(c *fiber.Ctx) error {
		// Skip if no IP ranges configured (allows all)
		if len(allowedNets) == 0 {
			return c.Next()
		}

		clientIP := getClientIP(c)

		// Check if IP is in any allowed range
		for _, network := range allowedNets {
			if network.Contains(clientIP) {
				log.Debug().
					Str("ip", clientIP.String()).
					Str("network", network.String()).
					Msg("Migrations API access allowed - IP in allowlist")
				return c.Next()
			}
		}

		// IP not in allowlist
		log.Warn().
			Str("ip", clientIP.String()).
			Str("path", c.Path()).
			Msg("Migrations API access denied - IP not in allowlist")

		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "Access denied - IP not allowlisted for migrations",
		})
	}
}

// RequireServiceKeyOnly enforces service key authentication (service keys or service_role JWT)
// Migrations API requires the highest level of authentication
// Accepts: 1) Service keys (sk_*) via X-Service-Key, Authorization, or apikey headers
//  2. JWT tokens with service_role via Authorization or apikey headers
func RequireServiceKeyOnly(db *pgxpool.Pool, authService *auth.Service) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Try JWT authentication first (from apikey or Authorization header)
		authHeader := c.Get("Authorization")
		apikey := c.Get("apikey")

		// Extract JWT token from headers
		var jwtToken string
		if strings.HasPrefix(authHeader, "Bearer ") {
			token := strings.TrimPrefix(authHeader, "Bearer ")
			// Check if it's a JWT (starts with eyJ) not a service key (starts with sk_)
			if strings.HasPrefix(token, "eyJ") {
				jwtToken = token
			}
		}
		if jwtToken == "" && strings.HasPrefix(apikey, "eyJ") {
			jwtToken = apikey
		}

		// If we have a JWT token, validate it
		if jwtToken != "" {
			log.Debug().Str("jwt_prefix", jwtToken[:min(20, len(jwtToken))]).Msg("Validating JWT token")

			claims, err := authService.ValidateToken(jwtToken)
			if err == nil {
				// Check if role is service_role
				if claims.Role == "service_role" {
					log.Debug().
						Str("role", claims.Role).
						Msg("Migrations API access granted - service_role JWT")

					// Set context locals
					c.Locals("auth_type", "jwt")
					c.Locals("role", claims.Role)
					c.Locals("user_id", claims.UserID)

					return c.Next()
				}

				log.Warn().
					Str("role", claims.Role).
					Msg("Migrations API access denied - JWT role must be service_role")
				return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
					"error": "Migrations API requires service_role JWT",
				})
			}

			log.Debug().Err(err).Msg("JWT validation failed, trying service key")
		}

		// Try service key authentication
		serviceKey := ""

		// Try X-Service-Key header first (most explicit)
		serviceKey = c.Get("X-Service-Key")
		log.Debug().Str("X-Service-Key", serviceKey).Msg("Checking X-Service-Key header")

		// Try Authorization header (ServiceKey or Bearer with service key)
		if serviceKey == "" && strings.HasPrefix(authHeader, "ServiceKey ") {
			serviceKey = strings.TrimPrefix(authHeader, "ServiceKey ")
		} else if serviceKey == "" && strings.HasPrefix(authHeader, "Bearer ") {
			// Accept Bearer token if it looks like a service key
			token := strings.TrimPrefix(authHeader, "Bearer ")
			if strings.HasPrefix(token, "sk_") {
				serviceKey = token
			}
		}

		// Try apikey header (used by SDK)
		if serviceKey == "" && strings.HasPrefix(apikey, "sk_") {
			serviceKey = apikey
		}

		if serviceKey != "" {
			log.Debug().Str("key_prefix", serviceKey[:min(16, len(serviceKey))]).Msg("Validating service key")

			// Validate service key
			if validateMigrationServiceKey(c, db, serviceKey) {
				return c.Next()
			}

			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Invalid service key",
			})
		}

		// Neither JWT nor service key provided
		log.Warn().
			Str("path", c.Path()).
			Str("ip", c.IP()).
			Msg("Migrations API access denied - service key or service_role JWT required")
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Service key or service_role JWT authentication required for migrations API",
		})
	}
}

// min returns the minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// RequireMigrationScope checks if authentication has migrations permissions
// Accepts both service_role JWT (full access) and service keys with migrations:execute scope
func RequireMigrationScope() fiber.Handler {
	return func(c *fiber.Ctx) error {
		authType := c.Locals("auth_type")

		// JWT with service_role has full access (no scope check needed)
		if authType == "jwt" {
			role := c.Locals("role")
			if role == "service_role" {
				log.Debug().
					Str("auth_type", "jwt").
					Str("role", "service_role").
					Msg("Migrations scope check passed - service_role has full access")
				return c.Next()
			}

			log.Warn().
				Str("auth_type", "jwt").
				Interface("role", role).
				Msg("Migrations require service_role JWT")
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "Migrations require service_role JWT",
			})
		}

		// For service keys, check scopes
		if authType != "service_key" {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "Migrations require service key or service_role JWT authentication",
			})
		}

		scopes, ok := c.Locals("service_key_scopes").([]string)
		if !ok {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "No scopes found",
			})
		}

		// Check for migration scope
		hasScope := false
		requiredScope := "migrations:execute"
		for _, scope := range scopes {
			if scope == requiredScope || scope == "*" {
				hasScope = true
				break
			}
		}

		if !hasScope {
			log.Warn().
				Str("required", requiredScope).
				Interface("scopes", scopes).
				Msg("Service key missing required scope")
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "Service key does not have migrations:execute scope",
			})
		}

		return c.Next()
	}
}

// MigrationsAuditLog logs all migrations API requests for security auditing
func MigrationsAuditLog() fiber.Handler {
	return func(c *fiber.Ctx) error {
		start := time.Now()

		// Get service key info before processing
		serviceKeyID := c.Locals("service_key_id")
		serviceKeyName := c.Locals("service_key_name")

		// Log request
		log.Info().
			Str("method", c.Method()).
			Str("path", c.Path()).
			Str("ip", c.IP()).
			Interface("service_key_id", serviceKeyID).
			Interface("service_key_name", serviceKeyName).
			Msg("Migrations API request started")

		// Continue processing
		err := c.Next()

		// Log response
		log.Info().
			Str("method", c.Method()).
			Str("path", c.Path()).
			Int("status", c.Response().StatusCode()).
			Dur("duration", time.Since(start)).
			Str("ip", c.IP()).
			Interface("service_key_id", serviceKeyID).
			Msg("Migrations API request completed")

		return err
	}
}

// getClientIP extracts real client IP from headers (for proxy environments)
func getClientIP(c *fiber.Ctx) net.IP {
	// Try X-Forwarded-For header first (for proxies)
	xff := c.Get("X-Forwarded-For")
	if xff != "" {
		ips := strings.Split(xff, ",")
		if len(ips) > 0 {
			ip := strings.TrimSpace(ips[0])
			parsed := net.ParseIP(ip)
			if parsed != nil {
				return parsed
			}
		}
	}

	// Try X-Real-IP header
	xri := c.Get("X-Real-IP")
	if xri != "" {
		parsed := net.ParseIP(xri)
		if parsed != nil {
			return parsed
		}
	}

	// Fall back to RemoteAddr
	return net.ParseIP(c.IP())
}

// validateMigrationServiceKey validates a service key for migrations API
// This is similar to validateServiceKey but sets auth_type to "service_key"
func validateMigrationServiceKey(c *fiber.Ctx, db *pgxpool.Pool, serviceKey string) bool {
	// Extract key prefix (first 16 chars for identification)
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

	log.Debug().
		Str("key_id", keyID).
		Str("key_name", keyName).
		Interface("scopes", scopes).
		Msg("Service key validated for migrations API")

	return true
}
