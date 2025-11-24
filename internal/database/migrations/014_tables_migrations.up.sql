--
-- MIGRATIONS SCHEMA TABLES
-- API-managed user migrations
--

-- Main migrations table
CREATE TABLE IF NOT EXISTS migrations.migrations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    namespace TEXT NOT NULL DEFAULT 'default',
    name TEXT NOT NULL,
    description TEXT,
    up_sql TEXT NOT NULL,
    down_sql TEXT,
    version INTEGER DEFAULT 1,
    status TEXT DEFAULT 'pending',  -- pending, applied, failed, rolled_back
    created_by UUID REFERENCES auth.users(id) ON DELETE SET NULL,
    applied_by UUID REFERENCES auth.users(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ DEFAULT NOW() NOT NULL,
    updated_at TIMESTAMPTZ DEFAULT NOW() NOT NULL,
    applied_at TIMESTAMPTZ,
    rolled_back_at TIMESTAMPTZ,
    CONSTRAINT unique_migration_namespace UNIQUE (namespace, name),
    CONSTRAINT valid_status CHECK (status IN ('pending', 'applied', 'failed', 'rolled_back'))
);

-- Execution history for audit trail
CREATE TABLE IF NOT EXISTS migrations.execution_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    migration_id UUID NOT NULL REFERENCES migrations.migrations(id) ON DELETE CASCADE,
    action TEXT NOT NULL,  -- apply, rollback
    status TEXT NOT NULL,  -- success, failed
    duration_ms INTEGER,
    error_message TEXT,
    logs TEXT,
    executed_at TIMESTAMPTZ DEFAULT NOW() NOT NULL,
    executed_by UUID REFERENCES auth.users(id) ON DELETE SET NULL,
    CONSTRAINT valid_action CHECK (action IN ('apply', 'rollback')),
    CONSTRAINT valid_execution_status CHECK (status IN ('success', 'failed'))
);

-- Indexes for performance
CREATE INDEX IF NOT EXISTS idx_migrations_namespace ON migrations.migrations(namespace);
CREATE INDEX IF NOT EXISTS idx_migrations_status ON migrations.migrations(status);
CREATE INDEX IF NOT EXISTS idx_migrations_namespace_status ON migrations.migrations(namespace, status);
CREATE INDEX IF NOT EXISTS idx_execution_logs_migration ON migrations.execution_logs(migration_id);
CREATE INDEX IF NOT EXISTS idx_execution_logs_executed_at ON migrations.execution_logs(executed_at DESC);

-- Comments for documentation
COMMENT ON SCHEMA migrations IS 'API-managed user migrations (separate from system migrations)';
COMMENT ON TABLE migrations.migrations IS 'User-defined migrations managed via API, stored in database instead of filesystem';
COMMENT ON COLUMN migrations.migrations.namespace IS 'Namespace for multi-tenancy isolation (e.g., dev, staging, prod, app1, app2)';
COMMENT ON COLUMN migrations.migrations.name IS 'Migration name, should follow convention like 001_description for ordering';
COMMENT ON COLUMN migrations.migrations.up_sql IS 'SQL to apply the migration';
COMMENT ON COLUMN migrations.migrations.down_sql IS 'SQL to rollback the migration (optional)';
COMMENT ON COLUMN migrations.migrations.status IS 'Current status: pending (not applied), applied (successful), failed (error), rolled_back';
COMMENT ON TABLE migrations.execution_logs IS 'Audit log of all migration apply/rollback attempts';
