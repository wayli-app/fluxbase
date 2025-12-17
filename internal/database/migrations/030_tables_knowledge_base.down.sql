-- Rollback Knowledge Base Schema Migration

-- Drop helper functions
DROP FUNCTION IF EXISTS ai.search_chatbot_knowledge(UUID, vector(1536));
DROP FUNCTION IF EXISTS ai.search_chunks(UUID, vector(1536), INTEGER, FLOAT);

-- Drop triggers
DROP TRIGGER IF EXISTS chunks_update_counts ON ai.chunks;
DROP TRIGGER IF EXISTS documents_update_kb_counts ON ai.documents;
DROP TRIGGER IF EXISTS documents_update_updated_at ON ai.documents;
DROP TRIGGER IF EXISTS knowledge_bases_update_updated_at ON ai.knowledge_bases;

-- Drop trigger functions
DROP FUNCTION IF EXISTS ai.update_knowledge_base_chunk_count();
DROP FUNCTION IF EXISTS ai.update_knowledge_base_counts();
DROP FUNCTION IF EXISTS ai.update_knowledge_base_updated_at();

-- Drop tables in reverse order (respecting foreign keys)
DROP TABLE IF EXISTS ai.retrieval_log;
DROP TABLE IF EXISTS ai.chatbot_knowledge_bases;
DROP TABLE IF EXISTS ai.chunks;
DROP TABLE IF EXISTS ai.documents;
DROP TABLE IF EXISTS ai.knowledge_bases;
