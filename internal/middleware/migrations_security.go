package middleware

import (
	"context"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog/log"
	"golang.org/x/crypto/bcrypt"

	"github.com/nimbleflux/fluxbase/internal/auth"
	"github.com/nimbleflux/fluxbase/internal/config"
	apperrors "github.com/nimbleflux/fluxbase/internal/errors"
)

type MigrationsTenantPoolProvider interface {
	GetPool(tenantID string) (*pgxpool.Pool, error)
}

type migrationsRateLimiter struct {
	limit    int
	window   time.Duration
	storage  fiber.Storage
	requests map[string]*migrationsRateLimitEntry
	mu       sync.Mutex
}

type migrationsRateLimitEntry struct {
	count     int
	expiresAt time.Time
}

func newRateLimiter(limit int, window time.Duration, storage fiber.Storage) *migrationsRateLimiter {
	return &migrationsRateLimiter{
		limit:    limit,
		window:   window,
		storage:  storage,
		requests: make(map[string]*migrationsRateLimitEntry),
	}
}

func (rl *migrationsRateLimiter) allow(key string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	if entry, exists := rl.requests[key]; exists {
		if now.After(entry.expiresAt) {
			entry.count = 1
			entry.expiresAt = now.Add(rl.window)
			return true
		}
		if entry.count >= rl.limit {
			return false
		}
		entry.count++
		return true
	}

	rl.requests[key] = &migrationsRateLimitEntry{
		count:     1,
		expiresAt: now.Add(rl.window),
	}
	return true
}

func RequireMigrationsFullSecurity(
	cfg *config.MigrationsConfig,
	serverCfg *config.ServerConfig,
	db *pgxpool.Pool,
	authService *auth.Service,
	rateLimit int,
	rateWindow time.Duration,
	storage fiber.Storage,
) fiber.Handler {
	return RequireMigrationsFullSecurityWithTenantProvider(
		cfg, serverCfg, db, authService, rateLimit, rateWindow, storage, nil,
	)
}

func RequireMigrationsFullSecurityWithTenantProvider(
	cfg *config.MigrationsConfig,
	serverCfg *config.ServerConfig,
	db *pgxpool.Pool,
	authService *auth.Service,
	rateLimit int,
	rateWindow time.Duration,
	storage fiber.Storage,
	tenantPoolProvider MigrationsTenantPoolProvider,
) fiber.Handler {
	var allowedNets []*net.IPNet
	for _, ipRange := range cfg.AllowedIPRanges {
		_, network, err := net.ParseCIDR(ipRange)
		if err != nil {
			log.Error().Err(err).Str("range", ipRange).Msg("Invalid IP range in migrations config")
			continue
		}
		allowedNets = append(allowedNets, network)
	}

	limiter := newRateLimiter(rateLimit, rateWindow, storage)

	return func(c fiber.Ctx) error {
		if !cfg.Enabled {
			log.Warn().Str("path", c.Path()).Str("ip", c.IP()).Msg("Migrations API access denied - feature disabled")
			return apperrors.SendNotFound(c, "Not Found")
		}

		if len(allowedNets) > 0 {
			clientIP := GetTrustedClientIP(c, serverCfg)
			allowed := false
			for _, network := range allowedNets {
				if network.Contains(clientIP) {
					allowed = true
					break
				}
			}
			if !allowed {
				log.Warn().Str("ip", clientIP.String()).Str("path", c.Path()).Msg("Migrations API access denied - IP not in allowlist")
				return apperrors.SendErrorWithCode(c, fiber.StatusForbidden, "Access denied - IP not allowlisted for migrations", apperrors.ErrCodeAccessDenied)
			}
		}

		if !migrationsValidateAuthAndScope(c, db, authService, tenantPoolProvider) {
			return apperrors.SendErrorWithCode(c, fiber.StatusUnauthorized, "Service key or service_role JWT authentication required for migrations API", apperrors.ErrCodeMissingAuth)
		}

		rateLimitKey := migrationsGetRateLimitKey(c)
		if !limiter.allow(rateLimitKey) {
			log.Warn().Str("key", rateLimitKey).Int("limit", rateLimit).Msg("Migrations API rate limit exceeded")
			return apperrors.SendErrorWithCode(c, fiber.StatusTooManyRequests, "Rate limit exceeded", apperrors.ErrCodeRateLimited)
		}

		start := time.Now()
		serviceKeyID := c.Locals("service_key_id")
		serviceKeyName := c.Locals("service_key_name")
		log.Info().Str("method", c.Method()).Str("path", c.Path()).Str("ip", c.IP()).
			Interface("service_key_id", serviceKeyID).Interface("service_key_name", serviceKeyName).
			Msg("Migrations API request started")

		err := c.Next()

		log.Info().Str("method", c.Method()).Str("path", c.Path()).
			Int("status", c.Response().StatusCode()).Dur("duration", time.Since(start)).
			Str("ip", c.IP()).Interface("service_key_id", serviceKeyID).
			Msg("Migrations API request completed")

		return err
	}
}

