package config

import (
	"fmt"
	"math"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/joho/godotenv"
	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
)

// Config represents the application configuration
type Config struct {
	Server        ServerConfig     `mapstructure:"server"`
	Database      DatabaseConfig   `mapstructure:"database"`
	Auth          AuthConfig       `mapstructure:"auth"`
	Security      SecurityConfig   `mapstructure:"security"`
	CORS          CORSConfig       `mapstructure:"cors"`
	Storage       StorageConfig    `mapstructure:"storage"`
	Realtime      RealtimeConfig   `mapstructure:"realtime"`
	Email         EmailConfig      `mapstructure:"email"`
	Functions     FunctionsConfig  `mapstructure:"functions"`
	API           APIConfig        `mapstructure:"api"`
	Migrations    MigrationsConfig `mapstructure:"migrations"`
	Jobs          JobsConfig       `mapstructure:"jobs"`
	Deno          DenoConfig       `mapstructure:"deno"`
	Tracing       TracingConfig    `mapstructure:"tracing"`
	Metrics       MetricsConfig    `mapstructure:"metrics"`
	AI            AIConfig         `mapstructure:"ai"`
	RPC           RPCConfig        `mapstructure:"rpc"`
	GraphQL       GraphQLConfig    `mapstructure:"graphql"`
	MCP           MCPConfig        `mapstructure:"mcp"`
	Branching     BranchingConfig  `mapstructure:"branching"`
	Scaling       ScalingConfig    `mapstructure:"scaling"`
	Logging       LoggingConfig    `mapstructure:"logging"`
	Admin         AdminConfig      `mapstructure:"admin"`
	Tenants       TenantsConfig    `mapstructure:"tenants"`
	BaseURL       string           `mapstructure:"base_url"`        // Internal base URL (for server-to-server communication)
	PublicBaseURL string           `mapstructure:"public_base_url"` // Public base URL (for user-facing links, OAuth callbacks, etc.)
	Debug         bool             `mapstructure:"debug"`

	// EncryptionKey is used to encrypt sensitive data stored in the database (e.g., client keys, credentials)
	// Must be exactly 32 bytes for AES-256. Generate with: openssl rand -base64 32 | head -c 32
	// Only required if you configure providers (Email, AI) through the admin dashboard instead of env vars
	EncryptionKey      string `mapstructure:"encryption_key"`
	EncryptionKeyBytes []byte
}

// AdminConfig contains admin dashboard settings
type AdminConfig struct {
	Enabled bool `mapstructure:"enabled"` // Enable admin dashboard UI (React app). API routes are always available when setup_token is set.
}

// TenantsConfig contains tenant configuration settings
type TenantsConfig struct {
	Enabled        bool                       `mapstructure:"enabled"`
	DatabasePrefix string                     `mapstructure:"database_prefix"`
	MaxTenants     int                        `mapstructure:"max_tenants"`
	Pool           TenantPoolConfig           `mapstructure:"pool"`
	Migrations     TenantMigrationsConfig     `mapstructure:"migrations"`
	Declarative    TenantDeclarativeConfig    `mapstructure:"declarative"`
	Default        DefaultTenantConfig        `mapstructure:"default"`
	Configs        map[string]TenantOverrides `mapstructure:"configs"`
	ConfigDir      string                     `mapstructure:"config_dir"`
}

// TenantPoolConfig contains connection pool settings for tenant databases
type TenantPoolConfig struct {
	MaxTotalConnections int32         `mapstructure:"max_total_connections"`
	EvictionAge         time.Duration `mapstructure:"eviction_age"`
}

// TenantMigrationsConfig contains migration settings for tenant databases
type TenantMigrationsConfig struct {
	CheckInterval time.Duration `mapstructure:"check_interval"`
	OnCreate      bool          `mapstructure:"on_create"`
	OnAccess      bool          `mapstructure:"on_access"`
	Background    bool          `mapstructure:"background"`
}

// TenantDeclarativeConfig contains declarative schema settings for tenant databases
// This allows tenants to define their own schemas declaratively using SQL files
type TenantDeclarativeConfig struct {
	// Enabled controls whether tenant-specific declarative schemas are applied
	Enabled bool `mapstructure:"enabled"`
	// SchemaDir is the directory containing tenant schema files
	// Structure: {SchemaDir}/{tenant-slug}/public.sql
	// Example: schemas/acme-corp/public.sql
	SchemaDir string `mapstructure:"schema_dir"`
	// OnCreate applies declarative schemas when a tenant database is created
	OnCreate bool `mapstructure:"on_create"`
	// OnStartup applies declarative schemas on server startup (for existing tenants)
	OnStartup bool `mapstructure:"on_startup"`
	// AllowDestructive allows destructive schema changes (DROP, ALTER)
	AllowDestructive bool `mapstructure:"allow_destructive"`
}

// TenantOverrides holds configuration overrides for a specific tenant
// Only user-facing sections can be overridden; infrastructure sections remain global
type TenantOverrides struct {
	Auth      *AuthConfig      `mapstructure:"auth"`
	Storage   *StorageConfig   `mapstructure:"storage"`
	Email     *EmailConfig     `mapstructure:"email"`
	Functions *FunctionsConfig `mapstructure:"functions"`
	Jobs      *JobsConfig      `mapstructure:"jobs"`
	AI        *AIConfig        `mapstructure:"ai"`
	Realtime  *RealtimeConfig  `mapstructure:"realtime"`
	API       *APIConfig       `mapstructure:"api"`
	GraphQL   *GraphQLConfig   `mapstructure:"graphql"`
	RPC       *RPCConfig       `mapstructure:"rpc"`
}

// DefaultTenantConfig contains default tenant settings
type DefaultTenantConfig struct {
	Name           string `mapstructure:"name"`
	AnonKey        string `mapstructure:"anon_key"`
	ServiceKey     string `mapstructure:"service_key"`
	AnonKeyFile    string `mapstructure:"anon_key_file"`
	ServiceKeyFile string `mapstructure:"service_key_file"`
}

// DenoConfig contains Deno runtime settings for edge functions and background jobs
type DenoConfig struct {
	NpmRegistry string `mapstructure:"npm_registry"` // Custom npm registry URL (e.g., https://npm.your-company.com/)
	JsrRegistry string `mapstructure:"jsr_registry"` // Custom JSR registry URL (e.g., https://jsr.your-company.com/)
}

// ScalingConfig contains horizontal scaling settings for multi-instance deployments
type ScalingConfig struct {
	// WorkerOnly mode disables the API server and only runs job workers
	// Use this for dedicated worker containers that only process background jobs
	WorkerOnly bool `mapstructure:"worker_only"`

	// DisableScheduler prevents cron schedulers from running on this instance
	// Use this when running multiple instances to prevent duplicate scheduled jobs
	// Only one instance should run the scheduler (use leader election or manual config)
	DisableScheduler bool `mapstructure:"disable_scheduler"`

	// DisableRealtime prevents the realtime listener from starting
	// Useful for worker-only instances or when using an external realtime service
	DisableRealtime bool `mapstructure:"disable_realtime"`

	// EnableSchedulerLeaderElection enables automatic leader election for schedulers
	// When enabled, only one instance will run schedulers using PostgreSQL advisory locks
	// This is the recommended setting for multi-instance deployments
	EnableSchedulerLeaderElection bool `mapstructure:"enable_scheduler_leader_election"`

	// Backend for distributed state (rate limiting, pub/sub, sessions)
	// Options: "local" (single instance), "postgres", "redis"
	// "redis" works with Dragonfly (recommended), Redis, Valkey, KeyDB
	Backend string `mapstructure:"backend"`

	// RedisURL is the connection URL for Redis-compatible backends (Dragonfly recommended)
	// Only used when Backend is "redis"
	// Format: redis://[password@]host:port[/db]
	RedisURL string `mapstructure:"redis_url"`
}

// TracingConfig contains OpenTelemetry tracing settings
type TracingConfig struct {
	Enabled     bool    `mapstructure:"enabled"`      // Enable OpenTelemetry tracing
	Endpoint    string  `mapstructure:"endpoint"`     // OTLP endpoint (e.g., "localhost:4317")
	ServiceName string  `mapstructure:"service_name"` // Service name for traces (default: "fluxbase")
	Environment string  `mapstructure:"environment"`  // Environment name (development, staging, production)
	SampleRate  float64 `mapstructure:"sample_rate"`  // Sample rate 0.0-1.0 (1.0 = 100%)
	Insecure    bool    `mapstructure:"insecure"`     // Use insecure connection (for local dev)
}

// MetricsConfig contains Prometheus metrics settings
type MetricsConfig struct {
	Enabled bool   `mapstructure:"enabled"` // Enable Prometheus metrics endpoint
	Port    int    `mapstructure:"port"`    // Port for metrics server (default: 9090)
	Path    string `mapstructure:"path"`    // Path for metrics endpoint (default: /metrics)
}

// ServerConfig contains HTTP server settings
type ServerConfig struct {
	Address         string        `mapstructure:"address"`
	ReadTimeout     time.Duration `mapstructure:"read_timeout"`
	WriteTimeout    time.Duration `mapstructure:"write_timeout"`
	IdleTimeout     time.Duration `mapstructure:"idle_timeout"`
	BodyLimit       int           `mapstructure:"body_limit"`
	AllowedIPRanges []string      `mapstructure:"allowed_ip_ranges"` // Global IP CIDR ranges allowed to access server (empty = allow all)
	TrustedProxies  []string      `mapstructure:"trusted_proxies"`   // Trusted proxy IP ranges for X-Forwarded-For header validation (empty = trust none)

	// Per-endpoint body limits (if not specified, uses defaults from middleware)
	BodyLimits BodyLimitsConfig `mapstructure:"body_limits"`
}

