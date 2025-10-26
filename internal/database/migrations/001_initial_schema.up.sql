-- Create auth schema
CREATE SCHEMA IF NOT EXISTS auth;

-- Create auth.users table
CREATE TABLE IF NOT EXISTS auth.users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email TEXT UNIQUE NOT NULL,
    encrypted_password TEXT,
    email_confirmed BOOLEAN DEFAULT false,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Create auth.sessions table
CREATE TABLE IF NOT EXISTS auth.sessions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID REFERENCES auth.users(id) ON DELETE CASCADE,
    token TEXT UNIQUE NOT NULL,
    refresh_token TEXT UNIQUE,
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Create auth.magic_links table
CREATE TABLE IF NOT EXISTS auth.magic_links (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email TEXT NOT NULL,
    token TEXT UNIQUE NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    used BOOLEAN DEFAULT false,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Create storage schema
CREATE SCHEMA IF NOT EXISTS storage;

-- Create storage.buckets table
CREATE TABLE IF NOT EXISTS storage.buckets (
    id TEXT PRIMARY KEY,
    name TEXT UNIQUE NOT NULL,
    public BOOLEAN DEFAULT false,
    allowed_mime_types TEXT[],
    max_file_size BIGINT,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Create storage.objects table
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

-- Create realtime schema
CREATE SCHEMA IF NOT EXISTS realtime;

-- Create realtime.schema_registry table
CREATE TABLE IF NOT EXISTS realtime.schema_registry (
    id SERIAL PRIMARY KEY,
    schema_name TEXT NOT NULL,
    table_name TEXT NOT NULL,
    realtime_enabled BOOLEAN DEFAULT true,
    events TEXT[] DEFAULT ARRAY['INSERT', 'UPDATE', 'DELETE'],
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(schema_name, table_name)
);

-- Create functions schema for edge functions metadata
CREATE SCHEMA IF NOT EXISTS functions;

-- Create functions.registry table
CREATE TABLE IF NOT EXISTS functions.registry (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT UNIQUE NOT NULL,
    runtime TEXT NOT NULL DEFAULT 'javascript',
    code TEXT NOT NULL,
    config JSONB,
    schedule TEXT, -- cron expression
    http_enabled BOOLEAN DEFAULT true,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Create functions.invocations table for logging
CREATE TABLE IF NOT EXISTS functions.invocations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    function_id UUID REFERENCES functions.registry(id) ON DELETE CASCADE,
    status TEXT NOT NULL, -- 'success', 'error'
    request JSONB,
    response JSONB,
    error TEXT,
    duration_ms INTEGER,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Create indexes for better performance
CREATE INDEX IF NOT EXISTS idx_auth_users_email ON auth.users(email);
CREATE INDEX IF NOT EXISTS idx_auth_sessions_user_id ON auth.sessions(user_id);
CREATE INDEX IF NOT EXISTS idx_auth_sessions_token ON auth.sessions(token);
CREATE INDEX IF NOT EXISTS idx_auth_magic_links_token ON auth.magic_links(token);
CREATE INDEX IF NOT EXISTS idx_storage_objects_bucket_id ON storage.objects(bucket_id);
CREATE INDEX IF NOT EXISTS idx_storage_objects_owner_id ON storage.objects(owner_id);
CREATE INDEX IF NOT EXISTS idx_functions_invocations_function_id ON functions.invocations(function_id);
CREATE INDEX IF NOT EXISTS idx_functions_invocations_created_at ON functions.invocations(created_at DESC);

-- Create update trigger for updated_at columns
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

-- Apply update trigger to tables with updated_at
CREATE TRIGGER update_auth_users_updated_at BEFORE UPDATE ON auth.users
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_auth_sessions_updated_at BEFORE UPDATE ON auth.sessions
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_storage_buckets_updated_at BEFORE UPDATE ON storage.buckets
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_storage_objects_updated_at BEFORE UPDATE ON storage.objects
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_realtime_schema_registry_updated_at BEFORE UPDATE ON realtime.schema_registry
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_functions_registry_updated_at BEFORE UPDATE ON functions.registry
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();