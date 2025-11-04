---
title: Testing Guide
sidebar_position: 12
---

# Testing Guide

Fluxbase has comprehensive testing infrastructure covering unit tests, integration tests, end-to-end tests, and performance tests for both backend (Go) and SDK (TypeScript).

## Overview

```
┌─────────────────────────────────────────────────────────────┐
│                     Testing Pyramid                         │
│                                                              │
│                      ┌─────────────┐                         │
│                      │   E2E Tests │  (Slowest, Full Stack) │
│                      └─────────────┘                         │
│                 ┌───────────────────────┐                    │
│                 │  Integration Tests    │  (Database + API) │
│                 └───────────────────────┘                    │
│            ┌──────────────────────────────────┐             │
│            │         Unit Tests                │  (Fastest) │
│            └──────────────────────────────────┘             │
│                                                              │
└─────────────────────────────────────────────────────────────┘
```

---

## Backend Testing (Go)

Fluxbase backend uses Go's built-in testing framework with `testify` for assertions and test suites.

### Test Types

#### 1. Unit Tests

Test individual functions and components in isolation without external dependencies.

**Location**: `internal/*/` (alongside source files)

**Run Unit Tests:**

```bash
# Run all unit tests (fast)
make test-fast

# Run with race detector
make test

# Run specific package
go test -v ./internal/auth/...

# Run specific test
go test -v -run TestPasswordHasher ./internal/auth/...
```

