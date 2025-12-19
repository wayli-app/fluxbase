-- System Tables Migration
-- Creates system-level tables for infrastructure features
-- Note: system schema is created in 002_schemas

-- ============================================================================
-- RATE LIMITS
-- Distributed rate limiting storage for multi-instance deployments
-- Used when scaling.backend is set to "postgres"
-- ============================================================================

CREATE TABLE IF NOT EXISTS system.rate_limits (
    key TEXT PRIMARY KEY,
    count BIGINT NOT NULL DEFAULT 1,
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Index for cleanup queries (expired entries)
CREATE INDEX IF NOT EXISTS idx_rate_limits_expires_at
ON system.rate_limits (expires_at);

COMMENT ON TABLE system.rate_limits IS 'Distributed rate limiting storage for multi-instance deployments';
COMMENT ON COLUMN system.rate_limits.key IS 'Rate limit key (e.g., "login:192.168.1.1")';
COMMENT ON COLUMN system.rate_limits.count IS 'Number of requests in the current window';
COMMENT ON COLUMN system.rate_limits.expires_at IS 'When this rate limit window expires';
