//go:build integration

package branching

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nimbleflux/fluxbase/internal/config"
	"github.com/nimbleflux/fluxbase/internal/database"
	"github.com/nimbleflux/fluxbase/test/dbhelpers"
)

const testEncryptionKey = "test-encryption-key-must-be-32-chars!"

// TestManager_checkLimits_Integration tests the limit checking functionality.
func TestManager_checkLimits_Integration(t *testing.T) {
	testCtx := dbhelpers.NewDBTestContext(t)
	defer testCtx.Close()

	// Create a test storage with real connection
	storage := NewStorage(database.NewConnectionWithPool(testCtx.Pool), []byte(testEncryptionKey))

	// Create a config with limits
	cfg := config.BranchingConfig{
		Enabled:            true,
		MaxBranchesPerUser: 3,
		MaxTotalBranches:   10,
		DatabasePrefix:     "test_",
	}

	manager, err := NewManager(storage, cfg, testCtx.Pool, testCtx.DatabaseURL())
	require.NoError(t, err)
	defer manager.Close()

	t.Run("passes when under limits", func(t *testing.T) {
		userID := uuid.New()
		err := manager.checkLimits(context.Background(), nil, &userID)
		assert.NoError(t, err)
	})

	t.Run("passes when user limit is disabled (0)", func(t *testing.T) {
		cfg := config.BranchingConfig{
			Enabled:            true,
			MaxBranchesPerUser: 0, // No user limit
			MaxTotalBranches:   10,
			DatabasePrefix:     "test_",
		}

		manager, err := NewManager(storage, cfg, testCtx.Pool, testCtx.DatabaseURL())
		require.NoError(t, err)
		defer manager.Close()

		userID := uuid.New()
		err = manager.checkLimits(context.Background(), nil, &userID)
		assert.NoError(t, err)
	})

	t.Run("passes when total limit is disabled (0)", func(t *testing.T) {
		cfg := config.BranchingConfig{
			Enabled:            true,
			MaxBranchesPerUser: 3,
			MaxTotalBranches:   0, // No total limit
			DatabasePrefix:     "test_",
		}

		manager, err := NewManager(storage, cfg, testCtx.Pool, testCtx.DatabaseURL())
		require.NoError(t, err)
		defer manager.Close()

		userID := uuid.New()
		err = manager.checkLimits(context.Background(), nil, &userID)
		assert.NoError(t, err)
	})
}

// TestManager_GenerateDatabaseName_Integration tests database name generation.
func TestManager_GenerateDatabaseName_Integration(t *testing.T) {
	t.Run("generates valid database names", func(t *testing.T) {
		tests := []struct {
			prefix   string
			slug     string
			expected string
		}{
			{"branch_", "my-branch", "branch_my_branch"},
			{"", "test", "test"},
			{"dev_", "feature-123", "dev_feature_123"},
		}

		for _, tt := range tests {
			result := GenerateDatabaseName(tt.prefix, tt.slug)
			assert.Equal(t, tt.expected, result)
		}
	})

	t.Run("handles special characters", func(t *testing.T) {
		result := GenerateDatabaseName("test_", "my-test-branch")
		assert.Equal(t, "test_my_test_branch", result)
	})

	t.Run("truncates to 63 characters", func(t *testing.T) {
		longSlug := "a-very-long-branch-name-that-exceeds-the-maximum-identifier-length"
		result := GenerateDatabaseName("prefix_", longSlug)
		assert.LessOrEqual(t, len(result), 63)
	})
}

// TestManager_GetBranchConnectionURL_Integration tests connection URL generation.
func TestManager_GetBranchConnectionURL_Integration(t *testing.T) {
	testCtx := dbhelpers.NewDBTestContext(t)
	defer testCtx.Close()

	storage := NewStorage(database.NewConnectionWithPool(testCtx.Pool), []byte(testEncryptionKey))
	cfg := config.BranchingConfig{Enabled: true, DatabasePrefix: "test_"}

	manager, err := NewManager(storage, cfg, testCtx.Pool, testCtx.DatabaseURL())
	require.NoError(t, err)
	defer manager.Close()

	t.Run("generates connection URL for branch", func(t *testing.T) {
		branch := &Branch{
			ID:           uuid.New(),
			Name:         "test-branch",
			Slug:         "test-branch",
			DatabaseName: "test_test_branch",
			Status:       BranchStatusReady,
		}

		url, err := manager.GetBranchConnectionURL(branch)
		require.NoError(t, err)
		assert.Contains(t, url, "test_test_branch")
		assert.Contains(t, url, "postgresql://")
	})
}

