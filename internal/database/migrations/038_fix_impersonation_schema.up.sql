-- Fix impersonation_sessions schema to match Go code expectations
-- The Go code uses target_user_id but the schema has impersonated_user_id
-- Also adds missing columns: target_role, ip_address, user_agent

-- Rename column to match Go code
ALTER TABLE auth.impersonation_sessions
    RENAME COLUMN impersonated_user_id TO target_user_id;

-- Make nullable for anon/service impersonation (no target user)
ALTER TABLE auth.impersonation_sessions
    ALTER COLUMN target_user_id DROP NOT NULL;

-- Add missing columns that the Go code expects
ALTER TABLE auth.impersonation_sessions
    ADD COLUMN IF NOT EXISTS target_role TEXT,
    ADD COLUMN IF NOT EXISTS ip_address TEXT,
    ADD COLUMN IF NOT EXISTS user_agent TEXT;

-- Update indexes to use new column name
DROP INDEX IF EXISTS auth.idx_impersonation_sessions_impersonated_user_id;
DROP INDEX IF EXISTS auth.idx_auth_impersonation_impersonated_user_id;
CREATE INDEX IF NOT EXISTS idx_impersonation_sessions_target_user_id
    ON auth.impersonation_sessions(target_user_id);
