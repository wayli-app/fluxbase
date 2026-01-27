--
-- FIX: Add partial unique index for system settings
-- The composite index idx_app_settings_key_user covers (key, user_id) but
-- ON CONFLICT (key) requires a unique index on key alone.
-- This partial index enables upserts for system settings (user_id IS NULL).
--

CREATE UNIQUE INDEX IF NOT EXISTS idx_app_settings_system_key
    ON app.settings(key) WHERE user_id IS NULL;
