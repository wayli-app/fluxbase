package database

import (
	"context"
	"embed"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/fluxbase-eu/fluxbase/internal/config"
	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/pgx/v5"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog/log"
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
// The connection pool uses the runtime user, while migrations use the admin user
func NewConnection(cfg config.DatabaseConfig) (*Connection, error) {
	// Use runtime connection string for the connection pool
	poolConfig, err := pgxpool.ParseConfig(cfg.RuntimeConnectionString())
	if err != nil {
		return nil, fmt.Errorf("unable to parse connection string: %w", err)
	}

	// Configure pool settings
	poolConfig.MaxConns = cfg.MaxConnections
	poolConfig.MinConns = cfg.MinConnections
	poolConfig.MaxConnLifetime = cfg.MaxConnLifetime
	poolConfig.MaxConnIdleTime = cfg.MaxConnIdleTime
	poolConfig.HealthCheckPeriod = cfg.HealthCheck

	// Register custom types for PostgreSQL-specific types that pgx doesn't handle by default
	// This allows scanning tsvector, tsquery, and other types into interface{}
	poolConfig.AfterConnect = func(ctx context.Context, conn *pgx.Conn) error {
		// Register tsvector (OID 3614) as text
		conn.TypeMap().RegisterType(&pgtype.Type{
			Name:  "tsvector",
			OID:   3614,
			Codec: pgtype.TextCodec{},
		})
		// Register tsquery (OID 3615) as text
		conn.TypeMap().RegisterType(&pgtype.Type{
			Name:  "tsquery",
			OID:   3615,
			Codec: pgtype.TextCodec{},
		})
		// Register regclass (OID 2205) as text - used in some system views
		conn.TypeMap().RegisterType(&pgtype.Type{
			Name:  "regclass",
			OID:   2205,
			Codec: pgtype.TextCodec{},
		})
		return nil
	}

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

	log.Info().
		Str("database", cfg.Database).
		Str("user", cfg.User).
		Msg("Database connection established")

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

	// Step 3: Grant Fluxbase roles to runtime user
	// This allows the application to SET ROLE for RLS and service operations
	if err := c.grantRolesToRuntimeUser(); err != nil {
		return fmt.Errorf("failed to grant roles to runtime user: %w", err)
	}

	return nil
}

// runSystemMigrations runs migrations embedded in the binary
func (c *Connection) runSystemMigrations() error {
	// Ensure migrations schema and fluxbase table exist before migrations run
	// This is needed because the migration system needs the table to exist
	// We must connect as admin user to create the schema and table
	ctx := context.Background()

	// Create a temporary admin connection for schema setup
	// Use AdminPassword if set, otherwise fall back to Password
	adminPassword := c.config.AdminPassword
	if adminPassword == "" {
		adminPassword = c.config.Password
	}
	adminConnStr := fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=%s",
		c.config.AdminUser,
		adminPassword,
		c.config.Host,
		c.config.Port,
		c.config.Database,
		c.config.SSLMode,
	)

	adminConn, err := pgx.Connect(ctx, adminConnStr)
	if err != nil {
		return fmt.Errorf("failed to connect as admin user: %w", err)
	}
	defer func() { _ = adminConn.Close(ctx) }()

	// Create migrations schema as admin (if not exists)
	_, err = adminConn.Exec(ctx, "CREATE SCHEMA IF NOT EXISTS migrations")
	if err != nil {
		return fmt.Errorf("failed to create migrations schema: %w", err)
	}

	// Ensure the fluxbase migrations table exists as admin
	// The migrate library expects this table to exist in the specified schema
	_, err = adminConn.Exec(ctx, `CREATE TABLE IF NOT EXISTS "migrations"."fluxbase" (
		version bigint NOT NULL PRIMARY KEY,
		dirty boolean NOT NULL
	)`)
	if err != nil {
		return fmt.Errorf("failed to create migrations.fluxbase table: %w", err)
	}

	// Create migrations source from embedded filesystem
	sourceDriver, err := iofs.New(migrationsFS, "migrations")
	if err != nil {
		return fmt.Errorf("failed to create migration source: %w", err)
	}

	// Use connection string with system migrations table (admin user for migrations)
	connStr := fmt.Sprintf("pgx5://%s:%s@%s:%d/%s?sslmode=%s&x-migrations-table=\"migrations\".\"fluxbase\"&x-migrations-table-quoted=1",
		c.config.AdminUser,
		adminPassword,
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
	defer func() {
		srcErr, dbErr := m.Close()
		if srcErr != nil || dbErr != nil {
			log.Debug().AnErr("srcErr", srcErr).AnErr("dbErr", dbErr).Msg("Migration close returned errors")
		}
	}()

	// Run migrations with error handling
	if err := c.applyMigrations(m, "system"); err != nil {
		return err
	}

	return nil
}

// runUserMigrations runs migrations from the user-specified directory
// Migrations are tracked in migrations.app with namespace='filesystem'
func (c *Connection) runUserMigrations() error {
	// Check if directory exists
	if _, err := os.Stat(c.config.UserMigrationsPath); os.IsNotExist(err) {
		log.Warn().Str("path", c.config.UserMigrationsPath).Msg("User migrations directory does not exist, skipping")
		return nil
	}

	ctx := context.Background()

	// Use AdminPassword if set, otherwise fall back to Password
	adminPassword := c.config.AdminPassword
	if adminPassword == "" {
		adminPassword = c.config.Password
	}

	// Create admin connection for migrations
	adminConnStr := fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=%s",
		c.config.AdminUser,
		adminPassword,
		c.config.Host,
		c.config.Port,
		c.config.Database,
		c.config.SSLMode,
	)

	adminConn, err := pgx.Connect(ctx, adminConnStr)
	if err != nil {
		return fmt.Errorf("failed to connect as admin user: %w", err)
	}
	defer func() { _ = adminConn.Close(ctx) }()

	// Scan filesystem for migration files
	migrations, err := c.scanMigrationFiles(c.config.UserMigrationsPath)
	if err != nil {
		return fmt.Errorf("failed to scan migration files: %w", err)
	}

	if len(migrations) == 0 {
		log.Info().Str("path", c.config.UserMigrationsPath).Msg("No migration files found")
		return nil
	}

	// Get already-applied migrations from database
	applied, err := c.getAppliedMigrations(ctx, adminConn)
	if err != nil {
		return fmt.Errorf("failed to get applied migrations: %w", err)
	}

	// Apply new migrations in order
	appliedCount := 0
	for _, m := range migrations {
		if applied[m.Name] {
			continue
		}

		log.Info().Str("name", m.Name).Msg("Applying filesystem migration")

		start := time.Now()
		if err := c.applyFilesystemMigration(ctx, adminConn, m); err != nil {
			// Log the failure
			c.logMigrationExecution(ctx, adminConn, m.Name, "apply", "failed", time.Since(start), err.Error())
			return fmt.Errorf("failed to apply migration %s: %w", m.Name, err)
		}

		// Log success
		c.logMigrationExecution(ctx, adminConn, m.Name, "apply", "success", time.Since(start), "")
		appliedCount++
	}

	if appliedCount > 0 {
		log.Info().Int("count", appliedCount).Msg("Filesystem migrations applied successfully")
	} else {
		log.Info().Msg("No new filesystem migrations to apply")
	}

	return nil
}

