-- Rollback migration 006: Auth Improvements

-- Drop service_keys table
DROP TABLE IF EXISTS auth.service_keys;

-- Remove allow_unauthenticated column from edge_functions
ALTER TABLE functions.edge_functions
DROP COLUMN IF EXISTS allow_unauthenticated;
