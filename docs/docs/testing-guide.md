# Fluxbase Testing Guide

## Overview

Fluxbase has a comprehensive testing infrastructure that includes unit tests, integration tests, end-to-end tests, and load tests. This guide explains how to use the testing system.

## Test Types

### 1. Unit Tests

Unit tests focus on individual components in isolation.

```bash
# Run all unit tests
make test-unit

# Run unit tests with coverage
make test-coverage
```

### 2. Integration Tests

Integration tests verify that components work together correctly with the database.

```bash
# Run integration tests (requires database)
make test-integration
```

### 3. End-to-End (E2E) Tests

E2E tests verify the entire application stack, including HTTP endpoints.

```bash
# Run E2E tests with database setup
make test-e2e

# Run E2E tests without database setup (faster for iterations)
make test-e2e-quick
```

### 4. Load Tests

Load tests measure performance under stress using k6.

```bash
# Run load tests
make test-load
```

### 5. Run All Tests

```bash
# Run all test suites
make test

# Run all tests quickly (no database setup)
make test-quick
```

## Test Database Setup

The test database is automatically set up when you run `make test-e2e`. You can also set it up manually:

```bash
./test/scripts/setup_test_db.sh
```

This script:

- Creates a `fluxbase_test` database
- Sets up required extensions (uuid-ossp, pgcrypto, pg_trgm)
- Creates test schemas (auth, storage, realtime, functions)
- Creates test tables (items, products, categories, users, sessions, etc.)
- Inserts seed data for testing

## Test Infrastructure

### Test Context Helper

The `TestContext` provides a convenient way to set up and tear down test environments:

```go
package mypackage_test

import (
    "testing"
    "github.com/wayli-app/fluxbase/test"
)

func TestMyFeature(t *testing.T) {
    tc := test.NewTestContext(t)
    defer tc.Close()

    // Your test code here
}
```

### API Request Helper

Making HTTP requests in tests is easy with the request builder:

```go
// Make a GET request
resp := tc.NewRequest("GET", "/api/rest/items").Send()
resp.AssertStatus(200)

// Make a POST request with body
resp := tc.NewRequest("POST", "/api/rest/items").
    WithBody(map[string]interface{}{
        "name": "Test Item",
        "quantity": 10,
    }).
    Send()

resp.AssertStatus(201)

// Parse JSON response
var item map[string]interface{}
resp.JSON(&item)
assert.Equal(t, "Test Item", item["name"])
```

### Test Data Builder

Easily insert test data:

```go
tc.NewTestData("items").
    Row(map[string]interface{}{
        "name": "Item 1",
        "quantity": 10,
    }).
    Row(map[string]interface{}{
        "name": "Item 2",
        "quantity": 20,
    }).
    Insert()
```

### Database Utilities

Execute raw SQL or query data:

```go
// Execute SQL
tc.ExecuteSQL("INSERT INTO items (name, quantity) VALUES ($1, $2)", "Test", 5)

// Query data
results := tc.QuerySQL("SELECT * FROM items WHERE name = $1", "Test")
```

## Writing E2E Tests

Here's a complete example of an E2E test:

```go
package test

import (
    "testing"
    "github.com/gofiber/fiber/v2"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/suite"
)

type MyTestSuite struct {
    suite.Suite
    tc *TestContext
}

func (s *MyTestSuite) SetupSuite() {
    s.tc = NewTestContext(s.T())
}

func (s *MyTestSuite) TearDownSuite() {
    s.tc.Close()
}

func (s *MyTestSuite) SetupTest() {
    // Clean data before each test
    s.tc.ExecuteSQL("TRUNCATE TABLE items CASCADE")
}

func (s *MyTestSuite) TestCreateItem() {
    resp := s.tc.NewRequest("POST", "/api/rest/items").
        WithBody(map[string]interface{}{
            "name": "Test Item",
            "quantity": 10,
        }).
        Send()

    resp.AssertStatus(fiber.StatusCreated)

    var item map[string]interface{}
    resp.JSON(&item)
    assert.Equal(s.T(), "Test Item", item["name"])
}

func TestMySuite(t *testing.T) {
    if testing.Short() {
        t.Skip("Skipping E2E tests in short mode")
    }
    suite.Run(t, new(MyTestSuite))
}
```

