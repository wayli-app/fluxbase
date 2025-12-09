-- Remove execution_logs table for edge functions

-- Remove from realtime registry
DELETE FROM realtime.schema_registry
WHERE schema_name = 'functions' AND table_name = 'execution_logs';

-- Drop trigger
DROP TRIGGER IF EXISTS execution_logs_realtime_notify ON functions.execution_logs;

-- Drop table (cascade will handle indexes and policies)
DROP TABLE IF EXISTS functions.execution_logs;
