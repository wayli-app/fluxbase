//go:build integration

// Package dbhelpers provides minimal database test helpers for integration tests.
// This package intentionally does NOT import internal/api to avoid import cycles.
package dbhelpers

import (
	"context"
	"fmt"
	"math"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/nimbleflux/fluxbase/internal/config"
)

const (
	// defaultDBRetryAttempts is the number of times to retry database connection
	defaultDBRetryAttempts = 5
	// defaultDBHealthTimeout is the timeout for database health checks
	defaultDBHealthTimeout = 10 * time.Second
)

func init() {
	// Load .env file if it exists (for local development)
	_ = godotenv.Load()

	// Initialize test logger with default config
	initTestLogger(nil)
}

// initTestLogger initializes the logger for testing.
func initTestLogger(cfg *config.Config) {
	// Enable debug logging if FLUXBASE_LOG_DEBUG is set
	logLevel := zerolog.InfoLevel
	if cfg != nil && cfg.Debug {
		logLevel = zerolog.DebugLevel
	} else if os.Getenv("FLUXBASE_LOG_DEBUG") == "true" {
		logLevel = zerolog.DebugLevel
	}

	// Configure console logger for tests
	zerolog.SetGlobalLevel(logLevel)
	log.Logger = log.Output(zerolog.ConsoleWriter{
		Out:        os.Stdout,
		TimeFormat: "15:04:05",
		NoColor:    os.Getenv("CI") != "",
	})
}

// DBTestContext holds database connection pool and configuration for integration tests.
//
// Always close the context with defer tc.Close() to ensure proper cleanup.
type DBTestContext struct {
	Pool     *pgxpool.Pool
	DBConfig config.DatabaseConfig
	Config   *config.Config
	T        *testing.T
}

// NewDBTestContext creates a database test context using the fluxbase_app database user.
//
// This function intentionally does NOT create an API server to avoid import cycles.
// Use this for integration tests that only need database access.
//
// Example:
//
//	func TestSchemaInspector(t *testing.T) {
//	    testCtx := dbhelpers.NewDBTestContext(t)
//	    defer testCtx.Close()
//
//	    inspector := database.NewSchemaInspector(testCtx.Pool)
//	    // ... perform tests
//	}
func NewDBTestContext(t *testing.T) *DBTestContext {
	cfg := GetTestConfig()
	return newDBTestContextInternal(t, cfg)
}

// newDBTestContextInternal creates a DB test context without checking shared context.
func newDBTestContextInternal(t *testing.T, cfg *config.Config) *DBTestContext {
	// Initialize logger
	if cfg != nil {
		initTestLogger(cfg)
	}

	// Log the database configuration
	log.Info().
		Str("db_user", cfg.Database.User).
		Str("db_host", cfg.Database.Host).
		Str("db_database", cfg.Database.Database).
		Msg("Integration test database configuration")

	// Build connection URL
	connURL := buildConnectionURL(cfg.Database)

	// Create connection pool with retry logic
	pool, err := connectPoolWithRetry(connURL, defaultDBRetryAttempts)
	if err != nil {
		t.Fatalf("Failed to connect to test database after retries: %v", err)
	}

	return &DBTestContext{
		Pool:     pool,
		DBConfig: cfg.Database,
		Config:   cfg,
		T:        t,
	}
}

// connectPoolWithRetry attempts to connect to the test database with exponential backoff.
func connectPoolWithRetry(connURL string, maxAttempts int) (*pgxpool.Pool, error) {
	var pool *pgxpool.Pool
	var err error

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		log.Debug().
			Int("attempt", attempt).
			Int("max_attempts", maxAttempts).
			Msg("Attempting to connect to test database...")

		pool, err = pgxpool.New(context.Background(), connURL)
		if err == nil {
			// Connection successful, verify health
			ctx, cancel := context.WithTimeout(context.Background(), defaultDBHealthTimeout)
			healthErr := pool.Ping(ctx)
			cancel()

			if healthErr == nil {
				log.Debug().Msg("Test database connection and health check successful")
				return pool, nil
			}

			// Health check failed, close connection and retry
			pool.Close()
			err = healthErr
		}

		// If this was the last attempt, return the error
		if attempt >= maxAttempts {
			break
		}

		// Calculate exponential backoff (1s, 2s, 4s, 8s, 16s)
		backoff := time.Duration(math.Pow(2, float64(attempt-1))) * time.Second
		log.Debug().
			Err(err).
			Int("attempt", attempt).
			Dur("retry_in", backoff).
			Msg("Test database connection failed, retrying...")
		time.Sleep(backoff)
	}

	return nil, fmt.Errorf("failed to connect to test database after %d attempts: %w", maxAttempts, err)
}

