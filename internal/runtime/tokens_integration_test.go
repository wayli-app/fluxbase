//go:build integration

package runtime

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nimbleflux/fluxbase/internal/auth"
	"github.com/nimbleflux/fluxbase/internal/database"
	"github.com/nimbleflux/fluxbase/internal/middleware"
	"github.com/nimbleflux/fluxbase/test/dbhelpers"
)

const testJWTSecret = "integration-test-jwt-secret-minimum-32-chars!!"

func getDefaultTenantID(t *testing.T, pool interface {
	QueryRow(ctx context.Context, sql string, args ...interface{}) pgx.Row
}) string {
	t.Helper()
	var tenantID string
	err := pool.QueryRow(context.Background(),
		"SELECT id FROM platform.tenants WHERE is_default = true LIMIT 1",
	).Scan(&tenantID)
	require.NoError(t, err, "Failed to get default tenant ID")
	return tenantID
}

func parseServiceTokenClaims(t *testing.T, tokenString string) jwt.MapClaims {
	t.Helper()
	parsed, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		return []byte(testJWTSecret), nil
	})
	require.NoError(t, err, "Failed to parse service token")
	require.True(t, parsed.Valid)
	claims, ok := parsed.Claims.(jwt.MapClaims)
	require.True(t, ok, "Token claims should be MapClaims")
	return claims
}

func grantAllRoles(t *testing.T, ctx context.Context, pool *pgxpool.Pool, tableName string) {
	t.Helper()
	_, err := pool.Exec(ctx, fmt.Sprintf("GRANT ALL ON public.%s TO service_role, tenant_service, authenticated, anon", tableName))
	require.NoError(t, err, "Failed to grant roles on test table")
}

// Test 1: Service token resolves to tenant_service DB role with correct tenant context
func TestServiceToken_ResolvesToTenantServiceDBRole(t *testing.T) {
	testCtx := dbhelpers.NewDBTestContext(t)
	defer testCtx.Close()
	ctx := context.Background()

	tenantID := getDefaultTenantID(t, testCtx.Pool)

	req := ExecutionRequest{
		ID:       uuid.New(),
		Name:     "test-function",
		TenantID: tenantID,
	}
	token, err := generateServiceToken(testJWTSecret, req, RuntimeTypeFunction, 30*time.Second)
	require.NoError(t, err, "generateServiceToken should succeed with TenantID")
	require.NotEmpty(t, token)

	claims := parseServiceTokenClaims(t, token)
	assert.Equal(t, "tenant_service", claims["sub"])
	assert.Equal(t, "tenant_service", claims["role"])
	assert.Equal(t, tenantID, claims["tenant_id"])
	assert.Equal(t, req.ID.String(), claims["execution_id"])
	assert.Equal(t, "access", claims["token_type"])

	tx, err := testCtx.Pool.Begin(ctx)
	require.NoError(t, err)
	defer tx.Rollback(ctx)

	tokenClaims := &auth.TokenClaims{
		TenantID: &tenantID,
	}
	err = middleware.SetRLSContext(ctx, tx, "", "tenant_service", tokenClaims)
	require.NoError(t, err, "SetRLSContext with tenant_service should succeed")

	var currentUser string
	err = tx.QueryRow(ctx, "SELECT current_user").Scan(&currentUser)
	require.NoError(t, err)
	assert.Equal(t, "tenant_service", currentUser, "DB role should be tenant_service")

	var currentTenant string
	err = tx.QueryRow(ctx, "SELECT current_setting('app.current_tenant_id', true)").Scan(&currentTenant)
	require.NoError(t, err)
	assert.Equal(t, tenantID, currentTenant, "app.current_tenant_id should match token's tenant_id")

	var claimsJSON string
	err = tx.QueryRow(ctx, "SELECT current_setting('request.jwt.claims', true)").Scan(&claimsJSON)
	require.NoError(t, err)
	var parsed map[string]interface{}
	err = json.Unmarshal([]byte(claimsJSON), &parsed)
	require.NoError(t, err)
	assert.Equal(t, tenantID, parsed["tenant_id"])
	assert.Equal(t, "tenant_service", parsed["role"])
}

