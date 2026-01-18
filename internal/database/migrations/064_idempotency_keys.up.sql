-- Create schema for API management features
CREATE SCHEMA IF NOT EXISTS api;

-- Table to store idempotency keys and their responses
-- Used to safely retry POST/PUT/DELETE requests without duplicate processing
CREATE TABLE IF NOT EXISTS api.idempotency_keys (
    -- The idempotency key (client-provided, typically UUID)
    key TEXT PRIMARY KEY,

    -- HTTP method (POST, PUT, DELETE, PATCH)
    method TEXT NOT NULL,

    -- Request path (e.g., /api/v1/rest/users)
    path TEXT NOT NULL,

    -- User ID if authenticated (NULL for anonymous)
    user_id UUID,

    -- Hash of request body for validation
    request_hash TEXT,

    -- Processing status
    status TEXT NOT NULL DEFAULT 'processing' CHECK (status IN ('processing', 'completed', 'failed')),

    -- HTTP status code of the response
    response_status INTEGER,

    -- Response headers (JSON)
    response_headers JSONB,

    -- Response body (stored as bytes, base64 encoded for binary)
    response_body BYTEA,

    -- When the key was created
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    -- When the response was stored
    completed_at TIMESTAMPTZ,

    -- When the key expires (default: 24 hours after creation)
    expires_at TIMESTAMPTZ NOT NULL DEFAULT NOW() + INTERVAL '24 hours'
);

-- Index for fast lookup by key
-- (Primary key already creates index)

-- Index for cleanup of expired keys
CREATE INDEX IF NOT EXISTS idx_idempotency_keys_expires_at
    ON api.idempotency_keys(expires_at);

-- Index for user-specific lookups
CREATE INDEX IF NOT EXISTS idx_idempotency_keys_user_id
    ON api.idempotency_keys(user_id)
    WHERE user_id IS NOT NULL;

-- Composite index for conflict detection
CREATE INDEX IF NOT EXISTS idx_idempotency_keys_method_path
    ON api.idempotency_keys(method, path);

-- Add comment
COMMENT ON TABLE api.idempotency_keys IS 'Stores idempotency keys for safe request retries';
COMMENT ON COLUMN api.idempotency_keys.key IS 'Client-provided idempotency key (typically UUID)';
COMMENT ON COLUMN api.idempotency_keys.status IS 'processing: request in progress, completed: response cached, failed: error occurred';
