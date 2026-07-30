package migrations

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/pgplex/pgparser/nodes"
	"github.com/pgplex/pgparser/parser"
	"github.com/rs/zerolog/log"

	"github.com/nimbleflux/fluxbase/internal/database/bootstrap"
	dbschema "github.com/nimbleflux/fluxbase/internal/database/schema"
)

// AppDeclarativeService manages an application's own schema (e.g. the `public`
// schema of the main/shared database) declaratively using pgschema.
//
// This is the app-developer counterpart to the always-on internal DeclarativeService:
// it is opt-in (gated by database.declarative_app_schema.enabled) and operates on
// schema content synced by the app (e.g. via `fluxbase schema sync`), stored in
// platform.app_schemas. It coexists with imperative user migrations — a given
// (namespace, schema) is owned by one mode, not both.
//
// The engine is the same proven pgschema plan/apply pipeline used for tenant
// schemas (see tenantdb.DeclarativeService.ApplyTenantSchemaFromContent).
type AppDeclarativeService struct {
	pgschemaPath     string
	dbHost           string
	dbPort           int
	dbUser           string // admin user used to apply DDL
	dbPassword       string
	dbName           string
	appUser          string // runtime user for {{APP_USER}} GRANT substitution
	pool             *pgxpool.Pool
	allowDestructive bool
}

// AppSchemaRecord represents a stored app schema in platform.app_schemas.
type AppSchemaRecord struct {
	Namespace         string    `json:"namespace"`
	SchemaName        string    `json:"schema_name"`
	SchemaContent     string    `json:"schema_content,omitempty"`
	SchemaFingerprint string    `json:"schema_fingerprint"`
	Enabled           bool      `json:"enabled"`
	UpdatedAt         time.Time `json:"updated_at"`
}

// AppSchemaStatus is the per-namespace/schema status (pending changes, drift).
type AppSchemaStatus struct {
	Namespace              string    `json:"namespace"`
	SchemaName             string    `json:"schema_name"`
	SchemaFingerprint      string    `json:"schema_fingerprint,omitempty"`
	LastAppliedFingerprint string    `json:"last_applied_fingerprint,omitempty"`
	HasStoredSchema        bool      `json:"has_stored_schema"`
	Enabled                bool      `json:"enabled"`
	UpdatedAt              time.Time `json:"updated_at,omitempty"`
}

// NewAppDeclarativeService creates a new declarative app-schema service.
func NewAppDeclarativeService(pgschemaPath, dbHost string, dbPort int, dbUser, dbPassword, dbName string, allowDestructive bool) *AppDeclarativeService {
	return &AppDeclarativeService{
		pgschemaPath:     pgschemaPath,
		dbHost:           dbHost,
		dbPort:           dbPort,
		dbUser:           dbUser,
		dbPassword:       dbPassword,
		dbName:           dbName,
		allowDestructive: allowDestructive,
	}
}

// SetPool sets the pool used for reading/writing stored schema content and state.
func (s *AppDeclarativeService) SetPool(pool *pgxpool.Pool) {
	s.pool = pool
}

// SetAppUser sets the runtime database user for {{APP_USER}} placeholder substitution.
func (s *AppDeclarativeService) SetAppUser(appUser string) {
	s.appUser = appUser
}

// SetAllowDestructive toggles whether destructive changes are permitted.
func (s *AppDeclarativeService) SetAllowDestructive(allow bool) {
	s.allowDestructive = allow
}

// ensureAppSchemasTable creates platform.app_schemas (and its state table) if absent.
// App schemas are application-owned, so they are NOT part of the embedded internal
// platform.sql; they are created lazily on first use (mirrors tenantdb.tenant_schemas).
func (s *AppDeclarativeService) ensureAppSchemasTable(ctx context.Context) error {
	_, err := s.pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS platform.app_schemas (
			namespace          TEXT NOT NULL,
			schema_name        TEXT NOT NULL DEFAULT 'public',
			schema_content     TEXT NOT NULL,
			schema_fingerprint TEXT NOT NULL,
			ignore_content     TEXT NOT NULL DEFAULT '',
			enabled            BOOLEAN NOT NULL DEFAULT true,
			updated_at         TIMESTAMPTZ NOT NULL DEFAULT now(),
			PRIMARY KEY (namespace, schema_name)
		)
	`)
	if err != nil {
		return fmt.Errorf("failed to ensure platform.app_schemas: %w", err)
	}

	// Add ignore_content column to existing tables (idempotent; ignores "exists" error).
	_, _ = s.pool.Exec(ctx, `
		ALTER TABLE platform.app_schemas ADD COLUMN IF NOT EXISTS ignore_content TEXT NOT NULL DEFAULT ''
	`)

	_, err = s.pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS platform.app_schema_state (
			namespace            TEXT NOT NULL,
			schema_name          TEXT NOT NULL DEFAULT 'public',
			last_fingerprint     TEXT NOT NULL DEFAULT '',
			applied_at           TIMESTAMPTZ,
			source               TEXT NOT NULL DEFAULT 'app_schema_apply',
			PRIMARY KEY (namespace, schema_name)
		)
	`)
	if err != nil {
		return fmt.Errorf("failed to ensure platform.app_schema_state: %w", err)
	}
	return nil
}

