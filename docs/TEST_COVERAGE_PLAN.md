# Test Coverage Improvement Plan

## Executive Summary

| Current State | Target State |
|--------------|--------------|
| Overall: **11.1%** | Overall: **80%** |
| Critical modules: **7-18%** | Critical modules: **90%** |
| Zero-coverage files: **156** | Zero-coverage files: **0** |

**Estimated scope**: ~37,000 uncovered statements across 275 source files.

---

## Coverage Targets (Industry Best Practices)

| Category | Target | Rationale |
|----------|--------|-----------|
| **Overall** | 80% | Industry standard for production systems |
| **Security-critical** (auth, session, crypto) | 90% | High bug impact, compliance requirements |
| **API handlers** | 85% | User-facing, high traffic |
| **Business logic** | 80% | Core functionality |
| **Infrastructure** | 60% | Often tested via E2E, harder to unit test |
| **Utilities/helpers** | 70% | Supporting code |

---

## Phase Overview

| Phase | Focus | Modules | Coverage Gain | Effort |
|-------|-------|---------|---------------|--------|
| **1** | Security-Critical | auth, crypto, middleware | +15% | HIGH |
| **2** | Core API | api (handlers, REST) | +20% | HIGH |
| **3** | Data Layer | storage, jobs, database | +12% | MEDIUM |
| **4** | Features | functions, ai, mcp, realtime | +18% | HIGH |
| **5** | Supporting | branching, rpc, config, email | +10% | MEDIUM |
| **6** | Polish | Remaining gaps, edge cases | +5% | LOW |

---

## Phase 1: Security-Critical (Target: 90%)

### Priority: CRITICAL

Security bugs in authentication/authorization can lead to data breaches. These modules must have the highest coverage.

### Files to Test (by priority)

#### 1.1 Authentication Core (`internal/auth/`)

| File | Current | Target | Statements | Priority |
|------|---------|--------|------------|----------|
| `service.go` | 0% | 90% | 431 | **P0** |
| `session.go` | 0% | 90% | 157 | **P0** |
| `user.go` | 0% | 90% | 214 | **P0** |
| `user_management.go` | 0% | 90% | 124 | **P0** |
| `dashboard.go` | 0% | 90% | 409 | **P1** |
| `oauth.go` | 0% | 85% | 89 | **P1** |
| `otp.go` | 0% | 85% | 111 | **P1** |
| `invitation.go` | 0% | 80% | 76 | **P2** |
| `impersonation.go` | 1.6% | 90% | 125 | **P1** |
| `identity.go` | 1.9% | 85% | 106 | **P1** |
| `clientkey.go` | 2.4% | 85% | 82 | **P1** |
| `saml.go` | 13.7% | 80% | 541 | **P2** |
| `settings_cache.go` | 19.2% | 80% | 120 | **P2** |

**Already well-tested** (maintain):
- `jwt.go` (78.8%) - Excellent tests, use as template
- `password.go` (97.4%)
- `validation.go` (98.6%)
- `scopes.go` (100%)

#### 1.2 Middleware (`internal/middleware/`)

| File | Current | Target | Statements | Priority |
|------|---------|--------|------------|----------|
| `clientkey_auth.go` | 0% | 90% | 350 | **P0** |
| `auth.go` | 0% | 90% | 180 | **P0** |
| `cors.go` | 0% | 80% | 85 | **P1** |
| `ratelimit.go` | 0% | 80% | 120 | **P1** |

#### 1.3 Crypto (`internal/crypto/`)

| File | Current | Target | Statements | Priority |
|------|---------|--------|------------|----------|
| `encrypt.go` | 76% | 95% | 50 | **P1** |

### Testing Strategy for Phase 1

```go
// Pattern: Use existing mock infrastructure
// See: internal/testutil/mocks.go

// For auth/service.go, create mock repositories:
type MockUserRepository struct {
    users map[string]*User
    // ... implement interface methods
}

type MockSessionRepository struct {
    sessions map[string]*Session
    // ...
}
```

