Bug fixes and maintenance improvements have been made, including fixes for tenant isolation, settings defaults, and various other issues. Additionally, some features such as multi-tenancy, declarative schemas, and tenant-scoped impersonation have been improved or introduced.

## What's Changed

## Multi-Tenancy

Full tenant isolation with per-tenant databases and Row Level Security across all modules.

- platform schema — tenants, service keys, tenant admin assignments, invitation tokens
- Per-tenant databases via postgres_fdw — isolated public schema with shared services (auth, storage, jobs, etc.) accessed through foreign tables
- Pool priority chain — branch > tenant > main, resolved per-request via X-FB-Tenant header
- Tenant middleware wired across all route groups (REST, GraphQL, storage, auth, DDL, webhooks, RPC, jobs, functions)
- Tenant-scoped branching — branches clone from the tenant's database (not main) with automatic FDW repair
- Tenant admin roles with scoped impersonation and JWT claims
- Tenant management API and admin UI for CRUD, members, service keys, and schema management

## Declarative Database Schemas

Internal schemas managed declaratively instead of via imperative migrations.

- internal/database/schema/schemas/ — one SQL file per schema (auth, storage, jobs, functions, branching, ai, logging, mcp, platform, rpc, realtime)
- Bootstrap system — creates schemas, extensions, roles, and default privileges
- pgschema integration — diff-based plan/apply/dump
- Per-tenant declarative schemas — {schema_dir}/{tenant-slug}/public.sql with on-create, on-startup, or on-demand application
- Tenant Schema API — upload, diff, and apply schema content per tenant

## API Route Registry

Centralized route registration replacing scattered handler wiring.

- internal/api/routes/ — 30+ route definition files organized by feature
- Structured definitions with middleware stacking, path parameters, and validation
- Tenant middleware enforced at the registry level

## Playwright E2E Tests for Admin UI

Full browser-based test suite with dedicated infrastructure.

- 30+ spec files — setup, login, tenant CRUD/isolation, service keys, impersonation, edge functions, jobs, chatbots, SSO
- 3-phase pipeline — setup > provisioning > E2E against a dedicated fluxbase_playwright database
- Rich test helpers — API, DB, mailhog, selectors, shared fixtures
- make test-e2e-ui and variants (headed, debug, dev)

## Security & Reliability

- Adaptive Trust System — risk-based CAPTCHA verification
- Service key revocation and rotation
- OAuth state persistence for multi-instance deployments
- Idempotency keys for mutations
- Per-endpoint body size limits
- Per-user TOTP rate limiting with encrypted secrets
- Security hardening — OTP/invitation hashing, sensitive env var blocking in edge functions, pool mutex, advisory locks on migrations

## AI & Knowledge Bases

- Knowledge graph with document relationships and graph-based retrieval
- Chatbot MCP tool integration — chatbots invoke custom MCP tools
- Custom MCP tools with full SDK access and @fluxbase:namespace annotations
- MCP tools management page in admin dashboard
- ReAct reasoning for chatbots
- Knowledge base namespaces

## Multi-Backend Logging

- PostgreSQL, TimescaleDB (with compression), S3/MinIO, Loki, Elasticsearch, Clickhouse
- Configurable batching, flush intervals, and retention policies

## Codebase

- Fiber v2 → v3 migration
- npm → bun migration for all TypeScript packages
- Centralized packages — internal/errors/, internal/sync/, internal/scheduler/, internal/keys/, internal/loader/, internal/util/
- Astro v6 migration for docs site
- Go 1.25 → 1.26
- Pre-commit hooks enforcing go fmt, golangci-lint, and TypeScript type-check

### Pull Requests

