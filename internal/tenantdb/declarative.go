package tenantdb

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog/log"
)

// Plan represents a migration plan from pgschema
type Plan struct {
	Changes    []Change `json:"changes"`
	DDL        string   `json:"ddl"`
	HasChanges bool     `json:"has_changes"`
	Duration   time.Duration
}

// Change represents a single schema change
type Change struct {
	Type        string `json:"type"`        // create, alter, drop
	ObjectType  string `json:"object_type"` // table, index, function, etc.
	Schema      string `json:"schema"`
	Name        string `json:"name"`
	SQL         string `json:"sql"`
	Destructive bool   `json:"destructive"`
}

// DeclarativeService manages tenant-specific declarative schemas using pgschema
// Each tenant can have their own schema files in {SchemaDir}/{tenant-slug}/public.sql
type DeclarativeService struct {
	config       DeclarativeConfig
	pgschemaPath string
	dbHost       string
	dbPort       int
	dbUser       string
	dbPassword   string
	adminPool    *pgxpool.Pool
}

// DeclarativeConfig holds configuration for tenant declarative schemas
type DeclarativeConfig struct {
	Enabled          bool
	SchemaDir        string
	OnCreate         bool
	OnStartup        bool
	AllowDestructive bool
}

// TenantSchemaStatus represents the state of a tenant's declarative schema
type TenantSchemaStatus struct {
	TenantID                string    `json:"tenant_id"`
	TenantSlug              string    `json:"tenant_slug"`
	SchemaFile              string    `json:"schema_file,omitempty"`
	SchemaFingerprint       string    `json:"schema_fingerprint,omitempty"`
	LastAppliedFingerprint  string    `json:"last_applied_fingerprint,omitempty"`
	LastAppliedAt           time.Time `json:"last_applied_at,omitempty"`
	HasPendingChanges       bool      `json:"has_pending_changes"`
	HasStoredSchema         bool      `json:"has_stored_schema"`
	StoredSchemaFingerprint string    `json:"stored_schema_fingerprint,omitempty"`
}

// NewDeclarativeService creates a new tenant declarative schema service
func NewDeclarativeService(
	config DeclarativeConfig,
	pgschemaPath string,
	dbHost string,
	dbPort int,
	dbUser, dbPassword string,
	adminPool *pgxpool.Pool,
) *DeclarativeService {
	return &DeclarativeService{
		config:       config,
		pgschemaPath: pgschemaPath,
		dbHost:       dbHost,
		dbPort:       dbPort,
		dbUser:       dbUser,
		dbPassword:   dbPassword,
		adminPool:    adminPool,
	}
}

// GetSchemaFilePath returns the path to the schema file for a tenant
func (s *DeclarativeService) GetSchemaFilePath(tenantSlug string) string {
	return filepath.Join(s.config.SchemaDir, tenantSlug, "public.sql")
}

// HasSchemaFile checks if a tenant has a declarative schema file
func (s *DeclarativeService) HasSchemaFile(tenantSlug string) bool {
	schemaFile := s.GetSchemaFilePath(tenantSlug)
	_, err := os.Stat(schemaFile)
	return err == nil
}

// CalculateFingerprint computes a SHA256 hash of a tenant's schema file
func (s *DeclarativeService) CalculateFingerprint(tenantSlug string) (string, error) {
	schemaFile := s.GetSchemaFilePath(tenantSlug)
	content, err := os.ReadFile(schemaFile)
	if err != nil {
		return "", fmt.Errorf("failed to read schema file: %w", err)
	}
	hash := sha256.Sum256(content)
	return hex.EncodeToString(hash[:]), nil
}

