-- Knowledge Base Schema Migration
-- Creates tables for RAG/knowledge base functionality with vector search
-- Note: ai schema is created in 002_schemas

-- ============================================================================
-- KNOWLEDGE BASES
-- Collections of documents for RAG retrieval
-- ============================================================================

CREATE TABLE IF NOT EXISTS ai.knowledge_bases (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL,
    namespace TEXT NOT NULL DEFAULT 'default',
    description TEXT,

    -- Embedding configuration
    embedding_model TEXT DEFAULT 'text-embedding-3-small',
    embedding_dimensions INTEGER DEFAULT 1536,

    -- Chunking configuration
    chunk_size INTEGER DEFAULT 512,          -- Target tokens per chunk
    chunk_overlap INTEGER DEFAULT 50,        -- Overlap tokens between chunks
    chunk_strategy TEXT DEFAULT 'recursive', -- 'recursive', 'sentence', 'paragraph', 'fixed'

    -- Metadata
    enabled BOOLEAN DEFAULT true,
    document_count INTEGER DEFAULT 0,
    total_chunks INTEGER DEFAULT 0,

    source TEXT NOT NULL DEFAULT 'api' CHECK (source IN ('filesystem', 'api', 'sdk')),
    created_by UUID REFERENCES auth.users(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),

    CONSTRAINT unique_knowledge_base_name_namespace UNIQUE (name, namespace)
);

CREATE INDEX IF NOT EXISTS idx_ai_knowledge_bases_name ON ai.knowledge_bases(name);
CREATE INDEX IF NOT EXISTS idx_ai_knowledge_bases_namespace ON ai.knowledge_bases(namespace);
CREATE INDEX IF NOT EXISTS idx_ai_knowledge_bases_enabled ON ai.knowledge_bases(enabled);

COMMENT ON TABLE ai.knowledge_bases IS 'Knowledge base collections for RAG retrieval';
COMMENT ON COLUMN ai.knowledge_bases.chunk_size IS 'Target number of tokens per chunk';
COMMENT ON COLUMN ai.knowledge_bases.chunk_overlap IS 'Number of overlapping tokens between chunks';
COMMENT ON COLUMN ai.knowledge_bases.chunk_strategy IS 'Chunking strategy: recursive (default), sentence, paragraph, or fixed';

-- ============================================================================
-- DOCUMENTS
-- Source documents within knowledge bases
-- ============================================================================

