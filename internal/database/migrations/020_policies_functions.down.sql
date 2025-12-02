-- Drop all functions schema RLS policies
DROP POLICY IF EXISTS functions_edge_executions_policy ON functions.edge_executions;
DROP POLICY IF EXISTS functions_edge_triggers_policy ON functions.edge_triggers;
DROP POLICY IF EXISTS functions_edge_functions_policy ON functions.edge_functions;
