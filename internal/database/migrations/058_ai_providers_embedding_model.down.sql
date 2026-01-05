-- Remove embedding model configuration
ALTER TABLE ai.providers DROP COLUMN IF EXISTS embedding_model;