// BodyLimitsConfig contains per-endpoint body size limits
type BodyLimitsConfig struct {
	// Enabled controls whether per-endpoint limits are enforced (default: true)
	Enabled bool `mapstructure:"enabled"`
	// DefaultLimit is used when no pattern matches (default: 1MB)
	DefaultLimit int64 `mapstructure:"default_limit"`
	// RESTLimit for REST API CRUD operations (default: 1MB)
	RESTLimit int64 `mapstructure:"rest_limit"`
	// AuthLimit for authentication endpoints (default: 64KB)
	AuthLimit int64 `mapstructure:"auth_limit"`
	// StorageLimit for file uploads (default: 500MB)
	StorageLimit int64 `mapstructure:"storage_limit"`
	// BulkLimit for bulk operations and RPC (default: 10MB)
	BulkLimit int64 `mapstructure:"bulk_limit"`
	// AdminLimit for admin endpoints (default: 5MB)
	AdminLimit int64 `mapstructure:"admin_limit"`
	// MaxJSONDepth limits nesting depth to prevent stack overflow (default: 64)
	MaxJSONDepth int `mapstructure:"max_json_depth"`
}

// DatabaseConfig contains PostgreSQL connection settings
type DatabaseConfig struct {
	Host               string        `mapstructure:"host"`
	Port               int           `mapstructure:"port"`
	User               string        `mapstructure:"user"`           // Database user for normal operations
	AdminUser          string        `mapstructure:"admin_user"`     // Optional admin user for migrations (defaults to User)
	Password           string        `mapstructure:"password"`       // Password for runtime user
	AdminPassword      string        `mapstructure:"admin_password"` // Optional password for admin user (defaults to Password)
	Database           string        `mapstructure:"database"`
	SSLMode            string        `mapstructure:"ssl_mode"`
	MaxConnections     int32         `mapstructure:"max_connections"`
	MinConnections     int32         `mapstructure:"min_connections"`
	MaxConnLifetime    time.Duration `mapstructure:"max_conn_lifetime"`
	MaxConnIdleTime    time.Duration `mapstructure:"max_conn_idle_time"`
	HealthCheck        time.Duration `mapstructure:"health_check_period"`
	UserMigrationsPath string        `mapstructure:"user_migrations_path"` // Path to user-provided migration files
	SlowQueryThreshold time.Duration `mapstructure:"slow_query_threshold"` // Log queries slower than this (default: 1s)
}

// AuthConfig contains authentication settings
type AuthConfig struct {
	JWTSecret           string        `mapstructure:"jwt_secret"`
	JWTExpiry           time.Duration `mapstructure:"jwt_expiry"`
	RefreshExpiry       time.Duration `mapstructure:"refresh_expiry"`
	ServiceRoleTTL      time.Duration `mapstructure:"service_role_ttl"` // TTL for service role tokens (default: 24h)
	AnonTTL             time.Duration `mapstructure:"anon_ttl"`         // TTL for anonymous tokens (default: 24h)
	MagicLinkExpiry     time.Duration `mapstructure:"magic_link_expiry"`
	PasswordResetExpiry time.Duration `mapstructure:"password_reset_expiry"`
	PasswordMinLen      int           `mapstructure:"password_min_length"`
	BcryptCost          int           `mapstructure:"bcrypt_cost"`
	SignupEnabled       bool          `mapstructure:"signup_enabled"`
	MagicLinkEnabled    bool          `mapstructure:"magic_link_enabled"`
	TOTPIssuer          string        `mapstructure:"totp_issuer"` // Issuer name displayed in authenticator apps for 2FA (e.g., "MyApp")

	// OAuth/OIDC provider configuration (unified for all providers)
	// Well-known providers (google, apple, microsoft) auto-detect issuer URLs
	// Custom providers require explicit issuer_url (supports base URLs like https://auth.domain.com or full .well-known URLs)
	OAuthProviders []OAuthProviderConfig `mapstructure:"oauth_providers"`

	// SAML SSO providers for enterprise authentication
	SAMLProviders []SAMLProviderConfig `mapstructure:"saml_providers"`

	// AllowUserClientKeys controls whether regular users can create their own client keys.
	// When false, only admins (service_role or instance_admin) can create/manage client keys,
	// and existing user-created keys are blocked from authenticating.
	// Default: true
	AllowUserClientKeys bool `mapstructure:"allow_user_client_keys"`

	// OAuthStateStorage configures how OAuth state tokens are stored.
	// "memory" - In-memory storage (default, single-instance only)
	// "database" - PostgreSQL storage (required for multi-instance deployments)
	// Default: "memory"
	OAuthStateStorage string `mapstructure:"oauth_state_storage"`
}

// SAMLProviderConfig represents a SAML 2.0 Identity Provider configuration
type SAMLProviderConfig struct {
	Name             string            `mapstructure:"name"`              // Provider name (e.g., "okta", "azure-ad")
	Enabled          bool              `mapstructure:"enabled"`           // Enable this provider
	IdPMetadataURL   string            `mapstructure:"idp_metadata_url"`  // IdP metadata URL (recommended)
	IdPMetadataXML   string            `mapstructure:"idp_metadata_xml"`  // IdP metadata XML (alternative to URL)
	EntityID         string            `mapstructure:"entity_id"`         // SP entity ID (unique identifier for this app)
	AcsURL           string            `mapstructure:"acs_url"`           // Assertion Consumer Service URL (callback)
	AttributeMapping map[string]string `mapstructure:"attribute_mapping"` // Map SAML attributes to user fields
	AutoCreateUsers  bool              `mapstructure:"auto_create_users"` // Create user if not exists
	DefaultRole      string            `mapstructure:"default_role"`      // Default role for new users (authenticated)

	// Security options
	AllowIDPInitiated        bool     `mapstructure:"allow_idp_initiated"`         // Allow IdP-initiated SSO (default: false for security)
	AllowedRedirectHosts     []string `mapstructure:"allowed_redirect_hosts"`      // Whitelist for RelayState redirect URLs
	AllowInsecureMetadataURL bool     `mapstructure:"allow_insecure_metadata_url"` // Allow HTTP metadata URLs (default: false)

	// Login targeting
	AllowDashboardLogin bool `mapstructure:"allow_dashboard_login"` // Allow for dashboard admin SSO (default: false)
	AllowAppLogin       bool `mapstructure:"allow_app_login"`       // Allow for app user authentication (default: true)

	// Role/Group-based access control
	RequiredGroups    []string `mapstructure:"required_groups"`     // User must be in at least ONE of these groups (OR logic)
	RequiredGroupsAll []string `mapstructure:"required_groups_all"` // User must be in ALL of these groups (AND logic)
	DeniedGroups      []string `mapstructure:"denied_groups"`       // Reject if user is in any of these groups
	GroupAttribute    string   `mapstructure:"group_attribute"`     // SAML attribute name for groups (default: "groups")

	// SP signing keys for SLO (Single Logout) - PEM-encoded
	SPCertificate string `mapstructure:"sp_certificate"` // PEM-encoded X.509 certificate for signing
	SPPrivateKey  string `mapstructure:"sp_private_key"` // PEM-encoded private key for signing

	// Logout signature verification
	RequireLogoutSignature *bool `mapstructure:"require_logout_signature"` // Require signed SAML logout messages (default: true)
}

// OAuthProviderConfig represents a unified OAuth/OIDC provider configuration
// Supports both well-known providers (Google, Apple, Microsoft) and custom providers
type OAuthProviderConfig struct {
	Name         string   `mapstructure:"name"`                    // Provider name (e.g., "google", "apple", "keycloak")
	Enabled      bool     `mapstructure:"enabled"`                 // Enable this provider (default: true)
	ClientID     string   `mapstructure:"client_id"`               // OAuth client ID (REQUIRED)
	ClientSecret string   `mapstructure:"client_secret,omitempty"` // Client secret (optional, can be stored in database)
	IssuerURL    string   `mapstructure:"issuer_url,omitempty"`    // OIDC issuer URL - supports base URLs (e.g., https://auth.domain.com) with auto-discovery or full .well-known URLs (auto-detected for well-known providers)
	Scopes       []string `mapstructure:"scopes,omitempty"`        // OAuth scopes
	DisplayName  string   `mapstructure:"display_name,omitempty"`  // Display name for UI

	// Login targeting
	AllowDashboardLogin bool `mapstructure:"allow_dashboard_login"` // Allow for dashboard admin SSO (default: false)
	AllowAppLogin       bool `mapstructure:"allow_app_login"`       // Allow for app user authentication (default: true)

	// Claims-based access control
	RequiredClaims map[string][]string `mapstructure:"required_claims"` // Claims that must be present in ID token, e.g., {"roles": ["admin"], "department": ["IT"]}
	DeniedClaims   map[string][]string `mapstructure:"denied_claims"`   // Deny access if these claim values are present
}

