-- RPC Schema Rollback Migration
-- Drops all RPC-related tables, policies, and settings

-- Remove from realtime registry
DELETE FROM realtime.schema_registry WHERE schema_name = 'rpc';

-- Drop triggers
DROP TRIGGER IF EXISTS executions_realtime_notify ON rpc.executions;
DROP TRIGGER IF EXISTS procedures_update_updated_at ON rpc.procedures;

-- Drop functions
DROP FUNCTION IF EXISTS rpc.notify_realtime_change();
DROP FUNCTION IF EXISTS rpc.update_updated_at();

-- Drop tables (CASCADE will drop policies, indexes, and foreign key references)
DROP TABLE IF EXISTS rpc.executions CASCADE;
DROP TABLE IF EXISTS rpc.procedures CASCADE;

-- Drop schema
DROP SCHEMA IF EXISTS rpc CASCADE;

-- Remove settings
DELETE FROM app.settings WHERE key LIKE 'app.rpc.%';
DELETE FROM app.settings WHERE key = 'app.features.enable_rpc';