// buildConnectionURL builds a PostgreSQL connection URL from config.
func buildConnectionURL(cfg config.DatabaseConfig) string {
	return fmt.Sprintf(
		"postgresql://%s:%s@%s:%d/%s?sslmode=%s",
		cfg.User,
		cfg.Password,
		cfg.Host,
		cfg.Port,
		cfg.Database,
		cfg.SSLMode,
	)
}

// Close closes the database connection pool.
func (tc *DBTestContext) Close() {
	if tc.Pool != nil {
		tc.Pool.Close()
	}
}

// ExecuteSQL executes a SQL query as the fluxbase_app user.
// This is a convenience method for test cleanup operations.
func (tc *DBTestContext) ExecuteSQL(query string, args ...interface{}) {
	_, err := tc.Pool.Exec(context.Background(), query, args...)
	if err != nil && tc.T != nil {
		tc.T.Logf("Warning: ExecuteSQL failed: %v", err)
	}
}

// DatabaseURL returns the database connection URL.
func (tc *DBTestContext) DatabaseURL() string {
	return buildConnectionURL(tc.DBConfig)
}

// GetTestConfig returns the test configuration from environment variables.
func GetTestConfig() *config.Config {
	return &config.Config{
		Database: config.DatabaseConfig{
			Host:            getEnv("FLUXBASE_DB_HOST", "postgres"),
			Port:            parseInt(getEnv("FLUXBASE_DB_PORT", "5432")),
			Database:        getEnv("FLUXBASE_DB_DATABASE", "fluxbase_dev"),
			User:            getEnv("FLUXBASE_DB_USER", "fluxbase_app"),
			Password:        getEnv("FLUXBASE_DB_PASSWORD", "fluxbase_app_password"),
			AdminUser:       getEnv("FLUXBASE_DB_ADMIN_USER", "postgres"),
			AdminPassword:   getEnv("FLUXBASE_DB_ADMIN_PASSWORD", "postgres"),
			SSLMode:         getEnv("FLUXBASE_DB_SSLMODE", "disable"),
			MaxConnections:  int32(parseInt(getEnv("FLUXBASE_DB_MAX_CONNECTIONS", "25"))),
			MinConnections:  int32(parseInt(getEnv("FLUXBASE_DB_MIN_CONNECTIONS", "5"))),
			MaxConnLifetime: parseDuration(getEnv("FLUXBASE_DB_MAX_CONN_LIFETIME", "5m")),
			MaxConnIdleTime: parseDuration(getEnv("FLUXBASE_DB_MAX_CONN_IDLE_TIME", "5m")),
			HealthCheck:     parseDuration(getEnv("FLUXBASE_DB_HEALTH_CHECK", "30s")),
		},
		Debug: os.Getenv("FLUXBASE_LOG_DEBUG") == "true",
	}
}

// getEnv gets an environment variable, trying both FLUXBASE_DATABASE_* and FLUXBASE_DB_* prefixes.
// This allows compatibility with docker-compose (FLUXBASE_DATABASE_*) and other configs (FLUXBASE_DB_*).
func getEnv(key, defaultVal string) string {
	// Try with FLUXBASE_DATABASE_ prefix first (docker-compose convention)
	if val := os.Getenv(strings.Replace(key, "FLUXBASE_DB_", "FLUXBASE_DATABASE_", 1)); val != "" {
		return val
	}
	// Fall back to FLUXBASE_DB_ prefix
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}

func parseInt(s string) int {
	var val int
	_, _ = fmt.Sscanf(s, "%d", &val)
	return val
}

func parseDuration(s string) time.Duration {
	val, _ := time.ParseDuration(s)
	return val
}
