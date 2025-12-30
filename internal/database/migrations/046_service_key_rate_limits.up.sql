-- Add rate limiting columns to service keys
-- Service keys (sk_*) can now have configurable rate limits

-- Add rate limit columns to auth.service_keys
ALTER TABLE auth.service_keys
ADD COLUMN IF NOT EXISTS rate_limit_per_minute INTEGER DEFAULT NULL,
ADD COLUMN IF NOT EXISTS rate_limit_per_hour INTEGER DEFAULT NULL;

-- Add comments for documentation
COMMENT ON COLUMN auth.service_keys.rate_limit_per_minute IS 'Maximum requests per minute. NULL means no limit (unlimited).';
COMMENT ON COLUMN auth.service_keys.rate_limit_per_hour IS 'Maximum requests per hour. NULL means no limit (unlimited).';

-- Create index for efficient rate limit lookups
CREATE INDEX IF NOT EXISTS idx_service_keys_rate_limits
ON auth.service_keys(id)
WHERE rate_limit_per_minute IS NOT NULL OR rate_limit_per_hour IS NOT NULL;