**Key test scenarios for auth:**
1. User registration (valid, invalid email, duplicate)
2. Login (correct password, wrong password, locked account, unverified email)
3. Session management (create, refresh, revoke, expire)
4. Token validation (valid, expired, blacklisted, wrong signature)
5. Password reset flow (request, validate token, complete)
6. MFA enrollment and verification
7. OAuth flows (authorization, callback, token exchange)
8. Role-based access control
9. Rate limiting on sensitive endpoints

### New Mocks Needed

```go
// Add to internal/testutil/mocks.go or internal/auth/mock_repositories.go

type MockUserRepository interface {
    Create(ctx context.Context, user *User) error
    GetByID(ctx context.Context, id string) (*User, error)
    GetByEmail(ctx context.Context, email string) (*User, error)
    Update(ctx context.Context, user *User) error
    Delete(ctx context.Context, id string) error
}

type MockSessionRepository interface {
    Create(ctx context.Context, session *Session) error
    GetByID(ctx context.Context, id string) (*Session, error)
    GetByUserID(ctx context.Context, userID string) ([]*Session, error)
    Delete(ctx context.Context, id string) error
    DeleteByUserID(ctx context.Context, userID string) error
}
```

### Milestone Checklist - Phase 1

- [ ] `auth/service.go` - 90% coverage
- [ ] `auth/session.go` - 90% coverage
- [ ] `auth/user.go` - 90% coverage
- [ ] `auth/user_management.go` - 90% coverage
- [ ] `middleware/clientkey_auth.go` - 90% coverage
- [ ] `middleware/auth.go` - 90% coverage
- [ ] `auth/dashboard.go` - 90% coverage
- [ ] `auth/oauth.go` - 85% coverage
- [ ] `auth/otp.go` - 85% coverage
- [ ] `auth/impersonation.go` - 90% coverage
- [ ] `auth/identity.go` - 85% coverage
- [ ] `crypto/encrypt.go` - 95% coverage
- [ ] Update `.testcoverage.yml` thresholds for auth module

---

## Phase 2: Core API (Target: 85%)

### Priority: HIGH

The API handlers are the primary user interface. Bugs here directly impact users.

### Files to Test (by priority)

#### 2.1 Server & Core Handlers (`internal/api/`)

| File | Current | Target | Statements | Priority |
|------|---------|--------|------------|----------|
| `server.go` | 0% | 70% | 967 | **P1** |
| `auth_handler.go` | 0% | 90% | 563 | **P0** |
| `dashboard_auth_handler.go` | 0% | 85% | 614 | **P1** |
| `rest_crud.go` | 0% | 85% | 192 | **P0** |
| `rest_handler.go` | 0% | 85% | 110 | **P0** |
| `rest_batch.go` | 0% | 80% | 160 | **P1** |

#### 2.2 Storage Handlers

| File | Current | Target | Statements | Priority |
|------|---------|--------|------------|----------|
| `storage_files.go` | 0% | 85% | 357 | **P0** |
| `storage_buckets.go` | 0% | 85% | 126 | **P1** |
| `storage_chunked.go` | 0% | 80% | 174 | **P2** |

#### 2.3 OAuth & SSO Handlers

| File | Current | Target | Statements | Priority |
|------|---------|--------|------------|----------|
| `oauth_handler.go` | 0% | 85% | 334 | **P1** |
| `oauth_provider_handler.go` | 0% | 80% | 334 | **P2** |
| `saml_provider_handler.go` | 0% | 80% | 310 | **P2** |
| `auth_saml.go` | 0% | 80% | 235 | **P2** |

#### 2.4 Admin & Settings Handlers

| File | Current | Target | Statements | Priority |
|------|---------|--------|------------|----------|
| `admin_auth_handler.go` | 0% | 85% | 90 | **P1** |
| `settings_handler.go` | 0% | 80% | 59 | **P2** |
| `app_settings_handler.go` | 0% | 80% | 242 | **P2** |
| `user_settings_handler.go` | 0% | 80% | 233 | **P2** |

