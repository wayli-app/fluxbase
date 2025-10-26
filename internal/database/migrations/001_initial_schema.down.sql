-- Drop triggers
DROP TRIGGER IF EXISTS update_functions_registry_updated_at ON functions.registry;
DROP TRIGGER IF EXISTS update_realtime_schema_registry_updated_at ON realtime.schema_registry;
DROP TRIGGER IF EXISTS update_storage_objects_updated_at ON storage.objects;
DROP TRIGGER IF EXISTS update_storage_buckets_updated_at ON storage.buckets;
DROP TRIGGER IF EXISTS update_auth_sessions_updated_at ON auth.sessions;
DROP TRIGGER IF EXISTS update_auth_users_updated_at ON auth.users;

-- Drop function
DROP FUNCTION IF EXISTS update_updated_at_column();

-- Drop indexes
DROP INDEX IF EXISTS idx_functions_invocations_created_at;
DROP INDEX IF EXISTS idx_functions_invocations_function_id;
DROP INDEX IF EXISTS idx_storage_objects_owner_id;
DROP INDEX IF EXISTS idx_storage_objects_bucket_id;
DROP INDEX IF EXISTS idx_auth_magic_links_token;
DROP INDEX IF EXISTS idx_auth_sessions_token;
DROP INDEX IF EXISTS idx_auth_sessions_user_id;
DROP INDEX IF EXISTS idx_auth_users_email;

-- Drop tables
DROP TABLE IF EXISTS functions.invocations;
DROP TABLE IF EXISTS functions.registry;
DROP TABLE IF EXISTS realtime.schema_registry;
DROP TABLE IF EXISTS storage.objects;
DROP TABLE IF EXISTS storage.buckets;
DROP TABLE IF EXISTS auth.magic_links;
DROP TABLE IF EXISTS auth.sessions;
DROP TABLE IF EXISTS auth.users;

-- Drop schemas
DROP SCHEMA IF EXISTS functions;
DROP SCHEMA IF EXISTS realtime;
DROP SCHEMA IF EXISTS storage;
DROP SCHEMA IF EXISTS auth;