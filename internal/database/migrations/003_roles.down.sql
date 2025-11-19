-- Rollback Fluxbase Database Roles
-- Table-level revokes are in migration 022

-- Revoke role grants from application users
REVOKE anon FROM fluxbase_app;
REVOKE authenticated FROM fluxbase_app;
REVOKE service_role FROM fluxbase_app;

DO $$
BEGIN
    IF EXISTS (SELECT FROM pg_catalog.pg_roles WHERE rolname = 'fluxbase_rls_test') THEN
        REVOKE anon FROM fluxbase_rls_test;
        REVOKE authenticated FROM fluxbase_rls_test;
        REVOKE service_role FROM fluxbase_rls_test;
    END IF;
END
$$;

-- Revoke schema permissions
REVOKE USAGE ON SCHEMA auth FROM anon, authenticated, service_role;
REVOKE USAGE ON SCHEMA app FROM anon, authenticated, service_role;
REVOKE USAGE ON SCHEMA storage FROM anon, authenticated, service_role;
REVOKE USAGE ON SCHEMA functions FROM anon, authenticated, service_role;
REVOKE USAGE ON SCHEMA realtime FROM anon, authenticated, service_role;
REVOKE USAGE ON SCHEMA dashboard FROM anon, authenticated, service_role;
REVOKE USAGE ON SCHEMA public FROM anon, authenticated, service_role;

-- Drop roles
DROP ROLE IF EXISTS service_role;
DROP ROLE IF EXISTS authenticated;
DROP ROLE IF EXISTS anon;