### Testing Strategy for Phase 2

```go
// Use httptest for handler testing
import (
    "net/http/httptest"
    "github.com/gofiber/fiber/v2"
)

func TestAuthHandler_Login(t *testing.T) {
    // Setup
    app := fiber.New()
    mockAuthService := NewMockAuthService()
    handler := NewAuthHandler(mockAuthService)

    app.Post("/auth/login", handler.Login)

    tests := []struct {
        name       string
        body       string
        setupMock  func()
        wantStatus int
        wantBody   string
    }{
        {
            name: "successful login",
            body: `{"email":"test@example.com","password":"correct"}`,
            setupMock: func() {
                mockAuthService.LoginResult = &TokenPair{...}
            },
            wantStatus: 200,
        },
        {
            name: "invalid credentials",
            body: `{"email":"test@example.com","password":"wrong"}`,
            setupMock: func() {
                mockAuthService.LoginError = ErrInvalidCredentials
            },
            wantStatus: 401,
        },
        // ... more cases
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            tt.setupMock()
            req := httptest.NewRequest("POST", "/auth/login", strings.NewReader(tt.body))
            req.Header.Set("Content-Type", "application/json")

            resp, _ := app.Test(req)
            assert.Equal(t, tt.wantStatus, resp.StatusCode)
        })
    }
}
```

### Milestone Checklist - Phase 2

- [ ] `api/auth_handler.go` - 90% coverage
- [ ] `api/rest_crud.go` - 85% coverage
- [ ] `api/rest_handler.go` - 85% coverage
- [ ] `api/storage_files.go` - 85% coverage
- [ ] `api/dashboard_auth_handler.go` - 85% coverage
- [ ] `api/server.go` - 70% coverage
- [ ] `api/oauth_handler.go` - 85% coverage
- [ ] `api/storage_buckets.go` - 85% coverage
- [ ] `api/rest_batch.go` - 80% coverage
- [ ] Update `.testcoverage.yml` thresholds for api module

---

## Phase 3: Data Layer (Target: 80%)

### Priority: MEDIUM-HIGH

Data layer bugs can cause data loss or corruption.

### Files to Test

#### 3.1 Storage (`internal/storage/`)

| File | Current | Target | Statements | Priority |
|------|---------|--------|------------|----------|
| `service.go` | ~20% | 85% | 400 | **P0** |
| `log_postgres.go` | 0% | 80% | 200 | **P1** |
| `s3.go` | 0% | 75% | 186 | **P2** |
| `local.go` | ~30% | 80% | 150 | **P1** |

#### 3.2 Jobs (`internal/jobs/`)

| File | Current | Target | Statements | Priority |
|------|---------|--------|------------|----------|
| `storage.go` | 0% | 80% | 394 | **P0** |
| `worker.go` | 0% | 80% | 261 | **P0** |
| `manager.go` | ~5% | 80% | 350 | **P1** |
| `scheduler.go` | ~5% | 75% | 200 | **P1** |

#### 3.3 Database (`internal/database/`)

| File | Current | Target | Statements | Priority |
|------|---------|--------|------------|----------|
| `schema_inspector.go` | 0% | 75% | 250 | **P1** |
| `executor.go` | ~10% | 80% | 200 | **P1** |
| `migrations.go` | ~5% | 70% | 150 | **P2** |

### Testing Strategy for Phase 3

For database-dependent code, use the test database:

```go
// Use build tags for integration tests
// +build integration

func TestJobStorage_Integration(t *testing.T) {
    if testing.Short() {
        t.Skip("skipping integration test")
    }

    db := testutil.GetTestDB(t)
    storage := jobs.NewStorage(db)

    // Test CRUD operations
    job := &jobs.Job{
        Name: "test-job",
        // ...
    }

    err := storage.Create(context.Background(), job)
    require.NoError(t, err)

    retrieved, err := storage.GetByID(context.Background(), job.ID)
    require.NoError(t, err)
    assert.Equal(t, job.Name, retrieved.Name)
}
```