// SecurityConfig contains security-related settings
type SecurityConfig struct {
	EnableGlobalRateLimit bool `mapstructure:"enable_global_rate_limit"` // Global API rate limiting (100 req/min per IP)

	// Service role token revocation behavior
	ServiceRoleFailOpen bool `mapstructure:"service_role_fail_open"` // If false (default), fail-closed when revocation check fails (503). If true, fail-open for backward compatibility.

	// Admin setup security token
	SetupToken string `mapstructure:"setup_token"` // Required token for admin setup. If empty, admin dashboard is disabled.

	// Rate limiting for specific endpoints
	AdminSetupRateLimit         int           `mapstructure:"admin_setup_rate_limit"`          // Max attempts for admin setup
	AdminSetupRateWindow        time.Duration `mapstructure:"admin_setup_rate_window"`         // Time window for admin setup rate limit
	AdminLoginRateLimit         int           `mapstructure:"admin_login_rate_limit"`          // Max attempts for admin login
	AdminLoginRateWindow        time.Duration `mapstructure:"admin_login_rate_window"`         // Time window for admin login rate limit
	AuthLoginRateLimit          int           `mapstructure:"auth_login_rate_limit"`           // Max attempts for auth login
	AuthLoginRateWindow         time.Duration `mapstructure:"auth_login_rate_window"`          // Time window for auth login rate limit
	AuthSignupRateLimit         int           `mapstructure:"auth_signup_rate_limit"`          // Max attempts for auth signup
	AuthSignupRateWindow        time.Duration `mapstructure:"auth_signup_rate_window"`         // Time window for auth signup rate limit
	AuthPasswordResetRateLimit  int           `mapstructure:"auth_password_reset_rate_limit"`  // Max attempts for password reset
	AuthPasswordResetRateWindow time.Duration `mapstructure:"auth_password_reset_rate_window"` // Time window for password reset rate limit
	Auth2FARateLimit            int           `mapstructure:"auth_2fa_rate_limit"`             // Max attempts for 2FA verification
	Auth2FARateWindow           time.Duration `mapstructure:"auth_2fa_rate_window"`            // Time window for 2FA rate limit
	AuthRefreshRateLimit        int           `mapstructure:"auth_refresh_rate_limit"`         // Max attempts for token refresh
	AuthRefreshRateWindow       time.Duration `mapstructure:"auth_refresh_rate_window"`        // Time window for token refresh rate limit
	AuthMagicLinkRateLimit      int           `mapstructure:"auth_magic_link_rate_limit"`      // Max attempts for magic link
	AuthMagicLinkRateWindow     time.Duration `mapstructure:"auth_magic_link_rate_window"`     // Time window for magic link rate limit

	// Rate limiting for service_role tokens (bypassed by default, but can be enabled)
	ServiceRoleRateLimit  int           `mapstructure:"service_role_rate_limit"`  // Max requests for service_role tokens (0 = unlimited)
	ServiceRoleRateWindow time.Duration `mapstructure:"service_role_rate_window"` // Time window for service_role rate limit

	// CAPTCHA configuration for bot protection
	Captcha CaptchaConfig `mapstructure:"captcha"`
}

// CaptchaConfig contains CAPTCHA verification settings for bot protection
type CaptchaConfig struct {
	Enabled        bool     `mapstructure:"enabled"`         // Enable CAPTCHA verification
	Provider       string   `mapstructure:"provider"`        // Provider: hcaptcha, recaptcha_v3, turnstile, cap
	SiteKey        string   `mapstructure:"site_key"`        // Public site key (sent to frontend)
	SecretKey      string   `mapstructure:"secret_key"`      // Secret key for server-side verification
	ScoreThreshold float64  `mapstructure:"score_threshold"` // Min score for reCAPTCHA v3 (0.0-1.0, default 0.5)
	Endpoints      []string `mapstructure:"endpoints"`       // Endpoints requiring CAPTCHA: signup, login, password_reset, magic_link
	// Cap provider settings (self-hosted proof-of-work CAPTCHA)
	CapServerURL string `mapstructure:"cap_server_url"` // URL of Cap server (e.g., http://localhost:3000)
	CapAPIKey    string `mapstructure:"cap_api_key"`    // API key for Cap server authentication
	// Adaptive trust settings for intelligent CAPTCHA decisions
	AdaptiveTrust AdaptiveTrustConfig `mapstructure:"adaptive_trust"`
}

// AdaptiveTrustConfig contains settings for the adaptive CAPTCHA trust system
type AdaptiveTrustConfig struct {
	Enabled bool `mapstructure:"enabled"` // Enable adaptive trust (skip CAPTCHA for trusted users)

	// Trust token settings
	TrustTokenTTL     time.Duration `mapstructure:"trust_token_ttl"`      // How long a CAPTCHA solution is trusted (default: 15m)
	TrustTokenBoundIP bool          `mapstructure:"trust_token_bound_ip"` // Token only valid from same IP (default: true)

	// Challenge settings
	ChallengeExpiry time.Duration `mapstructure:"challenge_expiry"` // How long a challenge_id is valid (default: 5m)

	// Trust score threshold - score below this requires CAPTCHA
	CaptchaThreshold int `mapstructure:"captcha_threshold"` // Default: 50

	// Trust signal weights (positive signals)
	WeightKnownIP          int `mapstructure:"weight_known_ip"`          // User logged in from this IP before (default: 30)
	WeightKnownDevice      int `mapstructure:"weight_known_device"`      // Device fingerprint seen before (default: 25)
	WeightRecentCaptcha    int `mapstructure:"weight_recent_captcha"`    // Solved CAPTCHA recently (default: 40)
	WeightVerifiedEmail    int `mapstructure:"weight_verified_email"`    // Email address is confirmed (default: 15)
	WeightAccountAge       int `mapstructure:"weight_account_age"`       // Account older than 7 days (default: 10)
	WeightSuccessfulLogins int `mapstructure:"weight_successful_logins"` // 3+ successful logins (default: 10)
	WeightMFAEnabled       int `mapstructure:"weight_mfa_enabled"`       // User has MFA configured (default: 20)

	// Trust signal weights (negative signals)
	WeightNewIP          int `mapstructure:"weight_new_ip"`          // Never seen this IP (default: -30)
	WeightNewDevice      int `mapstructure:"weight_new_device"`      // Unknown device fingerprint (default: -25)
	WeightFailedAttempts int `mapstructure:"weight_failed_attempts"` // Recent failed login attempts (default: -20)

	// Per-endpoint overrides (some actions always need CAPTCHA regardless of trust)
	AlwaysRequireEndpoints []string `mapstructure:"always_require_endpoints"` // Endpoints that always require CAPTCHA (default: ["password_reset"])
}

// CORSConfig contains CORS settings
type CORSConfig struct {
	AllowedOrigins   []string `mapstructure:"allowed_origins"`   // List of allowed origins (use ["*"] for all)
	AllowedMethods   []string `mapstructure:"allowed_methods"`   // List of allowed HTTP methods
	AllowedHeaders   []string `mapstructure:"allowed_headers"`   // List of allowed headers
	ExposedHeaders   []string `mapstructure:"exposed_headers"`   // List of exposed headers
	AllowCredentials bool     `mapstructure:"allow_credentials"` // Allow credentials (cookies, authorization headers)
	MaxAge           int      `mapstructure:"max_age"`           // Max age for preflight cache in seconds
}

// StorageConfig contains file storage settings
type StorageConfig struct {
	Enabled          bool     `mapstructure:"enabled"`  // Enable storage functionality
	Provider         string   `mapstructure:"provider"` // local or s3
	LocalPath        string   `mapstructure:"local_path"`
	S3Endpoint       string   `mapstructure:"s3_endpoint"`
	S3AccessKey      string   `mapstructure:"s3_access_key"`
	S3SecretKey      string   `mapstructure:"s3_secret_key"`
	S3Bucket         string   `mapstructure:"s3_bucket"`
	S3Region         string   `mapstructure:"s3_region"`
	S3ForcePathStyle bool     `mapstructure:"s3_force_path_style"` // Use path-style addressing (required for MinIO, R2, Spaces, etc.)
	DefaultBuckets   []string `mapstructure:"default_buckets"`     // Buckets to auto-create on startup
	MaxUploadSize    int64    `mapstructure:"max_upload_size"`

	// Image transformation settings
	Transforms TransformConfig `mapstructure:"transforms"`
}

// TransformConfig contains image transformation settings
type TransformConfig struct {
	Enabled        bool     `mapstructure:"enabled"`         // Enable on-the-fly image transformations
	DefaultQuality int      `mapstructure:"default_quality"` // Default output quality (1-100)
	MaxWidth       int      `mapstructure:"max_width"`       // Maximum output width in pixels
	MaxHeight      int      `mapstructure:"max_height"`      // Maximum output height in pixels
	AllowedFormats []string `mapstructure:"allowed_formats"` // Allowed output formats (webp, jpg, png, avif)

	// Security settings
	MaxTotalPixels int           `mapstructure:"max_total_pixels"` // Maximum total pixels (width * height), default 16M
	BucketSize     int           `mapstructure:"bucket_size"`      // Dimension bucketing size (default 50px)
	RateLimit      int           `mapstructure:"rate_limit"`       // Transforms per minute per user (default 60)
	Timeout        time.Duration `mapstructure:"timeout"`          // Max transform duration (default 30s)
	MaxConcurrent  int           `mapstructure:"max_concurrent"`   // Max concurrent transforms (default 4)

	// Caching settings
	CacheEnabled bool          `mapstructure:"cache_enabled"`  // Enable transform caching
	CacheTTL     time.Duration `mapstructure:"cache_ttl"`      // Cache TTL (default 24h)
	CacheMaxSize int64         `mapstructure:"cache_max_size"` // Max cache size in bytes (default 1GB)
}

// RealtimeConfig contains realtime/websocket settings
type RealtimeConfig struct {
	Enabled                bool          `mapstructure:"enabled"`
	MaxConnections         int           `mapstructure:"max_connections"`
	MaxConnectionsPerUser  int           `mapstructure:"max_connections_per_user"`  // Max connections per authenticated user (0 = unlimited)
	MaxConnectionsPerIP    int           `mapstructure:"max_connections_per_ip"`    // Max connections per IP for anonymous connections (0 = unlimited)
	RLSCacheSize           int           `mapstructure:"rls_cache_size"`            // Maximum entries in RLS cache (default: 100000)
	RLSCacheTTL            time.Duration `mapstructure:"rls_cache_ttl"`             // TTL for RLS cache entries (default: 30s)
	ListenerPoolSize       int           `mapstructure:"listener_pool_size"`        // Number of LISTEN connections for redundancy (default: 2)
	NotificationWorkers    int           `mapstructure:"notification_workers"`      // Number of workers for parallel notification processing (default: 4)
	ClientMessageQueueSize int           `mapstructure:"client_message_queue_size"` // Size of per-client message queue for async sending (default: 256)
	SlowClientThreshold    int           `mapstructure:"slow_client_threshold"`     // Queue length threshold for slow client detection (default: 100)
	SlowClientTimeout      time.Duration `mapstructure:"slow_client_timeout"`       // Duration before disconnecting slow clients (default: 30s)
}

