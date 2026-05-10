package tenantdb

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog/log"

	"github.com/nimbleflux/fluxbase/internal/database/bootstrap"
	"github.com/nimbleflux/fluxbase/internal/database/schema"
)

var (
	ErrMaxTenantsReached   = errors.New("maximum number of tenants reached")
	ErrCannotDeleteDefault = errors.New("cannot delete the default tenant")
	ErrTenantNotActive     = errors.New("tenant is not in active state")
)

// StorageQuerier covers the storage methods Manager calls.
type StorageQuerier interface {
	CountTenants(ctx context.Context) (int, error)
	CreateTenant(ctx context.Context, tenant *Tenant) error
	GetTenant(ctx context.Context, id string) (*Tenant, error)
	GetAllActiveTenants(ctx context.Context) ([]Tenant, error)
	GetDeletedTenants(ctx context.Context) ([]Tenant, error)
	UpdateTenantStatus(ctx context.Context, id string, status TenantStatus) error
	UpdateTenantDBName(ctx context.Context, id string, dbName string) error
	SoftDeleteTenant(ctx context.Context, id string) error
	HardDeleteTenant(ctx context.Context, id string) error
	RecoverTenant(ctx context.Context, id string) error
	CleanupTenantData(ctx context.Context, tenantID string) error
}

type Manager struct {
	storage        StorageQuerier
	config         Config
	adminPool      *pgxpool.Pool
	router         *Router
	dbURL          string
	adminDBURL     string
	fdwConfig      *FDWConfig
	declarative    *DeclarativeService
	declarativeCfg DeclarativeConfig
}

func NewManager(
	storage *Repository,
	config Config,
	adminPool *pgxpool.Pool,
	dbURL string,
) *Manager {
	return &Manager{
		storage:   storage,
		config:    config,
		adminPool: adminPool,
		dbURL:     dbURL,
	}
}

// SetDeclarativeService sets the declarative schema service for tenant databases
func (m *Manager) SetDeclarativeService(svc *DeclarativeService) {
	m.declarative = svc
}

// SetDeclarativeConfig sets the declarative schema configuration
func (m *Manager) SetDeclarativeConfig(cfg DeclarativeConfig) {
	m.declarativeCfg = cfg
}

// SetFDWConfig sets the FDW configuration for tenant databases
func (m *Manager) SetFDWConfig(cfg FDWConfig) {
	m.fdwConfig = &cfg
}

// SetAdminDBURL sets the admin database URL used for bootstrap operations
// that require elevated privileges (e.g., CREATE EXTENSION).
func (m *Manager) SetAdminDBURL(url string) {
	m.adminDBURL = url
}

func (m *Manager) SetRouter(router *Router) {
	m.router = router
}

