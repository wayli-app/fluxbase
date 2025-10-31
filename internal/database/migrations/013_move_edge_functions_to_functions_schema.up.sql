-- Move edge functions tables from public schema to functions schema

-- First, create the functions schema if it doesn't exist
CREATE SCHEMA IF NOT EXISTS functions;

-- Move tables to functions schema
ALTER TABLE public.edge_functions SET SCHEMA functions;
ALTER TABLE public.edge_function_executions SET SCHEMA functions;
ALTER TABLE public.edge_function_triggers SET SCHEMA functions;

-- Move the trigger function to functions schema
ALTER FUNCTION public.update_edge_function_updated_at() SET SCHEMA functions;
ALTER FUNCTION public.cleanup_old_edge_function_executions() SET SCHEMA functions;

-- Update the trigger to use the new schema
DROP TRIGGER IF EXISTS edge_functions_updated_at ON functions.edge_functions;
CREATE TRIGGER edge_functions_updated_at
    BEFORE UPDATE ON functions.edge_functions
    FOR EACH ROW
    EXECUTE FUNCTION functions.update_edge_function_updated_at();

-- Update grants to reference the correct schema
REVOKE ALL ON public.edge_functions FROM authenticated;
REVOKE ALL ON public.edge_function_executions FROM authenticated;
REVOKE ALL ON public.edge_function_triggers FROM authenticated;

GRANT SELECT, INSERT, UPDATE, DELETE ON functions.edge_functions TO authenticated;
GRANT SELECT ON functions.edge_function_executions TO authenticated;
GRANT SELECT ON functions.edge_function_triggers TO authenticated;

-- Update comments
COMMENT ON TABLE functions.edge_functions IS 'Stores Deno-based serverless functions';
COMMENT ON TABLE functions.edge_function_executions IS 'Logs all function execution attempts with results';
COMMENT ON TABLE functions.edge_function_triggers IS 'Database triggers that invoke edge functions on table changes';
