---
title: "Branching API Reference"
description: "REST API and CLI reference for database branching"
---

## REST API

All branching endpoints require authentication via service key, client key, or JWT token.

Base path: `/api/v1/admin/branches`

### Create Branch

**POST** `/api/v1/admin/branches`

```bash
curl -X POST http://localhost:8080/api/v1/admin/branches \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "feature-login",
    "parent_branch_id": null,
    "data_clone_mode": "schema_only",
    "type": "preview",
    "expires_in": "24h"
  }'
```

**Request Body:**

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `name` | string | Yes | Branch name (1-100 chars) |
| `parent_branch_id` | uuid | No | Parent branch ID (defaults to main) |
| `data_clone_mode` | string | No | `schema_only`, `full_clone`, or `seed_data` |
| `type` | string | No | `preview` or `persistent` |
| `expires_in` | string | No | Duration like `24h`, `7d` |
| `github_pr_number` | int | No | Associated GitHub PR number |
| `github_pr_url` | string | No | GitHub PR URL |
| `github_repo` | string | No | GitHub repository (owner/repo) |

**Response:** `201 Created`

```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "name": "feature-login",
  "slug": "feature-login",
  "database_name": "branch_feature_login",
  "status": "creating",
  "type": "preview",
  "data_clone_mode": "schema_only",
  "created_at": "2024-01-15T10:00:00Z",
  "expires_at": "2024-01-16T10:00:00Z"
}
```

### List Branches

**GET** `/api/v1/admin/branches`

```bash
curl http://localhost:8080/api/v1/admin/branches \
  -H "Authorization: Bearer $TOKEN"
```

**Query Parameters:**

| Param | Type | Description |
|-------|------|-------------|
| `status` | string | Filter by status |
| `type` | string | Filter by type |
| `created_by` | uuid | Filter by creator |
| `limit` | int | Max results (default 50) |
| `offset` | int | Pagination offset |

**Response:** `200 OK`

```json
{
  "branches": [
    {
      "id": "...",
      "name": "feature-login",
      "slug": "feature-login",
      "status": "ready",
      "type": "preview",
      "created_at": "2024-01-15T10:00:00Z"
    }
  ],
  "total": 5
}
```

### Get Branch

**GET** `/api/v1/admin/branches/:id`

The `:id` can be either a UUID or a slug.

```bash
curl http://localhost:8080/api/v1/admin/branches/feature-login \
  -H "Authorization: Bearer $TOKEN"
```

**Response:** `200 OK`

```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "name": "feature-login",
  "slug": "feature-login",
  "database_name": "branch_feature_login",
  "status": "ready",
  "type": "preview",
  "data_clone_mode": "schema_only",
  "parent_branch_id": null,
  "created_by": "user-123",
  "created_at": "2024-01-15T10:00:00Z",
  "updated_at": "2024-01-15T10:00:15Z",
  "expires_at": "2024-01-16T10:00:00Z"
}
```

### Delete Branch

**DELETE** `/api/v1/admin/branches/:id`

```bash
curl -X DELETE http://localhost:8080/api/v1/admin/branches/feature-login \
  -H "Authorization: Bearer $TOKEN"
```

**Response:** `204 No Content`

**Errors:**

- `403 Forbidden` - Cannot delete main branch
- `403 Forbidden` - No admin access to branch
- `404 Not Found` - Branch not found

### Reset Branch

**POST** `/api/v1/admin/branches/:id/reset`

Resets the branch to its parent state (drops and recreates the database).

```bash
curl -X POST http://localhost:8080/api/v1/admin/branches/feature-login/reset \
  -H "Authorization: Bearer $TOKEN"
```

**Response:** `200 OK`

```json
{
  "id": "...",
  "status": "ready",
  "updated_at": "2024-01-15T12:00:00Z"
}
```

### Get Branch Activity

**GET** `/api/v1/admin/branches/:id/activity`