func (m *Manager) CreateTenantDatabase(ctx context.Context, req CreateTenantRequest) (*Tenant, error) {
	if m.config.MaxTenants > 0 {
		count, err := m.storage.CountTenants(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to check tenant count: %w", err)
		}
		if count >= m.config.MaxTenants {
			return nil, ErrMaxTenantsReached
		}
	}

	// Determine database name based on mode
	var dbName string
	if req.DBMode == "existing" {
		if req.DBName == nil || *req.DBName == "" {
			return nil, fmt.Errorf("db_name is required when db_mode is 'existing'")
		}
		dbName = *req.DBName
	} else {
		dbName = fmt.Sprintf("%s%s", m.config.DatabasePrefix, req.Slug)
	}

	tenant := &Tenant{
		Slug:     req.Slug,
		Name:     req.Name,
		Status:   TenantStatusCreating,
		DBName:   &dbName,
		Metadata: req.Metadata,
	}
	if tenant.Metadata == nil {
		tenant.Metadata = make(map[string]any)
	}

	if err := m.storage.CreateTenant(ctx, tenant); err != nil {
		return nil, fmt.Errorf("failed to create tenant record: %w", err)
	}

	if req.DBMode != "existing" {
		_, err := m.adminPool.Exec(ctx, fmt.Sprintf(
			"CREATE DATABASE %s ENCODING 'UTF8'",
			quoteIdent(dbName),
		))
		if err != nil {
			if statusErr := m.storage.UpdateTenantStatus(ctx, tenant.ID, TenantStatusError); statusErr != nil {
				log.Warn().Err(statusErr).Str("tenant_id", tenant.ID).Msg("Failed to update tenant status to error")
			}
			if deleteErr := m.storage.HardDeleteTenant(ctx, tenant.ID); deleteErr != nil {
				log.Warn().Err(deleteErr).Str("tenant_id", tenant.ID).Msg("Failed to hard delete tenant record")
			}
			return nil, fmt.Errorf("failed to create database: %w", err)
		}

		log.Info().Str("tenant", req.Slug).Str("db", dbName).Msg("Created tenant database")
	}

	// Bootstrap tenant database (create schemas, roles, privileges)
	// Use admin URL if available (required for CREATE EXTENSION), otherwise app URL
	bootstrapBaseURL := m.adminDBURL
	if bootstrapBaseURL == "" {
		bootstrapBaseURL = m.dbURL
	}
	tenantDBURL := replaceDBName(bootstrapBaseURL, dbName)
	appUser := extractDBUser(bootstrapBaseURL)
	if err := bootstrap.RunBootstrapOnDB(ctx, tenantDBURL, appUser); err != nil {
		log.Warn().Err(err).Str("tenant", req.Slug).Msg("Failed to bootstrap tenant database")
		if statusErr := m.storage.UpdateTenantStatus(ctx, tenant.ID, TenantStatusError); statusErr != nil {
			log.Warn().Err(statusErr).Str("tenant_id", tenant.ID).Msg("Failed to update tenant status to error")
		}
		return nil, fmt.Errorf("failed to bootstrap tenant database: %w", err)
	}
	log.Info().Str("tenant", req.Slug).Msg("Bootstrapped tenant database")

	// Apply internal Fluxbase schemas (storage.buckets, functions.registry, etc.)
	// This creates tables, functions, types, and extensions in the tenant database.
	// When FDW is enabled, tables will be replaced by foreign imports in the next step.
	if err := m.applyInternalSchemas(ctx, dbName, m.fdwConfig != nil); err != nil {
		if statusErr := m.storage.UpdateTenantStatus(ctx, tenant.ID, TenantStatusError); statusErr != nil {
			log.Warn().Err(statusErr).Str("tenant_id", tenant.ID).Msg("Failed to update tenant status to error")
		}
		return nil, fmt.Errorf("failed to apply internal schemas: %w", err)
	}
	log.Info().Str("tenant", req.Slug).Msg("Applied internal schemas to tenant database")

	// Set up FDW so tenant can access all tables from the main database.
	// This replaces the local tables created by applyInternalSchemas with
	// foreign table imports, providing access to real data with RLS enforcement.
	if m.fdwConfig != nil && m.adminDBURL != "" {
		// Create per-tenant FDW role on main database for RLS enforcement
		fdwRole, roleErr := CreateFDWRole(ctx, m.adminPool, tenant.ID)
		if roleErr != nil {
			log.Warn().Err(roleErr).Str("tenant", req.Slug).Msg("Failed to create FDW role")
		} else {
			// Connect to tenant DB with admin privileges for FDW setup
			fdwURL := replaceDBName(m.adminDBURL, dbName)
			fdwPool, fdwPoolErr := pgxpool.New(ctx, fdwURL)
			if fdwPoolErr != nil {
				log.Warn().Err(fdwPoolErr).Str("tenant", req.Slug).Msg("Failed to create admin pool for FDW setup")
			} else {
				defer fdwPool.Close()

				if fdwErr := SetupFDW(ctx, fdwPool, *m.fdwConfig); fdwErr != nil {
					log.Warn().Err(fdwErr).Str("tenant", req.Slug).Msg("Failed to set up FDW for tenant database")
				} else {
					// Create user mapping for the app user with the per-tenant FDW role
					// so queries via the router pool use the tenant-scoped role
					appUser := extractDBUser(m.dbURL)
					if appUser != "" {
						if mapErr := CreateFDWUserMapping(ctx, fdwPool, appUser, fdwRole); mapErr != nil {
							log.Warn().Err(mapErr).Str("tenant", req.Slug).Msg("Failed to create app user FDW mapping")
						}
					}
					log.Info().Str("tenant", req.Slug).Msg("Set up FDW for tenant database")
				}
			}
		}
	}

	// Apply tenant-specific declarative schema if configured
	if m.declarative != nil && m.declarativeCfg.OnCreate {
		if m.declarative.HasSchemaFile(req.Slug) {
			if err := m.declarative.ApplyTenantSchema(ctx, tenant); err != nil {
				log.Warn().Err(err).Str("tenant", req.Slug).Msg("Failed to apply tenant declarative schema")
				// Don't fail the entire creation, just log the warning
			} else {
				log.Info().Str("tenant", req.Slug).Msg("Applied tenant declarative schema")
			}
		}
	}

	if err := m.storage.UpdateTenantStatus(ctx, tenant.ID, TenantStatusActive); err != nil {
		log.Error().Err(err).Str("tenant_id", tenant.ID).Msg("Failed to update tenant status")
	}
	tenant.Status = TenantStatusActive

	log.Info().Str("tenant_id", tenant.ID).Str("slug", req.Slug).Str("db", dbName).Msg("Tenant database created successfully")
	return tenant, nil
}

