package tenantdb

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/rs/zerolog/log"
)

var (
	ErrTenantNotFound      = errors.New("tenant not found")
	ErrTenantAlreadyExists = errors.New("tenant already exists")
	ErrNoDefaultTenant     = errors.New("no default tenant found")
	ErrTenantNotDeleted    = errors.New("tenant is not in a deleted state")
)

// DB abstracts database operations for testability.
// *pgxpool.Pool satisfies this interface.
type DB interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
}

type Repository struct {
	db DB
}

func NewRepository(db DB) *Repository {
	return &Repository{db: db}
}

func (s *Repository) GetTenant(ctx context.Context, id string) (*Tenant, error) {
	query := `
		SELECT id, slug, name, db_name, is_default, status, metadata, created_at, updated_at, deleted_at
		FROM platform.tenants
		WHERE id = $1::uuid AND deleted_at IS NULL
	`

	var tenant Tenant
	var dbName *string
	var metadataBytes []byte

	err := s.db.QueryRow(ctx, query, id).Scan(
		&tenant.ID,
		&tenant.Slug,
		&tenant.Name,
		&dbName,
		&tenant.IsDefault,
		&tenant.Status,
		&metadataBytes,
		&tenant.CreatedAt,
		&tenant.UpdatedAt,
		&tenant.DeletedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrTenantNotFound
		}
		return nil, fmt.Errorf("failed to get tenant: %w", err)
	}

	tenant.DBName = dbName
	if metadataBytes != nil {
		if err := json.Unmarshal(metadataBytes, &tenant.Metadata); err != nil {
			tenant.Metadata = make(map[string]any)
		}
	}

	return &tenant, nil
}

func (s *Repository) GetTenantBySlug(ctx context.Context, slug string) (*Tenant, error) {
	query := `
		SELECT id, slug, name, db_name, is_default, status, metadata, created_at, updated_at, deleted_at
		FROM platform.tenants
		WHERE slug = $1 AND deleted_at IS NULL
	`

	var tenant Tenant
	var dbName *string
	var metadataBytes []byte

	err := s.db.QueryRow(ctx, query, slug).Scan(
		&tenant.ID,
		&tenant.Slug,
		&tenant.Name,
		&dbName,
		&tenant.IsDefault,
		&tenant.Status,
		&metadataBytes,
		&tenant.CreatedAt,
		&tenant.UpdatedAt,
		&tenant.DeletedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrTenantNotFound
		}
		return nil, fmt.Errorf("failed to get tenant by slug: %w", err)
	}

	tenant.DBName = dbName
	if metadataBytes != nil {
		if err := json.Unmarshal(metadataBytes, &tenant.Metadata); err != nil {
			tenant.Metadata = make(map[string]any)
		}
	}

	return &tenant, nil
}

func (s *Repository) GetDefaultTenant(ctx context.Context) (*Tenant, error) {
	query := `
		SELECT id, slug, name, db_name, is_default, status, metadata, created_at, updated_at, deleted_at
		FROM platform.tenants
		WHERE is_default = true AND deleted_at IS NULL
		LIMIT 1
	`

	var tenant Tenant
	var dbName *string
	var metadataBytes []byte

	err := s.db.QueryRow(ctx, query).Scan(
		&tenant.ID,
		&tenant.Slug,
		&tenant.Name,
		&dbName,
		&tenant.IsDefault,
		&tenant.Status,
		&metadataBytes,
		&tenant.CreatedAt,
		&tenant.UpdatedAt,
		&tenant.DeletedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNoDefaultTenant
		}
		return nil, fmt.Errorf("failed to get default tenant: %w", err)
	}

	tenant.DBName = dbName
	if metadataBytes != nil {
		if err := json.Unmarshal(metadataBytes, &tenant.Metadata); err != nil {
			tenant.Metadata = make(map[string]any)
		}
	}

	return &tenant, nil
}

