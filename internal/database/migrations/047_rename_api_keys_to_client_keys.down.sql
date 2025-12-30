-- ============================================================================
-- REVERT: RENAME CLIENT KEYS BACK TO API KEYS
-- ============================================================================

-- Revert RLS policies for client_key_usage table
DROP POLICY IF EXISTS client_key_usage_user_read ON auth.client_key_usage;
CREATE POLICY api_key_usage_user_read ON auth.client_key_usage
    FOR SELECT
    USING (
        client_key_id IN (
            SELECT id FROM auth.client_keys WHERE user_id = auth.current_user_id()
        )
        OR auth.is_admin()
        OR auth.current_user_role() = 'dashboard_admin'
        OR auth.current_user_role() = 'service_role'
    );

COMMENT ON POLICY api_key_usage_user_read ON auth.client_key_usage IS 'Users can view usage for their own API keys. Admins can view all usage.';

DROP POLICY IF EXISTS client_key_usage_service_write ON auth.client_key_usage;
CREATE POLICY api_key_usage_service_write ON auth.client_key_usage
    FOR INSERT
    WITH CHECK (auth.current_user_role() = 'service_role');

COMMENT ON POLICY api_key_usage_service_write ON auth.client_key_usage IS 'Service role can record API key usage.';

-- Revert RLS policies for client_keys table
DROP POLICY IF EXISTS auth_client_keys_policy ON auth.client_keys;
CREATE POLICY auth_api_keys_policy ON auth.client_keys
    FOR ALL
    USING (
        auth.is_admin()
        OR auth.current_user_role() = 'dashboard_admin'
        OR auth.current_user_id()::TEXT = user_id::TEXT
    );

-- Revert trigger
DROP TRIGGER IF EXISTS update_auth_client_keys_updated_at ON auth.client_keys;
CREATE TRIGGER update_auth_api_keys_updated_at BEFORE UPDATE ON auth.client_keys
    FOR EACH ROW EXECUTE FUNCTION public.update_updated_at();

-- Revert indexes on client_key_usage table
ALTER INDEX idx_auth_client_key_usage_created_at RENAME TO idx_auth_api_key_usage_created_at;
ALTER INDEX idx_auth_client_key_usage_client_key_id RENAME TO idx_auth_api_key_usage_api_key_id;

-- Revert indexes on client_keys table
ALTER INDEX idx_auth_client_keys_key_prefix RENAME TO idx_auth_api_keys_key_prefix;
ALTER INDEX idx_auth_client_keys_user_id RENAME TO idx_auth_api_keys_user_id;
ALTER INDEX idx_auth_client_keys_key_hash RENAME TO idx_auth_api_keys_key_hash;

-- Revert foreign key column name
ALTER TABLE auth.client_key_usage RENAME COLUMN client_key_id TO api_key_id;

-- Revert table names
ALTER TABLE auth.client_key_usage RENAME TO api_key_usage;
ALTER TABLE auth.client_keys RENAME TO api_keys;
