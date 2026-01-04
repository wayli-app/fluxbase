-- Remove embedding provider preference feature
DROP INDEX IF EXISTS ai.idx_ai_providers_single_embedding;
ALTER TABLE ai.providers DROP COLUMN IF EXISTS use_for_embeddings;
