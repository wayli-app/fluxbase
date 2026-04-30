package main

import (
	"context"
	"flag"
	"fmt"
	"math"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/nimbleflux/fluxbase/internal/api"
	"github.com/nimbleflux/fluxbase/internal/auth"
	"github.com/nimbleflux/fluxbase/internal/config"
	"github.com/nimbleflux/fluxbase/internal/database"
	"github.com/nimbleflux/fluxbase/internal/database/bootstrap"
	"github.com/nimbleflux/fluxbase/internal/database/schema"
	"github.com/nimbleflux/fluxbase/internal/keys"
	"github.com/nimbleflux/fluxbase/internal/migrations"
	"github.com/nimbleflux/fluxbase/internal/storage"
)

var (
	// Version information (set via ldflags during build)
	Version   = "dev"
	Commit    = "unknown"
	BuildDate = "unknown"

	// CLI flags
	showVersion      = flag.Bool("version", false, "Show version information")
	validateConfig   = flag.Bool("validate", false, "Validate configuration and exit")
	maxRetryAttempts = getEnvInt("FLUXBASE_DATABASE_RETRY_ATTEMPTS", 5)

	// Scaling CLI flags (override config file settings)
	workerOnly           = flag.Bool("worker-only", false, "Run in worker-only mode (disable API server, only process background jobs)")
	disableScheduler     = flag.Bool("disable-scheduler", false, "Disable cron schedulers (use for multi-instance deployments)")
	disableRealtime      = flag.Bool("disable-realtime", false, "Disable realtime listener")
	enableLeaderElection = flag.Bool("enable-leader-election", false, "Enable scheduler leader election using PostgreSQL advisory locks")
)

