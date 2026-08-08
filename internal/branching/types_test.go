package branching

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

// =============================================================================
// Branch Tenant Membership Tests
// =============================================================================
//
// BelongsToTenant / IsInstanceLevel are the inverse checks over Branch.TenantID
// (types.go:67,75). Currently 0.0% under -short. Pure logic over a *Branch.

func TestBranch_BelongsToTenant(t *testing.T) {
	t.Parallel()
	tenant := uuid.New()
	other := uuid.New()

	tests := []struct {
		name   string
		branch *Branch
		want   bool
	}{
		{"nil tenant id returns false", &Branch{TenantID: nil}, false},
		{"matching tenant returns true", &Branch{TenantID: &tenant}, true},
		{"different tenant returns false", &Branch{TenantID: &other}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, tt.branch.BelongsToTenant(tenant))
		})
	}
}

func TestBranch_IsInstanceLevel(t *testing.T) {
	t.Parallel()
	tenant := uuid.New()
	tests := []struct {
		name   string
		tenant *uuid.UUID
		want   bool
	}{
		{"nil tenant is instance-level", nil, true},
		{"tenant-scoped is not instance-level", &tenant, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			b := &Branch{TenantID: tt.tenant}
			assert.Equal(t, tt.want, b.IsInstanceLevel())
		})
	}
}

// =============================================================================
// Branch access helpers — push HasAccess / GetAccessLevel to full coverage
// =============================================================================

func TestBranch_HasAccess_FullMatrix(t *testing.T) {
	t.Parallel()
	userA := uuid.New()
	userB := uuid.New()

	t.Run("nil access list returns false", func(t *testing.T) {
		t.Parallel()
		b := &Branch{Access: nil}
		assert.False(t, b.HasAccess(userA))
	})

	t.Run("present user returns true", func(t *testing.T) {
		t.Parallel()
		b := &Branch{Access: []BranchAccess{{UserID: userA, AccessLevel: BranchAccessRead}}}
		assert.True(t, b.HasAccess(userA))
	})

	t.Run("absent user returns false", func(t *testing.T) {
		t.Parallel()
		b := &Branch{Access: []BranchAccess{{UserID: userA, AccessLevel: BranchAccessRead}}}
		assert.False(t, b.HasAccess(userB))
	})

	t.Run("empty access list returns false", func(t *testing.T) {
		t.Parallel()
		b := &Branch{Access: []BranchAccess{}}
		assert.False(t, b.HasAccess(userA))
	})
}

func TestBranch_GetAccessLevel_FullMatrix(t *testing.T) {
	t.Parallel()
	userA := uuid.New()
	userB := uuid.New()
	level := BranchAccessAdmin

	t.Run("nil access list returns nil", func(t *testing.T) {
		t.Parallel()
		b := &Branch{Access: nil}
		assert.Nil(t, b.GetAccessLevel(userA))
	})

	t.Run("present user returns level pointer", func(t *testing.T) {
		t.Parallel()
		b := &Branch{Access: []BranchAccess{{UserID: userA, AccessLevel: level}}}
		got := b.GetAccessLevel(userA)
		require.NotNil(t, got)
		assert.Equal(t, level, *got)
	})

	t.Run("absent user returns nil", func(t *testing.T) {
		t.Parallel()
		b := &Branch{Access: []BranchAccess{{UserID: userA, AccessLevel: level}}}
		assert.Nil(t, b.GetAccessLevel(userB))
	})
}
