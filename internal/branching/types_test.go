package branching

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

// =============================================================================
// BranchStatus Constants Tests
// =============================================================================

func TestBranchStatus_Constants(t *testing.T) {
	tests := []struct {
		status   BranchStatus
		expected string
	}{
		{BranchStatusCreating, "creating"},
		{BranchStatusReady, "ready"},
		{BranchStatusMigrating, "migrating"},
		{BranchStatusError, "error"},
		{BranchStatusDeleting, "deleting"},
		{BranchStatusDeleted, "deleted"},
	}

	for _, tt := range tests {
		t.Run(string(tt.status), func(t *testing.T) {
			assert.Equal(t, tt.expected, string(tt.status))
		})
	}
}

// =============================================================================
// BranchType Constants Tests
// =============================================================================

func TestBranchType_Constants(t *testing.T) {
	tests := []struct {
		branchType BranchType
		expected   string
	}{
		{BranchTypeMain, "main"},
		{BranchTypePreview, "preview"},
		{BranchTypePersistent, "persistent"},
	}

	for _, tt := range tests {
		t.Run(string(tt.branchType), func(t *testing.T) {
			assert.Equal(t, tt.expected, string(tt.branchType))
		})
	}
}

// =============================================================================
// DataCloneMode Constants Tests
// =============================================================================

func TestDataCloneMode_Constants(t *testing.T) {
	tests := []struct {
		mode     DataCloneMode
		expected string
	}{
		{DataCloneModeSchemaOnly, "schema_only"},
		{DataCloneModeFullClone, "full_clone"},
		{DataCloneModeSeedData, "seed_data"},
	}

	for _, tt := range tests {
		t.Run(string(tt.mode), func(t *testing.T) {
			assert.Equal(t, tt.expected, string(tt.mode))
		})
	}
}

// =============================================================================
// ActivityAction Constants Tests
// =============================================================================

func TestActivityAction_Constants(t *testing.T) {
	tests := []struct {
		action   ActivityAction
		expected string
	}{
		{ActivityActionCreated, "created"},
		{ActivityActionCloned, "cloned"},
		{ActivityActionMigrated, "migrated"},
		{ActivityActionReset, "reset"},
		{ActivityActionDeleted, "deleted"},
		{ActivityActionStatusChanged, "status_changed"},
		{ActivityActionAccessGranted, "access_granted"},
		{ActivityActionAccessRevoked, "access_revoked"},
		{ActivityActionSeeding, "seeding"},
	}

	for _, tt := range tests {
		t.Run(string(tt.action), func(t *testing.T) {
			assert.Equal(t, tt.expected, string(tt.action))
		})
	}
}

// =============================================================================
// ActivityStatus Constants Tests
// =============================================================================

func TestActivityStatus_Constants(t *testing.T) {
	tests := []struct {
		status   ActivityStatus
		expected string
	}{
		{ActivityStatusStarted, "started"},
		{ActivityStatusSuccess, "success"},
		{ActivityStatusFailed, "failed"},
	}

	for _, tt := range tests {
		t.Run(string(tt.status), func(t *testing.T) {
			assert.Equal(t, tt.expected, string(tt.status))
		})
	}
}

// =============================================================================
// BranchAccessLevel Constants Tests
// =============================================================================

func TestBranchAccessLevel_Constants(t *testing.T) {
	tests := []struct {
		level    BranchAccessLevel
		expected string
	}{
		{BranchAccessRead, "read"},
		{BranchAccessWrite, "write"},
		{BranchAccessAdmin, "admin"},
	}

	for _, tt := range tests {
		t.Run(string(tt.level), func(t *testing.T) {
			assert.Equal(t, tt.expected, string(tt.level))
		})
	}
}

// =============================================================================
// Branch.IsMain Tests
// =============================================================================

func TestBranch_IsMain(t *testing.T) {
	tests := []struct {
		name     string
		branch   Branch
		expected bool
	}{
		{
			name:     "main branch",
			branch:   Branch{Type: BranchTypeMain},
			expected: true,
		},
		{
			name:     "preview branch",
			branch:   Branch{Type: BranchTypePreview},
			expected: false,
		},
		{
			name:     "persistent branch",
			branch:   Branch{Type: BranchTypePersistent},
			expected: false,
		},
		{
			name:     "empty type",
			branch:   Branch{Type: ""},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.branch.IsMain()
			assert.Equal(t, tt.expected, result)
		})
	}
}

