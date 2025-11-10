-- Migration 008: Split user metadata into user_metadata and app_metadata
-- This aligns with Supabase's approach to JWT claims and metadata management
--
-- user_metadata: Can be updated by the user themselves
-- app_metadata: Can only be updated by admins/service role (application-controlled)

-- Add user_metadata and app_metadata columns to auth.users
ALTER TABLE auth.users
ADD COLUMN IF NOT EXISTS user_metadata JSONB DEFAULT '{}'::JSONB,
ADD COLUMN IF NOT EXISTS app_metadata JSONB DEFAULT '{}'::JSONB;

-- Migrate existing metadata to user_metadata
UPDATE auth.users
SET user_metadata = COALESCE(metadata, '{}'::JSONB)
WHERE user_metadata = '{}'::JSONB;

-- Drop the old metadata column
ALTER TABLE auth.users DROP COLUMN IF EXISTS metadata;

-- Add comments for documentation
COMMENT ON COLUMN auth.users.user_metadata IS
'User-editable metadata. Users can update this field themselves. Included in JWT claims.';

COMMENT ON COLUMN auth.users.app_metadata IS
'Application/admin-only metadata. Can only be updated by admins or service role. Included in JWT claims.';

-- Create indexes for JSON queries
CREATE INDEX IF NOT EXISTS idx_auth_users_user_metadata ON auth.users USING GIN (user_metadata);
CREATE INDEX IF NOT EXISTS idx_auth_users_app_metadata ON auth.users USING GIN (app_metadata);

-- Add the same fields to dashboard.users for consistency
ALTER TABLE dashboard.users
ADD COLUMN IF NOT EXISTS user_metadata JSONB DEFAULT '{}'::JSONB,
ADD COLUMN IF NOT EXISTS app_metadata JSONB DEFAULT '{}'::JSONB;

-- Migrate existing dashboard metadata to user_metadata
UPDATE dashboard.users
SET user_metadata = COALESCE(metadata, '{}'::JSONB)
WHERE user_metadata = '{}'::JSONB;

-- Drop the old metadata column from dashboard.users
ALTER TABLE dashboard.users DROP COLUMN IF EXISTS metadata;

COMMENT ON COLUMN dashboard.users.user_metadata IS
'User-editable metadata for dashboard users.';

COMMENT ON COLUMN dashboard.users.app_metadata IS
'Application/admin-only metadata for dashboard users.';

-- Create indexes for dashboard users as well
CREATE INDEX IF NOT EXISTS idx_dashboard_users_user_metadata ON dashboard.users USING GIN (user_metadata);
CREATE INDEX IF NOT EXISTS idx_dashboard_users_app_metadata ON dashboard.users USING GIN (app_metadata);
