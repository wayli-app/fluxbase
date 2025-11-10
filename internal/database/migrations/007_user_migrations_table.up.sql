-- Create user_migrations table for tracking user-provided migrations
-- This table is managed by golang-migrate and should not be modified manually

CREATE TABLE IF NOT EXISTS _fluxbase.user_migrations (
    version BIGINT NOT NULL PRIMARY KEY,
    dirty BOOLEAN NOT NULL
);

COMMENT ON TABLE _fluxbase.user_migrations IS 'Tracks user-provided database migration versions (managed by golang-migrate)';