CREATE TABLE IF NOT EXISTS ai.documents (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    knowledge_base_id UUID NOT NULL REFERENCES ai.knowledge_bases(id) ON DELETE CASCADE,

    -- Document identification
    title TEXT,
    source_url TEXT,                   -- Original URL if web-scraped or external reference
    source_type TEXT DEFAULT 'manual', -- 'file', 'url', 'api', 'manual'
    mime_type TEXT,

    -- Content
    content TEXT NOT NULL,             -- Full document content
    content_hash TEXT,                 -- SHA-256 hash for deduplication and change detection

    -- Processing status
    status TEXT DEFAULT 'pending' CHECK (status IN ('pending', 'processing', 'indexed', 'failed')),
    error_message TEXT,
    chunks_count INTEGER DEFAULT 0,

    -- Metadata for filtering
    metadata JSONB DEFAULT '{}',
    tags TEXT[] DEFAULT ARRAY[]::TEXT[],

    -- Timestamps
    created_by UUID REFERENCES auth.users(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    indexed_at TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_ai_documents_knowledge_base ON ai.documents(knowledge_base_id);
CREATE INDEX IF NOT EXISTS idx_ai_documents_status ON ai.documents(status);
CREATE INDEX IF NOT EXISTS idx_ai_documents_source_type ON ai.documents(source_type);
CREATE INDEX IF NOT EXISTS idx_ai_documents_content_hash ON ai.documents(content_hash);
CREATE INDEX IF NOT EXISTS idx_ai_documents_tags ON ai.documents USING GIN (tags);
CREATE INDEX IF NOT EXISTS idx_ai_documents_metadata ON ai.documents USING GIN (metadata);

COMMENT ON TABLE ai.documents IS 'Source documents in knowledge bases';
COMMENT ON COLUMN ai.documents.content_hash IS 'SHA-256 hash for detecting duplicate or changed content';
COMMENT ON COLUMN ai.documents.metadata IS 'Custom metadata for filtering during retrieval';

-- ============================================================================
-- CHUNKS
-- Document chunks with embeddings for vector search
-- ============================================================================

CREATE TABLE IF NOT EXISTS ai.chunks (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    document_id UUID NOT NULL REFERENCES ai.documents(id) ON DELETE CASCADE,
    knowledge_base_id UUID NOT NULL REFERENCES ai.knowledge_bases(id) ON DELETE CASCADE,

    -- Chunk content
    content TEXT NOT NULL,
    chunk_index INTEGER NOT NULL,      -- Position in document (0-based)
    start_offset INTEGER,              -- Character offset in original document
    end_offset INTEGER,
    token_count INTEGER,

    -- Vector embedding (using pgvector)
    -- Default 1536 dimensions for OpenAI text-embedding-3-small
    -- Other common sizes: 384 (all-MiniLM), 768 (nomic-embed), 1024 (mxbai), 3072 (text-embedding-3-large)
    embedding vector(1536),

    -- Metadata inherited from document + chunk-specific
    metadata JSONB DEFAULT '{}',

    created_at TIMESTAMPTZ DEFAULT NOW(),

    CONSTRAINT unique_chunk_document_index UNIQUE (document_id, chunk_index)
);

-- Vector similarity search indexes (IVFFlat for approximate nearest neighbor)
-- Using cosine distance as default metric
CREATE INDEX IF NOT EXISTS idx_ai_chunks_embedding_cosine ON ai.chunks
    USING ivfflat (embedding vector_cosine_ops) WITH (lists = 100);

-- Also create L2 distance index for flexibility
CREATE INDEX IF NOT EXISTS idx_ai_chunks_embedding_l2 ON ai.chunks
    USING ivfflat (embedding vector_l2_ops) WITH (lists = 100);

CREATE INDEX IF NOT EXISTS idx_ai_chunks_document ON ai.chunks(document_id);
CREATE INDEX IF NOT EXISTS idx_ai_chunks_knowledge_base ON ai.chunks(knowledge_base_id);
CREATE INDEX IF NOT EXISTS idx_ai_chunks_metadata ON ai.chunks USING GIN (metadata);

COMMENT ON TABLE ai.chunks IS 'Document chunks with vector embeddings for semantic search';
COMMENT ON COLUMN ai.chunks.embedding IS 'Vector embedding from configured embedding model';
COMMENT ON COLUMN ai.chunks.chunk_index IS 'Zero-based index of this chunk within the document';

-- ============================================================================
-- CHATBOT KNOWLEDGE BASE LINKS
-- Many-to-many relationship between chatbots and knowledge bases
-- ============================================================================

CREATE TABLE IF NOT EXISTS ai.chatbot_knowledge_bases (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    chatbot_id UUID NOT NULL REFERENCES ai.chatbots(id) ON DELETE CASCADE,
    knowledge_base_id UUID NOT NULL REFERENCES ai.knowledge_bases(id) ON DELETE CASCADE,

    -- Retrieval configuration
    enabled BOOLEAN DEFAULT true,
    max_chunks INTEGER DEFAULT 5,              -- Max chunks to retrieve per query
    similarity_threshold FLOAT DEFAULT 0.7,    -- Minimum similarity score (0-1)

    -- Priority for multiple knowledge bases
    priority INTEGER DEFAULT 0,                -- Higher = checked first

    created_at TIMESTAMPTZ DEFAULT NOW(),

    CONSTRAINT unique_chatbot_knowledge_base UNIQUE (chatbot_id, knowledge_base_id)
);

CREATE INDEX IF NOT EXISTS idx_ai_chatbot_kb_chatbot ON ai.chatbot_knowledge_bases(chatbot_id);
CREATE INDEX IF NOT EXISTS idx_ai_chatbot_kb_knowledge_base ON ai.chatbot_knowledge_bases(knowledge_base_id);
CREATE INDEX IF NOT EXISTS idx_ai_chatbot_kb_enabled ON ai.chatbot_knowledge_bases(enabled) WHERE enabled = true;

COMMENT ON TABLE ai.chatbot_knowledge_bases IS 'Links chatbots to their knowledge bases for RAG retrieval';
COMMENT ON COLUMN ai.chatbot_knowledge_bases.max_chunks IS 'Maximum number of chunks to retrieve per user query';
COMMENT ON COLUMN ai.chatbot_knowledge_bases.similarity_threshold IS 'Minimum cosine similarity score (0-1) to include chunk';

-- ============================================================================
-- RETRIEVAL AUDIT LOG
-- Track RAG retrievals for debugging and analytics
-- ============================================================================

CREATE TABLE IF NOT EXISTS ai.retrieval_log (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    chatbot_id UUID REFERENCES ai.chatbots(id) ON DELETE SET NULL,
    conversation_id UUID REFERENCES ai.conversations(id) ON DELETE SET NULL,
    knowledge_base_id UUID REFERENCES ai.knowledge_bases(id) ON DELETE SET NULL,
    user_id UUID REFERENCES auth.users(id) ON DELETE SET NULL,

    -- Query info
    query_text TEXT NOT NULL,
    query_embedding_model TEXT,

    -- Results
    chunks_retrieved INTEGER DEFAULT 0,
    chunk_ids UUID[],
    similarity_scores FLOAT[],

    -- Performance
    retrieval_duration_ms INTEGER,

    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_ai_retrieval_log_chatbot ON ai.retrieval_log(chatbot_id);
CREATE INDEX IF NOT EXISTS idx_ai_retrieval_log_kb ON ai.retrieval_log(knowledge_base_id);
CREATE INDEX IF NOT EXISTS idx_ai_retrieval_log_created ON ai.retrieval_log(created_at DESC);

COMMENT ON TABLE ai.retrieval_log IS 'Audit log for RAG retrieval operations';

-- ============================================================================
-- ROW LEVEL SECURITY
-- ============================================================================

ALTER TABLE ai.knowledge_bases ENABLE ROW LEVEL SECURITY;
ALTER TABLE ai.documents ENABLE ROW LEVEL SECURITY;
ALTER TABLE ai.chunks ENABLE ROW LEVEL SECURITY;
ALTER TABLE ai.chatbot_knowledge_bases ENABLE ROW LEVEL SECURITY;
ALTER TABLE ai.retrieval_log ENABLE ROW LEVEL SECURITY;

-- Service role can do everything (bypasses RLS)
DO $$ BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_policies WHERE schemaname = 'ai' AND tablename = 'knowledge_bases' AND policyname = 'ai_kb_service_all') THEN
        CREATE POLICY "ai_kb_service_all" ON ai.knowledge_bases FOR ALL TO service_role USING (true);
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_policies WHERE schemaname = 'ai' AND tablename = 'documents' AND policyname = 'ai_documents_service_all') THEN
        CREATE POLICY "ai_documents_service_all" ON ai.documents FOR ALL TO service_role USING (true);
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_policies WHERE schemaname = 'ai' AND tablename = 'chunks' AND policyname = 'ai_chunks_service_all') THEN
        CREATE POLICY "ai_chunks_service_all" ON ai.chunks FOR ALL TO service_role USING (true);
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_policies WHERE schemaname = 'ai' AND tablename = 'chatbot_knowledge_bases' AND policyname = 'ai_chatbot_kb_service_all') THEN
        CREATE POLICY "ai_chatbot_kb_service_all" ON ai.chatbot_knowledge_bases FOR ALL TO service_role USING (true);
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_policies WHERE schemaname = 'ai' AND tablename = 'retrieval_log' AND policyname = 'ai_retrieval_log_service_all') THEN
        CREATE POLICY "ai_retrieval_log_service_all" ON ai.retrieval_log FOR ALL TO service_role USING (true);
    END IF;
