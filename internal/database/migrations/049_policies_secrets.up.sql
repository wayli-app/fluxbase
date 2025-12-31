-- Enable Row Level Security on secrets tables
-- Secrets should only be accessible by service_role and dashboard_admin

-- Enable RLS on secrets table
ALTER TABLE functions.secrets ENABLE ROW LEVEL SECURITY;
ALTER TABLE functions.secrets FORCE ROW LEVEL SECURITY;

-- Enable RLS on secret_versions table
ALTER TABLE functions.secret_versions ENABLE ROW LEVEL SECURITY;
ALTER TABLE functions.secret_versions FORCE ROW LEVEL SECURITY;

-- Service role and dashboard admin have full access to secrets
DROP POLICY IF EXISTS secrets_service_and_admin_policy ON functions.secrets;
CREATE POLICY secrets_service_and_admin_policy ON functions.secrets
FOR ALL
USING (
    auth.current_user_role() = 'service_role'
    OR auth.current_user_role() = 'dashboard_admin'
)
WITH CHECK (
    auth.current_user_role() = 'service_role'
    OR auth.current_user_role() = 'dashboard_admin'
);

-- Service role and dashboard admin have full access to secret_versions
DROP POLICY IF EXISTS secret_versions_service_and_admin_policy ON functions.secret_versions;
CREATE POLICY secret_versions_service_and_admin_policy ON functions.secret_versions
FOR ALL
USING (
    auth.current_user_role() = 'service_role'
    OR auth.current_user_role() = 'dashboard_admin'
)
WITH CHECK (
    auth.current_user_role() = 'service_role'
    OR auth.current_user_role() = 'dashboard_admin'
);

-- Comments for documentation
COMMENT ON POLICY secrets_service_and_admin_policy ON functions.secrets
IS 'Service role and dashboard administrators can manage all secrets';
COMMENT ON POLICY secret_versions_service_and_admin_policy ON functions.secret_versions
IS 'Service role and dashboard administrators can manage all secret versions';
