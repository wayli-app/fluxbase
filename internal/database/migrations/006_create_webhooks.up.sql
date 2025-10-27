-- Webhooks system for HTTP notifications on database changes
CREATE TABLE IF NOT EXISTS auth.webhooks (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL,
    description TEXT,
    url TEXT NOT NULL, -- Target webhook URL
    secret TEXT, -- Optional HMAC secret for webhook verification
    enabled BOOLEAN DEFAULT true,

    -- Event configuration
    events JSONB NOT NULL DEFAULT '[]'::JSONB, -- Array of event configs: [{"table": "products", "operations": ["INSERT", "UPDATE"]}]

    -- Retry configuration
    max_retries INTEGER DEFAULT 3,
    retry_backoff_seconds INTEGER DEFAULT 5,
    timeout_seconds INTEGER DEFAULT 30,

    -- Custom headers (for authentication, etc.)
    headers JSONB DEFAULT '{}'::JSONB, -- {"Authorization": "Bearer xxx", "X-API-Key": "xxx"}

    -- Metadata
    created_by UUID REFERENCES auth.users(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Webhook delivery logs
CREATE TABLE IF NOT EXISTS auth.webhook_deliveries (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    webhook_id UUID REFERENCES auth.webhooks(id) ON DELETE CASCADE,

    -- Event details
    event_type TEXT NOT NULL, -- INSERT, UPDATE, DELETE
    table_name TEXT NOT NULL,
    record_id TEXT, -- Primary key of affected record (if available)
    payload JSONB NOT NULL, -- Full event payload

    -- Delivery attempt
    attempt_number INTEGER DEFAULT 1,
    status TEXT NOT NULL, -- pending, success, failed, retrying
    http_status_code INTEGER,
    response_body TEXT,
    error_message TEXT,

    -- Timing
    created_at TIMESTAMPTZ DEFAULT NOW(),
    delivered_at TIMESTAMPTZ,
    next_retry_at TIMESTAMPTZ
);

-- Indexes for webhook deliveries
CREATE INDEX IF NOT EXISTS idx_webhook_deliveries_webhook_id ON auth.webhook_deliveries(webhook_id);
CREATE INDEX IF NOT EXISTS idx_webhook_deliveries_status ON auth.webhook_deliveries(status);
CREATE INDEX IF NOT EXISTS idx_webhook_deliveries_next_retry ON auth.webhook_deliveries(next_retry_at) WHERE status = 'retrying';
CREATE INDEX IF NOT EXISTS idx_webhook_deliveries_created_at ON auth.webhook_deliveries(created_at);

-- Updated trigger
CREATE OR REPLACE FUNCTION auth.update_webhook_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER webhook_updated_at
BEFORE UPDATE ON auth.webhooks
FOR EACH ROW
EXECUTE FUNCTION auth.update_webhook_updated_at();

-- Add comment
COMMENT ON TABLE auth.webhooks IS 'Webhook configurations for HTTP notifications on database changes';
COMMENT ON TABLE auth.webhook_deliveries IS 'Log of webhook delivery attempts and their status';
