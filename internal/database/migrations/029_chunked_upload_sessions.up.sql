-- Chunked upload sessions for resumable large file uploads
CREATE TABLE IF NOT EXISTS storage.chunked_upload_sessions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    upload_id TEXT UNIQUE NOT NULL,
    bucket_id TEXT NOT NULL,
    path TEXT NOT NULL,
    total_size BIGINT NOT NULL,
    chunk_size INTEGER NOT NULL,
    total_chunks INTEGER NOT NULL,
    completed_chunks INTEGER[] DEFAULT '{}',
    content_type TEXT,
    metadata JSONB,
    cache_control TEXT,
    owner_id UUID,

    -- S3 multipart specific fields
    s3_upload_id TEXT,
    s3_part_etags JSONB,

    -- Session lifecycle
    status TEXT DEFAULT 'active' CHECK (status IN ('active', 'completing', 'completed', 'aborted', 'expired')),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    expires_at TIMESTAMPTZ DEFAULT (NOW() + INTERVAL '24 hours')
);

-- Indexes for efficient queries
CREATE INDEX idx_chunked_sessions_bucket ON storage.chunked_upload_sessions(bucket_id);
CREATE INDEX idx_chunked_sessions_status ON storage.chunked_upload_sessions(status);
CREATE INDEX idx_chunked_sessions_expires ON storage.chunked_upload_sessions(expires_at) WHERE status = 'active';
CREATE INDEX idx_chunked_sessions_owner ON storage.chunked_upload_sessions(owner_id) WHERE owner_id IS NOT NULL;
