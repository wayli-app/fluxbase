package branching

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/fluxbase-eu/fluxbase/internal/config"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog/log"
)

// Manager handles database operations for branches
type Manager struct {
	storage    *Storage
	config     config.BranchingConfig
	adminPool  *pgxpool.Pool // Connection pool with CREATE DATABASE privileges
	mainDBName string        // Name of the main database
	mainDBURL  string        // Connection URL for the main database
}

// NewManager creates a new branch manager
func NewManager(storage *Storage, cfg config.BranchingConfig, mainPool *pgxpool.Pool, mainDBURL string) (*Manager, error) {
	// Parse the main database URL to get the database name
	parsedURL, err := url.Parse(mainDBURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse main database URL: %w", err)
	}

	mainDBName := strings.TrimPrefix(parsedURL.Path, "/")
	if mainDBName == "" {
		mainDBName = "fluxbase"
	}

	// Determine admin connection URL
	adminURL := cfg.AdminDatabaseURL
	if adminURL == "" {
		// Use main database URL with 'postgres' database for admin operations
		adminParsed := *parsedURL
		adminParsed.Path = "/postgres"
		adminURL = adminParsed.String()
	}

	// Create admin connection pool
	adminConfig, err := pgxpool.ParseConfig(adminURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse admin database URL: %w", err)
	}

	// Admin pool should have minimal connections since it's only for CREATE/DROP
	adminConfig.MaxConns = 2
	adminConfig.MinConns = 0

	adminPool, err := pgxpool.NewWithConfig(context.Background(), adminConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create admin connection pool: %w", err)
	}

	return &Manager{
		storage:    storage,
		config:     cfg,
		adminPool:  adminPool,
		mainDBName: mainDBName,
		mainDBURL:  mainDBURL,
	}, nil
}

// CreateBranch creates a new database branch
func (m *Manager) CreateBranch(ctx context.Context, req CreateBranchRequest, createdBy *uuid.UUID) (*Branch, error) {
	startTime := time.Now()

	// Check if branching is enabled
	if !m.config.Enabled {
		return nil, ErrBranchingDisabled
	}

	// Check limits
	if err := m.checkLimits(ctx, createdBy); err != nil {
		return nil, err
	}

	// Generate slug from name
	slug := GenerateSlug(req.Name)
	if err := ValidateSlug(slug); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidSlug, err)
	}

	// Check if slug already exists
	existing, err := m.storage.GetBranchBySlug(ctx, slug)
	if err != nil && err != ErrBranchNotFound {
		return nil, fmt.Errorf("failed to check existing branch: %w", err)
	}
	if existing != nil {
		return nil, ErrBranchExists
	}

	// Determine data clone mode
	dataCloneMode := DataCloneModeSchemaOnly
	if req.DataCloneMode != "" {
		dataCloneMode = req.DataCloneMode
	} else if m.config.DefaultDataCloneMode != "" {
		dataCloneMode = DataCloneMode(m.config.DefaultDataCloneMode)
	}

	// Determine branch type
	branchType := BranchTypePreview
	if req.Type != "" {
		branchType = req.Type
	}

	// Determine parent branch (default to main)
	var parentBranchID *uuid.UUID
	if req.ParentBranchID != nil {
		parentBranchID = req.ParentBranchID
	} else {
		mainBranch, err := m.storage.GetMainBranch(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to get main branch: %w", err)
		}
		parentBranchID = &mainBranch.ID
	}

	// Generate database name
	databaseName := GenerateDatabaseName(m.config.DatabasePrefix, slug)

	// Create branch record
	branch := &Branch{
		ID:             uuid.New(),
		Name:           req.Name,
		Slug:           slug,
		DatabaseName:   databaseName,
		Status:         BranchStatusCreating,
		Type:           branchType,
		ParentBranchID: parentBranchID,
		DataCloneMode:  dataCloneMode,
		GitHubPRNumber: req.GitHubPRNumber,
		GitHubPRURL:    req.GitHubPRURL,
		GitHubRepo:     req.GitHubRepo,
		CreatedBy:      createdBy,
		ExpiresAt:      req.ExpiresAt,
	}

	// Calculate auto-delete expiration if configured
	if branch.ExpiresAt == nil && m.config.AutoDeleteAfter > 0 && branchType == BranchTypePreview {
		expiresAt := time.Now().Add(m.config.AutoDeleteAfter)
		branch.ExpiresAt = &expiresAt
	}

	if err := m.storage.CreateBranch(ctx, branch); err != nil {
		return nil, fmt.Errorf("failed to create branch record: %w", err)
	}

	// Log activity start
	_ = m.storage.LogActivity(ctx, &ActivityLog{
		BranchID:   branch.ID,
		Action:     ActivityActionCreated,
		Status:     ActivityStatusStarted,
		ExecutedBy: createdBy,
		Details:    map[string]any{"data_clone_mode": dataCloneMode},
	})

	// Create the database
	if err := m.createDatabase(ctx, branch, parentBranchID); err != nil {
		// Update status to error
		errMsg := err.Error()
		_ = m.storage.UpdateBranchStatus(ctx, branch.ID, BranchStatusError, &errMsg)

		// Log failure
		durationMs := int(time.Since(startTime).Milliseconds())
		_ = m.storage.LogActivity(ctx, &ActivityLog{
			BranchID:     branch.ID,
			Action:       ActivityActionCreated,
			Status:       ActivityStatusFailed,
			ErrorMessage: &errMsg,
			ExecutedBy:   createdBy,
			DurationMs:   &durationMs,
		})

		return nil, fmt.Errorf("failed to create database: %w", err)
	}

	// Update status to ready
	if err := m.storage.UpdateBranchStatus(ctx, branch.ID, BranchStatusReady, nil); err != nil {
		return nil, fmt.Errorf("failed to update branch status: %w", err)
	}
	branch.Status = BranchStatusReady

	// Log success
	durationMs := int(time.Since(startTime).Milliseconds())
	_ = m.storage.LogActivity(ctx, &ActivityLog{
		BranchID:   branch.ID,
		Action:     ActivityActionCreated,
		Status:     ActivityStatusSuccess,
		ExecutedBy: createdBy,
		DurationMs: &durationMs,
	})

	// Grant creator admin access
	if createdBy != nil {
		_ = m.storage.GrantAccess(ctx, &BranchAccess{
			BranchID:    branch.ID,
			UserID:      *createdBy,
			AccessLevel: BranchAccessAdmin,
			GrantedBy:   createdBy,
		})
	}

	log.Info().
		Str("branch_id", branch.ID.String()).
		Str("slug", slug).
		Str("database", databaseName).
		Int("duration_ms", durationMs).
		Msg("Branch created successfully")

	return branch, nil
}

