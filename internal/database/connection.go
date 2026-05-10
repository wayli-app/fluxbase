package database

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	pg_query "github.com/pganalyze/pg_query_go/v6"
	"github.com/rs/zerolog/log"

	"github.com/nimbleflux/fluxbase/internal/config"
	"github.com/nimbleflux/fluxbase/internal/observability"
)

type callerKey struct{}

func WithCaller(ctx context.Context, caller string) context.Context {
	return context.WithValue(ctx, callerKey{}, caller)
}

// These aliases allow the middleware and handlers to use simpler type names
type (
	Querier      interface{}
	TxConnection = pgx.Tx
)

// errRow implements pgx.Row to return an error when the pool is closed.
type errRow struct{ err error }

func (r errRow) Scan(dest ...interface{}) error { return r.err }

// quoteIdentifier safely quotes a PostgreSQL identifier to prevent SQL injection.
// It wraps the identifier in double quotes and escapes any embedded double quotes.
func quoteIdentifier(identifier string) string {
	return `"` + strings.ReplaceAll(identifier, `"`, `""`) + `"`
}

// Connection represents a database connection pool
type Connection struct {
	pool               *pgxpool.Pool
	poolMu             sync.RWMutex
	config             *config.DatabaseConfig
	inspector          *SchemaInspector
	metrics            *observability.Metrics
	slowQueryTracker   *slowQueryTracker
	slowQueryThreshold time.Duration
}

// ExtractTableName attempts to extract the table name from a SQL query
// Returns "unknown" if the table cannot be determined
func ExtractTableName(sql string) string {
	sql = strings.ToUpper(strings.TrimSpace(sql))

	// Match common SQL patterns
	patterns := []struct {
		prefix string
		regex  *regexp.Regexp
	}{
		{"SELECT", regexp.MustCompile(`FROM\s+["']?(\w+)["']?`)},
		{"INSERT", regexp.MustCompile(`INTO\s+["']?(\w+)["']?`)},
		{"UPDATE", regexp.MustCompile(`UPDATE\s+["']?(\w+)["']?`)},
		{"DELETE", regexp.MustCompile(`FROM\s+["']?(\w+)["']?`)},
	}

	for _, p := range patterns {
		if strings.HasPrefix(sql, p.prefix) {
			if matches := p.regex.FindStringSubmatch(sql); len(matches) > 1 {
				return strings.ToLower(matches[1])
			}
		}
	}

	return "unknown"
}

// ExtractOperation extracts the SQL operation type from a query
func ExtractOperation(sql string) string {
	sql = strings.ToUpper(strings.TrimSpace(sql))
	switch {
	case strings.HasPrefix(sql, "SELECT"):
		return "select"
	case strings.HasPrefix(sql, "INSERT"):
		return "insert"
	case strings.HasPrefix(sql, "UPDATE"):
		return "update"
	case strings.HasPrefix(sql, "DELETE"):
		return "delete"
	default:
		return "other"
	}
}

// ExtractDDLMetadata extracts operation type and target from a DDL query for logging
// Returns a safe, redacted string like "CREATE TABLE users", "DROP INDEX idx_name"
func ExtractDDLMetadata(sql string) string {
	sql = strings.TrimSpace(sql)
	if sql == "" {
		return "empty"
	}

	// Extract operation
	operation := ExtractOperation(sql)

	// Try to extract table name for better logging
	tableName := ExtractTableName(sql)

	if tableName != "unknown" && tableName != "" {
		return fmt.Sprintf("%s (table: %s)", operation, tableName)
	}

	return operation
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

	// Use QueryExecModeDescribeExec to avoid prepared statement caching issues.
	// This prevents nil pointer dereferences in pgx when statements are invalidated
	// (e.g., after schema changes or extension creation like pgvector).
	// The tradeoff is slightly higher overhead per query, but more robust connections.
	poolConfig.ConnConfig.DefaultQueryExecMode = pgx.QueryExecModeDescribeExec

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

		// Register pgvector 'vector' type if the extension is installed
		// The OID is dynamic and assigned when the extension is created
		// Use a separate context with timeout to avoid leaving connection in bad state
		queryCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()

		var vectorOID uint32
		err := conn.QueryRow(queryCtx, "SELECT oid FROM pg_type WHERE typname = 'vector'").Scan(&vectorOID)
		if err == nil && vectorOID > 0 {
			conn.TypeMap().RegisterType(&pgtype.Type{
				Name:  "vector",
				OID:   vectorOID,
				Codec: pgtype.TextCodec{}, // Vectors are text-encoded as '[0.1,0.2,...]'
			})
			// log.Debug().Uint32("oid", vectorOID).Msg("Registered pgvector type")
		}
		// If pgvector is not installed, the query will fail silently and we skip registration

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

	slowQueryThreshold := cfg.SlowQueryThreshold
	if slowQueryThreshold <= 0 {
		slowQueryThreshold = 1 * time.Second
	}

	conn := &Connection{
		pool:               pool,
		config:             &cfg,
		slowQueryTracker:   newSlowQueryTracker(),
		slowQueryThreshold: slowQueryThreshold,
	}

	// Initialize schema inspector
	conn.inspector = NewSchemaInspector(conn)

	log.Info().
		Str("database", cfg.Database).
		Str("user", cfg.User).
		Msg("Database connection established")

	return conn, nil
}

