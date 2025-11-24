# Migrations API Setup Guide

This guide explains how to securely configure and use the Fluxbase migrations API for managing database migrations from your application.

## Overview

The migrations API allows your application to submit and execute database migrations through Fluxbase, eliminating the need for direct database access with admin credentials.

## Security Architecture

The migrations API implements **6 layers of security**:

1. **Feature Flag** - API is enabled by default (can be disabled with `FLUXBASE_MIGRATIONS_ENABLED=false`)
2. **IP Allowlisting** - Only allow requests from trusted container networks
3. **Service Key Authentication** - Strongest authentication method (always required)
4. **Scope Validation** - Service key must have `migrations:execute` scope
5. **Rate Limiting** - Maximum 10 requests per hour per service key
6. **Audit Logging** - All requests are logged for security review

## Setup Steps

### 1. Setup Service Key

You have three options:

#### Option A: Use Existing Service Key with Wildcard Scope (Easiest)

If you already have a service key with `*` (all permissions), you can use it:

```sql
-- Check your existing service key
SELECT name, scopes FROM auth.service_keys WHERE enabled = true;

-- If it has '*' in scopes, you're done! Use that key.
```

#### Option B: Add Migrations Scope to Existing Key (Recommended)

Add the migrations scope to your existing service key:

```sql
-- Add migrations:execute scope to your existing service key
UPDATE auth.service_keys
SET scopes = array_append(scopes, 'migrations:execute')
WHERE name = 'Your Service Key Name'
AND NOT ('migrations:execute' = ANY(scopes));
```

#### Option C: Create Dedicated Migrations Key (Most Secure)

For production, you may want a dedicated key that can only run migrations:

```bash
./scripts/generate-migration-service-key.sh
```

This will:

- Generate a cryptographically secure random key
- Insert it with `migrations:execute` and `migrations:read` scopes only
- Display the key (save it securely!)

**Manual creation:**

```bash
# Generate a random key
SERVICE_KEY="sk_migrations_$(openssl rand -base64 32 | tr -d '=+/' | cut -c1-32)"

# Insert into database
psql $DATABASE_URL << EOF
INSERT INTO auth.service_keys (name, key_hash, key_prefix, scopes, enabled, expires_at)
VALUES (
    'Migration Service Key',
    crypt('$SERVICE_KEY', gen_salt('bf')),
    substring('$SERVICE_KEY', 1, 16),
    ARRAY['migrations:execute', 'migrations:read'],
    true,
    NOW() + INTERVAL '1 year'
);
EOF

# Save this key!
echo "FLUXBASE_MIGRATIONS_SERVICE_KEY=$SERVICE_KEY"
```

### 2. Configure Fluxbase

Enable the migrations API in your Fluxbase configuration:

```bash
# Enable migrations API
FLUXBASE_MIGRATIONS_ENABLED=true

# Require service key authentication
FLUXBASE_MIGRATIONS_REQUIRE_SERVICE_KEY=true

# IP allowlist (adjust for your network)
FLUXBASE_MIGRATIONS_ALLOWED_IP_RANGES=172.16.0.0/12,10.0.0.0/8,192.168.0.0/16

# Admin database credentials (for DDL operations)
FLUXBASE_DATABASE_ADMIN_USER=postgres
FLUXBASE_DATABASE_ADMIN_PASSWORD=your-secure-password
```

**Default IP Ranges:**

- `172.16.0.0/12` - Docker bridge networks
- `10.0.0.0/8` - Private networks (AWS VPC, etc.)
- `192.168.0.0/16` - Private networks
- `127.0.0.0/8` - Loopback (localhost)

### 3. Configure Your Application

Store the service key in your application's environment:

```bash
# Application environment
FLUXBASE_MIGRATIONS_SERVICE_KEY=sk_migrations_abc123...
```

### 4. Use the SDK

In your application code:

