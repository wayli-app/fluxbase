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
	"sort"
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

// DeclarativeService manages Fluxbase internal schema using pgschema
type DeclarativeService struct {
	pgschemaPath string
	dbHost       string
	dbPort       int
	dbUser       string
	dbPassword   string
	dbName       string
	appUser      string // Runtime app user for {{APP_USER}} GRANT substitution
	config       DeclarativeConfig
	pool         *pgxpool.Pool // Optional: for recording state
}

// DeclarativeConfig holds configuration for declarative schema management
type DeclarativeConfig struct {
	SchemaDir        string   // Directory containing per-schema SQL files (e.g., internal/database/schema/schemas/)
	Schemas          []string // Schemas to manage (Fluxbase internal only)
	AllowDestructive bool     // Allow destructive changes
	LockTimeout      int      // Lock timeout in seconds
}

// DefaultFluxbaseSchemas lists all Fluxbase internal schemas in dependency order
// platform must come first as auth FKs reference platform.tenants
// auth must come early as other schemas reference auth.users
var DefaultFluxbaseSchemas = []string{
	"platform", "auth", "storage", "jobs", "functions", "realtime",
	"ai", "rpc", "system", "migrations",
	"app", "api", "branching", "logging", "mcp",
}

// NewDeclarativeService creates a new declarative schema service
func NewDeclarativeService(pgschemaPath string, dbHost string, dbPort int, dbUser, dbPassword, dbName string, config DeclarativeConfig) *DeclarativeService {
	return &DeclarativeService{
		pgschemaPath: pgschemaPath,
		dbHost:       dbHost,
		dbPort:       dbPort,
		dbUser:       dbUser,
		dbPassword:   dbPassword,
		dbName:       dbName,
		config:       config,
	}
}

// SetPool sets the database pool for state recording
func (s *DeclarativeService) SetPool(pool *pgxpool.Pool) {
	s.pool = pool
}

// SetAppUser sets the runtime database user for {{APP_USER}} placeholder substitution.
func (s *DeclarativeService) SetAppUser(appUser string) {
	s.appUser = appUser
}

// preprocessSchemaFile reads a schema SQL file and substitutes {{APP_USER}} placeholders
// with the configured runtime user. If no substitution is needed, it returns the original
// file path with a nil cleanup function. Otherwise, it writes a temp file and returns
// its path with a cleanup function to remove it.
func (s *DeclarativeService) preprocessSchemaFile(schemaFile string) (string, func(), error) {
	if s.appUser == "" {
		return schemaFile, nil, nil
	}

	content, err := os.ReadFile(schemaFile)
	if err != nil {
		return "", nil, fmt.Errorf("failed to read schema file: %w", err)
	}

	if !strings.Contains(string(content), bootstrap.AppUserPlaceholder) {
		return schemaFile, nil, nil
	}

	substituted, err := bootstrap.SubstituteAppUser(string(content), s.appUser)
	if err != nil {
		return "", nil, fmt.Errorf("invalid app user in %s: %w", schemaFile, err)
	}
	tmpFile, err := os.CreateTemp("", "fluxbase-schema-*.sql")
	if err != nil {
		return "", nil, fmt.Errorf("failed to create temp file: %w", err)
	}

	if _, err := tmpFile.WriteString(substituted); err != nil {
		_ = tmpFile.Close()
		_ = os.Remove(tmpFile.Name())
		return "", nil, fmt.Errorf("failed to write temp file: %w", err)
	}
	_ = tmpFile.Close()

	cleanup := func() { _ = os.Remove(tmpFile.Name()) }
	return tmpFile.Name(), cleanup, nil
}

// PlanForSchema generates a migration plan for a single schema
func (s *DeclarativeService) PlanForSchema(ctx context.Context, schema string) (*Plan, error) {
	schemaFile := filepath.Join(s.config.SchemaDir, schema+".sql")

	// Check if schema file exists
	if _, err := os.Stat(schemaFile); os.IsNotExist(err) {
		return nil, fmt.Errorf("schema file not found: %s", schemaFile)
	}

	// Pre-process schema file for placeholder substitution if needed.
	// pgschema reads the file from disk, so we write a temp file with
	// {{APP_USER}} replaced by the actual runtime user.
	processedFile, cleanup, err := s.preprocessSchemaFile(schemaFile)
	if err != nil {
		return nil, fmt.Errorf("failed to preprocess schema file: %w", err)
	}
	if cleanup != nil {
		defer cleanup()
	}

	args := []string{
		"plan",
		"--host", s.dbHost,
		"--port", fmt.Sprintf("%d", s.dbPort),
		"--user", s.dbUser,
		"--db", s.dbName,
		"--file", processedFile,
		"--schema", schema,
		"--output-json", "stdout",
		// Use the actual database for plan validation (needed for roles, extensions, etc.)
		"--plan-host", s.dbHost,
		"--plan-port", fmt.Sprintf("%d", s.dbPort),
		"--plan-user", s.dbUser,
		"--plan-password", s.dbPassword,
		"--plan-db", s.dbName,
	}

	if s.config.AllowDestructive {
		args = append(args, "--allow-destructive")
	}

	cmd := exec.CommandContext(ctx, s.pgschemaPath, args...)
	cmd.Env = append(os.Environ(), fmt.Sprintf("PGPASSWORD=%s", s.dbPassword))

	start := time.Now()
	output, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return nil, fmt.Errorf("pgschema plan failed for schema %s: %w: %s", schema, err, string(exitErr.Stderr))
		}
		return nil, fmt.Errorf("pgschema plan failed for schema %s: %w: %s", schema, err, string(output))
	}

	var plan Plan
	if err := json.Unmarshal(output, &plan); err != nil {
		return nil, fmt.Errorf("failed to parse plan for schema %s: %w", schema, err)
	}

	// Extract changes from groups/steps structure
	plan.Changes = extractChangesFromGroups(&plan)

	plan.Duration = time.Since(start)
	return &plan, nil
}

