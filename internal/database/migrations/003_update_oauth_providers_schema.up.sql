-- Migration: Update OAuth Providers Schema
-- This migration updates the OAuth providers table to match the new schema

-- Rename provider column to provider_name
ALTER TABLE dashboard.oauth_providers RENAME COLUMN provider TO provider_name;

-- Add new columns
ALTER TABLE dashboard.oauth_providers ADD COLUMN IF NOT EXISTS display_name TEXT NOT NULL DEFAULT '';
ALTER TABLE dashboard.oauth_providers ADD COLUMN IF NOT EXISTS is_custom BOOLEAN DEFAULT FALSE;
ALTER TABLE dashboard.oauth_providers ADD COLUMN IF NOT EXISTS authorization_url TEXT;
ALTER TABLE dashboard.oauth_providers ADD COLUMN IF NOT EXISTS token_url TEXT;
ALTER TABLE dashboard.oauth_providers ADD COLUMN IF NOT EXISTS user_info_url TEXT;
ALTER TABLE dashboard.oauth_providers ADD COLUMN IF NOT EXISTS created_by UUID;
ALTER TABLE dashboard.oauth_providers ADD COLUMN IF NOT EXISTS updated_by UUID;

-- Update unique constraint to use new column name
ALTER TABLE dashboard.oauth_providers DROP CONSTRAINT IF EXISTS oauth_providers_provider_key;
ALTER TABLE dashboard.oauth_providers ADD CONSTRAINT oauth_providers_provider_name_key UNIQUE (provider_name);

-- Update index to use new column name
DROP INDEX IF EXISTS dashboard.idx_dashboard_oauth_providers_provider;
CREATE INDEX idx_dashboard_oauth_providers_provider_name ON dashboard.oauth_providers(provider_name);

-- Set display_name for existing providers (if any)
UPDATE dashboard.oauth_providers
SET display_name = INITCAP(provider_name)
WHERE display_name = '';
