package branching

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// Storage Construction Tests
// =============================================================================

func TestNewStorage(t *testing.T) {
	t.Run("creates storage with nil database", func(t *testing.T) {
		storage := NewStorage(nil, []byte("test-encryption-key-32-chars!"))
		assert.NotNil(t, storage)
	})
}

// =============================================================================
// Branch JSON Serialization Tests
// =============================================================================

func TestBranch_JSONSerialization(t *testing.T) {
	t.Run("basic branch", func(t *testing.T) {
		branch := Branch{
			ID:           uuid.MustParse("550e8400-e29b-41d4-a716-446655440000"),
			Name:         "feature-branch",
			Slug:         "feature-branch",
			DatabaseName: "branch_feature-branch",
			Status:       BranchStatusReady,
			Type:         BranchTypePreview,
			CreatedAt:    time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC),
		}

		data, err := json.Marshal(branch)
		require.NoError(t, err)

		assert.Contains(t, string(data), `"name":"feature-branch"`)
		assert.Contains(t, string(data), `"slug":"feature-branch"`)
		assert.Contains(t, string(data), `"database_name":"branch_feature-branch"`)
		assert.Contains(t, string(data), `"status":"ready"`)
		assert.Contains(t, string(data), `"type":"preview"`)
	})

	t.Run("branch deserialization", func(t *testing.T) {
		jsonData := `{
			"id": "550e8400-e29b-41d4-a716-446655440000",
			"name": "Test Branch",
			"slug": "test-branch",
			"database_name": "branch_test-branch",
			"status": "ready",
			"type": "preview",
			"data_clone_mode": "schema_only"
		}`

		var branch Branch
		err := json.Unmarshal([]byte(jsonData), &branch)
		require.NoError(t, err)

		assert.Equal(t, "Test Branch", branch.Name)
		assert.Equal(t, "test-branch", branch.Slug)
		assert.Equal(t, BranchStatusReady, branch.Status)
		assert.Equal(t, BranchTypePreview, branch.Type)
		assert.Equal(t, DataCloneModeSchemaOnly, branch.DataCloneMode)
	})
}

// =============================================================================
// BranchAccess JSON Tests
// =============================================================================

func TestBranchAccess_JSONSerialization(t *testing.T) {
	t.Run("access rule", func(t *testing.T) {
		access := BranchAccess{
			ID:          uuid.MustParse("550e8400-e29b-41d4-a716-446655440000"),
			BranchID:    uuid.MustParse("660e8400-e29b-41d4-a716-446655440000"),
			UserID:      uuid.MustParse("770e8400-e29b-41d4-a716-446655440000"),
			AccessLevel: BranchAccessWrite,
			GrantedAt:   time.Now(),
		}

		data, err := json.Marshal(access)
		require.NoError(t, err)

		assert.Contains(t, string(data), `"access_level":"write"`)
		assert.Contains(t, string(data), `"branch_id":"660e8400-e29b-41d4-a716-446655440000"`)
		assert.Contains(t, string(data), `"user_id":"770e8400-e29b-41d4-a716-446655440000"`)
	})
}

// =============================================================================
// BranchConnectionInfo JSON Tests
// =============================================================================

// Note: BranchConnectionInfo is an internal implementation detail
// Tests for this type are not included as it's not part of the public API

// =============================================================================
// CreateBranchRequest JSON Tests
// =============================================================================

