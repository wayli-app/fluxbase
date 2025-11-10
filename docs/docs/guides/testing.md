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

Test individual functions in isolation. Location: `internal/*/`

```bash
make test-fast                             # Run all unit tests
go test -v ./internal/auth/...             # Test specific package
go test -v -run TestPasswordHasher ...     # Run specific test
```

**Example:**

```go
func TestPasswordHasher(t *testing.T) {
    hasher := auth.NewPasswordHasher()
    hash, err := hasher.HashPassword("password123")
    assert.NoError(t, err)
    assert.NoError(t, hasher.VerifyPassword(hash, "password123"))
    assert.Error(t, hasher.VerifyPassword(hash, "wrong"))
}
```

#### 2. Integration Tests

Test components with real database. Location: `internal/api/*_integration_test.go`

```bash
go test -v ./internal/api/... -run Integration    # Run all integration tests
```

**Example:**

```go
func (s *Suite) TestUploadFile() {
    resp := s.tc.NewRequest("POST", "/api/v1/storage/buckets/test/files").
        WithFile("file", "test.txt", []byte("Hello")).Send()
    resp.AssertStatus(201)
}
```

#### 3. End-to-End (E2E) Tests

Test full application stack. Location: `test/e2e/`

```bash
make test-e2e                              # Run all E2E tests
make test-full                             # Run unit + integration + E2E
```

**Example:**

```go
func (s *Suite) TestSignUpAndSignIn() {
    resp := s.tc.NewRequest("POST", "/api/v1/auth/signup").
        WithBody(map[string]interface{}{"email": "test@example.com", "password": "pass"}).
        Send()
    resp.AssertStatus(201)
    s.Contains(resp.JSON(), "access_token")
}
```

#### 4. Performance Tests

Test system performance under load. Location: `test/performance/`

```bash
go test -bench=. -benchmem ./test/performance/...    # Run benchmarks
```

### Test Helpers

**TestContext** provides test dependencies:

```go
tc := test.NewTestContext(t)
defer tc.Close()
// tc.DB, tc.Server, tc.App, tc.Config
```

**HTTP requests:**

```go
resp := tc.NewRequest("POST", "/api/v1/tables/users").
    WithBody(map[string]interface{}{"name": "John"}).
    WithAuth("Bearer token").
    Send()
resp.AssertStatus(201)
```

**Database:**

```go
tc.ExecuteSQL("INSERT INTO users...")
tc.CleanDatabase()
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
import { describe, it, expect, beforeEach, vi } from "vitest";
import { FluxbaseAuth } from "./auth";
import type { FluxbaseFetch } from "./fetch";

describe("FluxbaseAuth", () => {
  let auth: FluxbaseAuth;
  let mockFetch: FluxbaseFetch;

  beforeEach(() => {
    // Create mock fetch
    mockFetch = {
      get: vi.fn(),
      post: vi.fn(),
      patch: vi.fn(),
      delete: vi.fn(),
    } as unknown as FluxbaseFetch;

    auth = new FluxbaseAuth(mockFetch);
  });

  describe("signUp", () => {
    it("should sign up a new user", async () => {
      const mockResponse = {
        user: {
          id: "user-123",
          email: "test@example.com",
        },
        access_token: "token-123",
        refresh_token: "refresh-123",
      };

      vi.mocked(mockFetch.post).mockResolvedValue(mockResponse);

      const result = await auth.signUp({
        email: "test@example.com",
        password: "password123",
      });

      expect(mockFetch.post).toHaveBeenCalledWith("/api/v1/auth/signup", {
        email: "test@example.com",
        password: "password123",
      });

      expect(result).toEqual(mockResponse);
      expect(result.user.email).toBe("test@example.com");
      expect(result.access_token).toBe("token-123");
    });

    it("should handle sign up errors", async () => {
      vi.mocked(mockFetch.post).mockRejectedValue(
        new Error("Email already exists"),
      );

      await expect(
        auth.signUp({
          email: "test@example.com",
          password: "password123",
        }),
      ).rejects.toThrow("Email already exists");
    });
  });

  describe("signIn", () => {
    it("should sign in existing user", async () => {
      const mockResponse = {
        user: { id: "user-123", email: "test@example.com" },
        access_token: "token-123",
        refresh_token: "refresh-123",
      };

      vi.mocked(mockFetch.post).mockResolvedValue(mockResponse);

      const result = await auth.signIn({
        email: "test@example.com",
        password: "password123",
      });

      expect(mockFetch.post).toHaveBeenCalledWith("/api/v1/auth/signin", {
        email: "test@example.com",
        password: "password123",
      });

      expect(result.access_token).toBe("token-123");
    });
  });

  describe("signOut", () => {
    it("should sign out user", async () => {
      vi.mocked(mockFetch.post).mockResolvedValue({});

      await auth.signOut();

      expect(mockFetch.post).toHaveBeenCalledWith("/api/v1/auth/signout", {});
    });
  });
});
```

