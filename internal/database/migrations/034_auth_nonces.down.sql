-- Rollback: 034_auth_nonces
-- Remove nonces table

DROP TABLE IF EXISTS auth.nonces;
