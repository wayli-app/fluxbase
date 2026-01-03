--
-- Rollback OAuth Single Logout (SLO) Support
--

-- Drop logout states table
DROP TABLE IF EXISTS auth.oauth_logout_states;

-- Remove ID token column from oauth_tokens
ALTER TABLE auth.oauth_tokens
DROP COLUMN IF EXISTS id_token;

-- Remove logout endpoint columns from OAuth providers
ALTER TABLE dashboard.oauth_providers
DROP COLUMN IF EXISTS revocation_endpoint;

ALTER TABLE dashboard.oauth_providers
DROP COLUMN IF EXISTS end_session_endpoint;
