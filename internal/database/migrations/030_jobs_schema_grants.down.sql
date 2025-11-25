-- Revoke default privileges
ALTER DEFAULT PRIVILEGES IN SCHEMA jobs
    REVOKE SELECT ON TABLES FROM authenticated;

-- Revoke table grants
REVOKE SELECT ON ALL TABLES IN SCHEMA jobs FROM authenticated;

-- Revoke schema usage
REVOKE USAGE ON SCHEMA jobs FROM authenticated;
