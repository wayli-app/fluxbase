package main

import (
	"context"
	"flag"
	"fmt"
	"math"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/wayli-app/fluxbase/internal/api"
	"github.com/wayli-app/fluxbase/internal/config"
	"github.com/wayli-app/fluxbase/internal/database"
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

	// Set log level
	if cfg.Debug {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	} else {
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
	}

	// Print configuration summary
	printConfigSummary(cfg)

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

	// Initialize API server
	server := api.NewServer(cfg, db)

	// Validate storage provider health
	log.Info().Msg("Validating storage provider...")
	if err := validateStorageHealth(server); err != nil {
		log.Fatal().Err(err).Msg("Storage validation failed")
	}
	log.Info().Str("provider", cfg.Storage.Provider).Msg("Storage provider validated successfully")

	// Start server in a goroutine
	go func() {
		log.Info().Str("address", cfg.Server.Address).Msg("Starting Fluxbase server")
		if err := server.Start(); err != nil {
			log.Fatal().Err(err).Msg("Failed to start server")
		}
	}()

	// Wait for interrupt signal to gracefully shutdown the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info().Msg("Shutting down server...")

	// Graceful shutdown
	ctx := context.Background()
	if err := server.Shutdown(ctx); err != nil {
		log.Fatal().Err(err).Msg("Server forced to shutdown")
	}

	log.Info().Msg("Server exited")
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