// migrationFile represents a migration file from the filesystem
type migrationFile struct {
	Name    string // e.g., "001_create_posts"
	UpSQL   string
	DownSQL string
}

// scanMigrationFiles scans a directory for migration files
func (c *Connection) scanMigrationFiles(dir string) ([]migrationFile, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("failed to read directory: %w", err)
	}

	// Map to collect up/down SQL by migration name
	migrationMap := make(map[string]*migrationFile)

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		var migName string
		var isUp bool

		if strings.HasSuffix(name, ".up.sql") {
			migName = strings.TrimSuffix(name, ".up.sql")
			isUp = true
		} else if strings.HasSuffix(name, ".down.sql") {
			migName = strings.TrimSuffix(name, ".down.sql")
			isUp = false
		} else {
			continue // Not a migration file
		}

		if _, exists := migrationMap[migName]; !exists {
			migrationMap[migName] = &migrationFile{Name: migName}
		}

		content, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			return nil, fmt.Errorf("failed to read file %s: %w", name, err)
		}

		if isUp {
			migrationMap[migName].UpSQL = string(content)
		} else {
			migrationMap[migName].DownSQL = string(content)
		}
	}

	// Convert map to sorted slice (migration names should be sortable, e.g., 001_, 002_)
	var migrations []migrationFile
	for _, m := range migrationMap {
		if m.UpSQL == "" {
			log.Warn().Str("name", m.Name).Msg("Migration missing .up.sql file, skipping")
			continue
		}
		migrations = append(migrations, *m)
	}

	// Sort by name (relies on naming convention like 001_, 002_, etc.)
	sort.Slice(migrations, func(i, j int) bool {
		return migrations[i].Name < migrations[j].Name
	})

	return migrations, nil
}