// EmailConfig contains email/SMTP settings
type EmailConfig struct {
	Enabled        bool   `mapstructure:"enabled"`
	Provider       string `mapstructure:"provider"` // smtp, sendgrid, mailgun, ses
	FromAddress    string `mapstructure:"from_address"`
	FromName       string `mapstructure:"from_name"`
	ReplyToAddress string `mapstructure:"reply_to_address"`

	// SMTP Settings
	SMTPHost     string `mapstructure:"smtp_host"`
	SMTPPort     int    `mapstructure:"smtp_port"`
	SMTPUsername string `mapstructure:"smtp_username"`
	SMTPPassword string `mapstructure:"smtp_password"`
	SMTPTLS      bool   `mapstructure:"smtp_tls"`

	// SendGrid Settings
	SendGridAPIKey string `mapstructure:"sendgrid_api_key"`

	// Mailgun Settings
	MailgunAPIKey string `mapstructure:"mailgun_api_key"`
	MailgunDomain string `mapstructure:"mailgun_domain"`

	// AWS SES Settings
	SESAccessKey string `mapstructure:"ses_access_key"`
	SESSecretKey string `mapstructure:"ses_secret_key"`
	SESRegion    string `mapstructure:"ses_region"`

	// Templates
	MagicLinkTemplate     string `mapstructure:"magic_link_template"`
	VerificationTemplate  string `mapstructure:"verification_template"`
	PasswordResetTemplate string `mapstructure:"password_reset_template"`
}

// FunctionsConfig contains edge functions settings
type FunctionsConfig struct {
	Enabled             bool     `mapstructure:"enabled"`
	FunctionsDir        string   `mapstructure:"functions_dir"`
	AutoLoadOnBoot      bool     `mapstructure:"auto_load_on_boot"`      // Load functions from filesystem at boot
	DefaultTimeout      int      `mapstructure:"default_timeout"`        // seconds
	MaxTimeout          int      `mapstructure:"max_timeout"`            // seconds
	DefaultMemoryLimit  int      `mapstructure:"default_memory_limit"`   // MB
	MaxMemoryLimit      int      `mapstructure:"max_memory_limit"`       // MB
	MaxOutputSize       int      `mapstructure:"max_output_size"`        // Max output size in bytes (0 = unlimited, default: 10MB)
	SyncAllowedIPRanges []string `mapstructure:"sync_allowed_ip_ranges"` // IP CIDR ranges allowed to sync functions
}

// APIConfig contains REST API settings
type APIConfig struct {
	MaxPageSize     int `mapstructure:"max_page_size"`     // Max rows per request (-1 = unlimited)
	MaxTotalResults int `mapstructure:"max_total_results"` // Max total retrievable rows via offset+limit (-1 = unlimited)
	DefaultPageSize int `mapstructure:"default_page_size"` // Auto-applied when no limit specified (-1 = no default)
	MaxBatchSize    int `mapstructure:"max_batch_size"`    // Max records in batch insert/update (-1 = unlimited, default: 1000)
}

// JobsConfig contains long-running background jobs settings
type JobsConfig struct {
	Enabled                   bool          `mapstructure:"enabled"`
	JobsDir                   string        `mapstructure:"jobs_dir"`
	AutoLoadOnBoot            bool          `mapstructure:"auto_load_on_boot"`            // Load jobs from filesystem at boot
	WorkerMode                string        `mapstructure:"worker_mode"`                  // "embedded", "standalone", "disabled"
	EmbeddedWorkerCount       int           `mapstructure:"embedded_worker_count"`        // Number of embedded workers
	MaxConcurrentPerWorker    int           `mapstructure:"max_concurrent_per_worker"`    // Max concurrent jobs per worker
	MaxConcurrentPerNamespace int           `mapstructure:"max_concurrent_per_namespace"` // Max concurrent jobs per namespace
	DefaultMaxDuration        time.Duration `mapstructure:"default_max_duration"`         // Default job timeout
	MaxMaxDuration            time.Duration `mapstructure:"max_max_duration"`             // Maximum allowed job timeout
	DefaultProgressTimeout    time.Duration `mapstructure:"default_progress_timeout"`     // Default progress timeout
	PollInterval              time.Duration `mapstructure:"poll_interval"`                // Worker poll interval
	WorkerHeartbeatInterval   time.Duration `mapstructure:"worker_heartbeat_interval"`    // Worker heartbeat interval
	WorkerTimeout             time.Duration `mapstructure:"worker_timeout"`               // Worker considered dead after this
	SyncAllowedIPRanges       []string      `mapstructure:"sync_allowed_ip_ranges"`       // IP CIDR ranges allowed to sync jobs
	GracefulShutdownTimeout   time.Duration `mapstructure:"graceful_shutdown_timeout"`    // Time to wait for running jobs during shutdown (default: 5m)
}

// MigrationsConfig contains migrations API security settings
type MigrationsConfig struct {
	Enabled         bool     `mapstructure:"enabled"`           // Enable migrations API (enabled by default)
	AllowedIPRanges []string `mapstructure:"allowed_ip_ranges"` // IP CIDR ranges allowed to access migrations API
}

// AIConfig contains AI chatbot settings
type AIConfig struct {
	Enabled              bool          `mapstructure:"enabled"`                // Enable AI chatbot functionality
	ChatbotsDir          string        `mapstructure:"chatbots_dir"`           // Directory for chatbot definitions
	AutoLoadOnBoot       bool          `mapstructure:"auto_load_on_boot"`      // Load chatbots from filesystem at boot
	DefaultMaxTokens     int           `mapstructure:"default_max_tokens"`     // Default max tokens per request
	DefaultModel         string        `mapstructure:"default_model"`          // Default AI model
	QueryTimeout         time.Duration `mapstructure:"query_timeout"`          // SQL query execution timeout
	MaxRowsPerQuery      int           `mapstructure:"max_rows_per_query"`     // Max rows returned per query
	ConversationCacheTTL time.Duration `mapstructure:"conversation_cache_ttl"` // TTL for conversation cache
	MaxConversationTurns int           `mapstructure:"max_conversation_turns"` // Max turns per conversation
	SyncAllowedIPRanges  []string      `mapstructure:"sync_allowed_ip_ranges"` // IP CIDR ranges allowed to sync chatbots

	// Provider Configuration (read-only in dashboard when set)
	// If ProviderType is set, a config-based provider will be added to the list
	ProviderType  string `mapstructure:"provider_type"`  // Provider type: openai, azure, ollama
	ProviderName  string `mapstructure:"provider_name"`  // Display name for config provider
	ProviderModel string `mapstructure:"provider_model"` // Default model for config provider

	// Embedding Configuration (for vector search)
	EmbeddingEnabled  bool   `mapstructure:"embedding_enabled"`  // Enable embedding generation for vector search
	EmbeddingProvider string `mapstructure:"embedding_provider"` // Embedding provider: openai, azure, ollama (defaults to ProviderType)
	EmbeddingModel    string `mapstructure:"embedding_model"`    // Embedding model: text-embedding-3-small, text-embedding-3-large, etc.

	// OpenAI Settings
	OpenAIAPIKey         string `mapstructure:"openai_api_key"`
	OpenAIOrganizationID string `mapstructure:"openai_organization_id"`
	OpenAIBaseURL        string `mapstructure:"openai_base_url"`

	// Azure Settings
	AzureAPIKey         string `mapstructure:"azure_api_key"`
	AzureEndpoint       string `mapstructure:"azure_endpoint"`
	AzureDeploymentName string `mapstructure:"azure_deployment_name"`
	AzureAPIVersion     string `mapstructure:"azure_api_version"`

	// Azure Embedding Settings (optional, falls back to Azure Settings)
	AzureEmbeddingDeploymentName string `mapstructure:"azure_embedding_deployment_name"` // Separate deployment for embeddings

	// Ollama Settings
	OllamaEndpoint string `mapstructure:"ollama_endpoint"`
	OllamaModel    string `mapstructure:"ollama_model"`

	// OCR Configuration (for image-based PDF extraction in knowledge bases)
	OCREnabled   bool     `mapstructure:"ocr_enabled"`   // Enable OCR for image-based PDFs
	OCRProvider  string   `mapstructure:"ocr_provider"`  // OCR provider: tesseract
	OCRLanguages []string `mapstructure:"ocr_languages"` // Default languages for OCR (e.g., ["eng", "deu"])

	// RAG Configuration (for retrieval-augmented generation)
	RAGGraphBoostWeight float64 `mapstructure:"rag_graph_boost_weight"` // How much to weight entity matches vs vector similarity (0.0-1.0, default 0)
}

// RPCConfig contains RPC (Remote Procedure Call) configuration
type RPCConfig struct {
	Enabled             bool     `mapstructure:"enabled"`                // Enable RPC functionality
	ProceduresDir       string   `mapstructure:"procedures_dir"`         // Directory for RPC procedure definitions
	AutoLoadOnBoot      bool     `mapstructure:"auto_load_on_boot"`      // Load procedures from filesystem at boot
	DefaultMaxRows      int      `mapstructure:"default_max_rows"`       // Default max rows returned
	SyncAllowedIPRanges []string `mapstructure:"sync_allowed_ip_ranges"` // IP CIDR ranges allowed to sync procedures
}

