package middleware

import (
	"github.com/gofiber/fiber/v3"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/nimbleflux/fluxbase/internal/auth"
	"github.com/nimbleflux/fluxbase/internal/config"
	"github.com/nimbleflux/fluxbase/internal/database"
	"github.com/nimbleflux/fluxbase/internal/settings"
)

// AuthMiddlewareFactory creates auth middlewares with consistent configuration.
// It centralizes the creation of authentication and authorization middleware
// to ensure consistent behavior across all routes.
//
// Usage:
//
//	factory := middleware.NewAuthMiddlewareFactory(authService, clientKeyService, db, jwtManager, settingsCache)
//	deps.RequireAuth = factory.Required()
//	deps.OptionalAuth = factory.Optional()
type AuthMiddlewareFactory struct {
	authService      *auth.Service
	clientKeyService *auth.ClientKeyService
	db               *database.Connection
	pool             *pgxpool.Pool
	jwtManager       *auth.JWTManager
	settingsCache    *settings.SettingsCache
	serverConfig     *config.ServerConfig
	securityCfg      *config.SecurityConfig
}

// AuthMiddlewareFactoryOption is a functional option for configuring the factory.
type AuthMiddlewareFactoryOption func(*AuthMiddlewareFactory)

// WithDBConnection sets the database connection (wraps pool with additional functionality).
func WithDBConnection(db *database.Connection) AuthMiddlewareFactoryOption {
	return func(f *AuthMiddlewareFactory) {
		f.db = db
	}
}

// WithPool sets the database pool directly.
func WithPool(pool *pgxpool.Pool) AuthMiddlewareFactoryOption {
	return func(f *AuthMiddlewareFactory) {
		f.pool = pool
	}
}

// WithServerConfig sets the server configuration for IP-based middleware.
func WithServerConfig(cfg *config.ServerConfig) AuthMiddlewareFactoryOption {
	return func(f *AuthMiddlewareFactory) {
		f.serverConfig = cfg
	}
}

// WithSecurityConfig sets the security configuration for fail-open/fail-closed behavior.
func WithSecurityConfig(cfg *config.SecurityConfig) AuthMiddlewareFactoryOption {
	return func(f *AuthMiddlewareFactory) {
		f.securityCfg = cfg
	}
}

// NewAuthMiddlewareFactory creates a new auth middleware factory.
func NewAuthMiddlewareFactory(
	authService *auth.Service,
	clientKeyService *auth.ClientKeyService,
	settingsCache *settings.SettingsCache,
	jwtManager *auth.JWTManager,
	opts ...AuthMiddlewareFactoryOption,
) *AuthMiddlewareFactory {
	f := &AuthMiddlewareFactory{
		authService:      authService,
		clientKeyService: clientKeyService,
		settingsCache:    settingsCache,
		jwtManager:       jwtManager,
	}

	for _, opt := range opts {
		opt(f)
	}

	// Use db.Pool() if db is set but pool is not
	if f.pool == nil && f.db != nil {
		f.pool = f.db.Pool()
	}

	return f
}

// Required returns middleware that requires authentication.
// Accepts JWT tokens, client keys, or service keys.
func (f *AuthMiddlewareFactory) Required() fiber.Handler {
	return RequireAuthOrServiceKey(f.authService, f.clientKeyService, f.pool, f.securityCfg, f.jwtManager)
}

// Optional returns middleware that optionally extracts auth context.
// Sets user context if valid credentials are present, but allows anonymous access.
func (f *AuthMiddlewareFactory) Optional() fiber.Handler {
	return OptionalAuthOrServiceKey(f.authService, f.clientKeyService, f.pool, f.securityCfg, f.jwtManager)
}

// Internal returns middleware for internal-only endpoints.
// Only allows requests from localhost/internal IPs.
func (f *AuthMiddlewareFactory) Internal() fiber.Handler {
	return RequireInternal()
}

// Admin returns middleware that requires admin role.
func (f *AuthMiddlewareFactory) Admin() fiber.Handler {
	return RequireAdmin()
}

// Scope returns middleware that requires specific scopes.
func (f *AuthMiddlewareFactory) Scope(scopes ...string) fiber.Handler {
	return RequireScope(scopes...)
}

// FeatureEnabled returns middleware that checks if a feature is enabled.
func (f *AuthMiddlewareFactory) FeatureEnabled(featureKey string) fiber.Handler {
	return RequireFeatureEnabled(f.settingsCache, featureKey)
}

// RealtimeEnabled returns middleware that ensures realtime feature is enabled.
func (f *AuthMiddlewareFactory) RealtimeEnabled() fiber.Handler {
	return RequireRealtimeEnabled(f.settingsCache)
}

// FunctionsEnabled returns middleware that ensures functions feature is enabled.
func (f *AuthMiddlewareFactory) FunctionsEnabled() fiber.Handler {
	return RequireFunctionsEnabled(f.settingsCache)
}

// JobsEnabled returns middleware that ensures jobs feature is enabled.
func (f *AuthMiddlewareFactory) JobsEnabled() fiber.Handler {
	return RequireJobsEnabled(f.settingsCache)
}

// AIEnabled returns middleware that ensures AI feature is enabled.
func (f *AuthMiddlewareFactory) AIEnabled() fiber.Handler {
	return RequireAIEnabled(f.settingsCache)
}

// RPCEnabled returns middleware that ensures RPC feature is enabled.
func (f *AuthMiddlewareFactory) RPCEnabled() fiber.Handler {
	return RequireRPCEnabled(f.settingsCache)
}

// StorageEnabled returns middleware that ensures storage feature is enabled.
func (f *AuthMiddlewareFactory) StorageEnabled() fiber.Handler {
	return RequireStorageEnabled(f.settingsCache)
}

// AdminIfClientKeysDisabled returns middleware that requires admin role if client keys are disabled.
func (f *AuthMiddlewareFactory) AdminIfClientKeysDisabled() fiber.Handler {
	return RequireAdminIfClientKeysDisabled(f.settingsCache)
}

// SyncIPAllowlist returns middleware that checks sync IP allowlist for a feature.
func (f *AuthMiddlewareFactory) SyncIPAllowlist(allowedRanges []string, featureName string) fiber.Handler {
	return RequireSyncIPAllowlist(allowedRanges, featureName, f.serverConfig)
}

// TenantRole returns middleware that requires a specific tenant role.
func (f *AuthMiddlewareFactory) TenantRole(requiredRole string) fiber.Handler {
	return RequireTenantRole(requiredRole)
}

// InstanceAdmin returns middleware that requires instance admin role.
func (f *AuthMiddlewareFactory) InstanceAdmin() fiber.Handler {
	return RequireInstanceAdmin()
}

// EitherAuth returns middleware that requires either auth service OR client key authentication.
// Deprecated: Use Required() instead, which includes service key support.
func (f *AuthMiddlewareFactory) EitherAuth() fiber.Handler {
	return RequireEitherAuth(f.authService, f.clientKeyService)
}
