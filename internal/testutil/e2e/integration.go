//go:build integration && !no_e2e

// Package e2e provides a shared integration-test context that embeds the E2E
// test infrastructure from package test (github.com/nimbleflux/fluxbase/test).
//
// It lives in a subpackage of testutil so that internal packages can import
// testutil (for mocks, which are cycle-free) without pulling in package test,
// which would create an import cycle (test -> internal/auth -> testutil).
package e2e

import (
	"fmt"
	"strings"
	"sync"
	"testing"

	"github.com/google/uuid"

	test "github.com/nimbleflux/fluxbase/test"
)

// Package-level singleton for shared test context.
// All tests in the same package share a single database connection to avoid pool exhaustion.
var (
	sharedContext            *IntegrationTestContext
	sharedContextOnce        sync.Once
	sharedContextMu          sync.RWMutex
	sharedContextInitialized bool // Track if context has been initialized
)

// IntegrationTestContext provides test context for integration tests.
// This embeds test.TestContext to provide all helper methods (NewRequest, CreateTestUser, etc.)
// while allowing internal packages to use E2E test infrastructure.
//
// Integration tests use real PostgreSQL database connections and test actual behavior
// instead of mocked behavior. They should be run with the `integration` build tag.
//
// IMPORTANT: All tests in the same package share a single database connection via singleton pattern.
// Tests should NOT call Close() on the returned context - the connection is managed at package level.
type IntegrationTestContext struct {
	*test.TestContext // Embedded to provide all helper methods (NewRequest, CreateTestUser, etc.)
	T                 *testing.T
	Namespace         string // Unique namespace for test data isolation between packages
}

// NewIntegrationTestContext creates or returns a shared test context for integration tests.
//
// All tests in the same package share a single database connection to avoid pool exhaustion.
// The first call creates the connection, subsequent calls reuse it.
//
// This uses fluxbase_app database user WITH BYPASSRLS privilege (RLS policies are NOT enforced).
//
// Use this for:
//   - General integration testing
//   - Testing business logic with real database
//   - API endpoint testing
//
// Do NOT use for:
//   - Testing RLS policies (use NewRLSIntegrationTestContext instead)
//
// IMPORTANT: Tests should NOT call Close() on the returned context.
// The shared connection is automatically managed by the package-level singleton.
func NewIntegrationTestContext(t *testing.T) *IntegrationTestContext {
	return newIntegrationTestContext(t, "default")
}

// NewIntegrationTestContextWithNamespace creates a shared test context with a unique namespace.
// The namespace isolates test data between packages so cleanup only affects this package's data.
//
// Example: testutil.NewIntegrationTestContextWithNamespace(t, "auth")
//
// This generates emails like "itest-auth-abc123@auth.test.local",
// settings keys like "auth.itest.mykey", and secret names like "AUTH_ITEST_MYSECRET".
func NewIntegrationTestContextWithNamespace(t *testing.T, namespace string) *IntegrationTestContext {
	return newIntegrationTestContext(t, namespace)
}

// newIntegrationTestContext is the shared implementation for both constructors.
func newIntegrationTestContext(t *testing.T, namespace string) *IntegrationTestContext {
	sharedContextMu.Lock()
	defer sharedContextMu.Unlock()

	// Check if this is the first call (before initialization)
	wasInitialized := sharedContextInitialized

	// Initialize shared context once per package
	sharedContextOnce.Do(func() {
		sharedContext = &IntegrationTestContext{
			TestContext: test.NewTestContext(&testing.T{}),
			T:           &testing.T{},
			Namespace:   namespace,
		}
		sharedContextInitialized = true
	})

	// Reset global state at the beginning of each test (except the first)
	// This prevents rate limiting and other cross-test interference
	if wasInitialized {
		test.ResetGlobalTestState()
	}

	// Update T with current test for proper error reporting
	sharedContext.T = t
	sharedContext.TestContext.T = t

	return sharedContext
}

// NewRLSIntegrationTestContext creates or returns a shared RLS-enabled test context.
//
// All tests in the same package share a single database connection to avoid pool exhaustion.
// The first call creates the connection, subsequent calls reuse it.
//
// This uses fluxbase_rls_test database user WITHOUT BYPASSRLS privilege (RLS policies ARE enforced).
//
// Use this ONLY for:
//   - Testing RLS policies
//   - Verifying data isolation between users
//   - Testing security boundaries
//
// Do NOT use for:
//   - General API testing (use NewIntegrationTestContext instead)
//
// IMPORTANT: Tests should NOT call Close() on the returned context.
// The shared connection is automatically managed by the package-level singleton.
func NewRLSIntegrationTestContext(t *testing.T) *IntegrationTestContext {
	return newRLSIntegrationTestContext(t, "default")
}

// NewRLSIntegrationTestContextWithNamespace creates a shared RLS-enabled test context with a namespace.
func NewRLSIntegrationTestContextWithNamespace(t *testing.T, namespace string) *IntegrationTestContext {
	return newRLSIntegrationTestContext(t, namespace)
}

// newRLSIntegrationTestContext is the shared implementation for both RLS constructors.
func newRLSIntegrationTestContext(t *testing.T, namespace string) *IntegrationTestContext {
	sharedContextMu.Lock()
	defer sharedContextMu.Unlock()

	// Check if this is the first call (before initialization)
	wasInitialized := sharedContextInitialized

	// Initialize shared RLS context once per package
	sharedContextOnce.Do(func() {
		sharedContext = &IntegrationTestContext{
			TestContext: test.NewRLSTestContext(&testing.T{}),
			T:           &testing.T{},
			Namespace:   namespace,
		}
		sharedContextInitialized = true
	})

	// Reset global state at the beginning of each test (except the first)
	// This prevents rate limiting and other cross-test interference
	if wasInitialized {
		test.ResetGlobalTestState()
	}

	// Update T with current test for proper error reporting
	sharedContext.T = t
	sharedContext.TestContext.T = t

	return sharedContext
}

