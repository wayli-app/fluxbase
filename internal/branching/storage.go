package branching

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Storage handles CRUD operations for branch metadata
type Storage struct {
	pool *pgxpool.Pool
}

// NewStorage creates a new Storage instance
func NewStorage(pool *pgxpool.Pool) *Storage {
	return &Storage{pool: pool}
}

// CreateBranch creates a new branch record
func (s *Storage) CreateBranch(ctx context.Context, branch *Branch) error {
	query := `
		INSERT INTO branching.branches (
			id, name, slug, database_name, status, type, parent_branch_id,
			data_clone_mode, github_pr_number, github_pr_url, github_repo,
			created_by, expires_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13
		) RETURNING created_at, updated_at`

	if branch.ID == uuid.Nil {
		branch.ID = uuid.New()
	}

	return s.pool.QueryRow(ctx, query,
		branch.ID,
		branch.Name,
		branch.Slug,
		branch.DatabaseName,
		branch.Status,
		branch.Type,
		branch.ParentBranchID,
		branch.DataCloneMode,
		branch.GitHubPRNumber,
		branch.GitHubPRURL,
		branch.GitHubRepo,
		branch.CreatedBy,
		branch.ExpiresAt,
	).Scan(&branch.CreatedAt, &branch.UpdatedAt)
}

// GetBranch retrieves a branch by ID
func (s *Storage) GetBranch(ctx context.Context, id uuid.UUID) (*Branch, error) {
	query := `
		SELECT id, name, slug, database_name, status, type, parent_branch_id,
			data_clone_mode, github_pr_number, github_pr_url, github_repo,
			error_message, created_by, created_at, updated_at, expires_at
		FROM branching.branches
		WHERE id = $1 AND status != 'deleted'`

	branch := &Branch{}
	err := s.pool.QueryRow(ctx, query, id).Scan(
		&branch.ID,
		&branch.Name,
		&branch.Slug,
		&branch.DatabaseName,
		&branch.Status,
		&branch.Type,
		&branch.ParentBranchID,
		&branch.DataCloneMode,
		&branch.GitHubPRNumber,
		&branch.GitHubPRURL,
		&branch.GitHubRepo,
		&branch.ErrorMessage,
		&branch.CreatedBy,
		&branch.CreatedAt,
		&branch.UpdatedAt,
		&branch.ExpiresAt,
	)
	if err == pgx.ErrNoRows {
		return nil, ErrBranchNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get branch: %w", err)
	}
	return branch, nil
}

// GetBranchBySlug retrieves a branch by slug
func (s *Storage) GetBranchBySlug(ctx context.Context, slug string) (*Branch, error) {
	query := `
		SELECT id, name, slug, database_name, status, type, parent_branch_id,
			data_clone_mode, github_pr_number, github_pr_url, github_repo,
			error_message, created_by, created_at, updated_at, expires_at
		FROM branching.branches
		WHERE slug = $1 AND status != 'deleted'`

	branch := &Branch{}
	err := s.pool.QueryRow(ctx, query, slug).Scan(
		&branch.ID,
		&branch.Name,
		&branch.Slug,
		&branch.DatabaseName,
		&branch.Status,
		&branch.Type,
		&branch.ParentBranchID,
		&branch.DataCloneMode,
		&branch.GitHubPRNumber,
		&branch.GitHubPRURL,
		&branch.GitHubRepo,
		&branch.ErrorMessage,
		&branch.CreatedBy,
		&branch.CreatedAt,
		&branch.UpdatedAt,
		&branch.ExpiresAt,
	)
	if err == pgx.ErrNoRows {
		return nil, ErrBranchNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get branch by slug: %w", err)
	}
	return branch, nil
}

