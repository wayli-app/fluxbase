---
title: "Database Branching"
description: "Manage database branches with the TypeScript SDK"
---

The TypeScript SDK provides full support for database branching, allowing you to create, manage, and delete isolated database copies for development, testing, and preview environments.

## Overview

Database branches are isolated copies of your database that can be used for:

- **Feature development** - Work on new features without affecting production
- **PR previews** - Automatic preview environments for pull requests
- **Migration testing** - Test schema changes safely before applying to production
- **Bug investigation** - Clone production data to debug issues

## Basic Usage

Access branching operations via `client.branching`:

```typescript
import { createClient } from '@fluxbase/sdk'

const client = createClient('http://localhost:8080', 'your-key')

// List all branches
const { data, error } = await client.branching.list()

// Create a new branch
const { data: branch } = await client.branching.create('feature/add-auth')

// Get branch details
const { data: details } = await client.branching.get('feature/add-auth')

// Delete a branch
await client.branching.delete('feature/add-auth')
```

## Creating Branches

### Basic Branch

```typescript
const { data, error } = await client.branching.create('my-feature')

if (error) {
  console.error('Failed to create branch:', error.message)
  return
}

console.log('Created branch:', data.slug)
console.log('Status:', data.status) // 'creating' initially
```

### With Options

```typescript
const { data } = await client.branching.create('feature/add-auth', {
  // How to clone data from parent
  dataCloneMode: 'schema_only', // 'schema_only' | 'full_clone' | 'seed_data'

  // Branch type
  type: 'persistent', // 'main' | 'preview' | 'persistent'

  // Auto-delete after duration
  expiresIn: '7d', // Duration string like '24h', '7d', '30d'

  // Optional: clone from specific parent branch
  parentBranchId: 'parent-branch-id'
})
```

### PR Preview Branch

For GitHub PR integration:

```typescript
const { data } = await client.branching.create(`pr-${prNumber}`, {
  type: 'preview',
  githubPRNumber: prNumber,
  githubRepo: 'owner/repo',
  githubPRUrl: `https://github.com/owner/repo/pull/${prNumber}`,
  expiresIn: '7d'
})
```

## Waiting for Branch to be Ready

Branch creation is asynchronous. Use `waitForReady()` to poll until the branch is ready:

```typescript
// Create branch
const { data: branch } = await client.branching.create('my-feature')

// Wait for it to be ready (default: 30s timeout)
const { data: ready, error } = await client.branching.waitForReady(branch.slug)

if (error) {
  console.error('Branch failed:', error.message)
  return
}

console.log('Branch is ready!')
```

With custom timeout and polling interval:

```typescript
const { data, error } = await client.branching.waitForReady('my-feature', {
  timeout: 60000,      // 60 seconds max wait
  pollInterval: 2000   // Check every 2 seconds
})
```

## Listing Branches

### All Branches

```typescript
const { data } = await client.branching.list()

for (const branch of data.branches) {
  console.log(`${branch.name}: ${branch.status}`)
}
```

### With Filters

```typescript
// Only ready branches
const { data } = await client.branching.list({
  status: 'ready'
})

// Only preview branches
const { data } = await client.branching.list({
  type: 'preview'
})

// Only my branches
const { data } = await client.branching.list({
  mine: true
})

// Filter by GitHub repo
const { data } = await client.branching.list({
  githubRepo: 'owner/repo'
})

// Pagination
const { data } = await client.branching.list({
  limit: 10,
  offset: 20
})
```

## Getting Branch Details

Get a branch by ID or slug:

```typescript
// By slug
const { data, error } = await client.branching.get('feature/add-auth')

// By ID
const { data } = await client.branching.get('123e4567-e89b-12d3-a456-426614174000')

if (data) {
  console.log('Name:', data.name)
  console.log('Status:', data.status)
  console.log('Type:', data.type)
  console.log('Created:', data.created_at)
  console.log('Expires:', data.expires_at)
}
```

## Checking if Branch Exists

```typescript
const exists = await client.branching.exists('feature/add-auth')

if (!exists) {
  await client.branching.create('feature/add-auth')
}
```

## Resetting a Branch

Reset a branch to its parent state, dropping all changes:

```typescript
const { data, error } = await client.branching.reset('feature/add-auth')

if (error) {
  console.error('Failed to reset:', error.message)
  return
}

console.log('Branch reset, status:', data.status)
```

:::caution
Resetting a branch permanently deletes all data and schema changes in that branch. This cannot be undone.
:::

## Deleting a Branch

```typescript
const { error } = await client.branching.delete('feature/add-auth')

