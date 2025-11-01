-- Initialize databases for development and testing

-- Create test database
CREATE DATABASE fluxbase_test;

-- Create fluxbase_app user for non-superuser operations (required for RLS)
CREATE USER fluxbase_app WITH PASSWORD 'fluxbase_app_password' LOGIN;

-- Grant all privileges
GRANT ALL PRIVILEGES ON DATABASE fluxbase_dev TO postgres;
GRANT ALL PRIVILEGES ON DATABASE fluxbase_test TO postgres;
GRANT ALL PRIVILEGES ON DATABASE fluxbase_dev TO fluxbase_app;
GRANT ALL PRIVILEGES ON DATABASE fluxbase_test TO fluxbase_app;

-- Connect to dev database and create extensions
\c fluxbase_dev;

-- Create useful extensions
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pg_trgm";
CREATE EXTENSION IF NOT EXISTS "btree_gin";

-- Same for test database
\c fluxbase_test;

CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pg_trgm";
CREATE EXTENSION IF NOT EXISTS "btree_gin";

-- Back to postgres database
\c postgres;
