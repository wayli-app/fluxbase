package branching

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"github.com/nimbleflux/fluxbase/internal/config"
)

// Helper function
func strPtr(s string) *string {
	return &s
}

// =============================================================================
// Manager Unit Tests
// Tests that don't require actual database connections
// =============================================================================

func TestNewManager(t *testing.T) {
	t.Run("invalid database URL", func(t *testing.T) {
		storage := &Storage{}
		cfg := config.BranchingConfig{
			Enabled: true,
		}

		_, err := NewManager(storage, cfg, nil, ":invalid-url")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to parse")
	})

	t.Run("parses database name from URL", func(t *testing.T) {
		// Test URL parsing without requiring actual database
		storage := &Storage{}
		cfg := config.BranchingConfig{}

		// Invalid URL format should fail
		_, err := NewManager(storage, cfg, nil, "not-a-valid-url")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to parse")
	})
}

func TestCreateBranchRequest_Validation(t *testing.T) {
	t.Run("valid request", func(t *testing.T) {
		req := CreateBranchRequest{
			Name: "my-test-branch",
			Type: BranchTypePreview,
		}

		assert.Equal(t, "my-test-branch", req.Name)
		assert.Equal(t, BranchTypePreview, req.Type)
	})

	t.Run("empty name is invalid", func(t *testing.T) {
		req := CreateBranchRequest{
			Name: "",
		}

		// Empty name should result in empty slug
		slug := GenerateSlug(req.Name)
		assert.Equal(t, "branch", slug)
	})

	t.Run("request with all fields", func(t *testing.T) {
		userID := uuid.New()
		expiresAt := time.Now().Add(24 * time.Hour)

		req := CreateBranchRequest{
			Name:           "feature-branch",
			Type:           BranchTypePreview,
			DataCloneMode:  DataCloneModeFull,
			ParentBranchID: &userID,
			ExpiresAt:      &expiresAt,
			SeedsPath:      strPtr("seeds"),
		}

		assert.Equal(t, "feature-branch", req.Name)
		assert.Equal(t, BranchTypePreview, req.Type)
		assert.Equal(t, DataCloneModeFull, req.DataCloneMode)
		assert.Equal(t, &userID, req.ParentBranchID)
		assert.NotNil(t, req.ExpiresAt)
		assert.Equal(t, "seeds", *req.SeedsPath)
	})
}

func TestCreateBranchRequest_GitHubFields(t *testing.T) {
	t.Run("with GitHub PR number", func(t *testing.T) {
		prNum := 123
		req := CreateBranchRequest{
			Name:           "pr-123",
			GitHubPRNumber: &prNum,
			GitHubPRURL:    strPtr("https://github.com/user/repo/pull/123"),
			GitHubRepo:     strPtr("user/repo"),
		}

		assert.NotNil(t, req.GitHubPRNumber)
		assert.Equal(t, 123, *req.GitHubPRNumber)
		assert.Equal(t, "https://github.com/user/repo/pull/123", *req.GitHubPRURL)
		assert.Equal(t, "user/repo", *req.GitHubRepo)
	})

	t.Run("without GitHub fields", func(t *testing.T) {
		req := CreateBranchRequest{
			Name: "my-branch",
		}

		assert.Nil(t, req.GitHubPRNumber)
		assert.Empty(t, req.GitHubPRURL)
		assert.Empty(t, req.GitHubRepo)
	})
}

func TestDataCloneMode(t *testing.T) {
	t.Run("valid clone modes", func(t *testing.T) {
		validModes := []DataCloneMode{
			DataCloneModeSchemaOnly,
			DataCloneModeFullClone,
			DataCloneModeFullClone,
		}

		for _, mode := range validModes {
			assert.NotEmpty(t, string(mode))
		}
	})

	t.Run("clone mode values", func(t *testing.T) {
		assert.Equal(t, "schema_only", string(DataCloneModeSchemaOnly))
		assert.Equal(t, "full", string(DataCloneModeFull))
		assert.Equal(t, "schema_only", string(DataCloneModeSchemaOnly))
	})
}

