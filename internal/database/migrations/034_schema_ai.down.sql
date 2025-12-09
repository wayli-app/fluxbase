-- AI Schema Rollback Migration
-- Removes all AI-related tables and schema

-- Remove settings
DELETE FROM app.settings WHERE key IN (
    'app.features.enable_ai',
    'app.ai.allow_user_provider_override',
    'app.ai.default_rate_limit_per_minute',
    'app.ai.default_daily_token_budget'
);

-- Drop tables in reverse dependency order
DROP TABLE IF EXISTS ai.user_chatbot_usage CASCADE;
DROP TABLE IF EXISTS ai.query_audit_log CASCADE;
DROP TABLE IF EXISTS ai.messages CASCADE;
DROP TABLE IF EXISTS ai.conversations CASCADE;
DROP TABLE IF EXISTS ai.chatbots CASCADE;
DROP TABLE IF EXISTS ai.user_provider_preferences CASCADE;
DROP TABLE IF EXISTS ai.providers CASCADE;

-- Drop schema
DROP SCHEMA IF EXISTS ai CASCADE;
