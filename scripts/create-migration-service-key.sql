-- Create Migration Service Key
-- This script creates a dedicated service key for the migrations API
--
-- Usage:
-- 1. Generate a secure random key (32+ characters recommended):
--    openssl rand -base64 32 | tr -d "=+/" | cut -c1-40
-- 2. Set the generated key in the variable below
-- 3. Run this script: psql -f scripts/create-migration-service-key.sql
-- 4. Store the key securely in your app's environment variables
--
-- IMPORTANT: Save the generated service key before running this script!
-- The key cannot be recovered after hashing.

-- Configuration variables (REPLACE THESE VALUES)
\set service_key_value '''sk_migrations_YOUR_GENERATED_KEY_HERE'''
\set service_key_name '''Production Migrations Service Key'''
\set service_key_description '''Dedicated service key for executing database migrations from application container'''

-- Insert the service key with migrations scope
-- Note: The key will be hashed with bcrypt before storage
INSERT INTO auth.service_keys (
    name,
    description,
    key_hash,
    key_prefix,
    scopes,
    enabled,
    expires_at
) VALUES (
    :service_key_name,
    :service_key_description,
    crypt(:service_key_value, gen_salt('bf')),  -- Bcrypt hash
    substring(:service_key_value, 1, 16),        -- First 16 chars for identification
    ARRAY['migrations:execute', 'migrations:read'],  -- Limited scopes
    true,
    NOW() + INTERVAL '1 year'  -- Expire after 1 year (adjust as needed)
)
RETURNING
    id,
    name,
    key_prefix,
    scopes,
    enabled,
    expires_at;

-- Display success message
\echo ''
\echo 'Migration service key created successfully!'
\echo ''
\echo 'IMPORTANT: Store this key securely in your application environment:'
\echo 'FLUXBASE_MIGRATIONS_SERVICE_KEY=' :service_key_value
\echo ''
\echo 'In your Fluxbase configuration, enable migrations API:'
\echo 'FLUXBASE_MIGRATIONS_ENABLED=true'
\echo 'FLUXBASE_MIGRATIONS_REQUIRE_SERVICE_KEY=true'
\echo ''
\echo 'The key will expire after 1 year. Remember to rotate it before expiration.'
\echo ''
