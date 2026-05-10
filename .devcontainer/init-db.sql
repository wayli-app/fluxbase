-- Initialize databases for development and testing

-- Create test database (legacy, used by make test-setup-db)
CREATE DATABASE fluxbase_test;

-- Create Go E2E test database (self-sufficient, auto-bootstrapped by tests)
CREATE DATABASE fluxbase_go_e2e;

-- Create Playwright test database (used by admin UI E2E tests)
CREATE DATABASE fluxbase_playwright;

-- Create fluxbase_app user for backward compatibility and testing
-- Note: Fluxbase now uses configurable user/admin user via environment variables:
-- - FLUXBASE_DATABASE_USER (runtime operations, defaults to 'postgres')
-- - FLUXBASE_DATABASE_ADMIN_USER (runs migrations, optional, defaults to user)
-- BYPASSRLS is needed for test setup and general testing (RLS tests use fluxbase_rls_test)
CREATE USER fluxbase_app WITH PASSWORD 'fluxbase_app_password' LOGIN BYPASSRLS CREATEDB;

-- Create fluxbase_rls_test user for testing RLS (without BYPASSRLS, but with CREATEDB for schema creation)
CREATE USER fluxbase_rls_test WITH PASSWORD 'fluxbase_rls_test_password' LOGIN NOBYPASSRLS CREATEDB;

-- Configure search_path for users (required for golang-migrate)
ALTER USER fluxbase_app SET search_path TO public;
ALTER USER fluxbase_rls_test SET search_path TO public;

-- Note: Fluxbase roles (anon, authenticated, service_role) are created by migration 003_roles.up.sql
-- Role grants to fluxbase_app and fluxbase_rls_test are handled by the Makefile's db-reset target
-- after migrations run, ensuring they're always granted when the database is reset.

-- Grant all privileges
GRANT ALL PRIVILEGES ON DATABASE fluxbase_dev TO postgres;
GRANT ALL PRIVILEGES ON DATABASE fluxbase_test TO postgres;
GRANT ALL PRIVILEGES ON DATABASE fluxbase_go_e2e TO postgres;
GRANT ALL PRIVILEGES ON DATABASE fluxbase_playwright TO postgres;
GRANT ALL PRIVILEGES ON DATABASE fluxbase_dev TO fluxbase_app;
GRANT ALL PRIVILEGES ON DATABASE fluxbase_test TO fluxbase_app;
GRANT ALL PRIVILEGES ON DATABASE fluxbase_go_e2e TO fluxbase_app;
GRANT ALL PRIVILEGES ON DATABASE fluxbase_playwright TO fluxbase_app;
GRANT ALL PRIVILEGES ON DATABASE fluxbase_dev TO fluxbase_rls_test;
GRANT ALL PRIVILEGES ON DATABASE fluxbase_test TO fluxbase_rls_test;
GRANT ALL PRIVILEGES ON DATABASE fluxbase_go_e2e TO fluxbase_rls_test;
GRANT ALL PRIVILEGES ON DATABASE fluxbase_playwright TO fluxbase_rls_test;

-- Grant CREATE on databases for schema creation
GRANT CREATE ON DATABASE fluxbase_test TO fluxbase_rls_test;
GRANT CREATE ON DATABASE fluxbase_go_e2e TO fluxbase_rls_test;
GRANT CREATE ON DATABASE fluxbase_playwright TO fluxbase_rls_test;

-- Connect to dev database and create extensions
\c fluxbase_dev;

-- Create useful extensions
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pg_trgm";
CREATE EXTENSION IF NOT EXISTS "btree_gin";
CREATE EXTENSION IF NOT EXISTS "postgis";
CREATE EXTENSION IF NOT EXISTS "postgis_topology";

-- Grant schema permissions to users (required for migrations)
GRANT USAGE, CREATE ON SCHEMA public TO fluxbase_app;
GRANT USAGE, CREATE ON SCHEMA public TO fluxbase_rls_test;
GRANT USAGE, CREATE ON SCHEMA auth TO fluxbase_app, fluxbase_rls_test;
GRANT USAGE, CREATE ON SCHEMA platform TO fluxbase_app, fluxbase_rls_test;
GRANT USAGE, CREATE ON SCHEMA functions TO fluxbase_app, fluxbase_rls_test;
GRANT USAGE, CREATE ON SCHEMA storage TO fluxbase_app, fluxbase_rls_test;
GRANT USAGE, CREATE ON SCHEMA realtime TO fluxbase_app, fluxbase_rls_test;
GRANT USAGE, CREATE ON SCHEMA _fluxbase TO fluxbase_app, fluxbase_rls_test;

