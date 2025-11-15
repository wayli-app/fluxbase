# Admin Setup Integration Tests

Comprehensive integration tests for the Fluxbase admin setup examples.

## Overview

These tests verify that all admin example scripts work correctly against a real Fluxbase instance. They provide:

- **End-to-end validation** of admin operations
- **Integration testing** with real database operations
- **Regression prevention** for admin features
- **Documentation** through executable tests

## Prerequisites

Before running the tests, ensure you have:

1. **Fluxbase server running** on `localhost:8080` (or configure `FLUXBASE_BASE_URL`)
2. **Database initialized** with migrations applied
3. **Admin credentials** configured in `.env` file
4. **Test dependencies installed**: `npm install`

## Setup

### 1. Create `.env` file

```bash
cp .env.example .env
```

Edit `.env` with your admin credentials:

```env
FLUXBASE_BASE_URL=http://localhost:8080
ADMIN_EMAIL=admin@fluxbase.local
ADMIN_PASSWORD=your-admin-password
```

### 2. Install Dependencies

```bash
npm install
```

### 3. Start Fluxbase Server

```bash
# In workspace root
make dev
```

## Running Tests

### All Tests

```bash
npm test
```

### Integration Tests Only

```bash
npm run test:integration
```

### Watch Mode (Development)

```bash
npm run test:watch
```

### With Coverage

```bash
npm run test:coverage
```

## Test Structure

The integration test suite is organized into test suites matching the admin examples:

### 01: Admin Authentication

Tests admin login, logout, token management, and credential validation.

**Coverage:**

- ✓ Admin login with valid credentials
- ✓ Get admin info
- ✓ Reject invalid credentials
- ✓ Admin logout

### 02: User Management

Tests user CRUD operations, role management, and search functionality.

**Coverage:**

- ✓ List users with pagination
- ✓ Invite new users
- ✓ Get user by ID
- ✓ Update user roles
- ✓ Search users by email
- ✓ Reset user passwords
- ✓ Delete users

### 03: OAuth Configuration

Tests OAuth provider management and authentication settings.

**Coverage:**

- ✓ List OAuth providers
- ✓ Get authentication settings
- ✓ Update authentication settings

### 04: Settings Management

Tests system settings (key-value storage) and application settings.

**Coverage:**

- ✓ List system settings
- ✓ Create/update system settings
- ✓ Get specific settings
- ✓ Delete system settings
- ✓ Get application settings
- ✓ Update application settings

### 05: DDL Operations

Tests database schema and table creation.

**Coverage:**

- ✓ List schemas
- ✓ Create schema
- ✓ Create table with columns
- ✓ List tables in schema
- ✓ Drop schema

### 06: Impersonation

Tests user impersonation, anonymous access, and audit trail.

**Coverage:**

- ✓ Check impersonation status
- ✓ Impersonate specific user
- ✓ Stop impersonation
- ✓ Impersonate anonymous user
- ✓ Impersonate with service role
- ✓ List impersonation sessions
- ✓ Filter sessions by type

### Complete Workflow

Tests a full admin workflow combining multiple features.

**Coverage:**

- ✓ End-to-end admin workflow
- ✓ Schema creation
- ✓ Table creation
- ✓ User management
- ✓ Settings configuration
- ✓ Impersonation
- ✓ Cleanup

## Test Configuration

### Timeouts

Tests have a 30-second timeout per test case:

```typescript
testTimeout: 30000;
hookTimeout: 30000;
```

### Environment Variables

| Variable              | Default                 | Description         |
| --------------------- | ----------------------- | ------------------- |
| `FLUXBASE_BASE_URL`   | `http://localhost:8080` | Fluxbase server URL |
| `ADMIN_EMAIL`         | `admin@fluxbase.local`  | Admin email         |
| `ADMIN_PASSWORD`      | `password`              | Admin password      |

### Test Data

Tests create temporary data with timestamps to avoid conflicts:

- Test users: `test-{timestamp}@example.com`
- Test schemas: `test_schema_{timestamp}`
- Test settings: `test.setting.{timestamp}`

All test data is cleaned up after test completion.

## Debugging Tests

### Enable Verbose Output

```bash
npm test -- --reporter=verbose
```

### Run Specific Test

```bash
npm test -- -t "should authenticate admin"
```

### Run Single Test Suite

```bash
npm test -- test/integration.test.ts -t "Admin Authentication"
```

### Check Test Coverage

```bash
npm run test:coverage
open coverage/index.html
```

## Continuous Integration

### GitHub Actions

Example workflow:

```yaml
name: Integration Tests

on: [push, pull_request]

jobs:
  test:
    runs-on: ubuntu-latest
    services:
      postgres:
        image: postgres:15
        env:
          POSTGRES_PASSWORD: postgres
        options: >-
          --health-cmd pg_isready
          --health-interval 10s
          --health-timeout 5s
          --health-retries 5

    steps:
      - uses: actions/checkout@v5
      - uses: actions/setup-node@v3
        with:
          node-version: "20"

      - name: Install dependencies
        run: npm install
        working-directory: ./examples/admin-setup

      - name: Start Fluxbase
        run: make dev &

      - name: Wait for server
        run: npx wait-on http://localhost:8080/health

      - name: Run integration tests
        run: npm run test:integration
        working-directory: ./examples/admin-setup
```

## Troubleshooting

### Tests Failing to Connect

**Problem:** Cannot connect to Fluxbase server

**Solution:**

1. Verify server is running: `curl http://localhost:8080/health`
2. Check `FLUXBASE_BASE_URL` in `.env`
3. Ensure no firewall blocking port 8080

### Authentication Errors

**Problem:** Admin login fails

**Solution:**

1. Verify admin credentials in `.env`
2. Reset admin password if needed
3. Check server logs for authentication errors

### Permission Errors

**Problem:** Tests fail with permission denied

**Solution:**

1. Ensure user is actually an admin
2. Check RLS policies aren't blocking operations
3. Verify admin token is valid

### Cleanup Failures

**Problem:** Test data not cleaned up

**Solution:**

1. Tests cleanup in `afterAll` hooks
2. Manually clean test data: `npm run example:users`
3. Check for orphaned test schemas/users

## Best Practices

### Writing New Tests

1. **Use descriptive test names**

   ```typescript
   it("should create user with admin role", async () => {
     // Test implementation
   });
   ```

2. **Clean up test data**

   ```typescript
   afterAll(async () => {
     await client.admin.deleteUser(testUserId);
   });
   ```

3. **Use timestamps for unique data**

   ```typescript
   const email = `test-${Date.now()}@example.com`;
   ```

4. **Test both success and failure cases**
   ```typescript
   await expect(
     client.admin.login({ email, password: "wrong" })
   ).rejects.toThrow();
   ```

### Test Organization

- Group related tests in `describe` blocks
- Use `beforeAll` for test setup
- Use `afterAll` for cleanup
- Keep tests independent and idempotent

### Performance

- Run tests in parallel when possible
- Use `beforeAll` for expensive setup
- Clean up only what you created
- Avoid unnecessary API calls

## Contributing

When adding new admin features:

1. Add corresponding tests to `integration.test.ts`
2. Update this README with new test coverage
3. Ensure tests pass before submitting PR
4. Add any new environment variables to `.env.example`

## License

MIT