**Example Unit Test** ([internal/auth/password_test.go:1-50](internal/auth/password_test.go#L1-L50)):

```go
package auth_test

import (
    "testing"
    "github.com/stretchr/testify/assert"
    "github.com/wayli-app/fluxbase/internal/auth"
)

func TestPasswordHasher(t *testing.T) {
    hasher := auth.NewPasswordHasherWithConfig(auth.PasswordHasherConfig{
        MinLength: 8,
        Cost:      10,
    })

    t.Run("Hash password", func(t *testing.T) {
        password := "MyPassword123!"
        hash, err := hasher.HashPassword(password)

        assert.NoError(t, err)
        assert.NotEmpty(t, hash)
        assert.NotEqual(t, password, hash)
    })

    t.Run("Verify correct password", func(t *testing.T) {
        password := "MyPassword123!"
        hash, _ := hasher.HashPassword(password)

        err := hasher.VerifyPassword(hash, password)
        assert.NoError(t, err)
    })

    t.Run("Reject incorrect password", func(t *testing.T) {
        password := "MyPassword123!"
        hash, _ := hasher.HashPassword(password)

        err := hasher.VerifyPassword(hash, "WrongPassword")
        assert.Error(t, err)
    })
}
```

#### 2. Integration Tests

Test components with real database interactions.

**Location**: `internal/api/*_integration_test.go`

**Run Integration Tests:**

```bash
# Run all integration tests
go test -v ./internal/api/... -run Integration

# Run specific integration test
go test -v -run TestStorageIntegration ./internal/api/...
```

**Example Integration Test** ([internal/api/storage_integration_test.go:1-80](internal/api/storage_integration_test.go#L1-L80)):

```go
package api_test

import (
    "testing"
    "github.com/stretchr/testify/suite"
    "github.com/wayli-app/fluxbase/test"
)

type StorageIntegrationSuite struct {
    suite.Suite
    tc *test.TestContext
}

func (s *StorageIntegrationSuite) SetupSuite() {
    s.tc = test.NewTestContext(s.T())
}

func (s *StorageIntegrationSuite) TearDownSuite() {
    s.tc.Close()
}

func (s *StorageIntegrationSuite) TestUploadAndDownloadFile() {
    // Upload file
    resp := s.tc.NewRequest("POST", "/api/v1/storage/buckets/test/files").
        WithFile("file", "test.txt", []byte("Hello World")).
        Send()

    resp.AssertStatus(201)

    var result map[string]interface{}
    resp.JSON(&result)

    // Download file
    downloadResp := s.tc.NewRequest("GET", "/api/v1/storage/buckets/test/files/test.txt").
        Send()

    downloadResp.AssertStatus(200)
    s.Equal("Hello World", string(downloadResp.Body))
}

func TestStorageIntegrationSuite(t *testing.T) {
    if testing.Short() {
        t.Skip("Skipping integration tests in short mode")
    }
    suite.Run(t, new(StorageIntegrationSuite))
}
```

#### 3. End-to-End (E2E) Tests

Test the entire application stack including HTTP endpoints, database, and services.

**Location**: `test/e2e/`

**Run E2E Tests:**

```bash
# Run all E2E tests
make test-e2e

# Run specific E2E test suite
go test -v ./test/e2e/ -run TestAuthSuite

# Run full test suite (unit + integration + E2E)
make test-full
```

**Example E2E Test** ([test/e2e/auth_test.go:1-100](test/e2e/auth_test.go#L1-L100)):

```go
package e2e_test

import (
    "testing"
    "github.com/gofiber/fiber/v2"
    "github.com/stretchr/testify/suite"
    "github.com/wayli-app/fluxbase/test"
)

type AuthTestSuite struct {
    suite.Suite
    tc *test.TestContext
}

func (s *AuthTestSuite) SetupSuite() {
    s.tc = test.NewTestContext(s.T())
}

func (s *AuthTestSuite) TearDownSuite() {
    s.tc.Close()
}

func (s *AuthTestSuite) SetupTest() {
    // Clean auth data before each test
    s.tc.ExecuteSQL("TRUNCATE TABLE auth.users CASCADE")
}

func (s *AuthTestSuite) TestSignUpAndSignIn() {
    email := "test@example.com"
    password := "SecurePass123!"

    // Sign up
    signUpResp := s.tc.NewRequest("POST", "/api/v1/auth/signup").
        WithBody(map[string]interface{}{
            "email":    email,
            "password": password,
        }).
        Send()

    signUpResp.AssertStatus(fiber.StatusCreated)

    var signUpResult map[string]interface{}
    signUpResp.JSON(&signUpResult)
    s.Contains(signUpResult, "user")
    s.Contains(signUpResult, "access_token")

    // Sign in
    signInResp := s.tc.NewRequest("POST", "/api/v1/auth/signin").
        WithBody(map[string]interface{}{
            "email":    email,
            "password": password,
        }).
        Send()

    signInResp.AssertStatus(fiber.StatusOK)

    var signInResult map[string]interface{}
    signInResp.JSON(&signInResult)
    s.Contains(signInResult, "access_token")
    s.Contains(signInResult, "refresh_token")
}

func TestAuthSuite(t *testing.T) {
    if testing.Short() {
        t.Skip("Skipping E2E tests in short mode")
    }
    suite.Run(t, new(AuthTestSuite))
}
```

#### 4. Performance Tests

Test system performance under load.

**Location**: `test/performance/`

**Run Performance Tests:**

```bash
# Run performance tests
go test -v ./test/performance/...

# Run with benchmarks
go test -bench=. -benchmem ./test/performance/...
```

**Example Performance Test** ([test/performance/rest_load_test.go:1-60](test/performance/rest_load_test.go#L1-L60)):

```go
package performance_test

import (
    "sync"
    "testing"
    "time"
)

func TestRESTAPILoad(t *testing.T) {
    if testing.Short() {
        t.Skip("Skipping load tests in short mode")
    }

    tc := test.NewTestContext(t)
    defer tc.Close()

    // Concurrent requests
    concurrency := 100
    requests := 1000

    var wg sync.WaitGroup
    results := make(chan time.Duration, requests)

    startTime := time.Now()

    for i := 0; i < concurrency; i++ {
        wg.Add(1)
        go func() {
            defer wg.Done()

            for j := 0; j < requests/concurrency; j++ {
                reqStart := time.Now()

                resp := tc.NewRequest("GET", "/api/v1/tables/items").Send()

                duration := time.Since(reqStart)
                results <- duration

                if resp.Status() != 200 {
                    t.Errorf("Unexpected status: %d", resp.Status())
                }
            }
        }()
    }

    wg.Wait()
    close(results)

    totalDuration := time.Since(startTime)

    // Calculate statistics
    var total time.Duration
    count := 0
    for d := range results {
        total += d
        count++
    }

    avgLatency := total / time.Duration(count)
    throughput := float64(count) / totalDuration.Seconds()

    t.Logf("Total requests: %d", count)
    t.Logf("Total duration: %v", totalDuration)
    t.Logf("Average latency: %v", avgLatency)
    t.Logf("Throughput: %.2f req/s", throughput)

    // Assert performance targets
    if avgLatency > 100*time.Millisecond {
        t.Errorf("Average latency too high: %v", avgLatency)
    }

    if throughput < 100 {
        t.Errorf("Throughput too low: %.2f req/s", throughput)
    }
}
```

### Test Helpers

#### TestContext

The `TestContext` provides all dependencies for tests:

```go
package mypackage_test

import (
    "testing"
    "github.com/wayli-app/fluxbase/test"
)

func TestMyFeature(t *testing.T) {
    tc := test.NewTestContext(t)
    defer tc.Close()

    // tc.DB - Database connection
    // tc.Server - API server
    // tc.App - Fiber app
    // tc.Config - Configuration
}
```

#### HTTP Request Builder

Make HTTP requests easily:

```go
// GET request
resp := tc.NewRequest("GET", "/api/v1/tables/users").Send()
resp.AssertStatus(200)

// POST with JSON body
resp := tc.NewRequest("POST", "/api/v1/tables/users").
    WithBody(map[string]interface{}{
        "name": "John Doe",
        "email": "john@example.com",
    }).
    Send()

// With authentication
resp := tc.NewRequest("GET", "/api/v1/tables/private_data").
    WithAuth("Bearer your-token-here").
    Send()

// With custom headers
resp := tc.NewRequest("POST", "/api/v1/tables/items").
    WithHeader("X-API-Key", "your-api-key").
    WithBody(data).
    Send()

// Parse JSON response
var result map[string]interface{}
resp.JSON(&result)
```

#### Database Helpers

Execute SQL and manage test data:

```go
// Execute SQL
tc.ExecuteSQL("INSERT INTO users (name, email) VALUES ($1, $2)", "John", "john@example.com")

// Query data
rows := tc.QuerySQL("SELECT * FROM users WHERE email = $1", "john@example.com")

// Clean database
tc.CleanDatabase()

// Create test table
tc.CreateTestTable("test_items", `
    CREATE TABLE test_items (
        id SERIAL PRIMARY KEY,
        name TEXT NOT NULL
    )
`)

// Drop test table
tc.DropTestTable("test_items")
```

---

## SDK Testing (TypeScript)

The TypeScript SDK uses [Vitest](https://vitest.dev/) for testing with mocking and assertions.

### Running SDK Tests

```bash
cd sdk

# Run all tests
npm test

# Run tests in watch mode
npm test -- --watch

# Run specific test file
npm test -- src/auth.test.ts

# Run with UI
npm run test:ui

# Type checking
npm run type-check
```

### Test Structure

**Location**: `sdk/src/*.test.ts`

**Available Test Files**:
- [src/auth.test.ts](sdk/src/auth.test.ts) - Authentication tests
- [src/admin.test.ts](sdk/src/admin.test.ts) - Admin SDK tests
- [src/management.test.ts](sdk/src/management.test.ts) - Management SDK tests
- [src/query-builder.test.ts](sdk/src/query-builder.test.ts) - Query builder tests
- [src/realtime.test.ts](sdk/src/realtime.test.ts) - Realtime tests
- [src/storage.test.ts](sdk/src/storage.test.ts) - Storage tests
- [src/aggregations.test.ts](sdk/src/aggregations.test.ts) - Aggregation tests

### Example SDK Test

**Authentication Test** ([sdk/src/auth.test.ts:1-150](sdk/src/auth.test.ts#L1-L150)):

```typescript
import { describe, it, expect, beforeEach, vi } from 'vitest'
import { FluxbaseAuth } from './auth'
import type { FluxbaseFetch } from './fetch'

describe('FluxbaseAuth', () => {
  let auth: FluxbaseAuth
  let mockFetch: FluxbaseFetch

  beforeEach(() => {
    // Create mock fetch
    mockFetch = {
      get: vi.fn(),
      post: vi.fn(),
      patch: vi.fn(),
      delete: vi.fn(),
    } as unknown as FluxbaseFetch

    auth = new FluxbaseAuth(mockFetch)
  })

  describe('signUp', () => {
    it('should sign up a new user', async () => {
      const mockResponse = {
        user: {
          id: 'user-123',
          email: 'test@example.com',
        },
        access_token: 'token-123',
        refresh_token: 'refresh-123',
      }

      vi.mocked(mockFetch.post).mockResolvedValue(mockResponse)

      const result = await auth.signUp({
        email: 'test@example.com',
        password: 'password123',
      })

      expect(mockFetch.post).toHaveBeenCalledWith('/api/v1/auth/signup', {
        email: 'test@example.com',
        password: 'password123',
      })

      expect(result).toEqual(mockResponse)
      expect(result.user.email).toBe('test@example.com')
      expect(result.access_token).toBe('token-123')
    })

    it('should handle sign up errors', async () => {
      vi.mocked(mockFetch.post).mockRejectedValue(
        new Error('Email already exists')
      )

      await expect(
        auth.signUp({
          email: 'test@example.com',
          password: 'password123',
        })
      ).rejects.toThrow('Email already exists')
    })
  })

  describe('signIn', () => {
    it('should sign in existing user', async () => {
      const mockResponse = {
        user: { id: 'user-123', email: 'test@example.com' },
        access_token: 'token-123',
        refresh_token: 'refresh-123',
      }

      vi.mocked(mockFetch.post).mockResolvedValue(mockResponse)

      const result = await auth.signIn({
        email: 'test@example.com',
        password: 'password123',
      })

      expect(mockFetch.post).toHaveBeenCalledWith('/api/v1/auth/signin', {
        email: 'test@example.com',
        password: 'password123',
      })

      expect(result.access_token).toBe('token-123')
    })
  })

  describe('signOut', () => {
    it('should sign out user', async () => {
      vi.mocked(mockFetch.post).mockResolvedValue({})

      await auth.signOut()

      expect(mockFetch.post).toHaveBeenCalledWith('/api/v1/auth/signout', {})
    })
  })
})
```

### Mocking with Vitest

**Mock HTTP Fetch:**

```typescript
import { vi } from 'vitest'

const mockFetch = {
  get: vi.fn(),
  post: vi.fn(),
  patch: vi.fn(),
  delete: vi.fn(),
}

// Set mock return value
vi.mocked(mockFetch.post).mockResolvedValue({
  data: { id: 123, name: 'Test' }
})

// Verify mock was called
expect(mockFetch.post).toHaveBeenCalledWith('/api/endpoint', { data: 'test' })

// Check call count
expect(mockFetch.post).toHaveBeenCalledTimes(1)
```

**Mock Modules:**

```typescript
import { vi } from 'vitest'

vi.mock('./storage', () => ({
  FluxbaseStorage: vi.fn(() => ({
    upload: vi.fn().mockResolvedValue({ url: 'https://example.com/file.jpg' }),
    download: vi.fn().mockResolvedValue(new Blob()),
  })),
}))
```

---

## Test Configuration

### Backend Test Configuration

Tests use a separate `fluxbase_test` database:

**Environment Variables:**

```bash
# Test database
FLUXBASE_DATABASE_HOST=postgres
FLUXBASE_DATABASE_USER=fluxbase_app
FLUXBASE_DATABASE_PASSWORD=fluxbase_app_password
FLUXBASE_DATABASE_DATABASE=fluxbase_test

# Test services
FLUXBASE_EMAIL_SMTP_HOST=mailhog
FLUXBASE_STORAGE_S3_ENDPOINT=minio:9000
```

**Test Config** ([test/e2e_helpers.go:174-230](test/e2e_helpers.go#L174-L230)):

```go
func GetTestConfig() *config.Config {
    return &config.Config{
        Server: config.ServerConfig{
            Address:      ":8081",
            ReadTimeout:  15 * time.Second,
            WriteTimeout: 15 * time.Second,
            IdleTimeout:  60 * time.Second,
            BodyLimit:    10 * 1024 * 1024,
        },
        Database: config.DatabaseConfig{
            Host:            "postgres",
            Port:            5432,
            User:            "fluxbase_app",
            Password:        "fluxbase_app_password",
            Database:        "fluxbase_test",
            SSLMode:         "disable",
            MaxConnections:  25,
            MinConnections:  5,
        },
        Auth: config.AuthConfig{
            JWTSecret:     "test-secret-key-for-testing-only",
            JWTExpiry:     15 * time.Minute,
            RefreshExpiry: 168 * time.Hour,
            EnableSignup:  true,
            EnableRLS:     true,
        },
        Debug: true,
    }
}
```

### SDK Test Configuration

SDK tests use mocked HTTP calls and don't require a running server.

**Vitest Configuration** (`sdk/vitest.config.ts`):

```typescript
import { defineConfig } from 'vitest/config'

export default defineConfig({
  test: {
    globals: true,
    environment: 'node',
    coverage: {
      provider: 'v8',
      reporter: ['text', 'json', 'html'],
      exclude: ['node_modules/', 'dist/'],
    },
  },
})
```

---

## CI/CD Testing

Tests run automatically in GitHub Actions on every push and pull request.

**GitHub Actions Workflow** (`.github/workflows/test.yml`):

```yaml
name: Tests

on:
  push:
    branches: [main, develop]
  pull_request:
    branches: [main, develop]

jobs:
  test-backend:
    runs-on: ubuntu-latest

    services:
      postgres:
        image: postgres:16-alpine
        env:
          POSTGRES_PASSWORD: postgres
        options: >-
          --health-cmd pg_isready
          --health-interval 10s
          --health-timeout 5s
          --health-retries 5
        ports:
          - 5432:5432

    steps:
      - uses: actions/checkout@v4

      - name: Set up Go
        uses: actions/setup-go@v5
        with:
          go-version: '1.22'

      - name: Setup test database
        run: ./test/scripts/setup_test_db.sh

      - name: Run tests
        run: make test-full

      - name: Upload coverage
        uses: codecov/codecov-action@v3
        with:
          file: ./coverage.out

  test-sdk:
    runs-on: ubuntu-latest

    steps:
      - uses: actions/checkout@v4

      - name: Set up Node.js
        uses: actions/setup-node@v4
        with:
          node-version: '20'

      - name: Install dependencies
        working-directory: ./sdk
        run: npm install

      - name: Run SDK tests
        working-directory: ./sdk
        run: npm test

      - name: Type check
        working-directory: ./sdk
        run: npm run type-check
```

---

## Test Coverage

### Backend Coverage

Generate test coverage report:

```bash
# Generate coverage report
go test -cover -coverprofile=coverage.out ./...

# View coverage in terminal
go tool cover -func=coverage.out

# Generate HTML report
go tool cover -html=coverage.out -o coverage.html

# Open in browser
open coverage.html  # macOS
xdg-open coverage.html  # Linux
```

**Coverage Targets**:
- Overall: > 70%
- Critical paths (auth, RLS, security): > 90%
- Utilities: > 80%

### SDK Coverage

```bash
cd sdk

# Run tests with coverage
npm test -- --coverage

# View coverage report
open coverage/index.html
```

**Coverage Targets**:
- Overall: > 80%
- Core modules: > 90%

---

## Best Practices

### 1. Use Test Suites

Organize related tests with `testify/suite`:

```go
type MyTestSuite struct {
    suite.Suite
    tc *test.TestContext
}

func (s *MyTestSuite) SetupSuite() {
    s.tc = test.NewTestContext(s.T())
}

func (s *MyTestSuite) TearDownSuite() {
    s.tc.Close()
}

func (s *MyTestSuite) SetupTest() {
    // Clean data before each test
    s.tc.ExecuteSQL("TRUNCATE TABLE items CASCADE")
}

func TestMySuite(t *testing.T) {
    if testing.Short() {
        t.Skip("Skipping E2E tests in short mode")
    }
    suite.Run(t, new(MyTestSuite))
}
```

### 2. Test Isolation

Always clean data between tests:

```go
func (s *MyTestSuite) SetupTest() {
    s.tc.ExecuteSQL("TRUNCATE TABLE items CASCADE")
    s.tc.ExecuteSQL("TRUNCATE TABLE users CASCADE")
}
```

For TypeScript:

```typescript
beforeEach(() => {
    vi.clearAllMocks()
})
```

### 3. Descriptive Test Names

```go
// ✅ GOOD
func (s *MyTestSuite) TestCreateUserWithValidEmail() {}
func (s *MyTestSuite) TestCreateUserWithInvalidEmailReturns400() {}

// ❌ BAD
func (s *MyTestSuite) TestUser() {}
func (s *MyTestSuite) TestCreate() {}
```

### 4. Test Error Cases

Don't just test the happy path:

```go
func (s *MyTestSuite) TestErrorHandling() {
    // Test missing required field
    resp := s.tc.NewRequest("POST", "/api/v1/tables/users").
        WithBody(map[string]interface{}{
            // Missing email field
            "name": "John",
        }).
        Send()

    resp.AssertStatus(400)

    // Test invalid data type
    resp = s.tc.NewRequest("POST", "/api/v1/tables/users").
        WithBody(map[string]interface{}{
            "email": 12345, // Should be string
        }).
        Send()

    resp.AssertStatus(400)
}
```

### 5. Use Parallel Tests

For independent unit tests:

```go
func TestMyFeature(t *testing.T) {
    t.Parallel()

    t.Run("SubTest1", func(t *testing.T) {
        t.Parallel()
        // Test code
    })

    t.Run("SubTest2", func(t *testing.T) {
        t.Parallel()
        // Test code
    })
}
```

### 6. Skip Slow Tests

Allow fast test runs:

```go
func TestSlowOperation(t *testing.T) {
    if testing.Short() {
        t.Skip("Skipping slow test in short mode")
    }
    // Test code
}
```

Run with: `go test -short ./...`

### 7. Use Table-Driven Tests

Test multiple scenarios efficiently:

```go
func TestValidateEmail(t *testing.T) {
    tests := []struct {
        name    string
        email   string
        wantErr bool
    }{
        {"Valid email", "user@example.com", false},
        {"Invalid - no @", "userexample.com", true},
        {"Invalid - no domain", "user@", true},
        {"Invalid - empty", "", true},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            err := ValidateEmail(tt.email)
            if (err != nil) != tt.wantErr {
                t.Errorf("ValidateEmail() error = %v, wantErr %v", err, tt.wantErr)
            }
        })
    }
}
```

### 8. Mock External Services

Always mock external services in tests:

```typescript
// Mock HTTP client
vi.mock('./http-client', () => ({
  httpClient: {
    post: vi.fn().mockResolvedValue({ data: 'mocked' })
  }
}))

// Mock WebSocket
vi.mock('./websocket', () => ({
  WebSocketClient: vi.fn(() => ({
    connect: vi.fn(),
    send: vi.fn(),
    close: vi.fn(),
  }))
}))
```

---

## Debugging Tests

### Backend Debugging

**Run Single Test:**

```bash
go test -v -run TestSpecificTest ./test/e2e/...
```

**Run with Race Detector:**

```bash
go test -race ./...
```

**Enable Debug Logging:**

```bash
FLUXBASE_DEBUG=true go test -v ./test/e2e/...
```

**Use Debugger (Delve):**

```bash
# Install delve
go install github.com/go-delve/delve/cmd/dlv@latest

# Debug specific test
dlv test ./test/e2e -- -test.run TestAuthSuite
```

### SDK Debugging

**Run with Verbose Output:**

```bash
npm test -- --reporter=verbose
```

**Debug in VS Code:**

Add to `.vscode/launch.json`:

```json
{
  "type": "node",
  "request": "launch",
  "name": "Debug Vitest Tests",
  "runtimeExecutable": "npm",
  "runtimeArgs": ["test", "--", "--run"],
  "console": "integratedTerminal",
  "internalConsoleOptions": "neverOpen"
}
```

**Watch Mode:**

```bash
npm test -- --watch
```

---

## Troubleshooting

### Database Connection Issues

```bash
# Check PostgreSQL is running
pg_isready -h postgres -U postgres

# Restart database
docker-compose -f .devcontainer/docker-compose.yml restart postgres

# Re-setup test database
./test/scripts/setup_test_db.sh
```

### Port Conflicts

```bash
# Check what's using port 8080
lsof -i :8080

# Kill process
kill -9 <PID>

# Or kill all Go test processes
pkill -9 go
```

### Race Conditions

```bash
# Run with race detector
go test -race ./...

# If race detected, fix shared state access
```

### Flaky Tests

**Common causes**:
- Time-dependent tests (use fake time)
- Race conditions (add proper synchronization)
- External dependencies (mock them)
- Test data cleanup issues (ensure proper cleanup)

**Fix example**:

```go
// ❌ BAD: Time-dependent
func TestTokenExpiry(t *testing.T) {
    token := CreateToken(time.Now().Add(1 * time.Second))
    time.Sleep(2 * time.Second)
    assert.True(t, token.IsExpired())
}

// ✅ GOOD: Use deterministic time
func TestTokenExpiry(t *testing.T) {
    now := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
    expiryTime := now.Add(1 * time.Hour)

    token := CreateTokenWithExpiry(expiryTime)

    // Test before expiry
    assert.False(t, token.IsExpiredAt(now.Add(30 * time.Minute)))

    // Test after expiry
    assert.True(t, token.IsExpiredAt(now.Add(2 * time.Hour)))
}
```

---

## Performance Testing

### Go Benchmarks

```go
func BenchmarkPasswordHashing(b *testing.B) {
    hasher := auth.NewPasswordHasher()
    password := "MySecurePassword123!"

    b.ResetTimer()

    for i := 0; i < b.N; i++ {
        _, _ = hasher.HashPassword(password)
    }
}

func BenchmarkQueryExecution(b *testing.B) {
    tc := test.NewTestContext(&testing.T{})
    defer tc.Close()

    b.ResetTimer()

    for i := 0; i < b.N; i++ {
        _ = tc.QuerySQL("SELECT * FROM users LIMIT 100")
    }
}
```

**Run Benchmarks:**

```bash
# Run all benchmarks
go test -bench=. ./...

# Run specific benchmark
go test -bench=BenchmarkPasswordHashing ./internal/auth/...

# With memory profiling
go test -bench=. -benchmem ./...

# CPU profiling
go test -bench=. -cpuprofile=cpu.prof ./...
go tool pprof cpu.prof
```

### Load Testing with k6

**Install k6:**

```bash
# macOS
brew install k6

# Linux
sudo snap install k6

# Docker
docker pull grafana/k6
```

**Load Test Script** (`test/k6/load-test.js`):

```javascript
import http from 'k6/http';
import { check, sleep } from 'k6';

export const options = {
  stages: [
    { duration: '30s', target: 20 },  // Ramp up to 20 users
    { duration: '1m', target: 20 },   // Stay at 20 users
    { duration: '30s', target: 100 }, // Ramp up to 100 users
    { duration: '2m', target: 100 },  // Stay at 100 users
    { duration: '30s', target: 0 },   // Ramp down
  ],
  thresholds: {
    http_req_duration: ['p(95)<500', 'p(99)<1000'],
    http_req_failed: ['rate<0.05'],
  },
};

export default function () {
  // Test GET request
  const getRes = http.get('http://localhost:8080/api/v1/tables/items');
  check(getRes, {
    'GET status is 200': (r) => r.status === 200,
    'GET response time < 500ms': (r) => r.timings.duration < 500,
  });

  sleep(1);

  // Test POST request
  const postRes = http.post(
    'http://localhost:8080/api/v1/tables/items',
    JSON.stringify({ name: 'Test Item', quantity: 10 }),
    {
      headers: { 'Content-Type': 'application/json' },
    }
  );

  check(postRes, {
    'POST status is 201': (r) => r.status === 201,
    'POST response time < 500ms': (r) => r.timings.duration < 500,
  });

  sleep(1);
}
```

**Run Load Test:**

```bash
# Run locally
k6 run test/k6/load-test.js

# Run with cloud results
k6 cloud test/k6/load-test.js

# Run in Docker
docker run --network="host" -i grafana/k6 run - <test/k6/load-test.js
```

---

## Summary

Fluxbase provides comprehensive testing infrastructure:

✅ **Backend Testing**: Unit, integration, E2E, performance tests with Go
✅ **SDK Testing**: Vitest-based TypeScript testing with mocking
✅ **Test Helpers**: TestContext, HTTP builders, database utilities
✅ **CI/CD Integration**: Automated tests on every push
✅ **Coverage Reporting**: Track test coverage over time
✅ **Performance Testing**: Benchmarks and load tests
✅ **Debugging Tools**: Race detector, verbose logging, debugger support

Write tests early, test all paths (happy and error), maintain high coverage, and run tests frequently to ensure code quality and prevent regressions.