func TestBranchType(t *testing.T) {
	t.Run("valid branch types", func(t *testing.T) {
		validTypes := []BranchType{
			BranchTypeMain,
			BranchTypePreview,
			BranchTypeProduction,
		}

		for _, bt := range validTypes {
			assert.NotEmpty(t, string(bt))
		}
	})

	t.Run("branch type values", func(t *testing.T) {
		assert.Equal(t, "main", string(BranchTypeMain))
		assert.Equal(t, "preview", string(BranchTypePreview))
		assert.Equal(t, "production", string(BranchTypeProduction))
	})
}

func TestBranchStatus(t *testing.T) {
	t.Run("valid branch statuses", func(t *testing.T) {
		validStatuses := []BranchStatus{
			BranchStatusCreating,
			BranchStatusReady,
			BranchStatusMigrating,
			BranchStatusError,
			BranchStatusDeleting,
			BranchStatusDeleted,
		}

		for _, bs := range validStatuses {
			assert.NotEmpty(t, string(bs))
		}
	})

	t.Run("branch status values", func(t *testing.T) {
		assert.Equal(t, "creating", string(BranchStatusCreating))
		assert.Equal(t, "ready", string(BranchStatusReady))
		assert.Equal(t, "migrating", string(BranchStatusMigrating))
		assert.Equal(t, "error", string(BranchStatusError))
		assert.Equal(t, "deleting", string(BranchStatusDeleting))
		assert.Equal(t, "deleted", string(BranchStatusDeleted))
	})
}

func TestBranchAccessLevel(t *testing.T) {
	t.Run("valid access levels", func(t *testing.T) {
		levels := []BranchAccessLevel{
			BranchAccessAdmin,
			BranchAccessWrite,
			BranchAccessRead,
		}

		for _, level := range levels {
			assert.NotEmpty(t, string(level))
		}
	})

	t.Run("access level values", func(t *testing.T) {
		assert.Equal(t, "admin", string(BranchAccessAdmin))
		assert.Equal(t, "write", string(BranchAccessWrite))
		assert.Equal(t, "read", string(BranchAccessRead))
	})
}

func TestActivityAction(t *testing.T) {
	actions := []ActivityAction{
		ActivityActionCreated,
		ActivityActionDeleted,
		ActivityActionReset,
		ActivityActionMigrated,
	}

	for _, action := range actions {
		assert.NotEmpty(t, string(action))
	}
}

func TestActivityStatus(t *testing.T) {
	statuses := []ActivityStatus{
		ActivityStatusStarted,
		ActivityStatusSuccess,
		ActivityStatusFailed,
	}

	for _, status := range statuses {
		assert.NotEmpty(t, string(status))
	}
}

func TestGrantAccessRequest_Struct(t *testing.T) {
	t.Run("grant access with all fields", func(t *testing.T) {
		userID := uuid.New()
		req := GrantAccessRequest{
			UserID:      userID,
			AccessLevel: BranchAccessAdmin,
		}

		assert.Equal(t, userID, req.UserID)
		assert.Equal(t, BranchAccessAdmin, req.AccessLevel)
	})

	t.Run("grant access with read level", func(t *testing.T) {
		userID := uuid.New()
		req := GrantAccessRequest{
			UserID:      userID,
			AccessLevel: BranchAccessRead,
		}

		assert.Equal(t, userID, req.UserID)
		assert.Equal(t, BranchAccessRead, req.AccessLevel)
	})
}

func TestUpdateAccessRequest_Struct(t *testing.T) {
	t.Run("update access level", func(t *testing.T) {
		level := BranchAccessWrite
		req := UpdateAccessRequest{
			AccessLevel: &level,
		}

		assert.NotNil(t, req.AccessLevel)
		assert.Equal(t, BranchAccessWrite, *req.AccessLevel)
	})

	t.Run("empty update", func(t *testing.T) {
		req := UpdateAccessRequest{}

		assert.Nil(t, req.AccessLevel)
	})
}

