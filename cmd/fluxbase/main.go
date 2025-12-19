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

	"github.com/fluxbase-eu/fluxbase/internal/api"
	"github.com/fluxbase-eu/fluxbase/internal/auth"
	"github.com/fluxbase-eu/fluxbase/internal/config"
	"github.com/fluxbase-eu/fluxbase/internal/database"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
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
	workerOnly       = flag.Bool("worker-only", false, "Run in worker-only mode (disable API server, only process background jobs)")
	disableScheduler = flag.Bool("disable-scheduler", false, "Disable cron schedulers (use for multi-instance deployments)")
	disableRealtime  = flag.Bool("disable-realtime", false, "Disable realtime listener")
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

	// If validate flag is set, exit after validation
	if *validateConfig {
		log.Info().Msg("Configuration validation successful")

		// Test database connection
		log.Info().Msg("Testing database connection...")
		db, err := connectDatabaseWithRetry(cfg.Database, 1)
		if err != nil {
			log.Fatal().Err(err).Msg("Database connection test failed")
		}
		db.Close()
		log.Info().Msg("Database connection test successful")

		log.Info().Msg("All validation checks passed")
		os.Exit(0)
	}

	// Initialize database connection with retry logic
	db, err := connectDatabaseWithRetry(cfg.Database, maxRetryAttempts)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to connect to database after multiple attempts")
	}
	defer db.Close()

	// Run migrations
	log.Info().Msg("Running database migrations...")
	if err := db.Migrate(); err != nil {
		log.Fatal().Err(err).Msg("Failed to run migrations")
	}
	log.Info().Msg("Database migrations completed successfully")

	// Reset the pool after migrations to clear any stale prepared statement cache
	// Migrations can invalidate cached statement plans, causing panics in pgx
	log.Debug().Msg("Resetting connection pool after migrations...")
	db.Pool().Reset()
	log.Debug().Msg("Connection pool reset complete")

	// Initialize API server
	server := api.NewServer(cfg, db, Version)

	// Generate and set service role and anon keys for edge functions
	// These are JWT tokens that edge functions can use to call the Fluxbase API
	jwtManager := auth.NewJWTManager(cfg.Auth.JWTSecret, cfg.Auth.JWTExpiry, cfg.Auth.RefreshExpiry)

	// Generate service role token (full admin access, bypasses RLS)
	serviceRoleKey, err := jwtManager.GenerateServiceRoleToken()
	if err != nil {
		log.Warn().Err(err).Msg("Failed to generate service role key")
	} else {
		if err := os.Setenv("FLUXBASE_SERVICE_ROLE_KEY", serviceRoleKey); err != nil {
			log.Warn().Err(err).Msg("Failed to set FLUXBASE_SERVICE_ROLE_KEY")
		}
		log.Debug().Msg("Service role key generated for edge functions")
	}

	// Generate anon token (public access)
	anonKey, err := jwtManager.GenerateAnonToken()
	if err != nil {
		log.Warn().Err(err).Msg("Failed to generate anon key")
	} else {
		if err := os.Setenv("FLUXBASE_ANON_KEY", anonKey); err != nil {
			log.Warn().Err(err).Msg("Failed to set FLUXBASE_ANON_KEY")
		}
		log.Debug().Msg("Anon key generated for edge functions")
	}

	// Ensure BASE_URL is set for edge functions
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

	// Validate storage provider health
	log.Info().Msg("Validating storage provider...")
	if err := validateStorageHealth(server); err != nil {
		log.Fatal().Err(err).Msg("Storage validation failed")
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
	log.Info().Str("base_url", cfg.BaseURL).Msg("  Base URL")
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
		Bool("signup_enabled", cfg.Auth.EnableSignup).
		Bool("magic_link_enabled", cfg.Auth.EnableMagicLink).
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