// GetBranchByGitHubPR retrieves a branch by GitHub repo and PR number
func (s *Storage) GetBranchByGitHubPR(ctx context.Context, repo string, prNumber int) (*Branch, error) {
	query := `
		SELECT id, name, slug, database_name, status, type, parent_branch_id,
			data_clone_mode, github_pr_number, github_pr_url, github_repo,
			error_message, created_by, created_at, updated_at, expires_at
		FROM branching.branches
		WHERE github_repo = $1 AND github_pr_number = $2 AND status != 'deleted'`

	branch := &Branch{}
	err := s.pool.QueryRow(ctx, query, repo, prNumber).Scan(
		&branch.ID,
		&branch.Name,
		&branch.Slug,
		&branch.DatabaseName,
		&branch.Status,
		&branch.Type,
		&branch.ParentBranchID,
		&branch.DataCloneMode,
		&branch.GitHubPRNumber,
		&branch.GitHubPRURL,
		&branch.GitHubRepo,
		&branch.ErrorMessage,
		&branch.CreatedBy,
		&branch.CreatedAt,
		&branch.UpdatedAt,
		&branch.ExpiresAt,
	)
	if err == pgx.ErrNoRows {
		return nil, ErrBranchNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get branch by GitHub PR: %w", err)
	}
	return branch, nil
}

// GetMainBranch retrieves the main branch
func (s *Storage) GetMainBranch(ctx context.Context) (*Branch, error) {
	query := `
		SELECT id, name, slug, database_name, status, type, parent_branch_id,
			data_clone_mode, github_pr_number, github_pr_url, github_repo,
			error_message, created_by, created_at, updated_at, expires_at
		FROM branching.branches
		WHERE type = 'main' AND status != 'deleted'
		LIMIT 1`

	branch := &Branch{}
	err := s.pool.QueryRow(ctx, query).Scan(
		&branch.ID,
		&branch.Name,
		&branch.Slug,
		&branch.DatabaseName,
		&branch.Status,
		&branch.Type,
		&branch.ParentBranchID,
		&branch.DataCloneMode,
		&branch.GitHubPRNumber,
		&branch.GitHubPRURL,
		&branch.GitHubRepo,
		&branch.ErrorMessage,
		&branch.CreatedBy,
		&branch.CreatedAt,
		&branch.UpdatedAt,
		&branch.ExpiresAt,
	)
	if err == pgx.ErrNoRows {
		return nil, ErrBranchNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get main branch: %w", err)
	}
	return branch, nil
}

// ListBranches lists branches with optional filtering
func (s *Storage) ListBranches(ctx context.Context, filter ListBranchesFilter) ([]*Branch, error) {
	query := `
		SELECT id, name, slug, database_name, status, type, parent_branch_id,
			data_clone_mode, github_pr_number, github_pr_url, github_repo,
			error_message, created_by, created_at, updated_at, expires_at
		FROM branching.branches
		WHERE status != 'deleted'`

	args := []any{}
	argCounter := 1

	if filter.Status != nil {
		query += fmt.Sprintf(" AND status = $%d", argCounter)
		args = append(args, *filter.Status)
		argCounter++
	}

	if filter.Type != nil {
		query += fmt.Sprintf(" AND type = $%d", argCounter)
		args = append(args, *filter.Type)
		argCounter++
	}

	if filter.CreatedBy != nil {
		query += fmt.Sprintf(" AND created_by = $%d", argCounter)
		args = append(args, *filter.CreatedBy)
		argCounter++
	}

	if filter.GitHubRepo != nil {
		query += fmt.Sprintf(" AND github_repo = $%d", argCounter)
		args = append(args, *filter.GitHubRepo)
		argCounter++
	}

	query += " ORDER BY created_at DESC"

	// Use parameterized queries for LIMIT and OFFSET to prevent SQL injection
	if filter.Limit > 0 {
		query += fmt.Sprintf(" LIMIT $%d", argCounter)
		args = append(args, filter.Limit)
		argCounter++
	}

	if filter.Offset > 0 {
		query += fmt.Sprintf(" OFFSET $%d", argCounter)
		args = append(args, filter.Offset)
		argCounter++ //nolint:ineffassign // keeping for consistency
	}

	rows, err := s.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list branches: %w", err)
	}
	defer rows.Close()

	var branches []*Branch
	for rows.Next() {
		branch := &Branch{}
		err := rows.Scan(
			&branch.ID,
			&branch.Name,
			&branch.Slug,
			&branch.DatabaseName,
			&branch.Status,
			&branch.Type,
			&branch.ParentBranchID,
			&branch.DataCloneMode,
			&branch.GitHubPRNumber,
			&branch.GitHubPRURL,
			&branch.GitHubRepo,
			&branch.ErrorMessage,
			&branch.CreatedBy,
			&branch.CreatedAt,
			&branch.UpdatedAt,
			&branch.ExpiresAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan branch: %w", err)
		}
		branches = append(branches, branch)
	}

	return branches, rows.Err()
}