// StoreSchemaContent upserts synced schema content for a (namespace, schema).
// Returns the computed fingerprint and whether the content changed.
// ignoreContent is optional pgschema .pgschemaignore content; empty disables filtering.
func (s *AppDeclarativeService) StoreSchemaContent(ctx context.Context, namespace, schemaName, schemaContent, ignoreContent string) (fingerprint string, changed bool, err error) {
	if namespace == "" {
		return "", false, fmt.Errorf("namespace is required")
	}
	if schemaContent == "" {
		return "", false, fmt.Errorf("schema content cannot be empty")
	}
	if schemaName == "" {
		schemaName = "public"
	}

	if err := s.ensureAppSchemasTable(ctx); err != nil {
		return "", false, err
	}

	sum := sha256.Sum256([]byte(schemaContent))
	fingerprint = hex.EncodeToString(sum[:])

	// Check if content is unchanged (schema + ignore).
	var existing string
	err = s.pool.QueryRow(ctx, `
		SELECT schema_fingerprint FROM platform.app_schemas
		WHERE namespace = $1 AND schema_name = $2
	`, namespace, schemaName).Scan(&existing)
	if err == nil && existing == fingerprint {
		// Unchanged — touch updated_at + keep ignore_content in sync.
		_, _ = s.pool.Exec(ctx, `
			UPDATE platform.app_schemas SET updated_at = now(), ignore_content = $3
			WHERE namespace = $1 AND schema_name = $2
		`, namespace, schemaName, ignoreContent)
		return fingerprint, false, nil
	}

	_, err = s.pool.Exec(ctx, `
		INSERT INTO platform.app_schemas (namespace, schema_name, schema_content, schema_fingerprint, ignore_content, enabled, updated_at)
		VALUES ($1, $2, $3, $4, $5, true, now())
		ON CONFLICT (namespace, schema_name) DO UPDATE SET
			schema_content     = EXCLUDED.schema_content,
			schema_fingerprint = EXCLUDED.schema_fingerprint,
			ignore_content     = EXCLUDED.ignore_content,
			enabled            = true,
			updated_at         = now()
	`, namespace, schemaName, schemaContent, fingerprint, ignoreContent)
	if err != nil {
		return "", false, fmt.Errorf("failed to store app schema: %w", err)
	}
	return fingerprint, true, nil
}

// GetStoredSchemaContent retrieves stored schema content for a (namespace, schema).
func (s *AppDeclarativeService) GetStoredSchemaContent(ctx context.Context, namespace, schemaName string) (content string, fingerprint string, updatedAt time.Time, err error) {
	if schemaName == "" {
		schemaName = "public"
	}
	if err := s.ensureAppSchemasTable(ctx); err != nil {
		return "", "", time.Time{}, err
	}
	err = s.pool.QueryRow(ctx, `
		SELECT schema_content, schema_fingerprint, updated_at
		FROM platform.app_schemas
		WHERE namespace = $1 AND schema_name = $2
	`, namespace, schemaName).Scan(&content, &fingerprint, &updatedAt)
	return content, fingerprint, updatedAt, err
}

// GetStoredIgnoreContent retrieves the stored .pgschemaignore content for a
// (namespace, schema). Returns "" if none is stored.
func (s *AppDeclarativeService) GetStoredIgnoreContent(ctx context.Context, namespace, schemaName string) (string, error) {
	if schemaName == "" {
		schemaName = "public"
	}
	if err := s.ensureAppSchemasTable(ctx); err != nil {
		return "", err
	}
	var ignore string
	err := s.pool.QueryRow(ctx, `
		SELECT COALESCE(ignore_content, '') FROM platform.app_schemas
		WHERE namespace = $1 AND schema_name = $2
	`, namespace, schemaName).Scan(&ignore)
	if err == pgx.ErrNoRows {
		return "", nil
	}
	return ignore, err
}