// LoggingConfig contains central logging configuration
type LoggingConfig struct {
	// Console output settings
	ConsoleEnabled bool   `mapstructure:"console_enabled"` // Enable console output (default: true)
	ConsoleLevel   string `mapstructure:"console_level"`   // Minimum level for console: trace, debug, info, warn, error
	ConsoleFormat  string `mapstructure:"console_format"`  // Output format: json or console

	// Backend settings
	Backend string `mapstructure:"backend"` // Primary backend: postgres (default), s3, local, timescaledb, loki, elasticsearch, opensearch, clickhouse

	// S3 backend settings (when backend is "s3")
	S3Bucket string `mapstructure:"s3_bucket"` // S3 bucket for logs
	S3Prefix string `mapstructure:"s3_prefix"` // Prefix for log objects (default: "logs")

	// Local backend settings (when backend is "local")
	LocalPath string `mapstructure:"local_path"` // Directory for log files (default: "./logs")

	// TimescaleDB settings (when backend is "timescaledb")
	TimescaleDBEnabled       bool          `mapstructure:"timescaledb_enabled"`
	TimescaleDBCompression   bool          `mapstructure:"timescaledb_compression"`
	TimescaleDBCompressAfter time.Duration `mapstructure:"timescaledb_compress_after"` // Compress after this duration (default: 7d)
	TimescaleDBRetainAfter   time.Duration `mapstructure:"timescaledb_retain_after"`   // Drop chunks older than this (default: 90d)

	// Loki settings (when backend is "loki")
	LokiURL      string   `mapstructure:"loki_url"`       // Loki server URL (required)
	LokiUsername string   `mapstructure:"loki_username"`  // Username for basic auth
	LokiPassword string   `mapstructure:"loki_password"`  // Password for basic auth
	LokiTenantID string   `mapstructure:"loki_tenant_id"` // Tenant ID for multi-tenant Loki
	LokiLabels   []string `mapstructure:"loki_labels"`    // Static labels to add to all logs

	// Elasticsearch settings (when backend is "elasticsearch")
	ElasticsearchURLs     []string `mapstructure:"elasticsearch_urls"`     // Elasticsearch node URLs
	ElasticsearchUsername string   `mapstructure:"elasticsearch_username"` // Username for basic auth
	ElasticsearchPassword string   `mapstructure:"elasticsearch_password"` // Password for basic auth
	ElasticsearchIndex    string   `mapstructure:"elasticsearch_index"`    // Index name pattern (default: "fluxbase-logs")
	ElasticsearchVersion  int      `mapstructure:"elasticsearch_version"`  // Major version: 8 or 9 (default: 8)

	// OpenSearch settings (when backend is "opensearch")
	OpenSearchURLs     []string `mapstructure:"opensearch_urls"`     // OpenSearch node URLs
	OpenSearchUsername string   `mapstructure:"opensearch_username"` // Username for basic auth
	OpenSearchPassword string   `mapstructure:"opensearch_password"` // Password for basic auth
	OpenSearchIndex    string   `mapstructure:"opensearch_index"`    // Index name pattern (default: "fluxbase-logs")
	OpenSearchVersion  int      `mapstructure:"opensearch_version"`  // Major version (default: 2)

	// ClickHouse settings (when backend is "clickhouse")
	ClickHouseAddresses []string `mapstructure:"clickhouse_addresses"` // ClickHouse node addresses (default: ["localhost:9000"])
	ClickHouseUsername  string   `mapstructure:"clickhouse_username"`  // Username (default: "default")
	ClickHousePassword  string   `mapstructure:"clickhouse_password"`  // Password
	ClickHouseDatabase  string   `mapstructure:"clickhouse_database"`  // Database name (default: "fluxbase")
	ClickHouseTable     string   `mapstructure:"clickhouse_table"`     // Table name (default: "logs")
	ClickHouseTTL       int      `mapstructure:"clickhouse_ttl_days"`  // TTL in days (default: 30)

	// Batching settings
	BatchSize     int           `mapstructure:"batch_size"`     // Number of entries per batch (default: 100)
	FlushInterval time.Duration `mapstructure:"flush_interval"` // Max time before flushing (default: 1s)
	BufferSize    int           `mapstructure:"buffer_size"`    // Async buffer size (default: 10000)

	// PubSub notifications (for realtime streaming)
	PubSubEnabled bool `mapstructure:"pubsub_enabled"` // Enable PubSub notifications for execution logs

	// Retention settings (days, 0 = keep forever)
	SystemRetentionDays    int `mapstructure:"system_retention_days"`    // App/system logs (default: 7)
	HTTPRetentionDays      int `mapstructure:"http_retention_days"`      // HTTP access logs (default: 30)
	SecurityRetentionDays  int `mapstructure:"security_retention_days"`  // Security/audit logs (default: 90)
	ExecutionRetentionDays int `mapstructure:"execution_retention_days"` // Function/job/RPC logs (default: 30)
	AIRetentionDays        int `mapstructure:"ai_retention_days"`        // AI query audit logs (default: 30)

	// Retention service settings
	RetentionEnabled       bool          `mapstructure:"retention_enabled"`        // Enable retention cleanup (default: true)
	RetentionCheckInterval time.Duration `mapstructure:"retention_check_interval"` // Interval between cleanup checks (default: 24h)

	// Custom categories
	CustomCategories    []string `mapstructure:"custom_categories"`     // List of allowed custom category names
	CustomRetentionDays int      `mapstructure:"custom_retention_days"` // Retention for custom categories (default: 30)
}

// Load loads configuration from file and environment variables
func Load() (*Config, error) {
	// Load .env file if it exists (for local development)
	if err := loadEnvFile(); err != nil {
		log.Debug().Msg("No .env file found - using environment variables and defaults")
	}

	// Set defaults
	setDefaults()

	// Enable environment variable support with underscore replacer
	viper.AutomaticEnv()
	viper.SetEnvPrefix("FLUXBASE")
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	// Try to load config file from specific paths (in order of priority)
	// This is more explicit than using SetConfigName which would also match .example files
	configPaths := []string{
		"./fluxbase.yaml",
		"./fluxbase.yml",
		"./config/fluxbase.yaml",
		"./config/fluxbase.yml",
		"/etc/fluxbase/fluxbase.yaml",
		"/etc/fluxbase/fluxbase.yml",
	}

	var configLoaded bool
	for _, configPath := range configPaths {
		if _, err := os.Stat(configPath); err == nil {
			viper.SetConfigFile(configPath)
			if err := viper.ReadInConfig(); err != nil {
				log.Warn().Err(err).Str("file", configPath).Msg("Config file found but could not be parsed, using environment variables and defaults")
			} else {
				log.Info().Str("file", configPath).Msg("Config file loaded")
				configLoaded = true
			}
			break
		}
	}

	if !configLoaded {
		log.Info().Msg("No config file found, using environment variables and defaults")
	}

	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("unable to decode config: %w", err)
	}

	// Validate configuration
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return &config, nil
}

// loadEnvFile loads environment variables from .env file
func loadEnvFile() error {
	// Check multiple locations for .env file
	locations := []string{
		".env",
		".env.local",
		"../.env", // For when running from subdirectories
	}

	for _, location := range locations {
		if _, err := os.Stat(location); err == nil {
			if err := godotenv.Load(location); err != nil {
				return fmt.Errorf("error loading .env file from %s: %w", location, err)
			}
			log.Info().Str("file", location).Msg(".env file loaded")
			return nil
		}
	}

	return fmt.Errorf("no .env file found")
}

// Validate validates the configuration
func (c *Config) Validate() error {
	// Validate server configuration
	if err := c.Server.Validate(); err != nil {
		return fmt.Errorf("server configuration error: %w", err)
	}

	// Validate database configuration
	if err := c.Database.Validate(); err != nil {
		return fmt.Errorf("database configuration error: %w", err)
	}

	// Validate auth configuration
	if err := c.Auth.Validate(); err != nil {
		return fmt.Errorf("auth configuration error: %w", err)
	}

	// Validate storage configuration
	if err := c.Storage.Validate(); err != nil {
		return fmt.Errorf("storage configuration error: %w", err)
	}

	// Validate security configuration
	if err := c.Security.Validate(); err != nil {
		return fmt.Errorf("security configuration error: %w", err)
	}

	// Validate email configuration if enabled
	if c.Email.Enabled {
		if err := c.Email.Validate(); err != nil {
			return fmt.Errorf("email configuration error: %w", err)
		}
	}

	// Validate functions configuration if enabled
	if c.Functions.Enabled {
		if err := c.Functions.Validate(); err != nil {
			return fmt.Errorf("functions configuration error: %w", err)
		}
	}

	// Validate API configuration
	if err := c.API.Validate(); err != nil {
		return fmt.Errorf("api configuration error: %w", err)
	}

	// Validate jobs configuration if enabled
	if c.Jobs.Enabled {
		if err := c.Jobs.Validate(); err != nil {
			return fmt.Errorf("jobs configuration error: %w", err)
		}
	}

	// Validate tracing configuration if enabled
	if c.Tracing.Enabled {
		if err := c.Tracing.Validate(); err != nil {
			return fmt.Errorf("tracing configuration error: %w", err)
		}
	}

	// Validate metrics configuration if enabled
	if c.Metrics.Enabled {
		if err := c.Metrics.Validate(); err != nil {
			return fmt.Errorf("metrics configuration error: %w", err)
		}
	}

	// Validate AI configuration if enabled
	if c.AI.Enabled {
		if err := c.AI.Validate(); err != nil {
			return fmt.Errorf("ai configuration error: %w", err)
		}
	}

	// Validate GraphQL configuration if enabled
	if c.GraphQL.Enabled {
		if err := c.GraphQL.Validate(); err != nil {
			return fmt.Errorf("graphql configuration error: %w", err)
		}
	}

	// Validate MCP configuration if enabled
	if c.MCP.Enabled {
		if err := c.MCP.Validate(); err != nil {
			return fmt.Errorf("mcp configuration error: %w", err)
		}
	}

	// Validate branching configuration if enabled
	if c.Branching.Enabled {
		if err := c.Branching.Validate(); err != nil {
			return fmt.Errorf("branching configuration error: %w", err)
		}
	}

	// Validate scaling configuration
	if err := c.Scaling.Validate(); err != nil {
		return fmt.Errorf("scaling configuration error: %w", err)
	}

	// Validate logging configuration
	if err := c.Logging.Validate(); err != nil {
		return fmt.Errorf("logging configuration error: %w", err)
	}

	// Validate encryption key - required for secure secrets storage
	if c.EncryptionKey == "" {
		return fmt.Errorf("encryption_key is required for AES-256 encryption (must be exactly 32 bytes)")
	}
	if len(c.EncryptionKey) != 32 {
		return fmt.Errorf("encryption_key must be exactly 32 bytes for AES-256, got %d bytes", len(c.EncryptionKey))
	}
	c.EncryptionKeyBytes = []byte(c.EncryptionKey)

	// Validate base URL
	if c.BaseURL != "" {
		parsedURL, err := url.Parse(c.BaseURL)
		if err != nil {
			return fmt.Errorf("invalid base_url: %w", err)
		}
		if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
			return fmt.Errorf("base_url must use http or https scheme, got: %s", parsedURL.Scheme)
		}
	}

	// Validate public base URL if set
	if c.PublicBaseURL != "" {
		parsedURL, err := url.Parse(c.PublicBaseURL)
		if err != nil {
			return fmt.Errorf("invalid public_base_url: %w", err)
		}
		if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
			return fmt.Errorf("public_base_url must use http or https scheme, got: %s", parsedURL.Scheme)
		}
	}

	return nil
}