- fix: Update settings defaults (#206) @bartcode
- fix: add viper config fallback to settings cache so features default to enabled (#205) @bartcode
- docs: fix middleware ordering, feature grid, CLI tenant flag, and cleanup (#204) @bartcode
- fix(tenancy): harden FDW schema import against transient connection failures (#203) @bartcode
- chore(deps): update module github.com/gofiber/fiber/v3 to v3.2.0 [security] (#202) @app/renovate
- docs: comprehensive documentation overhaul (#201) @bartcode
- fix: admin redirect, tenant repair, extension deps, schema graph, instance-level OAuth/SAML (#200) @bartcode
- refactor: simplify Go codebase for security and maintainability (#199) @bartcode
- refactor: centralize error handling, validation, scheduling, and storage patterns (#198) @bartcode
- chore(deps): update module github.com/jackc/pgx/v5 to v5.9.2 [security] (#197) @app/renovate
- chore(deps): update dependency astro to ^6.1.6 [security] (#196) @app/renovate
- fix: Resolve tenant isolation bugs. (#195) @bartcode
- fix: Enforce tenant isolation across all handlers and add repair endpoint (#194) @bartcode
- fix: Wire settings cache to fix email provider switcher not persisting (#193) @bartcode
- fix: Reorder jobs routes and fix RPC executions infinite loop (#192) @bartcode
- Fix tenant_service role authorization across all modules (#191) @bartcode
- Improve tenant-scoped impersonation for tenant admins (#190) @bartcode
- Add more UI tests and fix encountered bugs (#189) @bartcode
- fix: Wire tenant middleware into Jobs, Webhooks, RPC, and GraphQL routes (#188) @bartcode
- chore(deps): update dependency axios to ^1.15.0 [security] (#187) @app/renovate
- feat: Introduce Playwright testing and fix storage handler bug. (#186) @bartcode
- chore(deps): update dependency go to v1.26.1 (#185) @app/renovate
- Refactor API routes, use declarative schemas, add multi-tenancy, testing improvements, etc. (#183) @bartcode
- chore(deps): update go dependencies (#182) @app/renovate
- chore(deps): update module google.golang.org/grpc to v1.79.3 [security] (#181) @app/renovate
- chore(deps): update all dependencies (#180) @app/renovate
- chore(deps): update all dependencies (#175) @app/renovate
- chore(deps): update go dependencies (#174) @app/renovate

### Commits

**Features:**

- 5db3de0a fix: add viper config fallback to settings cache so features default to enabled (#205)
- ecaded60 feat: Introduce Playwright testing and fix storage handler bug. (#186)

**Bug Fixes:**

- b4304d16 fix: Update settings defaults (#206)
- 5db3de0a fix: add viper config fallback to settings cache so features default to enabled (#205)
- 1726ee44 fix(tenancy): harden FDW schema import against transient connection failures (#203)
- 9776aa70 fix: admin redirect, tenant repair, extension deps, schema graph, instance-level OAuth/SAML (#200)
- f781812a fix: Resolve tenant isolation bugs. (#195)
- 32073a6e fix: Enforce tenant isolation across all handlers and add repair endpoint (#194)
- 93f98693 fix: Wire settings cache to fix email provider switcher not persisting (#193)
- c290c38d fix: Reorder jobs routes and fix RPC executions infinite loop (#192)
- b58e9643 fix: Ensure routes for secrest are the same as before.
- 87d830a9 fix: Wire tenant middleware into Jobs, Webhooks, RPC, and GraphQL routes (#188)
- fb370921 fix: Add e2e tests and tenant_id column.
- 55f6cb1e fix: Resolve edge cases for multi-tenancy.
- e0c32e6c fix: Update service key middleware.
- 4d7cb1c5 fix: Resolve issues with declarative schemas.
- cdfc976a fix: Update renovate.json to use a minimum release age.

**Other Changes:**

- f949482a docs: fix voice, middleware ordering, feature grid, CLI tenant flag, and cleanup (#204)
- 4b44e727 docs: comprehensive documentation overhaul (#201)
- 8bd23fc1 refactor: simplify Go codebase for security and maintainability (#199)
- cee3f703 refactor: centralize error handling, validation, scheduling, and storage patterns (#198)
- f183d813 Fix tenant_service role authorization across all modules (#191)
- d7f64ce5 Improve tenant-scoped impersonation for tenant admins (#190)
- 6bae9992 docs: More alignment between docs and code.
- f8f2260a docs: Update docs to match the codebase.
- 7ed68f71 Add more UI tests and fix encountered bugs (#189)
- 46f37057 ci: Upgrade Helm setup.

### Stats

- **55** commits
- **3** contributors

## Installation

**Docker:**

```bash
docker pull ghcr.io/nimbleflux/fluxbase:2026.5.1
```

**NPM SDK:**

```bash
npm install @nimbleflux/fluxbase-sdk@2026.5.1
```

**Helm:**

```bash
helm install fluxbase oci://ghcr.io/nimbleflux/charts/fluxbase --version 2026.5.1
```

**CLI:**

```bash
curl -fsSL https://raw.githubusercontent.com/nimbleflux/fluxbase/main/install-cli.sh | bash -s -- v2026.5.1
```

---

_Release automatically generated by GitHub Actions_

---

## Smoke Test Results

| Component     | Status      |
| ------------- | ----------- |
| Docker Image  | ✅ Verified |
| NPM SDK       | ✅ Verified |
| NPM React SDK | ✅ Verified |

_Smoke tests completed at 2026-05-01 09:11:13 UTC_