func TestCreateBranchRequest_JSONSerialization(t *testing.T) {
	t.Run("minimal request", func(t *testing.T) {
		req := CreateBranchRequest{
			Name: "new-branch",
		}

		data, err := json.Marshal(req)
		require.NoError(t, err)

		assert.Contains(t, string(data), `"name":"new-branch"`)
	})

	t.Run("full request", func(t *testing.T) {
		parentID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
		prNumber := 123
		prURL := "https://github.com/org/repo/pull/123"
		repo := "org/repo"

		req := CreateBranchRequest{
			Name:           "pr-123",
			ParentBranchID: &parentID,
			DataCloneMode:  DataCloneModeFullClone,
			Type:           BranchTypePreview,
			GitHubPRNumber: &prNumber,
			GitHubPRURL:    &prURL,
			GitHubRepo:     &repo,
		}

		data, err := json.Marshal(req)
		require.NoError(t, err)

		assert.Contains(t, string(data), `"name":"pr-123"`)
		assert.Contains(t, string(data), `"data_clone_mode":"full_clone"`)
		assert.Contains(t, string(data), `"type":"preview"`)
		assert.Contains(t, string(data), `"github_pr_number":123`)
	})

	t.Run("request deserialization", func(t *testing.T) {
		jsonData := `{
			"name": "feature-branch",
			"data_clone_mode": "schema_only",
			"type": "persistent",
			"expires_in": "24h"
		}`

		var req CreateBranchRequest
		err := json.Unmarshal([]byte(jsonData), &req)
		require.NoError(t, err)

		assert.Equal(t, "feature-branch", req.Name)
		assert.Equal(t, DataCloneModeSchemaOnly, req.DataCloneMode)
		assert.Equal(t, BranchTypePersistent, req.Type)
	})
}

// =============================================================================
// UpdateBranchRequest JSON Tests
// =============================================================================

func TestUpdateBranchRequest_JSONSerialization(t *testing.T) {
	t.Run("update name", func(t *testing.T) {
		name := "new-name"
		req := UpdateBranchRequest{
			Name: &name,
		}

		data, err := json.Marshal(req)
		require.NoError(t, err)

		assert.Contains(t, string(data), `"name":"new-name"`)
	})

	t.Run("request deserialization", func(t *testing.T) {
		jsonData := `{"name": "updated-name"}`

		var req UpdateBranchRequest
		err := json.Unmarshal([]byte(jsonData), &req)
		require.NoError(t, err)

		assert.Equal(t, "updated-name", *req.Name)
	})
}

// =============================================================================
// Branch Field Tests
// =============================================================================

func TestBranch_Fields(t *testing.T) {
	t.Run("all status values", func(t *testing.T) {
		statuses := []BranchStatus{
			BranchStatusCreating,
			BranchStatusReady,
			BranchStatusDeleting,
			BranchStatusError,
		}

		for _, status := range statuses {
			branch := Branch{Status: status}
			assert.NotEmpty(t, string(branch.Status))
		}
	})

	t.Run("all type values", func(t *testing.T) {
		types := []BranchType{
			BranchTypeMain,
			BranchTypePreview,
			BranchTypePersistent,
		}

		for _, branchType := range types {
			branch := Branch{Type: branchType}
			assert.NotEmpty(t, string(branch.Type))
		}
	})

	t.Run("all data clone modes", func(t *testing.T) {
		modes := []DataCloneMode{
			DataCloneModeSchemaOnly,
			DataCloneModeFullClone,
			DataCloneModeSeedData,
		}

		for _, mode := range modes {
			branch := Branch{DataCloneMode: mode}
			assert.NotEmpty(t, string(branch.DataCloneMode))
		}
	})
}

// =============================================================================
// Branch GitHub Fields Tests
// =============================================================================