// GetPublicBaseURL returns the public-facing base URL.
// If PublicBaseURL is set, it returns that; otherwise, it falls back to BaseURL.
// This should be used for all user-facing URLs (magic links, OAuth callbacks, invitation links, etc.)
func (c *Config) GetPublicBaseURL() string {
	if c.PublicBaseURL != "" {
		return c.PublicBaseURL
	}
	return c.BaseURL
}

// Validate validates server configuration
func (sc *ServerConfig) Validate() error {
	if sc.Address == "" {
		return fmt.Errorf("server address cannot be empty")
	}

	// Validate timeouts are positive
	if sc.ReadTimeout <= 0 {
		return fmt.Errorf("read_timeout must be positive, got: %v", sc.ReadTimeout)
	}
	if sc.WriteTimeout <= 0 {
		return fmt.Errorf("write_timeout must be positive, got: %v", sc.WriteTimeout)
	}
	if sc.IdleTimeout <= 0 {
		return fmt.Errorf("idle_timeout must be positive, got: %v", sc.IdleTimeout)
	}

	// Validate body limit
	if sc.BodyLimit <= 0 {
		return fmt.Errorf("body_limit must be positive, got: %d", sc.BodyLimit)
	}

	return nil
}

// Validate validates database configuration
func (dc *DatabaseConfig) Validate() error {
	if dc.Host == "" {
		return fmt.Errorf("database host is required")
	}

	if dc.Port < 1 || dc.Port > 65535 {
		return fmt.Errorf("database port must be between 1 and 65535, got: %d", dc.Port)
	}

	if dc.User == "" {
		return fmt.Errorf("database user is required")
	}

	// If AdminUser is not set, default it to User
	if dc.AdminUser == "" {
		dc.AdminUser = dc.User
	}

	if dc.Database == "" {
		return fmt.Errorf("database name is required")
	}

	// Validate SSL mode
	validSSLModes := []string{"disable", "allow", "prefer", "require", "verify-ca", "verify-full"}
	sslModeValid := false
	for _, mode := range validSSLModes {
		if dc.SSLMode == mode {
			sslModeValid = true
			break
		}
	}
	if !sslModeValid {
		return fmt.Errorf("invalid ssl_mode: %s (must be one of: %v)", dc.SSLMode, validSSLModes)
	}
	if dc.SSLMode == "disable" {
		log.Warn().Msg("database.ssl_mode is 'disable' — database connections are unencrypted. Set ssl_mode to 'require' or higher in production.")
	}

	// Validate connection pool settings
	// MaxConnections must be between 1 and 1000 to prevent resource exhaustion
	if dc.MaxConnections < 1 {
		return fmt.Errorf("max_connections must be at least 1, got: %d", dc.MaxConnections)
	}
	if dc.MaxConnections > 1000 {
		return fmt.Errorf("max_connections must be at most 1000, got: %d", dc.MaxConnections)
	}

	// MinConnections must be non-negative and cannot exceed MaxConnections
	if dc.MinConnections < 0 {
		return fmt.Errorf("min_connections must be at least 0, got: %d", dc.MinConnections)
	}

	if dc.MinConnections > dc.MaxConnections {
		return fmt.Errorf("min_connections (%d) cannot exceed max_connections (%d)",
			dc.MinConnections, dc.MaxConnections)
	}

	// Validate timeouts are positive
	if dc.MaxConnLifetime <= 0 {
		return fmt.Errorf("max_conn_lifetime must be positive, got: %v", dc.MaxConnLifetime)
	}
	if dc.MaxConnIdleTime <= 0 {
		return fmt.Errorf("max_conn_idle_time must be positive, got: %v", dc.MaxConnIdleTime)
	}
	if dc.HealthCheck <= 0 {
		return fmt.Errorf("health_check_period must be positive, got: %v", dc.HealthCheck)
	}

	return nil
}

// Validate validates auth configuration
func (ac *AuthConfig) Validate() error {
	if ac.JWTSecret == "" {
		return fmt.Errorf("jwt_secret is required")
	}

	if ac.JWTSecret == "your-secret-key-change-in-production" {
		return fmt.Errorf("please set a secure JWT secret (current value is the default insecure value)")
	}

	// Validate JWT secret length (should be at least 32 characters for security)
	if len(ac.JWTSecret) < 32 {
		log.Warn().Msg("JWT secret is shorter than 32 characters - consider using a longer secret for better security")
	}

	// SECURITY: Validate JWT secret entropy to prevent weak secrets
	// Calculate Shannon entropy of the secret to ensure it has sufficient randomness
	entropy := calculateEntropy(ac.JWTSecret)
	// Minimum 4.5 bits per character Shannon entropy (catches repetitive patterns)
	// For reference: random alphanumeric = ~6 bits/char, all same = 0 bits, alternating = ~1 bit
	// 4.5 bits/char ensures good character variety without being overly strict
	minEntropyPerChar := 4.5
	if entropy < minEntropyPerChar {
		return fmt.Errorf("jwt_secret has insufficient entropy (%.2f bits < %.2f bits per character minimum). Generate a secure random secret: openssl rand -base64 32 | head -c 32", entropy, minEntropyPerChar)
	}

	// Validate expiry durations are positive
	if ac.JWTExpiry <= 0 {
		return fmt.Errorf("jwt_expiry must be positive, got: %v", ac.JWTExpiry)
	}
	if ac.RefreshExpiry <= 0 {
		return fmt.Errorf("refresh_expiry must be positive, got: %v", ac.RefreshExpiry)
	}
	if ac.MagicLinkExpiry <= 0 {
		return fmt.Errorf("magic_link_expiry must be positive, got: %v", ac.MagicLinkExpiry)
	}
	if ac.PasswordResetExpiry <= 0 {
		return fmt.Errorf("password_reset_expiry must be positive, got: %v", ac.PasswordResetExpiry)
	}

	// Validate password settings
	if ac.PasswordMinLen < 1 {
		return fmt.Errorf("password_min_length must be at least 1, got: %d", ac.PasswordMinLen)
	}
	if ac.PasswordMinLen < 8 {
		log.Warn().Int("min_length", ac.PasswordMinLen).Msg("Password minimum length is less than 8 - consider increasing for better security")
	}

	// Validate bcrypt cost (valid range is 4-31, recommended is 10-14)
	if ac.BcryptCost < 4 || ac.BcryptCost > 31 {
		return fmt.Errorf("bcrypt_cost must be between 4 and 31, got: %d", ac.BcryptCost)
	}

	// Validate OAuth providers
	providerNames := make(map[string]bool)
	for i, provider := range ac.OAuthProviders {
		if err := provider.Validate(); err != nil {
			return fmt.Errorf("oauth_providers[%d]: %w", i, err)
		}

		// Check for duplicate provider names
		if providerNames[provider.Name] {
			return fmt.Errorf("duplicate OAuth provider name: %s", provider.Name)
		}
		providerNames[provider.Name] = true
	}

	return nil
}

// Validate validates OAuth provider configuration
func (opc *OAuthProviderConfig) Validate() error {
	if opc.Name == "" {
		return fmt.Errorf("oauth provider name is required")
	}
	if opc.ClientID == "" {
		return fmt.Errorf("oauth provider '%s': client_id is required", opc.Name)
	}

	// Normalize name to lowercase
	opc.Name = strings.ToLower(opc.Name)

	// Check if well-known provider
	wellKnown := map[string]bool{
		"google":    true,
		"apple":     true,
		"microsoft": true,
	}

	// Custom providers require issuer_url
	if !wellKnown[opc.Name] && opc.IssuerURL == "" {
		return fmt.Errorf("oauth provider '%s': issuer_url is required for custom providers", opc.Name)
	}

	return nil
}