// PlanForTenant generates a migration plan for a tenant's public schema using pgschema
func (s *DeclarativeService) PlanForTenant(ctx context.Context, tenant *Tenant) (*Plan, error) {
	if tenant.UsesMainDatabase() {
		return nil, fmt.Errorf("cannot plan schema for tenant using main database")
	}

	schemaFile := s.GetSchemaFilePath(tenant.Slug)
	if _, err := os.Stat(schemaFile); os.IsNotExist(err) {
		return &Plan{}, nil // No schema file = no changes
	}

	args := []string{
		"plan",
		"--host", s.dbHost,
		"--port", fmt.Sprintf("%d", s.dbPort),
		"--user", s.dbUser,
		"--db", *tenant.DBName,
		"--file", schemaFile,
		"--schema", "public",
		"--output-json", "stdout",
		"--plan-host", s.dbHost,
		"--plan-port", fmt.Sprintf("%d", s.dbPort),
		"--plan-user", s.dbUser,
		"--plan-password", s.dbPassword,
		"--plan-db", *tenant.DBName,
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
			return nil, fmt.Errorf("pgschema plan failed for tenant %s: %w: %s", tenant.Slug, err, string(exitErr.Stderr))
		}
		return nil, fmt.Errorf("pgschema plan failed for tenant %s: %w: %s", tenant.Slug, err, string(output))
	}

	var plan Plan
	if err := json.Unmarshal(output, &plan); err != nil {
		return nil, fmt.Errorf("failed to parse plan for tenant %s: %w", tenant.Slug, err)
	}

	plan.Duration = time.Since(start)
	return &plan, nil
}

// ApplyTenantSchema applies a tenant's declarative schema using pgschema
func (s *DeclarativeService) ApplyTenantSchema(ctx context.Context, tenant *Tenant) error {
	if !s.config.Enabled {
		return nil
	}

	if tenant.UsesMainDatabase() {
		log.Debug().Str("tenant", tenant.Slug).Msg("Tenant uses main database, skipping declarative schema")
		return nil
	}

	schemaFile := s.GetSchemaFilePath(tenant.Slug)
	if _, err := os.Stat(schemaFile); os.IsNotExist(err) {
		log.Debug().Str("tenant", tenant.Slug).Msg("No declarative schema file found for tenant")
		return nil
	}

	log.Info().
		Str("tenant", tenant.Slug).
		Str("db", *tenant.DBName).
		Str("schema_file", schemaFile).
		Msg("Applying tenant declarative schema with pgschema")

	// Generate plan
	plan, err := s.PlanForTenant(ctx, tenant)
	if err != nil {
		return err
	}

	if len(plan.Changes) == 0 {
		log.Info().Str("tenant", tenant.Slug).Msg("No schema changes to apply")
		return nil
	}

	// Write plan to temp file
	planFile, err := s.writePlanToTemp(plan)
	if err != nil {
		return err
	}
	defer func() { _ = os.Remove(planFile) }()

	// Apply using pgschema
	args := []string{
		"apply",
		"--host", s.dbHost,
		"--port", fmt.Sprintf("%d", s.dbPort),
		"--user", s.dbUser,
		"--db", *tenant.DBName,
		"--file", schemaFile,
		"--schema", "public",
		"--plan", planFile,
		"--auto-approve",
	}

	cmd := exec.CommandContext(ctx, s.pgschemaPath, args...)
	cmd.Env = append(os.Environ(), fmt.Sprintf("PGPASSWORD=%s", s.dbPassword))

	start := time.Now()
	if _, err := cmd.Output(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return fmt.Errorf("pgschema apply failed for tenant %s: %w: %s", tenant.Slug, err, string(exitErr.Stderr))
		}
		return fmt.Errorf("pgschema apply failed for tenant %s: %w", tenant.Slug, err)
	}

	// Record the applied state (for fingerprint-based change detection)
	fingerprint, err := s.CalculateFingerprint(tenant.Slug)
	if err != nil {
		log.Warn().Err(err).Str("tenant", tenant.Slug).Msg("Failed to calculate fingerprint after apply")
	} else {
		if err := s.recordApply(ctx, tenant, fingerprint); err != nil {
			log.Warn().Err(err).Str("tenant", tenant.Slug).Msg("Failed to record schema state")
		}
	}

	log.Info().
		Str("tenant", tenant.Slug).
		Int("changes", len(plan.Changes)).
		Dur("duration", time.Since(start)).
		Msg("Tenant declarative schema applied successfully")

	return nil
}