### Milestone Checklist - Phase 3

- [ ] `storage/service.go` - 85% coverage
- [ ] `jobs/storage.go` - 80% coverage
- [ ] `jobs/worker.go` - 80% coverage
- [ ] `storage/local.go` - 80% coverage
- [ ] `storage/log_postgres.go` - 80% coverage
- [ ] `jobs/manager.go` - 80% coverage
- [ ] `database/schema_inspector.go` - 75% coverage
- [ ] `database/executor.go` - 80% coverage

---

## Phase 4: Features (Target: 80%)

### Priority: MEDIUM

Feature modules that provide key functionality.

### Files to Test

#### 4.1 Functions (`internal/functions/`)

| File | Current | Target | Statements | Priority |
|------|---------|--------|------------|----------|
| `handler.go` | 0% | 80% | 759 | **P0** |
| `storage.go` | 0% | 80% | 277 | **P1** |
| `scheduler.go` | ~10% | 75% | 150 | **P2** |

#### 4.2 AI (`internal/ai/`)

| File | Current | Target | Statements | Priority |
|------|---------|--------|------------|----------|
| `handler.go` | 0% | 80% | 666 | **P0** |
| `storage.go` | 0% | 80% | 446 | **P1** |
| `knowledge_base_storage.go` | 0% | 80% | 438 | **P1** |
| `chat_handler.go` | 0% | 75% | 408 | **P1** |
| `knowledge_base_handler.go` | 0% | 75% | 374 | **P2** |
| `document_processor.go` | 0% | 70% | 230 | **P2** |
| `schema_builder.go` | 0% | 70% | 246 | **P2** |

#### 4.3 MCP (`internal/mcp/`)

| File | Current | Target | Statements | Priority |
|------|---------|--------|------------|----------|
| `server.go` | ~10% | 80% | 300 | **P0** |
| `handler.go` | ~10% | 80% | 250 | **P1** |
| `auth.go` | ~15% | 85% | 150 | **P1** |
| `tools/branching.go` | 0% | 75% | 220 | **P2** |
| `tools/query.go` | ~20% | 80% | 180 | **P2** |

#### 4.4 Realtime (`internal/realtime/`)

| File | Current | Target | Statements | Priority |
|------|---------|--------|------------|----------|
| `hub.go` | ~25% | 80% | 300 | **P1** |
| `client.go` | ~30% | 80% | 200 | **P1** |
| `broadcaster.go` | ~20% | 75% | 150 | **P2** |

### Milestone Checklist - Phase 4

- [ ] `functions/handler.go` - 80% coverage
- [ ] `ai/handler.go` - 80% coverage
- [ ] `mcp/server.go` - 80% coverage
- [ ] `ai/storage.go` - 80% coverage
- [ ] `ai/knowledge_base_storage.go` - 80% coverage
- [ ] `functions/storage.go` - 80% coverage
- [ ] `mcp/handler.go` - 80% coverage
- [ ] `mcp/auth.go` - 85% coverage
- [ ] `realtime/hub.go` - 80% coverage
- [ ] `realtime/client.go` - 80% coverage

---

## Phase 5: Supporting Modules (Target: 75%)

### Priority: MEDIUM

Supporting functionality that enables core features.

### Files to Test

#### 5.1 Branching (`internal/branching/`) - Currently 0%

| File | Current | Target | Statements | Priority |
|------|---------|--------|------------|----------|
| `manager.go` | 0% | 80% | 269 | **P0** |
| `storage.go` | 0% | 80% | 304 | **P0** |
| `router.go` | 0% | 75% | 150 | **P1** |

#### 5.2 RPC (`internal/rpc/`)

| File | Current | Target | Statements | Priority |
|------|---------|--------|------------|----------|
| `handler.go` | 0% | 80% | 351 | **P0** |
| `executor.go` | 0% | 80% | 220 | **P1** |

#### 5.3 Email (`internal/email/`)

