package database

import (
	"context"
	"embed"
	"fmt"
	"os"
	"time"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/pgx/v5"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog/log"
	"github.com/wayli-app/fluxbase/internal/config"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

// Connection represents a database connection pool
type Connection struct {
	pool      *pgxpool.Pool
	config    *config.DatabaseConfig
	inspector *SchemaInspector
}

// NewConnection creates a new database connection pool
func NewConnection(cfg config.DatabaseConfig) (*Connection, error) {
	poolConfig, err := pgxpool.ParseConfig(cfg.ConnectionString())
	if err != nil {
		return nil, fmt.Errorf("unable to parse connection string: %w", err)
	}

	// Configure pool settings
	poolConfig.MaxConns = cfg.MaxConnections
	poolConfig.MinConns = cfg.MinConnections
	poolConfig.MaxConnLifetime = cfg.MaxConnLifetime
	poolConfig.MaxConnIdleTime = cfg.MaxConnIdleTime
	poolConfig.HealthCheckPeriod = cfg.HealthCheck

	// Create connection pool
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return nil, fmt.Errorf("unable to create connection pool: %w", err)
	}

	// Test the connection
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("unable to ping database: %w", err)
	}

	conn := &Connection{
		pool:   pool,
		config: &cfg,
	}

	// Initialize schema inspector
	conn.inspector = NewSchemaInspector(conn)

	log.Info().Str("database", cfg.Database).Msg("Database connection established")

	return conn, nil
}

// Close closes the database connection pool
func (c *Connection) Close() {
	c.pool.Close()
	log.Info().Msg("Database connection closed")
}

// Pool returns the underlying connection pool
func (c *Connection) Pool() *pgxpool.Pool {
	return c.pool
}

// Migrate runs database migrations from both system and user sources
func (c *Connection) Migrate() error {
	// Step 1: Run system migrations (embedded in binary)
	log.Info().Msg("Running system migrations...")
	if err := c.runSystemMigrations(); err != nil {
		return fmt.Errorf("failed to run system migrations: %w", err)
	}

	// Step 2: Run user migrations (from file system) if path is configured
	if c.config.UserMigrationsPath != "" {
		log.Info().Str("path", c.config.UserMigrationsPath).Msg("Running user migrations...")
		if err := c.runUserMigrations(); err != nil {
			return fmt.Errorf("failed to run user migrations: %w", err)
		}
	} else {
		log.Debug().Msg("No user migrations path configured, skipping user migrations")
	}

	return nil
}

// runSystemMigrations runs migrations embedded in the binary
func (c *Connection) runSystemMigrations() error {
	// Ensure _fluxbase schema exists before migrations run
	// This is needed because the migration system needs the schema to exist
	// before it can create the schema_migrations table
	ctx := context.Background()
	_, err := c.pool.Exec(ctx, "CREATE SCHEMA IF NOT EXISTS _fluxbase")
	if err != nil {
		return fmt.Errorf("failed to create _fluxbase schema: %w", err)
	}

	// Create migrations source from embedded filesystem
	sourceDriver, err := iofs.New(migrationsFS, "migrations")
	if err != nil {
		return fmt.Errorf("failed to create migration source: %w", err)
	}

	// Use connection string with system migrations table
	connStr := fmt.Sprintf("pgx5://%s:%s@%s:%d/%s?sslmode=%s&x-migrations-table=\"_fluxbase\".\"schema_migrations\"&x-migrations-table-quoted=1",
		c.config.User,
		c.config.Password,
		c.config.Host,
		c.config.Port,
		c.config.Database,
		c.config.SSLMode,
	)

	// Create migration instance
	m, err := migrate.NewWithSourceInstance("iofs", sourceDriver, connStr)
	if err != nil {
		return fmt.Errorf("failed to create migration instance: %w", err)
	}
	defer m.Close()

	// Run migrations with error handling
	if err := c.applyMigrations(m, "system"); err != nil {
		return err
	}

	return nil
}

