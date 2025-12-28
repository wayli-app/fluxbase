--
-- ROLLBACK: SETTINGS ENCRYPTION SUPPORT
--

-- Drop user-specific RLS policies
DROP POLICY IF EXISTS "Users can read their own secret settings" ON app.settings;
DROP POLICY IF EXISTS "Users can create their own settings" ON app.settings;
DROP POLICY IF EXISTS "Users can update their own settings" ON app.settings;
DROP POLICY IF EXISTS "Users can delete their own settings" ON app.settings;

-- Drop indexes
DROP INDEX IF EXISTS idx_app_settings_key_user;
DROP INDEX IF EXISTS idx_app_settings_user_id;
DROP INDEX IF EXISTS idx_app_settings_encrypted;

-- Restore original unique constraint on key
CREATE UNIQUE INDEX IF NOT EXISTS idx_app_settings_key ON app.settings(key);

-- Remove columns (this will delete any encrypted data!)
ALTER TABLE app.settings DROP COLUMN IF EXISTS encrypted_value;
ALTER TABLE app.settings DROP COLUMN IF EXISTS user_id;
