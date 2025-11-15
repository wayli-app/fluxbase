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