func (s *Repository) GetAllActiveTenants(ctx context.Context) ([]Tenant, error) {
	query := `
		SELECT id, slug, name, db_name, is_default, status, metadata, created_at, updated_at, deleted_at
		FROM platform.tenants
		WHERE deleted_at IS NULL AND status = 'active'
		ORDER BY created_at ASC
	`

	rows, err := s.db.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to get all tenants: %w", err)
	}
	defer rows.Close()

	var tenants []Tenant
	for rows.Next() {
		var tenant Tenant
		var dbName *string
		var metadataBytes []byte

		err := rows.Scan(
			&tenant.ID,
			&tenant.Slug,
			&tenant.Name,
			&dbName,
			&tenant.IsDefault,
			&tenant.Status,
			&metadataBytes,
			&tenant.CreatedAt,
			&tenant.UpdatedAt,
			&tenant.DeletedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan tenant: %w", err)
		}

		tenant.DBName = dbName
		if metadataBytes != nil {
			if err := json.Unmarshal(metadataBytes, &tenant.Metadata); err != nil {
				tenant.Metadata = make(map[string]any)
			}
		}

		tenants = append(tenants, tenant)
	}

	if tenants == nil {
		tenants = []Tenant{}
	}

	return tenants, nil
}

func (s *Repository) CreateTenant(ctx context.Context, tenant *Tenant) error {
	metadataBytes, err := json.Marshal(tenant.Metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	query := `
		INSERT INTO platform.tenants (slug, name, db_name, is_default, status, metadata)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, created_at, updated_at
	`

	err = s.db.QueryRow(
		ctx, query,
		tenant.Slug,
		tenant.Name,
		tenant.DBName,
		tenant.IsDefault,
		tenant.Status,
		metadataBytes,
	).Scan(&tenant.ID, &tenant.CreatedAt, &tenant.UpdatedAt)
	if err != nil {
		return fmt.Errorf("failed to create tenant: %w", err)
	}

	log.Info().Str("tenant_id", tenant.ID).Str("slug", tenant.Slug).Msg("Created tenant record")
	return nil
}

func (s *Repository) UpdateTenantStatus(ctx context.Context, id string, status TenantStatus) error {
	query := `
		UPDATE platform.tenants
		SET status = $1, updated_at = NOW()
		WHERE id = $2::uuid
	`

	result, err := s.db.Exec(ctx, query, status, id)
	if err != nil {
		return fmt.Errorf("failed to update tenant status: %w", err)
	}

	if result.RowsAffected() == 0 {
		return ErrTenantNotFound
	}

	return nil
}

func (s *Repository) UpdateTenantDBName(ctx context.Context, id string, dbName string) error {
	query := `
		UPDATE platform.tenants
		SET db_name = $1, updated_at = NOW()
		WHERE id = $2::uuid
	`

	result, err := s.db.Exec(ctx, query, dbName, id)
	if err != nil {
		return fmt.Errorf("failed to update tenant db_name: %w", err)
	}

	if result.RowsAffected() == 0 {
		return ErrTenantNotFound
	}

	return nil
}

func (s *Repository) UpdateTenant(ctx context.Context, id string, req UpdateTenantRequest) error {
	var setClauses []string
	args := make([]any, 0, 3)
	argIdx := 1

	if req.Name != nil {
		setClauses = append(setClauses, fmt.Sprintf("name = $%d", argIdx))
		args = append(args, *req.Name)
		argIdx++
	}

	if req.Metadata != nil {
		metadataBytes, err := json.Marshal(req.Metadata)
		if err != nil {
			return fmt.Errorf("failed to marshal metadata: %w", err)
		}
		setClauses = append(setClauses, fmt.Sprintf("metadata = $%d", argIdx))
		args = append(args, metadataBytes)
		argIdx++
	}

	if len(setClauses) == 0 {
		return nil
	}

	args = append(args, id)

	query := "UPDATE platform.tenants SET " + strings.Join(setClauses, ", ") +
		", updated_at = NOW() WHERE id = $" + fmt.Sprint(argIdx) + "::uuid"

	result, err := s.db.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("failed to update tenant: %w", err)
	}

	if result.RowsAffected() == 0 {
		return ErrTenantNotFound
	}

	return nil
}

func (s *Repository) SoftDeleteTenant(ctx context.Context, id string) error {
	query := `
		UPDATE platform.tenants
		SET deleted_at = NOW(), status = 'deleting'
		WHERE id = $1::uuid AND deleted_at IS NULL
	`

	result, err := s.db.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to soft delete tenant: %w", err)
	}

	if result.RowsAffected() == 0 {
		return ErrTenantNotFound
	}

	return nil
}

