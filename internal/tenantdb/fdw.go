package tenantdb

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog/log"
)

// FDWConfig holds the connection details for the main database,
// used to set up postgres_fdw foreign tables in tenant databases.
type FDWConfig struct {
	Host     string // Main DB host
	Port     string // Main DB port
	DBName   string // Main DB name
	User     string // Main DB user (needs SELECT on auth.*)
	Password string // Main DB password
}

// FDWRoleCredentials holds the credentials for the per-tenant FDW role.
type FDWRoleCredentials struct {
	RoleName string
	Password string
}

// fdwSchemas lists schemas whose tables should be imported via FDW.
// These are shared infrastructure tables living in the main database,
// accessed by tenant databases through foreign data wrappers.
var fdwSchemas = []string{
	"platform", "auth", "storage", "jobs", "functions", "realtime",
	"ai", "rpc", "branching", "logging", "mcp", "app",
}

// fdwExcludeTables lists tables that should NOT be imported via FDW.
// These are true singletons or instance-level infrastructure with no tenant_id.
var fdwExcludeTables = map[string][]string{
	"platform": {
		"schema_migrations", "migrations", "migration_execution_logs",
		"bootstrap_state", "fluxbase_migrations", "declarative_state",
		"rate_limits", "idempotency_keys",
	},
	"logging": {"execution_logs_migration_status"},
	"auth": {
		"captcha_challenges", "captcha_trust_tokens", "user_trust_signals",
		"emergency_revocation", "rls_audit_log", "token_blacklist",
		"webhook_monitored_tables", "saml_assertion_ids",
		"nonces", "oauth_states",
	},
}

// fdwImportRetryDelays defines the backoff delays for retrying transient FDW
// connection errors (SQLSTATE 08001) during schema import.
var fdwImportRetryDelays = []time.Duration{2 * time.Second, 4 * time.Second, 8 * time.Second}

// isFDWConnectionError returns true if the error is a transient FDW connection
// failure (SQLSTATE 08001 — sqlclient_unable_to_establish_sqlconnection).
func isFDWConnectionError(err error) bool {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code == "08001"
	}
	return false
}

// ParseFDWConfig extracts FDW connection details from a database URL.
func ParseFDWConfig(dbURL string) (FDWConfig, error) {
	u, err := url.Parse(dbURL)
	if err != nil {
		return FDWConfig{}, fmt.Errorf("failed to parse database URL: %w", err)
	}

	host := u.Hostname()
	port := u.Port()
	if port == "" {
		port = "5432"
	}
	dbName := strings.TrimPrefix(u.Path, "/")
	user := u.User.Username()
	password, _ := u.User.Password()

	if host == "" || dbName == "" || user == "" {
		return FDWConfig{}, fmt.Errorf("incomplete database URL: need host, dbname, and user")
	}

	return FDWConfig{
		Host:     host,
		Port:     port,
		DBName:   dbName,
		User:     user,
		Password: password,
	}, nil
}

const fdwServerName = "main_server"

