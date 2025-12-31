---
title: "Database Branching"
description: "Isolated database environments for development and testing"
---

Database branching allows you to create isolated copies of your database for development, testing, or preview environments. Each branch is a separate PostgreSQL database that can be used independently.

## Overview

Use database branches to:

- Test migrations before applying to production
- Create isolated environments for PR previews
- Safely experiment with schema changes
- Run integration tests with real data structures

## Quick Start with CLI

The easiest way to work with branches is using the Fluxbase CLI. The server handles all the database operations - you just run commands.

### Common Workflow

```bash
# Create a branch for your feature
fluxbase branch create my-feature

# Work with the branch (it's automatically used by the CLI)
fluxbase branch get my-feature

# When done, delete the branch
fluxbase branch delete my-feature
```

### Essential Commands

```bash
# Create a branch (copies schema from main by default)
fluxbase branch create my-feature

# Create a branch with full data copy
fluxbase branch create my-feature --clone-mode full_clone

# List all branches
fluxbase branch list

# Show branch details
fluxbase branch get my-feature

# Reset branch to parent state (useful for testing migrations)
fluxbase branch reset my-feature

# Delete a branch
fluxbase branch delete my-feature
```

### Creating Nested Branches

Create a branch from another branch:

```bash
# Create feature-b from feature-a (instead of main)
fluxbase branch create feature-b --from feature-a
```

This creates a chain: `main` → `feature-a` → `feature-b`

## Server Configuration

**This section is for server operators only.** Users of the CLI or API don't need to configure anything - the server handles branching automatically.

### Prerequisites

The PostgreSQL user must have `CREATE DATABASE` privilege. Grant it if needed:

```sql
-- Connect to postgres database as superuser
GRANT CREATE ON DATABASE postgres TO your_fluxbase_user;
```

### Minimal Configuration

Enable branching in your `fluxbase.yaml`:

```yaml
branching:
  enabled: true
```

That's it! The server will use its existing database credentials to create and manage branches.

### Optional Settings

```yaml
branching:
  enabled: true
  max_total_branches: 50
  default_data_clone_mode: schema_only
  auto_delete_after: 24h
  database_prefix: branch_
```

| Option | Default | Description |
|--------|---------|-------------|
| `enabled` | `false` | Enable database branching |
| `max_total_branches` | `50` | Maximum total branches across all users |
| `default_data_clone_mode` | `schema_only` | Default cloning mode (schema_only, full_clone, seed_data) |
| `auto_delete_after` | `0` | Auto-delete preview branches after this duration (0 = disabled) |
| `database_prefix` | `branch_` | Prefix for branch database names |

**Note:** When `auto_delete_after` is set (e.g., `24h`, `7d`), a background cleanup scheduler runs automatically to delete expired branches.

### Data Clone Modes

| Mode | Description |
|------|-------------|
| `schema_only` | Copy schema only, no data (fast) |
| `full_clone` | Copy schema and all data (slower, useful for testing with real data) |
| `seed_data` | Copy schema with seed data (coming soon) |

### How It Works

When you create a branch, the server:

1. Uses its database credentials to connect to the `postgres` database
2. Executes `CREATE DATABASE branch_my_feature` (or similar)
3. Copies the schema (and optionally data) from the parent branch
4. Tracks the branch metadata in the `branching.branches` table

The server never needs separate admin credentials - it uses the same PostgreSQL user it already has.

## Using TypeScript SDK

```typescript
import { createClient } from '@fluxbase/sdk'

const client = createClient('http://localhost:8080', 'your-key')

// Create a branch
const { data: branch } = await client.branching.create('my-feature', {
  dataCloneMode: 'schema_only',
  expiresIn: '7d'
})

// Wait for it to be ready
await client.branching.waitForReady('my-feature')

// Delete when done
await client.branching.delete('my-feature')
```

See the [TypeScript SDK Branching Guide](/guides/typescript-sdk/branching) for complete documentation.

## Using the REST API

For advanced users and custom integrations:

```bash
# Create a branch
curl -X POST http://localhost:8080/api/v1/admin/branches \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"name": "my-feature"}'

# Access branch data
curl http://localhost:8080/api/v1/tables/users \
  -H "Authorization: Bearer $TOKEN" \
  -H "X-Fluxbase-Branch: my-feature"
```

## Branch Types

| Type | Description | Auto-Delete |
|------|-------------|-------------|
| `main` | Primary database | Never |
| `preview` | Temporary environments | After `auto_delete_after` |
| `persistent` | Long-lived branches | Never |

## Connecting to Branches

### Via HTTP Header

Include the `X-Fluxbase-Branch` header in your requests:

```bash
curl http://localhost:8080/api/v1/tables/users \
  -H "X-Fluxbase-Branch: my-feature"
```

### Via Query Parameter

Append `?branch=` to the URL:

```bash
curl "http://localhost:8080/api/v1/tables/users?branch=my-feature"
```

### Direct Database Connection

Get the branch connection URL for direct PostgreSQL access:

```bash
fluxbase branch get my-feature --output json | jq -r .connection_url
```

## Branch Lifecycle

```
┌─────────────────────────────────────────────────────────────┐
│                        MAIN BRANCH                          │
│                    (Always exists)                          │
└───────────────────────────┬─────────────────────────────────┘
                            │
                            ▼
┌───────────────────────────────────────────────────────────────┐
│  Creating  →  Ready  →  Migrating  →  Ready  →  Deleting     │
│    (new)       (use)     (update)      (use)     (cleanup)   │
└───────────────────────────────────────────────────────────────┘
```

### States

| State | Description |
|-------|-------------|
| `creating` | Database is being created |
| `ready` | Branch is available for use |
| `migrating` | Migrations are running |
| `error` | An error occurred |
| `deleting` | Branch is being deleted |
| `deleted` | Branch has been deleted |

## Access Control

Branch access is controlled by:

1. **Creator** - Automatically has admin access
2. **Explicit Grants** - Can grant read/write/admin access to others
3. **Service Keys** - Have full access to all branches
4. **Dashboard Admins** - Have full access to all branches

### Access Levels

| Level | Permissions |
|-------|-------------|
| `read` | View branch, query data |
| `write` | Read + modify data |
| `admin` | Write + delete/reset branch |

## Next Steps

- [Branching Workflows](/guides/branching/workflows) - Development workflow examples
- [GitHub Integration](/guides/branching/github-integration) - Automatic PR branches
- [TypeScript SDK Branching](/guides/typescript-sdk/branching) - SDK documentation
- [Security Best Practices](/security/branching-security) - Security considerations