// ApplyAllTenantSchemas applies declarative schemas to all active tenants
func (s *DeclarativeService) ApplyAllTenantSchemas(ctx context.Context, storage *Repository) error {
	if !s.config.Enabled {
		return nil
	}

	tenants, err := storage.GetAllActiveTenants(ctx)
	if err != nil {
		return fmt.Errorf("failed to list tenants: %w", err)
	}

	for i := range tenants {
		if s.HasSchemaFile(tenants[i].Slug) {
			if err := s.ApplyTenantSchema(ctx, &tenants[i]); err != nil {
				log.Error().Err(err).Str("tenant", tenants[i].Slug).Msg("Failed to apply tenant schema")
				// Continue with other tenants
			}
		}
	}

	return nil
}

// GetTenantSchemaStatus returns the schema status for a tenant
func (s *DeclarativeService) GetTenantSchemaStatus(ctx context.Context, tenant *Tenant) (*TenantSchemaStatus, error) {
	status := &TenantSchemaStatus{
		TenantID:   tenant.ID,
		TenantSlug: tenant.Slug,
	}

	schemaFile := s.GetSchemaFilePath(tenant.Slug)
	status.SchemaFile = schemaFile

	if _, err := os.Stat(schemaFile); os.IsNotExist(err) {
		return status, nil
	}

	// Calculate current fingerprint
	fingerprint, err := s.CalculateFingerprint(tenant.Slug)
	if err != nil {
		return nil, err
	}
	status.SchemaFingerprint = fingerprint

	// If tenant uses main database, no state to check
	if tenant.UsesMainDatabase() {
		return status, nil
	}

	// Check if there are pending changes using pgschema plan
	plan, err := s.PlanForTenant(ctx, tenant)
	if err != nil {
		log.Warn().Err(err).Str("tenant", tenant.Slug).Msg("Failed to get plan for status check")
		status.HasPendingChanges = true // Assume pending on error
		return status, nil
	}

	status.HasPendingChanges = len(plan.Changes) > 0

	// Get last applied state from tenant database
	connStr := fmt.Sprintf("postgresql://%s:%s@%s:%d/%s", s.dbUser, s.dbPassword, s.dbHost, s.dbPort, *tenant.DBName)
	pool, err := pgxpool.New(ctx, connStr)
	if err != nil {
		return status, nil // Can't connect, just return what we have
	}
	defer pool.Close()

	var lastFingerprint string
	var appliedAt time.Time
	err = pool.QueryRow(ctx, `
		SELECT schema_fingerprint, applied_at
		FROM platform.tenant_declarative_state
		WHERE tenant_id = $1
		ORDER BY applied_at DESC
		LIMIT 1
	`, tenant.ID).Scan(&lastFingerprint, &appliedAt)

	if err == pgx.ErrNoRows {
		return status, nil
	}
	if err != nil {
		return status, nil
	}

	status.LastAppliedFingerprint = lastFingerprint
	status.LastAppliedAt = appliedAt

	return status, nil
}

// writePlanToTemp writes the plan JSON to a temporary file
func (s *DeclarativeService) writePlanToTemp(plan *Plan) (string, error) {
	tmpFile, err := os.CreateTemp("", "pgschema-plan-*.json")
	if err != nil {
		return "", fmt.Errorf("failed to create temp file: %w", err)
	}
	defer func() { _ = tmpFile.Close() }()

	encoder := json.NewEncoder(tmpFile)
	if err := encoder.Encode(plan); err != nil {
		_ = os.Remove(tmpFile.Name())
		return "", fmt.Errorf("failed to write plan: %w", err)
	}

	return tmpFile.Name(), nil
}