if (error) {
  console.error('Failed to delete:', error.message)
}
```

:::note
You cannot delete the main branch.
:::

## Branch Activity Log

View the activity history for a branch:

```typescript
const { data } = await client.branching.getActivity('feature/add-auth')

for (const entry of data) {
  console.log(`${entry.action}: ${entry.status} at ${entry.created_at}`)
}

// Get more entries (default: 50, max: 100)
const { data } = await client.branching.getActivity('feature/add-auth', 100)
```

## Connection Pool Statistics

Monitor database connections across branches:

```typescript
const { data } = await client.branching.getPoolStats()

for (const pool of data) {
  console.log(`${pool.slug}:`)
  console.log(`  Active: ${pool.active_connections}`)
  console.log(`  Idle: ${pool.idle_connections}`)
  console.log(`  Total: ${pool.total_connections}`)
}
```

## Connecting to a Branch

To query data from a specific branch, pass the branch name in client options:

```typescript
// Create client connected to a specific branch
const branchClient = createClient(
  'http://localhost:8080',
  'your-key',
  { headers: { 'X-Fluxbase-Branch': 'feature/add-auth' } }
)

// All queries now use the branch database
const { data } = await branchClient.from('users').select('*').execute()
```

Or use the branch header per-request with the HTTP client:

```typescript
const response = await fetch('http://localhost:8080/api/v1/tables/public/users', {
  headers: {
    'Authorization': 'Bearer your-token',
    'X-Fluxbase-Branch': 'feature/add-auth'
  }
})
```

## Type Reference

### Branch

```typescript
interface Branch {
  id: string
  name: string
  slug: string
  database_name: string
  status: 'creating' | 'ready' | 'migrating' | 'error' | 'deleting' | 'deleted'
  type: 'main' | 'preview' | 'persistent'
  parent_branch_id?: string
  data_clone_mode: 'schema_only' | 'full_clone' | 'seed_data'
  github_pr_number?: number
  github_pr_url?: string
  github_repo?: string
  error_message?: string
  created_by?: string
  created_at: string
  updated_at: string
  expires_at?: string
}
```

### CreateBranchOptions

```typescript
interface CreateBranchOptions {
  parentBranchId?: string
  dataCloneMode?: 'schema_only' | 'full_clone' | 'seed_data'
  type?: 'main' | 'preview' | 'persistent'
  githubPRNumber?: number
  githubPRUrl?: string
  githubRepo?: string
  expiresIn?: string  // e.g., '24h', '7d'
}
```

### ListBranchesOptions

```typescript
interface ListBranchesOptions {
  status?: 'creating' | 'ready' | 'migrating' | 'error' | 'deleting' | 'deleted'
  type?: 'main' | 'preview' | 'persistent'
  githubRepo?: string
  mine?: boolean
  limit?: number
  offset?: number
}
```

## Complete Example

```typescript
import { createClient } from '@fluxbase/sdk'

const client = createClient('http://localhost:8080', 'your-key')

async function developFeature(featureName: string) {
  const branchName = `feature/${featureName}`

  // Check if branch already exists
  if (await client.branching.exists(branchName)) {
    console.log('Branch already exists, resetting...')
    await client.branching.reset(branchName)
  } else {
    console.log('Creating new branch...')
    const { data, error } = await client.branching.create(branchName, {
      dataCloneMode: 'schema_only',
      expiresIn: '7d'
    })

    if (error) {
      throw new Error(`Failed to create branch: ${error.message}`)
    }
  }

  // Wait for branch to be ready
  const { data: branch, error } = await client.branching.waitForReady(branchName, {
    timeout: 60000
  })

  if (error) {
    throw new Error(`Branch failed to become ready: ${error.message}`)
  }

  console.log(`Branch ${branch.slug} is ready!`)
  console.log(`Database: ${branch.database_name}`)
  console.log(`Expires: ${branch.expires_at}`)

  return branch
}

async function cleanupBranch(branchName: string) {
  const { error } = await client.branching.delete(branchName)

  if (error) {
    console.error(`Failed to delete branch: ${error.message}`)
  } else {
    console.log(`Branch ${branchName} deleted`)
  }
}

// Usage
const branch = await developFeature('add-user-profiles')
// ... do development work ...
await cleanupBranch(branch.slug)
```

## Next Steps

- [Database Branching Overview](/guides/branching/) - Core concepts and architecture
- [Branching Workflows](/guides/branching/workflows/) - Development workflow examples
- [GitHub Integration](/guides/branching/github-integration/) - Automated PR preview branches
- [CLI Branch Commands](/cli/commands/#branch-commands) - CLI reference
