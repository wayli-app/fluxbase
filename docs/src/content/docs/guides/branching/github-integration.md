---
title: "GitHub Integration"
description: "Automatic database branches for GitHub pull requests"
---

Fluxbase can automatically create and delete database branches when GitHub pull requests are opened, closed, or merged.

## Overview

When integrated with GitHub:

1. **PR Opened/Reopened** - Creates a database branch named `pr-{number}`
2. **PR Closed/Merged** - Deletes the associated branch

This enables isolated preview environments for each pull request.

## Setup

### 1. Configure Repository

Register your GitHub repository with Fluxbase:

```bash
curl -X POST http://localhost:8080/api/v1/admin/branches/github/configs \
  -H "Authorization: Bearer $SERVICE_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "repository": "owner/repo-name",
    "webhook_secret": "your-secure-secret",
    "auto_create_on_pr": true,
    "auto_delete_on_merge": true,
    "default_data_clone_mode": "schema_only"
  }'
```

### 2. Add Webhook in GitHub

1. Go to your repository **Settings** > **Webhooks**
2. Click **Add webhook**
3. Configure:
   - **Payload URL**: `https://your-fluxbase-server.com/api/v1/webhooks/github`
   - **Content type**: `application/json`
   - **Secret**: Same as `webhook_secret` above
   - **Events**: Select "Pull requests"
4. Click **Add webhook**

### 3. Verify Setup

Create a test PR and check the Fluxbase logs:

```bash
fluxbase branch list
```

You should see a new branch named `pr-{number}`.

## Configuration Options

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `repository` | string | Required | GitHub repository (owner/repo) |
| `webhook_secret` | string | - | Webhook signature secret |
| `auto_create_on_pr` | bool | `true` | Create branch on PR open |
| `auto_delete_on_merge` | bool | `true` | Delete branch on PR close |
| `default_data_clone_mode` | string | `schema_only` | Clone mode for PR branches |

## Webhook Security

### Signature Verification

GitHub signs webhook payloads with HMAC-SHA256. Fluxbase verifies signatures when `webhook_secret` is configured.

**Behavior:**

| Secret Configured | Signature Sent | Result |
|------------------|----------------|--------|
| Yes | Yes (valid) | Accepted |
| Yes | Yes (invalid) | Rejected |
| Yes | No | Rejected |
| No | Yes | Accepted (with warning) |
| No | No | Rejected* |

*Unconfigured repositories are rejected to prevent abuse. Configure the repository first.

### Best Practices

1. **Always configure a webhook secret** - Use a strong, random secret
2. **Use HTTPS** - Never send webhooks over plain HTTP
3. **Rotate secrets periodically** - Update both GitHub and Fluxbase config

Generate a secure secret:

```bash
openssl rand -hex 32
```

## Webhook Events

### Handled Events

| Event | Action | Result |
|-------|--------|--------|
| `pull_request` | `opened` | Create branch `pr-{number}` |
| `pull_request` | `reopened` | Create branch if not exists |
| `pull_request` | `closed` | Delete branch |
| `pull_request` | `synchronize` | (Acknowledged, no action) |
| `ping` | - | Returns "pong" |

### Ignored Events

All other events are acknowledged but ignored.

## Branch Naming

PR branches are automatically named:

- **Name**: `PR #{number}`
- **Slug**: `pr-{number}`
- **Database**: `branch_pr_{number}`

Example for PR #42:
- Slug: `pr-42`
- Database: `branch_pr_42`

## Using PR Branches

### Via HTTP Header

```bash
curl http://localhost:8080/api/v1/tables/users \
  -H "X-Fluxbase-Branch: pr-42"
```

### In CI/CD

Set the branch in your CI environment:

```yaml
# GitHub Actions example
steps:
  - name: Run tests
    env:
      FLUXBASE_BRANCH: pr-${{ github.event.number }}
    run: npm test
```

### In Application Code

```typescript
const client = createClient(FLUXBASE_URL, API_KEY, {
  headers: {
    'X-Fluxbase-Branch': `pr-${prNumber}`
  }
})
```

## Troubleshooting

### Branch Not Created

1. Check webhook delivery in GitHub (Settings > Webhooks > Recent Deliveries)
2. Verify Fluxbase logs for errors
3. Ensure `auto_create_on_pr` is enabled
4. Check if max branches limit reached

### Signature Verification Failed

1. Verify webhook secret matches in both GitHub and Fluxbase
2. Check for whitespace in the secret
3. Ensure using SHA-256 (not SHA-1)

### Branch Not Deleted

1. Check if `auto_delete_on_merge` is enabled
2. Verify the PR was closed (not just updated)
3. Check Fluxbase logs for deletion errors

### Webhook Returns 401

Repository not configured or signature mismatch. Configure the repository:

```bash
curl -X POST http://localhost:8080/api/v1/admin/branches/github/configs \
  -H "Authorization: Bearer $SERVICE_KEY" \
  -H "Content-Type: application/json" \
  -d '{"repository": "owner/repo"}'
```

## API Reference

### Webhook Endpoint

**POST** `/api/v1/webhooks/github`

No authentication required (uses signature verification).

**Headers:**

| Header | Description |
|--------|-------------|
| `X-GitHub-Event` | Event type (e.g., `pull_request`) |
| `X-GitHub-Delivery` | Unique delivery ID |
| `X-Hub-Signature-256` | HMAC-SHA256 signature |

**Response:**

```json
{
  "status": "created",
  "branch_id": "...",
  "branch_slug": "pr-42",
  "database": "branch_pr_42",
  "pr_number": 42
}
```

## Limitations

- Only GitHub is supported (GitLab, Bitbucket coming soon)
- One branch per PR (reopening uses existing branch if present)
- Branch data is not persisted after deletion