// ListStoredSchemas lists stored app schemas, optionally filtered by namespace.
func (s *AppDeclarativeService) ListStoredSchemas(ctx context.Context, namespace string) ([]AppSchemaRecord, error) {
	if err := s.ensureAppSchemasTable(ctx); err != nil {
		return nil, err
	}
	rows, err := s.pool.Query(ctx, `
		SELECT namespace, schema_name, schema_fingerprint, enabled, updated_at
		FROM platform.app_schemas
		WHERE ($1 = '' OR namespace = $1)
		ORDER BY namespace, schema_name
	`, namespace)
	if err != nil {
		return nil, fmt.Errorf("failed to list app schemas: %w", err)
	}
	defer rows.Close()

	var records []AppSchemaRecord
	for rows.Next() {
		var r AppSchemaRecord
		if err := rows.Scan(&r.Namespace, &r.SchemaName, &r.SchemaFingerprint, &r.Enabled, &r.UpdatedAt); err != nil {
			return nil, err
		}
		records = append(records, r)
	}
	return records, rows.Err()
}

// DeleteStoredSchema removes stored schema content for a (namespace, schema).
func (s *AppDeclarativeService) DeleteStoredSchema(ctx context.Context, namespace, schemaName string) error {
	if schemaName == "" {
		schemaName = "public"
	}
	if err := s.ensureAppSchemasTable(ctx); err != nil {
		return err
	}
	_, err := s.pool.Exec(ctx, `
		DELETE FROM platform.app_schemas WHERE namespace = $1 AND schema_name = $2
	`, namespace, schemaName)
	return err
}

// Plan generates a pgschema plan comparing stored content to the live database
// for a (namespace, schema). The schema is applied to the main database (s.dbName).
func (s *AppDeclarativeService) Plan(ctx context.Context, namespace, schemaName string) (*Plan, error) {
	if schemaName == "" {
		schemaName = "public"
	}
	content, _, _, err := s.GetStoredSchemaContent(ctx, namespace, schemaName)
	if err != nil {
		return nil, fmt.Errorf("no stored schema for namespace=%s schema=%s: %w", namespace, schemaName, err)
	}
	if content == "" {
		return &Plan{Changes: []Change{}}, nil
	}
	ignore, _ := s.GetStoredIgnoreContent(ctx, namespace, schemaName)
	return s.planFromContent(ctx, schemaName, content, ignore)
}

// planFromContent writes content to a temp dir (alongside an optional
// .pgschemaignore) and runs `pgschema plan`. Setting cmd.Dir to that dir makes
// pgschema auto-discover the ignore file.
func (s *AppDeclarativeService) planFromContent(ctx context.Context, schemaName, schemaContent, ignoreContent string) (*Plan, error) {
	content, err := s.substituteAppUserForContent(schemaContent)
	if err != nil {
		return nil, fmt.Errorf("invalid app user placeholder: %w", err)
	}
	workDir, schemaFile, err := writeSchemaWorkDir(content, ignoreContent)
	if err != nil {
		return nil, err
	}
	defer func() { _ = os.RemoveAll(workDir) }()

	args := []string{
		"plan",
		"--host", s.dbHost,
		"--port", fmt.Sprintf("%d", s.dbPort),
		"--user", s.dbUser,
		"--db", s.dbName,
		"--file", schemaFile,
		"--schema", schemaName,
		"--output-json", "stdout",
		"--plan-host", s.dbHost,
		"--plan-port", fmt.Sprintf("%d", s.dbPort),
		"--plan-user", s.dbUser,
		"--plan-password", s.dbPassword,
		"--plan-db", s.dbName,
	}
	if s.allowDestructive {
		args = append(args, "--allow-destructive")
	}

	cmd := exec.CommandContext(ctx, s.pgschemaPath, args...)
	cmd.Dir = workDir
	cmd.Env = append(os.Environ(), fmt.Sprintf("PGPASSWORD=%s", s.dbPassword))

	start := time.Now()
	output, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return nil, fmt.Errorf("pgschema plan failed: %w: %s", err, string(exitErr.Stderr))
		}
		return nil, fmt.Errorf("pgschema plan failed: %w: %s", err, string(output))
	}

	var plan Plan
	if err := json.Unmarshal(output, &plan); err != nil {
		return nil, fmt.Errorf("failed to parse plan: %w", err)
	}
	plan.Changes = extractChangesFromGroups(&plan)
	plan.Duration = time.Since(start)
	return &plan, nil
}