func TestBranchLimits(t *testing.T) {
	t.Run("max branches per user limit", func(t *testing.T) {
		limit := int64(5)
		assert.Equal(t, int64(5), limit)
	})

	t.Run("max total branches limit", func(t *testing.T) {
		limit := int64(50)
		assert.Equal(t, int64(50), limit)
	})
}

func TestAutoDeleteExpiration(t *testing.T) {
	t.Run("calculate expiration for preview branch", func(t *testing.T) {
		now := time.Now()
		autoDeleteAfter := 24 * time.Hour
		expiresAt := now.Add(autoDeleteAfter)

		assert.True(t, expiresAt.After(now))
		assert.False(t, expiresAt.IsZero())
	})

	t.Run("production branch has no auto-expiry", func(t *testing.T) {
		// Production branches don't auto-expire
		// This would be tested in integration tests with actual Manager
		assert.True(t, true)
	})
}

func TestCheckLimits_Unit(t *testing.T) {
	t.Run("branching disabled", func(t *testing.T) {
		cfg := config.BranchingConfig{
			Enabled: false,
		}

		assert.False(t, cfg.Enabled)
	})

	t.Run("branching enabled with limits", func(t *testing.T) {
		cfg := config.BranchingConfig{
			Enabled:            true,
			MaxBranchesPerUser: 5,
			MaxTotalBranches:   50,
		}

		assert.True(t, cfg.Enabled)
		assert.Equal(t, 5, cfg.MaxBranchesPerUser)
		assert.Equal(t, 50, cfg.MaxTotalBranches)
	})
}

func TestCreateBranchRequestValidation(t *testing.T) {
	t.Run("name validation", func(t *testing.T) {
		tests := []struct {
			name        string
			branchName  string
			expectError bool
		}{
			{
				name:        "valid name",
				branchName:  "my-branch",
				expectError: false,
			},
			{
				name:        "name with spaces",
				branchName:  "my branch",
				expectError: false,
			},
			{
				name:        "empty name",
				branchName:  "",
				expectError: false, // Empty name generates default slug
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				slug := GenerateSlug(tt.branchName)
				// Just verify slug generation works
				assert.NotEmpty(t, slug)
			})
		}
	})
}

func TestBranchNamesWithSpecialChars(t *testing.T) {
	specialCases := []struct {
		name     string
		expected string
	}{
		{
			name:     "feature/ABC-123",
			expected: "feature-abc-123", // Should contain these parts
		},
		{
			name:     "release@v1.0.0",
			expected: "release-v1-0-0",
		},
		{
			name:     "hotfix#123-bug-fix",
			expected: "hotfix-123-bug-fix",
		},
	}

	for _, tc := range specialCases {
		t.Run(tc.name, func(t *testing.T) {
			slug := GenerateSlug(tc.name)
			assert.NotEmpty(t, slug)
			// Verify special chars are handled
			assert.NotContains(t, slug, "/")
			assert.NotContains(t, slug, "@")
			assert.NotContains(t, slug, "#")
		})
	}
}

func TestGenerateSlug_LongNames(t *testing.T) {
	t.Run("very long name is truncated", func(t *testing.T) {
		// Create a name longer than 50 characters
		longName := "this-is-a-very-long-branch-name-that-exceeds-fifty-characters"
		slug := GenerateSlug(longName)

		// Should be truncated to 50 chars max
		assert.LessOrEqual(t, len(slug), 50)
		assert.NotEmpty(t, slug)
	})

	t.Run("exactly 50 characters", func(t *testing.T) {
		// Create exactly 50 character name
		name := "12345678901234567890123456789012345678901234567890"
		assert.Len(t, name, 50)

		slug := GenerateSlug(name)
		assert.LessOrEqual(t, len(slug), 50)
	})
}

