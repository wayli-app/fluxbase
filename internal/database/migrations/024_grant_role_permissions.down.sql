-- ============================================================================
-- REVOKE TABLE PERMISSIONS FROM RLS ROLES
-- ============================================================================
-- This migration revokes table-level permissions from anon, authenticated,
-- and service_role. Use when rolling back to request.jwt.claims-only approach.
-- ============================================================================

-- Revoke default privileges
ALTER DEFAULT PRIVILEGES IN SCHEMA auth REVOKE ALL ON TABLES FROM anon, authenticated, service_role;
ALTER DEFAULT PRIVILEGES IN SCHEMA app REVOKE ALL ON TABLES FROM anon, authenticated, service_role;
ALTER DEFAULT PRIVILEGES IN SCHEMA storage REVOKE ALL ON TABLES FROM anon, authenticated, service_role;
ALTER DEFAULT PRIVILEGES IN SCHEMA functions REVOKE ALL ON TABLES FROM anon, authenticated, service_role;
ALTER DEFAULT PRIVILEGES IN SCHEMA realtime REVOKE ALL ON TABLES FROM anon, authenticated, service_role;
ALTER DEFAULT PRIVILEGES IN SCHEMA dashboard REVOKE ALL ON TABLES FROM anon, authenticated, service_role;
ALTER DEFAULT PRIVILEGES FOR ROLE CURRENT_USER IN SCHEMA public REVOKE ALL ON TABLES FROM service_role;
ALTER DEFAULT PRIVILEGES FOR ROLE CURRENT_USER IN SCHEMA public REVOKE EXECUTE ON FUNCTIONS FROM service_role;

-- Revoke table permissions
REVOKE ALL ON ALL TABLES IN SCHEMA auth FROM anon, authenticated, service_role;
REVOKE ALL ON ALL SEQUENCES IN SCHEMA auth FROM anon, authenticated, service_role;

REVOKE ALL ON ALL TABLES IN SCHEMA app FROM anon, authenticated, service_role;
REVOKE ALL ON ALL SEQUENCES IN SCHEMA app FROM anon, authenticated, service_role;

REVOKE ALL ON ALL TABLES IN SCHEMA storage FROM anon, authenticated, service_role;
REVOKE ALL ON ALL SEQUENCES IN SCHEMA storage FROM anon, authenticated, service_role;

REVOKE ALL ON ALL TABLES IN SCHEMA functions FROM anon, authenticated, service_role;
REVOKE ALL ON ALL SEQUENCES IN SCHEMA functions FROM anon, authenticated, service_role;

REVOKE ALL ON ALL TABLES IN SCHEMA realtime FROM anon, authenticated, service_role;
REVOKE ALL ON ALL SEQUENCES IN SCHEMA realtime FROM anon, authenticated, service_role;

REVOKE ALL ON ALL TABLES IN SCHEMA dashboard FROM anon, authenticated, service_role;
REVOKE ALL ON ALL SEQUENCES IN SCHEMA dashboard FROM anon, authenticated, service_role;

REVOKE ALL ON ALL TABLES IN SCHEMA public FROM service_role;
REVOKE ALL ON ALL SEQUENCES IN SCHEMA public FROM service_role;
REVOKE EXECUTE ON ALL FUNCTIONS IN SCHEMA public FROM service_role;
