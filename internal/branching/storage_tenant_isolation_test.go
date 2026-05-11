//go:build integration

package branching

import (
	"context"
	"fmt"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nimbleflux/fluxbase/internal/database"
	"github.com/nimbleflux/fluxbase/test/dbhelpers"
)

const tenantIsolationEncryptionKey = "test-encryption-key-must-be-32-chars!"

// =============================================================================
// GetBranchBySlug Tenant Isolation Tests
// =============================================================================

// TestStorage_TenantIsolation verifies that storage methods properly filter by tenant
// when two branches share the same slug across different tenants.
func TestStorage_TenantIsolation(t *testing.T) {
	testCtx := dbhelpers.NewDBTestContext(t)
	defer testCtx.Close()

	storage := NewStorage(database.NewConnectionWithPool(testCtx.Pool), []byte(tenantIsolationEncryptionKey))

	tenantA := uuid.New()
	tenantB := uuid.New()
	slug := "shared-slug"

	// Create two branches with the same slug but different tenants
	branchA := &Branch{
		Name:          "Tenant A Branch",
		Slug:          slug,
		DatabaseName:  fmt.Sprintf("tenant_a_%s", slug),
		Status:        BranchStatusReady,
		Type:          BranchTypePreview,
		TenantID:      &tenantA,
		DataCloneMode: DataCloneModeSchemaOnly,
	}

	branchB := &Branch{
		Name:          "Tenant B Branch",
		Slug:          slug,
		DatabaseName:  fmt.Sprintf("tenant_b_%s", slug),
		Status:        BranchStatusReady,
		Type:          BranchTypePreview,
		TenantID:      &tenantB,
		DataCloneMode: DataCloneModeSchemaOnly,
	}

	err := storage.CreateBranch(context.Background(), branchA)
	require.NoError(t, err)
	err = storage.CreateBranch(context.Background(), branchB)
	require.NoError(t, err)

	// Cleanup
	defer func() {
		testCtx.ExecuteSQL(`DELETE FROM branching.branches WHERE tenant_id IN ($1, $2)`, tenantA, tenantB)
	}()

	t.Run("GetBranchBySlug with tenant A filter returns tenant A's branch", func(t *testing.T) {
		branch, err := storage.GetBranchBySlug(context.Background(), slug, &tenantA)
		require.NoError(t, err)
		assert.Equal(t, "Tenant A Branch", branch.Name)
		assert.Equal(t, tenantA, *branch.TenantID)
	})

	t.Run("GetBranchBySlug with tenant B filter returns tenant B's branch", func(t *testing.T) {
		branch, err := storage.GetBranchBySlug(context.Background(), slug, &tenantB)
		require.NoError(t, err)
		assert.Equal(t, "Tenant B Branch", branch.Name)
		assert.Equal(t, tenantB, *branch.TenantID)
	})

	t.Run("GetBranchBySlug with nil filter returns first match", func(t *testing.T) {
		// nil means no tenant filter (admin/system access)
		branch, err := storage.GetBranchBySlug(context.Background(), slug, nil)
		require.NoError(t, err)
		assert.Equal(t, slug, branch.Slug)
	})

	t.Run("GetBranchBySlug with wrong tenant returns not found", func(t *testing.T) {
		wrongTenant := uuid.New()
		_, err := storage.GetBranchBySlug(context.Background(), slug, &wrongTenant)
		assert.Equal(t, ErrBranchNotFound, err)
	})
}

// =============================================================================
// GetBranch Tenant Filter Tests
// =============================================================================