// Validate validates storage configuration
func (sc *StorageConfig) Validate() error {
	if sc.Provider != "local" && sc.Provider != "s3" {
		return fmt.Errorf("storage provider must be 'local' or 's3', got: %s", sc.Provider)
	}

	if sc.Provider == "local" {
		if sc.LocalPath == "" {
			return fmt.Errorf("local_path is required when using local storage provider")
		}
	}

	if sc.Provider == "s3" {
		if sc.S3Endpoint == "" {
			return fmt.Errorf("s3_endpoint is required when using S3 storage provider")
		}
		if sc.S3AccessKey == "" {
			return fmt.Errorf("s3_access_key is required when using S3 storage provider")
		}
		if sc.S3SecretKey == "" {
			return fmt.Errorf("s3_secret_key is required when using S3 storage provider")
		}
		if sc.S3Bucket == "" {
			return fmt.Errorf("s3_bucket is required when using S3 storage provider")
		}
		// S3Region is optional for some S3-compatible services
	}

	// Validate max upload size
	if sc.MaxUploadSize <= 0 {
		return fmt.Errorf("max_upload_size must be positive, got: %d", sc.MaxUploadSize)
	}

	return nil
}

// ConnectionString returns the PostgreSQL connection string using the runtime user
//
// Deprecated: Use RuntimeConnectionString() or AdminConnectionString() instead
func (dc *DatabaseConfig) ConnectionString() string {
	return dc.RuntimeConnectionString()
}

// RuntimeConnectionString returns the PostgreSQL connection string for the runtime user
// Uses url.URL for secure credential handling to prevent password injection
func (dc *DatabaseConfig) RuntimeConnectionString() string {
	return dc.buildSecureConnString(dc.User, dc.Password)
}

// AdminConnectionString returns the PostgreSQL connection string for the admin user
// Uses url.URL for secure credential handling to prevent password injection
func (dc *DatabaseConfig) AdminConnectionString() string {
	user := dc.AdminUser
	if user == "" {
		user = dc.User
	}
	password := dc.AdminPassword
	if password == "" {
		password = dc.Password
	}
	return dc.buildSecureConnString(user, password)
}

// buildSecureConnString creates a connection string using url.URL for secure credential handling
// This prevents password injection via special characters in passwords
func (dc *DatabaseConfig) buildSecureConnString(user, password string) string {
	// Use url.URL to properly encode credentials and prevent injection
	u := &url.URL{
		Scheme:   "postgres",
		Host:     fmt.Sprintf("%s:%d", dc.Host, dc.Port),
		Path:     "/" + dc.Database,
		RawQuery: fmt.Sprintf("sslmode=%s", dc.SSLMode),
	}
	u.User = url.UserPassword(user, password)
	return u.String()
}

// RedactConnString returns a connection string with the password redacted for logging
// Example: postgres://user:****@localhost:5432/db?sslmode=disable
func (dc *DatabaseConfig) RedactConnString(connStr string) string {
	// Parse the connection string
	u, err := url.Parse(connStr)
	if err != nil || u.Scheme == "" {
		// If parsing fails or it's not a valid URL, return a fully redacted string
		return "postgres://****@****:****/****?sslmode=****"
	}

	// Redact the password
	if u.User != nil {
		_, passwordSet := u.User.Password()
		if passwordSet {
			u.User = url.UserPassword(u.User.Username(), "****")
		}
	}

	return u.String()
}

// Validate validates security configuration
func (sc *SecurityConfig) Validate() error {
	// Check for insecure default setup token if admin dashboard is enabled
	if sc.SetupToken != "" {
		insecureDefaults := []string{
			"your-secret-setup-token-change-in-production",
			"your-secret-setup-token",
			"changeme",
			"test",
		}
		for _, insecure := range insecureDefaults {
			if sc.SetupToken == insecure {
				return fmt.Errorf("please set a secure setup token (current value '%s' is insecure)", sc.SetupToken)
			}
		}

		// Warn if setup token is too short
		if len(sc.SetupToken) < 32 {
			log.Warn().Msg("Security setup token is shorter than 32 characters - consider using a longer token for better security")
		}
	}

	return nil
}

// Validate validates email configuration
func (ec *EmailConfig) Validate() error {
	// Validate provider if specified
	if ec.Provider != "" {
		validProviders := []string{"smtp", "sendgrid", "mailgun", "ses"}
		providerValid := false
		for _, p := range validProviders {
			if ec.Provider == p {
				providerValid = true
				break
			}
		}
		if !providerValid {
			return fmt.Errorf("invalid email provider: %s (must be one of: %v)", ec.Provider, validProviders)
		}
	}

	// Provider-specific settings are validated at runtime when sending emails,
	// allowing configuration via admin UI after startup

	return nil
}

// IsConfigured returns true if the email provider is fully configured and ready to send emails
func (ec *EmailConfig) IsConfigured() bool {
	if !ec.Enabled || ec.FromAddress == "" {
		return false
	}

	switch ec.Provider {
	case "smtp", "":
		return ec.SMTPHost != "" && ec.SMTPPort != 0
	case "sendgrid":
		return ec.SendGridAPIKey != ""
	case "mailgun":
		return ec.MailgunAPIKey != "" && ec.MailgunDomain != ""
	case "ses":
		// SES credentials are optional (can use AWS default credential chain)
		return ec.SESRegion != ""
	default:
		return false
	}
}

// Validate validates functions configuration
func (fc *FunctionsConfig) Validate() error {
	// Validate functions directory
	if fc.FunctionsDir == "" {
		return fmt.Errorf("functions_dir cannot be empty")
	}

	// Validate timeout settings
	if fc.DefaultTimeout <= 0 {
		return fmt.Errorf("default_timeout must be positive, got: %d", fc.DefaultTimeout)
	}
	if fc.MaxTimeout <= 0 {
		return fmt.Errorf("max_timeout must be positive, got: %d", fc.MaxTimeout)
	}
	if fc.DefaultTimeout > fc.MaxTimeout {
		return fmt.Errorf("default_timeout (%d) cannot be greater than max_timeout (%d)", fc.DefaultTimeout, fc.MaxTimeout)
	}

	// Validate memory limit settings
	if fc.DefaultMemoryLimit <= 0 {
		return fmt.Errorf("default_memory_limit must be positive, got: %d", fc.DefaultMemoryLimit)
	}
	if fc.MaxMemoryLimit <= 0 {
		return fmt.Errorf("max_memory_limit must be positive, got: %d", fc.MaxMemoryLimit)
	}
	if fc.DefaultMemoryLimit > fc.MaxMemoryLimit {
		return fmt.Errorf("default_memory_limit (%d) cannot be greater than max_memory_limit (%d)", fc.DefaultMemoryLimit, fc.MaxMemoryLimit)
	}

	// Warn if max_timeout is very high (over 5 minutes)
	if fc.MaxTimeout > 300 {
		log.Warn().Int("max_timeout", fc.MaxTimeout).Msg("max_timeout is over 5 minutes - long-running functions may impact performance")
	}

	// Warn if max_memory_limit is very high (over 1GB)
	if fc.MaxMemoryLimit > 1024 {
		log.Warn().Int("max_memory_limit", fc.MaxMemoryLimit).Msg("max_memory_limit is over 1GB - high memory functions may impact performance")
	}

	return nil
}

// Validate validates API configuration
func (ac *APIConfig) Validate() error {
	// Validate max_page_size (-1 is allowed for unlimited)
	if ac.MaxPageSize == 0 || ac.MaxPageSize < -1 {
		return fmt.Errorf("max_page_size must be positive or -1 for unlimited, got: %d", ac.MaxPageSize)
	}

	// Validate max_total_results (-1 is allowed for unlimited)
	if ac.MaxTotalResults == 0 || ac.MaxTotalResults < -1 {
		return fmt.Errorf("max_total_results must be positive or -1 for unlimited, got: %d", ac.MaxTotalResults)
	}

	// Validate default_page_size (-1 is allowed for no default)
	if ac.DefaultPageSize == 0 || ac.DefaultPageSize < -1 {
		return fmt.Errorf("default_page_size must be positive or -1 for no default, got: %d", ac.DefaultPageSize)
	}

	// Validate that default_page_size doesn't exceed max_page_size (unless either is -1)
	if ac.DefaultPageSize > 0 && ac.MaxPageSize > 0 && ac.DefaultPageSize > ac.MaxPageSize {
		return fmt.Errorf("default_page_size (%d) cannot exceed max_page_size (%d)", ac.DefaultPageSize, ac.MaxPageSize)
	}

	// Warn if limits are disabled
	if ac.MaxPageSize == -1 {
		log.Warn().Msg("max_page_size is set to -1 (unlimited) - this may allow expensive queries")
	}
	if ac.MaxTotalResults == -1 {
		log.Warn().Msg("max_total_results is set to -1 (unlimited) - this may allow deep pagination attacks")
	}
	if ac.DefaultPageSize == -1 {
		log.Warn().Msg("default_page_size is set to -1 (no default) - queries without limit parameter will return all rows")
	}

	return nil
}

