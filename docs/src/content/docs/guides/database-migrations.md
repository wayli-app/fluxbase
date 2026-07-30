---
title: Database Migrations
description: Manage database schema changes in Fluxbase with declarative schema management for platform tables and your choice of declarative or imperative schema for your application tables.
---

Fluxbase uses a **declarative schema management** approach for internal platform tables, and lets you choose between **declarative** or **imperative** schema management for your own application tables.

## Overview

Fluxbase provides three schema mechanisms:

1. **Declarative Schema (Internal)** - Fluxbase platform tables managed automatically via bootstrap + pgschema (always on)
2. **Declarative App Schema (Optional)** - Your own application tables managed declaratively from a desired-state SQL file (opt-in)
3. **User Migrations (Optional)** - Your custom application tables via imperative SQL migration files

You pick **one** of (2) or (3) for a given `(namespace, schema)` — do not use both against the same schema.

```mermaid
graph LR
    subgraph "Internal Schema (Declarative)"
        B1[bootstrap.sql] --> B2[pgschema]
        B2 --> B3[Platform Tables]
    end

    subgraph "App Schema (choose one)"
        D1[public.sql desired-state] --> D2[platform.app_schemas]
        D2 --> D3[App Tables via pgschema diff]
        U1[Migration Files] --> U2[platform.migrations]
        U2 --> U3[App Tables imperative]
    end

    B3 -.-> DB[(PostgreSQL)]
    D3 -.-> DB
    U3 -.-> DB
```

### Which app-schema mode should I pick?

| Use Declarative App Schema | Use User Migrations (Imperative) |
| -------------------------- | -------------------------------- |
| You want the schema file to be the single source of truth | You need ordered, reversible migrations |
| You want automatic drift reconciliation on every sync | You need data-transformations / backfills between versions |
| You want the deployed schema to be readable at a glance | You prefer explicit up/down rollback files |
| You want a simpler mental model (no version numbers) | You have complex, stepwise data migrations |

## Declarative Schema (Internal)

**Purpose:** Platform infrastructure managed automatically by Fluxbase

The internal Fluxbase schema (auth, storage, functions, jobs, etc.) is managed declaratively:

- **bootstrap.sql** - Creates schemas, extensions, roles, and default privileges (idempotent)
- **pgschema** - Compares desired schema files to actual database state and applies diffs

**Tracking:** `platform.declarative_state` table stores schema fingerprint

**Execution:** Automatically applied on server startup

**Schema files location:** `internal/database/schema/schemas/`

### How It Works

1. On startup, Fluxbase runs `bootstrap.sql` to ensure schemas and roles exist
2. The pgschema tool compares schema files to the actual database
3. Any differences are applied automatically
4. The schema fingerprint is stored for drift detection

### Benefits

- **No version numbers** - Schema is the source of truth
- **Automatic drift detection** - Can detect if database was modified outside Fluxbase
- **Safe by default** - Destructive changes require explicit approval
- **Works with CI/CD** - Schema is applied automatically, no migration commands needed

## Declarative App Schema (Optional)

**Purpose:** Your own application tables managed declaratively from a desired-state SQL file

The declarative app schema applies the same proven pgschema engine used for internal
platform tables to **your** application schema (e.g. `public`). You commit a single
desired-state SQL file and sync it; Fluxbase stores the content and reconciles the live
database to match on every sync. This is an opt-in alternative to imperative user
migrations.

- **Schema file:** `fluxbase/schema/public.sql` (or `<dir>/<schema>.sql`) — a pgschema-style desired-state dump
- **Storage:** `platform.app_schemas` (content + fingerprint) and `platform.app_schema_state` (last applied)
- **Execution:** Applied on startup when `database.declarative_app_schema.enabled=true`; also runnable on demand
- **Safety:** Destructive changes (`DROP`, etc.) are blocked by default and require `allow_destructive=true`

### Enabling it

Set the following on the Fluxbase server (environment variables use the `FLUXBASE_` prefix):

```yaml
database:
  declarative_app_schema:
    enabled: true
    schema: public
    namespaces: ["wayli"]   # restrict which synced namespaces apply on startup; empty = all
    on_startup: true
    allow_destructive: false
```