// CreateFDWRole creates a dedicated PostgreSQL role on the main database
// for a specific tenant. The role has NOBYPASSRLS so RLS policies are
// enforced, and has a default app.current_tenant_id set so queries
// through FDW are automatically filtered by tenant.
func CreateFDWRole(ctx context.Context, adminPool *pgxpool.Pool, tenantID string) (FDWRoleCredentials, error) {
	// Use first 8 chars of UUID for readable role name
	suffix := tenantID
	if len(suffix) > 8 {
		suffix = suffix[:8]
	}
	roleName := fmt.Sprintf("fdw_tenant_%s", suffix)

	// Generate a random password
	keyBytes := make([]byte, 24)
	if _, err := rand.Read(keyBytes); err != nil {
		return FDWRoleCredentials{}, fmt.Errorf("failed to generate FDW role password: %w", err)
	}
	password := base64.URLEncoding.EncodeToString(keyBytes)

	// Create role with NOBYPASSRLS so RLS policies are enforced
	_, err := adminPool.Exec(ctx, fmt.Sprintf(
		`DO $$ BEGIN
			IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = '%s') THEN
				CREATE ROLE %s NOBYPASSRLS LOGIN PASSWORD '%s';
			END IF;
		END $$`,
		escapeSQLString(roleName),
		quoteIdent(roleName),
		escapeSQLString(password),
	))
	if err != nil {
		return FDWRoleCredentials{}, fmt.Errorf("failed to create FDW role: %w", err)
	}

	// Set default tenant context so RLS policies filter automatically
	_, err = adminPool.Exec(ctx, fmt.Sprintf(
		`ALTER ROLE %s SET app.current_tenant_id = '%s'`,
		quoteIdent(roleName),
		escapeSQLString(tenantID),
	))
	if err != nil {
		return FDWRoleCredentials{}, fmt.Errorf("failed to set FDW role tenant context: %w", err)
	}

	// Grant usage on all FDW schemas
	for _, schema := range fdwSchemas {
		_, err := adminPool.Exec(ctx, fmt.Sprintf(
			`GRANT USAGE ON SCHEMA %s TO %s`,
			quoteIdent(schema), quoteIdent(roleName),
		))
		if err != nil {
			log.Warn().Err(err).Str("schema", schema).Msg("Failed to grant schema usage to FDW role")
		}
	}

	// Grant ALL on all tables in FDW schemas (read + write through FDW)
	for _, schema := range fdwSchemas {
		_, err := adminPool.Exec(ctx, fmt.Sprintf(
			`GRANT ALL ON ALL TABLES IN SCHEMA %s TO %s`,
			quoteIdent(schema), quoteIdent(roleName),
		))
		if err != nil {
			log.Warn().Err(err).Str("schema", schema).Msg("Failed to grant table permissions to FDW role")
		}
	}

	// Grant usage on sequences (needed for INSERT with serial columns)
	for _, schema := range fdwSchemas {
		_, _ = adminPool.Exec(ctx, fmt.Sprintf(
			`GRANT USAGE ON ALL SEQUENCES IN SCHEMA %s TO %s`,
			quoteIdent(schema), quoteIdent(roleName),
		))
	}

	log.Info().Str("role", roleName).Str("tenant_id", tenantID).Msg("Created FDW role for tenant")

	return FDWRoleCredentials{
		RoleName: roleName,
		Password: password,
	}, nil
}

// DropFDWRole removes the per-tenant FDW role from the main database.
func DropFDWRole(ctx context.Context, adminPool *pgxpool.Pool, tenantID string) {
	suffix := tenantID
	if len(suffix) > 8 {
		suffix = suffix[:8]
	}
	roleName := fmt.Sprintf("fdw_tenant_%s", suffix)

	_, err := adminPool.Exec(ctx, fmt.Sprintf(`DROP ROLE IF EXISTS %s`, quoteIdent(roleName)))
	if err != nil {
		log.Warn().Err(err).Str("role", roleName).Msg("Failed to drop FDW role")
	}
}

// GetFDWRoleForTenant retrieves the FDW role credentials for a tenant by reading
// the user mapping from the tenant database. The role name is deterministic based
// on the tenant ID, and the password is extracted from the existing user mapping.
func GetFDWRoleForTenant(ctx context.Context, tenantPool *pgxpool.Pool, tenantID string) (FDWRoleCredentials, error) {
	suffix := tenantID
	if len(suffix) > 8 {
		suffix = suffix[:8]
	}
	roleName := fmt.Sprintf("fdw_tenant_%s", suffix)

	var umOptions string
	err := tenantPool.QueryRow(ctx, `
		SELECT umoptions::text FROM pg_user_mapping
		WHERE umserver = (SELECT oid FROM pg_foreign_server WHERE srvname = $1)
		LIMIT 1
	`, fdwServerName).Scan(&umOptions)
	if err != nil {
		return FDWRoleCredentials{}, fmt.Errorf("failed to read FDW user mapping: %w", err)
	}

	password := extractOptionValue(umOptions, "password")
	if password == "" {
		return FDWRoleCredentials{}, fmt.Errorf("FDW user mapping has no password for tenant %s", tenantID)
	}

	return FDWRoleCredentials{
		RoleName: roleName,
		Password: password,
	}, nil
}