```typescript
import { createClient } from "@fluxbase/sdk";

// Initialize client with service key as the second parameter
const client = createClient(
  process.env.FLUXBASE_URL || "http://localhost:8080",
  process.env.FLUXBASE_MIGRATIONS_SERVICE_KEY || ""
);

// Register migrations
client.admin.migrations
  .register({
    name: "001_create_users",
    namespace: "default",
    up_sql: `
      CREATE TABLE users (
        id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
        email TEXT UNIQUE NOT NULL,
        created_at TIMESTAMPTZ DEFAULT NOW()
      );
    `,
    down_sql: `DROP TABLE users;`,
  })
  .register({
    name: "002_add_user_profiles",
    namespace: "default",
    up_sql: `
      CREATE TABLE user_profiles (
        id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
        user_id UUID REFERENCES users(id) ON DELETE CASCADE,
        name TEXT,
        bio TEXT
      );
    `,
    down_sql: `DROP TABLE user_profiles;`,
  });

// Sync migrations to Fluxbase
await client.admin.migrations.sync("default", {
  autoApply: true, // Automatically apply new migrations
});
```

**Note:** The service key is passed as the second parameter to `createClient()`. The SDK automatically sends it in the `apikey` and `Authorization` headers, which the migrations API accepts.

## Docker Compose Example

```yaml
services:
  fluxbase:
    image: fluxbase:latest
    environment:
      # Enable migrations API
      FLUXBASE_MIGRATIONS_ENABLED: "true"
      FLUXBASE_MIGRATIONS_REQUIRE_SERVICE_KEY: "true"
      FLUXBASE_MIGRATIONS_ALLOWED_IP_RANGES: "172.18.0.0/16"

      # Admin credentials for DDL operations
      FLUXBASE_DATABASE_ADMIN_USER: "postgres"
      FLUXBASE_DATABASE_ADMIN_PASSWORD: "postgres"

      # Runtime credentials (lower privileges)
      FLUXBASE_DATABASE_USER: "fluxbase_app"
      FLUXBASE_DATABASE_PASSWORD: "fluxbase_app_password"
    networks:
      - backend

  app:
    image: your-app:latest
    environment:
      FLUXBASE_URL: "http://fluxbase:8080"
      FLUXBASE_MIGRATIONS_SERVICE_KEY: "${MIGRATION_SERVICE_KEY}"
    networks:
      - backend
    depends_on:
      - fluxbase

networks:
  backend:
    driver: bridge
```

## Monitoring & Alerts

### Check Migration Status

```typescript
// Get specific migration
const migration = await client.admin.migrations.get(
  "001_create_users",
  "default"
);
console.log(migration.status); // 'pending', 'applied', 'failed'

// List all migrations
const all = await client.admin.migrations.list("default");

// Get execution history
const executions = await client.admin.migrations.getExecutions(
  "001_create_users",
  "default"
);
executions.forEach((exec) => {
  console.log(`${exec.action} - ${exec.status} - ${exec.duration_ms}ms`);
  if (exec.error_message) {
    console.error("Error:", exec.error_message);
  }
});
```

### Audit Logs

All migrations API requests are logged with:

- Timestamp
- IP address
- Service key ID and name
- HTTP method and path
- Response status
- Duration

View logs:

```bash
docker logs fluxbase-container | grep "Migrations API"
```

## Security Best Practices

### 1. Rotate Service Keys Regularly

Service keys have an expiration date (default: 1 year). Rotate them before expiration:

```bash
# Disable old key
UPDATE auth.service_keys SET enabled = false WHERE key_prefix = 'sk_migrations_old';

# Generate new key
./scripts/generate-migration-service-key.sh

# Update application environment
# Deploy with new FLUXBASE_MIGRATIONS_SERVICE_KEY
```

### 2. Restrict IP Ranges

Configure the tightest possible IP allowlist:

```bash
# Development - Allow Docker network
FLUXBASE_MIGRATIONS_ALLOWED_IP_RANGES=172.18.0.0/16

# Production - Allow only app subnet
FLUXBASE_MIGRATIONS_ALLOWED_IP_RANGES=10.0.1.0/24
```

### 3. Monitor Failed Attempts

Set up alerts for:

- Multiple failed authentication attempts
- Rate limit exceeded events
- Migration failures
- Requests from unexpected IPs