// =============================================================================
// Branch.IsReady Tests
// =============================================================================

func TestBranch_IsReady(t *testing.T) {
	tests := []struct {
		name     string
		branch   Branch
		expected bool
	}{
		{
			name:     "ready status",
			branch:   Branch{Status: BranchStatusReady},
			expected: true,
		},
		{
			name:     "creating status",
			branch:   Branch{Status: BranchStatusCreating},
			expected: false,
		},
		{
			name:     "migrating status",
			branch:   Branch{Status: BranchStatusMigrating},
			expected: false,
		},
		{
			name:     "error status",
			branch:   Branch{Status: BranchStatusError},
			expected: false,
		},
		{
			name:     "deleting status",
			branch:   Branch{Status: BranchStatusDeleting},
			expected: false,
		},
		{
			name:     "deleted status",
			branch:   Branch{Status: BranchStatusDeleted},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.branch.IsReady()
			assert.Equal(t, tt.expected, result)
		})
	}
}

// =============================================================================
// Branch Struct Tests
// =============================================================================

func TestBranch_Struct(t *testing.T) {
	t.Run("all fields set", func(t *testing.T) {
		id := uuid.New()
		parentID := uuid.New()
		createdBy := uuid.New()
		now := time.Now()
		expiresAt := now.Add(24 * time.Hour)
		prNumber := 123
		prURL := "https://github.com/owner/repo/pull/123"
		repo := "owner/repo"
		errorMsg := "some error"
		seedsPath := "./seeds"

		branch := Branch{
			ID:             id,
			Name:           "Feature Branch",
			Slug:           "feature-branch",
			DatabaseName:   "branch_feature_branch",
			Status:         BranchStatusReady,
			Type:           BranchTypePreview,
			ParentBranchID: &parentID,
			DataCloneMode:  DataCloneModeSchemaOnly,
			GitHubPRNumber: &prNumber,
			GitHubPRURL:    &prURL,
			GitHubRepo:     &repo,
			ErrorMessage:   &errorMsg,
			SeedsPath:      &seedsPath,
			CreatedBy:      &createdBy,
			CreatedAt:      now,
			UpdatedAt:      now,
			ExpiresAt:      &expiresAt,
		}

		assert.Equal(t, id, branch.ID)
		assert.Equal(t, "Feature Branch", branch.Name)
		assert.Equal(t, "feature-branch", branch.Slug)
		assert.Equal(t, "branch_feature_branch", branch.DatabaseName)
		assert.Equal(t, BranchStatusReady, branch.Status)
		assert.Equal(t, BranchTypePreview, branch.Type)
		assert.Equal(t, &parentID, branch.ParentBranchID)
		assert.Equal(t, DataCloneModeSchemaOnly, branch.DataCloneMode)
		assert.Equal(t, &prNumber, branch.GitHubPRNumber)
		assert.Equal(t, &prURL, branch.GitHubPRURL)
		assert.Equal(t, &repo, branch.GitHubRepo)
		assert.Equal(t, &errorMsg, branch.ErrorMessage)
		assert.Equal(t, &seedsPath, branch.SeedsPath)
		assert.Equal(t, &createdBy, branch.CreatedBy)
		assert.Equal(t, now, branch.CreatedAt)
		assert.Equal(t, now, branch.UpdatedAt)
		assert.Equal(t, &expiresAt, branch.ExpiresAt)
	})

	t.Run("minimal branch", func(t *testing.T) {
		branch := Branch{
			ID:           uuid.New(),
			Name:         "main",
			Slug:         "main",
			DatabaseName: "fluxbase",
			Status:       BranchStatusReady,
			Type:         BranchTypeMain,
		}

		assert.NotEqual(t, uuid.Nil, branch.ID)
		assert.Nil(t, branch.ParentBranchID)
		assert.Nil(t, branch.GitHubPRNumber)
		assert.Nil(t, branch.GitHubPRURL)
		assert.Nil(t, branch.GitHubRepo)
		assert.Nil(t, branch.ErrorMessage)
		assert.Nil(t, branch.SeedsPath)
		assert.Nil(t, branch.CreatedBy)
		assert.Nil(t, branch.ExpiresAt)
	})
}

// =============================================================================
// MigrationHistory Struct Tests
// =============================================================================