func main() {
	flag.Parse()

	// Show version and exit
	if *showVersion {
		fmt.Printf("Fluxbase %s\n", Version)
		fmt.Printf("Commit: %s\n", Commit)
		fmt.Printf("Build Date: %s\n", BuildDate)
		os.Exit(0)
	}

	// Initialize logger
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})

	log.Info().
		Str("version", Version).
		Str("commit", Commit).
		Str("build_date", BuildDate).
		Msg("Starting Fluxbase")

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to load configuration")
	}

	// Apply CLI flag overrides for scaling settings
	// CLI flags take precedence over config file and environment variables
	if *workerOnly {
		cfg.Scaling.WorkerOnly = true
	}
	if *disableScheduler {
		cfg.Scaling.DisableScheduler = true
	}
	if *disableRealtime {
		cfg.Scaling.DisableRealtime = true
	}
	if *enableLeaderElection {
		cfg.Scaling.EnableSchedulerLeaderElection = true
	}

	// Set log level
	if cfg.Debug {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	} else {
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
	}

	// Print configuration summary
	printConfigSummary(cfg)

	// Log scaling mode if non-default settings are active
	if cfg.Scaling.WorkerOnly || cfg.Scaling.DisableScheduler || cfg.Scaling.DisableRealtime || cfg.Scaling.EnableSchedulerLeaderElection {
		log.Info().
			Bool("worker_only", cfg.Scaling.WorkerOnly).
			Bool("disable_scheduler", cfg.Scaling.DisableScheduler).
			Bool("disable_realtime", cfg.Scaling.DisableRealtime).
			Bool("leader_election", cfg.Scaling.EnableSchedulerLeaderElection).
			Str("backend", cfg.Scaling.Backend).
			Msg("Scaling configuration active")
	}

	// Initialize image transformation library (vips) if enabled
	var cleanupVips func()
	if cfg.Storage.Transforms.Enabled {
		log.Info().Msg("Initializing image transformation library (libvips)...")
		storage.InitVips()
		cleanupVips = func() {
			log.Debug().Msg("Shutting down image transformation library...")
			storage.ShutdownVips()
		}
		log.Info().
			Int("max_width", cfg.Storage.Transforms.MaxWidth).
			Int("max_height", cfg.Storage.Transforms.MaxHeight).
			Int("default_quality", cfg.Storage.Transforms.DefaultQuality).
			Msg("Image transformations enabled")
	}

	// If validate flag is set, exit after validation
	if *validateConfig {
		log.Info().Msg("Configuration validation successful")

		// Test database connection
		log.Info().Msg("Testing database connection...")
		db, err := connectDatabaseWithRetry(cfg.Database, 1)
		if err != nil {
			db.Close()
			if cleanupVips != nil {
				cleanupVips()
			}
			log.Error().Err(err).Msg("Database connection test failed")
			os.Exit(1)
		}
		log.Info().Msg("Database connection test successful")
		db.Close() // Close connection after test

		log.Info().Msg("All validation checks passed")
		if cleanupVips != nil {
			cleanupVips()
		}
		os.Exit(0)
	}

	// Initialize database connection with retry logic
	db, err := connectDatabaseWithRetry(cfg.Database, maxRetryAttempts)
	if err != nil {
		if cleanupVips != nil {
			cleanupVips()
		}
		log.Error().Err(err).Msg("Failed to connect to database after multiple attempts")
		os.Exit(1)
	}

	// Run bootstrap (extensions, schemas, roles, default privileges)
	// This handles operations that pgschema cannot manage
	log.Info().Msg("Running database bootstrap...")
	bootstrapConfig := bootstrap.Config{
		Host:          cfg.Database.Host,
		Port:          cfg.Database.Port,
		Database:      cfg.Database.Database,
		User:          cfg.Database.User,
		Password:      cfg.Database.Password,
		AdminUser:     cfg.Database.AdminUser,
		AdminPassword: cfg.Database.AdminPassword,
	}
	bootstrapSvc := bootstrap.NewServiceWithConfig(db.Pool(), bootstrapConfig)
	if err := bootstrapSvc.EnsureBootstrap(context.Background()); err != nil {
		if cleanupVips != nil {
			cleanupVips()
		}
		db.Close()
		log.Error().Err(err).Msg("Failed to run bootstrap")
		os.Exit(1)
	}
	log.Info().Msg("Database bootstrap completed successfully")

	// Apply declarative schema (tables, indexes, functions, policies)
	// This uses pgschema to apply the internal Fluxbase schema
	log.Info().Msg("Applying declarative schema...")

	// Extract embedded schema files to a temp directory so they work
	// regardless of the deployment environment (Docker, bare metal, etc.)
	schemaDir, err := schema.ExtractSchemas()
	if err != nil {
		if cleanupVips != nil {
			cleanupVips()
		}
		db.Close()
		log.Error().Err(err).Msg("Failed to extract embedded schemas")
		os.Exit(1)
	}
	defer func() { _ = os.RemoveAll(schemaDir) }()

	declarativeConfig := migrations.DeclarativeConfig{
		SchemaDir:        schemaDir,
		Schemas:          migrations.DefaultFluxbaseSchemas,
		AllowDestructive: false,
		LockTimeout:      30,
	}
	// Apply admin credential fallback: if admin user/password are not
	// explicitly set, use the runtime user/password (same as connection.go).
	adminUser := cfg.Database.AdminUser
	if adminUser == "" {
		adminUser = cfg.Database.User
	}
	adminPassword := cfg.Database.AdminPassword
	if adminPassword == "" {
		adminPassword = cfg.Database.Password
	}

	declarativeSvc := migrations.NewDeclarativeService(
		"pgschema",
		cfg.Database.Host,
		cfg.Database.Port,
		adminUser,
		adminPassword,
		cfg.Database.Database,
		declarativeConfig,
	)
	declarativeSvc.SetPool(db.Pool())
	declarativeSvc.SetAppUser(cfg.Database.User)

	// Detect migration state for smooth transition from imperative to declarative
	validator := migrations.NewValidator(declarativeSvc, db.Pool())
	migrationState, err := validator.DetectMigrationState(context.Background())
	if err != nil {
		log.Warn().Err(err).Msg("Failed to detect migration state, continuing with startup")
	}

	// Determine source based on migration state
	// - "fresh_install": New installation with no prior migrations
	// - "transitioned": Existing installation with imperative migrations
	// - "schema_apply": Default when state is unknown
	source := "fresh_install"
	if migrationState != nil && migrationState.HasImperativeMigrations && !migrationState.HasDeclarativeState {
		// Existing installation detected - log and proceed with declarative
		// Note: Dirty migrations are not blocking - declarative system compares
		// actual DB state to desired state regardless of how it got there
		log.Info().
			Int64("last_migration_version", migrationState.LastAppliedVersion).
			Bool("had_dirty_migrations", migrationState.HasDirtyMigrations).
			Msg("Detected existing installation with imperative migrations - proceeding with declarative schema")
		source = "transitioned"
	} else if migrationState != nil && migrationState.HasDeclarativeState {
		// Already using declarative system
		source = "schema_apply"
	}

	if err := declarativeSvc.ApplyDeclarativeWithSource(context.Background(), source); err != nil {
		if cleanupVips != nil {
			cleanupVips()
		}
		db.Close()
		log.Error().Err(err).Msg("Failed to apply declarative schema")
		os.Exit(1)
	}
	log.Info().Str("source", source).Msg("Declarative schema applied successfully")

	// Recreate the pool after migrations to clear any stale prepared statement cache
	// Migrations can invalidate cached statement plans, causing panics in pgx
	// We use RecreatePool() instead of Reset() to avoid edge cases where Reset()
	// can cause the pool to enter a closed state
	log.Debug().Msg("Recreating connection pool after migrations...")
	if err := db.RecreatePool(); err != nil {
		log.Warn().Err(err).Msg("Failed to recreate connection pool, continuing with existing pool")
	}

	// Ensure default tenant and service keys exist
	log.Info().Msg("Initializing default tenant and service keys...")
	if err := ensureDefaultTenantAndKeys(db.Pool(), cfg); err != nil {
		log.Warn().Err(err).Msg("Failed to initialize default tenant and keys - continuing startup")
	}

	// Backfill NULL tenant_id to default tenant for pre-multi-tenant data.
	// Must run after ensureDefaultTenantAndKeys because the default tenant
	// doesn't exist during bootstrap (which runs earlier in the startup).
	if err := backfillTenantIDToDefault(db.Pool()); err != nil {
		log.Warn().Err(err).Msg("Failed to backfill tenant_id for pre-tenant data - continuing startup")
	}
	// Initialize tenant config loader for multi-tenant configuration overrides
	tenantConfigLoader, err := config.NewTenantConfigLoader(cfg)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to load tenant configurations")
	}
	log.Info().
		Int("tenant_configs", len(tenantConfigLoader.GetLoadedSlugs())).
		Str("config_dir", cfg.Tenants.ConfigDir).
		Msg("Tenant configuration loader initialized")

	// Initialize API server
	server := api.NewServer(cfg, db, Version)

	// Set tenant config loader for multi-tenant config overrides
	server.SetTenantConfigLoader(tenantConfigLoader)

	// Generate and set service role and anon keys for edge functions
	// These are JWT tokens that edge functions can use to call the Fluxbase API
	jwtManager, err := auth.NewJWTManagerWithConfig(
		cfg.Auth.JWTSecret,
		cfg.Auth.JWTExpiry,
		cfg.Auth.RefreshExpiry,
		cfg.Auth.ServiceRoleTTL,
		cfg.Auth.AnonTTL,
	)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to create JWT manager")
	}

	// Get default tenant ID for JWT claims
	defaultTenantID := getDefaultTenantID(db.Pool())

	// Generate service role token (full admin access, bypasses RLS)
	serviceRoleKey, err := jwtManager.GenerateServiceRoleTokenWithTenant(defaultTenantID)
	if err != nil {
		log.Warn().Err(err).Msg("Failed to generate service role key")
	} else {
		if err := os.Setenv("FLUXBASE_SERVICE_ROLE_KEY", serviceRoleKey); err != nil {
			log.Warn().Err(err).Msg("Failed to set FLUXBASE_SERVICE_ROLE_KEY")
		}
		log.Debug().Msg("Service role key generated for edge functions")
	}

	// Generate anon token (public access)
	anonKey, err := jwtManager.GenerateAnonTokenWithTenant(defaultTenantID)
	if err != nil {
		log.Warn().Err(err).Msg("Failed to generate anon key")
	} else {
		if err := os.Setenv("FLUXBASE_ANON_KEY", anonKey); err != nil {
			log.Warn().Err(err).Msg("Failed to set FLUXBASE_ANON_KEY")
		}
		log.Debug().Msg("Anon key generated for edge functions")
	}

	// Ensure BASE_URL is set for edge functions (internal URL for server-to-server communication)
	if os.Getenv("FLUXBASE_BASE_URL") == "" {
		baseURL := fmt.Sprintf("http://%s", strings.TrimPrefix(cfg.Server.Address, ":"))
		if strings.HasPrefix(cfg.Server.Address, ":") {
			baseURL = fmt.Sprintf("http://localhost%s", cfg.Server.Address)
		}
		if err := os.Setenv("FLUXBASE_BASE_URL", baseURL); err != nil {
			log.Warn().Err(err).Msg("Failed to set FLUXBASE_BASE_URL")
		}
		log.Debug().Str("url", baseURL).Msg("Base URL set for edge functions")
	}

	// Log the public URL configuration if it differs from base URL
	if cfg.PublicBaseURL != "" && cfg.PublicBaseURL != cfg.BaseURL {
		log.Info().
			Str("public_url", cfg.PublicBaseURL).
			Str("internal_url", cfg.BaseURL).
			Msg("Using separate public and internal URLs")
	}

	// Validate storage provider health
	log.Info().Msg("Validating storage provider...")
	if err := validateStorageHealth(server); err != nil {
		if cleanupVips != nil {
			cleanupVips()
		}
		db.Close() // Explicitly close since defer won't run with os.Exit
		log.Error().Err(err).Msg("Storage validation failed")
		os.Exit(1)
	}
	log.Info().Str("provider", cfg.Storage.Provider).Msg("Storage provider validated successfully")

	// Auto-load functions from filesystem if enabled
	if cfg.Functions.Enabled && cfg.Functions.AutoLoadOnBoot {
		log.Info().Msg("Auto-loading edge functions from filesystem...")
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		defer cancel()

		if err := server.LoadFunctionsFromFilesystem(ctx); err != nil {
			log.Warn().Err(err).Msg("Failed to auto-load functions - continuing startup")
		} else {
			log.Info().Msg("Functions auto-loaded successfully")
		}
	}

	// Auto-load jobs from filesystem if enabled
	if cfg.Jobs.Enabled && cfg.Jobs.AutoLoadOnBoot {
		log.Info().Msg("Auto-loading job functions from filesystem...")
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		defer cancel()

		if err := server.LoadJobsFromFilesystem(ctx); err != nil {
			log.Warn().Err(err).Msg("Failed to auto-load jobs - continuing startup")
		} else {
			log.Info().Msg("Job functions auto-loaded successfully")
		}
	}

	// Start server in a goroutine (unless in worker-only mode)
	if cfg.Scaling.WorkerOnly {
		log.Info().Msg("Running in worker-only mode - API server disabled, only processing background jobs")
	} else {
		go func() {
			log.Info().Str("address", cfg.Server.Address).Msg("Starting Fluxbase server")
			if err := server.Start(); err != nil {
				// Log at ERROR level to make server startup failures visible
				// This includes port binding errors, network issues, etc.
				log.Error().Err(err).Msg("Server failed to start or stopped with error")
			}
		}()
	}

	// Wait for interrupt signal to gracefully shutdown the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info().Msg("Shutting down server...")

	// Graceful shutdown with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Error().Err(err).Msg("Graceful shutdown failed")
	}

	// Close database connection AFTER server shutdown completes
	// This ensures all workers and background services have stopped
	log.Debug().Msg("Closing database connection...")
	db.Close()

	log.Info().Msg("Server exited")

	// Safety: force exit after a short delay if the process hasn't exited
	// This handles edge cases where goroutines might keep the process alive
	go func() {
		time.Sleep(2 * time.Second)
		log.Warn().Msg("Force exiting - cleanup took too long")
		os.Exit(0)
	}()
}