// extractChangesFromGroups converts pgschema's groups/steps structure to a flat Changes slice
func extractChangesFromGroups(plan *Plan) []Change {
	var changes []Change
	for _, group := range plan.Groups {
		for _, step := range group.Steps {
			// Skip directive-only steps (like wait for index)
			if step.SQL == "" {
				continue
			}

			change := Change{
				SQL: step.SQL,
			}

			// Parse path to extract schema, object type, and name
			// Path format: "schema.object_type.name" or "schema.object_type.constraint_name"
			parts := strings.Split(step.Path, ".")
			if len(parts) >= 1 {
				change.Schema = parts[0]
			}
			if len(parts) >= 2 {
				change.ObjectType = parts[1]
			}
			if len(parts) >= 3 {
				change.Name = strings.Join(parts[2:], ".")
			}

			// Convert operation to ChangeType
			switch step.Operation {
			case "create":
				change.Type = ChangeCreate
			case "drop":
				change.Type = ChangeDrop
				change.Destructive = true
			case "alter":
				change.Type = ChangeAlter
			}

			changes = append(changes, change)
		}
	}
	return changes
}

// Plan generates a combined migration plan for all schemas
func (s *DeclarativeService) Plan(ctx context.Context) (*Plan, error) {
	combined := &Plan{
		Changes: []Change{},
	}

	for _, schema := range s.config.Schemas {
		plan, err := s.PlanForSchema(ctx, schema)
		if err != nil {
			return nil, err
		}
		combined.Changes = append(combined.Changes, plan.Changes...)
		combined.Duration += plan.Duration
	}

	return combined, nil
}

// ApplyForSchema applies the migration plan for a single schema
func (s *DeclarativeService) ApplyForSchema(ctx context.Context, schema string, autoApprove bool) (*ApplyResult, error) {
	// First, generate plan
	plan, err := s.PlanForSchema(ctx, schema)
	if err != nil {
		return nil, err
	}

	if len(plan.Changes) == 0 {
		return &ApplyResult{Applied: []Change{}, Duration: 0}, nil
	}

	// Write plan to temp file
	planFile, err := s.writePlanToTemp(plan)
	if err != nil {
		return nil, err
	}
	defer func() { _ = os.Remove(planFile) }()

	args := []string{
		"apply",
		"--host", s.dbHost,
		"--port", fmt.Sprintf("%d", s.dbPort),
		"--user", s.dbUser,
		"--db", s.dbName,
		"--schema", schema,
		"--plan", planFile,
	}

	if autoApprove {
		args = append(args, "--auto-approve")
	}

	cmd := exec.CommandContext(ctx, s.pgschemaPath, args...)
	cmd.Env = append(os.Environ(), fmt.Sprintf("PGPASSWORD=%s", s.dbPassword))

	start := time.Now()
	output, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return nil, fmt.Errorf("pgschema apply failed for schema %s: %w: %s", schema, err, string(exitErr.Stderr))
		}
		return nil, fmt.Errorf("pgschema apply failed for schema %s: %w: %s", schema, err, string(output))
	}

	return &ApplyResult{
		Applied:  plan.Changes,
		Duration: time.Since(start),
	}, nil
}

// applySchemaDirect applies schema changes directly using psql instead of pgschema
// This is used for schemas with cross-schema references that can't be validated in isolation
// pgschema validates schema SQL in a temporary schema, even with --plan-host, which fails
// for schemas that reference tables defined later in the same file or in other schemas.
// Apply executes the migration plan for all schemas
func (s *DeclarativeService) Apply(ctx context.Context, autoApprove bool) (*ApplyResult, error) {
	combined := &ApplyResult{
		Applied: []Change{},
	}

	for _, schema := range s.config.Schemas {
		result, err := s.ApplyForSchema(ctx, schema, autoApprove)
		if err != nil {
			return nil, err
		}
		combined.Applied = append(combined.Applied, result.Applied...)
		combined.Duration += result.Duration
	}

	// Record in declarative_state table
	if err := s.recordApply(ctx, combined); err != nil {
		log.Warn().Err(err).Msg("Failed to record declarative state")
	}

	return combined, nil
}

// Dump exports current database schema for a single schema
func (s *DeclarativeService) DumpForSchema(ctx context.Context, schema string, outputPath string) error {
	// Create output directory if needed
	dir := filepath.Dir(outputPath)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Open output file
	f, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer func() { _ = f.Close() }()

	args := []string{
		"dump",
		"--host", s.dbHost,
		"--port", fmt.Sprintf("%d", s.dbPort),
		"--user", s.dbUser,
		"--db", s.dbName,
		"--schema", schema,
	}

	cmd := exec.CommandContext(ctx, s.pgschemaPath, args...)
	cmd.Env = append(os.Environ(), fmt.Sprintf("PGPASSWORD=%s", s.dbPassword))
	cmd.Stdout = f

	if err := cmd.Run(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return fmt.Errorf("pgschema dump failed for schema %s: %w: %s", schema, err, string(exitErr.Stderr))
		}
		return fmt.Errorf("pgschema dump failed for schema %s: %w", schema, err)
	}

	log.Info().Str("schema", schema).Str("path", outputPath).Msg("Schema dump completed")
	return nil
}