// Validate validates jobs configuration
func (jc *JobsConfig) Validate() error {
	// Validate jobs directory
	if jc.JobsDir == "" {
		return fmt.Errorf("jobs_dir cannot be empty")
	}

	// Validate worker mode
	validModes := []string{"embedded", "standalone", "disabled"}
	modeValid := false
	for _, mode := range validModes {
		if jc.WorkerMode == mode {
			modeValid = true
			break
		}
	}
	if !modeValid {
		return fmt.Errorf("invalid worker_mode: %s (must be one of: %v)", jc.WorkerMode, validModes)
	}

	// Validate worker counts
	if jc.EmbeddedWorkerCount < 0 {
		return fmt.Errorf("embedded_worker_count cannot be negative, got: %d", jc.EmbeddedWorkerCount)
	}
	if jc.MaxConcurrentPerWorker <= 0 {
		return fmt.Errorf("max_concurrent_per_worker must be positive, got: %d", jc.MaxConcurrentPerWorker)
	}
	if jc.MaxConcurrentPerNamespace <= 0 {
		return fmt.Errorf("max_concurrent_per_namespace must be positive, got: %d", jc.MaxConcurrentPerNamespace)
	}

	// Validate timeout settings
	if jc.DefaultMaxDuration <= 0 {
		return fmt.Errorf("default_max_duration must be positive, got: %v", jc.DefaultMaxDuration)
	}
	if jc.MaxMaxDuration <= 0 {
		return fmt.Errorf("max_max_duration must be positive, got: %v", jc.MaxMaxDuration)
	}
	if jc.DefaultMaxDuration > jc.MaxMaxDuration {
		return fmt.Errorf("default_max_duration (%v) cannot be greater than max_max_duration (%v)", jc.DefaultMaxDuration, jc.MaxMaxDuration)
	}
	if jc.DefaultProgressTimeout <= 0 {
		return fmt.Errorf("default_progress_timeout must be positive, got: %v", jc.DefaultProgressTimeout)
	}

	// Validate intervals
	if jc.PollInterval <= 0 {
		return fmt.Errorf("poll_interval must be positive, got: %v", jc.PollInterval)
	}
	if jc.WorkerHeartbeatInterval <= 0 {
		return fmt.Errorf("worker_heartbeat_interval must be positive, got: %v", jc.WorkerHeartbeatInterval)
	}
	if jc.WorkerTimeout <= 0 {
		return fmt.Errorf("worker_timeout must be positive, got: %v", jc.WorkerTimeout)
	}

	// Warn if max_max_duration is very high (over 1 hour)
	if jc.MaxMaxDuration > time.Hour {
		log.Warn().Dur("max_max_duration", jc.MaxMaxDuration).Msg("max_max_duration is over 1 hour - very long-running jobs may impact performance")
	}

	// Warn if worker count is 0 in embedded mode
	if jc.WorkerMode == "embedded" && jc.EmbeddedWorkerCount == 0 {
		log.Warn().Msg("worker_mode is 'embedded' but embedded_worker_count is 0 - no jobs will be processed")
	}

	return nil
}

// Validate validates tracing configuration
func (tc *TracingConfig) Validate() error {
	if !tc.Enabled {
		return nil // No validation needed if tracing is disabled
	}

	// Validate endpoint
	if tc.Endpoint == "" {
		return fmt.Errorf("tracing endpoint is required when tracing is enabled")
	}

	// Validate sample rate
	if tc.SampleRate < 0 || tc.SampleRate > 1 {
		return fmt.Errorf("tracing sample_rate must be between 0.0 and 1.0, got: %f", tc.SampleRate)
	}

	// Warn if sample rate is 100% in production
	if tc.Environment == "production" && tc.SampleRate >= 1.0 {
		log.Warn().Msg("Tracing sample_rate is 100% in production - consider reducing to lower overhead")
	}

	return nil
}

// Validate validates metrics configuration
func (mc *MetricsConfig) Validate() error {
	if !mc.Enabled {
		return nil // No validation needed if metrics is disabled
	}

	// Validate port
	if mc.Port < 1 || mc.Port > 65535 {
		return fmt.Errorf("metrics port must be between 1 and 65535, got: %d", mc.Port)
	}

	// Validate path
	if mc.Path == "" {
		return fmt.Errorf("metrics path cannot be empty")
	}
	if !strings.HasPrefix(mc.Path, "/") {
		return fmt.Errorf("metrics path must start with '/', got: %s", mc.Path)
	}

	return nil
}

// Validate validates scaling configuration
func (sc *ScalingConfig) Validate() error {
	// Validate backend
	validBackends := []string{"local", "postgres", "redis"}
	backendValid := false
	for _, b := range validBackends {
		if sc.Backend == b {
			backendValid = true
			break
		}
	}
	if !backendValid {
		return fmt.Errorf("invalid scaling backend: %s (must be one of: %v)", sc.Backend, validBackends)
	}

	// Validate redis_url is set when backend is redis
	if sc.Backend == "redis" && sc.RedisURL == "" {
		return fmt.Errorf("redis_url is required when scaling backend is 'redis'")
	}

	// Warn about conflicting settings
	if sc.WorkerOnly && !sc.DisableScheduler {
		log.Warn().Msg("Worker-only mode is enabled but scheduler is not disabled - consider setting disable_scheduler=true for worker containers")
	}

	if sc.WorkerOnly && !sc.DisableRealtime {
		log.Warn().Msg("Worker-only mode is enabled but realtime is not disabled - realtime will be skipped in worker-only mode anyway")
	}

	return nil
}

// Validate validates AI configuration
func (ac *AIConfig) Validate() error {
	// Validate chatbots directory
	if ac.ChatbotsDir == "" {
		return fmt.Errorf("chatbots_dir cannot be empty")
	}

	// Validate token settings
	if ac.DefaultMaxTokens <= 0 {
		return fmt.Errorf("default_max_tokens must be positive, got: %d", ac.DefaultMaxTokens)
	}

	// Validate query timeout
	if ac.QueryTimeout <= 0 {
		return fmt.Errorf("query_timeout must be positive, got: %v", ac.QueryTimeout)
	}

	// Validate max rows per query
	if ac.MaxRowsPerQuery <= 0 {
		return fmt.Errorf("max_rows_per_query must be positive, got: %d", ac.MaxRowsPerQuery)
	}

	// Validate conversation settings
	if ac.ConversationCacheTTL <= 0 {
		return fmt.Errorf("conversation_cache_ttl must be positive, got: %v", ac.ConversationCacheTTL)
	}
	if ac.MaxConversationTurns <= 0 {
		return fmt.Errorf("max_conversation_turns must be positive, got: %d", ac.MaxConversationTurns)
	}

	// Warn if max rows is very high
	if ac.MaxRowsPerQuery > 10000 {
		log.Warn().Int("max_rows_per_query", ac.MaxRowsPerQuery).Msg("max_rows_per_query is over 10000 - large result sets may impact performance")
	}

	return nil
}

// Validate validates logging configuration
func (lc *LoggingConfig) Validate() error {
	// Validate console level
	validLevels := []string{"trace", "debug", "info", "warn", "error"}
	levelValid := false
	for _, level := range validLevels {
		if lc.ConsoleLevel == level {
			levelValid = true
			break
		}
	}
	if !levelValid && lc.ConsoleLevel != "" {
		return fmt.Errorf("invalid console_level: %s (must be one of: %v)", lc.ConsoleLevel, validLevels)
	}

	// Validate console format
	if lc.ConsoleFormat != "" && lc.ConsoleFormat != "json" && lc.ConsoleFormat != "console" {
		return fmt.Errorf("invalid console_format: %s (must be 'json' or 'console')", lc.ConsoleFormat)
	}

	// Validate backend
	validBackends := []string{"postgres", "postgres-timescaledb", "timescaledb", "s3", "local", "elasticsearch", "opensearch", "clickhouse", "loki"}
	backendValid := false
	for _, backend := range validBackends {
		if lc.Backend == backend {
			backendValid = true
			break
		}
	}
	if !backendValid && lc.Backend != "" {
		return fmt.Errorf("invalid logging backend: %s (must be one of: %v)", lc.Backend, validBackends)
	}

	// Validate S3 settings when backend is s3
	if lc.Backend == "s3" && lc.S3Bucket == "" {
		return fmt.Errorf("s3_bucket is required when logging backend is 's3'")
	}

	// Validate batching settings
	if lc.BatchSize < 0 {
		return fmt.Errorf("batch_size cannot be negative, got: %d", lc.BatchSize)
	}
	if lc.FlushInterval < 0 {
		return fmt.Errorf("flush_interval cannot be negative, got: %v", lc.FlushInterval)
	}
	if lc.BufferSize < 0 {
		return fmt.Errorf("buffer_size cannot be negative, got: %d", lc.BufferSize)
	}

	// Validate retention settings
	if lc.SystemRetentionDays < 0 {
		return fmt.Errorf("system_retention_days cannot be negative, got: %d", lc.SystemRetentionDays)
	}
	if lc.HTTPRetentionDays < 0 {
		return fmt.Errorf("http_retention_days cannot be negative, got: %d", lc.HTTPRetentionDays)
	}
	if lc.SecurityRetentionDays < 0 {
		return fmt.Errorf("security_retention_days cannot be negative, got: %d", lc.SecurityRetentionDays)
	}
	if lc.ExecutionRetentionDays < 0 {
		return fmt.Errorf("execution_retention_days cannot be negative, got: %d", lc.ExecutionRetentionDays)
	}
	if lc.AIRetentionDays < 0 {
		return fmt.Errorf("ai_retention_days cannot be negative, got: %d", lc.AIRetentionDays)
	}

	// Warn about short retention periods for security logs
	if lc.SecurityRetentionDays > 0 && lc.SecurityRetentionDays < 30 {
		log.Warn().Int("security_retention_days", lc.SecurityRetentionDays).Msg("Security log retention is less than 30 days - consider increasing for compliance")
	}

	return nil
}

// calculateEntropy calculates the Shannon entropy of a string in bits.
// Higher entropy indicates more randomness and better security.
// Formula: H = -Σ p(x) * log2(p(x)) where p(x) is the probability of character x
func calculateEntropy(s string) float64 {
	if len(s) == 0 {
		return 0
	}

	// Count frequency of each character
	freq := make(map[rune]int)
	for _, char := range s {
		freq[char]++
	}

	// Calculate Shannon entropy
	length := float64(len(s))
	entropy := 0.0

	for _, count := range freq {
		probability := float64(count) / length
		entropy -= probability * math.Log2(probability)
	}

	return entropy
}