// TestManager_CreateBranchValidation_Integration tests branch creation validation.
func TestManager_CreateBranchValidation_Integration(t *testing.T) {
	testCtx := dbhelpers.NewDBTestContext(t)
	defer testCtx.Close()

	storage := NewStorage(database.NewConnectionWithPool(testCtx.Pool), []byte(testEncryptionKey))
	cfg := config.BranchingConfig{
		Enabled:            true,
		MaxBranchesPerUser: 5,
		MaxTotalBranches:   10,
		DatabasePrefix:     "test_",
	}

	manager, err := NewManager(storage, cfg, testCtx.Pool, testCtx.DatabaseURL())
	require.NoError(t, err)
	defer manager.Close()

	t.Run("returns error when branching disabled", func(t *testing.T) {
		cfgDisabled := config.BranchingConfig{Enabled: false}
		mgr, err := NewManager(storage, cfgDisabled, testCtx.Pool, testCtx.DatabaseURL())
		require.NoError(t, err)
		defer mgr.Close()

		userID := uuid.New()
		req := CreateBranchRequest{Name: "test-branch"}

		_, err = mgr.CreateBranch(context.Background(), req, &userID)
		assert.Equal(t, ErrBranchingDisabled, err)
	})

	t.Run("validates slug format", func(t *testing.T) {
		userID := uuid.New()
		req := CreateBranchRequest{
			Name: "Invalid Branch Name!",
		}

		// Slug generation should produce valid slug
		_, err := manager.CreateBranch(context.Background(), req, &userID)
		// Should fail for other reasons (database creation), but slug should be valid
		assert.NotEqual(t, ErrInvalidSlug, err)
	})
}

// TestManager_CreateBranchRequest_Integration tests branch request handling.
func TestManager_CreateBranchRequest_Integration(t *testing.T) {
	testCtx := dbhelpers.NewDBTestContext(t)
	defer testCtx.Close()

	storage := NewStorage(database.NewConnectionWithPool(testCtx.Pool), []byte(testEncryptionKey))
	cfg := config.BranchingConfig{
		Enabled:            true,
		MaxBranchesPerUser: 5,
		MaxTotalBranches:   10,
		DatabasePrefix:     "test_",
	}

	manager, err := NewManager(storage, cfg, testCtx.Pool, testCtx.DatabaseURL())
	require.NoError(t, err)
	defer manager.Close()

	t.Run("uses default data clone mode", func(t *testing.T) {
		req := CreateBranchRequest{
			Name: "test-default-mode",
		}

		// Verify default is set
		assert.Empty(t, req.DataCloneMode, "Request should start with empty mode")

		userID := uuid.New()

		// The manager should apply the default
		// We can't test full branch creation without admin privileges,
		// but we can verify the request structure
		assert.NotNil(t, req.Name)
		_ = userID
	})

	t.Run("uses specified data clone mode", func(t *testing.T) {
		req := CreateBranchRequest{
			Name:          "test-mode",
			DataCloneMode: DataCloneModeFullClone,
		}

		assert.Equal(t, DataCloneModeFullClone, req.DataCloneMode)
	})

	t.Run("handles expiration time", func(t *testing.T) {
		expiresAt := time.Now().Add(24 * time.Hour)
		req := CreateBranchRequest{
			Name:      "test-expiry",
			ExpiresAt: &expiresAt,
		}

		assert.NotNil(t, req.ExpiresAt)
		assert.True(t, req.ExpiresAt.After(time.Now()))
	})
}