// NewConnectionWithPool creates a new Connection wrapper around an existing pgxpool.Pool.
// This is useful for tests where you have a pre-configured pool.
func NewConnectionWithPool(pool *pgxpool.Pool) *Connection {
	return &Connection{pool: pool}
}

// Close closes the database connection pool
func (c *Connection) Close() {
	c.poolMu.Lock()
	p := c.pool
	c.pool = nil
	c.poolMu.Unlock()
	if p != nil {
		p.Close()
	}
	log.Info().Msg("Database connection closed")
}

// Pool returns the underlying connection pool
func (c *Connection) Pool() *pgxpool.Pool {
	if c == nil {
		return nil
	}
	c.poolMu.RLock()
	defer c.poolMu.RUnlock()
	return c.pool
}

// NewConnectionFromPool creates a minimal Connection wrapping an existing pool.
// Intended for tests only.
func NewConnectionFromPool(pool *pgxpool.Pool) *Connection {
	return &Connection{pool: pool}
}

// RecreatePool closes the current pool and creates a new one.
// This is safer than Reset() as it ensures a completely fresh pool state.
// Use this after schema changes (migrations) to avoid prepared statement cache issues.
func (c *Connection) RecreatePool() error {
	c.poolMu.RLock()
	oldPool := c.pool
	c.poolMu.RUnlock()

	// Create a new pool with the same configuration
	poolConfig, err := pgxpool.ParseConfig(c.config.RuntimeConnectionString())
	if err != nil {
		return fmt.Errorf("unable to parse connection string: %w", err)
	}

	// Apply same configuration as NewConnection
	poolConfig.MaxConns = c.config.MaxConnections
	poolConfig.MinConns = c.config.MinConnections
	poolConfig.MaxConnLifetime = c.config.MaxConnLifetime
	poolConfig.MaxConnIdleTime = c.config.MaxConnIdleTime
	poolConfig.HealthCheckPeriod = c.config.HealthCheck
	poolConfig.ConnConfig.DefaultQueryExecMode = pgx.QueryExecModeDescribeExec

	// Copy the AfterConnect hook logic for custom type registration
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

		// Register pgvector 'vector' type if the extension is installed
		queryCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()

		var vectorOID uint32
		err := conn.QueryRow(queryCtx, "SELECT oid FROM pg_type WHERE typname = 'vector'").Scan(&vectorOID)
		if err == nil && vectorOID > 0 {
			conn.TypeMap().RegisterType(&pgtype.Type{
				Name:  "vector",
				OID:   vectorOID,
				Codec: pgtype.TextCodec{}, // Vectors are text-encoded as '[0.1,0.2,...]'
			})
		}

		return nil
	}

	// Create new pool
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return fmt.Errorf("unable to create connection pool: %w", err)
	}

	// Test the new pool
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return fmt.Errorf("unable to ping database: %w", err)
	}

	c.poolMu.Lock()
	c.pool = pool
	c.poolMu.Unlock()

	// Close old pool outside the lock
	if oldPool != nil {
		oldPool.Close()
	}

	log.Info().Msg("Connection pool recreated successfully")
	return nil
}