func (m *Manager) DeleteTenantDatabase(ctx context.Context, tenantID string) error {
	tenant, err := m.storage.GetTenant(ctx, tenantID)
	if err != nil {
		return fmt.Errorf("failed to get tenant: %w", err)
	}

	if tenant.IsDefault {
		return ErrCannotDeleteDefault
	}

	if err := m.storage.SoftDeleteTenant(ctx, tenantID); err != nil {
		return fmt.Errorf("failed to soft delete tenant: %w", err)
	}

	// Remove connection pool from cache so no new connections are created.
	if m.router != nil {
		m.router.RemovePool(tenantID)
	}

	log.Info().Str("tenant_id", tenantID).Str("slug", tenant.Slug).Msg("Tenant soft-deleted")
	return nil
}

func (m *Manager) HardDeleteTenantDatabase(ctx context.Context, tenantID string) error {
	tenant, err := m.storage.GetTenant(ctx, tenantID)
	if err != nil {
		return fmt.Errorf("failed to get tenant: %w", err)
	}

	if tenant.IsDefault {
		return ErrCannotDeleteDefault
	}

	// Soft delete first if not already done
	if tenant.DeletedAt == nil {
		if err := m.storage.SoftDeleteTenant(ctx, tenantID); err != nil {
			return fmt.Errorf("failed to soft delete tenant: %w", err)
		}
	}

	// Remove connection pool from cache
	if m.router != nil {
		m.router.RemovePool(tenantID)
	}

	// Cleanup tenant-related data from the main database.
	// Must happen before HardDeleteTenant because branching.branches has
	// RESTRICT FK and would block the tenant row deletion.
	if err := m.storage.CleanupTenantData(ctx, tenantID); err != nil {
		if statusErr := m.storage.UpdateTenantStatus(ctx, tenantID, TenantStatusError); statusErr != nil {
			log.Warn().Err(statusErr).Str("tenant_id", tenantID).Msg("Failed to update tenant status to error")
		}
		return fmt.Errorf("failed to cleanup tenant data: %w", err)
	}

	// Drop the separate database if one exists.
	if tenant.DBName != nil {
		_, err = m.adminPool.Exec(ctx, fmt.Sprintf("DROP DATABASE IF EXISTS %s WITH (FORCE)", quoteIdent(*tenant.DBName)))
		if err != nil {
			if statusErr := m.storage.UpdateTenantStatus(ctx, tenantID, TenantStatusError); statusErr != nil {
				log.Warn().Err(statusErr).Str("tenant_id", tenantID).Msg("Failed to update tenant status to error")
			}
			return fmt.Errorf("failed to drop database: %w", err)
		}
	}

	// Drop the per-tenant FDW role from the main database
	DropFDWRole(ctx, m.adminPool, tenantID)

	if err := m.storage.HardDeleteTenant(ctx, tenantID); err != nil {
		log.Warn().Err(err).Str("tenant_id", tenantID).Msg("Failed to hard delete tenant record")
	}

	log.Info().Str("tenant_id", tenantID).Str("slug", tenant.Slug).Msg("Tenant hard-deleted")
	return nil
}