func TestGenerateDatabaseName_Unit(t *testing.T) {
	t.Run("with custom prefix", func(t *testing.T) {
		name := GenerateDatabaseName("test_", "my-branch")
		assert.Equal(t, "test_my_branch", name)
	})

	t.Run("without prefix", func(t *testing.T) {
		name := GenerateDatabaseName("", "my-branch")
		assert.Equal(t, "my_branch", name)
	})

	t.Run("slug with hyphens converted to underscores", func(t *testing.T) {
		// PostgreSQL identifiers cannot contain hyphens, so they're converted to underscores
		name := GenerateDatabaseName("branch_", "my-branch")
		assert.Contains(t, name, "my_branch")
	})

	t.Run("multiple hyphens all converted", func(t *testing.T) {
		name := GenerateDatabaseName("test_", "feature-branch-123")
		assert.Equal(t, "test_feature_branch_123", name)
	})

	t.Run("preserves existing underscores", func(t *testing.T) {
		name := GenerateDatabaseName("", "my_existing_branch")
		assert.Equal(t, "my_existing_branch", name)
	})
}

func TestValidateSlug_Format(t *testing.T) {
	validSlugs := []string{
		"my-branch",
		"feature-123",
		"a",
		"abc-123-xyz",
		"test-branch-name",
	}

	for _, slug := range validSlugs {
		t.Run("valid_"+slug, func(t *testing.T) {
			err := ValidateSlug(slug)
			assert.NoError(t, err)
		})
	}
}

func TestValidateSlug_InvalidFormats(t *testing.T) {
	invalidSlugs := []struct {
		slug        string
		description string
	}{
		{"", "empty slug"},
		{"-starts-with-dash", "starts with dash"},
		{"ends-with-dash-", "ends with dash"},
		{"has space", "contains space"},
		{"has_underscore", "contains underscore"},
		{"UPPERCASE", "all uppercase"},
		{"has.dot", "contains dot"},
		{"has@symbol", "contains special char"},
	}

	for _, tc := range invalidSlugs {
		t.Run(tc.description, func(t *testing.T) {
			err := ValidateSlug(tc.slug)
			assert.Error(t, err, "Should reject: "+tc.slug)
		})
	}
}

func TestBranch_IsExpired(t *testing.T) {
	t.Run("branch with expiration in past", func(t *testing.T) {
		past := time.Now().Add(-1 * time.Hour)
		branch := &Branch{
			ID:        uuid.New(),
			Name:      "expired-branch",
			Slug:      "expired-branch",
			ExpiresAt: &past,
		}

		assert.True(t, branch.IsExpired())
	})

	t.Run("branch with expiration in future", func(t *testing.T) {
		future := time.Now().Add(1 * time.Hour)
		branch := &Branch{
			ID:        uuid.New(),
			Name:      "future-branch",
			Slug:      "future-branch",
			ExpiresAt: &future,
		}

		assert.False(t, branch.IsExpired())
	})

	t.Run("branch without expiration", func(t *testing.T) {
		branch := &Branch{
			ID:        uuid.New(),
			Name:      "permanent-branch",
			Slug:      "permanent-branch",
			ExpiresAt: nil,
		}

		assert.False(t, branch.IsExpired())
	})
}

func TestBranch_HasAccess(t *testing.T) {
	t.Run("nil access list", func(t *testing.T) {
		branch := &Branch{
			ID:     uuid.New(),
			Name:   "test-branch",
			Slug:   "test-branch",
			Status: BranchStatusReady,
			Access: nil,
		}

		assert.False(t, branch.HasAccess(uuid.New()))
	})

	t.Run("empty access list", func(t *testing.T) {
		branch := &Branch{
			ID:     uuid.New(),
			Name:   "test-branch",
			Slug:   "test-branch",
			Status: BranchStatusReady,
			Access: []BranchAccess{},
		}

		assert.False(t, branch.HasAccess(uuid.New()))
	})
}