### Mocking with Vitest

**Mock HTTP Fetch:**

```typescript
import { vi } from "vitest";

const mockFetch = {
  get: vi.fn(),
  post: vi.fn(),
  patch: vi.fn(),
  delete: vi.fn(),
};

// Set mock return value
vi.mocked(mockFetch.post).mockResolvedValue({
  data: { id: 123, name: "Test" },
});

// Verify mock was called
expect(mockFetch.post).toHaveBeenCalledWith("/api/endpoint", { data: "test" });

// Check call count
expect(mockFetch.post).toHaveBeenCalledTimes(1);
```

**Mock Modules:**

```typescript
import { vi } from "vitest";

vi.mock("./storage", () => ({
  FluxbaseStorage: vi.fn(() => ({
    upload: vi.fn().mockResolvedValue({ url: "https://example.com/file.jpg" }),
    download: vi.fn().mockResolvedValue(new Blob()),
  })),
}));
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
import { defineConfig } from "vitest/config";

export default defineConfig({
  test: {
    globals: true,
    environment: "node",
    coverage: {
      provider: "v8",
      reporter: ["text", "json", "html"],
      exclude: ["node_modules/", "dist/"],
    },
  },
});
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
      - uses: actions/checkout@v5

      - name: Set up Go
        uses: actions/setup-go@v5
        with:
          go-version: "1.22"

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
      - uses: actions/checkout@v5

      - name: Set up Node.js
        uses: actions/setup-node@v4
        with:
          node-version: "20"

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

| Practice               | Description                                                       |
| ---------------------- | ----------------------------------------------------------------- |
| **Use test suites**    | Organize related tests with `testify/suite` for setup/teardown    |
| **Test isolation**     | Clean data between tests (`TRUNCATE TABLE`, `vi.clearAllMocks()`) |
| **Descriptive names**  | `TestCreateUserWithValidEmail` not `TestUser`                     |
| **Test error cases**   | Test happy path AND error conditions                              |
| **Parallel tests**     | Use `t.Parallel()` for independent unit tests                     |
| **Skip slow tests**    | `if testing.Short() { t.Skip() }` then run `go test -short`       |
| **Table-driven tests** | Test multiple scenarios efficiently with test tables              |

**Example table-driven test:**

```go
func TestValidateEmail(t *testing.T) {
    tests := []struct {
        email   string
        wantErr bool
    }{
        {"user@example.com", false},
        {"invalid", true},
    }
    for _, tt := range tests {
        err := ValidateEmail(tt.email)
        if (err != nil) != tt.wantErr {
            t.Errorf("error = %v, wantErr %v", err, tt.wantErr)
        }
    }
}
```

### 8. Mock External Services

Always mock external services in tests:

```typescript
// Mock HTTP client
vi.mock("./http-client", () => ({
  httpClient: {
    post: vi.fn().mockResolvedValue({ data: "mocked" }),
  },
}));

// Mock WebSocket
vi.mock("./websocket", () => ({
  WebSocketClient: vi.fn(() => ({
    connect: vi.fn(),
    send: vi.fn(),
    close: vi.fn(),
  })),
}));
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

| Issue               | Solution                                               |
| ------------------- | ------------------------------------------------------ |
| Database connection | `pg_isready -h postgres -U postgres`, restart database |
| Port conflicts      | `lsof -i :8080`, `kill -9 <PID>`                       |
| Race conditions     | Run `go test -race ./...`, fix shared state access     |
| Flaky tests         | Use deterministic time, proper mocking, ensure cleanup |

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

---

## Summary

Fluxbase provides comprehensive testing infrastructure:

- ✅ **Backend Testing**: Unit, integration, E2E, performance tests with Go
- ✅ **SDK Testing**: Vitest-based TypeScript testing with mocking
- ✅ **Test Helpers**: TestContext, HTTP builders, database utilities
- ✅ **CI/CD Integration**: Automated tests on every push
- ✅ **Coverage Reporting**: Track test coverage over time
- ✅ **Performance Testing**: Benchmarks and load tests
- ✅ **Debugging Tools**: Race detector, verbose logging, debugger support

Write tests early, test all paths (happy and error), maintain high coverage, and run tests frequently to ensure code quality and prevent regressions.
