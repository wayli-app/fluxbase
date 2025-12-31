---
title: "Branching Workflows"
description: "End-to-end development workflows with database branches"
---

This guide demonstrates practical workflows for using database branches in your development process.

## Development Workflow Overview

Database branches let you:

1. **Test migrations safely** - Apply schema changes to a branch before production
2. **Isolate feature development** - Each feature gets its own database copy
3. **Create PR previews** - Automatic preview environments for pull requests
4. **Debug production issues** - Clone production data to investigate bugs

## Workflow 1: Feature Development

Use branches to develop and test new features in isolation.

### Create a Feature Branch

```bash
# Create a branch for your feature
fluxbase branch create add-user-profiles

# Check the branch was created
fluxbase branch status add-user-profiles
```

Output:
```
Branch: add-user-profiles (add-user-profiles)
Status: ready
```

### Connect to Your Branch

Configure your application to use the branch:

```typescript
// TypeScript SDK
import { createClient } from '@fluxbase/sdk'

const fluxbase = createClient(
  'http://localhost:8080',
  'your-client-key',
  { branch: 'add-user-profiles' }  // Connect to branch
)
```

Or use HTTP headers:

```bash
curl http://localhost:8080/api/v1/tables/public/users \
  -H "X-Fluxbase-Branch: add-user-profiles" \
  -H "Authorization: Bearer $TOKEN"
```

### Make Schema Changes

Create a migration for your feature:

```sql
-- migrations/003_add_user_profiles.up.sql
CREATE TABLE public.profiles (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES auth.users(id),
    bio TEXT,
    avatar_url TEXT,
    created_at TIMESTAMPTZ DEFAULT now()
);

CREATE INDEX idx_profiles_user_id ON public.profiles(user_id);
```

Apply to your branch:

```bash
fluxbase migrations sync
```

### Test Your Changes

Run your application tests against the branch:

```bash
# Your tests automatically use the branch if configured
npm test

# Or specify the branch in your test setup
FLUXBASE_BRANCH=add-user-profiles npm test
```

### Review and Merge

Once satisfied:

1. Commit your migration files
2. Create a pull request
3. After review, merge to main
4. Apply migrations to production:

```bash
fluxbase --profile prod migrations apply-pending
```

### Clean Up

Delete the branch when done:

```bash
fluxbase branch delete add-user-profiles
```

## Workflow 2: PR Preview Environments

Automatically create database branches for each pull request.

### GitHub Actions Setup

Add this workflow to `.github/workflows/pr-preview.yml`:

```yaml
name: PR Preview

on:
  pull_request:
    types: [opened, synchronize, reopened, closed]

jobs:
  preview:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - name: Install Fluxbase CLI
        run: |
          curl -fsSL https://raw.githubusercontent.com/fluxbase-eu/fluxbase/main/install-cli.sh | bash

      - name: Create/Update Preview Branch
        if: github.event.action != 'closed'
        env:
          FLUXBASE_SERVER: ${{ secrets.FLUXBASE_SERVER }}
          FLUXBASE_TOKEN: ${{ secrets.FLUXBASE_TOKEN }}
        run: |
          # Create or reset the preview branch
          fluxbase branch create "pr-${{ github.event.number }}" \
            --pr ${{ github.event.number }} \
            --repo ${{ github.repository }} \
            --expires-in 7d 2>/dev/null || \
          fluxbase branch reset "pr-${{ github.event.number }}" --force

          # Apply migrations
          fluxbase migrations sync

          # Deploy functions
          fluxbase functions sync

      - name: Comment Preview Info
        if: github.event.action == 'opened'
        uses: actions/github-script@v7
        with:
          script: |
            github.rest.issues.createComment({
              issue_number: context.issue.number,
              owner: context.repo.owner,
              repo: context.repo.repo,
              body: `## Preview Environment Ready

              Branch: \`pr-${{ github.event.number }}\`

              **Connect via SDK:**
              \`\`\`typescript
              const client = createClient(url, key, { branch: 'pr-${{ github.event.number }}' })
              \`\`\`

              **Connect via Header:**
              \`\`\`
              X-Fluxbase-Branch: pr-${{ github.event.number }}
              \`\`\`

              This preview will be automatically deleted when the PR is closed.`
            })

      - name: Delete Preview Branch
        if: github.event.action == 'closed'
        env:
          FLUXBASE_SERVER: ${{ secrets.FLUXBASE_SERVER }}
          FLUXBASE_TOKEN: ${{ secrets.FLUXBASE_TOKEN }}
        run: |
          fluxbase branch delete "pr-${{ github.event.number }}" --force || true