func (m *Manager) RecoverTenantDatabase(ctx context.Context, tenantID string) error {
	// Recovery only un-deletes the metadata row. The database (if it existed)
	// was already dropped during hard delete, so this is only useful when the
	// tenant was soft-deleted but not yet hard-deleted.
	if err := m.storage.RecoverTenant(ctx, tenantID); err != nil {
		return fmt.Errorf("failed to recover tenant: %w", err)
	}

	log.Info().Str("tenant_id", tenantID).Msg("Tenant recovered")
	return nil
}

func (m *Manager) ListDeletedTenants(ctx context.Context) ([]Tenant, error) {
	return m.storage.GetDeletedTenants(ctx)
}

func (m *Manager) MigrateTenant(ctx context.Context, tenantID string) error {
	tenant, err := m.storage.GetTenant(ctx, tenantID)
	if err != nil {
		return fmt.Errorf("failed to get tenant: %w", err)
	}

	if tenant.UsesMainDatabase() {
		return nil
	}

	if m.router == nil {
		return fmt.Errorf("router not initialized")
	}

	_, err = m.router.GetPool(tenantID)
	if err != nil {
		return fmt.Errorf("failed to get tenant pool: %w", err)
	}

	return m.runSystemMigrationsForDB(ctx, *tenant.DBName)
}

func (m *Manager) StartMigrationWorker(ctx context.Context) {
	if !m.config.Migrations.Background {
		return
	}

	ticker := time.NewTicker(m.config.Migrations.CheckInterval)
	defer ticker.Stop()

	log.Info().Dur("interval", m.config.Migrations.CheckInterval).Msg("Starting tenant migration worker")

	for {
		select {
		case <-ctx.Done():
			log.Info().Msg("Tenant migration worker stopped")
			return
		case <-ticker.C:
			m.migrateAllTenants(ctx)
		}
	}
}

func (m *Manager) migrateAllTenants(ctx context.Context) {
	tenants, err := m.storage.GetAllActiveTenants(ctx)
	if err != nil {
		log.Error().Err(err).Msg("Failed to get tenants for migration")
		return
	}

	for _, tenant := range tenants {
		if tenant.UsesMainDatabase() {
			continue
		}

		if m.router == nil {
			continue
		}

		pool, err := m.router.GetPool(tenant.ID)
		if err != nil {
			log.Error().Err(err).Str("tenant", tenant.Slug).Msg("Failed to get pool for migration check")
			continue
		}

		pending, err := m.hasPendingMigrations(ctx, pool)
		if err != nil {
			log.Debug().Err(err).Str("tenant", tenant.Slug).Msg("Failed to check pending migrations")
			continue
		}

		if pending {
			log.Info().Str("tenant", tenant.Slug).Msg("Migrating tenant database")
			if err := m.runSystemMigrationsForDB(ctx, *tenant.DBName); err != nil {
				log.Error().Err(err).Str("tenant", tenant.Slug).Msg("Tenant migration failed")
			} else {
				log.Info().Str("tenant", tenant.Slug).Msg("Tenant migration completed")
			}
		}
	}
}