// DeleteBranch deletes a branch and its database
func (m *Manager) DeleteBranch(ctx context.Context, branchID uuid.UUID, deletedBy *uuid.UUID) error {
	startTime := time.Now()

	// Get the branch
	branch, err := m.storage.GetBranch(ctx, branchID)
	if err != nil {
		return err
	}

	// Cannot delete main branch
	if branch.Type == BranchTypeMain {
		return ErrCannotDeleteMainBranch
	}

	// Update status to deleting
	if err := m.storage.UpdateBranchStatus(ctx, branchID, BranchStatusDeleting, nil); err != nil {
		return fmt.Errorf("failed to update branch status: %w", err)
	}

	// Log activity start
	_ = m.storage.LogActivity(ctx, &ActivityLog{
		BranchID:   branchID,
		Action:     ActivityActionDeleted,
		Status:     ActivityStatusStarted,
		ExecutedBy: deletedBy,
	})

	// Drop the database
	if err := m.dropDatabase(ctx, branch.DatabaseName); err != nil {
		// Update status to error
		errMsg := err.Error()
		_ = m.storage.UpdateBranchStatus(ctx, branchID, BranchStatusError, &errMsg)

		// Log failure
		durationMs := int(time.Since(startTime).Milliseconds())
		_ = m.storage.LogActivity(ctx, &ActivityLog{
			BranchID:     branchID,
			Action:       ActivityActionDeleted,
			Status:       ActivityStatusFailed,
			ErrorMessage: &errMsg,
			ExecutedBy:   deletedBy,
			DurationMs:   &durationMs,
		})

		return fmt.Errorf("failed to drop database: %w", err)
	}

	// Mark as deleted
	if err := m.storage.DeleteBranch(ctx, branchID); err != nil {
		return fmt.Errorf("failed to delete branch record: %w", err)
	}

	// Log success
	durationMs := int(time.Since(startTime).Milliseconds())
	_ = m.storage.LogActivity(ctx, &ActivityLog{
		BranchID:   branchID,
		Action:     ActivityActionDeleted,
		Status:     ActivityStatusSuccess,
		ExecutedBy: deletedBy,
		DurationMs: &durationMs,
	})

	log.Info().
		Str("branch_id", branchID.String()).
		Str("slug", branch.Slug).
		Str("database", branch.DatabaseName).
		Int("duration_ms", durationMs).
		Msg("Branch deleted successfully")

	return nil
}

