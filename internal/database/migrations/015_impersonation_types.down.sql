-- Rollback: Impersonation Types

-- Remove the constraint
ALTER TABLE auth.impersonation_sessions
    DROP CONSTRAINT IF EXISTS check_target_user_for_user_type;

-- Make target_user_id NOT NULL again
ALTER TABLE auth.impersonation_sessions
    ALTER COLUMN target_user_id SET NOT NULL;

-- Re-add self-impersonation constraint
ALTER TABLE auth.impersonation_sessions
    ADD CONSTRAINT check_no_self_impersonation
    CHECK (admin_user_id != target_user_id);

-- Remove new columns
ALTER TABLE auth.impersonation_sessions
    DROP COLUMN IF EXISTS target_role,
    DROP COLUMN IF EXISTS impersonation_type;

-- Drop the enum type
DROP TYPE IF EXISTS auth.impersonation_type;
