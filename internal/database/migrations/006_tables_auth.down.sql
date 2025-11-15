-- Drop auth tables in reverse dependency order
DROP TABLE IF EXISTS auth.impersonation_sessions;
DROP TABLE IF EXISTS auth.webhook_events;
DROP TABLE IF EXISTS auth.webhook_deliveries;
DROP TABLE IF EXISTS auth.webhooks;
DROP TABLE IF EXISTS auth.two_factor_recovery_attempts;
DROP TABLE IF EXISTS auth.two_factor_setups;
DROP TABLE IF EXISTS auth.oauth_tokens;
DROP TABLE IF EXISTS auth.oauth_links;
DROP TABLE IF EXISTS auth.service_keys;
DROP TABLE IF EXISTS auth.api_key_usage;
DROP TABLE IF EXISTS auth.api_keys;
DROP TABLE IF EXISTS auth.token_blacklist;
DROP TABLE IF EXISTS auth.password_reset_tokens;
DROP TABLE IF EXISTS auth.magic_links;
DROP TABLE IF EXISTS auth.sessions;
DROP TABLE IF EXISTS auth.users;
