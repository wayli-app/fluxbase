package tenantdb

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTestStorage(t *testing.T) (*Repository, *pgxpool.Pool) {
	t.Helper()

	// Skip in short mode
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	databaseURL := getTestDatabaseURL(t)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		t.Skip("Requires database connection - skipping integration test")
	}

	// Verify connection is working
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		t.Skip("Requires database connection - skipping integration test")
	}

	t.Cleanup(func() {
		pool.Close()
	})

	storage := NewRepository(pool)
	return storage, pool
}

func getTestDatabaseURL(t *testing.T) string {
	t.Helper()

	dbName := os.Getenv("FLUXBASE_TEST_DATABASE")
	if dbName == "" {
		dbName = "fluxbase_test"
	}
	dbUser := os.Getenv("FLUXBASE_DATABASE_ADMIN_USER")
	if dbUser == "" {
		dbUser = "postgres"
	}
	dbPassword := os.Getenv("FLUXBASE_DATABASE_ADMIN_PASSWORD")
	if dbPassword == "" {
		dbPassword = "postgres"
	}
	dbHost := os.Getenv("FLUXBASE_DATABASE_HOST")
	if dbHost == "" {
		dbHost = "localhost"
	}
	dbPort := os.Getenv("FLUXBASE_DATABASE_PORT")
	if dbPort == "" {
		dbPort = "5432"
	}
	url := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable", dbUser, dbPassword, dbHost, dbPort, dbName)
	return url
}

func TestStorage_GetTenant(t *testing.T) {
	storage, pool := setupTestStorage(t)
	ctx := context.Background()

	t.Run("returns error for non-existent tenant", func(t *testing.T) {
		_, err := storage.GetTenant(ctx, "00000000-0000-0000-0000-000000000000")
		assert.ErrorIs(t, err, ErrTenantNotFound)
	})

	t.Run("returns tenant for valid id", func(t *testing.T) {
		tenant := &Tenant{
			Slug:     "test-tenant-get",
			Name:     "Test Tenant Get",
			Status:   TenantStatusActive,
			Metadata: map[string]any{"key": "value"},
		}

		err := storage.CreateTenant(ctx, tenant)
		require.NoError(t, err)

		t.Cleanup(func() {
			pool.Exec(ctx, "DELETE FROM platform.tenants WHERE id = $1", tenant.ID)
		})

		got, err := storage.GetTenant(ctx, tenant.ID)
		require.NoError(t, err)

		assert.Equal(t, tenant.ID, got.ID)
		assert.Equal(t, tenant.Slug, got.Slug)
		assert.Equal(t, tenant.Name, got.Name)
		assert.Equal(t, TenantStatusActive, got.Status)
		assert.Equal(t, "value", got.Metadata["key"])
	})
}

func TestStorage_GetTenantBySlug(t *testing.T) {
	storage, pool := setupTestStorage(t)
	ctx := context.Background()

	t.Run("returns error for non-existent slug", func(t *testing.T) {
		_, err := storage.GetTenantBySlug(ctx, "non-existent-slug")
		assert.ErrorIs(t, err, ErrTenantNotFound)
	})

	t.Run("returns tenant for valid slug", func(t *testing.T) {
		tenant := &Tenant{
			Slug:   "test-tenant-slug",
			Name:   "Test Tenant Slug",
			Status: TenantStatusActive,
		}

		err := storage.CreateTenant(ctx, tenant)
		require.NoError(t, err)

		t.Cleanup(func() {
			pool.Exec(ctx, "DELETE FROM platform.tenants WHERE id = $1", tenant.ID)
		})

		got, err := storage.GetTenantBySlug(ctx, "test-tenant-slug")
		require.NoError(t, err)
		assert.Equal(t, tenant.ID, got.ID)
	})
}

func TestStorage_GetDefaultTenant(t *testing.T) {
	storage, pool := setupTestStorage(t)
	ctx := context.Background()

	t.Run("returns error when no default tenant exists", func(t *testing.T) {
		pool.Exec(ctx, "UPDATE platform.tenants SET is_default = false")
		defer pool.Exec(ctx, "UPDATE platform.tenants SET is_default = true WHERE is_default IS NULL OR is_default = false LIMIT 1")

		_, err := storage.GetDefaultTenant(ctx)
		assert.ErrorIs(t, err, ErrNoDefaultTenant)
	})
}

