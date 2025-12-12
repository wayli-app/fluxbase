-- ============================================================================
-- FUNCTIONS SCHEMA RLS
-- ============================================================================
-- This file contains all Row Level Security (RLS) policies for the Functions schema.
-- These policies control access to edge functions, triggers, and execution logs.
-- ============================================================================

-- Edge functions
ALTER TABLE functions.edge_functions ENABLE ROW LEVEL SECURITY;
ALTER TABLE functions.edge_functions FORCE ROW LEVEL SECURITY;

DROP POLICY IF EXISTS functions_edge_functions_policy ON functions.edge_functions;
CREATE POLICY functions_edge_functions_policy ON functions.edge_functions
    FOR ALL
    USING (auth.current_user_role() = 'dashboard_admin');

-- Edge triggers
ALTER TABLE functions.edge_triggers ENABLE ROW LEVEL SECURITY;
ALTER TABLE functions.edge_triggers FORCE ROW LEVEL SECURITY;

DROP POLICY IF EXISTS functions_edge_triggers_policy ON functions.edge_triggers;
CREATE POLICY functions_edge_triggers_policy ON functions.edge_triggers
    FOR ALL
    USING (auth.current_user_role() = 'dashboard_admin');

-- Edge executions
ALTER TABLE functions.edge_executions ENABLE ROW LEVEL SECURITY;
ALTER TABLE functions.edge_executions FORCE ROW LEVEL SECURITY;

DROP POLICY IF EXISTS functions_edge_executions_policy ON functions.edge_executions;
CREATE POLICY functions_edge_executions_policy ON functions.edge_executions
    FOR ALL
    USING (auth.current_user_role() = 'dashboard_admin');

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
