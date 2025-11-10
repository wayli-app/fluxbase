-- Migration 008 Rollback: Remove user_metadata and app_metadata columns, restore metadata

-- Remove indexes
DROP INDEX IF EXISTS auth.idx_auth_users_user_metadata;
DROP INDEX IF EXISTS auth.idx_auth_users_app_metadata;
DROP INDEX IF EXISTS dashboard.idx_dashboard_users_user_metadata;
DROP INDEX IF EXISTS dashboard.idx_dashboard_users_app_metadata;

-- Restore metadata column for auth.users
ALTER TABLE auth.users ADD COLUMN IF NOT EXISTS metadata JSONB DEFAULT '{}'::JSONB;

-- Migrate user_metadata back to metadata
UPDATE auth.users
SET metadata = COALESCE(user_metadata, '{}'::JSONB)
WHERE metadata = '{}'::JSONB;

-- Remove new columns from auth.users
ALTER TABLE auth.users
DROP COLUMN IF EXISTS user_metadata,
DROP COLUMN IF EXISTS app_metadata;

-- Restore metadata column for dashboard.users
ALTER TABLE dashboard.users ADD COLUMN IF NOT EXISTS metadata JSONB DEFAULT '{}'::JSONB;

-- Migrate user_metadata back to metadata
UPDATE dashboard.users
SET metadata = COALESCE(user_metadata, '{}'::JSONB)
WHERE metadata = '{}'::JSONB;

-- Remove new columns from dashboard.users
ALTER TABLE dashboard.users
DROP COLUMN IF EXISTS user_metadata,
DROP COLUMN IF EXISTS app_metadata;