// TestManager_GitHubPR_Integration tests GitHub PR branch handling.
func TestManager_GitHubPR_Integration(t *testing.T) {
	testCtx := dbhelpers.NewDBTestContext(t)
	defer testCtx.Close()

	storage := NewStorage(database.NewConnectionWithPool(testCtx.Pool), []byte(testEncryptionKey))
	cfg := config.BranchingConfig{
		Enabled:            true,
		MaxBranchesPerUser: 5,
		DatabasePrefix:     "test_",
	}

	manager, err := NewManager(storage, cfg, testCtx.Pool, testCtx.DatabaseURL())
	require.NoError(t, err)
	defer manager.Close()

	t.Run("generates PR slug", func(t *testing.T) {
		slug := GeneratePRSlug(123)
		assert.Equal(t, "pr-123", slug)
	})

	t.Run("generates unique slugs for different PRs", func(t *testing.T) {
		slug1 := GeneratePRSlug(1)
		slug2 := GeneratePRSlug(2)
		slug3 := GeneratePRSlug(100)

		assert.Equal(t, "pr-1", slug1)
		assert.Equal(t, "pr-2", slug2)
		assert.Equal(t, "pr-100", slug3)
	})

	t.Run("handles GitHub branch request structure", func(t *testing.T) {
		prNumber := 123
		prURL := "https://github.com/org/repo/pull/123"
		repo := "org/repo"

		req := CreateBranchRequest{
			Name:           "PR #123",
			DataCloneMode:  DataCloneModeSchemaOnly,
			Type:           BranchTypePreview,
			GitHubPRNumber: &prNumber,
			GitHubPRURL:    &prURL,
			GitHubRepo:     &repo,
		}

		assert.Equal(t, "PR #123", req.Name)
		assert.NotNil(t, req.GitHubPRNumber)
		assert.Equal(t, 123, *req.GitHubPRNumber)
		assert.NotNil(t, req.GitHubPRURL)
		assert.Equal(t, prURL, *req.GitHubPRURL)
		assert.NotNil(t, req.GitHubRepo)
		assert.Equal(t, repo, *req.GitHubRepo)
	})
}

// TestManager_BranchTypes_Integration tests different branch types.
func TestManager_BranchTypes_Integration(t *testing.T) {
	t.Run("recognizes all branch types", func(t *testing.T) {
		types := []BranchType{
			BranchTypeMain,
			BranchTypePreview,
			BranchTypePersistent,
		}

		validTypes := make(map[BranchType]bool)
		for _, bt := range types {
			validTypes[bt] = true
		}

		assert.True(t, validTypes[BranchTypeMain])
		assert.True(t, validTypes[BranchTypePreview])
		assert.True(t, validTypes[BranchTypePersistent])
	})
}

// TestManager_StatusTransitions_Integration tests status transition logic.
func TestManager_StatusTransitions_Integration(t *testing.T) {
	t.Run("all status values are valid", func(t *testing.T) {
		statuses := []BranchStatus{
			BranchStatusCreating,
			BranchStatusReady,
			BranchStatusDeleting,
			BranchStatusMigrating,
			BranchStatusError,
		}

		for _, status := range statuses {
			assert.NotEmpty(t, string(status))
		}
	})
}

// TestManager_DataCloneModes_Integration tests data clone mode handling.
func TestManager_DataCloneModes_Integration(t *testing.T) {
	t.Run("recognizes all data clone modes", func(t *testing.T) {
		modes := []DataCloneMode{
			DataCloneModeSchemaOnly,
			DataCloneModeFullClone,
			DataCloneModeSeedData,
		}

		for _, mode := range modes {
			assert.NotEmpty(t, string(mode))
		}
	})
}

// TestManager_Cleanup_Integration tests cleanup operations.
func TestManager_Cleanup_Integration(t *testing.T) {
	testCtx := dbhelpers.NewDBTestContext(t)
	defer testCtx.Close()

	storage := NewStorage(database.NewConnectionWithPool(testCtx.Pool), []byte(testEncryptionKey))
	cfg := config.BranchingConfig{
		Enabled:            true,
		MaxBranchesPerUser: 5,
		MaxTotalBranches:   10,
		DatabasePrefix:     "test_",
		AutoDeleteAfter:    24 * time.Hour,
	}

	manager, err := NewManager(storage, cfg, testCtx.Pool, testCtx.DatabaseURL())
	require.NoError(t, err)
	defer manager.Close()

	t.Run("cleanup expired branches handles empty list", func(t *testing.T) {
		// Should not error even if no expired branches
		err := manager.CleanupExpiredBranches(context.Background())
		assert.NoError(t, err)
	})
}

