-- Add intent validation columns to ai.chatbots
ALTER TABLE ai.chatbots
ADD COLUMN IF NOT EXISTS intent_rules JSONB DEFAULT NULL,
ADD COLUMN IF NOT EXISTS required_columns JSONB DEFAULT NULL,
ADD COLUMN IF NOT EXISTS default_table TEXT DEFAULT NULL;

COMMENT ON COLUMN ai.chatbots.intent_rules IS 'Intent validation rules: [{keywords:[], requiredTable:"", forbiddenTable:""}]';
COMMENT ON COLUMN ai.chatbots.required_columns IS 'Required columns per table: {"table1":["col1","col2"]}';
COMMENT ON COLUMN ai.chatbots.default_table IS 'Default table for queries (from @fluxbase:default-table)';

-- Create index for chatbots with intent rules (for monitoring/analytics)
CREATE INDEX IF NOT EXISTS idx_ai_chatbots_has_intent_rules
ON ai.chatbots ((intent_rules IS NOT NULL));