-- Grant table and sequence permissions to both test users
GRANT ALL ON ALL TABLES IN SCHEMA auth TO fluxbase_app, fluxbase_rls_test;
GRANT ALL ON ALL SEQUENCES IN SCHEMA auth TO fluxbase_app, fluxbase_rls_test;
GRANT ALL ON ALL TABLES IN SCHEMA public TO fluxbase_app, fluxbase_rls_test;
GRANT ALL ON ALL SEQUENCES IN SCHEMA public TO fluxbase_app, fluxbase_rls_test;
GRANT ALL ON ALL TABLES IN SCHEMA platform TO fluxbase_app, fluxbase_rls_test;
GRANT ALL ON ALL SEQUENCES IN SCHEMA platform TO fluxbase_app, fluxbase_rls_test;
GRANT ALL ON ALL TABLES IN SCHEMA functions TO fluxbase_app, fluxbase_rls_test;
GRANT ALL ON ALL SEQUENCES IN SCHEMA functions TO fluxbase_app, fluxbase_rls_test;
GRANT ALL ON ALL TABLES IN SCHEMA storage TO fluxbase_app, fluxbase_rls_test;
GRANT ALL ON ALL SEQUENCES IN SCHEMA storage TO fluxbase_app, fluxbase_rls_test;
GRANT ALL ON ALL TABLES IN SCHEMA realtime TO fluxbase_app, fluxbase_rls_test;
GRANT ALL ON ALL SEQUENCES IN SCHEMA realtime TO fluxbase_app, fluxbase_rls_test;
GRANT ALL ON ALL TABLES IN SCHEMA _fluxbase TO fluxbase_app, fluxbase_rls_test;
GRANT ALL ON ALL SEQUENCES IN SCHEMA _fluxbase TO fluxbase_app, fluxbase_rls_test;
GRANT EXECUTE ON ALL FUNCTIONS IN SCHEMA functions TO fluxbase_app, fluxbase_rls_test;
GRANT EXECUTE ON ALL FUNCTIONS IN SCHEMA auth TO fluxbase_app, fluxbase_rls_test;
GRANT EXECUTE ON ALL FUNCTIONS IN SCHEMA storage TO fluxbase_app, fluxbase_rls_test;

-- Set default privileges for future objects (for all users who might create tables)
ALTER DEFAULT PRIVILEGES IN SCHEMA auth GRANT ALL ON TABLES TO fluxbase_app, fluxbase_rls_test;
ALTER DEFAULT PRIVILEGES IN SCHEMA auth GRANT ALL ON SEQUENCES TO fluxbase_app, fluxbase_rls_test;
ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT ALL ON TABLES TO fluxbase_app, fluxbase_rls_test;
ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT ALL ON SEQUENCES TO fluxbase_app, fluxbase_rls_test;
ALTER DEFAULT PRIVILEGES IN SCHEMA platform GRANT ALL ON TABLES TO fluxbase_app, fluxbase_rls_test;
ALTER DEFAULT PRIVILEGES IN SCHEMA platform GRANT ALL ON SEQUENCES TO fluxbase_app, fluxbase_rls_test;
ALTER DEFAULT PRIVILEGES IN SCHEMA functions GRANT ALL ON TABLES TO fluxbase_app, fluxbase_rls_test;
ALTER DEFAULT PRIVILEGES IN SCHEMA functions GRANT ALL ON SEQUENCES TO fluxbase_app, fluxbase_rls_test;
ALTER DEFAULT PRIVILEGES IN SCHEMA storage GRANT ALL ON TABLES TO fluxbase_app, fluxbase_rls_test;
ALTER DEFAULT PRIVILEGES IN SCHEMA storage GRANT ALL ON SEQUENCES TO fluxbase_app, fluxbase_rls_test;
ALTER DEFAULT PRIVILEGES IN SCHEMA realtime GRANT ALL ON TABLES TO fluxbase_app, fluxbase_rls_test;
ALTER DEFAULT PRIVILEGES IN SCHEMA realtime GRANT ALL ON SEQUENCES TO fluxbase_app, fluxbase_rls_test;
ALTER DEFAULT PRIVILEGES IN SCHEMA _fluxbase GRANT ALL ON TABLES TO fluxbase_app, fluxbase_rls_test;
ALTER DEFAULT PRIVILEGES IN SCHEMA _fluxbase GRANT ALL ON SEQUENCES TO fluxbase_app, fluxbase_rls_test;