// applyDirectFallback applies schema content directly via SQL execution, as a
// fallback when pgschema plan cannot validate the desired state (e.g. references
// to extension-provided operators like pgvector's <=>, or cross-schema objects).
// Mirrors DeclarativeService.applySchemaDirectFallback. The content is made
// idempotent via MakeSQLIdempotent, so re-applying a matching schema is a no-op.
//
// Returns an ApplyResult with Fallback=true. When !allowDestructive, destructive
// statements in the content (DROP TABLE/COLUMN/INDEX, TRUNCATE) are blocked —
// the plan-path blocking doesn't cover the fallback, so it is enforced here.
func (s *AppDeclarativeService) applyDirectFallback(ctx context.Context, schemaName, schemaContent string) (*ApplyResult, error) {
	content, err := s.substituteAppUserForContent(schemaContent)
	if err != nil {
		return nil, fmt.Errorf("invalid app user placeholder: %w", err)
	}

	// Block destructive changes on the fallback path when not allowed. The
	// plan-path destructive check (in ApplyFromContent) doesn't run when plan
	// fails, so enforce it here by scanning the parsed content. Note:
	// MakeSQLIdempotent's own DROP-prepends (for POLICY/TRIGGER/INDEX/CONSTRAINT)
	// are safe re-creation aids, not destructive user intent — they are emitted
	// by the transform, not present in the source content.
	if !s.allowDestructive {
		if destructive := countDestructiveStatements(content); destructive > 0 {
			return &ApplyResult{
				Applied:  []Change{},
				Error:    fmt.Errorf("app schema content contains %d destructive statement(s) (DROP TABLE/COLUMN/INDEX, TRUNCATE); blocked (set allow_destructive=true to permit)", destructive),
				Fallback: true,
			}, nil
		}
	}

	connStr := fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=disable",
		s.dbUser, s.dbPassword, s.dbHost, s.dbPort, s.dbName)
	pool, err := pgxpool.New(ctx, connStr)
	if err != nil {
		return nil, fmt.Errorf("failed to create connection pool: %w", err)
	}
	defer pool.Close()

	// Set search_path so unqualified names and extension operators resolve. The
	// target schema comes first; `public` is included for shared types.
	if schemaName == "" {
		schemaName = "public"
	}
	searchPath := schemaName
	if schemaName != "public" {
		searchPath = schemaName + ", public"
	}
	if _, err := pool.Exec(ctx, fmt.Sprintf("SET search_path TO %s", searchPath)); err != nil {
		return nil, fmt.Errorf("failed to set search_path: %w", err)
	}

	// Add missing columns BEFORE the bulk SQL exec. LANGUAGE sql function bodies
	// are validated at creation time and reference columns by name — if a column
	// is declared in a CREATE TABLE but missing from the live DB (additive change),
	// a CREATE OR REPLACE FUNCTION referencing it would fail before the column is
	// added. Running ensureMissingColumns first prevents this ordering issue.
	if err := s.ensureMissingColumnsFromContent(ctx, schemaName, content, pool); err != nil {
		log.Warn().Err(err).Str("schema", schemaName).Msg("Failed to ensure missing columns, continuing")
	}

	idempotentSQL := dbschema.MakeSQLIdempotent(content)
	if _, err := pool.Exec(ctx, idempotentSQL); err != nil {
		return nil, fmt.Errorf("failed to execute schema: %w", err)
	}

	// Report the count of top-level statements executed (best-effort feedback;
	// the fallback doesn't have per-change visibility like pgschema's plan).
	appliedCount := countStatements(content)
	return &ApplyResult{
		Applied:  make([]Change, appliedCount),
		Fallback: true,
	}, nil
}