// Migrate runs database migrations from user sources
// Note: Internal Fluxbase schema is now managed declaratively (see bootstrap + pgschema)
func (c *Connection) Migrate() error {
	// Run user migrations (from file system) if path is configured
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

// runUserMigrations runs migrations from the user-specified directory
// Migrations are tracked in platform.migrations with namespace='filesystem'
func (c *Connection) runUserMigrations() error {
	// Check if directory exists
	if _, err := os.Stat(c.config.UserMigrationsPath); os.IsNotExist(err) {
		log.Debug().Str("path", c.config.UserMigrationsPath).Msg("User migrations directory does not exist, skipping")
		return nil
	}

	ctx := context.Background()

	// Use AdminPassword if set, otherwise fall back to Password
	adminPassword := c.config.AdminPassword
	if adminPassword == "" {
		adminPassword = c.config.Password
	}

	// Create admin connection for migrations
	adminConnStr := fmt.Sprintf(
		"postgres://%s:%s@%s:%d/%s?sslmode=%s",
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

		//nolint:gocritic
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

		sql := string(content)

		// Validate SQL syntax before applying
		if err := c.validateMigrationSQL(sql, migName); err != nil {
			return nil, fmt.Errorf("invalid SQL in migration file %s: %w", name, err)
		}

		if isUp {
			migrationMap[migName].UpSQL = sql
		} else {
			migrationMap[migName].DownSQL = sql
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
		SELECT name FROM platform.migrations
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
		INSERT INTO platform.migrations (namespace, name, up_sql, down_sql, status, applied_at)
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
		INSERT INTO platform.migration_execution_logs (migration_id, action, status, duration_ms, error_message, executed_at)
		SELECT id, $2, $3, $4, $5, NOW()
		FROM platform.migrations
		WHERE namespace = 'filesystem' AND name = $1
	`, migrationName, action, status, duration.Milliseconds(), errMsg)
	if err != nil {
		log.Warn().Err(err).Str("migration", migrationName).Msg("Failed to log migration execution")
	}
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

	adminConnStr := fmt.Sprintf(
		"postgres://%s:%s@%s:%d/%s?sslmode=%s",
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
		err := adminConn.QueryRow(
			ctx,
			"SELECT EXISTS(SELECT FROM pg_catalog.pg_roles WHERE rolname = $1)",
			role,
		).Scan(&exists)
		if err != nil {
			log.Warn().Err(err).Str("role", role).Msg("Failed to check if role exists")
			continue
		}

		if exists {
			// Use quoteIdentifier to prevent SQL injection (defense in depth)
			// Both role and user are quoted as PostgreSQL identifiers
			query := fmt.Sprintf("GRANT %s TO %s", quoteIdentifier(role), quoteIdentifier(c.config.User))
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
	c.poolMu.RLock()
	defer c.poolMu.RUnlock()
	if c.pool == nil {
		return nil, fmt.Errorf("database connection closed")
	}
	return c.pool.Begin(ctx)
}

// Query executes a query that returns rows
func (c *Connection) Query(ctx context.Context, sql string, args ...interface{}) (pgx.Rows, error) {
	c.poolMu.RLock()
	if c.pool == nil {
		c.poolMu.RUnlock()
		return nil, fmt.Errorf("database connection closed")
	}
	start := time.Now()
	rows, err := c.pool.Query(ctx, sql, args...)
	c.poolMu.RUnlock()
	duration := time.Since(start)

	// Record metrics
	if c.metrics != nil {
		operation := ExtractOperation(sql)
		table := ExtractTableName(sql)
		c.metrics.RecordDBQuery(operation, table, duration, err)
	}

	// Log slow queries
	c.logSlowQuery(ctx, sql, duration, "query")

	return rows, err
}

// QueryRow executes a query that returns a single row
func (c *Connection) QueryRow(ctx context.Context, sql string, args ...interface{}) pgx.Row {
	c.poolMu.RLock()
	defer c.poolMu.RUnlock()
	if c.pool == nil {
		return errRow{fmt.Errorf("database connection closed")}
	}
	start := time.Now()
	row := c.pool.QueryRow(ctx, sql, args...)
	duration := time.Since(start)

	// Record metrics
	if c.metrics != nil {
		operation := ExtractOperation(sql)
		table := ExtractTableName(sql)
		c.metrics.RecordDBQuery(operation, table, duration, nil)
	}

	// Log slow queries
	c.logSlowQuery(ctx, sql, duration, "query_row")

	return row
}

// Exec executes a query that doesn't return rows
func (c *Connection) Exec(ctx context.Context, sql string, args ...interface{}) (pgconn.CommandTag, error) {
	c.poolMu.RLock()
	if c.pool == nil {
		c.poolMu.RUnlock()
		return pgconn.CommandTag{}, fmt.Errorf("database connection closed")
	}
	start := time.Now()
	tag, err := c.pool.Exec(ctx, sql, args...)
	c.poolMu.RUnlock()
	duration := time.Since(start)

	// Record metrics
	if c.metrics != nil {
		operation := ExtractOperation(sql)
		table := ExtractTableName(sql)
		c.metrics.RecordDBQuery(operation, table, duration, err)
	}

	// Log slow queries
	c.logSlowQuery(ctx, sql, duration, "exec")

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
	c.poolMu.RLock()
	defer c.poolMu.RUnlock()
	if c.pool == nil {
		return nil
	}
	return c.pool.Stat()
}

// WrapWithServiceRole wraps a database operation with service_role context
// Used for privileged operations like auth, admin tasks, and webhooks
func WrapWithServiceRole(ctx context.Context, conn *Connection, fn func(tx pgx.Tx) error) error {
	// Start transaction
	tx, err := conn.Pool().Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	// SET LOCAL ROLE service_role - bypasses RLS for privileged operations
	// This provides the same security model as separate admin connections
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

// WrapWithTenantContext wraps a database operation with tenant context for multi-tenancy.
// This sets the app.current_tenant_id session variable so that RLS policies and triggers
// can enforce tenant isolation. Use this for storage operations on tenant-scoped tables.
func WrapWithTenantContext(ctx context.Context, conn *Connection, tenantID string, fn func(tx pgx.Tx) error) error {
	// Start transaction
	tx, err := conn.Pool().Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	// Set tenant context if provided
	if tenantID != "" {
		_, err = tx.Exec(ctx, "SELECT set_config('app.current_tenant_id', $1, true)", tenantID)
		if err != nil {
			log.Error().Err(err).Str("tenant_id", tenantID).Msg("Failed to set tenant context")
			return fmt.Errorf("failed to set tenant context: %w", err)
		}
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

// WrapWithServiceRoleAndTenant wraps a database operation with both service_role and tenant context.
// This bypasses RLS but still sets tenant_id for new records via the set_tenant_id trigger.
// Use this for privileged operations that still need to associate records with a tenant.
func WrapWithServiceRoleAndTenant(ctx context.Context, conn *Connection, tenantID string, fn func(tx pgx.Tx) error) error {
	// Start transaction
	tx, err := conn.Pool().Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	// SET LOCAL ROLE service_role - bypasses RLS for privileged operations
	_, err = tx.Exec(ctx, "SET LOCAL ROLE service_role")
	if err != nil {
		log.Error().Err(err).Msg("Failed to SET LOCAL ROLE service_role")
		return fmt.Errorf("failed to SET LOCAL ROLE service_role: %w", err)
	}

	// Set tenant context if provided (for triggers that auto-populate tenant_id)
	if tenantID != "" {
		_, err = tx.Exec(ctx, "SELECT set_config('app.current_tenant_id', $1, true)", tenantID)
		if err != nil {
			log.Error().Err(err).Str("tenant_id", tenantID).Msg("Failed to set tenant context")
			return fmt.Errorf("failed to set tenant context: %w", err)
		}
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

// WrapWithTenantAwareRole wraps a database operation with the appropriate role
// based on tenant context. When a tenant context is active, it uses tenant_service
// (NOBYPASSRLS) so RLS policies enforce tenant isolation. When no tenant context,
// it uses service_role (BYPASSRLS) for full instance-admin access.
func WrapWithTenantAwareRole(ctx context.Context, conn *Connection, tenantID string, fn func(tx pgx.Tx) error) error {
	tx, err := conn.Pool().Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if tenantID != "" {
		// Tenant context active: use tenant_service (respects RLS)
		_, err = tx.Exec(ctx, "SET LOCAL ROLE tenant_service")
		if err != nil {
			return fmt.Errorf("failed to SET LOCAL ROLE tenant_service: %w", err)
		}
		_, err = tx.Exec(ctx, "SELECT set_config('app.current_tenant_id', $1, true)", tenantID)
		if err != nil {
			return fmt.Errorf("failed to set tenant context: %w", err)
		}
	} else {
		// No tenant: use service_role (bypasses RLS for instance admin)
		_, err = tx.Exec(ctx, "SET LOCAL ROLE service_role")
		if err != nil {
			return fmt.Errorf("failed to SET LOCAL ROLE service_role: %w", err)
		}
	}

	if err := fn(tx); err != nil {
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}
	return nil
}

// ExecuteWithAdminRole executes a database operation using admin credentials
// Used for migrations that require DDL privileges (CREATE TABLE, ALTER, etc.)
// Creates a temporary admin connection that is closed after execution
func (c *Connection) ExecuteWithAdminRole(ctx context.Context, fn func(tx pgx.Tx) error) error {
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

	// Execute the wrapped function with the transaction
	if err := fn(tx); err != nil {
		return err
	}

	// Commit transaction
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	log.Debug().Msg("Migration executed successfully with admin privileges")
	return nil
}

// ExecuteWithAdminRoleForDB executes a function with admin privileges against
// a specific database (for tenant DDL operations). It replaces the database name
// in the admin connection string with the provided dbName.
func (c *Connection) ExecuteWithAdminRoleForDB(ctx context.Context, dbName string, fn func(tx pgx.Tx) error) error {
	adminConnStr := c.config.AdminConnectionString()

	adminUser := c.config.AdminUser
	if adminUser == "" {
		adminUser = c.config.User
	}

	// Replace database name in connection string
	u, err := url.Parse(adminConnStr)
	if err != nil {
		return fmt.Errorf("failed to parse admin connection string: %w", err)
	}
	u.Path = dbName
	adminConnStrForDB := u.String()

	log.Info().
		Str("admin_user", adminUser).
		Str("database", dbName).
		Msg("Connecting as admin user for tenant DDL")

	adminConn, err := pgx.Connect(ctx, adminConnStrForDB)
	if err != nil {
		log.Error().Err(err).Str("admin_user", adminUser).Str("database", dbName).Msg("Failed to connect as admin user for tenant DDL")
		return fmt.Errorf("failed to connect as admin to database %s: %w", dbName, err)
	}
	defer func() { _ = adminConn.Close(ctx) }()

	tx, err := adminConn.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if err := fn(tx); err != nil {
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	log.Debug().Str("database", dbName).Msg("Tenant DDL executed successfully with admin privileges")
	return nil
}

// TenantOrNil converts an empty tenant string to nil for UUID column compatibility.
// PostgreSQL UUID columns accept NULL but reject empty strings, so this helper
// is used when passing tenant IDs as query parameters.
func TenantOrNil(tenantID string) interface{} {
	if tenantID == "" {
		return nil
	}
	return tenantID
}

// TenantAware provides a reusable embedded struct for tenant-scoped database operations.
// Storage types embed this to get the WithTenant helper method.
type TenantAware struct {
	DB *Connection
}

// WithTenant wraps a database operation with tenant-aware role selection.
// When a tenant context is active, uses tenant_service (respects RLS).
// When no tenant context, uses service_role (bypasses RLS).
func (t *TenantAware) WithTenant(ctx context.Context, fn func(tx pgx.Tx) error) error {
	tenantID := TenantFromContext(ctx)
	return WrapWithTenantAwareRole(ctx, t.DB, tenantID, fn)
}

// validateMigrationSQL validates SQL syntax for user-provided migration files
// This validates that the SQL is valid PostgreSQL syntax without executing it
func (c *Connection) validateMigrationSQL(sql, migrationName string) error {
	// Parse the SQL using pg_query to validate syntax
	tree, err := pg_query.Parse(sql)
	if err != nil {
		return fmt.Errorf("SQL syntax error: %w", err)
	}

	// Log the migration SQL for audit trail (security feature)
	// This helps track what schema changes were applied
	log.Info().
		Str("migration", migrationName).
		Str("sql_preview", truncateQuery(sql, 200)).
		Int("statement_count", len(tree.Stmts)).
		Msg("Validated user migration SQL")

	return nil
}
