--
-- Rollback: Revoke permissions for migrations schema
--

-- Revoke default privileges
ALTER DEFAULT PRIVILEGES IN SCHEMA migrations
    REVOKE ALL ON SEQUENCES FROM service_role;

ALTER DEFAULT PRIVILEGES IN SCHEMA migrations
    REVOKE ALL ON TABLES FROM service_role;

-- Revoke direct grants
REVOKE ALL ON ALL SEQUENCES IN SCHEMA migrations FROM service_role;
REVOKE ALL ON ALL TABLES IN SCHEMA migrations FROM service_role;
REVOKE USAGE ON SCHEMA migrations FROM service_role;
