-- ============================================================================
-- FUNCTIONS SCHEMA RLS
-- ============================================================================
-- This file contains all Row Level Security (RLS) policies for the Functions schema.
-- These policies control access to edge functions, triggers, execution logs,
-- edge files, shared modules, and function dependencies.
--
-- Access Pattern:
-- - service_role: Full access to all tables (internal system operations)
-- - dashboard_admin: Full access to all tables (admin operations)
-- - Authenticated users: Can manage their own functions (created_by match)
-- - Public functions: Anyone can read enabled public functions
-- ============================================================================

-- ============================================================================
-- EDGE FUNCTIONS
-- ============================================================================
ALTER TABLE functions.edge_functions ENABLE ROW LEVEL SECURITY;
ALTER TABLE functions.edge_functions FORCE ROW LEVEL SECURITY;

-- Drop old policy
DROP POLICY IF EXISTS functions_edge_functions_policy ON functions.edge_functions;

-- Service role: full access
DROP POLICY IF EXISTS functions_edge_functions_service ON functions.edge_functions;
CREATE POLICY functions_edge_functions_service ON functions.edge_functions
    FOR ALL
    USING (auth.current_user_role() = 'service_role')
    WITH CHECK (auth.current_user_role() = 'service_role');

-- Dashboard admin: full access
DROP POLICY IF EXISTS functions_edge_functions_admin ON functions.edge_functions;
CREATE POLICY functions_edge_functions_admin ON functions.edge_functions
    FOR ALL
    USING (auth.current_user_role() = 'dashboard_admin')
    WITH CHECK (auth.current_user_role() = 'dashboard_admin');

-- Owner: can manage their own functions
DROP POLICY IF EXISTS functions_edge_functions_owner ON functions.edge_functions;
CREATE POLICY functions_edge_functions_owner ON functions.edge_functions
    FOR ALL
    USING (auth.current_user_id() IS NOT NULL AND created_by = auth.current_user_id())
    WITH CHECK (auth.current_user_id() IS NOT NULL AND created_by = auth.current_user_id());

-- Public read: anyone can read enabled public functions
DROP POLICY IF EXISTS functions_edge_functions_public_read ON functions.edge_functions;
CREATE POLICY functions_edge_functions_public_read ON functions.edge_functions
    FOR SELECT
    USING (is_public = true AND enabled = true);

-- ============================================================================
-- EDGE TRIGGERS
-- ============================================================================
ALTER TABLE functions.edge_triggers ENABLE ROW LEVEL SECURITY;
ALTER TABLE functions.edge_triggers FORCE ROW LEVEL SECURITY;

-- Drop old policy
DROP POLICY IF EXISTS functions_edge_triggers_policy ON functions.edge_triggers;

-- Service role: full access
DROP POLICY IF EXISTS functions_edge_triggers_service ON functions.edge_triggers;
CREATE POLICY functions_edge_triggers_service ON functions.edge_triggers
    FOR ALL
    USING (auth.current_user_role() = 'service_role')
    WITH CHECK (auth.current_user_role() = 'service_role');

-- Dashboard admin: full access
DROP POLICY IF EXISTS functions_edge_triggers_admin ON functions.edge_triggers;
CREATE POLICY functions_edge_triggers_admin ON functions.edge_triggers
    FOR ALL
    USING (auth.current_user_role() = 'dashboard_admin')
    WITH CHECK (auth.current_user_role() = 'dashboard_admin');

-- Owner: can manage triggers for their own functions (inherit via function_id)
DROP POLICY IF EXISTS functions_edge_triggers_owner ON functions.edge_triggers;
CREATE POLICY functions_edge_triggers_owner ON functions.edge_triggers
    FOR ALL
    USING (
        auth.current_user_id() IS NOT NULL
        AND EXISTS (
            SELECT 1 FROM functions.edge_functions ef
            WHERE ef.id = function_id AND ef.created_by = auth.current_user_id()
        )
    )
    WITH CHECK (
        auth.current_user_id() IS NOT NULL
        AND EXISTS (
            SELECT 1 FROM functions.edge_functions ef
            WHERE ef.id = function_id AND ef.created_by = auth.current_user_id()
        )
    );

-- ============================================================================
-- EDGE EXECUTIONS
-- ============================================================================
ALTER TABLE functions.edge_executions ENABLE ROW LEVEL SECURITY;
ALTER TABLE functions.edge_executions FORCE ROW LEVEL SECURITY;

-- Drop old policy
DROP POLICY IF EXISTS functions_edge_executions_policy ON functions.edge_executions;

