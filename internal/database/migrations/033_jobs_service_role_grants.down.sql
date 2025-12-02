-- Revoke schema usage from service_role
REVOKE USAGE ON SCHEMA jobs FROM service_role;

-- Revoke all privileges on jobs tables from service_role
REVOKE ALL ON ALL TABLES IN SCHEMA jobs FROM service_role;

-- Revoke sequence privileges from service_role
REVOKE ALL ON ALL SEQUENCES IN SCHEMA jobs FROM service_role;

-- Remove default privileges
ALTER DEFAULT PRIVILEGES IN SCHEMA jobs
    REVOKE ALL ON TABLES FROM service_role;

ALTER DEFAULT PRIVILEGES IN SCHEMA jobs
    REVOKE ALL ON SEQUENCES FROM service_role;