-- Set default privileges for postgres role (test setup and migrations create tables as postgres)
ALTER DEFAULT PRIVILEGES FOR ROLE postgres IN SCHEMA auth GRANT ALL ON TABLES TO fluxbase_app, fluxbase_rls_test;
ALTER DEFAULT PRIVILEGES FOR ROLE postgres IN SCHEMA auth GRANT ALL ON SEQUENCES TO fluxbase_app, fluxbase_rls_test;
ALTER DEFAULT PRIVILEGES FOR ROLE postgres IN SCHEMA public GRANT ALL ON TABLES TO fluxbase_app, fluxbase_rls_test;
ALTER DEFAULT PRIVILEGES FOR ROLE postgres IN SCHEMA public GRANT ALL ON SEQUENCES TO fluxbase_app, fluxbase_rls_test;
ALTER DEFAULT PRIVILEGES FOR ROLE postgres IN SCHEMA platform GRANT ALL ON TABLES TO fluxbase_app, fluxbase_rls_test;
ALTER DEFAULT PRIVILEGES FOR ROLE postgres IN SCHEMA platform GRANT ALL ON SEQUENCES TO fluxbase_app, fluxbase_rls_test;
ALTER DEFAULT PRIVILEGES FOR ROLE postgres IN SCHEMA functions GRANT ALL ON TABLES TO fluxbase_app, fluxbase_rls_test;
ALTER DEFAULT PRIVILEGES FOR ROLE postgres IN SCHEMA functions GRANT ALL ON SEQUENCES TO fluxbase_app, fluxbase_rls_test;
ALTER DEFAULT PRIVILEGES FOR ROLE postgres IN SCHEMA storage GRANT ALL ON TABLES TO fluxbase_app, fluxbase_rls_test;
ALTER DEFAULT PRIVILEGES FOR ROLE postgres IN SCHEMA storage GRANT ALL ON SEQUENCES TO fluxbase_app, fluxbase_rls_test;
ALTER DEFAULT PRIVILEGES FOR ROLE postgres IN SCHEMA realtime GRANT ALL ON TABLES TO fluxbase_app, fluxbase_rls_test;
ALTER DEFAULT PRIVILEGES FOR ROLE postgres IN SCHEMA realtime GRANT ALL ON SEQUENCES TO fluxbase_app, fluxbase_rls_test;
ALTER DEFAULT PRIVILEGES FOR ROLE postgres IN SCHEMA _fluxbase GRANT ALL ON TABLES TO fluxbase_app, fluxbase_rls_test;
ALTER DEFAULT PRIVILEGES FOR ROLE postgres IN SCHEMA _fluxbase GRANT ALL ON SEQUENCES TO fluxbase_app, fluxbase_rls_test;

-- Set default privileges for fluxbase_app role (in case fluxbase_app creates tables)
ALTER DEFAULT PRIVILEGES FOR ROLE fluxbase_app IN SCHEMA auth GRANT ALL ON TABLES TO fluxbase_app, fluxbase_rls_test;
ALTER DEFAULT PRIVILEGES FOR ROLE fluxbase_app IN SCHEMA auth GRANT ALL ON SEQUENCES TO fluxbase_app, fluxbase_rls_test;
ALTER DEFAULT PRIVILEGES FOR ROLE fluxbase_app IN SCHEMA public GRANT ALL ON TABLES TO fluxbase_app, fluxbase_rls_test;
ALTER DEFAULT PRIVILEGES FOR ROLE fluxbase_app IN SCHEMA public GRANT ALL ON SEQUENCES TO fluxbase_app, fluxbase_rls_test;
ALTER DEFAULT PRIVILEGES FOR ROLE fluxbase_app IN SCHEMA platform GRANT ALL ON TABLES TO fluxbase_app, fluxbase_rls_test;
ALTER DEFAULT PRIVILEGES FOR ROLE fluxbase_app IN SCHEMA platform GRANT ALL ON SEQUENCES TO fluxbase_app, fluxbase_rls_test;
ALTER DEFAULT PRIVILEGES FOR ROLE fluxbase_app IN SCHEMA functions GRANT ALL ON TABLES TO fluxbase_app, fluxbase_rls_test;
ALTER DEFAULT PRIVILEGES FOR ROLE fluxbase_app IN SCHEMA functions GRANT ALL ON SEQUENCES TO fluxbase_app, fluxbase_rls_test;
ALTER DEFAULT PRIVILEGES FOR ROLE fluxbase_app IN SCHEMA storage GRANT ALL ON TABLES TO fluxbase_app, fluxbase_rls_test;
ALTER DEFAULT PRIVILEGES FOR ROLE fluxbase_app IN SCHEMA storage GRANT ALL ON SEQUENCES TO fluxbase_app, fluxbase_rls_test;
ALTER DEFAULT PRIVILEGES FOR ROLE fluxbase_app IN SCHEMA realtime GRANT ALL ON TABLES TO fluxbase_app, fluxbase_rls_test;
ALTER DEFAULT PRIVILEGES FOR ROLE fluxbase_app IN SCHEMA realtime GRANT ALL ON SEQUENCES TO fluxbase_app, fluxbase_rls_test;
ALTER DEFAULT PRIVILEGES FOR ROLE fluxbase_app IN SCHEMA _fluxbase GRANT ALL ON TABLES TO fluxbase_app, fluxbase_rls_test;
ALTER DEFAULT PRIVILEGES FOR ROLE fluxbase_app IN SCHEMA _fluxbase GRANT ALL ON SEQUENCES TO fluxbase_app, fluxbase_rls_test;