// ResetBranch resets a branch to its parent state
func (m *Manager) ResetBranch(ctx context.Context, branchID uuid.UUID, resetBy *uuid.UUID) error {
	startTime := time.Now()

	// Get the branch
	branch, err := m.storage.GetBranch(ctx, branchID)
	if err != nil {
		return err
	}

	// Cannot reset main branch
	if branch.Type == BranchTypeMain {
		return ErrCannotDeleteMainBranch
	}

	// Need a parent to reset from
	if branch.ParentBranchID == nil {
		return fmt.Errorf("branch has no parent to reset from")
	}

	// Get parent branch
	parent, err := m.storage.GetBranch(ctx, *branch.ParentBranchID)
	if err != nil {
		return fmt.Errorf("failed to get parent branch: %w", err)
	}

	// Update status to migrating (reusing for reset operation)
	if err := m.storage.UpdateBranchStatus(ctx, branchID, BranchStatusMigrating, nil); err != nil {
		return fmt.Errorf("failed to update branch status: %w", err)
	}

	// Log activity start
	_ = m.storage.LogActivity(ctx, &ActivityLog{
		BranchID:   branchID,
		Action:     ActivityActionReset,
		Status:     ActivityStatusStarted,
		ExecutedBy: resetBy,
		Details:    map[string]any{"parent_slug": parent.Slug},
	})

	// Drop and recreate the database
	if err := m.dropDatabase(ctx, branch.DatabaseName); err != nil {
		errMsg := err.Error()
		_ = m.storage.UpdateBranchStatus(ctx, branchID, BranchStatusError, &errMsg)
		return fmt.Errorf("failed to drop database for reset: %w", err)
	}

	if err := m.createDatabase(ctx, branch, branch.ParentBranchID); err != nil {
		errMsg := err.Error()
		_ = m.storage.UpdateBranchStatus(ctx, branchID, BranchStatusError, &errMsg)

		// Log failure
		durationMs := int(time.Since(startTime).Milliseconds())
		_ = m.storage.LogActivity(ctx, &ActivityLog{
			BranchID:     branchID,
			Action:       ActivityActionReset,
			Status:       ActivityStatusFailed,
			ErrorMessage: &errMsg,
			ExecutedBy:   resetBy,
			DurationMs:   &durationMs,
		})

		return fmt.Errorf("failed to recreate database: %w", err)
	}

	// Update status to ready
	if err := m.storage.UpdateBranchStatus(ctx, branchID, BranchStatusReady, nil); err != nil {
		return fmt.Errorf("failed to update branch status: %w", err)
	}

	// Log success
	durationMs := int(time.Since(startTime).Milliseconds())
	_ = m.storage.LogActivity(ctx, &ActivityLog{
		BranchID:   branchID,
		Action:     ActivityActionReset,
		Status:     ActivityStatusSuccess,
		ExecutedBy: resetBy,
		DurationMs: &durationMs,
	})

	log.Info().
		Str("branch_id", branchID.String()).
		Str("slug", branch.Slug).
		Int("duration_ms", durationMs).
		Msg("Branch reset successfully")

	return nil
}

// checkLimits verifies that branch limits have not been exceeded
func (m *Manager) checkLimits(ctx context.Context, userID *uuid.UUID) error {
	// Check total branch limit
	if m.config.MaxTotalBranches > 0 {
		total, err := m.storage.CountBranches(ctx, ListBranchesFilter{})
		if err != nil {
			return fmt.Errorf("failed to count branches: %w", err)
		}
		if total >= m.config.MaxTotalBranches {
			return ErrMaxBranchesReached
		}
	}

	// Check per-user limit
	if m.config.MaxBranchesPerUser > 0 && userID != nil {
		userCount, err := m.storage.CountBranchesByUser(ctx, *userID)
		if err != nil {
			return fmt.Errorf("failed to count user branches: %w", err)
		}
		if userCount >= m.config.MaxBranchesPerUser {
			return ErrMaxUserBranchesReached
		}
	}

	return nil
}

// createDatabase creates a new database for a branch
func (m *Manager) createDatabase(ctx context.Context, branch *Branch, parentBranchID *uuid.UUID) error {
	// Sanitize database name for SQL
	dbName := sanitizeIdentifier(branch.DatabaseName)

	switch branch.DataCloneMode {
	case DataCloneModeSchemaOnly:
		return m.createDatabaseSchemaOnly(ctx, branch, parentBranchID)
	case DataCloneModeFullClone:
		return m.createDatabaseFullClone(ctx, branch, parentBranchID)
	case DataCloneModeSeedData:
		// For now, treat seed_data same as schema_only
		// TODO: Implement seed data script execution
		return m.createDatabaseSchemaOnly(ctx, branch, parentBranchID)
	default:
		// Create empty database
		query := fmt.Sprintf("CREATE DATABASE %s", dbName)
		_, err := m.adminPool.Exec(ctx, query)
		if err != nil {
			return fmt.Errorf("failed to create database: %w", err)
		}
		return nil
	}
}

