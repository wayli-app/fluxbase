-- Drop chunked upload sessions table and indexes
DROP INDEX IF EXISTS storage.idx_chunked_sessions_owner;
DROP INDEX IF EXISTS storage.idx_chunked_sessions_expires;
DROP INDEX IF EXISTS storage.idx_chunked_sessions_status;
DROP INDEX IF EXISTS storage.idx_chunked_sessions_bucket;
DROP TABLE IF EXISTS storage.chunked_upload_sessions;