-- Same for test database
\c fluxbase_test;

CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pg_trgm";
CREATE EXTENSION IF NOT EXISTS "btree_gin";
CREATE EXTENSION IF NOT EXISTS "postgis";
CREATE EXTENSION IF NOT EXISTS "postgis_topology";

-- Grant schema permissions to users (required for migrations)
GRANT USAGE, CREATE ON SCHEMA public TO fluxbase_app;
GRANT USAGE, CREATE ON SCHEMA public TO fluxbase_rls_test;
GRANT USAGE, CREATE ON SCHEMA auth TO fluxbase_app, fluxbase_rls_test;
GRANT USAGE, CREATE ON SCHEMA platform TO fluxbase_app, fluxbase_rls_test;
GRANT USAGE, CREATE ON SCHEMA functions TO fluxbase_app, fluxbase_rls_test;
GRANT USAGE, CREATE ON SCHEMA storage TO fluxbase_app, fluxbase_rls_test;
GRANT USAGE, CREATE ON SCHEMA realtime TO fluxbase_app, fluxbase_rls_test;
GRANT USAGE, CREATE ON SCHEMA _fluxbase TO fluxbase_app, fluxbase_rls_test;

-- Grant table and sequence permissions to both test users
GRANT ALL ON ALL TABLES IN SCHEMA auth TO fluxbase_app, fluxbase_rls_test;
GRANT ALL ON ALL SEQUENCES IN SCHEMA auth TO fluxbase_app, fluxbase_rls_test;
GRANT ALL ON ALL TABLES IN SCHEMA public TO fluxbase_app, fluxbase_rls_test;
GRANT ALL ON ALL SEQUENCES IN SCHEMA public TO fluxbase_app, fluxbase_rls_test;
GRANT ALL ON ALL TABLES IN SCHEMA platform TO fluxbase_app, fluxbase_rls_test;
GRANT ALL ON ALL SEQUENCES IN SCHEMA platform TO fluxbase_app, fluxbase_rls_test;
GRANT ALL ON ALL TABLES IN SCHEMA functions TO fluxbase_app, fluxbase_rls_test;
GRANT ALL ON ALL SEQUENCES IN SCHEMA functions TO fluxbase_app, fluxbase_rls_test;
GRANT ALL ON ALL TABLES IN SCHEMA storage TO fluxbase_app, fluxbase_rls_test;
GRANT ALL ON ALL SEQUENCES IN SCHEMA storage TO fluxbase_app, fluxbase_rls_test;
GRANT ALL ON ALL TABLES IN SCHEMA realtime TO fluxbase_app, fluxbase_rls_test;
GRANT ALL ON ALL SEQUENCES IN SCHEMA realtime TO fluxbase_app, fluxbase_rls_test;
GRANT ALL ON ALL TABLES IN SCHEMA _fluxbase TO fluxbase_app, fluxbase_rls_test;
GRANT ALL ON ALL SEQUENCES IN SCHEMA _fluxbase TO fluxbase_app, fluxbase_rls_test;
GRANT EXECUTE ON ALL FUNCTIONS IN SCHEMA functions TO fluxbase_app, fluxbase_rls_test;
GRANT EXECUTE ON ALL FUNCTIONS IN SCHEMA auth TO fluxbase_app, fluxbase_rls_test;
GRANT EXECUTE ON ALL FUNCTIONS IN SCHEMA storage TO fluxbase_app, fluxbase_rls_test;

