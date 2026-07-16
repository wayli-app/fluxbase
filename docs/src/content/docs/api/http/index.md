---
title: HTTP API Reference
description: Complete HTTP API documentation for Fluxbase REST endpoints including authentication, storage, database operations, multi-tenancy, functions, jobs, and more.
---
The Fluxbase HTTP API provides RESTful endpoints for authentication, storage, database operations, multi-tenancy management, edge functions, background jobs, and more. All endpoints are prefixed with `/api/v1/` unless otherwise noted.

## Base URL

```
http://localhost:8080/api/v1
```

## Authentication

Most endpoints require authentication via JWT bearer tokens or service keys. Include the token in the `Authorization` header:

```bash
curl -H "Authorization: Bearer YOUR_ACCESS_TOKEN" \
  http://localhost:8080/api/v1/auth/user
```

### Multi-Tenant Requests

For multi-tenant deployments, specify the tenant context via the `X-FB-Tenant` header:

```bash
curl -H "Authorization: Bearer <service-key>" \
     -H "X-FB-Tenant: acme-corp" \
     http://localhost:8080/api/v1/tables/posts
```

When using a tenant-scoped service key, the tenant context is embedded in the key and the header is optional.

## API Categories

### Authentication

Endpoints for user registration, login, and session management.

#### Public

| Method | Endpoint | Description |
|--------|----------|-------------|
| `GET` | `/auth/config` | Get auth configuration |
| `GET` | `/auth/csrf` | Get CSRF token |
| `GET` | `/auth/captcha/config` | Get CAPTCHA configuration |
| `POST` | `/auth/captcha/check` | Check if CAPTCHA is required |
| `POST` | `/auth/signup` | Register a new user |
| `POST` | `/auth/signin` | Sign in with email/password |
| `POST` | `/auth/signin/idtoken` | Sign in with ID token |
| `POST` | `/auth/refresh` | Refresh access token |
| `POST` | `/auth/magiclink` | Request magic link |
| `POST` | `/auth/magiclink/verify` | Verify magic link token |
| `POST` | `/auth/password/reset` | Request password reset email |
| `POST` | `/auth/password/reset/verify` | Verify password reset token |
| `POST` | `/auth/password/reset/confirm` | Confirm password reset |
| `POST` | `/auth/verify-email` | Verify email address |
| `POST` | `/auth/verify-email/resend` | Resend email verification |
| `POST` | `/auth/2fa/verify` | Verify 2FA (TOTP) code |
| `POST` | `/auth/otp/signin` | Send OTP code |
| `POST` | `/auth/otp/verify` | Verify OTP code |
| `POST` | `/auth/otp/resend` | Resend OTP code |

#### Authenticated

| Method | Endpoint | Description |
|--------|----------|-------------|
| `POST` | `/auth/signout` | Sign out current session |
| `GET` | `/auth/user` | Get current user |
| `PATCH` | `/auth/user` | Update current user |
| `POST` | `/auth/reauthenticate` | Reauthenticate current session |
| `GET` | `/auth/user/identities` | List linked identities |
| `POST` | `/auth/user/identities` | Link an identity |
| `DELETE` | `/auth/user/identities/{id}` | Unlink an identity |

#### Two-Factor Authentication

| Method | Endpoint | Description |
|--------|----------|-------------|
| `POST` | `/auth/2fa/setup` | Set up TOTP 2FA |
| `POST` | `/auth/2fa/enable` | Enable 2FA after setup |
| `POST` | `/auth/2fa/disable` | Disable 2FA |
| `GET` | `/auth/2fa/status` | Get 2FA status |

#### Impersonation

| Method | Endpoint | Description |
|--------|----------|-------------|
| `POST` | `/auth/impersonate` | Start impersonating a user |
| `POST` | `/auth/impersonate/anon` | Start anonymous impersonation |
| `POST` | `/auth/impersonate/service` | Start service impersonation |
| `DELETE` | `/auth/impersonate` | Stop impersonation |
| `GET` | `/auth/impersonate` | Get active impersonation |
| `GET` | `/auth/impersonate/sessions` | List impersonation sessions |

