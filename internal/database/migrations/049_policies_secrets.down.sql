-- Revert RLS policies on secrets tables

-- Drop policies on secrets table
DROP POLICY IF EXISTS "Service role has full access to secrets" ON functions.secrets;
DROP POLICY IF EXISTS "Dashboard admin has full access to secrets" ON functions.secrets;

-- Drop policies on secret_versions table
DROP POLICY IF EXISTS "Service role has full access to secret_versions" ON functions.secret_versions;
DROP POLICY IF EXISTS "Dashboard admin has full access to secret_versions" ON functions.secret_versions;

-- Disable RLS (revert to previous state)
ALTER TABLE functions.secrets DISABLE ROW LEVEL SECURITY;
ALTER TABLE functions.secret_versions DISABLE ROW LEVEL SECURITY;
