-- Drop all functions schema RLS policies
DROP POLICY IF EXISTS functions_edge_function_executions_policy ON functions.edge_function_executions;
DROP POLICY IF EXISTS functions_edge_function_triggers_policy ON functions.edge_function_triggers;
DROP POLICY IF EXISTS functions_edge_functions_policy ON functions.edge_functions;