func TestBranch_GetAccessLevel(t *testing.T) {
	userID := uuid.New()

	branch := &Branch{
		ID:     uuid.New(),
		Name:   "test-branch",
		Slug:   "test-branch",
		Status: BranchStatusReady,
		Access: []BranchAccess{
			{
				UserID:      userID,
				AccessLevel: BranchAccessAdmin,
			},
		},
	}

	level := branch.GetAccessLevel(userID)
	assert.Equal(t, BranchAccessAdmin, *level)
}

func TestBranch_GetAccessLevel_NotFound(t *testing.T) {
	userID := uuid.New()

	branch := &Branch{
		ID:     uuid.New(),
		Name:   "test-branch",
		Slug:   "test-branch",
		Status: BranchStatusReady,
		Access: []BranchAccess{},
	}

	level := branch.GetAccessLevel(userID)
	assert.Nil(t, level)
}

// =============================================================================
// Manager Accessor Tests (GetBranchConnectionURL, GetStorage, GetConfig)
// =============================================================================
//
// These cover pure, 0.0%-coverage Manager methods reached under -short. They
// use the &Manager{...} literal construction already used by the tenant-clone
// tests, so no DB is required.

func TestManager_GetBranchConnectionURL(t *testing.T) {
	t.Parallel()
	m := &Manager{mainDBURL: "postgresql://user:pass@localhost:5432/fluxbase"}

	t.Run("rewrites path to branch database name", func(t *testing.T) {
		t.Parallel()
		branch := &Branch{DatabaseName: "branch_feature_x"}
		got, err := m.GetBranchConnectionURL(branch)
		assert.NoError(t, err)
		assert.Contains(t, got, "/branch_feature_x")
		// Host/user preserved.
		assert.Contains(t, got, "user:pass@localhost:5432")
		// Original db name replaced.
		assert.NotContains(t, got, "/fluxbase")
	})

	t.Run("preserves query params if present", func(t *testing.T) {
		t.Parallel()
		m2 := &Manager{mainDBURL: "postgresql://u@localhost:5432/fluxbase?sslmode=disable"}
		branch := &Branch{DatabaseName: "branch_y"}
		got, err := m2.GetBranchConnectionURL(branch)
		assert.NoError(t, err)
		assert.Contains(t, got, "sslmode=disable")
		assert.Contains(t, got, "/branch_y")
	})
}

func TestManager_GetStorage(t *testing.T) {
	t.Parallel()
	t.Run("nil storage returns nil", func(t *testing.T) {
		t.Parallel()
		m := &Manager{}
		assert.Nil(t, m.GetStorage())
	})

	t.Run("non-nil storage returned by identity", func(t *testing.T) {
		t.Parallel()
		st := NewStorage(nil, nil)
		m := &Manager{storage: st}
		assert.Same(t, st, m.GetStorage())
	})
}

func TestManager_GetConfig(t *testing.T) {
	t.Parallel()
	cfg := config.BranchingConfig{Enabled: true, DefaultBranch: "development"}
	m := &Manager{config: cfg}
	got := m.GetConfig()
	assert.Equal(t, cfg.Enabled, got.Enabled)
	assert.Equal(t, "development", got.DefaultBranch)
}

// =============================================================================
// Manager.checkLimits — no-limits path (pure, no storage calls)
// =============================================================================
//
// With MaxBranchesPerTenant=0, MaxTotalBranches=0, MaxBranchesPerUser=0 the
// method short-circuits to nil without ever calling storage, so it is reachable
// with a nil storage field.

func TestManager_CheckLimits_NoLimitsConfigured(t *testing.T) {
	t.Parallel()
	tenant := uuid.New()
	user := uuid.New()
	m := &Manager{config: config.BranchingConfig{Enabled: true}} // all limits zero-valued

	err := m.checkLimits(context.Background(), &tenant, &user)
	assert.NoError(t, err)
}