// ensureMissingColumnsFromContent scans CREATE TABLE IF NOT EXISTS column
// definitions in the given SQL content and adds any columns missing from the
// live database via ALTER TABLE ADD COLUMN. Content-based counterpart to
// DeclarativeService.ensureMissingColumns.
func (s *AppDeclarativeService) ensureMissingColumnsFromContent(ctx context.Context, schemaName, content string, pool *pgxpool.Pool) error {
	if schemaName == "" {
		schemaName = "public"
	}
	stmts, err := parser.Parse(content)
	if err != nil {
		return nil // Unparseable; nothing to do — the idempotent run already executed.
	}
	for _, item := range stmts.Items {
		create, ok := item.(*nodes.CreateStmt)
		if !ok || !create.IfNotExists || create.Relation == nil || create.TableElts == nil {
			continue
		}
		tableName := create.Relation.Relname

		var tableExists bool
		if err := pool.QueryRow(ctx,
			"SELECT EXISTS(SELECT 1 FROM information_schema.tables WHERE table_schema = $1 AND table_name = $2)",
			schemaName, tableName,
		).Scan(&tableExists); err != nil || !tableExists {
			continue
		}

		for _, elt := range create.TableElts.Items {
			col, ok := elt.(*nodes.ColumnDef)
			if !ok {
				continue
			}
			var colExists bool
			if err := pool.QueryRow(ctx,
				"SELECT EXISTS(SELECT 1 FROM information_schema.columns WHERE table_schema = $1 AND table_name = $2 AND column_name = $3)",
				schemaName, tableName, col.Colname,
			).Scan(&colExists); err != nil || colExists {
				continue
			}
			colDef := extractColumnDefByName(content, tableName, col.Colname)
			if colDef == "" {
				// Fall back to the byte-offset-based extractor (works when the
				// parser tracks locations, which it often does not).
				colDef = extractColumnSQL(content, col.Location)
			}
			if colDef == "" {
				continue
			}
			if _, err := pool.Exec(ctx, fmt.Sprintf("ALTER TABLE %s.%s ADD COLUMN %s", schemaName, tableName, colDef)); err != nil {
				log.Warn().Err(err).Str("schema", schemaName).Str("table", tableName).Str("column", col.Colname).Msg("Failed to add missing column")
			} else {
				log.Info().Str("schema", schemaName).Str("table", tableName).Str("column", col.Colname).Msg("Added missing column for app schema upgrade")
			}
		}
	}
	return nil
}

// extractColumnDefByName locates a column definition within the CREATE TABLE
// block for the given table by scanning the raw content for the column name and
// reading to the next top-level comma. This is a content-based fallback used
// because pgparser frequently reports Location=-1 for column definitions, making
// the byte-offset-based extractColumnSQL unreliable.
func extractColumnDefByName(content, tableName, colName string) string {
	// Find the CREATE TABLE IF NOT EXISTS <table> ( ... block.
	createRe := regexp.MustCompile(`(?is)CREATE TABLE IF NOT EXISTS\s+"?` + regexp.QuoteMeta(tableName) + `"?\s*\(`)
	loc := createRe.FindStringIndex(content)
	if loc == nil {
		return ""
	}
	// Scan from the opening paren for the column at top-level (depth 0).
	start := loc[1]
	depth := 0
	lineStart := start
	for i := start; i < len(content); i++ {
		switch content[i] {
		case '(':
			depth++
		case ')':
			if depth == 0 {
				return "" // reached end of table block
			}
			depth--
		case '\n':
			lineStart = i + 1
		case ',':
			if depth == 0 {
				// This comma ends a top-level column/constraint entry.
				entry := strings.TrimSpace(content[lineStart:i])
				if matchesColumnDef(entry, colName) {
					return stripLeadingComma(entry)
				}
				lineStart = i + 1
			}
		}
	}
	return ""
}

// matchesColumnDef reports whether a CREATE TABLE entry is the column named
// colName (as opposed to a table-level CONSTRAINT). Column entries begin with
// the column identifier; constraint entries begin with CONSTRAINT/PRIMARY/etc.
func matchesColumnDef(entry, colName string) bool {
	if entry == "" {
		return false
	}
	upper := strings.ToUpper(entry)
	for _, kw := range []string{"CONSTRAINT ", "PRIMARY KEY", "UNIQUE", "CHECK", "FOREIGN KEY"} {
		if strings.HasPrefix(upper, kw) {
			return false
		}
	}
	// Entry should start with the (optionally quoted) column name.
	trimmed := strings.TrimPrefix(entry, `"`)
	return strings.HasPrefix(trimmed, colName)
}

// stripLeadingComma removes a stray leading comma if the scanner captured one.
func stripLeadingComma(s string) string {
	return strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(s), ","))
}

