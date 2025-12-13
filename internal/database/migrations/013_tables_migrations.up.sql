--
-- MIGRATIONS SCHEMA TABLES
-- All migration tracking tables
--

-- ============================================================================
-- SYSTEM MIGRATION TRACKING (golang-migrate)
-- ============================================================================

-- Fluxbase system migrations (embedded in binary)
CREATE TABLE IF NOT EXISTS migrations.fluxbase (
    version BIGINT NOT NULL PRIMARY KEY,
    dirty BOOLEAN NOT NULL
);

COMMENT ON TABLE migrations.fluxbase IS 'Tracks Fluxbase system migration versions (managed by golang-migrate)';

-- ============================================================================
-- APPLICATION MIGRATIONS (filesystem + API)
-- ============================================================================

-- Main migrations table for all user-facing migrations
-- Filesystem migrations use namespace='filesystem'
-- API migrations use custom namespaces (default, staging, prod, etc.)
CREATE TABLE IF NOT EXISTS migrations.app (
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
    migration_id UUID NOT NULL REFERENCES migrations.app(id) ON DELETE CASCADE,
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

-- ============================================================================
-- INDEXES
-- ============================================================================

CREATE INDEX IF NOT EXISTS idx_migrations_app_namespace ON migrations.app(namespace);
CREATE INDEX IF NOT EXISTS idx_migrations_app_status ON migrations.app(status);
CREATE INDEX IF NOT EXISTS idx_migrations_app_namespace_status ON migrations.app(namespace, status);
CREATE INDEX IF NOT EXISTS idx_execution_logs_migration ON migrations.execution_logs(migration_id);
CREATE INDEX IF NOT EXISTS idx_execution_logs_executed_at ON migrations.execution_logs(executed_at DESC);

-- ============================================================================
-- COMMENTS
-- ============================================================================

COMMENT ON TABLE migrations.app IS 'All user-facing migrations (filesystem and API-managed)';
COMMENT ON COLUMN migrations.app.namespace IS 'Namespace for isolation: filesystem for local files, or custom (default, staging, prod, etc.) for API';
COMMENT ON COLUMN migrations.app.name IS 'Migration name, should follow convention like 001_description for ordering';
COMMENT ON COLUMN migrations.app.up_sql IS 'SQL to apply the migration';
COMMENT ON COLUMN migrations.app.down_sql IS 'SQL to rollback the migration (optional)';
COMMENT ON COLUMN migrations.app.status IS 'Current status: pending (not applied), applied (successful), failed (error), rolled_back';
COMMENT ON TABLE migrations.execution_logs IS 'Audit log of all migration apply/rollback attempts';