-- Set default privileges for future objects (for all users who might create tables)
ALTER DEFAULT PRIVILEGES IN SCHEMA auth GRANT ALL ON TABLES TO fluxbase_app, fluxbase_rls_test;
ALTER DEFAULT PRIVILEGES IN SCHEMA auth GRANT ALL ON SEQUENCES TO fluxbase_app, fluxbase_rls_test;
ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT ALL ON TABLES TO fluxbase_app, fluxbase_rls_test;
ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT ALL ON SEQUENCES TO fluxbase_app, fluxbase_rls_test;
ALTER DEFAULT PRIVILEGES IN SCHEMA platform GRANT ALL ON TABLES TO fluxbase_app, fluxbase_rls_test;
ALTER DEFAULT PRIVILEGES IN SCHEMA platform GRANT ALL ON SEQUENCES TO fluxbase_app, fluxbase_rls_test;
ALTER DEFAULT PRIVILEGES IN SCHEMA functions GRANT ALL ON TABLES TO fluxbase_app, fluxbase_rls_test;
ALTER DEFAULT PRIVILEGES IN SCHEMA functions GRANT ALL ON SEQUENCES TO fluxbase_app, fluxbase_rls_test;
ALTER DEFAULT PRIVILEGES IN SCHEMA storage GRANT ALL ON TABLES TO fluxbase_app, fluxbase_rls_test;
ALTER DEFAULT PRIVILEGES IN SCHEMA storage GRANT ALL ON SEQUENCES TO fluxbase_app, fluxbase_rls_test;
ALTER DEFAULT PRIVILEGES IN SCHEMA realtime GRANT ALL ON TABLES TO fluxbase_app, fluxbase_rls_test;
ALTER DEFAULT PRIVILEGES IN SCHEMA realtime GRANT ALL ON SEQUENCES TO fluxbase_app, fluxbase_rls_test;
ALTER DEFAULT PRIVILEGES IN SCHEMA _fluxbase GRANT ALL ON TABLES TO fluxbase_app, fluxbase_rls_test;
ALTER DEFAULT PRIVILEGES IN SCHEMA _fluxbase GRANT ALL ON SEQUENCES TO fluxbase_app, fluxbase_rls_test;

-- Set default privileges for postgres role (test setup and migrations create tables as postgres)
ALTER DEFAULT PRIVILEGES FOR ROLE postgres IN SCHEMA auth GRANT ALL ON TABLES TO fluxbase_app, fluxbase_rls_test;
ALTER DEFAULT PRIVILEGES FOR ROLE postgres IN SCHEMA auth GRANT ALL ON SEQUENCES TO fluxbase_app, fluxbase_rls_test;
ALTER DEFAULT PRIVILEGES FOR ROLE postgres IN SCHEMA public GRANT ALL ON TABLES TO fluxbase_app, fluxbase_rls_test;
ALTER DEFAULT PRIVILEGES FOR ROLE postgres IN SCHEMA public GRANT ALL ON SEQUENCES TO fluxbase_app, fluxbase_rls_test;
ALTER DEFAULT PRIVILEGES FOR ROLE postgres IN SCHEMA platform GRANT ALL ON TABLES TO fluxbase_app, fluxbase_rls_test;
ALTER DEFAULT PRIVILEGES FOR ROLE postgres IN SCHEMA platform GRANT ALL ON SEQUENCES TO fluxbase_app, fluxbase_rls_test;
ALTER DEFAULT PRIVILEGES FOR ROLE postgres IN SCHEMA functions GRANT ALL ON TABLES TO fluxbase_app, fluxbase_rls_test;
ALTER DEFAULT PRIVILEGES FOR ROLE postgres IN SCHEMA functions GRANT ALL ON SEQUENCES TO fluxbase_app, fluxbase_rls_test;
ALTER DEFAULT PRIVILEGES FOR ROLE postgres IN SCHEMA storage GRANT ALL ON TABLES TO fluxbase_app, fluxbase_rls_test;
ALTER DEFAULT PRIVILEGES FOR ROLE postgres IN SCHEMA storage GRANT ALL ON SEQUENCES TO fluxbase_app, fluxbase_rls_test;
ALTER DEFAULT PRIVILEGES FOR ROLE postgres IN SCHEMA realtime GRANT ALL ON TABLES TO fluxbase_app, fluxbase_rls_test;
ALTER DEFAULT PRIVILEGES FOR ROLE postgres IN SCHEMA realtime GRANT ALL ON SEQUENCES TO fluxbase_app, fluxbase_rls_test;
ALTER DEFAULT PRIVILEGES FOR ROLE postgres IN SCHEMA _fluxbase GRANT ALL ON TABLES TO fluxbase_app, fluxbase_rls_test;
ALTER DEFAULT PRIVILEGES FOR ROLE postgres IN SCHEMA _fluxbase GRANT ALL ON SEQUENCES TO fluxbase_app, fluxbase_rls_test;

