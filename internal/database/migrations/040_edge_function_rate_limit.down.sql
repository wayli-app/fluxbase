--
-- Remove rate limiting columns from edge functions
--

ALTER TABLE functions.edge_functions
DROP COLUMN IF EXISTS rate_limit_per_minute,
DROP COLUMN IF EXISTS rate_limit_per_hour,
DROP COLUMN IF EXISTS rate_limit_per_day;
