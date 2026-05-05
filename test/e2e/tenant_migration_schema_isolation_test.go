//go:build integration

package e2e

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nimbleflux/fluxbase/test"
)

func TestTenantMigrations_SchemaIsolation(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping e2e test")
	}

	t.Log("Schema isolation is inherently enforced by the migration executor")
	t.Log("- Main-DB tenants: RLS on public schema prevents cross-tenant visibility")
	t.Log("- Separate-DB tenants: completely isolated databases")

	tc := test.NewRLSTestContext(t)
	tc.EnsureAuthSchema()

	// Verify that the RLS tenant isolation already tested in tenant_isolation_test.go
	// extends to DDL operations: a table created by one tenant is not visible to another

	// Create a test table for tenant isolation verification
	tc.ExecuteSQLAsSuperuser(`
		CREATE TABLE IF NOT EXISTS public.schema_isolation_test (
			id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
			data text,
			tenant_id uuid
		)
	`)

	// Enable RLS on the table
	tc.ExecuteSQLAsSuperuser(
		`ALTER TABLE public.schema_isolation_test ENABLE ROW LEVEL SECURITY`,
	)
	tc.ExecuteSQLAsSuperuser(
		`ALTER TABLE public.schema_isolation_test FORCE ROW LEVEL SECURITY`,
	)

	// Create tenant isolation policy
	tc.ExecuteSQLAsSuperuser(`
		DO $$ BEGIN
			IF NOT EXISTS (
				SELECT 1 FROM pg_policy WHERE polname = 'schema_isolation_tenant'
				AND polrelid = 'public.schema_isolation_test'::regclass
			) THEN
				EXECUTE 'CREATE POLICY schema_isolation_tenant ON public.schema_isolation_test
					USING (auth.has_tenant_access(tenant_id))
					WITH CHECK (auth.has_tenant_access(tenant_id))';
			END IF;
		END $$
	`)

	// Ensure tenant records exist in platform.tenants (needed for FK constraints)
	tc.ExecuteSQLAsSuperuser(`
		INSERT INTO platform.tenants (id, slug, name, status)
		VALUES ($1, 'test-tenant-1', 'Test Tenant 1', 'active')
		ON CONFLICT (id) DO NOTHING
	`, tenantTestID1)
	tc.ExecuteSQLAsSuperuser(`
		INSERT INTO platform.tenants (id, slug, name, status)
		VALUES ($1, 'test-tenant-2', 'Test Tenant 2', 'active')
		ON CONFLICT (id) DO NOTHING
	`, tenantTestID2)

	// Insert data for both tenants
	tc.ExecuteSQLAsSuperuser(`
		INSERT INTO public.schema_isolation_test (data, tenant_id) VALUES ($1, $2::uuid)
	`, "tenant1-data", tenantTestID1)
	tc.ExecuteSQLAsSuperuser(`
		INSERT INTO public.schema_isolation_test (data, tenant_id) VALUES ($1, $2::uuid)
	`, "tenant2-data", tenantTestID2)

	// Query as tenant1 - should only see own data
	rows1 := tc.QuerySQLAsTenant(tenantTestID1,
		"SELECT data FROM public.schema_isolation_test")
	assert.Len(t, rows1, 1)
	require.NotNil(t, rows1[0]["data"])
	assert.Equal(t, "tenant1-data", rows1[0]["data"].(string))

	// Query as tenant2 - should only see own data
	rows2 := tc.QuerySQLAsTenant(tenantTestID2,
		"SELECT data FROM public.schema_isolation_test")
	assert.Len(t, rows2, 1)
	require.NotNil(t, rows2[0]["data"])
	assert.Equal(t, "tenant2-data", rows2[0]["data"].(string))

	// Cleanup
	tc.ExecuteSQLAsSuperuser("DROP TABLE IF EXISTS public.schema_isolation_test")
}
