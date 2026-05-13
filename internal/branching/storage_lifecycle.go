package branching

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/nimbleflux/fluxbase/internal/crypto"
)

// LogActivity records an activity log entry
func (s *Storage) LogActivity(ctx context.Context, log *ActivityLog) error {
	query := `
		INSERT INTO branching.activity_log (
			id, branch_id, tenant_id, action, status, details, error_message, executed_by, duration_ms
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
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

	return s.WithTenant(ctx, func(tx pgx.Tx) error {
		return tx.QueryRow(
			ctx, query,
			log.ID,
			log.BranchID,
			log.TenantID,
			log.Action,
			log.Status,
			detailsJSON,
			log.ErrorMessage,
			log.ExecutedBy,
			log.DurationMs,
		).Scan(&log.ExecutedAt)
	})
}

// GetActivityLog retrieves activity logs for a branch
func (s *Storage) GetActivityLog(ctx context.Context, branchID uuid.UUID, limit int) ([]*ActivityLog, error) {
	if limit <= 0 {
		limit = 50
	}

	query := `
		SELECT id, branch_id, tenant_id, action, status, details, error_message, executed_by, executed_at, duration_ms
		FROM branching.activity_log
		WHERE branch_id = $1
		ORDER BY executed_at DESC
		LIMIT $2`

	var logs []*ActivityLog
	err := s.WithTenant(ctx, func(tx pgx.Tx) error {
		rows, err := tx.Query(ctx, query, branchID, limit)
		if err != nil {
			return fmt.Errorf("failed to get activity log: %w", err)
		}
		defer rows.Close()

		for rows.Next() {
			log := &ActivityLog{}
			var detailsJSON []byte
			err := rows.Scan(
				&log.ID,
				&log.BranchID,
				&log.TenantID,
				&log.Action,
				&log.Status,
				&detailsJSON,
				&log.ErrorMessage,
				&log.ExecutedBy,
				&log.ExecutedAt,
				&log.DurationMs,
			)
			if err != nil {
				return fmt.Errorf("failed to scan activity log: %w", err)
			}
			if detailsJSON != nil {
				if err := json.Unmarshal(detailsJSON, &log.Details); err != nil {
					return fmt.Errorf("failed to unmarshal details: %w", err)
				}
			}
			logs = append(logs, log)
		}

		return rows.Err()
	})
	if err != nil {
		return nil, err
	}

	return logs, nil
}

// RecordMigration records a migration applied to a branch
func (s *Storage) RecordMigration(ctx context.Context, branchID uuid.UUID, version int64, name string) error {
	query := `
		INSERT INTO branching.migration_history (branch_id, migration_version, migration_name)
		VALUES ($1, $2, $3)
		ON CONFLICT (branch_id, migration_version) DO NOTHING`

	return s.WithTenant(ctx, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx, query, branchID, version, name)
		if err != nil {
			return fmt.Errorf("failed to record migration: %w", err)
		}
		return nil
	})
}

// GetMigrationHistory retrieves the migration history for a branch
func (s *Storage) GetMigrationHistory(ctx context.Context, branchID uuid.UUID) ([]*MigrationHistory, error) {
	query := `
		SELECT id, branch_id, migration_version, migration_name, applied_at
		FROM branching.migration_history
		WHERE branch_id = $1
		ORDER BY migration_version ASC`

	var history []*MigrationHistory
	err := s.WithTenant(ctx, func(tx pgx.Tx) error {
		rows, err := tx.Query(ctx, query, branchID)
		if err != nil {
			return fmt.Errorf("failed to get migration history: %w", err)
		}
		defer rows.Close()

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
				return fmt.Errorf("failed to scan migration history: %w", err)
			}
			history = append(history, mh)
		}

		return rows.Err()
	})
	if err != nil {
		return nil, err
	}

	return history, nil
}

// GitHub Config methods

// GetGitHubConfig retrieves GitHub config for a repository
func (s *Storage) GetGitHubConfig(ctx context.Context, repository string) (*GitHubConfig, error) {
	query := `
		SELECT id, repository, tenant_id, auto_create_on_pr, auto_delete_on_merge,
			default_data_clone_mode, webhook_secret, created_at, updated_at
		FROM branching.github_config
		WHERE repository = $1`

	config := &GitHubConfig{}
	var encryptedSecret *string
	err := s.WithTenant(ctx, func(tx pgx.Tx) error {
		return tx.QueryRow(ctx, query, repository).Scan(
			&config.ID,
			&config.Repository,
			&config.TenantID,
			&config.AutoCreateOnPR,
			&config.AutoDeleteOnMerge,
			&config.DefaultDataCloneMode,
			&encryptedSecret,
			&config.CreatedAt,
			&config.UpdatedAt,
		)
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrGitHubConfigNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get GitHub config: %w", err)
	}

	if encryptedSecret != nil && *encryptedSecret != "" {
		decrypted, err := crypto.DecryptWithBytesKey(*encryptedSecret, s.encryptionKey)
		if err != nil {
			return nil, fmt.Errorf("failed to decrypt webhook secret: %w", err)
		}
		config.WebhookSecret = &decrypted
	}

	return config, nil
}

// UpsertGitHubConfig creates or updates GitHub config
func (s *Storage) UpsertGitHubConfig(ctx context.Context, config *GitHubConfig) error {
	var encryptedSecret *string
	if config.WebhookSecret != nil && *config.WebhookSecret != "" {
		encrypted, err := crypto.EncryptWithBytesKey(*config.WebhookSecret, s.encryptionKey)
		if err != nil {
			return fmt.Errorf("failed to encrypt webhook secret: %w", err)
		}
		encryptedSecret = &encrypted
	}

	query := `
		INSERT INTO branching.github_config (
			id, repository, tenant_id, auto_create_on_pr, auto_delete_on_merge,
			default_data_clone_mode, webhook_secret
		) VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (repository, tenant_id) DO UPDATE SET
			auto_create_on_pr = EXCLUDED.auto_create_on_pr,
			auto_delete_on_merge = EXCLUDED.auto_delete_on_merge,
			default_data_clone_mode = EXCLUDED.default_data_clone_mode,
			webhook_secret = EXCLUDED.webhook_secret,
			updated_at = NOW()
		RETURNING id, created_at, updated_at`

	if config.ID == uuid.Nil {
		config.ID = uuid.New()
	}

	return s.WithTenant(ctx, func(tx pgx.Tx) error {
		return tx.QueryRow(
			ctx, query,
			config.ID,
			config.Repository,
			config.TenantID,
			config.AutoCreateOnPR,
			config.AutoDeleteOnMerge,
			config.DefaultDataCloneMode,
			encryptedSecret,
		).Scan(&config.ID, &config.CreatedAt, &config.UpdatedAt)
	})
}

// DeleteGitHubConfig deletes GitHub config for a repository
func (s *Storage) DeleteGitHubConfig(ctx context.Context, repository string) error {
	query := `DELETE FROM branching.github_config WHERE repository = $1`

	return s.WithTenant(ctx, func(tx pgx.Tx) error {
		result, err := tx.Exec(ctx, query, repository)
		if err != nil {
			return fmt.Errorf("failed to delete GitHub config: %w", err)
		}

		if result.RowsAffected() == 0 {
			return ErrGitHubConfigNotFound
		}

		return nil
	})
}

// ListGitHubConfigs lists all GitHub configurations
func (s *Storage) ListGitHubConfigs(ctx context.Context, tenantID *uuid.UUID) ([]*GitHubConfig, error) {
	query := `
		SELECT id, repository, tenant_id, auto_create_on_pr, auto_delete_on_merge,
			default_data_clone_mode, webhook_secret, created_at, updated_at
		FROM branching.github_config`

	args := []any{}
	if tenantID != nil {
		query += " WHERE tenant_id = $1"
		args = append(args, *tenantID)
	}
	query += " ORDER BY repository"

	var configs []*GitHubConfig
	err := s.WithTenant(ctx, func(tx pgx.Tx) error {
		rows, err := tx.Query(ctx, query, args...)
		if err != nil {
			return fmt.Errorf("failed to list GitHub configs: %w", err)
		}
		defer rows.Close()

		for rows.Next() {
			config := &GitHubConfig{}
			var encryptedSecret *string
			err := rows.Scan(
				&config.ID,
				&config.Repository,
				&config.TenantID,
				&config.AutoCreateOnPR,
				&config.AutoDeleteOnMerge,
				&config.DefaultDataCloneMode,
				&encryptedSecret,
				&config.CreatedAt,
				&config.UpdatedAt,
			)
			if err != nil {
				return fmt.Errorf("failed to scan GitHub config: %w", err)
			}

			if encryptedSecret != nil && *encryptedSecret != "" {
				decrypted, err := crypto.DecryptWithBytesKey(*encryptedSecret, s.encryptionKey)
				if err != nil {
					return fmt.Errorf("failed to decrypt webhook secret: %w", err)
				}
				config.WebhookSecret = &decrypted
			}

			configs = append(configs, config)
		}

		return rows.Err()
	})
	if err != nil {
		return nil, err
	}

	return configs, nil
}

// Branch Access methods

// GrantAccess grants a user access to a branch
func (s *Storage) GrantAccess(ctx context.Context, access *BranchAccess) error {
	query := `
		INSERT INTO branching.branch_access (id, branch_id, tenant_id, user_id, access_level, granted_by)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (branch_id, user_id) DO UPDATE SET
			access_level = EXCLUDED.access_level,
			granted_by = EXCLUDED.granted_by,
			granted_at = NOW()
		RETURNING id, granted_at`

	if access.ID == uuid.Nil {
		access.ID = uuid.New()
	}

	return s.WithTenant(ctx, func(tx pgx.Tx) error {
		return tx.QueryRow(
			ctx, query,
			access.ID,
			access.BranchID,
			access.TenantID,
			access.UserID,
			access.AccessLevel,
			access.GrantedBy,
		).Scan(&access.ID, &access.GrantedAt)
	})
}

// RevokeAccess revokes a user's access to a branch
func (s *Storage) RevokeAccess(ctx context.Context, branchID, userID uuid.UUID) error {
	query := `DELETE FROM branching.branch_access WHERE branch_id = $1 AND user_id = $2`

	return s.WithTenant(ctx, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx, query, branchID, userID)
		return err
	})
}

// GetBranchAccessList returns all access grants for a branch
func (s *Storage) GetBranchAccessList(ctx context.Context, branchID uuid.UUID) ([]*BranchAccess, error) {
	query := `
		SELECT id, branch_id, tenant_id, user_id, access_level, granted_at, granted_by
		FROM branching.branch_access
		WHERE branch_id = $1
		ORDER BY granted_at DESC`

	var accessList []*BranchAccess
	err := s.WithTenant(ctx, func(tx pgx.Tx) error {
		rows, err := tx.Query(ctx, query, branchID)
		if err != nil {
			return fmt.Errorf("failed to list branch access: %w", err)
		}
		defer rows.Close()

		for rows.Next() {
			access := &BranchAccess{}
			if err := rows.Scan(
				&access.ID,
				&access.BranchID,
				&access.TenantID,
				&access.UserID,
				&access.AccessLevel,
				&access.GrantedAt,
				&access.GrantedBy,
			); err != nil {
				return fmt.Errorf("failed to scan branch access: %w", err)
			}
			accessList = append(accessList, access)
		}

		return rows.Err()
	})
	if err != nil {
		return nil, err
	}

	return accessList, nil
}

// GetUserAccess returns the access level for a specific user on a branch
func (s *Storage) GetUserAccess(ctx context.Context, branchID, userID uuid.UUID) (*BranchAccess, error) {
	query := `
		SELECT id, branch_id, tenant_id, user_id, access_level, granted_at, granted_by
		FROM branching.branch_access
		WHERE branch_id = $1 AND user_id = $2`

	access := &BranchAccess{}
	err := s.WithTenant(ctx, func(tx pgx.Tx) error {
		return tx.QueryRow(ctx, query, branchID, userID).Scan(
			&access.ID,
			&access.BranchID,
			&access.TenantID,
			&access.UserID,
			&access.AccessLevel,
			&access.GrantedAt,
			&access.GrantedBy,
		)
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrBranchNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get user access: %w", err)
	}

	return access, nil
}

// HasAccess checks if a user has at least the specified access level to a branch
func (s *Storage) HasAccess(ctx context.Context, branchID, userID uuid.UUID, minLevel BranchAccessLevel) (bool, error) {
	var result bool
	err := s.WithTenant(ctx, func(tx pgx.Tx) error {
		// First check if user is the creator (always has admin access)
		var createdBy *uuid.UUID
		err := tx.QueryRow(
			ctx,
			`SELECT created_by FROM branching.branches WHERE id = $1`,
			branchID,
		).Scan(&createdBy)
		if err != nil {
			return fmt.Errorf("failed to check branch creator: %w", err)
		}

		if createdBy != nil && *createdBy == userID {
			result = true
			return nil
		}

		// Then check explicit access grants
		query := `
			SELECT access_level FROM branching.branch_access
			WHERE branch_id = $1 AND user_id = $2`

		var accessLevel BranchAccessLevel
		err = tx.QueryRow(ctx, query, branchID, userID).Scan(&accessLevel)
		if errors.Is(err, pgx.ErrNoRows) {
			result = false
			return nil
		}
		if err != nil {
			return fmt.Errorf("failed to check access: %w", err)
		}

		// Check if access level is sufficient
		result = isAccessSufficient(accessLevel, minLevel)
		return nil
	})
	if err != nil {
		return false, err
	}

	return result, nil
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
	// Get the branch first (no tenant filter — access check is cross-tenant for admin users)
	branch, err := s.GetBranchBySlug(ctx, slug, nil)
	if err != nil {
		if errors.Is(err, ErrBranchNotFound) {
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
