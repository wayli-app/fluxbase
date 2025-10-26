-- Initialize databases for development and testing

-- Create test database
CREATE DATABASE fluxbase_test;

-- Grant all privileges
GRANT ALL PRIVILEGES ON DATABASE fluxbase_dev TO postgres;
GRANT ALL PRIVILEGES ON DATABASE fluxbase_test TO postgres;

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