// getEnvInt retrieves an integer environment variable or returns a default value
func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

// connectDatabaseWithRetry attempts to connect to the database with exponential backoff
func connectDatabaseWithRetry(cfg config.DatabaseConfig, maxAttempts int) (*database.Connection, error) {
	var db *database.Connection
	var err error

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		log.Info().
			Int("attempt", attempt).
			Int("max_attempts", maxAttempts).
			Str("host", cfg.Host).
			Int("port", cfg.Port).
			Msg("Attempting to connect to database...")

		db, err = database.NewConnection(cfg)
		if err == nil {
			log.Info().Msg("Successfully connected to database")
			return db, nil
		}

		// If this was the last attempt, return the error
		if attempt >= maxAttempts {
			break
		}

		// Calculate exponential backoff (1s, 2s, 4s, 8s, 16s)
		backoff := time.Duration(math.Pow(2, float64(attempt-1))) * time.Second
		log.Warn().
			Err(err).
			Int("attempt", attempt).
			Dur("retry_in", backoff).
			Msg("Database connection failed, retrying...")
		time.Sleep(backoff)
	}

	return nil, fmt.Errorf("failed to connect after %d attempts: %w", maxAttempts, err)
}

// validateStorageHealth checks if the storage provider is accessible
func validateStorageHealth(server *api.Server) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Access the storage service from the server
	storageService := server.GetStorageService()
	if storageService == nil {
		return fmt.Errorf("storage service not initialized")
	}

	// Perform health check
	if err := storageService.Provider.Health(ctx); err != nil {
		return fmt.Errorf("storage health check failed: %w", err)
	}

	return nil
}