// Dump exports all schemas to separate files in the schema directory
func (s *DeclarativeService) Dump(ctx context.Context, schemaDir string) error {
	// Create output directory if needed
	if err := os.MkdirAll(schemaDir, 0o755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	for _, schema := range s.config.Schemas {
		outputPath := filepath.Join(schemaDir, schema+".sql")
		if err := s.DumpForSchema(ctx, schema, outputPath); err != nil {
			return err
		}
	}

	log.Info().Str("dir", schemaDir).Int("schemas", len(s.config.Schemas)).Msg("All schemas dumped")
	return nil
}

// Validate checks for schema drift across all schemas
func (s *DeclarativeService) Validate(ctx context.Context) (*ValidationResult, error) {
	combined := &ValidationResult{Valid: true}

	for _, schema := range s.config.Schemas {
		plan, err := s.PlanForSchema(ctx, schema)
		if err != nil {
			return &ValidationResult{Valid: false, Error: err}, nil
		}

		if len(plan.Changes) > 0 {
			combined.Valid = false
			for _, change := range plan.Changes {
				combined.Drifts = append(combined.Drifts, Drift{
					Type:        string(change.Type),
					ObjectType:  change.ObjectType,
					Schema:      change.Schema,
					Name:        change.Name,
					SQL:         change.SQL,
					Destructive: change.Destructive,
				})
			}
		}
	}

	return combined, nil
}

// CalculateFingerprint computes a SHA256 hash of all schema files
func (s *DeclarativeService) CalculateFingerprint() (string, error) {
	hasher := sha256.New()

	for _, schema := range s.config.Schemas {
		schemaFile := filepath.Join(s.config.SchemaDir, schema+".sql")
		content, err := os.ReadFile(schemaFile)
		if err != nil {
			if os.IsNotExist(err) {
				continue // Skip missing schema files
			}
			return "", fmt.Errorf("failed to read schema file %s: %w", schemaFile, err)
		}
		hasher.Write(content)
		hasher.Write([]byte("|")) // Separator between schemas
	}

	return hex.EncodeToString(hasher.Sum(nil)), nil
}

// writePlanToTemp writes the plan JSON to a temporary file
func (s *DeclarativeService) writePlanToTemp(plan *Plan) (string, error) {
	tmpFile, err := os.CreateTemp("", "pgschema-plan-*.json")
	if err != nil {
		return "", fmt.Errorf("failed to create temp file: %w", err)
	}
	defer func() { _ = tmpFile.Close() }()

	planJSON, err := json.Marshal(plan)
	if err != nil {
		_ = os.Remove(tmpFile.Name())
		return "", fmt.Errorf("failed to marshal plan: %w", err)
	}

	if _, err := tmpFile.Write(planJSON); err != nil {
		_ = os.Remove(tmpFile.Name())
		return "", fmt.Errorf("failed to write plan: %w", err)
	}

	return tmpFile.Name(), nil
}

// recordApply records the applied schema state in the database with default source
func (s *DeclarativeService) recordApply(ctx context.Context, result *ApplyResult) error {
	return s.recordApplyWithSource(ctx, result, "schema_apply")
}

// recordApplyWithSource records the applied schema state in the database with specified source
func (s *DeclarativeService) recordApplyWithSource(ctx context.Context, result *ApplyResult, source string) error {
	fingerprint, err := s.CalculateFingerprint()
	if err != nil {
		log.Warn().Err(err).Msg("Failed to calculate fingerprint")
		fingerprint = "unknown"
	}

	log.Info().
		Str("fingerprint", fingerprint).
		Str("source", source).
		Int("changes", len(result.Applied)).
		Msg("Schema applied")

	// Record to database if pool is available
	if s.pool != nil {
		_, err := s.pool.Exec(ctx, `
			INSERT INTO platform.declarative_state (schema_fingerprint, applied_by, source)
			VALUES ($1, 'fluxbase', $2)
		`, fingerprint, source)
		if err != nil {
			log.Warn().Err(err).Msg("Failed to record declarative state to database")
		}
	}

	return nil
}

// ApplyDeclarative applies the declarative schema on startup with default source.
// It plans and applies any pending schema changes automatically.
// This is the main entry point for automatic schema management.
func (s *DeclarativeService) ApplyDeclarative(ctx context.Context) error {
	return s.ApplyDeclarativeWithSource(ctx, "schema_apply")
}

// ApplyDeclarativeWithSource applies the declarative schema on startup with specified source.
// It uses pgschema for proper schema diffing and evolution.
// The source parameter indicates the origin: 'fresh_install', 'transitioned', or 'schema_apply'.
func (s *DeclarativeService) ApplyDeclarativeWithSource(ctx context.Context, source string) error {
	log.Info().Str("schema_dir", s.config.SchemaDir).Msg("Checking declarative schema...")

	// Check if schema directory exists
	if _, err := os.Stat(s.config.SchemaDir); os.IsNotExist(err) {
		return fmt.Errorf("schema directory not found: %s", s.config.SchemaDir)
	}

	// Phase 1: Apply each schema using pgschema for proper diffing
	// pgschema handles schema evolution correctly by:
	// - Detecting existing tables and only adding new columns
	// - Generating proper ALTER TABLE statements
	// - Handling index and constraint changes
	for _, schema := range s.config.Schemas {
		schemaFile := filepath.Join(s.config.SchemaDir, schema+".sql")

		// Check if schema file exists
		if _, err := os.Stat(schemaFile); os.IsNotExist(err) {
			log.Debug().Str("schema", schema).Msg("Schema file not found, skipping")
			continue
		}

		// Generate plan first to see if there are any changes
		plan, err := s.PlanForSchema(ctx, schema)
		if err != nil {
			// If plan fails (e.g., due to cross-schema FK validation in temporary schema),
			// fall back to direct schema application with idempotent transforms
			log.Warn().Err(err).Str("schema", schema).Msg("pgschema plan failed, using direct fallback")
			if err := s.applySchemaDirectFallback(ctx, schema); err != nil {
				return fmt.Errorf("failed to apply schema %s: %w", schema, err)
			}
			log.Info().Str("schema", schema).Msg("Schema applied via direct fallback")
			continue
		}

		// Filter out FK drops for cross-schema constraints managed by post-schema-fks.sql.
		// These are applied separately in Phase 2 and should not be dropped during per-schema apply.
		plan.Changes = s.filterManagedFKDrops(plan.Changes)

		if len(plan.Changes) == 0 {
			log.Debug().Str("schema", schema).Msg("No schema changes needed")
			continue
		}

		// Apply the filtered plan directly using SQL execution.
		// We use applyPlanDirectly instead of ApplyForSchema because ApplyForSchema
		// regenerates the plan internally, which would not include our FK filtering.
		if err := s.applyPlanDirectly(ctx, schema, plan); err != nil {
			return fmt.Errorf("failed to apply schema %s: %w", schema, err)
		}
		log.Info().Str("schema", schema).Int("changes", len(plan.Changes)).Msg("Schema changes applied via plan execution")
	}

	// Phase 2: Apply cross-schema FKs from post-schema-fks.sql
	if err := s.applyCrossSchemaFKs(ctx); err != nil {
		return fmt.Errorf("failed to apply cross-schema FKs: %w", err)
	}

	// Phase 3: Apply post-schema.sql for cross-schema policies
	if err := s.applyPostSchemaPolicies(ctx); err != nil {
		return fmt.Errorf("failed to apply post-schema: %w", err)
	}

	// Record the schema state with the specified source
	if err := s.recordApplyWithSource(ctx, &ApplyResult{Applied: []Change{}}, source); err != nil {
		log.Warn().Err(err).Msg("Failed to record schema state")
	}

	return nil
}

// applySchemaDirectFallback applies a schema file directly (fallback when pgschema fails)
func (s *DeclarativeService) applySchemaDirectFallback(ctx context.Context, schema string) error {
	schemaFile := filepath.Join(s.config.SchemaDir, schema+".sql")

	// Read the schema file
	content, err := os.ReadFile(schemaFile)
	if err != nil {
		return fmt.Errorf("failed to read schema file: %w", err)
	}

	// Substitute {{APP_USER}} placeholder with the runtime user
	contentStr, err := bootstrap.SubstituteAppUser(string(content), s.appUser)
	if err != nil {
		return fmt.Errorf("invalid app user for schema apply: %w", err)
	}

	// Create connection
	connStr := fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=disable",
		s.dbUser, s.dbPassword, s.dbHost, s.dbPort, s.dbName)
	pool, err := pgxpool.New(ctx, connStr)
	if err != nil {
		return fmt.Errorf("failed to create connection pool: %w", err)
	}
	defer pool.Close()

	// Set search_path to include all schemas for function/type references
	// Schemas are applied in dependency order, so earlier schemas are already created
	allSchemas := strings.Join(s.config.Schemas, ", ")
	searchPath := fmt.Sprintf("%s, %s, public", schema, allSchemas)
	_, err = pool.Exec(ctx, fmt.Sprintf("SET search_path TO %s", searchPath))
	if err != nil {
		return fmt.Errorf("failed to set search_path: %w", err)
	}

	idempotentSQL := dbschema.MakeSQLIdempotent(contentStr)
	_, err = pool.Exec(ctx, idempotentSQL)
	if err != nil {
		return fmt.Errorf("failed to execute schema: %w", err)
	}

	log.Info().Str("schema", schema).Msg("Schema applied directly")
	return nil
}

