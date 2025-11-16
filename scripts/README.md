# Database Setup Scripts

## setup-dev-user.sql

This script grants all necessary permissions to the `fluxbase_app` database user for local development and testing.

### Why is this needed?

For proper Row Level Security (RLS) enforcement, the application must use a **non-superuser** database connection. PostgreSQL superusers (like `postgres`) bypass RLS policies, which would make RLS tests ineffective.

The `fluxbase_app` user is a regular (non-superuser) user that:
- Has all necessary permissions to run migrations and operate the application
- **Cannot** bypass RLS policies, ensuring they work correctly
- Owns all tables and sequences (required for operations like upsert)

### When to run this script

Run this script after setting up your local development database and running migrations:

```bash
# Run migrations first
migrate -path internal/database/migrations -database "postgres://postgres:postgres@localhost:5432/fluxbase_dev?sslmode=disable" up

# Then grant permissions to fluxbase_app
PGPASSWORD=postgres psql -h localhost -U postgres -d fluxbase_dev -f scripts/setup-dev-user.sql
```

### What it does

1. Grants CREATE permission on the database
2. Grants ALL permissions on all schemas (public, auth, storage, etc.)
3. Grants ALL PRIVILEGES on all tables, sequences, and functions
4. Changes ownership of all tables and sequences to `fluxbase_app`
5. Changes ownership of all schemas to `fluxbase_app`

### Configuration

After running this script, update your `.env` file to use the `fluxbase_app` user:

```env
FLUXBASE_DATABASE_USER=fluxbase_app
FLUXBASE_DATABASE_PASSWORD=fluxbase_app_password
FLUXBASE_DATABASE_DATABASE=fluxbase_dev
```

This ensures RLS works correctly in both development and tests.

---

## Git Hooks

### Pre-commit Hook

The pre-commit hook automatically runs before each commit to catch issues early and reduce CI failures.

**Checks performed:**
1. **Go Formatting** - Runs `go fmt ./...` on all staged Go files and auto-stages formatting changes
2. **Admin TypeScript** - Type checks admin UI code (`cd admin && npm run type-check`)
3. **SDK TypeScript** - Type checks SDK code (`cd sdk && npm run type-check`)
4. **SDK React TypeScript** - Type checks React SDK code (`cd sdk-react && npm run type-check`)

**Installation:**
```bash
# Automatically installed with:
make setup-dev

# Or install manually:
make install-hooks
```

**Usage:**
The hook runs automatically on `git commit`. If any checks fail, the commit is blocked.

To bypass the hook (not recommended):
```bash
git commit --no-verify
```

**Uninstall:**
```bash
make uninstall-hooks
```

**Benefits:**
- Catches formatting and type errors before they reach CI
- Auto-formats Go code
- Reduces failed CI pipeline runs
- Provides fast feedback during development
