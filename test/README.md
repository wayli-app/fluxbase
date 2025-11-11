# Fluxbase Test Suite

This directory contains the comprehensive test suite for Fluxbase. This guide will help you understand the test structure, write new tests, and debug failing tests.

## Table of Contents

- [Test Categories](#test-categories)
- [Quick Start](#quick-start)
- [Test Contexts: Critical Differences](#test-contexts-critical-differences)
- [Database Setup](#database-setup)
- [Authentication Methods](#authentication-methods)
- [Writing Tests](#writing-tests)
- [Running Tests](#running-tests)
- [Prerequisites](#prerequisites)
- [Troubleshooting](#troubleshooting)

## Test Categories

### 1. Unit Tests

**Location**: `/test/unit/` and `/internal/*/`
**Speed**: Very fast (milliseconds)
**Dependencies**: None (pure functions, mocks)

Tests individual functions in isolation:

- Password hashing and validation
- JWT token generation/validation
- API key generation
- Filter parsing
- Query building

**Run with**: `make test` (includes `-short` flag)

### 2. Integration Tests

**Location**: `/internal/*/` (skipped in short mode)
**Speed**: Moderate (hundreds of milliseconds)
**Dependencies**: Specific services (MailHog, MinIO, PostgreSQL)

Tests component interactions:

- Email sending via SMTP with MailHog
- Storage operations with MinIO
- Database query execution
- Auth middleware with real tokens

**Run with**: `make test-full`

### 3. E2E Tests

**Location**: `/test/e2e/`
**Speed**: Slow (seconds per test)
**Dependencies**: ALL services (PostgreSQL, MailHog, MinIO, Fiber app)

Tests complete user workflows from HTTP request to database:

- Authentication flows (signup → signin → profile → reset)
- REST API CRUD operations
- Row-Level Security policies
- Storage operations
- Realtime WebSocket subscriptions
- OAuth flows
- Webhook delivery

**Run with**: `make test-e2e` or `make test-full`

### 4. Load Tests

**Location**: `/test/load/` (K6 scripts)
**Purpose**: Performance testing and capacity planning

See [load/README.md](load/README.md) for detailed documentation.

## Quick Start

### Your First Test

Here's a minimal test to get started:

```go
package e2e_test

import (
	"testing"
	test "fluxbase/test"
	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/require"
)

func TestHealthCheck(t *testing.T) {
	// GIVEN: A running Fluxbase instance
	tc := test.NewTestContext(t)
	defer tc.Close()

	// WHEN: Requesting the health endpoint
	resp := tc.NewRequest("GET", "/health").
		Send()

	// THEN: Server responds with 200 OK
	resp.AssertStatus(fiber.StatusOK)

	var result map[string]interface{}
	resp.JSON(&result)
	require.Equal(t, "ok", result["status"])
}
```

### Test Structure Pattern

All tests should follow this pattern:

```go
func TestFeatureName(t *testing.T) {
	// GIVEN: Setup - describe the initial state
	tc := setupFeatureTest(t)
	defer tc.Close()

	// WHEN: Action - describe what you're testing
	resp := tc.NewRequest("POST", "/api/v1/endpoint").
		WithAuth(token).
		WithBody(data).
		Send()

	// THEN: Assertion - verify the outcome
	resp.AssertStatus(fiber.StatusCreated)

	// AND: Verify database state (for mutations)
	rows := tc.QuerySQL("SELECT * FROM table WHERE id = $1", id)
	require.Len(t, rows, 1)
	require.Equal(t, expectedValue, rows[0]["field"])
}
```

## Test Contexts: Critical Differences

⚠️ **CRITICAL**: Understanding these two contexts is essential for writing correct tests.

### NewTestContext() - Default Context

```go
tc := test.NewTestContext(t)
```

**Database User**: `fluxbase_app`
**Privilege**: Has **BYPASSRLS** (Row-Level Security is NOT enforced)

**Use for**:

- ✅ General REST API testing
- ✅ Authentication flows
- ✅ Storage operations
- ✅ Any test where RLS should be bypassed

**Example**: Testing CRUD operations where you want to verify the API works correctly regardless of RLS policies.

### NewRLSTestContext() - RLS Testing Context

```go
tc := test.NewRLSTestContext(t)
```

**Database User**: `fluxbase_rls_test`
**Privilege**: Does **NOT** have BYPASSRLS (Row-Level Security IS enforced)

**Use for**:

- ✅ Testing RLS policies
- ✅ Verifying data isolation between users
- ✅ Testing security boundaries

**Example**: Verifying that User A cannot access User B's private data.

### ⚠️ Common Mistake

```go
// ❌ WRONG: Using NewTestContext for RLS tests
func TestRLSUserIsolation(t *testing.T) {
	tc := test.NewTestContext(t)  // This has BYPASSRLS!
	// RLS policies will NOT be enforced - test will pass incorrectly
}

// ✅ CORRECT: Using NewRLSTestContext for RLS tests
func TestRLSUserIsolation(t *testing.T) {
	tc := test.NewRLSTestContext(t)  // No BYPASSRLS
	// RLS policies WILL be enforced - test works correctly
}
```

## Database Setup

### Database Users

Three database users exist for different purposes:

1. **`postgres`** (superuser)

   - Used ONLY for granting permissions
   - Never used directly in tests

2. **`fluxbase_app`** (default test user)

   - Has `BYPASSRLS` privilege
   - Used by `NewTestContext()`
   - Used for migrations and general testing
   - Can freely manage data without RLS restrictions

3. **`fluxbase_rls_test`** (RLS test user)
   - Does NOT have `BYPASSRLS` privilege
   - Used by `NewRLSTestContext()`
   - Used ONLY for testing RLS policies
   - Subject to all RLS restrictions

### Test Tables

Two test tables are created by `TestMain` before tests run:

#### 1. `products` Table

**Schema**: `id`, `name`, `price`, `created_at`, `updated_at`
**RLS**: Disabled
**Purpose**: General REST API testing

```sql
CREATE TABLE products (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL,
    price NUMERIC(10, 2),
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);
```

#### 2. `tasks` Table

**Schema**: `id`, `user_id`, `title`, `description`, `completed`, `is_public`, `created_at`, `updated_at`
**RLS**: Enabled and enforced
**Purpose**: RLS policy testing

**RLS Policies**:

- `tasks_select_own`: Users can select their own tasks
- `tasks_select_public`: Anyone can select public tasks
- `tasks_insert_own`: Authenticated users can insert their own tasks
- `tasks_update_own`: Users can update their own tasks
- `tasks_delete_own`: Users can delete their own tasks
- Admin policies: Can select/update/delete all tasks

See [e2e/setup_test.go](e2e/setup_test.go) for full schema definitions.

### Migrations

Migrations are handled automatically:

- **CI**: Migrations run once by `postgres` user before tests
- **Local**: Migrations run by `fluxbase_app` user in `NewTestContext()`
- Tests skip migrations if already applied (via `ErrNoChange`)

## Authentication Methods

The fluent API supports multiple authentication methods:

### WithAuth(token) / WithBearerToken(token)

```go
resp := tc.NewRequest("GET", "/api/v1/auth/user").
    WithAuth(userJWT).  // Alias for WithBearerToken
    Send()
```

**Sets**: `Authorization: Bearer {token}`
**Use for**: User JWT authentication
**RLS**: Respects RLS policies for the authenticated user

### WithAPIKey(apiKey)

```go
resp := tc.NewRequest("GET", "/api/v1/tables/products").
    WithAPIKey(tc.APIKey).
    Send()
```

**Sets**: `X-API-Key: {apiKey}`
**Use for**: Project API key authentication
**RLS**: Respects RLS policies

### WithServiceKey(serviceKey)

```go
resp := tc.NewRequest("POST", "/api/v1/admin/users").
    WithServiceKey(tc.ServiceKey).
    Send()
```

**Sets**: `X-Service-Key: {serviceKey}`
**Use for**: Service role authentication (admin operations)
**⚠️ WARNING**: Service keys **BYPASS RLS POLICIES**. Use only for admin operations.

### Unauthenticated()

```go
resp := tc.NewRequest("GET", "/api/v1/public/data").
    Unauthenticated().
    Send()
```

**Use for**: Testing public endpoints or error handling for missing auth

### Authentication Helpers

Create users and get tokens:

```go
// Create test user and get JWT
userID, token := tc.CreateTestUser("user@example.com", "password123")

// Create dashboard admin
adminID, adminToken := tc.CreateDashboardAdminUser("admin@example.com", "password123")

// Create API key
apiKey := tc.CreateAPIKey("My API Key", []string{"read", "write"})

// Create service key (bypasses RLS!)
serviceKey := tc.CreateServiceKey("My Service Key")

// Generate anonymous key
anonKey := tc.GenerateAnonKey()
```

## Writing Tests

### Test Setup Pattern

Every test file should have a setup function:

```go
// setupFeatureTest creates a clean test context for feature testing.
// This ensures test isolation by truncating relevant tables.
func setupFeatureTest(t *testing.T) *test.TestContext {
	tc := test.NewTestContext(t)
	tc.EnsureAuthSchema()  // If auth needed
	tc.ExecuteSQL("TRUNCATE TABLE my_table CASCADE")

	// Any feature-specific configuration
	tc.Config.Feature.Enabled = true

	return tc
}
```

Use it in every test:

```go
func TestFeature(t *testing.T) {
	tc := setupFeatureTest(t)
	defer tc.Close()

	// Test logic
}
```

### Given-When-Then Structure

Structure tests with clear comments:

```go
func TestRESTCreateAndRetrieve(t *testing.T) {
	// GIVEN: A clean database and authenticated API client
	tc := setupRESTTest(t)
	defer tc.Close()

	// WHEN: Creating a new product via POST request
	resp := tc.NewRequest("POST", "/api/v1/tables/products").
		WithAPIKey(tc.APIKey).
		WithBody(map[string]interface{}{
			"name":  "Test Product",
			"price": 29.99,
		}).
		Send()

	// THEN: Product is created successfully
	resp.AssertStatus(fiber.StatusCreated)
	var result map[string]interface{}
	resp.JSON(&result)
	require.NotNil(t, result["id"])
	productID := result["id"].(string)

	// AND: Product exists in database with correct values
	rows := tc.QuerySQL("SELECT * FROM products WHERE id = $1", productID)
	require.Len(t, rows, 1)
	require.Equal(t, "Test Product", rows[0]["name"])
	require.Equal(t, 29.99, rows[0]["price"])
}
```

### Verify Database State

**Always verify database state after mutations:**

```go
// After CREATE
resp := tc.NewRequest("POST", "/api/v1/tables/products").
	WithBody(product).Send().AssertStatus(fiber.StatusCreated)

var result map[string]interface{}
resp.JSON(&result)
productID := result["id"].(string)

// ✅ Verify in database
rows := tc.QuerySQL("SELECT * FROM products WHERE id = $1", productID)
require.Len(t, rows, 1)
require.Equal(t, expectedValue, rows[0]["field"])

// After UPDATE
resp := tc.NewRequest("PUT", "/api/v1/tables/products/"+productID).
	WithBody(updates).Send().AssertStatus(fiber.StatusOK)

// ✅ Verify changes persisted
rows := tc.QuerySQL("SELECT * FROM products WHERE id = $1", productID)
require.Equal(t, updatedValue, rows[0]["field"])

// After DELETE
resp := tc.NewRequest("DELETE", "/api/v1/tables/products/"+productID).
	Send().AssertStatus(fiber.StatusNoContent)

// ✅ Verify record removed
rows := tc.QuerySQL("SELECT * FROM products WHERE id = $1", productID)
require.Len(t, rows, 0, "Product should be deleted")
```

### Specific Status Assertions

**Always use specific status codes:**

```go
// ❌ BAD: Too permissive
require.True(t, resp.Status() >= 400)

// ✅ GOOD: Specific status
require.Equal(t, fiber.StatusBadRequest, resp.Status())

// ✅ GOOD: Multiple acceptable statuses (when truly appropriate)
require.Contains(t, []int{fiber.StatusBadRequest, fiber.StatusUnprocessableEntity},
	resp.Status())
```

### Testing Error Responses

**Test both success AND failure paths.** The API properly returns specific HTTP status codes for different error types:

- **409 Conflict**: Duplicate key violations, foreign key constraints
- **400 Bad Request**: Invalid data types, check constraint violations, malformed requests
- **404 Not Found**: Resource doesn't exist
- **401 Unauthorized**: Missing or invalid authentication
- **500 Internal Server Error**: Unexpected server errors

```go
// Test duplicate key violation
func TestDuplicateKeyError(t *testing.T) {
	tc := setupTest(t)
	defer tc.Close()

	apiKey := tc.CreateAPIKey("Test", nil)

	// Add unique constraint
	tc.ExecuteSQL("ALTER TABLE products ADD CONSTRAINT products_name_key UNIQUE (name)")
	defer tc.ExecuteSQL("ALTER TABLE products DROP CONSTRAINT IF EXISTS products_name_key")

	// GIVEN: An existing product
	tc.NewRequest("POST", "/api/v1/tables/products").
		WithAPIKey(apiKey).
		WithBody(map[string]interface{}{"name": "Unique Product", "price": 10.00}).
		Send().
		AssertStatus(fiber.StatusCreated)

	// WHEN: Attempting to insert duplicate
	resp := tc.NewRequest("POST", "/api/v1/tables/products").
		WithAPIKey(apiKey).
		WithBody(map[string]interface{}{"name": "Unique Product", "price": 15.00}).
		Send()

	// THEN: Returns 409 Conflict (NOT 500)
	resp.AssertStatus(fiber.StatusConflict)

	var errResp map[string]interface{}
	resp.JSON(&errResp)
	require.NotNil(t, errResp["error"], "Error message should be present")
}

// Test invalid data type
func TestInvalidDataType(t *testing.T) {
	tc := setupTest(t)
	defer tc.Close()

	apiKey := tc.CreateAPIKey("Test", nil)

	// WHEN: Sending invalid data type for numeric field
	resp := tc.NewRequest("POST", "/api/v1/tables/products").
		WithAPIKey(apiKey).
		WithBody(map[string]interface{}{
			"name":  "Test Product",
			"price": "not-a-number",  // Invalid: string instead of number
		}).
		Send()

	// THEN: Returns 400 Bad Request (NOT 500)
	resp.AssertStatus(fiber.StatusBadRequest)

	var errResp map[string]interface{}
	resp.JSON(&errResp)
	require.NotNil(t, errResp["error"])
}
```

**❌ BAD: Permissive error checks**

```go
// Too permissive - accepts any error status
require.True(t, resp.Status() >= 400, "Should return error")
```

**✅ GOOD: Specific error status**

```go
// Specific - tests exact error behavior
resp.AssertStatus(fiber.StatusConflict)
// or
resp.AssertStatus(fiber.StatusBadRequest)
```

### Testing Email Delivery

Use `WaitForEmail()` for reliable email testing:

```go
// ❌ BAD: Optional assertion
resetEmail := tc.GetMailHogEmails()
if resetEmail != nil {
	require.Contains(t, resetEmail.Content.Body, "reset")
}

// ✅ GOOD: Required assertion with timeout
resetEmail := tc.WaitForEmail(5*time.Second, func(msg test.MailHogMessage) bool {
	return strings.Contains(msg.To[0].Mailbox, "reset")
})
require.NotNil(t, resetEmail, "Password reset email must be sent")
require.Contains(t, resetEmail.Content.Body, "reset")
```

### Avoid Hard-Coded Sleeps

Use `WaitForCondition()` instead:

```go
// ❌ BAD: Hard-coded sleep
time.Sleep(2 * time.Second)

// ✅ GOOD: Poll with timeout
success := tc.WaitForCondition(5*time.Second, 100*time.Millisecond, func() bool {
	results := tc.QuerySQL("SELECT COUNT(*) FROM events WHERE id = $1", eventID)
	return results[0]["count"].(int64) > 0
})
require.True(t, success, "Event should be created within 5 seconds")
```

## Running Tests

### Via Make Targets

```bash
# Run unit tests only (fast, ~2min)
make test

# Run all tests including e2e (slow, ~15min)
make test-full

# Run e2e tests only
make test-e2e

# Run specific test category
make test-auth       # Authentication tests
make test-rls        # RLS security tests
make test-rest       # REST API tests
make test-storage    # Storage tests
```

### Via Go Commands

```bash
# Run all e2e tests
go test -v ./test/e2e/...

# Run specific test suite
go test -v ./test/e2e/ -run TestAuth
go test -v ./test/e2e/ -run TestRLS

# Run specific test
go test -v ./test/e2e/ -run TestRESTCreateRecord

# Run with race detector
go test -v -race ./test/e2e/...

# Run unit tests only (skip slow tests)
go test -short ./...
```

### In CI/CD

Tests run automatically in GitHub Actions:

1. **Lint**: Go + TypeScript linting
2. **SDK Tests**: TypeScript SDK + React SDK
3. **Go Tests**: Unit tests (`-short -race`)
4. **E2E Tests**: Full e2e suite (`-race -parallel=1`)
5. **Coverage**: Upload to Codecov (target: 80% project, 70% patch)

## Prerequisites

### Required Services

All e2e tests require these services to be running:

#### 1. PostgreSQL 17

```bash
docker run -d --name postgres \
  -e POSTGRES_PASSWORD=postgres \
  -e POSTGRES_DB=fluxbase \
  -p 5432:5432 \
  postgres:18
```

**Required users**:

- `fluxbase_app` (with BYPASSRLS)
- `fluxbase_rls_test` (without BYPASSRLS)

#### 2. MailHog (Email Testing)

```bash
docker run -d --name mailhog \
  -p 1025:1025 \
  -p 8025:8025 \
  mailhog/mailhog
```

**Access**: Web UI at http://localhost:8025

#### 3. MinIO (S3-Compatible Storage)

```bash
docker run -d --name minio \
  -e MINIO_ROOT_USER=minioadmin \
  -e MINIO_ROOT_PASSWORD=minioadmin \
  -p 9000:9000 \
  -p 9001:9001 \
  minio/minio server /data --console-address ":9001"
```

**Access**: Console at http://localhost:9001

### Environment Variables

Tests use these environment variables:

```bash
# Database
DATABASE_HOST=localhost
DATABASE_PORT=5432
DATABASE_USER=fluxbase_app
DATABASE_PASSWORD=password
DATABASE_NAME=fluxbase_test

# Email (MailHog)
SMTP_HOST=localhost
SMTP_PORT=1025
MAILHOG_API_URL=http://localhost:8025

# Storage (MinIO)
STORAGE_PROVIDER=s3
S3_ENDPOINT=http://localhost:9000
S3_ACCESS_KEY=minioadmin
S3_SECRET_KEY=minioadmin
S3_BUCKET=fluxbase-test
```

### Docker Compose

Use the provided docker-compose file to start all services:

```bash
docker-compose up -d postgres mailhog minio
```

## Troubleshooting

### Tests Pass Locally But Fail in CI

**Cause**: Different database user configuration or missing services

**Solution**:

- Verify database users exist with correct privileges
- Check environment variables match CI configuration
- Ensure services are healthy before running tests

### RLS Tests Pass When They Shouldn't

**Cause**: Using `NewTestContext()` instead of `NewRLSTestContext()`

**Solution**: Use `NewRLSTestContext()` for all RLS tests to ensure policies are enforced.

### Email Tests Fail

**Cause**: MailHog not running or not accessible

**Solution**:

- Verify MailHog is running: `curl http://localhost:8025/api/v2/messages`
- Check MailHog logs: `docker logs mailhog`
- Use `WaitForEmail()` with adequate timeout (5 seconds)

### Database Connection Exhaustion

**Cause**: Too many parallel tests opening database connections

**Solution**:

- Run e2e tests with `-parallel=1`
- Increase PostgreSQL `max_connections`
- Ensure `tc.Close()` is called with `defer`

### Flaky Tests

**Common causes**:

- Hard-coded `time.Sleep()` - replace with `WaitForCondition()`
- Race conditions - run with `-race` flag to detect
- Shared state - ensure tables are truncated in setup
- External service delays - use appropriate timeouts

### Migration Errors

**Cause**: Migrations already applied or permission issues

**Solution**:

- Tests automatically skip already-applied migrations
- Reset database: `make db-reset`
- Verify user has correct permissions

## Test Helper Reference

See [e2e_helpers.go](e2e_helpers.go) for complete documentation of all helper methods.

### Common Helpers

**Context Creation**:

- `NewTestContext(t)` - Standard context (with BYPASSRLS)
- `NewRLSTestContext(t)` - RLS testing context (no BYPASSRLS)

**User Management**:

- `CreateTestUser(email, password)` - Create user, returns (userID, JWT)
- `CreateDashboardAdminUser(email, password)` - Create admin user
- `CreateAPIKey(name, scopes)` - Create API key
- `CreateServiceKey(name)` - Create service key (bypasses RLS!)

**Database Operations**:

- `ExecuteSQL(sql, args...)` - Execute as fluxbase_app
- `ExecuteSQLAsSuperuser(sql, args...)` - Execute as postgres
- `QuerySQL(sql, args...)` - Query as fluxbase_app
- `QuerySQLAsSuperuser(sql, args...)` - Query as postgres

**Email Testing**:

- `GetMailHogEmails()` - Get all emails
- `ClearMailHogEmails()` - Clear all emails
- `WaitForEmail(timeout, condition)` - Wait for specific email

**Utilities**:

- `WaitForCondition(timeout, interval, condition)` - Poll until condition met
- `CleanupStorageFiles()` - Clean storage bucket
- `EnsureAuthSchema()` - Ensure auth tables exist
- `EnsureStorageSchema()` - Ensure storage tables exist

## Contributing

When adding new tests:

1. ✅ Use the standard setup pattern (`setupFeatureTest()`)
2. ✅ Add Given-When-Then comments
3. ✅ Verify database state after mutations
4. ✅ Use specific status code assertions
5. ✅ Use `WaitForCondition()` instead of `time.Sleep()`
6. ✅ Choose correct context (`NewTestContext` vs `NewRLSTestContext`)
7. ✅ Clean up with `defer tc.Close()`
8. ✅ Add test to appropriate file or create new file if needed
9. ✅ Run locally with race detector: `go test -race ./test/e2e/...`
10. ✅ Ensure coverage remains ≥ 80%

## Resources

- [E2E Test Details](e2e/README.md) - In-depth e2e testing guide
- [Load Testing Guide](load/README.md) - Performance testing documentation
- [Helper API Documentation](e2e_helpers.go) - Complete helper reference
- [CI Configuration](../.github/workflows/ci.yml) - GitHub Actions setup