// printConfigSummary logs a summary of the current configuration
func printConfigSummary(cfg *config.Config) {
	log.Info().Msg("Configuration Summary:")
	log.Info().Str("base_url", cfg.BaseURL).Str("public_base_url", cfg.GetPublicBaseURL()).Msg("  Base URL")
	log.Info().Str("address", cfg.Server.Address).Msg("  Server Address")
	log.Info().
		Str("host", cfg.Database.Host).
		Int("port", cfg.Database.Port).
		Str("database", cfg.Database.Database).
		Str("user", cfg.Database.User).
		Str("admin_user", cfg.Database.AdminUser).
		Str("ssl_mode", cfg.Database.SSLMode).
		Msg("  Database")
	log.Info().
		Str("provider", cfg.Storage.Provider).
		Str("path", getStoragePath(cfg.Storage)).
		Msg("  Storage")
	log.Info().
		Str("jwt_expiry", cfg.Auth.JWTExpiry.String()).
		Bool("signup_enabled", cfg.Auth.SignupEnabled).
		Bool("magic_link_enabled", cfg.Auth.MagicLinkEnabled).
		Msg("  Authentication")
	log.Info().
		Bool("email_enabled", cfg.Email.Enabled).
		Str("email_provider", getEmailProviderInfo(cfg.Email)).
		Msg("  Email")
	log.Info().
		Bool("realtime_enabled", cfg.Realtime.Enabled).
		Msg("  Realtime")
	log.Info().
		Bool("functions_enabled", cfg.Functions.Enabled).
		Str("functions_dir", cfg.Functions.FunctionsDir).
		Bool("auto_load_on_boot", cfg.Functions.AutoLoadOnBoot).
		Msg("  Functions")
	log.Info().
		Bool("jobs_enabled", cfg.Jobs.Enabled).
		Str("jobs_dir", cfg.Jobs.JobsDir).
		Bool("auto_load_on_boot", cfg.Jobs.AutoLoadOnBoot).
		Int("embedded_workers", cfg.Jobs.EmbeddedWorkerCount).
		Msg("  Jobs")
	log.Info().Bool("debug_mode", cfg.Debug).Msg("  Debug Mode")
}

