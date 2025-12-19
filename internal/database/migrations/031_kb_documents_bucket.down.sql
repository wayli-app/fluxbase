--
-- ROLLBACK: KNOWLEDGE BASE DOCUMENTS STORAGE BUCKET
--

-- Remove the index
DROP INDEX IF EXISTS ai.idx_ai_documents_storage_object_id;

-- Remove the columns from ai.documents
ALTER TABLE ai.documents DROP COLUMN IF EXISTS storage_object_id;
ALTER TABLE ai.documents DROP COLUMN IF EXISTS original_filename;

-- Delete the bucket (this will cascade delete any objects in it)
DELETE FROM storage.buckets WHERE id = 'knowledge-base';
