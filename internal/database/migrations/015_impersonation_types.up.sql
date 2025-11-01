-- Migration: Impersonation Types
-- Add support for different types of impersonation (user, anon, service)

-- Create enum type for impersonation types
CREATE TYPE auth.impersonation_type AS ENUM ('user', 'anon', 'service');

-- Add new columns to impersonation_sessions table
ALTER TABLE auth.impersonation_sessions
    ADD COLUMN IF NOT EXISTS impersonation_type auth.impersonation_type DEFAULT 'user',
    ADD COLUMN IF NOT EXISTS target_role TEXT;

-- Make target_user_id nullable (for anon/service impersonation)
ALTER TABLE auth.impersonation_sessions
    ALTER COLUMN target_user_id DROP NOT NULL;

-- Drop the self-impersonation constraint as it doesn't apply to anon/service
ALTER TABLE auth.impersonation_sessions
    DROP CONSTRAINT IF EXISTS check_no_self_impersonation;

-- Add new constraint: target_user_id required only for 'user' type
ALTER TABLE auth.impersonation_sessions
    ADD CONSTRAINT check_target_user_for_user_type
    CHECK (
        (impersonation_type = 'user' AND target_user_id IS NOT NULL) OR
        (impersonation_type IN ('anon', 'service') AND target_user_id IS NULL)
    );

-- Update existing records to have proper impersonation_type
UPDATE auth.impersonation_sessions
SET impersonation_type = 'user'
WHERE impersonation_type IS NULL;

-- Add comments
COMMENT ON COLUMN auth.impersonation_sessions.impersonation_type IS
'Type of impersonation: user (specific user), anon (anonymous/anon key), service (service role)';

COMMENT ON COLUMN auth.impersonation_sessions.target_role IS
'The role being impersonated (e.g., "admin", "user", "anon", "service")';
