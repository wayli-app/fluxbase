package branching

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/nimbleflux/fluxbase/internal/database"
)

type Storage struct {
	database.TenantAware
	pool          *pgxpool.Pool
	encryptionKey []byte
}

func NewStorage(db *database.Connection, encryptionKey []byte) *Storage {
	var pool *pgxpool.Pool
	if db != nil {
		pool = db.Pool()
	}
	return &Storage{
		TenantAware:   database.TenantAware{DB: db},
		pool:          pool,
		encryptionKey: encryptionKey,
	}
}

// CreateBranch creates a new branch record
func (s *Storage) CreateBranch(ctx context.Context, branch *Branch) error {
	query := `
		INSERT INTO branching.branches (
			id, name, slug, database_name, status, type, tenant_id, parent_branch_id,
			data_clone_mode, github_pr_number, github_pr_url, github_repo,
			created_by, expires_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14
		) RETURNING created_at, updated_at`

	if branch.ID == uuid.Nil {
		branch.ID = uuid.New()
	}

	return s.WithTenant(ctx, func(tx pgx.Tx) error {
		return tx.QueryRow(
			ctx, query,
			branch.ID,
			branch.Name,
			branch.Slug,
			branch.DatabaseName,
			branch.Status,
			branch.Type,
			branch.TenantID,
			branch.ParentBranchID,
			branch.DataCloneMode,
			branch.GitHubPRNumber,
			branch.GitHubPRURL,
			branch.GitHubRepo,
			branch.CreatedBy,
			branch.ExpiresAt,
		).Scan(&branch.CreatedAt, &branch.UpdatedAt)
	})
}

// GetBranch retrieves a branch by ID.
// tenantID filters to a specific tenant when non-nil; nil means no tenant filter (admin/system access).
func (s *Storage) GetBranch(ctx context.Context, id uuid.UUID, tenantID *uuid.UUID) (*Branch, error) {
	query := `
		SELECT id, name, slug, database_name, status, type, tenant_id, parent_branch_id,
			data_clone_mode, github_pr_number, github_pr_url, github_repo,
			error_message, created_by, created_at, updated_at, expires_at
		FROM branching.branches
		WHERE id = $1 AND status != 'deleted'`

	args := []any{id}
	if tenantID != nil {
		query += fmt.Sprintf(" AND tenant_id = $%d", len(args)+1)
		args = append(args, *tenantID)
	}

	branch := &Branch{}
	err := s.WithTenant(ctx, func(tx pgx.Tx) error {
		return tx.QueryRow(ctx, query, args...).Scan(
			&branch.ID,
			&branch.Name,
			&branch.Slug,
			&branch.DatabaseName,
			&branch.Status,
			&branch.Type,
			&branch.TenantID,
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
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrBranchNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get branch: %w", err)
	}
	return branch, nil
}

// GetBranchBySlug retrieves a branch by slug.
// tenantID filters to a specific tenant when non-nil; nil means no tenant filter (admin/system access).
func (s *Storage) GetBranchBySlug(ctx context.Context, slug string, tenantID *uuid.UUID) (*Branch, error) {
	query := `
		SELECT id, name, slug, database_name, status, type, tenant_id, parent_branch_id,
			data_clone_mode, github_pr_number, github_pr_url, github_repo,
			error_message, created_by, created_at, updated_at, expires_at
		FROM branching.branches
		WHERE slug = $1 AND status != 'deleted'`

	args := []any{slug}
	if tenantID != nil {
		query += fmt.Sprintf(" AND tenant_id = $%d", len(args)+1)
		args = append(args, *tenantID)
	}

	branch := &Branch{}
	err := s.WithTenant(ctx, func(tx pgx.Tx) error {
		return tx.QueryRow(ctx, query, args...).Scan(
			&branch.ID,
			&branch.Name,
			&branch.Slug,
			&branch.DatabaseName,
			&branch.Status,
			&branch.Type,
			&branch.TenantID,
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
	})
	if errors.Is(err, pgx.ErrNoRows) {
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
		SELECT id, name, slug, database_name, status, type, tenant_id, parent_branch_id,
			data_clone_mode, github_pr_number, github_pr_url, github_repo,
			error_message, created_by, created_at, updated_at, expires_at
		FROM branching.branches
		WHERE github_repo = $1 AND github_pr_number = $2 AND status != 'deleted'`

	branch := &Branch{}
	err := s.WithTenant(ctx, func(tx pgx.Tx) error {
		return tx.QueryRow(ctx, query, repo, prNumber).Scan(
			&branch.ID,
			&branch.Name,
			&branch.Slug,
			&branch.DatabaseName,
			&branch.Status,
			&branch.Type,
			&branch.TenantID,
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
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrBranchNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get branch by GitHub PR: %w", err)
	}
	return branch, nil
}

// UpdateBranchStatus updates the status of a branch
func (s *Storage) UpdateBranchStatus(ctx context.Context, id uuid.UUID, status BranchStatus, errorMessage *string) error {
	query := `
		UPDATE branching.branches
		SET status = $1, error_message = $2, updated_at = NOW()
		WHERE id = $3`

	return s.WithTenant(ctx, func(tx pgx.Tx) error {
		result, err := tx.Exec(ctx, query, status, errorMessage, id)
		if err != nil {
			return fmt.Errorf("failed to update branch status: %w", err)
		}

		if result.RowsAffected() == 0 {
			return ErrBranchNotFound
		}

		return nil
	})
}

// DeleteBranch marks a branch as deleted (soft delete).
// tenantID scopes the delete to a specific tenant when non-nil; nil means no tenant filter.
func (s *Storage) DeleteBranch(ctx context.Context, id uuid.UUID, tenantID *uuid.UUID) error {
	query := `
		UPDATE branching.branches
		SET status = 'deleted', updated_at = NOW()
		WHERE id = $1 AND type != 'main'`

	args := []any{id}
	if tenantID != nil {
		query += fmt.Sprintf(" AND tenant_id = $%d", len(args)+1)
		args = append(args, *tenantID)
	}

	return s.WithTenant(ctx, func(tx pgx.Tx) error {
		result, err := tx.Exec(ctx, query, args...)
		if err != nil {
			return fmt.Errorf("failed to delete branch: %w", err)
		}

		if result.RowsAffected() == 0 {
			return ErrBranchNotFound
		}

		return nil
	})
}

// SetBranchExpiresAt sets the expiration time for a branch
func (s *Storage) SetBranchExpiresAt(ctx context.Context, id uuid.UUID, expiresAt *time.Time) error {
	query := `
		UPDATE branching.branches
		SET expires_at = $1, updated_at = NOW()
		WHERE id = $2`

	return s.WithTenant(ctx, func(tx pgx.Tx) error {
		result, err := tx.Exec(ctx, query, expiresAt, id)
		if err != nil {
			return fmt.Errorf("failed to set branch expiration: %w", err)
		}

		if result.RowsAffected() == 0 {
			return ErrBranchNotFound
		}

		return nil
	})
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

// GenerateTenantBranchDatabaseName generates a database name for a tenant-scoped branch
func GenerateTenantBranchDatabaseName(prefix, tenantSlug, branchSlug string) string {
	// tenantSlug: "acme-corp" or "default"
	// branchSlug: "my-feature"
	// Result: "branch_acme_corp_my_feature" or "branch_default_my_feature"

	sanitizedTenant := strings.ReplaceAll(tenantSlug, "-", "_")
	sanitizedBranch := strings.ReplaceAll(branchSlug, "-", "_")
	name := fmt.Sprintf("%s%s_%s", prefix, sanitizedTenant, sanitizedBranch)

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