// ApplyFromContent stores the content and applies it. This is the entry point
// used by `fluxbase schema sync` (store + apply) and by ApplyAllPending (startup)
// against already-stored content.
//
// Apply uses the DIRECT-APPLY FALLBACK as the primary path, not pgschema's plan.
// Rationale: pgschema's plan is a whole-schema diff that includes Fluxbase's
// infrastructure grants (service_role, tenant roles, default privileges) which
// the app's schema file doesn't (and shouldn't) declare — producing destructive
// REVOKEs that would break Fluxbase if applied. The fallback applies only the
// app's declared DDL idempotently (CREATE IF NOT EXISTS + ensureMissingColumns +
// MakeSQLIdempotent for policies/triggers/indexes/constraints), which is the
// correct model for a Fluxbase app. pgschema's plan remains available for
// plan/validate (drift detection) via the Plan() method.
func (s *AppDeclarativeService) ApplyFromContent(ctx context.Context, namespace, schemaName, schemaContent, ignoreContent string) (*ApplyResult, error) {
	if schemaName == "" {
		schemaName = "public"
	}

	// Store first (also computes fingerprint + ensures tables exist).
	fingerprint, _, err := s.StoreSchemaContent(ctx, namespace, schemaName, schemaContent, ignoreContent)
	if err != nil {
		return nil, err
	}

	log.Info().
		Str("namespace", namespace).
		Str("schema", schemaName).
		Str("fingerprint", fingerprint[:12]).
		Msg("Applying declarative app schema")

	res, err := s.applyDirectFallback(ctx, schemaName, schemaContent)
	if err != nil {
		return nil, fmt.Errorf("failed to apply app schema for namespace=%s schema=%s: %w", namespace, schemaName, err)
	}
	// If the fallback blocked destructive changes, surface that result without recording apply.
	if res != nil && res.Error != nil {
		return res, nil
	}
	if err := s.recordApply(ctx, namespace, schemaName, fingerprint); err != nil {
		log.Warn().Err(err).Msg("Failed to record app schema state")
	}
	log.Info().
		Str("namespace", namespace).
		Str("schema", schemaName).
		Int("statements", len(res.Applied)).
		Msg("App schema applied via direct fallback")
	return res, nil
}

// ApplyStored applies the already-stored content for a (namespace, schema).
func (s *AppDeclarativeService) ApplyStored(ctx context.Context, namespace, schemaName string) (*ApplyResult, error) {
	content, _, _, err := s.GetStoredSchemaContent(ctx, namespace, schemaName)
	if err != nil {
		return nil, fmt.Errorf("no stored schema for namespace=%s schema=%s: %w", namespace, schemaName, err)
	}
	if content == "" {
		return &ApplyResult{Applied: []Change{}}, nil
	}
	ignore, _ := s.GetStoredIgnoreContent(ctx, namespace, schemaName)
	return s.ApplyFromContent(ctx, namespace, schemaName, content, ignore)
}

// ApplyAllPending applies stored content for all enabled (namespace, schema) pairs,
// optionally filtered to a set of namespaces (e.g. config.database.declarative_app_schema.namespaces).
// Called on startup when the feature is enabled.
func (s *AppDeclarativeService) ApplyAllPending(ctx context.Context, namespaces []string) error {
	records, err := s.ListStoredSchemas(ctx, "")
	if err != nil {
		return fmt.Errorf("failed to list app schemas: %w", err)
	}

	allowed := make(map[string]bool, len(namespaces))
	for _, n := range namespaces {
		allowed[n] = true
	}
	filter := len(allowed) > 0

	applied := 0
	for _, r := range records {
		if !r.Enabled {
			continue
		}
		if filter && !allowed[r.Namespace] {
			continue
		}
		res, err := s.ApplyStored(ctx, r.Namespace, r.SchemaName)
		if err != nil {
			if res != nil && res.Error != nil {
				log.Error().Err(res.Error).Str("namespace", r.Namespace).Str("schema", r.SchemaName).Msg("App schema apply blocked")
				return res.Error
			}
			return fmt.Errorf("failed to apply app schema for namespace=%s schema=%s: %w", r.Namespace, r.SchemaName, err)
		}
		applied++
		log.Info().
			Str("namespace", r.Namespace).
			Str("schema", r.SchemaName).
			Int("changes", len(res.Applied)).
			Msg("App schema applied on startup")
	}

	if applied == 0 {
		log.Info().Msg("No declarative app schemas to apply on startup")
	}
	return nil
}

// recordApply stores the applied fingerprint in platform.app_schema_state.
func (s *AppDeclarativeService) recordApply(ctx context.Context, namespace, schemaName, fingerprint string) error {
	if schemaName == "" {
		schemaName = "public"
	}
	if err := s.ensureAppSchemasTable(ctx); err != nil {
		return err
	}
	_, err := s.pool.Exec(ctx, `
		INSERT INTO platform.app_schema_state (namespace, schema_name, last_fingerprint, applied_at, source)
		VALUES ($1, $2, $3, now(), 'app_schema_apply')
		ON CONFLICT (namespace, schema_name) DO UPDATE SET
			last_fingerprint = EXCLUDED.last_fingerprint,
			applied_at       = EXCLUDED.applied_at,
			source           = EXCLUDED.source
	`, namespace, schemaName, fingerprint)
	return err
}

