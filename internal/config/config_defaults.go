package config

import (
	"github.com/spf13/viper"
)

// setDefaults sets default configuration values
func setDefaults() {
	// Server defaults
	viper.SetDefault("server.address", ":8080")
	viper.SetDefault("server.read_timeout", "300s")          // 5 min for large file streaming
	viper.SetDefault("server.write_timeout", "300s")         // 5 min for large file streaming
	viper.SetDefault("server.idle_timeout", "120s")          // 2 min idle timeout
	viper.SetDefault("server.body_limit", 2*1024*1024*1024)  // 2GB
	viper.SetDefault("server.allowed_ip_ranges", []string{}) // Empty = allow all (backward compatible)
	viper.SetDefault("server.trusted_proxies", []string{})   // Empty = trust no proxies (most secure)

	// Per-endpoint body limits (more granular than global body_limit)
	viper.SetDefault("server.body_limits.enabled", true)
	viper.SetDefault("server.body_limits.default_limit", 1*1024*1024)   // 1MB default
	viper.SetDefault("server.body_limits.rest_limit", 1*1024*1024)      // 1MB for REST CRUD
	viper.SetDefault("server.body_limits.auth_limit", 64*1024)          // 64KB for auth
	viper.SetDefault("server.body_limits.storage_limit", 500*1024*1024) // 500MB for uploads
	viper.SetDefault("server.body_limits.bulk_limit", 10*1024*1024)     // 10MB for bulk/RPC
	viper.SetDefault("server.body_limits.admin_limit", 5*1024*1024)     // 5MB for admin
	viper.SetDefault("server.body_limits.max_json_depth", 64)           // Max JSON nesting

	// Database defaults
	viper.SetDefault("database.host", "localhost")
	viper.SetDefault("database.port", 5432)
	viper.SetDefault("database.user", "postgres") // Default runtime user
	viper.SetDefault("database.admin_user", "")   // Empty means use user
	viper.SetDefault("database.password", "postgres")
	viper.SetDefault("database.admin_password", "") // Empty means use password
	viper.SetDefault("database.database", "fluxbase")
	viper.SetDefault("database.ssl_mode", "disable")
	// Connection pool sizing: 50 is suitable for single-instance deployments.
	// For multi-instance deployments, divide by instance count (e.g., 3 instances = 17 per instance).
	// Approximate connection usage: API (20), Jobs (15), Realtime (10), Schema cache (5).
	// Monitor pg_stat_activity and pool exhaustion metrics in production.
	viper.SetDefault("database.max_connections", 50)
	viper.SetDefault("database.min_connections", 10)
	viper.SetDefault("database.max_conn_lifetime", "1h")
	viper.SetDefault("database.max_conn_idle_time", "30m")
	viper.SetDefault("database.health_check_period", "1m")
	viper.SetDefault("database.user_migrations_path", "/migrations/user")
	viper.SetDefault("database.slow_query_threshold", "1s")

	// Declarative app-schema defaults (opt-in — off by default so existing apps are unaffected)
	viper.SetDefault("database.declarative_app_schema.enabled", false)
	viper.SetDefault("database.declarative_app_schema.schema", "public")
	viper.SetDefault("database.declarative_app_schema.namespaces", []string{})
	viper.SetDefault("database.declarative_app_schema.on_startup", true)
	viper.SetDefault("database.declarative_app_schema.allow_destructive", false)

	// Auth defaults
	viper.SetDefault("auth.jwt_secret", "your-secret-key-change-in-production")
	viper.SetDefault("auth.jwt_expiry", "15m")
	viper.SetDefault("auth.refresh_expiry", "168h")  // 7 days in hours
	viper.SetDefault("auth.service_role_ttl", "24h") // Service role tokens: 24 hours (was 365 days)
	viper.SetDefault("auth.anon_ttl", "24h")         // Anonymous tokens: 24 hours (was 365 days)
	viper.SetDefault("auth.magic_link_expiry", "15m")
	viper.SetDefault("auth.password_reset_expiry", "1h")
	viper.SetDefault("auth.password_min_length", 12) // Increased for better security
	viper.SetDefault("auth.bcrypt_cost", 10)
	viper.SetDefault("auth.signup_enabled", true) // Default to enabled to allow user registration
	viper.SetDefault("auth.magic_link_enabled", true)
	viper.SetDefault("auth.totp_issuer", "Fluxbase") // Default issuer name for 2FA TOTP (shown in authenticator apps)

	// Security defaults
	viper.SetDefault("security.enable_global_rate_limit", true)    // Enabled by default for security (can be disabled if needed)
	viper.SetDefault("security.setup_token", "")                   // Empty by default - required when admin.enabled=true
	viper.SetDefault("security.admin_setup_rate_limit", 5)         // 5 attempts
	viper.SetDefault("security.admin_setup_rate_window", "15m")    // per 15 minutes
	viper.SetDefault("security.auth_login_rate_limit", 10)         // 10 attempts
	viper.SetDefault("security.auth_login_rate_window", "1m")      // per minute
	viper.SetDefault("security.admin_login_rate_limit", 10)        // 10 attempts
	viper.SetDefault("security.admin_login_rate_window", "1m")     // per minute
	viper.SetDefault("security.dashboard_login_rate_limit", 60)    // 60 attempts
	viper.SetDefault("security.dashboard_login_rate_window", "1m") // per minute

	// service_role rate limiting defaults (H-2: enabled by default to prevent abuse)
	viper.SetDefault("security.service_role_rate_limit", 10000) // 10000 requests per minute for service_role tokens (H-2)
	viper.SetDefault("security.service_role_rate_window", "1m") // per minute

	// CAPTCHA defaults
	viper.SetDefault("security.captcha.enabled", false)       // Disabled by default
	viper.SetDefault("security.captcha.provider", "hcaptcha") // Default to hCaptcha (privacy-focused)
	viper.SetDefault("security.captcha.site_key", "")         // Must be configured
	viper.SetDefault("security.captcha.secret_key", "")       // Must be configured
	viper.SetDefault("security.captcha.score_threshold", 0.5) // For reCAPTCHA v3
	viper.SetDefault("security.captcha.endpoints", []string{"signup", "login", "password_reset", "magic_link"})

	// Adaptive CAPTCHA trust defaults
	viper.SetDefault("security.captcha.adaptive_trust.enabled", false)         // Disabled by default
	viper.SetDefault("security.captcha.adaptive_trust.trust_token_ttl", "15m") // 15 minutes trust after CAPTCHA
	viper.SetDefault("security.captcha.adaptive_trust.trust_token_bound_ip", true)
	viper.SetDefault("security.captcha.adaptive_trust.challenge_expiry", "5m") // 5 minute challenge validity
	viper.SetDefault("security.captcha.adaptive_trust.captcha_threshold", 50)  // Score below 50 requires CAPTCHA
	// Positive trust weights
	viper.SetDefault("security.captcha.adaptive_trust.weight_known_ip", 30)
	viper.SetDefault("security.captcha.adaptive_trust.weight_known_device", 25)
	viper.SetDefault("security.captcha.adaptive_trust.weight_recent_captcha", 40)
	viper.SetDefault("security.captcha.adaptive_trust.weight_verified_email", 15)
	viper.SetDefault("security.captcha.adaptive_trust.weight_account_age", 10)
	viper.SetDefault("security.captcha.adaptive_trust.weight_successful_logins", 10)
	viper.SetDefault("security.captcha.adaptive_trust.weight_mfa_enabled", 20)
	// Negative trust weights
	viper.SetDefault("security.captcha.adaptive_trust.weight_new_ip", -30)
	viper.SetDefault("security.captcha.adaptive_trust.weight_new_device", -25)
	viper.SetDefault("security.captcha.adaptive_trust.weight_failed_attempts", -20)
	// Endpoints that always require CAPTCHA regardless of trust
	viper.SetDefault("security.captcha.adaptive_trust.always_require_endpoints", []string{"password_reset"})

	// Admin defaults
	viper.SetDefault("admin.enabled", true) // Admin dashboard enabled by default (protected by setup_token gate)

	// Tenants defaults
	viper.SetDefault("tenants.enabled", true)
	viper.SetDefault("tenants.database_prefix", "tenant_")
	viper.SetDefault("tenants.max_tenants", 100)
	viper.SetDefault("tenants.pool.max_total_connections", 100)
	viper.SetDefault("tenants.pool.eviction_age", "30m")
	viper.SetDefault("tenants.migrations.check_interval", "5m")
	viper.SetDefault("tenants.migrations.on_create", true)
	viper.SetDefault("tenants.migrations.on_access", true)
	viper.SetDefault("tenants.migrations.background", true)
	viper.SetDefault("tenants.default.name", "default")
	viper.SetDefault("tenants.default.anon_key", "")
	viper.SetDefault("tenants.default.service_key", "")
	viper.SetDefault("tenants.default.anon_key_file", "")
	viper.SetDefault("tenants.default.service_key_file", "")

	// Tenant declarative schema defaults
	viper.SetDefault("tenants.declarative.enabled", false)
	viper.SetDefault("tenants.declarative.schema_dir", "")
	viper.SetDefault("tenants.declarative.on_create", false)
	viper.SetDefault("tenants.declarative.on_startup", false)
	viper.SetDefault("tenants.declarative.allow_destructive", false)

	// CORS defaults
	viper.SetDefault("cors.allowed_origins", "http://localhost:5173,http://localhost:8080")
	viper.SetDefault("cors.allowed_methods", "GET,POST,PUT,PATCH,DELETE,OPTIONS")
	viper.SetDefault("cors.allowed_headers", "Origin,Content-Type,Accept,Authorization,X-Request-ID,X-CSRF-Token,X-Impersonation-Token,Prefer,apikey,x-client-app")
	viper.SetDefault("cors.exposed_headers", "Content-Range,Content-Encoding,Content-Length,X-Request-ID,X-RateLimit-Limit,X-RateLimit-Remaining,X-RateLimit-Reset")
	viper.SetDefault("cors.allow_credentials", true) // Required for CSRF tokens
	viper.SetDefault("cors.max_age", 300)

	// Storage defaults
	viper.SetDefault("storage.enabled", true)
	viper.SetDefault("storage.provider", "local")
	viper.SetDefault("storage.local_path", "./storage")
	viper.SetDefault("storage.s3_endpoint", "")
	viper.SetDefault("storage.s3_access_key", "")
	viper.SetDefault("storage.s3_secret_key", "")
	viper.SetDefault("storage.s3_bucket", "")
	viper.SetDefault("storage.s3_region", "")
	viper.SetDefault("storage.s3_force_path_style", true) // Default true for S3-compatible services (MinIO, R2, Spaces, etc.)
	viper.SetDefault("storage.default_buckets", []string{"uploads", "temp-files", "public"})
	viper.SetDefault("storage.max_upload_size", 2*1024*1024*1024) // 2GB

	// Storage transform defaults
	viper.SetDefault("storage.transforms.enabled", true)
	viper.SetDefault("storage.transforms.default_quality", 80)
	viper.SetDefault("storage.transforms.max_width", 4096)
	viper.SetDefault("storage.transforms.max_height", 4096)
	viper.SetDefault("storage.transforms.allowed_formats", []string{"webp", "jpg", "png", "avif"})
	// Security settings
	viper.SetDefault("storage.transforms.max_total_pixels", 16_000_000) // 16 megapixels
	viper.SetDefault("storage.transforms.bucket_size", 50)              // Round dimensions to 50px
	viper.SetDefault("storage.transforms.rate_limit", 60)               // 60 transforms/min/user
	viper.SetDefault("storage.transforms.timeout", "30s")               // 30 second timeout
	viper.SetDefault("storage.transforms.max_concurrent", 4)            // 4 concurrent transforms
	// Caching settings
	viper.SetDefault("storage.transforms.cache_enabled", true)
	viper.SetDefault("storage.transforms.cache_ttl", "24h")
	viper.SetDefault("storage.transforms.cache_max_size", 1024*1024*1024) // 1GB

	// Realtime defaults
	viper.SetDefault("realtime.enabled", true)
	viper.SetDefault("realtime.max_connections", 1000)
	viper.SetDefault("realtime.max_connections_per_user", 10) // Limit per authenticated user
	viper.SetDefault("realtime.max_connections_per_ip", 20)   // Limit per IP for anonymous connections
	viper.SetDefault("realtime.rls_cache_size", 100000)       // 100K entries for high-throughput realtime
	viper.SetDefault("realtime.rls_cache_ttl", "30s")         // 30 second TTL (balance freshness vs DB load)
	viper.SetDefault("realtime.listener_pool_size", 2)        // 2 LISTEN connections for redundancy/failover
	viper.SetDefault("realtime.notification_workers", 4)
	viper.SetDefault("realtime.client_message_queue_size", 256) // Per-client message queue for async sending
	viper.SetDefault("realtime.slow_client_threshold", 100)     // Disconnect clients with 100+ pending messages
	viper.SetDefault("realtime.slow_client_timeout", "30s")     // After 30s of being slow

	// Email defaults
	viper.SetDefault("email.enabled", true)
	viper.SetDefault("email.provider", "smtp")
	viper.SetDefault("email.from_address", "noreply@localhost")
	viper.SetDefault("email.from_name", "Fluxbase")
	viper.SetDefault("email.reply_to_address", "")
	// SMTP defaults - empty strings allow env vars to be picked up by Unmarshal
	viper.SetDefault("email.smtp_host", "")
	viper.SetDefault("email.smtp_port", 587)
	viper.SetDefault("email.smtp_username", "")
	viper.SetDefault("email.smtp_password", "")
	viper.SetDefault("email.smtp_tls", true)
	// SendGrid defaults
	viper.SetDefault("email.sendgrid_api_key", "")
	// Mailgun defaults
	viper.SetDefault("email.mailgun_api_key", "")
	viper.SetDefault("email.mailgun_domain", "")
	// AWS SES defaults
	viper.SetDefault("email.ses_access_key", "")
	viper.SetDefault("email.ses_secret_key", "")
	viper.SetDefault("email.ses_region", "")
	// Template defaults
	viper.SetDefault("email.magic_link_template", "")
	viper.SetDefault("email.verification_template", "")
	viper.SetDefault("email.password_reset_template", "")

	// Functions defaults
	viper.SetDefault("functions.enabled", true)
	viper.SetDefault("functions.functions_dir", "./functions")
	viper.SetDefault("functions.auto_load_on_boot", true)       // Enabled by default for better DX
	viper.SetDefault("functions.default_timeout", 30)           // 30 seconds
	viper.SetDefault("functions.max_timeout", 300)              // 5 minutes
	viper.SetDefault("functions.default_memory_limit", 128)     // 128MB
	viper.SetDefault("functions.max_memory_limit", 1024)        // 1GB
	viper.SetDefault("functions.max_output_size", 10*1024*1024) // 10MB - prevents OOM from large function output
	viper.SetDefault("functions.sync_allowed_ip_ranges", []string{
		"172.16.0.0/12",  // Docker default bridge networks
		"10.0.0.0/8",     // Private networks (AWS VPC, etc.)
		"192.168.0.0/16", // Private networks
		"127.0.0.0/8",    // Loopback (localhost)
	})

	// API defaults
	viper.SetDefault("api.max_page_size", 1000)      // Max 1000 rows per request
	viper.SetDefault("api.max_total_results", 10000) // Max 10k total rows retrievable
	viper.SetDefault("api.default_page_size", 1000)  // Default to 1000 rows if not specified
	viper.SetDefault("api.max_batch_size", 1000)     // Max 1000 records in batch insert/update (H-4)

	// Migrations defaults
	viper.SetDefault("migrations.enabled", true) // Enabled by default for better DX (security still enforced via service key + IP allowlist)
	viper.SetDefault("migrations.allowed_ip_ranges", []string{
		"172.16.0.0/12",  // Docker default bridge networks
		"10.0.0.0/8",     // Private networks (AWS VPC, etc.)
		"192.168.0.0/16", // Private networks
		"127.0.0.0/8",    // Loopback (localhost)
	})

	// Jobs defaults
	viper.SetDefault("jobs.enabled", true) // Enabled by default (controlled by feature flag at runtime)
	viper.SetDefault("jobs.jobs_dir", "./jobs")
	viper.SetDefault("jobs.auto_load_on_boot", true)          // Auto-load jobs by default for better DX
	viper.SetDefault("jobs.worker_mode", "embedded")          // embedded, standalone, disabled
	viper.SetDefault("jobs.embedded_worker_count", 4)         // 4 workers by default for good performance
	viper.SetDefault("jobs.max_concurrent_per_worker", 5)     // Max concurrent jobs per worker
	viper.SetDefault("jobs.max_concurrent_per_namespace", 20) // Max concurrent jobs per namespace
	viper.SetDefault("jobs.default_max_duration", "5m")       // 5 minutes default job timeout
	viper.SetDefault("jobs.max_max_duration", "1h")           // 1 hour maximum job timeout
	viper.SetDefault("jobs.default_progress_timeout", "300s") // 5 minutes progress timeout
	viper.SetDefault("jobs.poll_interval", "1s")              // Worker polls every 1 second
	viper.SetDefault("jobs.worker_heartbeat_interval", "10s") // Worker heartbeat every 10 seconds
	viper.SetDefault("jobs.worker_timeout", "30s")            // Worker considered dead after 30 seconds
	viper.SetDefault("jobs.sync_allowed_ip_ranges", []string{
		"172.16.0.0/12",  // Docker default bridge networks
		"10.0.0.0/8",     // Private networks (AWS VPC, etc.)
		"192.168.0.0/16", // Private networks
		"127.0.0.0/8",    // Loopback (localhost)
	})
	viper.SetDefault("jobs.graceful_shutdown_timeout", "5m") // Wait up to 5 minutes for jobs during shutdown

	// Tracing defaults (OpenTelemetry)
	viper.SetDefault("tracing.enabled", false)             // Disabled by default
	viper.SetDefault("tracing.endpoint", "localhost:4317") // Default OTLP gRPC endpoint
	viper.SetDefault("tracing.service_name", "fluxbase")   // Service name for traces
	viper.SetDefault("tracing.environment", "development") // Default environment
	viper.SetDefault("tracing.sample_rate", 1.0)           // 100% sampling by default (reduce in production)
	viper.SetDefault("tracing.insecure", true)             // Use insecure connection by default (for local dev)

	// Metrics defaults (Prometheus)
	viper.SetDefault("metrics.enabled", true)    // Enabled by default
	viper.SetDefault("metrics.port", 9090)       // Default Prometheus metrics port
	viper.SetDefault("metrics.path", "/metrics") // Default metrics endpoint path

	// AI defaults
	viper.SetDefault("ai.enabled", true)                 // Enabled by default (controlled by feature flag at runtime)
	viper.SetDefault("ai.chatbots_dir", "./chatbots")    // Default chatbots directory
	viper.SetDefault("ai.auto_load_on_boot", true)       // Auto-load chatbots by default for better DX
	viper.SetDefault("ai.default_max_tokens", 4096)      // Default max tokens per request
	viper.SetDefault("ai.default_model", "gpt-4-turbo")  // Default AI model
	viper.SetDefault("ai.query_timeout", "30s")          // 30 second query timeout
	viper.SetDefault("ai.max_rows_per_query", 1000)      // Max 1000 rows per query
	viper.SetDefault("ai.conversation_cache_ttl", "30m") // 30 minute cache TTL
	viper.SetDefault("ai.max_conversation_turns", 50)    // Max 50 turns per conversation
	viper.SetDefault("ai.sync_allowed_ip_ranges", []string{
		"172.16.0.0/12",  // Docker default bridge networks
		"10.0.0.0/8",     // Private networks (AWS VPC, etc.)
		"192.168.0.0/16", // Private networks
		"127.0.0.0/8",    // Loopback (localhost)
	})

	// AI Provider Configuration defaults
	viper.SetDefault("ai.provider_type", "")          // No default type (if set, config-based provider is enabled)
	viper.SetDefault("ai.provider_name", "")          // No default name
	viper.SetDefault("ai.provider_model", "")         // No default model
	viper.SetDefault("ai.openai_api_key", "")         // No default API key
	viper.SetDefault("ai.openai_organization_id", "") // No default org ID
	viper.SetDefault("ai.openai_base_url", "")        // No default base URL
	viper.SetDefault("ai.azure_api_key", "")          // No default API key
	viper.SetDefault("ai.azure_endpoint", "")         // No default endpoint
	viper.SetDefault("ai.azure_deployment_name", "")  // No default deployment
	viper.SetDefault("ai.azure_api_version", "")      // No default version
	viper.SetDefault("ai.ollama_endpoint", "")        // No default endpoint
	viper.SetDefault("ai.ollama_model", "")           // No default model

	// AI Tavily websearch defaults (Web Agent specialist)
	viper.SetDefault("ai.tavily_api_key", "")       // No default; off when empty
	viper.SetDefault("ai.tavily_default_depth", "") // Empty = client default ("basic")
	viper.SetDefault("ai.tavily_base_url", "")      // Empty = https://api.tavily.com

	// AI Embedding Configuration defaults (for vector search)
	viper.SetDefault("ai.embedding_enabled", false)            // Disabled by default
	viper.SetDefault("ai.embedding_provider", "")              // Defaults to ai.provider_type if empty
	viper.SetDefault("ai.embedding_model", "")                 // Empty = use provider-specific default (openai: text-embedding-3-small, azure: text-embedding-ada-002, ollama: nomic-embed-text)
	viper.SetDefault("ai.azure_embedding_deployment_name", "") // Optional separate Azure embedding deployment

	// AI OCR Configuration defaults (for image-based PDF extraction)
	viper.SetDefault("ai.ocr_enabled", true)              // Enabled by default (will gracefully degrade if Tesseract not installed)
	viper.SetDefault("ai.ocr_provider", "tesseract")      // Default OCR provider
	viper.SetDefault("ai.ocr_languages", []string{"eng"}) // Default to English

	// RPC defaults
	viper.SetDefault("rpc.enabled", true)           // Enabled by default (controlled by feature flag at runtime)
	viper.SetDefault("rpc.procedures_dir", "./rpc") // Default procedures directory
	viper.SetDefault("rpc.auto_load_on_boot", true) // Auto-load procedures by default
	viper.SetDefault("rpc.default_max_rows", 1000)  // Max 1000 rows per query
	viper.SetDefault("rpc.sync_allowed_ip_ranges", []string{
		"172.16.0.0/12",  // Docker default bridge networks
		"10.0.0.0/8",     // Private networks (AWS VPC, etc.)
		"192.168.0.0/16", // Private networks
		"127.0.0.0/8",    // Loopback (localhost)
	})

	// GraphQL defaults
	viper.SetDefault("graphql.enabled", true)          // Enabled by default
	viper.SetDefault("graphql.max_depth", 10)          // Maximum query depth
	viper.SetDefault("graphql.max_complexity", 1000)   // Maximum query complexity
	viper.SetDefault("graphql.introspection", true)    // Enable introspection (disable in production for security)
	viper.SetDefault("graphql.allow_fragments", false) // H-5: Fragment spreads disabled by default (security)
	viper.SetDefault("graphql.max_fields_per_lvl", 50) // H-6: Max 50 unique fields per level (alias abuse protection)

	// MCP defaults (Model Context Protocol server for AI assistants)
	viper.SetDefault("mcp.enabled", true)                      // Enabled by default
	viper.SetDefault("mcp.base_path", "/mcp")                  // Default MCP endpoint path
	viper.SetDefault("mcp.session_timeout", "30m")             // 30 minute session timeout
	viper.SetDefault("mcp.max_message_size", 10*1024*1024)     // 10MB max message size
	viper.SetDefault("mcp.allowed_tools", []string{})          // Empty = all tools enabled
	viper.SetDefault("mcp.allowed_resources", []string{})      // Empty = all resources enabled
	viper.SetDefault("mcp.rate_limit_per_min", 100)            // 100 requests per minute per client
	viper.SetDefault("mcp.tools_dir", "/app/mcp-tools")        // Default custom MCP tools directory
	viper.SetDefault("mcp.auto_load_on_boot", true)            // Auto-load custom tools on startup
	viper.SetDefault("mcp.oauth.enabled", true)                // OAuth enabled by default for zero-config MCP clients
	viper.SetDefault("mcp.oauth.dcr_enabled", true)            // Dynamic Client Registration enabled by default
	viper.SetDefault("mcp.oauth.token_expiry", "1h")           // 1 hour access token expiry
	viper.SetDefault("mcp.oauth.refresh_token_expiry", "168h") // 7 days refresh token expiry

	// MCP OAuth defaults (OAuth 2.1 authentication for MCP clients)
	viper.SetDefault("mcp.oauth.enabled", true)                     // Enable OAuth for MCP
	viper.SetDefault("mcp.oauth.dcr_enabled", true)                 // Enable Dynamic Client Registration
	viper.SetDefault("mcp.oauth.token_expiry", "1h")                // Access token lifetime
	viper.SetDefault("mcp.oauth.refresh_token_expiry", "168h")      // Refresh token lifetime (7 days)
	viper.SetDefault("mcp.oauth.allowed_redirect_uris", []string{}) // Empty = use defaults

	// Branching defaults (database branching for isolated environments)
	viper.SetDefault("branching.enabled", true)                          // Enabled by default
	viper.SetDefault("branching.max_branches_per_user", 5)               // Max 5 branches per user
	viper.SetDefault("branching.max_total_branches", 50)                 // Max 50 branches total
	viper.SetDefault("branching.default_data_clone_mode", "schema_only") // Clone schema only by default
	viper.SetDefault("branching.auto_delete_after", "0")                 // Never auto-delete (0 = disabled)
	viper.SetDefault("branching.database_prefix", "branch_")             // Prefix for branch databases
	viper.SetDefault("branching.max_branches_per_tenant", 0)             // 0 = unlimited
	viper.SetDefault("branching.default_branch", "main")                 // Default branch name
	viper.SetDefault("branching.max_total_connections", 500)             // Global limit for branch pool connections
	viper.SetDefault("branching.pool_eviction_age", "1h")                // Evict idle branch pools after 1h

	// Scaling defaults (for multi-instance deployments)
	viper.SetDefault("scaling.worker_only", false)                      // Run full server by default
	viper.SetDefault("scaling.disable_scheduler", false)                // Run schedulers by default
	viper.SetDefault("scaling.disable_realtime", false)                 // Run realtime by default
	viper.SetDefault("scaling.enable_scheduler_leader_election", false) // Disabled by default (single instance)
	viper.SetDefault("scaling.backend", "local")                        // Use local in-memory storage by default
	viper.SetDefault("scaling.redis_url", "")                           // No Redis URL by default

	// Logging defaults
	viper.SetDefault("logging.console_enabled", true)
	viper.SetDefault("logging.console_level", "info")
	viper.SetDefault("logging.console_format", "console")       // "json" or "console"
	viper.SetDefault("logging.backend", "postgres")             // postgres, s3, local
	viper.SetDefault("logging.s3_bucket", "")                   // Required when backend is s3
	viper.SetDefault("logging.s3_prefix", "logs")               // Prefix for S3 objects
	viper.SetDefault("logging.local_path", "./logs")            // Path for local logs
	viper.SetDefault("logging.batch_size", 100)                 // Entries per batch
	viper.SetDefault("logging.flush_interval", "1s")            // Flush interval
	viper.SetDefault("logging.buffer_size", 10000)              // Async buffer size
	viper.SetDefault("logging.pubsub_enabled", true)            // Enable PubSub for execution logs
	viper.SetDefault("logging.system_retention_days", 7)        // App logs retention
	viper.SetDefault("logging.http_retention_days", 30)         // HTTP logs retention
	viper.SetDefault("logging.security_retention_days", 90)     // Security logs retention
	viper.SetDefault("logging.execution_retention_days", 30)    // Execution logs retention
	viper.SetDefault("logging.ai_retention_days", 30)           // AI logs retention
	viper.SetDefault("logging.retention_enabled", true)         // Enable retention service
	viper.SetDefault("logging.retention_check_interval", "24h") // Check interval for retention cleanup
	viper.SetDefault("logging.custom_categories", []string{})   // Custom categories (empty by default)
	viper.SetDefault("logging.custom_retention_days", 30)       // Custom category retention

	// TimescaleDB logging defaults
	viper.SetDefault("logging.timescaledb_enabled", false)       // Disabled by default (requires TimescaleDB extension)
	viper.SetDefault("logging.timescaledb_compression", false)   // Disabled by default
	viper.SetDefault("logging.timescaledb_compress_after", "0s") // 0 = disabled (must be enabled explicitly)
	viper.SetDefault("logging.timescaledb_retain_after", "0s")   // 0 = disabled (must be enabled explicitly)

	// General defaults
	viper.SetDefault("base_url", "http://localhost:8080")
	viper.SetDefault("public_base_url", "") // Empty means use base_url for backward compatibility
	viper.SetDefault("debug", false)
	viper.SetDefault("encryption_key", "") // REQUIRED: Must be exactly 32 bytes for AES-256
}
