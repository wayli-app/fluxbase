-- Remove rate limiting columns from service keys

DROP INDEX IF EXISTS auth.idx_service_keys_rate_limits;

ALTER TABLE auth.service_keys
DROP COLUMN IF EXISTS rate_limit_per_minute,
DROP COLUMN IF EXISTS rate_limit_per_hour;