END $$;

-- Authenticated users can read enabled knowledge bases (for listing)
DO $$ BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_policies WHERE schemaname = 'ai' AND tablename = 'knowledge_bases' AND policyname = 'ai_kb_read') THEN
        CREATE POLICY "ai_kb_read" ON ai.knowledge_bases
            FOR SELECT TO authenticated
            USING (enabled = true);
    END IF;
END $$;

-- Dashboard admins can manage knowledge bases
DO $$ BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_policies WHERE schemaname = 'ai' AND tablename = 'knowledge_bases' AND policyname = 'ai_kb_dashboard_admin') THEN
        CREATE POLICY "ai_kb_dashboard_admin" ON ai.knowledge_bases
            FOR ALL TO authenticated
            USING (auth.role() = 'dashboard_admin');
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_policies WHERE schemaname = 'ai' AND tablename = 'documents' AND policyname = 'ai_documents_dashboard_admin') THEN
        CREATE POLICY "ai_documents_dashboard_admin" ON ai.documents
            FOR ALL TO authenticated
            USING (auth.role() = 'dashboard_admin');
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_policies WHERE schemaname = 'ai' AND tablename = 'chunks' AND policyname = 'ai_chunks_dashboard_admin') THEN
        CREATE POLICY "ai_chunks_dashboard_admin" ON ai.chunks
            FOR ALL TO authenticated
            USING (auth.role() = 'dashboard_admin');
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_policies WHERE schemaname = 'ai' AND tablename = 'chatbot_knowledge_bases' AND policyname = 'ai_chatbot_kb_dashboard_admin') THEN
        CREATE POLICY "ai_chatbot_kb_dashboard_admin" ON ai.chatbot_knowledge_bases
            FOR ALL TO authenticated
            USING (auth.role() = 'dashboard_admin');
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_policies WHERE schemaname = 'ai' AND tablename = 'retrieval_log' AND policyname = 'ai_retrieval_log_dashboard_admin') THEN
        CREATE POLICY "ai_retrieval_log_dashboard_admin" ON ai.retrieval_log
            FOR SELECT TO authenticated
            USING (auth.role() = 'dashboard_admin');
    END IF;