```

### How It Works

1. **PR Opened**: Creates a new branch named `pr-{number}`, applies migrations and functions
2. **PR Updated**: Resets the branch and reapplies changes
3. **PR Closed**: Deletes the branch and database

### Testing Against PR Preview

In your CI tests:

```yaml
test:
  runs-on: ubuntu-latest
  steps:
    - uses: actions/checkout@v4

    - name: Run Tests
      env:
        FLUXBASE_SERVER: ${{ secrets.FLUXBASE_SERVER }}
        FLUXBASE_TOKEN: ${{ secrets.FLUXBASE_TOKEN }}
        FLUXBASE_BRANCH: pr-${{ github.event.number }}
      run: npm test
```

## Workflow 3: Migration Testing

Test database migrations before applying to production.

### Create a Test Branch

```bash
# Create branch with full data clone (for realistic testing)
fluxbase branch create test-migrations --clone-data full_clone
```

### Apply and Test Migrations

```bash
# Apply pending migrations to the test branch
fluxbase migrations sync

# Run migration tests
fluxbase tables query users --limit 10

# Check for any data issues
fluxbase graphql query '{ users { id email } }'
```

### Verify Rollback Works

```bash
# Rollback the migration
fluxbase migrations rollback --step 1

# Verify data is intact
fluxbase tables query users --limit 10

# Re-apply
fluxbase migrations apply-pending
```

### Apply to Production

Once testing passes:

```bash
# Switch to production profile
fluxbase --profile prod migrations apply-pending

# Verify
fluxbase --profile prod migrations list
```

### Clean Up

```bash
fluxbase branch delete test-migrations
```

## Workflow 4: Debugging Production Issues

Clone production data to investigate issues safely.

### Create a Debug Branch

```bash
# Clone with full data (be careful with sensitive data!)
fluxbase branch create debug-issue-123 --clone-data full_clone

# Or schema only if data isn't needed
fluxbase branch create debug-issue-123 --clone-data schema_only
```

### Investigate the Issue

```bash
# Query problematic data
fluxbase tables query orders \
  --where "status=eq.stuck" \
  --select "id,user_id,status,created_at"

# Run custom SQL
fluxbase graphql query '{
  orders(filter: { status_eq: "stuck" }) {
    id
    user { email }
    items { product_id quantity }
  }
}'
```

### Test a Fix

Make your fix, then test:

```bash
# Apply your fix migration
fluxbase migrations sync

# Verify the fix
fluxbase tables query orders --where "status=eq.stuck"
```

### Apply Fix to Production

```bash
fluxbase --profile prod migrations apply-pending
```

### Clean Up

```bash
fluxbase branch delete debug-issue-123
```

## Best Practices

### Branch Naming Conventions

Use consistent naming for easy identification:

```bash
# Feature branches
fluxbase branch create feature/add-auth
fluxbase branch create feature/update-billing

# PR previews
fluxbase branch create pr-123

# Bug investigation
fluxbase branch create debug/order-stuck-issue

# Testing
fluxbase branch create test/migration-v2
```

### Set Expiration for Temporary Branches

Prevent abandoned branches from accumulating:

```bash
# Auto-delete after 24 hours
fluxbase branch create temp-test --expires-in 24h

# Auto-delete after 7 days
fluxbase branch create pr-preview --expires-in 7d
```

### Use Appropriate Clone Modes

| Mode | Use Case |
|------|----------|
| `schema_only` | Most development work, migrations |
| `full_clone` | Bug investigation, data-dependent tests |
| `seed_data` | Development with sample data (coming soon) |

### Monitor Branch Usage

Keep track of active branches:

```bash
# List all branches
fluxbase branch list

# See only your branches
fluxbase branch list --mine

# Check connection pool stats
fluxbase branch stats
```

### Clean Up Regularly

Remove branches you no longer need:

```bash
# Delete a specific branch
fluxbase branch delete my-old-branch

# Force delete (skip confirmation)
fluxbase branch delete abandoned-branch --force
```

## Troubleshooting

### Branch Creation Fails

**Error**: "Maximum branches limit reached"

```bash
# List branches to find ones to delete
fluxbase branch list --mine

# Delete unused branches
fluxbase branch delete old-feature --force
```

**Error**: "Failed to create database"

Check that:
1. The admin database URL is configured
2. The database user has CREATE DATABASE privileges
3. There's enough disk space

### Branch Not Found

```bash
# Check if branch exists
fluxbase branch list | grep my-branch

# Get full branch details
fluxbase branch get my-branch
```

### Migrations Fail on Branch

```bash
# Check migration status
fluxbase migrations list

# View branch activity for errors
fluxbase branch activity my-branch --limit 10

# Reset and retry
fluxbase branch reset my-branch --force
fluxbase migrations sync
```

## Next Steps

- [Database Branching Overview](/guides/branching/) - Core concepts
- [TypeScript SDK Branching](/guides/typescript-sdk/branching/) - SDK documentation
- [GitHub Integration](/guides/branching/github-integration/) - Automated PR branches
- [CLI Commands Reference](/cli/commands/#branch-commands) - All branch commands
