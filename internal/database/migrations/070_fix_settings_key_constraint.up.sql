--
-- FIX: Settings key constraint name
-- Migration 042 used wrong constraint name (app_settings_key_key instead of settings_key_key)
-- This prevented user-specific secrets from sharing keys with system secrets
--

-- Drop the old global unique constraint on key (correct name without schema prefix)
ALTER TABLE app.settings DROP CONSTRAINT IF EXISTS settings_key_key;

-- Also try the fully qualified name just in case
ALTER TABLE app.settings DROP CONSTRAINT IF EXISTS app_settings_key_key;
