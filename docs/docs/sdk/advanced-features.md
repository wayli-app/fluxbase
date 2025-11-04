---
sidebar_position: 6
---

# Advanced Features Overview

This guide provides an overview of all advanced SDK features for admin operations, covering the complete suite of management and configuration tools available in Fluxbase.

## Feature Categories

### 1. Admin Management
Core admin authentication and user management capabilities.

**Available Features:**
- Admin authentication (login, setup, token refresh)
- User management (list, invite, delete, role updates)
- Password resets

**Learn More:** [Admin SDK](/docs/sdk/admin)

---

### 2. API Keys & Webhooks
Programmatic access and event-driven integrations.

**API Keys:**
- Generate service keys for backend integrations
- Set expiration dates and permissions
- Revoke keys when needed

**Webhooks:**
- Subscribe to database events (INSERT, UPDATE, DELETE)
- Filter by table and schema
- View delivery history and retry failed deliveries

**Learn More:** [Management SDK](/docs/sdk/management)

---

### 3. Settings Management
Configure application behavior and store custom configuration.

**System Settings:**
- Key-value storage for custom configuration
- JSON value support for complex data
- Flexible schema for any configuration needs

**Application Settings:**
- Authentication configuration
- Feature toggles (realtime, storage, functions)
- Email service settings
- Security and rate limiting options

**Learn More:** [Settings SDK](/docs/sdk/settings)

---

### 4. Database Schema Management (DDL)
Programmatic database schema and table creation.

**Capabilities:**
- Create and list database schemas
- Create tables with full column definitions
- Support for all PostgreSQL data types
- Primary keys, defaults, and constraints

**Use Cases:**
- Multi-tenant architecture
- Dynamic table generation
- Database migrations
- Test data setup

**Learn More:** [DDL SDK](/docs/sdk/ddl)

---

### 5. OAuth Provider Configuration
Manage authentication providers and auth settings.

**OAuth Providers:**
- Configure built-in providers (GitHub, Google, GitLab, etc.)
- Create custom OAuth2 providers
- Enable/disable providers
- Update credentials and scopes

**Auth Settings:**
- Password complexity requirements
- Session timeout configuration
- Email verification settings
- Magic link authentication

**Learn More:** [OAuth SDK](/docs/sdk/oauth)

---

### 6. User Impersonation
Debug issues and test RLS policies by viewing data as different users.

**Impersonation Types:**
- User impersonation (see data as specific user)
- Anonymous impersonation (test public access)
- Service role impersonation (administrative operations)

**Features:**
- Complete audit trail
- Session metadata tracking
- Required reason field for accountability

**Use Cases:**
- Debugging user-reported issues
- Testing Row Level Security policies
- Customer support investigations
- Verifying public data access

**Learn More:** [Impersonation SDK](/docs/sdk/impersonation)

---

## Complete Example: Admin Dashboard Setup

Here's a complete example showing how to use multiple advanced features together to set up an admin dashboard:

```typescript
import { createClient } from '@fluxbase/sdk'

const client = createClient({
  url: 'http://localhost:8080'
})

async function setupAdminDashboard() {
  // 1. Admin Authentication
  await client.admin.login({
    email: 'admin@example.com',
    password: 'secure-password'
  })

  // 2. Configure Authentication Settings
  await client.admin.oauth.authSettings.update({
    password_min_length: 16,
    password_require_uppercase: true,
    password_require_number: true,
    password_require_special: true,
    session_timeout_minutes: 240,
    require_email_verification: true
  })

  // 3. Set up OAuth Provider
  await client.admin.oauth.providers.createProvider({
    provider_name: 'github',
    display_name: 'GitHub',
    enabled: true,
    client_id: process.env.GITHUB_CLIENT_ID!,
    client_secret: process.env.GITHUB_CLIENT_SECRET!,
    redirect_url: 'https://app.example.com/auth/callback',
    scopes: ['user:email', 'read:user'],
    is_custom: false
  })

  // 4. Configure Application Settings
  await client.admin.settings.app.update({
    features: {
      enable_realtime: true,
      enable_storage: true,
      enable_functions: false
    },
    security: {
      enable_rate_limiting: true,
      rate_limit_requests_per_minute: 60
    }
  })

  // 5. Create Multi-tenant Database Schema
  await client.admin.ddl.createSchema('tenant_acme')
  await client.admin.ddl.createTable('tenant_acme', 'users', [
    { name: 'id', type: 'UUID', primaryKey: true, defaultValue: 'gen_random_uuid()' },
    { name: 'email', type: 'CITEXT', nullable: false },
    { name: 'name', type: 'TEXT', nullable: false },
    { name: 'created_at', type: 'TIMESTAMPTZ', defaultValue: 'NOW()' }
  ])

  // 6. Create API Key for Backend Service
  const { key } = await client.admin.management.apiKeys.create({
    name: 'Backend Service Key',
    description: 'API key for internal backend service',
    expires_at: new Date(Date.now() + 365 * 24 * 60 * 60 * 1000).toISOString() // 1 year
  })

  console.log('Service API Key:', key.key)

  // 7. Set up Webhook for User Events
  await client.admin.management.webhooks.create({
    name: 'User Events Webhook',
    url: 'https://api.example.com/webhooks/users',
    events: ['INSERT', 'UPDATE', 'DELETE'],
    table: 'users',
    schema: 'public',
    enabled: true,
    secret: 'webhook-secret-key'
  })

  // 8. Store Custom Configuration
  await client.admin.settings.system.update('app.feature_flags', {
    value: {
      beta_features: true,
      new_ui: false,
      advanced_analytics: true
    },
    description: 'Feature flags for gradual rollout'
  })

  console.log('Admin dashboard setup complete!')
}

// Execute setup
setupAdminDashboard().catch(console.error)
```

---

## Security Best Practices

### 1. API Key Management

```typescript
// Rotate API keys regularly
async function rotateAPIKey(oldKeyId: string) {
  // Create new key
  const { key: newKey } = await client.admin.management.apiKeys.create({
    name: 'Service Key (Rotated)',
    description: 'Rotated API key',
    expires_at: new Date(Date.now() + 90 * 24 * 60 * 60 * 1000).toISOString()
  })

  // Update your service with new key
  await updateServiceConfiguration(newKey.key)

  // Revoke old key
  await client.admin.management.apiKeys.revoke(oldKeyId)

  return newKey
}
```

### 2. Webhook Security

```typescript
// Always use secrets to verify webhook authenticity
await client.admin.management.webhooks.create({
  name: 'Secure Webhook',
  url: 'https://api.example.com/webhook',
  events: ['INSERT'],
  table: 'orders',
  enabled: true,
  secret: crypto.randomBytes(32).toString('hex') // Strong secret
})
```

### 3. Impersonation Audit

```typescript
// Regularly review impersonation sessions
async function auditImpersonation() {
  const { sessions } = await client.admin.impersonation.listSessions({
    limit: 100
  })

  // Check for suspicious patterns
  const sessionsByAdmin = new Map<string, number>()

  sessions.forEach(session => {
    const count = sessionsByAdmin.get(session.admin_user_id) || 0
    sessionsByAdmin.set(session.admin_user_id, count + 1)
  })

  // Alert on excessive usage
  sessionsByAdmin.forEach((count, adminId) => {
    if (count > 50) {
      console.warn(`Admin ${adminId} has ${count} impersonation sessions`)
    }
  })
}
```

### 4. Password Policy Enforcement

```typescript
// Enforce strong password policies
await client.admin.oauth.authSettings.update({
  password_min_length: 16,
  password_require_uppercase: true,
  password_require_lowercase: true,
  password_require_number: true,
  password_require_special: true,
  session_timeout_minutes: 120, // 2 hours
  max_sessions_per_user: 3
})
```

---

## Common Patterns