// UpdateBranchStatus updates the status of a branch
func (s *Storage) UpdateBranchStatus(ctx context.Context, id uuid.UUID, status BranchStatus, errorMessage *string) error {
	query := `
		UPDATE branching.branches
		SET status = $1, error_message = $2, updated_at = NOW()
		WHERE id = $3`

	result, err := s.pool.Exec(ctx, query, status, errorMessage, id)
	if err != nil {
		return fmt.Errorf("failed to update branch status: %w", err)
	}

	if result.RowsAffected() == 0 {
		return ErrBranchNotFound
	}

	return nil
}

// DeleteBranch marks a branch as deleted (soft delete)
func (s *Storage) DeleteBranch(ctx context.Context, id uuid.UUID) error {
	query := `
		UPDATE branching.branches
		SET status = 'deleted', updated_at = NOW()
		WHERE id = $1 AND type != 'main'`

	result, err := s.pool.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete branch: %w", err)
	}

	if result.RowsAffected() == 0 {
		return ErrBranchNotFound
	}

	return nil
}

// CountBranches counts branches matching the filter
func (s *Storage) CountBranches(ctx context.Context, filter ListBranchesFilter) (int, error) {
	query := `SELECT COUNT(*) FROM branching.branches WHERE status != 'deleted'`

	args := []any{}
	argCounter := 1

	if filter.Status != nil {
		query += fmt.Sprintf(" AND status = $%d", argCounter)
		args = append(args, *filter.Status)
		argCounter++
	}

	if filter.Type != nil {
		query += fmt.Sprintf(" AND type = $%d", argCounter)
		args = append(args, *filter.Type)
		argCounter++
	}

	if filter.CreatedBy != nil {
		query += fmt.Sprintf(" AND created_by = $%d", argCounter)
		args = append(args, *filter.CreatedBy)
	}

	var count int
	err := s.pool.QueryRow(ctx, query, args...).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count branches: %w", err)
	}

	return count, nil
}

// CountBranchesByUser counts branches created by a specific user
func (s *Storage) CountBranchesByUser(ctx context.Context, userID uuid.UUID) (int, error) {
	query := `SELECT COUNT(*) FROM branching.branches WHERE created_by = $1 AND status NOT IN ('deleted', 'deleting')`

	var count int
	err := s.pool.QueryRow(ctx, query, userID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count user branches: %w", err)
	}

	return count, nil
}

// LogActivity records an activity log entry
func (s *Storage) LogActivity(ctx context.Context, log *ActivityLog) error {
	query := `
		INSERT INTO branching.activity_log (
			id, branch_id, action, status, details, error_message, executed_by, duration_ms
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING executed_at`

	if log.ID == uuid.Nil {
		log.ID = uuid.New()
	}

	var detailsJSON []byte
	if log.Details != nil {
		var err error
		detailsJSON, err = json.Marshal(log.Details)
		if err != nil {
			return fmt.Errorf("failed to marshal details: %w", err)
		}
	}

	return s.pool.QueryRow(ctx, query,
		log.ID,
		log.BranchID,
		log.Action,
		log.Status,
		detailsJSON,
		log.ErrorMessage,
		log.ExecutedBy,
		log.DurationMs,
	).Scan(&log.ExecutedAt)
}