// applyPlanDirectly executes the SQL statements from a plan directly
// This is used when pgschema apply fails due to validation issues but the plan is valid
func (s *DeclarativeService) applyPlanDirectly(ctx context.Context, schema string, plan *Plan) error {
	if len(plan.Changes) == 0 {
		return nil
	}

	// Create connection
	connStr := fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=disable",
		s.dbUser, s.dbPassword, s.dbHost, s.dbPort, s.dbName)
	pool, err := pgxpool.New(ctx, connStr)
	if err != nil {
		return fmt.Errorf("failed to create connection pool: %w", err)
	}
	defer pool.Close()

	// Set search_path to include all schemas for cross-schema references.
	// The target schema comes first, then all other schemas for FK references.
	// Note: We include the target schema because pgschema generates unqualified
	// SQL (e.g., "bootstrap_state_id_seq" instead of "platform.bootstrap_state_id_seq").
	allSchemas := strings.Join(s.config.Schemas, ", ")
	searchPath := fmt.Sprintf("%s, %s, public", schema, allSchemas)
	_, err = pool.Exec(ctx, fmt.Sprintf("SET search_path TO %s", searchPath))
	if err != nil {
		return fmt.Errorf("failed to set search_path: %w", err)
	}

	// Reorder changes so that table structure changes (ALTER TABLE ADD COLUMN)
	// execute before policy changes that reference those columns.
	// pgschema may generate ALTER POLICY before ALTER TABLE ADD COLUMN,
	// causing "column does not exist" errors during upgrades.
	reordered := make([]Change, len(plan.Changes))
	copy(reordered, plan.Changes)
	sort.SliceStable(reordered, func(i, j int) bool {
		return changePriority(reordered[i]) < changePriority(reordered[j])
	})
	plan.Changes = reordered

	// Ensure columns defined in CREATE TABLE IF NOT EXISTS exist in the database.
	// pgschema treats CREATE TABLE IF NOT EXISTS as atomic — it doesn't diff
	// individual columns, so ALTER TABLE ADD COLUMN is never generated for
	// missing columns. This pre-step scans the schema SQL file for column
	// definitions and adds any that are missing before the plan runs.
	if err := s.ensureMissingColumns(ctx, schema, pool); err != nil {
		log.Warn().Err(err).Str("schema", schema).Msg("Failed to ensure missing columns, continuing with plan")
	}

	// Execute each change's SQL statement
	// Pre-load partition table names so we can skip ALTER TABLE ADD COLUMN
	// on partitions (columns propagate from the parent automatically).
	partitions := s.listPartitionTables(ctx, schema, pool)
	partitionedTables := s.listPartitionedTables(ctx, schema, pool)

	for i, change := range plan.Changes {
		if change.SQL == "" {
			continue
		}

		// Skip ALTER TABLE ADD COLUMN on partition tables
		sqlUpper := strings.ToUpper(strings.TrimSpace(change.SQL))
		if strings.HasPrefix(sqlUpper, "ALTER TABLE") && strings.Contains(sqlUpper, "ADD COLUMN") {
			tableName := extractAlterTableName(change.SQL)
			if partitions[tableName] {
				log.Debug().Str("schema", schema).Str("table", tableName).Msg("Skipping ADD COLUMN on partition table (propagated from parent)")
				continue
			}
		}

		// Skip CREATE TRIGGER on partition tables (cloned from parent automatically)
		if strings.HasPrefix(sqlUpper, "CREATE") && strings.Contains(sqlUpper, "TRIGGER") {
			trigTable := extractTriggerTableName(change.SQL)
			if partitions[trigTable] {
				log.Debug().Str("schema", schema).Str("table", trigTable).Msg("Skipping trigger on partition table (cloned from parent)")
				continue
			}
		}

		// Skip DROP TRIGGER on partition tables (inherited from parent, cannot be dropped independently)
		// Also skip on partitioned parent tables (triggers managed outside pgschema, e.g. in post-schema.sql)
		if strings.HasPrefix(sqlUpper, "DROP TRIGGER") {
			trigTable := extractDropTriggerTableName(change.SQL)
			if partitions[trigTable] || partitionedTables[trigTable] {
				log.Debug().Str("schema", schema).Str("table", trigTable).Msg("Skipping DROP TRIGGER on partition/partitioned table (managed outside schema SQL)")
				continue
			}
		}

		// Skip ALTER TABLE ... DROP CONSTRAINT on partition tables.
		// Partition tables inherit constraints from their parent and PostgreSQL
		// rejects direct drops with "cannot drop inherited constraint" (SQLSTATE 42P16).
		if strings.HasPrefix(sqlUpper, "ALTER TABLE") && strings.Contains(sqlUpper, "DROP CONSTRAINT") {
			constraintTable := extractAlterTableName(change.SQL)
			if partitions[constraintTable] {
				log.Debug().Str("schema", schema).Str("table", constraintTable).Msg("Skipping DROP CONSTRAINT on partition table (inherited from parent)")
				continue
			}
		}

		// Skip ALTER TABLE ... ADD CONSTRAINT ... PRIMARY KEY on partition tables.
		// The primary key is inherited from the partition parent after ATTACH PARTITION,
		// so adding one on the child fails with "multiple primary keys" (SQLSTATE 42P16).
		if strings.HasPrefix(sqlUpper, "ALTER TABLE") && strings.Contains(sqlUpper, "ADD CONSTRAINT") && strings.Contains(sqlUpper, "PRIMARY KEY") {
			constraintTable := extractAlterTableName(change.SQL)
			if partitions[constraintTable] {
				log.Debug().Str("schema", schema).Str("table", constraintTable).Msg("Skipping ADD CONSTRAINT PRIMARY KEY on partition table (inherited from parent)")
				continue
			}
		}

		// Substitute {{APP_USER}} placeholder with the runtime user
		sql, subErr := bootstrap.SubstituteAppUser(change.SQL, s.appUser)
		if subErr != nil {
			log.Warn().Err(subErr).Str("schema", schema).Msg("Skipping change with invalid app user")
			continue
		}

		// Strip CONCURRENTLY from CREATE INDEX on partitioned tables
		if strings.Contains(sqlUpper, "CREATE INDEX CONCURRENTLY") {
			idxTable := extractIndexTableName(sql)
			if partitionedTables[idxTable] {
				sql = strings.Replace(sql, "CONCURRENTLY ", "", 1)
				log.Debug().Str("schema", schema).Str("table", idxTable).Msg("Stripped CONCURRENTLY from index on partitioned table")
			}
		}

		// Log the change being applied (truncate SQL if too long)
		sqlPreview := sql
		if len(sqlPreview) > 200 {
			sqlPreview = sqlPreview[:197] + "..."
		}
		log.Info().
			Str("schema", schema).
			Int("change_num", i+1).
			Int("total_changes", len(plan.Changes)).
			Str("type", string(change.Type)).
			Str("object", change.Name).
			Str("sql", sqlPreview).
			Msg("Applying schema change")

		_, err := pool.Exec(ctx, sql)
		if err != nil {
			// Log the error with context
			log.Error().
				Err(err).
				Str("schema", schema).
				Str("sql", sql).
				Msg("Failed to execute plan SQL")
			return fmt.Errorf("failed to execute plan SQL for schema %s (change %d/%d): %w", schema, i+1, len(plan.Changes), err)
		}
	}

	log.Info().Str("schema", schema).Int("changes", len(plan.Changes)).Msg("Plan SQL executed successfully")
	return nil
}