// extractOptionValue extracts a value from pg_user_mapping options string format.
// The umoptions column returns strings like {"user=fdw_tenant_xxx","password=yyy"}
func extractOptionValue(options, key string) string {
	prefix := key + "="
	for _, part := range strings.Split(strings.Trim(options, "{}"), ",") {
		part = strings.Trim(part, "\"")
		if strings.HasPrefix(part, prefix) {
			return strings.TrimPrefix(part, prefix)
		}
	}
	return ""
}

// SetupFDW configures postgres_fdw in a tenant database so it can access
// tables from the main database. It creates a foreign server, user mapping
// (using the per-tenant FDW role for RLS), and imports foreign tables from
// all non-public schemas.
//
// Local tables that would conflict with FDW imports are dropped first.
// Functions, types, and other non-table objects remain local.
func SetupFDW(ctx context.Context, tenantPool *pgxpool.Pool, cfg FDWConfig, tables []string) error {
	if tenantPool == nil {
		return fmt.Errorf("tenant pool is nil")
	}
	if cfg.Host == "" || cfg.DBName == "" {
		return fmt.Errorf("FDW config incomplete: host and dbname required")
	}

	// Legacy path: if specific tables are requested, use old auth-only import
	if len(tables) > 0 {
		return setupFDWLegacy(ctx, tenantPool, cfg, tables)
	}

	return setupFDWAllSchemas(ctx, tenantPool, cfg)
}

// setupFDWAllSchemas imports all tables from all FDW schemas.
func setupFDWAllSchemas(ctx context.Context, tenantPool *pgxpool.Pool, cfg FDWConfig) error {
	// 1. Ensure postgres_fdw extension exists
	_, err := tenantPool.Exec(ctx, `CREATE EXTENSION IF NOT EXISTS postgres_fdw`)
	if err != nil {
		return fmt.Errorf("failed to create postgres_fdw extension: %w", err)
	}

	// 2. Create or update foreign server pointing at main database.
	// Tenant databases always live on the same PostgreSQL instance as the
	// main database, so the FDW connection is a loopback. Use "localhost"
	// instead of the external hostname (which may be a Docker service name
	// that doesn't resolve from within the PostgreSQL process).
	fdwHost := "localhost"
	_, err = tenantPool.Exec(ctx, fmt.Sprintf(
		`CREATE SERVER IF NOT EXISTS %s FOREIGN DATA WRAPPER postgres_fdw
		  OPTIONS (host '%s', port '%s', dbname '%s')`,
		quoteIdent(fdwServerName), fdwHost, cfg.Port, cfg.DBName,
	))
	if err != nil {
		return fmt.Errorf("failed to create foreign server: %w", err)
	}

	// Ensure options are current (CREATE SERVER IF NOT EXISTS skips if server
	// already exists, leaving stale options from previous runs).
	_, _ = tenantPool.Exec(ctx, fmt.Sprintf(
		`ALTER SERVER %s OPTIONS (SET host '%s', SET port '%s', SET dbname '%s')`,
		quoteIdent(fdwServerName), fdwHost, cfg.Port, cfg.DBName,
	))

	// 3. Create user mapping using admin credentials
	// This will be replaced by the per-tenant role mapping in CreateTenantDatabase
	userMappingSQL := fmt.Sprintf(
		`CREATE USER MAPPING IF NOT EXISTS FOR CURRENT_USER SERVER %s
		  OPTIONS (user '%s'`,
		quoteIdent(fdwServerName), cfg.User,
	)
	if cfg.Password != "" {
		userMappingSQL += fmt.Sprintf(`, password '%s'`, escapeSQLString(cfg.Password))
	}
	userMappingSQL += ")"
	_, err = tenantPool.Exec(ctx, userMappingSQL)
	if err != nil {
		return fmt.Errorf("failed to create user mapping: %w", err)
	}

	// 4. Import tables from each FDW schema
	var failedSchemas []string
	for _, schema := range fdwSchemas {
		if err := importSchemaFDW(ctx, tenantPool, schema); err != nil {
			log.Warn().Err(err).Str("schema", schema).Msg("Failed to import schema via FDW")
			failedSchemas = append(failedSchemas, schema)
		}
	}

	if len(failedSchemas) == 0 {
		log.Info().
			Strs("schemas", fdwSchemas).
			Str("main_db", cfg.DBName).
			Msg("Set up FDW for tenant database (all schemas)")
	} else {
		log.Warn().
			Strs("succeeded", filterSlice(fdwSchemas, failedSchemas)).
			Strs("failed", failedSchemas).
			Str("main_db", cfg.DBName).
			Msg("Set up FDW for tenant database with partial failures")
	}

	return nil
}

