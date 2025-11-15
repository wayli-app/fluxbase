# Admin Setup Example

This example demonstrates how to use all the advanced admin features in Fluxbase SDK including:

- Admin authentication
- User management
- OAuth provider configuration
- Settings management (system and app)
- DDL operations for multi-tenancy
- API keys and webhooks
- User impersonation for debugging

## Prerequisites

- Node.js 18+ installed
- Fluxbase server running (default: http://localhost:8080)
- Admin account created (via `fluxbase admin setup`)

## Installation

```bash
npm install
```

## Configuration

Create a `.env` file:

```env
FLUXBASE_BASE_URL=http://localhost:8080
ADMIN_EMAIL=admin@example.com
ADMIN_PASSWORD=your-admin-password

# Optional: OAuth provider credentials
GITHUB_CLIENT_ID=your-github-client-id
GITHUB_CLIENT_SECRET=your-github-client-secret
GOOGLE_CLIENT_ID=your-google-client-id
GOOGLE_CLIENT_SECRET=your-google-client-secret
```

## Running the Examples

### Complete Admin Setup

Run the full admin setup script that configures everything:

```bash
npm run setup
```

This will:
1. Authenticate as admin
2. Configure OAuth providers (GitHub, Google)
3. Update authentication settings (password requirements, session timeout)
4. Configure app settings (enable features, rate limiting)
5. Create multi-tenant database schemas
6. Generate API keys for services
7. Set up webhooks for events
8. Store custom configuration

### Individual Examples

Run specific examples:

```bash
# User management
npm run example:users

# OAuth configuration
npm run example:oauth

# Settings management
npm run example:settings

# DDL operations (multi-tenancy)
npm run example:ddl

# API keys and webhooks
npm run example:management

# User impersonation
npm run example:impersonation

# Complete workflow
npm run example:workflow
```

## Examples Overview

### 1. Admin Authentication (`src/01-auth.ts`)

Demonstrates admin login and token management.

### 2. User Management (`src/02-users.ts`)

Shows how to:
- List users with pagination and filtering
- Invite new users
- Update user roles
- Delete users
- Reset passwords

### 3. OAuth Configuration (`src/03-oauth.ts`)

Demonstrates:
- Creating OAuth providers (GitHub, Google, custom)
- Updating provider settings
- Enabling/disabling providers
- Configuring authentication settings (password policies, sessions)
- Rotating credentials

### 4. Settings Management (`src/04-settings.ts`)

Shows how to:
- Store custom configuration in system settings
- Update app settings (features, security, email)
- Create feature flag systems
- Environment-specific configuration

### 5. DDL Operations (`src/05-ddl.ts`)

Demonstrates:
- Creating database schemas for multi-tenancy
- Creating tables with column definitions
- Listing schemas and tables
- Building multi-tenant architectures

### 6. API Keys & Webhooks (`src/06-management.ts`)

Shows how to:
- Generate API keys for backend services
- Set expiration dates
- Revoke keys
- Create webhooks for database events
- View webhook delivery history
- Retry failed deliveries

### 7. User Impersonation (`src/07-impersonation.ts`)

Demonstrates:
- Impersonating users for debugging
- Testing RLS policies as different users
- Impersonating anonymous users
- Using service role for admin operations
- Viewing impersonation audit trail

### 8. Complete Workflow (`src/08-complete-workflow.ts`)

End-to-end example showing:
1. Admin authentication
2. Creating a new tenant
3. Setting up OAuth for tenant
4. Configuring tenant settings
5. Creating tenant schema and tables
6. Generating tenant API key
7. Setting up tenant webhooks
8. Testing with user impersonation

## File Structure

```
admin-setup/
├── README.md
├── package.json
├── tsconfig.json
├── .env.example
└── src/
    ├── 01-auth.ts
    ├── 02-users.ts
    ├── 03-oauth.ts
    ├── 04-settings.ts
    ├── 05-ddl.ts
    ├── 06-management.ts
    ├── 07-impersonation.ts
    ├── 08-complete-workflow.ts
    └── utils/
        ├── client.ts
        └── logger.ts
```

## Key Concepts

### Authentication

Fluxbase uses JWT-based authentication with separate admin and user contexts:

```typescript
// Admin authentication
await client.admin.login({
  email: 'admin@example.com',
  password: 'password'
})

// Now you can use admin features
await client.admin.listUsers()
```

### Row Level Security (RLS)

Test RLS policies by impersonating users:

```typescript
// Start impersonation
await client.admin.impersonation.impersonateUser({
  target_user_id: 'user-uuid',
  reason: 'Testing RLS policies'
})

// Now queries run with user's permissions
const userView = await client.from('documents').select('*')

// Stop impersonation
await client.admin.impersonation.stop()
```

### Multi-Tenancy

Create isolated schemas for each tenant:

```typescript
// Create tenant schema
await client.admin.ddl.createSchema('tenant_acme')

// Create tenant tables
await client.admin.ddl.createTable('tenant_acme', 'users', [
  { name: 'id', type: 'UUID', primaryKey: true },
  { name: 'email', type: 'TEXT', nullable: false }
])
```

### Webhooks

Get notified of database events:

```typescript
await client.admin.management.webhooks.create({
  name: 'User Events',
  url: 'https://api.example.com/webhook',
  events: ['INSERT', 'UPDATE', 'DELETE'],
  table: 'users',
  schema: 'public',
  enabled: true,
  secret: 'webhook-secret'
})
```

## Best Practices

1. **Secure Credentials**: Always use environment variables, never hardcode secrets
2. **Rotate Keys**: Regularly rotate API keys and OAuth credentials
3. **Audit Trail**: Review impersonation sessions regularly
4. **Strong Passwords**: Enforce strong password policies in production
5. **Rate Limiting**: Enable rate limiting to prevent abuse
6. **Error Handling**: Always wrap API calls in try-catch blocks
7. **Session Management**: Set appropriate session timeouts

## Troubleshooting

### Authentication Errors

If you get 401 errors:
```bash
# Verify admin account
fluxbase admin setup

# Check credentials in .env file
cat .env
```

### Connection Errors

If you can't connect to Fluxbase:
```bash
# Check if server is running
curl http://localhost:8080/api/v1/health

# Verify FLUXBASE_BASE_URL in .env
```

### Permission Errors

If you get 403 errors:
```bash
# Ensure you're authenticated as admin
# Admin operations require admin role
```

## Testing

This project includes comprehensive integration tests that verify all admin operations.

### Running Tests

```bash
# All tests
npm test

# Integration tests only
npm run test:integration

# Watch mode
npm run test:watch

# With coverage
npm run test:coverage
```

### Test Coverage

The test suite covers:
- ✓ Admin authentication and session management
- ✓ User CRUD operations and role management
- ✓ OAuth provider configuration
- ✓ System and application settings
- ✓ Database schema and table creation (DDL)
- ✓ User impersonation and audit trail
- ✓ Complete admin workflows

See [test/README.md](./test/README.md) for detailed testing documentation.

## Learn More

- [Admin SDK Documentation](https://docs.fluxbase.sh/sdk/admin)
- [Management SDK Documentation](https://docs.fluxbase.sh/sdk/management)
- [Settings SDK Documentation](https://docs.fluxbase.sh/sdk/settings)
- [DDL SDK Documentation](https://docs.fluxbase.sh/sdk/ddl)
- [OAuth SDK Documentation](https://docs.fluxbase.sh/sdk/oauth)
- [Impersonation SDK Documentation](https://docs.fluxbase.sh/sdk/impersonation)
- [Advanced Features Overview](https://docs.fluxbase.sh/sdk/advanced-features)

## License

MIT