// Test 2: Service token rejected when TenantID is empty
func TestServiceToken_RejectsEmptyTenantID(t *testing.T) {
	t.Run("generateServiceToken returns error", func(t *testing.T) {
		req := ExecutionRequest{
			ID:   uuid.New(),
			Name: "test-function",
		}
		_, err := generateServiceToken(testJWTSecret, req, RuntimeTypeFunction, 30*time.Second)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "tenant context required")
	})

	t.Run("job token also rejected without TenantID", func(t *testing.T) {
		req := ExecutionRequest{
			ID:   uuid.New(),
			Name: "test-job",
		}
		_, err := generateServiceToken(testJWTSecret, req, RuntimeTypeJob, 5*time.Minute)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "tenant context required")
	})
}

// Test 3: Cross-tenant data isolation via RLS
func TestServiceToken_CrossTenantDataIsolation(t *testing.T) {
	testCtx := dbhelpers.NewDBTestContext(t)
	defer testCtx.Close()
	ctx := context.Background()

	tenantA := uuid.New().String()
	tenantB := uuid.New().String()
	tableName := fmt.Sprintf("_test_rls_isolation_%s", uuid.New().String()[:8])

	_, err := testCtx.Pool.Exec(ctx, fmt.Sprintf(`
		CREATE TABLE public.%s (
			id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
			tenant_id text NOT NULL,
			data text NOT NULL
		)
	`, tableName))
	require.NoError(t, err, "Failed to create test table")
	t.Cleanup(func() {
		_, _ = testCtx.Pool.Exec(context.Background(), fmt.Sprintf("DROP TABLE IF EXISTS public.%s", tableName))
	})

	_, err = testCtx.Pool.Exec(ctx, fmt.Sprintf("ALTER TABLE public.%s ENABLE ROW LEVEL SECURITY", tableName))
	require.NoError(t, err)

	_, err = testCtx.Pool.Exec(ctx, fmt.Sprintf("ALTER TABLE public.%s FORCE ROW LEVEL SECURITY", tableName))
	require.NoError(t, err)

	grantAllRoles(t, ctx, testCtx.Pool, tableName)

	_, err = testCtx.Pool.Exec(ctx, fmt.Sprintf(`
		CREATE POLICY "%s_tenant_isolation" ON public.%s
			FOR ALL TO PUBLIC
			USING (auth.has_tenant_access(tenant_id::uuid))
			WITH CHECK (auth.has_tenant_access(tenant_id::uuid))
	`, tableName, tableName))
	require.NoError(t, err)
	t.Cleanup(func() {
		_, _ = testCtx.Pool.Exec(context.Background(), fmt.Sprintf("DROP POLICY IF EXISTS \"%s_tenant_isolation\" ON public.%s", tableName, tableName))
	})

	err = database.WrapWithServiceRoleAndTenant(ctx, database.NewConnectionWithPool(testCtx.Pool), tenantA, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx, fmt.Sprintf("INSERT INTO public.%s (tenant_id, data) VALUES ($1, $2)", tableName), tenantA, "tenant-a-data")
		return err
	})
	require.NoError(t, err, "Failed to insert tenant A data")

	err = database.WrapWithServiceRoleAndTenant(ctx, database.NewConnectionWithPool(testCtx.Pool), tenantB, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx, fmt.Sprintf("INSERT INTO public.%s (tenant_id, data) VALUES ($1, $2)", tableName), tenantB, "tenant-b-data")
		return err
	})
	require.NoError(t, err, "Failed to insert tenant B data")

	t.Run("tenant_service for tenant A sees only tenant A rows", func(t *testing.T) {
		tx, err := testCtx.Pool.Begin(ctx)
		require.NoError(t, err)
		defer tx.Rollback(ctx)

		claims := &auth.TokenClaims{TenantID: &tenantA}
		err = middleware.SetRLSContext(ctx, tx, "", "tenant_service", claims)
		require.NoError(t, err)

		rows, err := tx.Query(ctx, fmt.Sprintf("SELECT data FROM public.%s ORDER BY data", tableName))
		require.NoError(t, err)
		defer rows.Close()

		var results []string
		for rows.Next() {
			var data string
			require.NoError(t, rows.Scan(&data))
			results = append(results, data)
		}
		require.NoError(t, rows.Err())

		assert.Equal(t, []string{"tenant-a-data"}, results, "Should only see tenant A's data")
	})

	t.Run("tenant_service for tenant B sees only tenant B rows", func(t *testing.T) {
		tx, err := testCtx.Pool.Begin(ctx)
		require.NoError(t, err)
		defer tx.Rollback(ctx)

		claims := &auth.TokenClaims{TenantID: &tenantB}
		err = middleware.SetRLSContext(ctx, tx, "", "tenant_service", claims)
		require.NoError(t, err)

		rows, err := tx.Query(ctx, fmt.Sprintf("SELECT data FROM public.%s ORDER BY data", tableName))
		require.NoError(t, err)
		defer rows.Close()

		var results []string
		for rows.Next() {
			var data string
			require.NoError(t, rows.Scan(&data))
			results = append(results, data)
		}
		require.NoError(t, rows.Err())

		assert.Equal(t, []string{"tenant-b-data"}, results, "Should only see tenant B's data")
	})
}