// importSchemaFDW drops local and foreign tables and imports foreign tables
// for a single schema. Transient FDW connection errors are retried with backoff.
func importSchemaFDW(ctx context.Context, tenantPool *pgxpool.Pool, schema string) error {
	excluded := fdwExcludeTables[schema]

	excludeSet := make(map[string]bool, len(excluded))
	for _, ex := range excluded {
		excludeSet[ex] = true
	}

	rows, err := tenantPool.Query(ctx, `
		SELECT tablename FROM pg_tables WHERE schemaname = $1
	`, schema)
	if err != nil {
		return fmt.Errorf("failed to list tables in schema %s: %w", schema, err)
	}

	var tablesToDrop []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			rows.Close()
			return fmt.Errorf("failed to scan table name: %w", err)
		}
		if !excludeSet[name] {
			tablesToDrop = append(tablesToDrop, name)
		}
	}
	rows.Close()

	for _, table := range tablesToDrop {
		_, err := tenantPool.Exec(ctx, fmt.Sprintf(
			`DROP TABLE IF EXISTS %s.%s CASCADE`,
			quoteIdent(schema), quoteIdent(table),
		))
		if err != nil {
			log.Warn().Err(err).Str("schema", schema).Str("table", table).
				Msg("Failed to drop local table before FDW import")
		}
	}

	foreignRows, err := tenantPool.Query(ctx, `
		SELECT ft.relname FROM pg_foreign_table f
		JOIN pg_class ft ON f.ftrelid = ft.oid
		JOIN pg_namespace n ON ft.relnamespace = n.oid
		WHERE n.nspname = $1
	`, schema)
	if err != nil {
		return fmt.Errorf("failed to list foreign tables in schema %s: %w", schema, err)
	}

	var foreignTables []string
	for foreignRows.Next() {
		var name string
		if err := foreignRows.Scan(&name); err != nil {
			foreignRows.Close()
			return fmt.Errorf("failed to scan foreign table name: %w", err)
		}
		if !excludeSet[name] {
			foreignTables = append(foreignTables, name)
		}
	}
	foreignRows.Close()

	for _, table := range foreignTables {
		_, _ = tenantPool.Exec(ctx, fmt.Sprintf(
			`DROP FOREIGN TABLE IF EXISTS %s.%s CASCADE`,
			quoteIdent(schema), quoteIdent(table),
		))
	}

	for _, ex := range excluded {
		_, _ = tenantPool.Exec(ctx, fmt.Sprintf(
			`DROP FOREIGN TABLE IF EXISTS %s.%s CASCADE`,
			quoteIdent(schema), quoteIdent(ex),
		))
	}

	importSQL := fmt.Sprintf(
		`IMPORT FOREIGN SCHEMA %s FROM SERVER %s INTO %s`,
		quoteIdent(schema), quoteIdent(fdwServerName), quoteIdent(schema),
	)

	if len(excluded) > 0 {
		excludedList := make([]string, len(excluded))
		for i, t := range excluded {
			excludedList[i] = quoteIdent(t)
		}
		importSQL = fmt.Sprintf(
			`IMPORT FOREIGN SCHEMA %s EXCEPT (%s) FROM SERVER %s INTO %s`,
			quoteIdent(schema), strings.Join(excludedList, ", "),
			quoteIdent(fdwServerName), quoteIdent(schema),
		)
	}

	var lastErr error
	attempts := 1 + len(fdwImportRetryDelays)
	for attempt := range attempts {
		_, err = tenantPool.Exec(ctx, importSQL)
		if err == nil {
			break
		}
		lastErr = err
		if isFDWConnectionError(err) && attempt < len(fdwImportRetryDelays) {
			delay := fdwImportRetryDelays[attempt]
			log.Warn().Err(err).Str("schema", schema).
				Dur("retry_after", delay).Int("attempt", attempt+1).
				Msg("FDW connection error, retrying schema import")
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(delay):
			}
			continue
		}
		break
	}
	if lastErr != nil {
		return fmt.Errorf("failed to import foreign schema %s: %w", schema, lastErr)
	}

	for _, role := range []string{"tenant_service", "service_role"} {
		_, err = tenantPool.Exec(ctx, fmt.Sprintf(
			`GRANT ALL ON ALL TABLES IN SCHEMA %s TO %s`,
			quoteIdent(schema), quoteIdent(role),
		))
		if err != nil {
			log.Warn().Err(err).Str("schema", schema).Str("role", role).
				Msg("Failed to grant permissions on imported foreign tables")
		}
	}

	importedCount := len(tablesToDrop) + len(foreignTables)
	if importedCount > 0 {
		log.Debug().Str("schema", schema).Int("tables", importedCount).
			Msg("Imported schema tables via FDW")
	}

	return nil
}