#### OAuth

| Method | Endpoint | Description |
|--------|----------|-------------|
| `GET` | `/auth/oauth/providers` | List available OAuth providers |
| `GET` | `/auth/oauth/{provider}/authorize` | Start OAuth authorization flow |
| `GET` | `/auth/oauth/{provider}/callback` | OAuth callback |
| `GET` | `/auth/oauth/{provider}/token` | Get OAuth provider token |
| `POST` | `/auth/oauth/{provider}/logout` | Initiate OAuth provider logout |
| `GET` | `/auth/oauth/{provider}/logout/callback` | OAuth logout callback |

#### SAML SSO

| Method | Endpoint | Description |
|--------|----------|-------------|
| `GET` | `/auth/saml/providers` | List SAML providers |
| `GET` | `/auth/saml/metadata/{provider}` | Get SAML SP metadata |
| `GET` | `/auth/saml/login/{provider}` | Initiate SAML login |
| `POST` | `/auth/saml/acs` | SAML Assertion Consumer Service |
| `POST` | `/auth/saml/slo` | SAML Single Logout (POST) |
| `GET` | `/auth/saml/slo` | SAML Single Logout (GET) |
| `GET` | `/auth/saml/logout/{provider}` | Initiate SAML logout |

### Storage

Endpoints for file storage operations.

| Method | Endpoint | Description |
|--------|----------|-------------|
| `GET` | `/storage/object` | Download file via signed URL (public, token-auth) |
| `GET` | `/storage/config/transforms` | Get image transformation configuration |
| `GET` | `/storage/buckets` | List all buckets |
| `POST` | `/storage/buckets/{bucket}` | Create bucket |
| `PUT` | `/storage/buckets/{bucket}` | Update bucket settings |
| `DELETE` | `/storage/buckets/{bucket}` | Delete bucket |
| `GET` | `/storage/{bucket}` | List files in bucket |
| `POST` | `/storage/{bucket}/{key}` | Upload file |
| `GET` | `/storage/{bucket}/{key}` | Download file |
| `HEAD` | `/storage/{bucket}/{key}` | Get file metadata |
| `DELETE` | `/storage/{bucket}/{key}` | Delete file |
| `POST` | `/storage/{bucket}/multipart` | Multipart file upload |
| `POST` | `/storage/{bucket}/stream/{key}` | Streaming file upload |
| `POST` | `/storage/{bucket}/sign/{key}` | Generate signed URL |
| `POST` | `/storage/{bucket}/{key}/share` | Share file with another user |
| `DELETE` | `/storage/{bucket}/{key}/share/{user_id}` | Revoke file share |
| `GET` | `/storage/{bucket}/{key}/shares` | List file shares |
| `POST` | `/storage/{bucket}/chunked/init` | Initialize chunked upload |
| `PUT` | `/storage/{bucket}/chunked/{uploadId}/{chunkIndex}` | Upload a chunk |
| `POST` | `/storage/{bucket}/chunked/{uploadId}/complete` | Complete chunked upload |
| `GET` | `/storage/{bucket}/chunked/{uploadId}/status` | Get chunked upload status |
| `DELETE` | `/storage/{bucket}/chunked/{uploadId}` | Abort chunked upload |

### GraphQL

A full GraphQL API auto-generated from your database schema.

| Method | Endpoint | Description |
|--------|----------|-------------|
| `POST` | `/graphql` | Execute GraphQL queries and mutations |
| `GET` | `/graphql` | Return the GraphQL introspection schema (when enabled) |

See the [GraphQL API documentation](/api/http/graphql) for complete details on queries, mutations, filtering, and SDK usage.

### Database Tables

Auto-generated, PostgREST-style CRUD for your PostgreSQL tables. Paths are schema-scoped (`/tables/{schema}/{table}`); `/tables/{schema}` addresses all tables in a schema, and `/tables/` lists tables.

