-- Remove http_allowed_domains column from ai.chatbots table

DROP INDEX IF EXISTS idx_ai_chatbots_http_domains;
ALTER TABLE ai.chatbots DROP COLUMN IF EXISTS http_allowed_domains;