### 4. Use Separate Admin Credentials

Never use the superuser account. Create a dedicated admin user:

```sql
CREATE USER fluxbase_admin WITH PASSWORD 'secure-password';
GRANT ALL PRIVILEGES ON DATABASE fluxbase TO fluxbase_admin;
GRANT ALL ON SCHEMA public TO fluxbase_admin;
```

### 5. Test Migrations

Always test migrations in a staging environment first:

```typescript
// In staging
await client.admin.migrations.sync("default", {
  autoApply: false, // Don't auto-apply in staging
});

// Review pending migrations
const pending = await client.admin.migrations.list("default", "pending");

// Apply manually after review
await client.admin.migrations.apply("001_create_users", "default");
```

## Troubleshooting

### Migrations API returns 404

- Check: `FLUXBASE_MIGRATIONS_ENABLED=true`
- Restart Fluxbase after changing config

### 403 Forbidden (IP not allowlisted)

- Check your app container's IP: `docker inspect <container> | grep IPAddress`
- Ensure IP range includes your app's subnet
- Example: If IP is `172.18.0.5`, use `172.18.0.0/16`

### 401 Unauthorized (Invalid service key)

- Verify service key in database: `SELECT key_prefix, enabled, expires_at FROM auth.service_keys`
- Check service key in app environment matches database
- Ensure service key hasn't expired

### 403 Forbidden (Missing scope)

- Check service key scopes: `SELECT scopes FROM auth.service_keys WHERE key_prefix = 'sk_migrations_...'`
- Ensure `migrations:execute` scope is present
- Update scopes if needed:
  ```sql
  UPDATE auth.service_keys
  SET scopes = ARRAY['migrations:execute', 'migrations:read']
  WHERE key_prefix = 'sk_migrations_...';
  ```

### 429 Too Many Requests

- Rate limit: 10 requests/hour per service key
- Wait for rate limit window to reset
- For higher limits, modify `middleware.MigrationAPILimiter()` in code

### Migration Failed with Permission Error

- Check admin credentials: `FLUXBASE_DATABASE_ADMIN_USER` and `FLUXBASE_DATABASE_ADMIN_PASSWORD`
- Ensure admin user has DDL privileges:
  ```sql
  GRANT CREATE ON DATABASE fluxbase TO fluxbase_admin;
  GRANT ALL ON SCHEMA public TO fluxbase_admin;
  ```
- Check execution logs:
  ```typescript
  const executions = await client.admin.migrations.getExecutions(
    "migration_name",
    "default"
  );
  console.log(executions[0].error_message);
  ```

## Architecture Diagram

```
┌─────────────────┐
│  App Container  │
│                 │
│  • Migrations   │
│  • Service Key  │
└────────┬────────┘
         │ X-Service-Key
         │
         ▼
┌─────────────────────────────────────┐
│      Fluxbase Container             │
│                                     │
│  ┌───────────────────────────────┐ │
│  │   Security Middleware Stack   │ │
│  │   1. Feature Flag             │ │
│  │   2. IP Allowlist             │ │
│  │   3. Service Key Auth         │ │
│  │   4. Scope Validation         │ │
│  │   5. Rate Limiting            │ │
│  │   6. Audit Logging            │ │
│  └──────────┬────────────────────┘ │
│             ▼                       │
│  ┌───────────────────────────────┐ │
│  │   Migrations Executor         │ │
│  │   • Connect as admin user     │ │
│  │   • Execute DDL operations    │ │
│  │   • Log results               │ │
│  └──────────┬────────────────────┘ │
└─────────────┼───────────────────────┘
              │ Admin Credentials
              ▼
┌──────────────────────────┐
│   PostgreSQL Database    │
│                          │
│  • migrations.migrations │
│  • migrations.exec_logs  │
│  • User tables           │
└──────────────────────────┘
```

## Additional Resources

- [Fluxbase Migrations Documentation](https://fluxbase.dev/docs/migrations)
- [Security Best Practices](https://fluxbase.dev/docs/security)
- [API Reference](https://fluxbase.dev/api-reference/migrations)
