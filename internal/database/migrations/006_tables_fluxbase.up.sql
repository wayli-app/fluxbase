--
-- _FLUXBASE SCHEMA TABLES
-- Internal Fluxbase system tables for migration tracking
--

-- User migrations tracking table (for user-provided migrations)
CREATE TABLE IF NOT EXISTS _fluxbase.user_migrations (
    version BIGINT NOT NULL PRIMARY KEY,
    dirty BOOLEAN NOT NULL
);

COMMENT ON TABLE _fluxbase.user_migrations IS 'Tracks user-provided database migration versions (managed by golang-migrate)';
