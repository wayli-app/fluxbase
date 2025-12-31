-- 051_namespace_authorization.up.sql
-- Add namespace authorization support to client keys and service keys

-- Add allowed_namespaces column to client_keys
ALTER TABLE auth.client_keys
ADD COLUMN allowed_namespaces TEXT[] DEFAULT NULL;

COMMENT ON COLUMN auth.client_keys.allowed_namespaces IS
'Allowed namespaces for this key. NULL = all namespaces (no restrictions), empty array = default namespace only, populated array = specific namespaces allowed.';

-- Add allowed_namespaces column to service_keys
ALTER TABLE auth.service_keys
ADD COLUMN allowed_namespaces TEXT[] DEFAULT NULL;

COMMENT ON COLUMN auth.service_keys.allowed_namespaces IS
'Allowed namespaces for this key. NULL = all namespaces (no restrictions), empty array = default namespace only, populated array = specific namespaces allowed.';

-- Ensure indexes exist for efficient namespace filtering
CREATE INDEX IF NOT EXISTS idx_functions_namespace
ON functions.edge_functions(namespace);

CREATE INDEX IF NOT EXISTS idx_rpc_procedures_namespace
ON rpc.procedures(namespace);

CREATE INDEX IF NOT EXISTS idx_jobs_functions_namespace
ON jobs.functions(namespace);