// TestStorage_GetBranch_TenantFilter verifies that GetBranch respects tenant scoping.
func TestStorage_GetBranch_TenantFilter(t *testing.T) {
	testCtx := dbhelpers.NewDBTestContext(t)
	defer testCtx.Close()

	storage := NewStorage(database.NewConnectionWithPool(testCtx.Pool), []byte(tenantIsolationEncryptionKey))

	tenantA := uuid.New()
	tenantB := uuid.New()

	branchA := &Branch{
		Name:          "Tenant A Branch",
		Slug:          fmt.Sprintf("get-test-a-%s", uuid.New().String()[:8]),
		DatabaseName:  "get_test_a_db",
		Status:        BranchStatusReady,
		Type:          BranchTypePreview,
		TenantID:      &tenantA,
		DataCloneMode: DataCloneModeSchemaOnly,
	}

	branchB := &Branch{
		Name:          "Tenant B Branch",
		Slug:          fmt.Sprintf("get-test-b-%s", uuid.New().String()[:8]),
		DatabaseName:  "get_test_b_db",
		Status:        BranchStatusReady,
		Type:          BranchTypePreview,
		TenantID:      &tenantB,
		DataCloneMode: DataCloneModeSchemaOnly,
	}

	err := storage.CreateBranch(context.Background(), branchA)
	require.NoError(t, err)
	err = storage.CreateBranch(context.Background(), branchB)
	require.NoError(t, err)

	defer func() {
		testCtx.ExecuteSQL(`DELETE FROM branching.branches WHERE id IN ($1, $2)`, branchA.ID, branchB.ID)
	}()

	tests := []struct {
		name       string
		branchID   uuid.UUID
		tenantID   *uuid.UUID
		expectErr  error
		expectName string
	}{
		{
			name:       "correct tenant returns branch",
			branchID:   branchA.ID,
			tenantID:   &tenantA,
			expectErr:  nil,
			expectName: "Tenant A Branch",
		},
		{
			name:      "wrong tenant returns not found",
			branchID:  branchA.ID,
			tenantID:  &tenantB,
			expectErr: ErrBranchNotFound,
		},
		{
			name:       "nil filter returns any branch",
			branchID:   branchA.ID,
			tenantID:   nil,
			expectErr:  nil,
			expectName: "Tenant A Branch",
		},
		{
			name:       "nil filter returns tenant B branch too",
			branchID:   branchB.ID,
			tenantID:   nil,
			expectErr:  nil,
			expectName: "Tenant B Branch",
		},
		{
			name:      "non-existent branch returns not found",
			branchID:  uuid.New(),
			tenantID:  nil,
			expectErr: ErrBranchNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			branch, err := storage.GetBranch(context.Background(), tt.branchID, tt.tenantID)
			if tt.expectErr != nil {
				assert.Equal(t, tt.expectErr, err)
				assert.Nil(t, branch)
			} else {
				require.NoError(t, err)
				require.NotNil(t, branch)
				assert.Equal(t, tt.expectName, branch.Name)
			}
		})
	}
}

// =============================================================================
// Instance-Level Branch Tests
// =============================================================================

// TestStorage_InstanceLevelBranch verifies that instance-level branches (nil tenant_id) work correctly.
func TestStorage_InstanceLevelBranch(t *testing.T) {
	testCtx := dbhelpers.NewDBTestContext(t)
	defer testCtx.Close()

	storage := NewStorage(database.NewConnectionWithPool(testCtx.Pool), []byte(tenantIsolationEncryptionKey))

	slug := fmt.Sprintf("instance-branch-%s", uuid.New().String()[:8])
	branch := &Branch{
		Name:          "Instance Branch",
		Slug:          slug,
		DatabaseName:  "instance_test_db",
		Status:        BranchStatusReady,
		Type:          BranchTypePreview,
		TenantID:      nil, // Instance-level
		DataCloneMode: DataCloneModeSchemaOnly,
	}

	err := storage.CreateBranch(context.Background(), branch)
	require.NoError(t, err)

	defer func() {
		testCtx.ExecuteSQL(`DELETE FROM branching.branches WHERE id = $1`, branch.ID)
	}()

	t.Run("GetBranchBySlug with nil filter returns instance branch", func(t *testing.T) {
		found, err := storage.GetBranchBySlug(context.Background(), slug, nil)
		require.NoError(t, err)
		assert.Equal(t, "Instance Branch", found.Name)
		assert.Nil(t, found.TenantID)
	})

	t.Run("GetBranchBySlug with tenant filter does not return instance branch", func(t *testing.T) {
		tenantID := uuid.New()
		_, err := storage.GetBranchBySlug(context.Background(), slug, &tenantID)
		assert.Equal(t, ErrBranchNotFound, err)
	})

	t.Run("GetBranch with nil filter returns instance branch", func(t *testing.T) {
		found, err := storage.GetBranch(context.Background(), branch.ID, nil)
		require.NoError(t, err)
		assert.Equal(t, "Instance Branch", found.Name)
	})

	t.Run("GetBranch with tenant filter does not return instance branch", func(t *testing.T) {
		tenantID := uuid.New()
		_, err := storage.GetBranch(context.Background(), branch.ID, &tenantID)
		assert.Equal(t, ErrBranchNotFound, err)
	})
}

