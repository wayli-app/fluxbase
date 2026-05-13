package database

import (
	"context"
	"fmt"
	"net/url"
	"regexp"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
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
	metrics            atomic.Pointer[observability.Metrics]
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
	if c.slowQueryTracker != nil {
		c.slowQueryTracker.stop()
	}
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
	if m := c.metrics.Load(); m != nil {
		operation := ExtractOperation(sql)
		table := ExtractTableName(sql)
		m.RecordDBQuery(operation, table, duration, err)
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
	if m := c.metrics.Load(); m != nil {
		operation := ExtractOperation(sql)
		table := ExtractTableName(sql)
		m.RecordDBQuery(operation, table, duration, nil)
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
	if m := c.metrics.Load(); m != nil {
		operation := ExtractOperation(sql)
		table := ExtractTableName(sql)
		m.RecordDBQuery(operation, table, duration, err)
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