// Test 4: tenant_service cannot bypass RLS on FORCE ROW LEVEL SECURITY tables
func TestServiceToken_CannotBypassRLS(t *testing.T) {
	testCtx := dbhelpers.NewDBTestContext(t)
	defer testCtx.Close()
	ctx := context.Background()

	tenantA := uuid.New().String()
	tenantB := uuid.New().String()
	tableName := fmt.Sprintf("_test_rls_bypass_%s", uuid.New().String()[:8])

	_, err := testCtx.Pool.Exec(ctx, fmt.Sprintf(`
		CREATE TABLE public.%s (
			id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
			tenant_id text NOT NULL,
			secret text NOT NULL
		)
	`, tableName))
	require.NoError(t, err)
	t.Cleanup(func() {
		_, _ = testCtx.Pool.Exec(context.Background(), fmt.Sprintf("DROP TABLE IF EXISTS public.%s", tableName))
	})

	_, err = testCtx.Pool.Exec(ctx, fmt.Sprintf("ALTER TABLE public.%s ENABLE ROW LEVEL SECURITY", tableName))
	require.NoError(t, err)
	_, err = testCtx.Pool.Exec(ctx, fmt.Sprintf("ALTER TABLE public.%s FORCE ROW LEVEL SECURITY", tableName))
	require.NoError(t, err)

	grantAllRoles(t, ctx, testCtx.Pool, tableName)

	_, err = testCtx.Pool.Exec(ctx, fmt.Sprintf(`
		CREATE POLICY "%s_tenant_only" ON public.%s
			FOR ALL TO PUBLIC
			USING (auth.has_tenant_access(tenant_id::uuid))
	`, tableName, tableName))
	require.NoError(t, err)
	t.Cleanup(func() {
		_, _ = testCtx.Pool.Exec(context.Background(), fmt.Sprintf("DROP POLICY IF EXISTS \"%s_tenant_only\" ON public.%s", tableName, tableName))
	})

	err = database.WrapWithServiceRoleAndTenant(ctx, database.NewConnectionWithPool(testCtx.Pool), tenantB, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx, fmt.Sprintf("INSERT INTO public.%s (tenant_id, secret) VALUES ($1, $2)", tableName), tenantB, "sensitive-tenant-b-data")
		return err
	})
	require.NoError(t, err)

	t.Run("tenant_service for tenant A cannot read tenant B data even with explicit WHERE", func(t *testing.T) {
		tx, err := testCtx.Pool.Begin(ctx)
		require.NoError(t, err)
		defer tx.Rollback(ctx)

		claims := &auth.TokenClaims{TenantID: &tenantA}
		err = middleware.SetRLSContext(ctx, tx, "", "tenant_service", claims)
		require.NoError(t, err)

		rows, err := tx.Query(ctx, fmt.Sprintf("SELECT secret FROM public.%s WHERE tenant_id = $1", tableName), tenantB)
		require.NoError(t, err)
		defer rows.Close()

		var results []string
		for rows.Next() {
			var secret string
			require.NoError(t, rows.Scan(&secret))
			results = append(results, secret)
		}
		require.NoError(t, rows.Err())

		assert.Empty(t, results, "tenant_service for tenant A should NOT see tenant B's data")
	})

	t.Run("tenant_service for tenant A cannot insert data for tenant B", func(t *testing.T) {
		tx, err := testCtx.Pool.Begin(ctx)
		require.NoError(t, err)
		defer tx.Rollback(ctx)

		claims := &auth.TokenClaims{TenantID: &tenantA}
		err = middleware.SetRLSContext(ctx, tx, "", "tenant_service", claims)
		require.NoError(t, err)

		_, err = tx.Exec(ctx, fmt.Sprintf("INSERT INTO public.%s (tenant_id, secret) VALUES ($1, $2)", tableName), tenantB, "injected-data")
		assert.Error(t, err, "INSERT with mismatched tenant_id should be blocked by RLS WITH CHECK")
	})
}
