package database

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	pg_query "github.com/pganalyze/pg_query_go/v6"
	"github.com/rs/zerolog/log"
)

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
