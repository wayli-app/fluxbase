-- Add http_allowed_domains column to ai.chatbots table
-- This enables the http_request tool for chatbots to make external API calls

ALTER TABLE ai.chatbots
ADD COLUMN IF NOT EXISTS http_allowed_domains TEXT[] DEFAULT ARRAY[]::TEXT[];

-- Add GIN index for efficient array lookups
CREATE INDEX IF NOT EXISTS idx_ai_chatbots_http_domains
ON ai.chatbots USING GIN (http_allowed_domains);

COMMENT ON COLUMN ai.chatbots.http_allowed_domains IS 'Allowed domains for HTTP requests (from @fluxbase:http-allowed-domains annotation)';