func migrationsValidateAuthAndScope(c fiber.Ctx, db *pgxpool.Pool, authService *auth.Service, tenantPoolProvider MigrationsTenantPoolProvider) bool {
	authHeader := c.Get("Authorization")
	clientkey := c.Get("clientkey")

	var jwtToken string
	if strings.HasPrefix(authHeader, "Bearer ") {
		token := strings.TrimPrefix(authHeader, "Bearer ")
		if strings.HasPrefix(token, "eyJ") {
			jwtToken = token
		}
	}
	if jwtToken == "" && strings.HasPrefix(clientkey, "eyJ") {
		jwtToken = clientkey
	}

	if jwtToken != "" {
		claims, err := authService.JWTManager().ValidateToken(jwtToken)
		if err == nil {
			c.Locals("auth_type", "jwt")
			c.Locals("user_role", claims.Role)
			c.Locals("user_id", claims.UserID)
			c.Locals("claims", claims)
			c.Locals("jwt_claims", claims)

			if claims.Role == "service_role" {
				tenantID := migrationsGetTenantIDFromHeader(c)
				if tenantID != "" && tenantPoolProvider != nil {
					c.Locals("tenant_id", tenantID)
					c.Locals("is_tenant_migration", true)
					return true
				}
				c.Locals("is_tenant_migration", false)
				return true
			}

			if claims.Role == "admin" || claims.Role == "instance_admin" {
				c.Locals("is_tenant_migration", false)
				return true
			}

			if claims.Role == "tenant_admin" || (claims.TenantID != nil && *claims.TenantID != "") {
				tenantID := migrationsGetTenantID(c, claims)
				if tenantID == "" {
					log.Warn().Str("user_id", claims.UserID).Str("role", claims.Role).Msg("Tenant admin must have tenant context")
					return false
				}

				if !migrationsValidateTenantMembership(c.RequestCtx(), db, claims.UserID, tenantID) {
					log.Warn().Str("user_id", claims.UserID).Str("tenant_id", tenantID).Msg("User not member of tenant")
					return false
				}

				c.Locals("tenant_id", tenantID)
				c.Locals("is_tenant_migration", true)
				return true
			}
		}
	}

	serviceKey := c.Get("X-Service-Key")
	if serviceKey == "" {
		if strings.HasPrefix(authHeader, "ServiceKey ") {
			serviceKey = strings.TrimPrefix(authHeader, "ServiceKey ")
		} else if serviceKey == "" && strings.HasPrefix(authHeader, "Bearer ") {
			token := strings.TrimPrefix(authHeader, "Bearer ")
			if strings.HasPrefix(token, "sk_") {
				serviceKey = token
			}
		}
	}
	if serviceKey == "" && strings.HasPrefix(clientkey, "sk_") {
		serviceKey = clientkey
	}

	if serviceKey != "" {
		tenantID := migrationsGetTenantIDFromHeader(c)

		if tenantID != "" && tenantPoolProvider != nil {
			tenantPool, err := tenantPoolProvider.GetPool(tenantID)
			if err != nil {
				log.Warn().Err(err).Str("tenant_id", tenantID).Msg("Failed to get tenant pool for service key validation")
				return false
			}

			if migrationsValidateServiceKeyWithScope(c, tenantPool, serviceKey, "migrations:execute") {
				c.Locals("tenant_id", tenantID)
				c.Locals("is_tenant_migration", true)
				return true
			}
		}

		if migrationsValidateServiceKeyWithScope(c, db, serviceKey, "migrations:execute") {
			// A tenant-scoped service key must not be used without a tenant header
			keyType, _ := c.Locals("service_key_type").(string)
			if keyType == "tenant_service" {
				log.Warn().Str("key_type", keyType).Msg("Tenant-scoped service key used without tenant header")
				return false
			}
			c.Locals("is_tenant_migration", false)
			return true
		}
	}

	log.Warn().Str("path", c.Path()).Str("ip", c.IP()).Msg("Migrations API auth failed")
	return false
}