-- Set default privileges for fluxbase_app role (in case fluxbase_app creates tables)
ALTER DEFAULT PRIVILEGES FOR ROLE fluxbase_app IN SCHEMA auth GRANT ALL ON TABLES TO fluxbase_app, fluxbase_rls_test;
ALTER DEFAULT PRIVILEGES FOR ROLE fluxbase_app IN SCHEMA auth GRANT ALL ON SEQUENCES TO fluxbase_app, fluxbase_rls_test;
ALTER DEFAULT PRIVILEGES FOR ROLE fluxbase_app IN SCHEMA public GRANT ALL ON TABLES TO fluxbase_app, fluxbase_rls_test;
ALTER DEFAULT PRIVILEGES FOR ROLE fluxbase_app IN SCHEMA public GRANT ALL ON SEQUENCES TO fluxbase_app, fluxbase_rls_test;
ALTER DEFAULT PRIVILEGES FOR ROLE fluxbase_app IN SCHEMA platform GRANT ALL ON TABLES TO fluxbase_app, fluxbase_rls_test;
ALTER DEFAULT PRIVILEGES FOR ROLE fluxbase_app IN SCHEMA platform GRANT ALL ON SEQUENCES TO fluxbase_app, fluxbase_rls_test;
ALTER DEFAULT PRIVILEGES FOR ROLE fluxbase_app IN SCHEMA functions GRANT ALL ON TABLES TO fluxbase_app, fluxbase_rls_test;
ALTER DEFAULT PRIVILEGES FOR ROLE fluxbase_app IN SCHEMA functions GRANT ALL ON SEQUENCES TO fluxbase_app, fluxbase_rls_test;
ALTER DEFAULT PRIVILEGES FOR ROLE fluxbase_app IN SCHEMA storage GRANT ALL ON TABLES TO fluxbase_app, fluxbase_rls_test;
ALTER DEFAULT PRIVILEGES FOR ROLE fluxbase_app IN SCHEMA storage GRANT ALL ON SEQUENCES TO fluxbase_app, fluxbase_rls_test;
ALTER DEFAULT PRIVILEGES FOR ROLE fluxbase_app IN SCHEMA realtime GRANT ALL ON TABLES TO fluxbase_app, fluxbase_rls_test;
ALTER DEFAULT PRIVILEGES FOR ROLE fluxbase_app IN SCHEMA realtime GRANT ALL ON SEQUENCES TO fluxbase_app, fluxbase_rls_test;
ALTER DEFAULT PRIVILEGES FOR ROLE fluxbase_app IN SCHEMA _fluxbase GRANT ALL ON TABLES TO fluxbase_app, fluxbase_rls_test;
ALTER DEFAULT PRIVILEGES FOR ROLE fluxbase_app IN SCHEMA _fluxbase GRANT ALL ON SEQUENCES TO fluxbase_app, fluxbase_rls_test;

-- Back to postgres database
\c postgres;

-- Same for Playwright test database
\c fluxbase_playwright;

CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pg_trgm";
CREATE EXTENSION IF NOT EXISTS "btree_gin";
CREATE EXTENSION IF NOT EXISTS "postgis";
CREATE EXTENSION IF NOT EXISTS "postgis_topology";

-- Grant schema permissions to users
GRANT USAGE, CREATE ON SCHEMA public TO fluxbase_app;
GRANT USAGE, CREATE ON SCHEMA public TO fluxbase_rls_test;
GRANT USAGE, CREATE ON SCHEMA auth TO fluxbase_app, fluxbase_rls_test;
GRANT USAGE, CREATE ON SCHEMA platform TO fluxbase_app, fluxbase_rls_test;
GRANT USAGE, CREATE ON SCHEMA functions TO fluxbase_app, fluxbase_rls_test;
GRANT USAGE, CREATE ON SCHEMA storage TO fluxbase_app, fluxbase_rls_test;
GRANT USAGE, CREATE ON SCHEMA realtime TO fluxbase_app, fluxbase_rls_test;
GRANT USAGE, CREATE ON SCHEMA _fluxbase TO fluxbase_app, fluxbase_rls_test;