// GetActivityLog retrieves activity logs for a branch
func (s *Storage) GetActivityLog(ctx context.Context, branchID uuid.UUID, limit int) ([]*ActivityLog, error) {
	if limit <= 0 {
		limit = 50
	}

	query := `
		SELECT id, branch_id, action, status, details, error_message, executed_by, executed_at, duration_ms
		FROM branching.activity_log
		WHERE branch_id = $1
		ORDER BY executed_at DESC
		LIMIT $2`

	rows, err := s.pool.Query(ctx, query, branchID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get activity log: %w", err)
	}
	defer rows.Close()

	var logs []*ActivityLog
	for rows.Next() {
		log := &ActivityLog{}
		var detailsJSON []byte
		err := rows.Scan(
			&log.ID,
			&log.BranchID,
			&log.Action,
			&log.Status,
			&detailsJSON,
			&log.ErrorMessage,
			&log.ExecutedBy,
			&log.ExecutedAt,
			&log.DurationMs,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan activity log: %w", err)
		}
		if detailsJSON != nil {
			if err := json.Unmarshal(detailsJSON, &log.Details); err != nil {
				return nil, fmt.Errorf("failed to unmarshal details: %w", err)
			}
		}
		logs = append(logs, log)
	}

	return logs, rows.Err()
}

// RecordMigration records a migration applied to a branch
func (s *Storage) RecordMigration(ctx context.Context, branchID uuid.UUID, version int64, name string) error {
	query := `
		INSERT INTO branching.migration_history (branch_id, migration_version, migration_name)
		VALUES ($1, $2, $3)
		ON CONFLICT (branch_id, migration_version) DO NOTHING`

	_, err := s.pool.Exec(ctx, query, branchID, version, name)
	if err != nil {
		return fmt.Errorf("failed to record migration: %w", err)
	}

	return nil
}

// GetMigrationHistory retrieves the migration history for a branch
func (s *Storage) GetMigrationHistory(ctx context.Context, branchID uuid.UUID) ([]*MigrationHistory, error) {
	query := `
		SELECT id, branch_id, migration_version, migration_name, applied_at
		FROM branching.migration_history
		WHERE branch_id = $1
		ORDER BY migration_version ASC`

	rows, err := s.pool.Query(ctx, query, branchID)
	if err != nil {
		return nil, fmt.Errorf("failed to get migration history: %w", err)
	}
	defer rows.Close()

	var history []*MigrationHistory
	for rows.Next() {
		mh := &MigrationHistory{}
		err := rows.Scan(
			&mh.ID,
			&mh.BranchID,
			&mh.MigrationVersion,
			&mh.MigrationName,
			&mh.AppliedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan migration history: %w", err)
		}
		history = append(history, mh)
	}

	return history, rows.Err()
}

// GetExpiredBranches returns branches that have passed their expiration time
func (s *Storage) GetExpiredBranches(ctx context.Context) ([]*Branch, error) {
	query := `
		SELECT id, name, slug, database_name, status, type, parent_branch_id,
			data_clone_mode, github_pr_number, github_pr_url, github_repo,
			error_message, created_by, created_at, updated_at, expires_at
		FROM branching.branches
		WHERE expires_at IS NOT NULL
			AND expires_at < NOW()
			AND status NOT IN ('deleted', 'deleting')
			AND type != 'main'`

	rows, err := s.pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to get expired branches: %w", err)
	}
	defer rows.Close()

	var branches []*Branch
	for rows.Next() {
		branch := &Branch{}
		err := rows.Scan(
			&branch.ID,
			&branch.Name,
			&branch.Slug,
			&branch.DatabaseName,
			&branch.Status,
			&branch.Type,
			&branch.ParentBranchID,
			&branch.DataCloneMode,
			&branch.GitHubPRNumber,
			&branch.GitHubPRURL,
			&branch.GitHubRepo,
			&branch.ErrorMessage,
			&branch.CreatedBy,
			&branch.CreatedAt,
			&branch.UpdatedAt,
			&branch.ExpiresAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan expired branch: %w", err)
		}
		branches = append(branches, branch)
	}

	return branches, rows.Err()
}