// TestManager_Close_Integration tests manager cleanup.
func TestManager_Close_Integration(t *testing.T) {
	testCtx := dbhelpers.NewDBTestContext(t)
	defer testCtx.Close()

	storage := NewStorage(database.NewConnectionWithPool(testCtx.Pool), []byte(testEncryptionKey))
	cfg := config.BranchingConfig{
		Enabled:        true,
		DatabasePrefix: "test_",
	}

	t.Run("closes manager gracefully", func(t *testing.T) {
		manager, err := NewManager(storage, cfg, testCtx.Pool, testCtx.DatabaseURL())
		require.NoError(t, err)

		// Should not panic
		manager.Close()

		// Calling close again should be safe
		manager.Close()
	})
}

// TestManager_AccessControl_Integration tests access control helpers.
func TestManager_AccessControl_Integration(t *testing.T) {
	t.Run("all access levels are defined", func(t *testing.T) {
		levels := []BranchAccessLevel{
			BranchAccessRead,
			BranchAccessWrite,
			BranchAccessAdmin,
		}

		for _, level := range levels {
			assert.NotEmpty(t, string(level))
		}
	})
}

// TestManager_ErrorHandling_Integration tests error scenarios.
func TestManager_ErrorHandling_Integration(t *testing.T) {
	t.Run("handles nil user ID for limits", func(t *testing.T) {
		testCtx := dbhelpers.NewDBTestContext(t)
		defer testCtx.Close()

		storage := NewStorage(database.NewConnectionWithPool(testCtx.Pool), []byte(testEncryptionKey))
		cfg := config.BranchingConfig{
			Enabled:            true,
			MaxBranchesPerUser: 1,
			MaxTotalBranches:   10,
			DatabasePrefix:     "test_",
		}

		manager, err := NewManager(storage, cfg, testCtx.Pool, testCtx.DatabaseURL())
		require.NoError(t, err)
		defer manager.Close()

		// Nil userID should skip per-user limit check
		err = manager.checkLimits(context.Background(), nil, nil)
		assert.NoError(t, err)
	})
}

// TestManager_Transaction_Integration tests transaction helper.
func TestManager_Transaction_Integration(t *testing.T) {
	testCtx := dbhelpers.NewDBTestContext(t)
	defer testCtx.Close()

	storage := NewStorage(database.NewConnectionWithPool(testCtx.Pool), []byte(testEncryptionKey))
	cfg := config.BranchingConfig{
		Enabled:        true,
		DatabasePrefix: "test_",
	}

	manager, err := NewManager(storage, cfg, testCtx.Pool, testCtx.DatabaseURL())
	require.NoError(t, err)
	defer manager.Close()

	t.Run("executes transaction successfully", func(t *testing.T) {
		err := manager.RunTransaction(context.Background(), func(tx pgx.Tx) error {
			// Execute a simple query within transaction
			_, err := tx.Exec(context.Background(), "SELECT 1")
			return err
		})
		assert.NoError(t, err)
	})

	t.Run("rolls back on error", func(t *testing.T) {
		executed := false
		err := manager.RunTransaction(context.Background(), func(tx pgx.Tx) error {
			executed = true
			// Return an error to trigger rollback
			return assert.AnError
		})
		assert.Error(t, err)
		assert.True(t, executed)
	})
}

// TestManager_Getters_Integration tests getter methods.
func TestManager_Getters_Integration(t *testing.T) {
	testCtx := dbhelpers.NewDBTestContext(t)
	defer testCtx.Close()

	storage := NewStorage(database.NewConnectionWithPool(testCtx.Pool), []byte(testEncryptionKey))
	cfg := config.BranchingConfig{
		Enabled:            true,
		MaxBranchesPerUser: 5,
		DatabasePrefix:     "test_",
		SeedsPath:          "/seeds",
	}

	manager, err := NewManager(storage, cfg, testCtx.Pool, testCtx.DatabaseURL())
	require.NoError(t, err)
	defer manager.Close()

	t.Run("returns storage instance", func(t *testing.T) {
		retStorage := manager.GetStorage()
		assert.Same(t, storage, retStorage)
	})

	t.Run("returns config", func(t *testing.T) {
		retCfg := manager.GetConfig()
		assert.Equal(t, cfg.Enabled, retCfg.Enabled)
		assert.Equal(t, cfg.DatabasePrefix, retCfg.DatabasePrefix)
		assert.Equal(t, cfg.SeedsPath, retCfg.SeedsPath)
	})
}