| File | Current | Target | Statements | Priority |
|------|---------|--------|------------|----------|
| `service.go` | ~20% | 80% | 150 | **P1** |
| `templates.go` | ~15% | 75% | 100 | **P2** |

#### 5.4 Config (`internal/config/`)

| File | Current | Target | Statements | Priority |
|------|---------|--------|------------|----------|
| `config.go` | ~35% | 80% | 400 | **P1** |
| `validation.go` | ~30% | 85% | 150 | **P1** |

### Milestone Checklist - Phase 5

- [ ] `branching/manager.go` - 80% coverage
- [ ] `branching/storage.go` - 80% coverage
- [ ] `rpc/handler.go` - 80% coverage
- [ ] `rpc/executor.go` - 80% coverage
- [ ] `email/service.go` - 80% coverage
- [ ] `config/config.go` - 80% coverage
- [ ] `branching/router.go` - 75% coverage

---

## Phase 6: Polish & Edge Cases (Target: 80% overall)

### Priority: LOW

Final pass to reach 80% overall coverage.

### Focus Areas

1. **Error paths** - Ensure error handling branches are tested
2. **Edge cases** - Boundary conditions, empty inputs, large inputs
3. **Concurrency** - Race condition tests (use `-race` flag)
4. **Timeouts** - Context cancellation handling
5. **Cleanup** - Resource cleanup on errors

### Remaining Files

Any files not yet at target after phases 1-5.

---

## Test Patterns & Best Practices

### 1. Table-Driven Tests (Preferred)

```go
func TestFunction(t *testing.T) {
    tests := []struct {
        name    string
        input   InputType
        want    OutputType
        wantErr bool
    }{
        {"valid input", validInput, expectedOutput, false},
        {"invalid input", invalidInput, nil, true},
        {"edge case", edgeInput, edgeOutput, false},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got, err := Function(tt.input)

            if tt.wantErr {
                require.Error(t, err)
                return
            }
            require.NoError(t, err)
            assert.Equal(t, tt.want, got)
        })
    }
}
```

### 2. Test Naming Convention

```
TestFunctionName_Scenario_ExpectedBehavior

Examples:
- TestLogin_ValidCredentials_ReturnsToken
- TestLogin_InvalidPassword_ReturnsUnauthorized
- TestLogin_LockedAccount_ReturnsLocked
```

### 3. Assertion Library

Use `testify` consistently:
- `require.*` for fatal assertions (test should stop)
- `assert.*` for non-fatal assertions (test continues)

```go
import (
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

// Use require for setup/preconditions
require.NoError(t, err, "failed to create user")

// Use assert for actual test assertions
assert.Equal(t, expected, actual)
assert.Len(t, slice, 3)
assert.Contains(t, str, "expected substring")
```

### 4. Mock Patterns

```go
// Interface-based mocking
type MockService struct {
    // Return values
    LoginResult *TokenPair
    LoginError  error

    // Call tracking
    LoginCalls []LoginCall
}

type LoginCall struct {
    Email    string
    Password string
}

func (m *MockService) Login(email, password string) (*TokenPair, error) {
    m.LoginCalls = append(m.LoginCalls, LoginCall{email, password})
    return m.LoginResult, m.LoginError
}
```

### 5. HTTP Handler Testing

```go
func TestHandler(t *testing.T) {
    app := fiber.New()
    handler := NewHandler(mockDeps)
    app.Post("/endpoint", handler.Method)

    req := httptest.NewRequest("POST", "/endpoint", strings.NewReader(body))
    req.Header.Set("Content-Type", "application/json")
    req.Header.Set("Authorization", "Bearer "+token)

    resp, err := app.Test(req)
    require.NoError(t, err)

    assert.Equal(t, http.StatusOK, resp.StatusCode)

    var result ResponseType
    err = json.NewDecoder(resp.Body).Decode(&result)
    require.NoError(t, err)
    assert.Equal(t, expected, result)
}
```

### 6. Database Integration Tests

