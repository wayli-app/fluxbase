-- Add embedding model configuration per provider
ALTER TABLE ai.providers
ADD COLUMN embedding_model TEXT DEFAULT NULL;

COMMENT ON COLUMN ai.providers.embedding_model IS
'Embedding model to use for this provider. NULL means use provider-specific default (e.g., text-embedding-3-small for OpenAI).';