// =============================================================================
// DeleteBranch Tenant Filter Tests
// =============================================================================

// TestStorage_DeleteBranch_TenantFilter verifies that DeleteBranch respects tenant scoping.
func TestStorage_DeleteBranch_TenantFilter(t *testing.T) {
	testCtx := dbhelpers.NewDBTestContext(t)
	defer testCtx.Close()

	storage := NewStorage(database.NewConnectionWithPool(testCtx.Pool), []byte(tenantIsolationEncryptionKey))

	tenantA := uuid.New()
	tenantB := uuid.New()

	t.Run("DeleteBranch with wrong tenant does not delete", func(t *testing.T) {
		branchA := &Branch{
			Name:          "Delete Test A",
			Slug:          fmt.Sprintf("del-test-a-%s", uuid.New().String()[:8]),
			DatabaseName:  fmt.Sprintf("del_test_a_%s", uuid.New().String()[:8]),
			Status:        BranchStatusReady,
			Type:          BranchTypePreview,
			TenantID:      &tenantA,
			DataCloneMode: DataCloneModeSchemaOnly,
		}

		err := storage.CreateBranch(context.Background(), branchA)
		require.NoError(t, err)
		defer testCtx.ExecuteSQL(`DELETE FROM branching.branches WHERE id = $1`, branchA.ID)

		err = storage.DeleteBranch(context.Background(), branchA.ID, &tenantB)
		assert.Equal(t, ErrBranchNotFound, err)

		// Verify branch still exists
		found, err := storage.GetBranch(context.Background(), branchA.ID, nil)
		require.NoError(t, err)
		assert.Equal(t, "Delete Test A", found.Name)
	})

	t.Run("DeleteBranch with correct tenant deletes", func(t *testing.T) {
		branchB := &Branch{
			Name:          "Delete Test B",
			Slug:          fmt.Sprintf("del-test-b-%s", uuid.New().String()[:8]),
			DatabaseName:  fmt.Sprintf("del_test_b_%s", uuid.New().String()[:8]),
			Status:        BranchStatusReady,
			Type:          BranchTypePreview,
			TenantID:      &tenantA,
			DataCloneMode: DataCloneModeSchemaOnly,
		}

		err := storage.CreateBranch(context.Background(), branchB)
		require.NoError(t, err)

		err = storage.DeleteBranch(context.Background(), branchB.ID, &tenantA)
		require.NoError(t, err)

		// Verify branch is deleted
		_, err = storage.GetBranch(context.Background(), branchB.ID, nil)
		assert.Equal(t, ErrBranchNotFound, err)
	})

	t.Run("DeleteBranch with nil filter deletes regardless of tenant", func(t *testing.T) {
		branchC := &Branch{
			Name:          "Delete Test C",
			Slug:          fmt.Sprintf("del-test-c-%s", uuid.New().String()[:8]),
			DatabaseName:  fmt.Sprintf("del_test_c_%s", uuid.New().String()[:8]),
			Status:        BranchStatusReady,
			Type:          BranchTypePreview,
			TenantID:      &tenantB,
			DataCloneMode: DataCloneModeSchemaOnly,
		}

		err := storage.CreateBranch(context.Background(), branchC)
		require.NoError(t, err)

		// Delete with nil filter (admin access)
		err = storage.DeleteBranch(context.Background(), branchC.ID, nil)
		require.NoError(t, err)

		_, err = storage.GetBranch(context.Background(), branchC.ID, nil)
		assert.Equal(t, ErrBranchNotFound, err)
	})
}

// =============================================================================
// ListBranches Tenant Filter Tests
// =============================================================================