// getStoragePath returns the appropriate storage path/info based on provider
func getStoragePath(storage config.StorageConfig) string {
	if storage.Provider == "local" {
		return storage.LocalPath
	}
	return storage.S3Bucket
}

// getEmailProviderInfo returns email provider info with masked credentials
func getEmailProviderInfo(email config.EmailConfig) string {
	if !email.Enabled {
		return "disabled"
	}
	if email.Provider == "smtp" && email.SMTPHost != "" {
		return fmt.Sprintf("smtp (%s:%d)", email.SMTPHost, email.SMTPPort)
	}
	return email.Provider
}

// ensureDefaultTenantAndKeys ensures the default tenant and service keys exist
func ensureDefaultTenantAndKeys(pool *pgxpool.Pool, cfg *config.Config) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	var tenantID uuid.UUID
	var tenantExists bool

	// Check if default tenant exists (with retry for transient connection errors
	// that can occur right after pool recreation).
	var err error
	for attempt := 0; attempt < 3; attempt++ {
		err = pool.QueryRow(ctx,
			"SELECT id, true FROM platform.tenants WHERE slug = 'default' AND deleted_at IS NULL",
		).Scan(&tenantID, &tenantExists)
		if err == nil {
			break
		}
		if isNoRowsError(err) {
			err = nil
			break
		}
		// Also check for the standard pgx "no rows" message
		if strings.Contains(err.Error(), "no rows in result set") {
			err = nil
			break
		}
		if attempt < 2 {
			log.Warn().Err(err).Msg("Retrying default tenant check due to connection error")
			time.Sleep(500 * time.Millisecond)
		}
	}
	if err != nil && !isNoRowsError(err) {
		return fmt.Errorf("failed to check for default tenant: %w", err)
	}

	if !tenantExists {
		// Create default tenant
		tenantName := cfg.Tenants.Default.Name
		if tenantName == "" {
			tenantName = "Default"
		}

		err := pool.QueryRow(ctx,
			"INSERT INTO platform.tenants (slug, name, is_default) VALUES ('default', $1, true) RETURNING id",
			tenantName,
		).Scan(&tenantID)
		if err != nil {
			return fmt.Errorf("failed to create default tenant: %w", err)
		}
		log.Info().Str("id", tenantID.String()).Msg("Created default tenant")
	} else {
		log.Debug().Str("id", tenantID.String()).Msg("Default tenant already exists")
	}

	// Handle service key
	if err := ensureServiceKey(ctx, pool, cfg, tenantID, keys.KeyTypeTenantService); err != nil {
		return fmt.Errorf("failed to ensure service key: %w", err)
	}

	// Handle anon key
	if err := ensureServiceKey(ctx, pool, cfg, tenantID, keys.KeyTypeAnon); err != nil {
		return fmt.Errorf("failed to ensure anon key: %w", err)
	}

	return nil
}