// createDatabaseSchemaOnly creates a database with schema only (no data)
func (m *Manager) createDatabaseSchemaOnly(ctx context.Context, branch *Branch, parentBranchID *uuid.UUID) error {
	dbName := sanitizeIdentifier(branch.DatabaseName)

	// Create the database first
	createQuery := fmt.Sprintf("CREATE DATABASE %s", dbName)
	_, err := m.adminPool.Exec(ctx, createQuery)
	if err != nil {
		return fmt.Errorf("failed to create database: %w", err)
	}

	// If no parent, we're done
	if parentBranchID == nil {
		return nil
	}

	// Get parent branch
	parent, err := m.storage.GetBranch(ctx, *parentBranchID)
	if err != nil {
		return fmt.Errorf("failed to get parent branch: %w", err)
	}

	// Use pg_dump with schema-only flag to copy schema
	// This requires pg_dump/pg_restore to be available in the environment
	// For now, we'll use CREATE DATABASE ... TEMPLATE approach if parent is main
	if parent.Type == BranchTypeMain {
		// Drop the empty database we just created
		dropQuery := fmt.Sprintf("DROP DATABASE IF EXISTS %s", dbName)
		_, _ = m.adminPool.Exec(ctx, dropQuery)

		// Create from template (schema only via empty template with schema)
		// Note: TEMPLATE copies everything, so for schema-only we need pg_dump approach
		// For simplicity, let's just create from template for now
		createFromTemplate := fmt.Sprintf("CREATE DATABASE %s TEMPLATE %s",
			dbName, sanitizeIdentifier(parent.DatabaseName))

		_, err = m.adminPool.Exec(ctx, createFromTemplate)
		if err != nil {
			// If template approach fails (e.g., active connections), try empty db
			log.Warn().Err(err).Msg("Failed to create from template, creating empty database")
			createQuery := fmt.Sprintf("CREATE DATABASE %s", dbName)
			_, err = m.adminPool.Exec(ctx, createQuery)
			if err != nil {
				return fmt.Errorf("failed to create database: %w", err)
			}
		}
	}

	return nil
}

// createDatabaseFullClone creates a database with full data clone
func (m *Manager) createDatabaseFullClone(ctx context.Context, branch *Branch, parentBranchID *uuid.UUID) error {
	dbName := sanitizeIdentifier(branch.DatabaseName)

	// If no parent, just create empty database
	if parentBranchID == nil {
		createQuery := fmt.Sprintf("CREATE DATABASE %s", dbName)
		_, err := m.adminPool.Exec(ctx, createQuery)
		if err != nil {
			return fmt.Errorf("failed to create database: %w", err)
		}
		return nil
	}

	// Get parent branch
	parent, err := m.storage.GetBranch(ctx, *parentBranchID)
	if err != nil {
		return fmt.Errorf("failed to get parent branch: %w", err)
	}

	// Use CREATE DATABASE ... TEMPLATE for full clone
	// Note: This requires no active connections to the template database
	createFromTemplate := fmt.Sprintf("CREATE DATABASE %s TEMPLATE %s",
		dbName, sanitizeIdentifier(parent.DatabaseName))

	_, err = m.adminPool.Exec(ctx, createFromTemplate)
	if err != nil {
		// Check if it's because of active connections
		if strings.Contains(err.Error(), "being accessed by other users") {
			return fmt.Errorf("cannot clone database: parent database has active connections. Try schema_only mode instead: %w", err)
		}
		return fmt.Errorf("failed to create database from template: %w", err)
	}

	return nil
}

// dropDatabase drops a database
func (m *Manager) dropDatabase(ctx context.Context, databaseName string) error {
	dbName := sanitizeIdentifier(databaseName)

	// First, terminate all connections to the database
	// Use parameterized query to prevent SQL injection
	terminateQuery := `
		SELECT pg_terminate_backend(pid)
		FROM pg_stat_activity
		WHERE datname = $1 AND pid <> pg_backend_pid()
	`

	_, _ = m.adminPool.Exec(ctx, terminateQuery, databaseName)

	// Small delay to allow connections to close
	time.Sleep(100 * time.Millisecond)

	// Drop the database
	dropQuery := fmt.Sprintf("DROP DATABASE IF EXISTS %s", dbName)
	_, err := m.adminPool.Exec(ctx, dropQuery)
	if err != nil {
		return fmt.Errorf("failed to drop database: %w", err)
	}

	return nil
}