### Pattern 1: Multi-Tenant Setup

```typescript
async function createTenant(tenantName: string) {
  const schemaName = `tenant_${tenantName.toLowerCase()}`

  // Create schema
  await client.admin.ddl.createSchema(schemaName)

  // Create tenant tables
  const tables = ['users', 'products', 'orders']

  for (const table of tables) {
    await client.admin.ddl.createTable(schemaName, table, [
      { name: 'id', type: 'UUID', primaryKey: true, defaultValue: 'gen_random_uuid()' },
      { name: 'created_at', type: 'TIMESTAMPTZ', defaultValue: 'NOW()' },
      { name: 'updated_at', type: 'TIMESTAMPTZ', defaultValue: 'NOW()' }
    ])
  }

  // Store tenant configuration
  await client.admin.settings.system.update(`tenant.${tenantName}.config`, {
    value: {
      name: tenantName,
      schema: schemaName,
      created_at: new Date().toISOString()
    },
    description: `Configuration for tenant ${tenantName}`
  })
}
```

### Pattern 2: Feature Flag System

```typescript
async function manageFeatureFlags() {
  // Store feature flags
  await client.admin.settings.system.update('features.flags', {
    value: {
      'new-dashboard': { enabled: true, rollout: 100 },
      'beta-api': { enabled: true, rollout: 10 },
      'experimental-ui': { enabled: false, rollout: 0 }
    },
    description: 'Feature flags for gradual rollout'
  })

  // Check if feature is enabled
  const flags = await client.admin.settings.system.get('features.flags')
  const betaAPI = flags.value['beta-api']

  if (betaAPI.enabled && Math.random() * 100 < betaAPI.rollout) {
    // Use new API
  }
}
```

### Pattern 3: Webhook Event Processing

```typescript
async function setupEventProcessing() {
  // Create webhook for order events
  await client.admin.management.webhooks.create({
    name: 'Order Processing',
    url: 'https://api.example.com/process-order',
    events: ['INSERT'],
    table: 'orders',
    enabled: true,
    secret: process.env.WEBHOOK_SECRET!
  })

  // Monitor webhook deliveries
  const webhooks = await client.admin.management.webhooks.list()
  const orderWebhook = webhooks.webhooks.find(w => w.name === 'Order Processing')

  if (orderWebhook) {
    const deliveries = await client.admin.management.webhooks.getDeliveries(
      orderWebhook.id,
      { limit: 20 }
    )

    // Retry failed deliveries
    for (const delivery of deliveries.deliveries) {
      if (delivery.status === 'failed') {
        await client.admin.management.webhooks.retryDelivery(
          orderWebhook.id,
          delivery.id
        )
      }
    }
  }
}
```

### Pattern 4: User Debugging Workflow

```typescript
async function debugUserIssue(userEmail: string, ticketId: string) {
  try {
    // Find user
    const { users } = await client.admin.listUsers({
      search: userEmail,
      limit: 1
    })

    if (users.length === 0) {
      throw new Error(`User ${userEmail} not found`)
    }

    const user = users[0]

    // Start impersonation
    console.log(`Starting impersonation of ${user.email}`)
    await client.admin.impersonation.impersonateUser({
      target_user_id: user.id,
      reason: `Support ticket #${ticketId}`
    })

    // Query data as the user would see it
    const userDocuments = await client
      .from('documents')
      .select('*')
      .execute()

    console.log('User can see documents:', userDocuments.data?.length)

    // Check permissions
    const userPermissions = await client
      .from('user_permissions')
      .select('*')
      .execute()

    console.log('User permissions:', userPermissions.data)

  } finally {
    // Always stop impersonation
    await client.admin.impersonation.stop()
    console.log('Impersonation ended')
  }
}
```

---

## Performance Considerations

### 1. Connection Pooling

The SDK automatically manages connections, but for high-traffic scenarios:

```typescript
// Reuse client instances
const client = createClient({ url: 'http://localhost:8080' })

