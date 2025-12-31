-- 051_namespace_authorization.down.sql
-- Rollback namespace authorization support

-- Remove allowed_namespaces column from api_keys
ALTER TABLE auth.api_keys
DROP COLUMN IF EXISTS allowed_namespaces;

-- Remove allowed_namespaces column from service_keys
ALTER TABLE auth.service_keys
DROP COLUMN IF EXISTS allowed_namespaces;

-- Note: We don't drop the indexes as they may still be useful for performance
-- even without the namespace authorization feature
