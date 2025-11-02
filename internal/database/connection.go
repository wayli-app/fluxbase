package database

import (
	"context"
	"embed"
	"fmt"
	"time"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/pgx/v5"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
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

// Migrate runs database migrations
func (c *Connection) Migrate() error {
	// Create migrations source from embedded filesystem
	sourceDriver, err := iofs.New(migrationsFS, "migrations")
	if err != nil {
		return fmt.Errorf("failed to create migration source: %w", err)
	}

	// Use connection string with custom migrations table in _fluxbase schema
	// Note: dashboard.schema_migrations is for tracking DDL migrations, not golang-migrate migrations
	// We use "pgx5" scheme which is registered by the pgx/v5 driver
	// x-migrations-table-quoted=1 requires quoted schema.table format
	connStr := fmt.Sprintf("pgx5://%s:%s@%s:%d/%s?sslmode=%s&x-migrations-table=\"_fluxbase\".\"schema_migrations\"&x-migrations-table-quoted=1",
		c.config.User,
		c.config.Password,
		c.config.Host,
		c.config.Port,
		c.config.Database,
		c.config.SSLMode,
	)

	log.Debug().Str("connection_string", connStr).Msg("Migration connection string")

	// Create migration instance
	m, err := migrate.NewWithSourceInstance("iofs", sourceDriver, connStr)
	if err != nil {
		return fmt.Errorf("failed to create migration instance: %w", err)
	}
	defer m.Close()

	// Check current version and dirty state
	version, dirty, err := m.Version()
	if err != nil && err != migrate.ErrNilVersion {
		return fmt.Errorf("failed to get migration version: %w", err)
	}

	// If database is in dirty state, force the version to clean it
	if dirty {
		log.Warn().Uint("version", version).Msg("Database is in dirty state, forcing version to clean")
		if err := m.Force(int(version)); err != nil {
			return fmt.Errorf("failed to force migration version: %w", err)
		}
	}

	// Run migrations
	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	if err == migrate.ErrNoChange {
		log.Info().Msg("No new migrations to apply")
	} else {
		version, _, _ := m.Version()
		log.Info().Uint("version", version).Msg("Migrations applied successfully")
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