// recordApply records that a schema has been applied to a tenant database
func (s *DeclarativeService) recordApply(ctx context.Context, tenant *Tenant, fingerprint string) error {
	connStr := fmt.Sprintf("postgresql://%s:%s@%s:%d/%s", s.dbUser, s.dbPassword, s.dbHost, s.dbPort, *tenant.DBName)
	pool, err := pgxpool.New(ctx, connStr)
	if err != nil {
		return fmt.Errorf("failed to connect to tenant database: %w", err)
	}
	defer pool.Close()

	// Ensure state table exists
	if _, err := pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS platform.tenant_declarative_state (
			id SERIAL PRIMARY KEY,
			tenant_id uuid NOT NULL,
			schema_fingerprint TEXT NOT NULL,
			applied_at TIMESTAMPTZ DEFAULT NOW(),
			applied_by TEXT DEFAULT 'fluxbase'
		);
	`); err != nil {
		return fmt.Errorf("failed to ensure state table: %w", err)
	}

	_, err = pool.Exec(ctx, `
		INSERT INTO platform.tenant_declarative_state (tenant_id, schema_fingerprint, applied_by)
		VALUES ($1, $2, 'fluxbase')
	`, tenant.ID, fingerprint)

	return err
}

// StoreSchemaContent stores schema content for a tenant in the platform database
// This allows schemas to be managed via API instead of files
func (s *DeclarativeService) StoreSchemaContent(ctx context.Context, tenantSlug, schemaContent string) error {
	// Store in platform.tenant_schemas table (created if needed)
	_, err := s.adminPool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS platform.tenant_schemas (
			tenant_slug TEXT PRIMARY KEY,
			schema_content TEXT NOT NULL,
			schema_fingerprint TEXT NOT NULL,
			updated_at TIMESTAMPTZ DEFAULT now()
		)
	`)
	if err != nil {
		return fmt.Errorf("failed to ensure tenant_schemas table: %w", err)
	}

	fingerprint := sha256.Sum256([]byte(schemaContent))
	fingerprintStr := hex.EncodeToString(fingerprint[:])

	_, err = s.adminPool.Exec(ctx, `
		INSERT INTO platform.tenant_schemas (tenant_slug, schema_content, schema_fingerprint, updated_at)
		VALUES ($1, $2, $3, now())
		ON CONFLICT (tenant_slug) DO UPDATE SET
			schema_content = EXCLUDED.schema_content,
			schema_fingerprint = EXCLUDED.schema_fingerprint,
			updated_at = now()
	`, tenantSlug, schemaContent, fingerprintStr)

	return err
}

// GetStoredSchemaContent retrieves stored schema content for a tenant
func (s *DeclarativeService) GetStoredSchemaContent(ctx context.Context, tenantSlug string) (content string, fingerprint string, updatedAt time.Time, err error) {
	var schemaContent, schemaFingerprint string
	var schemaUpdatedAt time.Time
	err = s.adminPool.QueryRow(ctx, `
		SELECT schema_content, schema_fingerprint, updated_at
		FROM platform.tenant_schemas
		WHERE tenant_slug = $1
	`, tenantSlug).Scan(&schemaContent, &schemaFingerprint, &schemaUpdatedAt)
	if err == pgx.ErrNoRows {
		return "", "", time.Time{}, nil
	}
	if err != nil {
		return "", "", time.Time{}, fmt.Errorf("failed to get stored schema: %w", err)
	}
	return schemaContent, schemaFingerprint, schemaUpdatedAt, nil
}

// HasStoredSchema checks if a tenant has a schema stored in the database
func (s *DeclarativeService) HasStoredSchema(ctx context.Context, tenantSlug string) bool {
	var exists bool
	err := s.adminPool.QueryRow(ctx, `
		SELECT EXISTS(SELECT 1 FROM platform.tenant_schemas WHERE tenant_slug = $1)
	`, tenantSlug).Scan(&exists)
	return err == nil && exists
}