// applyCrossSchemaFKs applies cross-schema foreign keys from post-schema-fks.sql
func (s *DeclarativeService) applyCrossSchemaFKs(ctx context.Context) error {
	fksFile := filepath.Join(s.config.SchemaDir, "post-schema-fks.sql")

	// Check if file exists
	if _, err := os.Stat(fksFile); os.IsNotExist(err) {
		log.Debug().Msg("post-schema-fks.sql not found, skipping")
		return nil
	}

	// Read the file
	content, err := os.ReadFile(fksFile)
	if err != nil {
		return fmt.Errorf("failed to read post-schema-fks.sql: %w", err)
	}

	// Create connection
	connStr := fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=disable",
		s.dbUser, s.dbPassword, s.dbHost, s.dbPort, s.dbName)
	pool, err := pgxpool.New(ctx, connStr)
	if err != nil {
		return fmt.Errorf("failed to create connection pool: %w", err)
	}
	defer pool.Close()

	// Execute the FK additions (they use idempotent DO blocks)
	_, err = pool.Exec(ctx, string(content))
	if err != nil {
		return fmt.Errorf("failed to apply cross-schema FKs: %w", err)
	}

	log.Info().Msg("Cross-schema FKs applied")
	return nil
}

// applyPostSchemaPolicies applies post-schema.sql for cross-schema policies
func (s *DeclarativeService) applyPostSchemaPolicies(ctx context.Context) error {
	postSchemaFile := filepath.Join(s.config.SchemaDir, "post-schema.sql")

	// Check if file exists
	if _, err := os.Stat(postSchemaFile); os.IsNotExist(err) {
		log.Debug().Msg("post-schema.sql not found, skipping")
		return nil
	}

	// Read the file
	content, err := os.ReadFile(postSchemaFile)
	if err != nil {
		return fmt.Errorf("failed to read post-schema.sql: %w", err)
	}

	// Create connection
	connStr := fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=disable",
		s.dbUser, s.dbPassword, s.dbHost, s.dbPort, s.dbName)
	pool, err := pgxpool.New(ctx, connStr)
	if err != nil {
		return fmt.Errorf("failed to create connection pool: %w", err)
	}
	defer pool.Close()

	// Build search_path with all schemas
	allSchemas := strings.Join(s.config.Schemas, ", ")
	_, err = pool.Exec(ctx, fmt.Sprintf("SET search_path TO %s, public", allSchemas))
	if err != nil {
		return fmt.Errorf("failed to set search_path: %w", err)
	}

	// Execute with idempotent transforms
	idempotentSQL := dbschema.MakeSQLIdempotent(string(content))
	_, err = pool.Exec(ctx, idempotentSQL)
	if err != nil {
		return fmt.Errorf("failed to apply post-schema.sql: %w", err)
	}

	log.Info().Msg("Post-schema policies applied")
	return nil
}

