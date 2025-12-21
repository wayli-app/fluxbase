-- Remove response_language column from ai.chatbots

ALTER TABLE ai.chatbots DROP COLUMN IF EXISTS response_language;
