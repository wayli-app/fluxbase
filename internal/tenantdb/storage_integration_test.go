//go:build integration

package tenantdb

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nimbleflux/fluxbase/test/dbhelpers"
)

func TestStorage_Integration_CRUD(t *testing.T) {
	testCtx := dbhelpers.NewDBTestContext(t)
	defer testCtx.Close()

	storage := NewRepository(testCtx.Pool)
	ctx := context.Background()

	t.Run("create and get tenant", func(t *testing.T) {
		tenant := &Tenant{
			Slug:     "itest-" + uuid.New().String()[:8],
			Name:     "Integration Test Tenant",
			Status:   TenantStatusActive,
			Metadata: map[string]any{"key": "value", "nested": map[string]any{"a": 1}},
		}

		err := storage.CreateTenant(ctx, tenant)
		require.NoError(t, err)
		assert.NotEmpty(t, tenant.ID)

		t.Cleanup(func() {
			_, _ = testCtx.Pool.Exec(ctx, "DELETE FROM platform.tenants WHERE id = $1", tenant.ID)
		})

		got, err := storage.GetTenant(ctx, tenant.ID)
		require.NoError(t, err)
		assert.Equal(t, tenant.Slug, got.Slug)
		assert.Equal(t, tenant.Name, got.Name)
		assert.Equal(t, TenantStatusActive, got.Status)
		assert.Equal(t, "value", got.Metadata["key"])
	})

	t.Run("get tenant by slug", func(t *testing.T) {
		slug := "itest-slug-" + uuid.New().String()[:8]
		tenant := &Tenant{
			Slug:   slug,
			Name:   "Slug Test Tenant",
			Status: TenantStatusActive,
		}

		err := storage.CreateTenant(ctx, tenant)
		require.NoError(t, err)
		t.Cleanup(func() {
			_, _ = testCtx.Pool.Exec(ctx, "DELETE FROM platform.tenants WHERE id = $1", tenant.ID)
		})

		got, err := storage.GetTenantBySlug(ctx, slug)
		require.NoError(t, err)
		assert.Equal(t, tenant.ID, got.ID)
	})

	t.Run("get non-existent tenant returns error", func(t *testing.T) {
		_, err := storage.GetTenant(ctx, uuid.New().String())
		assert.ErrorIs(t, err, ErrTenantNotFound)
	})

	t.Run("get default tenant", func(t *testing.T) {
		// Create a default tenant
		tenant := &Tenant{
			Slug:      "itest-default-" + uuid.New().String()[:8],
			Name:      "Default Test Tenant",
			Status:    TenantStatusActive,
			IsDefault: true,
			Metadata:  map[string]any{},
		}

		err := storage.CreateTenant(ctx, tenant)
		require.NoError(t, err)
		t.Cleanup(func() {
			_, _ = testCtx.Pool.Exec(ctx, "DELETE FROM platform.tenants WHERE id = $1", tenant.ID)
		})

		got, err := storage.GetDefaultTenant(ctx)
		require.NoError(t, err)
		assert.Equal(t, tenant.ID, got.ID)
		assert.True(t, got.IsDefault)
	})

	t.Run("update tenant status", func(t *testing.T) {
		tenant := &Tenant{
			Slug:   "itest-status-" + uuid.New().String()[:8],
			Name:   "Status Test Tenant",
			Status: TenantStatusCreating,
		}

		err := storage.CreateTenant(ctx, tenant)
		require.NoError(t, err)
		t.Cleanup(func() {
			_, _ = testCtx.Pool.Exec(ctx, "DELETE FROM platform.tenants WHERE id = $1", tenant.ID)
		})

		err = storage.UpdateTenantStatus(ctx, tenant.ID, TenantStatusActive)
		require.NoError(t, err)

		got, err := storage.GetTenant(ctx, tenant.ID)
		require.NoError(t, err)
		assert.Equal(t, TenantStatusActive, got.Status)
	})

	t.Run("update tenant db_name", func(t *testing.T) {
		tenant := &Tenant{
			Slug:   "itest-dbname-" + uuid.New().String()[:8],
			Name:   "DBName Test Tenant",
			Status: TenantStatusActive,
		}

		err := storage.CreateTenant(ctx, tenant)
		require.NoError(t, err)
		t.Cleanup(func() {
			_, _ = testCtx.Pool.Exec(ctx, "DELETE FROM platform.tenants WHERE id = $1", tenant.ID)
		})

		err = storage.UpdateTenantDBName(ctx, tenant.ID, "tenant_testdb")
		require.NoError(t, err)

		got, err := storage.GetTenant(ctx, tenant.ID)
		require.NoError(t, err)
		require.NotNil(t, got.DBName)
		assert.Equal(t, "tenant_testdb", *got.DBName)
	})

	t.Run("soft and hard delete", func(t *testing.T) {
		tenant := &Tenant{
			Slug:   "itest-del-" + uuid.New().String()[:8],
			Name:   "Delete Test Tenant",
			Status: TenantStatusActive,
		}

		err := storage.CreateTenant(ctx, tenant)
		require.NoError(t, err)

		err = storage.SoftDeleteTenant(ctx, tenant.ID)
		require.NoError(t, err)

		// Should not be found after soft delete
		_, err = storage.GetTenant(ctx, tenant.ID)
		assert.ErrorIs(t, err, ErrTenantNotFound)

		// Hard delete removes record completely
		err = storage.HardDeleteTenant(ctx, tenant.ID)
		require.NoError(t, err)
	})

	t.Run("count tenants", func(t *testing.T) {
		countBefore, err := storage.CountTenants(ctx)
		require.NoError(t, err)

		tenant := &Tenant{
			Slug:   "itest-count-" + uuid.New().String()[:8],
			Name:   "Count Test Tenant",
			Status: TenantStatusActive,
		}

		err = storage.CreateTenant(ctx, tenant)
		require.NoError(t, err)
		t.Cleanup(func() {
			_, _ = testCtx.Pool.Exec(ctx, "DELETE FROM platform.tenants WHERE id = $1", tenant.ID)
		})

		countAfter, err := storage.CountTenants(ctx)
		require.NoError(t, err)
		assert.Equal(t, countBefore+1, countAfter)
	})

	t.Run("get all active tenants", func(t *testing.T) {
		tenant := &Tenant{
			Slug:   "itest-active-" + uuid.New().String()[:8],
			Name:   "Active Test Tenant",
			Status: TenantStatusActive,
		}

		err := storage.CreateTenant(ctx, tenant)
		require.NoError(t, err)
		t.Cleanup(func() {
			_, _ = testCtx.Pool.Exec(ctx, "DELETE FROM platform.tenants WHERE id = $1", tenant.ID)
		})

		tenants, err := storage.GetAllActiveTenants(ctx)
		require.NoError(t, err)
		assert.NotEmpty(t, tenants)
	})

	t.Run("metadata JSON roundtrip", func(t *testing.T) {
		complexMeta := map[string]any{
			"string":  "hello",
			"number":  float64(42),
			"boolean": true,
			"nested": map[string]any{
				"deep": map[string]any{
					"value": "inner",
				},
			},
			"array": []any{1, "two", true},
		}

		tenant := &Tenant{
			Slug:     "itest-meta-" + uuid.New().String()[:8],
			Name:     "Metadata Test Tenant",
			Status:   TenantStatusActive,
			Metadata: complexMeta,
		}

		err := storage.CreateTenant(ctx, tenant)
		require.NoError(t, err)
		t.Cleanup(func() {
			_, _ = testCtx.Pool.Exec(ctx, "DELETE FROM platform.tenants WHERE id = $1", tenant.ID)
		})

		got, err := storage.GetTenant(ctx, tenant.ID)
		require.NoError(t, err)

		assert.Equal(t, "hello", got.Metadata["string"])
		assert.Equal(t, float64(42), got.Metadata["number"])
		assert.Equal(t, true, got.Metadata["boolean"])
	})
}