```bash
curl http://localhost:8080/api/v1/admin/branches/feature-login/activity \
  -H "Authorization: Bearer $TOKEN"
```

**Query Parameters:**

| Param | Type | Description |
|-------|------|-------------|
| `limit` | int | Max results (default 50, max 100) |

**Response:** `200 OK`

```json
{
  "activity": [
    {
      "id": "...",
      "action": "created",
      "user_id": "user-123",
      "details": {"data_clone_mode": "schema_only"},
      "created_at": "2024-01-15T10:00:00Z"
    }
  ]
}
```

### Get Pool Stats

**GET** `/api/v1/admin/branches/stats/pools`

Returns connection pool statistics for all branches (for debugging/monitoring).

```bash
curl http://localhost:8080/api/v1/admin/branches/stats/pools \
  -H "Authorization: Bearer $TOKEN"
```

**Response:** `200 OK`

```json
{
  "pools": {
    "feature-login": {
      "total_conns": 10,
      "idle_conns": 8,
      "acquired_conns": 2
    }
  }
}
```

## GitHub Configuration API

### List GitHub Configs

**GET** `/api/v1/admin/branches/github/configs`

```bash
curl http://localhost:8080/api/v1/admin/branches/github/configs \
  -H "Authorization: Bearer $TOKEN"
```

### Create/Update GitHub Config

**POST** `/api/v1/admin/branches/github/configs`

```bash
curl -X POST http://localhost:8080/api/v1/admin/branches/github/configs \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "repository": "owner/repo",
    "webhook_secret": "your-webhook-secret",
    "auto_create_on_pr": true,
    "auto_delete_on_merge": true,
    "default_data_clone_mode": "schema_only"
  }'
```

### Delete GitHub Config

**DELETE** `/api/v1/admin/branches/github/configs/:repository`

```bash
curl -X DELETE "http://localhost:8080/api/v1/admin/branches/github/configs/owner%2Frepo" \
  -H "Authorization: Bearer $TOKEN"
```

## CLI Reference

### branch list

List all database branches.

```bash
fluxbase branch list [flags]

Flags:
  -o, --output string   Output format: table, json, yaml (default "table")
      --status string   Filter by status
      --type string     Filter by type
```

### branch get

Get details of a specific branch.

```bash
fluxbase branch get <name-or-id> [flags]

Flags:
  -o, --output string   Output format: table, json, yaml (default "table")
```

### branch create

Create a new database branch.

```bash
fluxbase branch create <name> [flags]

Flags:
      --from string         Parent branch slug or ID
      --clone-mode string   Data clone mode: schema_only, full_clone (default "schema_only")
      --type string         Branch type: preview, persistent (default "preview")
      --expires-in string   Expiration duration (e.g., 24h, 7d)
```

### branch delete

Delete a database branch.

```bash
fluxbase branch delete <name-or-id> [flags]

Flags:
      --force   Skip confirmation
```

### branch reset

Reset a branch to its parent state.

```bash
fluxbase branch reset <name-or-id> [flags]

Flags:
      --force   Skip confirmation
```

### branch status

Show the status of a branch.

```bash
fluxbase branch status <name-or-id>
```

### branch activity

Show the activity log for a branch.

```bash
fluxbase branch activity <name-or-id> [flags]

Flags:
      --limit int   Maximum number of entries (default 50)
```

### branch stats

Show connection pool statistics.

```bash
fluxbase branch stats
```

## Error Codes

| Code | Error | Description |
|------|-------|-------------|
| `branching_disabled` | 503 | Branching is not enabled |
| `branch_not_found` | 404 | Branch does not exist |
| `branch_exists` | 409 | Branch with this name already exists |
| `cannot_delete_main` | 403 | Cannot delete the main branch |
| `cannot_reset_main` | 403 | Cannot reset the main branch |
| `max_branches_reached` | 403 | Maximum branches limit reached |
| `access_denied` | 403 | No permission for this operation |
| `validation_error` | 400 | Invalid request parameters |