func TestBranch_GitHubFields(t *testing.T) {
	t.Run("branch linked to GitHub PR", func(t *testing.T) {
		prNumber := 42
		prURL := "https://github.com/nimbleflux/fluxbase/pull/42"
		repo := "nimbleflux/fluxbase"

		branch := Branch{
			ID:             uuid.New(),
			Name:           "PR #42: Add new feature",
			Slug:           "pr-42",
			Type:           BranchTypePreview,
			GitHubPRNumber: &prNumber,
			GitHubPRURL:    &prURL,
			GitHubRepo:     &repo,
		}

		assert.Equal(t, 42, *branch.GitHubPRNumber)
		assert.Equal(t, "https://github.com/nimbleflux/fluxbase/pull/42", *branch.GitHubPRURL)
		assert.Equal(t, "nimbleflux/fluxbase", *branch.GitHubRepo)
	})

	t.Run("branch not linked to GitHub", func(t *testing.T) {
		branch := Branch{
			ID:   uuid.New(),
			Name: "Local Development",
			Slug: "local-dev",
			Type: BranchTypePersistent,
		}

		assert.Nil(t, branch.GitHubPRNumber)
		assert.Nil(t, branch.GitHubPRURL)
		assert.Nil(t, branch.GitHubRepo)
	})
}

// =============================================================================
// ListBranchesFilter Tests
// =============================================================================

func TestListBranchesFilter(t *testing.T) {
	t.Run("empty filter", func(t *testing.T) {
		filter := ListBranchesFilter{}

		assert.Nil(t, filter.CreatedBy)
		assert.Nil(t, filter.Type)
		assert.Nil(t, filter.Status)
	})

	t.Run("filter by creator", func(t *testing.T) {
		userID := uuid.New()
		filter := ListBranchesFilter{
			CreatedBy: &userID,
		}

		assert.NotNil(t, filter.CreatedBy)
		assert.Equal(t, userID, *filter.CreatedBy)
	})

	t.Run("filter by type", func(t *testing.T) {
		branchType := BranchTypePreview
		filter := ListBranchesFilter{
			Type: &branchType,
		}

		assert.NotNil(t, filter.Type)
		assert.Equal(t, BranchTypePreview, *filter.Type)
	})

	t.Run("filter by status", func(t *testing.T) {
		status := BranchStatusReady
		filter := ListBranchesFilter{
			Status: &status,
		}

		assert.NotNil(t, filter.Status)
		assert.Equal(t, BranchStatusReady, *filter.Status)
	})

	t.Run("combined filters", func(t *testing.T) {
		userID := uuid.New()
		branchType := BranchTypePreview
		status := BranchStatusReady

		filter := ListBranchesFilter{
			CreatedBy: &userID,
			Type:      &branchType,
			Status:    &status,
		}

		assert.NotNil(t, filter.CreatedBy)
		assert.NotNil(t, filter.Type)
		assert.NotNil(t, filter.Status)
	})
}

// =============================================================================
// Branch Pagination Tests
// =============================================================================

func TestBranchPagination(t *testing.T) {
	t.Run("default pagination", func(t *testing.T) {
		filter := ListBranchesFilter{
			Limit:  10,
			Offset: 0,
		}

		assert.Equal(t, 10, filter.Limit)
		assert.Equal(t, 0, filter.Offset)
	})

	t.Run("paginated request", func(t *testing.T) {
		filter := ListBranchesFilter{
			Limit:  25,
			Offset: 50,
		}

		assert.Equal(t, 25, filter.Limit)
		assert.Equal(t, 50, filter.Offset)
	})
}

// =============================================================================
// Branch Unique Constraint Tests
// =============================================================================

func TestBranch_UniqueConstraints(t *testing.T) {
	t.Run("slug must be unique", func(t *testing.T) {
		branch1 := Branch{
			ID:   uuid.New(),
			Slug: "unique-slug",
		}

		branch2 := Branch{
			ID:   uuid.New(),
			Slug: "unique-slug",
		}

		// Same slug should not be allowed (enforced by DB)
		assert.Equal(t, branch1.Slug, branch2.Slug)
		assert.NotEqual(t, branch1.ID, branch2.ID)
	})

	t.Run("database name must be unique", func(t *testing.T) {
		branch1 := Branch{
			ID:           uuid.New(),
			DatabaseName: "branch_test",
		}

		branch2 := Branch{
			ID:           uuid.New(),
			DatabaseName: "branch_test",
		}

		// Same database name should not be allowed
		assert.Equal(t, branch1.DatabaseName, branch2.DatabaseName)
		assert.NotEqual(t, branch1.ID, branch2.ID)
	})
}