func TestStorage_CreateTenant(t *testing.T) {
	storage, pool := setupTestStorage(t)
	ctx := context.Background()

	t.Run("creates tenant with generated ID", func(t *testing.T) {
		tenant := &Tenant{
			Slug:   "test-create-tenant",
			Name:   "Test Create Tenant",
			Status: TenantStatusActive,
		}

		err := storage.CreateTenant(ctx, tenant)
		require.NoError(t, err)

		t.Cleanup(func() {
			pool.Exec(ctx, "DELETE FROM platform.tenants WHERE id = $1", tenant.ID)
		})

		assert.NotEmpty(t, tenant.ID)
		assert.NotZero(t, tenant.CreatedAt)
	})

	t.Run("creates tenant with db_name", func(t *testing.T) {
		dbName := "tenant_test_db"
		tenant := &Tenant{
			Slug:   "test-tenant-with-db",
			Name:   "Test Tenant With DB",
			DBName: &dbName,
			Status: TenantStatusCreating,
		}

		err := storage.CreateTenant(ctx, tenant)
		require.NoError(t, err)

		t.Cleanup(func() {
			pool.Exec(ctx, "DELETE FROM platform.tenants WHERE id = $1", tenant.ID)
		})

		assert.Equal(t, &dbName, tenant.DBName)
	})
}

func TestStorage_UpdateTenantStatus(t *testing.T) {
	storage, pool := setupTestStorage(t)
	ctx := context.Background()

	t.Run("updates tenant status", func(t *testing.T) {
		tenant := &Tenant{
			Slug:   "test-update-status",
			Name:   "Test Update Status",
			Status: TenantStatusCreating,
		}

		err := storage.CreateTenant(ctx, tenant)
		require.NoError(t, err)

		t.Cleanup(func() {
			pool.Exec(ctx, "DELETE FROM platform.tenants WHERE id = $1", tenant.ID)
		})

		err = storage.UpdateTenantStatus(ctx, tenant.ID, TenantStatusActive)
		require.NoError(t, err)

		got, err := storage.GetTenant(ctx, tenant.ID)
		require.NoError(t, err)
		assert.Equal(t, TenantStatusActive, got.Status)
	})

	t.Run("returns error for non-existent tenant", func(t *testing.T) {
		err := storage.UpdateTenantStatus(ctx, "00000000-0000-0000-0000-000000000000", TenantStatusActive)
		assert.ErrorIs(t, err, ErrTenantNotFound)
	})
}

func TestStorage_UpdateTenantDBName(t *testing.T) {
	storage, pool := setupTestStorage(t)
	ctx := context.Background()

	t.Run("updates tenant db_name", func(t *testing.T) {
		tenant := &Tenant{
			Slug:   "test-update-dbname",
			Name:   "Test Update DBName",
			Status: TenantStatusActive,
		}

		err := storage.CreateTenant(ctx, tenant)
		require.NoError(t, err)

		t.Cleanup(func() {
			pool.Exec(ctx, "DELETE FROM platform.tenants WHERE id = $1", tenant.ID)
		})

		err = storage.UpdateTenantDBName(ctx, tenant.ID, "tenant_new_db")
		require.NoError(t, err)

		got, err := storage.GetTenant(ctx, tenant.ID)
		require.NoError(t, err)
		assert.Equal(t, "tenant_new_db", *got.DBName)
	})
}

func TestStorage_SoftDeleteTenant(t *testing.T) {
	storage, pool := setupTestStorage(t)
	ctx := context.Background()

	t.Run("soft deletes tenant", func(t *testing.T) {
		tenant := &Tenant{
			Slug:   "test-soft-delete",
			Name:   "Test Soft Delete",
			Status: TenantStatusActive,
		}

		err := storage.CreateTenant(ctx, tenant)
		require.NoError(t, err)

		t.Cleanup(func() {
			pool.Exec(ctx, "DELETE FROM platform.tenants WHERE id = $1", tenant.ID)
		})

		err = storage.SoftDeleteTenant(ctx, tenant.ID)
		require.NoError(t, err)

		_, err = storage.GetTenant(ctx, tenant.ID)
		assert.ErrorIs(t, err, ErrTenantNotFound)
	})
}

