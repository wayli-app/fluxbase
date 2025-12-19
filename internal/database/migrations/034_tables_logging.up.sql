-- Central Logging System Schema
-- Provides unified log storage for system, HTTP, security, execution, and AI logs

-- Create logging schema
CREATE SCHEMA IF NOT EXISTS logging;

-- Create the main log entries table with partitioning by category
CREATE TABLE IF NOT EXISTS logging.entries (
    id UUID NOT NULL DEFAULT gen_random_uuid(),
    timestamp TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    category TEXT NOT NULL,
    level TEXT NOT NULL,
    message TEXT NOT NULL,
    request_id TEXT,
    trace_id TEXT,
    component TEXT,
    user_id UUID,
    ip_address INET,
    fields JSONB,
    execution_id UUID,
    line_number INTEGER,

    CONSTRAINT valid_category CHECK (category IN ('system', 'http', 'security', 'execution', 'ai')),
    CONSTRAINT valid_level CHECK (level IN ('trace', 'debug', 'info', 'warn', 'error', 'fatal', 'panic')),
    PRIMARY KEY (id, category)
) PARTITION BY LIST (category);

-- Create partitions for each category
-- This allows efficient querying and cleanup per category
CREATE TABLE IF NOT EXISTS logging.entries_system
    PARTITION OF logging.entries FOR VALUES IN ('system');

CREATE TABLE IF NOT EXISTS logging.entries_http
    PARTITION OF logging.entries FOR VALUES IN ('http');

CREATE TABLE IF NOT EXISTS logging.entries_security
    PARTITION OF logging.entries FOR VALUES IN ('security');

CREATE TABLE IF NOT EXISTS logging.entries_execution
    PARTITION OF logging.entries FOR VALUES IN ('execution');

CREATE TABLE IF NOT EXISTS logging.entries_ai
    PARTITION OF logging.entries FOR VALUES IN ('ai');

-- Indexes for common query patterns

-- Index on timestamp for time-range queries (most common)
CREATE INDEX IF NOT EXISTS idx_logging_entries_timestamp
    ON logging.entries (timestamp DESC);

-- Index on category + timestamp for filtered time-range queries
CREATE INDEX IF NOT EXISTS idx_logging_entries_category_timestamp
    ON logging.entries (category, timestamp DESC);

-- Index on request_id for correlation (tracing a request across logs)
CREATE INDEX IF NOT EXISTS idx_logging_entries_request_id
    ON logging.entries (request_id)
    WHERE request_id IS NOT NULL;

-- Index on trace_id for distributed tracing
CREATE INDEX IF NOT EXISTS idx_logging_entries_trace_id
    ON logging.entries (trace_id)
    WHERE trace_id IS NOT NULL;

-- Index on user_id for user activity queries
CREATE INDEX IF NOT EXISTS idx_logging_entries_user_id
    ON logging.entries (user_id)
    WHERE user_id IS NOT NULL;

-- Index on execution_id for execution log streaming
CREATE INDEX IF NOT EXISTS idx_logging_entries_execution_id
    ON logging.entries (execution_id)
    WHERE execution_id IS NOT NULL;

-- Composite index for execution log streaming (get logs after line number)
CREATE INDEX IF NOT EXISTS idx_logging_entries_execution_line
    ON logging.entries (execution_id, line_number)
    WHERE execution_id IS NOT NULL;

-- Index on level for filtering by severity
CREATE INDEX IF NOT EXISTS idx_logging_entries_level
    ON logging.entries (level);

-- Index on component for filtering by source
CREATE INDEX IF NOT EXISTS idx_logging_entries_component
    ON logging.entries (component)
    WHERE component IS NOT NULL;

-- Full-text search index on message
CREATE INDEX IF NOT EXISTS idx_logging_entries_message_search
    ON logging.entries USING gin(to_tsvector('english', message));

-- Enable replica identity for execution logs partition (for potential future CDC)
ALTER TABLE logging.entries_execution REPLICA IDENTITY FULL;

-- Grant permissions to roles
GRANT USAGE ON SCHEMA logging TO anon, authenticated, service_role;
GRANT ALL ON ALL TABLES IN SCHEMA logging TO service_role;
GRANT ALL ON ALL SEQUENCES IN SCHEMA logging TO service_role;

-- Comment on schema and tables
COMMENT ON SCHEMA logging IS 'Central logging system for unified log storage';
COMMENT ON TABLE logging.entries IS 'Unified log entries table, partitioned by category';