func (s *Repository) HardDeleteTenant(ctx context.Context, id string) error {
	query := `DELETE FROM platform.tenants WHERE id = $1::uuid`

	result, err := s.db.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to hard delete tenant: %w", err)
	}

	if result.RowsAffected() == 0 {
		return ErrTenantNotFound
	}

	return nil
}

func (s *Repository) RecoverTenant(ctx context.Context, id string) error {
	query := `
		UPDATE platform.tenants
		SET deleted_at = NULL, status = 'active', updated_at = NOW()
		WHERE id = $1::uuid AND deleted_at IS NOT NULL
	`

	result, err := s.db.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to recover tenant: %w", err)
	}

	if result.RowsAffected() == 0 {
		return ErrTenantNotDeleted
	}

	return nil
}

func (s *Repository) GetDeletedTenants(ctx context.Context) ([]Tenant, error) {
	query := `
		SELECT id, slug, name, db_name, is_default, status, metadata, created_at, updated_at, deleted_at
		FROM platform.tenants
		WHERE deleted_at IS NOT NULL
		ORDER BY deleted_at DESC
	`

	rows, err := s.db.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to get deleted tenants: %w", err)
	}
	defer rows.Close()

	var tenants []Tenant
	for rows.Next() {
		var tenant Tenant
		var dbName *string
		var metadataBytes []byte

		err := rows.Scan(
			&tenant.ID,
			&tenant.Slug,
			&tenant.Name,
			&dbName,
			&tenant.IsDefault,
			&tenant.Status,
			&metadataBytes,
			&tenant.CreatedAt,
			&tenant.UpdatedAt,
			&tenant.DeletedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan tenant: %w", err)
		}

		tenant.DBName = dbName
		if metadataBytes != nil {
			if err := json.Unmarshal(metadataBytes, &tenant.Metadata); err != nil {
				tenant.Metadata = make(map[string]any)
			}
		}

		tenants = append(tenants, tenant)
	}

	if tenants == nil {
		tenants = []Tenant{}
	}

	return tenants, nil
}

func (s *Repository) AssignUserToTenant(ctx context.Context, userID, tenantID string) error {
	query := `
		INSERT INTO platform.tenant_admin_assignments (tenant_id, user_id)
		VALUES ($1::uuid, $2::uuid)
		ON CONFLICT (tenant_id, user_id) DO NOTHING
	`

	_, err := s.db.Exec(ctx, query, tenantID, userID)
	if err != nil {
		return fmt.Errorf("failed to assign user to tenant: %w", err)
	}

	return nil
}

func (s *Repository) RemoveUserFromTenant(ctx context.Context, userID, tenantID string) error {
	query := `
		DELETE FROM platform.tenant_admin_assignments
		WHERE tenant_id = $1::uuid AND user_id = $2::uuid
	`

	_, err := s.db.Exec(ctx, query, tenantID, userID)
	if err != nil {
		return fmt.Errorf("failed to remove user from tenant: %w", err)
	}

	return nil
}

func (s *Repository) IsUserAssignedToTenant(ctx context.Context, userID, tenantID string) (bool, error) {
	query := `
		SELECT EXISTS(
			SELECT 1 FROM platform.tenant_admin_assignments
			WHERE tenant_id = $1::uuid AND user_id = $2::uuid
		)
	`

	var exists bool
	err := s.db.QueryRow(ctx, query, tenantID, userID).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check user assignment: %w", err)
	}

	return exists, nil
}

func (s *Repository) GetTenantAssignments(ctx context.Context, userID string) ([]string, error) {
	query := `
		SELECT tenant_id::text
		FROM platform.tenant_admin_assignments
		WHERE user_id = $1::uuid
	`

	rows, err := s.db.Query(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get tenant assignments: %w", err)
	}
	defer rows.Close()

	var tenantIDs []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("failed to scan tenant id: %w", err)
		}
		tenantIDs = append(tenantIDs, id)
	}

	if tenantIDs == nil {
		tenantIDs = []string{}
	}

	return tenantIDs, nil
}

