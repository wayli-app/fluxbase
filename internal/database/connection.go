package database

import (
	"context"
	"embed"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/fluxbase-eu/fluxbase/internal/config"
	"github.com/fluxbase-eu/fluxbase/internal/observability"
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

// quoteIdentifier safely quotes a PostgreSQL identifier to prevent SQL injection.
// It wraps the identifier in double quotes and escapes any embedded double quotes.
func quoteIdentifier(identifier string) string {
	return `"` + strings.ReplaceAll(identifier, `"`, `""`) + `"`
}

// Connection represents a database connection pool
type Connection struct {
	pool      *pgxpool.Pool
	config    *config.DatabaseConfig
	inspector *SchemaInspector
	metrics   *observability.Metrics
}

// SetMetrics sets the metrics instance for recording database metrics
func (c *Connection) SetMetrics(m *observability.Metrics) {
	c.metrics = m
}

// extractTableName attempts to extract the table name from a SQL query
// Returns "unknown" if the table cannot be determined
func extractTableName(sql string) string {
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

// extractOperation extracts the SQL operation type from a query
func extractOperation(sql string) string {
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

	// BeforeAcquire is called before a connection is acquired from the pool.
	// Return false to discard the connection and try another one.
	// This prevents returning stale/closed connections that would cause "conn closed" errors.
	poolConfig.BeforeAcquire = func(ctx context.Context, conn *pgx.Conn) bool {
		// Check if connection is still alive with a simple ping
		// Use a short timeout to avoid blocking on dead connections
		pingCtx, cancel := context.WithTimeout(ctx, 1*time.Second)
		defer cancel()
		if err := conn.Ping(pingCtx); err != nil {
			log.Debug().Err(err).Msg("Discarding unhealthy connection from pool")
			return false // Discard this connection
		}
		return true // Connection is healthy, use it
	}

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
			log.Debug().Uint32("oid", vectorOID).Msg("Registered pgvector type")
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

	// Check if database has a version that doesn't exist in our embedded migration files
	// This can happen when switching branches or after migrations are renumbered
	// We must fix this BEFORE creating the migrate instance, as golang-migrate validates files on Force()
	highestAvailable := c.findHighestMigrationVersion()
	log.Debug().Int("highest_available", highestAvailable).Msg("Found highest available migration version")

	if highestAvailable > 0 {
		var recordedVersion int64
		var dirty bool
		err = adminConn.QueryRow(ctx, `SELECT version, dirty FROM "migrations"."fluxbase" LIMIT 1`).Scan(&recordedVersion, &dirty)
		if err == nil {
			fileExists := c.migrationFileExists(int(recordedVersion))
			log.Debug().
				Int64("recorded_version", recordedVersion).
				Bool("dirty", dirty).
				Bool("file_exists", fileExists).
				Msg("Checking migration state")

			needsReset := false
			reason := ""

			if recordedVersion > int64(highestAvailable) {
				needsReset = true
				reason = "version higher than available migrations"
			} else if !fileExists {
				needsReset = true
				reason = "migration file does not exist"
			} else if dirty {
				// If dirty but file exists, just clear the dirty flag
				// This happens when a previous migration was interrupted
				log.Warn().
					Int64("recorded_version", recordedVersion).
					Bool("was_dirty", dirty).
					Msg("Clearing dirty flag for existing migration version")

				_, err = adminConn.Exec(ctx, `UPDATE "migrations"."fluxbase" SET dirty = false WHERE version = $1`, recordedVersion)
				if err != nil {
					return fmt.Errorf("failed to clear dirty flag: %w", err)
				}
			}

			if needsReset {
				log.Warn().
					Int64("recorded_version", recordedVersion).
					Int("highest_available", highestAvailable).
					Bool("was_dirty", dirty).
					Str("reason", reason).
					Msg("Resetting database migration version to highest available")

				_, err = adminConn.Exec(ctx, `UPDATE "migrations"."fluxbase" SET version = $1, dirty = false`, highestAvailable)
				if err != nil {
					return fmt.Errorf("failed to reset migration version: %w", err)
				}
			}
		}
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
		// Check if error is due to missing migration file
		if strings.Contains(err.Error(), "file does not exist") || strings.Contains(err.Error(), "no migration found") {
			// Find the highest available migration version
			highestVersion := c.findHighestMigrationVersion()
			if highestVersion > 0 && version > uint(highestVersion) {
				log.Warn().
					Str("source", source).
					Uint("recorded_version", version).
					Int("highest_available", highestVersion).
					Msg("Database version higher than available migrations, resetting to highest available")

				// Force to highest available version
				if forceErr := m.Force(highestVersion); forceErr != nil {
					return fmt.Errorf("failed to force migration version to %d: %w", highestVersion, forceErr)
				}

				// Try running migrations again
				if retryErr := m.Up(); retryErr != nil && retryErr != migrate.ErrNoChange {
					return fmt.Errorf("failed to run %s migrations after version reset: %w", source, retryErr)
				}
				log.Info().Str("source", source).Int("version", highestVersion).Msg("Migrations recovered after version reset")
				return nil
			}
		}
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

// findHighestMigrationVersion scans embedded migrations to find the highest version number
func (c *Connection) findHighestMigrationVersion() int {
	entries, err := migrationsFS.ReadDir("migrations")
	if err != nil {
		log.Warn().Err(err).Msg("Failed to read embedded migrations directory")
		return 0
	}

	highest := 0
	versionRegex := regexp.MustCompile(`^(\d+)_.*\.up\.sql$`)

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		matches := versionRegex.FindStringSubmatch(entry.Name())
		if len(matches) > 1 {
			var version int
			if _, err := fmt.Sscanf(matches[1], "%d", &version); err == nil {
				if version > highest {
					highest = version
				}
			}
		}
	}

	return highest
}

// migrationFileExists checks if both up and down migration files exist in the embedded filesystem
// golang-migrate requires both files to be present for a version to be valid
func (c *Connection) migrationFileExists(version int) bool {
	entries, err := migrationsFS.ReadDir("migrations")
	if err != nil {
		log.Debug().Err(err).Msg("Failed to read migrations directory")
		return false
	}

	// Try both zero-padded (057_) and non-padded (57_) prefixes
	prefixes := []string{
		fmt.Sprintf("%03d_", version),
		fmt.Sprintf("%d_", version),
	}
	hasUp := false
	hasDown := false

	// Log first few files to see what's embedded
	for i, entry := range entries {
		if i < 5 || i >= len(entries)-5 {
			log.Debug().Str("file", entry.Name()).Int("index", i).Msg("Embedded migration file")
		}
		for _, prefix := range prefixes {
			if strings.HasPrefix(entry.Name(), prefix) {
				log.Debug().Str("file", entry.Name()).Str("prefix", prefix).Msg("Found matching migration file")
				if strings.HasSuffix(entry.Name(), ".up.sql") {
					hasUp = true
				} else if strings.HasSuffix(entry.Name(), ".down.sql") {
					hasDown = true
				}
				break
			}
		}
		if hasUp && hasDown {
			return true
		}
	}
	log.Debug().Int("version", version).Bool("hasUp", hasUp).Bool("hasDown", hasDown).Msg("Migration file check result")
	return hasUp && hasDown
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
	return c.pool.Begin(ctx)
}

// Query executes a query that returns rows
func (c *Connection) Query(ctx context.Context, sql string, args ...interface{}) (pgx.Rows, error) {
	start := time.Now()
	rows, err := c.pool.Query(ctx, sql, args...)
	duration := time.Since(start)

	// Record metrics
	if c.metrics != nil {
		operation := extractOperation(sql)
		table := extractTableName(sql)
		c.metrics.RecordDBQuery(operation, table, duration, err)
	}

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

	// Record metrics
	if c.metrics != nil {
		operation := extractOperation(sql)
		table := extractTableName(sql)
		c.metrics.RecordDBQuery(operation, table, duration, nil)
	}

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

	// Record metrics
	if c.metrics != nil {
		operation := extractOperation(sql)
		table := extractTableName(sql)
		c.metrics.RecordDBQuery(operation, table, duration, err)
	}

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
