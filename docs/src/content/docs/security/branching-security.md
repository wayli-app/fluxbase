---
title: "Branching Security"
description: "Security best practices for database branching"
---

This guide covers security considerations and best practices for database branching.

## Access Control

### Branch Ownership

When a user creates a branch, they automatically receive admin access. This cannot be revoked without deleting the branch.

### Access Levels

| Level | Permissions |
|-------|-------------|
| `read` | Query branch data |
| `write` | Read + insert/update/delete data |
| `admin` | Write + reset/delete branch |

### Service Keys

Service keys have full access to all branches, including:
- Creating and deleting any branch
- Resetting any branch
- Accessing any branch's data

**Recommendation:** Don't expose service keys to untrusted applications.

### Dashboard Admins

Users with `dashboard_admin` or `admin` role have full access to all branches.

## Database Isolation

### Separate Databases

Each branch is a completely separate PostgreSQL database:

```
PostgreSQL Server
├── fluxbase (main)
├── branch_feature_login
├── branch_pr_42
└── branch_staging
```

Benefits:
- No accidental cross-branch data access
- Independent connection pools
- Complete schema isolation

### Connection Routing

Branches are accessed via:

1. **HTTP Header**: `X-Fluxbase-Branch: branch-slug`
2. **Query Parameter**: `?branch=branch-slug`

The router validates the branch exists and the user has access before routing.

## GitHub Webhook Security

### Signature Verification

Always configure a webhook secret:

```bash
# Generate a secure secret
openssl rand -hex 32

# Configure in Fluxbase
curl -X POST http://localhost:8080/api/v1/admin/branches/github/configs \
  -H "Authorization: Bearer $SERVICE_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "repository": "owner/repo",
    "webhook_secret": "your-secret-here"
  }'
```

### Unconfigured Repositories

Webhooks from unconfigured repositories are rejected by default. This prevents:
- Unauthorized branch creation
- Resource exhaustion attacks
- Abuse of the webhook endpoint

### IP Whitelisting

For additional security, configure your firewall to only accept webhooks from GitHub's IP ranges:
- `140.82.112.0/20`
- `143.55.64.0/20`
- `192.30.252.0/22`

Check [GitHub's documentation](https://docs.github.com/en/authentication/keeping-your-account-and-data-secure/about-githubs-ip-addresses) for current ranges.

## Data Security

### Clone Modes

| Mode | Data Copied | Use Case |
|------|-------------|----------|
| `schema_only` | Schema only | Development, testing |
| `full_clone` | All data | Staging, data analysis |

**Recommendation:** Use `schema_only` for PR previews to avoid copying sensitive production data.

### Data in Branches

Data in branches:
- Is isolated from other branches
- Is deleted when the branch is deleted
- Is not backed up separately (branches use PostgreSQL templates)

### Sensitive Data

Avoid copying production data to preview branches:

```yaml
branching:
  default_data_clone_mode: schema_only  # Don't copy production data
```

If you need test data, use seed scripts instead of `full_clone`.

## Resource Limits

### Branch Limits

Configure limits to prevent resource exhaustion:

```yaml
branching:
  max_branches_per_user: 5    # Per user limit
  max_total_branches: 50      # System-wide limit
  auto_delete_after: 24h      # Auto-cleanup for preview branches
```

### Connection Pools

Each branch has its own connection pool:
- Max 10 connections per branch
- 30-minute connection lifetime
- Pools are created on-demand

### Database Resources

Branch databases consume:
- Disk space (schema + data if full_clone)
- Connection slots
- PostgreSQL resources

Monitor usage and clean up unused branches.

## Authorization Best Practices

### 1. Use Minimal Permissions

Grant only necessary access levels:

```sql
-- Read-only access for testers
INSERT INTO branching.branch_access (branch_id, user_id, access_level)
VALUES ('...', 'tester-uuid', 'read');
```

### 2. Regular Cleanup

Implement regular cleanup of expired branches:

```bash
# Manual cleanup
curl -X DELETE http://localhost:8080/api/v1/admin/branches/cleanup \
  -H "Authorization: Bearer $SERVICE_KEY"
```

### 3. Audit Branch Operations

Monitor branch operations in logs:

```bash
# View recent branch activity
fluxbase branch activity feature-login
```

### 4. Separate Service Keys

Use separate service keys for:
- GitHub webhook integration
- CI/CD pipelines
- Administrative operations

## Credential Management

### Admin Database URL

The `admin_database_url` requires CREATE DATABASE privileges:

```yaml
branching:
  admin_database_url: "postgresql://branching_admin:password@localhost:5432/postgres"
```

**Security recommendations:**
- Use a dedicated PostgreSQL role with minimal privileges
- Store credentials in environment variables or secrets manager
- Don't use superuser credentials

```sql
-- Create dedicated role for branching
CREATE ROLE branching_admin WITH LOGIN PASSWORD 'secure-password';
GRANT CREATE ON DATABASE postgres TO branching_admin;
```

### Connection URLs

Branch connection URLs are derived from the main database URL. They inherit:
- Authentication credentials
- SSL settings
- Connection parameters

## Audit and Monitoring

### Activity Logging

All branch operations are logged to the `branching.activity_log` table:

| Action | Description |
|--------|-------------|
| `created` | Branch was created |
| `deleted` | Branch was deleted |
| `reset` | Branch was reset to parent state |
| `cloned` | Data was cloned from parent |
| `migrated` | Migrations were applied |
| `access_granted` | User was granted access |
| `access_revoked` | User's access was revoked |

Each activity includes:
- `executed_by` - User ID who performed the action
- `status` - started, success, or failed
- `details` - JSON with additional context
- `duration_ms` - Time taken (for long operations)

```sql
SELECT * FROM branching.activity_log
WHERE branch_id = '...'
ORDER BY executed_at DESC;
```

### Monitoring Queries

```sql
-- Active branches by type
SELECT type, status, COUNT(*)
FROM branching.branches
WHERE status != 'deleted'
GROUP BY type, status;

-- Branches per user
SELECT created_by, COUNT(*)
FROM branching.branches
WHERE status != 'deleted'
GROUP BY created_by;

-- Expired but not deleted
SELECT * FROM branching.branches
WHERE expires_at < NOW()
  AND status != 'deleted';
```

## Known Limitations

### Current Limitations

1. **Migrations** - Branches don't support running migrations via the API independently. Workaround: connect directly to the branch database using `psql` or database tools.
2. **RLS in Branches** - RLS policies are copied but may need adjustment
3. **Extensions** - PostgreSQL extensions must be installed server-wide
4. **Sequences** - Sequence values are reset in `schema_only` mode

### Supported Features

- **REST API on Branches** - Use `X-Fluxbase-Branch` header or `?branch=` parameter to query data on non-main branches
- **Branch Access Control** - Grant read/write/admin access to specific users
- **Audit Logging** - All branch operations are logged for security monitoring

### Security Implications

- Branch owners have full control over their branches
- Service keys can access all branches
- Deleted branch data is not recoverable
- Child branches become orphaned if parent is deleted
