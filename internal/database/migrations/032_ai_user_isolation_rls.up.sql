-- Migration: Add user-level RLS policies for AI tables
-- This ensures users can only see their own documents/chunks + global content (no user_id)

-- Drop existing read policy for authenticated users on documents (if exists)
DROP POLICY IF EXISTS "ai_documents_read" ON ai.documents;

-- User-level isolation policy for documents
-- Users see: their own content (user_id matches) OR global content (no user_id in metadata)
CREATE POLICY "ai_documents_user_isolation" ON ai.documents
    FOR SELECT TO authenticated
    USING (
        metadata->>'user_id' IS NULL OR
        metadata->>'user_id' = (current_setting('request.jwt.claims', true)::json->>'sub')
    );

-- Drop existing read policy for authenticated users on chunks (if exists)
DROP POLICY IF EXISTS "ai_chunks_read" ON ai.chunks;

-- User-level isolation policy for chunks
-- Chunks inherit access from their parent document
CREATE POLICY "ai_chunks_user_isolation" ON ai.chunks
    FOR SELECT TO authenticated
    USING (
        EXISTS (
            SELECT 1 FROM ai.documents d
            WHERE d.id = ai.chunks.document_id
            AND (
                d.metadata->>'user_id' IS NULL OR
                d.metadata->>'user_id' = (current_setting('request.jwt.claims', true)::json->>'sub')
            )
        )
    );

-- Create index on metadata->>'user_id' for better query performance
CREATE INDEX IF NOT EXISTS idx_ai_documents_user_id
    ON ai.documents ((metadata->>'user_id'));
