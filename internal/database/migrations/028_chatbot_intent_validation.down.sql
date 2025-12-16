-- Remove intent validation columns from ai.chatbots
DROP INDEX IF EXISTS ai.idx_ai_chatbots_has_intent_rules;

ALTER TABLE ai.chatbots
DROP COLUMN IF EXISTS intent_rules,
DROP COLUMN IF EXISTS required_columns,
DROP COLUMN IF EXISTS default_table;