// CreateFDWUserMapping creates a user mapping for the app user using
// the per-tenant FDW role credentials. This overrides the default
// admin user mapping with the tenant-specific role for RLS enforcement.
// Also creates mappings for tenant_service and service_role so that
// SET LOCAL ROLE can be used with FDW queries.
func CreateFDWUserMapping(ctx context.Context, tenantPool *pgxpool.Pool, appUser string, fdwRole FDWRoleCredentials) error {
	for _, role := range []string{appUser, "tenant_service", "service_role"} {
		_, _ = tenantPool.Exec(ctx, fmt.Sprintf(
			`DROP USER MAPPING IF EXISTS FOR %s SERVER %s`,
			quoteIdent(role), quoteIdent(fdwServerName),
		))

		_, err := tenantPool.Exec(ctx, fmt.Sprintf(
			`CREATE USER MAPPING FOR %s SERVER %s OPTIONS (user '%s', password '%s')`,
			quoteIdent(role), quoteIdent(fdwServerName),
			escapeSQLString(fdwRole.RoleName), escapeSQLString(fdwRole.Password),
		))
		if err != nil {
			return fmt.Errorf("failed to create FDW user mapping for %s: %w", role, err)
		}
	}

	log.Debug().Str("app_user", appUser).Str("fdw_role", fdwRole.RoleName).
		Msg("Created FDW user mappings for app user, tenant_service, and service_role")

	return nil
}