// Don't create new clients for each request
// ❌ Bad
function handleRequest() {
  const client = createClient({ url: 'http://localhost:8080' })
  // ...
}

// ✅ Good
const globalClient = createClient({ url: 'http://localhost:8080' })
function handleRequest() {
  // Use globalClient
}
```

### 2. Batch Operations

When creating multiple resources, batch them:

```typescript
// Create multiple API keys at once
const apiKeys = await Promise.all([
  client.admin.management.apiKeys.create({ name: 'Service A' }),
  client.admin.management.apiKeys.create({ name: 'Service B' }),
  client.admin.management.apiKeys.create({ name: 'Service C' })
])
```

### 3. Pagination

Always use pagination for large datasets:

```typescript
// Good: Paginate through users
let offset = 0
const limit = 100

while (true) {
  const { users, total } = await client.admin.listUsers({
    limit,
    offset
  })

  // Process users
  users.forEach(processUser)

  offset += limit
  if (offset >= total) break
}
```

---

## Error Handling

### Comprehensive Error Handling

```typescript
import { FluxbaseError } from '@fluxbase/sdk'

async function robustAdminOperation() {
  try {
    // Attempt operation
    await client.admin.settings.app.update({
      features: { enable_realtime: true }
    })

  } catch (error) {
    if (error instanceof FluxbaseError) {
      // Handle Fluxbase-specific errors
      console.error('Fluxbase Error:', {
        code: error.code,
        message: error.message,
        status: error.status
      })

      // Specific error handling
      if (error.status === 401) {
        // Re-authenticate
        await client.admin.login(credentials)
        // Retry operation
      } else if (error.status === 429) {
        // Rate limited, wait and retry
        await sleep(1000)
      }
    } else {
      // Handle unexpected errors
      console.error('Unexpected error:', error)
    }
  }
}
```

---

## Testing Advanced Features

### Unit Testing with Mocks

```typescript
import { vi } from 'vitest'
import { createClient } from '@fluxbase/sdk'

describe('Admin Operations', () => {
  it('should set up OAuth provider', async () => {
    const client = createClient({ url: 'http://localhost:8080' })

    // Mock the API call
    vi.spyOn(client.admin.oauth.providers, 'createProvider')
      .mockResolvedValue({
        success: true,
        id: 'provider-123',
        provider: 'github',
        message: 'Provider created'
      })

    const result = await client.admin.oauth.providers.createProvider({
      provider_name: 'github',
      // ... other fields
    })

    expect(result.success).toBe(true)
  })
})
```

---

## Migration Guide

### Migrating from Direct API Calls

If you're currently using direct API calls, migrating to the SDK is straightforward:

```typescript
// Before: Direct API calls
const response = await fetch('http://localhost:8080/api/v1/admin/settings/app', {
  method: 'PUT',
  headers: {
    'Content-Type': 'application/json',
    'Authorization': `Bearer ${token}`
  },
  body: JSON.stringify({
    features: { enable_realtime: true }
  })
})
const data = await response.json()

// After: Using SDK
const client = createClient({ url: 'http://localhost:8080' })
await client.admin.setToken(token)
await client.admin.settings.app.update({
  features: { enable_realtime: true }
})
```

**Benefits:**
- Type safety
- Automatic error handling
- Built-in retry logic
- Better developer experience

---

## Related Resources

- [Admin SDK](/docs/sdk/admin) - Admin authentication and user management
- [Management SDK](/docs/sdk/management) - API keys, webhooks, and invitations
- [Settings SDK](/docs/sdk/settings) - Application and system configuration
- [DDL SDK](/docs/sdk/ddl) - Database schema operations
- [OAuth SDK](/docs/sdk/oauth) - Authentication provider configuration
- [Impersonation SDK](/docs/sdk/impersonation) - User impersonation and debugging
- [TypeScript SDK Guide](/docs/guides/typescript-sdk) - General SDK usage
- [API Cookbook](/docs/api-cookbook) - Common API patterns
