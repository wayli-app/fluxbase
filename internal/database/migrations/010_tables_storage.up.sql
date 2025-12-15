--
-- STORAGE SCHEMA TABLES
-- File storage buckets and objects
--

-- Storage buckets table
CREATE TABLE IF NOT EXISTS storage.buckets (
    id TEXT PRIMARY KEY,
    name TEXT UNIQUE NOT NULL,
    public BOOLEAN DEFAULT false,
    allowed_mime_types TEXT[],
    max_file_size BIGINT,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Storage objects table
CREATE TABLE IF NOT EXISTS storage.objects (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    bucket_id TEXT REFERENCES storage.buckets(id) ON DELETE CASCADE,
    path TEXT NOT NULL,
    mime_type TEXT,
    size BIGINT,
    metadata JSONB,
    owner_id UUID REFERENCES auth.users(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(bucket_id, path)
);

CREATE INDEX IF NOT EXISTS idx_storage_objects_bucket_id ON storage.objects(bucket_id);
CREATE INDEX IF NOT EXISTS idx_storage_objects_owner_id ON storage.objects(owner_id);

-- Supabase compatibility: Add 'name' as an alias for 'path' using a generated column
-- This allows Supabase migrations that use 'name' to work seamlessly
ALTER TABLE storage.objects ADD COLUMN IF NOT EXISTS name TEXT GENERATED ALWAYS AS (path) STORED;

COMMENT ON COLUMN storage.objects.name IS 'Supabase-compatible alias for path column. Automatically synchronized with path.';
COMMENT ON COLUMN storage.objects.path IS 'Full path to the object within the bucket. Also accessible via the "name" column for Supabase compatibility.';

-- Storage object permissions table (for file sharing)
CREATE TABLE IF NOT EXISTS storage.object_permissions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    object_id UUID REFERENCES storage.objects(id) ON DELETE CASCADE,
    user_id UUID REFERENCES auth.users(id) ON DELETE CASCADE,
    permission TEXT NOT NULL CHECK (permission IN ('read', 'write')),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(object_id, user_id)
);

CREATE INDEX IF NOT EXISTS idx_storage_object_permissions_object_id ON storage.object_permissions(object_id);
CREATE INDEX IF NOT EXISTS idx_storage_object_permissions_user_id ON storage.object_permissions(user_id);

COMMENT ON TABLE storage.object_permissions IS 'Tracks file sharing permissions between users';
COMMENT ON COLUMN storage.object_permissions.permission IS 'Permission level: read (download only) or write (download, update, delete)';

-- Insert default buckets
INSERT INTO storage.buckets (id, name, public) VALUES
    ('public', 'public', true),
    ('temp-files', 'temp-files', false),
    ('user-uploads', 'user-uploads', false)
ON CONFLICT (id) DO NOTHING;

COMMENT ON TABLE storage.buckets IS 'Storage buckets configuration. Public buckets allow unauthenticated read access.';
COMMENT ON TABLE storage.objects IS 'Storage objects metadata. All file operations are tracked here for RLS enforcement.';

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
