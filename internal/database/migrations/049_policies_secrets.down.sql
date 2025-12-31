-- Revert RLS policies on secrets tables

-- Drop policies on secrets table
DROP POLICY IF EXISTS secrets_service_and_admin_policy ON functions.secrets;

-- Drop policies on secret_versions table
DROP POLICY IF EXISTS secret_versions_service_and_admin_policy ON functions.secret_versions;

-- Disable RLS (revert to previous state)
ALTER TABLE functions.secrets DISABLE ROW LEVEL SECURITY;
ALTER TABLE functions.secret_versions DISABLE ROW LEVEL SECURITY;