// getAppliedMigrations returns a set of already-applied filesystem migrations
func (c *Connection) getAppliedMigrations(ctx context.Context, conn *pgx.Conn) (map[string]bool, error) {
	applied := make(map[string]bool)

	rows, err := conn.Query(ctx, `
		SELECT name FROM migrations.app
		WHERE namespace = 'filesystem' AND status = 'applied'
	`)
	if err != nil {
		// Table might not exist yet on first run
		return applied, nil
	}
	defer rows.Close()

	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, fmt.Errorf("failed to scan migration name: %w", err)
		}
		applied[name] = true
	}

	return applied, rows.Err()
}

// applyFilesystemMigration applies a single filesystem migration
func (c *Connection) applyFilesystemMigration(ctx context.Context, conn *pgx.Conn, m migrationFile) error {
	tx, err := conn.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	// Insert migration record
	_, err = tx.Exec(ctx, `
		INSERT INTO migrations.app (namespace, name, up_sql, down_sql, status, applied_at)
		VALUES ('filesystem', $1, $2, $3, 'applied', NOW())
		ON CONFLICT (namespace, name) DO UPDATE SET
			status = 'applied',
			applied_at = NOW(),
			updated_at = NOW()
	`, m.Name, m.UpSQL, m.DownSQL)
	if err != nil {
		return fmt.Errorf("failed to insert migration record: %w", err)
	}

	// Execute the migration SQL
	_, err = tx.Exec(ctx, m.UpSQL)
	if err != nil {
		return fmt.Errorf("failed to execute migration SQL: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// logMigrationExecution logs a migration execution to the execution_logs table
func (c *Connection) logMigrationExecution(ctx context.Context, conn *pgx.Conn, migrationName, action, status string, duration time.Duration, errMsg string) {
	_, err := conn.Exec(ctx, `
		INSERT INTO migrations.execution_logs (migration_id, action, status, duration_ms, error_message, executed_at)
		SELECT id, $2, $3, $4, $5, NOW()
		FROM migrations.app
		WHERE namespace = 'filesystem' AND name = $1
	`, migrationName, action, status, duration.Milliseconds(), errMsg)
	if err != nil {
		log.Warn().Err(err).Str("migration", migrationName).Msg("Failed to log migration execution")
	}
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

// grantRolesToRuntimeUser grants Fluxbase roles to the runtime database user
// This allows the application to SET ROLE for RLS and service operations
// Only runs if runtime user is different from admin user
func (c *Connection) grantRolesToRuntimeUser() error {
	// Skip if runtime user is the same as admin user
	if c.config.User == c.config.AdminUser {
		log.Debug().Str("user", c.config.User).Msg("Runtime user is same as admin user, skipping role grants")
		return nil
	}

	ctx := context.Background()

	// Use admin connection to grant roles
	adminPassword := c.config.AdminPassword
	if adminPassword == "" {
		adminPassword = c.config.Password
	}

	adminConnStr := fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=%s",
		c.config.AdminUser,
		adminPassword,
		c.config.Host,
		c.config.Port,
		c.config.Database,
		c.config.SSLMode,
	)

	adminConn, err := pgx.Connect(ctx, adminConnStr)
	if err != nil {
		return fmt.Errorf("failed to connect as admin user: %w", err)
	}
	defer func() { _ = adminConn.Close(ctx) }()

	// Grant roles to runtime user
	roles := []string{"anon", "authenticated", "service_role"}
	for _, role := range roles {
		// Check if role exists before granting
		var exists bool
		err := adminConn.QueryRow(ctx,
			"SELECT EXISTS(SELECT FROM pg_catalog.pg_roles WHERE rolname = $1)",
			role,
		).Scan(&exists)

		if err != nil {
			log.Warn().Err(err).Str("role", role).Msg("Failed to check if role exists")
			continue
		}

		if exists {
			query := fmt.Sprintf("GRANT %s TO %s", role, c.config.User)
			_, err = adminConn.Exec(ctx, query)
			if err != nil {
				log.Warn().Err(err).Str("role", role).Str("user", c.config.User).Msg("Failed to grant role")
			} else {
				log.Debug().Str("role", role).Str("user", c.config.User).Msg("Granted role to runtime user")
			}
		}
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

// WrapWithServiceRole wraps a database operation with service_role context
// Used for privileged operations like auth, admin tasks, and webhooks
// This is equivalent to how Supabase's auth service (GoTrue) uses supabase_auth_admin
func WrapWithServiceRole(ctx context.Context, conn *Connection, fn func(tx pgx.Tx) error) error {
	// Start transaction
	tx, err := conn.Pool().Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	// SET LOCAL ROLE service_role - bypasses RLS for privileged operations
	// This provides the same security model as Supabase's separate admin connections
	_, err = tx.Exec(ctx, "SET LOCAL ROLE service_role")
	if err != nil {
		log.Error().Err(err).Msg("Failed to SET LOCAL ROLE service_role")
		return fmt.Errorf("failed to SET LOCAL ROLE service_role: %w", err)
	}

	// Execute the wrapped function
	if err := fn(tx); err != nil {
		return err
	}

	// Commit transaction
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// ExecuteWithAdminRole executes a database operation using admin credentials
// Used for migrations that require DDL privileges (CREATE TABLE, ALTER, etc.)
// Creates a temporary admin connection that is closed after execution
func (c *Connection) ExecuteWithAdminRole(ctx context.Context, fn func(conn *pgx.Conn) error) error {
	// Get admin connection string
	adminConnStr := c.config.AdminConnectionString()

	adminUser := c.config.AdminUser
	if adminUser == "" {
		adminUser = c.config.User
	}

	log.Info().
		Str("admin_user", adminUser).
		Str("database", c.config.Database).
		Str("host", c.config.Host).
		Msg("Connecting as admin user for migration")

	// Create admin connection
	adminConn, err := pgx.Connect(ctx, adminConnStr)
	if err != nil {
		log.Error().Err(err).Str("admin_user", adminUser).Msg("Failed to connect as admin user for migration")
		return fmt.Errorf("failed to connect as admin: %w", err)
	}
	defer func() { _ = adminConn.Close(ctx) }()

	// Verify we're connected as the expected user
	var currentUser string
	var sessionUser string
	err = adminConn.QueryRow(ctx, "SELECT CURRENT_USER, SESSION_USER").Scan(&currentUser, &sessionUser)
	if err != nil {
		log.Warn().Err(err).Msg("Failed to verify current user")
	} else {
		log.Info().
			Str("current_user", currentUser).
			Str("session_user", sessionUser).
			Msg("Executing migration with user")
	}

	// Start transaction
	tx, err := adminConn.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	// Execute the wrapped function with the connection
	if err := fn(adminConn); err != nil {
		return err
	}

	// Commit transaction
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	log.Debug().Msg("Migration executed successfully with admin privileges")
	return nil
}
