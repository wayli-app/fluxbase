package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/nimbleflux/fluxbase/internal/api"
	"github.com/nimbleflux/fluxbase/internal/auth"
	"github.com/nimbleflux/fluxbase/internal/config"
	"github.com/nimbleflux/fluxbase/internal/database"
	"github.com/nimbleflux/fluxbase/internal/database/bootstrap"
	"github.com/nimbleflux/fluxbase/internal/database/schema"
	"github.com/nimbleflux/fluxbase/internal/extensions"
	"github.com/nimbleflux/fluxbase/internal/migrations"
	"github.com/nimbleflux/fluxbase/internal/storage"
	"github.com/nimbleflux/fluxbase/internal/tenantdb"
)

var (
	// Version information (set via ldflags during build)
	Version   = "dev"
	Commit    = "unknown"
	BuildDate = "unknown"

	// CLI flags
	showVersion    = flag.Bool("version", false, "Show version information")
	validateConfig = flag.Bool("validate", false, "Validate configuration and exit")
	// Legacy env override for the DB connection retry count. When unset, the
	// value comes from the config (database.retry_attempts, default 8).
	envRetryAttempts = getEnvInt("FLUXBASE_DATABASE_RETRY_ATTEMPTS", 0)

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
		db, err := database.ConnectWithRetry(cfg.Database, 1)
		if err != nil {
			if db != nil {
				db.Close()
			}
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

	// Initialize database connection with retry logic. The legacy
	// FLUXBASE_DATABASE_RETRY_ATTEMPTS env var overrides the config attempt count
	// for the initial connection when set.
	connectAttempts := cfg.Database.RetryAttempts
	if envRetryAttempts > 0 {
		connectAttempts = envRetryAttempts
	}
	db, err := database.ConnectWithRetry(cfg.Database, connectAttempts)
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
	// Bootstrap opens its own admin pool internally, so a transient DB blip after
	// the initial connect surfaces here. Retry transient errors before aborting.
	runStartupStep("database bootstrap", dbRetryConfig(cfg.Database), func() {
		if cleanupVips != nil {
			cleanupVips()
		}
		db.Close()
	}, func() error {
		return bootstrapSvc.EnsureBootstrap(context.Background())
	})
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

	// The declarative apply opens its own pools, so retry transient errors
	// before aborting startup.
	runStartupStep("declarative schema apply", dbRetryConfig(cfg.Database), func() {
		if cleanupVips != nil {
			cleanupVips()
		}
		db.Close()
	}, func() error {
		return declarativeSvc.ApplyDeclarativeWithSource(context.Background(), source)
	})
	log.Info().Str("source", source).Msg("Declarative schema applied successfully")

	// Recreate the pool after migrations to clear any stale prepared statement cache
	// Migrations can invalidate cached statement plans, causing panics in pgx
	// We use RecreatePool() instead of Reset() to avoid edge cases where Reset()
	// can cause the pool to enter a closed state
	log.Debug().Msg("Recreating connection pool after migrations...")
	if err := db.RecreatePool(); err != nil {
		log.Warn().Err(err).Msg("Failed to recreate connection pool, continuing with existing pool")
	}

	// Sync the extension catalog from pg_available_extensions so that
	// `fluxbase extensions enable <name>` works out-of-the-box. Must run AFTER
	// the declarative schema apply (which creates platform.available_extensions).
	extService := extensions.NewService(db)
	if count, err := extService.SyncExtensionCatalog(context.Background()); err != nil {
		log.Warn().Err(err).Msg("Failed to sync extension catalog, continuing")
	} else if count > 0 {
		log.Info().Int("synced", count).Msg("Extension catalog synced from PostgreSQL")
	}

	// Apply declarative app schema (opt-in). Lets an application developer manage
	// their own tables (e.g. the `public` schema) declaratively via synced schema
	// content, as an alternative to imperative user migrations. Off by default —
	// no-op unless database.declarative_app_schema.enabled is true.
	if cfg.Database.DeclarativeAppSchema != nil && cfg.Database.DeclarativeAppSchema.Enabled {
		log.Info().Msg("Applying declarative app schema...")
		appDeclarative := migrations.NewAppDeclarativeService(
			"pgschema",
			cfg.Database.Host,
			cfg.Database.Port,
			adminUser,
			adminPassword,
			cfg.Database.Database,
			cfg.Database.DeclarativeAppSchema.AllowDestructive,
		)
		appDeclarative.SetPool(db.Pool())
		appDeclarative.SetAppUser(cfg.Database.User)

		appNamespaces := cfg.Database.DeclarativeAppSchema.Namespaces
		if cfg.Database.DeclarativeAppSchema.OnStartup {
			// App-schema apply opens its own pools; retry transient errors.
			runStartupStep("declarative app schema apply", dbRetryConfig(cfg.Database), nil, func() error {
				return appDeclarative.ApplyAllPending(context.Background(), appNamespaces)
			})
		}

		// Coexistence guard: warn (do not fail) if the same namespace also has
		// imperative migrations in platform.migrations. A (namespace, schema) should
		// be owned by one mode; the developer controls which sync commands run.
		appDeclarative.WarnIfImperativeCoexists(context.Background(), appNamespaces)

		// Refresh the pool again — app-schema DDL may invalidate cached plans.
		log.Debug().Msg("Recreating connection pool after app schema...")
		if err := db.RecreatePool(); err != nil {
			log.Warn().Err(err).Msg("Failed to recreate connection pool after app schema, continuing")
		}
		log.Info().Msg("Declarative app schema applied successfully")
	}

	// Ensure default tenant and service keys exist
	log.Info().Msg("Initializing default tenant and service keys...")
	if err := tenantdb.EnsureDefaultTenantAndKeys(db.Pool(), cfg); err != nil {
		log.Warn().Err(err).Msg("Failed to initialize default tenant and keys - continuing startup")
	}

	// Backfill NULL tenant_id to default tenant for pre-multi-tenant data.
	// Must run after ensureDefaultTenantAndKeys because the default tenant
	// doesn't exist during bootstrap (which runs earlier in the startup).
	if err := tenantdb.BackfillTenantIDToDefault(db.Pool()); err != nil {
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

	// Generate service role and anon keys for external clients (SDKs, CLI, MCP)
	// Note: these keys are NOT passed to edge functions (blocked by runtime/env.go).
	// Edge functions receive per-request tokens (FLUXBASE_SERVICE_TOKEN, FLUXBASE_USER_TOKEN).
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
	defaultTenantID := tenantdb.GetDefaultTenantID(db.Pool())

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
		log.Error().Err(err).Msg("Storage validation failed")

		if cleanupVips != nil {
			cleanupVips()
		}
		// Stop background services (e.g. the log-retention goroutine started in
		// NewServer) BEFORE closing the pool. Connection.Close() sets pool=nil,
		// so an in-flight retention cleanup that calls conn.Pool().Begin would
		// dereference a nil pool and panic. The graceful-shutdown path below
		// already does shutdown-then-close; this early-exit path must match it.
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
		if err := server.Shutdown(shutdownCtx); err != nil {
			log.Warn().Err(err).Msg("Graceful shutdown failed during validation error exit")
		}
		shutdownCancel()
		db.Close() // Explicitly close since defer won't run with os.Exit
		os.Exit(1)
	}
	log.Info().Str("provider", cfg.Storage.Provider).Msg("Storage provider validated successfully")

	// Auto-load functions from filesystem if enabled
	if cfg.Functions.Enabled && cfg.Functions.AutoLoadOnBoot {
		log.Info().Msg("Auto-loading edge functions from filesystem...")
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		defer cancel()

		if err := server.Functions.Handler.LoadFromFilesystem(ctx); err != nil {
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

		if err := server.Jobs.Handler.LoadFromFilesystem(ctx, "default"); err != nil {
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

// dbRetryConfig builds the RetryConfig for DB-dependent startup steps from the
// database config. The legacy FLUXBASE_DATABASE_RETRY_ATTEMPTS env var, when set,
// overrides the configured attempt count for the initial connection only.
func dbRetryConfig(db config.DatabaseConfig) database.RetryConfig {
	rc := database.DefaultRetryConfig()
	if db.RetryAttempts > 0 {
		rc.MaxAttempts = db.RetryAttempts
	}
	if db.RetryInitialBackoff > 0 {
		rc.InitialBackoff = db.RetryInitialBackoff
	}
	if db.RetryMaxBackoff > 0 {
		rc.MaxBackoff = db.RetryMaxBackoff
	}
	return rc
}

// runStartupStep runs a DB-dependent startup step with transient-error retry.
// On success it returns nil; on final failure it logs and exits the process
// (matching the prior os.Exit(1) behavior). cleanup runs once before exit when
// the step fails terminally (e.g. close the DB, release vips).
func runStartupStep(name string, rc database.RetryConfig, cleanup func(), fn func() error) {
	err := database.RetryTransient(context.Background(), name, rc, fn)
	if err == nil {
		return
	}
	if cleanup != nil {
		cleanup()
	}
	log.Error().Err(err).Msg(name + " failed")
	os.Exit(1)
}

// validateStorageHealth checks if the storage provider is accessible
func validateStorageHealth(server *api.Server) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	storageService := server.GetStorageService()
	if storageService == nil {
		return fmt.Errorf("storage service not initialized")
	}

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

func getStoragePath(storage config.StorageConfig) string {
	if storage.Provider == "local" {
		return storage.LocalPath
	}
	return storage.S3Bucket
}

func getEmailProviderInfo(email config.EmailConfig) string {
	if !email.Enabled {
		return "disabled"
	}
	if email.Provider == "smtp" && email.SMTPHost != "" {
		return fmt.Sprintf("smtp (%s:%d)", email.SMTPHost, email.SMTPPort)
	}
	return email.Provider
}
