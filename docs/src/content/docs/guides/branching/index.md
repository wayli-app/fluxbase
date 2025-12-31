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

## Configuration

Enable branching in your `fluxbase.yaml`:

```yaml
branching:
  enabled: true
  max_branches_per_user: 5
  max_total_branches: 50
  default_data_clone_mode: schema_only
  auto_delete_after: 24h
  database_prefix: branch_
  admin_database_url: "postgresql://admin:password@localhost:5432/postgres"
```

### Configuration Options

| Option | Default | Description |
|--------|---------|-------------|
| `enabled` | `false` | Enable database branching |
| `max_branches_per_user` | `5` | Maximum branches per user |
| `max_total_branches` | `50` | Maximum total branches |
| `default_data_clone_mode` | `schema_only` | Default cloning mode |
| `auto_delete_after` | `0` | Auto-delete preview branches after this duration (0 = disabled) |
| `database_prefix` | `branch_` | Prefix for branch database names |
| `admin_database_url` | - | Connection URL with CREATE DATABASE privileges |

**Note:** When `auto_delete_after` is set (e.g., `24h`, `7d`), a background cleanup scheduler runs automatically to delete expired branches. The scheduler runs at intervals equal to the `auto_delete_after` duration (minimum 1 hour).

### Data Clone Modes

| Mode | Description |
|------|-------------|
| `schema_only` | Copy schema only, no data |
| `full_clone` | Copy schema and all data |
| `seed_data` | Copy schema with seed data (coming soon) |

## Quick Start

### Using the CLI

```bash
# Create a branch
fluxbase branch create my-feature

# List all branches
fluxbase branch list

# Get branch details
fluxbase branch get my-feature

# Reset branch to parent state
fluxbase branch reset my-feature

# Delete a branch
fluxbase branch delete my-feature
```

### Using the TypeScript SDK

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

See the [TypeScript SDK Branching Guide](/guides/typescript-sdk/branching/) for complete documentation.

### Using the REST API

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

Include the `X-Fluxbase-Branch` header:

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

Get the branch connection URL:

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

## Nested Branches

Create a branch from another branch:

```bash
fluxbase branch create feature-b --from feature-a
```

This creates a chain: `main` → `feature-a` → `feature-b`

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

- [TypeScript SDK Branching](/guides/typescript-sdk/branching/) - SDK documentation
- [Branching Workflows](/guides/branching/workflows/) - Development workflow examples
- [GitHub Integration](/guides/branching/github-integration/) - Automatic PR branches
- [CLI Commands](/cli/commands/#branch-commands) - CLI reference
- [Security](/security/branching-security/) - Security best practices