// ApplyTenantSchemaFromContent applies schema from raw SQL content
// This writes the content to a temp file and uses pgschema
func (s *DeclarativeService) ApplyTenantSchemaFromContent(ctx context.Context, tenant *Tenant, schemaContent string) error {
	if !s.config.Enabled {
		return fmt.Errorf("tenant declarative schemas are not enabled")
	}

	if tenant.UsesMainDatabase() {
		return fmt.Errorf("cannot apply declarative schema to tenant using main database")
	}

	if schemaContent == "" {
		return fmt.Errorf("schema content cannot be empty")
	}

	log.Info().
		Str("tenant", tenant.Slug).
		Str("db", *tenant.DBName).
		Int("content_len", len(schemaContent)).
		Msg("Applying tenant declarative schema from content")

	// Write to temp file
	tmpFile, err := os.CreateTemp("", "tenant-schema-*.sql")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	defer func() { _ = os.Remove(tmpFile.Name()) }()

	if _, err := tmpFile.WriteString(schemaContent); err != nil {
		return fmt.Errorf("failed to write schema content: %w", err)
	}
	if err := tmpFile.Close(); err != nil {
		return fmt.Errorf("failed to close temp file: %w", err)
	}

	// Plan using the temp file
	args := []string{
		"plan",
		"--host", s.dbHost,
		"--port", fmt.Sprintf("%d", s.dbPort),
		"--user", s.dbUser,
		"--db", *tenant.DBName,
		"--file", tmpFile.Name(),
		"--schema", "public",
		"--output-json", "stdout",
		"--plan-host", s.dbHost,
		"--plan-port", fmt.Sprintf("%d", s.dbPort),
		"--plan-user", s.dbUser,
		"--plan-password", s.dbPassword,
		"--plan-db", *tenant.DBName,
	}

	if s.config.AllowDestructive {
		args = append(args, "--allow-destructive")
	}

	cmd := exec.CommandContext(ctx, s.pgschemaPath, args...)
	cmd.Env = append(os.Environ(), fmt.Sprintf("PGPASSWORD=%s", s.dbPassword))

	output, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return fmt.Errorf("pgschema plan failed: %w: %s", err, string(exitErr.Stderr))
		}
		return fmt.Errorf("pgschema plan failed: %w", err)
	}

	var plan Plan
	if err := json.Unmarshal(output, &plan); err != nil {
		return fmt.Errorf("failed to parse plan: %w", err)
	}

	if len(plan.Changes) == 0 {
		log.Info().Str("tenant", tenant.Slug).Msg("No schema changes to apply")
		return nil
	}

	// Write plan to temp file
	planFile, err := s.writePlanToTemp(&plan)
	if err != nil {
		return err
	}
	defer func() { _ = os.Remove(planFile) }()

	// Apply
	applyArgs := []string{
		"apply",
		"--host", s.dbHost,
		"--port", fmt.Sprintf("%d", s.dbPort),
		"--user", s.dbUser,
		"--db", *tenant.DBName,
		"--file", tmpFile.Name(),
		"--schema", "public",
		"--plan", planFile,
		"--auto-approve",
	}

	applyCmd := exec.CommandContext(ctx, s.pgschemaPath, applyArgs...)
	applyCmd.Env = append(os.Environ(), fmt.Sprintf("PGPASSWORD=%s", s.dbPassword))

	if _, err := applyCmd.Output(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return fmt.Errorf("pgschema apply failed: %w: %s", err, string(exitErr.Stderr))
		}
		return fmt.Errorf("pgschema apply failed: %w", err)
	}

	// Record the applied state
	fingerprint := sha256.Sum256([]byte(schemaContent))
	fingerprintStr := hex.EncodeToString(fingerprint[:])
	if err := s.recordApply(ctx, tenant, fingerprintStr); err != nil {
		log.Warn().Err(err).Msg("Failed to record schema state")
	}

	log.Info().
		Str("tenant", tenant.Slug).
		Str("fingerprint", fingerprintStr[:12]).
		Int("changes", len(plan.Changes)).
		Msg("Tenant declarative schema applied successfully")

	return nil
}

// DeleteStoredSchema removes stored schema content for a tenant
func (s *DeclarativeService) DeleteStoredSchema(ctx context.Context, tenantSlug string) error {
	_, err := s.adminPool.Exec(ctx, `
		DELETE FROM platform.tenant_schemas WHERE tenant_slug = $1
	`, tenantSlug)
	return err
}

// DumpTenantSchema exports the current schema of a tenant database to a file
func (s *DeclarativeService) DumpTenantSchema(ctx context.Context, tenant *Tenant, outputPath string) error {
	if tenant.UsesMainDatabase() {
		return fmt.Errorf("cannot dump schema for tenant using main database")
	}

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
		"--db", *tenant.DBName,
		"--schema", "public",
	}

	cmd := exec.CommandContext(ctx, s.pgschemaPath, args...)
	cmd.Env = append(os.Environ(), fmt.Sprintf("PGPASSWORD=%s", s.dbPassword))
	cmd.Stdout = f

	if err := cmd.Run(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return fmt.Errorf("pgschema dump failed: %w: %s", err, string(exitErr.Stderr))
		}
		return fmt.Errorf("pgschema dump failed: %w", err)
	}

	log.Info().Str("tenant", tenant.Slug).Str("path", outputPath).Msg("Tenant schema dumped successfully")
	return nil
}
