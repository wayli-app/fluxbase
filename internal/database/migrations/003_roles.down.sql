-- Rollback Fluxbase Database Roles

-- Revoke schema permissions
REVOKE USAGE ON SCHEMA auth FROM anon, authenticated, service_role;
REVOKE USAGE ON SCHEMA app FROM anon, authenticated, service_role;
REVOKE USAGE ON SCHEMA storage FROM anon, authenticated, service_role;
REVOKE USAGE ON SCHEMA functions FROM anon, authenticated, service_role;
REVOKE USAGE ON SCHEMA realtime FROM anon, authenticated, service_role;
REVOKE USAGE ON SCHEMA public FROM anon, authenticated, service_role;

-- Drop roles
DROP ROLE IF EXISTS service_role;
DROP ROLE IF EXISTS authenticated;
DROP ROLE IF EXISTS anon;