// hasDestructiveChanges checks if any changes are destructive
func hasDestructiveChanges(changes []Change) bool {
	for _, c := range changes {
		if c.Destructive {
			return true
		}
	}
	return false
}

// crossSchemaFKNames extracts FK constraint names from post-schema-fks.sql.
// These constraints are managed outside per-schema SQL files, so pgschema should not
// attempt to drop them during per-schema plan+apply.
var crossSchemaFKNames map[string]bool

// loadCrossSchemaFKNames parses post-schema-fks.sql and returns a set of FK constraint names.
func (s *DeclarativeService) loadCrossSchemaFKNames() map[string]bool {
	if crossSchemaFKNames != nil {
		return crossSchemaFKNames
	}

	names := make(map[string]bool)
	fksFile := filepath.Join(s.config.SchemaDir, "post-schema-fks.sql")
	content, err := os.ReadFile(fksFile)
	if err != nil {
		return names
	}

	// Match: conname = 'constraint_name'
	re := regexp.MustCompile(`conname\s*=\s*'([^']+)'`)
	matches := re.FindAllStringSubmatch(string(content), -1)
	for _, m := range matches {
		if len(m) > 1 {
			names[m[1]] = true
		}
	}

	crossSchemaFKNames = names
	return names
}

// filterManagedFKDrops removes DROP CONSTRAINT changes for FKs managed by post-schema-fks.sql.
// These cross-schema FKs are applied separately in Phase 2, so pgschema must not drop them.
func (s *DeclarativeService) filterManagedFKDrops(changes []Change) []Change {
	managedFKs := s.loadCrossSchemaFKNames()
	var filtered []Change
	for _, change := range changes {
		if change.Type == ChangeDrop && managedFKs[change.Name] {
			log.Debug().
				Str("constraint", change.Name).
				Msg("Skipping FK drop - managed by post-schema-fks.sql")
			continue
		}
		filtered = append(filtered, change)
	}
	return filtered
}

