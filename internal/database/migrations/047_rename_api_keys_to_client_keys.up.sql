-- ============================================================================
-- RENAME client keys TO CLIENT KEYS
-- ============================================================================
-- This migration renames "client keys" to "Client Keys" throughout the database
-- to better distinguish user-scoped client keys (RLS-enforced) from
-- system-scoped service keys (RLS-bypassing).
-- ============================================================================

-- Rename main table
ALTER TABLE auth.api_keys RENAME TO client_keys;

-- Rename usage tracking table
ALTER TABLE auth.api_key_usage RENAME TO client_key_usage;

-- Rename foreign key column in usage table
ALTER TABLE auth.client_key_usage RENAME COLUMN api_key_id TO client_key_id;

-- Rename indexes on client_keys table (use IF EXISTS since some indexes may not exist
-- depending on PostgreSQL version behavior with CREATE INDEX IF NOT EXISTS on unique columns)
ALTER INDEX IF EXISTS idx_auth_api_keys_key_hash RENAME TO idx_auth_client_keys_key_hash;
ALTER INDEX IF EXISTS idx_auth_api_keys_user_id RENAME TO idx_auth_client_keys_user_id;
ALTER INDEX IF EXISTS idx_auth_api_keys_key_prefix RENAME TO idx_auth_client_keys_key_prefix;

-- Create the indexes if they don't exist (ensures consistent state after migration)
CREATE INDEX IF NOT EXISTS idx_auth_client_keys_key_hash ON auth.client_keys(key_hash);
CREATE INDEX IF NOT EXISTS idx_auth_client_keys_user_id ON auth.client_keys(user_id);
CREATE INDEX IF NOT EXISTS idx_auth_client_keys_key_prefix ON auth.client_keys(key_prefix);

-- Rename indexes on client_key_usage table
ALTER INDEX IF EXISTS idx_auth_api_key_usage_api_key_id RENAME TO idx_auth_client_key_usage_client_key_id;
ALTER INDEX IF EXISTS idx_auth_api_key_usage_created_at RENAME TO idx_auth_client_key_usage_created_at;

-- Create the usage indexes if they don't exist
CREATE INDEX IF NOT EXISTS idx_auth_client_key_usage_client_key_id ON auth.client_key_usage(client_key_id);
CREATE INDEX IF NOT EXISTS idx_auth_client_key_usage_created_at ON auth.client_key_usage(created_at DESC);

-- Rename trigger (drop and recreate since ALTER TRIGGER ... RENAME is not supported for ON clause)
DROP TRIGGER IF EXISTS update_auth_api_keys_updated_at ON auth.client_keys;
CREATE TRIGGER update_auth_client_keys_updated_at BEFORE UPDATE ON auth.client_keys
    FOR EACH ROW EXECUTE FUNCTION public.update_updated_at();

-- Update RLS policies for client_keys table
-- First drop old policies (they reference old table name in policy name)
DROP POLICY IF EXISTS auth_api_keys_policy ON auth.client_keys;

-- Recreate with new name
CREATE POLICY auth_client_keys_policy ON auth.client_keys
    FOR ALL
    USING (
        auth.is_admin()
        OR auth.current_user_role() = 'dashboard_admin'
        OR auth.current_user_id()::TEXT = user_id::TEXT
    );

-- Update RLS policies for client_key_usage table
DROP POLICY IF EXISTS api_key_usage_service_write ON auth.client_key_usage;
CREATE POLICY client_key_usage_service_write ON auth.client_key_usage
    FOR INSERT
    WITH CHECK (auth.current_user_role() = 'service_role');

COMMENT ON POLICY client_key_usage_service_write ON auth.client_key_usage IS 'Service role can record client key usage.';

DROP POLICY IF EXISTS api_key_usage_user_read ON auth.client_key_usage;
CREATE POLICY client_key_usage_user_read ON auth.client_key_usage
    FOR SELECT
    USING (
        client_key_id IN (
            SELECT id FROM auth.client_keys WHERE user_id = auth.current_user_id()
        )
        OR auth.is_admin()
        OR auth.current_user_role() = 'dashboard_admin'
        OR auth.current_user_role() = 'service_role'
    );

COMMENT ON POLICY client_key_usage_user_read ON auth.client_key_usage IS 'Users can view usage for their own client keys. Admins can view all usage.';