func TestMigrationHistory_Struct(t *testing.T) {
	t.Run("with migration name", func(t *testing.T) {
		id := uuid.New()
		branchID := uuid.New()
		name := "001_create_users"
		now := time.Now()

		history := MigrationHistory{
			ID:               id,
			BranchID:         branchID,
			MigrationVersion: 1,
			MigrationName:    &name,
			AppliedAt:        now,
		}

		assert.Equal(t, id, history.ID)
		assert.Equal(t, branchID, history.BranchID)
		assert.Equal(t, int64(1), history.MigrationVersion)
		assert.Equal(t, &name, history.MigrationName)
		assert.Equal(t, now, history.AppliedAt)
	})

	t.Run("without migration name", func(t *testing.T) {
		history := MigrationHistory{
			ID:               uuid.New(),
			BranchID:         uuid.New(),
			MigrationVersion: 42,
			AppliedAt:        time.Now(),
		}

		assert.Nil(t, history.MigrationName)
	})
}

// =============================================================================
// ActivityLog Struct Tests
// =============================================================================

func TestActivityLog_Struct(t *testing.T) {
	t.Run("successful activity", func(t *testing.T) {
		id := uuid.New()
		branchID := uuid.New()
		executedBy := uuid.New()
		now := time.Now()
		duration := 1500

		log := ActivityLog{
			ID:         id,
			BranchID:   branchID,
			Action:     ActivityActionCreated,
			Status:     ActivityStatusSuccess,
			Details:    map[string]any{"source": "api"},
			ExecutedBy: &executedBy,
			ExecutedAt: now,
			DurationMs: &duration,
		}

		assert.Equal(t, id, log.ID)
		assert.Equal(t, branchID, log.BranchID)
		assert.Equal(t, ActivityActionCreated, log.Action)
		assert.Equal(t, ActivityStatusSuccess, log.Status)
		assert.NotNil(t, log.Details)
		assert.Nil(t, log.ErrorMessage)
		assert.Equal(t, &executedBy, log.ExecutedBy)
		assert.Equal(t, now, log.ExecutedAt)
		assert.Equal(t, &duration, log.DurationMs)
	})

	t.Run("failed activity", func(t *testing.T) {
		errorMsg := "database connection failed"

		log := ActivityLog{
			ID:           uuid.New(),
			BranchID:     uuid.New(),
			Action:       ActivityActionCloned,
			Status:       ActivityStatusFailed,
			ErrorMessage: &errorMsg,
			ExecutedAt:   time.Now(),
		}

		assert.Equal(t, ActivityStatusFailed, log.Status)
		assert.Equal(t, &errorMsg, log.ErrorMessage)
	})
}

// =============================================================================
// GitHubConfig Struct Tests
// =============================================================================

func TestGitHubConfig_Struct(t *testing.T) {
	t.Run("full config", func(t *testing.T) {
		id := uuid.New()
		secret := "webhook_secret_123"
		now := time.Now()

		config := GitHubConfig{
			ID:                   id,
			Repository:           "owner/repo",
			AutoCreateOnPR:       true,
			AutoDeleteOnMerge:    true,
			DefaultDataCloneMode: DataCloneModeSchemaOnly,
			WebhookSecret:        &secret,
			CreatedAt:            now,
			UpdatedAt:            now,
		}

		assert.Equal(t, id, config.ID)
		assert.Equal(t, "owner/repo", config.Repository)
		assert.True(t, config.AutoCreateOnPR)
		assert.True(t, config.AutoDeleteOnMerge)
		assert.Equal(t, DataCloneModeSchemaOnly, config.DefaultDataCloneMode)
		assert.Equal(t, &secret, config.WebhookSecret)
	})

	t.Run("minimal config", func(t *testing.T) {
		config := GitHubConfig{
			ID:         uuid.New(),
			Repository: "owner/repo",
		}

		assert.False(t, config.AutoCreateOnPR)
		assert.False(t, config.AutoDeleteOnMerge)
		assert.Nil(t, config.WebhookSecret)
	})
}

// =============================================================================
// BranchAccess Struct Tests
// =============================================================================