func TestStorage_HardDeleteTenant(t *testing.T) {
	storage, pool := setupTestStorage(t)
	ctx := context.Background()

	t.Run("hard deletes tenant", func(t *testing.T) {
		tenant := &Tenant{
			Slug:   "test-hard-delete",
			Name:   "Test Hard Delete",
			Status: TenantStatusActive,
		}

		err := storage.CreateTenant(ctx, tenant)
		require.NoError(t, err)

		err = storage.HardDeleteTenant(ctx, tenant.ID)
		require.NoError(t, err)

		var count int
		err = pool.QueryRow(ctx, "SELECT COUNT(*) FROM platform.tenants WHERE id = $1", tenant.ID).Scan(&count)
		require.NoError(t, err)
		assert.Equal(t, 0, count)
	})
}

func TestStorage_AssignUserToTenant(t *testing.T) {
	storage, pool := setupTestStorage(t)
	ctx := context.Background()

	t.Run("assigns user to tenant", func(t *testing.T) {
		tenant := &Tenant{
			Slug:   "test-assign-user",
			Name:   "Test Assign User",
			Status: TenantStatusActive,
		}

		err := storage.CreateTenant(ctx, tenant)
		require.NoError(t, err)

		t.Cleanup(func() {
			pool.Exec(ctx, "DELETE FROM platform.tenant_admin_assignments WHERE tenant_id = $1", tenant.ID)
			pool.Exec(ctx, "DELETE FROM platform.tenants WHERE id = $1", tenant.ID)
		})

		var userID string
		err = pool.QueryRow(ctx, "SELECT id FROM auth.users LIMIT 1").Scan(&userID)
		if err != nil {
			t.Skip("No users available for testing")
		}

		err = storage.AssignUserToTenant(ctx, userID, tenant.ID)
		require.NoError(t, err)

		hasAccess, err := storage.IsUserAssignedToTenant(ctx, userID, tenant.ID)
		require.NoError(t, err)
		assert.True(t, hasAccess)
	})
}

func TestStorage_IsUserAssignedToTenant(t *testing.T) {
	storage, _ := setupTestStorage(t)
	ctx := context.Background()

	t.Run("returns false for unassigned user", func(t *testing.T) {
		hasAccess, err := storage.IsUserAssignedToTenant(ctx, "00000000-0000-0000-0000-000000000000", "00000000-0000-0000-0000-000000000001")
		require.NoError(t, err)
		assert.False(t, hasAccess)
	})
}

func TestStorage_GetTenantsForUser(t *testing.T) {
	storage, _ := setupTestStorage(t)
	ctx := context.Background()

	t.Run("returns empty list for user with no tenants", func(t *testing.T) {
		tenants, err := storage.GetTenantsForUser(ctx, "00000000-0000-0000-0000-000000000000")
		require.NoError(t, err)
		assert.Empty(t, tenants)
	})
}

func TestStorage_CountTenants(t *testing.T) {
	storage, _ := setupTestStorage(t)
	ctx := context.Background()

	count, err := storage.CountTenants(ctx)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, count, 0)
}

func TestStorage_GetAllActiveTenants(t *testing.T) {
	storage, _ := setupTestStorage(t)
	ctx := context.Background()

	tenants, err := storage.GetAllActiveTenants(ctx)
	require.NoError(t, err)
	assert.NotNil(t, tenants)
}

func TestTenant_UsesMainDatabase(t *testing.T) {
	t.Run("returns true when db_name is nil", func(t *testing.T) {
		tenant := &Tenant{DBName: nil}
		assert.True(t, tenant.UsesMainDatabase())
	})

	t.Run("returns true when db_name is empty", func(t *testing.T) {
		empty := ""
		tenant := &Tenant{DBName: &empty}
		assert.True(t, tenant.UsesMainDatabase())
	})

	t.Run("returns false when db_name is set", func(t *testing.T) {
		dbName := "tenant_test"
		tenant := &Tenant{DBName: &dbName}
		assert.False(t, tenant.UsesMainDatabase())
	})
}