// makeSQLIdempotent transforms SQL to be idempotent by:
// - Converting CREATE POLICY to DROP POLICY IF EXISTS + CREATE POLICY
// - Converting ALTER TABLE ... ADD CONSTRAINT to DROP CONSTRAINT IF EXISTS + ADD CONSTRAINT
// ensureMissingColumns scans the schema SQL file for columns defined in
// CREATE TABLE IF NOT EXISTS statements and adds any that are missing from
// the actual database. pgschema treats CREATE TABLE IF NOT EXISTS as atomic
// and doesn't diff individual columns, so ALTER TABLE ADD COLUMN is never
// generated for missing columns during upgrades.
func (s *DeclarativeService) ensureMissingColumns(ctx context.Context, schema string, pool *pgxpool.Pool) error {
	schemaFile := filepath.Join(s.config.SchemaDir, schema+".sql")
	content, err := os.ReadFile(schemaFile)
	if err != nil {
		return nil // No file for this schema, skip
	}

	// Preprocess {{APP_USER}} substitution
	sql, subErr := bootstrap.SubstituteAppUser(string(content), s.appUser)
	if subErr != nil {
		return fmt.Errorf("invalid app user in schema %s: %w", schema, subErr)
	}

	// Parse SQL using pgparser to find CREATE TABLE column definitions
	stmts, err := parser.Parse(sql)
	if err != nil {
		return nil
	}

	// Collect partition table names so we can skip them — columns must be
	// added to the parent table and propagate automatically.
	partitions := s.listPartitionTables(ctx, schema, pool)

	for _, item := range stmts.Items {
		create, ok := item.(*nodes.CreateStmt)
		if !ok || !create.IfNotExists || create.Relation == nil || create.TableElts == nil {
			continue
		}
		tableName := create.Relation.Relname

		// Skip partition tables — columns are added to the parent and propagate
		if partitions[tableName] {
			continue
		}

		// Skip tables that don't exist yet (CREATE TABLE will handle them)
		var tableExists bool
		if err := pool.QueryRow(
			ctx,
			"SELECT EXISTS(SELECT 1 FROM information_schema.tables WHERE table_schema = $1 AND table_name = $2)",
			schema, tableName,
		).Scan(&tableExists); err != nil || !tableExists {
			continue
		}

		for _, elt := range create.TableElts.Items {
			col, ok := elt.(*nodes.ColumnDef)
			if !ok {
				continue // Skip table-level constraints
			}

			var colExists bool
			if err := pool.QueryRow(
				ctx,
				"SELECT EXISTS(SELECT 1 FROM information_schema.columns WHERE table_schema = $1 AND table_name = $2 AND column_name = $3)",
				schema, tableName, col.Colname,
			).Scan(&colExists); err != nil || colExists {
				continue
			}

			// Extract raw column SQL from original text using AST location
			colDef := extractColumnSQL(sql, col.Location)
			if colDef == "" {
				continue
			}

			_, err := pool.Exec(ctx, fmt.Sprintf("ALTER TABLE %s.%s ADD COLUMN %s", schema, tableName, colDef))
			if err != nil {
				log.Warn().Err(err).Str("schema", schema).Str("table", tableName).Str("column", col.Colname).Msg("Failed to add missing column")
			} else {
				log.Info().Str("schema", schema).Str("table", tableName).Str("column", col.Colname).Msg("Added missing column for upgrade")
			}
		}
	}

	return nil
}

// extractColumnSQL extracts a column definition from the original SQL text
// using the byte offset provided by pgparser's AST location.
// listPartitionTables returns a set of table names that are partitions
// within the given schema. Columns added to the parent propagate automatically.
func (s *DeclarativeService) listPartitionTables(ctx context.Context, schema string, pool *pgxpool.Pool) map[string]bool {
	partitions := make(map[string]bool)
	rows, err := pool.Query(
		ctx,
		`SELECT c.relname FROM pg_class c
		 JOIN pg_namespace n ON n.oid = c.relnamespace
		 WHERE n.nspname = $1 AND c.relispartition`,
		schema,
	)
	if err != nil {
		return partitions
	}
	defer rows.Close()
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err == nil {
			partitions[name] = true
		}
	}
	return partitions
}

// extractAlterTableName extracts the table name from an ALTER TABLE statement.
// Returns empty string if the table name can't be determined.
func extractAlterTableName(sql string) string {
	upper := strings.ToUpper(sql)
	if !strings.HasPrefix(strings.TrimSpace(upper), "ALTER TABLE") {
		return ""
	}
	// "ALTER TABLE [IF EXISTS] <schema>.<name>" or "ALTER TABLE <name>"
	rest := strings.TrimSpace(sql[len("ALTER TABLE"):])
	if strings.HasPrefix(strings.ToUpper(rest), "IF EXISTS") {
		rest = strings.TrimSpace(rest[len("IF EXISTS"):])
	}
	// Handle schema-qualified names: "schema.name"
	rest = strings.TrimSpace(rest)
	fields := strings.Fields(rest)
	if len(fields) == 0 {
		return ""
	}
	name := fields[0]
	// Strip schema prefix if present
	if idx := strings.LastIndex(name, "."); idx >= 0 {
		name = name[idx+1:]
	}
	return strings.Trim(name, `"`)
}

// listPartitionedTables returns a set of table names that are partitioned parents
// (relkind = 'p') within the given schema.
func (s *DeclarativeService) listPartitionedTables(ctx context.Context, schema string, pool *pgxpool.Pool) map[string]bool {
	tables := make(map[string]bool)
	rows, err := pool.Query(
		ctx,
		`SELECT c.relname FROM pg_class c
		 JOIN pg_namespace n ON n.oid = c.relnamespace
		 WHERE n.nspname = $1 AND c.relkind = 'p'`,
		schema,
	)
	if err != nil {
		return tables
	}
	defer rows.Close()
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err == nil {
			tables[name] = true
		}
	}
	return tables
}

// extractIndexTableName extracts the table name from a CREATE INDEX statement.
func extractIndexTableName(sql string) string {
	// "CREATE [UNIQUE] INDEX [CONCURRENTLY] [IF NOT EXISTS] name ON [schema.]table"
	upper := strings.ToUpper(sql)
	// Find " ON " to locate the table name
	onIdx := strings.Index(upper, " ON ")
	if onIdx < 0 {
		return ""
	}
	rest := strings.TrimSpace(sql[onIdx+len(" ON "):])
	fields := strings.Fields(rest)
	if len(fields) == 0 {
		return ""
	}
	name := fields[0]
	// Strip schema prefix
	if idx := strings.LastIndex(name, "."); idx >= 0 {
		name = name[idx+1:]
	}
	return strings.TrimRight(strings.Trim(name, `"`), ";")
}