func (m *Manager) hasPendingMigrations(ctx context.Context, pool *pgxpool.Pool) (bool, error) {
	const migrationsDir = "/migrations"
	if _, err := os.Stat(migrationsDir); os.IsNotExist(err) {
		return false, nil
	}

	var currentVersion int
	if err := pool.QueryRow(ctx, `
		SELECT COALESCE(MAX(version), 0)::int FROM platform.fluxbase_migrations
	`).Scan(&currentVersion); err != nil {
		return true, fmt.Errorf("failed to query migration version: %w", err)
	}

	entries, err := os.ReadDir(migrationsDir)
	if err != nil {
		return true, fmt.Errorf("failed to read migrations directory: %w", err)
	}

	var latestVersion int
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if !strings.HasSuffix(entry.Name(), ".up.sql") {
			continue
		}
		prefix := strings.SplitN(entry.Name(), "_", 2)[0]
		v, parseErr := strconv.Atoi(prefix)
		if parseErr != nil {
			continue
		}
		if v > latestVersion {
			latestVersion = v
		}
	}

	if latestVersion == 0 {
		return false, nil
	}

	return latestVersion > currentVersion, nil
}

func (m *Manager) runSystemMigrationsForDB(ctx context.Context, dbName string) error {
	const migrationsDir = "/migrations"
	if _, err := os.Stat(migrationsDir); os.IsNotExist(err) {
		return nil
	}

	dbURL := replaceDBName(m.dbURL, dbName)

	migrator, err := migrate.New("file://"+migrationsDir, dbURL)
	if err != nil {
		return fmt.Errorf("failed to create migrator: %w", err)
	}
	defer func() {
		if _, closeErr := migrator.Close(); closeErr != nil {
			log.Warn().Err(closeErr).Str("db", dbName).Msg("Failed to close migrator")
		}
	}()

	if err := migrator.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("migration failed: %w", err)
	}

	return nil
}

// applyInternalSchemas applies Fluxbase internal schema SQL files (storage.sql,
// auth.sql, etc.) to a tenant database. These create the tables, indexes,
// functions, and policies needed for all features to work in tenant databases.
// The SQL is transformed with MakeSQLIdempotent so re-application is safe.
func (m *Manager) applyInternalSchemas(ctx context.Context, dbName string, fdwEnabled bool) error {
	// Extract embedded schemas to a temp directory
	schemaDir, err := schema.ExtractSchemas()
	if err != nil {
		return fmt.Errorf("failed to extract embedded schemas: %w", err)
	}
	defer func() {
		if removeErr := os.RemoveAll(schemaDir); removeErr != nil {
			log.Warn().Err(removeErr).Msg("Failed to remove temp schema directory")
		}
	}()

	// Determine app user from the dbURL for {{APP_USER}} substitution
	appUser := extractDBUser(m.dbURL)

	// Schemas must be applied in dependency order (platform first, then auth, etc.)
	schemaOrder := []string{
		"platform", "auth", "storage", "jobs", "functions", "realtime",
		"ai", "rpc", "branching", "logging", "mcp",
	}

	dbURL := replaceDBName(m.dbURL, dbName)
	// Use admin URL if available (required for CREATE TABLE in system schemas)
	if m.adminDBURL != "" {
		dbURL = replaceDBName(m.adminDBURL, dbName)
	}
	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		return fmt.Errorf("failed to connect to tenant database %s: %w", dbName, err)
	}
	defer pool.Close()

	for _, schemaName := range schemaOrder {
		schemaFile := filepath.Join(schemaDir, schemaName+".sql")
		data, err := os.ReadFile(schemaFile)
		if err != nil {
			if os.IsNotExist(err) {
				continue // Skip missing schema files
			}
			return fmt.Errorf("failed to read schema file %s: %w", schemaFile, err)
		}

		sql := string(data)
		if appUser != "" {
			var subErr error
			sql, subErr = bootstrap.SubstituteAppUser(sql, appUser)
			if subErr != nil {
				return fmt.Errorf("invalid app user for schema %s: %w", schemaName, subErr)
			}
		}

		sql = schema.MakeSQLIdempotent(sql)

		if _, err := pool.Exec(ctx, sql); err != nil {
			return fmt.Errorf("failed to apply schema %s to tenant database %s: %w", schemaName, dbName, err)
		}
	}

	// Apply cross-schema policies.
	// Skip post-schema-fks.sql when FDW is enabled because FKs cannot
	// reference foreign tables — they're already enforced on the main DB side.
	extraFiles := []string{"post-schema.sql"}
	if !fdwEnabled {
		extraFiles = append([]string{"post-schema-fks.sql"}, extraFiles...)
	}
	for _, extraFile := range extraFiles {
		extraPath := filepath.Join(schemaDir, extraFile)
		data, err := os.ReadFile(extraPath)
		if err != nil {
			continue
		}
		sql := string(data)
		if appUser != "" {
			var subErr error
			sql, subErr = bootstrap.SubstituteAppUser(sql, appUser)
			if subErr != nil {
				return fmt.Errorf("invalid app user for %s: %w", extraFile, subErr)
			}
		}
		sql = schema.MakeSQLIdempotent(sql)
		if _, err := pool.Exec(ctx, sql); err != nil {
			return fmt.Errorf("failed to apply %s to tenant database %s: %w", extraFile, dbName, err)
		}
	}

	return nil
}