// TestStorage_ListBranches_TenantFilter verifies that ListBranches respects tenant scoping.
func TestStorage_ListBranches_TenantFilter(t *testing.T) {
	testCtx := dbhelpers.NewDBTestContext(t)
	defer testCtx.Close()

	storage := NewStorage(database.NewConnectionWithPool(testCtx.Pool), []byte(tenantIsolationEncryptionKey))

	tenantA := uuid.New()
	tenantB := uuid.New()

	// Create branches for tenant A
	branchA1 := &Branch{
		Name:          "Tenant A Branch 1",
		Slug:          fmt.Sprintf("tenant-a-1-%s", uuid.New().String()[:8]),
		DatabaseName:  fmt.Sprintf("tenant_a_1_%s", uuid.New().String()[:8]),
		Status:        BranchStatusReady,
		Type:          BranchTypePreview,
		TenantID:      &tenantA,
		DataCloneMode: DataCloneModeSchemaOnly,
	}
	branchA2 := &Branch{
		Name:          "Tenant A Branch 2",
		Slug:          fmt.Sprintf("tenant-a-2-%s", uuid.New().String()[:8]),
		DatabaseName:  fmt.Sprintf("tenant_a_2_%s", uuid.New().String()[:8]),
		Status:        BranchStatusReady,
		Type:          BranchTypePreview,
		TenantID:      &tenantA,
		DataCloneMode: DataCloneModeSchemaOnly,
	}

	// Create branch for tenant B
	branchB1 := &Branch{
		Name:          "Tenant B Branch 1",
		Slug:          fmt.Sprintf("tenant-b-1-%s", uuid.New().String()[:8]),
		DatabaseName:  fmt.Sprintf("tenant_b_1_%s", uuid.New().String()[:8]),
		Status:        BranchStatusReady,
		Type:          BranchTypePreview,
		TenantID:      &tenantB,
		DataCloneMode: DataCloneModeSchemaOnly,
	}

	for _, b := range []*Branch{branchA1, branchA2, branchB1} {
		err := storage.CreateBranch(context.Background(), b)
		require.NoError(t, err)
	}

	defer func() {
		testCtx.ExecuteSQL(`DELETE FROM branching.branches WHERE id IN ($1, $2, $3)`,
			branchA1.ID, branchA2.ID, branchB1.ID)
	}()

	t.Run("ListBranches with tenant A filter returns only tenant A branches", func(t *testing.T) {
		branches, err := storage.ListBranches(context.Background(), ListBranchesFilter{
			TenantID: &tenantA,
		})
		require.NoError(t, err)

		for _, b := range branches {
			require.NotNil(t, b.TenantID, "expected tenant_id to be set")
			assert.Equal(t, tenantA, *b.TenantID, "branch should belong to tenant A")
		}
	})

	t.Run("ListBranches with tenant B filter returns only tenant B branches", func(t *testing.T) {
		branches, err := storage.ListBranches(context.Background(), ListBranchesFilter{
			TenantID: &tenantB,
		})
		require.NoError(t, err)

		for _, b := range branches {
			require.NotNil(t, b.TenantID, "expected tenant_id to be set")
			assert.Equal(t, tenantB, *b.TenantID, "branch should belong to tenant B")
		}
	})

	t.Run("ListBranches with nil filter returns all branches", func(t *testing.T) {
		branches, err := storage.ListBranches(context.Background(), ListBranchesFilter{})
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(branches), 3, "should return at least the 3 test branches")
	})

	t.Run("ListBranches with non-existent tenant returns empty", func(t *testing.T) {
		unknownTenant := uuid.New()
		branches, err := storage.ListBranches(context.Background(), ListBranchesFilter{
			TenantID: &unknownTenant,
		})
		require.NoError(t, err)
		assert.Empty(t, branches)
	})
}

// =============================================================================
// CountBranches Tenant Filter Tests
// =============================================================================

