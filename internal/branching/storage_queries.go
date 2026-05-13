package branching

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// GetMainBranch retrieves the main branch
func (s *Storage) GetMainBranch(ctx context.Context) (*Branch, error) {
	query := `
		SELECT id, name, slug, database_name, status, type, tenant_id, parent_branch_id,
			data_clone_mode, github_pr_number, github_pr_url, github_repo,
			error_message, created_by, created_at, updated_at, expires_at
		FROM branching.branches
		WHERE type = 'main' AND status != 'deleted'
		LIMIT 1`

	branch := &Branch{}
	err := s.WithTenant(ctx, func(tx pgx.Tx) error {
		return tx.QueryRow(ctx, query).Scan(
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
		return nil, fmt.Errorf("failed to get main branch: %w", err)
	}
	return branch, nil
}

// ListBranches lists branches with optional filtering
func (s *Storage) ListBranches(ctx context.Context, filter ListBranchesFilter) ([]*Branch, error) {
	query := `
		SELECT id, name, slug, database_name, status, type, tenant_id, parent_branch_id,
			data_clone_mode, github_pr_number, github_pr_url, github_repo,
			error_message, created_by, created_at, updated_at, expires_at
		FROM branching.branches
		WHERE status != 'deleted'`

	args := []any{}
	argCounter := 1

	if filter.TenantID != nil {
		query += fmt.Sprintf(" AND tenant_id = $%d", argCounter)
		args = append(args, *filter.TenantID)
		argCounter++
	}

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

	var branches []*Branch
	err := s.WithTenant(ctx, func(tx pgx.Tx) error {
		rows, err := tx.Query(ctx, query, args...)
		if err != nil {
			return fmt.Errorf("failed to list branches: %w", err)
		}
		defer rows.Close()

		for rows.Next() {
			branch := &Branch{}
			err := rows.Scan(
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
			if err != nil {
				return fmt.Errorf("failed to scan branch: %w", err)
			}
			branches = append(branches, branch)
		}

		return rows.Err()
	})
	if err != nil {
		return nil, err
	}

	return branches, nil
}

// CountBranches counts branches matching the filter
func (s *Storage) CountBranches(ctx context.Context, filter ListBranchesFilter) (int, error) {
	query := `SELECT COUNT(*) FROM branching.branches WHERE status != 'deleted'`

	args := []any{}
	argCounter := 1

	if filter.TenantID != nil {
		query += fmt.Sprintf(" AND tenant_id = $%d", argCounter)
		args = append(args, *filter.TenantID)
		argCounter++
	}

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
	err := s.WithTenant(ctx, func(tx pgx.Tx) error {
		return tx.QueryRow(ctx, query, args...).Scan(&count)
	})
	if err != nil {
		return 0, fmt.Errorf("failed to count branches: %w", err)
	}

	return count, nil
}

// CountBranchesByUser counts branches created by a specific user
func (s *Storage) CountBranchesByUser(ctx context.Context, userID uuid.UUID) (int, error) {
	query := `SELECT COUNT(*) FROM branching.branches WHERE created_by = $1 AND status NOT IN ('deleted', 'deleting')`

	var count int
	err := s.WithTenant(ctx, func(tx pgx.Tx) error {
		return tx.QueryRow(ctx, query, userID).Scan(&count)
	})
	if err != nil {
		return 0, fmt.Errorf("failed to count user branches: %w", err)
	}

	return count, nil
}

// CountBranchesByTenant counts branches for a specific tenant
func (s *Storage) CountBranchesByTenant(ctx context.Context, tenantID uuid.UUID) (int, error) {
	query := `SELECT COUNT(*) FROM branching.branches WHERE tenant_id = $1 AND status NOT IN ('deleted', 'deleting')`

	var count int
	err := s.WithTenant(ctx, func(tx pgx.Tx) error {
		return tx.QueryRow(ctx, query, tenantID).Scan(&count)
	})
	if err != nil {
		return 0, fmt.Errorf("failed to count tenant branches: %w", err)
	}

	return count, nil
}

// GetExpiredBranches returns branches that have passed their expiration time
func (s *Storage) GetExpiredBranches(ctx context.Context) ([]*Branch, error) {
	query := `
		SELECT id, name, slug, database_name, status, type, tenant_id, parent_branch_id,
			data_clone_mode, github_pr_number, github_pr_url, github_repo,
			error_message, created_by, created_at, updated_at, expires_at
		FROM branching.branches
		WHERE expires_at IS NOT NULL
			AND expires_at < NOW()
			AND status NOT IN ('deleted', 'deleting')
			AND type != 'main'`

	var branches []*Branch
	err := s.WithTenant(ctx, func(tx pgx.Tx) error {
		rows, err := tx.Query(ctx, query)
		if err != nil {
			return fmt.Errorf("failed to get expired branches: %w", err)
		}
		defer rows.Close()

		for rows.Next() {
			branch := &Branch{}
			err := rows.Scan(
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
			if err != nil {
				return fmt.Errorf("failed to scan expired branch: %w", err)
			}
			branches = append(branches, branch)
		}

		return rows.Err()
	})
	if err != nil {
		return nil, err
	}

	return branches, nil
}
