-- Enable Row Level Security on secrets tables
-- Secrets should only be accessible by service_role and dashboard_admin

-- Enable RLS on secrets table
ALTER TABLE functions.secrets ENABLE ROW LEVEL SECURITY;

-- Enable RLS on secret_versions table
ALTER TABLE functions.secret_versions ENABLE ROW LEVEL SECURITY;

-- Service role has full access to secrets
CREATE POLICY "Service role has full access to secrets"
ON functions.secrets
FOR ALL
TO service_role
USING (true)
WITH CHECK (true);

-- Dashboard admin has full access to secrets
CREATE POLICY "Dashboard admin has full access to secrets"
ON functions.secrets
FOR ALL
TO dashboard_admin
USING (true)
WITH CHECK (true);

-- Service role has full access to secret_versions
CREATE POLICY "Service role has full access to secret_versions"
ON functions.secret_versions
FOR ALL
TO service_role
USING (true)
WITH CHECK (true);

-- Dashboard admin has full access to secret_versions
CREATE POLICY "Dashboard admin has full access to secret_versions"
ON functions.secret_versions
FOR ALL
TO dashboard_admin
USING (true)
WITH CHECK (true);

-- Comments for documentation
COMMENT ON POLICY "Service role has full access to secrets" ON functions.secrets
IS 'Service role tokens can manage all secrets';
COMMENT ON POLICY "Dashboard admin has full access to secrets" ON functions.secrets
IS 'Dashboard administrators can manage all secrets';
