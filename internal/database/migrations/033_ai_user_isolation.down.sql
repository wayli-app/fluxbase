-- Rollback: AI User Isolation Migration
-- Reverses user-level isolation for AI/Knowledge Base documents

-- ============================================================================
-- ROLLBACK TRIGGERS (must be first)
-- ============================================================================

DROP TRIGGER IF EXISTS set_document_user_id_trigger ON ai.documents;
DROP FUNCTION IF EXISTS ai.set_document_user_id();

DROP TRIGGER IF EXISTS set_chunk_user_id_trigger ON ai.chunks;
DROP FUNCTION IF EXISTS ai.set_chunk_user_id();

-- ============================================================================
-- ROLLBACK RLS POLICIES
-- ============================================================================

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

-- ============================================================================
-- ROLLBACK STORAGE BUCKET
-- ============================================================================

-- Remove the index
DROP INDEX IF EXISTS ai.idx_ai_documents_storage_object_id;

-- Remove the columns from ai.documents
ALTER TABLE ai.documents DROP COLUMN IF EXISTS storage_object_id;
ALTER TABLE ai.documents DROP COLUMN IF EXISTS original_filename;

-- Delete the bucket (this will cascade delete any objects in it)
DELETE FROM storage.buckets WHERE id = 'knowledge-base';