### Syncing your schema

Commit a desired-state SQL file and sync it with the CLI:

```bash
fluxbase schema sync --dir fluxbase/schema --namespace wayli
# Reads fluxbase/schema/public.sql, stores it, and applies the diff
```

Other commands:

```bash
fluxbase schema status                       # list all stored app schemas
fluxbase schema status --namespace wayli     # status for one namespace
fluxbase schema plan --namespace wayli       # preview pending changes
fluxbase schema validate --namespace wayli --fail-on-drift   # CI drift check
fluxbase schema apply --namespace wayli      # re-apply stored content (reconcile drift)
```

### Generating the schema file

To adopt declarative schema on an existing app, dump the current live schema so the
first sync is a zero-diff no-op:

```bash
pgschema dump --host $DB_HOST --port 5432 --user $DB_USER --db $DB --schema public > fluxbase/schema/public.sql
```

Then edit the file to keep only your application objects (strip Fluxbase-owned schemas
like `auth`, `storage`, `platform`, `app`). After the first clean sync, future edits to
the file are diffed and applied automatically.

> **Note:** Extensions (`CREATE EXTENSION`) cannot be managed by pgschema. Keep them in a
> separate bootstrap step (e.g. a tiny `extensions.sql` applied out-of-band), not in
> `public.sql`. Fluxbase's bootstrap already enables `uuid-ossp`, `pgcrypto`, `pg_trgm`,
> `btree_gin`, and `vector`; add any others (e.g. `postgis`) separately.

### Coexistence with imperative migrations

A `(namespace, schema)` should be owned by **one** mode. If Fluxbase detects that a
declaratively-managed namespace also has imperative migrations in `platform.migrations`,
it logs a warning on startup. To switch an app to declarative: stop running
`fluxbase migrations sync` for that namespace and remove its imperative migrations from
the sync path.

### Declarative App Schema API

| Endpoint                                  | Description                          |
| ----------------------------------------- | ------------------------------------ |
| `POST /api/v1/admin/app-schema/sync`      | Store schema content (+ apply)       |
| `POST /api/v1/admin/app-schema/apply`     | Apply already-stored content         |
| `POST /api/v1/admin/app-schema/plan`      | Preview pending changes              |
| `GET  /api/v1/admin/app-schema/validate`  | Check for drift                      |
| `GET  /api/v1/admin/app-schema/status`    | Status / list stored schemas         |
| `DELETE /api/v1/admin/app-schema`         | Remove stored content (no DB change) |

## User Migrations (Optional)

**Purpose:** Application-specific schema managed by you

User migrations allow you to add your own custom database schema using traditional imperative migration files.

**Tracking:** `platform.migrations` table

**Execution:** Run on startup if `FLUXBASE_DATABASE_USER_MIGRATIONS_PATH` is configured

**File format:** Standard golang-migrate format with `.up.sql` and `.down.sql` files

### When to Use User Migrations

| Use Declarative (Internal)  | Use User Migrations               |
| --------------------------- | --------------------------------- |
| Never (managed by Fluxbase) | Application tables                |
|                             | Custom indexes                    |
|                             | Data transformations              |
|                             | Business logic triggers           |
|                             | Application-specific RLS policies |

### Migration File Format

User migrations follow the standard golang-migrate format:

```text
001_create_users_table.up.sql
001_create_users_table.down.sql
002_add_timestamps.up.sql
002_add_timestamps.down.sql
```

Each migration has two files:

- **`.up.sql`** - Applied when migrating forward
- **`.down.sql`** - Applied when rolling back (optional but recommended)

### Migration Numbering

Migrations are executed in numerical order based on the prefix. Best practices:

- Use sequential numbering: `001`, `002`, `003`, etc.
- Zero-pad numbers for proper sorting
- Never reuse or skip numbers
- Never modify a migration that has already been applied

### Example Migration

**001_create_products_table.up.sql:**

