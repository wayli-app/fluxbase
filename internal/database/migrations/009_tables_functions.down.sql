-- Drop functions tables in reverse dependency order

-- Remove enhancements from edge_functions
ALTER TABLE functions.edge_functions
    DROP COLUMN IF EXISTS cors_origins,
    DROP COLUMN IF EXISTS cors_methods,
    DROP COLUMN IF EXISTS cors_headers,
    DROP COLUMN IF EXISTS cors_credentials,
    DROP COLUMN IF EXISTS cors_max_age,
    DROP COLUMN IF EXISTS needs_rebundle,
    DROP COLUMN IF EXISTS is_public;

-- Drop dependency tracking and configuration
DROP TABLE IF EXISTS functions.function_dependencies;
DROP TABLE IF EXISTS functions.shared_modules;
DROP TABLE IF EXISTS functions.edge_files;
DROP TABLE IF EXISTS functions.edge_executions;
DROP TABLE IF EXISTS functions.edge_triggers;
DROP TABLE IF EXISTS functions.edge_functions;