// GitHub Config methods

// GetGitHubConfig retrieves GitHub config for a repository
func (s *Storage) GetGitHubConfig(ctx context.Context, repository string) (*GitHubConfig, error) {
	query := `
		SELECT id, repository, auto_create_on_pr, auto_delete_on_merge,
			default_data_clone_mode, webhook_secret, created_at, updated_at
		FROM branching.github_config
		WHERE repository = $1`

	config := &GitHubConfig{}
	err := s.pool.QueryRow(ctx, query, repository).Scan(
		&config.ID,
		&config.Repository,
		&config.AutoCreateOnPR,
		&config.AutoDeleteOnMerge,
		&config.DefaultDataCloneMode,
		&config.WebhookSecret,
		&config.CreatedAt,
		&config.UpdatedAt,
	)
	if err == pgx.ErrNoRows {
		return nil, ErrGitHubConfigNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get GitHub config: %w", err)
	}
	return config, nil
}

// UpsertGitHubConfig creates or updates GitHub config
func (s *Storage) UpsertGitHubConfig(ctx context.Context, config *GitHubConfig) error {
	query := `
		INSERT INTO branching.github_config (
			id, repository, auto_create_on_pr, auto_delete_on_merge,
			default_data_clone_mode, webhook_secret
		) VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (repository) DO UPDATE SET
			auto_create_on_pr = EXCLUDED.auto_create_on_pr,
			auto_delete_on_merge = EXCLUDED.auto_delete_on_merge,
			default_data_clone_mode = EXCLUDED.default_data_clone_mode,
			webhook_secret = EXCLUDED.webhook_secret,
			updated_at = NOW()
		RETURNING id, created_at, updated_at`

	if config.ID == uuid.Nil {
		config.ID = uuid.New()
	}

	return s.pool.QueryRow(ctx, query,
		config.ID,
		config.Repository,
		config.AutoCreateOnPR,
		config.AutoDeleteOnMerge,
		config.DefaultDataCloneMode,
		config.WebhookSecret,
	).Scan(&config.ID, &config.CreatedAt, &config.UpdatedAt)
}

// DeleteGitHubConfig deletes GitHub config for a repository
func (s *Storage) DeleteGitHubConfig(ctx context.Context, repository string) error {
	query := `DELETE FROM branching.github_config WHERE repository = $1`

	result, err := s.pool.Exec(ctx, query, repository)
	if err != nil {
		return fmt.Errorf("failed to delete GitHub config: %w", err)
	}

	if result.RowsAffected() == 0 {
		return ErrGitHubConfigNotFound
	}

	return nil
}

// ListGitHubConfigs lists all GitHub configurations
func (s *Storage) ListGitHubConfigs(ctx context.Context) ([]*GitHubConfig, error) {
	query := `
		SELECT id, repository, auto_create_on_pr, auto_delete_on_merge,
			default_data_clone_mode, webhook_secret, created_at, updated_at
		FROM branching.github_config
		ORDER BY repository`

	rows, err := s.pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to list GitHub configs: %w", err)
	}
	defer rows.Close()

	var configs []*GitHubConfig
	for rows.Next() {
		config := &GitHubConfig{}
		err := rows.Scan(
			&config.ID,
			&config.Repository,
			&config.AutoCreateOnPR,
			&config.AutoDeleteOnMerge,
			&config.DefaultDataCloneMode,
			&config.WebhookSecret,
			&config.CreatedAt,
			&config.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan GitHub config: %w", err)
		}
		configs = append(configs, config)
	}

	return configs, rows.Err()
}

// Branch Access methods

// GrantAccess grants a user access to a branch
func (s *Storage) GrantAccess(ctx context.Context, access *BranchAccess) error {
	query := `
		INSERT INTO branching.branch_access (id, branch_id, user_id, access_level, granted_by)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (branch_id, user_id) DO UPDATE SET
			access_level = EXCLUDED.access_level,
			granted_by = EXCLUDED.granted_by,
			granted_at = NOW()
		RETURNING id, granted_at`

	if access.ID == uuid.Nil {
		access.ID = uuid.New()
	}

	return s.pool.QueryRow(ctx, query,
		access.ID,
		access.BranchID,
		access.UserID,
		access.AccessLevel,
		access.GrantedBy,
	).Scan(&access.ID, &access.GrantedAt)
}