```sql
-- Create products table in public schema
CREATE TABLE IF NOT EXISTS public.products (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL,
    description TEXT,
    price DECIMAL(10,2) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Add RLS policies
ALTER TABLE public.products ENABLE ROW LEVEL SECURITY;

-- Allow all authenticated users to read products
CREATE POLICY "Products are viewable by authenticated users"
    ON public.products
    FOR SELECT
    TO authenticated
    USING (true);

-- Allow only admins to insert/update/delete products
CREATE POLICY "Products are manageable by admins"
    ON public.products
    FOR ALL
    TO authenticated
    USING (auth.role() = 'admin')
    WITH CHECK (auth.role() = 'admin');
```

**001_create_products_table.down.sql:**

```sql
-- Drop the table (this will also drop policies)
DROP TABLE IF EXISTS public.products CASCADE;
```

## Configuration

### Docker Compose

To enable user migrations in Docker Compose:

1. Create a directory for your migrations:

```bash
mkdir -p deploy/migrations/user
```

2. Add your migration files to this directory

3. Update `docker-compose.yml`:

```yaml
services:
  fluxbase:
    environment:
      # Enable user migrations
      FLUXBASE_DATABASE_USER_MIGRATIONS_PATH: /migrations/user
    volumes:
      # Mount migrations directory (read-only)
      - ./migrations/user:/migrations/user:ro
```

4. Restart Fluxbase:

```bash
docker compose restart fluxbase
```

### Kubernetes (Helm)

To enable user migrations in Kubernetes:

1. Create a ConfigMap or PVC with your migration files

**Option A: Using ConfigMap (for small migrations):**

```bash
kubectl create configmap user-migrations \
  --from-file=migrations/user/ \
  -n fluxbase
```

**Option B: Using PVC (recommended for production):**

```yaml
# values.yaml
migrationsPersistence:
  enabled: true
  size: 100Mi
  storageClass: "" # Use cluster default

config:
  database:
    userMigrationsPath: /migrations/user
```

2. Install or upgrade the Helm chart:

```bash
helm upgrade --install fluxbase ./deploy/helm/fluxbase \
  --namespace fluxbase \
  --create-namespace \
  -f values.yaml
```

3. Copy your migration files to the PVC:

```bash
# Find a pod
POD_NAME=$(kubectl get pod -n fluxbase -l app.kubernetes.io/name=fluxbase -o jsonpath="{.items[0].metadata.name}")

# Copy migrations
kubectl cp migrations/user/ fluxbase/$POD_NAME:/migrations/user/
```

4. Restart the deployment:

```bash
kubectl rollout restart deployment/fluxbase -n fluxbase
```

### Environment Variables

You can configure user migrations via environment variables:

| Variable                  | Description                       | Default         |
| ------------------------- | --------------------------------- | --------------- |
| `FLUXBASE_DATABASE_USER_MIGRATIONS_PATH` | Path to user migrations directory | `""` (disabled) |

When `FLUXBASE_DATABASE_USER_MIGRATIONS_PATH` is empty or not set, user migrations are skipped.

## Startup Flow

When Fluxbase starts, schema is applied in this order:

1. **Bootstrap SQL** - Creates schemas, extensions, roles (idempotent)
2. **Declarative Schema** - Applies internal Fluxbase schema via pgschema
3. **User Migrations** - Applies your custom migrations (if configured)

### Logs

Migration progress is logged during startup:

```
INFO Running bootstrap SQL...
INFO Bootstrap completed successfully
INFO Applying declarative schema...
INFO Schema applied successfully
INFO Running user migrations... path=/migrations/user
INFO Migrations applied successfully source=user version=3
INFO Database schema management completed
```

## Local Development

For local development, Fluxbase provides Make commands:

### Database Reset Commands

```bash
# Partial reset - preserves user data in public schema
make db-reset

# Full reset - drops ALL schemas (WARNING: destroys all data)
make db-reset-full
```

After a reset, the bootstrap and declarative schema are applied automatically on the next server startup with `make dev`.

### User Migration Commands

If you have user migrations configured:

```bash
# Create new user migration
make migrate-create name=add_products
# Creates: migrations/XXX_add_products.up.sql and .down.sql

# Apply migrations
make migrate-up

# Rollback last migration
make migrate-down
```

**Note:** These commands are for user-provided migrations only. The internal Fluxbase schema is managed declaratively and applied automatically.

## Best Practices

### 1. Test Migrations Locally First

Always test migrations in a development environment before applying to production:

