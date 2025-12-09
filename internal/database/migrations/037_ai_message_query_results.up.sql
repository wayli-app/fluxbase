-- Add query_results JSONB column to store full query results for assistant messages
-- This enables persisting SQL query results so they can be displayed when loading
-- conversations from history.

ALTER TABLE ai.messages
ADD COLUMN IF NOT EXISTS query_results JSONB DEFAULT NULL;

CREATE INDEX IF NOT EXISTS idx_ai_messages_has_query_results
    ON ai.messages(conversation_id)
    WHERE query_results IS NOT NULL;

COMMENT ON COLUMN ai.messages.query_results IS 'Array of query results with query, summary, row_count, and data for assistant messages';
