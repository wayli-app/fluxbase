-- Drop revocation audit table
DROP TABLE IF EXISTS auth.service_key_revocations;

-- Drop indexes
DROP INDEX IF EXISTS auth.idx_service_keys_revoked_at;
DROP INDEX IF EXISTS auth.idx_service_keys_grace_period;

-- Remove columns from service_keys
ALTER TABLE auth.service_keys
DROP COLUMN IF EXISTS revoked_at,
DROP COLUMN IF EXISTS revoked_by,
DROP COLUMN IF EXISTS revocation_reason,
DROP COLUMN IF EXISTS deprecated_at,
DROP COLUMN IF EXISTS grace_period_ends_at,
DROP COLUMN IF EXISTS replaced_by;
