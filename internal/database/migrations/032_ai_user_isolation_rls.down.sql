-- Rollback: Remove user-level RLS policies and restore original policies

-- Drop user isolation policies
DROP POLICY IF EXISTS "ai_documents_user_isolation" ON ai.documents;
DROP POLICY IF EXISTS "ai_chunks_user_isolation" ON ai.chunks;

-- Drop the user_id index
DROP INDEX IF EXISTS ai.idx_ai_documents_user_id;

-- Restore original read policies (allow all authenticated users to read enabled content)
CREATE POLICY "ai_documents_read" ON ai.documents
    FOR SELECT TO authenticated
    USING (true);

CREATE POLICY "ai_chunks_read" ON ai.chunks
    FOR SELECT TO authenticated
    USING (true);