// Close is a no-op for shared contexts.
//
// The shared connection is managed at the package level by the singleton pattern.
// Individual tests should not call Close() - the connection persists for all tests in the package.
func (tc *IntegrationTestContext) Close() {
	// No-op for shared contexts
	// Connection cleanup happens automatically when the package's tests complete
}

// EmailDomain returns the email domain for this namespace.
// Example: "auth.test.local"
func (tc *IntegrationTestContext) EmailDomain() string {
	return tc.Namespace + ".test.local"
}

// EmailPrefix returns the email prefix for this namespace.
// Example: "itest-auth-"
func (tc *IntegrationTestContext) EmailPrefix() string {
	return "itest-" + tc.Namespace + "-"
}

// TestEmail generates a unique test email scoped to this namespace.
// Example: "itest-auth-abc12345@auth.test.local"
func (tc *IntegrationTestContext) TestEmail() string {
	return fmt.Sprintf("%s%s@%s", tc.EmailPrefix(), uuid.New().String()[:8], tc.EmailDomain())
}

// SettingsPrefix returns the settings key prefix for this namespace.
// Example: "auth.itest."
func (tc *IntegrationTestContext) SettingsPrefix() string {
	return tc.Namespace + ".itest."
}

// SecretPrefix returns the secret name prefix for this namespace (uppercase).
// Example: "AUTH_ITEST_"
func (tc *IntegrationTestContext) SecretPrefix() string {
	return strings.ToUpper(tc.Namespace) + "_ITEST_"
}

// CleanupTestData cleans up test data from the database.
// This should be called between tests to ensure test isolation.
//
// When a non-default namespace is set, cleanup runs BOTH namespace-scoped patterns
// AND legacy broad patterns. This ensures backward compatibility during migration
// while providing namespace isolation for packages that adopt namespaced data.
//
// It cleans up:
// - Auth/dashboard users with test email prefixes
// - Password reset tokens and magic links
// - Settings and secrets
func (tc *IntegrationTestContext) CleanupTestData() {
	// Always clean up legacy patterns (backward compatible)
	tc.ExecuteSQL(`DELETE FROM auth.users WHERE email LIKE 'e2e-test-%'`)
	tc.ExecuteSQL(`DELETE FROM auth.users WHERE email LIKE 'test-%@example.com'`)
	tc.ExecuteSQL(`DELETE FROM auth.users WHERE email LIKE 'test-%@test.com'`)
	tc.ExecuteSQL(`DELETE FROM auth.users WHERE email LIKE '%@test.local'`)

	tc.ExecuteSQL(`DELETE FROM platform.users WHERE email LIKE 'e2e-test-%'`)
	tc.ExecuteSQL(`DELETE FROM platform.users WHERE email LIKE 'test-%@example.com'`)
	tc.ExecuteSQL(`DELETE FROM platform.users WHERE email LIKE 'test-%@test.com'`)

	tc.ExecuteSQL(`DELETE FROM auth.password_reset_tokens WHERE user_id NOT IN (SELECT id FROM auth.users)`)

	tc.ExecuteSQL(`DELETE FROM auth.magic_links WHERE email LIKE 'test-%@example.com'`)
	tc.ExecuteSQL(`DELETE FROM auth.magic_links WHERE email LIKE 'test-%@test.com'`)

	tc.ExecuteSQL(`DELETE FROM app.settings WHERE key LIKE 'custom.%' OR key LIKE 'test.%' OR key LIKE 'secret.%'`)

	tc.ExecuteSQL(`DELETE FROM functions.secrets WHERE name LIKE ANY(ARRAY[
		'TEST_%', 'DUPLICATE_%', 'GLOBAL_%', 'NS_%',
		'API_KEY_%', 'EXPIRED_%', 'GET_%', 'TEMP_%',
		'SHARED_%', 'DB_PASSWORD_%', 'DUP_TEST_%',
		'GET_BY_%', 'WRONG_NS_%', 'USER_TRACKED_%',
		'UPDATE_%', 'DELETE_%', 'LIST_%', 'NO_%',
		'WRONG_%', 'SHARED_NAME_%', 'CREATE_%',
		'NEW_%', 'OLD_%', 'ANOTHER_%', 'SECOND_%'
	])`)

	// Also clean up namespace-scoped data when a non-default namespace is set
	if tc.Namespace != "" && tc.Namespace != "default" {
		emailPrefix := tc.EmailPrefix() + "%"
		tc.ExecuteSQL(fmt.Sprintf(`DELETE FROM auth.users WHERE email LIKE '%s'`, emailPrefix))
		tc.ExecuteSQL(fmt.Sprintf(`DELETE FROM platform.users WHERE email LIKE '%s'`, emailPrefix))
		tc.ExecuteSQL(fmt.Sprintf(`DELETE FROM auth.magic_links WHERE email LIKE '%s'`, emailPrefix))
		tc.ExecuteSQL(fmt.Sprintf(`DELETE FROM app.settings WHERE key LIKE '%s%%'`, tc.SettingsPrefix()))
		tc.ExecuteSQL(fmt.Sprintf(`DELETE FROM functions.secrets WHERE name LIKE '%s%%'`, tc.SecretPrefix()))
	}
}