-- Grant table and sequence permissions to both test users
GRANT ALL ON ALL TABLES IN SCHEMA auth TO fluxbase_app, fluxbase_rls_test;
GRANT ALL ON ALL SEQUENCES IN SCHEMA auth TO fluxbase_app, fluxbase_rls_test;
GRANT ALL ON ALL TABLES IN SCHEMA public TO fluxbase_app, fluxbase_rls_test;
GRANT ALL ON ALL SEQUENCES IN SCHEMA public TO fluxbase_app, fluxbase_rls_test;
GRANT ALL ON ALL TABLES IN SCHEMA platform TO fluxbase_app, fluxbase_rls_test;
GRANT ALL ON ALL SEQUENCES IN SCHEMA platform TO fluxbase_app, fluxbase_rls_test;
GRANT ALL ON ALL TABLES IN SCHEMA functions TO fluxbase_app, fluxbase_rls_test;
GRANT ALL ON ALL SEQUENCES IN SCHEMA functions TO fluxbase_app, fluxbase_rls_test;
GRANT ALL ON ALL TABLES IN SCHEMA storage TO fluxbase_app, fluxbase_rls_test;
GRANT ALL ON ALL SEQUENCES IN SCHEMA storage TO fluxbase_app, fluxbase_rls_test;
GRANT ALL ON ALL TABLES IN SCHEMA realtime TO fluxbase_app, fluxbase_rls_test;
GRANT ALL ON ALL SEQUENCES IN SCHEMA realtime TO fluxbase_app, fluxbase_rls_test;
GRANT ALL ON ALL TABLES IN SCHEMA _fluxbase TO fluxbase_app, fluxbase_rls_test;
GRANT ALL ON ALL SEQUENCES IN SCHEMA _fluxbase TO fluxbase_app, fluxbase_rls_test;
GRANT EXECUTE ON ALL FUNCTIONS IN SCHEMA functions TO fluxbase_app, fluxbase_rls_test;
GRANT EXECUTE ON ALL FUNCTIONS IN SCHEMA auth TO fluxbase_app, fluxbase_rls_test;
GRANT EXECUTE ON ALL FUNCTIONS IN SCHEMA storage TO fluxbase_app, fluxbase_rls_test;

-- Set default privileges for future objects
ALTER DEFAULT PRIVILEGES IN SCHEMA auth GRANT ALL ON TABLES TO fluxbase_app, fluxbase_rls_test;
ALTER DEFAULT PRIVILEGES IN SCHEMA auth GRANT ALL ON SEQUENCES TO fluxbase_app, fluxbase_rls_test;
ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT ALL ON TABLES TO fluxbase_app, fluxbase_rls_test;
ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT ALL ON SEQUENCES TO fluxbase_app, fluxbase_rls_test;
ALTER DEFAULT PRIVILEGES IN SCHEMA platform GRANT ALL ON TABLES TO fluxbase_app, fluxbase_rls_test;
ALTER DEFAULT PRIVILEGES IN SCHEMA platform GRANT ALL ON SEQUENCES TO fluxbase_app, fluxbase_rls_test;
ALTER DEFAULT PRIVILEGES IN SCHEMA functions GRANT ALL ON TABLES TO fluxbase_app, fluxbase_rls_test;
ALTER DEFAULT PRIVILEGES IN SCHEMA functions GRANT ALL ON SEQUENCES TO fluxbase_app, fluxbase_rls_test;
ALTER DEFAULT PRIVILEGES IN SCHEMA storage GRANT ALL ON TABLES TO fluxbase_app, fluxbase_rls_test;
ALTER DEFAULT PRIVILEGES IN SCHEMA storage GRANT ALL ON SEQUENCES TO fluxbase_app, fluxbase_rls_test;
ALTER DEFAULT PRIVILEGES IN SCHEMA realtime GRANT ALL ON TABLES TO fluxbase_app, fluxbase_rls_test;
ALTER DEFAULT PRIVILEGES IN SCHEMA realtime GRANT ALL ON SEQUENCES TO fluxbase_app, fluxbase_rls_test;
ALTER DEFAULT PRIVILEGES IN SCHEMA _fluxbase GRANT ALL ON TABLES TO fluxbase_app, fluxbase_rls_test;
ALTER DEFAULT PRIVILEGES IN SCHEMA _fluxbase GRANT ALL ON SEQUENCES TO fluxbase_app, fluxbase_rls_test;

