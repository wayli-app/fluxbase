-- AI User Isolation Migration
-- Implements user-level isolation for AI/Knowledge Base documents
-- Combines storage configuration, RLS policies, and auto-population triggers

-- ============================================================================
-- KNOWLEDGE BASE DOCUMENTS STORAGE BUCKET
-- Storage bucket for uploaded documents in knowledge bases
-- ============================================================================

-- Create the knowledge-base bucket
INSERT INTO storage.buckets (id, name, public, allowed_mime_types, max_file_size)
VALUES (
    'knowledge-base',
    'knowledge-base',
    false,
    ARRAY[
        'application/pdf',
        'text/plain',
        'text/markdown',
        'text/html',
        'text/csv',
        'application/vnd.openxmlformats-officedocument.wordprocessingml.document',
        'application/vnd.openxmlformats-officedocument.spreadsheetml.sheet',
        'application/rtf',
        'application/epub+zip',
        'application/json'
    ],
    52428800  -- 50MB in bytes
)
ON CONFLICT (id) DO UPDATE SET
    allowed_mime_types = EXCLUDED.allowed_mime_types,
    max_file_size = EXCLUDED.max_file_size;

-- Add storage_object_id column to ai.documents if it doesn't exist
ALTER TABLE ai.documents
ADD COLUMN IF NOT EXISTS storage_object_id UUID REFERENCES storage.objects(id) ON DELETE SET NULL;

-- Add original_filename column to ai.documents if it doesn't exist
ALTER TABLE ai.documents
ADD COLUMN IF NOT EXISTS original_filename TEXT;

-- Create index for looking up documents by storage object
CREATE INDEX IF NOT EXISTS idx_ai_documents_storage_object_id ON ai.documents(storage_object_id);

COMMENT ON COLUMN ai.documents.storage_object_id IS 'Reference to the uploaded file in storage.objects';
COMMENT ON COLUMN ai.documents.original_filename IS 'Original filename of the uploaded document';

-- ============================================================================
-- USER-LEVEL RLS POLICIES
-- Users can only see their own documents/chunks + global content (no user_id)
-- ============================================================================

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

-- ============================================================================
-- USER ID AUTO-POPULATION TRIGGERS
-- SECURITY: Auto-populates user_id in document/chunk metadata from JWT claims
-- This prevents documents from being inadvertently visible to all users
-- when developers forget to set metadata->>'user_id' during document creation
-- ============================================================================

-- Function to auto-set user_id in document metadata
CREATE OR REPLACE FUNCTION ai.set_document_user_id()
RETURNS TRIGGER AS $$
DECLARE
    v_user_id TEXT;
    v_role TEXT;
BEGIN
    -- Get current user ID from JWT claims
    BEGIN
        v_user_id := current_setting('request.jwt.claims', true)::json->>'sub';
        v_role := current_setting('request.jwt.claims', true)::json->>'role';
    EXCEPTION WHEN OTHERS THEN
        v_user_id := NULL;
        v_role := NULL;
    END;

    -- Skip if service_role (admin operations may want global documents)
    -- Or if user_id is already explicitly set in metadata
    IF v_role = 'service_role' THEN
        -- Service role can create global documents
        RETURN NEW;
    END IF;

    -- Initialize metadata if NULL
    IF NEW.metadata IS NULL THEN
        NEW.metadata = '{}'::jsonb;
    END IF;

    -- Only set user_id if not already present AND we have a user context
    IF NEW.metadata->>'user_id' IS NULL AND v_user_id IS NOT NULL THEN
        NEW.metadata = jsonb_set(
            NEW.metadata,
            '{user_id}',
            to_jsonb(v_user_id)
        );
    END IF;

    RETURN NEW;
END;
$$ LANGUAGE plpgsql SECURITY DEFINER;

COMMENT ON FUNCTION ai.set_document_user_id() IS 'Auto-populates user_id in document metadata from JWT claims for RLS enforcement';

-- Create trigger on document insert
DROP TRIGGER IF EXISTS set_document_user_id_trigger ON ai.documents;
CREATE TRIGGER set_document_user_id_trigger
    BEFORE INSERT ON ai.documents
    FOR EACH ROW
    EXECUTE FUNCTION ai.set_document_user_id();

-- Also apply to chunks table for direct chunk inserts
CREATE OR REPLACE FUNCTION ai.set_chunk_user_id()
RETURNS TRIGGER AS $$
DECLARE
    v_user_id TEXT;
    v_role TEXT;
BEGIN
    -- Get current user ID from JWT claims
    BEGIN
        v_user_id := current_setting('request.jwt.claims', true)::json->>'sub';
        v_role := current_setting('request.jwt.claims', true)::json->>'role';
    EXCEPTION WHEN OTHERS THEN
        v_user_id := NULL;
        v_role := NULL;
    END;

    -- Skip if service_role
    IF v_role = 'service_role' THEN
        RETURN NEW;
    END IF;

    -- Initialize metadata if NULL
    IF NEW.metadata IS NULL THEN
        NEW.metadata = '{}'::jsonb;
    END IF;

    -- Only set user_id if not already present AND we have a user context
    IF NEW.metadata->>'user_id' IS NULL AND v_user_id IS NOT NULL THEN
        NEW.metadata = jsonb_set(
            NEW.metadata,
            '{user_id}',
            to_jsonb(v_user_id)
        );
    END IF;

    RETURN NEW;
END;
$$ LANGUAGE plpgsql SECURITY DEFINER;

COMMENT ON FUNCTION ai.set_chunk_user_id() IS 'Auto-populates user_id in chunk metadata from JWT claims for RLS enforcement';

DROP TRIGGER IF EXISTS set_chunk_user_id_trigger ON ai.chunks;
CREATE TRIGGER set_chunk_user_id_trigger
    BEFORE INSERT ON ai.chunks
    FOR EACH ROW
    EXECUTE FUNCTION ai.set_chunk_user_id();