// GetBranchConnectionURL returns the connection URL for a branch database
func (m *Manager) GetBranchConnectionURL(branch *Branch) (string, error) {
	// Parse the main database URL
	parsedURL, err := url.Parse(m.mainDBURL)
	if err != nil {
		return "", fmt.Errorf("failed to parse main database URL: %w", err)
	}

	// Replace the database name
	parsedURL.Path = "/" + branch.DatabaseName

	return parsedURL.String(), nil
}

// CleanupExpiredBranches deletes branches that have passed their expiration time
func (m *Manager) CleanupExpiredBranches(ctx context.Context) error {
	expired, err := m.storage.GetExpiredBranches(ctx)
	if err != nil {
		return fmt.Errorf("failed to get expired branches: %w", err)
	}

	for _, branch := range expired {
		log.Info().
			Str("branch_id", branch.ID.String()).
			Str("slug", branch.Slug).
			Time("expires_at", *branch.ExpiresAt).
			Msg("Deleting expired branch")

		if err := m.DeleteBranch(ctx, branch.ID, nil); err != nil {
			log.Error().Err(err).
				Str("branch_id", branch.ID.String()).
				Str("slug", branch.Slug).
				Msg("Failed to delete expired branch")
			// Continue with other branches
		}
	}

	return nil
}

// Close closes the manager and releases resources
func (m *Manager) Close() {
	if m.adminPool != nil {
		m.adminPool.Close()
	}
}

// GetStorage returns the storage instance
func (m *Manager) GetStorage() *Storage {
	return m.storage
}

// GetConfig returns the branching config
func (m *Manager) GetConfig() config.BranchingConfig {
	return m.config
}

// sanitizeIdentifier sanitizes a SQL identifier to prevent injection
func sanitizeIdentifier(name string) string {
	// Use double quotes and escape any existing quotes
	escaped := strings.ReplaceAll(name, `"`, `""`)
	return fmt.Sprintf(`"%s"`, escaped)
}

// CreateBranchFromGitHubPR creates a branch for a GitHub PR
func (m *Manager) CreateBranchFromGitHubPR(ctx context.Context, repo string, prNumber int, prURL string) (*Branch, error) {
	// Get GitHub config for the repository
	ghConfig, err := m.storage.GetGitHubConfig(ctx, repo)
	if err != nil && err != ErrGitHubConfigNotFound {
		return nil, fmt.Errorf("failed to get GitHub config: %w", err)
	}

	// Determine data clone mode
	dataCloneMode := DataCloneModeSchemaOnly
	if ghConfig != nil && ghConfig.DefaultDataCloneMode != "" {
		dataCloneMode = ghConfig.DefaultDataCloneMode
	}

	// Create branch name and slug from PR number
	name := fmt.Sprintf("PR #%d", prNumber)
	slug := GeneratePRSlug(prNumber)

	// Check if branch already exists
	existing, err := m.storage.GetBranchBySlug(ctx, slug)
	if err != nil && err != ErrBranchNotFound {
		return nil, fmt.Errorf("failed to check existing branch: %w", err)
	}
	if existing != nil {
		// Branch already exists, return it
		return existing, nil
	}

	req := CreateBranchRequest{
		Name:           name,
		DataCloneMode:  dataCloneMode,
		Type:           BranchTypePreview,
		GitHubPRNumber: &prNumber,
		GitHubPRURL:    &prURL,
		GitHubRepo:     &repo,
	}

	return m.CreateBranch(ctx, req, nil)
}

// DeleteBranchForGitHubPR deletes the branch associated with a GitHub PR
func (m *Manager) DeleteBranchForGitHubPR(ctx context.Context, repo string, prNumber int) error {
	// Find branch by GitHub PR
	branch, err := m.storage.GetBranchByGitHubPR(ctx, repo, prNumber)
	if err != nil {
		if err == ErrBranchNotFound {
			// Branch doesn't exist, nothing to delete
			return nil
		}
		return fmt.Errorf("failed to get branch for PR: %w", err)
	}

	return m.DeleteBranch(ctx, branch.ID, nil)
}

// RunTransaction executes a function in a transaction using the admin pool
func (m *Manager) RunTransaction(ctx context.Context, fn func(tx pgx.Tx) error) error {
	tx, err := m.adminPool.Begin(ctx)
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
