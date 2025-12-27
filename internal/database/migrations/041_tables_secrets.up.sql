-- Secrets Management Tables
-- Stores encrypted secrets that are injected into edge functions at runtime

-- Secrets table (stores encrypted secret values)
CREATE TABLE IF NOT EXISTS functions.secrets (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL,
    -- Scope: 'global' or 'namespace'
    scope TEXT NOT NULL DEFAULT 'global' CHECK (scope IN ('global', 'namespace')),
    -- For namespace-scoped secrets (NULL for global)
    namespace TEXT,
    -- Encrypted value using AES-256-GCM (base64 encoded with prepended nonce)
    encrypted_value TEXT NOT NULL,
    -- Description for documentation
    description TEXT,
    -- Version for tracking changes (incremented on each update)
    version INTEGER NOT NULL DEFAULT 1,
    -- Optional expiration timestamp
    expires_at TIMESTAMPTZ,
    -- Audit fields
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by UUID REFERENCES auth.users(id) ON DELETE SET NULL,
    updated_by UUID REFERENCES auth.users(id) ON DELETE SET NULL,
    -- Ensure unique names within scope
    -- For global: name must be unique where namespace IS NULL
    -- For namespace: name must be unique per namespace
    CONSTRAINT unique_secret_name_scope UNIQUE (name, scope, namespace)
);

-- Secret versions table (for audit trail and rollback)
CREATE TABLE IF NOT EXISTS functions.secret_versions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    secret_id UUID NOT NULL REFERENCES functions.secrets(id) ON DELETE CASCADE,
    version INTEGER NOT NULL,
    encrypted_value TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by UUID REFERENCES auth.users(id) ON DELETE SET NULL,
    CONSTRAINT unique_secret_version UNIQUE (secret_id, version)
);

-- Indexes for efficient lookups
CREATE INDEX IF NOT EXISTS idx_secrets_scope ON functions.secrets(scope);
CREATE INDEX IF NOT EXISTS idx_secrets_namespace ON functions.secrets(namespace) WHERE namespace IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_secrets_expires_at ON functions.secrets(expires_at) WHERE expires_at IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_secrets_name ON functions.secrets(name);
CREATE INDEX IF NOT EXISTS idx_secret_versions_secret_id ON functions.secret_versions(secret_id);

-- Comments for documentation
COMMENT ON TABLE functions.secrets IS 'Encrypted secrets injected into edge functions at runtime';
COMMENT ON COLUMN functions.secrets.scope IS 'Scope: global (all functions) or namespace (functions in specific namespace)';
COMMENT ON COLUMN functions.secrets.encrypted_value IS 'AES-256-GCM encrypted secret value (base64 with prepended nonce)';
COMMENT ON COLUMN functions.secrets.version IS 'Incremented on each update for tracking changes';
COMMENT ON TABLE functions.secret_versions IS 'Version history for secrets (audit trail and rollback capability)';