```go
// +build integration

func TestStorage_Integration(t *testing.T) {
    if testing.Short() {
        t.Skip("skipping integration test in short mode")
    }

    // Use test database from devcontainer
    db := testutil.GetTestDB(t)
    t.Cleanup(func() {
        testutil.CleanupTestDB(t, db)
    })

    storage := NewStorage(db)
    // ... test cases
}
```

---

## Configuration Updates

### Update `.testcoverage.yml` Incrementally

After each phase, update thresholds:

```yaml
# After Phase 1
threshold:
  file: 0
  package: 0
  total: 25  # Increased from 10

override:
  - path: ^internal/auth/
    threshold: 80
  - path: ^internal/middleware/
    threshold: 80

# After Phase 2
threshold:
  total: 45

override:
  - path: ^internal/auth/
    threshold: 85
  - path: ^internal/api/
    threshold: 75

# ... continue incrementally
```

---

## Progress Tracking

### Metrics to Track

1. **Overall coverage %** - Primary metric
2. **Coverage by module** - Ensure balanced improvement
3. **Zero-coverage files remaining** - Should decrease to 0
4. **Test count** - Track new tests added
5. **Test execution time** - Keep under 5 minutes for unit tests

### Weekly Review Checklist

- [ ] Run `make test-coverage` and record metrics
- [ ] Update `.testcoverage.yml` thresholds if targets met
- [ ] Review any flaky tests
- [ ] Identify blockers for next week's targets

---

## Appendix A: Files Excluded from Coverage

These files are intentionally excluded (per `.testcoverage.yml`):

| Pattern | Reason |
|---------|--------|
| `internal/query/types.go` | Pure type definitions |
| `internal/*/types.go` | Pure type definitions |
| `internal/*/errors.go` | Error constant definitions |
| `internal/auth/interfaces.go` | Interface definitions |
| `internal/api/openapi.go` | Generated specification |
| `internal/scaling/leader.go` | Requires distributed system |
| `internal/database/connection.go` | Requires live database |
| `internal/ai/ocr_tesseract.go` | Requires Tesseract binary |
| `cmd/fluxbase/main.go` | Entry point |
| `internal/adminui/*` | Embedded UI assets |
| `cli/*` | Tested via integration tests |
| `internal/testutil/*` | Test utilities |

---

## Appendix B: Existing Test Templates

Use these well-tested files as templates:

| File | Coverage | Good For |
|------|----------|----------|
| `internal/auth/jwt_test.go` | 78.8% | Token management, table-driven tests |
| `internal/realtime/filter_test.go` | 100% | Parser testing, edge cases, benchmarks |
| `internal/auth/validation_test.go` | 98.6% | Input validation |
| `internal/auth/password_test.go` | 97.4% | Security functions |
| `internal/logging/service_test.go` | 87% | Service layer testing |

---

## Appendix C: New Mocks to Create

| Mock | Location | Used By |
|------|----------|---------|
| `MockAuthService` | `internal/auth/mocks_test.go` | auth tests |
| `MockUserRepository` | `internal/auth/mocks_test.go` | auth tests |
| `MockSessionRepository` | `internal/auth/mocks_test.go` | auth tests |
| `MockJobStorage` | `internal/jobs/mocks_test.go` | jobs tests |
| `MockFunctionStorage` | `internal/functions/mocks_test.go` | functions tests |
| `MockBranchManager` | `internal/branching/mocks_test.go` | branching tests |
| `MockAIService` | `internal/ai/mocks_test.go` | AI tests |
| `MockDatabaseExecutor` | `internal/database/mocks_test.go` | database tests |

---

## Getting Started

1. **Start with Phase 1** - Security is the priority
2. **Pick one file at a time** - Focus on completeness
3. **Use existing tests as templates** - Copy patterns from `jwt_test.go`
4. **Run tests frequently** - `go test -v -cover ./internal/auth/...`
5. **Update thresholds incrementally** - Don't wait until the end
6. **Ask for help** - Some tests may need architectural discussion

Good luck!
