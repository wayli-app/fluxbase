-- Migration 006: Auth Improvements
-- Adds service role key support and per-function authentication configuration

-- Grant temporary ALTER permission to fluxbase_app for this migration
-- This is needed because the table was created by postgres user
DO $$
BEGIN
    -- Check if column already exists to make migration idempotent
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_schema = 'functions'
        AND table_name = 'edge_functions'
        AND column_name = 'allow_unauthenticated'
    ) THEN
        -- Temporarily grant ownership or use postgres role
        -- Since we can't switch roles in a migration, we'll handle this differently
        -- The column will be added by postgres role via admin
        RAISE NOTICE 'Column allow_unauthenticated needs to be added to functions.edge_functions';
        RAISE NOTICE 'Please run as postgres: ALTER TABLE functions.edge_functions ADD COLUMN allow_unauthenticated BOOLEAN DEFAULT false;';
    END IF;
END $$;

-- Alternative: Add column with dynamic SQL if we have sufficient privileges
-- This will succeed if run by postgres or if fluxbase_app has been granted ALTER permission
DO $$
BEGIN
    ALTER TABLE functions.edge_functions
    ADD COLUMN IF NOT EXISTS allow_unauthenticated BOOLEAN DEFAULT false;
EXCEPTION
    WHEN insufficient_privilege THEN
        RAISE NOTICE 'Insufficient privileges to alter functions.edge_functions. Column will need to be added manually.';
    WHEN duplicate_column THEN
        RAISE NOTICE 'Column allow_unauthenticated already exists';
END $$;

-- Add comment if column exists
DO $$
BEGIN
    IF EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_schema = 'functions'
        AND table_name = 'edge_functions'
        AND column_name = 'allow_unauthenticated'
    ) THEN
        COMMENT ON COLUMN functions.edge_functions.allow_unauthenticated IS
        'When true, allows this function to be invoked without authentication. Use with caution.';
    END IF;
END $$;

-- Create service_keys table for service role authentication
CREATE TABLE IF NOT EXISTS auth.service_keys (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL,
    description TEXT,
    key_hash TEXT NOT NULL, -- bcrypt hash of the service key
    key_prefix TEXT NOT NULL, -- First 16 chars for identification (e.g., "sk_test_Ab3xY...")
    scopes TEXT[] DEFAULT ARRAY[]::TEXT[], -- Optional scope restrictions
    enabled BOOLEAN DEFAULT true,
    created_by UUID REFERENCES auth.users(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    last_used_at TIMESTAMPTZ,
    expires_at TIMESTAMPTZ,
    UNIQUE(key_prefix)
);

COMMENT ON TABLE auth.service_keys IS
'Service role keys with elevated privileges that bypass RLS. Use for backend services only.';

COMMENT ON COLUMN auth.service_keys.key_hash IS
'Bcrypt hash of the full service key. Never store keys in plaintext.';

COMMENT ON COLUMN auth.service_keys.key_prefix IS
'First 16 characters of the key for identification in logs (e.g., "sk_test_Ab3xY...").';

COMMENT ON COLUMN auth.service_keys.scopes IS
'Optional array of scope restrictions. Empty array means full service role access.';

-- Create index for efficient key prefix lookups
CREATE INDEX IF NOT EXISTS idx_service_keys_prefix ON auth.service_keys(key_prefix);
CREATE INDEX IF NOT EXISTS idx_service_keys_enabled ON auth.service_keys(enabled);

-- Grant appropriate permissions to roles if they exist
-- These roles are typically created in production environments
DO $$
BEGIN
    -- Grant to authenticated role if it exists
    IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'authenticated') THEN
        GRANT SELECT ON auth.service_keys TO authenticated;
    END IF;

    -- Grant to service_role if it exists
    IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'service_role') THEN
        GRANT ALL ON auth.service_keys TO service_role;
    END IF;
END $$;