```bash
# Start local environment
docker compose up -d

# Check logs for migration success
docker compose logs fluxbase | grep -i migration
```

### 2. Use Transactions

Wrap DDL statements in transactions when possible:

```sql
BEGIN;

CREATE TABLE products (...);
CREATE INDEX IF NOT EXISTS idx_products_name ON products(name);

COMMIT;
```

### 3. Make Migrations Idempotent

Use conditional statements to make migrations safe to re-run:

```sql
-- Good: Uses IF NOT EXISTS
CREATE TABLE IF NOT EXISTS products (...);

-- Bad: Will fail if table exists
CREATE TABLE products (...);
```

### 4. Add Indexes Concurrently

For large tables, create indexes without locking:

```sql
-- Add indexes concurrently (won't block reads/writes)
CREATE INDEX CONCURRENTLY idx_products_category ON products(category);
```

### 5. Plan for Rollbacks

Always include `.down.sql` files to support rollback scenarios:

```sql
-- down.sql should reverse the up.sql changes
DROP INDEX IF EXISTS idx_products_category;
DROP TABLE IF EXISTS products;
```

### 6. Document Complex Migrations

Add comments explaining the purpose of complex migrations:

```sql
-- Migration: Add full-text search to products
-- Author: Your Name
-- Date: 2024-01-15
-- Reason: Enable product search functionality

ALTER TABLE products ADD COLUMN search_vector tsvector;

CREATE INDEX IF NOT EXISTS idx_products_search
  ON products
  USING gin(search_vector);
```

## Troubleshooting

### Migration Failed

If a migration fails partway through, check the logs and fix the issue:

```sql
-- Connect to database
psql -h localhost -U fluxbase -d fluxbase

-- Check migration state
SELECT * FROM platform.migrations WHERE status = 'failed';

-- After fixing the issue, mark as pending to retry
UPDATE platform.migrations SET status = 'pending', error_message = '' WHERE name = 'failed_migration';
```

### Migration Not Running

If your migration isn't being applied:

1. **Check file naming**: Ensure files follow the format `NNN_name.up.sql`
2. **Check file location**: Verify files are in the configured directory
3. **Check permissions**: Ensure Fluxbase can read the migration files
4. **Check logs**: Look for migration errors in Fluxbase logs
5. **Check configuration**: Verify `FLUXBASE_DATABASE_USER_MIGRATIONS_PATH` is set correctly

### Checking Migration Status

To see which user migrations have been applied:

```sql
-- Check user migrations
SELECT * FROM platform.migrations ORDER BY applied_at DESC;
```

### Checking Declarative Schema Status

To check the internal declarative schema status:

```sql
-- Check declarative state
SELECT * FROM platform.declarative_state;
```

Or use the admin API:

```bash
curl http://localhost:8080/api/v1/admin/internal-schema/status
```

## Advanced Topics

### Declarative Schema Management API

Fluxbase provides internal API endpoints for schema management:

| Endpoint                                     | Description                  |
| -------------------------------------------- | ---------------------------- |
| `GET /api/v1/admin/internal-schema/status`   | Check schema status          |
| `POST /api/v1/admin/internal-schema/plan`    | Preview pending changes      |
| `POST /api/v1/admin/internal-schema/apply`   | Apply schema changes         |
| `GET /api/v1/admin/internal-schema/validate` | Validate schema for drift    |
| `POST /api/v1/admin/internal-schema/dump`    | Dump current schema to files |

These endpoints are useful for CI/CD pipelines and manual schema inspection.

### Schema Drift Detection

Fluxbase can detect if the database schema has drifted from the expected state:

```bash
# Check for drift via API
curl http://localhost:8080/api/v1/admin/internal-schema/validate
```

If drift is detected, you can either:

1. Apply the declarative schema to update the database
2. Dump the current schema to update the schema files

### Running Migrations Separately

In production, you may want to run migrations separately from application startup:

1. Use the internal schema API to apply changes before deploying
2. Or use a separate init container in Kubernetes

## Related Resources

- [Row-Level Security Guide](/guides/row-level-security/)
- [Configuration Reference](/reference/configuration/)
- [Deployment Guides](/deployment/overview/)
