--
-- SETTINGS ENCRYPTION SUPPORT
-- Adds encrypted_value column for secret settings and user_id for user-specific settings
--

-- Add encrypted_value column for storing AES-256-GCM encrypted secrets
ALTER TABLE app.settings ADD COLUMN IF NOT EXISTS encrypted_value TEXT;

-- Add user_id column for user-specific settings
ALTER TABLE app.settings ADD COLUMN IF NOT EXISTS user_id UUID REFERENCES auth.users(id) ON DELETE CASCADE;

-- Drop the old unique constraint on key (we need to include user_id)
ALTER TABLE app.settings DROP CONSTRAINT IF EXISTS app_settings_key_key;
DROP INDEX IF EXISTS idx_app_settings_key;

-- Create new unique constraint: same key can exist for different users
-- NULL user_id = system setting, non-NULL user_id = user-specific setting
CREATE UNIQUE INDEX idx_app_settings_key_user
    ON app.settings(key, COALESCE(user_id, '00000000-0000-0000-0000-000000000000'::UUID));

-- Index for efficient user-specific settings queries
CREATE INDEX IF NOT EXISTS idx_app_settings_user_id
    ON app.settings(user_id) WHERE user_id IS NOT NULL;

-- Index for finding encrypted settings
CREATE INDEX IF NOT EXISTS idx_app_settings_encrypted
    ON app.settings(is_secret) WHERE is_secret = true AND encrypted_value IS NOT NULL;

-- Comments
COMMENT ON COLUMN app.settings.encrypted_value IS 'AES-256-GCM encrypted value (base64). Used when is_secret=true. The value column contains a placeholder.';
COMMENT ON COLUMN app.settings.user_id IS 'Owner user ID for user-specific settings. NULL means system-level setting.';

--
-- RLS POLICIES FOR USER-SPECIFIC SECRETS
--

-- Users can read their own secret settings
DROP POLICY IF EXISTS "Users can read their own secret settings" ON app.settings;
CREATE POLICY "Users can read their own secret settings"
    ON app.settings
    FOR SELECT
    TO authenticated
    USING (
        user_id = auth.current_user_id()
    );

-- Users can create their own secret settings
DROP POLICY IF EXISTS "Users can create their own settings" ON app.settings;
CREATE POLICY "Users can create their own settings"
    ON app.settings
    FOR INSERT
    TO authenticated
    WITH CHECK (
        user_id = auth.current_user_id()
    );

-- Users can update their own secret settings
DROP POLICY IF EXISTS "Users can update their own settings" ON app.settings;
CREATE POLICY "Users can update their own settings"
    ON app.settings
    FOR UPDATE
    TO authenticated
    USING (user_id = auth.current_user_id())
    WITH CHECK (user_id = auth.current_user_id());

-- Users can delete their own secret settings
DROP POLICY IF EXISTS "Users can delete their own settings" ON app.settings;
CREATE POLICY "Users can delete their own settings"
    ON app.settings
    FOR DELETE
    TO authenticated
    USING (user_id = auth.current_user_id());

COMMENT ON POLICY "Users can read their own secret settings" ON app.settings
    IS 'Users can read settings that belong to them (user_id matches)';
COMMENT ON POLICY "Users can create their own settings" ON app.settings
    IS 'Users can create settings tied to their user_id';
COMMENT ON POLICY "Users can update their own settings" ON app.settings
    IS 'Users can update settings that belong to them';
COMMENT ON POLICY "Users can delete their own settings" ON app.settings
    IS 'Users can delete settings that belong to them';