## Test Configuration

Tests use a separate test configuration that connects to `fluxbase_test` database:

```go
cfg := test.GetTestConfig()
// Returns a config with:
// - Database: fluxbase_test
// - Host: postgres (in devcontainer)
// - Debug: true
// - JWT secret: test-secret-key-for-testing-only
```

## Running Tests in CI/CD

Tests are automatically run in GitHub Actions on every push and pull request. See [.github/workflows/](.github/workflows/) for CI configuration.

## Test Coverage

Generate a test coverage report:

```bash
make test-coverage
```

This creates a `coverage.html` file that you can open in your browser.

## Best Practices

### 1. Use Test Suites

Use `testify/suite` for better test organization and setup/teardown:

```go
type MyTestSuite struct {
    suite.Suite
    tc *TestContext
}
```

### 2. Clean Data Between Tests

Always clean test data in `SetupTest` to ensure test isolation:

```go
func (s *MyTestSuite) SetupTest() {
    s.tc.ExecuteSQL("TRUNCATE TABLE items CASCADE")
}
```

### 3. Use Descriptive Test Names

Test names should clearly describe what they test:

```go
func (s *MyTestSuite) TestCreateItemWithValidData() {}
func (s *MyTestSuite) TestCreateItemWithInvalidDataReturns400() {}
```

### 4. Test Error Cases

Don't just test the happy path:

```go
func (s *MyTestSuite) TestInvalidRequests() {
    // Test invalid table
    resp := s.tc.NewRequest("GET", "/api/rest/nonexistent_table").Send()
    assert.NotEqual(s.T(), fiber.StatusOK, resp.Status())

    // Test invalid JSON
    resp = s.tc.NewRequest("POST", "/api/rest/items").
        WithHeader("Content-Type", "application/json").
        WithBody("invalid json").
        Send()
    assert.NotEqual(s.T(), fiber.StatusCreated, resp.Status())
}
```

### 5. Use Parallel Tests When Possible

For independent unit tests, run them in parallel:

```go
func TestMyFeature(t *testing.T) {
    t.Parallel()
    // Test code
}
```

### 6. Skip Slow Tests in Short Mode

Allow fast test runs with `-short` flag:

```go
func TestSlowOperation(t *testing.T) {
    if testing.Short() {
        t.Skip("Skipping slow test in short mode")
    }
    // Test code
}
```

## Debugging Tests

### Run a Single Test

```bash
go test -v -run TestSpecificTest ./test/...
```

### Run with Race Detector

```bash
go test -race ./test/...
```

### Verbose Output

```bash
go test -v ./test/...
```

### See Database Queries

Set `Debug: true` in the test config to see all SQL queries in the logs.

## Troubleshooting

### Database Connection Issues

If tests fail with database connection errors:

```bash
# Check if PostgreSQL is running
pg_isready -h postgres -U postgres

# Restart the database
docker-compose -f .devcontainer/docker-compose.yml restart postgres

# Re-setup test database
./test/scripts/setup_test_db.sh
```

### Port Conflicts

If you get port conflicts, make sure no other instances of Fluxbase are running:

```bash
lsof -i :8080
kill -9 <PID>
```

### Race Conditions

If tests fail intermittently, you might have race conditions. Run with the race detector:

```bash
go test -race ./test/...
```

## Load Testing

Load tests use k6 to simulate high traffic:

```bash
# Run load tests against running server
make test-load

# Or run directly with custom parameters
k6 run test/k6/load-test.js
```

The load test simulates:

- Ramp up to 100 concurrent users
- CRUD operations
- Complex queries
- Concurrent requests

Performance thresholds:

- 95th percentile < 500ms
- 99th percentile < 1000ms
- Error rate < 5%

## Next Steps

- Read the [IMPLEMENTATION_PLAN.md](the implementation plan) for the development roadmap
- Check [TODO.md](the TODO list) for current tasks
- See the [Getting Started Guide](the getting started guide) for development setup
