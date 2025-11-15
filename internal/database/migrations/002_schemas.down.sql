-- Rollback Schemas
-- Drop all Fluxbase schemas and their contents
-- This will cascade and remove all tables, functions, and other objects

DROP SCHEMA IF EXISTS realtime CASCADE;
DROP SCHEMA IF EXISTS storage CASCADE;
DROP SCHEMA IF EXISTS functions CASCADE;
DROP SCHEMA IF EXISTS dashboard CASCADE;
DROP SCHEMA IF EXISTS app CASCADE;
DROP SCHEMA IF EXISTS auth CASCADE;
DROP SCHEMA IF EXISTS _fluxbase CASCADE;