// RepairTenant re-runs schema application and FDW setup for an existing tenant.
// This is useful when a tenant was partially created due to errors.
func (m *Manager) RepairTenant(ctx context.Context, tenant *Tenant) error {
	if tenant.UsesMainDatabase() {
		return fmt.Errorf("cannot repair default tenant (uses main database)")
	}

	dbName := *tenant.DBName
	log.Info().Str("tenant_id", tenant.ID).Str("slug", tenant.Slug).Str("db", dbName).Msg("Repairing tenant database")

	// Teardown existing FDW before bootstrap to prevent stale foreign tables
	// from triggering broken FDW connections during bootstrap DO blocks.
	bootstrapBaseURL := m.adminDBURL
	if bootstrapBaseURL == "" {
		bootstrapBaseURL = m.dbURL
	}
	tenantDBURL := replaceDBName(bootstrapBaseURL, dbName)
	appUser := extractDBUser(bootstrapBaseURL)

	if m.fdwConfig != nil {
		fdwURL := replaceDBName(bootstrapBaseURL, dbName)
		cleanupPool, poolErr := pgxpool.New(ctx, fdwURL)
		if poolErr == nil {
			if teardownErr := TeardownFDW(ctx, cleanupPool); teardownErr != nil {
				log.Warn().Err(teardownErr).Str("tenant", tenant.Slug).Msg("Failed to teardown FDW before repair bootstrap")
			}
			cleanupPool.Close()
		}
	}

	// Re-run bootstrap
	if err := bootstrap.RunBootstrapOnDB(ctx, tenantDBURL, appUser); err != nil {
		return fmt.Errorf("failed to bootstrap tenant database: %w", err)
	}

	// Re-apply internal schemas
	if err := m.applyInternalSchemas(ctx, dbName, m.fdwConfig != nil); err != nil {
		return fmt.Errorf("failed to apply internal schemas: %w", err)
	}

	// Re-setup FDW if configured
	if m.fdwConfig != nil && m.adminDBURL != "" {
		fdwRole, roleErr := CreateFDWRole(ctx, m.adminPool, tenant.ID)
		if roleErr != nil {
			log.Warn().Err(roleErr).Str("tenant", tenant.Slug).Msg("Failed to create FDW role during repair")
		} else {
			fdwURL := replaceDBName(m.adminDBURL, dbName)
			fdwPool, fdwPoolErr := pgxpool.New(ctx, fdwURL)
			if fdwPoolErr != nil {
				log.Warn().Err(fdwPoolErr).Str("tenant", tenant.Slug).Msg("Failed to create admin pool for FDW repair")
			} else {
				defer fdwPool.Close()
				if fdwErr := SetupFDW(ctx, fdwPool, *m.fdwConfig); fdwErr != nil {
					log.Warn().Err(fdwErr).Str("tenant", tenant.Slug).Msg("Failed to repair FDW")
				} else {
					appUser := extractDBUser(m.dbURL)
					if appUser != "" {
						if mapErr := CreateFDWUserMapping(ctx, fdwPool, appUser, fdwRole); mapErr != nil {
							log.Warn().Err(mapErr).Str("tenant", tenant.Slug).Msg("Failed to repair FDW user mapping")
						}
					}
				}
			}
		}
	}

	// Ensure tenant is marked active
	if tenant.Status != TenantStatusActive {
		if err := m.storage.UpdateTenantStatus(ctx, tenant.ID, TenantStatusActive); err != nil {
			log.Warn().Err(err).Str("tenant_id", tenant.ID).Msg("Failed to update tenant status after repair")
		}
	}

	log.Info().Str("tenant_id", tenant.ID).Str("slug", tenant.Slug).Msg("Tenant database repaired successfully")
	return nil
}