END $$;

-- ============================================================================
-- PERMISSIONS
-- ============================================================================

GRANT SELECT ON ai.knowledge_bases TO authenticated;
GRANT ALL ON ai.knowledge_bases TO service_role;

GRANT SELECT ON ai.documents TO authenticated;
GRANT ALL ON ai.documents TO service_role;

GRANT SELECT ON ai.chunks TO authenticated;
GRANT ALL ON ai.chunks TO service_role;

GRANT SELECT ON ai.chatbot_knowledge_bases TO authenticated;
GRANT ALL ON ai.chatbot_knowledge_bases TO service_role;

GRANT SELECT ON ai.retrieval_log TO authenticated;
GRANT ALL ON ai.retrieval_log TO service_role;

-- ============================================================================
-- TRIGGERS
-- ============================================================================

-- Auto-update timestamps
CREATE OR REPLACE FUNCTION ai.update_knowledge_base_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS knowledge_bases_update_updated_at ON ai.knowledge_bases;
CREATE TRIGGER knowledge_bases_update_updated_at
BEFORE UPDATE ON ai.knowledge_bases
FOR EACH ROW EXECUTE FUNCTION ai.update_knowledge_base_updated_at();

DROP TRIGGER IF EXISTS documents_update_updated_at ON ai.documents;
CREATE TRIGGER documents_update_updated_at
BEFORE UPDATE ON ai.documents
FOR EACH ROW EXECUTE FUNCTION ai.update_knowledge_base_updated_at();

-- Update knowledge base counters when documents change
CREATE OR REPLACE FUNCTION ai.update_knowledge_base_counts()
RETURNS TRIGGER AS $$
BEGIN
    IF TG_OP = 'INSERT' THEN
        UPDATE ai.knowledge_bases
        SET document_count = document_count + 1,
            updated_at = NOW()
        WHERE id = NEW.knowledge_base_id;
    ELSIF TG_OP = 'DELETE' THEN
        UPDATE ai.knowledge_bases
        SET document_count = GREATEST(0, document_count - 1),
            updated_at = NOW()
        WHERE id = OLD.knowledge_base_id;
    END IF;
    RETURN COALESCE(NEW, OLD);
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS documents_update_kb_counts ON ai.documents;
CREATE TRIGGER documents_update_kb_counts
AFTER INSERT OR DELETE ON ai.documents
FOR EACH ROW EXECUTE FUNCTION ai.update_knowledge_base_counts();