// TestStorage_CountBranches_TenantFilter verifies that CountBranches respects tenant scoping.
func TestStorage_CountBranches_TenantFilter(t *testing.T) {
	testCtx := dbhelpers.NewDBTestContext(t)
	defer testCtx.Close()

	storage := NewStorage(database.NewConnectionWithPool(testCtx.Pool), []byte(tenantIsolationEncryptionKey))

	tenantA := uuid.New()
	tenantB := uuid.New()

	// Create two branches for tenant A
	for i := 0; i < 2; i++ {
		branch := &Branch{
			Name:          fmt.Sprintf("Count Test A-%d", i),
			Slug:          fmt.Sprintf("count-a-%d-%s", i, uuid.New().String()[:8]),
			DatabaseName:  fmt.Sprintf("count_a_%d_%s", i, uuid.New().String()[:8]),
			Status:        BranchStatusReady,
			Type:          BranchTypePreview,
			TenantID:      &tenantA,
			DataCloneMode: DataCloneModeSchemaOnly,
		}
		err := storage.CreateBranch(context.Background(), branch)
		require.NoError(t, err)
		defer testCtx.ExecuteSQL(`DELETE FROM branching.branches WHERE id = $1`, branch.ID)
	}

	// Create one branch for tenant B
	branchB := &Branch{
		Name:          "Count Test B",
		Slug:          fmt.Sprintf("count-b-%s", uuid.New().String()[:8]),
		DatabaseName:  fmt.Sprintf("count_b_%s", uuid.New().String()[:8]),
		Status:        BranchStatusReady,
		Type:          BranchTypePreview,
		TenantID:      &tenantB,
		DataCloneMode: DataCloneModeSchemaOnly,
	}
	err := storage.CreateBranch(context.Background(), branchB)
	require.NoError(t, err)
	defer testCtx.ExecuteSQL(`DELETE FROM branching.branches WHERE id = $1`, branchB.ID)

	t.Run("CountBranches for tenant A returns at least 2", func(t *testing.T) {
		count, err := storage.CountBranches(context.Background(), ListBranchesFilter{
			TenantID: &tenantA,
		})
		require.NoError(t, err)
		assert.GreaterOrEqual(t, count, 2)
	})

	t.Run("CountBranches for tenant B returns at least 1", func(t *testing.T) {
		count, err := storage.CountBranches(context.Background(), ListBranchesFilter{
			TenantID: &tenantB,
		})
		require.NoError(t, err)
		assert.GreaterOrEqual(t, count, 1)
	})

	t.Run("CountBranches with no filter returns total", func(t *testing.T) {
		count, err := storage.CountBranches(context.Background(), ListBranchesFilter{})
		require.NoError(t, err)
		assert.GreaterOrEqual(t, count, 3)
	})

	t.Run("CountBranches for non-existent tenant returns 0", func(t *testing.T) {
		unknownTenant := uuid.New()
		count, err := storage.CountBranches(context.Background(), ListBranchesFilter{
			TenantID: &unknownTenant,
		})
		require.NoError(t, err)
		assert.Equal(t, 0, count)
	})
}

// =============================================================================
// CountBranchesByTenant Tests
// =============================================================================

// TestStorage_CountBranchesByTenant verifies the dedicated tenant counting method.
func TestStorage_CountBranchesByTenant(t *testing.T) {
	testCtx := dbhelpers.NewDBTestContext(t)
	defer testCtx.Close()

	storage := NewStorage(database.NewConnectionWithPool(testCtx.Pool), []byte(tenantIsolationEncryptionKey))

	tenantID := uuid.New()

	// Create a branch for the tenant
	branch := &Branch{
		Name:          "Tenant Count Test",
		Slug:          fmt.Sprintf("tenant-count-%s", uuid.New().String()[:8]),
		DatabaseName:  fmt.Sprintf("tenant_count_%s", uuid.New().String()[:8]),
		Status:        BranchStatusReady,
		Type:          BranchTypePreview,
		TenantID:      &tenantID,
		DataCloneMode: DataCloneModeSchemaOnly,
	}
	err := storage.CreateBranch(context.Background(), branch)
	require.NoError(t, err)
	defer testCtx.ExecuteSQL(`DELETE FROM branching.branches WHERE id = $1`, branch.ID)

	t.Run("counts branches for existing tenant", func(t *testing.T) {
		count, err := storage.CountBranchesByTenant(context.Background(), tenantID)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, count, 1)
	})

	t.Run("returns 0 for non-existent tenant", func(t *testing.T) {
		count, err := storage.CountBranchesByTenant(context.Background(), uuid.New())
		require.NoError(t, err)
		assert.Equal(t, 0, count)
	})
}
