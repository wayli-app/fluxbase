-- Add response_language column to ai.chatbots
-- Allows chatbots to enforce a specific response language or auto-detect from user message

ALTER TABLE ai.chatbots ADD COLUMN IF NOT EXISTS response_language TEXT DEFAULT 'auto';

COMMENT ON COLUMN ai.chatbots.response_language IS
    'Response language setting: "auto" (match user language), ISO code (e.g., "en"), or language name (e.g., "German")';