func (m *Manager) GetRepository() *Repository {
	if s, ok := m.storage.(*Repository); ok {
		return s
	}
	return nil
}

func (m *Manager) GetRouter() *Router {
	return m.router
}

func (m *Manager) GetConfig() Config {
	return m.config
}

func (m *Manager) GetDeclarativeService() *DeclarativeService {
	return m.declarative
}

// ApplyDeclarativeSchemas applies declarative schemas to all tenants with schema files
// This is called on startup if configured
func (m *Manager) ApplyDeclarativeSchemas(ctx context.Context) error {
	if m.declarative == nil {
		return nil
	}

	tenants, err := m.storage.GetAllActiveTenants(ctx)
	if err != nil {
		return fmt.Errorf("failed to list tenants: %w", err)
	}

	for i := range tenants {
		if m.declarative.HasSchemaFile(tenants[i].Slug) {
			if err := m.declarative.ApplyTenantSchema(ctx, &tenants[i]); err != nil {
				log.Error().Err(err).Str("tenant", tenants[i].Slug).Msg("Failed to apply declarative schema")
				// Continue with other tenants
			}
		}
	}

	return nil
}

// ApplyTenantDeclarativeSchema applies the declarative schema for a specific tenant
func (m *Manager) ApplyTenantDeclarativeSchema(ctx context.Context, tenantID string) error {
	if m.declarative == nil {
		return fmt.Errorf("declarative schema service not configured")
	}

	tenant, err := m.storage.GetTenant(ctx, tenantID)
	if err != nil {
		return fmt.Errorf("failed to get tenant: %w", err)
	}

	if !m.declarative.HasSchemaFile(tenant.Slug) {
		return fmt.Errorf("no declarative schema file found for tenant %s", tenant.Slug)
	}

	return m.declarative.ApplyTenantSchema(ctx, tenant)
}

// GetTenantSchemaStatus returns the schema status for a specific tenant
func (m *Manager) GetTenantSchemaStatus(ctx context.Context, tenantID string) (*TenantSchemaStatus, error) {
	if m.declarative == nil {
		return nil, fmt.Errorf("declarative schema service not configured")
	}

	tenant, err := m.storage.GetTenant(ctx, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to get tenant: %w", err)
	}

	return m.declarative.GetTenantSchemaStatus(ctx, tenant)
}

// UpgradeTenantFDW sets up FDW for an existing tenant database.
// This is used to migrate tenant databases that were created before FDW was enabled.
// It creates the per-tenant FDW role, applies schemas if needed, and imports foreign tables.
func (m *Manager) UpgradeTenantFDW(ctx context.Context, tenantID string) error {
	if m.fdwConfig == nil || m.adminDBURL == "" {
		return fmt.Errorf("FDW not configured")
	}

	tenant, err := m.storage.GetTenant(ctx, tenantID)
	if err != nil {
		return fmt.Errorf("failed to get tenant: %w", err)
	}

	if tenant.UsesMainDatabase() || tenant.DBName == nil {
		return nil // Skip default tenants
	}

	// Create per-tenant FDW role on main database
	fdwRole, roleErr := CreateFDWRole(ctx, m.adminPool, tenant.ID)
	if roleErr != nil {
		return fmt.Errorf("failed to create FDW role: %w", roleErr)
	}

	// Connect to tenant DB with admin privileges
	fdwURL := replaceDBName(m.adminDBURL, *tenant.DBName)
	fdwPool, fdwPoolErr := pgxpool.New(ctx, fdwURL)
	if fdwPoolErr != nil {
		return fmt.Errorf("failed to connect to tenant database: %w", fdwPoolErr)
	}
	defer fdwPool.Close()

	// Import all schema tables via FDW
	if fdwErr := SetupFDW(ctx, fdwPool, *m.fdwConfig); fdwErr != nil {
		return fmt.Errorf("failed to set up FDW: %w", fdwErr)
	}

	// Create user mapping for the app user with the per-tenant FDW role
	appUser := extractDBUser(m.dbURL)
	if appUser != "" {
		if mapErr := CreateFDWUserMapping(ctx, fdwPool, appUser, fdwRole); mapErr != nil {
			log.Warn().Err(mapErr).Str("tenant", tenant.Slug).Msg("Failed to create app user FDW mapping during upgrade")
		}
	}

	log.Info().Str("tenant_id", tenantID).Str("slug", tenant.Slug).Msg("Upgraded tenant database with FDW")
	return nil
}

