-- Rollback Central Logging System Schema

-- Drop all partitions (they'll be dropped with the parent table, but be explicit)
DROP TABLE IF EXISTS logging.entries_system;
DROP TABLE IF EXISTS logging.entries_http;
DROP TABLE IF EXISTS logging.entries_security;
DROP TABLE IF EXISTS logging.entries_execution;
DROP TABLE IF EXISTS logging.entries_ai;

-- Drop the main table (this also drops all indexes)
DROP TABLE IF EXISTS logging.entries;

-- Drop the schema
DROP SCHEMA IF EXISTS logging CASCADE;
