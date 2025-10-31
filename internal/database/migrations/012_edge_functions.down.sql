-- Rollback edge functions schema

DROP TRIGGER IF EXISTS edge_functions_updated_at ON edge_functions;
DROP FUNCTION IF EXISTS update_edge_function_updated_at();
DROP FUNCTION IF EXISTS cleanup_old_edge_function_executions();

DROP TABLE IF EXISTS edge_function_triggers CASCADE;
DROP TABLE IF EXISTS edge_function_executions CASCADE;
DROP TABLE IF EXISTS edge_functions CASCADE;
