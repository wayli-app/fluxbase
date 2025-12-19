--
-- KNOWLEDGE BASE DOCUMENTS STORAGE BUCKET
-- Storage bucket for uploaded documents in knowledge bases
--

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