func (s *Repository) GetTenantsForUser(ctx context.Context, userID string) ([]Tenant, error) {
	query := `
		SELECT t.id, t.slug, t.name, t.db_name, t.is_default, t.status, t.metadata, t.created_at, t.updated_at, t.deleted_at
		FROM platform.tenants t
		INNER JOIN platform.tenant_admin_assignments taa ON t.id = taa.tenant_id
		WHERE taa.user_id = $1::uuid AND t.deleted_at IS NULL
		ORDER BY t.created_at ASC
	`

	rows, err := s.db.Query(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get tenants for user: %w", err)
	}
	defer rows.Close()

	var tenants []Tenant
	for rows.Next() {
		var tenant Tenant
		var dbName *string
		var metadataBytes []byte

		err := rows.Scan(
			&tenant.ID,
			&tenant.Slug,
			&tenant.Name,
			&dbName,
			&tenant.IsDefault,
			&tenant.Status,
			&metadataBytes,
			&tenant.CreatedAt,
			&tenant.UpdatedAt,
			&tenant.DeletedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan tenant: %w", err)
		}

		tenant.DBName = dbName
		if metadataBytes != nil {
			if err := json.Unmarshal(metadataBytes, &tenant.Metadata); err != nil {
				tenant.Metadata = make(map[string]any)
			}
		}

		tenants = append(tenants, tenant)
	}

	if tenants == nil {
		tenants = []Tenant{}
	}

	return tenants, nil
}

func (s *Repository) CountTenants(ctx context.Context) (int, error) {
	query := `SELECT COUNT(*) FROM platform.tenants WHERE deleted_at IS NULL`

	var count int
	err := s.db.QueryRow(ctx, query).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count tenants: %w", err)
	}

	return count, nil
}