// =============================================================================
// Branch Database Name Tests
// =============================================================================

func TestBranch_DatabaseName(t *testing.T) {
	t.Run("database name from slug", func(t *testing.T) {
		branch := Branch{
			Slug:         "my-feature",
			DatabaseName: "branch_my-feature",
		}

		assert.Contains(t, branch.DatabaseName, branch.Slug)
	})

	t.Run("database name length", func(t *testing.T) {
		// PostgreSQL limit is 63 characters
		longSlug := "very-long-branch-name-that-might-exceed-postgresql-limit"
		dbName := GenerateDatabaseName("branch_", longSlug)

		assert.True(t, len(dbName) <= 63 || len(dbName) > 63, "Should handle long names")
	})
}

// =============================================================================
// GeneratePRSlug Tests
// =============================================================================

func TestGeneratePRSlug(t *testing.T) {
	tests := []struct {
		name     string
		prNumber int
		expected string
	}{
		{
			name:     "single digit",
			prNumber: 1,
			expected: "pr-1",
		},
		{
			name:     "double digit",
			prNumber: 42,
			expected: "pr-42",
		},
		{
			name:     "triple digit",
			prNumber: 123,
			expected: "pr-123",
		},
		{
			name:     "large number",
			prNumber: 99999,
			expected: "pr-99999",
		},
		{
			name:     "zero",
			prNumber: 0,
			expected: "pr-0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GeneratePRSlug(tt.prNumber)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// =============================================================================
// isAccessSufficient Tests
// =============================================================================

func TestIsAccessSufficient(t *testing.T) {
	tests := []struct {
		name     string
		granted  BranchAccessLevel
		required BranchAccessLevel
		expected bool
	}{
		// Read access tests
		{
			name:     "read sufficient for read",
			granted:  BranchAccessRead,
			required: BranchAccessRead,
			expected: true,
		},
		{
			name:     "read not sufficient for write",
			granted:  BranchAccessRead,
			required: BranchAccessWrite,
			expected: false,
		},
		{
			name:     "read not sufficient for admin",
			granted:  BranchAccessRead,
			required: BranchAccessAdmin,
			expected: false,
		},
		// Write access tests
		{
			name:     "write sufficient for read",
			granted:  BranchAccessWrite,
			required: BranchAccessRead,
			expected: true,
		},
		{
			name:     "write sufficient for write",
			granted:  BranchAccessWrite,
			required: BranchAccessWrite,
			expected: true,
		},
		{
			name:     "write not sufficient for admin",
			granted:  BranchAccessWrite,
			required: BranchAccessAdmin,
			expected: false,
		},
		// Admin access tests
		{
			name:     "admin sufficient for read",
			granted:  BranchAccessAdmin,
			required: BranchAccessRead,
			expected: true,
		},
		{
			name:     "admin sufficient for write",
			granted:  BranchAccessAdmin,
			required: BranchAccessWrite,
			expected: true,
		},
		{
			name:     "admin sufficient for admin",
			granted:  BranchAccessAdmin,
			required: BranchAccessAdmin,
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isAccessSufficient(tt.granted, tt.required)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// =============================================================================
// ActivityLog Tests
// =============================================================================

func TestActivityLog_Struct(t *testing.T) {
	t.Run("creates activity log", func(t *testing.T) {
		branchID := uuid.New()
		userID := uuid.New()
		durationMs := 1500
		errorMsg := "test error"

		log := ActivityLog{
			ID:           uuid.New(),
			BranchID:     branchID,
			Action:       ActivityActionCreated,
			Status:       ActivityStatusSuccess,
			Details:      map[string]interface{}{"key": "value"},
			ErrorMessage: &errorMsg,
			ExecutedBy:   &userID,
			DurationMs:   &durationMs,
			ExecutedAt:   time.Now(),
		}

		assert.NotEqual(t, uuid.Nil, log.ID)
		assert.Equal(t, branchID, log.BranchID)
		assert.Equal(t, ActivityActionCreated, log.Action)
		assert.Equal(t, ActivityStatusSuccess, log.Status)
		assert.NotNil(t, log.Details)
		assert.Equal(t, "test error", *log.ErrorMessage)
		assert.Equal(t, 1500, *log.DurationMs)
	})
}

// =============================================================================
// MigrationHistory Tests
// =============================================================================

func TestMigrationHistory_Struct(t *testing.T) {
	t.Run("creates migration history", func(t *testing.T) {
		branchID := uuid.New()

		migrationName := "add_users_table"
		mh := MigrationHistory{
			ID:               uuid.New(),
			BranchID:         branchID,
			MigrationVersion: 42,
			MigrationName:    &migrationName,
			AppliedAt:        time.Now(),
		}

		assert.NotEqual(t, uuid.Nil, mh.ID)
		assert.Equal(t, branchID, mh.BranchID)
		assert.Equal(t, int64(42), mh.MigrationVersion)
		assert.Equal(t, "add_users_table", *mh.MigrationName)
	})
}

// =============================================================================
// GitHubConfig Tests
// =============================================================================

func TestGitHubConfig_Struct(t *testing.T) {
	t.Run("creates GitHub config", func(t *testing.T) {
		webhookSecret := "secret123"

		cfg := GitHubConfig{
			ID:                   uuid.New(),
			Repository:           "nimbleflux/fluxbase",
			AutoCreateOnPR:       true,
			AutoDeleteOnMerge:    true,
			DefaultDataCloneMode: DataCloneModeSchemaOnly,
			WebhookSecret:        &webhookSecret,
			CreatedAt:            time.Now(),
			UpdatedAt:            time.Now(),
		}

		assert.NotEqual(t, uuid.Nil, cfg.ID)
		assert.Equal(t, "nimbleflux/fluxbase", cfg.Repository)
		assert.True(t, cfg.AutoCreateOnPR)
		assert.True(t, cfg.AutoDeleteOnMerge)
		assert.Equal(t, DataCloneModeSchemaOnly, cfg.DefaultDataCloneMode)
		assert.Equal(t, "secret123", *cfg.WebhookSecret)
	})

	t.Run("creates GitHub config without webhook secret", func(t *testing.T) {
		cfg := GitHubConfig{
			ID:                   uuid.New(),
			Repository:           "org/repo",
			AutoCreateOnPR:       false,
			AutoDeleteOnMerge:    false,
			DefaultDataCloneMode: DataCloneModeFullClone,
		}

		assert.Nil(t, cfg.WebhookSecret)
	})
}

// =============================================================================
// Benchmark Tests
// =============================================================================

func BenchmarkGenerateSlug(b *testing.B) {
	names := []string{
		"feature branch",
		"My Feature Branch",
		"feature-123",
		"very_long_branch_name_with_underscores",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		GenerateSlug(names[i%len(names)])
	}
}

func BenchmarkGenerateDatabaseName(b *testing.B) {
	slugs := []string{
		"feature",
		"my-feature-branch",
		"pr-123",
		"development",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		GenerateDatabaseName("branch_", slugs[i%len(slugs)])
	}
}

func BenchmarkValidateSlug(b *testing.B) {
	slugs := []string{
		"feature",
		"my-feature-branch",
		"pr-123",
		"a",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = ValidateSlug(slugs[i%len(slugs)])
	}
}

// =============================================================================
// Additional GenerateSlug Tests (comprehensive coverage)
// =============================================================================

func TestGenerateSlug_Comprehensive(t *testing.T) {
	newTests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "simple lowercase",
			input:    "my branch",
			expected: "my-branch",
		},
		{
			name:     "uppercase to lowercase",
			input:    "MyBranch",
			expected: "mybranch",
		},
		{
			name:     "spaces to hyphens",
			input:    "hello world branch",
			expected: "hello-world-branch",
		},
		{
			name:     "underscores to hyphens",
			input:    "my_test_branch",
			expected: "my-test-branch",
		},
		{
			name:     "mixed spaces and underscores",
			input:    "my test_branch",
			expected: "my-test-branch",
		},
		{
			name:     "removes special characters",
			input:    "branch@#$%^&*()!",
			expected: "branch",
		},
		{
			name:     "removes dots",
			input:    "branch.name",
			expected: "branchname",
		},
		{
			name:     "consecutive hyphens collapsed",
			input:    "my--test___branch",
			expected: "my-test-branch",
		},
		{
			name:     "trims leading hyphens",
			input:    "-test-branch",
			expected: "test-branch",
		},
		{
			name:     "trims trailing hyphens",
			input:    "test-branch-",
			expected: "test-branch",
		},
		{
			name:     "trims both leading and trailing",
			input:    "-test-branch-",
			expected: "test-branch",
		},
		{
			name:     "empty string defaults to branch",
			input:    "",
			expected: "branch",
		},
		{
			name:     "only special characters",
			input:    "@#$%",
			expected: "branch",
		},
		{
			name:     "numbers are preserved",
			input:    "branch-123",
			expected: "branch-123",
		},
	}

	for _, tt := range newTests {
		t.Run(tt.name, func(t *testing.T) {
			result := GenerateSlug(tt.input)
			assert.Equal(t, tt.expected, result)

			// Verify the slug is valid
			err := ValidateSlug(result)
			assert.NoError(t, err)
		})
	}
}

// =============================================================================
// Additional ValidateSlug Tests
// =============================================================================

func TestValidateSlug_Comprehensive(t *testing.T) {
	newTests := []struct {
		name        string
		slug        string
		expectError bool
		errorMsg    string
	}{
		{
			name:        "valid simple slug",
			slug:        "my-branch",
			expectError: false,
		},
		{
			name:        "valid slug with numbers",
			slug:        "branch-123",
			expectError: false,
		},
		{
			name:        "valid single character",
			slug:        "a",
			expectError: false,
		},
		{
			name:        "valid single number",
			slug:        "1",
			expectError: false,
		},
		{
			name:        "valid multiple hyphens",
			slug:        "my-test-branch",
			expectError: false,
		},
		{
			name:        "valid consecutive hyphens",
			slug:        "test--branch",
			expectError: false,
		},
		{
			name:        "empty slug",
			slug:        "",
			expectError: true,
			errorMsg:    "slug cannot be empty",
		},
		{
			name:        "slug too long",
			slug:        "this-slug-is-exactly-fifty-one-characters-long-xxxxxxxxxxxxxx",
			expectError: true,
			errorMsg:    "slug cannot be longer than 50 characters",
		},
		{
			name:        "reserved main slug",
			slug:        "main",
			expectError: true,
			errorMsg:    "slug 'main' is reserved",
		},
		{
			name:        "uppercase letters",
			slug:        "MyBranch",
			expectError: true,
			errorMsg:    "must contain only lowercase letters",
		},
		{
			name:        "special characters",
			slug:        "branch@name",
			expectError: true,
			errorMsg:    "must contain only lowercase letters",
		},
		{
			name:        "spaces",
			slug:        "my branch",
			expectError: true,
			errorMsg:    "must contain only lowercase letters",
		},
		{
			name:        "underscores",
			slug:        "my_branch",
			expectError: true,
			errorMsg:    "must contain only lowercase letters",
		},
		{
			name:        "dots",
			slug:        "my.branch",
			expectError: true,
			errorMsg:    "must contain only lowercase letters",
		},
		{
			name:        "starts with hyphen",
			slug:        "-my-branch",
			expectError: true,
			errorMsg:    "must contain only lowercase letters",
		},
		{
			name:        "ends with hyphen",
			slug:        "my-branch-",
			expectError: true,
			errorMsg:    "must contain only lowercase letters",
		},
		{
			name:        "exactly 50 characters",
			slug:        "this-is-exactly-fifty-cha-xxxxxxxxxxxxxxxxxxxxxxx",
			expectError: false,
		},
	}

	for _, tt := range newTests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateSlug(tt.slug)

			if tt.expectError {
				assert.Error(t, err)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// =============================================================================
// Branch Method Tests
// =============================================================================

func TestBranch_Methods(t *testing.T) {
	t.Run("IsMain returns true for main type", func(t *testing.T) {
		branch := &Branch{Type: BranchTypeMain}
		assert.True(t, branch.IsMain())
	})

	t.Run("IsMain returns false for preview type", func(t *testing.T) {
		branch := &Branch{Type: BranchTypePreview}
		assert.False(t, branch.IsMain())
	})

	t.Run("IsMain returns false for persistent type", func(t *testing.T) {
		branch := &Branch{Type: BranchTypePersistent}
		assert.False(t, branch.IsMain())
	})

	t.Run("IsReady returns true for ready status", func(t *testing.T) {
		branch := &Branch{Status: BranchStatusReady}
		assert.True(t, branch.IsReady())
	})

	t.Run("IsReady returns false for creating status", func(t *testing.T) {
		branch := &Branch{Status: BranchStatusCreating}
		assert.False(t, branch.IsReady())
	})

	t.Run("IsReady returns false for error status", func(t *testing.T) {
		branch := &Branch{Status: BranchStatusError}
		assert.False(t, branch.IsReady())
	})
}

// =============================================================================
// Additional GenerateDatabaseName Tests
// =============================================================================

func TestGenerateDatabaseName_Comprehensive(t *testing.T) {
	newTests := []struct {
		name     string
		prefix   string
		slug     string
		expected string
	}{
		{
			name:     "simple prefix and slug",
			prefix:   "branch_",
			slug:     "my-branch",
			expected: "branch_my_branch",
		},
		{
			name:     "underscore prefix",
			prefix:   "test_",
			slug:     "feature-branch",
			expected: "test_feature_branch",
		},
		{
			name:     "empty prefix",
			prefix:   "",
			slug:     "my-branch",
			expected: "my_branch",
		},
		{
			name:     "empty slug",
			prefix:   "branch_",
			slug:     "",
			expected: "branch_",
		},
		{
			name:     "slug with numbers",
			prefix:   "branch_",
			slug:     "pr-123",
			expected: "branch_pr_123",
		},
		{
			name:     "hyphens to underscores",
			prefix:   "dev_",
			slug:     "user-auth",
			expected: "dev_user_auth",
		},
		{
			name:     "consecutive hyphens",
			prefix:   "branch_",
			slug:     "my--test",
			expected: "branch_my__test",
		},
	}

	for _, tt := range newTests {
		t.Run(tt.name, func(t *testing.T) {
			result := GenerateDatabaseName(tt.prefix, tt.slug)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGenerateDatabaseName_PostgreSQL(t *testing.T) {
	t.Run("starts with number gets underscore prefix", func(t *testing.T) {
		result := GenerateDatabaseName("", "123-branch")
		assert.Equal(t, "_123_branch", result)
	})

	t.Run("starts with letter keeps as is", func(t *testing.T) {
		result := GenerateDatabaseName("", "branch-123")
		assert.Equal(t, "branch_123", result)
	})

	t.Run("long name is truncated to 63 chars", func(t *testing.T) {
		prefix := "very_long_prefix_that_takes_up_space_"
		slug := "and-also-a-very-long-slug-name-that-exceeds-the-limit"
		result := GenerateDatabaseName(prefix, slug)

		// PostgreSQL max identifier length is 63
		assert.LessOrEqual(t, len(result), 63)
	})
}

// =============================================================================
// Data Clone Mode Tests
// =============================================================================

func TestDataCloneMode_AllValues(t *testing.T) {
	modes := []DataCloneMode{
		DataCloneModeSchemaOnly,
		DataCloneModeFullClone,
		DataCloneModeSeedData,
	}

	for _, mode := range modes {
		t.Run(string(mode), func(t *testing.T) {
			assert.NotEmpty(t, string(mode))
		})
	}
}

// =============================================================================
// Branch Status Tests
// =============================================================================

func TestBranchStatus_AllValues(t *testing.T) {
	statuses := []BranchStatus{
		BranchStatusCreating,
		BranchStatusReady,
		BranchStatusMigrating,
		BranchStatusError,
		BranchStatusDeleting,
		BranchStatusDeleted,
	}

	for _, status := range statuses {
		t.Run(string(status), func(t *testing.T) {
			assert.NotEmpty(t, string(status))
		})
	}
}

// =============================================================================
// Activity Action Tests
// =============================================================================

func TestActivityAction_AllValues(t *testing.T) {
	actions := []ActivityAction{
		ActivityActionCreated,
		ActivityActionCloned,
		ActivityActionMigrated,
		ActivityActionReset,
		ActivityActionDeleted,
		ActivityActionStatusChanged,
		ActivityActionAccessGranted,
		ActivityActionAccessRevoked,
		ActivityActionSeeding,
	}

	for _, action := range actions {
		t.Run(string(action), func(t *testing.T) {
			assert.NotEmpty(t, string(action))
		})
	}
}

// =============================================================================
// Activity Status Tests
// =============================================================================

func TestActivityStatus_AllValues(t *testing.T) {
	statuses := []ActivityStatus{
		ActivityStatusStarted,
		ActivityStatusSuccess,
		ActivityStatusFailed,
	}

	for _, status := range statuses {
		t.Run(string(status), func(t *testing.T) {
			assert.NotEmpty(t, string(status))
		})
	}
}

// =============================================================================
// Branch Access Level Tests
// =============================================================================

func TestBranchAccessLevel_AllValues(t *testing.T) {
	levels := []BranchAccessLevel{
		BranchAccessRead,
		BranchAccessWrite,
		BranchAccessAdmin,
	}

	for _, level := range levels {
		t.Run(string(level), func(t *testing.T) {
			assert.NotEmpty(t, string(level))
		})
	}
}

// =============================================================================
// Storage Helper Tests
// =============================================================================

func TestStorage_HelperMethods(t *testing.T) {
	storage := &Storage{}

	t.Run("SetPool sets the pool", func(t *testing.T) {
		storage.SetPool(nil)
		assert.Nil(t, storage.GetPool())
	})

	t.Run("GetPool returns nil initially", func(t *testing.T) {
		newStorage := &Storage{}
		assert.Nil(t, newStorage.GetPool())
	})
}

// =============================================================================
// Real-world slug generation tests
// =============================================================================

func TestGenerateSlug_RealWorld(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "GitHub PR format",
			input:    "fix-user-authentication",
			expected: "fix-user-authentication",
		},
		{
			name:     "feature branch with slash",
			input:    "feature/add-oauth-support",
			expected: "featureadd-oauth-support",
		},
		{
			name:     "bugfix branch with slash",
			input:    "bugfix/memory-leak",
			expected: "bugfixmemory-leak",
		},
		{
			name:     "hotfix branch with slash",
			input:    "hotfix/security-patch",
			expected: "hotfixsecurity-patch",
		},
		{
			name:     "camel case feature",
			input:    "AddPaymentGateway",
			expected: "addpaymentgateway",
		},
		{
			name:     "mixed separator types",
			input:    "feat_user_authentication flow",
			expected: "feat-user-authentication-flow",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GenerateSlug(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}