func TestStorage_Integration_UserAssignments(t *testing.T) {
	testCtx := dbhelpers.NewDBTestContext(t)
	defer testCtx.Close()

	storage := NewRepository(testCtx.Pool)
	ctx := context.Background()

	tenant := &Tenant{
		Slug:   "itest-assign-" + uuid.New().String()[:8],
		Name:   "Assignment Test Tenant",
		Status: TenantStatusActive,
	}

	err := storage.CreateTenant(ctx, tenant)
	require.NoError(t, err)
	t.Cleanup(func() {
		_, _ = testCtx.Pool.Exec(ctx, "DELETE FROM platform.tenants WHERE id = $1", tenant.ID)
	})

	userID := uuid.New().String()

	t.Run("assign user to tenant", func(t *testing.T) {
		err := storage.AssignUserToTenant(ctx, userID, tenant.ID)
		require.NoError(t, err)
	})

	t.Run("check user is assigned", func(t *testing.T) {
		assigned, err := storage.IsUserAssignedToTenant(ctx, userID, tenant.ID)
		require.NoError(t, err)
		assert.True(t, assigned)
	})

	t.Run("get tenant assignments for user", func(t *testing.T) {
		ids, err := storage.GetTenantAssignments(ctx, userID)
		require.NoError(t, err)
		assert.Contains(t, ids, tenant.ID)
	})

	t.Run("remove user from tenant", func(t *testing.T) {
		err := storage.RemoveUserFromTenant(ctx, userID, tenant.ID)
		require.NoError(t, err)

		assigned, err := storage.IsUserAssignedToTenant(ctx, userID, tenant.ID)
		require.NoError(t, err)
		assert.False(t, assigned)
	})

	t.Run("get tenants for user returns empty after removal", func(t *testing.T) {
		tenants, err := storage.GetTenantsForUser(ctx, userID)
		require.NoError(t, err)
		assert.Empty(t, tenants)
	})
}