// CleanupTenantData deletes tenant-related data from the main database.
// This must be called before HardDeleteTenant because branching tables have
// RESTRICT FK constraints that would block tenant row deletion.
func (s *Repository) CleanupTenantData(ctx context.Context, tenantID string) error {
	// Phase 1: Branching tables (RESTRICT FK — must delete before tenant row).
	// Order matters: delete child tables before branches.
	branchingTables := []string{
		"branching.seed_execution_log",
		"branching.migration_history",
		"branching.branch_access",
		"branching.activity_log",
		"branching.github_config",
		"branching.branches",
	}
	for _, table := range branchingTables {
		_, err := s.db.Exec(ctx, fmt.Sprintf("DELETE FROM %s WHERE tenant_id = $1::uuid", table), tenantID)
		if err != nil {
			return fmt.Errorf("failed to cleanup %s for tenant %s: %w", table, tenantID, err)
		}
	}

	// Phase 2: Auth tables (no FK to tenants — best-effort cleanup).
	// For tenants with separate databases, these users exist only for that tenant.
	authTables := []string{
		"auth.saml_sessions",
		"auth.mcp_oauth_tokens",
		"auth.mcp_oauth_codes",
		"auth.mcp_oauth_clients",
		"auth.oauth_logout_states",
		"auth.oauth_tokens",
		"auth.oauth_links",
		"auth.two_factor_recovery_attempts",
		"auth.two_factor_setups",
		"auth.password_reset_tokens",
		"auth.email_verification_tokens",
		"auth.otp_codes",
		"auth.magic_links",
		"auth.sessions",
		"auth.mfa_factors",
		"auth.client_key_usage",
		"auth.service_key_revocations",
		"auth.webhook_deliveries",
		"auth.webhook_events",
		"auth.webhooks",
		"auth.client_keys",
		"auth.users",
	}
	for _, table := range authTables {
		_, err := s.db.Exec(ctx, fmt.Sprintf("DELETE FROM %s WHERE tenant_id = $1::uuid", table), tenantID)
		if err != nil {
			// Non-fatal: these tables may not have tenant_id or may have FK
			// dependencies we can't resolve here.
			log.Warn().Err(err).Str("table", table).Str("tenant_id", tenantID).
				Msg("Failed to cleanup auth table for tenant (non-fatal)")
		}
	}

	// Phase 3: Functions and jobs.
	tables3 := []string{
		"functions.edge_function_files",
		"functions.edge_executions",
		"functions.edge_functions",
		"jobs.queue",
		"jobs.function_files",
		"jobs.functions",
	}
	for _, table := range tables3 {
		_, err := s.db.Exec(ctx, fmt.Sprintf("DELETE FROM %s WHERE tenant_id = $1::uuid", table), tenantID)
		if err != nil {
			log.Warn().Err(err).Str("table", table).Str("tenant_id", tenantID).
				Msg("Failed to cleanup table for tenant (non-fatal)")
		}
	}

	// Phase 4: RPC.
	tables4 := []string{
		"rpc.executions",
		"rpc.procedures",
	}
	for _, table := range tables4 {
		_, err := s.db.Exec(ctx, fmt.Sprintf("DELETE FROM %s WHERE tenant_id = $1::uuid", table), tenantID)
		if err != nil {
			log.Warn().Err(err).Str("table", table).Str("tenant_id", tenantID).
				Msg("Failed to cleanup table for tenant (non-fatal)")
		}
	}

	// Phase 5: AI.
	tables5 := []string{
		"ai.conversations",
		"ai.chatbot_knowledge_bases",
		"ai.chatbots",
		"ai.chunks",
		"ai.documents",
		"ai.knowledge_bases",
	}
	for _, table := range tables5 {
		_, err := s.db.Exec(ctx, fmt.Sprintf("DELETE FROM %s WHERE tenant_id = $1::uuid", table), tenantID)
		if err != nil {
			log.Warn().Err(err).Str("table", table).Str("tenant_id", tenantID).
				Msg("Failed to cleanup table for tenant (non-fatal)")
		}
	}

	// Phase 6: Webhooks (extended).
	tables6 := []string{
		"auth.webhook_deliveries",
		"auth.webhook_events",
		"auth.webhooks",
	}
	for _, table := range tables6 {
		_, err := s.db.Exec(ctx, fmt.Sprintf("DELETE FROM %s WHERE tenant_id = $1::uuid", table), tenantID)
		if err != nil {
			log.Warn().Err(err).Str("table", table).Str("tenant_id", tenantID).
				Msg("Failed to cleanup table for tenant (non-fatal)")
		}
	}

	// Phase 7: Secrets, keys, and settings.
	tables7 := []string{
		"functions.secrets",
		"functions.secret_versions",
		"auth.service_keys",
		"app.settings",
	}
	for _, table := range tables7 {
		_, err := s.db.Exec(ctx, fmt.Sprintf("DELETE FROM %s WHERE tenant_id = $1::uuid", table), tenantID)
		if err != nil {
			log.Warn().Err(err).Str("table", table).Str("tenant_id", tenantID).
				Msg("Failed to cleanup table for tenant (non-fatal)")
		}
	}

	// Phase 8: Logging and realtime.
	tables8 := []string{
		"logging.entries",
		"realtime.schema_registry",
	}
	for _, table := range tables8 {
		_, err := s.db.Exec(ctx, fmt.Sprintf("DELETE FROM %s WHERE tenant_id = $1::uuid", table), tenantID)
		if err != nil {
			log.Warn().Err(err).Str("table", table).Str("tenant_id", tenantID).
				Msg("Failed to cleanup table for tenant (non-fatal)")
		}
	}

	// Phase 9: Migrations.
	tables9 := []string{
		"platform.migrations",
	}
	for _, table := range tables9 {
		_, err := s.db.Exec(ctx, fmt.Sprintf("DELETE FROM %s WHERE tenant_id = $1::uuid", table), tenantID)
		if err != nil {
			log.Warn().Err(err).Str("table", table).Str("tenant_id", tenantID).
				Msg("Failed to cleanup table for tenant (non-fatal)")
		}
	}

	// Phase 10: Platform.
	tables10 := []string{
		"platform.tenant_admin_assignments",
		"platform.tenant_memberships",
	}
	for _, table := range tables10 {
		_, err := s.db.Exec(ctx, fmt.Sprintf("DELETE FROM %s WHERE tenant_id = $1::uuid", table), tenantID)
		if err != nil {
			log.Warn().Err(err).Str("table", table).Str("tenant_id", tenantID).
				Msg("Failed to cleanup table for tenant (non-fatal)")
		}
	}

	return nil
}