func migrationsGetTenantIDFromHeader(c fiber.Ctx) string {
	return c.Get("X-FB-Tenant")
}

func migrationsGetTenantID(c fiber.Ctx, claims *auth.TokenClaims) string {
	if headerTenant := c.Get("X-FB-Tenant"); headerTenant != "" {
		return headerTenant
	}
	if claims.TenantID != nil {
		return *claims.TenantID
	}
	return ""
}

func migrationsValidateTenantMembership(ctx context.Context, db *pgxpool.Pool, userID, tenantID string) bool {
	var isMember bool
	err := db.QueryRow(
		ctx,
		`SELECT EXISTS(
			SELECT 1 FROM platform.tenant_admin_assignments taa
			INNER JOIN platform.tenants t ON t.id = taa.tenant_id
			WHERE taa.user_id = $1::uuid
			AND taa.tenant_id = $2::uuid
			AND t.deleted_at IS NULL
		)`,
		userID, tenantID,
	).Scan(&isMember)
	if err != nil {
		log.Debug().Err(err).Str("user_id", userID).Str("tenant_id", tenantID).Msg("Failed to validate tenant membership")
		return false
	}
	return isMember
}

func migrationsValidateServiceKeyWithScope(c fiber.Ctx, db *pgxpool.Pool, serviceKey, requiredScope string) bool {
	if len(serviceKey) < 16 || !strings.HasPrefix(serviceKey, "sk_") {
		return false
	}
	keyPrefix := serviceKey[:16]

	var keyHash, keyID, keyName string
	var scopes []string
	var enabled bool
	var expiresAt *time.Time
	var keyType string

	err := db.QueryRow(
		c.RequestCtx(),
		`SELECT id, name, key_hash, COALESCE(key_type, 'service'), scopes, enabled, expires_at FROM auth.service_keys WHERE key_prefix = $1`,
		keyPrefix,
	).Scan(&keyID, &keyName, &keyHash, &keyType, &scopes, &enabled, &expiresAt)
	if err != nil {
		return false
	}

	if !enabled || (expiresAt != nil && expiresAt.Before(time.Now())) {
		return false
	}

	if err := bcrypt.CompareHashAndPassword([]byte(keyHash), []byte(serviceKey)); err != nil {
		return false
	}

	hasScope := false
	for _, scope := range scopes {
		if scope == requiredScope || scope == "*" {
			hasScope = true
			break
		}
	}
	if !hasScope {
		log.Warn().Str("required", requiredScope).Interface("scopes", scopes).Msg("Service key missing required scope")
		return false
	}

	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_, _ = db.Exec(ctx, `UPDATE auth.service_keys SET last_used_at = NOW() WHERE id = $1`, keyID)
	}()

	c.Locals("service_key_id", keyID)
	c.Locals("service_key_name", keyName)
	c.Locals("service_key_scopes", scopes)
	c.Locals("service_key_type", keyType)
	c.Locals("auth_type", "service_key")

	return true
}

func migrationsGetRateLimitKey(c fiber.Ctx) string {
	if keyID := c.Locals("service_key_id"); keyID != nil {
		if id, ok := keyID.(string); ok {
			return "migration:" + id
		}
	}
	if userID := c.Locals("user_id"); userID != nil {
		if id, ok := userID.(string); ok && id != "" {
			return "migration:" + id
		}
	}
	return "migration:ip:" + c.IP()
}