// GetStatus returns status for a (namespace, schema), including whether content
// is stored and the last-applied fingerprint.
func (s *AppDeclarativeService) GetStatus(ctx context.Context, namespace, schemaName string) (*AppSchemaStatus, error) {
	if schemaName == "" {
		schemaName = "public"
	}
	if err := s.ensureAppSchemasTable(ctx); err != nil {
		return nil, err
	}
	status := &AppSchemaStatus{Namespace: namespace, SchemaName: schemaName}

	err := s.pool.QueryRow(ctx, `
		SELECT schema_fingerprint, enabled, updated_at
		FROM platform.app_schemas
		WHERE namespace = $1 AND schema_name = $2
	`, namespace, schemaName).Scan(&status.SchemaFingerprint, &status.Enabled, &status.UpdatedAt)
	if err == nil {
		status.HasStoredSchema = true
	}

	_ = s.pool.QueryRow(ctx, `
		SELECT last_fingerprint FROM platform.app_schema_state
		WHERE namespace = $1 AND schema_name = $2
	`, namespace, schemaName).Scan(&status.LastAppliedFingerprint)

	return status, nil
}

// substituteAppUserForContent replaces {{APP_USER}} in content, returning either the
// original (no placeholder / no app user) or substituted bytes.
func (s *AppDeclarativeService) substituteAppUserForContent(content string) (string, error) {
	if s.appUser == "" {
		return content, nil
	}
	return bootstrap.SubstituteAppUser(content, s.appUser)
}

// IsNamespaceManaged reports whether a (namespace, schema) is under declarative
// management (i.e. has stored, enabled content). Used by the imperative migration
// executor to skip + warn rather than race a declaratively-managed schema.
func (s *AppDeclarativeService) IsNamespaceManaged(ctx context.Context, namespace, schemaName string) bool {
	if schemaName == "" {
		schemaName = "public"
	}
	if err := s.ensureAppSchemasTable(ctx); err != nil {
		return false
	}
	var enabled bool
	err := s.pool.QueryRow(ctx, `
		SELECT enabled FROM platform.app_schemas
		WHERE namespace = $1 AND schema_name = $2
	`, namespace, schemaName).Scan(&enabled)
	return err == nil && enabled
}

// WarnIfImperativeCoexists logs a warning for any declaratively-managed namespace
// that also has imperative migrations in platform.migrations. A (namespace, schema)
// should be owned by exactly one mode; this is advisory (does not fail startup)
// because the developer controls which sync commands run against their app.
func (s *AppDeclarativeService) WarnIfImperativeCoexists(ctx context.Context, namespaces []string) {
	if s.pool == nil {
		return
	}
	records, err := s.ListStoredSchemas(ctx, "")
	if err != nil {
		return
	}
	allowed := make(map[string]bool, len(namespaces))
	for _, n := range namespaces {
		allowed[n] = true
	}
	filter := len(allowed) > 0

	for _, r := range records {
		if !r.Enabled || (filter && !allowed[r.Namespace]) {
			continue
		}
		var count int
		// platform.migrations may not exist if the app never used imperative migrations;
		// a query error is benign here.
		err := s.pool.QueryRow(ctx, `
			SELECT count(*) FROM platform.migrations WHERE namespace = $1
		`, r.Namespace).Scan(&count)
		if err == nil && count > 0 {
			log.Warn().
				Str("namespace", r.Namespace).
				Str("schema", r.SchemaName).
				Int("imperative_migrations", count).
				Msg("Namespace is managed declaratively but also has imperative migrations in platform.migrations; a schema should use one mode. Stop running 'fluxbase migrations sync' for this namespace or remove its declarative schema.")
		}
	}
}