// setupFDWLegacy implements the original auth-only FDW import for backward compatibility.
func setupFDWLegacy(ctx context.Context, tenantPool *pgxpool.Pool, cfg FDWConfig, tablesToImport []string) error {
	// 1. Create foreign server
	_, err := tenantPool.Exec(ctx, fmt.Sprintf(
		`CREATE SERVER IF NOT EXISTS %s FOREIGN DATA WRAPPER postgres_fdw
		  OPTIONS (host 'localhost', port '%s', dbname '%s')`,
		quoteIdent(fdwServerName), cfg.Port, cfg.DBName,
	))
	if err != nil {
		return fmt.Errorf("failed to create foreign server: %w", err)
	}

	// 2. Create user mapping
	userMappingSQL := fmt.Sprintf(
		`CREATE USER MAPPING IF NOT EXISTS FOR CURRENT_USER SERVER %s
		  OPTIONS (user '%s'`,
		quoteIdent(fdwServerName), cfg.User,
	)
	if cfg.Password != "" {
		userMappingSQL += fmt.Sprintf(`, password '%s'`, escapeSQLString(cfg.Password))
	}
	userMappingSQL += ")"
	_, err = tenantPool.Exec(ctx, userMappingSQL)
	if err != nil {
		return fmt.Errorf("failed to create user mapping: %w", err)
	}

	// 3. Drop local tables that will be replaced by foreign tables
	for _, table := range tablesToImport {
		_, err := tenantPool.Exec(ctx, fmt.Sprintf(
			`DROP TABLE IF EXISTS auth.%s CASCADE`,
			quoteIdent(table),
		))
		if err != nil {
			log.Warn().Err(err).Str("table", table).Msg("Failed to drop local table before FDW import")
		}
		// Also drop existing foreign tables so IMPORT FOREIGN SCHEMA is idempotent
		_, err = tenantPool.Exec(ctx, fmt.Sprintf(
			`DROP FOREIGN TABLE IF EXISTS auth.%s CASCADE`,
			quoteIdent(table),
		))
		if err != nil {
			log.Warn().Err(err).Str("table", table).Msg("Failed to drop foreign table before FDW import")
		}
	}

	// 4. Import foreign tables into auth schema
	tableList := make([]string, len(tablesToImport))
	for i, t := range tablesToImport {
		tableList[i] = quoteIdent(t)
	}

	_, err = tenantPool.Exec(ctx, fmt.Sprintf(
		`IMPORT FOREIGN SCHEMA auth LIMIT TO (%s) FROM SERVER %s INTO auth`,
		strings.Join(tableList, ", "),
		quoteIdent(fdwServerName),
	))
	if err != nil {
		return fmt.Errorf("failed to import foreign schema: %w", err)
	}

	for _, role := range []string{"tenant_service", "service_role"} {
		_, err = tenantPool.Exec(ctx, fmt.Sprintf(
			`GRANT ALL ON ALL TABLES IN SCHEMA auth TO %s`,
			quoteIdent(role),
		))
		if err != nil {
			log.Warn().Err(err).Str("role", role).
				Msg("Failed to grant permissions on imported foreign tables (legacy)")
		}
	}

	log.Info().
		Str("tables", strings.Join(tablesToImport, ", ")).
		Str("main_db", cfg.DBName).
		Msg("Set up FDW for tenant database (legacy auth-only)")

	return nil
}

// TeardownFDW removes all FDW artifacts from a tenant database.
func TeardownFDW(ctx context.Context, tenantPool *pgxpool.Pool) error {
	if tenantPool == nil {
		return fmt.Errorf("tenant pool is nil")
	}

	// Drop user mapping
	_, _ = tenantPool.Exec(ctx, fmt.Sprintf(
		`DROP USER MAPPING IF EXISTS FOR CURRENT_USER SERVER %s`,
		quoteIdent(fdwServerName),
	))

	// Drop foreign server (cascades to foreign tables)
	_, err := tenantPool.Exec(ctx, fmt.Sprintf(
		`DROP SERVER IF EXISTS %s CASCADE`,
		quoteIdent(fdwServerName),
	))
	if err != nil {
		return fmt.Errorf("failed to drop foreign server: %w", err)
	}

	log.Info().Msg("Tore down FDW from tenant database")
	return nil
}

// quoteIdent quotes a PostgreSQL identifier.
func quoteIdent(name string) string {
	return `"` + strings.ReplaceAll(name, `"`, `""`) + `"`
}

// escapeSQLString escapes a string for use in SQL single-quoted literals.
func escapeSQLString(s string) string {
	return strings.ReplaceAll(s, "'", "''")
}

// filterSlice returns elements from slice that are not in exclude.
func filterSlice(slice, exclude []string) []string {
	excludeSet := make(map[string]bool, len(exclude))
	for _, s := range exclude {
		excludeSet[s] = true
	}
	var result []string
	for _, s := range slice {
		if !excludeSet[s] {
			result = append(result, s)
		}
	}
	return result
}
