-- Rollback: OAuth User Linking and Token Storage

-- Drop RLS policies
DROP POLICY IF EXISTS oauth_tokens_service_all ON auth.oauth_tokens;
DROP POLICY IF EXISTS oauth_links_service_all ON auth.oauth_links;
DROP POLICY IF EXISTS oauth_tokens_select ON auth.oauth_tokens;
DROP POLICY IF EXISTS oauth_links_select ON auth.oauth_links;

-- Drop tables
DROP TABLE IF EXISTS auth.oauth_tokens;
DROP TABLE IF EXISTS auth.oauth_links;