// ensureServiceKey ensures a service key exists for the given type
// For database-per-tenant, keys are stored in auth.service_keys in the tenant's database
func ensureServiceKey(ctx context.Context, pool *pgxpool.Pool, cfg *config.Config, tenantID uuid.UUID, keyType string) error {
	var configKey string
	var keyName string

	switch keyType {
	case keys.KeyTypeTenantService:
		configKey = cfg.Tenants.Default.ServiceKey
		if cfg.Tenants.Default.ServiceKeyFile != "" {
			if data, err := os.ReadFile(cfg.Tenants.Default.ServiceKeyFile); err == nil {
				configKey = strings.TrimSpace(string(data))
			} else {
				log.Warn().Err(err).Str("file", cfg.Tenants.Default.ServiceKeyFile).Msg("Failed to read service key file")
			}
		}
		keyName = "Default Service Key"
	case keys.KeyTypeAnon:
		configKey = cfg.Tenants.Default.AnonKey
		if cfg.Tenants.Default.AnonKeyFile != "" {
			if data, err := os.ReadFile(cfg.Tenants.Default.AnonKeyFile); err == nil {
				configKey = strings.TrimSpace(string(data))
			} else {
				log.Warn().Err(err).Str("file", cfg.Tenants.Default.AnonKeyFile).Msg("Failed to read anon key file")
			}
		}
		keyName = "Default Anon Key"
	default:
		return fmt.Errorf("unsupported key type: %s", keyType)
	}

	// Check for existing key of this type in auth.service_keys
	var existingKeyID uuid.UUID
	var existingKeyHash string
	err := pool.QueryRow(ctx,
		"SELECT id, key_hash FROM auth.service_keys WHERE key_type = $1 AND enabled = true AND revoked_at IS NULL",
		keyType,
	).Scan(&existingKeyID, &existingKeyHash)
	hasExistingKey := err == nil

	if configKey != "" {
		// Config-managed key provided
		keyHash, err := keys.HashKey(configKey)
		if err != nil {
			return fmt.Errorf("failed to hash config key: %w", err)
		}

		if hasExistingKey {
			// Check if hash matches
			if keys.VerifyKey(configKey, existingKeyHash) {
				log.Debug().Str("type", keyType).Msg("Config-managed key already stored")
				return nil
			}
			// Key changed, disable old and create new
			_, err := pool.Exec(ctx,
				"UPDATE auth.service_keys SET enabled = false WHERE id = $1",
				existingKeyID,
			)
			if err != nil {
				return fmt.Errorf("failed to disable old key: %w", err)
			}
		}

		// Insert new config-managed key
		keyPrefix := keys.ExtractPrefix(configKey)
		_, err = pool.Exec(ctx,
			`INSERT INTO auth.service_keys 
			(name, key_hash, key_prefix, key_type, enabled, scopes, rate_limit_per_minute)
			VALUES ($1, $2, $3, $4, true, $5, $6)`,
			keyName, keyHash, keyPrefix, keyType, getDefaultScopes(keyType), getDefaultRateLimit(keyType),
		)
		if err != nil {
			return fmt.Errorf("failed to insert config-managed key: %w", err)
		}

		log.Info().Str("type", keyType).Msg("Stored config-managed key")
		return nil
	}

	// No config key provided
	if hasExistingKey {
		log.Debug().Str("type", keyType).Msg("Service key already exists")
		return nil
	}

	// Generate new key
	_, keyHash, keyPrefix, err := keys.GenerateKey(keyType)
	if err != nil {
		return fmt.Errorf("failed to generate key: %w", err)
	}

	// Insert generated key
	_, err = pool.Exec(ctx,
		`INSERT INTO auth.service_keys 
		(name, key_hash, key_prefix, key_type, enabled, scopes, rate_limit_per_minute)
		VALUES ($1, $2, $3, $4, true, $5, $6)`,
		keyName, keyHash, keyPrefix, keyType, getDefaultScopes(keyType), getDefaultRateLimit(keyType),
	)
	if err != nil {
		return fmt.Errorf("failed to insert generated key: %w", err)
	}

	log.Info().
		Str("type", keyType).
		Str("prefix", keyPrefix).
		Msg("Generated new service key - configure via tenants.default.anon_key or tenants.default.service_key to persist")

	return nil
}