// countDestructiveStatements counts top-level destructive SQL statements in
// content (DROP TABLE, DROP COLUMN, DROP INDEX not part of MakeSQLIdempotent's
// re-creation aids, DROP TYPE, DROP FUNCTION/PROCEDURE, DROP VIEW, TRUNCATE).
// Used by the fallback path to enforce allowDestructive=false. Parsing is
// best-effort; on parse failure it conservatively scans keywords.
func countDestructiveStatements(content string) int {
	count := 0
	upper := strings.ToUpper(content)
	for _, line := range strings.Split(upper, "\n") {
		trimmed := strings.TrimSpace(line)
		// Skip MakeSQLIdempotent-generated DROPs (safe re-creation aids) and
		// comment lines. These prefixes are emitted by MakeSQLIdempotent.
		if strings.HasPrefix(trimmed, "DROP POLICY IF EXISTS") ||
			strings.HasPrefix(trimmed, "DROP TRIGGER IF EXISTS") ||
			strings.HasPrefix(trimmed, "DROP CONSTRAINT IF EXISTS") ||
			strings.HasPrefix(trimmed, "ALTER TABLE") && strings.Contains(trimmed, "DROP CONSTRAINT IF EXISTS") ||
			strings.HasPrefix(trimmed, "DROP INDEX IF EXISTS") {
			continue
		}
		if strings.HasPrefix(trimmed, "DROP TABLE") ||
			strings.HasPrefix(trimmed, "DROP COLUMN") ||
			strings.Contains(trimmed, " DROP COLUMN ") ||
			strings.HasPrefix(trimmed, "DROP TYPE") ||
			strings.HasPrefix(trimmed, "DROP FUNCTION") ||
			strings.HasPrefix(trimmed, "DROP PROCEDURE") ||
			strings.HasPrefix(trimmed, "DROP VIEW") ||
			strings.HasPrefix(trimmed, "DROP SCHEMA") ||
			strings.HasPrefix(trimmed, "TRUNCATE") {
			count++
		}
	}
	return count
}

// countStatements returns the number of top-level SQL statements in content,
// estimated via pgparser. Used for best-effort apply feedback on the fallback
// path (which lacks pgschema's per-change plan). Returns 0 on parse failure.
func countStatements(content string) int {
	stmts, err := parser.Parse(content)
	if err != nil || stmts == nil {
		return 0
	}
	return len(stmts.Items)
}

// writeSchemaWorkDir creates a temp directory, writes the schema SQL to
// "schema.sql" inside it, and (if ignoreContent is non-empty) writes a
// ".pgschemaignore" next to it. Returns the dir path (to use as pgschema's
// working directory so it auto-discovers the ignore file) and the schema file
// path. Caller must remove the dir.
func writeSchemaWorkDir(schemaContent, ignoreContent string) (dir, schemaFile string, err error) {
	dir, err = os.MkdirTemp("", "app-schema-*")
	if err != nil {
		return "", "", fmt.Errorf("failed to create temp dir: %w", err)
	}
	schemaFile = filepath.Join(dir, "schema.sql")
	if err := os.WriteFile(schemaFile, []byte(schemaContent), 0o644); err != nil { //nolint:gosec
		_ = os.RemoveAll(dir)
		return "", "", fmt.Errorf("failed to write schema file: %w", err)
	}
	if ignoreContent != "" {
		if err := os.WriteFile(filepath.Join(dir, ".pgschemaignore"), []byte(ignoreContent), 0o644); err != nil { //nolint:gosec
			_ = os.RemoveAll(dir)
			return "", "", fmt.Errorf("failed to write ignore file: %w", err)
		}
	}
	return dir, schemaFile, nil
}

// writeContentToTemp writes content to a temp file and returns its path. The
// {{APP_USER}} placeholder is NOT substituted here (pgschema needs the raw file);
// substitution happens for the idempotent fallback path only.
func writeContentToTemp(pattern, content string) (string, error) {
	tmpFile, err := os.CreateTemp("", pattern)
	if err != nil {
		return "", fmt.Errorf("failed to create temp file: %w", err)
	}
	if _, err := tmpFile.WriteString(content); err != nil {
		_ = tmpFile.Close()
		_ = os.Remove(tmpFile.Name())
		return "", fmt.Errorf("failed to write temp file: %w", err)
	}
	if err := tmpFile.Close(); err != nil {
		return "", fmt.Errorf("failed to close temp file: %w", err)
	}
	return tmpFile.Name(), nil
}

// writePlanToTempFile marshals a plan to JSON in a temp file and returns its path.
func writePlanToTempFile(plan *Plan) (string, error) {
	tmpFile, err := os.CreateTemp("", "pgschema-app-plan-*.json")
	if err != nil {
		return "", fmt.Errorf("failed to create plan temp file: %w", err)
	}
	defer func() { _ = tmpFile.Close() }()

	planJSON, err := json.Marshal(plan)
	if err != nil {
		_ = os.Remove(tmpFile.Name())
		return "", fmt.Errorf("failed to marshal plan: %w", err)
	}
	if _, err := tmpFile.Write(planJSON); err != nil {
		_ = os.Remove(tmpFile.Name())
		return "", fmt.Errorf("failed to write plan temp file: %w", err)
	}
	return tmpFile.Name(), nil
}