func TestBranchAccess_Struct(t *testing.T) {
	t.Run("access with granter", func(t *testing.T) {
		id := uuid.New()
		branchID := uuid.New()
		userID := uuid.New()
		grantedBy := uuid.New()
		now := time.Now()

		access := BranchAccess{
			ID:          id,
			BranchID:    branchID,
			UserID:      userID,
			AccessLevel: BranchAccessAdmin,
			GrantedAt:   now,
			GrantedBy:   &grantedBy,
		}

		assert.Equal(t, id, access.ID)
		assert.Equal(t, branchID, access.BranchID)
		assert.Equal(t, userID, access.UserID)
		assert.Equal(t, BranchAccessAdmin, access.AccessLevel)
		assert.Equal(t, now, access.GrantedAt)
		assert.Equal(t, &grantedBy, access.GrantedBy)
	})

	t.Run("access without granter", func(t *testing.T) {
		access := BranchAccess{
			ID:          uuid.New(),
			BranchID:    uuid.New(),
			UserID:      uuid.New(),
			AccessLevel: BranchAccessRead,
			GrantedAt:   time.Now(),
		}

		assert.Nil(t, access.GrantedBy)
	})
}

// =============================================================================
// CreateBranchRequest Struct Tests
// =============================================================================

func TestCreateBranchRequest_Struct(t *testing.T) {
	t.Run("minimal request", func(t *testing.T) {
		req := CreateBranchRequest{
			Name: "my-branch",
		}

		assert.Equal(t, "my-branch", req.Name)
		assert.Nil(t, req.ParentBranchID)
		assert.Empty(t, req.DataCloneMode)
		assert.Empty(t, req.Type)
	})

	t.Run("full request", func(t *testing.T) {
		parentID := uuid.New()
		prNumber := 456
		prURL := "https://github.com/owner/repo/pull/456"
		repo := "owner/repo"
		seedsPath := "./seeds/test"
		expiresAt := time.Now().Add(48 * time.Hour)

		req := CreateBranchRequest{
			Name:           "feature-456",
			ParentBranchID: &parentID,
			DataCloneMode:  DataCloneModeFullClone,
			Type:           BranchTypePreview,
			GitHubPRNumber: &prNumber,
			GitHubPRURL:    &prURL,
			GitHubRepo:     &repo,
			SeedsPath:      &seedsPath,
			ExpiresAt:      &expiresAt,
		}

		assert.Equal(t, "feature-456", req.Name)
		assert.Equal(t, &parentID, req.ParentBranchID)
		assert.Equal(t, DataCloneModeFullClone, req.DataCloneMode)
		assert.Equal(t, BranchTypePreview, req.Type)
		assert.Equal(t, &prNumber, req.GitHubPRNumber)
		assert.Equal(t, &prURL, req.GitHubPRURL)
		assert.Equal(t, &repo, req.GitHubRepo)
		assert.Equal(t, &seedsPath, req.SeedsPath)
		assert.Equal(t, &expiresAt, req.ExpiresAt)
	})
}

// =============================================================================
// ListBranchesFilter Struct Tests
// =============================================================================

func TestListBranchesFilter_Struct(t *testing.T) {
	t.Run("empty filter", func(t *testing.T) {
		filter := ListBranchesFilter{}

		assert.Nil(t, filter.Status)
		assert.Nil(t, filter.Type)
		assert.Nil(t, filter.CreatedBy)
		assert.Nil(t, filter.GitHubRepo)
		assert.Zero(t, filter.Limit)
		assert.Zero(t, filter.Offset)
	})

	t.Run("full filter", func(t *testing.T) {
		status := BranchStatusReady
		branchType := BranchTypePreview
		createdBy := uuid.New()
		repo := "owner/repo"

		filter := ListBranchesFilter{
			Status:     &status,
			Type:       &branchType,
			CreatedBy:  &createdBy,
			GitHubRepo: &repo,
			Limit:      50,
			Offset:     100,
		}

		assert.Equal(t, &status, filter.Status)
		assert.Equal(t, &branchType, filter.Type)
		assert.Equal(t, &createdBy, filter.CreatedBy)
		assert.Equal(t, &repo, filter.GitHubRepo)
		assert.Equal(t, 50, filter.Limit)
		assert.Equal(t, 100, filter.Offset)
	})
}

// =============================================================================
// Benchmarks
// =============================================================================

func BenchmarkBranch_IsMain(b *testing.B) {
	branch := Branch{Type: BranchTypeMain}

	for i := 0; i < b.N; i++ {
		_ = branch.IsMain()
	}
}

func BenchmarkBranch_IsReady(b *testing.B) {
	branch := Branch{Status: BranchStatusReady}

	for i := 0; i < b.N; i++ {
		_ = branch.IsReady()
	}
}
