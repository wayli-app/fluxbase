-- Rollback: Update OAuth Providers Schema

-- Remove new columns
ALTER TABLE dashboard.oauth_providers DROP COLUMN IF EXISTS updated_by;
ALTER TABLE dashboard.oauth_providers DROP COLUMN IF EXISTS created_by;
ALTER TABLE dashboard.oauth_providers DROP COLUMN IF EXISTS user_info_url;
ALTER TABLE dashboard.oauth_providers DROP COLUMN IF EXISTS token_url;
ALTER TABLE dashboard.oauth_providers DROP COLUMN IF EXISTS authorization_url;
ALTER TABLE dashboard.oauth_providers DROP COLUMN IF EXISTS is_custom;
ALTER TABLE dashboard.oauth_providers DROP COLUMN IF EXISTS display_name;

-- Rename provider_name back to provider
ALTER TABLE dashboard.oauth_providers RENAME COLUMN provider_name TO provider;

-- Restore old unique constraint
ALTER TABLE dashboard.oauth_providers DROP CONSTRAINT IF EXISTS oauth_providers_provider_name_key;
ALTER TABLE dashboard.oauth_providers ADD CONSTRAINT oauth_providers_provider_key UNIQUE (provider);

-- Restore old index
DROP INDEX IF EXISTS dashboard.idx_dashboard_oauth_providers_provider_name;
CREATE INDEX idx_dashboard_oauth_providers_provider ON dashboard.oauth_providers(provider);