// runUserMigrations runs migrations from the user-specified directory
func (c *Connection) runUserMigrations() error {
	// Check if directory exists
	if _, err := os.Stat(c.config.UserMigrationsPath); os.IsNotExist(err) {
		log.Warn().Str("path", c.config.UserMigrationsPath).Msg("User migrations directory does not exist, skipping")
		return nil
	}

	// Use connection string with user migrations table
	connStr := fmt.Sprintf("pgx5://%s:%s@%s:%d/%s?sslmode=%s&x-migrations-table=\"_fluxbase\".\"user_migrations\"&x-migrations-table-quoted=1",
		c.config.User,
		c.config.Password,
		c.config.Host,
		c.config.Port,
		c.config.Database,
		c.config.SSLMode,
	)

	// Create migration instance from file system
	sourceURL := fmt.Sprintf("file://%s", c.config.UserMigrationsPath)
	m, err := migrate.New(sourceURL, connStr)
	if err != nil {
		return fmt.Errorf("failed to create user migration instance: %w", err)
	}
	defer m.Close()

	// Run migrations with error handling
	if err := c.applyMigrations(m, "user"); err != nil {
		return err
	}

	return nil
}

// applyMigrations applies pending migrations and handles errors
func (c *Connection) applyMigrations(m *migrate.Migrate, source string) error {
	// Check current version and dirty state
	version, dirty, err := m.Version()
	if err != nil && err != migrate.ErrNilVersion {
		return fmt.Errorf("failed to get migration version: %w", err)
	}

	// If database is in dirty state, force the version to clean it
	if dirty {
		log.Warn().Str("source", source).Uint("version", version).Msg("Database is in dirty state, forcing version to clean")
		if err := m.Force(int(version)); err != nil {
			return fmt.Errorf("failed to force migration version: %w", err)
		}
	}

	// Run migrations
	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("failed to run %s migrations: %w", source, err)
	}

	if err == migrate.ErrNoChange {
		log.Info().Str("source", source).Msg("No new migrations to apply")
	} else {
		version, _, _ := m.Version()
		log.Info().Str("source", source).Uint("version", version).Msg("Migrations applied successfully")
	}

	return nil
}

// BeginTx starts a new transaction
func (c *Connection) BeginTx(ctx context.Context) (pgx.Tx, error) {
	return c.pool.Begin(ctx)
}

// Query executes a query that returns rows
func (c *Connection) Query(ctx context.Context, sql string, args ...interface{}) (pgx.Rows, error) {
	start := time.Now()
	rows, err := c.pool.Query(ctx, sql, args...)
	duration := time.Since(start)

	// Log slow queries (> 1 second)
	if duration > 1*time.Second {
		log.Warn().
			Dur("duration", duration).
			Int64("duration_ms", duration.Milliseconds()).
			Str("query", truncateQuery(sql, 200)).
			Bool("slow_query", true).
			Msg("Slow query detected")
	}

	return rows, err
}

// QueryRow executes a query that returns a single row
func (c *Connection) QueryRow(ctx context.Context, sql string, args ...interface{}) pgx.Row {
	start := time.Now()
	row := c.pool.QueryRow(ctx, sql, args...)
	duration := time.Since(start)

	// Log slow queries (> 1 second)
	if duration > 1*time.Second {
		log.Warn().
			Dur("duration", duration).
			Int64("duration_ms", duration.Milliseconds()).
			Str("query", truncateQuery(sql, 200)).
			Bool("slow_query", true).
			Msg("Slow query detected")
	}

	return row
}

// Exec executes a query that doesn't return rows
func (c *Connection) Exec(ctx context.Context, sql string, args ...interface{}) (pgconn.CommandTag, error) {
	start := time.Now()
	tag, err := c.pool.Exec(ctx, sql, args...)
	duration := time.Since(start)

	// Log slow queries (> 1 second)
	if duration > 1*time.Second {
		log.Warn().
			Dur("duration", duration).
			Int64("duration_ms", duration.Milliseconds()).
			Str("query", truncateQuery(sql, 200)).
			Bool("slow_query", true).
			Msg("Slow query detected")
	}

	return tag, err
}

// Inspector returns the schema inspector
func (c *Connection) Inspector() *SchemaInspector {
	return c.inspector
}

// Health checks the health of the database connection
func (c *Connection) Health(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	var result int
	err := c.QueryRow(ctx, "SELECT 1").Scan(&result)
	if err != nil {
		return fmt.Errorf("health check failed: %w", err)
	}

	if result != 1 {
		return fmt.Errorf("unexpected health check result: %d", result)
	}

	return nil
}

// Stats returns database connection pool statistics
func (c *Connection) Stats() *pgxpool.Stat {
	return c.pool.Stat()
}

// truncateQuery truncates a SQL query to a maximum length for logging
func truncateQuery(query string, maxLen int) string {
	if len(query) <= maxLen {
		return query
	}
	return query[:maxLen] + "... (truncated)"
}
