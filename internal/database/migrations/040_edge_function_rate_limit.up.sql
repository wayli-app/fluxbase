--
-- Add rate limiting columns to edge functions
-- Supports per-minute, per-hour, and per-day rate limits
--

ALTER TABLE functions.edge_functions
ADD COLUMN IF NOT EXISTS rate_limit_per_minute INTEGER,
ADD COLUMN IF NOT EXISTS rate_limit_per_hour INTEGER,
ADD COLUMN IF NOT EXISTS rate_limit_per_day INTEGER;

COMMENT ON COLUMN functions.edge_functions.rate_limit_per_minute IS 'Maximum requests per minute per user/IP. NULL means unlimited.';
COMMENT ON COLUMN functions.edge_functions.rate_limit_per_hour IS 'Maximum requests per hour per user/IP. NULL means unlimited.';
COMMENT ON COLUMN functions.edge_functions.rate_limit_per_day IS 'Maximum requests per day per user/IP. NULL means unlimited.';