// getDefaultScopes returns default scopes for a key type
func getDefaultScopes(keyType string) []string {
	switch keyType {
	case keys.KeyTypeTenantService:
		return []string{"*"}
	case keys.KeyTypeAnon:
		return []string{"read"}
	default:
		return []string{}
	}
}

// getDefaultRateLimit returns default rate limit for a key type
func getDefaultRateLimit(keyType string) int {
	switch keyType {
	case keys.KeyTypeTenantService:
		return 10000
	case keys.KeyTypeAnon:
		return 60
	default:
		return 60
	}
}

// getDefaultTenantID returns the default tenant ID
func getDefaultTenantID(pool *pgxpool.Pool) *string {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var tenantID string
	err := pool.QueryRow(ctx,
		"SELECT id::text FROM platform.tenants WHERE slug = 'default' AND deleted_at IS NULL",
	).Scan(&tenantID)
	if err != nil {
		log.Debug().Err(err).Msg("Failed to get default tenant ID")
		return nil
	}
	return &tenantID
}

// isNoRowsError checks if the error is a "no rows" error
func isNoRowsError(err error) bool {
	return err != nil && (err.Error() == "no rows in result set" || strings.Contains(err.Error(), "no rows"))
}

// backfillTenantIDToDefault assigns NULL tenant_id rows to the default tenant.
// This handles the upgrade path from pre-multi-tenant Fluxbase where all data
// was created without tenant context. Without this backfill, selecting the
// default tenant in the admin UI makes all pre-existing data invisible because
// has_tenant_access(NULL) returns FALSE when a tenant context is active.
//
// Tables where NULL is semantically meaningful (settings cascade, shared OAuth
// providers, global service keys) are excluded from the backfill.
func backfillTenantIDToDefault(pool *pgxpool.Pool) error {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Get default tenant ID
	var defaultTenantID string
	err := pool.QueryRow(ctx,
		"SELECT id::text FROM platform.tenants WHERE is_default = true AND deleted_at IS NULL LIMIT 1",
	).Scan(&defaultTenantID)
	if err != nil {
		if err.Error() == "no rows in result set" || strings.Contains(err.Error(), "no rows") {
			log.Debug().Msg("No default tenant found, skipping tenant_id backfill")
			return nil
		}
		return fmt.Errorf("failed to get default tenant: %w", err)
	}

	// Tables to backfill: user data that should belong to the default tenant.
	// Excluded:
	//   - platform.instance_settings (NULL = instance defaults shared with all tenants)
	//   - platform.oauth_providers (NULL = shared SSO providers)
	//   - platform.service_keys (NULL = global_service keys)
	//   - platform.enabled_extensions (NULL = instance-level extensions)
	//   - auth.service_keys (NULL = global_service keys)
	//   - auth.saml_providers (NULL = shared SAML providers, like OAuth)
	tenantIDDedupTables := map[string][]string{
		"auth.users":               {"email"},
		"functions.edge_functions": {"name, namespace"},
		"storage.buckets":          {"name"},
		"branching.branches":       {"name", "slug"},
		"branching.github_config":  {"repository"},
		"mcp.custom_tools":         {"name, namespace"},
		"mcp.custom_resources":     {"uri, namespace"},
	}

	tables := []string{
		"auth.users",
		"auth.client_keys",
		"auth.impersonation_sessions",
		"auth.webhooks",
		"auth.webhook_deliveries",
		"auth.webhook_events",
		"auth.saml_providers",
		"auth.sessions",
		"auth.oauth_links",
		"auth.oauth_tokens",
		"auth.mfa_factors",
		"auth.saml_sessions",
		"auth.magic_links",
		"auth.otp_codes",
		"auth.email_verification_tokens",
		"auth.password_reset_tokens",
		"auth.two_factor_setups",
		"auth.two_factor_recovery_attempts",
		"auth.oauth_logout_states",
		"auth.mcp_oauth_clients",
		"auth.mcp_oauth_codes",
		"auth.mcp_oauth_tokens",
		"auth.client_key_usage",
		"auth.service_key_revocations",
		"functions.edge_functions",
		"functions.edge_executions",
		"functions.edge_files",
		"functions.edge_triggers",
		"functions.secrets",
		"functions.secret_versions",
		"functions.shared_modules",
		"functions.function_dependencies",
		"jobs.functions",
		"jobs.function_files",
		"jobs.workers",
		"jobs.queue",
		"ai.knowledge_bases",
		"ai.knowledge_base_permissions",
		"ai.documents",
		"ai.document_permissions",
		"ai.chunks",
		"ai.entities",
		"ai.document_entities",
		"ai.entity_relationships",
		"ai.providers",
		"ai.chatbots",
		"ai.chatbot_knowledge_bases",
		"ai.conversations",
		"ai.messages",
		"ai.query_audit_log",
		"ai.retrieval_log",
		"ai.table_export_sync_configs",
		"ai.user_chatbot_usage",
		"ai.user_provider_preferences",
		"ai.user_quotas",
		"rpc.procedures",
		"rpc.executions",
		"realtime.schema_registry",
		"storage.buckets",
		"storage.objects",
		"storage.chunked_upload_sessions",
		"storage.object_permissions",
		"branching.branches",
		"branching.activity_log",
		"branching.branch_access",
		"branching.github_config",
		"branching.migration_history",
		"branching.seed_execution_log",
		"logging.entries",
		"logging.entries_ai",
		"logging.entries_custom",
		"logging.entries_execution",
		"logging.entries_http",
		"logging.entries_security",
		"logging.entries_system",
		"mcp.custom_resources",
		"mcp.custom_tools",
		"platform.invitation_tokens",
	}

	var totalBackfilled int
	for _, table := range tables {
		if dedupSets, needsDedup := tenantIDDedupTables[table]; needsDedup {
			for _, dedupCols := range dedupSets {
				dedupQuery := fmt.Sprintf(
					"DELETE FROM %s WHERE id IN (SELECT id FROM (SELECT id, row_number() OVER (PARTITION BY %s ORDER BY created_at DESC) AS rn FROM %s WHERE tenant_id IS NULL) sub WHERE rn > 1)",
					table, dedupCols, table,
				)
				if dedupResult, err := pool.Exec(ctx, dedupQuery); err != nil {
					log.Warn().Err(err).Str("table", table).Str("cols", dedupCols).Msg("Failed to dedup NULL-tenant rows before backfill")
				} else if n := dedupResult.RowsAffected(); n > 0 {
					log.Info().Str("table", table).Str("cols", dedupCols).Int64("duplicates_removed", n).Msg("Removed duplicate NULL-tenant rows before backfill")
				}

				conflictQuery := fmt.Sprintf(
					"DELETE FROM %s WHERE id IN (SELECT n.id FROM %s n WHERE n.tenant_id IS NULL AND EXISTS (SELECT 1 FROM %s e WHERE e.tenant_id = $1 AND (%s)))",
					table, table, table, buildJoinCondition(dedupCols, "n", "e"),
				)
				if conflictResult, err := pool.Exec(ctx, conflictQuery, defaultTenantID); err != nil {
					log.Warn().Err(err).Str("table", table).Str("cols", dedupCols).Msg("Failed to remove NULL-tenant rows conflicting with existing tenant rows")
				} else if n := conflictResult.RowsAffected(); n > 0 {
					log.Info().Str("table", table).Str("cols", dedupCols).Int64("conflicts_removed", n).Msg("Removed NULL-tenant rows that would conflict with existing tenant rows")
				}
			}
		}

		result, err := pool.Exec(ctx,
			fmt.Sprintf("UPDATE %s SET tenant_id = $1::uuid WHERE tenant_id IS NULL", table),
			defaultTenantID,
		)
		if err != nil {
			log.Warn().Err(err).Str("table", table).Msg("Failed to backfill tenant_id")
			continue
		}
		if n := result.RowsAffected(); n > 0 {
			log.Info().Str("table", table).Int64("rows", n).Msg("Backfilled tenant_id to default tenant")
			totalBackfilled += int(n)
		}
	}

	if totalBackfilled > 0 {
		log.Info().Int("total_rows", totalBackfilled).Msg("Tenant_id backfill complete")
	} else {
		log.Debug().Msg("No NULL tenant_id rows found to backfill")
	}

	return nil
}

func buildJoinCondition(cols, leftAlias, rightAlias string) string {
	parts := strings.Split(cols, ",")
	conditions := make([]string, len(parts))
	for i, col := range parts {
		col = strings.TrimSpace(col)
		conditions[i] = fmt.Sprintf("%s.%s = %s.%s", leftAlias, col, rightAlias, col)
	}
	return strings.Join(conditions, " AND ")
}