// RevokeAccess revokes a user's access to a branch
func (s *Storage) RevokeAccess(ctx context.Context, branchID, userID uuid.UUID) error {
	query := `DELETE FROM branching.branch_access WHERE branch_id = $1 AND user_id = $2`

	_, err := s.pool.Exec(ctx, query, branchID, userID)
	return err
}

// GetBranchAccessList returns all access grants for a branch
func (s *Storage) GetBranchAccessList(ctx context.Context, branchID uuid.UUID) ([]*BranchAccess, error) {
	query := `
		SELECT id, branch_id, user_id, access_level, granted_at, granted_by
		FROM branching.branch_access
		WHERE branch_id = $1
		ORDER BY granted_at DESC`

	rows, err := s.pool.Query(ctx, query, branchID)
	if err != nil {
		return nil, fmt.Errorf("failed to list branch access: %w", err)
	}
	defer rows.Close()

	var accessList []*BranchAccess
	for rows.Next() {
		access := &BranchAccess{}
		if err := rows.Scan(
			&access.ID,
			&access.BranchID,
			&access.UserID,
			&access.AccessLevel,
			&access.GrantedAt,
			&access.GrantedBy,
		); err != nil {
			return nil, fmt.Errorf("failed to scan branch access: %w", err)
		}
		accessList = append(accessList, access)
	}

	return accessList, nil
}

// GetUserAccess returns the access level for a specific user on a branch
func (s *Storage) GetUserAccess(ctx context.Context, branchID, userID uuid.UUID) (*BranchAccess, error) {
	query := `
		SELECT id, branch_id, user_id, access_level, granted_at, granted_by
		FROM branching.branch_access
		WHERE branch_id = $1 AND user_id = $2`

	access := &BranchAccess{}
	err := s.pool.QueryRow(ctx, query, branchID, userID).Scan(
		&access.ID,
		&access.BranchID,
		&access.UserID,
		&access.AccessLevel,
		&access.GrantedAt,
		&access.GrantedBy,
	)
	if err == pgx.ErrNoRows {
		return nil, ErrBranchNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get user access: %w", err)
	}

	return access, nil
}

// HasAccess checks if a user has at least the specified access level to a branch
func (s *Storage) HasAccess(ctx context.Context, branchID, userID uuid.UUID, minLevel BranchAccessLevel) (bool, error) {
	// First check if user is the creator (always has admin access)
	var createdBy *uuid.UUID
	err := s.pool.QueryRow(ctx,
		`SELECT created_by FROM branching.branches WHERE id = $1`,
		branchID,
	).Scan(&createdBy)
	if err != nil {
		return false, fmt.Errorf("failed to check branch creator: %w", err)
	}

	if createdBy != nil && *createdBy == userID {
		return true, nil
	}

	// Then check explicit access grants
	query := `
		SELECT access_level FROM branching.branch_access
		WHERE branch_id = $1 AND user_id = $2`

	var accessLevel BranchAccessLevel
	err = s.pool.QueryRow(ctx, query, branchID, userID).Scan(&accessLevel)
	if err == pgx.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("failed to check access: %w", err)
	}

	// Check if access level is sufficient
	return isAccessSufficient(accessLevel, minLevel), nil
}

// isAccessSufficient checks if the granted level meets the minimum required level
func isAccessSufficient(granted, required BranchAccessLevel) bool {
	levels := map[BranchAccessLevel]int{
		BranchAccessRead:  1,
		BranchAccessWrite: 2,
		BranchAccessAdmin: 3,
	}
	return levels[granted] >= levels[required]
}