-- Update knowledge base chunk count when chunks change
CREATE OR REPLACE FUNCTION ai.update_knowledge_base_chunk_count()
RETURNS TRIGGER AS $$
BEGIN
    IF TG_OP = 'INSERT' THEN
        UPDATE ai.knowledge_bases
        SET total_chunks = total_chunks + 1
        WHERE id = NEW.knowledge_base_id;
        UPDATE ai.documents
        SET chunks_count = chunks_count + 1
        WHERE id = NEW.document_id;
    ELSIF TG_OP = 'DELETE' THEN
        UPDATE ai.knowledge_bases
        SET total_chunks = GREATEST(0, total_chunks - 1)
        WHERE id = OLD.knowledge_base_id;
        UPDATE ai.documents
        SET chunks_count = GREATEST(0, chunks_count - 1)
        WHERE id = OLD.document_id;
    END IF;
    RETURN COALESCE(NEW, OLD);
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS chunks_update_counts ON ai.chunks;
CREATE TRIGGER chunks_update_counts
AFTER INSERT OR DELETE ON ai.chunks
FOR EACH ROW EXECUTE FUNCTION ai.update_knowledge_base_chunk_count();

-- ============================================================================
-- HELPER FUNCTIONS
-- ============================================================================

-- Function to search chunks by similarity
CREATE OR REPLACE FUNCTION ai.search_chunks(
    p_knowledge_base_id UUID,
    p_query_embedding vector(1536),
    p_limit INTEGER DEFAULT 5,
    p_threshold FLOAT DEFAULT 0.7
)
RETURNS TABLE (
    chunk_id UUID,
    document_id UUID,
    content TEXT,
    similarity FLOAT,
    metadata JSONB
) AS $$
BEGIN
    RETURN QUERY
    SELECT
        c.id as chunk_id,
        c.document_id,
        c.content,
        1 - (c.embedding <=> p_query_embedding) as similarity,
        c.metadata
    FROM ai.chunks c
    WHERE c.knowledge_base_id = p_knowledge_base_id
      AND 1 - (c.embedding <=> p_query_embedding) >= p_threshold
    ORDER BY c.embedding <=> p_query_embedding
    LIMIT p_limit;
END;
$$ LANGUAGE plpgsql SECURITY DEFINER;

-- Function to search across multiple knowledge bases
CREATE OR REPLACE FUNCTION ai.search_chatbot_knowledge(
    p_chatbot_id UUID,
    p_query_embedding vector(1536)
)
RETURNS TABLE (
    chunk_id UUID,
    document_id UUID,
    knowledge_base_id UUID,
    knowledge_base_name TEXT,
    content TEXT,
    similarity FLOAT,
    metadata JSONB
) AS $$
BEGIN
    RETURN QUERY
    SELECT
        c.id as chunk_id,
        c.document_id,
        c.knowledge_base_id,
        kb.name as knowledge_base_name,
        c.content,
        1 - (c.embedding <=> p_query_embedding) as similarity,
        c.metadata
    FROM ai.chatbot_knowledge_bases ckb
    JOIN ai.knowledge_bases kb ON kb.id = ckb.knowledge_base_id
    JOIN ai.chunks c ON c.knowledge_base_id = kb.id
    WHERE ckb.chatbot_id = p_chatbot_id
      AND ckb.enabled = true
      AND kb.enabled = true
      AND 1 - (c.embedding <=> p_query_embedding) >= ckb.similarity_threshold
    ORDER BY ckb.priority DESC, c.embedding <=> p_query_embedding
    LIMIT (
        SELECT SUM(max_chunks) FROM ai.chatbot_knowledge_bases
        WHERE chatbot_id = p_chatbot_id AND enabled = true
    );
END;
$$ LANGUAGE plpgsql SECURITY DEFINER;

COMMENT ON FUNCTION ai.search_chunks IS 'Search chunks in a knowledge base by vector similarity';
COMMENT ON FUNCTION ai.search_chatbot_knowledge IS 'Search all knowledge bases linked to a chatbot';
