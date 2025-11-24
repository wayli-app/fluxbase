--
-- Rollback: Revoke permissions for migrations schema
--

REVOKE ALL ON ALL SEQUENCES IN SCHEMA migrations FROM authenticated;
REVOKE ALL ON ALL TABLES IN SCHEMA migrations FROM authenticated;
REVOKE USAGE ON SCHEMA migrations FROM authenticated;

REVOKE ALL ON ALL SEQUENCES IN SCHEMA migrations FROM service_role;
REVOKE ALL ON ALL TABLES IN SCHEMA migrations FROM service_role;
REVOKE USAGE ON SCHEMA migrations FROM service_role;
