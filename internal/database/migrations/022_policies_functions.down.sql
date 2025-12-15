-- Drop all functions schema RLS policies

-- Execution logs
DROP POLICY IF EXISTS functions_execution_logs_owner ON functions.execution_logs;
DROP POLICY IF EXISTS functions_execution_logs_admin ON functions.execution_logs;
DROP POLICY IF EXISTS functions_execution_logs_service_all ON functions.execution_logs;

-- Function dependencies
DROP POLICY IF EXISTS functions_dependencies_owner ON functions.function_dependencies;
DROP POLICY IF EXISTS functions_dependencies_admin ON functions.function_dependencies;
DROP POLICY IF EXISTS functions_dependencies_service ON functions.function_dependencies;

-- Shared modules
DROP POLICY IF EXISTS functions_shared_modules_read ON functions.shared_modules;
DROP POLICY IF EXISTS functions_shared_modules_owner ON functions.shared_modules;
DROP POLICY IF EXISTS functions_shared_modules_admin ON functions.shared_modules;
DROP POLICY IF EXISTS functions_shared_modules_service ON functions.shared_modules;

-- Edge files
DROP POLICY IF EXISTS functions_edge_files_owner ON functions.edge_files;
DROP POLICY IF EXISTS functions_edge_files_admin ON functions.edge_files;
DROP POLICY IF EXISTS functions_edge_files_service ON functions.edge_files;

-- Edge executions
DROP POLICY IF EXISTS functions_edge_executions_owner ON functions.edge_executions;
DROP POLICY IF EXISTS functions_edge_executions_admin ON functions.edge_executions;
DROP POLICY IF EXISTS functions_edge_executions_service ON functions.edge_executions;
DROP POLICY IF EXISTS functions_edge_executions_policy ON functions.edge_executions;

-- Edge triggers
DROP POLICY IF EXISTS functions_edge_triggers_owner ON functions.edge_triggers;
DROP POLICY IF EXISTS functions_edge_triggers_admin ON functions.edge_triggers;
DROP POLICY IF EXISTS functions_edge_triggers_service ON functions.edge_triggers;
DROP POLICY IF EXISTS functions_edge_triggers_policy ON functions.edge_triggers;

-- Edge functions
DROP POLICY IF EXISTS functions_edge_functions_public_read ON functions.edge_functions;
DROP POLICY IF EXISTS functions_edge_functions_owner ON functions.edge_functions;
DROP POLICY IF EXISTS functions_edge_functions_admin ON functions.edge_functions;
DROP POLICY IF EXISTS functions_edge_functions_service ON functions.edge_functions;
DROP POLICY IF EXISTS functions_edge_functions_policy ON functions.edge_functions;