// extractTriggerTableName extracts the table name from a CREATE TRIGGER statement.
// Format: "CREATE [OR REPLACE] TRIGGER name {BEFORE|AFTER|INSTEAD OF} ... ON [schema.]table"
func extractTriggerTableName(sql string) string {
	upper := strings.ToUpper(sql)
	onIdx := strings.Index(upper, " ON ")
	if onIdx < 0 {
		return ""
	}
	rest := strings.TrimSpace(sql[onIdx+len(" ON "):])
	fields := strings.Fields(rest)
	if len(fields) == 0 {
		return ""
	}
	name := fields[0]
	if idx := strings.LastIndex(name, "."); idx >= 0 {
		name = name[idx+1:]
	}
	return strings.Trim(name, `"`)
}

// extractDropTriggerTableName extracts the table name from a DROP TRIGGER statement.
// Format: "DROP TRIGGER [IF EXISTS] name ON [schema.]table"
func extractDropTriggerTableName(sql string) string {
	upper := strings.ToUpper(sql)
	onIdx := strings.Index(upper, " ON ")
	if onIdx < 0 {
		return ""
	}
	rest := strings.TrimSpace(sql[onIdx+len(" ON "):])
	rest = strings.TrimRight(rest, ";")
	fields := strings.Fields(rest)
	if len(fields) == 0 {
		return ""
	}
	name := fields[0]
	if idx := strings.LastIndex(name, "."); idx >= 0 {
		name = name[idx+1:]
	}
	return strings.Trim(name, `"`)
}

func extractColumnSQL(sql string, loc nodes.ParseLoc) string {
	if loc < 0 || int(loc) >= len(sql) {
		return ""
	}

	// Scan backward past whitespace to find the true column start
	start := int(loc)
	for start > 0 && (sql[start-1] == ' ' || sql[start-1] == '\t' || sql[start-1] == '\n' || sql[start-1] == '\r') {
		start--
	}

	// Scan forward to find end: next top-level comma or closing paren
	depth := 0
	end := start
	for end < len(sql) {
		switch sql[end] {
		case '(':
			depth++
		case ')':
			if depth == 0 {
				return strings.TrimSpace(sql[start:end])
			}
			depth--
		case ',':
			if depth == 0 {
				return strings.TrimSpace(sql[start:end])
			}
		}
		end++
	}

	return strings.TrimSpace(sql[start:end])
}

// changePriority returns a priority value for ordering schema changes.
// Lower values execute first. Table structure changes must come before
// policy changes to avoid "column does not exist" errors during upgrades.
// We inspect the SQL content directly rather than relying on pgschema's
// path format, which may vary between versions.
func changePriority(c Change) int {
	sqlUpper := strings.ToUpper(strings.TrimSpace(c.SQL))
	switch {
	// Table structure changes: CREATE TABLE, ALTER TABLE ADD/DROP COLUMN, etc.
	case strings.HasPrefix(sqlUpper, "CREATE TABLE"),
		strings.HasPrefix(sqlUpper, "ALTER TABLE"):
		return 0
	// Indexes and sequences
	case strings.HasPrefix(sqlUpper, "CREATE INDEX"),
		strings.HasPrefix(sqlUpper, "CREATE UNIQUE INDEX"),
		strings.HasPrefix(sqlUpper, "DROP INDEX"),
		strings.HasPrefix(sqlUpper, "CREATE SEQUENCE"):
		return 1
	// Policies reference columns, must run after table changes
	case strings.HasPrefix(sqlUpper, "CREATE POLICY"),
		strings.HasPrefix(sqlUpper, "ALTER POLICY"),
		strings.HasPrefix(sqlUpper, "DROP POLICY"):
		return 3
	default:
		return 2 // Functions, triggers, comments, GRANTs, etc.
	}
}

// GetSchemaStatus returns information about the current schema state
func (s *DeclarativeService) GetSchemaStatus(ctx context.Context) (*SchemaStatus, error) {
	status := &SchemaStatus{
		SchemaFile: s.config.SchemaDir,
	}

	// Calculate fingerprint
	fingerprint, err := s.CalculateFingerprint()
	if err != nil {
		log.Warn().Err(err).Msg("Failed to calculate fingerprint")
	} else {
		status.SchemaFingerprint = fingerprint
	}

	// Get pending changes for all schemas
	totalChanges := 0
	for _, schema := range s.config.Schemas {
		plan, err := s.PlanForSchema(ctx, schema)
		if err != nil {
			log.Warn().Err(err).Str("schema", schema).Msg("Failed to plan schema")
			continue
		}
		totalChanges += len(plan.Changes)
		if hasDestructiveChanges(plan.Changes) {
			status.HasDestructiveChanges = true
		}
	}
	status.PendingChanges = totalChanges

	// Get last applied state from database
	if s.pool != nil {
		var dbFingerprint, source string
		var appliedAt time.Time
		err := s.pool.QueryRow(ctx, `
			SELECT schema_fingerprint, applied_at, source
			FROM platform.declarative_state
			ORDER BY id DESC
			LIMIT 1
		`).Scan(&dbFingerprint, &appliedAt, &source)
		if err == nil {
			status.LastAppliedFingerprint = dbFingerprint
			status.LastAppliedAt = appliedAt
			status.Source = source
		} else if err != pgx.ErrNoRows {
			log.Warn().Err(err).Msg("Failed to get declarative state from database")
		}
	}

	return status, nil
}

// SchemaStatus represents the current state of the declarative schema
type SchemaStatus struct {
	SchemaFile             string    `json:"schema_file"`
	SchemaFingerprint      string    `json:"schema_fingerprint"`
	LastAppliedFingerprint string    `json:"last_applied_fingerprint"`
	LastAppliedAt          time.Time `json:"last_applied_at"`
	Source                 string    `json:"source"`
	PendingChanges         int       `json:"pending_changes"`
	HasDestructiveChanges  bool      `json:"has_destructive_changes"`
}