// UserHasAccess checks if a user has access to a branch (any level)
func (s *Storage) UserHasAccess(ctx context.Context, slug string, userID uuid.UUID) (bool, error) {
	// Get the branch first
	branch, err := s.GetBranchBySlug(ctx, slug)
	if err != nil {
		if err == ErrBranchNotFound {
			return false, nil
		}
		return false, err
	}

	// Main branch is accessible to all authenticated users
	if branch.Type == BranchTypeMain {
		return true, nil
	}

	return s.HasAccess(ctx, branch.ID, userID, BranchAccessRead)
}

// Helper functions

// slugRegex validates branch slugs
var slugRegex = regexp.MustCompile(`^[a-z0-9][a-z0-9-]*[a-z0-9]$|^[a-z0-9]$`)

// GenerateSlug generates a URL-safe slug from a branch name
func GenerateSlug(name string) string {
	// Convert to lowercase
	slug := strings.ToLower(name)

	// Replace spaces and underscores with hyphens
	slug = strings.ReplaceAll(slug, " ", "-")
	slug = strings.ReplaceAll(slug, "_", "-")

	// Remove invalid characters
	reg := regexp.MustCompile(`[^a-z0-9-]`)
	slug = reg.ReplaceAllString(slug, "")

	// Remove consecutive hyphens
	reg = regexp.MustCompile(`-+`)
	slug = reg.ReplaceAllString(slug, "-")

	// Trim hyphens from start and end
	slug = strings.Trim(slug, "-")

	// Ensure slug is not empty
	if slug == "" {
		slug = "branch"
	}

	// Limit length
	if len(slug) > 50 {
		slug = slug[:50]
		slug = strings.TrimRight(slug, "-")
	}

	return slug
}

// GeneratePRSlug generates a slug for a GitHub PR branch
func GeneratePRSlug(prNumber int) string {
	return fmt.Sprintf("pr-%d", prNumber)
}

// GenerateDatabaseName generates a database name for a branch
func GenerateDatabaseName(prefix, slug string) string {
	// Sanitize for PostgreSQL identifier
	name := prefix + slug

	// Replace hyphens with underscores (PostgreSQL identifiers)
	name = strings.ReplaceAll(name, "-", "_")

	// Ensure it starts with a letter or underscore
	if len(name) > 0 && name[0] >= '0' && name[0] <= '9' {
		name = "_" + name
	}

	// Limit to PostgreSQL max identifier length (63 chars)
	if len(name) > 63 {
		name = name[:63]
	}

	return name
}

// ValidateSlug validates that a slug is valid
func ValidateSlug(slug string) error {
	if slug == "" {
		return fmt.Errorf("slug cannot be empty")
	}

	if len(slug) > 50 {
		return fmt.Errorf("slug cannot be longer than 50 characters")
	}

	if slug == "main" {
		return fmt.Errorf("slug 'main' is reserved")
	}

	if !slugRegex.MatchString(slug) {
		return fmt.Errorf("slug must contain only lowercase letters, numbers, and hyphens")
	}

	return nil
}

// SetPool sets the connection pool (for testing)
func (s *Storage) SetPool(pool *pgxpool.Pool) {
	s.pool = pool
}

// GetPool returns the connection pool
func (s *Storage) GetPool() *pgxpool.Pool {
	return s.pool
}

// Transaction executes a function within a database transaction
func (s *Storage) Transaction(ctx context.Context, fn func(tx pgx.Tx) error) error {
	tx, err := s.pool.Begin(ctx)
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

	return nil
}

// SetBranchExpiresAt sets the expiration time for a branch
func (s *Storage) SetBranchExpiresAt(ctx context.Context, id uuid.UUID, expiresAt *time.Time) error {
	query := `
		UPDATE branching.branches
		SET expires_at = $1, updated_at = NOW()
		WHERE id = $2`

	result, err := s.pool.Exec(ctx, query, expiresAt, id)
	if err != nil {
		return fmt.Errorf("failed to set branch expiration: %w", err)
	}

	if result.RowsAffected() == 0 {
		return ErrBranchNotFound
	}

	return nil
}
