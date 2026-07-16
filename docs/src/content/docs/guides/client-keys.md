---
title: "Client Keys"
description: Create and manage API keys for programmatic access to Fluxbase. Client keys provide scoped, rate-limited authentication distinct from service keys.
---

Client keys are user-scoped API keys that provide programmatic access to the Fluxbase API. They are distinct from service keys and JWT tokens, offering fine-grained scope control and per-key rate limiting.

## Overview

Client keys enable:

- **Scoped access** - Each key has specific permission scopes (e.g., `clientkeys:read`, `clientkeys:write`)
- **Rate limiting** - Per-key rate limits configurable at creation time
- **User isolation** - Regular users can only see and manage their own keys; admins can see all
- **Revocation** - Keys can be revoked (deactivated) without deleting them
- **Expiration** - Optional expiry time for temporary access

## Client Keys vs Service Keys

| Feature           | Client Keys                        | Service Keys                        |
| ----------------- | ---------------------------------- | ----------------------------------- |
| **Scope**         | Per-user, configurable scopes      | Per-tenant, full tenant access      |
| **Rate limiting** | Per-key configurable               | Global tenant limits                |
| **Management**    | Users manage their own keys        | Admin-only management               |
| **Auth header**   | `X-Client-Key`                     | `X-Service-Key`                     |
| **Expiration**    | Optional                           | Optional (enforced via `expires_at`) |

## API Endpoints

| Method   | Endpoint                        | Description          | Scope              |
| -------- | ------------------------------- | -------------------- | ------------------ |
| `GET`    | `/api/v1/client-keys`           | List client keys     | `clientkeys:read`  |
| `GET`    | `/api/v1/client-keys/:id`       | Get a client key     | `clientkeys:read`  |
| `POST`   | `/api/v1/client-keys`           | Create a client key  | `clientkeys:write` |
| `PATCH`  | `/api/v1/client-keys/:id`       | Update a client key  | `clientkeys:write` |
| `DELETE` | `/api/v1/client-keys/:id`       | Delete a client key  | `clientkeys:write` |
| `POST`   | `/api/v1/client-keys/:id/revoke`| Revoke a client key  | `clientkeys:write` |

When the system setting `app.auth.allow_user_client_keys` is disabled, only admins can access these endpoints.

## Usage

### Create a Client Key

```bash
curl -X POST \
  -H "Authorization: Bearer <jwt-token>" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "My CI/CD Key",
    "description": "Used for automated deployments",
    "scopes": ["clientkeys:read"],
    "rate_limit_per_minute": 100
  }' \
  http://localhost:8080/api/v1/client-keys
```

The raw key is returned only on creation. Store it securely — it cannot be retrieved later.

**Request fields:**

| Field                  | Type     | Required | Description                    |
| ---------------------- | -------- | -------- | ------------------------------ |
| `name`                 | string   | Yes      | Descriptive name               |
| `description`          | string   | No       | Optional details               |
| `scopes`               | string[] | No       | Permission scopes              |
| `rate_limit_per_minute`| int      | No       | Per-key rate limit             |
| `expires_at`           | string   | No       | ISO 8601 expiration timestamp  |

### List Client Keys

```bash
curl -H "Authorization: Bearer <jwt-token>" \
  http://localhost:8080/api/v1/client-keys
```

Admins can filter by user: `?user_id=<uuid>`

### Get a Client Key

```bash
curl -H "Authorization: Bearer <jwt-token>" \
  http://localhost:8080/api/v1/client-keys/<id>
```

### Update a Client Key

```bash
curl -X PATCH \
  -H "Authorization: Bearer <jwt-token>" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Updated Key Name",
    "scopes": ["clientkeys:read", "clientkeys:write"],
    "rate_limit_per_minute": 200
  }' \
  http://localhost:8080/api/v1/client-keys/<id>
```

### Revoke a Client Key

```bash
curl -X POST \
  -H "Authorization: Bearer <jwt-token>" \
  http://localhost:8080/api/v1/client-keys/<id>/revoke
```

Revocation deactivates the key without deleting it, preserving audit history.

### Delete a Client Key

```bash
curl -X DELETE \
  -H "Authorization: Bearer <jwt-token>" \
  http://localhost:8080/api/v1/client-keys/<id>
```

## Authentication with Client Keys

Use the `X-Client-Key` header to authenticate requests:

```bash
curl -H "X-Client-Key: <your-client-key>" \
  http://localhost:8080/api/v1/some-endpoint
```

## Learn More

- [Authentication](/guides/authentication/) - JWT and service key authentication
- [Service Keys](/guides/service-keys/) - Admin-managed privileged keys (rotate/revoke/audit)
- [Rate Limiting](/guides/rate-limiting/) - Global rate limiting configuration
- [Multi-Tenancy](/guides/multi-tenancy/) - Tenant-scoped access control