-- Service role: full access
DROP POLICY IF EXISTS functions_edge_executions_service ON functions.edge_executions;
CREATE POLICY functions_edge_executions_service ON functions.edge_executions
    FOR ALL
    USING (auth.current_user_role() = 'service_role')
    WITH CHECK (auth.current_user_role() = 'service_role');

-- Dashboard admin: full access
DROP POLICY IF EXISTS functions_edge_executions_admin ON functions.edge_executions;
CREATE POLICY functions_edge_executions_admin ON functions.edge_executions
    FOR ALL
    USING (auth.current_user_role() = 'dashboard_admin')
    WITH CHECK (auth.current_user_role() = 'dashboard_admin');

-- Owner: can view executions of their own functions (SELECT only - executions are system-created)
DROP POLICY IF EXISTS functions_edge_executions_owner ON functions.edge_executions;
CREATE POLICY functions_edge_executions_owner ON functions.edge_executions
    FOR SELECT
    USING (
        auth.current_user_id() IS NOT NULL
        AND EXISTS (
            SELECT 1 FROM functions.edge_functions ef
            WHERE ef.id = function_id AND ef.created_by = auth.current_user_id()
        )
    );

-- ============================================================================
-- EDGE FILES
-- ============================================================================
ALTER TABLE functions.edge_files ENABLE ROW LEVEL SECURITY;
ALTER TABLE functions.edge_files FORCE ROW LEVEL SECURITY;

-- Service role: full access
DROP POLICY IF EXISTS functions_edge_files_service ON functions.edge_files;
CREATE POLICY functions_edge_files_service ON functions.edge_files
    FOR ALL
    USING (auth.current_user_role() = 'service_role')
    WITH CHECK (auth.current_user_role() = 'service_role');

-- Dashboard admin: full access
DROP POLICY IF EXISTS functions_edge_files_admin ON functions.edge_files;
CREATE POLICY functions_edge_files_admin ON functions.edge_files
    FOR ALL
    USING (auth.current_user_role() = 'dashboard_admin')
    WITH CHECK (auth.current_user_role() = 'dashboard_admin');

-- Owner: can manage files for their own functions (inherit via function_id)
DROP POLICY IF EXISTS functions_edge_files_owner ON functions.edge_files;
CREATE POLICY functions_edge_files_owner ON functions.edge_files
    FOR ALL
    USING (
        auth.current_user_id() IS NOT NULL
        AND EXISTS (
            SELECT 1 FROM functions.edge_functions ef
            WHERE ef.id = function_id AND ef.created_by = auth.current_user_id()
        )
    )
    WITH CHECK (
        auth.current_user_id() IS NOT NULL
        AND EXISTS (
            SELECT 1 FROM functions.edge_functions ef
            WHERE ef.id = function_id AND ef.created_by = auth.current_user_id()
        )
    );

-- ============================================================================
-- SHARED MODULES
-- ============================================================================
ALTER TABLE functions.shared_modules ENABLE ROW LEVEL SECURITY;
ALTER TABLE functions.shared_modules FORCE ROW LEVEL SECURITY;

-- Service role: full access
DROP POLICY IF EXISTS functions_shared_modules_service ON functions.shared_modules;
CREATE POLICY functions_shared_modules_service ON functions.shared_modules
    FOR ALL
    USING (auth.current_user_role() = 'service_role')
    WITH CHECK (auth.current_user_role() = 'service_role');

-- Dashboard admin: full access
DROP POLICY IF EXISTS functions_shared_modules_admin ON functions.shared_modules;
CREATE POLICY functions_shared_modules_admin ON functions.shared_modules
    FOR ALL
    USING (auth.current_user_role() = 'dashboard_admin')
    WITH CHECK (auth.current_user_role() = 'dashboard_admin');

-- Owner: can manage their own shared modules
DROP POLICY IF EXISTS functions_shared_modules_owner ON functions.shared_modules;
CREATE POLICY functions_shared_modules_owner ON functions.shared_modules
    FOR ALL
    USING (auth.current_user_id() IS NOT NULL AND created_by = auth.current_user_id())
    WITH CHECK (auth.current_user_id() IS NOT NULL AND created_by = auth.current_user_id());

