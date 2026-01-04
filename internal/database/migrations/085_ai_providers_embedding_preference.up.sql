-- Add embedding provider preference column
-- This allows users to explicitly select which AI provider to use for embeddings
-- NULL = auto (follow default provider)
-- TRUE = explicitly use this provider for embeddings
ALTER TABLE ai.providers
ADD COLUMN use_for_embeddings BOOLEAN DEFAULT NULL;

-- Create unique partial index to ensure only one embedding provider
-- NULL values are ignored by this index, so only one provider can have use_for_embeddings = true
CREATE UNIQUE INDEX idx_ai_providers_single_embedding
ON ai.providers(use_for_embeddings)
WHERE use_for_embeddings = true;

-- Add comment for documentation
COMMENT ON COLUMN ai.providers.use_for_embeddings IS
'When true, this provider is explicitly used for embedding generation. NULL means follow default provider (auto mode).';