// UpgradeAllTenantsFDW upgrades all existing tenant databases with FDW.
// Called on startup when FDW is enabled to migrate pre-existing tenants.
func (m *Manager) UpgradeAllTenantsFDW(ctx context.Context) {
	if m.fdwConfig == nil || m.adminDBURL == "" {
		return
	}

	tenants, err := m.storage.GetAllActiveTenants(ctx)
	if err != nil {
		log.Error().Err(err).Msg("Failed to list tenants for FDW upgrade")
		return
	}

	for i := range tenants {
		if tenants[i].UsesMainDatabase() {
			continue
		}

		if err := m.UpgradeTenantFDW(ctx, tenants[i].ID); err != nil {
			log.Error().Err(err).Str("tenant", tenants[i].Slug).Msg("Failed to upgrade tenant with FDW")
		}
	}
}

// RepairFDWForBranch repairs FDW configuration in a branch database that was
// cloned from a tenant database. After cloning via TEMPLATE, the branch database
// inherits foreign table definitions but the user mapping may reference stale
// credentials. This method recreates the user mapping with the tenant's FDW role.
func (m *Manager) RepairFDWForBranch(ctx context.Context, branchDBURL string, tenantID uuid.UUID) error {
	if m.fdwConfig == nil {
		log.Debug().Msg("FDW not configured, skipping FDW repair for branch")
		return nil
	}

	tenantPool, err := m.router.GetPool(tenantID.String())
	if err != nil {
		return fmt.Errorf("failed to get tenant pool for FDW role lookup: %w", err)
	}

	fdwRole, err := GetFDWRoleForTenant(ctx, tenantPool, tenantID.String())
	if err != nil {
		return fmt.Errorf("failed to get FDW role for tenant: %w", err)
	}

	branchPool, err := pgxpool.New(ctx, branchDBURL)
	if err != nil {
		return fmt.Errorf("failed to connect to branch database: %w", err)
	}
	defer branchPool.Close()

	appUser := extractDBUser(branchDBURL)
	if appUser == "" {
		appUser = "fluxbase"
	}

	if err := CreateFDWUserMapping(ctx, branchPool, appUser, fdwRole); err != nil {
		return fmt.Errorf("failed to create FDW user mapping in branch: %w", err)
	}

	log.Info().
		Str("tenant_id", tenantID.String()).
		Str("fdw_role", fdwRole.RoleName).
		Msg("Repaired FDW user mapping in branch database")

	return nil
}

// replaceDBName constructs a database URL for a specific database name
// by parsing the base URL and replacing the path component.
func replaceDBName(baseDBURL, dbName string) string {
	u, err := url.Parse(baseDBURL)
	if err != nil {
		return baseDBURL + dbName // fallback
	}
	u.Path = "/" + dbName
	return u.String()
}

// extractDBUser returns the username from a database URL, or empty string on error.
func extractDBUser(dbURL string) string {
	u, err := url.Parse(dbURL)
	if err != nil {
		return ""
	}
	return u.User.Username()
}
