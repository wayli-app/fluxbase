-- Drop idempotency keys table
DROP TABLE IF EXISTS api.idempotency_keys;

-- Only drop schema if empty (other tables might use it)
-- DROP SCHEMA IF EXISTS api;