| Method | Endpoint | Description |
|--------|----------|-------------|
| `GET` | `/tables/` | List all tables |
| `POST` | `/tables/{schema}/{table}/query` | Query with a complex filter body |
| `GET` | `/tables/{schema}/{table}` | List records (filters via query params) |
| `POST` | `/tables/{schema}/{table}` | Create record(s) — send a JSON array for bulk insert |
| `PATCH` | `/tables/{schema}/{table}` | Update records matching a query filter |
| `DELETE` | `/tables/{schema}/{table}` | Delete records matching a query filter |
| `GET` | `/tables/{schema}/{table}/{id}` | Get a record by primary key |
| `PUT` | `/tables/{schema}/{table}/{id}` | Replace a record |
| `PATCH` | `/tables/{schema}/{table}/{id}` | Update a record |
| `DELETE` | `/tables/{schema}/{table}/{id}` | Delete a record |

Batch update/delete use query-parameter filters (see [Query Parameters](#query-parameters)); there are no dedicated `/bulk` or `/export` routes.

### Tenant Management

Manage tenants in multi-tenant deployments. Requires `admin`, `instance_admin`, or `tenant_admin` role.

| Method | Endpoint | Description |
|--------|----------|-------------|
| `GET` | `/admin/tenants` | List all tenants |
| `GET` | `/admin/tenants/mine` | List tenants for current user |
| `GET` | `/admin/tenants/deleted` | List soft-deleted tenants |
| `POST` | `/admin/tenants` | Create tenant |
| `GET` | `/admin/tenants/{id}` | Get tenant details |
| `PATCH` | `/admin/tenants/{id}` | Update tenant |
| `DELETE` | `/admin/tenants/{id}` | Soft delete tenant (`?hard=true` for hard delete) |
| `POST` | `/admin/tenants/{id}/recover` | Recover soft-deleted tenant |
| `POST` | `/admin/tenants/{id}/migrate` | Migrate tenant to latest schema |
| `POST` | `/admin/tenants/{id}/repair` | Repair tenant (re-run bootstrap + FDW) |

### Tenant Members

| Method | Endpoint | Description |
|--------|----------|-------------|
| `GET` | `/admin/tenants/{id}/members` | List tenant members |
| `POST` | `/admin/tenants/{id}/members` | Add member to tenant |
| `DELETE` | `/admin/tenants/{id}/members/{user_id}` | Remove member from tenant |
| `GET` | `/admin/tenants/{id}/admins` | List tenant admins |
| `POST` | `/admin/tenants/{id}/admins` | Assign tenant admin |
| `DELETE` | `/admin/tenants/{id}/admins/{user_id}` | Remove tenant admin |

### Tenant Settings

| Method | Endpoint | Description |
|--------|----------|-------------|
| `GET` | `/admin/tenants/{id}/settings` | Get tenant settings |
| `PATCH` | `/admin/tenants/{id}/settings` | Update tenant settings |
| `DELETE` | `/admin/tenants/{id}/settings/{key}` | Delete a tenant setting |
| `GET` | `/admin/tenants/{id}/settings/{key}` | Get a specific tenant setting |

### Tenant Declarative Schema

| Method | Endpoint | Description |
|--------|----------|-------------|
| `GET` | `/admin/tenants/{id}/schema` | Get schema status |
| `POST` | `/admin/tenants/{id}/schema/apply` | Apply schema from filesystem |
| `GET` | `/admin/tenants/{id}/schema/content` | Get stored schema SQL |
| `POST` | `/admin/tenants/{id}/schema/content` | Upload schema SQL |
| `POST` | `/admin/tenants/{id}/schema/content/apply` | Upload and apply schema SQL |
| `DELETE` | `/admin/tenants/{id}/schema/content` | Delete stored schema |

### Service Keys

Manage API service keys. Scoped to the current tenant context via `X-FB-Tenant`.

| Method | Endpoint | Description |
|--------|----------|-------------|
| `GET` | `/admin/service-keys` | List service keys |
| `POST` | `/admin/service-keys` | Create service key |
| `GET` | `/admin/service-keys/{id}` | Get service key details |
| `PUT` | `/admin/service-keys/{id}` | Update service key |
| `DELETE` | `/admin/service-keys/{id}` | Delete service key |
| `POST` | `/admin/service-keys/{id}/disable` | Disable service key |
| `POST` | `/admin/service-keys/{id}/enable` | Enable service key |
| `POST` | `/admin/service-keys/{id}/revoke` | Revoke service key |
| `POST` | `/admin/service-keys/{id}/deprecate` | Deprecate key with grace period |
| `POST` | `/admin/service-keys/{id}/rotate` | Rotate service key |
| `GET` | `/admin/service-keys/{id}/revocations` | Get revocation history |

### Client Keys

Manage client keys for key-based authentication.

| Method | Endpoint | Description |
|--------|----------|-------------|
| `GET` | `/client-keys` | List client keys |
| `GET` | `/client-keys/{id}` | Get a client key |
| `POST` | `/client-keys` | Create a client key |
| `PATCH` | `/client-keys/{id}` | Update a client key |
| `DELETE` | `/client-keys/{id}` | Delete a client key |
| `POST` | `/client-keys/{id}/revoke` | Revoke a client key |

### Edge Functions

| Method | Endpoint | Description |
|--------|----------|-------------|
| `GET` | `/functions` | List functions |
| `POST` | `/functions` | Create function |
| `GET` | `/functions/{name}` | Get function details |
| `PUT` | `/functions/{name}` | Update function |
| `DELETE` | `/functions/{name}` | Delete function |
| `POST` | `/functions/{name}/invoke` | Invoke function (POST) |
| `GET` | `/functions/{name}/invoke` | Invoke function (GET, for health checks) |
| `GET` | `/functions/{name}/executions` | Get function execution history |
| `GET` | `/functions/shared` | List shared modules |
| `GET` | `/functions/shared/{path}` | Get a shared module |
| `POST` | `/functions/shared` | Create a shared module |
| `PUT` | `/functions/shared/{path}` | Update a shared module |
| `DELETE` | `/functions/shared/{path}` | Delete a shared module |

### Background Jobs

| Method | Endpoint | Description |
|--------|----------|-------------|
| `POST` | `/jobs/submit` | Submit a new job |
| `GET` | `/jobs` | List jobs |
| `GET` | `/jobs/{id}` | Get job details by ID |
| `POST` | `/jobs/{id}/cancel` | Cancel a job |
| `POST` | `/jobs/{id}/retry` | Retry a job |
| `GET` | `/jobs/{id}/logs` | Get job logs |

### RPC (Remote Procedures)

| Method | Endpoint | Description |
|--------|----------|-------------|
| `GET` | `/rpc/procedures` | List available RPC procedures |
| `POST` | `/rpc/{namespace}/{name}` | Invoke an RPC procedure |
| `GET` | `/rpc/executions/{id}` | Get RPC execution status |
| `GET` | `/rpc/executions/{id}/logs` | Get RPC execution logs |

### Database Branching

Manage database branches for isolated dev/test environments. All routes require `admin`, `instance_admin`, `tenant_admin`, or `service_role`.

| Method | Endpoint | Description |
|--------|----------|-------------|
| `GET` | `/admin/branches` | List branches |
| `POST` | `/admin/branches` | Create branch |
| `GET` | `/admin/branches/{id}` | Get branch details |
| `DELETE` | `/admin/branches/{id}` | Delete branch |
| `POST` | `/admin/branches/{id}/reset` | Reset branch |
| `GET` | `/admin/branches/{id}/activity` | Get branch activity |
| `GET` | `/admin/branches/active` | Get active branch |
| `POST` | `/admin/branches/active` | Set active branch |
| `DELETE` | `/admin/branches/active` | Reset active branch |
| `GET` | `/admin/branches/stats/pools` | Get branch pool stats |
| `GET` | `/admin/branches/{id}/access` | List branch access grants |
| `POST` | `/admin/branches/{id}/access` | Grant branch access |
| `DELETE` | `/admin/branches/{id}/access/{user_id}` | Revoke branch access |
| `GET` | `/admin/branches/github/configs` | List GitHub webhook configs |
| `POST` | `/admin/branches/github/configs` | Upsert GitHub webhook config |
| `DELETE` | `/admin/branches/github/configs/{repository}` | Delete GitHub webhook config |

### Webhooks

| Method | Endpoint | Description |
|--------|----------|-------------|
| `GET` | `/webhooks` | List webhooks |
| `POST` | `/webhooks` | Create webhook |
| `GET` | `/webhooks/{id}` | Get webhook details |
| `PATCH` | `/webhooks/{id}` | Update webhook |
| `DELETE` | `/webhooks/{id}` | Delete webhook |
| `POST` | `/webhooks/{id}/test` | Test webhook delivery |
| `GET` | `/webhooks/{id}/deliveries` | List webhook deliveries |

### Migrations

All migration endpoints require a service key or `admin`/`instance_admin`/`tenant_admin` role and are gated by the migrations IP allowlist.

| Method | Endpoint | Description |
|--------|----------|-------------|
| `GET` | `/admin/migrations` | List migrations |
| `POST` | `/admin/migrations` | Create migration |
| `GET` | `/admin/migrations/{name}` | Get migration details |
| `PUT` | `/admin/migrations/{name}` | Update migration |
| `DELETE` | `/admin/migrations/{name}` | Delete migration |
| `POST` | `/admin/migrations/{name}/apply` | Apply migration |
| `POST` | `/admin/migrations/{name}/rollback` | Rollback migration |
| `GET` | `/admin/migrations/{name}/executions` | Get execution history |
| `POST` | `/admin/migrations/apply-pending` | Apply all pending migrations |
| `POST` | `/admin/migrations/sync` | Sync migrations (batch upload) |

### Secrets

Manage secrets for edge functions and background jobs.

| Method | Endpoint | Description |
|--------|----------|-------------|
| `GET` | `/secrets` | List secrets |
| `GET` | `/secrets/stats` | Get secrets stats |
| `POST` | `/secrets` | Create a secret |
| `GET` | `/secrets/{id}` | Get secret by ID |
| `PUT` | `/secrets/{id}` | Update secret by ID |
| `DELETE` | `/secrets/{id}` | Delete secret by ID |
| `GET` | `/secrets/{id}/versions` | Get secret versions by ID |
| `POST` | `/secrets/{id}/rollback/{version}` | Rollback secret to version |
| `GET` | `/secrets/by-name/{name}` | Get secret by name |
| `PUT` | `/secrets/by-name/{name}` | Update secret by name |
| `DELETE` | `/secrets/by-name/{name}` | Delete secret by name |
| `GET` | `/secrets/by-name/{name}/versions` | Get secret versions by name |
| `POST` | `/secrets/by-name/{name}/rollback/{version}` | Rollback secret by name |

### AI Chatbots (Public)

Public chatbot discovery and the conversational WebSocket. Chat is streaming over a WebSocket — there is no REST `POST .../chat` endpoint.

| Method | Endpoint | Description |
|--------|----------|-------------|
| `GET` | `/ai/chatbots` | List enabled (public) chatbots |
| `GET` | `/ai/chatbots/by-name/{name}` | Look up a chatbot by name |
| `GET` | `/ai/chatbots/{id}` | Get a public chatbot |
| `GET` | `/ai/ws` | Chat WebSocket (streaming responses) |
| `GET` | `/ai/conversations` | List the current user's conversations |
| `GET` | `/ai/conversations/{id}` | Get a conversation |
| `PATCH` | `/ai/conversations/{id}` | Update a conversation (e.g. title) |
| `DELETE` | `/ai/conversations/{id}` | Delete a conversation |
| `GET` | `/ai/usage/{chatbotId}` | Current user's daily quota snapshot |

:::note[Chatbot management]
Creating, updating, deleting, and toggling chatbots is admin-only via `/admin/ai/chatbots/*` (see the Admin AI section).
:::

### Knowledge Bases

User-facing knowledge-base routes. All require authentication, and the caller must have access to the given knowledge base.

| Method | Endpoint | Description |
|--------|----------|-------------|
| `GET` | `/ai/knowledge-bases` | List the user's knowledge bases |
| `POST` | `/ai/knowledge-bases` | Create a knowledge base |
| `GET` | `/ai/knowledge-bases/{id}` | Get a knowledge base |
| `POST` | `/ai/knowledge-bases/{id}/share` | Share a knowledge base with a user |
| `GET` | `/ai/knowledge-bases/{id}/permissions` | List KB permissions |
| `DELETE` | `/ai/knowledge-bases/{id}/permissions/{user_id}` | Revoke KB permission |
| `POST` | `/ai/knowledge-bases/{id}/documents` | Add a document (JSON) |
| `POST` | `/ai/knowledge-bases/{id}/documents/upload` | Upload a document file |
| `GET` | `/ai/knowledge-bases/{id}/documents` | List documents |
| `GET` | `/ai/knowledge-bases/{id}/documents/{doc_id}` | Get a document |
| `PATCH` | `/ai/knowledge-bases/{id}/documents/{doc_id}` | Update a document |
| `DELETE` | `/ai/knowledge-bases/{id}/documents/{doc_id}` | Delete a document |
| `POST` | `/ai/knowledge-bases/{id}/documents/delete-by-filter` | Delete documents by filter |
| `POST` | `/ai/knowledge-bases/{id}/search` | Semantic search |
| `POST` | `/ai/knowledge-bases/{id}/debug-search` | Debug search (scores/explanation) |
| `GET` | `/ai/knowledge-bases/{id}/entities` | List entities |
| `GET` | `/ai/knowledge-bases/{id}/entities/search` | Search entities |
| `GET` | `/ai/knowledge-bases/{id}/entities/{entity_id}/relationships` | Get entity relationships |
| `GET` | `/ai/knowledge-bases/{id}/graph` | Get the full knowledge graph |
| `GET` | `/ai/knowledge-bases/{id}/chatbots` | List chatbots linked to the KB |

### Realtime

WebSocket endpoint for realtime subscriptions:

```
ws://localhost:8080/realtime
```

Channels: `table:{schema}.{table}`, `presence:{room}`, `broadcast:{channel}`

| Method | Endpoint | Description |
|--------|----------|-------------|
| `GET` | `/realtime/stats` | Get realtime connection statistics |

### Public Settings

Public settings endpoints (no authentication required).

| Method | Endpoint | Description |
|--------|----------|-------------|
| `GET` | `/settings` | List all settings |
| `GET` | `/settings/{key}` | Get a setting |
| `POST` | `/settings/batch` | Batch get settings |

### User Settings

Authenticated user settings management.

| Method | Endpoint | Description |
|--------|----------|-------------|
| `GET` | `/settings/user/list` | List user's own settings |
| `GET` | `/settings/user/own/{key}` | Get user's own setting |
| `GET` | `/settings/user/system/{key}` | Get system setting (public info) |
| `GET` | `/settings/user/{key}` | Get a user setting |
| `PUT` | `/settings/user/{key}` | Set a user setting |
| `DELETE` | `/settings/user/{key}` | Delete a user setting |

### User Secrets

Authenticated user secrets management.

| Method | Endpoint | Description |
|--------|----------|-------------|
| `POST` | `/settings/secret` | Create a user secret |
| `GET` | `/settings/secret` | List user secrets |
| `GET` | `/settings/secret/{path}` | Get a user secret |
| `PUT` | `/settings/secret/{path}` | Update a user secret |
| `DELETE` | `/settings/secret/{path}` | Delete a user secret |

### Monitoring

System monitoring endpoints. Requires authentication.

| Method | Endpoint | Description |
|--------|----------|-------------|
| `GET` | `/monitoring/metrics` | Get system metrics |
| `GET` | `/monitoring/health` | Get system health status |
| `GET` | `/monitoring/logs` | Get system logs |

### Invitations

Public invitation endpoints (token-based, no auth required).

| Method | Endpoint | Description |
|--------|----------|-------------|
| `GET` | `/invitations/{token}/validate` | Validate invitation token |
| `POST` | `/invitations/{token}/accept` | Accept invitation |

### MCP (Model Context Protocol)

Built-in JSON-RPC 2.0 endpoint for AI assistant integration. The base path is configurable (default: `/mcp`).

| Method | Endpoint | Description |
|--------|----------|-------------|
| `POST` | `/mcp` | MCP JSON-RPC requests |
| `GET` | `/mcp` | MCP SSE stream |
| `GET` | `/mcp/health` | MCP health check |

See [MCP Server Guide](/guides/mcp/) for details.

### MCP OAuth

OAuth 2.0 endpoints for MCP authentication. All endpoints are public (no auth required).

| Method | Endpoint | Description |
|--------|----------|-------------|
| `GET` | `/mcp/.well-known/oauth-authorization-server` | OAuth authorization server metadata |
| `GET` | `/mcp/.well-known/oauth-protected-resource` | OAuth protected resource metadata |
| `GET` | `/mcp/.well-known/oauth-protected-resource/mcp` | OAuth protected resource metadata for MCP |
| `POST` | `/mcp/oauth/register` | Dynamic client registration |
| `GET` | `/mcp/oauth/authorize` | OAuth authorization |
| `POST` | `/mcp/oauth/authorize` | OAuth authorization consent |
| `POST` | `/mcp/oauth/token` | OAuth token exchange |
| `POST` | `/mcp/oauth/revoke` | OAuth token revocation |

### Custom MCP Tools & Resources

Admin-only management of custom MCP tools and resources. Requires `admin` role.

| Method | Endpoint | Description |
|--------|----------|-------------|
| `GET` | `/mcp/config` | Get MCP configuration |
| `GET` | `/mcp/tools` | List custom MCP tools |
| `POST` | `/mcp/tools` | Create custom MCP tool |
| `POST` | `/mcp/tools/sync` | Sync custom MCP tool (upsert) |
| `GET` | `/mcp/tools/{id}` | Get custom MCP tool |
| `PUT` | `/mcp/tools/{id}` | Update custom MCP tool |
| `DELETE` | `/mcp/tools/{id}` | Delete custom MCP tool |
| `POST` | `/mcp/tools/{id}/test` | Test custom MCP tool |
| `GET` | `/mcp/resources` | List custom MCP resources |
| `POST` | `/mcp/resources` | Create custom MCP resource |
| `POST` | `/mcp/resources/sync` | Sync custom MCP resource (upsert) |
| `GET` | `/mcp/resources/{id}` | Get custom MCP resource |
| `PUT` | `/mcp/resources/{id}` | Update custom MCP resource |
| `DELETE` | `/mcp/resources/{id}` | Delete custom MCP resource |
| `POST` | `/mcp/resources/{id}/test` | Test custom MCP resource |

### Sync

Admin sync endpoints for loading definitions from filesystem or database. Requires `admin`, `instance_admin`, or `service_role` role.

| Method | Endpoint | Description |
|--------|----------|-------------|
| `POST` | `/admin/functions/sync` | Sync functions from filesystem |
| `POST` | `/admin/jobs/sync` | Sync jobs from filesystem |
| `POST` | `/admin/ai/chatbots/sync` | Sync AI chatbots from filesystem |
| `POST` | `/admin/rpc/sync` | Sync RPC procedures from database |

### GitHub Webhook

Public endpoint for GitHub webhook integration (no auth, uses HMAC signature verification).

| Method | Endpoint | Description |
|--------|----------|-------------|
| `POST` | `/webhooks/github` | GitHub webhook for branch automation |

### Dashboard Auth

Admin dashboard authentication endpoints. All endpoints are public (no auth required for setup/login, unified auth for authenticated endpoints).

| Method | Endpoint | Description |
|--------|----------|-------------|
| `GET` | `/admin/setup/status` | Get dashboard setup status |
| `POST` | `/admin/setup` | Initial dashboard setup |
| `POST` | `/admin/login` | Dashboard admin login |
| `POST` | `/admin/refresh` | Refresh dashboard token |
| `POST` | `/admin/logout` | Dashboard admin logout |
| `GET` | `/admin/me` | Get current admin user |

### Dashboard User Auth (`/dashboard/auth`)

A parallel auth surface for Fluxbase **operators** (`platform.users`, `instance_admin`/`tenant_admin` roles), distinct from application auth (`/api/v1/auth/*`). See the [Dashboard Auth guide](/guides/dashboard-auth/) for the full route reference. Key endpoints:

| Method | Endpoint | Description |
|--------|----------|-------------|
| `POST` | `/dashboard/auth/login` | Operator login (returns tokens or `requires_2fa`) |
| `POST` | `/dashboard/auth/refresh` | Refresh an operator token |
| `POST` | `/dashboard/auth/2fa/{setup,enable,disable,verify}` | TOTP management |
| `POST` | `/dashboard/auth/password/{reset,reset/verify,reset/confirm,change}` | Password flows |
| `GET` | `/dashboard/auth/sso/{providers,oauth/:provider,saml/:provider}` | Dashboard SSO |
| `GET` | `/dashboard/auth/me` | Current operator (protected) |

### Health

Public health check endpoints (no auth required).

| Method | Endpoint | Description |
|--------|----------|-------------|
| `GET` | `/` | Root health check |
| `GET` | `/health` | Detailed health check with database status |

## Query Parameters

Table endpoints support PostgREST-compatible query parameters:

| Parameter | Description | Example |
|-----------|-------------|---------|
| `select` | Columns to return | `?select=id,name,email` |
| `order` | Sort order | `?order=created_at.desc` |
| `limit` | Max results | `?limit=10` |
| `offset` | Pagination offset | `?offset=20` |
| `{column}.{op}` | Column filter | `?name.eq=John&age.gt=18` |

### Filter Operators

| Operator | Description | Example |
|----------|-------------|---------|
| `eq` | Equal | `?status.eq=active` |
| `neq` | Not equal | `?status.neq=deleted` |
| `gt` | Greater than | `?age.gt=18` |
| `gte` | Greater than or equal | `?age.gte=18` |
| `lt` | Less than | `?price.lt=100` |
| `lte` | Less than or equal | `?price.lte=100` |
| `like` | Pattern match | `?name.like=John%` |
| `ilike` | Case-insensitive pattern | `?name.ilike=john%` |
| `in` | In list | `?status.in=(active,pending)` |
| `is` | Is null/not null | `?deleted_at.is.null` |

## Common Headers

| Header | Description |
|--------|-------------|
| `Authorization` | Bearer token for authentication (`Bearer <jwt>`) |
| `X-Client-Key` | Client key for key-based authentication |
| `X-FB-Tenant` | Tenant slug for multi-tenant context |
| `X-Fluxbase-Branch` | Branch name for database branching context |
| `Content-Type` | Request body format (`application/json`, `multipart/form-data`) |
| `Prefer` | Response preferences (`return=representation`, `count=exact`) |

## API Reference & OpenAPI

An interactive API reference (Scalar) is served at:

```
GET /api-docs
```

A live OpenAPI 3.0 specification is also available:

```
GET /openapi.json
```

The specification is generated dynamically from the registered routes and your database schema, and includes request/response schemas for all endpoints.

## Error Responses

Errors return JSON: `{"error": "description"}`. Standard HTTP status codes apply (400, 401, 403, 404, 409, 429, 500, 503).
