-- Revert impersonation_sessions schema changes

-- Drop new indexes
DROP INDEX IF EXISTS auth.idx_impersonation_sessions_target_user_id;

-- Remove added columns
ALTER TABLE auth.impersonation_sessions
    DROP COLUMN IF EXISTS target_role,
    DROP COLUMN IF EXISTS ip_address,
    DROP COLUMN IF EXISTS user_agent;

-- Restore NOT NULL constraint (must be done before rename)
-- Note: This may fail if there are NULL values, which is expected for anon/service sessions
ALTER TABLE auth.impersonation_sessions
    ALTER COLUMN target_user_id SET NOT NULL;

-- Rename column back
ALTER TABLE auth.impersonation_sessions
    RENAME COLUMN target_user_id TO impersonated_user_id;

-- Restore original indexes
CREATE INDEX IF NOT EXISTS idx_auth_impersonation_impersonated_user_id
    ON auth.impersonation_sessions(impersonated_user_id);
