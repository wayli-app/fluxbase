-- Remove query_results column from ai.messages

DROP INDEX IF EXISTS ai.idx_ai_messages_has_query_results;

ALTER TABLE ai.messages
DROP COLUMN IF EXISTS query_results;