-- Authenticated users: can read all shared modules (they're shared by design)
DROP POLICY IF EXISTS functions_shared_modules_read ON functions.shared_modules;
CREATE POLICY functions_shared_modules_read ON functions.shared_modules
    FOR SELECT
    USING (auth.current_user_id() IS NOT NULL);

-- ============================================================================
-- FUNCTION DEPENDENCIES
-- ============================================================================
ALTER TABLE functions.function_dependencies ENABLE ROW LEVEL SECURITY;
ALTER TABLE functions.function_dependencies FORCE ROW LEVEL SECURITY;

-- Service role: full access
DROP POLICY IF EXISTS functions_dependencies_service ON functions.function_dependencies;
CREATE POLICY functions_dependencies_service ON functions.function_dependencies
    FOR ALL
    USING (auth.current_user_role() = 'service_role')
    WITH CHECK (auth.current_user_role() = 'service_role');

-- Dashboard admin: full access
DROP POLICY IF EXISTS functions_dependencies_admin ON functions.function_dependencies;
CREATE POLICY functions_dependencies_admin ON functions.function_dependencies
    FOR ALL
    USING (auth.current_user_role() = 'dashboard_admin')
    WITH CHECK (auth.current_user_role() = 'dashboard_admin');

-- Owner: can manage dependencies for their own functions (inherit via function_id)
DROP POLICY IF EXISTS functions_dependencies_owner ON functions.function_dependencies;
CREATE POLICY functions_dependencies_owner ON functions.function_dependencies
    FOR ALL
    USING (
        auth.current_user_id() IS NOT NULL
        AND EXISTS (
            SELECT 1 FROM functions.edge_functions ef
            WHERE ef.id = function_id AND ef.created_by = auth.current_user_id()
        )
    )
    WITH CHECK (
        auth.current_user_id() IS NOT NULL
        AND EXISTS (
            SELECT 1 FROM functions.edge_functions ef
            WHERE ef.id = function_id AND ef.created_by = auth.current_user_id()
        )
    );

-- ============================================================================
-- EXECUTION LOGS
-- ============================================================================
ALTER TABLE functions.execution_logs ENABLE ROW LEVEL SECURITY;
ALTER TABLE functions.execution_logs FORCE ROW LEVEL SECURITY;

-- Service role: full access (logs are system-generated)
DROP POLICY IF EXISTS functions_execution_logs_service_all ON functions.execution_logs;
CREATE POLICY functions_execution_logs_service_all ON functions.execution_logs
    FOR ALL
    USING (auth.current_user_role() = 'service_role')
    WITH CHECK (auth.current_user_role() = 'service_role');

-- Dashboard admin: full access
DROP POLICY IF EXISTS functions_execution_logs_admin ON functions.execution_logs;
CREATE POLICY functions_execution_logs_admin ON functions.execution_logs
    FOR ALL
    USING (auth.current_user_role() = 'dashboard_admin')
    WITH CHECK (auth.current_user_role() = 'dashboard_admin');

-- Owner: can view logs for executions of their own functions (SELECT only)
DROP POLICY IF EXISTS functions_execution_logs_owner ON functions.execution_logs;
CREATE POLICY functions_execution_logs_owner ON functions.execution_logs
    FOR SELECT
    USING (
        auth.current_user_id() IS NOT NULL
        AND EXISTS (
            SELECT 1 FROM functions.edge_executions ee
            JOIN functions.edge_functions ef ON ef.id = ee.function_id
            WHERE ee.id = execution_id AND ef.created_by = auth.current_user_id()
        )
    );

--
-- PERFORMANCE INDEXES FOR RLS POLICIES
--

-- Index for auth.api_keys RLS policy (filtering by user_id)
CREATE INDEX IF NOT EXISTS idx_api_keys_user_id ON auth.api_keys(user_id);

-- Index for auth.api_key_usage RLS policy (filtering by api_key_id)
CREATE INDEX IF NOT EXISTS idx_api_key_usage_api_key_id ON auth.api_key_usage(api_key_id);

-- Index for auth.sessions RLS policy (filtering by user_id)
CREATE INDEX IF NOT EXISTS idx_auth_sessions_user_id ON auth.sessions(user_id);

-- Index for auth.webhook_deliveries RLS policy (filtering by webhook_id)
CREATE INDEX IF NOT EXISTS idx_webhook_deliveries_webhook_id ON auth.webhook_deliveries(webhook_id);

-- Indexes for auth.impersonation_sessions RLS policy (filtering by admin_user_id and target_user_id)
CREATE INDEX IF NOT EXISTS idx_impersonation_sessions_admin_user_id ON auth.impersonation_sessions(admin_user_id);
CREATE INDEX IF NOT EXISTS idx_impersonation_sessions_target_user_id ON auth.impersonation_sessions(target_user_id);