-- Set default privileges for postgres role
ALTER DEFAULT PRIVILEGES FOR ROLE postgres IN SCHEMA auth GRANT ALL ON TABLES TO fluxbase_app, fluxbase_rls_test;
ALTER DEFAULT PRIVILEGES FOR ROLE postgres IN SCHEMA auth GRANT ALL ON SEQUENCES TO fluxbase_app, fluxbase_rls_test;
ALTER DEFAULT PRIVILEGES FOR ROLE postgres IN SCHEMA public GRANT ALL ON TABLES TO fluxbase_app, fluxbase_rls_test;
ALTER DEFAULT PRIVILEGES FOR ROLE postgres IN SCHEMA public GRANT ALL ON SEQUENCES TO fluxbase_app, fluxbase_rls_test;
ALTER DEFAULT PRIVILEGES FOR ROLE postgres IN SCHEMA platform GRANT ALL ON TABLES TO fluxbase_app, fluxbase_rls_test;
ALTER DEFAULT PRIVILEGES FOR ROLE postgres IN SCHEMA platform GRANT ALL ON SEQUENCES TO fluxbase_app, fluxbase_rls_test;
ALTER DEFAULT PRIVILEGES FOR ROLE postgres IN SCHEMA functions GRANT ALL ON TABLES TO fluxbase_app, fluxbase_rls_test;
ALTER DEFAULT PRIVILEGES FOR ROLE postgres IN SCHEMA functions GRANT ALL ON SEQUENCES TO fluxbase_app, fluxbase_rls_test;
ALTER DEFAULT PRIVILEGES FOR ROLE postgres IN SCHEMA storage GRANT ALL ON TABLES TO fluxbase_app, fluxbase_rls_test;
ALTER DEFAULT PRIVILEGES FOR ROLE postgres IN SCHEMA storage GRANT ALL ON SEQUENCES TO fluxbase_app, fluxbase_rls_test;
ALTER DEFAULT PRIVILEGES FOR ROLE postgres IN SCHEMA realtime GRANT ALL ON TABLES TO fluxbase_app, fluxbase_rls_test;
ALTER DEFAULT PRIVILEGES FOR ROLE postgres IN SCHEMA realtime GRANT ALL ON SEQUENCES TO fluxbase_app, fluxbase_rls_test;
ALTER DEFAULT PRIVILEGES FOR ROLE postgres IN SCHEMA _fluxbase GRANT ALL ON TABLES TO fluxbase_app, fluxbase_rls_test;
ALTER DEFAULT PRIVILEGES FOR ROLE postgres IN SCHEMA _fluxbase GRANT ALL ON SEQUENCES TO fluxbase_app, fluxbase_rls_test;

-- Set default privileges for fluxbase_app role
ALTER DEFAULT PRIVILEGES FOR ROLE fluxbase_app IN SCHEMA auth GRANT ALL ON TABLES TO fluxbase_app, fluxbase_rls_test;
ALTER DEFAULT PRIVILEGES FOR ROLE fluxbase_app IN SCHEMA auth GRANT ALL ON SEQUENCES TO fluxbase_app, fluxbase_rls_test;
ALTER DEFAULT PRIVILEGES FOR ROLE fluxbase_app IN SCHEMA public GRANT ALL ON TABLES TO fluxbase_app, fluxbase_rls_test;
ALTER DEFAULT PRIVILEGES FOR ROLE fluxbase_app IN SCHEMA public GRANT ALL ON SEQUENCES TO fluxbase_app, fluxbase_rls_test;
ALTER DEFAULT PRIVILEGES FOR ROLE fluxbase_app IN SCHEMA platform GRANT ALL ON TABLES TO fluxbase_app, fluxbase_rls_test;
ALTER DEFAULT PRIVILEGES FOR ROLE fluxbase_app IN SCHEMA platform GRANT ALL ON SEQUENCES TO fluxbase_app, fluxbase_rls_test;
ALTER DEFAULT PRIVILEGES FOR ROLE fluxbase_app IN SCHEMA functions GRANT ALL ON TABLES TO fluxbase_app, fluxbase_rls_test;
ALTER DEFAULT PRIVILEGES FOR ROLE fluxbase_app IN SCHEMA functions GRANT ALL ON SEQUENCES TO fluxbase_app, fluxbase_rls_test;
ALTER DEFAULT PRIVILEGES FOR ROLE fluxbase_app IN SCHEMA storage GRANT ALL ON TABLES TO fluxbase_app, fluxbase_rls_test;
ALTER DEFAULT PRIVILEGES FOR ROLE fluxbase_app IN SCHEMA storage GRANT ALL ON SEQUENCES TO fluxbase_app, fluxbase_rls_test;
ALTER DEFAULT PRIVILEGES FOR ROLE fluxbase_app IN SCHEMA realtime GRANT ALL ON TABLES TO fluxbase_app, fluxbase_rls_test;
ALTER DEFAULT PRIVILEGES FOR ROLE fluxbase_app IN SCHEMA realtime GRANT ALL ON SEQUENCES TO fluxbase_app, fluxbase_rls_test;
ALTER DEFAULT PRIVILEGES FOR ROLE fluxbase_app IN SCHEMA _fluxbase GRANT ALL ON TABLES TO fluxbase_app, fluxbase_rls_test;
ALTER DEFAULT PRIVILEGES FOR ROLE fluxbase_app IN SCHEMA _fluxbase GRANT ALL ON SEQUENCES TO fluxbase_app, fluxbase_rls_test;

-- Back to postgres database
\c postgres;
